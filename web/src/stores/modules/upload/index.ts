import { sha256 } from 'js-sha256'
import pLimit from 'p-limit'

import api from '@/api/panel/file'
import { dirname, getFilename, joinPath, type PickedFile } from '@/utils/file'
import { $gettext } from '@/utils/gettext'

export type UploadStatus =
  | 'pending'
  | 'waiting'
  | 'uploading'
  | 'paused'
  | 'done'
  | 'error'
  | 'skipped'
export type UploadPriority = 'high' | 'normal' | 'low'
export type ConflictPolicy = 'ask' | 'skip' | 'rename' | 'overwrite'
export type ConflictAction = 'skip' | 'rename' | 'overwrite'

export interface UploadItem {
  id: string
  file: File
  name: string // 相对路径（文件夹上传时含子目录）
  size: number
  target: string // 服务器上的完整目标路径，重命名时只改文件名部分
  status: UploadStatus
  priority: UploadPriority
  loaded: number // 已上传字节
  speed: number // B/s
  error: string
  force: boolean // 覆盖已存在文件
  planned: boolean // 已完成同名检查
  hash: string // 分块上传的文件标识，用于续传和清理临时分块
}

export interface ConflictItem {
  id: string
  name: string
  newName: string
  action: ConflictAction
}

// 超过此大小走分块上传，可暂停续传
const CHUNK_THRESHOLD = 20 * 1024 * 1024
const CHUNK_SIZE = 5 * 1024 * 1024
const CHUNK_RETRY = 5
// 同时上传的文件数 / 单文件同时上传的分块数
const FILE_CONCURRENCY = 3
const CHUNK_CONCURRENCY = 3

const PRIORITY_RANK: Record<UploadPriority, number> = { high: 0, normal: 1, low: 2 }

// 一个文件本次上传的运行时句柄
interface Task {
  aborted: boolean
  requests: Set<{ abort: () => void }>
}

const abortTask = (task?: Task) => {
  if (!task) return
  task.aborted = true
  task.requests.forEach((r) => r.abort())
}

const sleep = (ms: number) => new Promise((resolve) => setTimeout(resolve, ms))

// 拆分文件名和扩展名
const splitExt = (filename: string) => {
  const dot = filename.lastIndexOf('.')
  return dot > 0 ? [filename.slice(0, dot), filename.slice(dot)] : [filename, '']
}

// SHA-256 十六进制：安全上下文（HTTPS/localhost）下用原生 WebCrypto，否则退回纯 JS 实现
const sha256Hex = async (data: Uint8Array): Promise<string> => {
  if (globalThis.crypto?.subtle) {
    const digest = await crypto.subtle.digest('SHA-256', data as BufferSource)
    return Array.from(new Uint8Array(digest), (b) => b.toString(16).padStart(2, '0')).join('')
  }
  return sha256(data)
}

// 文件标识：大小 + 修改时间 + 首/中/尾各 1MB 采样，不用读整个文件；同一文件重新加入可续传
const fileIdentifier = async (file: File): Promise<string> => {
  const sample = 1024 * 1024
  const offsets = [0, Math.floor(file.size / 2 - sample / 2), file.size - sample]
  const parts = [new TextEncoder().encode(`${file.size}|${file.lastModified}|`)]
  for (const offset of offsets) {
    const start = Math.max(0, offset)
    parts.push(new Uint8Array(await file.slice(start, start + sample).arrayBuffer()))
  }

  const combined = new Uint8Array(parts.reduce((sum, p) => sum + p.byteLength, 0))
  let cursor = 0
  for (const part of parts) {
    combined.set(part, cursor)
    cursor += part.byteLength
  }
  return sha256Hex(combined)
}

// 上传队列：全局单例，关闭弹窗不影响后台上传
export const useUploadStore = defineStore('upload', () => {
  const items = ref<UploadItem[]>([])
  const conflictPolicy = ref<ConflictPolicy>('ask')

  // 同名冲突询问弹窗的数据，由 UploadModal 渲染，非空即显示
  const conflictItems = ref<ConflictItem[]>([])
  let conflictResolver: ((resolved: ConflictItem[] | null) => void) | null = null

  const tasks = new Map<string, Task>()
  let seq = 0
  // 多次开始操作串行处理，避免同时弹出多个冲突询问
  let planChain: Promise<void> = Promise.resolve()
  // 本轮上传的完成/失败计数，全部结束时提示一次
  let roundDone = 0
  let roundError = 0

  const stats = computed(() => {
    const s = {
      total: items.value.length,
      pending: 0,
      waiting: 0,
      uploading: 0,
      paused: 0,
      done: 0,
      error: 0,
      skipped: 0,
      speed: 0,
    }
    for (const item of items.value) {
      s[item.status]++
      s.speed += item.speed
    }
    return s
  })

  // 进行中的数量（上传中 + 排队中）
  const activeCount = computed(() => stats.value.uploading + stats.value.waiting)

  const pick = (ids?: string[]) => {
    if (!ids) return items.value
    const set = new Set(ids)
    return items.value.filter((i) => set.has(i.id))
  }

  // ==================== 列表刷新 & 速度统计 ====================

  let refreshTimer: ReturnType<typeof setTimeout> | null = null
  const scheduleRefresh = () => {
    if (refreshTimer) return
    refreshTimer = setTimeout(() => {
      refreshTimer = null
      window.$bus.emit('file:refresh')
    }, 1000)
  }

  const lastLoaded = new Map<string, number>()
  let ticker: ReturnType<typeof setInterval> | null = null
  const tick = () => {
    for (const item of items.value) {
      if (item.status !== 'uploading') continue
      const prev = lastLoaded.get(item.id)
      if (prev !== undefined) {
        const instant = Math.max(0, item.loaded - prev)
        item.speed = item.speed ? Math.round(item.speed * 0.6 + instant * 0.4) : instant
      }
      lastLoaded.set(item.id, item.loaded)
    }
  }
  const startTicker = () => {
    if (!ticker) ticker = setInterval(tick, 1000)
  }
  const stopTicker = () => {
    if (ticker) clearInterval(ticker)
    ticker = null
  }

  // ==================== 传输 ====================

  // 发送请求并登记到任务上，暂停/取消时统一中止
  async function send(method: any, task: Task) {
    if (task.aborted) throw new Error('aborted')
    task.requests.add(method)
    try {
      return await method
    } finally {
      task.requests.delete(method)
    }
  }

  async function uploadWhole(item: UploadItem, task: Task) {
    item.loaded = 0
    const form = new FormData()
    form.append('path', item.target)
    form.append('file', item.file)
    form.append('force', String(item.force))
    const method = api.upload(form)
    method.onUpload(({ loaded }: { loaded: number }) => (item.loaded = loaded))
    await send(method, task)
  }

  async function uploadChunked(item: UploadItem, task: Task) {
    if (!item.hash) item.hash = await fileIdentifier(item.file)
    if (task.aborted) throw new Error('aborted')

    const dir = dirname(item.target)
    const fileName = getFilename(item.target)
    const chunkCount = Math.ceil(item.size / CHUNK_SIZE)
    const chunkLength = (index: number) => Math.min(CHUNK_SIZE, item.size - index * CHUNK_SIZE)
    const meta = {
      path: dir,
      file_name: fileName,
      file_hash: item.hash,
      chunk_count: chunkCount,
      chunk_size: CHUNK_SIZE,
      force: item.force,
    }

    // 服务端返回已有分块，续传时跳过
    const { uploaded_chunks } = await send(api.chunkStart(meta), task)
    const uploaded = new Set<number>(uploaded_chunks)

    // 进度 = 已完成分块 + 在途分块已发送字节
    let doneBytes = 0
    uploaded.forEach((index) => (doneBytes += chunkLength(index)))
    const inflight = new Map<number, number>()
    const report = () => {
      let sum = doneBytes
      inflight.forEach((v) => (sum += v))
      item.loaded = sum
    }
    report()

    const uploadChunk = async (index: number) => {
      const start = index * CHUNK_SIZE
      const blob = item.file.slice(start, start + chunkLength(index))
      const chunkHash = await sha256Hex(new Uint8Array(await blob.arrayBuffer()))
      for (let attempt = 1; ; attempt++) {
        if (task.aborted) throw new Error('aborted')
        const form = new FormData()
        form.append('path', dir)
        form.append('file_name', fileName)
        form.append('file_hash', item.hash)
        form.append('chunk_index', String(index))
        form.append('chunk_hash', chunkHash)
        form.append('file', blob)
        const method = api.chunkUpload(form)
        method.onUpload(({ loaded }: { loaded: number }) => {
          inflight.set(index, loaded)
          report()
        })
        try {
          await send(method, task)
          inflight.delete(index)
          doneBytes += blob.size
          report()
          return
        } catch (error) {
          inflight.delete(index)
          if (task.aborted || attempt >= CHUNK_RETRY) throw error
          await sleep(Math.min(1000 * 2 ** (attempt - 1), 8000))
        }
      }
    }

    const limit = pLimit(CHUNK_CONCURRENCY)
    const pending: number[] = []
    for (let i = 0; i < chunkCount; i++) {
      if (!uploaded.has(i)) pending.push(i)
    }
    await Promise.all(pending.map((index) => limit(() => uploadChunk(index))))

    if (task.aborted) throw new Error('aborted')
    await send(api.chunkFinish(meta), task)
  }

  // ==================== 调度 ====================

  async function run(item: UploadItem) {
    const task: Task = { aborted: false, requests: new Set() }
    tasks.set(item.id, task)
    item.status = 'uploading'
    item.error = ''
    startTicker()

    try {
      if (item.size > CHUNK_THRESHOLD) {
        await uploadChunked(item, task)
      } else {
        await uploadWhole(item, task)
      }
      item.loaded = item.size
      item.status = 'done'
      roundDone++
      scheduleRefresh()
    } catch (error) {
      // 暂停/取消触发的中止，状态已由操作方设置
      if (!task.aborted) {
        abortTask(task)
        item.status = 'error'
        item.error = error instanceof Error ? error.message : String(error)
        roundError++
      }
    } finally {
      tasks.delete(item.id)
      lastLoaded.delete(item.id)
      item.speed = 0
      if (tasks.size === 0) stopTicker()
      schedule()
      finishRound()
    }
  }

  // 按优先级取排队项填满并发槽位，同优先级先进先出
  function schedule() {
    while (tasks.size < FILE_CONCURRENCY) {
      const next = items.value
        .filter((i) => i.status === 'waiting' && i.planned && !tasks.has(i.id))
        .sort((a, b) => PRIORITY_RANK[a.priority] - PRIORITY_RANK[b.priority])[0]
      if (!next) break
      run(next)
    }
  }

  function finishRound() {
    if (tasks.size > 0 || items.value.some((i) => i.status === 'waiting')) return
    if (roundError > 0) {
      window.$message.warning($gettext('%{count} file(s) failed to upload', { count: roundError }))
    } else if (roundDone > 0) {
      window.$message.success($gettext('Upload completed'))
    }
    roundDone = 0
    roundError = 0
  }

  // ==================== 同名检查 ====================

  // 为冲突文件批量找不重名的新文件名（追加 -1 -2 …），每轮一次 exist 请求检查一批候选
  async function uniqueNames(conflicts: UploadItem[]): Promise<string[]> {
    const result: string[] = new Array(conflicts.length).fill('')
    const reserved = new Set(items.value.map((i) => i.target))
    const batch = 10
    for (let offset = 1; offset <= 1000 && result.includes(''); offset += batch) {
      const candidates: { idx: number; name: string; path: string }[] = []
      conflicts.forEach((item, idx) => {
        if (result[idx]) return
        const [base, ext] = splitExt(getFilename(item.target))
        for (let k = 0; k < batch; k++) {
          const name = `${base}-${offset + k}${ext}`
          candidates.push({ idx, name, path: joinPath(dirname(item.target), name) })
        }
      })
      let exists: boolean[] = []
      try {
        exists = await api.exist(candidates.map((c) => c.path))
      } catch {
        exists = []
      }
      candidates.forEach((c, j) => {
        if (!result[c.idx] && !exists[j] && !reserved.has(c.path)) {
          result[c.idx] = c.name
          reserved.add(c.path)
        }
      })
    }
    return result.map((name, idx) => {
      if (name) return name
      const [base, ext] = splitExt(getFilename(conflicts[idx]!.target))
      return `${base}-${Date.now()}${ext}`
    })
  }

  // 检查目标是否已存在，按策略处理冲突；询问策略下等待用户在弹窗中选择
  async function plan(targets: UploadItem[]) {
    const unplanned = targets.filter((i) => !i.planned)
    if (unplanned.length === 0) return

    if (conflictPolicy.value === 'overwrite') {
      unplanned.forEach((i) => {
        i.force = true
        i.planned = true
      })
      return
    }

    let exists: boolean[] = []
    try {
      exists = await api.exist(unplanned.map((i) => i.target))
    } catch {
      // 检查失败按不存在处理，由后端在上传时兜底报错
      exists = []
    }
    const conflicts = unplanned.filter((_, idx) => exists[idx])
    unplanned.forEach((i) => (i.planned = true))
    if (conflicts.length === 0) return

    if (conflictPolicy.value === 'skip') {
      conflicts.forEach((i) => (i.status = 'skipped'))
      return
    }

    const names = await uniqueNames(conflicts)
    if (conflictPolicy.value === 'rename') {
      conflicts.forEach((i, idx) => (i.target = joinPath(dirname(i.target), names[idx]!)))
      return
    }

    conflictItems.value = conflicts.map((i, idx) => ({
      id: i.id,
      name: i.name,
      newName: names[idx]!,
      action: 'rename',
    }))
    const resolved = await new Promise<ConflictItem[] | null>((resolve) => {
      conflictResolver = resolve
    })

    // 取消询问：冲突项退回待开始，其余不受影响
    if (!resolved) {
      conflicts.forEach((i) => {
        i.status = 'pending'
        i.planned = false
      })
      return
    }
    const byId = new Map(conflicts.map((i) => [i.id, i]))
    for (const c of resolved) {
      const item = byId.get(c.id)
      if (!item) continue
      if (c.action === 'skip') {
        item.status = 'skipped'
      } else if (c.action === 'rename') {
        item.target = joinPath(dirname(item.target), c.newName)
      } else {
        item.force = true
      }
    }
  }

  function resolveConflicts(resolved: ConflictItem[] | null) {
    conflictItems.value = []
    conflictResolver?.(resolved)
    conflictResolver = null
  }

  // ==================== 对外操作 ====================

  // 加入队列，不自动开始；同一目标路径的未完成项不重复加入
  function add(files: PickedFile[], dir: string) {
    const occupied = new Set(
      items.value.filter((i) => i.status !== 'done' && i.status !== 'skipped').map((i) => i.target),
    )
    let duplicated = 0
    for (const { file, name } of files) {
      const target = joinPath(dir, name)
      if (occupied.has(target)) {
        duplicated++
        continue
      }
      occupied.add(target)
      items.value.push({
        id: `${Date.now()}-${seq++}`,
        file: markRaw(file),
        name,
        size: file.size,
        target,
        status: 'pending',
        priority: 'normal',
        loaded: 0,
        speed: 0,
        error: '',
        force: false,
        planned: false,
        hash: '',
      })
    }
    if (duplicated > 0) {
      window.$message.info($gettext('%{count} file(s) already in the queue', { count: duplicated }))
    }
  }

  // 开始/继续/重试，不传 ids 则作用于全部
  function start(ids?: string[]) {
    const targets = pick(ids).filter((i) =>
      ['pending', 'paused', 'error', 'skipped'].includes(i.status),
    )
    if (targets.length === 0) return
    targets.forEach((i) => {
      // 失败/跳过的项重试时重新检查目标，失败可能已留下半截文件
      if (i.status === 'error' || i.status === 'skipped') i.planned = false
      i.status = 'waiting'
      i.error = ''
    })
    // 全部被跳过/取消询问时不会有 run 触发收尾，这里兜底
    planChain = planChain
      .then(() => plan(targets))
      .then(schedule)
      .then(finishRound)
  }

  // 暂停：分块上传保留已传分块，继续时从断点续传；整文件上传只能重头开始
  function pause(ids?: string[]) {
    for (const item of pick(ids)) {
      if (item.status !== 'waiting' && item.status !== 'uploading') continue
      item.status = 'paused'
      if (!item.hash) item.loaded = 0
      abortTask(tasks.get(item.id))
    }
  }

  // 取消/移除：中止传输并清理服务器上残留的临时分块
  function remove(ids?: string[]) {
    const removing = new Set(pick(ids).map((i) => i.id))
    for (const item of items.value) {
      if (!removing.has(item.id)) continue
      abortTask(tasks.get(item.id))
      if (item.hash && item.status !== 'done') {
        api
          .chunkCancel({
            path: dirname(item.target),
            file_name: getFilename(item.target),
            file_hash: item.hash,
          })
          .catch(() => {})
      }
    }
    items.value = items.value.filter((i) => !removing.has(i.id))
  }

  function setPriority(ids: string[] | undefined, priority: UploadPriority) {
    pick(ids).forEach((i) => (i.priority = priority))
  }

  function clearFinished() {
    remove(
      items.value.filter((i) => i.status === 'done' || i.status === 'skipped').map((i) => i.id),
    )
  }

  // 有上传进行中时离开页面需确认
  window.addEventListener('beforeunload', (e) => {
    if (activeCount.value === 0) return
    e.preventDefault()
    e.returnValue = ''
  })

  return {
    items,
    conflictPolicy,
    conflictItems,
    stats,
    activeCount,
    add,
    start,
    pause,
    remove,
    setPriority,
    clearFinished,
    resolveConflicts,
  }
})

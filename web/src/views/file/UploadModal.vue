<script setup lang="ts">
import { sha256 } from 'js-sha256'
import type { UploadCustomRequestOptions, UploadFileInfo, UploadInst } from 'naive-ui'
import pLimit from 'p-limit'
import { useGettext } from 'vue3-gettext'

import api from '@/api/panel/file'

const { $gettext } = useGettext()
const show = defineModel<boolean>('show', { type: Boolean, required: true })
const path = defineModel<string>('path', { type: String, required: true })

const props = defineProps<{
  initialFiles?: File[]
}>()

const upload = ref<UploadInst | null>(null)
const fileList = ref<UploadFileInfo[]>([])

// 覆盖上传选项
const forceOverwrite = ref(false)

// 文件数量阈值，超过此数量需要二次确认
const FILE_COUNT_THRESHOLD = 100
// 大文件阈值，超过此大小使用分块上传 (100MB)
const LARGE_FILE_THRESHOLD = 100 * 1024 * 1024
// 分块大小 (5MB)
const CHUNK_SIZE = 5 * 1024 * 1024
// 分块上传重试次数
const CHUNK_RETRY_COUNT = 10
// 并发上传数
const CONCURRENT_UPLOADS = 3

// 上传速度状态（每个文件独立）
interface UploadProgress {
  fileName: string
  speed: string
}
const uploadProgressMap = ref<Map<string, UploadProgress>>(new Map())

// 每个上传任务的状态
interface UploadTask {
  isCancelled: boolean
  activeRequests: { abort: () => void }[]
}

// 以文件唯一标识为 key 存储每个上传任务的状态
const uploadTasks = new Map<string, UploadTask>()

// 每个文件的上传计划：实际写入目录的名字（重命名时与原名不同）+ 是否覆盖
interface UploadPlan {
  uploadName: string
  force: boolean
}
const uploadPlanMap = new Map<string, UploadPlan>()

// 获取文件唯一标识
const getFileKey = (file: File) => `${file.name}-${file.size}-${file.lastModified}`

// 拼接目标完整路径
const buildTargetPath = (fileName: string) => {
  const base = path.value.endsWith('/') ? path.value.slice(0, -1) : path.value
  return `${base}/${fileName}`
}

// 读取单个文件的上传计划，无记录时按原名 + 不覆盖处理
const getUploadPlan = (file: File): UploadPlan => {
  return uploadPlanMap.get(getFileKey(file)) ?? { uploadName: file.name, force: false }
}

// 找一个目录里不存在、且本批次未占用的新文件名（追加 -1 -2 -3）
const generateUniqueName = async (fileName: string, reserved: Set<string>): Promise<string> => {
  const dot = fileName.lastIndexOf('.')
  const base = dot > 0 ? fileName.slice(0, dot) : fileName
  const ext = dot > 0 ? fileName.slice(dot) : ''

  const batchSize = 10
  let offset = 1
  for (let round = 0; round < 100; round++) {
    const candidates: string[] = []
    for (let i = 0; i < batchSize; i++) {
      candidates.push(`${base}-${offset + i}${ext}`)
    }
    let existsArr: boolean[] = []
    try {
      existsArr = await api.exist(candidates.map(buildTargetPath))
    } catch {
      existsArr = candidates.map(() => false)
    }
    for (let i = 0; i < batchSize; i++) {
      if (!existsArr[i] && !reserved.has(candidates[i])) {
        return candidates[i]
      }
    }
    offset += batchSize
  }
  // 兜底：极端情况下用纳秒级后缀避免阻塞
  return `${base}-${offset}${ext}`
}

// 冲突解决弹窗的状态
type ConflictAction = 'skip' | 'rename' | 'overwrite'
interface ConflictItem {
  file: File
  suggestedName: string
  action: ConflictAction
}
const conflictItems = ref<ConflictItem[]>([])
const conflictModalShow = ref(false)
let conflictResolver: ((items: ConflictItem[]) => void) | null = null

const setAllConflictAction = (action: ConflictAction) => {
  conflictItems.value.forEach((i) => (i.action = action))
}

const onConflictConfirm = () => {
  const items = conflictItems.value
  const renameItems = items.filter((i) => i.action === 'rename')

  // 文件名非空 + 无路径分隔符 + 不能用 . / ..
  for (const item of renameItems) {
    const name = item.suggestedName.trim()
    if (!name) {
      window.$message.error(
        $gettext('New name for "%{name}" cannot be empty', { name: item.file.name }),
      )
      return
    }
    if (name.includes('/') || name.includes('\\') || name === '.' || name === '..') {
      window.$message.error($gettext('Invalid new name "%{name}"', { name }))
      return
    }
    item.suggestedName = name
  }

  // 本批次内重命名后不能重名
  const seen = new Set<string>()
  for (const item of renameItems) {
    if (seen.has(item.suggestedName)) {
      window.$message.error(
        $gettext('Duplicate new name "%{name}"', { name: item.suggestedName }),
      )
      return
    }
    seen.add(item.suggestedName)
  }

  conflictModalShow.value = false
  conflictResolver?.(items)
  conflictResolver = null
}

const onConflictCancel = () => {
  const items = conflictItems.value.map((i) => ({ ...i, action: 'skip' as ConflictAction }))
  conflictModalShow.value = false
  conflictResolver?.(items)
  conflictResolver = null
}

// 上传前检查：批量查询存在性，对冲突文件统一弹一个窗，让用户逐个选择
const precheckFiles = async (files: File[]): Promise<File[]> => {
  if (files.length === 0) return []

  // 全局勾选覆盖：跳过询问，全部按覆盖处理
  if (forceOverwrite.value) {
    files.forEach((f) => uploadPlanMap.set(getFileKey(f), { uploadName: f.name, force: true }))
    return files
  }

  let existsArr: boolean[] = []
  try {
    existsArr = await api.exist(files.map((f) => buildTargetPath(f.name)))
  } catch {
    // 检查失败时按不存在处理，让后端在上传阶段兜底
    existsArr = files.map(() => false)
  }

  const accepted: File[] = []
  const conflicts: File[] = []
  for (let i = 0; i < files.length; i++) {
    if (existsArr[i]) {
      conflicts.push(files[i])
    } else {
      uploadPlanMap.set(getFileKey(files[i]), { uploadName: files[i].name, force: false })
      accepted.push(files[i])
    }
  }

  if (conflicts.length === 0) return accepted

  // 为每个冲突文件预算一个不冲突的重命名候选
  const reserved = new Set<string>()
  const items: ConflictItem[] = []
  for (const file of conflicts) {
    const suggestedName = await generateUniqueName(file.name, reserved)
    reserved.add(suggestedName)
    items.push({ file, suggestedName, action: 'rename' })
  }

  conflictItems.value = items
  conflictModalShow.value = true
  const resolved = await new Promise<ConflictItem[]>((resolve) => {
    conflictResolver = resolve
  })

  for (const item of resolved) {
    if (item.action === 'skip') continue
    const uploadName = item.action === 'rename' ? item.suggestedName : item.file.name
    const force = item.action === 'overwrite'
    uploadPlanMap.set(getFileKey(item.file), { uploadName, force })
    accepted.push(item.file)
  }
  return accepted
}

// 取消单个文件的上传
const cancelUpload = (file: File) => {
  const fileKey = getFileKey(file)
  const task = uploadTasks.get(fileKey)
  if (task) {
    task.isCancelled = true
    task.activeRequests.forEach((req) => req.abort())
    uploadTasks.delete(fileKey)
  }
}

// 取消所有上传
const cancelAllUploads = () => {
  uploadTasks.forEach((task) => {
    task.isCancelled = true
    task.activeRequests.forEach((req) => req.abort())
  })
  uploadTasks.clear()
}

// 计算文件标识符（快速，用于断点续传识别）
// 使用文件元数据 + 首尾采样计算，避免读取整个大文件
const calculateFileIdentifier = async (file: File): Promise<string> => {
  const sampleSize = 1024 * 1024 // 1MB samples

  // 读取首部
  const headChunk = file.slice(0, Math.min(sampleSize, file.size))
  const headBuffer = await headChunk.arrayBuffer()

  // 读取尾部（如果文件足够大）
  let tailBuffer: ArrayBuffer
  if (file.size > sampleSize * 2) {
    const tailChunk = file.slice(file.size - sampleSize, file.size)
    tailBuffer = await tailChunk.arrayBuffer()
  } else {
    tailBuffer = headBuffer
  }

  // 组合元数据
  const metadata = `${file.name}|${file.size}|${file.lastModified}`
  const metaBuffer = new TextEncoder().encode(metadata)

  // 合并所有数据计算hash
  const combined = new Uint8Array(
    metaBuffer.byteLength + headBuffer.byteLength + tailBuffer.byteLength,
  )
  combined.set(new Uint8Array(metaBuffer), 0)
  combined.set(new Uint8Array(headBuffer), metaBuffer.byteLength)
  combined.set(new Uint8Array(tailBuffer), metaBuffer.byteLength + headBuffer.byteLength)

  return sha256(combined)
}

// 计算分块SHA256
const calculateChunkHash = async (chunk: Blob): Promise<string> => {
  const buffer = await chunk.arrayBuffer()
  return sha256(new Uint8Array(buffer))
}

// 格式化速度显示
const formatSpeed = (bytesPerSecond: number): string => {
  if (bytesPerSecond < 1024) {
    return `${bytesPerSecond.toFixed(0)} B/s`
  } else if (bytesPerSecond < 1024 * 1024) {
    return `${(bytesPerSecond / 1024).toFixed(1)} KB/s`
  } else if (bytesPerSecond < 1024 * 1024 * 1024) {
    return `${(bytesPerSecond / 1024 / 1024).toFixed(1)} MB/s`
  } else {
    return `${(bytesPerSecond / 1024 / 1024 / 1024).toFixed(2)} GB/s`
  }
}

// 带重试的分块上传
const uploadChunkWithRetry = async (
  formData: FormData,
  chunkIndex: number,
  chunkSize: number,
  onChunkComplete: (size: number) => void,
  task: UploadTask,
): Promise<void> => {
  let lastError: Error | null = null
  for (let attempt = 1; attempt <= CHUNK_RETRY_COUNT; attempt++) {
    // 检查是否已取消
    if (task.isCancelled) {
      throw new DOMException('Upload cancelled', 'AbortError')
    }
    try {
      const method = api.chunkUpload(formData)
      task.activeRequests.push(method)
      try {
        await method
        onChunkComplete(chunkSize)
        return
      } finally {
        // 从活跃请求列表中移除
        const index = task.activeRequests.indexOf(method)
        if (index > -1) {
          task.activeRequests.splice(index, 1)
        }
      }
    } catch (error) {
      // 如果是取消错误，直接抛出
      if (task.isCancelled || (error as Error).message?.includes('abort')) {
        throw new DOMException('Upload cancelled', 'AbortError')
      }
      lastError = error as Error
      console.warn(
        `Chunk ${chunkIndex} upload failed (attempt ${attempt}/${CHUNK_RETRY_COUNT}):`,
        error,
      )
      if (attempt < CHUNK_RETRY_COUNT) {
        // 等待一段时间后重试，指数退避
        await new Promise((resolve) =>
          setTimeout(resolve, Math.min(1000 * Math.pow(2, attempt - 1), 10000)),
        )
      }
    }
  }
  throw new Error(
    `Chunk ${chunkIndex} upload failed after ${CHUNK_RETRY_COUNT} attempts: ${lastError?.message}`,
  )
}

// 分块上传
const chunkedUpload = async (
  file: File,
  onProgress: (e: { percent: number }) => void,
  onFinish: () => void,
  onError: () => void,
) => {
  // 创建此文件的上传任务
  const fileKey = getFileKey(file)
  const plan = getUploadPlan(file)
  const task: UploadTask = { isCancelled: false, activeRequests: [] }
  uploadTasks.set(fileKey, task)

  // 初始化进度显示
  uploadProgressMap.value.set(fileKey, { fileName: file.name, speed: '' })

  try {
    // 计算文件标识符（快速）
    onProgress({ percent: 0 })
    const fileHash = await calculateFileIdentifier(file)

    // 检查是否已取消
    if (task.isCancelled) {
      throw new DOMException('Upload cancelled', 'AbortError')
    }

    // 计算分块数量
    const chunkCount = Math.ceil(file.size / CHUNK_SIZE)

    // 开始分块上传（查询已上传的分块）
    const startMethod = api.chunkStart({
      path: path.value,
      file_name: plan.uploadName,
      file_hash: fileHash,
      chunk_count: chunkCount,
      force: plan.force,
    })
    task.activeRequests.push(startMethod)
    const startRes = await startMethod
    task.activeRequests = task.activeRequests.filter((r) => r !== startMethod)

    const uploadedChunks: Set<number> = new Set(startRes.uploaded_chunks)

    // 速度计算相关
    let uploadedBytes = uploadedChunks.size * CHUNK_SIZE
    let lastTime = Date.now()
    let lastBytes = uploadedBytes

    // 更新进度和速度
    const updateProgress = () => {
      const now = Date.now()
      const timeDiff = (now - lastTime) / 1000 // 秒
      if (timeDiff >= 0.5) {
        // 每0.5秒更新一次速度
        const bytesDiff = uploadedBytes - lastBytes
        const speed = bytesDiff / timeDiff
        uploadProgressMap.value.set(fileKey, { fileName: file.name, speed: formatSpeed(speed) })
        lastTime = now
        lastBytes = uploadedBytes
      }
      const percent = Math.ceil((uploadedBytes / file.size) * 100)
      onProgress({ percent: Math.min(percent, 99) }) // 最多99%，留1%给finish
    }

    // 分块完成回调
    const onChunkComplete = (size: number) => {
      uploadedBytes += size
      updateProgress()
    }

    // 构建待上传分块列表
    const pendingChunks: number[] = []
    for (let i = 0; i < chunkCount; i++) {
      if (!uploadedChunks.has(i)) {
        pendingChunks.push(i)
      }
    }

    // 并发上传单个分块
    const uploadChunk = async (chunkIndex: number) => {
      // 检查是否已取消
      if (task.isCancelled) {
        throw new DOMException('Upload cancelled', 'AbortError')
      }

      const start = chunkIndex * CHUNK_SIZE
      const end = Math.min(start + CHUNK_SIZE, file.size)
      const chunk = file.slice(start, end)
      const chunkSize = end - start

      // 计算分块hash
      const chunkHash = await calculateChunkHash(chunk)

      // 上传分块
      const formData = new FormData()
      formData.append('path', path.value)
      formData.append('file_name', plan.uploadName)
      formData.append('file_hash', fileHash)
      formData.append('chunk_index', chunkIndex.toString())
      formData.append('chunk_hash', chunkHash)
      formData.append('file', chunk)

      await uploadChunkWithRetry(formData, chunkIndex, chunkSize, onChunkComplete, task)
    }

    // 控制并发
    const limit = pLimit(CONCURRENT_UPLOADS)
    await Promise.all(pendingChunks.map((chunkIndex) => limit(() => uploadChunk(chunkIndex))))

    // 检查是否已取消
    if (task.isCancelled) {
      throw new DOMException('Upload cancelled', 'AbortError')
    }

    // 完成分块上传（合并）
    const finishMethod = api.chunkFinish({
      path: path.value,
      file_name: plan.uploadName,
      file_hash: fileHash,
      chunk_count: chunkCount,
      force: plan.force,
    })
    task.activeRequests.push(finishMethod)
    await finishMethod
    task.activeRequests = task.activeRequests.filter((r) => r !== finishMethod)

    onProgress({ percent: 100 })
    uploadProgressMap.value.delete(fileKey)
    uploadTasks.delete(fileKey)
    if (!task.isCancelled) {
      onFinish()
      window.$message.success($gettext('Upload %{ fileName } successful', { fileName: file.name }))
    }
    window.$bus.emit('file:refresh')
  } catch (error) {
    uploadProgressMap.value.delete(fileKey)
    uploadTasks.delete(fileKey)

    // 如果是取消错误，静默处理
    if ((error as Error).name === 'AbortError' || task.isCancelled) {
      console.log('Upload cancelled by user')
      return
    }

    console.error('Chunked upload error:', error)
    if (!task.isCancelled) {
      onError()
    }
  }
}

// 监听预拖入的文件
watch(
  () => props.initialFiles,
  async (files) => {
    if (files && files.length > 0) {
      // 如果文件数量超过阈值，弹窗确认
      if (files.length > FILE_COUNT_THRESHOLD) {
        window.$dialog.warning({
          title: $gettext('Confirm Upload'),
          content: $gettext(
            'You are about to upload %{count} files. This may take a while. Do you want to continue?',
            { count: files.length },
          ),
          positiveText: $gettext('Continue'),
          negativeText: $gettext('Cancel'),
          onPositiveClick: () => {
            addFilesToList(files, true)
          },
          onNegativeClick: () => {
            show.value = false
          },
        })
      } else {
        addFilesToList(files, true)
      }
    }
  },
  { immediate: true },
)

// 将文件添加到上传列表
const addFilesToList = async (files: File[], autoUpload: boolean = false) => {
  const accepted = await precheckFiles(files)
  if (accepted.length === 0) {
    fileList.value = []
    if (autoUpload) show.value = false
    return
  }

  fileList.value = accepted.map((file, index) => ({
    id: `dropped-${Date.now()}-${index}`,
    name: file.name,
    status: 'pending' as const,
    file: file,
  }))

  // 自动开始上传
  if (autoUpload) {
    nextTick(() => {
      upload.value?.submit()
    })
  }
}

// 监听弹窗关闭，清空文件列表并取消上传
watch(show, (val) => {
  if (!val) {
    cancelAllUploads()
    fileList.value = []
    uploadPlanMap.clear()
    // 主弹窗关闭时也兜底关掉冲突弹窗并放弃等待
    if (conflictResolver) {
      conflictResolver([])
      conflictResolver = null
    }
    conflictModalShow.value = false
    conflictItems.value = []
  }
})

const uploadRequest = ({ file, onFinish, onError, onProgress }: UploadCustomRequestOptions) => {
  const fileObj = file.file as File

  // 大文件使用分块上传
  if (fileObj.size > LARGE_FILE_THRESHOLD) {
    chunkedUpload(fileObj, onProgress, onFinish, onError)
    return
  }

  // 小文件使用普通上传
  const fileKey = getFileKey(fileObj)
  const plan = getUploadPlan(fileObj)
  const task: UploadTask = { isCancelled: false, activeRequests: [] }
  uploadTasks.set(fileKey, task)

  const formData = new FormData()
  formData.append('path', `${path.value}/${plan.uploadName}`)
  formData.append('file', fileObj)
  formData.append('force', plan.force.toString())

  const method = api.upload(formData)
  task.activeRequests.push(method)

  const { uploading } = useRequest(method)
    .onSuccess(() => {
      uploadTasks.delete(fileKey)
      if (!task.isCancelled) {
        onFinish()
        window.$bus.emit('file:refresh')
        window.$message.success(
          $gettext('Upload %{ fileName } successful', { fileName: file.name }),
        )
      }
    })
    .onError(() => {
      uploadTasks.delete(fileKey)
      if (!task.isCancelled) {
        onError()
      }
    })
    .onComplete(() => {
      stopWatch()
    })
  const stopWatch = watch(uploading, (progress) => {
    if (!task.isCancelled) {
      onProgress({ percent: Math.ceil((progress.loaded / progress.total) * 100) })
    }
  })
}

// 处理文件移除（取消上传）
const handleRemove = ({ file }: { file: UploadFileInfo }) => {
  if (file.file) {
    cancelUpload(file.file)
  }
}

// 处理文件选择变化：对新增文件做 precheck 与数量确认后再触发上传
const handleChange = async (data: { fileList: UploadFileInfo[] }) => {
  const existingIds = new Set(fileList.value.map((f) => f.id))
  const newInfos = data.fileList.filter((f) => !existingIds.has(f.id))
  const keptInfos = data.fileList.filter((f) => existingIds.has(f.id))

  // 没有新增（可能是移除场景），直接同步
  if (newInfos.length === 0) {
    fileList.value = keptInfos
    return
  }

  // 数量阈值确认
  if (newInfos.length > FILE_COUNT_THRESHOLD) {
    const confirmed = await new Promise<boolean>((resolve) => {
      window.$dialog.warning({
        title: $gettext('Confirm Upload'),
        content: $gettext(
          'You are about to upload %{count} files. This may take a while. Do you want to continue?',
          { count: newInfos.length },
        ),
        positiveText: $gettext('Continue'),
        negativeText: $gettext('Cancel'),
        onPositiveClick: () => resolve(true),
        onNegativeClick: () => resolve(false),
      })
    })
    if (!confirmed) {
      fileList.value = keptInfos
      return
    }
  }

  // 对新文件做存在性 precheck
  const newFiles = newInfos.map((info) => info.file).filter((f): f is File => !!f)
  const accepted = await precheckFiles(newFiles)
  const acceptedKeys = new Set(accepted.map(getFileKey))

  const filteredNewInfos = newInfos.filter(
    (info) => info.file && acceptedKeys.has(getFileKey(info.file as File)),
  )

  fileList.value = [...keptInfos, ...filteredNewInfos]

  if (filteredNewInfos.length > 0) {
    nextTick(() => {
      upload.value?.submit()
    })
  }
}
</script>

<template>
  <n-modal
    v-model:show="show"
    preset="card"
    :title="$gettext('Upload')"
    style="width: 60vw"
    size="huge"
    :bordered="false"
    :segmented="false"
  >
    <n-flex vertical>
      <!-- 覆盖上传选项 -->
      <n-checkbox v-model:checked="forceOverwrite">
        {{ $gettext('Overwrite existing files') }}
      </n-checkbox>
      <!-- 上传速度显示 -->
      <n-flex
        v-for="[key, progress] in uploadProgressMap"
        :key="key"
        justify="space-between"
        align="center"
        class="upload-speed-bar"
      >
        <NText>{{ progress.fileName }}</NText>
        <NText type="success">{{ progress.speed || $gettext('Preparing...') }}</NText>
      </n-flex>
      <n-upload
        ref="upload"
        v-model:file-list="fileList"
        multiple
        directory-dnd
        :default-upload="false"
        :custom-request="uploadRequest"
        @change="handleChange"
        @remove="handleRemove"
      >
        <n-upload-dragger>
          <div class="mb-3">
            <the-icon :size="60" icon="mdi:arrow-up-bold-box-outline" />
          </div>
          <NText text-xl> {{ $gettext('Click or drag files to this area to upload') }}</NText>
          <NP depth="3" m-10>
            {{
              $gettext('For large files, it is recommended to use SFTP and other methods to upload')
            }}
          </NP>
        </n-upload-dragger>
      </n-upload>
    </n-flex>
  </n-modal>

  <!-- 文件冲突处理弹窗：批量列出冲突文件，每个单独选 跳过/重命名/覆盖 -->
  <n-modal
    v-model:show="conflictModalShow"
    preset="card"
    :title="$gettext('Files already exist')"
    style="width: 60vw"
    size="huge"
    :bordered="false"
    :segmented="false"
    :mask-closable="false"
    :closable="false"
  >
    <n-flex vertical :size="16">
      <NText depth="3">
        {{
          $gettext(
            'The following files already exist in the target directory. Choose an action for each.',
          )
        }}
      </NText>
      <n-flex align="center" :size="8">
        <NText>{{ $gettext('Apply to all:') }}</NText>
        <n-button size="small" @click="setAllConflictAction('skip')">
          {{ $gettext('Skip') }}
        </n-button>
        <n-button size="small" @click="setAllConflictAction('rename')">
          {{ $gettext('Rename') }}
        </n-button>
        <n-button size="small" @click="setAllConflictAction('overwrite')">
          {{ $gettext('Overwrite') }}
        </n-button>
      </n-flex>
      <n-scrollbar style="max-height: 50vh">
        <n-flex vertical :size="8">
          <n-flex
            v-for="item in conflictItems"
            :key="item.file.name"
            align="center"
            justify="space-between"
            class="conflict-row"
          >
            <n-flex vertical :size="4" style="min-width: 0; flex: 1">
              <NText style="word-break: break-all">{{ item.file.name }}</NText>
              <n-flex
                v-if="item.action === 'rename'"
                align="center"
                :size="6"
                style="min-width: 0"
              >
                <NText depth="3" style="font-size: 12px">→</NText>
                <n-input
                  v-model:value="item.suggestedName"
                  size="small"
                  :placeholder="$gettext('New file name')"
                  style="flex: 1; min-width: 160px"
                />
              </n-flex>
            </n-flex>
            <n-radio-group v-model:value="item.action" size="small">
              <n-radio-button value="skip">{{ $gettext('Skip') }}</n-radio-button>
              <n-radio-button value="rename">{{ $gettext('Rename') }}</n-radio-button>
              <n-radio-button value="overwrite">{{ $gettext('Overwrite') }}</n-radio-button>
            </n-radio-group>
          </n-flex>
        </n-flex>
      </n-scrollbar>
      <n-flex justify="end" :size="8">
        <n-button @click="onConflictCancel">{{ $gettext('Cancel') }}</n-button>
        <n-button type="primary" @click="onConflictConfirm">{{ $gettext('Confirm') }}</n-button>
      </n-flex>
    </n-flex>
  </n-modal>
</template>

<style scoped lang="scss">
.upload-speed-bar {
  padding: 8px 12px;
  background: var(--n-color-embedded);
  border-radius: 4px;
  margin-bottom: 12px;
}

.conflict-row {
  padding: 8px 12px;
  background: var(--n-color-embedded);
  border-radius: 4px;
}
</style>

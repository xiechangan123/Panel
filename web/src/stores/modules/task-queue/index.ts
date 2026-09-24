export type QueueTaskStatus = 'waiting' | 'running' | 'finished' | 'failed' | 'canceled'

export interface QueueTask {
  id: number
  name: string
  status: QueueTaskStatus
  log: string // 日志路径，任务开始执行后才有
}

// 文件管理提交的后台任务，存 localStorage，关掉文件管理再打开也能接着跟踪
export const useTaskQueueStore = defineStore('task-queue', {
  state: () => ({
    tasks: [] as QueueTask[],
    finished: 0, // 本轮已完成的数量
    show: false,
    minimized: false,
  }),
  getters: {
    active: (state) => state.tasks.filter((t) => t.status === 'waiting' || t.status === 'running'),
  },
  actions: {
    // 上一轮留下的失败任务已经提示过，加新任务时一并清掉
    add({ id, name, status, log }: QueueTask) {
      if (!this.active.length) this.finished = 0
      this.tasks = [...this.active, { id, name, status, log }]
      this.open()
    },
    open() {
      this.show = true
      this.minimized = false
    },
    // 用查询结果更新进行中的任务：完成和取消的移出队列，失败的留着看日志，查不到的说明已在任务列表里删除
    sync(ids: number[], fresh: QueueTask[]) {
      const map = new Map(fresh.map((t) => [t.id, t]))
      this.tasks = this.tasks.flatMap((t) => {
        const f = map.get(t.id)
        if (!ids.includes(t.id)) return [t]
        if (!f || f.status === 'canceled') return []
        if (f.status === 'finished') {
          this.finished++
          return []
        }
        return [{ ...t, status: f.status, log: f.log }]
      })
    },
  },
  persist: true,
})

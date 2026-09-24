<script setup lang="ts">
import type { TagProps } from 'naive-ui'
import { useGettext } from 'vue3-gettext'

import task from '@/api/panel/task'
import DraggableWindow from '@/components/common/DraggableWindow.vue'
import RealtimeLog from '@/components/common/RealtimeLog.vue'
import { useConfirm } from '@/components/system/composables/useConfirm'
import { type QueueTask, type QueueTaskStatus, useTaskQueueStore } from '@/stores'

const { $gettext, $ngettext } = useGettext()
const { confirmAction } = useConfirm()
const queue = useTaskQueueStore()

const title = computed(() => `${$gettext('Task Queue')} (${queue.tasks.length})`)

const statusMeta = computed<Record<QueueTaskStatus, { label: string; type: TagProps['type'] }>>(
  () => ({
    waiting: { label: $gettext('Waiting'), type: 'default' },
    running: { label: $gettext('Running'), type: 'info' },
    finished: { label: $gettext('Completed'), type: 'success' },
    failed: { label: $gettext('Failed'), type: 'error' },
    canceled: { label: $gettext('Canceled'), type: 'warning' },
  }),
)

const collapsed = ref(true)

// 没手动选过就跟随正在执行的任务
const selectedId = ref<number>()
const current = computed(
  () =>
    queue.tasks.find((t) => t.id === selectedId.value) ??
    queue.tasks.find((t) => t.status === 'running') ??
    queue.tasks[0],
)

const handleCancel = async (item: QueueTask) => {
  const ok = await confirmAction({
    title: $gettext('Cancel Task'),
    content: $gettext('Are you sure you want to cancel task %{ name }?', { name: item.name }),
  })
  if (ok) useRequest(task.cancel(item.id))
}

// 全部结束时队列里只剩失败的，留着看日志，否则关闭窗口
const settle = () => {
  if (queue.tasks.length) {
    const n = queue.tasks.length
    window.$message.error($ngettext('%{ n } task failed', '%{ n } tasks failed', n, { n }))
    return
  }
  const done = queue.finished
  queue.$reset()
  if (done) window.$message.success($gettext('All tasks completed'))
}

const poll = async () => {
  const ids = queue.active.map((t) => t.id)
  if (!ids.length) return
  const fresh: QueueTask[] | null = await task.query(ids).catch(() => null)
  if (!fresh) return
  queue.sync(ids, fresh)
  // 有任务结束就刷新，结果文件随之出现
  if (queue.active.length < ids.length) window.$bus.emit('file:refresh')
  if (!queue.active.length) settle()
}

const { pause, resume } = useTimeoutPoll(poll, 1000, { immediate: false, immediateCallback: true })
watch(
  () => queue.active.length > 0,
  (busy) => (busy ? resume() : pause()),
  { immediate: true },
)

// 只剩失败任务时关闭窗口就清空队列，还有任务在跑则只是隐藏
watch(
  () => queue.show,
  (show) => {
    if (!show && !queue.active.length) queue.$reset()
  },
)
</script>

<template>
  <DraggableWindow
    v-model:show="queue.show"
    v-model:minimized="queue.minimized"
    :title="title"
    :default-width="960"
    :default-height="560"
    :min-width="640"
    :min-height="360"
  >
    <template #icon>
      <i-mdi-sync v-if="queue.active.length" class="animate-spin" />
      <i-mdi-alert-circle-outline v-else class="text-error" />
    </template>
    <n-layout has-sider class="h-full !bg-transparent">
      <n-layout-sider
        v-model:collapsed="collapsed"
        :width="200"
        :collapsed-width="16"
        collapse-mode="width"
        show-trigger="arrow-circle"
        bordered
        :native-scrollbar="false"
        content-class="p-2"
        class="!bg-transparent"
      >
        <n-flex v-show="!collapsed" vertical :size="4">
          <div
            v-for="item in queue.tasks"
            :key="item.id"
            class="hover-bg px-3 py-2 rounded cursor-pointer"
            :class="{ 'bg-bg-subtle': item.id === current?.id }"
            @click="selectedId = item.id"
          >
            <n-ellipsis class="w-full">{{ item.name }}</n-ellipsis>
            <n-flex align="center" justify="space-between" class="mt-1">
              <n-tag size="small" :bordered="false" :type="statusMeta[item.status].type">
                {{ statusMeta[item.status].label }}
              </n-tag>
              <n-button
                v-if="item.status === 'waiting' || item.status === 'running'"
                size="tiny"
                quaternary
                type="error"
                @click.stop="handleCancel(item)"
              >
                {{ $gettext('Cancel') }}
              </n-button>
            </n-flex>
          </div>
        </n-flex>
      </n-layout-sider>
      <n-layout-content class="!bg-transparent" content-class="h-full p-3 flex">
        <realtime-log
          v-if="current?.log"
          :path="current.log"
          class="flex-1 min-w-0 !h-auto !min-h-0"
        />
        <n-empty v-else :description="$gettext('Waiting')" class="flex-1 justify-center" />
      </n-layout-content>
    </n-layout>
  </DraggableWindow>
</template>

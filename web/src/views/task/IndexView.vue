<script setup lang="ts">
defineOptions({
  name: 'task-index'
})

import TheIcon from '@/components/custom/TheIcon.vue'
import CreateModal from '@/views/task/CreateModal.vue'
import CronView from '@/views/task/CronView.vue'
import SystemView from '@/views/task/SystemView.vue'
import TaskView from '@/views/task/TaskView.vue'
import { NButton } from 'naive-ui'
import { useGettext } from 'vue3-gettext'

const { $gettext } = useGettext()
const current = ref('cron')

const create = ref(false)
</script>

<template>
  <common-page show-footer>
    <template #action>
      <n-button v-if="current == 'cron'" type="primary" @click="create = true">
        <the-icon :size="18" icon="material-symbols:add" />
        {{ $gettext('Create Task') }}
      </n-button>
    </template>
    <n-tabs v-model:value="current" type="line" animated>
      <n-tab-pane name="cron" :tab="$gettext('Scheduled Tasks')">
        <cron-view />
      </n-tab-pane>
      <n-tab-pane name="system" :tab="$gettext('System Processes')">
        <system-view />
      </n-tab-pane>
      <n-tab-pane name="task" :tab="$gettext('Panel Tasks')">
        <task-view />
      </n-tab-pane>
    </n-tabs>
  </common-page>
  <create-modal v-model:show="create" />
</template>

<style scoped lang="scss"></style>

<script setup lang="ts">
defineOptions({
  name: 'task-index'
})

import CreateModal from '@/views/task/CreateModal.vue'
import CronView from '@/views/task/CronView.vue'
import TaskView from '@/views/task/TaskView.vue'
import { NButton } from 'naive-ui'
import { useGettext } from 'vue3-gettext'

const { $gettext } = useGettext()
const route = useRoute()
const current = ref(route.query.tab === 'task' ? 'task' : 'cron')

const create = ref(false)
const cronViewRef = ref<InstanceType<typeof CronView>>()
</script>

<template>
  <common-page show-header show-footer>
    <template #tabbar>
      <n-tabs v-model:value="current" animated>
        <n-tab name="cron" :tab="$gettext('Scheduled Tasks')" />
        <n-tab name="task" :tab="$gettext('Panel Tasks')" />
      </n-tabs>
    </template>
    <n-flex vertical>
      <n-flex v-if="current === 'cron'">
        <n-button type="primary" @click="create = true">
          {{ $gettext('Create Task') }}
        </n-button>
        <delete-confirm @positive-click="cronViewRef?.bulkDelete">
          <template #trigger>
            <n-button type="error" :disabled="!cronViewRef?.selectedRowKeys?.length" ghost>
              {{ $gettext('Delete') }}
            </n-button>
          </template>
          {{ $gettext('Are you sure you want to delete the selected tasks?') }}
        </delete-confirm>
      </n-flex>
      <cron-view v-if="current === 'cron'" ref="cronViewRef" />
      <task-view v-if="current === 'task'" />
    </n-flex>
  </common-page>
  <create-modal v-model:show="create" mode="create" />
</template>

<style scoped lang="scss"></style>

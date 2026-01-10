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
      <n-flex>
        <n-button v-if="current == 'cron'" type="primary" @click="create = true">
          {{ $gettext('Create Task') }}
        </n-button>
      </n-flex>
      <cron-view v-if="current === 'cron'" />
      <task-view v-if="current === 'task'" />
    </n-flex>
  </common-page>
  <create-modal v-model:show="create" />
</template>

<style scoped lang="scss"></style>

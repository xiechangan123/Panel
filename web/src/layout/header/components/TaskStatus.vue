<script lang="ts" setup>
import task from '@/api/panel/task'
import { useGettext } from 'vue3-gettext'

const { $gettext } = useGettext()
const router = useRouter()

const { data } = useRequest(() => task.status(), { initialData: { task: false } })

const goToTask = () => {
  router.push({ path: '/task', query: { tab: 'task' } })
}
</script>

<template>
  <n-tooltip trigger="hover">
    <template #trigger>
      <n-icon mr-20 cursor-pointer size="20" @click="goToTask">
        <i-mdi-sync v-if="data.task" class="animate-spin" />
        <i-mdi-checkbox-outline v-else />
      </n-icon>
    </template>
    {{ data.task ? $gettext('Tasks Running') : $gettext('Panel Tasks') }}
  </n-tooltip>
</template>

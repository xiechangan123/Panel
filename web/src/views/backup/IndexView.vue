<script setup lang="ts">
defineOptions({
  name: 'backup-index'
})

import dashboard from '@/api/panel/dashboard'
import ListView from '@/views/backup/ListView.vue'
import { useGettext } from 'vue3-gettext'

const { $gettext } = useGettext()
const currentTab = ref('website')

const { data: installedDbAndPhp } = useRequest(dashboard.installedDbAndPhp, {
  initialData: {
    db: [
      {
        label: '',
        value: ''
      }
    ]
  }
})

const mySQLInstalled = computed(() => {
  return installedDbAndPhp.value.db.find((item: any) => item.value === 'mysql')
})

const postgreSQLInstalled = computed(() => {
  return installedDbAndPhp.value.db.find((item: any) => item.value === 'postgresql')
})
</script>

<template>
  <common-page show-header show-footer>
    <template #tabbar>
      <n-tabs v-model:value="currentTab" animated>
        <n-tab name="website" :tab="$gettext('Website')" />
        <n-tab v-if="mySQLInstalled" name="mysql" tab="MySQL" />
        <n-tab v-if="postgreSQLInstalled" name="postgres" tab="PostgreSQL" />
      </n-tabs>
    </template>
    <list-view v-model:type="currentTab" />
  </common-page>
</template>

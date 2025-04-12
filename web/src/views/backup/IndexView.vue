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
  <common-page show-footer>
    <n-flex vertical>
      <n-tabs v-model:value="currentTab" type="line" animated>
        <n-tab-pane name="website" :tab="$gettext('Website')">
          <list-view v-model:type="currentTab" />
        </n-tab-pane>
        <n-tab-pane v-if="mySQLInstalled" name="mysql" tab="MySQL">
          <list-view v-model:type="currentTab" />
        </n-tab-pane>
        <n-tab-pane v-if="postgreSQLInstalled" name="postgres" tab="PostgreSQL">
          <list-view v-model:type="currentTab" />
        </n-tab-pane>
      </n-tabs>
    </n-flex>
  </common-page>
</template>

<script setup lang="ts">
defineOptions({
  name: 'backup-index'
})

import home from '@/api/panel/home'
import ListView from '@/views/backup/ListView.vue'
import AccountView from '@/views/backup/AccountView.vue'
import { useGettext } from 'vue3-gettext'

const { $gettext } = useGettext()
const currentTab = ref('website')

const { data: installedEnvironment } = useRequest(home.installedEnvironment, {
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
  return installedEnvironment.value.db.find((item: any) => item.value === 'mysql')
})

const postgreSQLInstalled = computed(() => {
  return installedEnvironment.value.db.find((item: any) => item.value === 'postgresql')
})
</script>

<template>
  <common-page show-header show-footer>
    <template #tabbar>
      <n-tabs v-model:value="currentTab" animated>
        <n-tab name="website" :tab="$gettext('Website')" />
        <n-tab v-if="mySQLInstalled" name="mysql" tab="MySQL" />
        <n-tab v-if="postgreSQLInstalled" name="postgres" tab="PostgreSQL" />
        <n-tab name="account" :tab="$gettext('Account')" />
      </n-tabs>
    </template>
    <list-view v-if="currentTab !== 'account'" v-model:type="currentTab" />
    <account-view v-else />
  </common-page>
</template>

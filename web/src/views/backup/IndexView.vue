<script setup lang="ts">
defineOptions({
  name: 'backup-index',
})

import { useGettext } from 'vue3-gettext'

import app from '@/api/panel/app'
import home from '@/api/panel/home'
import ListView from '@/views/backup/ListView.vue'
import StorageView from '@/views/backup/StorageView.vue'

const { $gettext } = useGettext()
const currentTab = ref('website')

const { data: installedEnvironment } = useRequest(home.installedEnvironment, {
  initialData: {
    db: [
      {
        label: '',
        value: '',
      },
    ],
  },
})

const mySQLInstalled = computed(() => {
  return installedEnvironment.value.db.find((item: any) => item.value === 'mysql')
})

const postgreSQLInstalled = computed(() => {
  return installedEnvironment.value.db.find((item: any) => item.value === 'postgresql')
})

const clickHouseInstalled = computed(() => {
  return installedEnvironment.value.db.find((item: any) => item.value === 'clickhouse')
})

const redisInstalled = ref(false)
const valkeyInstalled = ref(false)
useRequest(app.isInstalled('redis')).onSuccess(({ data }) => {
  redisInstalled.value = !!data
})
useRequest(app.isInstalled('valkey')).onSuccess(({ data }) => {
  valkeyInstalled.value = !!data
})
</script>

<template>
  <PageContainer :show-footer="true">
    <template #tabs>
      <n-tabs v-model:value="currentTab" animated>
        <n-tab name="website" :tab="$gettext('Website')" />
        <n-tab v-if="mySQLInstalled" name="mysql" tab="MySQL" />
        <n-tab v-if="postgreSQLInstalled" name="postgresql" tab="PostgreSQL" />
        <n-tab v-if="clickHouseInstalled" name="clickhouse" tab="ClickHouse" />
        <n-tab v-if="redisInstalled" name="redis" tab="Redis" />
        <n-tab v-if="valkeyInstalled" name="valkey" tab="Valkey" />
        <n-tab name="storage" :tab="$gettext('Storage')" />
      </n-tabs>
    </template>
    <list-view v-if="currentTab !== 'storage'" v-model:type="currentTab" />
    <storage-view v-else />
  </PageContainer>
</template>

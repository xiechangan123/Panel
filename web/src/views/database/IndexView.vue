<script setup lang="ts">
defineOptions({
  name: 'database-index',
})

import { NButton } from 'naive-ui'
import { useGettext } from 'vue3-gettext'

import pgadmin from '@/api/apps/pgadmin'
import phpmyadmin from '@/api/apps/phpmyadmin'
import app from '@/api/panel/app'
import database from '@/api/panel/database'
import CreateDatabaseModal from '@/views/database/CreateDatabaseModal.vue'
import CreateServerModal from '@/views/database/CreateServerModal.vue'
import CreateUserModal from '@/views/database/CreateUserModal.vue'
import DatabaseList from '@/views/database/DatabaseList.vue'
import ElasticsearchDataView from '@/views/database/ElasticsearchDataView.vue'
import RedisDataView from '@/views/database/RedisDataView.vue'
import ServerList from '@/views/database/ServerList.vue'
import UserList from '@/views/database/UserList.vue'

const { $gettext } = useGettext()
const currentTab = ref('mysql')

const createDatabaseModalShow = ref(false)
const createUserModalShow = ref(false)
const createServerModalShow = ref(false)

const phpMyAdminInstalled = ref(false)
const phpMyAdminLoading = ref(false)
const mysqlServers = ref<any[]>([])

const pgAdminInstalled = ref(false)
const pgAdminLoading = ref(false)
const postgresqlServers = ref<any[]>([])

useRequest(app.isInstalled('phpmyadmin')).onSuccess(({ data }: any) => {
  phpMyAdminInstalled.value = data
})
useRequest(app.isInstalled('pgadmin')).onSuccess(({ data }: any) => {
  pgAdminInstalled.value = data
})

// 切换标签页时刷新对应类型的可用服务器列表
watch(
  currentTab,
  (tab) => {
    if (tab === 'mysql') {
      useRequest(database.serverList(1, 10000, 'mysql')).onSuccess(({ data }: any) => {
        mysqlServers.value = data.items || []
      })
    } else if (tab === 'postgresql') {
      useRequest(database.serverList(1, 10000, 'postgresql')).onSuccess(({ data }: any) => {
        postgresqlServers.value = data.items || []
      })
    }
  },
  { immediate: true },
)

const serverOptions = computed(() =>
  mysqlServers.value.map((item: any) => ({ label: item.name, key: item.id })),
)

const postgresqlServerOptions = computed(() =>
  postgresqlServers.value.map((item: any) => ({ label: item.name, key: item.id })),
)

const handlePhpMyAdmin = (serverID: number) => {
  // 预先打开空白窗口,避免异步回调中 window.open 被浏览器拦截
  const win = window.open('about:blank', '_blank')
  phpMyAdminLoading.value = true
  useRequest(phpmyadmin.login(serverID))
    .onSuccess(({ data }: any) => {
      const url = `http://${window.location.hostname}:${data.port}/${data.path}/`
      if (win) {
        win.location.href = url
      } else {
        window.open(url, '_blank')
      }
    })
    .onError(() => {
      win?.close()
    })
    .onComplete(() => {
      phpMyAdminLoading.value = false
    })
}

const handlePgAdmin = (serverID: number) => {
  // 预先打开空白窗口,避免异步回调中 window.open 被浏览器拦截
  const win = window.open('about:blank', '_blank')
  pgAdminLoading.value = true
  useRequest(pgadmin.login(serverID))
    .onSuccess(({ data }: any) => {
      const url = `http://${window.location.hostname}:${data.port}/`
      if (win) {
        win.location.href = url
      } else {
        window.open(url, '_blank')
      }
    })
    .onError(() => {
      win?.close()
    })
    .onComplete(() => {
      pgAdminLoading.value = false
    })
}
</script>

<template>
  <PageContainer :show-footer="true">
    <template #tabs>
      <n-tabs v-model:value="currentTab" animated>
        <n-tab name="mysql" tab="MySQL" />
        <n-tab name="postgresql" tab="PostgreSQL" />
        <n-tab name="clickhouse" tab="ClickHouse" />
        <n-tab name="mongodb" tab="MongoDB" />
        <n-tab name="sqlite" tab="SQLite" />
        <n-tab name="elasticsearch" tab="Elasticsearch" />
        <n-tab name="redis" tab="Redis" />
        <n-tab name="user" :tab="$gettext('User')" />
        <n-tab name="server" :tab="$gettext('Server')" />
      </n-tabs>
    </template>
    <n-flex vertical>
      <n-flex v-if="!['redis', 'elasticsearch'].includes(currentTab)">
        <n-button
          v-if="['mysql', 'postgresql', 'clickhouse', 'mongodb'].includes(currentTab)"
          type="primary"
          @click="createDatabaseModalShow = true"
        >
          {{ $gettext('Create Database') }}
        </n-button>
        <template v-if="currentTab === 'mysql' && phpMyAdminInstalled && mysqlServers.length > 0">
          <n-dropdown
            v-if="mysqlServers.length > 1"
            :options="serverOptions"
            trigger="click"
            @select="handlePhpMyAdmin"
          >
            <n-button :loading="phpMyAdminLoading" :disabled="phpMyAdminLoading">
              phpMyAdmin
            </n-button>
          </n-dropdown>
          <n-button
            v-else
            :loading="phpMyAdminLoading"
            :disabled="phpMyAdminLoading"
            @click="handlePhpMyAdmin(mysqlServers[0].id)"
          >
            phpMyAdmin
          </n-button>
        </template>
        <template
          v-if="currentTab === 'postgresql' && pgAdminInstalled && postgresqlServers.length > 0"
        >
          <n-dropdown
            v-if="postgresqlServers.length > 1"
            :options="postgresqlServerOptions"
            trigger="click"
            @select="handlePgAdmin"
          >
            <n-button :loading="pgAdminLoading" :disabled="pgAdminLoading"> pgAdmin </n-button>
          </n-dropdown>
          <n-button
            v-else
            :loading="pgAdminLoading"
            :disabled="pgAdminLoading"
            @click="handlePgAdmin(postgresqlServers[0].id)"
          >
            pgAdmin
          </n-button>
        </template>
        <n-button v-if="currentTab === 'user'" type="primary" @click="createUserModalShow = true">
          {{ $gettext('Create User') }}
        </n-button>
        <n-button
          v-if="currentTab === 'server'"
          type="primary"
          @click="createServerModalShow = true"
        >
          {{ $gettext('Add Server') }}
        </n-button>
      </n-flex>
      <database-list v-if="currentTab === 'mysql'" type="mysql" />
      <database-list v-if="currentTab === 'postgresql'" type="postgresql" />
      <database-list v-if="currentTab === 'clickhouse'" type="clickhouse" />
      <database-list v-if="currentTab === 'mongodb'" type="mongodb" />
      <database-list v-if="currentTab === 'sqlite'" type="sqlite" />
      <elasticsearch-data-view v-if="currentTab === 'elasticsearch'" type="elasticsearch" />
      <redis-data-view v-if="currentTab === 'redis'" />
      <user-list v-if="currentTab === 'user'" />
      <server-list v-if="currentTab === 'server'" />
    </n-flex>
  </PageContainer>
  <create-database-modal v-model:show="createDatabaseModalShow" :type="currentTab" />
  <create-user-modal v-model:show="createUserModalShow" />
  <create-server-modal v-model:show="createServerModalShow" />
</template>

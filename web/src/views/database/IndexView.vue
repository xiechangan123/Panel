<script setup lang="ts">
defineOptions({
  name: 'database-index'
})

import CreateDatabaseModal from '@/views/database/CreateDatabaseModal.vue'
import CreateServerModal from '@/views/database/CreateServerModal.vue'
import CreateUserModal from '@/views/database/CreateUserModal.vue'
import DatabaseList from '@/views/database/DatabaseList.vue'
import RedisDataView from '@/views/database/RedisDataView.vue'
import ServerList from '@/views/database/ServerList.vue'
import UserList from '@/views/database/UserList.vue'
import { NButton } from 'naive-ui'
import { useGettext } from 'vue3-gettext'

const { $gettext } = useGettext()
const currentTab = ref('mysql')

const createDatabaseModalShow = ref(false)
const createUserModalShow = ref(false)
const createServerModalShow = ref(false)
</script>

<template>
  <common-page show-header show-footer>
    <template #tabbar>
      <n-tabs v-model:value="currentTab" animated>
        <n-tab name="mysql" tab="MySQL" />
        <n-tab name="postgresql" tab="PostgreSQL" />
        <n-tab name="redis" tab="Redis" />
        <n-tab name="user" :tab="$gettext('User')" />
        <n-tab name="server" :tab="$gettext('Server')" />
      </n-tabs>
    </template>
    <n-flex vertical>
      <n-flex v-if="currentTab !== 'redis'">
        <n-button
          v-if="currentTab === 'mysql' || currentTab === 'postgresql'"
          type="primary"
          @click="createDatabaseModalShow = true"
        >
          {{ $gettext('Create Database') }}
        </n-button>
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
      <redis-data-view v-if="currentTab === 'redis'" />
      <user-list v-if="currentTab === 'user'" />
      <server-list v-if="currentTab === 'server'" />
    </n-flex>
  </common-page>
  <create-database-modal v-model:show="createDatabaseModalShow" :type="currentTab" />
  <create-user-modal v-model:show="createUserModalShow" />
  <create-server-modal v-model:show="createServerModalShow" />
</template>

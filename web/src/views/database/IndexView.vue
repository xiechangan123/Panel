<script setup lang="ts">
defineOptions({
  name: 'database-index'
})

import CreateDatabaseModal from '@/views/database/CreateDatabaseModal.vue'
import CreateServerModal from '@/views/database/CreateServerModal.vue'
import CreateUserModal from '@/views/database/CreateUserModal.vue'
import DatabaseList from '@/views/database/DatabaseList.vue'
import ServerList from '@/views/database/ServerList.vue'
import UserList from '@/views/database/UserList.vue'
import { NButton } from 'naive-ui'
import { useGettext } from 'vue3-gettext'

const { $gettext } = useGettext()
const currentTab = ref('database')

const createDatabaseModalShow = ref(false)
const createUserModalShow = ref(false)
const createServerModalShow = ref(false)
</script>

<template>
  <common-page show-header show-footer>
    <template #tabbar>
      <n-tabs v-model:value="currentTab" animated>
        <n-tab name="database" :tab="$gettext('Database')" />
        <n-tab name="user" :tab="$gettext('User')" />
        <n-tab name="server" :tab="$gettext('Server')" />
      </n-tabs>
    </template>
    <n-flex vertical>
      <n-flex>
        <n-button
          v-if="currentTab === 'database'"
          type="primary"
          @click="createDatabaseModalShow = true"
        >
          <the-icon :size="18" icon="material-symbols:add" />
          {{ $gettext('Create Database') }}
        </n-button>
        <n-button v-if="currentTab === 'user'" type="primary" @click="createUserModalShow = true">
          <the-icon :size="18" icon="material-symbols:add" />
          {{ $gettext('Create User') }}
        </n-button>
        <n-button
          v-if="currentTab === 'server'"
          type="primary"
          @click="createServerModalShow = true"
        >
          <the-icon :size="18" icon="material-symbols:add" />
          {{ $gettext('Add Server') }}
        </n-button>
      </n-flex>
      <database-list v-if="currentTab === 'database'" />
      <user-list v-if="currentTab === 'user'" />
      <server-list v-if="currentTab === 'server'" />
    </n-flex>
  </common-page>
  <create-database-modal v-model:show="createDatabaseModalShow" />
  <create-user-modal v-model:show="createUserModalShow" />
  <create-server-modal v-model:show="createServerModalShow" />
</template>

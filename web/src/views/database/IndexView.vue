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
  <common-page show-footer>
    <template #action>
      <n-button
        v-if="currentTab === 'database'"
        type="primary"
        @click="createDatabaseModalShow = true"
      >
        <TheIcon :size="18" icon="material-symbols:add" />
        {{ $gettext('Create Database') }}
      </n-button>
      <n-button v-if="currentTab === 'user'" type="primary" @click="createUserModalShow = true">
        <TheIcon :size="18" icon="material-symbols:add" />
        {{ $gettext('Create User') }}
      </n-button>
      <n-button v-if="currentTab === 'server'" type="primary" @click="createServerModalShow = true">
        <TheIcon :size="18" icon="material-symbols:add" />
        {{ $gettext('Add Server') }}
      </n-button>
    </template>
    <n-flex vertical>
      <n-tabs v-model:value="currentTab" type="line" animated>
        <n-tab-pane name="database" :tab="$gettext('Database')">
          <database-list />
        </n-tab-pane>
        <n-tab-pane name="user" :tab="$gettext('User')">
          <user-list />
        </n-tab-pane>
        <n-tab-pane name="server" :tab="$gettext('Server')">
          <server-list />
        </n-tab-pane>
      </n-tabs>
    </n-flex>
  </common-page>
  <create-database-modal v-model:show="createDatabaseModalShow" />
  <create-user-modal v-model:show="createUserModalShow" />
  <create-server-modal v-model:show="createServerModalShow" />
</template>

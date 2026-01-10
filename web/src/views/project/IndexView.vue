<script lang="ts" setup>
defineOptions({
  name: 'project-index'
})

import CreateModal from '@/views/project/CreateModal.vue'
import ListView from '@/views/project/ListView.vue'

const currentTab = ref('all')

const createModal = ref(false)
</script>

<template>
  <common-page show-header show-footer>
    <template #tabbar>
      <n-tabs v-model:value="currentTab" animated>
        <n-tab name="all" :tab="$gettext('All')" />
        <n-tab name="general" :tab="$gettext('General')" />
        <n-tab name="php" :tab="$gettext('PHP')" />
        <n-tab name="java" :tab="$gettext('Java')" />
        <n-tab name="go" :tab="$gettext('Go')" />
        <n-tab name="python" :tab="$gettext('Python')" />
        <n-tab name="nodejs" :tab="$gettext('Node.js')" />
      </n-tabs>
    </template>
    <list-view v-model:type="currentTab" v-model:createModal="createModal" />
    <create-modal v-model:show="createModal" v-model:type="currentTab" />
  </common-page>
</template>

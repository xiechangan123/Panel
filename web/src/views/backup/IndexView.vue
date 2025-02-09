<script setup lang="ts">
defineOptions({
  name: 'backup-index'
})

import backup from '@/api/panel/backup'
import ListView from '@/views/backup/ListView.vue'
import { NButton, NInput } from 'naive-ui'

const currentTab = ref('website')
const createModal = ref(false)
const createModel = ref({
  target: '',
  path: ''
})

const handleCreate = () => {
  useRequest(
    backup.create(currentTab.value, createModel.value.target, createModel.value.path)
  ).onSuccess(() => {
    createModal.value = false
    window.$bus.emit('backup:refresh')
    window.$message.success('创建成功')
  })
}
</script>

<template>
  <common-page show-footer>
    <template #action>
      <n-button type="primary" @click="createModal = true">
        <TheIcon :size="18" icon="material-symbols:add" />
        创建备份
      </n-button>
    </template>
    <n-flex vertical>
      <n-alert type="info">该页面预计下版本移除！</n-alert>
      <n-tabs v-model:value="currentTab" type="line" animated>
        <n-tab-pane name="website" tab="网站">
          <list-view v-model:type="currentTab" />
        </n-tab-pane>
        <n-tab-pane name="mysql" tab="MySQL">
          <list-view v-model:type="currentTab" />
        </n-tab-pane>
        <n-tab-pane name="postgres" tab="PostgreSQL">
          <list-view v-model:type="currentTab" />
        </n-tab-pane>
      </n-tabs>
    </n-flex>
  </common-page>
  <n-modal
    v-model:show="createModal"
    preset="card"
    title="创建备份"
    style="width: 60vw"
    size="huge"
    :bordered="false"
    :segmented="false"
    @close="createModal = false"
  >
    <n-form :model="createModel">
      <n-form-item path="name" label="名称">
        <n-input
          v-model:value="createModel.target"
          type="text"
          @keydown.enter.prevent
          placeholder="输入网站/数据库名称"
        />
      </n-form-item>
      <n-form-item path="path" label="目录">
        <n-input
          v-model:value="createModel.path"
          type="text"
          @keydown.enter.prevent
          placeholder="留空使用默认路径"
        />
      </n-form-item>
    </n-form>
    <n-button type="info" block @click="handleCreate">提交</n-button>
  </n-modal>
</template>

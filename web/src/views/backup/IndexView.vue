<script setup lang="ts">
import website from '@/api/panel/website'

defineOptions({
  name: 'backup-index'
})

import app from '@/api/panel/app'
import backup from '@/api/panel/backup'
import dashboard from '@/api/panel/dashboard'
import ListView from '@/views/backup/ListView.vue'
import { NButton, NInput } from 'naive-ui'

const currentTab = ref('website')
const createModal = ref(false)
const createModel = ref({
  target: '',
  path: ''
})

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

const websites = ref<any>([])

const handleCreate = () => {
  useRequest(
    backup.create(currentTab.value, createModel.value.target, createModel.value.path)
  ).onSuccess(() => {
    createModal.value = false
    window.$bus.emit('backup:refresh')
    window.$message.success('创建成功')
  })
}

watch(currentTab, () => {
  if (currentTab.value === 'website') {
    createModel.value.target = websites.value[0]?.value
  } else {
    createModel.value.target = ''
  }
})

onMounted(() => {
  useRequest(app.isInstalled('nginx')).onSuccess(({ data }) => {
    if (data.installed) {
      useRequest(website.list(1, 10000)).onSuccess(({ data }: { data: any }) => {
        for (const item of data.items) {
          websites.value.push({
            label: item.name,
            value: item.name
          })
        }
        createModel.value.target = websites.value[0]?.value
      })
    }
  })
})
</script>

<template>
  <common-page show-footer>
    <template #action>
      <n-button v-if="currentTab == 'website'" type="primary" @click="createModal = true">
        <TheIcon :size="18" icon="material-symbols:add" />
        备份网站
      </n-button>
      <n-button v-if="currentTab == 'mysql'" type="primary" @click="createModal = true">
        <TheIcon :size="18" icon="material-symbols:add" />
        备份 MySQL
      </n-button>
      <n-button v-if="currentTab == 'postgres'" type="primary" @click="createModal = true">
        <TheIcon :size="18" icon="material-symbols:add" />
        备份 PostgreSQL
      </n-button>
    </template>
    <n-flex vertical>
      <n-tabs v-model:value="currentTab" type="line" animated>
        <n-tab-pane name="website" tab="网站">
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
      <n-form-item v-if="currentTab == 'website'" path="name" label="网站">
        <n-select v-model:value="createModel.target" :options="websites" placeholder="选择网站" />
      </n-form-item>
      <n-form-item v-if="currentTab != 'website'" path="name" label="数据库名">
        <n-input
          v-model:value="createModel.target"
          type="text"
          @keydown.enter.prevent
          placeholder="输入数据库名称"
        />
      </n-form-item>
      <n-form-item path="path" label="保存目录">
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

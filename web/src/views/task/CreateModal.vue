<script setup lang="ts">
import app from '@/api/panel/app'
import container from '@/api/panel/container'
import database from '@/api/panel/database'
import storage from '@/api/panel/backup-storage'
import cron from '@/api/panel/cron'
import home from '@/api/panel/home'
import website from '@/api/panel/website'
import CronSelector from '@/components/common/CronSelector.vue'
import { useGettext } from 'vue3-gettext'

const { $gettext } = useGettext()
const show = defineModel<boolean>('show', { type: Boolean, required: true })
const loading = ref(false)

const props = defineProps<{
  mode: 'create' | 'edit'
  editData?: any
}>()

const emit = defineEmits(['saved'])

const defaultModel = () => ({
  id: 0,
  name: '',
  type: 'shell',
  targets: [] as string[],
  keep: 1,
  sub_type: 'website',
  storage: 0,
  script:
    `#!/bin/bash\nexport PATH=/bin:/sbin:/usr/bin:/usr/sbin:/usr/local/bin:/usr/local/sbin:$PATH\n\n` +
    $gettext('# Enter your script content here') +
    `\n`,
  time: '*/30 * * * *'
})

const formModel = ref(defaultModel())

const websites = ref<any[]>([])
const storages = ref<any[]>([])
const containers = ref<any[]>([])
const databases = ref<any[]>([])

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

const containerInstalled = ref(false)

const handleSubmit = async () => {
  loading.value = true
  const data = { ...formModel.value }
  const request =
    props.mode === 'create' ? cron.create(data) : cron.update(data.id, data)

  useRequest(request)
    .onSuccess(() => {
      show.value = false
      window.$message.success(
        props.mode === 'create' ? $gettext('Created successfully') : $gettext('Modified successfully')
      )
      emit('saved')
      window.$bus.emit('task:refresh-cron')
    })
    .onComplete(() => {
      loading.value = false
    })
}

const generateTaskName = () => {
  const type = formModel.value.type
  const targets = formModel.value.targets

  if (type === 'backup') {
    const backupTypeMap: Record<string, string> = {
      website: $gettext('Backup Website'),
      mysql: $gettext('Backup MySQL'),
      postgres: $gettext('Backup PostgreSQL')
    }
    const prefix = backupTypeMap[formModel.value.sub_type] || $gettext('Backup')
    formModel.value.name = targets.length ? `${prefix} - ${targets.join(', ')}` : prefix
  } else if (type === 'cutoff') {
    const cutoffTypeMap: Record<string, string> = {
      website: $gettext('Log Rotation - Website'),
      container: $gettext('Log Rotation - Container')
    }
    const prefix = cutoffTypeMap[formModel.value.sub_type] || $gettext('Log Rotation')
    formModel.value.name = targets.length ? `${prefix} - ${targets.join(', ')}` : prefix
  }
}

// 监听类型变更自动生成名称
watch(
  () => [formModel.value.type, formModel.value.sub_type, formModel.value.targets],
  () => {
    if (props.mode === 'create') {
      generateTaskName()
    }
  },
  { deep: true }
)

const skipClearTargets = ref(false)

// 切换类型或子类型时重置 targets
watch(
  () => formModel.value.sub_type,
  () => {
    if (skipClearTargets.value) return
    formModel.value.targets = []
  }
)
watch(
  () => formModel.value.type,
  () => {
    if (skipClearTargets.value) return
    formModel.value.targets = []
  }
)

// 加载容器列表
const loadContainers = () => {
  useRequest(container.containerList(1, 10000)).onSuccess(({ data }: { data: any }) => {
    containers.value = (data.items || []).map((item: any) => ({
      label: item.name,
      value: item.name
    }))
  })
}

// 加载数据库列表
const loadDatabases = (type: string) => {
  useRequest(database.list(1, 10000, type)).onSuccess(({ data }: { data: any }) => {
    databases.value = (data.items || []).map((item: any) => ({
      label: `${item.name} (${item.server_name || 'local'})`,
      value: item.name
    }))
  })
}

// 显示时初始化数据
watch(show, (val) => {
  if (val) {
    if (props.mode === 'edit' && props.editData) {
      const config = props.editData.config || {}
      skipClearTargets.value = true
      formModel.value = {
        id: props.editData.id,
        name: props.editData.name,
        type: props.editData.type,
        time: props.editData.time,
        targets: config.targets || [],
        keep: config.keep || 1,
        sub_type: config.type || '',
        storage: config.storage || 0,
        script: ''
      }
      nextTick(() => {
        skipClearTargets.value = false
      })
      // 加载关联数据
      if (props.editData.type === 'cutoff' && config.type === 'container') {
        loadContainers()
      }
      if (props.editData.type === 'backup' && (config.type === 'mysql' || config.type === 'postgres')) {
        loadDatabases(config.type)
      }
    } else {
      formModel.value = defaultModel()
    }
  }
})

// 监听子类型变化加载数据
watch(
  () => formModel.value.sub_type,
  (val) => {
    if (formModel.value.type === 'cutoff' && val === 'container') {
      loadContainers()
    }
    if (formModel.value.type === 'backup' && (val === 'mysql' || val === 'postgres')) {
      loadDatabases(val)
    }
  }
)

onMounted(() => {
  useRequest(app.isInstalled('nginx,openresty,apache,caddy')).onSuccess(({ data }) => {
    if (data) {
      useRequest(website.list('all', 1, 10000)).onSuccess(({ data }: { data: any }) => {
        for (const item of data.items) {
          websites.value.push({
            label: item.name,
            value: item.name
          })
        }
      })
    }
  })
  useRequest(app.isInstalled('docker,podman')).onSuccess(({ data }) => {
    containerInstalled.value = !!data
  })
  useRequest(storage.list(1, 10000)).onSuccess(({ data }: { data: any }) => {
    storages.value = []
    for (const item of data.items) {
      storages.value.push({
        label: item.name,
        value: item.id
      })
    }
  })
})
</script>

<template>
  <n-modal
    v-model:show="show"
    preset="card"
    :title="mode === 'create' ? $gettext('Create Scheduled Task') : $gettext('Edit Scheduled Task')"
    style="width: 60vw"
    size="huge"
    :bordered="false"
    :segmented="false"
  >
    <n-form>
      <n-form-item v-if="mode === 'create'" :label="$gettext('Task Type')">
        <n-select
          v-model:value="formModel.type"
          :options="[
            { label: $gettext('Run Script'), value: 'shell' },
            { label: $gettext('Backup Data'), value: 'backup' },
            { label: $gettext('Log Rotation'), value: 'cutoff' }
          ]"
        />
      </n-form-item>
      <n-form-item :label="$gettext('Task Name')">
        <n-input v-model:value="formModel.name" :placeholder="$gettext('Task Name')" />
      </n-form-item>
      <n-form-item :label="$gettext('Task Schedule')">
        <cron-selector v-model:value="formModel.time" />
      </n-form-item>
      <div v-if="formModel.type === 'shell'">
        <n-text>{{ $gettext('Script Content') }}</n-text>
        <common-editor v-model:value="formModel.script" lang="shell" height="40vh" />
      </div>
      <!-- 备份类型 -->
      <n-form-item v-if="formModel.type === 'backup'" :label="$gettext('Backup Type')">
        <n-radio-group v-model:value="formModel.sub_type">
          <n-radio value="website">{{ $gettext('Website') }}</n-radio>
          <n-radio value="mysql" :disabled="!mySQLInstalled">
            {{ $gettext('MySQL Database') }}
          </n-radio>
          <n-radio value="postgres" :disabled="!postgreSQLInstalled">
            {{ $gettext('PostgreSQL Database') }}
          </n-radio>
        </n-radio-group>
      </n-form-item>
      <!-- 日志切割子类型 -->
      <n-form-item v-if="formModel.type === 'cutoff'" :label="$gettext('Rotation Type')">
        <n-radio-group v-model:value="formModel.sub_type">
          <n-radio value="website">{{ $gettext('Website') }}</n-radio>
          <n-radio value="container" :disabled="!containerInstalled">{{ $gettext('Container') }}</n-radio>
        </n-radio-group>
      </n-form-item>
      <!-- 网站多选 -->
      <n-form-item
        v-if="
          (formModel.sub_type === 'website' && formModel.type === 'backup') ||
          (formModel.type === 'cutoff' && formModel.sub_type === 'website')
        "
        :label="$gettext('Select Website')"
      >
        <n-select
          v-model:value="formModel.targets"
          :options="websites"
          multiple
          :placeholder="$gettext('Select Website')"
        />
      </n-form-item>
      <!-- 数据库多选 -->
      <n-form-item
        v-if="formModel.type === 'backup' && (formModel.sub_type === 'mysql' || formModel.sub_type === 'postgres')"
        :label="$gettext('Select Database')"
      >
        <n-select
          v-model:value="formModel.targets"
          :options="databases"
          multiple
          :placeholder="$gettext('Select Database')"
        />
      </n-form-item>
      <!-- 容器多选 -->
      <n-form-item
        v-if="formModel.type === 'cutoff' && formModel.sub_type === 'container'"
        :label="$gettext('Select Container')"
      >
        <n-select
          v-model:value="formModel.targets"
          :options="containers"
          multiple
          :placeholder="$gettext('Select Container')"
        />
      </n-form-item>
      <!-- 存储选择 -->
      <n-form-item
        v-if="formModel.type === 'backup' || formModel.type === 'cutoff'"
        :label="$gettext('Storage')"
      >
        <n-select
          v-model:value="formModel.storage"
          :options="storages"
          :placeholder="$gettext('Select storage')"
        />
      </n-form-item>
      <!-- 保留份数 -->
      <n-form-item v-if="formModel.type !== 'shell'" :label="$gettext('Retention Count')">
        <n-input-number v-model:value="formModel.keep" :min="1" />
      </n-form-item>
    </n-form>
    <n-button type="info" :loading="loading" :disabled="loading" @click="handleSubmit" mt-10 block>
      {{ mode === 'create' ? $gettext('Submit') : $gettext('Save') }}
    </n-button>
  </n-modal>
</template>

<style scoped lang="scss"></style>

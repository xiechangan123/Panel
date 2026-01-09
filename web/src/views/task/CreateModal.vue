<script setup lang="ts">
import app from '@/api/panel/app'
import cron from '@/api/panel/cron'
import home from '@/api/panel/home'
import website from '@/api/panel/website'
import CronSelector from '@/components/common/CronSelector.vue'
import { NInput } from 'naive-ui'
import { useGettext } from 'vue3-gettext'

const { $gettext } = useGettext()
const show = defineModel<boolean>('show', { type: Boolean, required: true })
const loading = ref(false)

const createModel = ref({
  name: '',
  type: 'shell',
  target: '',
  save: 1,
  backup_type: 'website',
  backup_path: '',
  script: $gettext('# Enter your script content here'),
  time: '* * * * *'
})

const websites = ref<any>([])

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

const handleSubmit = async () => {
  loading.value = true
  useRequest(cron.create(createModel.value))
    .onSuccess(() => {
      window.$message.success($gettext('Created successfully'))
      window.$bus.emit('task:refresh-cron')
    })
    .onComplete(() => {
      show.value = false
      loading.value = false
    })
}

watch(createModel, (value) => {
  if (value.backup_type === 'website') {
    createModel.value.target = websites.value[0]?.value
  } else {
    createModel.value.target = ''
  }
})

onMounted(() => {
  useRequest(app.isInstalled('nginx')).onSuccess(({ data }) => {
    if (data) {
      useRequest(website.list('all', 1, 10000)).onSuccess(({ data }: { data: any }) => {
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
  <n-modal
    v-model:show="show"
    preset="card"
    :title="$gettext('Create Scheduled Task')"
    style="width: 60vw"
    size="huge"
    :bordered="false"
    :segmented="false"
  >
    <n-form>
      <n-form-item :label="$gettext('Task Type')">
        <n-select
          v-model:value="createModel.type"
          :options="[
            { label: $gettext('Run Script'), value: 'shell' },
            { label: $gettext('Backup Data'), value: 'backup' },
            { label: $gettext('Log Rotation'), value: 'cutoff' }
          ]"
        >
        </n-select>
      </n-form-item>
      <n-form-item :label="$gettext('Task Name')">
        <n-input v-model:value="createModel.name" :placeholder="$gettext('Task Name')" />
      </n-form-item>
      <n-form-item :label="$gettext('Task Schedule')">
        <cron-selector v-model:value="createModel.time" />
      </n-form-item>
      <div v-if="createModel.type === 'shell'">
        <n-text>{{ $gettext('Script Content') }}</n-text>
        <common-editor v-model:value="createModel.script" lang="sh" height="40vh" />
      </div>
      <n-form-item v-if="createModel.type === 'backup'" :label="$gettext('Backup Type')">
        <n-radio-group v-model:value="createModel.backup_type">
          <n-radio value="website">{{ $gettext('Website') }}</n-radio>
          <n-radio value="mysql" :disabled="!mySQLInstalled">
            {{ $gettext('MySQL Database') }}</n-radio
          >
          <n-radio value="postgres" :disabled="!postgreSQLInstalled">
            {{ $gettext('PostgreSQL Database') }}
          </n-radio>
        </n-radio-group>
      </n-form-item>
      <n-form-item
        v-if="
          (createModel.backup_type === 'website' && createModel.type === 'backup') ||
          createModel.type === 'cutoff'
        "
        :label="$gettext('Select Website')"
      >
        <n-select
          v-model:value="createModel.target"
          :options="websites"
          :placeholder="$gettext('Select Website')"
        />
      </n-form-item>
      <n-form-item
        v-if="createModel.backup_type !== 'website' && createModel.type === 'backup'"
        :label="$gettext('Database Name')"
      >
        <n-input v-model:value="createModel.target" :placeholder="$gettext('Database Name')" />
      </n-form-item>
      <n-form-item v-if="createModel.type === 'backup'" :label="$gettext('Save Directory')">
        <n-input
          v-model:value="createModel.backup_path"
          :placeholder="$gettext('Save Directory')"
        />
      </n-form-item>
      <n-form-item v-if="createModel.type !== 'shell'" :label="$gettext('Retention Count')">
        <n-input-number v-model:value="createModel.save" />
      </n-form-item>
    </n-form>
    <n-button type="info" :loading="loading" @click="handleSubmit" mt-10 block>
      {{ $gettext('Submit') }}
    </n-button>
  </n-modal>
</template>

<style scoped lang="scss"></style>

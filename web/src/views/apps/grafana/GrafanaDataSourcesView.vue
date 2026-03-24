<script setup lang="ts">
defineOptions({
  name: 'grafana-datasources'
})

import { NButton, NDataTable, NPopconfirm, NSpace, NTag, NModal, NForm, NFormItem, NInput, NSelect, NSwitch } from 'naive-ui'
import { useGettext } from 'vue3-gettext'

import grafana from '@/api/apps/grafana'

const { $gettext } = useGettext()

const { data: datasources, send: refreshList } = useRequest(grafana.datasources, {
  initialData: []
})

const showModal = ref(false)
const editMode = ref(false)
const editOldName = ref('')
const saveLoading = ref(false)

const formModel = ref({
  name: '',
  type: 'prometheus',
  url: 'http://127.0.0.1:9090',
  access: 'proxy',
  is_default: false,
  database: '',
  user: '',
  password: ''
})

const typeOptions = [
  { label: 'Prometheus', value: 'prometheus' },
  { label: 'MySQL', value: 'mysql' },
  { label: 'PostgreSQL', value: 'postgres' },
  { label: 'InfluxDB', value: 'influxdb' },
  { label: 'Loki', value: 'loki' },
  { label: 'Elasticsearch', value: 'elasticsearch' }
]

const accessOptions = [
  { label: $gettext('Server (Proxy)'), value: 'proxy' },
  { label: $gettext('Browser (Direct)'), value: 'direct' }
]

const needsDbFields = computed(() => ['mysql', 'postgres', 'influxdb'].includes(formModel.value.type))

const columns: any = [
  {
    title: $gettext('Name'),
    key: 'name',
    minWidth: 150,
    ellipsis: { tooltip: true }
  },
  {
    title: $gettext('Type'),
    key: 'type',
    width: 140
  },
  {
    title: 'URL',
    key: 'url',
    minWidth: 200,
    ellipsis: { tooltip: true }
  },
  {
    title: $gettext('Default'),
    key: 'isDefault',
    width: 100,
    render(row: any) {
      return row.isDefault
        ? h(NTag, { type: 'success', size: 'small' }, { default: () => $gettext('Yes') })
        : h(NTag, { type: 'default', size: 'small' }, { default: () => $gettext('No') })
    }
  },
  {
    title: $gettext('Actions'),
    key: 'actions',
    width: 200,
    render(row: any) {
      return h(NSpace, { size: 'small' }, {
        default: () => [
          h(NButton, { size: 'small', onClick: () => handleEdit(row) }, { default: () => $gettext('Edit') }),
          h(
            NPopconfirm,
            { onPositiveClick: () => handleDelete(row.name) },
            {
              default: () => $gettext('Are you sure you want to delete datasource %{ name }?', { name: row.name }),
              trigger: () => h(NButton, { size: 'small', type: 'error' }, { default: () => $gettext('Delete') })
            }
          )
        ]
      })
    }
  }
]

const handleAdd = () => {
  editMode.value = false
  formModel.value = {
    name: '',
    type: 'prometheus',
    url: 'http://127.0.0.1:9090',
    access: 'proxy',
    is_default: false,
    database: '',
    user: '',
    password: ''
  }
  showModal.value = true
}

const handleEdit = (row: any) => {
  editMode.value = true
  editOldName.value = row.name
  formModel.value = {
    name: row.name,
    type: row.type,
    url: row.url,
    access: row.access || 'proxy',
    is_default: row.isDefault || false,
    database: row.database || '',
    user: row.user || '',
    password: ''
  }
  showModal.value = true
}

const handleSave = () => {
  saveLoading.value = true
  const req = editMode.value
    ? grafana.updateDatasource(editOldName.value, formModel.value)
    : grafana.createDatasource(formModel.value)
  useRequest(req)
    .onSuccess(() => {
      window.$message.success($gettext('Saved successfully'))
      showModal.value = false
      refreshList()
    })
    .onComplete(() => {
      saveLoading.value = false
    })
}

const handleDelete = (name: string) => {
  useRequest(grafana.deleteDatasource(name)).onSuccess(() => {
    window.$message.success($gettext('Deleted successfully'))
    refreshList()
  })
}
</script>

<template>
  <n-flex vertical>
    <n-flex>
      <n-button type="primary" @click="handleAdd">
        {{ $gettext('Add Data Source') }}
      </n-button>
    </n-flex>
    <n-data-table striped :columns="columns" :data="datasources" :scroll-x="800" />
    <n-modal v-model:show="showModal" preset="card" :title="editMode ? $gettext('Edit Data Source') : $gettext('Add Data Source')" style="width: 600px">
      <n-form :model="formModel" label-placement="left" label-width="auto">
        <n-form-item :label="$gettext('Name')">
          <n-input v-model:value="formModel.name" :placeholder="$gettext('e.g. Prometheus')" />
        </n-form-item>
        <n-form-item :label="$gettext('Type')">
          <n-select v-model:value="formModel.type" :options="typeOptions" />
        </n-form-item>
        <n-form-item label="URL">
          <n-input v-model:value="formModel.url" placeholder="http://127.0.0.1:9090" />
        </n-form-item>
        <n-form-item :label="$gettext('Access')">
          <n-select v-model:value="formModel.access" :options="accessOptions" />
        </n-form-item>
        <n-form-item :label="$gettext('Default')">
          <n-switch v-model:value="formModel.is_default" />
        </n-form-item>
        <template v-if="needsDbFields">
          <n-form-item :label="$gettext('Database')">
            <n-input v-model:value="formModel.database" />
          </n-form-item>
          <n-form-item :label="$gettext('User')">
            <n-input v-model:value="formModel.user" />
          </n-form-item>
          <n-form-item :label="$gettext('Password')">
            <n-input v-model:value="formModel.password" type="password" :placeholder="editMode ? $gettext('Leave empty to keep unchanged') : ''" />
          </n-form-item>
        </template>
      </n-form>
      <template #footer>
        <n-flex justify="end">
          <n-button @click="showModal = false">{{ $gettext('Cancel') }}</n-button>
          <n-button type="primary" :loading="saveLoading" :disabled="saveLoading" @click="handleSave">
            {{ $gettext('Save') }}
          </n-button>
        </n-flex>
      </template>
    </n-modal>
  </n-flex>
</template>

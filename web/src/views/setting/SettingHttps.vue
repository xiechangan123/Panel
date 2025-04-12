<script setup lang="ts">
import setting from '@/api/panel/setting'
import { useGettext } from 'vue3-gettext'

const { $gettext } = useGettext()

const { data: model } = useRequest(setting.list, {
  initialData: {
    name: '',
    locale: '',
    username: '',
    password: '',
    email: '',
    port: 8888,
    entrance: '',
    offline_mode: false,
    website_path: '',
    backup_path: '',
    https: false,
    cert: '',
    key: ''
  }
})

const handleSave = () => {
  useRequest(setting.update(model.value)).onSuccess(() => {
    window.$message.success($gettext('Saved successfully'))
  })
}
</script>

<template>
  <n-space vertical>
    <n-alert type="warning"> {{ $gettext('Incorrect certificates may cause the panel to be inaccessible. Please proceed with caution!') }}</n-alert>
    <n-form>
      <n-form-item :label="$gettext('Panel HTTPS')">
        <n-switch v-model:value="model.https" />
      </n-form-item>
      <n-form-item v-if="model.https" :label="$gettext('Certificate')">
        <n-input
          v-model:value="model.cert"
          type="textarea"
          :autosize="{ minRows: 10, maxRows: 15 }"
        />
      </n-form-item>
      <n-form-item v-if="model.https" :label="$gettext('Private Key')">
        <n-input
          v-model:value="model.key"
          type="textarea"
          :autosize="{ minRows: 10, maxRows: 15 }"
        />
      </n-form-item>
    </n-form>
  </n-space>
  <n-button type="primary" @click="handleSave">
    {{ $gettext('Save') }}
  </n-button>
</template>

<style scoped lang="scss"></style>

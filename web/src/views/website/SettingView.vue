<script setup lang="ts">
import website from '@/api/panel/website'
import Editor from '@guolao/vue-monaco-editor'
import { useGettext } from 'vue3-gettext'

const { $gettext } = useGettext()

const currentTab = ref('default-page')

const defaultPageModel = ref({
  index: '',
  not_found: '',
  stop: ''
})

const defaultSettingModel = ref({
  tls_version: ['TLSv1.2', 'TLSv1.3'],
  cipher_suites: ''
})

const getDefaultPage = async () => {
  defaultPageModel.value = await website.defaultConfig()
}

const handleSaveDefaultPage = () => {
  useRequest(
    website.saveDefaultConfig(defaultPageModel.value.index, defaultPageModel.value.stop)
  ).onSuccess(() => {
    window.$message.success($gettext('Modified successfully'))
  })
}

onMounted(() => {
  getDefaultPage()
})
</script>

<template>
  <n-tabs v-model:value="currentTab" type="line" placement="left" animated>
    <n-tab-pane name="default-page" :tab="$gettext('Default Page')">
      <Editor
        v-model:value="defaultPageModel.index"
        language="html"
        theme="vs-dark"
        height="60vh"
        mt-8
        :options="{
          automaticLayout: true,
          formatOnType: true,
          formatOnPaste: true
        }"
      />
    </n-tab-pane>
    <n-tab-pane name="404-page" :tab="$gettext('404 Page')">
      <Editor
        v-model:value="defaultPageModel.not_found"
        language="html"
        theme="vs-dark"
        height="60vh"
        mt-8
        :options="{
          automaticLayout: true,
          formatOnType: true,
          formatOnPaste: true
        }"
      />
    </n-tab-pane>
    <n-tab-pane name="stop-page" :tab="$gettext('Stop Page')">
      <Editor
        v-model:value="defaultPageModel.stop"
        language="html"
        theme="vs-dark"
        height="60vh"
        mt-8
        :options="{
          automaticLayout: true,
          formatOnType: true,
          formatOnPaste: true
        }"
      />
    </n-tab-pane>
    <n-tab-pane name="default-site" :tab="$gettext('Default Site')">
      <n-alert type="info">待开发</n-alert>
    </n-tab-pane>
    <n-tab-pane name="default-setting" :tab="$gettext('Default Settings')">
      <n-form>
        <n-form-item :label="$gettext('Default TLS Version')">
          <n-select
            v-model:value="defaultSettingModel.tls_version"
            :options="[
              { label: 'TLS 1.0', value: 'TLSv1.0' },
              { label: 'TLS 1.1', value: 'TLSv1.1' },
              { label: 'TLS 1.2', value: 'TLSv1.2' },
              { label: 'TLS 1.3', value: 'TLSv1.3' }
            ]"
            multiple
          />
        </n-form-item>
        <n-form-item :label="$gettext('Default Cipher Suites')">
          <n-input
            type="textarea"
            v-model:value="defaultSettingModel.cipher_suites"
            :placeholder="
              $gettext('Enter the default cipher suite, leave blank to reset to default')
            "
            rows="4"
          />
        </n-form-item>
        <n-button type="primary">
          {{ $gettext('Save Changes') }}
        </n-button>
      </n-form>
    </n-tab-pane>
  </n-tabs>
</template>

<style scoped lang="scss"></style>

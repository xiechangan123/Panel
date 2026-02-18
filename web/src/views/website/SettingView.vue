<script setup lang="ts">
import website from '@/api/panel/website'
import home from '@/api/panel/home'
import { useGettext } from 'vue3-gettext'

const { $gettext } = useGettext()

const currentTab = ref('default-page')
const saveLoading = ref(false)

const { data: model } = useRequest(website.defaultConfig, {
  initialData: {
    index: '',
    stop: '',
    not_found: '',
    tls_versions: ['TLSv1.2', 'TLSv1.3'],
    cipher_suites: ''
  }
})

// 判断 webserver 类型
const isNginx = ref(false)
useRequest(home.installedEnvironment()).onSuccess(({ data }: any) => {
  isNginx.value = data.webserver === 'nginx'
})

// 统计设置
const statSetting = ref({
  days: 30,
  err_buf_max: 10000,
  uv_max_keys: 1000000,
  ip_max_keys: 500000,
  detail_max_keys: 100000,
  body_enabled: true
})
const statSaveLoading = ref(false)

useRequest(website.statSetting()).onSuccess(({ data }: any) => {
  statSetting.value = data
})

watch(
  () => model.value.tls_versions,
  (newVal) => {
    if (!newVal.includes('TLSv1.1') && !newVal.includes('TLSv1.0')) {
      // 不包含 TLSv1.0 和 TLSv1.1
      model.value.cipher_suites =
        'ECDHE-ECDSA-AES128-GCM-SHA256:ECDHE-RSA-AES128-GCM-SHA256:ECDHE-ECDSA-AES256-GCM-SHA384:ECDHE-RSA-AES256-GCM-SHA384:ECDHE-ECDSA-CHACHA20-POLY1305:ECDHE-RSA-CHACHA20-POLY1305:DHE-RSA-AES128-GCM-SHA256:DHE-RSA-AES256-GCM-SHA384:DHE-RSA-CHACHA20-POLY1305'
    } else {
      // 包含 TLSv1.0 或 TLSv1.1
      model.value.cipher_suites =
        '@SECLEVEL=0:ECDHE-ECDSA-AES128-GCM-SHA256:ECDHE-RSA-AES128-GCM-SHA256:ECDHE-ECDSA-AES256-GCM-SHA384:ECDHE-RSA-AES256-GCM-SHA384:ECDHE-ECDSA-CHACHA20-POLY1305:ECDHE-RSA-CHACHA20-POLY1305:DHE-RSA-AES128-GCM-SHA256:DHE-RSA-AES256-GCM-SHA384:DHE-RSA-CHACHA20-POLY1305:ECDHE-ECDSA-AES128-SHA256:ECDHE-RSA-AES128-SHA256:ECDHE-ECDSA-AES128-SHA:ECDHE-RSA-AES128-SHA:ECDHE-ECDSA-AES256-SHA384:ECDHE-RSA-AES256-SHA384:ECDHE-ECDSA-AES256-SHA:ECDHE-RSA-AES256-SHA:DHE-RSA-AES128-SHA256:DHE-RSA-AES256-SHA256:AES128-GCM-SHA256:AES256-GCM-SHA384:AES128-SHA256:AES256-SHA256:AES128-SHA:AES256-SHA:DES-CBC3-SHA'
    }
  }
)

const handleSave = () => {
  saveLoading.value = true
  useRequest(website.saveDefaultConfig(model.value))
    .onSuccess(() => {
      window.$message.success($gettext('Modified successfully'))
    })
    .onComplete(() => {
      saveLoading.value = false
    })
}

const handleSaveStatSetting = () => {
  statSaveLoading.value = true
  useRequest(website.saveStatSetting(statSetting.value))
    .onSuccess(() => {
      window.$message.success($gettext('Modified successfully'))
    })
    .onComplete(() => {
      statSaveLoading.value = false
    })
}
</script>

<template>
  <n-tabs v-model:value="currentTab" type="line" placement="left" animated>
    <n-tab-pane name="default-page" :tab="$gettext('Default Page')">
      <n-flex vertical>
        <common-editor v-model:value="model.index" height="60vh" />
        <n-flex>
          <n-button type="primary" :loading="saveLoading" :disabled="saveLoading" @click="handleSave">
            {{ $gettext('Save Changes') }}
          </n-button>
        </n-flex>
      </n-flex>
    </n-tab-pane>
    <n-tab-pane name="stop-page" :tab="$gettext('Stop Page')">
      <n-flex>
        <common-editor v-model:value="model.stop" height="60vh" />
        <n-flex>
          <n-button type="primary" :loading="saveLoading" :disabled="saveLoading" @click="handleSave">
            {{ $gettext('Save Changes') }}
          </n-button>
        </n-flex>
      </n-flex>
    </n-tab-pane>
    <n-tab-pane name="404-page" :tab="$gettext('404 Page')">
      <n-flex>
        <common-editor v-model:value="model.not_found" height="60vh" />
        <n-flex>
          <n-button type="primary" :loading="saveLoading" :disabled="saveLoading" @click="handleSave">
            {{ $gettext('Save Changes') }}
          </n-button>
        </n-flex>
      </n-flex>
    </n-tab-pane>
    <n-tab-pane name="default-site" :tab="$gettext('Default Site')">
      <n-alert type="info">待开发</n-alert>
    </n-tab-pane>
    <n-tab-pane name="default-setting" :tab="$gettext('Default Settings')">
      <n-flex vertical>
        <n-alert type="info">
          {{
            $gettext(
              'Modifying the default TLS version and cipher suites will affect all newly created websites. Existing websites will not be affected.'
            )
          }}
        </n-alert>
        <n-alert type="warning">
          {{
            $gettext(
              'Please adjust the settings carefully, improper configuration may lead to website inaccessible.'
            )
          }}
        </n-alert>
        <n-form>
          <n-form-item :label="$gettext('Default TLS Version')">
            <n-select
              v-model:value="model.tls_versions"
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
              v-model:value="model.cipher_suites"
              :placeholder="
                $gettext('Enter the default cipher suite, leave blank to reset to default')
              "
              rows="4"
            />
          </n-form-item>
          <n-button type="primary" :loading="saveLoading" :disabled="saveLoading" @click="handleSave">
            {{ $gettext('Save Changes') }}
          </n-button>
        </n-form>
      </n-flex>
    </n-tab-pane>
    <n-tab-pane v-if="isNginx" name="stat-setting" :tab="$gettext('Statistics')">
      <n-flex vertical>
        <n-alert type="info">
          {{ $gettext('Modifications to the limits below will take effect after restarting the panel.') }}
        </n-alert>
        <n-form>
          <n-form-item :label="$gettext('Data Retention Days')">
            <n-input-number
              v-model:value="statSetting.days"
              :min="1"
              :max="365"
              style="width: 200px"
            />
          </n-form-item>
          <n-form-item :label="$gettext('Error Buffer Max Size')">
            <n-input-number
              v-model:value="statSetting.err_buf_max"
              :min="100"
              :max="1000000"
              :step="1000"
              style="width: 200px"
            />
          </n-form-item>
          <n-form-item :label="$gettext('UV Max Keys')">
            <n-input-number
              v-model:value="statSetting.uv_max_keys"
              :min="1000"
              :max="100000000"
              :step="100000"
              style="width: 200px"
            />
          </n-form-item>
          <n-form-item :label="$gettext('IP Max Keys')">
            <n-input-number
              v-model:value="statSetting.ip_max_keys"
              :min="1000"
              :max="100000000"
              :step="100000"
              style="width: 200px"
            />
          </n-form-item>
          <n-form-item :label="$gettext('Detail Max Keys')">
            <n-input-number
              v-model:value="statSetting.detail_max_keys"
              :min="1000"
              :max="100000000"
              :step="10000"
              style="width: 200px"
            />
          </n-form-item>
          <n-form-item :label="$gettext('Record Request Body')">
            <n-switch v-model:value="statSetting.body_enabled" />
          </n-form-item>
          <n-button
            type="primary"
            :loading="statSaveLoading"
            :disabled="statSaveLoading"
            @click="handleSaveStatSetting"
          >
            {{ $gettext('Save Changes') }}
          </n-button>
        </n-form>
      </n-flex>
    </n-tab-pane>
  </n-tabs>
</template>

<style scoped lang="scss"></style>

<script setup lang="ts">
import { NInput } from 'naive-ui'
import { useGettext } from 'vue3-gettext'

const { $gettext } = useGettext()
const show = defineModel<boolean>('show', { type: Boolean, required: true })
const config = defineModel<string>('config', { type: String, required: true })
const setting = ref({
  auto_resolve: true,
  sni: true,
  cache: false,
  cache_time: 1,
  no_buffer: false,
  proxy_pass: '',
  host: '$host',
  match_type: '^~',
  match: '/',
  replace: []
})

const handleSubmit = () => {
  if (setting.value.cache && setting.value.no_buffer) {
    window.$message.error($gettext('Disabled buffer and enabled cache cannot be used simultaneously'))
    return
  }
  if (setting.value.match.length === 0) {
    window.$message.error($gettext('Matching expression cannot be empty'))
    return
  }
  if (setting.value.proxy_pass.length === 0) {
    window.$message.error($gettext('Proxy address cannot be empty'))
    return
  }
  if (setting.value.match_type === '=' && setting.value.match[0] !== '/') {
    window.$message.error($gettext('Exact match expression must start with /'))
    return
  }
  if (
    (setting.value.match_type === '^~' || setting.value.match_type === ' ') &&
    setting.value.match[0] !== '/'
  ) {
    window.$message.error($gettext('Prefix match expression must start with /'))
    return
  }
  try {
    new URL(setting.value.proxy_pass)
  } catch (error) {
    window.$message.error($gettext('Proxy address format error'))
    return
  }

  let builder: string
  builder = 'location'
  switch (setting.value.match_type) {
    case '=':
      builder += ' ='
      break
    case '^~':
      builder += ' ^~'
      break
    case '~':
      builder += ' ~'
      break
    case '~*':
      builder += ' ~*'
      break
  }
  builder += ` ${setting.value.match}\n{\n`
  if (setting.value.auto_resolve) {
    builder += `    set $empty "";\n    proxy_pass ${setting.value.proxy_pass}$empty;\n`
  } else {
    builder += `    proxy_pass ${setting.value.proxy_pass};\n`
  }
  if (setting.value.host) {
    builder += `    proxy_set_header Host ${setting.value.host};\n`
  }
  builder += `    proxy_set_header X-Real-IP $remote_addr;\n    proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;\n    proxy_set_header X-Forwarded-Host $host;\n    proxy_set_header X-Forwarded-Port $server_port;\n    proxy_set_header X-Forwarded-Proto $scheme;\n    proxy_set_header X-Forwarded-Scheme $scheme;\n`
  builder += `    proxy_set_header Upgrade $http_upgrade;\n    proxy_set_header Connection $http_connection;\n    proxy_set_header Early-Data $ssl_early_data;\n    proxy_set_header Accept-Encoding "";\n    proxy_http_version 1.1;\n    proxy_ssl_protocols TLSv1.2 TLSv1.3;\n    proxy_ssl_session_reuse off;\n`
  if (setting.value.sni) {
    builder += `    proxy_ssl_server_name on;\n`
  }
  if (setting.value.auto_resolve) {
    builder += `    resolver 8.8.8.8 ipv6=off;\n    resolver_timeout 10s;\n`
  }
  if (setting.value.cache) {
    builder += `    proxy_ignore_headers X-Accel-Expires Expires Cache-Control Set-Cookie;\n    proxy_cache cache_one;\n    proxy_cache_key $scheme$host$uri$is_args$args;\n    proxy_cache_valid 200 304 301 302 ${setting.value.cache_time}m;\n    proxy_cache_lock on;\n    proxy_cache_lock_timeout 5s;\n    proxy_cache_lock_age 5s;\n    proxy_cache_background_update on;\n    proxy_cache_use_stale error timeout invalid_header updating http_500 http_502 http_503 http_504;\n    proxy_cache_revalidate on;\n`
  }
  if (setting.value.no_buffer) {
    builder += `    proxy_buffering off;\n    proxy_request_buffering off;\n`
  }
  if (setting.value.replace.length > 0) {
    builder += `    sub_filter_once off;\n    sub_filter_types *;\n`
    for (const item of setting.value.replace) {
      builder += `    sub_filter "${(item as any).key}" "${(item as any).value}";\n`
    }
  }
  builder += `}\n`
  config.value = builder
  show.value = false
  window.$message.success($gettext('Configuration generated successfully'))
}

// 通过代理地址尝试自动获取发送域名
watch(
  () => setting.value.proxy_pass,
  (val) => {
    if (val.length > 0) {
      try {
        const url = new URL(val)
        setting.value.host = url.hostname
      } catch (error) {}
    }
  }
)
</script>

<template>
  <n-modal
    v-model:show="show"
    preset="card"
    :title="$gettext('Generate Reverse Proxy Configuration')"
    style="width: 40vw"
    size="huge"
    :bordered="false"
    :segmented="false"
  >
    <n-flex vertical>
      <n-alert type="warning"> {{ $gettext('After generating the reverse proxy configuration, the original rewrite rules will be overwritten.') }} </n-alert>
      <n-alert type="info">
        {{ $gettext('If you need to proxy static resources like JS/CSS, please remove the static log recording part from the original configuration.') }}
      </n-alert>
      <n-form inline>
        <n-form-item :label="$gettext('Auto Refresh Resolution')">
          <n-switch v-model:value="setting.auto_resolve" />
        </n-form-item>
        <n-form-item :label="$gettext('Enable SNI')">
          <n-switch v-model:value="setting.sni" />
        </n-form-item>
        <n-form-item :label="$gettext('Enable Cache')">
          <n-switch v-model:value="setting.cache" />
        </n-form-item>
        <n-form-item :label="$gettext('Disable Buffer')">
          <n-switch v-model:value="setting.no_buffer" />
        </n-form-item>
      </n-form>
      <n-form>
        <n-form-item :label="$gettext('Match Type')">
          <n-select
            v-model:value="setting.match_type"
            :options="[
              { label: $gettext('Exact Match (=)'), value: '=' },
              { label: $gettext('Priority Prefix Match (^~)'), value: '^~' },
              { label: $gettext('Normal Prefix Match ( )'), value: ' ' },
              { label: $gettext('Case Sensitive Regex Match (~)'), value: '~' },
              { label: $gettext('Case Insensitive Regex Match (~*)'), value: '~*' }
            ]"
          />
        </n-form-item>
        <n-form-item :label="$gettext('Match Expression')">
          <n-input v-model:value="setting.match" placeholder="/" />
        </n-form-item>
        <n-form-item :label="$gettext('Proxy Address')">
          <n-input v-model:value="setting.proxy_pass" placeholder="http://127.0.0.1:3000" />
        </n-form-item>
        <n-form-item :label="$gettext('Send Domain')">
          <n-input v-model:value="setting.host" placeholder="$host" />
        </n-form-item>
        <n-form-item v-if="setting.cache" :label="$gettext('Cache Time')">
          <n-input-number
            v-model:value="setting.cache_time"
            w-full
            :min="1"
            :step="1"
            :placeholder="$gettext('Cache time (minutes)')"
          >
            <template #suffix> {{ $gettext('minutes') }} </template>
          </n-input-number>
        </n-form-item>
        <n-form-item :label="$gettext('Content Replacement')">
          <n-dynamic-input
            v-model:value="setting.replace"
            preset="pair"
            :max="5"
            :key-placeholder="$gettext('Target content')"
            :value-placeholder="$gettext('Replacement content')"
          />
        </n-form-item>
      </n-form>
      <n-button type="info" block @click="handleSubmit"> {{ $gettext('Submit') }} </n-button>
    </n-flex>
  </n-modal>
</template>

<style scoped lang="scss"></style>

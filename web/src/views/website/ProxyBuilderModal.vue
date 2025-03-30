<script setup lang="ts">
import { NInput } from 'naive-ui'

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
    window.$message.error('禁用缓冲区与启用缓存不能同时使用')
    return
  }
  if (setting.value.match.length === 0) {
    window.$message.error('匹配表达式不能为空')
    return
  }
  if (setting.value.proxy_pass.length === 0) {
    window.$message.error('代理地址不能为空')
    return
  }
  if (setting.value.match_type === '=' && setting.value.match[0] !== '/') {
    window.$message.error('精确匹配的表达式必须以 / 开头')
    return
  }
  if (
    (setting.value.match_type === '^~' || setting.value.match_type === ' ') &&
    setting.value.match[0] !== '/'
  ) {
    window.$message.error('前缀匹配的表达式必须以 / 开头')
    return
  }
  try {
    new URL(setting.value.proxy_pass)
  } catch (error) {
    window.$message.error('代理地址格式错误')
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
  window.$message.success('配置生成成功')
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
    title="生成反代配置"
    style="width: 40vw"
    size="huge"
    :bordered="false"
    :segmented="false"
  >
    <n-flex vertical>
      <n-alert type="warning"> 生成反代配置后，原有伪静态规则将被覆盖！ </n-alert>
      <n-form inline>
        <n-form-item label="自动刷新解析">
          <n-switch v-model:value="setting.auto_resolve" />
        </n-form-item>
        <n-form-item label="启用 SNI">
          <n-switch v-model:value="setting.sni" />
        </n-form-item>
        <n-form-item label="启用缓存">
          <n-switch v-model:value="setting.cache" />
        </n-form-item>
        <n-form-item label="禁用缓冲区">
          <n-switch v-model:value="setting.no_buffer" />
        </n-form-item>
      </n-form>
      <n-form>
        <n-form-item label="匹配方式">
          <n-select
            v-model:value="setting.match_type"
            :options="[
              { label: '精确匹配 (=)', value: '=' },
              { label: '优先前缀匹配 (^~)', value: '^~' },
              { label: '普通前缀匹配 ( )', value: ' ' },
              { label: '区分大小写正则匹配 (~)', value: '~' },
              { label: '不区分大小写正则匹配 (~*)', value: '~*' }
            ]"
          />
        </n-form-item>
        <n-form-item label="匹配表达式">
          <n-input v-model:value="setting.match" placeholder="/" />
        </n-form-item>
        <n-form-item label="代理地址">
          <n-input v-model:value="setting.proxy_pass" placeholder="http://127.0.0.1:3000" />
        </n-form-item>
        <n-form-item label="发送域名">
          <n-input v-model:value="setting.host" placeholder="$host" />
        </n-form-item>
        <n-form-item v-if="setting.cache" label="缓存时间">
          <n-input-number
            v-model:value="setting.cache_time"
            w-full
            :min="1"
            :step="1"
            :placeholder="'缓存时间（分钟）'"
          >
            <template #suffix> 分钟 </template>
          </n-input-number>
        </n-form-item>
        <n-form-item label="内容替换">
          <n-dynamic-input
            v-model:value="setting.replace"
            preset="pair"
            :max="5"
            key-placeholder="目标内容"
            value-placeholder="替换内容"
          />
        </n-form-item>
      </n-form>
      <n-button type="info" block @click="handleSubmit"> 提交 </n-button>
    </n-flex>
  </n-modal>
</template>

<style scoped lang="scss"></style>

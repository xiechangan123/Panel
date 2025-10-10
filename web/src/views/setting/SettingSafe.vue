<script setup lang="ts">
import { useGettext } from 'vue3-gettext'

const { $gettext } = useGettext()

const model = defineModel<any>('model', { type: Object, required: true })
</script>

<template>
  <n-space vertical>
    <n-form>
      <n-form-item>
        <template #label>
          <n-tooltip>
            <template #trigger>
              <div class="flex items-center">
                {{ $gettext('Login Timeout') }}
                <the-icon :size="16" icon="mdi:help-circle-outline" class="ml-1" />
              </div>
            </template>
            {{
              $gettext(
                'Set the maximum lifetime (in minutes) of the login state, after which you need to log in again'
              )
            }}
          </n-tooltip>
        </template>
        <n-input-number
          v-model:value="model.lifetime"
          :placeholder="$gettext('120')"
          :min="10"
          :max="43200"
          w-full
        >
          <template #suffix>
            {{ $gettext('minutes') }}
          </template>
        </n-input-number>
      </n-form-item>
      <n-form-item>
        <template #label>
          <n-tooltip>
            <template #trigger>
              <div class="flex items-center">
                {{ $gettext('Access Entrance') }}
                <the-icon :size="16" icon="mdi:help-circle-outline" class="ml-1" />
              </div>
            </template>
            {{
              $gettext(
                'Set the access entrance of the panel (e.g. /mypanel) to prevent some malicious access. Leave blank to disable (not recommended)'
              )
            }}
          </n-tooltip>
        </template>
        <n-input v-model:value="model.entrance" />
      </n-form-item>
      <n-form-item>
        <template #label>
          <n-tooltip>
            <template #trigger>
              <div class="flex items-center">
                {{ $gettext('Request IP Header') }}
                <the-icon :size="16" icon="mdi:help-circle-outline" class="ml-1" />
              </div>
            </template>
            {{
              $gettext(
                'Set the header that carries the real IP of the client, useful when using CDN or reverse proxy. Leave blank to use the client IP directly'
              )
            }}
          </n-tooltip>
        </template>
        <n-input v-model:value="model.ip_header" :placeholder="$gettext('X-Real-IP')" />
      </n-form-item>
      <n-form-item>
        <template #label>
          <n-tooltip>
            <template #trigger>
              <div class="flex items-center">
                {{ $gettext('Bind Domain') }}
                <the-icon :size="16" icon="mdi:help-circle-outline" class="ml-1" />
              </div>
            </template>
            {{
              $gettext(
                'Restrict panel access to the specified domain names. Leave blank to allow access from any domain'
              )
            }}
          </n-tooltip>
        </template>
        <n-dynamic-input
          v-model:value="model.bind_domain"
          placeholder="example.com"
          show-sort-button
        />
      </n-form-item>
      <n-form-item>
        <template #label>
          <n-tooltip>
            <template #trigger>
              <div class="flex items-center">
                {{ $gettext('Bind IP') }}
                <the-icon :size="16" icon="mdi:help-circle-outline" class="ml-1" />
              </div>
            </template>
            {{
              $gettext(
                'Restrict panel access to the specified IP addresses. Leave blank to allow access from any IP'
              )
            }}
          </n-tooltip>
        </template>
        <n-dynamic-input v-model:value="model.bind_ip" placeholder="127.0.0.1" show-sort-button />
      </n-form-item>
      <n-form-item>
        <template #label>
          <n-tooltip>
            <template #trigger>
              <div class="flex items-center">
                {{ $gettext('Bind UA') }}
                <the-icon :size="16" icon="mdi:help-circle-outline" class="ml-1" />
              </div>
            </template>
            {{
              $gettext(
                'Restrict panel access to the specified User-Agent strings. Leave blank to allow access from any User-Agent'
              )
            }}
          </n-tooltip>
        </template>
        <n-dynamic-input
          v-model:value="model.bind_ua"
          placeholder="Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Safari/537.36"
          show-sort-button
        />
      </n-form-item>
      <n-form-item>
        <template #label>
          <n-tooltip>
            <template #trigger>
              <div class="flex items-center">
                {{ $gettext('Offline Mode') }}
                <the-icon :size="16" icon="mdi:help-circle-outline" class="ml-1" />
              </div>
            </template>
            {{
              $gettext(
                'When enabled, the panel will not attempt to connect to external services for updates or other features. This may limit some functionalities'
              )
            }}
          </n-tooltip>
        </template>
        <n-switch v-model:value="model.offline_mode" />
      </n-form-item>
      <n-form-item>
        <template #label>
          <n-tooltip>
            <template #trigger>
              <div class="flex items-center">
                {{ $gettext('Auto Update') }}
                <the-icon :size="16" icon="mdi:help-circle-outline" class="ml-1" />
              </div>
            </template>
            {{
              $gettext(
                'When enabled, the panel will automatically check for and install updates when they are available. It is recommended to keep this enabled to ensure you have the latest features and security patches'
              )
            }}
          </n-tooltip>
        </template>
        <n-switch v-model:value="model.auto_update" />
      </n-form-item>
      <n-form-item>
        <template #label>
          <n-tooltip>
            <template #trigger>
              <div class="flex items-center">
                {{ $gettext('Panel HTTPS') }}
                <the-icon :size="16" icon="mdi:help-circle-outline" class="ml-1" />
              </div>
            </template>
            {{
              $gettext(
                'Enable HTTPS for the panel to ensure secure communication. You need to provide a valid SSL certificate and private key'
              )
            }}
          </n-tooltip>
        </template>
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
</template>

<style scoped lang="scss"></style>

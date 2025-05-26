<script setup lang="ts">
import { useGettext } from 'vue3-gettext'

const { $gettext } = useGettext()

const model = defineModel<any>('model', { type: Object, required: true })
</script>

<template>
  <n-space vertical>
    <n-form>
      <n-form-item :label="$gettext('Login Timeout')">
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
      <n-form-item :label="$gettext('Access Entrance')">
        <n-input v-model:value="model.entrance" :placeholder="$gettext('/admin')" />
      </n-form-item>
      <n-form-item :label="$gettext('Bind Domain')">
        <n-dynamic-input
          v-model:value="model.bind_domain"
          placeholder="example.com"
          show-sort-button
        />
      </n-form-item>
      <n-form-item :label="$gettext('Bind IP')">
        <n-dynamic-input v-model:value="model.bind_ip" placeholder="127.0.0.1" show-sort-button />
      </n-form-item>
      <n-form-item :label="$gettext('Bind UA')">
        <n-dynamic-input
          v-model:value="model.bind_ua"
          placeholder="Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Safari/537.36"
          show-sort-button
        />
      </n-form-item>
      <n-form-item :label="$gettext('Offline Mode')">
        <n-switch v-model:value="model.offline_mode" />
      </n-form-item>
      <n-form-item :label="$gettext('Auto Update')">
        <n-switch v-model:value="model.auto_update" />
      </n-form-item>
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
</template>

<style scoped lang="scss"></style>

<script setup lang="ts">
import cert from '@/api/panel/cert'
import type { MessageReactive } from 'naive-ui'
import { NButton, NInput, NSpace } from 'naive-ui'
import { useGettext } from 'vue3-gettext'

const { $gettext } = useGettext()
const show = defineModel<boolean>('show', { type: Boolean, required: true })

const props = defineProps({
  caProviders: {
    type: Array<any>,
    required: true
  },
  algorithms: {
    type: Array<any>,
    required: true
  }
})

const { caProviders, algorithms } = toRefs(props)

let messageReactive: MessageReactive | null = null

const model = ref<any>({
  hmac_encoded: '',
  email: '',
  kid: '',
  key_type: 'P256',
  ca: 'googlecn'
})

const showEAB = computed(() => {
  return model.value.ca === 'google' || model.value.ca === 'sslcom'
})

const handleCreateAccount = () => {
  messageReactive = window.$message.loading(
    $gettext('Registering account with CA, please wait patiently'),
    {
      duration: 0
    }
  )
  useRequest(cert.accountCreate(model.value))
    .onSuccess(() => {
      window.$bus.emit('cert:refresh-account')
      window.$bus.emit('cert:refresh-async')
      show.value = false
      model.value.email = ''
      model.value.hmac_encoded = ''
      model.value.kid = ''
      window.$message.success($gettext('Created successfully'))
    })
    .onComplete(() => {
      messageReactive?.destroy()
    })
}
</script>

<template>
  <n-modal
    v-model:show="show"
    preset="card"
    :title="$gettext('Create Account')"
    style="width: 60vw"
    size="huge"
    :bordered="false"
    :segmented="false"
  >
    <n-space vertical>
      <n-alert type="info">{{
        $gettext(
          'Google and SSL.com require obtaining KID and HMAC from their official websites first'
        )
      }}</n-alert>
      <n-alert type="warning">
        {{
          $gettext(
            "Google is not accessible in mainland China, and other CAs depend on network conditions. GoogleCN or Let's Encrypt are recommended"
          )
        }}
      </n-alert>
      <n-form :model="model">
        <n-form-item path="ca" :label="$gettext('CA')">
          <n-select
            v-model:value="model.ca"
            :placeholder="$gettext('Select CA')"
            clearable
            :options="caProviders"
          />
        </n-form-item>
        <n-form-item path="key_type" :label="$gettext('Key Type')">
          <n-select
            v-model:value="model.key_type"
            :placeholder="$gettext('Select key type')"
            clearable
            :options="algorithms"
          />
        </n-form-item>
        <n-form-item path="email" :label="$gettext('Email')">
          <n-input
            v-model:value="model.email"
            type="text"
            @keydown.enter.prevent
            :placeholder="$gettext('Enter email address')"
          />
        </n-form-item>
        <n-form-item v-if="showEAB" path="kid" label="KID">
          <n-input
            v-model:value="model.kid"
            type="text"
            @keydown.enter.prevent
            :placeholder="$gettext('Enter KID')"
          />
        </n-form-item>
        <n-form-item v-if="showEAB" path="hmac_encoded" label="HMAC">
          <n-input
            v-model:value="model.hmac_encoded"
            type="text"
            @keydown.enter.prevent
            :placeholder="$gettext('Enter HMAC')"
          />
        </n-form-item>
      </n-form>
      <n-button type="info" block @click="handleCreateAccount">{{ $gettext('Submit') }}</n-button>
    </n-space>
  </n-modal>
</template>

<style scoped lang="scss"></style>

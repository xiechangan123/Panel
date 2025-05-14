<script setup lang="ts">
import user from '@/api/panel/user'
import { NButton, NInput } from 'naive-ui'
import { useGettext } from 'vue3-gettext'

const { $gettext } = useGettext()
const show = defineModel<boolean>('show', { type: Boolean, required: true })
const id = defineModel<number>('id', { type: Number, required: true })

const model = ref({
  img: '',
  url: '',
  secret: ''
})
const code = ref('')
const qrCode = computed(() => {
  return `data:image/png;base64,${model.value.img}`
})

const handleUpdate = () => {
  useRequest(() => user.updateTwoFA(id.value, code.value, model.value.secret)).onSuccess(() => {
    show.value = false
    code.value = ''
    window.$message.success($gettext('Updated successfully'))
    window.$bus.emit('user:refresh')
  })
}

watch(
  () => show.value,
  (val) => {
    if (val) {
      useRequest(() => user.generateTwoFA(id.value)).onSuccess(({ data }) => {
        model.value = data
      })
    }
  },
  { immediate: true }
)
</script>

<template>
  <n-modal
    v-model:show="show"
    preset="card"
    :title="$gettext('Enable 2FA')"
    style="width: 60vw"
    size="huge"
    :bordered="false"
    :segmented="false"
    @close="show = false"
  >
    <n-flex vertical>
      <n-flex :wrap="false" justify="start" align="flex-start" :size="20">
        <n-image :src="qrCode" :alt="$gettext('QR Code')" width="200" height="200" />
        <n-flex vertical :size="12">
          <n-text style="max-width: 400px">
            {{ $gettext('Scan the QR code with your 2FA app and enter the code below') }}
          </n-text>
          <n-text style="max-width: 400px">
            {{
              $gettext('If you cannot scan the QR code, please enter the URL below in your 2FA app')
            }}
          </n-text>
          <n-text style="max-width: 400px; word-break: break-all">
            {{ model.url }}
          </n-text>
        </n-flex>
      </n-flex>

      <n-form>
        <n-form-item path="code" :label="$gettext('Code')">
          <n-input
            v-model:value="code"
            @keydown.enter.prevent
            :placeholder="$gettext('Enter the code')"
          />
        </n-form-item>
      </n-form>
      <n-button type="info" block @click="handleUpdate">{{ $gettext('Submit') }}</n-button>
    </n-flex>
  </n-modal>
</template>

<style scoped lang="scss"></style>

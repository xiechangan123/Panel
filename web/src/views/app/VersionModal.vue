<script setup lang="ts">
import type { App } from '@/views/app/types'
import { useGettext } from 'vue3-gettext'
import app from '../../api/panel/app'

const { $gettext } = useGettext()

const show = defineModel<boolean>('show', { type: Boolean, required: true })
const operation = defineModel<string>('operation', { type: String, required: true })
const info = defineModel<App>('info', { type: Object, required: true })

const doSubmit = ref(false)

const model = ref({
  channel: null,
  version: ''
})

const options = computed(() => {
  return info.value.channels.map((channel) => {
    return {
      label: channel.name,
      value: channel.slug
    }
  })
})

const handleSubmit = () => {
  useRequest(app.install(info.value.slug, model.value.channel))
    .onSuccess(() => {
      window.$message.success(
        $gettext('Task submitted, please check the progress in background tasks')
      )
    })
    .onComplete(() => {
      doSubmit.value = false
      show.value = false
      model.value = {
        channel: null,
        version: ''
      }
    })
}

const handleChannelUpdate = (value: string) => {
  const channel = info.value.channels.find((channel) => channel.slug === value)
  if (channel) {
    model.value.version = channel.subs[0].version
  }
}

const handleClose = () => {
  show.value = false
  model.value = {
    channel: null,
    version: ''
  }
}
</script>

<template>
  <n-modal
    v-model:show="show"
    preset="card"
    :title="operation + ' ' + info.name"
    style="width: 60vw"
    size="huge"
    :bordered="false"
    :segmented="false"
    @close="handleClose"
  >
    <n-form :model="model">
      <n-form-item path="channel" :label="$gettext('Channel')">
        <n-select
          v-model:value="model.channel"
          :options="options"
          @update-value="handleChannelUpdate"
        />
      </n-form-item>
      <n-form-item path="channel" :label="$gettext('Version')">
        <n-input
          v-model:value="model.version"
          :placeholder="$gettext('Please select a channel')"
          readonly
          disabled
        />
      </n-form-item>
    </n-form>
    <n-button
      type="info"
      block
      :loading="doSubmit"
      :disabled="model.channel == null || doSubmit"
      @click="handleSubmit"
    >
      {{ $gettext('Submit') }}
    </n-button>
  </n-modal>
</template>

<style scoped lang="scss"></style>

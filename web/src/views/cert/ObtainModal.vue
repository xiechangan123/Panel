<script setup lang="ts">
import cert from '@/api/panel/cert'
import type { MessageReactive } from 'naive-ui'
import { NButton, NTable } from 'naive-ui'
import { useGettext } from 'vue3-gettext'

const { $gettext } = useGettext()
let messageReactive: MessageReactive | null = null

const show = defineModel<boolean>('show', { type: Boolean, required: true })
const id = defineModel<number>('id', { type: Number, required: true })

const model = ref({
  type: 'auto'
})

const options = [
  { label: $gettext('Automatic'), value: 'auto' },
  { label: $gettext('Manual'), value: 'manual' },
  { label: $gettext('Self-signed'), value: 'self-signed' }
]

const handleSubmit = () => {
  messageReactive = window.$message.loading($gettext('Please wait...'), {
    duration: 0
  })
  if (model.value.type == 'auto') {
    useRequest(cert.obtainAuto(id.value))
      .onSuccess(() => {
        window.$bus.emit('cert:refresh-cert')
        window.$bus.emit('cert:refresh-async')
        show.value = false
        window.$message.success($gettext('Issuance successful'))
      })
      .onComplete(() => {
        messageReactive?.destroy()
      })
  } else if (model.value.type == 'manual') {
    useRequest(cert.manualDNS(id.value))
      .onSuccess(({ data }: { data: any }) => {
        window.$message.info(
          $gettext(
            'Please set up DNS resolution for the domain first, then continue with the issuance'
          )
        )
        const d = window.$dialog.info({
          style: 'width: 60vw',
          title: $gettext('DNS Records to Set'),
          content: () => {
            return h(
              NTable,
              {},
              {
                default: () => [
                  h('thead', [
                    h('tr', [
                      h('th', $gettext('Domain')),
                      h('th', $gettext('Type')),
                      h('th', $gettext('Host Record')),
                      h('th', $gettext('Record Value'))
                    ])
                  ]),
                  h(
                    'tbody',
                    data.map((item: any) =>
                      h('tr', [
                        h('td', item?.domain),
                        h('td', 'TXT'),
                        h('td', item?.name),
                        h('td', item?.value)
                      ])
                    )
                  )
                ]
              }
            )
          },
          positiveText: $gettext('Issue'),
          onPositiveClick: async () => {
            d.loading = true
            messageReactive = window.$message.loading($gettext('Please wait...'), {
              duration: 0
            })
            useRequest(cert.obtainManual(id.value))
              .onSuccess(() => {
                window.$bus.emit('cert:refresh-cert')
                window.$bus.emit('cert:refresh-async')
                show.value = false
                window.$message.success($gettext('Issuance successful'))
              })
              .onComplete(() => {
                d.loading = false
                messageReactive?.destroy()
              })
          }
        })
      })
      .onComplete(() => {
        messageReactive?.destroy()
      })
  } else {
    useRequest(cert.obtainSelfSigned(id.value))
      .onSuccess(() => {
        window.$bus.emit('cert:refresh-cert')
        window.$bus.emit('cert:refresh-async')
        show.value = false
        window.$message.success($gettext('Issuance successful'))
      })
      .onComplete(() => {
        messageReactive?.destroy()
      })
  }
}
</script>

<template>
  <n-modal
    v-model:show="show"
    preset="card"
    :title="$gettext('Issue Certificate')"
    style="width: 60vw"
    size="huge"
    :bordered="false"
    :segmented="false"
  >
    <n-form :model="model">
      <n-form-item path="type" :label="$gettext('Issuance Mode')">
        <n-select v-model:value="model.type" :options="options" />
      </n-form-item>
      <n-button type="info" block @click="handleSubmit">{{ $gettext('Submit') }}</n-button>
    </n-form>
  </n-modal>
</template>

<style scoped lang="scss"></style>

<script setup lang="ts">
defineOptions({
  name: 'dashboard-update'
})

import { MdPreview } from 'md-editor-v3'
import 'md-editor-v3/lib/style.css'
import type { MessageReactive } from 'naive-ui'
import { NButton } from 'naive-ui'
import { useGettext } from 'vue3-gettext'

import dashboard from '@/api/panel/dashboard'
import { router } from '@/router'
import { formatDateTime } from '@/utils'

const { $gettext } = useGettext()
const { data: versions } = useRequest(dashboard.updateInfo, {
  initialData: []
})
let messageReactive: MessageReactive | null = null

const handleUpdate = () => {
  window.$dialog.warning({
    title: $gettext('Update Panel'),
    content: $gettext('Are you sure you want to update the panel?'),
    positiveText: $gettext('Confirm'),
    negativeText: $gettext('Cancel'),
    onPositiveClick: () => {
      messageReactive = window.$message.loading($gettext('Panel updating...'), {
        duration: 0
      })
      useRequest(dashboard.update())
        .onSuccess(() => {
          setTimeout(() => {
            setTimeout(() => {
              window.location.reload()
            }, 400)
            router.push({ name: 'dashboard-index' })
          }, 2500)
          window.$message.success($gettext('Panel updated successfully'))
        })
        .onComplete(() => {
          messageReactive?.destroy()
        })
    },
    onNegativeClick: () => {
      window.$message.info($gettext('Update canceled'))
    }
  })
}
</script>

<template>
  <common-page show-footer>
    <template #action>
      <div>
        <n-button v-if="versions" class="ml-16" type="primary" @click="handleUpdate">
          <the-icon :size="18" icon="material-symbols:arrow-circle-up-outline-rounded" />
          {{ $gettext('Update Now') }}
        </n-button>
      </div>
    </template>
    <n-timeline v-if="versions" pt-10>
      <n-timeline-item
        v-for="(item, index) in versions"
        :type="Number(index) == 0 ? 'info' : 'default'"
        :key="index"
        :title="item.version"
        :time="formatDateTime(item.updated_at)"
      >
        <MdPreview
          v-model="item.description"
          noMermaid
          noKatex
          noIconfont
          noHighlight
          noImgZoomIn
        />
      </n-timeline-item>
    </n-timeline>
    <div v-else pt-40>
      <n-result
        status="418"
        title="Loading..."
        :description="$gettext('Loading update information, please wait a moment')"
      />
    </div>
  </common-page>
</template>

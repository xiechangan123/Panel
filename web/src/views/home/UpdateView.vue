<script setup lang="ts">
defineOptions({
  name: 'home-update',
})

import { useGettext } from 'vue3-gettext'

import home from '@/api/panel/home'
import ws from '@/api/ws'
import { formatDateTime, http } from '@/utils'

const { $gettext } = useGettext()
const { data: versions } = useRequest(home.updateInfo, { initialData: [] })
const { data: systemInfo } = useRequest(home.systemInfo)

const updating = ref(false)
const progressLogs = ref<string[]>([])
const errorMsg = ref('')
const waitingRestart = ref(false)
const restartTimedOut = ref(false)
let currentWs: WebSocket | null = null

const currentVersion = computed(() => systemInfo.value?.panel_version || '')
const latestVersion = computed(() => versions.value?.[0]?.version || '')

// 更新开始后仅展示进度，隐藏版本对比与更新日志
const showProgress = computed(() => updating.value || !!errorMsg.value)

const parseDescription = (text: string): string[] =>
  text.split('\n').filter((line) => line.trim())

const sleep = (ms: number) => new Promise((resolve) => setTimeout(resolve, ms))

const reload = () => window.location.reload()

// 升级会重启面板，重启期间请求失败（连接拒绝或 503）；静默轮询直到新版本就绪后刷新
const waitForRestart = async (targetVersion: string) => {
  waitingRestart.value = true
  const started = Date.now()
  while (Date.now() - started < 120000) {
    await sleep(3000)
    try {
      const data: any = await http.Get('/home/system_info', { meta: { noAlert: true } })
      if (data && (!targetVersion || data.panel_version === targetVersion)) {
        reload()
        return
      }
    } catch {
      // 面板重启中，继续等待
    }
  }
  restartTimedOut.value = true
}

const resetUpdate = () => {
  updating.value = false
  progressLogs.value = []
  errorMsg.value = ''
  waitingRestart.value = false
  restartTimedOut.value = false
}

const startUpdate = () => {
  resetUpdate()
  updating.value = true
  const targetVersion = latestVersion.value

  ws.panelUpdate()
    .then((socket) => {
      currentWs = socket
      socket.onmessage = (event) => {
        let data
        try {
          data = JSON.parse(event.data)
        } catch {
          return
        }
        if (data.status === 'progress') {
          progressLogs.value.push(data.msg)
        } else if (data.status === 'error') {
          errorMsg.value = data.msg
        } else if (data.status === 'success') {
          waitForRestart(targetVersion)
        }
      }
      socket.onclose = () => {
        currentWs = null
        // 连接断开且未报错/未进入等待，可能面板已重启，转入等待重启轮询
        if (updating.value && !waitingRestart.value && !errorMsg.value) {
          waitForRestart(targetVersion)
        }
      }
      socket.onerror = () => {
        currentWs = null
        if (updating.value && !waitingRestart.value && !errorMsg.value) {
          errorMsg.value = $gettext('WebSocket connection failed')
        }
      }
    })
    .catch(() => {
      errorMsg.value = $gettext('WebSocket connection failed')
    })
}

const handleUpdate = () => {
  window.$dialog.warning({
    title: $gettext('Update Panel'),
    content: $gettext('Are you sure you want to update the panel?'),
    positiveText: $gettext('Confirm'),
    negativeText: $gettext('Cancel'),
    onPositiveClick: startUpdate,
  })
}
</script>

<template>
  <PageContainer :show-footer="true">
    <!-- 更新进度 -->
    <n-card v-if="showProgress" :segmented="true" size="small">
      <n-flex vertical :size="20" py-4 items-center>
        <n-flex align="center" :size="12" justify="center">
          <n-spin v-if="!errorMsg && !restartTimedOut" :size="22" />
          <n-text v-else-if="errorMsg" type="error">
            <the-icon icon="mdi:alert-circle-outline" :size="26" />
          </n-text>
          <n-text v-else type="warning">
            <the-icon icon="mdi:alert-outline" :size="26" />
          </n-text>
          <n-text class="text-lg font-medium">
            <template v-if="errorMsg">{{ $gettext('Update failed') }}</template>
            <template v-else-if="restartTimedOut">{{ $gettext('Update timed out') }}</template>
            <template v-else>{{ $gettext('Updating to v%{ v }...', { v: latestVersion }) }}</template>
          </n-text>
        </n-flex>

        <n-timeline v-if="progressLogs.length || errorMsg" style="width: 100%; max-width: 460px">
          <n-timeline-item
            v-for="(log, i) in progressLogs"
            :key="i"
            type="success"
            :content="log"
          />
          <n-timeline-item
            v-if="waitingRestart && !restartTimedOut"
            type="info"
            :content="$gettext('Panel is restarting, please wait...')"
          />
          <n-timeline-item
            v-if="restartTimedOut"
            type="warning"
            :content="
              $gettext('Update may have failed, please check the panel logs and refresh manually.')
            "
          />
          <n-timeline-item
            v-if="errorMsg"
            type="error"
            :title="$gettext('Error')"
            :content="errorMsg"
          />
        </n-timeline>

        <n-flex v-if="errorMsg || restartTimedOut" justify="center">
          <n-button v-if="errorMsg" @click="resetUpdate">{{ $gettext('Back') }}</n-button>
          <n-button v-if="restartTimedOut" type="primary" @click="reload">
            {{ $gettext('Refresh') }}
          </n-button>
        </n-flex>
      </n-flex>
    </n-card>

    <!-- 版本对比 + 更新日志 -->
    <template v-else>
      <n-card :segmented="true" size="small">
        <n-flex vertical align="center" :size="18" py-6>
          <n-text type="success">
            <the-icon icon="mdi:rocket-launch-outline" :size="46" />
          </n-text>
          <n-text class="text-lg font-medium">{{ $gettext('A new version is available') }}</n-text>
          <n-flex align="center" :size="28" py-1>
            <n-flex vertical align="center" :size="6">
              <n-text depth="3" class="text-xs">{{ $gettext('Current') }}</n-text>
              <n-tag round :bordered="false" size="large">v{{ currentVersion }}</n-tag>
            </n-flex>
            <n-text depth="3">
              <the-icon icon="mdi:arrow-right-thin" :size="30" />
            </n-text>
            <n-flex vertical align="center" :size="6">
              <n-text depth="3" class="text-xs">{{ $gettext('Latest') }}</n-text>
              <n-tag type="success" round :bordered="false" size="large">v{{ latestVersion }}</n-tag>
            </n-flex>
          </n-flex>
          <n-button type="primary" size="large" @click="handleUpdate">
            <template #icon>
              <the-icon icon="mdi:download" :size="18" />
            </template>
            {{ $gettext('Update Now') }}
          </n-button>
        </n-flex>
      </n-card>

      <n-flex vertical :size="12" mt-4>
        <n-text depth="2" class="text-sm font-medium" px-1>{{ $gettext('Changelog') }}</n-text>
        <n-card v-for="(item, index) in versions" :key="index" :segmented="true" size="small">
          <template #header>
            <n-flex align="center" :size="8">
              <n-text class="font-medium">v{{ item.version }}</n-text>
              <n-tag v-if="index === 0" type="success" size="small" round :bordered="false">
                {{ $gettext('Latest') }}
              </n-tag>
              <n-tag size="small" round :bordered="false">{{ item.type }}</n-tag>
            </n-flex>
          </template>
          <template #header-extra>
            <n-text depth="3" class="text-xs">{{ formatDateTime(item.updated_at) }}</n-text>
          </template>
          <n-ol p-0 pl-5>
            <n-li v-for="(line, i) in parseDescription(item.description)" :key="i">{{ line }}</n-li>
          </n-ol>
        </n-card>
      </n-flex>
    </template>
  </PageContainer>
</template>

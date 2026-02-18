<script setup lang="ts">
import { useGettext } from 'vue3-gettext'

import website from '@/api/panel/website'

import ClientsTab from './stats/ClientsTab.vue'
import ErrorsTab from './stats/ErrorsTab.vue'
import IPsTab from './stats/IPsTab.vue'
import OverviewTab from './stats/OverviewTab.vue'
import SitesTab from './stats/SitesTab.vue'
import SpidersTab from './stats/SpidersTab.vue'
import URIsTab from './stats/URIsTab.vue'

const { $gettext } = useGettext()

// ============ 时间预设 ============

type TimePreset = 'today' | 'yesterday' | '7d' | '30d' | 'custom'

const activePreset = ref<TimePreset>('today')
const customRange = ref<[number, number] | null>(null)
const customPopover = ref(false)
const tempRange = ref<[number, number] | null>(null)

function getDateStr(d: Date): string {
  return `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}-${String(d.getDate()).padStart(2, '0')}`
}

const dateRange = computed<{ start: string; end: string }>(() => {
  const now = new Date()
  const todayDate = new Date(now.getFullYear(), now.getMonth(), now.getDate())

  switch (activePreset.value) {
    case 'today':
      return { start: getDateStr(todayDate), end: getDateStr(todayDate) }
    case 'yesterday': {
      const y = new Date(todayDate.getTime() - 86400000)
      return { start: getDateStr(y), end: getDateStr(y) }
    }
    case '7d': {
      const s = new Date(todayDate.getTime() - 6 * 86400000)
      return { start: getDateStr(s), end: getDateStr(todayDate) }
    }
    case '30d': {
      const s = new Date(todayDate.getTime() - 29 * 86400000)
      return { start: getDateStr(s), end: getDateStr(todayDate) }
    }
    case 'custom': {
      if (customRange.value) {
        return {
          start: getDateStr(new Date(customRange.value[0])),
          end: getDateStr(new Date(customRange.value[1]))
        }
      }
      return { start: getDateStr(todayDate), end: getDateStr(todayDate) }
    }
    default:
      return { start: getDateStr(todayDate), end: getDateStr(todayDate) }
  }
})

function setPreset(preset: TimePreset) {
  activePreset.value = preset
}

function confirmCustom() {
  if (tempRange.value) {
    customRange.value = tempRange.value
    activePreset.value = 'custom'
  }
  customPopover.value = false
}

// ============ 站点选择器 ============

const selectedSites = ref<string[]>([])

const sitesParam = computed(() => {
  return selectedSites.value.length > 0 ? selectedSites.value.join(',') : ''
})

// ============ 站点列表 ============

const siteOptions = ref<Array<{ label: string; value: string }>>([])

// ============ 清空数据 ============

const handleClear = () => {
  useRequest(website.statClear()).onSuccess(() => {
    window.$message.success($gettext('Cleared successfully'))
  })
}

// ============ Tab ============

const activeTab = ref('overview')

// ============ provide/inject 共享状态 ============

provide('statContext', {
  dateRange,
  sitesParam,
  siteOptions,
  activePreset
})
</script>

<template>
  <n-flex vertical :size="20">
    <!-- 共享工具栏 -->
    <div class="flex w-full flex-wrap items-center gap-12">
      <!-- 站点选择器 -->
      <n-select
        v-model:value="selectedSites"
        multiple
        clearable
        :options="siteOptions"
        :placeholder="$gettext('All Sites')"
        style="min-width: 200px; max-width: 400px"
      />

      <!-- 时间预设 -->
      <n-button-group size="small">
        <n-button
          :type="activePreset === 'today' ? 'primary' : 'default'"
          @click="setPreset('today')"
        >
          {{ $gettext('Today') }}
        </n-button>
        <n-button
          :type="activePreset === 'yesterday' ? 'primary' : 'default'"
          @click="setPreset('yesterday')"
        >
          {{ $gettext('Yesterday') }}
        </n-button>
        <n-button
          :type="activePreset === '7d' ? 'primary' : 'default'"
          @click="setPreset('7d')"
        >
          {{ $gettext('Last 7 Days') }}
        </n-button>
        <n-button
          :type="activePreset === '30d' ? 'primary' : 'default'"
          @click="setPreset('30d')"
        >
          {{ $gettext('Last 30 Days') }}
        </n-button>
        <n-popover
          v-model:show="customPopover"
          trigger="click"
          placement="bottom-end"
          :show-arrow="false"
        >
          <template #trigger>
            <n-button :type="activePreset === 'custom' ? 'primary' : 'default'">
              {{ $gettext('Custom') }}
            </n-button>
          </template>
          <n-date-picker
            v-model:value="tempRange"
            type="daterange"
            panel
            :default-time="['00:00:00', '23:59:59']"
            :actions="['confirm']"
            @confirm="confirmCustom"
          />
        </n-popover>
      </n-button-group>

      <div class="ml-auto">
        <n-popconfirm @positive-click="handleClear">
          <template #trigger>
            <n-button type="error" ghost size="small">
              {{ $gettext('Clear Data') }}
            </n-button>
          </template>
          {{ $gettext('Are you sure you want to clear all statistics data?') }}
        </n-popconfirm>
      </div>
    </div>

    <!-- Tab 导航 -->
    <n-tabs v-model:value="activeTab" type="line">
      <n-tab-pane name="overview" :tab="$gettext('Overview')">
        <OverviewTab />
      </n-tab-pane>
      <n-tab-pane name="sites" :tab="$gettext('Sites')">
        <SitesTab />
      </n-tab-pane>
      <n-tab-pane name="spiders" :tab="$gettext('Spiders')">
        <SpidersTab />
      </n-tab-pane>
      <n-tab-pane name="clients" :tab="$gettext('Clients')">
        <ClientsTab />
      </n-tab-pane>
      <n-tab-pane name="ips" :tab="$gettext('IPs')">
        <IPsTab />
      </n-tab-pane>
      <n-tab-pane name="uris" :tab="$gettext('URIs')">
        <URIsTab />
      </n-tab-pane>
      <n-tab-pane name="errors" :tab="$gettext('Errors')">
        <ErrorsTab />
      </n-tab-pane>
    </n-tabs>
  </n-flex>
</template>

<script setup lang="ts">
import type { EChartsOption } from 'echarts'
import { LineChart } from 'echarts/charts'
import { GridComponent, LegendComponent, TooltipComponent } from 'echarts/components'
import { use } from 'echarts/core'
import { CanvasRenderer } from 'echarts/renderers'
import VChart from 'vue-echarts'
import { useGettext } from 'vue3-gettext'

import website from '@/api/panel/website'
import { formatBytes } from '@/utils/file'

const { $gettext } = useGettext()

use([CanvasRenderer, LineChart, TooltipComponent, LegendComponent, GridComponent])

const ctx = inject<any>('statContext')!

// ============ 数据加载 ============

interface StatTotals {
  pv: number
  uv: number
  ip: number
  bandwidth: number
  requests: number
  errors: number
  spiders: number
}

interface SeriesPoint {
  key: string
  pv: number
  uv: number
  ip: number
  bandwidth: number
  requests: number
  errors: number
  spiders: number
}

interface OverviewData {
  current: StatTotals
  previous: StatTotals
  series: SeriesPoint[]
  previous_series: SeriesPoint[]
  sites: Array<{ id: number; name: string }>
}

const emptyTotals: StatTotals = {
  pv: 0,
  uv: 0,
  ip: 0,
  bandwidth: 0,
  requests: 0,
  errors: 0,
  spiders: 0
}

const overview = ref<OverviewData>({
  current: { ...emptyTotals },
  previous: { ...emptyTotals },
  series: [],
  previous_series: [],
  sites: []
})
const loading = ref(false)

const loadOverview = () => {
  loading.value = true
  useRequest(
    website.statOverview(ctx.dateRange.value.start, ctx.dateRange.value.end, ctx.sitesParam.value)
  )
    .onSuccess(({ data }: any) => {
      overview.value = data
    })
    .onComplete(() => {
      loading.value = false
    })
}

// 实时数据
const realtime = ref({ bandwidth: 0, rps: 0 })
let pollTimer: ReturnType<typeof setInterval> | null = null

const loadRealtime = () => {
  useRequest(website.statRealtime()).onSuccess(({ data }: any) => {
    realtime.value = data
  })
}

watch([() => ctx.dateRange.value, () => ctx.sitesParam.value], () => {
  loadOverview()
})

onMounted(() => {
  loadOverview()
  loadRealtime()
  pollTimer = setInterval(loadRealtime, 5000)
})

onUnmounted(() => {
  if (pollTimer) clearInterval(pollTimer)
})

// ============ 指标定义 ============

type MetricKey = 'pv' | 'uv' | 'ip' | 'bandwidth' | 'requests' | 'errors' | 'spiders'

const metrics: Array<{ key: MetricKey; label: string; isBytes?: boolean }> = [
  { key: 'pv', label: 'PV' },
  { key: 'uv', label: 'UV' },
  { key: 'ip', label: 'IP' },
  { key: 'bandwidth', label: $gettext('Bandwidth'), isBytes: true },
  { key: 'requests', label: $gettext('Requests') },
  { key: 'errors', label: $gettext('Errors') },
  { key: 'spiders', label: $gettext('Spiders') }
]

// ============ 统计卡片 ============

function formatValue(value: number, isBytes?: boolean): string {
  if (isBytes) return formatBytes(value)
  if (value >= 100000000) return (value / 1000000).toFixed(1) + 'M'
  if (value >= 100000) return (value / 1000).toFixed(1) + 'K'
  return String(value)
}

function formatDiff(
  current: number,
  previous: number
): { text: string; type: 'up' | 'down' | 'same' } {
  if (previous === 0) {
    if (current === 0) return { text: '-', type: 'same' }
    return { text: '+100%', type: 'up' }
  }
  const diff = ((current - previous) / previous) * 100
  if (diff === 0) return { text: '0%', type: 'same' }
  if (diff > 0) return { text: `+${diff.toFixed(1)}%`, type: 'up' }
  return { text: `${diff.toFixed(1)}%`, type: 'down' }
}

// ============ 图表 ============

const activeMetric = ref<MetricKey>('pv')
const showPrevious = ref(true)

const isSingleDay = computed(() => ctx.dateRange.value.start === ctx.dateRange.value.end)

function formatXLabel(key: string): string {
  if (isSingleDay.value) {
    return `${key.padStart(2, '0')}:00`
  }
  return key.slice(5)
}

const chartOption = computed<EChartsOption>(() => {
  const series = overview.value.series || []
  const prevSeries = overview.value.previous_series || []
  const metric = activeMetric.value
  const isBytes = metrics.find((m) => m.key === metric)?.isBytes

  const xData = series.map((s) => formatXLabel(s.key))
  const currentData = series.map((s) => s[metric] || 0)
  const previousData = prevSeries.map((s) => s[metric] || 0)

  const chartSeries: any[] = [
    {
      name: $gettext('Current Period'),
      type: 'line',
      smooth: true,
      symbol: 'none',
      areaStyle: { opacity: 0.15 },
      lineStyle: { width: 2 },
      data: currentData
    }
  ]

  if (showPrevious.value && previousData.length > 0) {
    chartSeries.push({
      name: $gettext('Previous Period'),
      type: 'line',
      smooth: true,
      symbol: 'none',
      lineStyle: { width: 1.5, type: 'dashed' },
      areaStyle: { opacity: 0.05 },
      data: previousData
    })
  }

  return {
    tooltip: {
      trigger: 'axis',
      formatter: (params: any) => {
        if (!Array.isArray(params) || params.length === 0) return ''
        let html = `<div style="font-size:12px"><div style="margin-bottom:4px;font-weight:bold">${params[0].name}</div>`
        for (const p of params) {
          const val = isBytes ? formatBytes(p.value) : p.value.toLocaleString()
          html += `<div>${p.marker} ${p.seriesName}: ${val}</div>`
        }
        html += '</div>'
        return html
      }
    },
    legend: { top: 0, right: 0 },
    grid: { left: 60, right: 20, top: 40, bottom: 30 },
    xAxis: {
      type: 'category',
      boundaryGap: false,
      data: xData,
      axisLabel: {
        interval: isSingleDay.value
          ? (_: number, value: string) => {
              const hour = parseInt(value.split(':')[0] || '0', 10)
              return hour % 3 === 0
            }
          : 'auto'
      }
    },
    yAxis: {
      type: 'value',
      axisLabel: { formatter: isBytes ? (v: number) => formatBytes(v) : undefined },
      splitLine: { lineStyle: { type: 'dashed' } }
    },
    series: chartSeries
  }
})

// ============ 对比周期标签 ============

const previousLabel = computed(() => {
  const preset = ctx.activePreset?.value
  if (preset === 'today') return $gettext('Yesterday')
  if (preset === 'yesterday') return $gettext('Day Before Yesterday')
  return $gettext('Previous Period')
})
</script>

<template>
  <n-flex vertical :size="20">
    <!-- 统计卡片 -->
    <n-spin :show="loading">
      <div class="grid grid-cols-3 gap-12 sm:grid-cols-5 lg:grid-cols-9">
        <n-card v-for="m in metrics" :key="m.key" :bordered="false" size="small">
          <div class="flex flex-col gap-4">
            <span class="text-12px text-[var(--text-color-3)]">{{ m.label }}</span>
            <span class="text-20px font-bold">{{
              formatValue(overview.current[m.key] || 0, m.isBytes)
            }}</span>
            <div class="text-12px flex items-center gap-4">
              <span
                :class="{
                  'text-[var(--success-color)]':
                    formatDiff(overview.current[m.key] || 0, overview.previous[m.key] || 0).type ===
                    'up',
                  'text-[var(--error-color)]':
                    formatDiff(overview.current[m.key] || 0, overview.previous[m.key] || 0).type ===
                    'down',
                  'text-[var(--text-color-3)]':
                    formatDiff(overview.current[m.key] || 0, overview.previous[m.key] || 0).type ===
                    'same'
                }"
              >
                {{
                  formatDiff(overview.current[m.key] || 0, overview.previous[m.key] || 0).text
                }}
              </span>
              <span class="text-[var(--text-color-3)]">{{ previousLabel }}</span>
            </div>
          </div>
        </n-card>
        <n-card :bordered="false" size="small">
          <div class="flex flex-col gap-4">
            <span class="text-12px text-[var(--text-color-3)]">{{
              $gettext('Realtime Bandwidth')
            }}</span>
            <span class="text-20px font-bold">{{ formatBytes(realtime.bandwidth) }}/s</span>
          </div>
        </n-card>
        <n-card :bordered="false" size="small">
          <div class="flex flex-col gap-4">
            <span class="text-12px text-[var(--text-color-3)]">RPS</span>
            <span class="text-20px font-bold">{{ realtime.rps.toFixed(1) }}</span>
          </div>
        </n-card>
      </div>
    </n-spin>

    <!-- 趋势图 -->
    <n-card :bordered="false">
      <template #header>
        <div class="flex flex-wrap items-center justify-between gap-8">
          <n-radio-group v-model:value="activeMetric" size="small">
            <n-radio-button v-for="m in metrics" :key="m.key" :value="m.key">
              {{ m.label }}
            </n-radio-button>
          </n-radio-group>
          <n-checkbox v-model:checked="showPrevious">
            {{ previousLabel }}
          </n-checkbox>
        </div>
      </template>
      <n-spin :show="loading">
        <v-chart class="h-350px" :option="chartOption" autoresize />
      </n-spin>
    </n-card>
  </n-flex>
</template>

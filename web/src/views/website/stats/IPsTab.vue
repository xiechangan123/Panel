<script setup lang="ts">
import type { EChartsOption } from 'echarts'
import { BarChart, PieChart } from 'echarts/charts'
import { GridComponent, LegendComponent, TooltipComponent } from 'echarts/components'
import { use } from 'echarts/core'
import { CanvasRenderer } from 'echarts/renderers'
import VChart from 'vue-echarts'
import { useGettext } from 'vue3-gettext'

import website from '@/api/panel/website'
import { formatBytes } from '@/utils/file'

const { $gettext } = useGettext()

use([CanvasRenderer, BarChart, PieChart, TooltipComponent, LegendComponent, GridComponent])

const ctx = inject<any>('statContext')!

const loading = ref(false)
const items = ref<any[]>([])
const total = ref(0)
const page = ref(1)
const limit = ref(50)

const loadData = () => {
  loading.value = true
  useRequest(
    website.statIPs(
      ctx.dateRange.value.start,
      ctx.dateRange.value.end,
      ctx.sitesParam.value,
      page.value,
      limit.value
    )
  )
    .onSuccess(({ data }: any) => {
      items.value = data.items || []
      total.value = data.total || 0
    })
    .onComplete(() => {
      loading.value = false
    })
}

watch([() => ctx.dateRange.value, () => ctx.sitesParam.value], () => {
  page.value = 1
  loadData()
})

onMounted(() => {
  loadData()
})

// ========== 国家饼图 ==========

const countryPieOption = computed<EChartsOption>(() => {
  const countryMap = new Map<string, number>()
  for (const item of items.value) {
    const name = item.country || $gettext('Unknown')
    countryMap.set(name, (countryMap.get(name) || 0) + item.requests)
  }
  const data = [...countryMap.entries()]
    .map(([name, requests]) => ({ name, value: requests }))
    .sort((a, b) => b.value - a.value)
    .slice(0, 10)

  return {
    tooltip: {
      trigger: 'item',
      formatter: '{b}: {c} ({d}%)'
    },
    legend: { orient: 'vertical', right: 10, top: 'center' },
    series: [
      {
        type: 'pie',
        radius: ['40%', '70%'],
        center: ['35%', '50%'],
        avoidLabelOverlap: false,
        label: { show: false },
        data
      }
    ]
  }
})

// ========== 省份柱状图 ==========

const regionBarOption = computed<EChartsOption>(() => {
  const regionMap = new Map<string, { requests: number; bandwidth: number }>()
  for (const item of items.value) {
    const name = item.region || item.country || $gettext('Unknown')
    const existing = regionMap.get(name)
    if (existing) {
      existing.requests += item.requests
      existing.bandwidth += item.bandwidth
    } else {
      regionMap.set(name, { requests: item.requests, bandwidth: item.bandwidth })
    }
  }
  const sorted = [...regionMap.entries()]
    .map(([name, v]) => ({ name, ...v }))
    .sort((a, b) => b.requests - a.requests)
    .slice(0, 15)
  const reversed = [...sorted].reverse()

  return {
    tooltip: {
      trigger: 'axis',
      axisPointer: { type: 'shadow' },
      formatter: (params: any) => {
        if (!Array.isArray(params) || params.length === 0) return ''
        const idx = sorted.length - 1 - params[0].dataIndex
        const item = sorted[idx]
        if (!item) return ''
        return `${params[0].name}<br/>${$gettext('Requests')}: ${params[0].value.toLocaleString()}<br/>${$gettext('Bandwidth')}: ${formatBytes(item.bandwidth)}`
      }
    },
    grid: { left: 100, right: 40, top: 10, bottom: 30 },
    xAxis: { type: 'value' },
    yAxis: {
      type: 'category',
      data: reversed.map((i) => i.name),
      axisLabel: { width: 80, overflow: 'truncate' }
    },
    series: [
      {
        type: 'bar',
        data: reversed.map((i) => i.requests),
        barMaxWidth: 30
      }
    ]
  }
})

// ========== 表格 ==========

const formatGeo = (row: any) => {
  const parts = [row.country, row.region, row.city].filter((s) => s && s !== '')
  return parts.join(' ') || '-'
}

const columns = computed(() => [
  { title: 'IP', key: 'ip', ellipsis: { tooltip: true } },
  {
    title: $gettext('Location'),
    key: 'country',
    render: (row: any) => formatGeo(row),
    ellipsis: { tooltip: true }
  },
  {
    title: $gettext('Requests'),
    key: 'requests',
    sorter: (a: any, b: any) => a.requests - b.requests
  },
  {
    title: $gettext('Bandwidth'),
    key: 'bandwidth',
    render: (row: any) => formatBytes(row.bandwidth),
    sorter: (a: any, b: any) => a.bandwidth - b.bandwidth
  }
])

const handlePageChange = (p: number) => {
  page.value = p
  loadData()
}

const handlePageSizeChange = (s: number) => {
  limit.value = s
  page.value = 1
  loadData()
}
</script>

<template>
  <n-flex vertical :size="20">
    <n-spin :show="loading">
      <div class="gap-12 grid grid-cols-1 lg:grid-cols-2" v-if="items.length > 0">
        <n-card :bordered="false" :title="$gettext('Country Distribution')">
          <v-chart class="h-300px" :option="countryPieOption" autoresize />
        </n-card>
        <n-card :bordered="false" :title="$gettext('Region Distribution')">
          <v-chart class="h-300px" :option="regionBarOption" autoresize />
        </n-card>
      </div>
      <n-data-table :columns="columns" :data="items" :bordered="false" size="small" />
      <n-flex justify="end" class="mt-12" v-if="total > 0">
        <n-pagination
          v-model:page="page"
          :page-size="limit"
          :item-count="total"
          :page-sizes="[20, 50, 100]"
          show-size-picker
          @update:page="handlePageChange"
          @update:page-size="handlePageSizeChange"
        />
      </n-flex>
    </n-spin>
  </n-flex>
</template>

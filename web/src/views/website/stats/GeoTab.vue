<script setup lang="ts">
import type { EChartsOption } from 'echarts'
import { BarChart } from 'echarts/charts'
import { GridComponent, TooltipComponent } from 'echarts/components'
import { use } from 'echarts/core'
import { CanvasRenderer } from 'echarts/renderers'
import VChart from 'vue-echarts'
import { useGettext } from 'vue3-gettext'

import website from '@/api/panel/website'
import { formatBytes } from '@/utils/file'

const { $gettext } = useGettext()

use([CanvasRenderer, BarChart, TooltipComponent, GridComponent])

const ctx = inject<any>('statContext')!

const loading = ref(false)
const items = ref<any[]>([])
const groupBy = ref('country')

const loadData = () => {
  loading.value = true
  useRequest(
    website.statGeos(
      ctx.dateRange.value.start,
      ctx.dateRange.value.end,
      ctx.sitesParam.value,
      groupBy.value,
      undefined,
      100
    )
  )
    .onSuccess(({ data }: any) => {
      items.value = data.items || []
    })
    .onComplete(() => {
      loading.value = false
    })
}

watch([() => ctx.dateRange.value, () => ctx.sitesParam.value, () => ctx.refreshKey.value], () => {
  loadData()
})

watch(groupBy, () => {
  loadData()
})

onMounted(() => {
  loadData()
})

// ========== 表格 ==========

const nameColumn = computed(() => {
  return groupBy.value === 'region'
    ? { title: $gettext('Region'), key: 'region' }
    : { title: $gettext('Country'), key: 'country' }
})

const columns = computed(() => [
  {
    title: nameColumn.value.title,
    key: nameColumn.value.key,
    render: (row: any) => row[nameColumn.value.key] || $gettext('Unknown')
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

// ========== 图表 ==========

const chartTitle = computed(() => {
  return groupBy.value === 'region'
    ? $gettext('Region Distribution')
    : $gettext('Country Distribution')
})

const barChartOption = computed<EChartsOption>(() => {
  const data = items.value.slice(0, 15)
  const reversed = [...data].reverse()
  const key = nameColumn.value.key
  return {
    tooltip: {
      trigger: 'axis',
      axisPointer: { type: 'shadow' },
      formatter: (params: any) => {
        if (!Array.isArray(params) || params.length === 0) return ''
        const idx = data.length - 1 - params[0].dataIndex
        const item = data[idx]
        if (!item) return ''
        return `${params[0].name}<br/>${$gettext('Requests')}: ${params[0].value.toLocaleString()}<br/>${$gettext('Bandwidth')}: ${formatBytes(item.bandwidth)}`
      }
    },
    grid: { left: 100, right: 40, top: 10, bottom: 30 },
    xAxis: { type: 'value' },
    yAxis: {
      type: 'category',
      data: reversed.map((i: any) => i[key] || $gettext('Unknown')),
      axisLabel: { width: 80, overflow: 'truncate' }
    },
    series: [
      {
        type: 'bar',
        data: reversed.map((i: any) => i.requests),
        barMaxWidth: 30
      }
    ]
  }
})
</script>

<template>
  <n-flex vertical :size="20">
    <n-spin :show="loading">
      <!-- 维度切换 -->
      <n-flex align="center" class="mb-12">
        <n-radio-group v-model:value="groupBy" size="small">
          <n-radio-button value="country">{{ $gettext('Country') }}</n-radio-button>
          <n-radio-button value="region">{{ $gettext('Region') }}</n-radio-button>
        </n-radio-group>
      </n-flex>

      <!-- 图表 -->
      <n-card :bordered="false" :title="chartTitle" v-if="items.length > 0">
        <v-chart class="h-300px" :option="barChartOption" autoresize />
      </n-card>

      <!-- 表格 -->
      <n-data-table :columns="columns" :data="items" :bordered="false" size="small" />
    </n-spin>
  </n-flex>
</template>

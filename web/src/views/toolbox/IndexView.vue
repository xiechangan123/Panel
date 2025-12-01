<script setup lang="ts">
defineOptions({
  name: 'toolbox-index'
})

import BenchmarkView from '@/views/toolbox/BenchmarkView.vue'
import ProcessView from '@/views/toolbox/ProcessView.vue'
import SystemView from '@/views/toolbox/SystemView.vue'
import { useGettext } from 'vue3-gettext'

const { $gettext } = useGettext()
const current = ref('process')
</script>

<template>
  <common-page show-header show-footer>
    <template #tabbar>
      <n-tabs v-model:value="current" animated>
        <n-tab name="process" :tab="$gettext('Process')" />
        <n-tab name="system" :tab="$gettext('System')" />
        <n-tab name="benchmark" :tab="$gettext('Benchmark')" />
      </n-tabs>
    </template>
    <n-flex vertical>
      <process-view v-if="current === 'process'" />
      <system-view v-if="current === 'system'" />
      <benchmark-view v-if="current === 'benchmark'" />
    </n-flex>
  </common-page>
</template>

<style scoped lang="scss"></style>

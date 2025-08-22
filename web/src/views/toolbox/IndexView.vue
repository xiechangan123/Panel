<script setup lang="ts">
defineOptions({
  name: 'toolbox-index'
})

import BenchmarkView from '@/views/toolbox/BenchmarkView.vue'
import SystemView from '@/views/toolbox/SystemView.vue'
import { useGettext } from 'vue3-gettext'

const { $gettext } = useGettext()
const current = ref('system')
</script>

<template>
  <common-page show-header show-footer>
    <template #tabbar>
      <n-tabs v-model:value="current" animated>
        <n-tab name="system" :tab="$gettext('System')" />
        <n-tab name="benchmark" :tab="$gettext('Benchmark')" />
      </n-tabs>
    </template>
    <n-flex vertical>
      <system-view v-if="current === 'system'" />
      <benchmark-view v-if="current === 'benchmark'" />
    </n-flex>
  </common-page>
</template>

<style scoped lang="scss"></style>

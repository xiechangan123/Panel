<script setup lang="ts">
defineOptions({
  name: 'firewall-index'
})

import ForwardView from '@/views/firewall/ForwardView.vue'
import IpRuleView from '@/views/firewall/IpRuleView.vue'
import RuleView from '@/views/firewall/RuleView.vue'
import ScanView from '@/views/firewall/ScanView.vue'
import SettingView from '@/views/firewall/SettingView.vue'
import { useGettext } from 'vue3-gettext'

const { $gettext } = useGettext()
const currentTab = ref('scan')
</script>

<template>
  <common-page show-header show-footer>
    <template #tabbar>
      <n-tabs v-model:value="currentTab" animated>
        <n-tab name="scan" :tab="$gettext('Scan Awareness')" />
        <n-tab name="rule" :tab="$gettext('Port Rules')" />
        <n-tab name="ip-rule" :tab="$gettext('IP Rules')" />
        <n-tab name="forward" :tab="$gettext('Port Forwarding')" />
        <n-tab name="setting" :tab="$gettext('Settings')" />
      </n-tabs>
    </template>
    <scan-view v-if="currentTab === 'scan'" />
    <rule-view v-if="currentTab === 'rule'" />
    <ip-rule-view v-if="currentTab === 'ip-rule'" />
    <forward-view v-if="currentTab === 'forward'" />
    <setting-view v-if="currentTab === 'setting'" />
  </common-page>
</template>

<style scoped lang="scss"></style>

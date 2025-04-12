<script setup lang="ts">
defineOptions({
  name: 'firewall-index'
})

import ForwardView from '@/views/firewall/ForwardView.vue'
import IpRuleView from '@/views/firewall/IpRuleView.vue'
import RuleView from '@/views/firewall/RuleView.vue'
import SettingView from '@/views/firewall/SettingView.vue'
import { useGettext } from 'vue3-gettext'

const { $gettext } = useGettext()
const currentTab = ref('rule')
</script>

<template>
  <common-page show-footer>
    <n-tabs v-model:value="currentTab" type="line" animated>
      <n-tab-pane name="rule" :tab="$gettext('Port Rules')">
        <rule-view />
      </n-tab-pane>
      <n-tab-pane name="ip-rule" :tab="$gettext('IP Rules')">
        <ip-rule-view />
      </n-tab-pane>
      <n-tab-pane name="forward" :tab="$gettext('Port Forwarding')">
        <forward-view />
      </n-tab-pane>
      <n-tab-pane name="setting" :tab="$gettext('Settings')">
        <setting-view />
      </n-tab-pane>
    </n-tabs>
  </common-page>
</template>

<style scoped lang="scss"></style>

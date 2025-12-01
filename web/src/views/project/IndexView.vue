<script lang="ts" setup>
defineOptions({
  name: 'project-index'
})

const currentTab = ref('general')
</script>

<template>
  <common-page show-header show-footer>
    <template #tabbar>
      <n-tabs v-model:value="currentTab" animated>
        <n-tab name="general" :tab="$gettext('General')" />
        <n-tab name="php" :tab="$gettext('PHP')" />
        <n-tab name="java" :tab="$gettext('Java')" />
        <n-tab name="python" :tab="$gettext('Python')" />
        <n-tab name="nodejs" :tab="$gettext('Node.js')" />
      </n-tabs>
    </template>
  </common-page>
</template>

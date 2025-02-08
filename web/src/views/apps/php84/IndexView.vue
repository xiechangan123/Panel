<script setup lang="ts">
defineOptions({
  name: 'apps-php84-index'
})

import PhpView from '@/views/apps/php/PhpView.vue'
</script>

<template>
  <php-view :version="84" />
</template>

<script setup lang="ts">
defineOptions({
  name: 'apps-php83-index'
})

import PhpView from '@/views/apps/php/PhpView.vue'
</script>

<template>
  <php-view :version="83" />
</template>

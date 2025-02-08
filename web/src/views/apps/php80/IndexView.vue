<script setup lang="ts">
defineOptions({
  name: 'apps-php80-index'
})

import PhpView from '@/views/apps/php/PhpView.vue'
</script>

<template>
  <php-view :version="80" />
</template>

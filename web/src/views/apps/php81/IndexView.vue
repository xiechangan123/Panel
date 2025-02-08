<script setup lang="ts">
defineOptions({
  name: 'apps-php81-index'
})

import PhpView from '@/views/apps/php/PhpView.vue'
</script>

<template>
  <php-view :version="81" />
</template>

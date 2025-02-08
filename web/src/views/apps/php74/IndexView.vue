<script setup lang="ts">
defineOptions({
  name: 'apps-php74-index'
})

import PhpView from '@/views/apps/php/PhpView.vue'
</script>

<template>
  <php-view :version="74" />
</template>

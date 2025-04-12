<script setup lang="ts">
import ComposeView from '@/views/container/ComposeView.vue'
import { useGettext } from 'vue3-gettext'

defineOptions({
  name: 'container-index'
})

import ContainerView from '@/views/container/ContainerView.vue'
import ImageView from '@/views/container/ImageView.vue'
import NetworkView from '@/views/container/NetworkView.vue'
import VolumeView from '@/views/container/VolumeView.vue'

const { $gettext } = useGettext()
const current = ref('container')
</script>

<template>
  <common-page show-footer>
    <n-tabs v-model:value="current" type="line" animated>
      <n-tab-pane name="container" :tab="$gettext('Containers')">
        <container-view />
      </n-tab-pane>
      <n-tab-pane name="compose" :tab="$gettext('Compose')">
        <compose-view />
      </n-tab-pane>
      <n-tab-pane name="image" :tab="$gettext('Images')">
        <image-view />
      </n-tab-pane>
      <n-tab-pane name="network" :tab="$gettext('Networks')">
        <network-view />
      </n-tab-pane>
      <n-tab-pane name="volume" :tab="$gettext('Volumes')">
        <volume-view />
      </n-tab-pane>
    </n-tabs>
  </common-page>
</template>

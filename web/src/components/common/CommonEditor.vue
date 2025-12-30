<script setup lang="ts">
import '@/utils/ace'
import { VAceEditor } from 'vue3-ace-editor'

const value = defineModel<string>('value', { type: String, required: true })
const props = defineProps({
  lang: {
    type: String,
    required: false,
    default: 'ini'
  },
  height: {
    type: String,
    required: false,
    default: '60vh'
  },
  readOnly: {
    type: Boolean,
    required: false
  }
})
</script>

<template>
  <v-ace-editor
    v-model:value="value"
    :lang="props.lang"
    :options="{ useWorker: true }"
    :readonly="props.readOnly"
    theme="monokai"
    :style="{ height: props.height }"
  />
</template>

<style scoped lang="scss"></style>

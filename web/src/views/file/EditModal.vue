<script setup lang="ts">
import FileEditor from '@/components/common/FileEditor.vue'
import { useGettext } from 'vue3-gettext'

const { $gettext } = useGettext()
const show = defineModel<boolean>('show', { type: Boolean, required: true })
const file = defineModel<string>('file', { type: String, required: true })
const editor = ref<any>(null)

const handleRefresh = () => {
  editor.value.get()
}

const handleSave = () => {
  editor.value.save()
}
</script>

<template>
  <n-modal
    v-model:show="show"
    preset="card"
    :title="$gettext('Edit - %{ file }', { file })"
    style="width: 60vw"
    size="huge"
    :bordered="false"
    :segmented="false"
  >
    <template #header-extra>
      <n-flex>
        <n-button @click="handleRefresh"> {{ $gettext('Refresh') }} </n-button>
        <n-button type="primary" @click="handleSave"> {{ $gettext('Save') }} </n-button>
      </n-flex>
    </template>
    <file-editor ref="editor" :path="file" :read-only="false" />
  </n-modal>
</template>

<style scoped lang="scss"></style>

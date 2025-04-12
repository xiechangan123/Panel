<script setup lang="ts">
import file from '@/api/panel/file'
import { decodeBase64 } from '@/utils'
import { languageByPath } from '@/utils/file'
import Editor from '@guolao/vue-monaco-editor'
import { useGettext } from 'vue3-gettext'

const { $gettext } = useGettext()
const props = defineProps({
  path: {
    type: String,
    required: true
  },
  readOnly: {
    type: Boolean,
    required: true
  }
})

const disabled = ref(false) // 在出现错误的情况下禁用保存
const content = ref('')

const get = () => {
  useRequest(file.content(props.path))
    .onSuccess(({ data }) => {
      content.value = decodeBase64(data.content)
      window.$message.success($gettext('Retrieved successfully'))
    })
    .onError(() => {
      disabled.value = true
    })
}

const save = () => {
  if (disabled.value) {
    window.$message.error($gettext('Cannot save in current state'))
    return
  }
  useRequest(file.save(props.path, content.value)).onSuccess(() => {
    window.$message.success($gettext('Saved successfully'))
  })
}

onMounted(() => {
  get()
})

defineExpose({
  get,
  save
})
</script>

<template>
  <Editor
    v-model:value="content"
    :language="languageByPath(props.path)"
    theme="vs-dark"
    height="60vh"
    :options="{
      automaticLayout: true,
      formatOnType: true,
      formatOnPaste: true,
      wordWrap: 'on'
    }"
  />
</template>

<style scoped lang="scss"></style>

<script setup lang="ts">
import file from '@/api/panel/file'
import { decodeBase64 } from '@/utils'
import { languageByPath } from '@/utils/file'
import Editor from '@guolao/vue-monaco-editor'

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
      window.$message.success('获取成功')
    })
    .onError(() => {
      disabled.value = true
    })
}

const save = () => {
  if (disabled.value) {
    window.$message.error('当前状态下不可保存')
    return
  }
  useRequest(file.save(props.path, content.value)).onSuccess(() => {
    window.$message.success('保存成功')
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

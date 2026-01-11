<script setup lang="ts">
import file from '@/api/panel/file'
import { FileEditorView } from '@/components/file-editor'
import { useEditorStore } from '@/store'
import { decodeBase64 } from '@/utils'
import { useGettext } from 'vue3-gettext'

const { $gettext } = useGettext()
const editorStore = useEditorStore()

const show = defineModel<boolean>('show', { type: Boolean, required: true })
const filePath = defineModel<string>('file', { type: String, required: true })

const editorRef = ref<InstanceType<typeof FileEditorView>>()

// 获取文件所在目录作为初始路径
const initialPath = computed(() => {
  if (!filePath.value) return '/'
  const parts = filePath.value.split('/')
  parts.pop()
  return parts.join('/') || '/'
})

// 打开时自动加载文件
watch(show, (newShow) => {
  if (newShow && filePath.value) {
    // 暂停文件管理的键盘快捷键
    window.$bus.emit('file:keyboard-pause')

    // 清空之前的标签页
    editorStore.closeAllTabs()
    // 设置根目录
    editorStore.setRootPath(initialPath.value)
    // 打开指定文件
    editorStore.openFile(filePath.value, '', 'utf-8')
    editorStore.setLoading(filePath.value, true)

    useRequest(file.content(encodeURIComponent(filePath.value)))
      .onSuccess(({ data }) => {
        const content = decodeBase64(data.content)
        editorStore.reloadFile(filePath.value, content)
      })
      .onError(() => {
        window.$message.error($gettext('Failed to load file'))
        editorStore.closeTab(filePath.value)
      })
      .onComplete(() => {
        editorStore.setLoading(filePath.value, false)
      })
  } else if (!newShow) {
    // 恢复文件管理的键盘快捷键
    window.$bus.emit('file:keyboard-resume')
  }
})
</script>

<template>
  <n-modal
    v-model:show="show"
    preset="card"
    :title="$gettext('File Editor')"
    style="width: 90vw; height: 85vh"
    content-style="padding: 0; height: calc(85vh - 60px); display: flex; flex-direction: column;"
    :bordered="false"
    :segmented="false"
  >
    <FileEditorView ref="editorRef" :initial-path="initialPath" />
  </n-modal>
</template>

<style scoped lang="scss"></style>

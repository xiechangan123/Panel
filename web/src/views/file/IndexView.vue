<script setup lang="ts">
defineOptions({
  name: 'file-index'
})

import { useFileStore } from '@/store'
import CompressModal from '@/views/file/CompressModal.vue'
import ListView from '@/views/file/ListView.vue'
import PathInput from '@/views/file/PathInput.vue'
import PermissionModal from '@/views/file/PermissionModal.vue'
import ToolBar from '@/views/file/ToolBar.vue'
import UploadModal from '@/views/file/UploadModal.vue'
import type { FileInfo, Marked } from '@/views/file/types'

const fileStore = useFileStore()

const selected = ref<string[]>([])
const marked = ref<Marked[]>([])
const markedType = ref<string>('copy')
// 权限编辑时的文件信息列表
const permissionFileInfoList = ref<FileInfo[]>([])

const compress = ref(false)
const permission = ref(false)

// 上传相关
const upload = ref(false)
const droppedFiles = ref<File[]>([])
const isDragging = ref(false)

// 处理拖拽进入
const handleDragEnter = (e: DragEvent) => {
  e.preventDefault()
  e.stopPropagation()
  // 检查是否有文件
  if (e.dataTransfer?.types.includes('Files')) {
    isDragging.value = true
  }
}

// 处理拖拽离开
const handleDragLeave = (e: DragEvent) => {
  e.preventDefault()
  e.stopPropagation()
  // 只有当离开整个容器时才隐藏
  const rect = (e.currentTarget as HTMLElement).getBoundingClientRect()
  if (
    e.clientX <= rect.left ||
    e.clientX >= rect.right ||
    e.clientY <= rect.top ||
    e.clientY >= rect.bottom
  ) {
    isDragging.value = false
  }
}

// 处理拖拽悬停
const handleDragOver = (e: DragEvent) => {
  e.preventDefault()
  e.stopPropagation()
}

// 递归读取目录中的所有文件
const readDirectoryRecursively = async (
  entry: FileSystemDirectoryEntry,
  basePath: string = ''
): Promise<File[]> => {
  const files: File[] = []
  const reader = entry.createReader()

  const readEntries = (): Promise<FileSystemEntry[]> => {
    return new Promise((resolve, reject) => {
      reader.readEntries(resolve, reject)
    })
  }

  let entries: FileSystemEntry[] = []
  // readEntries 可能需要多次调用才能获取所有条目
  let batch: FileSystemEntry[]
  do {
    batch = await readEntries()
    entries = entries.concat(batch)
  } while (batch.length > 0)

  for (const childEntry of entries) {
    const childPath = basePath ? `${basePath}/${childEntry.name}` : childEntry.name
    if (childEntry.isFile) {
      const fileEntry = childEntry as FileSystemFileEntry
      const file = await new Promise<File>((resolve, reject) => {
        fileEntry.file((f) => {
          // 创建带有相对路径的新 File 对象
          const newFile = new File([f], childPath, { type: f.type, lastModified: f.lastModified })
          resolve(newFile)
        }, reject)
      })
      files.push(file)
    } else if (childEntry.isDirectory) {
      const subFiles = await readDirectoryRecursively(
        childEntry as FileSystemDirectoryEntry,
        childPath
      )
      files.push(...subFiles)
    }
  }

  return files
}

// 处理拖拽放下
const handleDrop = async (e: DragEvent) => {
  e.preventDefault()
  e.stopPropagation()
  isDragging.value = false

  const items = e.dataTransfer?.items
  if (!items || items.length === 0) return

  const files: File[] = []

  // 使用 webkitGetAsEntry 来支持文件夹
  for (let i = 0; i < items.length; i++) {
    const item = items[i]
    if (item?.kind === 'file') {
      const entry = item.webkitGetAsEntry()
      if (entry) {
        if (entry.isFile) {
          const file = item.getAsFile()
          if (file) files.push(file)
        } else if (entry.isDirectory) {
          const dirFiles = await readDirectoryRecursively(
            entry as FileSystemDirectoryEntry,
            entry.name
          )
          files.push(...dirFiles)
        }
      }
    }
  }

  if (files.length > 0) {
    droppedFiles.value = files
    upload.value = true
  }
}

// 监听上传弹窗关闭，清空预拖入的文件
watch(upload, (val) => {
  if (!val) {
    droppedFiles.value = []
  }
})
</script>

<template>
  <common-page
    show-footer
    flex
    @dragenter="handleDragEnter"
    @dragleave="handleDragLeave"
    @dragover="handleDragOver"
    @drop="handleDrop"
  >
    <n-flex vertical :size="20" class="flex-1 min-h-0">
      <path-input
        v-model:path="fileStore.path"
        v-model:keyword="fileStore.keyword"
        v-model:sub="fileStore.sub"
      />
      <tool-bar
        v-model:path="fileStore.path"
        v-model:selected="selected"
        v-model:marked="marked"
        v-model:markedType="markedType"
        v-model:compress="compress"
        v-model:permission="permission"
        v-model:upload="upload"
      />
      <list-view
        v-model:path="fileStore.path"
        v-model:keyword="fileStore.keyword"
        v-model:sub="fileStore.sub"
        v-model:selected="selected"
        v-model:marked="marked"
        v-model:markedType="markedType"
        v-model:compress="compress"
        v-model:permission="permission"
        v-model:permission-file-info-list="permissionFileInfoList"
      />
      <compress-modal
        v-model:show="compress"
        v-model:path="fileStore.path"
        v-model:selected="selected"
      />
      <permission-modal
        v-model:show="permission"
        v-model:selected="selected"
        v-model:file-info-list="permissionFileInfoList"
      />
    </n-flex>

    <!-- 拖拽上传遮罩 -->
    <div v-if="isDragging" class="drag-overlay">
      <div class="drag-content">
        <the-icon icon="mdi:cloud-upload" :size="64" />
        <p>释放文件以上传</p>
      </div>
    </div>

    <!-- 上传弹窗 -->
    <upload-modal
      v-model:show="upload"
      v-model:path="fileStore.path"
      :initial-files="droppedFiles"
    />
  </common-page>
</template>

<style scoped lang="scss">
.drag-overlay {
  position: absolute;
  inset: 0;
  background: rgba(0, 0, 0, 0.5);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 1000;
  pointer-events: none;
}

.drag-content {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 16px;
  color: white;
  font-size: 18px;
}
</style>

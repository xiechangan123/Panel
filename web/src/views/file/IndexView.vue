<script setup lang="ts">
defineOptions({
  name: 'file-index'
})

import { useThemeVars } from 'naive-ui'
import draggable from 'vuedraggable'

import { useFileStore } from '@/store'
import type { FileTab } from '@/store/modules/file'
import CompressModal from '@/views/file/CompressModal.vue'
import ListView from '@/views/file/ListView.vue'
import PathInput from '@/views/file/PathInput.vue'
import PermissionModal from '@/views/file/PermissionModal.vue'
import ToolBar from '@/views/file/ToolBar.vue'
import UploadModal from '@/views/file/UploadModal.vue'
import type { FileInfo } from '@/views/file/types'

const fileStore = useFileStore()
const themeVars = useThemeVars()

const selected = ref<string[]>([])
// 权限编辑时的文件信息列表
const permissionFileInfoList = ref<FileInfo[]>([])

const compress = ref(false)
const permission = ref(false)

// 上传相关
const upload = ref(false)
const droppedFiles = ref<File[]>([])
const isDragging = ref(false)

// 拖拽排序用的本地副本
const localTabs = computed({
  get: () => fileStore.tabs,
  set: (val: FileTab[]) => fileStore.reorderTabs(val)
})

// 切换标签页时清空选中
watch(
  () => fileStore.activeTabId,
  () => {
    selected.value = []
  }
)

// 处理拖拽进入
const handleDragEnter = (e: DragEvent) => {
  e.preventDefault()
  e.stopPropagation()
  if (e.dataTransfer?.types.includes('Files')) {
    isDragging.value = true
  }
}

// 处理拖拽离开
const handleDragLeave = (e: DragEvent) => {
  e.preventDefault()
  e.stopPropagation()
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

// 中键点击标签页关闭
const handleTabMouseDown = (e: MouseEvent, tabId: string) => {
  if (e.button === 1) {
    e.preventDefault()
    fileStore.closeTab(tabId)
  }
}

// 主题变量映射到 CSS
const tabStyles = computed(() => ({
  '--tab-bg': themeVars.value.cardColor,
  '--tab-bg-hover': themeVars.value.hoverColor,
  '--tab-border': themeVars.value.borderColor,
  '--tab-text': themeVars.value.textColor2,
  '--tab-text-active': themeVars.value.textColor1,
  '--tab-text-muted': themeVars.value.textColor3,
  '--tab-primary': themeVars.value.primaryColor
}))
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
      <!-- 标签页栏 -->
      <div class="file-tabs" :style="tabStyles">
        <draggable
          v-model="localTabs"
          item-key="id"
          class="file-tabs-list"
          :animation="200"
          ghost-class="file-tab-ghost"
          drag-class="file-tab-drag"
        >
          <template #item="{ element: tab }">
            <div
              class="file-tab"
              :class="{ active: tab.id === fileStore.activeTabId }"
              @click="fileStore.switchTab(tab.id)"
              @mousedown="handleTabMouseDown($event, tab.id)"
            >
              <i-mdi-folder-outline class="file-tab-icon" />
              <span class="file-tab-label" :title="tab.path">{{ tab.label }}</span>
              <span
                v-if="fileStore.tabs.length > 1"
                class="file-tab-close"
                @click.stop="fileStore.closeTab(tab.id)"
              >
                <i-mdi-close :size="14" />
              </span>
            </div>
          </template>
          <template #footer>
            <div class="file-tab-add" @click="fileStore.createTab()">
              <i-mdi-plus :size="16" />
            </div>
          </template>
        </draggable>
      </div>

      <!-- 每个标签页内容（v-if 只渲染活跃的） -->
      <template v-for="tab in fileStore.tabs" :key="tab.id">
        <template v-if="tab.id === fileStore.activeTabId">
          <path-input :tab-id="tab.id" />
          <tool-bar
            :tab-id="tab.id"
            v-model:selected="selected"
            v-model:compress="compress"
            v-model:permission="permission"
            v-model:upload="upload"
          />
          <list-view
            :tab-id="tab.id"
            v-model:selected="selected"
            v-model:compress="compress"
            v-model:permission="permission"
            v-model:permission-file-info-list="permissionFileInfoList"
          />
        </template>
      </template>

      <compress-modal
        v-model:show="compress"
        v-model:path="fileStore.activeTab!.path"
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
      v-model:path="fileStore.activeTab!.path"
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

.file-tabs {
  flex-shrink: 0;
  margin-bottom: -8px;
  border-bottom: 1px solid var(--tab-border);
}

.file-tabs-list {
  display: flex;
  align-items: stretch;
  overflow-x: auto;
  padding: 0;

  // 隐藏滚动条
  scrollbar-width: none;
  &::-webkit-scrollbar {
    display: none;
  }
}

.file-tab {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 7px 12px;
  font-size: 13px;
  cursor: pointer;
  white-space: nowrap;
  user-select: none;
  color: var(--tab-text);
  transition: all 0.15s ease;
  position: relative;
  min-width: 0;

  &:hover {
    background: var(--tab-bg-hover);
    color: var(--tab-text-active);

    .file-tab-close {
      opacity: 1;
    }
  }

  &.active {
    background: var(--tab-bg);
    color: var(--tab-text-active);
    font-weight: 500;

    // 底部 primary 色指示条
    &::after {
      content: '';
      position: absolute;
      bottom: 0;
      left: 0;
      right: 0;
      height: 2px;
      background: var(--tab-primary);
    }

    .file-tab-close {
      opacity: 0.6;
    }
  }
}

.file-tab-icon {
  font-size: 15px;
  flex-shrink: 0;
  opacity: 0.6;
}

.file-tab-label {
  max-width: 140px;
  overflow: hidden;
  text-overflow: ellipsis;
}

.file-tab-close {
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
  width: 18px;
  height: 18px;
  border-radius: 4px;
  opacity: 0;
  color: var(--tab-text-muted);
  transition: all 0.1s ease;

  &:hover {
    background: rgba(0, 0, 0, 0.12);
    color: var(--tab-text-active);
    opacity: 1 !important;
  }
}

.file-tab-add {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 28px;
  height: 28px;
  margin: auto 0;
  border-radius: 6px;
  cursor: pointer;
  color: var(--tab-text-muted);
  flex-shrink: 0;
  transition: all 0.15s ease;

  &:hover {
    background: var(--tab-bg-hover);
    color: var(--tab-text-active);
  }
}

// 拖拽时的幽灵元素
.file-tab-ghost {
  opacity: 0.4;
}

// 拖拽中的元素
.file-tab-drag {
  background: var(--tab-bg) !important;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.15);
}
</style>

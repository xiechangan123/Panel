<script setup lang="ts">
import file from '@/api/panel/file'
import TheIcon from '@/components/custom/TheIcon.vue'
import { useEditorStore } from '@/store'
import { decodeBase64 } from '@/utils'
import { getExt, getIconByExt } from '@/utils/file'
import type { TreeOption } from 'naive-ui'
import { useThemeVars } from 'naive-ui'
import { useGettext } from 'vue3-gettext'

const { $gettext } = useGettext()
const editorStore = useEditorStore()
const themeVars = useThemeVars()

const props = defineProps<{
  rootPath: string
}>()

const emit = defineEmits<{
  (e: 'update:rootPath', path: string): void
}>()

// 文件树数据
const treeData = ref<TreeOption[]>([])
const expandedKeys = ref<string[]>([])
const selectedKeys = ref<string[]>([])
const loading = ref(false)
const searchKeyword = ref('')

// 新建文件/目录弹窗
const showCreateModal = ref(false)
const createType = ref<'file' | 'dir'>('file')
const createName = ref('')
const createParentPath = ref('')
const createLoading = ref(false)

// 加载目录内容
async function loadDirectory(path: string): Promise<TreeOption[]> {
  return new Promise((resolve, reject) => {
    useRequest(file.list(path, '', false, 'name', 1, 1000))
      .onSuccess(({ data }) => {
        const items = data.items || []
        // 排序：目录在前，文件在后，同类型按名称排序
        const sortedItems = [...items].sort((a: any, b: any) => {
          if (a.dir && !b.dir) return -1
          if (!a.dir && b.dir) return 1
          return a.name.localeCompare(b.name)
        })
        resolve(
          sortedItems.map((item: any) => ({
            key: item.full,
            label: item.name,
            isLeaf: !item.dir,
            prefix: () =>
              h(TheIcon, {
                icon: item.dir ? 'mdi:folder' : getIconByExt(getExt(item.name)),
                size: 18,
                color: item.dir ? '#f59e0b' : '#6b7280'
              }),
            isDir: item.dir,
            children: item.dir ? undefined : undefined
          }))
        )
      })
      .onError(() => {
        reject(new Error('Failed to load directory'))
      })
  })
}

// 初始化加载根目录
async function initTree() {
  loading.value = true
  try {
    treeData.value = await loadDirectory(props.rootPath)
    expandedKeys.value = []
  } catch {
    treeData.value = []
  } finally {
    loading.value = false
  }
}

// 懒加载子目录
async function handleLoad(node: TreeOption): Promise<void> {
  if (node.isLeaf) return
  try {
    const children = await loadDirectory(node.key as string)
    node.children = children
  } catch {
    node.children = []
  }
}

// 展开节点
function handleExpandedKeysUpdate(keys: string[]) {
  expandedKeys.value = keys
}

// 选择节点（打开文件）
async function handleSelect(keys: string[], option: TreeOption[]) {
  if (keys.length === 0) return
  selectedKeys.value = keys

  const node = option[0]
  if (node && node.isLeaf) {
    // 打开文件
    const path = node.key as string
    const existingTab = editorStore.tabs.find((t) => t.path === path)
    if (existingTab) {
      editorStore.switchTab(path)
    } else {
      // 加载文件内容
      editorStore.openFile(path, '', 'utf-8')
      editorStore.setLoading(path, true)
      useRequest(file.content(encodeURIComponent(path)))
        .onSuccess(({ data }) => {
          const content = decodeBase64(data.content)
          editorStore.reloadFile(path, content)
        })
        .onError(() => {
          editorStore.closeTab(path)
          window.$message.error($gettext('Failed to load file'))
        })
        .onComplete(() => {
          editorStore.setLoading(path, false)
        })
    }
  }
}

// 上一级目录
function handleGoUp() {
  const parts = props.rootPath.split('/').filter(Boolean)
  if (parts.length > 0) {
    parts.pop()
    const newPath = '/' + parts.join('/')
    emit('update:rootPath', newPath || '/')
  }
}

// 刷新
function handleRefresh() {
  initTree()
}

// 显示新建弹窗
function showCreate(type: 'file' | 'dir') {
  createType.value = type
  createName.value = ''
  // 使用选中的目录或根目录作为父目录
  if (selectedKeys.value.length > 0) {
    const selectedNode = findNode(treeData.value, selectedKeys.value[0])
    if (selectedNode && !selectedNode.isLeaf) {
      createParentPath.value = selectedKeys.value[0]
    } else {
      // 如果选中的是文件，使用其父目录
      const parts = selectedKeys.value[0].split('/')
      parts.pop()
      createParentPath.value = parts.join('/') || props.rootPath
    }
  } else {
    createParentPath.value = props.rootPath
  }
  showCreateModal.value = true
}

// 查找节点
function findNode(nodes: TreeOption[], key: string): TreeOption | null {
  for (const node of nodes) {
    if (node.key === key) return node
    if (node.children) {
      const found = findNode(node.children, key)
      if (found) return found
    }
  }
  return null
}

// 确认新建
async function handleCreate() {
  if (!createName.value.trim()) {
    window.$message.warning($gettext('Please enter a name'))
    return
  }

  const fullPath = `${createParentPath.value}/${createName.value}`.replace(/\/+/g, '/')

  createLoading.value = true
  useRequest(file.create(fullPath, createType.value === 'dir'))
    .onSuccess(async () => {
      window.$message.success($gettext('Created successfully'))
      showCreateModal.value = false

      // 刷新父目录
      if (expandedKeys.value.includes(createParentPath.value)) {
        const parentNode = findNode(treeData.value, createParentPath.value)
        if (parentNode) {
          parentNode.children = await loadDirectory(createParentPath.value)
        }
      } else if (createParentPath.value === props.rootPath) {
        // 如果是在根目录创建，刷新整个树
        await initTree()
      } else {
        // 展开父目录
        expandedKeys.value = [...expandedKeys.value, createParentPath.value]
      }

      // 如果是文件，自动打开
      if (createType.value === 'file') {
        editorStore.openFile(fullPath, '', 'utf-8')
      }
    })
    .onError(() => {
      window.$message.error($gettext('Failed to create'))
    })
    .onComplete(() => {
      createLoading.value = false
    })
}

// 搜索过滤
function filterTree(pattern: string, option: TreeOption): boolean {
  if (!pattern) return true
  return (option.label as string).toLowerCase().includes(pattern.toLowerCase())
}

// 路径编辑
const isEditingPath = ref(false)
const editingPath = ref('')
const pathInputRef = ref<HTMLInputElement | null>(null)

function startEditPath() {
  editingPath.value = props.rootPath
  isEditingPath.value = true
  nextTick(() => {
    pathInputRef.value?.focus()
    pathInputRef.value?.select()
  })
}

function confirmEditPath() {
  const newPath = editingPath.value.trim()
  if (newPath && newPath !== props.rootPath) {
    // 确保路径以 / 开头
    const normalizedPath = newPath.startsWith('/') ? newPath : '/' + newPath
    emit('update:rootPath', normalizedPath)
  }
  isEditingPath.value = false
}

function cancelEditPath() {
  isEditingPath.value = false
  editingPath.value = ''
}

// 监听根目录变化
watch(
  () => props.rootPath,
  () => {
    initTree()
  }
)

onMounted(() => {
  initTree()
})

defineExpose({
  refresh: handleRefresh
})
</script>

<template>
  <div class="file-tree">
    <!-- 工具栏 -->
    <div class="file-tree-toolbar">
      <n-button-group size="small">
        <n-button quaternary @click="handleGoUp" :title="$gettext('Go Up')">
          <template #icon>
            <i-mdi-arrow-up />
          </template>
        </n-button>
        <n-button quaternary @click="handleRefresh" :title="$gettext('Refresh')">
          <template #icon>
            <i-mdi-refresh />
          </template>
        </n-button>
        <n-popselect
          :options="[
            { label: $gettext('New File'), value: 'file' },
            { label: $gettext('New Folder'), value: 'dir' }
          ]"
          @update:value="showCreate"
        >
          <n-button quaternary :title="$gettext('New')">
            <template #icon>
              <i-mdi-plus />
            </template>
          </n-button>
        </n-popselect>
      </n-button-group>
      <n-input
        v-model:value="searchKeyword"
        size="small"
        :placeholder="$gettext('Search')"
        clearable
        class="search-input"
      >
        <template #prefix>
          <i-mdi-magnify />
        </template>
      </n-input>
    </div>

    <!-- 当前目录 -->
    <div class="current-path" @click="startEditPath" v-if="!isEditingPath">
      <the-icon icon="mdi:folder-open" :size="14" class="path-icon" />
      <n-ellipsis :tooltip="{ width: 300 }">
        {{ rootPath }}
      </n-ellipsis>
      <the-icon icon="mdi:pencil" :size="12" class="edit-icon" />
    </div>
    <div class="current-path editing" v-else>
      <n-input
        ref="pathInputRef"
        v-model:value="editingPath"
        size="tiny"
        :placeholder="$gettext('Enter path')"
        @keyup.enter="confirmEditPath"
        @keyup.escape="cancelEditPath"
        @blur="confirmEditPath"
      >
        <template #prefix>
          <the-icon icon="mdi:folder-open" :size="14" />
        </template>
      </n-input>
    </div>

    <!-- 文件树 -->
    <div class="tree-container">
      <n-spin :show="loading" class="tree-spin">
        <n-tree
          block-line
          :data="treeData"
          :expanded-keys="expandedKeys"
          :selected-keys="selectedKeys"
          :pattern="searchKeyword"
          :filter="filterTree"
          :on-load="handleLoad"
          @update:expanded-keys="handleExpandedKeysUpdate"
          @update:selected-keys="handleSelect"
          selectable
          expand-on-click
          virtual-scroll
          class="file-tree-content"
          style="height: 100%"
        />
      </n-spin>
    </div>

    <!-- 新建弹窗 -->
    <n-modal
      v-model:show="showCreateModal"
      preset="dialog"
      :title="createType === 'file' ? $gettext('New File') : $gettext('New Folder')"
      :positive-text="$gettext('Create')"
      :negative-text="$gettext('Cancel')"
      :loading="createLoading"
      @positive-click="handleCreate"
    >
      <n-form>
        <n-form-item :label="$gettext('Parent Directory')">
          <n-input :value="createParentPath" disabled />
        </n-form-item>
        <n-form-item :label="$gettext('Name')">
          <n-input
            v-model:value="createName"
            :placeholder="
              createType === 'file' ? $gettext('Enter file name') : $gettext('Enter folder name')
            "
            @keyup.enter="handleCreate"
          />
        </n-form-item>
      </n-form>
    </n-modal>
  </div>
</template>

<style scoped lang="scss">
.file-tree {
  display: flex;
  flex-direction: column;
  height: 100%;
  overflow: hidden;
}

.file-tree-toolbar {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px;
  border-bottom: 1px solid v-bind('themeVars.borderColor');
  flex-shrink: 0;
}

.search-input {
  flex: 1;
  min-width: 0;
}

.current-path {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 6px 8px;
  min-height: 32px;
  box-sizing: border-box;
  font-size: 12px;
  color: v-bind('themeVars.textColor3');
  border-bottom: 1px solid v-bind('themeVars.borderColor');
  flex-shrink: 0;
  cursor: pointer;
  transition: background-color 0.2s;

  &:hover:not(.editing) {
    background-color: v-bind('themeVars.buttonColor2Hover');

    .edit-icon {
      opacity: 1;
    }
  }

  &.editing {
    cursor: default;
    padding: 4px 8px;
  }

  .path-icon {
    flex-shrink: 0;
    color: #f59e0b;
  }

  .edit-icon {
    flex-shrink: 0;
    margin-left: auto;
    opacity: 0;
    transition: opacity 0.2s;
  }

  :deep(.n-input) {
    flex: 1;
  }
}

.tree-container {
  flex: 1;
  overflow: hidden;
  min-height: 0;
}

.tree-spin {
  height: 100%;

  :deep(.n-spin-content) {
    height: 100%;
  }
}

.file-tree-content {
  height: 100%;
  overflow: auto;
}
</style>

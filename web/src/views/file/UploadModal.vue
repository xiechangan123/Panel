<script setup lang="ts">
import { useEventListener } from '@vueuse/core'
import {
  type DataTableColumns,
  NButton,
  NFlex,
  NPopselect,
  NProgress,
  NTag,
  NText,
  NTooltip,
  useThemeVars,
} from 'naive-ui'
import type { VNode } from 'vue'
import { useGettext } from 'vue3-gettext'

import TheIcon from '@/components/custom/TheIcon.vue'
import {
  type ConflictAction,
  type UploadItem,
  type UploadPriority,
  type UploadStatus,
  useUploadStore,
} from '@/stores'
import {
  filesFromInput,
  formatBytes,
  getFilename,
  type PickedFile,
  readDroppedFiles,
} from '@/utils/file'

const { $gettext } = useGettext()
const themeVars = useThemeVars()
const uploadStore = useUploadStore()

const show = defineModel<boolean>('show', { type: Boolean, required: true })
// 新添加文件的目标目录
const props = defineProps<{ path: string }>()

const fileInput = ref<HTMLInputElement | null>(null)
const dirInput = ref<HTMLInputElement | null>(null)
const checked = ref<string[]>([])

const stats = computed(() => uploadStore.stats)
const canStart = computed(
  () => stats.value.pending + stats.value.paused + stats.value.error + stats.value.skipped > 0,
)
const canPause = computed(() => stats.value.waiting + stats.value.uploading > 0)
const hasFinished = computed(() => stats.value.done + stats.value.skipped > 0)

const summary = computed(() => {
  const s = stats.value
  const parts = [$gettext('%{n} in total', { n: s.total })]
  if (s.uploading) parts.push($gettext('%{n} uploading', { n: s.uploading }))
  if (s.waiting) parts.push($gettext('%{n} queued', { n: s.waiting }))
  if (s.paused) parts.push($gettext('%{n} paused', { n: s.paused }))
  if (s.done) parts.push($gettext('%{n} completed', { n: s.done }))
  if (s.error) parts.push($gettext('%{n} failed', { n: s.error }))
  if (s.speed) parts.push(`${formatBytes(s.speed)}/s`)
  return parts.join(' · ')
})

const policyOptions = computed(() => [
  { label: $gettext('Ask'), value: 'ask' },
  { label: $gettext('Skip'), value: 'skip' },
  { label: $gettext('Rename'), value: 'rename' },
  { label: $gettext('Overwrite'), value: 'overwrite' },
])

const priorityOptions = computed(() => [
  { label: $gettext('High'), value: 'high' },
  { label: $gettext('Normal'), value: 'normal' },
  { label: $gettext('Low'), value: 'low' },
])
const priorityType: Record<UploadPriority, 'warning' | 'default' | 'info'> = {
  high: 'warning',
  normal: 'default',
  low: 'info',
}

type TagType = 'default' | 'info' | 'primary' | 'warning' | 'success' | 'error'
const statusMeta = computed<Record<UploadStatus, { label: string; type: TagType }>>(() => ({
  pending: { label: $gettext('Pending'), type: 'default' },
  waiting: { label: $gettext('Queued'), type: 'info' },
  uploading: { label: $gettext('Uploading'), type: 'primary' },
  paused: { label: $gettext('Paused'), type: 'warning' },
  done: { label: $gettext('Completed'), type: 'success' },
  error: { label: $gettext('Failed'), type: 'error' },
  skipped: { label: $gettext('Skipped'), type: 'default' },
}))

// ==================== 添加文件 ====================

const addFiles = (files: PickedFile[]) => {
  if (files.length > 0) uploadStore.add(files, props.path)
}

const onPick = (e: Event) => {
  const input = e.target as HTMLInputElement
  addFiles(filesFromInput(input.files))
  input.value = ''
}

// 整个弹窗内容区都可拖入，用计数器抵消子元素间的 enter/leave
const dragDepth = ref(0)
const onDragEnter = (e: DragEvent) => {
  if (e.dataTransfer?.types.includes('Files')) dragDepth.value++
}
const onDragLeave = () => {
  dragDepth.value = Math.max(0, dragDepth.value - 1)
}
const onDrop = async (e: DragEvent) => {
  dragDepth.value = 0
  addFiles(await readDroppedFiles(e.dataTransfer))
}
// 拖到弹窗外松手时 leave 事件可能不成对，兜底复位
useEventListener(document, 'drop', () => (dragDepth.value = 0))

// ==================== 队列表格 ====================

const iconButton = (icon: string, label: string, onClick: () => void) =>
  h(NTooltip, null, {
    trigger: () =>
      h(
        NButton,
        { quaternary: true, circle: true, size: 'small', onClick },
        { icon: () => h(TheIcon, { icon, size: 18 }) },
      ),
    default: () => label,
  })

const columns = computed<DataTableColumns<UploadItem>>(() => [
  { type: 'selection' },
  {
    title: $gettext('File'),
    key: 'name',
    minWidth: 220,
    render: (row) => {
      const uploadName = getFilename(row.target)
      const renamed = uploadName !== getFilename(row.name)
      return h(
        NTooltip,
        { placement: 'top-start', delay: 500 },
        {
          trigger: () =>
            h('div', { class: 'truncate' }, [
              row.name,
              renamed ? h(NText, { depth: 3, class: 'ml-2' }, () => `→ ${uploadName}`) : null,
            ]),
          default: () => row.target,
        },
      )
    },
  },
  {
    title: $gettext('Size'),
    key: 'size',
    width: 100,
    render: (row) => formatBytes(row.size),
  },
  {
    title: $gettext('Progress'),
    key: 'loaded',
    width: 200,
    render: (row) => {
      const percent = row.size
        ? Math.min(100, Math.floor((row.loaded / row.size) * 100))
        : row.status === 'done'
          ? 100
          : 0
      return h(NFlex, { align: 'center', size: 8, wrap: false }, () => [
        h(NProgress, {
          type: 'line',
          percentage: percent,
          showIndicator: false,
          height: 6,
          processing: row.status === 'uploading',
          status: row.status === 'error' ? 'error' : row.status === 'done' ? 'success' : 'default',
          style: 'flex: 1',
        }),
        h(NText, { depth: 3, class: 'progress-text' }, () => `${percent}%`),
      ])
    },
  },
  {
    title: $gettext('Status'),
    key: 'status',
    width: 160,
    render: (row) => {
      const meta = statusMeta.value[row.status]
      const tag = h(NTag, { type: meta.type, size: 'small', bordered: false }, () => meta.label)
      if (row.status === 'error') {
        return h(NTooltip, null, { trigger: () => tag, default: () => row.error })
      }
      if (row.status === 'uploading') {
        return h(NFlex, { align: 'center', size: 6, wrap: false }, () => [
          tag,
          h(NText, { depth: 3, class: 'speed-text' }, () =>
            row.speed ? `${formatBytes(row.speed)}/s` : $gettext('Preparing...'),
          ),
        ])
      }
      return tag
    },
  },
  {
    title: $gettext('Priority'),
    key: 'priority',
    width: 90,
    render: (row) =>
      h(
        NPopselect,
        {
          value: row.priority,
          options: priorityOptions.value,
          onUpdateValue: (v: UploadPriority) => uploadStore.setPriority([row.id], v),
        },
        () =>
          h(
            NTag,
            {
              size: 'small',
              bordered: false,
              type: priorityType[row.priority],
              class: 'cursor-pointer',
            },
            () => priorityOptions.value.find((o) => o.value === row.priority)?.label,
          ),
      ),
  },
  {
    title: $gettext('Actions'),
    key: 'actions',
    width: 90,
    render: (row) => {
      const buttons: VNode[] = []
      if (['pending', 'paused', 'error', 'skipped'].includes(row.status)) {
        buttons.push(
          iconButton(
            'mdi:play',
            row.status === 'error' ? $gettext('Retry') : $gettext('Start'),
            () => uploadStore.start([row.id]),
          ),
        )
      }
      if (row.status === 'waiting' || row.status === 'uploading') {
        buttons.push(iconButton('mdi:pause', $gettext('Pause'), () => uploadStore.pause([row.id])))
      }
      const finished = row.status === 'done' || row.status === 'skipped'
      buttons.push(
        iconButton('mdi:close', finished ? $gettext('Remove') : $gettext('Cancel'), () =>
          uploadStore.remove([row.id]),
        ),
      )
      return h(NFlex, { size: 0, wrap: false }, () => buttons)
    },
  },
])

// 行被移除后同步清理选中
watch(
  () => uploadStore.items.length,
  () => {
    const ids = new Set(uploadStore.items.map((i) => i.id))
    checked.value = checked.value.filter((id) => ids.has(id))
  },
)

// 关闭弹窗不中断上传，提示一下
watch(show, (val) => {
  if (!val && uploadStore.activeCount > 0) {
    window.$message.info($gettext('Uploads will continue in the background'))
  }
})

// ==================== 同名冲突询问 ====================

const conflictShow = computed(() => uploadStore.conflictItems.length > 0)

const setAllConflictAction = (action: ConflictAction) => {
  uploadStore.conflictItems.forEach((i) => (i.action = action))
}

const onConflictConfirm = () => {
  const items = uploadStore.conflictItems
  const renameItems = items.filter((i) => i.action === 'rename')

  // 文件名非空 + 无路径分隔符 + 不能用 . / ..
  for (const item of renameItems) {
    const name = item.newName.trim()
    if (!name) {
      window.$message.error($gettext('New name for "%{name}" cannot be empty', { name: item.name }))
      return
    }
    if (name.includes('/') || name.includes('\\') || name === '.' || name === '..') {
      window.$message.error($gettext('Invalid new name "%{name}"', { name }))
      return
    }
    item.newName = name
  }

  // 本批次内重命名后不能重名
  const seen = new Set<string>()
  for (const item of renameItems) {
    if (seen.has(item.newName)) {
      window.$message.error($gettext('Duplicate new name "%{name}"', { name: item.newName }))
      return
    }
    seen.add(item.newName)
  }

  uploadStore.resolveConflicts([...items])
}

const onConflictCancel = () => {
  uploadStore.resolveConflicts(null)
}
</script>

<template>
  <n-modal
    v-model:show="show"
    preset="card"
    :title="$gettext('Upload')"
    style="width: 80vw; max-width: 1200px"
    size="huge"
    :bordered="false"
    :segmented="false"
  >
    <input ref="fileInput" type="file" multiple class="hidden" @change="onPick" />
    <input ref="dirInput" type="file" webkitdirectory class="hidden" @change="onPick" />

    <div
      class="upload-body"
      :style="{
        '--border-color': themeVars.borderColor,
        '--primary-color': themeVars.primaryColor,
        '--hover-color': themeVars.hoverColor,
        '--card-color': themeVars.modalColor,
      }"
      @dragenter.prevent="onDragEnter"
      @dragover.prevent
      @dragleave="onDragLeave"
      @drop.prevent="onDrop"
    >
      <n-flex vertical :size="12">
        <!-- 添加文件 + 同名策略 -->
        <n-flex align="center" justify="space-between">
          <n-flex align="center" :size="8">
            <n-button @click="fileInput?.click()">
              <template #icon>
                <the-icon icon="mdi:file-plus-outline" :size="18" />
              </template>
              {{ $gettext('Add Files') }}
            </n-button>
            <n-button @click="dirInput?.click()">
              <template #icon>
                <the-icon icon="mdi:folder-plus-outline" :size="18" />
              </template>
              {{ $gettext('Add Folder') }}
            </n-button>
            <NText depth="3" class="target-path truncate">
              {{ $gettext('Upload to %{path}', { path }) }}
            </NText>
          </n-flex>
          <n-flex align="center" :size="8">
            <NText depth="3">{{ $gettext('If file exists') }}</NText>
            <n-select
              v-model:value="uploadStore.conflictPolicy"
              :options="policyOptions"
              size="small"
              class="w-28"
            />
          </n-flex>
        </n-flex>

        <!-- 空队列：拖入区 -->
        <div
          v-if="uploadStore.items.length === 0"
          class="drop-zone"
          :class="{ active: dragDepth > 0 }"
          @click="fileInput?.click()"
        >
          <the-icon :size="48" icon="mdi:cloud-upload-outline" />
          <NText>{{ $gettext('Drag files or folders here, or click to select files') }}</NText>
          <NText depth="3" class="text-xs">
            {{
              $gettext(
                'Files are queued first; large files are uploaded in chunks and can be paused and resumed',
              )
            }}
          </NText>
        </div>

        <!-- 上传队列 -->
        <template v-else>
          <n-data-table
            size="small"
            virtual-scroll
            max-height="55vh"
            :columns="columns"
            :data="uploadStore.items"
            :row-key="(row: UploadItem) => row.id"
            :checked-row-keys="checked"
            @update:checked-row-keys="(keys: (string | number)[]) => (checked = keys as string[])"
          />

          <n-flex align="center" justify="space-between">
            <!-- 选中项批量操作 / 汇总 -->
            <n-flex v-if="checked.length > 0" align="center" :size="8">
              <NText>{{ $gettext('%{n} selected', { n: checked.length }) }}</NText>
              <n-button size="small" @click="uploadStore.start(checked)">
                {{ $gettext('Start') }}
              </n-button>
              <n-button size="small" @click="uploadStore.pause(checked)">
                {{ $gettext('Pause') }}
              </n-button>
              <n-popselect
                :options="priorityOptions"
                @update:value="(v: UploadPriority) => uploadStore.setPriority(checked, v)"
              >
                <n-button size="small">{{ $gettext('Priority') }}</n-button>
              </n-popselect>
              <n-button size="small" secondary type="error" @click="uploadStore.remove(checked)">
                {{ $gettext('Cancel') }}
              </n-button>
              <n-button size="small" quaternary @click="checked = []">
                {{ $gettext('Deselect') }}
              </n-button>
            </n-flex>
            <NText v-else depth="3">{{ summary }}</NText>

            <n-flex align="center" :size="8">
              <n-button :disabled="!hasFinished" @click="uploadStore.clearFinished()">
                {{ $gettext('Clear Finished') }}
              </n-button>
              <n-button :disabled="!canPause" @click="uploadStore.pause()">
                <template #icon>
                  <the-icon icon="mdi:pause" :size="18" />
                </template>
                {{ $gettext('Pause All') }}
              </n-button>
              <n-button type="primary" :disabled="!canStart" @click="uploadStore.start()">
                <template #icon>
                  <the-icon icon="mdi:play" :size="18" />
                </template>
                {{ $gettext('Start All') }}
              </n-button>
            </n-flex>
          </n-flex>
        </template>
      </n-flex>

      <!-- 拖入时的遮罩提示 -->
      <div v-if="dragDepth > 0 && uploadStore.items.length > 0" class="drop-overlay">
        <the-icon :size="40" icon="mdi:cloud-upload-outline" />
        <NText>{{ $gettext('Drop to add to the queue') }}</NText>
      </div>
    </div>
  </n-modal>

  <!-- 文件冲突处理弹窗：批量列出冲突文件，每个单独选 跳过/重命名/覆盖 -->
  <n-modal
    :show="conflictShow"
    preset="card"
    :title="$gettext('Files already exist')"
    style="width: 60vw"
    size="huge"
    :bordered="false"
    :segmented="false"
    :mask-closable="false"
    :closable="false"
  >
    <n-flex vertical :size="16">
      <NText depth="3">
        {{
          $gettext(
            'The following files already exist in the target directory. Choose an action for each.',
          )
        }}
      </NText>
      <n-flex align="center" :size="8">
        <NText>{{ $gettext('Apply to all:') }}</NText>
        <n-button size="small" @click="setAllConflictAction('skip')">
          {{ $gettext('Skip') }}
        </n-button>
        <n-button size="small" @click="setAllConflictAction('rename')">
          {{ $gettext('Rename') }}
        </n-button>
        <n-button size="small" @click="setAllConflictAction('overwrite')">
          {{ $gettext('Overwrite') }}
        </n-button>
      </n-flex>
      <n-scrollbar style="max-height: 50vh">
        <n-flex vertical :size="8">
          <n-flex
            v-for="item in uploadStore.conflictItems"
            :key="item.id"
            align="center"
            justify="space-between"
            class="conflict-row"
          >
            <n-flex vertical :size="4" style="min-width: 0; flex: 1">
              <NText style="word-break: break-all">{{ item.name }}</NText>
              <n-flex v-if="item.action === 'rename'" align="center" :size="6" style="min-width: 0">
                <NText depth="3" style="font-size: 12px">→</NText>
                <n-input
                  v-model:value="item.newName"
                  size="small"
                  :placeholder="$gettext('New file name')"
                  style="flex: 1; min-width: 160px"
                />
              </n-flex>
            </n-flex>
            <n-radio-group v-model:value="item.action" size="small">
              <n-radio-button value="skip">{{ $gettext('Skip') }}</n-radio-button>
              <n-radio-button value="rename">{{ $gettext('Rename') }}</n-radio-button>
              <n-radio-button value="overwrite">{{ $gettext('Overwrite') }}</n-radio-button>
            </n-radio-group>
          </n-flex>
        </n-flex>
      </n-scrollbar>
      <n-flex justify="end" :size="8">
        <n-button @click="onConflictCancel">{{ $gettext('Cancel') }}</n-button>
        <n-button type="primary" @click="onConflictConfirm">{{ $gettext('Confirm') }}</n-button>
      </n-flex>
    </n-flex>
  </n-modal>
</template>

<style scoped lang="scss">
.upload-body {
  position: relative;
}

.target-path {
  max-width: 360px;
}

.drop-zone {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 6px;
  height: 240px;
  border: 1px dashed var(--border-color);
  border-radius: 6px;
  cursor: pointer;
  transition:
    border-color 0.2s,
    background-color 0.2s;

  &:hover,
  &.active {
    border-color: var(--primary-color);
    background-color: var(--hover-color);
  }
}

.drop-overlay {
  position: absolute;
  inset: 0;
  z-index: 10;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 8px;
  border: 2px dashed var(--primary-color);
  border-radius: 6px;
  background-color: var(--card-color);
  opacity: 0.95;
  pointer-events: none;
}

:deep(.progress-text) {
  width: 36px;
  text-align: right;
  font-size: 12px;
}

:deep(.speed-text) {
  font-size: 12px;
  white-space: nowrap;
}

.conflict-row {
  padding: 8px 12px;
  background: var(--n-color-embedded);
  border-radius: 4px;
}
</style>

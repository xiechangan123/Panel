<script setup lang="ts">
import {
  NButton,
  NDataTable,
  NEllipsis,
  NFlex,
  NInput,
  NPopconfirm,
  NPopselect,
  NTag
} from 'naive-ui'

import type { DataTableColumns, DropdownOption } from 'naive-ui'
import type { RowData } from 'naive-ui/es/data-table/src/interface'

import file from '@/api/panel/file'
import TheIcon from '@/components/custom/TheIcon.vue'
import {
  checkName,
  checkPath,
  getExt,
  getFilename,
  getIconByExt,
  isCompress,
  isImage
} from '@/utils/file'
import EditModal from '@/views/file/EditModal.vue'
import PreviewModal from '@/views/file/PreviewModal.vue'
import type { Marked } from '@/views/file/types'

const sort = ref<string>('')
const path = defineModel<string>('path', { type: String, required: true })
const selected = defineModel<any[]>('selected', { type: Array, default: () => [] })
const marked = defineModel<Marked[]>('marked', { type: Array, default: () => [] })
const markedType = defineModel<string>('markedType', { type: String, required: true })
const compress = defineModel<boolean>('compress', { type: Boolean, required: true })
const permission = defineModel<boolean>('permission', { type: Boolean, required: true })
const editorModal = ref(false)
const previewModal = ref(false)
const currentFile = ref('')

const showDropdown = ref(false)
const selectedRow = ref<any>()
const dropdownX = ref(0)
const dropdownY = ref(0)

const renameModal = ref(false)
const renameModel = ref({
  source: '',
  target: ''
})
const unCompressModal = ref(false)
const unCompressModel = ref({
  path: '',
  file: ''
})

const options = computed<DropdownOption[]>(() => {
  if (selectedRow.value == null) return []
  const options = [
    {
      label: selectedRow.value.dir ? '打开' : isImage(selectedRow.value.name) ? '预览' : '编辑',
      key: selectedRow.value.dir ? 'open' : isImage(selectedRow.value.name) ? 'preview' : 'edit'
    },
    { label: '复制', key: 'copy' },
    { label: '移动', key: 'move' },
    { label: '权限', key: 'permission' },
    {
      label: selectedRow.value.dir ? '压缩' : '下载',
      key: selectedRow.value.dir ? 'compress' : 'download'
    },
    {
      label: '解压',
      key: 'uncompress',
      show: isCompress(selectedRow.value.full),
      disabled: !isCompress(selectedRow.value.full)
    },
    { label: '重命名', key: 'rename' },
    { label: () => h('span', { style: { color: 'red' } }, '删除'), key: 'delete' }
  ]
  if (marked.value.length) {
    options.unshift({
      label: '粘贴',
      key: 'paste'
    })
  }

  return options
})

const columns: DataTableColumns<RowData> = [
  {
    type: 'selection',
    fixed: 'left'
  },
  {
    title: '名称',
    key: 'name',
    minWidth: 180,
    defaultSortOrder: false,
    sorter: 'default',
    render(row) {
      let icon = 'bi:file-earmark'
      if (row.dir) {
        icon = 'bi:folder'
      } else {
        icon = getIconByExt(getExt(row.name))
      }

      return h(
        NFlex,
        {
          class: 'cursor-pointer hover:opacity-60',
          onClick: () => {
            if (row.dir) {
              path.value = row.full
            } else {
              currentFile.value = row.full
              editorModal.value = true
            }
          }
        },
        () => [
          h(TheIcon, { icon, size: 24, color: `var(--primary-color)` }),
          h(NEllipsis, null, {
            default: () => {
              if (row.symlink) {
                return row.name + ' -> ' + row.link
              } else {
                return row.name
              }
            }
          })
        ]
      )
    }
  },
  {
    title: '权限',
    key: 'mode',
    minWidth: 80,
    render(row: any): any {
      return h(
        NTag,
        { type: 'success', size: 'small', bordered: false },
        { default: () => row.mode }
      )
    }
  },
  {
    title: '所有者 / 组',
    key: 'owner/group',
    minWidth: 120,
    render(row: any): any {
      return h('div', null, [
        h(NTag, { type: 'primary', size: 'small', bordered: false }, { default: () => row.owner }),
        ' / ',
        h(NTag, { type: 'primary', size: 'small', bordered: false }, { default: () => row.group })
      ])
    }
  },
  {
    title: '大小',
    key: 'size',
    minWidth: 80,
    render(row: any): any {
      return h(NTag, { type: 'info', size: 'small', bordered: false }, { default: () => row.size })
    }
  },
  {
    title: '修改时间',
    key: 'modify',
    minWidth: 200,
    render(row: any): any {
      return h(
        NTag,
        { type: 'warning', size: 'small', bordered: false },
        { default: () => row.modify }
      )
    }
  },
  {
    title: '操作',
    key: 'action',
    width: 340,
    render(row) {
      return h(
        NFlex,
        {},
        {
          default: () => [
            h(
              NButton,
              {
                size: 'small',
                type: row.dir ? 'success' : 'primary',
                tertiary: true,
                onClick: () => {
                  if (!row.dir && !row.symlink) {
                    currentFile.value = row.full
                    if (isImage(row.name)) {
                      previewModal.value = true
                    } else {
                      editorModal.value = true
                    }
                  } else {
                    path.value = row.full
                  }
                }
              },
              {
                default: () => {
                  if (!row.dir && !row.symlink) {
                    return isImage(row.name) ? '预览' : '编辑'
                  } else {
                    return '打开'
                  }
                }
              }
            ),
            h(
              NButton,
              {
                size: 'small',
                type: row.dir ? 'primary' : 'info',
                tertiary: true,
                onClick: () => {
                  if (row.dir) {
                    selected.value = [row.full]
                    compress.value = true
                  } else {
                    window.open('/api/file/download?path=' + encodeURIComponent(row.full))
                  }
                }
              },
              {
                default: () => {
                  if (row.dir) {
                    return '压缩'
                  } else {
                    return '下载'
                  }
                }
              }
            ),
            h(
              NButton,
              {
                type: 'warning',
                size: 'small',
                tertiary: true,
                onClick: () => {
                  renameModel.value.source = getFilename(row.name)
                  renameModel.value.target = getFilename(row.name)
                  renameModal.value = true
                }
              },
              { default: () => '重命名' }
            ),
            h(
              NPopconfirm,
              {
                onPositiveClick: () => {
                  useRequest(file.delete(row.full)).onComplete(() => {
                    window.$bus.emit('file:refresh')
                    window.$message.success('删除成功')
                  })
                },
                onNegativeClick: () => {}
              },
              {
                default: () => {
                  return `确定删除 ${row.name} 吗？`
                },
                trigger: () => {
                  return h(
                    NButton,
                    {
                      size: 'small',
                      type: 'error',
                      tertiary: true
                    },
                    { default: () => '删除' }
                  )
                }
              }
            ),
            h(
              NPopselect,
              {
                options: [
                  { label: '复制', value: 'copy' },
                  { label: '移动', value: 'move' },
                  { label: '权限', value: 'permission' },
                  { label: '压缩', value: 'compress' },
                  { label: '解压', value: 'uncompress', disabled: !isCompress(row.name) }
                ],
                onUpdateValue: (value) => {
                  switch (value) {
                    case 'copy':
                      markedType.value = 'copy'
                      marked.value = [
                        {
                          name: row.name,
                          source: row.full,
                          force: false
                        }
                      ]
                      window.$message.success('标记成功，请前往目标路径粘贴')
                      break
                    case 'move':
                      markedType.value = 'move'
                      marked.value = [
                        {
                          name: row.name,
                          source: row.full,
                          force: false
                        }
                      ]
                      window.$message.success('标记成功，请前往目标路径粘贴')
                      break
                    case 'permission':
                      selected.value = [row.full]
                      permission.value = true
                      break
                    case 'compress':
                      selected.value = [row.full]
                      compress.value = true
                      break
                    case 'uncompress':
                      unCompressModel.value.file = row.full
                      unCompressModel.value.path = path.value
                      unCompressModal.value = true
                      break
                  }
                }
              },
              {
                default: () => {
                  return h(
                    NButton,
                    {
                      tertiary: true,
                      size: 'small'
                    },
                    { default: () => '更多' }
                  )
                }
              }
            )
          ]
        }
      )
    }
  }
]

const rowProps = (row: any) => {
  return {
    onContextmenu: (e: MouseEvent) => {
      e.preventDefault()
      showDropdown.value = false
      nextTick().then(() => {
        showDropdown.value = true
        selectedRow.value = row
        dropdownX.value = e.clientX
        dropdownY.value = e.clientY
      })
    }
  }
}

const { loading, data, page, total, pageSize, pageCount, refresh } = usePagination(
  (page, pageSize) => file.list(path.value, page, pageSize, sort.value),
  {
    initialData: { total: 0, list: [] },
    initialPageSize: 100,
    total: (res: any) => res.total,
    data: (res: any) => res.items
  }
)

const handleRename = () => {
  const source = path.value + '/' + renameModel.value.source
  const target = path.value + '/' + renameModel.value.target
  if (!checkName(renameModel.value.source) || !checkName(renameModel.value.target)) {
    window.$message.error('名称不合法')
    return
  }

  useRequest(file.exist([target])).onSuccess(({ data }) => {
    if (data[0]) {
      window.$dialog.warning({
        title: '警告',
        content: `存在同名项，是否强制覆盖？`,
        positiveText: '覆盖',
        negativeText: '取消',
        onPositiveClick: () => {
          useRequest(file.move([{ source, target, force: true }]))
            .onSuccess(() => {
              window.$bus.emit('file:refresh')
              window.$message.success(
                `重命名 ${renameModel.value.source} 为 ${renameModel.value.target} 成功`
              )
            })
            .onComplete(() => {
              renameModal.value = false
            })
        }
      })
    } else {
      useRequest(file.move([{ source, target, force: false }]))
        .onSuccess(() => {
          window.$bus.emit('file:refresh')
          window.$message.success(
            `重命名 ${renameModel.value.source} 为 ${renameModel.value.target} 成功`
          )
        })
        .onComplete(() => {
          renameModal.value = false
        })
    }
  })
}

const handleUnCompress = () => {
  // 移除首位的 / 去检测
  if (
    !unCompressModel.value.path.startsWith('/') ||
    !checkPath(unCompressModel.value.path.slice(1))
  ) {
    window.$message.error('路径不合法')
    return
  }
  const message = window.$message.loading('正在解压中...', {
    duration: 0
  })
  useRequest(file.unCompress(unCompressModel.value.file, unCompressModel.value.path))
    .onSuccess(() => {
      unCompressModal.value = false
      window.$bus.emit('file:refresh')
      window.$message.success('解压成功')
    })
    .onComplete(() => {
      message?.destroy()
    })
}

const onChecked = (rowKeys: any) => {
  selected.value = rowKeys
}

const handlePaste = () => {
  if (!marked.value.length) {
    window.$message.error('请先标记需要复制或移动的文件/文件夹')
    return
  }

  // 查重
  let flag = false
  const paths = marked.value.map((item) => {
    return {
      name: item.name,
      source: item.source,
      target: path.value + '/' + item.name,
      force: false
    }
  })
  const sources = paths.map((item: any) => item.target)
  useRequest(file.exist(sources)).onSuccess(({ data }) => {
    for (let i = 0; i < data.length; i++) {
      if (data[i]) {
        flag = true
        paths[i].force = true
      }
    }
    if (flag) {
      window.$dialog.warning({
        title: '警告',
        content: `存在同名项
      ${paths
        .filter((item) => item.force)
        .map((item) => item.name)
        .join(', ')} 是否覆盖？`,
        positiveText: '覆盖',
        negativeText: '取消',
        onPositiveClick: () => {
          if (markedType.value == 'copy') {
            useRequest(file.copy(paths)).onSuccess(() => {
              window.$message.success('复制成功')
            })
          } else {
            useRequest(file.move(paths)).onSuccess(() => {
              window.$message.success('移动成功')
            })
          }
          marked.value = []
          window.$bus.emit('file:refresh')
        },
        onNegativeClick: () => {
          marked.value = []
          window.$message.info('已取消')
        }
      })
    } else {
      if (markedType.value == 'copy') {
        useRequest(file.copy(paths)).onSuccess(() => {
          window.$message.success('复制成功')
        })
      } else {
        useRequest(file.move(paths)).onSuccess(() => {
          window.$message.success('移动成功')
        })
      }
      marked.value = []
      window.$bus.emit('file:refresh')
    }
  })
}

const handleSelect = (key: string) => {
  switch (key) {
    case 'paste':
      handlePaste()
      break
    case 'open':
      path.value = selectedRow.value.full
      break
    case 'edit':
      currentFile.value = selectedRow.value.full
      editorModal.value = true
      break
    case 'preview':
      currentFile.value = selectedRow.value.full
      previewModal.value = true
      break
    case 'copy':
      markedType.value = 'copy'
      marked.value = [
        {
          name: selectedRow.value.name,
          source: selectedRow.value.full,
          force: false
        }
      ]
      window.$message.success('标记成功，请前往目标路径粘贴')
      break
    case 'move':
      markedType.value = 'move'
      marked.value = [
        {
          name: selectedRow.value.name,
          source: selectedRow.value.full,
          force: false
        }
      ]
      window.$message.success('标记成功，请前往目标路径粘贴')
      break
    case 'permission':
      selected.value = [selectedRow.value.full]
      permission.value = true
      break
    case 'compress':
      selected.value = [selectedRow.value.full]
      compress.value = true
      break
    case 'download':
      window.open('/api/file/download?path=' + encodeURIComponent(selectedRow.value.full))
      break
    case 'uncompress':
      unCompressModel.value.file = selectedRow.value.full
      unCompressModel.value.path = path.value
      unCompressModal.value = true
      break
    case 'rename':
      renameModel.value.source = getFilename(selectedRow.value.name)
      renameModel.value.target = getFilename(selectedRow.value.name)
      renameModal.value = true
      break
    case 'delete':
      useRequest(file.delete(selectedRow.value.full)).onSuccess(() => {
        window.$bus.emit('file:refresh')
        window.$message.success('删除成功')
      })
      break
  }
  onCloseDropdown()
}

const onCloseDropdown = () => {
  selectedRow.value = null
  showDropdown.value = false
}

const handleSorterChange = (sorter: {
  columnKey: string | number | null
  order: 'ascend' | 'descend' | false
}) => {
  if (!sorter || sorter.columnKey === 'name') {
    if (!loading.value) {
      switch (sorter.order) {
        case 'ascend':
          sort.value = 'asc'
          refresh()
          break
        case 'descend':
          sort.value = 'desc'
          refresh()
          break
        default:
          sort.value = ''
          refresh()
          break
      }
    }
  }
}

onMounted(() => {
  watch(
    path,
    () => {
      selected.value = []
      refresh()
      window.$bus.emit('push-history', path.value)
    },
    { immediate: true }
  )
  window.$bus.on('file:refresh', refresh)
})

onUnmounted(() => {
  window.$bus.off('file:refresh')
})
</script>

<template>
  <n-data-table
    remote
    striped
    virtual-scroll
    size="small"
    :scroll-x="1200"
    :columns="columns"
    :data="data"
    :row-props="rowProps"
    :loading="loading"
    :row-key="(row: any) => row.full"
    :checked-row-keys="selected"
    max-height="60vh"
    @update:sorter="handleSorterChange"
    @update:checked-row-keys="onChecked"
    v-model:page="page"
    v-model:pageSize="pageSize"
    :pagination="{
      page: page,
      pageCount: pageCount,
      pageSize: pageSize,
      itemCount: total,
      showQuickJumper: true,
      showSizePicker: true,
      pageSizes: [100, 200, 500, 1000, 1500, 2000, 5000]
    }"
  />
  <n-dropdown
    placement="bottom-start"
    trigger="manual"
    :x="dropdownX"
    :y="dropdownY"
    :options="options"
    :show="showDropdown"
    :on-clickoutside="onCloseDropdown"
    @select="handleSelect"
  />
  <edit-modal v-model:show="editorModal" v-model:file="currentFile" />
  <preview-modal v-model:show="previewModal" v-model:path="currentFile" />
  <n-modal
    v-model:show="renameModal"
    preset="card"
    :title="'重命名 - ' + renameModel.source"
    style="width: 60vw"
    size="huge"
    :bordered="false"
    :segmented="false"
  >
    <n-flex vertical>
      <n-form>
        <n-form-item label="新名称">
          <n-input v-model:value="renameModel.target" />
        </n-form-item>
      </n-form>
      <n-button type="primary" @click="handleRename">保存</n-button>
    </n-flex>
  </n-modal>
  <n-modal
    v-model:show="unCompressModal"
    preset="card"
    :title="'解压缩 - ' + unCompressModel.file"
    style="width: 60vw"
    size="huge"
    :bordered="false"
    :segmented="false"
  >
    <n-flex vertical>
      <n-form>
        <n-form-item label="解压到">
          <n-input v-model:value="unCompressModel.path" />
        </n-form-item>
      </n-form>
      <n-button type="primary" @click="handleUnCompress">解压</n-button>
    </n-flex>
  </n-modal>
</template>

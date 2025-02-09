<script setup lang="ts">
import file from '@/api/panel/file'
import TheIcon from '@/components/custom/TheIcon.vue'
import { checkName, checkPath, getExt, getIconByExt } from '@/utils'
import type { DataTableColumns, InputInst } from 'naive-ui'
import { NButton, NDataTable, NEllipsis, NFlex, NTag } from 'naive-ui'
import type { RowData } from 'naive-ui/es/data-table/src/interface'

const show = defineModel<boolean>('show', { type: Boolean, required: true })
const path = defineModel<string>('path', { type: String, required: true })
const props = defineProps({
  dir: {
    type: Boolean,
    required: true
  }
})

const title = computed(() => (props.dir ? '选择目录' : '选择文件'))
const isInput = ref(false)
const pathInput = ref<InputInst | null>(null)
const input = ref('www')
const sort = ref<string>('')
const selected = defineModel<any[]>('selected', { type: Array, default: () => [] })
const create = ref(false)
const createModel = ref({
  dir: false,
  path: ''
})

const columns: DataTableColumns<RowData> = [
  {
    type: 'selection',
    multiple: false,
    fixed: 'left',
    disabled(row) {
      return props.dir ? !row.dir : row.dir
    }
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
              selected.value = [row.full]
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
  }
]

const { loading, data, page, total, pageSize, pageCount, refresh } = usePagination(
  (page, pageSize) => file.list(path.value, page, pageSize, sort.value),
  {
    initialData: { total: 0, list: [] },
    initialPageSize: 100,
    total: (res: any) => res.total,
    data: (res: any) => res.items
  }
)

const handleInput = () => {
  isInput.value = true
  nextTick(() => {
    pathInput.value?.focus()
  })
}

const handleBlur = () => {
  input.value = input.value.replace(/(^\/)|(\/$)/g, '')
  if (!checkPath(input.value)) {
    window.$message.error('路径不合法')
    return
  }

  isInput.value = false
  path.value = '/' + input.value
}

const handleUp = () => {
  const count = splitPath(path.value, '/').length
  setPath(count - 2)
}

const splitPath = (str: string, delimiter: string) => {
  if (str === delimiter || str === '') {
    return []
  }
  return str.split(delimiter).slice(1)
}

const setPath = (index: number) => {
  const newPath = splitPath(path.value, '/')
    .slice(0, index + 1)
    .join('/')
  path.value = '/' + newPath
  input.value = newPath
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

const showCreate = (value: string) => {
  createModel.value.dir = value !== 'file'
  createModel.value.path = ''
  create.value = true
}

const handleCreate = () => {
  if (!checkName(createModel.value.path)) {
    window.$message.error('名称不合法')
    return
  }

  const fullPath = path.value + '/' + createModel.value.path
  useRequest(file.create(fullPath, createModel.value.dir)).onSuccess(() => {
    create.value = false
    refresh()
    window.$message.success('新建成功')
  })
}

const closeWatch = watch(
  path,
  (value) => {
    input.value = value.slice(1)
    selected.value = []
    refresh()
  },
  { immediate: true }
)

const handleClose = () => {
  closeWatch()
  if (selected.value.length) {
    path.value = selected.value[0]
    selected.value = []
  }
  show.value = false
}
</script>

<template>
  <n-modal
    v-model:show="show"
    preset="card"
    :title="title"
    style="width: 60vw"
    size="huge"
    :bordered="false"
    :segmented="false"
    @close="handleClose"
    @mask-click="handleClose"
  >
    <n-flex>
      <n-popselect
        :options="[
          { label: '文件', value: 'file' },
          { label: '文件夹', value: 'folder' }
        ]"
        @update:value="showCreate"
      >
        <n-button type="primary"> 新建 </n-button>
      </n-popselect>
      <n-button @click="handleUp">
        <icon-bi-arrow-up />
      </n-button>
      <n-input-group flex-1>
        <n-tag size="large" v-if="!isInput" flex-1 @click="handleInput">
          <n-breadcrumb separator=">">
            <n-breadcrumb-item @click.stop="setPath(-1)"> 根目录 </n-breadcrumb-item>
            <n-breadcrumb-item
              v-for="(item, index) in splitPath(path, '/')"
              :key="index"
              @click.stop="setPath(index)"
            >
              {{ item }}
            </n-breadcrumb-item>
          </n-breadcrumb>
        </n-tag>
        <n-input-group-label v-if="isInput">/</n-input-group-label>
        <n-input
          ref="pathInput"
          v-model:value="input"
          v-if="isInput"
          @keyup.enter="handleBlur"
          @blur="handleBlur"
        />
      </n-input-group>
      <n-button @click="refresh">
        <icon-bi-arrow-clockwise />
      </n-button>
    </n-flex>
    <n-data-table
      remote
      striped
      virtual-scroll
      pt-20
      size="small"
      :scroll-x="600"
      :columns="columns"
      :data="data"
      :loading="loading"
      :row-key="(row: any) => row.full"
      max-height="60vh"
      @update:sorter="handleSorterChange"
      v-model:checked-row-keys="selected"
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
  </n-modal>
  <n-modal
    v-model:show="create"
    preset="card"
    title="新建"
    style="width: 60vw"
    size="huge"
    :bordered="false"
    :segmented="false"
  >
    <n-space vertical>
      <n-form :model="createModel">
        <n-form-item label="名称">
          <n-input v-model:value="createModel.path" />
        </n-form-item>
      </n-form>
      <n-button type="info" block @click="handleCreate">提交</n-button>
    </n-space>
  </n-modal>
</template>

<style scoped lang="scss"></style>

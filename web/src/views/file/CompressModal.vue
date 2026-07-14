<script setup lang="ts">
import { NButton, NInput } from 'naive-ui'
import { useGettext } from 'vue3-gettext'

import api from '@/api/panel/file'
import { useFileStore } from '@/stores'
import { generateRandomString, getBase } from '@/utils'

const { $gettext } = useGettext()
const fileStore = useFileStore()
const show = defineModel<boolean>('show', { type: Boolean, required: true })
const path = defineModel<string>('path', { type: String, required: true })
// 打开时快照选中项，弹窗内编辑不影响列表选中状态
const paths = ref<string[]>([])
const format = ref('.zip')
const loading = ref(false)

// 生成随机文件名
const generateName = () => {
  // 如果选择多个文件，文件名为目录名 + 随机字符串，否则就直接文件名 + 随机字符串
  // 特殊处理根目录，防止出现 //xxx 的情况
  if (path.value == '/') {
    return paths.value.length > 1
      ? `/${generateRandomString(6)}${format.value}`
      : `${paths.value[0]}-${generateRandomString(6)}${format.value}`
  }
  const parts = path.value.split('/')
  return paths.value.length > 1
    ? `${path.value}/${parts.pop()}-${generateRandomString(6)}${format.value}`
    : `${paths.value[0]}-${generateRandomString(6)}${format.value}`
}

const file = ref('')

const ensureExtension = (extension: string) => {
  if (!file.value.endsWith(extension)) {
    file.value = `${getBase(file.value)}${extension}`
  }
}

const handleArchive = () => {
  ensureExtension(format.value)
  loading.value = true
  const relative = paths.value.map((item) => item.replace(path.value, '').replace(/^\//, ''))
  useRequest(api.compress(path.value, relative, file.value))
    .onSuccess(() => {
      show.value = false
      if (fileStore.activeTab) {
        fileStore.activeTab.selected = []
      }
      window.$message.success(
        $gettext('Compress task created successfully, please check the task list for progress'),
      )
    })
    .onComplete(() => {
      loading.value = false
    })
}

// 弹窗打开时快照选中项并生成默认文件名，弹窗内编辑不重置用户已修改的名字
watch(show, (val) => {
  if (val) {
    paths.value = [...(fileStore.activeTab?.selected ?? [])]
    file.value = generateName()
  }
})
</script>

<template>
  <n-modal
    v-model:show="show"
    preset="card"
    :title="$gettext('Compress')"
    style="width: 60vw"
    size="huge"
    :bordered="false"
    :segmented="false"
  >
    <n-flex vertical>
      <n-form>
        <n-form-item :label="$gettext('Files to compress')">
          <n-dynamic-input v-model:value="paths" :min="1" />
        </n-form-item>
        <n-form-item :label="$gettext('Compress to')">
          <n-input v-model:value="file" />
        </n-form-item>
        <n-form-item :label="$gettext('Format')">
          <n-select
            v-model:value="format"
            :options="[
              { label: '.zip', value: '.zip' },
              { label: '.gz', value: '.gz' },
              { label: '.tar', value: '.tar' },
              { label: '.tar.gz', value: '.tar.gz' },
              { label: '.tgz', value: '.tgz' },
              { label: '.tar.bz2', value: '.tar.bz2' },
              { label: '.tar.xz', value: '.tar.xz' },
              { label: '.tar.zst', value: '.tar.zst' },
              { label: '.7z', value: '.7z' },
            ]"
            @update:value="ensureExtension"
          />
        </n-form-item>
      </n-form>
      <n-button :loading="loading" :disabled="loading" type="primary" @click="handleArchive">
        {{ $gettext('Compress') }}
      </n-button>
    </n-flex>
  </n-modal>
</template>

<style scoped lang="scss"></style>

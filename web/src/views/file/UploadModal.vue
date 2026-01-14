<script setup lang="ts">
import type { UploadCustomRequestOptions, UploadFileInfo, UploadInst } from 'naive-ui'
import { useGettext } from 'vue3-gettext'

import api from '@/api/panel/file'

const { $gettext } = useGettext()
const show = defineModel<boolean>('show', { type: Boolean, required: true })
const path = defineModel<string>('path', { type: String, required: true })

const props = defineProps<{
  initialFiles?: File[]
}>()

const upload = ref<UploadInst | null>(null)
const fileList = ref<UploadFileInfo[]>([])

// 文件数量阈值，超过此数量需要二次确认
const FILE_COUNT_THRESHOLD = 100

// 监听预拖入的文件
watch(
  () => props.initialFiles,
  async (files) => {
    if (files && files.length > 0) {
      // 如果文件数量超过阈值，弹窗确认
      if (files.length > FILE_COUNT_THRESHOLD) {
        window.$dialog.warning({
          title: $gettext('Confirm Upload'),
          content: $gettext(
            'You are about to upload %{count} files. This may take a while. Do you want to continue?',
            { count: files.length }
          ),
          positiveText: $gettext('Continue'),
          negativeText: $gettext('Cancel'),
          onPositiveClick: () => {
            addFilesToList(files, true)
          },
          onNegativeClick: () => {
            show.value = false
          }
        })
      } else {
        addFilesToList(files, true)
      }
    }
  },
  { immediate: true }
)

// 将文件添加到上传列表
const addFilesToList = (files: File[], autoUpload: boolean = false) => {
  const newFiles: UploadFileInfo[] = files.map((file, index) => ({
    id: `dropped-${Date.now()}-${index}`,
    name: file.name,
    status: 'pending' as const,
    file: file
  }))
  fileList.value = newFiles

  // 自动开始上传
  if (autoUpload) {
    nextTick(() => {
      upload.value?.submit()
    })
  }
}

// 监听弹窗关闭，清空文件列表
watch(show, (val) => {
  if (!val) {
    fileList.value = []
  }
})

const uploadRequest = ({ file, onFinish, onError, onProgress }: UploadCustomRequestOptions) => {
  const formData = new FormData()
  formData.append('path', `${path.value}/${file.name}`)
  formData.append('file', file.file as File)
  const { uploading } = useRequest(api.upload(formData))
    .onSuccess(() => {
      onFinish()
      window.$bus.emit('file:refresh')
      window.$message.success($gettext('Upload %{ fileName } successful', { fileName: file.name }))
    })
    .onError(() => {
      onError()
    })
    .onComplete(() => {
      stopWatch()
    })
  const stopWatch = watch(uploading, (progress) => {
    onProgress({ percent: Math.ceil((progress.loaded / progress.total) * 100) })
  })
}

// 处理文件选择变化（用于文件数量确认）
const handleChange = (data: { fileList: UploadFileInfo[] }) => {
  const newFiles = data.fileList.filter(
    (f) => !fileList.value.some((existing) => existing.id === f.id)
  )

  // 如果新增文件数量超过阈值，弹窗确认
  if (newFiles.length > FILE_COUNT_THRESHOLD) {
    window.$dialog.warning({
      title: $gettext('Confirm Upload'),
      content: $gettext(
        'You are about to upload %{count} files. This may take a while. Do you want to continue?',
        { count: newFiles.length }
      ),
      positiveText: $gettext('Continue'),
      negativeText: $gettext('Cancel'),
      onPositiveClick: () => {
        fileList.value = data.fileList
      }
    })
  } else {
    fileList.value = data.fileList
  }
}
</script>

<template>
  <n-modal
    v-model:show="show"
    preset="card"
    :title="$gettext('Upload')"
    style="width: 60vw"
    size="huge"
    :bordered="false"
    :segmented="false"
  >
    <n-flex vertical>
      <n-upload
        ref="upload"
        v-model:file-list="fileList"
        multiple
        directory-dnd
        :custom-request="uploadRequest"
        @change="handleChange"
      >
        <n-upload-dragger>
          <div style="margin-bottom: 12px">
            <the-icon :size="60" icon="mdi:arrow-up-bold-box-outline" />
          </div>
          <NText text-18> {{ $gettext('Click or drag files to this area to upload') }}</NText>
          <NP depth="3" m-10>
            {{
              $gettext('For large files, it is recommended to use SFTP and other methods to upload')
            }}
          </NP>
        </n-upload-dragger>
      </n-upload>
    </n-flex>
  </n-modal>
</template>

<style scoped lang="scss"></style>

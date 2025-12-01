<script setup lang="ts">
import { NButton } from 'naive-ui'
import { useGettext } from 'vue3-gettext'

import website from '@/api/panel/website'

const show = defineModel<boolean>('show', { type: Boolean, required: true })
const type = defineModel<string>('type', { type: String, required: true })

const { $gettext } = useGettext()

const bulkCreate = ref('')

const handleCreate = async () => {
  // 按行分割
  const lines = bulkCreate.value.split('\n')
  // 去除空行
  const filteredLines = lines.filter((line) => line.trim() !== '')
  // 解析每一行
  for (const line of filteredLines) {
    const parts = line.split('|')
    if (parts.length < 4) {
      window.$message.error($gettext('The format is incorrect, please check'))
      return
    }
    // 去除空格
    const name = parts[0].trim()
    const domains = parts[1]
      .trim()
      .split(',')
      .map((item) => item.trim())
    const listens = parts[2]
      .trim()
      .split(',')
      .map((item) => item.trim())
    const path = parts[3].trim()
    const remark = parts[4] ? parts[4].trim() : ''
    let model = {
      name: '',
      listens: [] as Array<string>,
      domains: [] as Array<string>,
      path: '',
      remark: ''
    }
    model.name = name
    model.domains = domains
    model.listens = listens
    model.path = path
    model.remark = remark
    // 去除空的域名和端口
    model.domains = model.domains.filter((item) => item !== '')
    model.listens = model.listens.filter((item) => item !== '')
    // 端口为空自动添加 80 端口
    if (model.listens.length === 0) {
      model.listens.push('80')
    }
    // 端口中去掉 443 端口，nginx 不允许在未配置证书下监听 443 端口
    model.listens = model.listens.filter((item) => item !== '443')
    useRequest(website.create(model)).onSuccess(() => {
      window.$message.success(
        $gettext('Website %{ name } created successfully', { name: model.name })
      )
      model = {
        name: '',
        domains: [] as Array<string>,
        listens: [] as Array<string>,
        path: '',
        remark: ''
      }
      window.$bus.emit('website:refresh')
    })
  }
}
</script>

<template>
  <n-modal
    v-model:show="show"
    :title="$gettext('Bulk Create Website')"
    preset="card"
    style="width: 60vw"
    size="huge"
    :bordered="false"
    :segmented="false"
    @close="show = false"
  >
    <n-flex vertical>
      <n-alert type="info">
        {{
          $gettext(
            'Please enter the website name, domain, port, path, and remark in the text area below, one per line.'
          )
        }}
      </n-alert>
      <n-input
        type="textarea"
        :autosize="{ minRows: 10, maxRows: 15 }"
        :placeholder="$gettext('name|domain|port|path|remark')"
        v-model:value="bulkCreate"
      />
      <n-text>
        {{
          $gettext(
            'Name: The name of the website, which will be displayed in the website list, must be unique.'
          )
        }}
      </n-text>
      <n-text>
        {{
          $gettext(
            'Domain: The domain name of the website, multiple domains can be separated by commas.'
          )
        }}
      </n-text>
      <n-text>
        {{
          $gettext(
            'Port: The port number of the website, multiple ports can be separated by commas.'
          )
        }}
      </n-text>
      <n-text>
        {{ $gettext('Path: The path of the website, can be empty to use the default path.') }}
      </n-text>
      <n-text>
        {{ $gettext('Remark: The remark of the website, can be empty.') }}
      </n-text>
      <n-button type="info" block @click="handleCreate">
        {{ $gettext('Create') }}
      </n-button>
    </n-flex>
  </n-modal>
</template>

<style scoped lang="scss"></style>

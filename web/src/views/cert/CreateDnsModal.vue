<script setup lang="ts">
import cert from '@/api/panel/cert'
import { NButton, NInput, NSpace } from 'naive-ui'

const show = defineModel<boolean>('show', { type: Boolean, required: true })

const props = defineProps({
  dnsProviders: {
    type: Array<any>,
    required: true
  }
})

const { dnsProviders } = toRefs(props)

const model = ref<any>({
  data: {
    ak: '',
    sk: ''
  },
  type: 'aliyun',
  name: ''
})

const handleCreateDNS = async () => {
  useRequest(cert.dnsCreate(model.value)).onSuccess(() => {
    window.$bus.emit('cert:refresh-dns')
    window.$bus.emit('cert:refresh-async')
    show.value = false
    model.value.data.ak = ''
    model.value.data.sk = ''
    model.value.name = ''
    window.$message.success('创建成功')
  })
}
</script>

<template>
  <n-modal
    v-model:show="show"
    preset="card"
    title="创建 DNS"
    style="width: 60vw"
    size="huge"
    :bordered="false"
    :segmented="false"
  >
    <n-space vertical>
      <n-form :model="model">
        <n-form-item path="name" label="备注名称">
          <n-input v-model:value="model.name" type="text" placeholder="输入备注名称" />
        </n-form-item>
        <n-form-item path="type" label="DNS">
          <n-select
            v-model:value="model.type"
            placeholder="选择 DNS"
            clearable
            :options="dnsProviders"
          />
        </n-form-item>
        <n-form-item v-if="model.type == 'aliyun'" path="ak" label="Access Key">
          <n-input v-model:value="model.data.ak" type="text" placeholder="输入阿里云 Access Key" />
        </n-form-item>
        <n-form-item v-if="model.type == 'aliyun'" path="sk" label="Secret Key">
          <n-input v-model:value="model.data.sk" type="text" placeholder="输入阿里云 Secret Key" />
        </n-form-item>
        <n-form-item v-if="model.type == 'tencent'" path="ak" label="SecretId">
          <n-input v-model:value="model.data.ak" type="text" placeholder="输入腾讯云 SecretId" />
        </n-form-item>
        <n-form-item v-if="model.type == 'tencent'" path="sk" label="SecretKey">
          <n-input v-model:value="model.data.sk" type="text" placeholder="输入腾讯云 SecretKey" />
        </n-form-item>
        <n-form-item v-if="model.type == 'huawei'" path="ak" label="AccessKeyId">
          <n-input v-model:value="model.data.ak" type="text" placeholder="输入华为云 AccessKeyId" />
        </n-form-item>
        <n-form-item v-if="model.type == 'huawei'" path="sk" label="SecretAccessKey">
          <n-input
            v-model:value="model.data.sk"
            type="text"
            placeholder="输入华为云 SecretAccessKey"
          />
        </n-form-item>
        <n-form-item v-if="model.type == 'westcn'" path="sk" label="Username">
          <n-input v-model:value="model.data.sk" type="text" placeholder="输入西部数码 Username" />
        </n-form-item>
        <n-form-item v-if="model.type == 'westcn'" path="ak" label="API Password">
          <n-input
            v-model:value="model.data.ak"
            type="text"
            placeholder="输入西部数码 API Password"
          />
        </n-form-item>
        <n-form-item v-if="model.type == 'cloudflare'" path="ak" label="API Key">
          <n-input
            v-model:value="model.data.ak"
            type="text"
            placeholder="输入 Cloudflare API Key"
          />
        </n-form-item>
        <n-form-item v-if="model.type == 'godaddy'" path="ak" label="Token">
          <n-input v-model:value="model.data.ak" type="text" placeholder="输入 GoDaddy Token" />
        </n-form-item>
        <n-form-item v-if="model.type == 'gcore'" path="ak" label="API Key">
          <n-input v-model:value="model.data.ak" type="text" placeholder="输入 G-Core API Key" />
        </n-form-item>
        <n-form-item v-if="model.type == 'porkbun'" path="ak" label="API Key">
          <n-input v-model:value="model.data.ak" type="text" placeholder="输入 Porkbun API Key" />
        </n-form-item>
        <n-form-item v-if="model.type == 'porkbun'" path="sk" label="Secret Key">
          <n-input
            v-model:value="model.data.sk"
            type="text"
            placeholder="输入 Porkbun Secret Key"
          />
        </n-form-item>
        <n-form-item v-if="model.type == 'namecheap'" path="sk" label="API Username">
          <n-input
            v-model:value="model.data.sk"
            type="text"
            placeholder="输入 Namecheap API Username"
          />
        </n-form-item>
        <n-form-item v-if="model.type == 'namecheap'" path="ak" label="API Key">
          <n-input v-model:value="model.data.ak" type="text" placeholder="输入 Namecheap API Key" />
        </n-form-item>
        <n-form-item v-if="model.type == 'namesilo'" path="ak" label="API Token">
          <n-input
            v-model:value="model.data.ak"
            type="text"
            placeholder="输入 NameSilo API Token"
          />
        </n-form-item>
        <n-form-item v-if="model.type == 'namecom'" path="sk" label="Username">
          <n-input v-model:value="model.data.sk" type="text" placeholder="输入 Name.com Username" />
        </n-form-item>
        <n-form-item v-if="model.type == 'namecom'" path="ak" label="Token">
          <n-input v-model:value="model.data.ak" type="text" placeholder="输入 Name.com Token" />
        </n-form-item>

        <n-form-item v-if="model.type == 'cloudns'" path="ak" label="Auth ID">
          <n-input
            v-model:value="model.data.ak"
            type="text"
            placeholder="输入 ClouDNS Auth ID（使用Sub Auth ID请添加sub-前缀）"
          />
        </n-form-item>
        <n-form-item v-if="model.type == 'cloudns'" path="sk" label="Auth Password">
          <n-input
            v-model:value="model.data.sk"
            type="text"
            placeholder="输入 ClouDNS Auth Password"
          />
        </n-form-item>
        <n-form-item v-if="model.type == 'duckdns'" path="ak" label="Token">
          <n-input v-model:value="model.data.ak" type="text" placeholder="输入 Duck DNS Token" />
        </n-form-item>
        <n-form-item v-if="model.type == 'hetzner'" path="ak" label="Auth API Token">
          <n-input
            v-model:value="model.data.ak"
            type="text"
            placeholder="输入 Hetzner Auth API Token"
          />
        </n-form-item>
        <n-form-item v-if="model.type == 'linode'" path="ak" label="Token">
          <n-input v-model:value="model.data.ak" type="text" placeholder="输入 Linode Token" />
        </n-form-item>
        <n-form-item v-if="model.type == 'vercel'" path="ak" label="Token">
          <n-input v-model:value="model.data.ak" type="text" placeholder="输入 Vercel Token" />
        </n-form-item>
      </n-form>
      <n-button type="info" block @click="handleCreateDNS">提交</n-button>
    </n-space>
  </n-modal>
</template>

<style scoped lang="scss"></style>

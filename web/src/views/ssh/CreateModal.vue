<script setup lang="ts">
import ssh from '@/api/panel/ssh'
import { NInput } from 'naive-ui'

const show = defineModel<boolean>('show', { type: Boolean, required: true })
const loading = ref(false)

const model = ref({
  name: '',
  host: '127.0.0.1',
  port: 22,
  auth_method: 'password',
  user: 'root',
  password: '',
  key: '',
  remark: ''
})

const handleSubmit = () => {
  loading.value = true
  useRequest(ssh.create(model.value))
    .onSuccess(() => {
      loading.value = false
      show.value = false
      model.value = {
        name: '',
        host: '127.0.0.1',
        port: 22,
        auth_method: 'password',
        user: 'root',
        password: '',
        key: '',
        remark: ''
      }
      window.$bus.emit('ssh:refresh')
      window.$message.success('创建成功')
    })
    .onComplete(() => {
      loading.value = false
    })
}
</script>

<template>
  <n-modal
    v-model:show="show"
    preset="card"
    title="创建主机"
    style="width: 60vw"
    size="huge"
    :bordered="false"
    :segmented="false"
  >
    <n-form>
      <n-form-item label="名称">
        <n-input v-model:value="model.name" placeholder="127.0.0.1" />
      </n-form-item>
      <n-row :gutter="[0, 24]" pt-20>
        <n-col :span="15">
          <n-form-item label="主机">
            <n-input v-model:value="model.host" placeholder="127.0.0.1" />
          </n-form-item>
        </n-col>
        <n-col :span="2"> </n-col>
        <n-col :span="7">
          <n-form-item label="端口">
            <n-input-number v-model:value="model.port" :min="1" :max="65535" />
          </n-form-item>
        </n-col>
      </n-row>
      <n-form-item label="认证方式">
        <n-select
          v-model:value="model.auth_method"
          :options="[
            { label: '密码', value: 'password' },
            { label: '私钥', value: 'publickey' }
          ]"
        >
        </n-select>
      </n-form-item>
      <n-form-item v-if="model.auth_method == 'password'" label="用户名">
        <n-input v-model:value="model.user" placeholder="root" />
      </n-form-item>
      <n-form-item v-if="model.auth_method == 'password'" label="密码">
        <n-input v-model:value="model.password" type="password" show-password-on="click" />
      </n-form-item>
      <n-form-item v-if="model.auth_method == 'publickey'" label="私钥">
        <n-input v-model:value="model.key" type="textarea" />
      </n-form-item>
      <n-form-item label="备注">
        <n-input v-model:value="model.remark" type="textarea" />
      </n-form-item>
    </n-form>
    <n-row :gutter="[0, 24]" pt-20>
      <n-col :span="24">
        <n-button type="info" block :loading="loading" @click="handleSubmit"> 提交 </n-button>
      </n-col>
    </n-row>
  </n-modal>
</template>

<style scoped lang="scss"></style>

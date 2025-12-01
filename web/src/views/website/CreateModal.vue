<script setup lang="ts">
import { NButton, NInput } from 'naive-ui'
import { useGettext } from 'vue3-gettext'

import home from '@/api/panel/home'
import website from '@/api/panel/website'
import { generateRandomString } from '@/utils'

const show = defineModel<boolean>('show', { type: Boolean, required: true })
const type = defineModel<string>('type', { type: String, required: true })

const { $gettext } = useGettext()

const createModel = ref({
  type: '',
  name: '',
  listens: [] as Array<string>,
  domains: [] as Array<string>,
  path: '',
  db: false,
  db_type: '0',
  db_name: '',
  db_user: '',
  db_password: '',
  remark: '',

  php: 0,
  proxy: ''
})

const { data: installedDbAndPhp } = useRequest(home.installedDbAndPhp, {
  initialData: {
    php: [
      {
        label: $gettext('Not used'),
        value: 0
      }
    ],
    db: [
      {
        label: '',
        value: ''
      }
    ]
  }
})

const handleCreate = async () => {
  createModel.value.type = type.value
  // 去除空的域名和端口
  createModel.value.domains = createModel.value.domains.filter((item) => item !== '')
  createModel.value.listens = createModel.value.listens.filter((item) => item !== '')
  // 端口为空自动添加 80 端口
  if (createModel.value.listens.length === 0) {
    createModel.value.listens.push('80')
  }
  // 端口中去掉 443 端口，不允许在未配置证书下监听 443 端口
  createModel.value.listens = createModel.value.listens.filter((item) => item !== '443')
  useRequest(website.create(createModel.value)).onSuccess(() => {
    window.$bus.emit('website:refresh')
    window.$message.success(
      $gettext('Website %{ name } created successfully', { name: createModel.value.name })
    )
    show.value = false
    createModel.value = {
      type: '',
      name: '',
      domains: [] as Array<string>,
      listens: [] as Array<string>,
      db: false,
      db_type: '0',
      db_name: '',
      db_user: '',
      db_password: '',
      path: '',
      remark: '',
      php: 0,
      proxy: ''
    }
  })
}

const formatDbValue = (value: string) => {
  value = value.replace(/\./g, '_')
  value = value.replace(/-/g, '_')
  if (value.length > 16) {
    value = value.substring(0, 16)
  }

  return value
}
</script>

<template>
  <n-modal
    v-model:show="show"
    :title="$gettext('Create Website')"
    preset="card"
    style="width: 60vw"
    size="huge"
    :bordered="false"
    :segmented="false"
    @close="show = false"
  >
    <n-form :model="createModel">
      <n-form-item path="name" :label="$gettext('Name')">
        <n-input
          v-model:value="createModel.name"
          type="text"
          @keydown.enter.prevent
          :placeholder="
            $gettext('Must use English for the website name, it cannot be modified after setting')
          "
        />
      </n-form-item>
      <n-row :gutter="[0, 24]">
        <n-col :span="11">
          <n-form-item :label="$gettext('Domain')">
            <n-dynamic-input
              v-model:value="createModel.domains"
              placeholder="example.com"
              :min="1"
              show-sort-button
            />
          </n-form-item>
        </n-col>
        <n-col :span="2"></n-col>
        <n-col :span="11">
          <n-form-item :label="$gettext('Port')">
            <n-dynamic-input
              v-model:value="createModel.listens"
              placeholder="80"
              :min="1"
              show-sort-button
            />
          </n-form-item>
        </n-col>
      </n-row>
      <n-row v-if="type == 'php'" :gutter="[0, 24]">
        <n-col :span="11">
          <n-form-item path="php" :label="$gettext('PHP Version')">
            <n-select
              v-model:value="createModel.php"
              :options="installedDbAndPhp.php"
              :placeholder="$gettext('Select PHP Version')"
              @keydown.enter.prevent
            >
            </n-select>
          </n-form-item>
        </n-col>
        <n-col :span="2"></n-col>
        <n-col :span="11">
          <n-form-item path="db" :label="$gettext('Database')">
            <n-select
              v-model:value="createModel.db_type"
              :options="installedDbAndPhp.db"
              :placeholder="$gettext('Select Database')"
              @keydown.enter.prevent
              @update:value="
                () => {
                  createModel.db = createModel.db_type != '0'
                  createModel.db_name = formatDbValue(createModel.name)
                  createModel.db_user = formatDbValue(createModel.name)
                  createModel.db_password = generateRandomString(16)
                }
              "
            >
            </n-select>
          </n-form-item>
        </n-col>
      </n-row>
      <n-row v-if="type == 'php'" :gutter="[0, 24]">
        <n-col :span="7">
          <n-form-item v-if="createModel.db" path="db_name" :label="$gettext('Database Name')">
            <n-input
              v-model:value="createModel.db_name"
              type="text"
              @keydown.enter.prevent
              :placeholder="$gettext('Database Name')"
            />
          </n-form-item>
        </n-col>
        <n-col :span="1"></n-col>
        <n-col :span="7">
          <n-form-item v-if="createModel.db" path="db_user" :label="$gettext('Database User')">
            <n-input
              v-model:value="createModel.db_user"
              type="text"
              @keydown.enter.prevent
              :placeholder="$gettext('Database User')"
            />
          </n-form-item>
        </n-col>
        <n-col :span="1"></n-col>
        <n-col :span="8">
          <n-form-item
            v-if="createModel.db"
            path="db_password"
            :label="$gettext('Database Password')"
          >
            <n-input
              v-model:value="createModel.db_password"
              type="text"
              @keydown.enter.prevent
              :placeholder="$gettext('Database Password')"
            />
          </n-form-item>
        </n-col>
      </n-row>
      <n-form-item v-if="type != 'proxy'" path="path" :label="$gettext('Directory')">
        <n-input
          v-model:value="createModel.path"
          type="text"
          @keydown.enter.prevent
          :placeholder="
            $gettext(
              'Website root directory (if left empty, defaults to website directory/website name/public)'
            )
          "
        />
      </n-form-item>
      <n-form-item v-if="type == 'proxy'" path="path" :label="$gettext('Proxy Target')">
        <n-input
          v-model:value="createModel.proxy"
          type="text"
          @keydown.enter.prevent
          :placeholder="$gettext('Proxy target address (e.g., http://127.0.0.1:3000)')"
        />
      </n-form-item>
      <n-form-item path="remark" :label="$gettext('Remark')">
        <n-input
          v-model:value="createModel.remark"
          type="textarea"
          @keydown.enter.prevent
          :placeholder="$gettext('Remark')"
        />
      </n-form-item>
    </n-form>
    <n-button type="info" block @click="handleCreate">
      {{ $gettext('Create') }}
    </n-button>
  </n-modal>
</template>

<style scoped lang="scss"></style>

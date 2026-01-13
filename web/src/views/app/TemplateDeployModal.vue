<script setup lang="ts">
import templateApi from '@/api/panel/template'
import PtyTerminalModal from '@/components/common/PtyTerminalModal.vue'
import { useGettext } from 'vue3-gettext'

import type { Template, TemplateEnvironment } from './types'

const { $gettext } = useGettext()

const props = defineProps<{
  template: Template | null
}>()

const emit = defineEmits<{
  success: []
}>()

const show = defineModel<boolean>('show', { type: Boolean, required: true })

const doSubmit = ref(false)
const currentTab = ref('basic')

// 启动终端
const upModal = ref(false)
const upCommand = ref('')

const deployModel = reactive({
  name: '',
  autoStart: true,
  autoFirewall: false,
  envs: {} as Record<string, any>
})

// 初始化环境变量默认值
const initEnvDefaults = () => {
  if (!props.template?.environments) return
  const envs: Record<string, string> = {}
  props.template.environments.forEach((env: TemplateEnvironment) => {
    envs[env.name] = env.default || ''
  })
  deployModel.envs = envs
}

// 获取 select 选项
const getSelectOptions = (env: TemplateEnvironment) => {
  if (!env.options) return []
  return Object.entries(env.options).map(([value, label]) => ({
    label,
    value
  }))
}

// 提交部署
const handleSubmit = async () => {
  if (!props.template) return

  if (!deployModel.name.trim()) {
    window.$message.warning($gettext('Please enter compose name'))
    return
  }

  doSubmit.value = true

  try {
    // 构建环境变量数组
    const envs = Object.entries(deployModel.envs).map(([key, value]) => ({
      key,
      value: String(value)
    }))

    // 创建 compose
    await templateApi.create({
      slug: props.template.slug,
      name: deployModel.name,
      envs,
      auto_firewall: deployModel.autoFirewall
    })

    window.$message.success($gettext('Created successfully'))

    if (deployModel.autoStart) {
      // 自动启动
      upCommand.value = `docker compose -f /opt/ace/server/compose/${deployModel.name}/docker-compose.yml up -d`
      upModal.value = true
    } else {
      show.value = false
      emit('success')
    }
  } finally {
    doSubmit.value = false
  }
}

// 启动完成
const handleUpComplete = () => {
  show.value = false
  emit('success')
}

const resetForm = () => {
  deployModel.name = ''
  deployModel.autoStart = true
  deployModel.autoFirewall = false
  deployModel.envs = {}
  currentTab.value = 'basic'
  initEnvDefaults()
}

watch(show, (val) => {
  if (val) {
    resetForm()
  }
})

watch(
  () => props.template,
  () => {
    if (props.template) {
      initEnvDefaults()
    }
  }
)
</script>

<template>
  <n-modal
    v-model:show="show"
    :title="$gettext('Deploy Template') + (template ? ` - ${template.name}` : '')"
    preset="card"
    style="width: 60vw"
    size="huge"
    :bordered="false"
    :segmented="false"
    :mask-closable="!doSubmit"
    :closable="!doSubmit"
  >
    <n-tabs v-model:value="currentTab" type="line" animated>
      <!-- 基本设置 -->
      <n-tab-pane name="basic" :tab="$gettext('Basic Settings')">
        <n-form :model="deployModel" label-placement="left" label-width="120">
          <n-form-item path="name" :label="$gettext('Compose Name')">
            <n-input
              v-model:value="deployModel.name"
              type="text"
              @keydown.enter.prevent
              :placeholder="$gettext('Enter compose name')"
            />
          </n-form-item>

          <n-divider title-placement="left">{{ $gettext('Deploy Options') }}</n-divider>

          <n-row :gutter="[24, 0]">
            <n-col :span="8">
              <n-form-item path="autoStart" :label="$gettext('Auto Start')">
                <n-switch v-model:value="deployModel.autoStart" />
              </n-form-item>
            </n-col>
            <n-col :span="8">
              <n-form-item path="autoFirewall" :label="$gettext('Auto Firewall')">
                <n-switch v-model:value="deployModel.autoFirewall" />
                <template #feedback>
                  <span>
                    {{ $gettext('Automatically allow ports defined in compose') }}
                  </span>
                </template>
              </n-form-item>
            </n-col>
          </n-row>
        </n-form>
      </n-tab-pane>

      <!-- 环境变量 -->
      <n-tab-pane
        v-if="template?.environments?.length"
        name="environment"
        :tab="$gettext('Environment Variables')"
      >
        <n-form :model="deployModel" label-placement="left" label-width="160">
          <n-form-item
            v-for="env in template.environments"
            :key="env.name"
            :label="env.description"
          >
            <!-- Select 类型 -->
            <n-select
              v-if="env.type === 'select'"
              v-model:value="deployModel.envs[env.name]"
              :options="getSelectOptions(env)"
              :placeholder="$gettext('Select value')"
            />
            <!-- Number/Port 类型 -->
            <n-input-number
              v-else-if="env.type === 'number' || env.type === 'port'"
              v-model:value="deployModel.envs[env.name]"
              :min="env.type === 'port' ? 1 : undefined"
              :max="env.type === 'port' ? 65535 : undefined"
              style="width: 100%"
              :placeholder="env.default || ''"
            />
            <!-- Password 类型 -->
            <n-input
              v-else-if="env.type === 'password'"
              v-model:value="deployModel.envs[env.name]"
              type="password"
              show-password-on="click"
              :placeholder="env.default || ''"
            />
            <!-- Text 类型 (默认) -->
            <n-input
              v-else
              v-model:value="deployModel.envs[env.name]"
              :placeholder="env.default || ''"
            />
          </n-form-item>
        </n-form>
      </n-tab-pane>

      <!-- Compose 预览 -->
      <n-tab-pane name="compose" :tab="$gettext('Compose Preview')">
        <common-editor :value="template?.compose || ''" lang="yaml" height="50vh" read-only />
      </n-tab-pane>
    </n-tabs>

    <template #footer>
      <n-flex justify="end">
        <n-button @click="show = false" :disabled="doSubmit">
          {{ $gettext('Cancel') }}
        </n-button>
        <n-button type="primary" :loading="doSubmit" :disabled="doSubmit" @click="handleSubmit">
          {{ $gettext('Deploy') }}
        </n-button>
      </n-flex>
    </template>
  </n-modal>

  <pty-terminal-modal
    v-model:show="upModal"
    :title="$gettext('Starting Compose') + ' - ' + deployModel.name"
    :command="upCommand"
    @complete="handleUpComplete"
  />
</template>

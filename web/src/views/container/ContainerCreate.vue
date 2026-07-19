<script setup lang="ts">
import { useGettext } from 'vue3-gettext'

import container from '@/api/panel/container'

import ImagePullModal from './ImagePullModal.vue'

const { $gettext } = useGettext()

const show = defineModel<boolean>('show', { type: Boolean, required: true })
const props = defineProps<{
  editId?: string
}>()

const isEdit = computed(() => !!props.editId)
const doSubmit = ref(false)
const currentTab = ref('basic')

// 镜像拉取
const showPullModal = ref(false)

const createModel = reactive({
  name: '',
  image: '',
  publish_all_ports: false,
  ports: [] as {
    container_start: number
    container_end: number
    host_start: number
    host_end: number
    host: string
    protocol: string
  }[],
  network: '',
  volumes: [] as {
    host: string
    container: string
    mode: string
  }[],
  cpus: 0,
  memory: 0,
  cpu_shares: 1024,
  env: [] as { key: string; value: string }[],
  labels: [] as { key: string; value: string }[],
  command: [] as string[],
  entrypoint: [] as string[],
  restart_policy: 'no',
  tty: false,
  open_stdin: false,
  auto_remove: false,
  privileged: false,
})

const networks = ref<{ label: string; value: string }[]>([])

const restartPolicyOptions = [
  { label: $gettext('None'), value: 'no' },
  { label: $gettext('Always'), value: 'always' },
  { label: $gettext('On failure (default 5 retries)'), value: 'on-failure' },
  { label: $gettext('Unless stopped'), value: 'unless-stopped' },
]

const protocolOptions = [
  { label: 'TCP', value: 'tcp' },
  { label: 'UDP', value: 'udp' },
]

const volumeModeOptions = [
  { label: $gettext('Read-Write'), value: 'rw' },
  { label: $gettext('Read-Only'), value: 'ro' },
]

// 端口映射操作
const onCreatePort = () => ({
  container_start: 80,
  container_end: 80,
  host_start: 80,
  host_end: 80,
  host: '',
  protocol: 'tcp',
})

// 挂载卷操作
const onCreateVolume = () => ({
  host: '/www',
  container: '/www',
  mode: 'rw',
})

// 环境变量操作
const onCreateEnv = () => ({ key: '', value: '' })

// 标签操作
const onCreateLabel = () => ({ key: '', value: '' })

const getNetworks = () => {
  useRequest(container.networkList(1, 1000)).onSuccess(({ data }) => {
    networks.value = data.items.map((item: any) => ({
      label: item.name,
      value: item.id,
    }))
    // 编辑模式下网络由 inspect 回填，避免竞态覆盖
    if (!isEdit.value && networks.value.length > 0) {
      createModel.network = networks.value[0]?.value ?? ''
    }
  })
}

// 创建/更新容器
const createContainer = () => {
  doSubmit.value = true
  const req = isEdit.value
    ? container.containerUpdate(props.editId!, createModel)
    : container.containerCreate(createModel)
  useRequest(req)
    .onSuccess(() => {
      window.$message.success(
        isEdit.value ? $gettext('Updated successfully') : $gettext('Created successfully'),
      )
      show.value = false
    })
    .onComplete(() => {
      doSubmit.value = false
    })
}

// 镜像拉取成功后创建容器
const onPullSuccess = () => {
  createContainer()
}

// 提交处理
const handleSubmit = () => {
  if (!createModel.image) {
    window.$message.warning($gettext('Please enter image name'))
    return
  }

  if (isEdit.value) {
    // 更新容器需要删除重建，二次确认
    window.$dialog.warning({
      title: $gettext('Confirm Update'),
      content: $gettext(
        'Updating will remove the current container and recreate it with the new configuration. Data not stored in mounted volumes will be lost. Continue?',
      ),
      positiveText: $gettext('Continue'),
      negativeText: $gettext('Cancel'),
      onPositiveClick: () => {
        checkImageAndSubmit()
      },
    })
    return
  }

  checkImageAndSubmit()
}

// 检查镜像是否存在后提交
const checkImageAndSubmit = () => {
  doSubmit.value = true

  useRequest(container.imageExist(createModel.image))
    .onSuccess(({ data }) => {
      if (data) {
        // 镜像存在，直接创建容器
        createContainer()
      } else {
        // 镜像不存在，显示拉取弹窗
        showPullModal.value = true
      }
    })
    .onComplete(() => {
      if (!showPullModal.value) {
        doSubmit.value = false
      }
    })
}

// 编辑模式：从 inspect 数据回填表单
const fillFromInspect = (info: any) => {
  createModel.name = String(info.Name || '').replace(/^\//, '')
  createModel.image = info.Config?.Image || ''
  createModel.restart_policy = info.HostConfig?.RestartPolicy?.Name || 'no'
  createModel.tty = !!info.Config?.Tty
  createModel.open_stdin = !!info.Config?.OpenStdin
  createModel.auto_remove = !!info.HostConfig?.AutoRemove
  createModel.privileged = !!info.HostConfig?.Privileged
  createModel.publish_all_ports = !!info.HostConfig?.PublishAllPorts
  createModel.cpus = (info.HostConfig?.NanoCpus || 0) / 1e9
  createModel.memory = Math.round((info.HostConfig?.Memory || 0) / 1024 / 1024)
  createModel.cpu_shares = info.HostConfig?.CpuShares || 1024
  createModel.command = info.Config?.Cmd || []
  createModel.entrypoint = info.Config?.Entrypoint || []

  // 端口映射："80/tcp" -> [{HostIp, HostPort}]
  createModel.ports = []
  for (const [containerPort, bindings] of Object.entries(info.HostConfig?.PortBindings || {})) {
    const [port, protocol] = containerPort.split('/')
    for (const binding of (bindings as any[]) || []) {
      createModel.ports.push({
        container_start: Number(port),
        container_end: Number(port),
        host_start: Number(binding.HostPort),
        host_end: Number(binding.HostPort),
        host: binding.HostIp || '',
        protocol: protocol || 'tcp',
      })
    }
  }

  // 挂载："host:container[:mode]"
  createModel.volumes = (info.HostConfig?.Binds || []).map((bind: string) => {
    const parts = bind.split(':')
    return {
      host: parts[0] || '',
      container: parts[1] || '',
      mode: parts[2] || 'rw',
    }
  })

  // 环境变量："K=V"
  createModel.env = (info.Config?.Env || []).map((env: string) => {
    const idx = env.indexOf('=')
    return { key: env.slice(0, idx), value: env.slice(idx + 1) }
  })

  // 标签
  createModel.labels = Object.entries(info.Config?.Labels || {}).map(([key, value]) => ({
    key,
    value: String(value),
  }))

  // 网络：按 NetworkID 匹配下拉项
  const networkSettings: any = Object.values(info.NetworkSettings?.Networks || {})[0]
  if (networkSettings?.NetworkID) {
    createModel.network = networkSettings.NetworkID
  }
}

// 编辑模式：加载容器当前配置
const loadContainer = () => {
  useRequest(container.containerInspect(props.editId!)).onSuccess(({ data }: any) => {
    fillFromInspect(data)
  })
}

const resetForm = () => {
  createModel.name = ''
  createModel.image = ''
  createModel.publish_all_ports = false
  createModel.ports = []
  createModel.volumes = []
  createModel.cpus = 0
  createModel.memory = 0
  createModel.cpu_shares = 1024
  createModel.env = []
  createModel.labels = []
  createModel.command = []
  createModel.entrypoint = []
  createModel.restart_policy = 'no'
  createModel.tty = false
  createModel.open_stdin = false
  createModel.auto_remove = false
  createModel.privileged = false
  currentTab.value = 'basic'
  showPullModal.value = false
}

watch(show, (val) => {
  if (val) {
    resetForm()
    getNetworks()
    if (isEdit.value) {
      loadContainer()
    }
  }
})
</script>

<template>
  <n-modal
    v-model:show="show"
    :title="isEdit ? $gettext('Edit Container') : $gettext('Create Container')"
    preset="card"
    :style="{ width: '70vw', maxWidth: '1080px' }"
    size="huge"
    :bordered="false"
    :segmented="false"
    :mask-closable="!doSubmit"
    :closable="!doSubmit"
  >
    <n-tabs v-model:value="currentTab" type="line" animated>
      <!-- 基本设置 -->
      <n-tab-pane name="basic" :tab="$gettext('Basic Settings')">
        <n-form :model="createModel" label-placement="left" label-width="120">
          <n-form-item path="name" :label="$gettext('Container Name')">
            <n-input
              v-model:value="createModel.name"
              type="text"
              @keydown.enter.prevent
              :placeholder="$gettext('Optional, auto-generated if empty')"
            />
          </n-form-item>

          <n-form-item path="image" :label="$gettext('Image')">
            <n-input
              v-model:value="createModel.image"
              type="text"
              @keydown.enter.prevent
              :placeholder="$gettext('e.g., nginx, mysql:8.4, your_username/your_image:tag')"
            />
          </n-form-item>

          <n-form-item path="network" :label="$gettext('Network')">
            <n-select
              v-model:value="createModel.network"
              :options="networks"
              :placeholder="$gettext('Select network')"
            />
          </n-form-item>

          <n-form-item path="restart_policy" :label="$gettext('Restart Policy')">
            <n-select
              v-model:value="createModel.restart_policy"
              :options="restartPolicyOptions"
              :placeholder="$gettext('Select restart policy')"
            />
          </n-form-item>

          <n-divider title-placement="left">{{ $gettext('Container Options') }}</n-divider>

          <n-row :gutter="[24, 0]">
            <n-col :span="6">
              <n-form-item path="tty" :label="$gettext('TTY (-t)')">
                <n-switch v-model:value="createModel.tty" />
              </n-form-item>
            </n-col>
            <n-col :span="6">
              <n-form-item path="open_stdin" :label="$gettext('STDIN (-i)')">
                <n-switch v-model:value="createModel.open_stdin" />
              </n-form-item>
            </n-col>
            <n-col :span="6">
              <n-form-item path="auto_remove" :label="$gettext('Auto Remove')">
                <n-switch v-model:value="createModel.auto_remove" />
              </n-form-item>
            </n-col>
            <n-col :span="6">
              <n-form-item path="privileged" :label="$gettext('Privileged')">
                <n-switch v-model:value="createModel.privileged" />
              </n-form-item>
            </n-col>
          </n-row>
        </n-form>
      </n-tab-pane>

      <!-- 端口映射 -->
      <n-tab-pane name="ports" :tab="$gettext('Port Mapping')">
        <n-form :model="createModel" label-placement="left" label-width="120">
          <n-form-item :label="$gettext('Port Mode')">
            <n-radio-group v-model:value="createModel.publish_all_ports">
              <n-radio-button :value="false">{{ $gettext('Map Ports') }}</n-radio-button>
              <n-radio-button :value="true">{{ $gettext('Expose All') }}</n-radio-button>
            </n-radio-group>
          </n-form-item>

          <n-form-item
            v-if="!createModel.publish_all_ports"
            :label="$gettext('Port Mapping')"
            :show-label="false"
          >
            <n-dynamic-input
              v-model:value="createModel.ports"
              :on-create="onCreatePort"
              show-sort-button
            >
              <template #default="{ value }">
                <n-flex align="center" :wrap="false" style="width: 100%">
                  <n-input
                    v-model:value="value.host"
                    :placeholder="$gettext('IP (optional)')"
                    class="w-30"
                  />
                  <span>:</span>
                  <n-input-number
                    v-model:value="value.host_start"
                    :min="1"
                    :max="65535"
                    :show-button="false"
                    :placeholder="$gettext('Host Start')"
                    class="w-22.5"
                  />
                  <span>-</span>
                  <n-input-number
                    v-model:value="value.host_end"
                    :min="1"
                    :max="65535"
                    :show-button="false"
                    :placeholder="$gettext('Host End')"
                    class="w-22.5"
                  />
                  <span>:</span>
                  <n-input-number
                    v-model:value="value.container_start"
                    :min="1"
                    :max="65535"
                    :show-button="false"
                    :placeholder="$gettext('Container Start')"
                    class="w-22.5"
                  />
                  <span>-</span>
                  <n-input-number
                    v-model:value="value.container_end"
                    :min="1"
                    :max="65535"
                    :show-button="false"
                    :placeholder="$gettext('Container End')"
                    class="w-22.5"
                  />
                  <n-select
                    v-model:value="value.protocol"
                    :options="protocolOptions"
                    class="w-22.5"
                  />
                </n-flex>
              </template>
            </n-dynamic-input>
          </n-form-item>

          <n-alert v-if="createModel.publish_all_ports" type="info">
            {{
              $gettext(
                'All exposed ports in the image will be automatically mapped to random host ports.',
              )
            }}
          </n-alert>
        </n-form>
      </n-tab-pane>

      <!-- 存储挂载 -->
      <n-tab-pane name="volumes" :tab="$gettext('Volumes')">
        <n-form :model="createModel" label-placement="left" label-width="120">
          <n-form-item :label="$gettext('Volume Mounts')" :show-label="false">
            <n-dynamic-input
              v-model:value="createModel.volumes"
              :on-create="onCreateVolume"
              show-sort-button
            >
              <template #default="{ value }">
                <n-flex align="center" :wrap="false" style="width: 100%">
                  <n-input
                    v-model:value="value.host"
                    :placeholder="$gettext('Host path')"
                    style="flex: 1"
                  />
                  <span>:</span>
                  <n-input
                    v-model:value="value.container"
                    :placeholder="$gettext('Container path')"
                    style="flex: 1"
                  />
                  <n-select v-model:value="value.mode" :options="volumeModeOptions" class="w-30" />
                </n-flex>
              </template>
            </n-dynamic-input>
          </n-form-item>

          <n-alert type="info" class="mt-4">
            {{
              $gettext(
                'Mount host directories or volumes into the container. Use absolute paths for host directories.',
              )
            }}
          </n-alert>
        </n-form>
      </n-tab-pane>

      <!-- 资源限制 -->
      <n-tab-pane name="resources" :tab="$gettext('Resource Limits')">
        <n-form :model="createModel" label-placement="left" label-width="120">
          <n-alert type="info" class="mb-4">
            {{
              $gettext(
                'Set resource limits to prevent the container from consuming too many system resources. Set to 0 for no limit.',
              )
            }}
          </n-alert>

          <n-row :gutter="[24, 0]">
            <n-col :span="8">
              <n-form-item path="memory" :label="$gettext('Memory (MB)')">
                <n-input-number
                  v-model:value="createModel.memory"
                  :min="0"
                  style="width: 100%"
                  :placeholder="$gettext('0 = no limit')"
                />
              </n-form-item>
            </n-col>
            <n-col :span="8">
              <n-form-item path="cpus" :label="$gettext('CPU Cores')">
                <n-input-number
                  v-model:value="createModel.cpus"
                  :min="0"
                  :precision="2"
                  :step="0.5"
                  style="width: 100%"
                  :placeholder="$gettext('0 = no limit')"
                />
              </n-form-item>
            </n-col>
            <n-col :span="8">
              <n-form-item path="cpu_shares" :label="$gettext('CPU Shares')">
                <n-input-number
                  v-model:value="createModel.cpu_shares"
                  :min="0"
                  :max="262144"
                  style="width: 100%"
                />
              </n-form-item>
            </n-col>
          </n-row>

          <n-collapse class="mt-4">
            <n-collapse-item :title="$gettext('Resource Limit Description')">
              <n-descriptions :column="1" label-placement="left">
                <n-descriptions-item :label="$gettext('Memory')">
                  {{ $gettext('Maximum memory the container can use, in MB. 0 means no limit.') }}
                </n-descriptions-item>
                <n-descriptions-item :label="$gettext('CPU Cores')">
                  {{
                    $gettext(
                      'Number of CPU cores the container can use. 0.5 means half a core, 2 means 2 cores.',
                    )
                  }}
                </n-descriptions-item>
                <n-descriptions-item :label="$gettext('CPU Shares')">
                  {{
                    $gettext(
                      'Relative CPU weight. Default is 1024. Higher values get more CPU time when competing.',
                    )
                  }}
                </n-descriptions-item>
              </n-descriptions>
            </n-collapse-item>
          </n-collapse>
        </n-form>
      </n-tab-pane>

      <!-- 环境与命令 -->
      <n-tab-pane name="environment" :tab="$gettext('Environment')">
        <n-form :model="createModel" label-placement="left" label-width="140">
          <n-form-item :label="$gettext('Environment Variables')">
            <n-dynamic-input
              v-model:value="createModel.env"
              :on-create="onCreateEnv"
              show-sort-button
            >
              <template #default="{ value }">
                <n-flex align="center" :wrap="false" style="width: 100%">
                  <n-input
                    v-model:value="value.key"
                    :placeholder="$gettext('Variable name')"
                    style="flex: 1"
                  />
                  <span>=</span>
                  <n-input
                    v-model:value="value.value"
                    :placeholder="$gettext('Variable value')"
                    style="flex: 2"
                  />
                </n-flex>
              </template>
            </n-dynamic-input>
          </n-form-item>

          <n-divider title-placement="left">{{ $gettext('Startup Commands') }}</n-divider>

          <n-form-item path="command" :label="$gettext('Command')">
            <n-dynamic-input
              v-model:value="createModel.command"
              :placeholder="$gettext('Command argument')"
            />
            <template #feedback>
              <span class="text-gray-400">
                {{ $gettext('Override the default CMD of the image') }}
              </span>
            </template>
          </n-form-item>

          <n-form-item path="entrypoint" :label="$gettext('Entrypoint')">
            <n-dynamic-input
              v-model:value="createModel.entrypoint"
              :placeholder="$gettext('Entrypoint argument')"
            />
            <template #feedback>
              <span class="text-gray-400">
                {{ $gettext('Override the default ENTRYPOINT of the image') }}
              </span>
            </template>
          </n-form-item>

          <n-divider title-placement="left">{{ $gettext('Labels') }}</n-divider>

          <n-form-item :label="$gettext('Container Labels')">
            <n-dynamic-input
              v-model:value="createModel.labels"
              :on-create="onCreateLabel"
              show-sort-button
            >
              <template #default="{ value }">
                <n-flex align="center" :wrap="false" style="width: 100%">
                  <n-input
                    v-model:value="value.key"
                    :placeholder="$gettext('Label name')"
                    style="flex: 1"
                  />
                  <span>=</span>
                  <n-input
                    v-model:value="value.value"
                    :placeholder="$gettext('Label value')"
                    style="flex: 2"
                  />
                </n-flex>
              </template>
            </n-dynamic-input>
          </n-form-item>
        </n-form>
      </n-tab-pane>
    </n-tabs>

    <template #footer>
      <n-flex justify="end">
        <n-button @click="show = false" :disabled="doSubmit">
          {{ $gettext('Cancel') }}
        </n-button>
        <n-button type="primary" :loading="doSubmit" :disabled="doSubmit" @click="handleSubmit">
          {{ isEdit ? $gettext('Save') : $gettext('Create') }}
        </n-button>
      </n-flex>
    </template>
  </n-modal>

  <!-- 镜像拉取弹窗 -->
  <image-pull-modal
    v-model:show="showPullModal"
    :image="createModel.image"
    @success="onPullSuccess"
  />
</template>

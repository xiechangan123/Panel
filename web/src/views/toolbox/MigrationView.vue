<script setup lang="ts">
defineOptions({
  name: 'toolbox-migration',
})

import { useRequest } from 'alova/client'
import { useGettext } from 'vue3-gettext'

import home from '@/api/panel/home'
import migration, {
  type MigrationConnection,
  type MigrationPanel,
  type MigrationResource,
  type MigrationResourceType,
  type MigrationResult,
} from '@/api/panel/toolbox-migration'
import ws from '@/api/ws'
import TheIcon from '@/components/custom/TheIcon.vue'

const { $gettext } = useGettext()

const currentStep = ref(1)
const loading = ref(false)
const connectionForm = ref<MigrationConnection>({
  source_panel: 'acepanel',
  url: '',
  token_id: 1,
  token: '',
  api_key: '',
})

const sourceInfo = ref<Record<string, any> | null>(null)
const targetInfo = ref<Record<string, any> | null>(null)
const resources = ref<MigrationResource[]>([])
const selectedKeys = ref<string[]>([])
const targetPaths = ref<Record<string, string>>({})
const targetUsers = ref<Record<string, string>>({})
const activeGroup = ref('website')
const skipIncompatible = ref(false)
const stopSourceDuringBackup = ref(false)

const migrationLogs = ref<string[]>([])
const migrationResults = ref<MigrationResult[]>([])
const migrationRunning = ref(false)
const migrationStartedAt = ref<string | null>(null)
const migrationEndedAt = ref<string | null>(null)
const logContainer = ref<HTMLElement | null>(null)

let progressWs: WebSocket | null = null
let pollTimer: ReturnType<typeof setInterval> | null = null
let reconnectTimer: ReturnType<typeof setTimeout> | null = null

const panelCards = computed(() => [
  {
    value: 'acepanel' as MigrationPanel,
    icon: 'solar:server-square-cloud-bold-duotone',
    title: $gettext('AcePanel → AcePanel'),
    description: $gettext('Move resources from this AcePanel to another AcePanel server.'),
    badge: $gettext('Existing capability'),
  },
  {
    value: 'baota' as MigrationPanel,
    icon: 'mdi:shield-crown-outline',
    title: $gettext('BaoTa → AcePanel'),
    description: $gettext(
      'Pull websites, databases, projects, containers, and Compose from BaoTa 11.x.',
    ),
    badge: $gettext('New'),
  },
  {
    value: 'onepanel' as MigrationPanel,
    icon: 'mdi:view-dashboard-variant-outline',
    title: $gettext('1Panel → AcePanel'),
    description: $gettext(
      'Pull websites, databases, runtimes, containers, and Compose from 1Panel v2.',
    ),
    badge: $gettext('New'),
  },
])

const resourceGroups = computed(() => [
  { key: 'website', label: $gettext('Websites'), icon: 'mdi:web', types: ['website'] },
  {
    key: 'database',
    label: $gettext('Databases'),
    icon: 'mdi:database-outline',
    types: ['database', 'database_user'],
  },
  { key: 'project', label: $gettext('Projects'), icon: 'mdi:code-braces', types: ['project'] },
  {
    key: 'container',
    label: $gettext('Containers'),
    icon: 'mdi:docker',
    types: ['container'],
  },
  {
    key: 'compose',
    label: $gettext('Compose'),
    icon: 'mdi:layers-triple-outline',
    types: ['compose'],
  },
])

const visibleResources = computed(() => {
  const group = resourceGroups.value.find((item) => item.key === activeGroup.value)
  return resources.value.filter((item) => group?.types.includes(item.type))
})

const selectedResources = computed(() =>
  resources.value.filter((item) => selectedKeys.value.includes(item.key)),
)

const selectedBlockedCount = computed(
  () => selectedResources.value.filter((item) => isResourceBlocked(item)).length,
)

const selectableVisibleResources = computed(() =>
  visibleResources.value.filter((item) => item.supported && item.blockers.length === 0),
)

const allVisibleSelected = computed(
  () =>
    selectableVisibleResources.value.length > 0 &&
    selectableVisibleResources.value.every((item) => selectedKeys.value.includes(item.key)),
)

const sourceCapabilities = computed<string[]>(() => sourceInfo.value?.capabilities || [])

const targetEnvironments = computed(() => {
  const result: { name: string; versions: string[] }[] = []
  const environments = targetInfo.value?.environments
  if (environments && typeof environments === 'object') {
    for (const [name, versions] of Object.entries(environments)) {
      result.push({ name, versions: (versions as string[]) || [] })
    }
    return result
  }
  for (const name of ['go', 'java', 'nodejs', 'php', 'python', 'dotnet']) {
    const versions = (targetInfo.value?.[name] || []).map((item: any) => item.label || item.value)
    result.push({ name, versions })
  }
  return result
})

const resultStats = computed(() => {
  const stats = { success: 0, partial: 0, failed: 0, skipped: 0 }
  for (const result of migrationResults.value) {
    if (result.status in stats) stats[result.status as keyof typeof stats]++
  }
  return stats
})

const overallProgress = computed(() => {
  if (migrationResults.value.length === 0) return 0
  const total = migrationResults.value.reduce((sum, item) => sum + resultProgress(item), 0)
  return Math.round(total / migrationResults.value.length)
})

const checkStatus = () => {
  useRequest(migration.status()).onSuccess(({ data }: any) => {
    if (data.step === 'running') {
      currentStep.value = 4
      migrationRunning.value = true
      migrationResults.value = data.results || []
      migrationStartedAt.value = data.started_at
      connectProgressWs()
    } else if (data.step === 'done') {
      currentStep.value = 5
      migrationResults.value = data.results || []
      migrationStartedAt.value = data.started_at
      migrationEndedAt.value = data.ended_at
    }
  })
}

onMounted(checkStatus)

onUnmounted(() => {
  closeProgressChannels()
})

watch(
  () => connectionForm.value.source_panel,
  () => {
    connectionForm.value.url = ''
    connectionForm.value.token = ''
    connectionForm.value.api_key = ''
  },
)

const selectPanel = (panel: MigrationPanel) => {
  connectionForm.value.source_panel = panel
}

const validateConnection = () => {
  if (!connectionForm.value.url) {
    window.$message.error($gettext('Please enter the panel address.'))
    return false
  }
  if (
    connectionForm.value.source_panel === 'acepanel' &&
    (!connectionForm.value.token_id || !connectionForm.value.token)
  ) {
    window.$message.error($gettext('Please enter the AcePanel Token ID and access token.'))
    return false
  }
  if (connectionForm.value.source_panel !== 'acepanel' && !connectionForm.value.api_key) {
    window.$message.error($gettext('Please enter the source panel API key.'))
    return false
  }
  return true
}

const handlePreCheck = () => {
  if (!validateConnection()) return
  loading.value = true
  useRequest(migration.precheck(connectionForm.value))
    .onSuccess(({ data }: any) => {
      sourceInfo.value = data.source || { panel: connectionForm.value.source_panel }
      targetInfo.value = data.target || data.remote || {}
      currentStep.value = 2

      if (connectionForm.value.source_panel === 'acepanel') {
        useRequest(home.installedEnvironment()).onSuccess(({ data: local }: any) => {
          sourceInfo.value = { ...local, panel: 'acepanel' }
        })
      }
    })
    .onComplete(() => {
      loading.value = false
    })
}

const handleRefreshPreCheck = () => {
  handlePreCheck()
}

const handleGetItems = () => {
  loading.value = true
  useRequest(migration.items())
    .onSuccess(({ data }: any) => {
      resources.value = (data.items || []).map((item: MigrationResource) => ({
        ...item,
        blockers: item.blockers || [],
        warnings: item.warnings || [],
        features: item.features || [],
        depends_on: item.depends_on || [],
      }))
      targetPaths.value = Object.fromEntries(
        resources.value.map((item) => [item.key, item.target_path || '']),
      )
      targetUsers.value = Object.fromEntries(resources.value.map((item) => [item.key, '']))
      selectedKeys.value = []
      const firstGroup = resourceGroups.value.find((group) => groupCount(group.types) > 0)
      activeGroup.value = firstGroup?.key || 'website'
      currentStep.value = 3
    })
    .onComplete(() => {
      loading.value = false
    })
}

const groupCount = (types: string[]) =>
  resources.value.filter((item) => types.includes(item.type)).length

const groupSelectedCount = (types: string[]) =>
  resources.value.filter(
    (item) => types.includes(item.type) && selectedKeys.value.includes(item.key),
  ).length

const missingDependencies = (item: MigrationResource) =>
  item.depends_on.filter((key) => !selectedKeys.value.includes(key))

const isResourceBlocked = (item: MigrationResource) =>
  !item.supported || item.blockers.length > 0 || missingDependencies(item).length > 0

const resourceDisabled = (item: MigrationResource) =>
  !item.supported || (item.blockers.length > 0 && !skipIncompatible.value)

const toggleResource = (item: MigrationResource, checked: boolean) => {
  const selected = new Set(selectedKeys.value)
  if (checked) {
    selected.add(item.key)
    for (const dependency of item.depends_on) {
      const dependencyItem = resources.value.find((resource) => resource.key === dependency)
      if (dependencyItem?.supported) selected.add(dependency)
    }
  } else {
    selected.delete(item.key)
    for (const resource of resources.value) {
      if (resource.depends_on.includes(item.key)) selected.delete(resource.key)
    }
  }
  selectedKeys.value = [...selected]
}

const toggleVisibleResources = () => {
  const selected = new Set(selectedKeys.value)
  if (allVisibleSelected.value) {
    for (const item of selectableVisibleResources.value) selected.delete(item.key)
  } else {
    for (const item of selectableVisibleResources.value) {
      selected.add(item.key)
      for (const dependency of item.depends_on) selected.add(dependency)
    }
  }
  selectedKeys.value = [...selected]
}

const handleStartMigration = () => {
  if (selectedKeys.value.length === 0) {
    window.$message.warning($gettext('Please select at least one resource to migrate.'))
    return
  }
  if (selectedBlockedCount.value > 0 && !skipIncompatible.value) {
    window.$message.error(
      $gettext('The selection contains blocked resources. Resolve them or enable skipping first.'),
    )
    return
  }

  const selections = selectedResources.value.map((item) => ({
    key: item.key,
    target_path: item.type === 'project' ? targetPaths.value[item.key] : undefined,
    target_user: item.type === 'project' ? targetUsers.value[item.key] : undefined,
  }))
  const stopNotice = stopSourceDuringBackup.value
    ? $gettext(
        'Selected running services will be stopped while source backups are generated and restored before downloading.',
      )
    : $gettext('Source services will remain running while backups are generated.')

  window.$dialog.warning({
    title: $gettext('Start migration'),
    content: `${$gettext(
      'AcePanel will create new target resources and will not overwrite, merge, rename, or automatically roll back existing resources.',
    )}\n\n${stopNotice}`,
    positiveText: $gettext('Start migration'),
    negativeText: $gettext('Cancel'),
    onPositiveClick: () => {
      loading.value = true
      migrationLogs.value = []
      migrationResults.value = []
      useRequest(
        migration.start({
          items: selections,
          skip_incompatible_items: skipIncompatible.value,
          stop_source_during_backup: stopSourceDuringBackup.value,
        }),
      )
        .onSuccess(() => {
          currentStep.value = 4
          migrationRunning.value = true
          connectProgressWs()
        })
        .onComplete(() => {
          loading.value = false
        })
    },
  })
}

const connectProgressWs = async () => {
  try {
    progressWs = await ws.migrationProgress()
    progressWs.onmessage = (event: MessageEvent) => {
      const data = JSON.parse(event.data)
      migrationResults.value = data.results || []
      migrationStartedAt.value = data.started_at
      migrationEndedAt.value = data.ended_at
      if (data.new_logs?.length) {
        migrationLogs.value.push(...data.new_logs)
        if (migrationLogs.value.length > 1500)
          migrationLogs.value = migrationLogs.value.slice(-1500)
        nextTick(scrollLogsToBottom)
      }
      if (data.step === 'done') {
        migrationRunning.value = false
        currentStep.value = 5
        closeProgressChannels()
      }
    }
    progressWs.onclose = () => {
      progressWs = null
      if (migrationRunning.value) reconnectTimer = setTimeout(connectProgressWs, 3000)
    }
  } catch {
    pollProgress()
  }
}

const pollProgress = () => {
  if (pollTimer) return
  pollTimer = setInterval(() => {
    useRequest(migration.results()).onSuccess(({ data }: any) => {
      migrationResults.value = data.results || []
      migrationStartedAt.value = data.started_at
      migrationEndedAt.value = data.ended_at
      migrationLogs.value = data.logs || migrationLogs.value
      nextTick(scrollLogsToBottom)
      if (data.step === 'done') {
        migrationRunning.value = false
        currentStep.value = 5
        closeProgressChannels()
      }
    })
  }, 2000)
}

const closeProgressChannels = () => {
  if (progressWs) {
    progressWs.onclose = null
    progressWs.close()
    progressWs = null
  }
  if (pollTimer) {
    clearInterval(pollTimer)
    pollTimer = null
  }
  if (reconnectTimer) {
    clearTimeout(reconnectTimer)
    reconnectTimer = null
  }
}

const scrollLogsToBottom = () => {
  if (logContainer.value) logContainer.value.scrollTop = logContainer.value.scrollHeight
}

const handleReset = () => {
  window.$dialog.warning({
    title: $gettext('Start a new migration'),
    content: $gettext('The current in-memory migration results and logs will be cleared.'),
    positiveText: $gettext('Reset'),
    negativeText: $gettext('Cancel'),
    onPositiveClick: () => {
      useRequest(migration.reset()).onSuccess(() => {
        closeProgressChannels()
        currentStep.value = 1
        connectionForm.value = {
          source_panel: 'acepanel',
          url: '',
          token_id: 1,
          token: '',
          api_key: '',
        }
        sourceInfo.value = null
        targetInfo.value = null
        resources.value = []
        selectedKeys.value = []
        targetPaths.value = {}
        targetUsers.value = {}
        skipIncompatible.value = false
        stopSourceDuringBackup.value = false
        migrationLogs.value = []
        migrationResults.value = []
        migrationStartedAt.value = null
        migrationEndedAt.value = null
      })
    },
  })
}

const sourcePanelName = (panel?: string) => {
  if (panel === 'baota') return $gettext('BaoTa')
  if (panel === 'onepanel') return $gettext('1Panel')
  return $gettext('AcePanel')
}

const sourcePanelIcon = (panel?: string) => {
  if (panel === 'baota') return 'mdi:shield-crown-outline'
  if (panel === 'onepanel') return 'mdi:view-dashboard-variant-outline'
  return 'solar:server-square-cloud-bold-duotone'
}

const connectionTitle = computed(() =>
  connectionForm.value.source_panel === 'acepanel'
    ? $gettext('Target AcePanel connection')
    : $gettext('Source panel connection'),
)

const connectionAddressLabel = computed(() =>
  connectionForm.value.source_panel === 'acepanel'
    ? $gettext('Target AcePanel address')
    : $gettext('Source panel address'),
)

const connectionPlaceholder = computed(() => {
  if (connectionForm.value.source_panel === 'baota') return 'https://source-server:8888'
  if (connectionForm.value.source_panel === 'onepanel') return 'https://source-server:9443'
  return 'https://target-server:8443'
})

const getResourceIcon = (type: MigrationResourceType | string) => {
  if (type === 'website') return 'mdi:web'
  if (type === 'database' || type === 'database_user') return 'mdi:database-outline'
  if (type === 'project') return 'mdi:code-braces'
  if (type === 'container') return 'mdi:docker'
  return 'mdi:layers-triple-outline'
}

const getTypeLabel = (type: MigrationResourceType | string) => {
  if (type === 'website') return $gettext('Website')
  if (type === 'database') return $gettext('Database')
  if (type === 'database_user') return $gettext('Database user')
  if (type === 'project') return $gettext('Project')
  if (type === 'container') return $gettext('Container')
  if (type === 'compose') return $gettext('Compose')
  return type
}

const getFeatureLabel = (feature: string) => {
  const labels: Record<string, string> = {
    files: $gettext('Files'),
    domains: $gettext('Domains'),
    php: $gettext('PHP'),
    rewrite: $gettext('Rewrite'),
    proxy: $gettext('Proxy'),
    redirect: $gettext('Redirect'),
    https: $gettext('HTTPS'),
    schema: $gettext('Schema'),
    data: $gettext('Data'),
    user: $gettext('User'),
    code: $gettext('Code'),
    command: $gettext('Command'),
    environment: $gettext('Environment'),
    image: $gettext('Image'),
    volumes: $gettext('Volumes'),
    networks: $gettext('Networks'),
    binds: $gettext('Bind data'),
    compose: $gettext('Compose YAML'),
    'writable-layer': $gettext('Writable layer'),
    'structured-config': $gettext('Structured config'),
    credentials: $gettext('Credentials'),
    privileges: $gettext('Privileges'),
  }
  return labels[feature] || feature
}

const getStatusLabel = (status: string) => {
  const labels: Record<string, string> = {
    pending: $gettext('Pending'),
    running: $gettext('Running'),
    success: $gettext('Success'),
    partial: $gettext('Partial'),
    failed: $gettext('Failed'),
    skipped: $gettext('Skipped'),
    stopped: $gettext('Stopped'),
    paused: $gettext('Paused'),
  }
  return labels[status] || status
}

const getStageLabel = (stage: string) => {
  const labels: Record<string, string> = {
    preparing: $gettext('Preparing source backup'),
    downloading: $gettext('Downloading'),
    validating: $gettext('Validating'),
    importing: $gettext('Importing data'),
    configuring: $gettext('Rebuilding configuration'),
    starting: $gettext('Starting'),
    done: $gettext('Done'),
  }
  return labels[stage] || $gettext('Waiting')
}

const getStatusType = (status: string) => {
  if (status === 'success') return 'success'
  if (status === 'partial' || status === 'running') return 'warning'
  if (status === 'failed') return 'error'
  if (status === 'skipped') return 'default'
  return 'info'
}

const resultProgress = (result: MigrationResult) => {
  if (['success', 'partial', 'failed', 'skipped'].includes(result.status)) return 100
  const stages: Record<string, number> = {
    preparing: 12,
    downloading: 32,
    validating: 48,
    importing: 65,
    configuring: 82,
    starting: 94,
    done: 100,
  }
  return stages[result.stage] || 4
}

const formatSize = (bytes: number) => {
  if (!bytes) return $gettext('Size unknown')
  const units = ['B', 'KB', 'MB', 'GB', 'TB']
  let value = bytes
  let unit = 0
  while (value >= 1024 && unit < units.length - 1) {
    value /= 1024
    unit++
  }
  return `${value.toFixed(unit > 1 ? 1 : 0)} ${units[unit]}`
}

const formatDuration = (seconds: number) => {
  if (!seconds) return '—'
  if (seconds < 60) return `${seconds.toFixed(1)}s`
  return `${Math.floor(seconds / 60)}m ${(seconds % 60).toFixed(0)}s`
}

const formatDate = (value: string | null) => (value ? new Date(value).toLocaleString() : '—')
</script>

<template>
  <div class="migration-page">
    <section class="migration-hero">
      <div>
        <div class="eyebrow">{{ $gettext('Toolbox · Server migration') }}</div>
        <h1>{{ $gettext('Move workloads into AcePanel') }}</h1>
        <p>
          {{
            $gettext(
              'Discover source resources through panel APIs, transfer backup artifacts, and rebuild supported settings with AcePanel.',
            )
          }}
        </p>
      </div>
      <div class="hero-principle">
        <TheIcon icon="mdi:file-tree-outline" :size="24" />
        <div>
          <strong>{{ $gettext('Structured migration') }}</strong>
          <span>{{
            $gettext('Never copy a complete Nginx or OpenResty virtual host configuration.')
          }}</span>
        </div>
      </div>
    </section>

    <n-card class="step-card" :bordered="false">
      <n-steps :current="currentStep" size="small" responsive>
        <n-step :title="$gettext('Direction')" :description="$gettext('Choose source')" />
        <n-step :title="$gettext('Pre-check')" :description="$gettext('Review capability')" />
        <n-step :title="$gettext('Resources')" :description="$gettext('Select workload')" />
        <n-step :title="$gettext('Migration')" :description="$gettext('Track stages')" />
        <n-step :title="$gettext('Results')" :description="$gettext('Handle follow-up')" />
      </n-steps>
    </n-card>

    <template v-if="currentStep === 1">
      <section class="section-heading">
        <div>
          <h2>{{ $gettext('Choose a migration direction') }}</h2>
          <p>
            {{
              $gettext(
                'The destination is AcePanel. External panels are always pulled into this server.',
              )
            }}
          </p>
        </div>
      </section>

      <div class="direction-grid">
        <button
          v-for="panel in panelCards"
          :key="panel.value"
          type="button"
          class="direction-card"
          :class="{ active: connectionForm.source_panel === panel.value }"
          @click="selectPanel(panel.value)"
        >
          <div class="direction-card__top">
            <span class="direction-icon"><TheIcon :icon="panel.icon" :size="29" /></span>
            <n-tag :type="panel.value === 'acepanel' ? 'default' : 'success'" size="small" round>
              {{ panel.badge }}
            </n-tag>
          </div>
          <strong>{{ panel.title }}</strong>
          <p>{{ panel.description }}</p>
          <span class="direction-check">
            <TheIcon
              :icon="
                connectionForm.source_panel === panel.value
                  ? 'mdi:check-circle'
                  : 'mdi:circle-outline'
              "
              :size="20"
            />
          </span>
        </button>
      </div>

      <n-card class="content-card connection-card" :bordered="false">
        <div class="connection-layout">
          <aside class="connection-aside">
            <div class="aside-icon">
              <TheIcon :icon="sourcePanelIcon(connectionForm.source_panel)" :size="34" />
            </div>
            <h3>{{ connectionTitle }}</h3>
            <p v-if="connectionForm.source_panel === 'acepanel'">
              {{
                $gettext(
                  'Create an API token on the destination AcePanel. This server remains the migration source.',
                )
              }}
            </p>
            <p v-else-if="connectionForm.source_panel === 'baota'">
              {{
                $gettext(
                  'Enable the BaoTa API and use its API key. The allowed IP list must include this AcePanel server.',
                )
              }}
            </p>
            <p v-else>
              {{
                $gettext('Create a 1Panel v2 API key and allow requests from this AcePanel server.')
              }}
            </p>
            <div class="destination-chip">
              <TheIcon icon="mdi:arrow-right" :size="18" />
              <span>
                {{
                  connectionForm.source_panel === 'acepanel'
                    ? $gettext('Destination: remote AcePanel')
                    : $gettext('Destination: this AcePanel')
                }}
              </span>
            </div>
          </aside>

          <n-form class="connection-form" label-placement="top">
            <n-form-item :label="connectionAddressLabel">
              <n-input
                v-model:value="connectionForm.url"
                size="large"
                :placeholder="connectionPlaceholder"
              >
                <template #prefix><TheIcon icon="mdi:link-variant" :size="18" /></template>
              </n-input>
            </n-form-item>
            <div v-if="connectionForm.source_panel === 'acepanel'" class="form-grid">
              <n-form-item :label="$gettext('Token ID')">
                <n-input-number v-model:value="connectionForm.token_id" :min="1" size="large" />
              </n-form-item>
              <n-form-item :label="$gettext('Access token')">
                <n-input
                  v-model:value="connectionForm.token"
                  type="password"
                  show-password-on="click"
                  size="large"
                  :placeholder="$gettext('AcePanel API access token')"
                />
              </n-form-item>
            </div>
            <n-form-item v-else :label="$gettext('API key')">
              <n-input
                v-model:value="connectionForm.api_key"
                type="password"
                show-password-on="click"
                size="large"
                :placeholder="
                  connectionForm.source_panel === 'baota'
                    ? $gettext('BaoTa API key')
                    : $gettext('1Panel API key')
                "
              />
            </n-form-item>
            <div class="form-footer">
              <span>
                <TheIcon icon="mdi:shield-lock-outline" :size="17" />
                {{ $gettext('Credentials stay only in the current in-memory migration state.') }}
              </span>
              <n-button type="primary" size="large" :loading="loading" @click="handlePreCheck">
                {{ $gettext('Connect and pre-check') }}
                <template #icon><TheIcon icon="mdi:arrow-right" :size="18" /></template>
              </n-button>
            </div>
          </n-form>
        </div>
      </n-card>
    </template>

    <template v-else-if="currentStep === 2">
      <section class="section-heading">
        <div>
          <h2>{{ $gettext('Connection and capability pre-check') }}</h2>
          <p>
            {{
              $gettext(
                'No environment is installed and no target setting is changed during this check.',
              )
            }}
          </p>
        </div>
        <n-button :loading="loading" @click="handleRefreshPreCheck">
          <template #icon><TheIcon icon="mdi:refresh" :size="18" /></template>
          {{ $gettext('Refresh') }}
        </n-button>
      </section>

      <div class="server-flow">
        <div class="server-card">
          <span class="server-card__icon">
            <TheIcon :icon="sourcePanelIcon(sourceInfo?.panel)" :size="32" />
          </span>
          <div>
            <small>{{ $gettext('Source') }}</small>
            <strong>{{ sourcePanelName(sourceInfo?.panel) }}</strong>
            <span>{{ sourceInfo?.version || $gettext('Version detected by API') }}</span>
          </div>
          <n-tag type="success" size="small" round>{{ $gettext('Connected') }}</n-tag>
        </div>
        <div class="flow-arrow"><TheIcon icon="mdi:arrow-right" :size="26" /></div>
        <div class="server-card target">
          <span class="server-card__icon"><TheIcon icon="mdi:server-network" :size="32" /></span>
          <div>
            <small>{{ $gettext('Destination') }}</small>
            <strong>{{ $gettext('AcePanel') }}</strong>
            <span>{{ targetInfo?.architecture || $gettext('Current server') }}</span>
          </div>
          <n-tag type="info" size="small" round>{{ $gettext('Ready for review') }}</n-tag>
        </div>
      </div>

      <div class="precheck-grid">
        <n-card class="content-card" :bordered="false">
          <template #header>
            <div class="card-title">
              <TheIcon icon="mdi:api" :size="21" />
              <span>{{ $gettext('Source API capability') }}</span>
            </div>
          </template>
          <div v-if="sourceCapabilities.length" class="chip-list">
            <n-tag v-for="capability in sourceCapabilities" :key="capability" round>
              {{ getFeatureLabel(capability) }}
            </n-tag>
          </div>
          <n-empty
            v-else
            size="small"
            :description="$gettext('AcePanel uses its existing migration API.')"
          />
          <n-descriptions :column="1" size="small" class="detail-list">
            <n-descriptions-item :label="$gettext('Hostname')">
              {{ sourceInfo?.hostname || '—' }}
            </n-descriptions-item>
            <n-descriptions-item :label="$gettext('Architecture')">
              {{ sourceInfo?.architecture || '—' }}
            </n-descriptions-item>
          </n-descriptions>
        </n-card>

        <n-card class="content-card" :bordered="false">
          <template #header>
            <div class="card-title">
              <TheIcon icon="mdi:cube-scan" :size="21" />
              <span>{{ $gettext('Destination capability') }}</span>
            </div>
          </template>
          <div class="capability-row">
            <span>{{ $gettext('Web server') }}</span>
            <n-tag :type="targetInfo?.webserver ? 'success' : 'default'" size="small">
              {{ targetInfo?.webserver || $gettext('Not detected') }}
            </n-tag>
          </div>
          <div class="capability-row">
            <span>{{ $gettext('Docker') }}</span>
            <n-tag :type="targetInfo?.docker === false ? 'error' : 'success'" size="small">
              {{ targetInfo?.docker === false ? $gettext('Unavailable') : $gettext('Available') }}
            </n-tag>
          </div>
          <div class="capability-row environments">
            <span>{{ $gettext('Runtimes') }}</span>
            <div class="chip-list compact">
              <n-tag
                v-for="environment in targetEnvironments.filter((item) => item.versions.length)"
                :key="environment.name"
                size="small"
                round
              >
                {{ environment.name.toUpperCase() }} · {{ environment.versions.length }}
              </n-tag>
              <n-text v-if="!targetEnvironments.some((item) => item.versions.length)" depth="3">
                {{ $gettext('No runtime detected') }}
              </n-text>
            </div>
          </div>
        </n-card>
      </div>

      <n-alert type="info" :show-icon="true" class="notice-alert">
        {{
          $gettext(
            'Detailed compatibility, name conflicts, missing runtimes, and special mounts are evaluated after the resource catalog is loaded.',
          )
        }}
      </n-alert>

      <div class="action-bar">
        <n-button @click="currentStep = 1">{{ $gettext('Back') }}</n-button>
        <n-button type="primary" size="large" :loading="loading" @click="handleGetItems">
          {{ $gettext('Load resource catalog') }}
          <template #icon><TheIcon icon="mdi:arrow-right" :size="18" /></template>
        </n-button>
      </div>
    </template>

    <template v-else-if="currentStep === 3">
      <section class="section-heading resource-heading">
        <div>
          <h2>{{ $gettext('Select resources') }}</h2>
          <p>
            {{
              $gettext(
                'Only immutable resource keys are submitted. AcePanel reads every source configuration again before migration.',
              )
            }}
          </p>
        </div>
        <div class="selection-summary">
          <strong>{{ selectedKeys.length }}</strong>
          <span>{{ $gettext('selected') }}</span>
        </div>
      </section>

      <div class="resource-layout">
        <nav class="resource-nav">
          <button
            v-for="group in resourceGroups"
            :key="group.key"
            type="button"
            :class="{ active: activeGroup === group.key }"
            @click="activeGroup = group.key"
          >
            <span class="nav-icon"><TheIcon :icon="group.icon" :size="20" /></span>
            <span>{{ group.label }}</span>
            <n-badge
              :value="groupSelectedCount(group.types) || groupCount(group.types)"
              :type="groupSelectedCount(group.types) ? 'success' : 'default'"
            />
          </button>
        </nav>

        <n-card class="content-card resource-panel" :bordered="false">
          <div class="resource-toolbar">
            <div>
              <strong>{{ resourceGroups.find((item) => item.key === activeGroup)?.label }}</strong>
              <span>
                {{
                  $gettext('%{count} resources discovered', {
                    count: visibleResources.length,
                  })
                }}
              </span>
            </div>
            <n-button
              v-if="selectableVisibleResources.length"
              text
              type="primary"
              @click="toggleVisibleResources"
            >
              {{
                allVisibleSelected ? $gettext('Clear this group') : $gettext('Select compatible')
              }}
            </n-button>
          </div>

          <div v-if="visibleResources.length" class="resource-list">
            <article
              v-for="item in visibleResources"
              :key="item.key"
              class="resource-item"
              :class="{
                selected: selectedKeys.includes(item.key),
                blocked: isResourceBlocked(item),
              }"
            >
              <n-checkbox
                :checked="selectedKeys.includes(item.key)"
                :disabled="resourceDisabled(item)"
                @update:checked="(checked: boolean) => toggleResource(item, checked)"
              />
              <span class="resource-icon"
                ><TheIcon :icon="getResourceIcon(item.type)" :size="23"
              /></span>
              <div class="resource-main">
                <div class="resource-title-row">
                  <strong>{{ item.name }}</strong>
                  <n-tag size="small" round>{{ item.subtype || getTypeLabel(item.type) }}</n-tag>
                  <n-tag
                    :type="item.status === 'running' ? 'success' : 'default'"
                    size="small"
                    round
                  >
                    {{ getStatusLabel(item.status) }}
                  </n-tag>
                </div>
                <div class="resource-meta">
                  <span>{{ formatSize(item.size) }}</span>
                  <span>{{ $gettext('Target name') }} · {{ item.target_name }}</span>
                </div>
                <div v-if="item.features.length" class="chip-list compact features">
                  <n-tag
                    v-for="feature in item.features"
                    :key="feature"
                    size="tiny"
                    :bordered="false"
                  >
                    {{ getFeatureLabel(feature) }}
                  </n-tag>
                </div>
                <div v-if="item.type === 'project'" class="project-targets">
                  <n-input
                    v-model:value="targetPaths[item.key]"
                    size="small"
                    :placeholder="$gettext('Target project path')"
                    :disabled="!selectedKeys.includes(item.key)"
                  >
                    <template #prefix><TheIcon icon="mdi:folder-outline" :size="16" /></template>
                  </n-input>
                  <n-input
                    v-model:value="targetUsers[item.key]"
                    size="small"
                    :placeholder="$gettext('Target user (keep source mapping)')"
                    :disabled="!selectedKeys.includes(item.key)"
                  >
                    <template #prefix><TheIcon icon="mdi:account-outline" :size="16" /></template>
                  </n-input>
                </div>
                <div v-if="item.depends_on.length" class="dependency-line">
                  <TheIcon icon="mdi:source-branch" :size="16" />
                  <span>{{ $gettext('Dependencies') }}:</span>
                  <n-tag v-for="dependency in item.depends_on" :key="dependency" size="tiny">
                    {{
                      resources.find((resource) => resource.key === dependency)?.name || dependency
                    }}
                  </n-tag>
                </div>
                <div v-if="item.blockers.length" class="message-list error">
                  <div v-for="blocker in item.blockers" :key="blocker">
                    <TheIcon icon="mdi:close-circle-outline" :size="16" />
                    <span>{{ blocker }}</span>
                  </div>
                </div>
                <div v-if="missingDependencies(item).length" class="message-list error">
                  <div>
                    <TheIcon icon="mdi:link-variant-off" :size="16" />
                    <span>{{ $gettext('A required dependency is not selected.') }}</span>
                  </div>
                </div>
                <div v-if="item.warnings.length" class="message-list warning">
                  <div v-for="warning in item.warnings" :key="warning">
                    <TheIcon icon="mdi:alert-outline" :size="16" />
                    <span>{{ warning }}</span>
                  </div>
                </div>
              </div>
            </article>
          </div>
          <n-empty v-else :description="$gettext('No resources in this group')" />
        </n-card>
      </div>

      <div class="option-grid">
        <label class="option-card" :class="{ active: skipIncompatible }">
          <n-switch v-model:value="skipIncompatible" />
          <span class="option-icon"><TheIcon icon="mdi:debug-step-over" :size="24" /></span>
          <span>
            <strong>{{ $gettext('Skip incompatible resources') }}</strong>
            <small>{{
              $gettext('Blocked selections are recorded as skipped instead of stopping the task.')
            }}</small>
          </span>
        </label>
        <label class="option-card" :class="{ active: stopSourceDuringBackup }">
          <n-switch v-model:value="stopSourceDuringBackup" />
          <span class="option-icon"><TheIcon icon="mdi:pause-circle-outline" :size="24" /></span>
          <span>
            <strong>{{ $gettext('Pause source services while backing up') }}</strong>
            <small>{{
              $gettext('Off by default. Services are restored before artifact downloads begin.')
            }}</small>
          </span>
        </label>
      </div>

      <div class="action-bar sticky-action">
        <n-button @click="currentStep = 2">{{ $gettext('Back') }}</n-button>
        <div>
          <span v-if="selectedBlockedCount" class="blocked-summary">
            {{
              $gettext('%{count} blocked selections', {
                count: selectedBlockedCount,
              })
            }}
          </span>
          <n-button type="primary" size="large" :loading="loading" @click="handleStartMigration">
            {{ $gettext('Migrate %{count} resources', { count: selectedKeys.length }) }}
            <template #icon><TheIcon icon="mdi:rocket-launch-outline" :size="18" /></template>
          </n-button>
        </div>
      </div>
    </template>

    <template v-else-if="currentStep === 4">
      <section class="section-heading">
        <div>
          <h2>{{ $gettext('Migration in progress') }}</h2>
          <p>
            {{
              $gettext(
                'Each resource continues independently. One failure does not stop later resources.',
              )
            }}
          </p>
        </div>
        <div class="live-pill"><span></span>{{ $gettext('Live') }}</div>
      </section>

      <n-card class="content-card progress-overview" :bordered="false">
        <div>
          <small>{{ $gettext('Overall progress') }}</small>
          <strong>{{ overallProgress }}%</strong>
        </div>
        <n-progress
          type="line"
          :percentage="overallProgress"
          :show-indicator="false"
          :height="10"
          :border-radius="6"
        />
        <span>
          {{
            $gettext('%{done} of %{total} finished', {
              done: migrationResults.filter((item) =>
                ['success', 'partial', 'failed', 'skipped'].includes(item.status),
              ).length,
              total: migrationResults.length,
            })
          }}
        </span>
      </n-card>

      <div class="progress-layout">
        <n-card class="content-card progress-list-card" :bordered="false">
          <template #header>
            <div class="card-title">
              <TheIcon icon="mdi:format-list-checks" :size="21" />
              <span>{{ $gettext('Resource stages') }}</span>
            </div>
          </template>
          <div v-if="migrationResults.length" class="progress-list">
            <div
              v-for="result in migrationResults"
              :key="result.key || `${result.type}:${result.name}`"
              class="progress-row"
            >
              <span class="resource-icon"
                ><TheIcon :icon="getResourceIcon(result.type)" :size="21"
              /></span>
              <div class="progress-row__main">
                <div>
                  <strong>{{ result.name }}</strong>
                  <span>{{ getStageLabel(result.stage) }}</span>
                </div>
                <n-progress
                  type="line"
                  :percentage="resultProgress(result)"
                  :show-indicator="false"
                  :status="result.status === 'failed' ? 'error' : undefined"
                  :height="5"
                />
              </div>
              <n-tag :type="getStatusType(result.status)" size="small" round>
                {{ getStatusLabel(result.status) }}
              </n-tag>
            </div>
          </div>
          <n-empty v-else :description="$gettext('Preparing migration task')" />
        </n-card>

        <n-card class="content-card log-card" :bordered="false">
          <template #header>
            <div class="card-title">
              <TheIcon icon="mdi:console-line" :size="21" />
              <span>{{ $gettext('Live log') }}</span>
            </div>
          </template>
          <template #header-extra>
            <n-button text tag="a" :href="migration.logUrl" target="_blank">
              {{ $gettext('Download') }}
            </n-button>
          </template>
          <div ref="logContainer" class="log-console">
            <div v-for="(log, index) in migrationLogs" :key="index">{{ log }}</div>
            <div v-if="migrationRunning" class="log-cursor">
              <span></span>{{ $gettext('Waiting for the next update') }}
            </div>
          </div>
        </n-card>
      </div>
    </template>

    <template v-else>
      <section class="result-hero">
        <span class="result-hero__icon">
          <TheIcon
            :icon="resultStats.failed ? 'mdi:alert-circle-outline' : 'mdi:check-circle-outline'"
            :size="42"
          />
        </span>
        <div>
          <div class="eyebrow">{{ $gettext('Migration finished') }}</div>
          <h2>
            {{
              resultStats.failed || resultStats.partial
                ? $gettext('Migration completed with follow-up items')
                : $gettext('Migration completed successfully')
            }}
          </h2>
          <p>
            {{ $gettext('Started') }} {{ formatDate(migrationStartedAt) }} ·
            {{ $gettext('Finished') }} {{ formatDate(migrationEndedAt) }}
          </p>
        </div>
      </section>

      <div class="result-stats">
        <div class="success">
          <strong>{{ resultStats.success }}</strong
          ><span>{{ $gettext('Successful') }}</span>
        </div>
        <div class="partial">
          <strong>{{ resultStats.partial }}</strong
          ><span>{{ $gettext('Partial') }}</span>
        </div>
        <div class="failed">
          <strong>{{ resultStats.failed }}</strong
          ><span>{{ $gettext('Failed') }}</span>
        </div>
        <div class="skipped">
          <strong>{{ resultStats.skipped }}</strong
          ><span>{{ $gettext('Skipped') }}</span>
        </div>
      </div>

      <n-card class="content-card result-list-card" :bordered="false">
        <template #header>
          <div class="card-title">
            <TheIcon icon="mdi:clipboard-check-outline" :size="21" />
            <span>{{ $gettext('Resource results') }}</span>
          </div>
        </template>
        <div class="result-list">
          <article
            v-for="result in migrationResults"
            :key="result.key || `${result.type}:${result.name}`"
            class="result-item"
            :class="result.status"
          >
            <span class="resource-icon"
              ><TheIcon :icon="getResourceIcon(result.type)" :size="23"
            /></span>
            <div class="result-main">
              <div class="result-title">
                <strong>{{ result.name }}</strong>
                <n-tag size="small" round>{{ getTypeLabel(result.type) }}</n-tag>
                <span>{{ formatDuration(result.duration) }}</span>
              </div>
              <p v-if="result.error" class="result-error">{{ result.error }}</p>
              <div v-if="result.created_resources?.length" class="result-detail success-detail">
                <strong>{{ $gettext('Created') }}</strong>
                <span v-for="resource in result.created_resources" :key="resource">{{
                  resource
                }}</span>
              </div>
              <div v-if="result.warnings?.length" class="result-detail warning-detail">
                <strong>{{ $gettext('Manual follow-up') }}</strong>
                <span v-for="warning in result.warnings" :key="warning">{{ warning }}</span>
              </div>
              <div v-if="result.residual_resources?.length" class="result-detail residual-detail">
                <strong>{{ $gettext('Possible residual resources') }}</strong>
                <span v-for="resource in result.residual_resources" :key="resource">{{
                  resource
                }}</span>
              </div>
            </div>
            <n-tag :type="getStatusType(result.status)" size="small" round>
              {{ getStatusLabel(result.status) }}
            </n-tag>
          </article>
        </div>
      </n-card>

      <div class="action-bar result-actions">
        <n-button tag="a" :href="migration.logUrl" target="_blank">
          <template #icon><TheIcon icon="mdi:download-outline" :size="18" /></template>
          {{ $gettext('Download migration log') }}
        </n-button>
        <n-button type="primary" size="large" @click="handleReset">
          {{ $gettext('Start a new migration') }}
        </n-button>
      </div>
    </template>
  </div>
</template>

<style scoped lang="scss">
.migration-page {
  --migration-primary: #3b82f6;
  --migration-primary-soft: color-mix(in srgb, var(--migration-primary) 11%, transparent);
  --migration-border: color-mix(in srgb, var(--n-border-color) 82%, transparent);
  display: flex;
  flex-direction: column;
  gap: 18px;
  max-width: 1440px;
  margin: 0 auto;
  padding-bottom: 28px;
}

.migration-hero {
  display: flex;
  align-items: flex-end;
  justify-content: space-between;
  gap: 24px;
  padding: 8px 4px 2px;

  h1 {
    margin: 4px 0 8px;
    font-size: clamp(26px, 3vw, 38px);
    line-height: 1.18;
    letter-spacing: -0.035em;
  }

  p {
    max-width: 750px;
    margin: 0;
    color: var(--n-text-color-3);
    font-size: 15px;
  }
}

.eyebrow {
  color: var(--migration-primary);
  font-size: 12px;
  font-weight: 700;
  letter-spacing: 0.09em;
  text-transform: uppercase;
}

.hero-principle {
  display: flex;
  align-items: center;
  gap: 12px;
  max-width: 380px;
  padding: 14px 16px;
  color: var(--n-text-color-2);
  background: linear-gradient(135deg, var(--migration-primary-soft), transparent);
  border: 1px solid color-mix(in srgb, var(--migration-primary) 23%, var(--n-border-color));
  border-radius: 14px;

  > svg {
    flex: 0 0 auto;
    color: var(--migration-primary);
  }

  div {
    display: flex;
    flex-direction: column;
    gap: 2px;
  }

  span {
    color: var(--n-text-color-3);
    font-size: 12px;
    line-height: 1.4;
  }
}

.step-card,
.content-card {
  border: 1px solid var(--migration-border);
  box-shadow: 0 8px 24px color-mix(in srgb, #020617 5%, transparent);
}

.step-card {
  overflow-x: auto;
  border-radius: 16px;

  :deep(.n-card__content) {
    min-width: 680px;
    padding: 19px 26px;
  }
}

.section-heading {
  display: flex;
  align-items: flex-end;
  justify-content: space-between;
  gap: 16px;
  margin-top: 4px;

  h2 {
    margin: 0 0 5px;
    font-size: 22px;
    letter-spacing: -0.02em;
  }

  p {
    margin: 0;
    color: var(--n-text-color-3);
    font-size: 13px;
  }
}

.direction-grid {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 14px;
}

.direction-card {
  position: relative;
  min-height: 180px;
  padding: 20px;
  color: var(--n-text-color);
  text-align: left;
  cursor: pointer;
  background: var(--n-color);
  border: 1px solid var(--migration-border);
  border-radius: 16px;
  transition:
    transform 0.2s ease,
    border-color 0.2s ease,
    box-shadow 0.2s ease;

  &:hover {
    transform: translateY(-2px);
    border-color: color-mix(in srgb, var(--migration-primary) 45%, var(--n-border-color));
  }

  &.active {
    background: linear-gradient(145deg, var(--migration-primary-soft), var(--n-color) 68%);
    border-color: var(--migration-primary);
    box-shadow: 0 12px 30px color-mix(in srgb, var(--migration-primary) 15%, transparent);
  }

  strong {
    display: block;
    margin: 18px 0 7px;
    font-size: 17px;
  }

  p {
    max-width: 92%;
    margin: 0;
    color: var(--n-text-color-3);
    font-size: 13px;
    line-height: 1.55;
  }
}

.direction-card__top {
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.direction-icon,
.aside-icon,
.server-card__icon,
.option-icon {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  color: var(--migration-primary);
  background: var(--migration-primary-soft);
  border-radius: 12px;
}

.direction-icon {
  width: 48px;
  height: 48px;
}

.direction-check {
  position: absolute;
  right: 18px;
  bottom: 16px;
  color: var(--migration-primary);
}

.connection-card {
  border-radius: 18px;
}

.connection-layout {
  display: grid;
  grid-template-columns: minmax(240px, 0.72fr) minmax(0, 1.65fr);
  gap: 34px;
}

.connection-aside {
  padding: 24px;
  background: linear-gradient(145deg, var(--migration-primary-soft), transparent 80%);
  border: 1px solid color-mix(in srgb, var(--migration-primary) 18%, var(--n-border-color));
  border-radius: 14px;

  .aside-icon {
    width: 56px;
    height: 56px;
  }

  h3 {
    margin: 20px 0 8px;
    font-size: 18px;
  }

  p {
    min-height: 62px;
    margin: 0;
    color: var(--n-text-color-3);
    font-size: 13px;
    line-height: 1.6;
  }
}

.destination-chip {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  margin-top: 18px;
  padding: 8px 10px;
  color: var(--migration-primary);
  font-size: 12px;
  font-weight: 600;
  background: var(--n-color);
  border-radius: 9px;
}

.connection-form {
  padding: 8px 6px 0 0;
}

.form-grid {
  display: grid;
  grid-template-columns: 180px minmax(0, 1fr);
  gap: 14px;
}

.form-footer {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 20px;
  margin-top: 8px;
  padding-top: 18px;
  border-top: 1px solid var(--migration-border);

  > span {
    display: flex;
    align-items: center;
    gap: 7px;
    color: var(--n-text-color-3);
    font-size: 12px;
  }
}

.server-flow {
  display: grid;
  grid-template-columns: minmax(0, 1fr) 50px minmax(0, 1fr);
  align-items: center;
}

.server-card {
  display: grid;
  grid-template-columns: auto minmax(0, 1fr) auto;
  align-items: center;
  gap: 14px;
  padding: 20px;
  background: var(--n-color);
  border: 1px solid var(--migration-border);
  border-radius: 15px;

  &.target {
    border-color: color-mix(in srgb, var(--migration-primary) 35%, var(--n-border-color));
  }

  .server-card__icon {
    width: 52px;
    height: 52px;
  }

  div {
    display: flex;
    flex-direction: column;
  }

  small,
  span {
    color: var(--n-text-color-3);
    font-size: 12px;
  }

  strong {
    margin: 2px 0;
    font-size: 17px;
  }
}

.flow-arrow {
  color: var(--n-text-color-3);
  text-align: center;
}

.precheck-grid,
.progress-layout {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 16px;
}

.content-card {
  border-radius: 16px;
}

.card-title {
  display: flex;
  align-items: center;
  gap: 9px;
  font-weight: 650;
}

.chip-list {
  display: flex;
  flex-wrap: wrap;
  gap: 7px;

  &.compact {
    gap: 5px;
  }
}

.detail-list {
  margin-top: 20px;
  padding-top: 14px;
  border-top: 1px solid var(--migration-border);
}

.capability-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  min-height: 40px;
  border-bottom: 1px solid var(--migration-border);

  &:last-child {
    border-bottom: 0;
  }

  > span {
    color: var(--n-text-color-2);
    font-size: 13px;
  }

  &.environments {
    align-items: flex-start;
    padding-top: 11px;
  }
}

.notice-alert {
  border-radius: 12px;
}

.action-bar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  padding-top: 2px;

  > div {
    display: flex;
    align-items: center;
    gap: 12px;
  }
}

.resource-heading {
  align-items: center;
}

.selection-summary {
  display: flex;
  align-items: baseline;
  gap: 7px;
  padding: 10px 15px;
  color: var(--migration-primary);
  background: var(--migration-primary-soft);
  border-radius: 11px;

  strong {
    font-size: 22px;
  }

  span {
    font-size: 12px;
  }
}

.resource-layout {
  display: grid;
  grid-template-columns: 205px minmax(0, 1fr);
  gap: 14px;
  align-items: start;
}

.resource-nav {
  display: flex;
  flex-direction: column;
  gap: 6px;
  padding: 8px;
  background: var(--n-color);
  border: 1px solid var(--migration-border);
  border-radius: 14px;

  button {
    display: grid;
    grid-template-columns: auto minmax(0, 1fr) auto;
    align-items: center;
    gap: 9px;
    width: 100%;
    padding: 11px 10px;
    color: var(--n-text-color-2);
    text-align: left;
    cursor: pointer;
    background: transparent;
    border: 0;
    border-radius: 9px;

    &:hover,
    &.active {
      color: var(--migration-primary);
      background: var(--migration-primary-soft);
    }

    &.active {
      font-weight: 650;
    }
  }
}

.nav-icon {
  display: inline-flex;
}

.resource-panel :deep(.n-card__content) {
  padding-top: 0;
}

.resource-toolbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  min-height: 65px;
  border-bottom: 1px solid var(--migration-border);

  > div {
    display: flex;
    flex-direction: column;
    gap: 2px;
  }

  span {
    color: var(--n-text-color-3);
    font-size: 12px;
  }
}

.resource-list {
  display: flex;
  flex-direction: column;
}

.resource-item {
  display: grid;
  grid-template-columns: auto auto minmax(0, 1fr);
  gap: 12px;
  padding: 17px 4px;
  border-bottom: 1px solid var(--migration-border);

  &:last-child {
    border-bottom: 0;
  }

  &.selected {
    margin: 0 -10px;
    padding-right: 14px;
    padding-left: 14px;
    background: linear-gradient(90deg, var(--migration-primary-soft), transparent 76%);
    border-radius: 10px;
  }

  &.blocked .resource-icon {
    color: #ef4444;
    background: color-mix(in srgb, #ef4444 11%, transparent);
  }
}

.resource-icon {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 38px;
  height: 38px;
  color: var(--migration-primary);
  background: var(--migration-primary-soft);
  border-radius: 10px;
}

.resource-main {
  min-width: 0;
}

.resource-title-row,
.result-title {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 7px;

  strong {
    min-width: 0;
    overflow: hidden;
    font-size: 14px;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
}

.resource-meta {
  display: flex;
  flex-wrap: wrap;
  gap: 8px 16px;
  margin: 5px 0 8px;
  color: var(--n-text-color-3);
  font-size: 12px;
}

.features :deep(.n-tag) {
  background: color-mix(in srgb, var(--n-text-color-3) 7%, transparent);
}

.project-targets {
  display: grid;
  grid-template-columns: minmax(240px, 2fr) minmax(190px, 1fr);
  gap: 10px;
  max-width: 620px;
  margin-top: 10px;
}

.dependency-line {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 6px;
  margin-top: 10px;
  color: var(--n-text-color-3);
  font-size: 12px;
}

.message-list {
  display: flex;
  flex-direction: column;
  gap: 5px;
  margin-top: 9px;

  > div {
    display: flex;
    align-items: flex-start;
    gap: 6px;
    font-size: 12px;
    line-height: 1.45;
  }

  svg {
    flex: 0 0 auto;
    margin-top: 1px;
  }

  &.error {
    color: #ef4444;
  }

  &.warning {
    color: #d97706;
  }
}

.option-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 14px;
}

.option-card {
  display: grid;
  grid-template-columns: auto auto minmax(0, 1fr);
  align-items: center;
  gap: 13px;
  padding: 17px;
  cursor: pointer;
  background: var(--n-color);
  border: 1px solid var(--migration-border);
  border-radius: 14px;

  &.active {
    border-color: color-mix(in srgb, var(--migration-primary) 52%, var(--n-border-color));
    box-shadow: 0 8px 22px var(--migration-primary-soft);
  }

  .option-icon {
    width: 42px;
    height: 42px;
  }

  > span:last-child {
    display: flex;
    flex-direction: column;
    gap: 3px;
  }

  small {
    color: var(--n-text-color-3);
    font-size: 12px;
    line-height: 1.45;
  }
}

.sticky-action {
  position: sticky;
  bottom: 14px;
  z-index: 4;
  padding: 13px 14px;
  background: color-mix(in srgb, var(--n-color) 90%, transparent);
  border: 1px solid var(--migration-border);
  border-radius: 13px;
  box-shadow: 0 12px 28px color-mix(in srgb, #020617 12%, transparent);
  backdrop-filter: blur(14px);
}

.blocked-summary {
  color: #ef4444;
  font-size: 12px;
}

.live-pill {
  display: inline-flex;
  align-items: center;
  gap: 7px;
  padding: 7px 11px;
  color: #16a34a;
  font-size: 12px;
  font-weight: 650;
  background: color-mix(in srgb, #22c55e 10%, transparent);
  border-radius: 999px;

  span {
    width: 7px;
    height: 7px;
    background: #22c55e;
    border-radius: 50%;
    box-shadow: 0 0 0 4px color-mix(in srgb, #22c55e 18%, transparent);
    animation: pulse 1.5s infinite;
  }
}

.progress-overview {
  display: grid;

  :deep(.n-card__content) {
    display: grid;
    grid-template-columns: 160px minmax(0, 1fr) auto;
    align-items: center;
    gap: 20px;
  }

  div {
    display: flex;
    align-items: baseline;
    gap: 8px;
  }

  small,
  > :deep(.n-card__content) > span {
    color: var(--n-text-color-3);
    font-size: 12px;
  }

  strong {
    color: var(--migration-primary);
    font-size: 24px;
  }
}

.progress-layout {
  grid-template-columns: minmax(390px, 0.9fr) minmax(0, 1.1fr);
  align-items: stretch;
}

.progress-list-card,
.log-card {
  min-height: 470px;
}

.progress-list {
  display: flex;
  flex-direction: column;
  gap: 13px;
}

.progress-row {
  display: grid;
  grid-template-columns: auto minmax(0, 1fr) auto;
  align-items: center;
  gap: 11px;
  padding-bottom: 13px;
  border-bottom: 1px solid var(--migration-border);
}

.progress-row__main {
  min-width: 0;

  > div {
    display: flex;
    justify-content: space-between;
    gap: 10px;
    margin-bottom: 7px;
  }

  strong {
    overflow: hidden;
    font-size: 13px;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  span {
    flex: 0 0 auto;
    color: var(--n-text-color-3);
    font-size: 11px;
  }
}

.log-console {
  height: 386px;
  padding: 15px;
  overflow-y: auto;
  color: #cbd5e1;
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
  font-size: 12px;
  line-height: 1.7;
  background: #0b1220;
  border: 1px solid #1e293b;
  border-radius: 11px;

  > div {
    word-break: break-all;
    white-space: pre-wrap;
  }
}

.log-cursor {
  display: flex;
  align-items: center;
  gap: 8px;
  color: #64748b;

  span {
    width: 7px;
    height: 14px;
    background: #60a5fa;
    animation: blink 1s steps(2) infinite;
  }
}

.result-hero {
  display: flex;
  align-items: center;
  gap: 18px;
  padding: 25px;
  background: linear-gradient(135deg, var(--migration-primary-soft), var(--n-color));
  border: 1px solid color-mix(in srgb, var(--migration-primary) 24%, var(--n-border-color));
  border-radius: 18px;

  .result-hero__icon {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    width: 68px;
    height: 68px;
    color: var(--migration-primary);
    background: var(--n-color);
    border-radius: 18px;
  }

  h2 {
    margin: 4px 0;
    font-size: 24px;
  }

  p {
    margin: 0;
    color: var(--n-text-color-3);
    font-size: 12px;
  }
}

.result-stats {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 12px;

  > div {
    display: flex;
    flex-direction: column;
    gap: 2px;
    padding: 17px 19px;
    background: var(--n-color);
    border: 1px solid var(--migration-border);
    border-radius: 13px;
  }

  strong {
    font-size: 24px;
  }

  span {
    color: var(--n-text-color-3);
    font-size: 12px;
  }

  .success strong {
    color: #16a34a;
  }

  .partial strong {
    color: #d97706;
  }

  .failed strong {
    color: #ef4444;
  }
}

.result-list {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.result-item {
  display: grid;
  grid-template-columns: auto minmax(0, 1fr) auto;
  align-items: start;
  gap: 13px;
  padding: 16px;
  background: color-mix(in srgb, var(--n-text-color-3) 3%, transparent);
  border: 1px solid var(--migration-border);
  border-left-width: 3px;
  border-radius: 11px;

  &.success {
    border-left-color: #22c55e;
  }

  &.partial {
    border-left-color: #f59e0b;
  }

  &.failed {
    border-left-color: #ef4444;
  }

  &.skipped {
    border-left-color: #94a3b8;
  }
}

.result-main {
  min-width: 0;
}

.result-title > span {
  margin-left: auto;
  color: var(--n-text-color-3);
  font-size: 11px;
}

.result-error {
  margin: 8px 0 0;
  color: #ef4444;
  font-size: 12px;
}

.result-detail {
  display: grid;
  grid-template-columns: 132px minmax(0, 1fr);
  gap: 5px 12px;
  margin-top: 10px;
  font-size: 12px;

  strong {
    grid-row: 1 / span 30;
  }

  span {
    color: var(--n-text-color-3);
  }
}

.success-detail strong {
  color: #16a34a;
}

.warning-detail strong {
  color: #d97706;
}

.residual-detail strong {
  color: #ef4444;
}

.result-actions {
  justify-content: flex-end;
}

@keyframes pulse {
  50% {
    opacity: 0.45;
  }
}

@keyframes blink {
  50% {
    opacity: 0;
  }
}

@media (max-width: 960px) {
  .migration-hero,
  .section-heading {
    align-items: flex-start;
    flex-direction: column;
  }

  .hero-principle {
    max-width: none;
  }

  .direction-grid,
  .precheck-grid,
  .option-grid,
  .progress-layout {
    grid-template-columns: 1fr;
  }

  .connection-layout {
    grid-template-columns: 1fr;
  }

  .connection-aside p {
    min-height: auto;
  }

  .resource-layout {
    grid-template-columns: 1fr;
  }

  .resource-nav {
    flex-direction: row;
    overflow-x: auto;

    button {
      min-width: 145px;
    }
  }

  .progress-list-card,
  .log-card {
    min-height: auto;
  }
}

@media (max-width: 680px) {
  .direction-grid,
  .result-stats {
    grid-template-columns: 1fr;
  }

  .form-grid {
    grid-template-columns: 1fr;
    gap: 0;
  }

  .form-footer,
  .action-bar {
    align-items: stretch;
    flex-direction: column;
  }

  .form-footer > span {
    order: 2;
  }

  .server-flow {
    grid-template-columns: 1fr;
    gap: 8px;
  }

  .flow-arrow {
    transform: rotate(90deg);
  }

  .progress-overview :deep(.n-card__content) {
    grid-template-columns: 1fr;
    gap: 10px;
  }

  .resource-item {
    grid-template-columns: auto minmax(0, 1fr);
  }

  .resource-item > .resource-icon {
    display: none;
  }

  .project-targets {
    grid-template-columns: 1fr;
  }

  .option-card {
    grid-template-columns: auto minmax(0, 1fr);

    .option-icon {
      display: none;
    }
  }

  .result-item {
    grid-template-columns: auto minmax(0, 1fr);
  }

  .result-item > .n-tag {
    grid-column: 2;
    justify-self: start;
  }

  .result-detail {
    grid-template-columns: 1fr;

    strong {
      grid-row: auto;
    }
  }
}
</style>

<script setup lang="ts">
defineOptions({
  name: 'postgresql-performance',
})

import { NButton, NTag } from 'naive-ui'
import { useGettext } from 'vue3-gettext'

import postgresql from '@/api/apps/postgresql'
import { useConfirm } from '@/components/system/composables/useConfirm'

const { $gettext } = useGettext()
const { confirmDelete, confirmAction } = useConfirm()

const enableTopSQLLoading = ref(false)

const { data: sessions, send: refreshSessions } = useRequest(postgresql.sessions, {
  initialData: [],
})
const { data: topSQL, send: refreshTopSQL } = useRequest(postgresql.topSQL, {
  initialData: { enabled: true, pending_restart: false, items: [] },
})

const formatDuration = (seconds: number) => {
  if (!seconds || seconds <= 0) return '-'
  if (seconds < 60) return `${seconds}s`
  if (seconds < 3600) return `${Math.floor(seconds / 60)}m${seconds % 60}s`
  return `${Math.floor(seconds / 3600)}h${Math.floor((seconds % 3600) / 60)}m`
}

const sessionColumns: any = [
  { title: 'PID', key: 'pid', width: 90 },
  { title: $gettext('Database'), key: 'database', width: 120, ellipsis: { tooltip: true } },
  { title: $gettext('User'), key: 'user', width: 120, ellipsis: { tooltip: true } },
  { title: $gettext('Client'), key: 'client_addr', width: 140, ellipsis: { tooltip: true } },
  { title: $gettext('State'), key: 'state', width: 100, ellipsis: { tooltip: true } },
  {
    title: $gettext('Wait Event'),
    key: 'wait_event',
    width: 160,
    ellipsis: { tooltip: true },
    render(row: any) {
      if (!row.wait_event) return '-'
      return `${row.wait_event_type}: ${row.wait_event}`
    },
  },
  {
    title: $gettext('Blocked By'),
    key: 'blocked_by',
    width: 110,
    render(row: any) {
      if (!row.blocked_by) return '-'
      return h(NTag, { type: 'error', size: 'small' }, { default: () => row.blocked_by })
    },
  },
  {
    title: $gettext('Transaction Duration'),
    key: 'xact_seconds',
    width: 170,
    render: (row: any) => formatDuration(row.xact_seconds),
  },
  {
    title: $gettext('Query Duration'),
    key: 'query_seconds',
    width: 150,
    render: (row: any) => formatDuration(row.query_seconds),
  },
  { title: 'SQL', key: 'query', minWidth: 250, ellipsis: { tooltip: true } },
  {
    title: $gettext('Actions'),
    key: 'actions',
    width: 120,
    render(row: any) {
      return h(
        NButton,
        {
          size: 'small',
          type: 'error',
          onClick: async () => {
            const ok = await confirmDelete({
              title: $gettext('Confirm Terminate'),
              content: $gettext('Are you sure you want to terminate session %{ pid }?', {
                pid: String(row.pid),
              }),
              positiveText: $gettext('Terminate'),
            })
            if (ok) handleTerminate(row.pid)
          },
        },
        { default: () => $gettext('Terminate') },
      )
    },
  },
]

const topSQLColumns: any = [
  { title: $gettext('Database'), key: 'database', width: 120, ellipsis: { tooltip: true } },
  { title: $gettext('Calls'), key: 'calls', width: 100 },
  { title: $gettext('Total Time (ms)'), key: 'total_ms', width: 150 },
  { title: $gettext('Mean Time (ms)'), key: 'mean_ms', width: 150 },
  { title: $gettext('Rows'), key: 'rows', width: 100 },
  {
    title: $gettext('Cache Hit Rate'),
    key: 'hit_rate',
    width: 140,
    render: (row: any) => `${row.hit_rate}%`,
  },
  { title: 'SQL', key: 'query', minWidth: 300, ellipsis: { tooltip: true } },
]

const handleTerminate = (pid: number) => {
  useRequest(postgresql.terminateSession(pid)).onSuccess(() => {
    window.$message.success($gettext('Terminated successfully'))
    refreshSessions()
  })
}

const handleEnableTopSQL = async () => {
  const ok = await confirmAction({
    type: 'info',
    title: $gettext('Confirm Enable'),
    content: $gettext(
      'This will add pg_stat_statements to shared_preload_libraries, a restart of PostgreSQL is required to take effect.',
    ),
  })
  if (!ok) return
  enableTopSQLLoading.value = true
  useRequest(postgresql.enableTopSQL())
    .onSuccess(() => {
      window.$message.success($gettext('Enabled, please restart PostgreSQL to take effect'))
      refreshTopSQL()
    })
    .onComplete(() => {
      enableTopSQLLoading.value = false
    })
}

const handleResetTopSQL = async () => {
  const ok = await confirmAction({
    type: 'warning',
    title: $gettext('Confirm Reset'),
    content: $gettext('Are you sure you want to reset all SQL statistics?'),
  })
  if (!ok) return
  useRequest(postgresql.resetTopSQL()).onSuccess(() => {
    window.$message.success($gettext('Reset successfully'))
    refreshTopSQL()
  })
}
</script>

<template>
  <n-tabs type="segment" animated>
    <n-tab-pane name="sessions" :tab="$gettext('Sessions')">
      <n-flex vertical>
        <n-flex>
          <n-button type="primary" @click="() => refreshSessions()">
            {{ $gettext('Refresh') }}
          </n-button>
        </n-flex>
        <n-data-table
          striped
          :columns="sessionColumns"
          :data="sessions"
          :scroll-x="1700"
          max-height="60vh"
        />
      </n-flex>
    </n-tab-pane>
    <n-tab-pane name="top-sql" :tab="'Top SQL'">
      <n-flex vertical>
        <n-alert v-if="!topSQL.enabled && topSQL.pending_restart" type="warning">
          {{
            $gettext('pg_stat_statements is configured, restart PostgreSQL to take effect.')
          }}
        </n-alert>
        <template v-else-if="!topSQL.enabled">
          <n-alert type="info">
            {{
              $gettext(
                'pg_stat_statements is not enabled. After enabling, SQL performance statistics will be collected, a restart of PostgreSQL is required.',
              )
            }}
          </n-alert>
          <n-flex>
            <n-button
              type="primary"
              :loading="enableTopSQLLoading"
              :disabled="enableTopSQLLoading"
              @click="handleEnableTopSQL"
            >
              {{ $gettext('Enable') }}
            </n-button>
          </n-flex>
        </template>
        <template v-if="topSQL.enabled">
          <n-flex>
            <n-button type="primary" @click="() => refreshTopSQL()">
              {{ $gettext('Refresh') }}
            </n-button>
            <n-button type="warning" @click="handleResetTopSQL">
              {{ $gettext('Reset Statistics') }}
            </n-button>
          </n-flex>
          <n-data-table
            striped
            :columns="topSQLColumns"
            :data="topSQL.items"
            :scroll-x="1300"
            max-height="60vh"
          />
        </template>
      </n-flex>
    </n-tab-pane>
  </n-tabs>
</template>

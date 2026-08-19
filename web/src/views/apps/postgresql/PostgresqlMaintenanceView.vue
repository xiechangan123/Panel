<script setup lang="ts">
defineOptions({
  name: 'postgresql-maintenance',
})

import { NButton, NSpace, NTag, NTooltip } from 'naive-ui'
import { useGettext } from 'vue3-gettext'

import postgresql from '@/api/apps/postgresql'
import { useConfirm } from '@/components/system/composables/useConfirm'

const { $gettext } = useGettext()
const { confirmDelete, confirmAction } = useConfirm()

const selectedDatabase = ref('')

const { data: databases } = useRequest(postgresql.databases, {
  initialData: [],
}).onSuccess(({ data }) => {
  if (data?.length && !selectedDatabase.value) {
    // 默认选中 postgres 库
    selectedDatabase.value = data.includes('postgres') ? 'postgres' : data[0]
    refreshBloat()
  }
})

const databaseOptions = computed(() =>
  (databases.value as string[]).map((name) => ({ label: name, value: name })),
)

const { data: bloat, send: sendBloat } = useRequest(
  () => postgresql.bloat(selectedDatabase.value),
  {
    immediate: false,
    initialData: { repack_installed: false, items: [] },
  },
)
const refreshBloat = () => {
  if (selectedDatabase.value) sendBloat()
}

const { data: wal, send: refreshWal } = useRequest(postgresql.wal, {
  initialData: {
    wal_size: '-',
    archiver: { archived_count: 0, failed_count: 0, last_archived_wal: '', last_failed_wal: '' },
    slots: [],
    replications: [],
  },
})

const handleMaintenance = async (row: any, operation: string) => {
  const ok = await confirmAction({
    type: 'warning',
    title: $gettext('Confirm Operation'),
    content: $gettext('Are you sure you want to run %{ op } on %{ table }?', {
      op: operation.toUpperCase(),
      table: `${row.schema}.${row.table}`,
    }),
  })
  if (!ok) return
  useRequest(
    postgresql.runMaintenance({
      database: selectedDatabase.value,
      schema: row.schema,
      table: row.table,
      operation,
    }),
  ).onSuccess(() => {
    window.$message.success($gettext('Task submitted, please check progress in background tasks'))
  })
}

const bloatColumns: any = [
  { title: $gettext('Schema'), key: 'schema', width: 110, ellipsis: { tooltip: true } },
  { title: $gettext('Table'), key: 'table', minWidth: 150, ellipsis: { tooltip: true } },
  { title: $gettext('Size'), key: 'size', width: 100 },
  { title: $gettext('Live Tuples'), key: 'live_tuples', width: 110 },
  { title: $gettext('Dead Tuples'), key: 'dead_tuples', width: 110 },
  {
    title: $gettext('Dead Rate'),
    key: 'dead_rate',
    width: 100,
    render(row: any) {
      const type = row.dead_rate >= 20 ? 'error' : row.dead_rate >= 10 ? 'warning' : 'default'
      return h(NTag, { type, size: 'small' }, { default: () => `${row.dead_rate}%` })
    },
  },
  {
    title: $gettext('Last Vacuum'),
    key: 'last_vacuum',
    width: 140,
    render: (row: any) => row.last_vacuum || row.last_autovacuum || '-',
  },
  {
    title: $gettext('Last Analyze'),
    key: 'last_analyze',
    width: 140,
    render: (row: any) => row.last_analyze || row.last_autoanalyze || '-',
  },
  {
    title: $gettext('Actions'),
    key: 'actions',
    width: 240,
    render(row: any) {
      const buttons = [
        h(
          NButton,
          { size: 'small', type: 'info', onClick: () => handleMaintenance(row, 'vacuum') },
          { default: () => 'VACUUM' },
        ),
        h(
          NButton,
          { size: 'small', onClick: () => handleMaintenance(row, 'analyze') },
          { default: () => 'ANALYZE' },
        ),
      ]
      if (bloat.value.repack_installed) {
        buttons.push(
          h(
            NButton,
            { size: 'small', type: 'warning', onClick: () => handleMaintenance(row, 'repack') },
            { default: () => 'REPACK' },
          ),
        )
      } else {
        buttons.push(
          h(
            NTooltip,
            {},
            {
              trigger: () =>
                h(NButton, { size: 'small', disabled: true }, { default: () => 'REPACK' }),
              default: () =>
                $gettext('pg_repack is not installed, please install it in the extensions tab'),
            },
          ),
        )
      }
      return h(NSpace, { size: 'small' }, { default: () => buttons })
    },
  },
]

const slotColumns: any = [
  { title: $gettext('Slot Name'), key: 'name', minWidth: 180, ellipsis: { tooltip: true } },
  { title: $gettext('Type'), key: 'type', width: 110 },
  {
    title: $gettext('Active'),
    key: 'active',
    width: 100,
    render(row: any) {
      return row.active
        ? h(NTag, { type: 'success', size: 'small' }, { default: () => $gettext('Yes') })
        : h(NTag, { type: 'warning', size: 'small' }, { default: () => $gettext('No') })
    },
  },
  { title: $gettext('Retained WAL'), key: 'retained_wal', width: 130 },
  {
    title: $gettext('Actions'),
    key: 'actions',
    width: 100,
    render(row: any) {
      if (row.active) return '-'
      return h(
        NButton,
        {
          size: 'small',
          type: 'error',
          onClick: async () => {
            const ok = await confirmDelete({
              title: $gettext('Confirm Delete'),
              content: $gettext(
                'After deleting slot %{ name }, the corresponding subscription or standby will not be able to continue syncing. Are you sure?',
                { name: row.name },
              ),
              positiveText: $gettext('Delete'),
              countdown: 5,
            })
            if (ok) handleDropSlot(row.name)
          },
        },
        { default: () => $gettext('Delete') },
      )
    },
  },
]

const replicationColumns: any = [
  { title: $gettext('Client'), key: 'client_addr', minWidth: 150 },
  { title: $gettext('State'), key: 'state', width: 120 },
  { title: $gettext('Sync State'), key: 'sync_state', width: 120 },
  { title: $gettext('Lag'), key: 'lag', width: 120 },
]

const handleDropSlot = (name: string) => {
  useRequest(postgresql.dropReplicationSlot(name)).onSuccess(() => {
    window.$message.success($gettext('Deleted successfully'))
    refreshWal()
  })
}
</script>

<template>
  <n-tabs type="segment" animated>
    <n-tab-pane name="bloat" :tab="$gettext('Table Bloat')">
      <n-flex vertical>
        <n-flex>
          <n-select
            v-model:value="selectedDatabase"
            :options="databaseOptions"
            class="w-60"
            @update:value="refreshBloat"
          />
          <n-button type="primary" @click="refreshBloat">
            {{ $gettext('Refresh') }}
          </n-button>
        </n-flex>
        <n-data-table
          striped
          :columns="bloatColumns"
          :data="bloat.items"
          :scroll-x="1200"
          max-height="60vh"
        />
      </n-flex>
    </n-tab-pane>
    <n-tab-pane name="wal" tab="WAL">
      <n-flex vertical>
        <n-flex>
          <n-button type="primary" @click="() => refreshWal()">
            {{ $gettext('Refresh') }}
          </n-button>
        </n-flex>
        <n-card :title="$gettext('WAL Status')">
          <n-flex>
            <n-statistic :label="$gettext('WAL Size')" :value="wal.wal_size" />
            <n-statistic
              class="ml-40"
              :label="$gettext('Archived Count')"
              :value="wal.archiver.archived_count"
            />
            <n-statistic
              class="ml-40"
              :label="$gettext('Archive Failed Count')"
              :value="wal.archiver.failed_count"
            />
          </n-flex>
        </n-card>
        <n-card :title="$gettext('Replication Slots')">
          <n-data-table striped :columns="slotColumns" :data="wal.slots" :scroll-x="620" />
        </n-card>
        <n-card v-if="wal.replications.length" :title="$gettext('Replication Status')">
          <n-data-table
            striped
            :columns="replicationColumns"
            :data="wal.replications"
            :scroll-x="510"
          />
        </n-card>
      </n-flex>
    </n-tab-pane>
  </n-tabs>
</template>

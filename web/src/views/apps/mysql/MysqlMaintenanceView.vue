<script setup lang="ts">
import { NButton, NSpace, NTag } from 'naive-ui'
import { useGettext } from 'vue3-gettext'

import type mysql from '@/api/apps/mysql'
import { useConfirm } from '@/components/system/composables/useConfirm'

const props = defineProps<{
  api: typeof mysql
}>()

const { $gettext } = useGettext()
const { confirmDelete, confirmAction } = useConfirm()

const { data: tables, send: refreshTables } = useRequest(props.api.tables, {
  initialData: [],
})
const { data: binlog, send: refreshBinlogs } = useRequest(props.api.binlogs, {
  initialData: { enabled: true, total_size: '-', items: [] },
})
const { data: replication, send: refreshReplication } = useRequest(props.api.replication, {
  initialData: { enabled: false },
})

const handleMaintenance = async (row: any, operation: string) => {
  const content =
    operation === 'optimize'
      ? $gettext(
          'OPTIMIZE will rebuild the InnoDB table, which may take a long time for large tables. Are you sure you want to run it on %{ table }?',
          { table: `${row.database}.${row.table}` },
        )
      : $gettext('Are you sure you want to run %{ op } on %{ table }?', {
          op: operation.toUpperCase(),
          table: `${row.database}.${row.table}`,
        })
  const ok = await confirmAction({
    type: 'warning',
    title: $gettext('Confirm Operation'),
    content,
  })
  if (!ok) return
  useRequest(
    props.api.runMaintenance({
      database: row.database,
      table: row.table,
      operation,
    }),
  ).onSuccess(() => {
    window.$message.success($gettext('Task submitted, please check progress in background tasks'))
  })
}

const tableColumns: any = [
  { title: $gettext('Database'), key: 'database', width: 130, ellipsis: { tooltip: true } },
  { title: $gettext('Table'), key: 'table', minWidth: 150, ellipsis: { tooltip: true } },
  { title: $gettext('Engine'), key: 'engine', width: 100 },
  { title: $gettext('Rows'), key: 'rows', width: 110 },
  { title: $gettext('Size'), key: 'size', width: 100 },
  {
    title: $gettext('Fragment Rate'),
    key: 'fragment_rate',
    width: 130,
    render(row: any) {
      const rate = Math.round(row.fragment_rate * 10) / 10
      const type = rate >= 30 ? 'error' : rate >= 10 ? 'warning' : 'default'
      return h(NTag, { type, size: 'small' }, { default: () => `${rate}%` })
    },
  },
  {
    title: $gettext('Actions'),
    key: 'actions',
    width: 230,
    render(row: any) {
      return h(NSpace, { size: 'small', wrap: false }, {
        default: () => [
          h(
            NButton,
            { size: 'small', type: 'warning', onClick: () => handleMaintenance(row, 'optimize') },
            { default: () => 'OPTIMIZE' },
          ),
          h(
            NButton,
            { size: 'small', onClick: () => handleMaintenance(row, 'analyze') },
            { default: () => 'ANALYZE' },
          ),
        ],
      })
    },
  },
]

const binlogColumns: any = [
  { title: $gettext('File'), key: 'name', minWidth: 200 },
  { title: $gettext('Size'), key: 'size', width: 120 },
  {
    title: $gettext('Actions'),
    key: 'actions',
    width: 160,
    render(row: any) {
      return h(
        NButton,
        {
          size: 'small',
          type: 'error',
          onClick: async () => {
            const ok = await confirmDelete({
              title: $gettext('Confirm Purge'),
              content: $gettext(
                'This will delete all binlogs before %{ file }. If a replica has not applied them yet, replication will break. Are you sure?',
                { file: row.name },
              ),
              positiveText: $gettext('Purge'),
              countdown: 5,
            })
            if (ok) handlePurge(row.name)
          },
        },
        { default: () => $gettext('Purge to Here') },
      )
    },
  },
]

const handlePurge = (file: string) => {
  useRequest(props.api.purgeBinlog(file)).onSuccess(() => {
    window.$message.success($gettext('Purged successfully'))
    refreshBinlogs()
  })
}

const replicationRunning = (value: string) => {
  return value === 'Yes'
    ? h(NTag, { type: 'success', size: 'small' }, { default: () => $gettext('Running') })
    : h(NTag, { type: 'error', size: 'small' }, { default: () => value || $gettext('Stopped') })
}
</script>

<template>
  <n-tabs type="segment" animated>
    <n-tab-pane name="tables" :tab="$gettext('Table Maintenance')">
      <n-flex vertical>
        <n-flex>
          <n-button type="primary" @click="() => refreshTables()">
            {{ $gettext('Refresh') }}
          </n-button>
        </n-flex>
        <n-data-table
          striped
          :columns="tableColumns"
          :data="tables"
          :scroll-x="990"
          max-height="60vh"
        />
      </n-flex>
    </n-tab-pane>
    <n-tab-pane name="binlog" :tab="'Binlog'">
      <n-flex vertical>
        <n-alert v-if="!binlog.enabled" type="info">
          {{ $gettext('Binary log is not enabled.') }}
        </n-alert>
        <template v-else>
          <n-flex>
            <n-button type="primary" @click="() => refreshBinlogs()">
              {{ $gettext('Refresh') }}
            </n-button>
          </n-flex>
          <n-card>
            <n-flex>
              <n-statistic :label="$gettext('File Count')" :value="binlog.items.length" />
              <n-statistic class="ml-40" :label="$gettext('Total Size')" :value="binlog.total_size" />
            </n-flex>
          </n-card>
          <n-data-table
            striped
            :columns="binlogColumns"
            :data="binlog.items"
            :scroll-x="520"
            max-height="50vh"
          />
        </template>
      </n-flex>
    </n-tab-pane>
    <n-tab-pane name="replication" :tab="$gettext('Replication')">
      <n-flex vertical>
        <n-flex>
          <n-button type="primary" @click="() => refreshReplication()">
            {{ $gettext('Refresh') }}
          </n-button>
        </n-flex>
        <n-alert v-if="!replication.enabled" type="info">
          {{ $gettext('This instance is not a replica.') }}
        </n-alert>
        <n-card v-else :title="$gettext('Replication Status')">
          <n-descriptions label-placement="left" :column="2">
            <n-descriptions-item :label="$gettext('Source Host')">
              {{ replication.source_host || '-' }}
            </n-descriptions-item>
            <n-descriptions-item :label="$gettext('Delay (s)')">
              {{ replication.seconds_behind || '-' }}
            </n-descriptions-item>
            <n-descriptions-item :label="$gettext('IO Thread')">
              <component :is="replicationRunning(replication.io_running)" />
            </n-descriptions-item>
            <n-descriptions-item :label="$gettext('SQL Thread')">
              <component :is="replicationRunning(replication.sql_running)" />
            </n-descriptions-item>
            <n-descriptions-item v-if="replication.last_error" :label="$gettext('Last Error')" :span="2">
              {{ replication.last_error }}
            </n-descriptions-item>
          </n-descriptions>
        </n-card>
      </n-flex>
    </n-tab-pane>
  </n-tabs>
</template>

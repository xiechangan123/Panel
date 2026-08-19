<script setup lang="ts">
defineOptions({
  name: 'postgresql-extensions',
})

import { NButton, NDataTable, NTag } from 'naive-ui'
import { useGettext } from 'vue3-gettext'

import postgresql from '@/api/apps/postgresql'
import { useConfirm } from '@/components/system/composables/useConfirm'

const { $gettext } = useGettext()
const { confirmDelete, confirmAction } = useConfirm()

const { data: extensions, send: refreshExtensions } = useRequest(postgresql.extensions, {
  initialData: [],
})

const columns: any = [
  {
    title: $gettext('Name'),
    key: 'name',
    minWidth: 150,
    ellipsis: { tooltip: true },
  },
  {
    title: $gettext('Extension Name'),
    key: 'ext_name',
    minWidth: 150,
    ellipsis: { tooltip: true },
  },
  {
    title: $gettext('Description'),
    key: 'description',
    minWidth: 250,
    ellipsis: { tooltip: true },
  },
  {
    title: $gettext('Status'),
    key: 'status',
    width: 150,
    render(row: any) {
      return row.installed
        ? h(NTag, { type: 'success', size: 'small' }, { default: () => $gettext('Installed') })
        : h(NTag, { type: 'default', size: 'small' }, { default: () => $gettext('Not Installed') })
    },
  },
  {
    title: $gettext('Actions'),
    key: 'actions',
    width: 150,
    render(row: any) {
      if (!row.installed) {
        return h(
          NButton,
          {
            size: 'small',
            type: 'info',
            onClick: async () => {
              const ok = await confirmAction({
                type: 'info',
                title: $gettext('Confirm Install'),
                content: $gettext('Are you sure you want to install %{ name }?', {
                  name: row.name,
                }),
              })
              if (ok) handleInstall(row.slug)
            },
          },
          { default: () => $gettext('Install') },
        )
      }
      return h(
        NButton,
        {
          size: 'small',
          type: 'error',
          onClick: async () => {
            const ok = await confirmDelete({
              title: $gettext('Confirm Uninstall'),
              content: $gettext(
                'Please make sure the extension %{ ext_name } has been dropped (DROP EXTENSION) in all databases that use it, otherwise those databases will fail to load it. Are you sure you want to uninstall %{ name }?',
                { name: row.name, ext_name: row.ext_name },
              ),
              positiveText: $gettext('Uninstall'),
              countdown: 5,
            })
            if (ok) handleUninstall(row.slug)
          },
        },
        { default: () => $gettext('Uninstall') },
      )
    },
  },
]

const handleInstall = (slug: string) => {
  useRequest(postgresql.installExtension(slug)).onSuccess(() => {
    window.$message.success($gettext('Task submitted, please check progress in background tasks'))
    refreshExtensions()
  })
}

const handleUninstall = (slug: string) => {
  useRequest(postgresql.uninstallExtension(slug)).onSuccess(() => {
    window.$message.success($gettext('Task submitted, please check progress in background tasks'))
    refreshExtensions()
  })
}
</script>

<template>
  <n-flex vertical>
    <n-alert type="info">
      {{
        $gettext(
          'Extensions are only installed at the file level. After installation, run CREATE EXTENSION in each database where you want to use them. Some extensions modify the main configuration during installation and require a restart of PostgreSQL to take effect.',
        )
      }}
    </n-alert>
    <n-data-table striped :columns="columns" :data="extensions" :scroll-x="850" />
  </n-flex>
</template>

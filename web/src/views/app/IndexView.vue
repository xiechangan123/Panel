<script setup lang="ts">
defineOptions({
  name: 'app-index'
})

import VersionModal from '@/views/app/VersionModal.vue'

import { NButton, NDataTable, NFlex, NPopconfirm, NSwitch } from 'naive-ui'
import { useI18n } from 'vue-i18n'

import app from '@/api/panel/app'
import TheIcon from '@/components/custom/TheIcon.vue'
import { router } from '@/router'
import { renderIcon } from '@/utils'

const { t } = useI18n()

const versionModalShow = ref(false)
const versionModalOperation = ref('安装')
const versionModalInfo = ref<any>({})

const columns: any = [
  {
    key: 'icon',
    fixed: 'left',
    width: 80,
    align: 'center',
    render(row: any) {
      return h(TheIcon, {
        icon: row.icon,
        size: 26,
        color: `var(--primary-color)`
      })
    }
  },
  {
    title: t('appIndex.columns.name'),
    key: 'name',
    width: 300,
    resizable: true,
    ellipsis: { tooltip: true }
  },
  {
    title: t('appIndex.columns.description'),
    key: 'description',
    minWidth: 300,
    resizable: true,
    ellipsis: { tooltip: true }
  },
  {
    title: t('appIndex.columns.installedVersion'),
    key: 'installed_version',
    width: 100,
    ellipsis: { tooltip: true }
  },
  {
    title: t('appIndex.columns.show'),
    key: 'show',
    width: 100,
    align: 'center',
    render(row: any) {
      return h(NSwitch, {
        size: 'small',
        rubberBand: false,
        value: row.show,
        onUpdateValue: () => handleShowChange(row)
      })
    }
  },
  {
    title: t('appIndex.columns.actions'),
    key: 'actions',
    width: 300,
    hideInExcel: true,
    render(row: any) {
      return h(
        NFlex,
        {
          justify: 'center'
        },
        {
          default: () => [
            row.installed && row.update_exist
              ? h(
                  NPopconfirm,
                  {
                    onPositiveClick: () => handleUpdate(row.slug)
                  },
                  {
                    default: () => {
                      return t('appIndex.confirm.update', { app: row.name })
                    },
                    trigger: () => {
                      return h(
                        NButton,
                        {
                          size: 'small',
                          type: 'warning'
                        },
                        {
                          default: () => t('appIndex.buttons.update'),
                          icon: renderIcon('material-symbols:arrow-circle-up-outline-rounded', {
                            size: 14
                          })
                        }
                      )
                    }
                  }
                )
              : null,
            row.installed
              ? h(
                  NButton,
                  {
                    size: 'small',
                    type: 'success',
                    onClick: () => handleManage(row.slug)
                  },
                  {
                    default: () => t('appIndex.buttons.manage'),
                    icon: renderIcon('material-symbols:settings-outline', { size: 14 })
                  }
                )
              : null,
            row.installed
              ? h(
                  NPopconfirm,
                  {
                    onPositiveClick: () => handleUninstall(row.slug)
                  },
                  {
                    default: () => {
                      return t('appIndex.confirm.uninstall', { app: row.name })
                    },
                    trigger: () => {
                      return h(
                        NButton,
                        {
                          size: 'small',
                          type: 'error'
                        },
                        {
                          default: () => t('appIndex.buttons.uninstall'),
                          icon: renderIcon('material-symbols:delete-outline', { size: 14 })
                        }
                      )
                    }
                  }
                )
              : null,
            !row.installed
              ? h(
                  NButton,
                  {
                    size: 'small',
                    type: 'info',
                    onClick: () => {
                      versionModalShow.value = true
                      versionModalOperation.value = '安装'
                      versionModalInfo.value = row
                    }
                  },
                  {
                    default: () => t('appIndex.buttons.install'),
                    icon: renderIcon('material-symbols:download-rounded', { size: 14 })
                  }
                )
              : null
          ]
        }
      )
    }
  }
]

const { loading, data, page, total, pageSize, pageCount, refresh } = usePagination(
  (page, pageSize) => app.list(page, pageSize),
  {
    initialData: { total: 0, list: [] },
    initialPageSize: 20,
    total: (res: any) => res.total,
    data: (res: any) => res.items
  }
)

const handleShowChange = (row: any) => {
  useRequest(app.updateShow(row.slug, !row.show)).onSuccess(() => {
    row.show = !row.show
    window.$message.success(t('appIndex.alerts.setup'))
  })
}

const handleUpdate = (slug: string) => {
  useRequest(app.update(slug)).onSuccess(() => {
    window.$message.success(t('appIndex.alerts.update'))
  })
}

const handleUninstall = (slug: string) => {
  useRequest(app.uninstall(slug)).onSuccess(() => {
    window.$message.success(t('appIndex.alerts.uninstall'))
  })
}

const handleManage = (slug: string) => {
  router.push({ name: 'apps-' + slug + '-index' })
}

const handleUpdateCache = () => {
  useRequest(app.updateCache()).onSuccess(() => {
    refresh()
    window.$message.success(t('appIndex.alerts.cache'))
  })
}

onMounted(() => {
  refresh()
})
</script>

<template>
  <common-page show-footer>
    <template #action>
      <n-button type="primary" @click="handleUpdateCache">
        <TheIcon :size="18" icon="material-symbols:refresh" />
        {{ $t('appIndex.buttons.updateCache') }}
      </n-button>
    </template>
    <n-flex vertical>
      <n-alert type="warning">{{ $t('appIndex.alerts.warning') }}</n-alert>
      <n-data-table
        striped
        remote
        :scroll-x="1200"
        :loading="loading"
        :columns="columns"
        :data="data"
        :row-key="(row: any) => row.slug"
        v-model:page="page"
        v-model:pageSize="pageSize"
        :pagination="{
          page: page,
          pageCount: pageCount,
          pageSize: pageSize,
          itemCount: total,
          showQuickJumper: true,
          showSizePicker: true,
          pageSizes: [20, 50, 100, 200]
        }"
      />
      <version-modal
        v-model:show="versionModalShow"
        v-model:operation="versionModalOperation"
        v-model:info="versionModalInfo"
      />
    </n-flex>
  </common-page>
</template>

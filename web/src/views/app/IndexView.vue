<script setup lang="ts">
defineOptions({
  name: 'app-index'
})

import VersionModal from '@/views/app/VersionModal.vue'

import { NButton, NDataTable, NFlex, NPopconfirm, NSwitch } from 'naive-ui'
import { useGettext } from 'vue3-gettext'

import app from '@/api/panel/app'
import TheIcon from '@/components/custom/TheIcon.vue'
import { router } from '@/router'
import { renderIcon } from '@/utils'

const { $gettext } = useGettext()

const versionModalShow = ref(false)
const versionModalOperation = ref($gettext('Install'))
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
    title: $gettext('App Name'),
    key: 'name',
    width: 300,
    resizable: true,
    ellipsis: { tooltip: true }
  },
  {
    title: $gettext('Description'),
    key: 'description',
    minWidth: 300,
    resizable: true,
    ellipsis: { tooltip: true }
  },
  {
    title: $gettext('Installed Version'),
    key: 'installed_version',
    width: 100,
    ellipsis: { tooltip: true }
  },
  {
    title: $gettext('Show in Home'),
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
    title: $gettext('Actions'),
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
                      return $gettext(
                        'Updating app %{ app } may reset related configurations to default state, are you sure to continue?',
                        { app: row.name }
                      )
                    },
                    trigger: () => {
                      return h(
                        NButton,
                        {
                          size: 'small',
                          type: 'warning'
                        },
                        {
                          default: () => $gettext('Update'),
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
                    default: () => $gettext('Manage'),
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
                      return $gettext('Are you sure to uninstall app %{ app }?', { app: row.name })
                    },
                    trigger: () => {
                      return h(
                        NButton,
                        {
                          size: 'small',
                          type: 'error'
                        },
                        {
                          default: () => $gettext('Uninstall'),
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
                      versionModalOperation.value = $gettext('Install')
                      versionModalInfo.value = row
                    }
                  },
                  {
                    default: () => $gettext('Install'),
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
    window.$message.success($gettext('Setup successfully'))
  })
}

const handleUpdate = (slug: string) => {
  useRequest(app.update(slug)).onSuccess(() => {
    window.$message.success(
      $gettext('Task submitted, please check the progress in background tasks')
    )
  })
}

const handleUninstall = (slug: string) => {
  useRequest(app.uninstall(slug)).onSuccess(() => {
    window.$message.success(
      $gettext('Task submitted, please check the progress in background tasks')
    )
  })
}

const handleManage = (slug: string) => {
  router.push({ name: 'apps-' + slug + '-index' })
}

const handleUpdateCache = () => {
  useRequest(app.updateCache()).onSuccess(() => {
    refresh()
    window.$message.success($gettext('Cache updated successfully'))
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
        {{ $gettext('Update Cache') }}
      </n-button>
    </template>
    <n-flex vertical>
      <n-alert type="warning">{{
        $gettext(
          'Before updating apps, it is strongly recommended to backup/snapshot first, so you can roll back immediately if there are any issues!'
        )
      }}</n-alert>
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
    </n-flex>
    <version-modal
      v-model:show="versionModalShow"
      v-model:operation="versionModalOperation"
      v-model:info="versionModalInfo"
    />
  </common-page>
</template>

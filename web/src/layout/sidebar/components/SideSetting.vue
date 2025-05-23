<script lang="ts" setup>
import type { TreeSelectOption } from 'naive-ui'
import { useGettext } from 'vue3-gettext'

import TheIcon from '@/components/custom/TheIcon.vue'
import MenuCollapse from '@/layout/header/components/MenuCollapse.vue'
import { translateTitle } from '@/locales/menu'
import { usePermissionStore, useThemeStore } from '@/store'
import type { RouteType } from '~/types/router'

const { $gettext } = useGettext()
const themeStore = useThemeStore()
const permissionStore = usePermissionStore()

const settingModal = ref(false)

const getOption = (route: RouteType): TreeSelectOption => {
  let menuItem: TreeSelectOption = {
    label: route.meta?.title ? translateTitle(route.meta.title) : route.name,
    key: route.name
  }

  const visibleChildren = route.children
    ? route.children.filter((item: RouteType) => item.name && !item.isHidden)
    : []

  if (!visibleChildren.length) return menuItem

  if (visibleChildren.length === 1) {
    // 单个子路由处理
    const singleRoute = visibleChildren[0]
    menuItem.label = singleRoute.meta?.title
      ? translateTitle(singleRoute.meta.title)
      : singleRoute.name
    const visibleItems = singleRoute.children
      ? singleRoute.children.filter((item: RouteType) => item.name && !item.isHidden)
      : []

    if (visibleItems.length === 1) menuItem = getOption(visibleItems[0])
    else if (visibleItems.length > 1)
      menuItem.children = visibleItems.map((item) => getOption(item))
  } else {
    menuItem.children = visibleChildren.map((item) => getOption(item))
  }

  return menuItem
}

const menus = computed<TreeSelectOption[]>(() => {
  return permissionStore.allMenus.map((item) => getOption(item))
})
</script>

<template>
  <div h-40 flex justify-between px-20>
    <menu-collapse />
    <n-tooltip trigger="hover">
      <template #trigger>
        <the-icon
          v-show="!themeStore.sider.collapsed"
          :size="22"
          icon="material-symbols:settings"
          @click="settingModal = true"
        />
      </template>
      {{ $gettext('Menu Settings') }}
    </n-tooltip>
    <n-modal
      v-model:show="settingModal"
      preset="card"
      :title="$gettext('Menu Settings')"
      style="width: 60vw"
      size="huge"
      :bordered="false"
      :segmented="false"
      @close="settingModal = false"
      @mask-click="settingModal = false"
    >
      <n-form>
        <n-flex vertical>
          <n-alert type="info">
            {{
              $gettext(
                'Settings are saved in the browser and will be reset after clearing the browser cache'
              )
            }}
          </n-alert>
          <n-form-item :label="$gettext('Custom Logo')">
            <n-input
              v-model:value="themeStore.logo"
              :placeholder="$gettext('Please enter the complete URL')"
            />
          </n-form-item>
          <n-form-item :label="$gettext('Hide Menu')">
            <n-tree-select
              cascade
              checkable
              clearable
              multiple
              :options="menus"
              v-model:value="permissionStore.hiddenRoutes"
            ></n-tree-select>
          </n-form-item>
        </n-flex>
      </n-form>
    </n-modal>
  </div>
</template>

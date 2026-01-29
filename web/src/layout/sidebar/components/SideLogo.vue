<script lang="ts" setup>
import logoImg from '@/assets/images/logo.svg'
import { useThemeStore } from '@/store'

const themeStore = useThemeStore()
const router = useRouter()
const logo = computed(() => themeStore.logo || logoImg)

const toHome = () => {
  router.push({ name: 'home-index' })
}
</script>

<template>
  <div class="px-12 f-c-c h-60 cursor-pointer" @click="toHome">
    <n-image :src="logo" preview-disabled class="flex-shrink-0 h-36" />
    <n-tooltip v-if="!themeStore.sider.collapsed" trigger="hover" :delay="500">
      <template #trigger>
        <h2 class="text-18 font-bold ml-10 max-w-140 truncate">
          {{ themeStore.name }}
        </h2>
      </template>
      {{ themeStore.name }}
    </n-tooltip>
  </div>
</template>

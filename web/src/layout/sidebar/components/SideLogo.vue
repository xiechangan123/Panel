<script lang="ts" setup>
import logoImg from '@/assets/images/logo.png'
import { useThemeStore } from '@/store'

const themeStore = useThemeStore()
const router = useRouter()
const logo = computed(() => themeStore.logo || logoImg)

const toDashboard = () => {
  router.push({ name: 'dashboard' })
}
</script>

<template>
  <div class="h-60 f-c-c cursor-pointer" @click="toDashboard">
    <n-image :src="logo" height="32" preview-disabled />
    <h2
      v-show="!themeStore.sider.collapsed"
      class="ml-10 max-w-140 flex-shrink-0 text-18 font-bold"
    >
      {{ themeStore.name }}
    </h2>
  </div>
</template>

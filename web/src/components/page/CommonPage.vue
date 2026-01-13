<script lang="ts" setup>
interface Props {
  showFooter?: boolean
  showHeader?: boolean
  title?: string
}

withDefaults(defineProps<Props>(), {
  showFooter: false,
  showHeader: false,
  title: undefined
})
</script>

<template>
  <app-page :show-footer="showFooter">
    <div class="flex flex-col flex-1 gap-10 min-h-0">
      <header v-if="showHeader">
        <slot v-if="$slots.header" name="header" />
        <n-card v-else size="small">
          <slot name="tabbar" />
        </n-card>
      </header>
      <n-card class="flex-1 min-h-0 overflow-auto">
        <slot />
      </n-card>
    </div>
  </app-page>
</template>

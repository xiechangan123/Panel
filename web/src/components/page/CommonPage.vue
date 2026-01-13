<script lang="ts" setup>
interface Props {
  showFooter?: boolean
  showHeader?: boolean
  title?: string
  flex?: boolean
}

withDefaults(defineProps<Props>(), {
  showFooter: false,
  showHeader: false,
  title: undefined,
  flex: false
})
</script>

<template>
  <app-page :show-footer="showFooter">
    <div class="flex flex-col flex-1 gap-10" :class="{ 'min-h-0': flex }">
      <header v-if="showHeader">
        <slot v-if="$slots.header" name="header" />
        <n-card v-else size="small">
          <slot name="tabbar" />
        </n-card>
      </header>
      <n-card
        :class="flex ? 'flex-1 min-h-0' : 'flex-1'"
        :content-class="flex ? 'flex flex-col min-h-0 h-full' : undefined"
      >
        <slot />
      </n-card>
    </div>
  </app-page>
</template>

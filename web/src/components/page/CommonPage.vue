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
    <div class="flex flex-col gap-10 flex-1 min-h-0">
      <header v-if="showHeader">
        <slot v-if="$slots.header" name="header" />
        <n-card v-else size="small">
          <slot name="tabbar" />
        </n-card>
      </header>
      <n-card class="main-card flex-1 min-h-0">
        <slot />
      </n-card>
    </div>
  </app-page>
</template>

<style scoped lang="scss">
.main-card {
  display: flex;
  flex-direction: column;

  :deep(.n-card__content) {
    flex: 1;
    display: flex;
    flex-direction: column;
    min-height: 0;

    // n-flex vertical 填满
    > .n-flex {
      flex: 1;
      min-height: 0;
    }

    // n-data-table 填满
    .n-data-table {
      flex: 1;
      min-height: 0;
    }
  }
}
</style>

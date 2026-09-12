<script setup lang="ts">
/**
 * 字符串列表输入框
 * 粘贴以换行、逗号、分号或空格分隔的文本时自动拆分成多项，无需逐个点加号
 */
defineOptions({
  name: 'ListInput',
})

const props = withDefaults(
  defineProps<{
    placeholder?: string
    disabled?: boolean
    /** 粘贴内容的拆分规则 */
    separator?: RegExp
  }>(),
  {
    placeholder: '',
    disabled: false,
    separator: () => /[\s,;]+/,
  },
)

const value = defineModel<string[]>('value', { default: () => [] })

// 列表为空时展示一个空输入框，省去先点一次加号
const items = computed({
  get: () => (value.value.length ? value.value : ['']),
  set: (next: string[]) => {
    value.value = next
  },
})

// 含分隔符的文本在 index 处展开成多项并去重，拆不出多项时返回 false
const expand = (index: number, text: string) => {
  const parts = text
    .split(props.separator)
    .map((item) => item.trim())
    .filter(Boolean)
  if (parts.length < 2) return false

  const next = [...items.value]
  next.splice(index, 1, ...parts)
  items.value = [...new Set(next.filter(Boolean))]
  return true
}

const updateItem = (index: number, item: string) => {
  if (expand(index, item)) return
  const next = [...items.value]
  next[index] = item
  items.value = next
}

// 拦截粘贴，避免先拼成一行再拆分导致的闪烁，也保证粘贴内容落在当前项而非末尾
const handlePaste = (e: ClipboardEvent, index: number) => {
  if (expand(index, e.clipboardData?.getData('text') ?? '')) {
    e.preventDefault()
  }
}
</script>

<template>
  <!-- 自定义插槽下 n-dynamic-input 新增项默认为 null，需用 on-create 指定为空串 -->
  <n-dynamic-input v-model:value="items" :disabled="disabled" :on-create="() => ''">
    <template #default="{ index }">
      <n-input
        :value="items[index]"
        :disabled="disabled"
        :placeholder="placeholder"
        @update:value="(item: string) => updateItem(index, item)"
        @paste="(e: ClipboardEvent) => handlePaste(e, index)"
      />
    </template>
  </n-dynamic-input>
</template>

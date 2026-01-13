<script setup lang="ts">
import { NButton, NCard, NEllipsis, NFlex, NGrid, NGridItem, NSpin, NTag } from 'naive-ui'
import { useGettext } from 'vue3-gettext'

import template from '@/api/panel/template'
import TemplateDeployModal from './TemplateDeployModal.vue'
import type { Template } from './types'

const { $gettext } = useGettext()

const selectedCategory = ref<string>('')
const deployModalShow = ref(false)
const selectedTemplate = ref<Template | null>(null)

const { loading, data, refresh } = usePagination(template.list, {
  initialData: []
})

// 获取所有分类
const categories = computed(() => {
  const cats = new Set<string>()
  data.value?.forEach((t: Template) => {
    t.categories?.forEach((c) => cats.add(c))
  })
  return Array.from(cats)
})

// 过滤后的模版列表
const filteredTemplates = computed(() => {
  if (!selectedCategory.value) {
    return data.value || []
  }
  return (data.value || []).filter((t: Template) => t.categories?.includes(selectedCategory.value))
})

const handleCategoryChange = (category: string) => {
  selectedCategory.value = category
}

const handleDeploy = (tpl: Template) => {
  selectedTemplate.value = tpl
  deployModalShow.value = true
}

onMounted(() => {
  refresh()
})
</script>

<template>
  <n-flex vertical :size="20">
    <n-flex>
      <n-tag
        :type="selectedCategory === '' ? 'primary' : 'default'"
        :bordered="selectedCategory !== ''"
        style="cursor: pointer"
        @click="handleCategoryChange('')"
      >
        {{ $gettext('All') }}
      </n-tag>
      <n-tag
        v-for="cat in categories"
        :key="cat"
        :type="selectedCategory === cat ? 'primary' : 'default'"
        :bordered="selectedCategory !== cat"
        style="cursor: pointer"
        @click="handleCategoryChange(cat)"
      >
        {{ cat }}
      </n-tag>
    </n-flex>

    <n-spin :show="loading">
      <n-grid :x-gap="16" :y-gap="16" cols="1 s:2 m:3 l:4" responsive="screen">
        <n-grid-item v-for="tpl in filteredTemplates" :key="tpl.slug">
          <n-card hoverable style="height: 100%">
            <n-flex vertical :size="12">
              <n-flex justify="space-between" align="center">
                <span>{{ tpl.name }}</span>
                <n-tag size="small" type="info">{{ tpl.version }}</n-tag>
              </n-flex>
              <n-ellipsis :line-clamp="2" :tooltip="{ width: 300 }">
                {{ tpl.description }}
              </n-ellipsis>
              <n-flex :size="4" style="margin-top: auto">
                <n-tag v-for="cat in tpl.categories" :key="cat" size="small">
                  {{ cat }}
                </n-tag>
              </n-flex>
            </n-flex>
            <template #action>
              <n-flex justify="end">
                <n-button size="small" type="primary" @click="handleDeploy(tpl)">
                  {{ $gettext('Deploy') }}
                </n-button>
              </n-flex>
            </template>
          </n-card>
        </n-grid-item>
      </n-grid>

      <n-empty v-if="!loading && filteredTemplates.length === 0" />
    </n-spin>
  </n-flex>

  <template-deploy-modal
    v-model:show="deployModalShow"
    :template="selectedTemplate"
    @success="refresh"
  />
</template>

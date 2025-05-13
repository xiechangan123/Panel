<script setup lang="ts">
import { locales as availableLocales } from '@/utils'
import { useGettext } from 'vue3-gettext'

const { $gettext } = useGettext()

const model = defineModel<any>('model', { type: Object, required: true })

const locales = computed(() => {
  return Object.entries(availableLocales).map(([code, name]: [string, string]) => {
    return {
      label: name,
      value: code
    }
  })
})

const channels = [
  {
    label: $gettext('Stable'),
    value: 'stable'
  },
  {
    label: $gettext('Beta'),
    value: 'beta'
  }
]
</script>

<template>
  <n-space vertical>
    <n-alert type="info">
      {{
        $gettext(
          'Modifying panel port/entrance requires corresponding changes in the browser address bar to access the panel!'
        )
      }}
    </n-alert>
    <n-form>
      <n-form-item :label="$gettext('Panel Name')">
        <n-input v-model:value="model.name" :placeholder="$gettext('Panel Name')" />
      </n-form-item>
      <n-form-item :label="$gettext('Language')">
        <n-select v-model:value="model.locale" :options="locales"> </n-select>
      </n-form-item>
      <n-form-item :label="$gettext('Update Channel')">
        <n-select v-model:value="model.channel" :options="channels"> </n-select>
      </n-form-item>
      <n-form-item :label="$gettext('Username')">
        <n-input v-model:value="model.username" :placeholder="$gettext('admin')" />
      </n-form-item>
      <n-form-item :label="$gettext('Password')">
        <n-input v-model:value="model.password" :placeholder="$gettext('admin')" />
      </n-form-item>
      <n-form-item :label="$gettext('Certificate Default Email')">
        <n-input v-model:value="model.email" :placeholder="$gettext('admin@yourdomain.com')" />
      </n-form-item>
      <n-form-item :label="$gettext('Port')">
        <n-input-number v-model:value="model.port" :placeholder="$gettext('8888')" w-full />
      </n-form-item>
      <n-form-item :label="$gettext('Default Website Directory')">
        <n-input v-model:value="model.website_path" :placeholder="$gettext('/www/wwwroot')" />
      </n-form-item>
      <n-form-item :label="$gettext('Default Backup Directory')">
        <n-input v-model:value="model.backup_path" :placeholder="$gettext('/www/backup')" />
      </n-form-item>
    </n-form>
  </n-space>
</template>

<style scoped lang="scss"></style>

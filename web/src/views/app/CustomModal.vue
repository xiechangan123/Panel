<script setup lang="ts">
import { useGettext } from 'vue3-gettext'

import app from '@/api/panel/app'
import CommonEditor from '@/components/common/CommonEditor.vue'

const { $gettext } = useGettext()

const show = defineModel<boolean>('show', { type: Boolean, required: true })
const props = defineProps({
  slug: {
    type: String,
    required: true,
  },
  name: {
    type: String,
    required: true,
  },
})

const doSubmit = ref(false)

const model = ref({
  pre_script: '',
  args: '',
})

watch(show, (value) => {
  if (value) {
    useRequest(app.getCustom(props.slug)).onSuccess(({ data }: any) => {
      model.value = {
        pre_script: data.pre_script,
        args: data.args,
      }
    })
  }
})

const handleSubmit = () => {
  doSubmit.value = true
  useRequest(app.saveCustom(props.slug, model.value))
    .onSuccess(() => {
      show.value = false
      window.$message.success($gettext('Saved successfully'))
    })
    .onComplete(() => {
      doSubmit.value = false
    })
}
</script>

<template>
  <n-modal
    v-model:show="show"
    preset="card"
    :title="$gettext('Custom Compile Params') + ' - ' + name"
    style="width: 60vw"
    size="huge"
    :bordered="false"
    :segmented="false"
  >
    <n-flex vertical>
      <n-alert type="info">
        {{
          $gettext(
            'Settings are saved persistently and applied automatically on every install and update (recompile), no need to re-enter after reinstalling.',
          )
        }}
      </n-alert>
      <n-form :model="model">
        <n-form-item :label="$gettext('Pre-script')">
          <n-flex vertical class="w-full">
            <n-text depth="3">
              {{
                $gettext(
                  'Runs before configure, can be used to download and prepare third-party module sources.',
                )
              }}
            </n-text>
            <common-editor v-model:value="model.pre_script" lang="shell" height="220px" />
          </n-flex>
        </n-form-item>
        <n-form-item :label="$gettext('Compile Params')">
          <n-flex vertical class="w-full">
            <n-text depth="3">
              {{
                $gettext(
                  'One param per line, appended to the end of configure. Lines starting with # are ignored.',
                )
              }}
            </n-text>
            <n-input
              v-model:value="model.args"
              type="textarea"
              :rows="6"
              class="font-mono"
              placeholder="--with-xxx_module
--add-module=/path/to/module"
            />
          </n-flex>
        </n-form-item>
      </n-form>
      <n-button type="primary" block :loading="doSubmit" :disabled="doSubmit" @click="handleSubmit">
        {{ $gettext('Save') }}
      </n-button>
    </n-flex>
  </n-modal>
</template>

<style scoped lang="scss"></style>

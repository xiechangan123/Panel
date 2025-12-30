<script setup lang="ts">
import file from '@/api/panel/file'
import { decodeBase64 } from '@/utils'
import { languageByPath } from '@/utils/file'
import ace from 'ace-builds'
import { VAceEditor } from 'vue3-ace-editor'
import { useGettext } from 'vue3-gettext'

import extBeautifyUrl from 'ace-builds/src-min-noconflict/ext-beautify?url'
import extCodeLensUrl from 'ace-builds/src-min-noconflict/ext-code_lens?url'
import extCommandBarUrl from 'ace-builds/src-min-noconflict/ext-command_bar?url'
import extEmmetUrl from 'ace-builds/src-min-noconflict/ext-emmet?url'
import extErrorMarkerUrl from 'ace-builds/src-min-noconflict/ext-error_marker?url'
import extInlineAutocompleteUrl from 'ace-builds/src-min-noconflict/ext-inline_autocomplete?url'
import extKeybindingMenuUrl from 'ace-builds/src-min-noconflict/ext-keybinding_menu?url'
import extLanguageToolsUrl from 'ace-builds/src-min-noconflict/ext-language_tools?url'
import extSearchboxUrl from 'ace-builds/src-min-noconflict/ext-searchbox?url'
import extSettingsMenuUrl from 'ace-builds/src-min-noconflict/ext-settings_menu?url'
import extSpellcheckUrl from 'ace-builds/src-min-noconflict/ext-spellcheck?url'
import extWhitespaceUrl from 'ace-builds/src-min-noconflict/ext-whitespace?url'
import modeApacheConfUrl from 'ace-builds/src-min-noconflict/mode-apache_conf?url'
import modeCssUrl from 'ace-builds/src-min-noconflict/mode-css?url'
import modeCsvUrl from 'ace-builds/src-min-noconflict/mode-csv?url'
import modeDockerfileUrl from 'ace-builds/src-min-noconflict/mode-dockerfile?url'
import modeDotUrl from 'ace-builds/src-min-noconflict/mode-dot?url'
import modeGolangUrl from 'ace-builds/src-min-noconflict/mode-golang?url'
import modeHtmlUrl from 'ace-builds/src-min-noconflict/mode-html?url'
import modeIniUrl from 'ace-builds/src-min-noconflict/mode-ini?url'
import modeJavaUrl from 'ace-builds/src-min-noconflict/mode-java?url'
import modeJavascriptUrl from 'ace-builds/src-min-noconflict/mode-javascript?url'
import modeJsonUrl from 'ace-builds/src-min-noconflict/mode-json?url'
import modeLuaUrl from 'ace-builds/src-min-noconflict/mode-lua?url'
import modeMakefileUrl from 'ace-builds/src-min-noconflict/mode-makefile?url'
import modeMarkdownUrl from 'ace-builds/src-min-noconflict/mode-markdown?url'
import modeMySqlUrl from 'ace-builds/src-min-noconflict/mode-mysql?url'
import modeNginxUrl from 'ace-builds/src-min-noconflict/mode-nginx?url'
import modePgSqlUrl from 'ace-builds/src-min-noconflict/mode-pgsql?url'
import modePhpUrl from 'ace-builds/src-min-noconflict/mode-php?url'
import modePythonUrl from 'ace-builds/src-min-noconflict/mode-python?url'
import modeRubyUrl from 'ace-builds/src-min-noconflict/mode-ruby?url'
import modeRustUrl from 'ace-builds/src-min-noconflict/mode-rust?url'
import modeScssUrl from 'ace-builds/src-min-noconflict/mode-scss?url'
import modeShUrl from 'ace-builds/src-min-noconflict/mode-sh?url'
import modeSqlUrl from 'ace-builds/src-min-noconflict/mode-sql?url'
import modeSvgUrl from 'ace-builds/src-min-noconflict/mode-svg?url'
import modeTextUrl from 'ace-builds/src-min-noconflict/mode-text?url'
import modeTomlUrl from 'ace-builds/src-min-noconflict/mode-toml?url'
import modeTypescriptUrl from 'ace-builds/src-min-noconflict/mode-typescript?url'
import modeVueUrl from 'ace-builds/src-min-noconflict/mode-vue?url'
import modeXmlUrl from 'ace-builds/src-min-noconflict/mode-xml?url'
import modeYamlUrl from 'ace-builds/src-min-noconflict/mode-yaml?url'
import themeMonokaiUrl from 'ace-builds/src-min-noconflict/theme-monokai?url'
import workerBaseUrl from 'ace-builds/src-min-noconflict/worker-base?url'
import workerCssUrl from 'ace-builds/src-min-noconflict/worker-css?url'
import workerHtmlUrl from 'ace-builds/src-min-noconflict/worker-html?url'
import workerJsUrl from 'ace-builds/src-min-noconflict/worker-javascript?url'
import workerJsonUrl from 'ace-builds/src-min-noconflict/worker-json?url'
import workerLuaUrl from 'ace-builds/src-min-noconflict/worker-lua?url'
import workerPhpUrl from 'ace-builds/src-min-noconflict/worker-php?url'
import workerYamlUrl from 'ace-builds/src-min-noconflict/worker-yaml?url'

ace.config.setModuleUrl('ace/theme/monokai', themeMonokaiUrl)
ace.config.setModuleUrl('ace/ext/inline_autocomplete', extInlineAutocompleteUrl)
ace.config.setModuleUrl('ace/ext/emmet', extEmmetUrl)
ace.config.setModuleUrl('ace/ext/command_bar', extCommandBarUrl)
ace.config.setModuleUrl('ace/ext/code_lens', extCodeLensUrl)
ace.config.setModuleUrl('ace/ext/error_marker', extErrorMarkerUrl)
ace.config.setModuleUrl('ace/ext/spellcheck', extSpellcheckUrl)
ace.config.setModuleUrl('ace/ext/settings_menu', extSettingsMenuUrl)
ace.config.setModuleUrl('ace/ext/keybinding_menu', extKeybindingMenuUrl)
ace.config.setModuleUrl('ace/ext/whitespace', extWhitespaceUrl)
ace.config.setModuleUrl('ace/ext/beautify', extBeautifyUrl)
ace.config.setModuleUrl('ace/ext/searchbox', extSearchboxUrl)
ace.config.setModuleUrl('ace/ext/language_tools', extLanguageToolsUrl)
ace.config.setModuleUrl('ace/mode/apache_conf', modeApacheConfUrl)
ace.config.setModuleUrl('ace/mode/css', modeCssUrl)
ace.config.setModuleUrl('ace/mode/csv', modeCsvUrl)
ace.config.setModuleUrl('ace/mode/dockerfile', modeDockerfileUrl)
ace.config.setModuleUrl('ace/mode/dot', modeDotUrl)
ace.config.setModuleUrl('ace/mode/golang', modeGolangUrl)
ace.config.setModuleUrl('ace/mode/html', modeHtmlUrl)
ace.config.setModuleUrl('ace/mode/ini', modeIniUrl)
ace.config.setModuleUrl('ace/mode/java', modeJavaUrl)
ace.config.setModuleUrl('ace/mode/javascript', modeJavascriptUrl)
ace.config.setModuleUrl('ace/mode/json', modeJsonUrl)
ace.config.setModuleUrl('ace/mode/lua', modeLuaUrl)
ace.config.setModuleUrl('ace/mode/makefile', modeMakefileUrl)
ace.config.setModuleUrl('ace/mode/markdown', modeMarkdownUrl)
ace.config.setModuleUrl('ace/mode/mysql', modeMySqlUrl)
ace.config.setModuleUrl('ace/mode/nginx', modeNginxUrl)
ace.config.setModuleUrl('ace/mode/pgsql', modePgSqlUrl)
ace.config.setModuleUrl('ace/mode/php', modePhpUrl)
ace.config.setModuleUrl('ace/mode/python', modePythonUrl)
ace.config.setModuleUrl('ace/mode/ruby', modeRubyUrl)
ace.config.setModuleUrl('ace/mode/rust', modeRustUrl)
ace.config.setModuleUrl('ace/mode/scss', modeScssUrl)
ace.config.setModuleUrl('ace/mode/sh', modeShUrl)
ace.config.setModuleUrl('ace/mode/sql', modeSqlUrl)
ace.config.setModuleUrl('ace/mode/svg', modeSvgUrl)
ace.config.setModuleUrl('ace/mode/text', modeTextUrl)
ace.config.setModuleUrl('ace/mode/toml', modeTomlUrl)
ace.config.setModuleUrl('ace/mode/typescript', modeTypescriptUrl)
ace.config.setModuleUrl('ace/mode/vue', modeVueUrl)
ace.config.setModuleUrl('ace/mode/xml', modeXmlUrl)
ace.config.setModuleUrl('ace/mode/yaml', modeYamlUrl)
ace.config.setModuleUrl('ace/mode/base_worker', workerBaseUrl)
ace.config.setModuleUrl('ace/mode/json_worker', workerJsonUrl)
ace.config.setModuleUrl('ace/mode/css_worker', workerCssUrl)
ace.config.setModuleUrl('ace/mode/html_worker', workerHtmlUrl)
ace.config.setModuleUrl('ace/mode/javascript_worker', workerJsUrl)
ace.config.setModuleUrl('ace/mode/php_worker', workerPhpUrl)
ace.config.setModuleUrl('ace/mode/lua_worker', workerLuaUrl)
ace.config.setModuleUrl('ace/mode/yaml_worker', workerYamlUrl)

const { $gettext } = useGettext()
const props = defineProps({
  path: {
    type: String,
    required: true
  },
  readOnly: {
    type: Boolean,
    required: true
  }
})

const disabled = ref(false) // 在出现错误的情况下禁用保存
const content = ref('')

const get = () => {
  useRequest(file.content(encodeURIComponent(props.path)))
    .onSuccess(({ data }) => {
      content.value = decodeBase64(data.content)
      window.$message.success($gettext('Retrieved successfully'))
    })
    .onError(() => {
      disabled.value = true
    })
}

const save = () => {
  if (disabled.value) {
    window.$message.error($gettext('Cannot save in current state'))
    return
  }
  useRequest(file.save(props.path, content.value)).onSuccess(() => {
    window.$message.success($gettext('Saved successfully'))
  })
}

onMounted(() => {
  get()
})

defineExpose({
  get,
  save
})
</script>

<template>
  <v-ace-editor
    v-model:value="content"
    :lang="languageByPath(props.path)"
    :options="{ useWorker: true }"
    theme="monokai"
    style="height: 60vh"
  />
</template>

<style scoped lang="scss"></style>

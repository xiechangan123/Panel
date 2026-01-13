<script setup lang="ts">
import ws from '@/api/ws'
import type { LogInst } from 'naive-ui'
import { useGettext } from 'vue3-gettext'

const { $gettext } = useGettext()
const props = defineProps({
  path: {
    type: String,
    required: false
  },
  service: {
    type: String,
    required: false
  },
  language: {
    type: String,
    required: false,
    default: 'systemdlog'
  }
})

const log = ref('')
const logRef = ref<LogInst | null>(null)
let logWs: WebSocket | null = null

const init = async () => {
  let cmd = ''
  if (props.path) {
    cmd = `tail -n 100 -f '${props.path}'`
  } else if (props.service) {
    cmd = `journalctl -u '${props.service}' -f`
  } else {
    window.$message.error($gettext('Path or service cannot be empty'))
    return
  }
  ws.exec(cmd)
    .then((ws: WebSocket) => {
      logWs = ws
      ws.onmessage = (event) => {
        log.value += event.data + '\n'
        const lines = log.value.split('\n')
        if (lines.length > 500) {
          log.value = lines.slice(lines.length - 500).join('\n')
        }
      }
    })
    .catch(() => {
      window.$message.error($gettext('Failed to get log stream'))
    })
}

const close = () => {
  if (logWs) {
    logWs.close()
  }
  log.value = ''
}

watch(
  () => props.path,
  () => {
    close()
    init()
  }
)

watchEffect(() => {
  if (log.value) {
    nextTick(() => {
      logRef.value?.scrollTo({ position: 'bottom', silent: true })
    })
  }
})

onMounted(() => {
  init()
})

onUnmounted(() => {
  close()
})

defineExpose({
  init
})
</script>

<template>
  <n-log v-if="log" ref="logRef" :log="log" trim :rows="40" :language="props.language" />
  <n-empty v-else :description="$gettext('No logs available')" />
</template>

<style scoped lang="scss"></style>

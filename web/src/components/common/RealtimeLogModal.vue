<script setup lang="ts">
import '@fontsource-variable/jetbrains-mono/wght-italic.css'
import '@fontsource-variable/jetbrains-mono/wght.css'
import { ClipboardAddon } from '@xterm/addon-clipboard'
import { FitAddon } from '@xterm/addon-fit'
import { Unicode11Addon } from '@xterm/addon-unicode11'
import { WebLinksAddon } from '@xterm/addon-web-links'
import { WebglAddon } from '@xterm/addon-webgl'
import { Terminal } from '@xterm/xterm'
import '@xterm/xterm/css/xterm.css'
import { useGettext } from 'vue3-gettext'

import ws from '@/api/ws'

const { $gettext } = useGettext()
const show = defineModel<boolean>('show', { type: Boolean, required: true })
const props = defineProps({
  path: {
    type: String,
    required: true
  }
})

const terminalRef = ref<HTMLElement | null>(null)
const term = ref<Terminal | null>(null)
let ptyWs: WebSocket | null = null
let fitAddon: FitAddon | null = null
let webglAddon: WebglAddon | null = null
let resizeObserver: ResizeObserver | null = null

// 初始化终端并连接 PTY
const initTerminal = async () => {
  if (!terminalRef.value || !props.path) return

  try {
    ptyWs = await ws.pty(`tail -n 1000 -F '${props.path}'`)
    ptyWs.binaryType = 'arraybuffer'

    term.value = new Terminal({
      allowProposedApi: true,
      lineHeight: 1.2,
      fontSize: 14,
      fontFamily: `'JetBrains Mono Variable', monospace`,
      cursorBlink: false,
      cursorStyle: 'bar',
      tabStopWidth: 4,
      disableStdin: false,
      convertEol: true,
      scrollback: 10000,
      theme: { background: '#111', foreground: '#fff' }
    })

    fitAddon = new FitAddon()
    webglAddon = new WebglAddon()

    term.value.loadAddon(fitAddon)
    term.value.loadAddon(new ClipboardAddon())
    term.value.loadAddon(new WebLinksAddon())
    term.value.loadAddon(new Unicode11Addon())
    term.value.unicode.activeVersion = '11'
    term.value.loadAddon(webglAddon)
    webglAddon.onContextLoss(() => {
      webglAddon?.dispose()
    })
    term.value.open(terminalRef.value)

    ptyWs.onmessage = (ev) => {
      const data: ArrayBuffer | string = ev.data
      term.value?.write(typeof data === 'string' ? data : new Uint8Array(data))
    }

    term.value.onData((data) => {
      if (ptyWs?.readyState === WebSocket.OPEN) {
        ptyWs.send(data)
      }
    })
    term.value.onBinary((data) => {
      if (ptyWs?.readyState === WebSocket.OPEN) {
        const buffer = new Uint8Array(data.length)
        for (let i = 0; i < data.length; ++i) {
          buffer[i] = data.charCodeAt(i) & 255
        }
        ptyWs.send(buffer)
      }
    })
    term.value.onResize(({ rows, cols }) => {
      if (ptyWs && ptyWs.readyState === WebSocket.OPEN) {
        ptyWs.send(
          JSON.stringify({
            resize: true,
            columns: cols,
            rows: rows
          })
        )
      }
    })

    fitAddon.fit()
    window.addEventListener('resize', onTerminalResize, false)

    if (terminalRef.value.parentElement) {
      resizeObserver = new ResizeObserver(() => onTerminalResize())
      resizeObserver.observe(terminalRef.value.parentElement)
    }

    ptyWs.onclose = () => {
      if (term.value) {
        term.value.write('\r\n' + $gettext('Connection closed.'))
      }
    }

    ptyWs.onerror = (event) => {
      if (term.value) {
        term.value.write('\r\n' + $gettext('Connection error.'))
      }
      console.error(event)
      ptyWs?.close()
    }
  } catch (error) {
    console.error('Failed to start PTY:', error)
    window.$message.error($gettext('Failed to get log stream'))
  }
}

// 关闭终端并清理资源
const closeTerminal = () => {
  try {
    if (ptyWs) {
      ptyWs.close()
      ptyWs = null
    }
    if (term.value) {
      term.value.dispose()
      term.value = null
    }
    fitAddon = null
    webglAddon = null
    if (resizeObserver) {
      resizeObserver.disconnect()
      resizeObserver = null
    }
    if (terminalRef.value) {
      terminalRef.value.innerHTML = ''
    }
    window.removeEventListener('resize', onTerminalResize)
  } catch {
    /* empty */
  }
}

const onTerminalResize = () => {
  if (fitAddon && term.value) {
    fitAddon.fit()
  }
}

const onTerminalWheel = (event: WheelEvent) => {
  if (event.ctrlKey && term.value && fitAddon) {
    event.preventDefault()
    if (event.deltaY > 0) {
      if (term.value.options.fontSize! > 12) {
        term.value.options.fontSize = term.value.options.fontSize! - 1
      }
    } else {
      term.value.options.fontSize = term.value.options.fontSize! + 1
    }
    fitAddon.fit()
  }
}

// path 或 show 变化时重建或销毁终端
watch([() => props.path, () => show.value], async () => {
  closeTerminal()
  if (show.value) {
    await nextTick()
    await initTerminal()
  }
})

onUnmounted(() => {
  closeTerminal()
})
</script>

<template>
  <n-modal
    v-model:show="show"
    preset="card"
    :title="$gettext('Logs')"
    style="width: 80vw"
    size="huge"
    :bordered="false"
    :segmented="false"
    @after-leave="closeTerminal"
  >
    <div
      ref="terminalRef"
      class="realtime-log-terminal"
      @wheel="onTerminalWheel"
    ></div>
  </n-modal>
</template>

<style scoped lang="scss">
.realtime-log-terminal {
  height: 60vh;
  min-height: 300px;
  background: #111;
}

:deep(.xterm) {
  padding: 1rem !important;
}

:deep(.xterm .xterm-viewport::-webkit-scrollbar) {
  border-radius: 0.4rem;
  height: 6px;
  width: 8px;
}

:deep(.xterm .xterm-viewport::-webkit-scrollbar-thumb) {
  background-color: #666;
  border-radius: 0.4rem;
  box-shadow: inset 0 0 5px rgba(0, 0, 0, 0.2);
  transition: all 1s;
}

:deep(.xterm .xterm-viewport:hover::-webkit-scrollbar-thumb) {
  background-color: #aaa;
}

:deep(.xterm .xterm-viewport::-webkit-scrollbar-track) {
  background-color: #111;
  border-radius: 0.4rem;
  box-shadow: inset 0 0 5px rgba(0, 0, 0, 0.2);
  transition: all 1s;
}

:deep(.xterm .xterm-viewport:hover::-webkit-scrollbar-track) {
  background-color: #444;
}
</style>

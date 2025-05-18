<script setup lang="ts">
defineOptions({
  name: 'toolbox-benchmark'
})

import benchmark from '@/api/panel/toolbox-benchmark'
import TheIcon from '@/components/custom/TheIcon.vue'
import { useGettext } from 'vue3-gettext'

const { $gettext } = useGettext()
const inTest = ref(false)
const current = ref($gettext('CPU'))
const progress = ref(0)

const tests = [
  'image',
  'machine',
  'compile',
  'encryption',
  'compression',
  'physics',
  'json',
  'memory',
  'disk'
]

const cpu = ref({
  image: {
    single: 0,
    multi: 0
  },
  machine: {
    single: 0,
    multi: 0
  },
  compile: {
    single: 0,
    multi: 0
  },
  encryption: {
    single: 0,
    multi: 0
  },
  compression: {
    single: 0,
    multi: 0
  },
  physics: {
    single: 0,
    multi: 0
  },
  json: {
    single: 0,
    multi: 0
  }
})

const cpuTotal = computed(() => {
  return {
    single: Object.values(cpu.value).reduce((a, b) => a + b.single, 0),
    multi: Object.values(cpu.value).reduce((a, b) => a + b.multi, 0)
  }
})

const memory = ref({
  score: 0,
  bandwidth: $gettext('Pending benchmark'),
  latency: $gettext('Pending benchmark')
})

const disk = ref({
  score: 0,
  1024: {
    read_iops: $gettext('Pending benchmark'),
    read_speed: $gettext('Pending benchmark'),
    write_iops: $gettext('Pending benchmark'),
    write_speed: $gettext('Pending benchmark')
  },
  4: {
    read_iops: $gettext('Pending benchmark'),
    read_speed: $gettext('Pending benchmark'),
    write_iops: $gettext('Pending benchmark'),
    write_speed: $gettext('Pending benchmark')
  },
  512: {
    read_iops: $gettext('Pending benchmark'),
    read_speed: $gettext('Pending benchmark'),
    write_iops: $gettext('Pending benchmark'),
    write_speed: $gettext('Pending benchmark')
  },
  64: {
    read_iops: $gettext('Pending benchmark'),
    read_speed: $gettext('Pending benchmark'),
    write_iops: $gettext('Pending benchmark'),
    write_speed: $gettext('Pending benchmark')
  }
})

const handleTest = async () => {
  inTest.value = true
  progress.value = 0
  for (let i = 0; i < tests.length; i++) {
    const test = tests[i]
    current.value = test
    if (test != 'memory' && test != 'disk') {
      for (let j = 0; j < 2; j++) {
        cpu.value[test as keyof typeof cpu.value][j === 1 ? 'multi' : 'single'] =
          await benchmark.test(test, j === 1)
      }
    } else {
      const data = await benchmark.test(test, false)
      if (test === 'memory') {
        memory.value = data
      } else {
        disk.value = data
      }
    }
    progress.value = Math.round(((i + 1) / tests.length) * 100)
  }
  inTest.value = false
}
</script>

<template>
  <common-page show-footer>
    <n-flex vertical>
      <n-alert type="warning">
        {{
          $gettext(
            'Benchmark results are for reference only and may differ from actual performance due to system resource scheduling, caching, and other factors!'
          )
        }}
      </n-alert>
      <n-alert
        v-if="inTest"
        :title="$gettext('Benchmarking in progress, it may take some time...')"
        type="info"
      >
        {{ $gettext('Current project: %{ current }', { current: current }) }}
      </n-alert>
      <n-progress v-if="inTest" :percentage="progress" color="var(--primary-color)" processing />
    </n-flex>
    <n-flex vertical items-center pt-40>
      <div w-800>
        <n-grid :cols="3">
          <n-gi>
            <n-popover trigger="hover">
              <template #trigger>
                <n-flex vertical items-center>
                  <div v-if="cpuTotal.single !== 0 && cpuTotal.multi !== 0">
                    {{ $gettext('Single-core') }}
                    <n-number-animation :from="0" :to="cpuTotal.single" show-separator />
                    / {{ $gettext('Multi-core') }}
                    <n-number-animation :from="0" :to="cpuTotal.multi" show-separator />
                  </div>
                  <div v-else>{{ $gettext('Pending benchmark') }}</div>
                  <n-progress
                    type="circle"
                    :percentage="100"
                    :stroke-width="3"
                    color="var(--primary-color)"
                  >
                    <the-icon :size="50" icon="bi:cpu" color="var(--primary-color)" />
                  </n-progress>
                  {{ $gettext('CPU') }}
                </n-flex>
              </template>
              <n-table :single-line="false" striped>
                <tr>
                  <th>{{ $gettext('Image Processing') }}</th>
                  <td>
                    {{
                      $gettext('Single-core %{ single } / Multi-core %{ multi }', {
                        single: cpu.image.single,
                        multi: cpu.image.multi
                      })
                    }}
                  </td>
                </tr>
                <tr>
                  <th>{{ $gettext('Machine Learning') }}</th>
                  <td>
                    {{
                      $gettext('Single-core %{ single } / Multi-core %{ multi }', {
                        single: cpu.machine.single,
                        multi: cpu.machine.multi
                      })
                    }}
                  </td>
                </tr>
                <tr>
                  <th>{{ $gettext('Program Compilation') }}</th>
                  <td>
                    {{
                      $gettext('Single-core %{ single } / Multi-core %{ multi }', {
                        single: cpu.compile.single,
                        multi: cpu.compile.multi
                      })
                    }}
                  </td>
                </tr>
                <tr>
                  <th>{{ $gettext('AES Encryption') }}</th>
                  <td>
                    {{
                      $gettext('Single-core %{ single } / Multi-core %{ multi }', {
                        single: cpu.encryption.single,
                        multi: cpu.encryption.multi
                      })
                    }}
                  </td>
                </tr>
                <tr>
                  <th>{{ $gettext('Compression/Decompression') }}</th>
                  <td>
                    {{
                      $gettext('Single-core %{ single } / Multi-core %{ multi }', {
                        single: cpu.compression.single,
                        multi: cpu.compression.multi
                      })
                    }}
                  </td>
                </tr>
                <tr>
                  <th>{{ $gettext('Physics Simulation') }}</th>
                  <td>
                    {{
                      $gettext('Single-core %{ single } / Multi-core %{ multi }', {
                        single: cpu.physics.single,
                        multi: cpu.physics.multi
                      })
                    }}
                  </td>
                </tr>
                <tr>
                  <th>{{ $gettext('JSON Parsing') }}</th>
                  <td>
                    {{
                      $gettext('Single-core %{ single } / Multi-core %{ multi }', {
                        single: cpu.json.single,
                        multi: cpu.json.multi
                      })
                    }}
                  </td>
                </tr>
              </n-table>
            </n-popover>
          </n-gi>
          <n-gi>
            <n-popover trigger="hover">
              <template #trigger>
                <n-flex vertical items-center>
                  <div v-if="memory.score !== 0">
                    <n-number-animation :from="0" :to="memory.score" show-separator />
                  </div>
                  <div v-else>{{ $gettext('Pending benchmark') }}</div>
                  <n-progress
                    type="circle"
                    :percentage="100"
                    :stroke-width="3"
                    color="var(--primary-color)"
                  >
                    <the-icon :size="50" icon="bi:memory" color="var(--primary-color)" />
                  </n-progress>
                  {{ $gettext('Memory') }}
                </n-flex>
              </template>
              <n-table :single-line="false" striped>
                <tr>
                  <th>{{ $gettext('Memory Bandwidth') }}</th>
                  <td>{{ memory.bandwidth }}</td>
                </tr>
                <tr>
                  <th>{{ $gettext('Memory Latency') }}</th>
                  <td>{{ memory.latency }}</td>
                </tr>
              </n-table>
            </n-popover>
          </n-gi>
          <n-gi>
            <n-popover trigger="hover">
              <template #trigger>
                <n-flex vertical items-center>
                  <div v-if="disk.score !== 0">
                    <n-number-animation :from="0" :to="disk.score" show-separator />
                  </div>
                  <div v-else>{{ $gettext('Pending benchmark') }}</div>
                  <n-progress
                    type="circle"
                    :percentage="100"
                    :stroke-width="3"
                    color="var(--primary-color)"
                  >
                    <the-icon :size="50" icon="bi:hdd-stack" color="var(--primary-color)" />
                  </n-progress>
                  {{ $gettext('Disk') }}
                </n-flex>
              </template>
              <n-table :single-line="false" striped>
                <tr>
                  <th>{{ $gettext('4KB Read') }}</th>
                  <td>
                    {{
                      $gettext('Speed %{ speed } / %{ iops } IOPS', {
                        speed: disk['4'].read_speed,
                        iops: disk['4'].read_iops
                      })
                    }}
                  </td>
                </tr>
                <tr>
                  <th>{{ $gettext('4KB Write') }}</th>
                  <td>
                    {{
                      $gettext('Speed %{ speed } / %{ iops } IOPS', {
                        speed: disk['4'].write_speed,
                        iops: disk['4'].write_iops
                      })
                    }}
                  </td>
                </tr>
                <tr>
                  <th>{{ $gettext('64KB Read') }}</th>
                  <td>
                    {{
                      $gettext('Speed %{ speed } / %{ iops } IOPS', {
                        speed: disk['64'].read_speed,
                        iops: disk['64'].read_iops
                      })
                    }}
                  </td>
                </tr>
                <tr>
                  <th>{{ $gettext('64KB Write') }}</th>
                  <td>
                    {{
                      $gettext('Speed %{ speed } / %{ iops } IOPS', {
                        speed: disk['64'].write_speed,
                        iops: disk['64'].write_iops
                      })
                    }}
                  </td>
                </tr>
                <tr>
                  <th>{{ $gettext('512KB Read') }}</th>
                  <td>
                    {{
                      $gettext('Speed %{ speed } / %{ iops } IOPS', {
                        speed: disk['512'].read_speed,
                        iops: disk['512'].read_iops
                      })
                    }}
                  </td>
                </tr>
                <tr>
                  <th>{{ $gettext('512KB Write') }}</th>
                  <td>
                    {{
                      $gettext('Speed %{ speed } / %{ iops } IOPS', {
                        speed: disk['512'].write_speed,
                        iops: disk['512'].write_iops
                      })
                    }}
                  </td>
                </tr>
                <tr>
                  <th>{{ $gettext('1MB Read') }}</th>
                  <td>
                    {{
                      $gettext('Speed %{ speed } / %{ iops } IOPS', {
                        speed: disk['1024'].read_speed,
                        iops: disk['1024'].read_iops
                      })
                    }}
                  </td>
                </tr>
                <tr>
                  <th>{{ $gettext('1MB Write') }}</th>
                  <td>
                    {{
                      $gettext('Speed %{ speed } / %{ iops } IOPS', {
                        speed: disk['1024'].write_speed,
                        iops: disk['1024'].write_iops
                      })
                    }}
                  </td>
                </tr>
              </n-table>
            </n-popover>
          </n-gi>
        </n-grid>
      </div>
      <n-button
        type="primary"
        size="large"
        :disabled="inTest"
        :loading="inTest"
        @click="handleTest"
        mt-40
        w-200
      >
        {{ inTest ? $gettext('Benchmarking...') : $gettext('Start Benchmark') }}
      </n-button>
    </n-flex>
  </common-page>
</template>

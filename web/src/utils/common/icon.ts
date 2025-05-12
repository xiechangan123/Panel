import { addAPIProvider, Icon } from '@iconify/vue'
import { NIcon } from 'naive-ui'

addAPIProvider('', {
  resources: ['https://iconify.cdn.haozi.net']
})

interface Props {
  size?: number
  color?: string
  class?: string
}

export function renderIcon(icon: string, props: Props = { size: 12 }) {
  return () => h(NIcon, props, { default: () => h(Icon, { icon }) })
}

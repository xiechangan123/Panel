import { icons as mdi } from '@iconify-json/mdi'
import { icons as simpleIcons } from '@iconify-json/simple-icons'
import { addCollection, Icon } from '@iconify/vue'

import { NIcon } from 'naive-ui'

addCollection(mdi)
addCollection(simpleIcons)

interface Props {
  size?: number
  color?: string
  class?: string
}

export function renderIcon(icon: string, props: Props = { size: 12 }) {
  return () => h(NIcon, props, { default: () => h(Icon, { icon }) })
}

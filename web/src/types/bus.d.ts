// 全局事件总线事件表：事件名 → payload 类型（无 payload 用 undefined）
// 新增事件必须在此登记，emit/on 的事件名和 payload 由此获得类型检查
type BusEvents = {
  'backup:refresh': undefined
  'cert:refresh-account': undefined
  'cert:refresh-async': undefined
  'cert:refresh-cert': undefined
  'cert:refresh-dns': undefined
  'database:refresh': undefined
  'database-server:refresh': undefined
  'database-user:refresh': undefined
  'file:edit': string
  'file:inline-create': boolean
  'file:keyboard-pause': undefined
  'file:keyboard-resume': undefined
  'file:refresh': undefined
  'file:search': undefined
  'project:refresh': undefined
  'ssh:refresh': undefined
  'task:refresh-cron': undefined
  'user:refresh': undefined
  'website:refresh': undefined
}

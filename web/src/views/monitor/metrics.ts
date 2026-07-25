import { useGettext } from 'vue3-gettext'

export interface MetricMeta {
  label: string
  value: string
  unit: string
  target: 'none' | 'optional' | 'required'
  placeholder: string
}

// 状态类指标语义固定为「不在运行」，不需要运算符与阈值
const statusMetrics = ['service', 'project', 'container', 'app', 'database']

export function isStatusMetric(type: string) {
  return statusMetrics.includes(type)
}

// useAlertMetrics 告警指标与运算符的可选项，供规则表单与列表共用
export function useAlertMetrics() {
  const { $gettext } = useGettext()

  const metrics = computed<MetricMeta[]>(() => [
    { label: $gettext('CPU Usage'), value: 'cpu', unit: '%', target: 'none', placeholder: '' },
    {
      label: $gettext('Memory Usage'),
      value: 'memory',
      unit: '%',
      target: 'none',
      placeholder: '',
    },
    { label: $gettext('Swap Usage'), value: 'swap', unit: '%', target: 'none', placeholder: '' },
    { label: $gettext('1 Minute Load'), value: 'load1', unit: '', target: 'none', placeholder: '' },
    {
      label: $gettext('5 Minutes Load'),
      value: 'load5',
      unit: '',
      target: 'none',
      placeholder: '',
    },
    {
      label: $gettext('15 Minutes Load'),
      value: 'load15',
      unit: '',
      target: 'none',
      placeholder: '',
    },
    {
      label: $gettext('Disk Usage'),
      value: 'disk',
      unit: '%',
      target: 'optional',
      placeholder: $gettext('Mount point, e.g. /, empty for all'),
    },
    {
      label: $gettext('Disk Inode Usage'),
      value: 'disk_inode',
      unit: '%',
      target: 'optional',
      placeholder: $gettext('Mount point, e.g. /, empty for all'),
    },
    {
      label: $gettext('Disk Read Speed'),
      value: 'disk_read',
      unit: 'MB/s',
      target: 'optional',
      placeholder: $gettext('Device name, e.g. sda, empty for all'),
    },
    {
      label: $gettext('Disk Write Speed'),
      value: 'disk_write',
      unit: 'MB/s',
      target: 'optional',
      placeholder: $gettext('Device name, e.g. sda, empty for all'),
    },
    {
      label: $gettext('Network Download Speed'),
      value: 'net_in',
      unit: 'MB/s',
      target: 'optional',
      placeholder: $gettext('Interface name, e.g. eth0, empty for all'),
    },
    {
      label: $gettext('Network Upload Speed'),
      value: 'net_out',
      unit: 'MB/s',
      target: 'optional',
      placeholder: $gettext('Interface name, e.g. eth0, empty for all'),
    },
    {
      label: $gettext('Website 5xx Responses (this hour)'),
      value: 'website_5xx',
      unit: $gettext('times'),
      target: 'optional',
      placeholder: $gettext('Website name, empty for all'),
    },
    {
      label: $gettext('Website Error Rate (this hour)'),
      value: 'website_error',
      unit: '%',
      target: 'optional',
      placeholder: $gettext('Website name, empty for all'),
    },
    {
      label: $gettext('Service Not Running'),
      value: 'service',
      unit: '',
      target: 'required',
      placeholder: $gettext('Service name, e.g. nginx'),
    },
    {
      label: $gettext('Project Not Running'),
      value: 'project',
      unit: '',
      target: 'optional',
      placeholder: $gettext('Project name, empty for all'),
    },
    {
      label: $gettext('Container Not Running'),
      value: 'container',
      unit: '',
      target: 'optional',
      placeholder: $gettext('Container name, empty for all'),
    },
    {
      label: $gettext('App Not Running'),
      value: 'app',
      unit: '',
      target: 'optional',
      placeholder: $gettext('App slug, e.g. nginx, empty for all'),
    },
    {
      label: $gettext('Database Server Unreachable'),
      value: 'database',
      unit: '',
      target: 'optional',
      placeholder: $gettext('Database server name, empty for all'),
    },
    {
      label: $gettext('Certificate Remaining Days'),
      value: 'cert_expire',
      unit: $gettext('days'),
      target: 'optional',
      placeholder: $gettext('Any domain of the certificate, empty for all'),
    },
    {
      label: $gettext('Website Remaining Days'),
      value: 'website_expire',
      unit: $gettext('days'),
      target: 'optional',
      placeholder: $gettext('Website name, empty for all'),
    },
  ])

  const operators = computed(() => [
    { label: $gettext('greater than'), value: 'gt' },
    { label: $gettext('greater than or equal to'), value: 'gte' },
    { label: $gettext('less than'), value: 'lt' },
    { label: $gettext('less than or equal to'), value: 'lte' },
  ])

  // 供 n-select 使用的精简选项
  const metricOptions = computed(() => metrics.value.map(({ label, value }) => ({ label, value })))

  const metricOf = (type: string): MetricMeta =>
    metrics.value.find((item) => item.value === type) ?? {
      label: type,
      value: type,
      unit: '',
      target: 'none',
      placeholder: '',
    }

  // conditionText 规则条件的可读文本，状态类固定为「不在运行」
  const conditionText = (rule: any) => {
    if (isStatusMetric(rule.type)) {
      return $gettext('not running')
    }
    const meta = metricOf(rule.type)
    const operator = operators.value.find((item) => item.value === rule.operator)
    return `${operator?.label ?? rule.operator} ${rule.threshold}${meta.unit ? ` ${meta.unit}` : ''}`
  }

  return { metrics, metricOptions, operators, metricOf, conditionText }
}

// 系统事件通知的可订阅事件
export function useNotifyEvents() {
  const { $gettext } = useGettext()

  return computed(() => [
    { label: $gettext('Certificate renewal failed'), value: 'cert_renew' },
    { label: $gettext('Backup failed'), value: 'backup' },
    { label: $gettext('Background task failed'), value: 'task_failed' },
    { label: $gettext('Cron task failed'), value: 'cron_failed' },
    { label: $gettext('Website expired and disabled'), value: 'website_expire' },
    { label: $gettext('Tamper protection blocked'), value: 'tamper' },
    { label: $gettext('Panel health issue'), value: 'health' },
    { label: $gettext('Panel login'), value: 'login' },
    { label: $gettext('Too many failed panel logins'), value: 'login_failed' },
    { label: $gettext('SSH login'), value: 'ssh_login' },
    { label: $gettext('SSH brute-force attempts'), value: 'ssh_bruteforce' },
  ])
}

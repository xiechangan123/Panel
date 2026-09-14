// 面板支持的 Web 服务器应用标识，用于安装检测
export const WEBSERVER_SLUGS = 'nginx,openresty,apache,openlitespeed,caddy'

// 站点相关页面按 Web 服务器差异化展示的能力，新增服务器时在此补一行
export interface WebServerFeatures {
  quic: boolean // HTTP/3 监听
  ipv6Listen: boolean // IPv6 监听
  resolver: boolean // 上游与代理的 DNS 解析器
  matchType: boolean // 代理匹配类型
  proxyCache: boolean // 代理缓存
  proxyReplaces: boolean // 代理响应内容替换
  proxyAdvanced: boolean // 代理高级配置
  rewritePresets: boolean // 伪静态预设
  stat: boolean // 访问统计
  defaultSite: boolean // 默认站点
  rateLimit: boolean // 限流限速
  realIP: boolean // 真实 IP
  lsCache: boolean // LiteSpeed 页面缓存
  upstreamAlgos: string[] // 上游负载均衡算法，空字符串为默认轮询
  lang: string // 配置文件语法高亮
}

const features: Record<string, WebServerFeatures> = {
  nginx: {
    quic: true,
    ipv6Listen: true,
    resolver: true,
    matchType: true,
    proxyCache: true,
    proxyReplaces: true,
    proxyAdvanced: true,
    rewritePresets: true,
    stat: true,
    defaultSite: true,
    rateLimit: true,
    realIP: true,
    lsCache: false,
    upstreamAlgos: ['', 'least_conn', 'ip_hash', 'hash', 'random'],
    lang: 'nginx',
  },
  apache: {
    quic: false,
    ipv6Listen: false,
    resolver: false,
    matchType: false,
    proxyCache: false,
    proxyReplaces: true,
    proxyAdvanced: false,
    rewritePresets: false,
    stat: false,
    defaultSite: false,
    rateLimit: true,
    realIP: true,
    lsCache: false,
    upstreamAlgos: ['', 'bybusyness', 'bytraffic'],
    lang: 'apacheconf',
  },
  openlitespeed: {
    quic: true,
    ipv6Listen: false,
    resolver: false,
    matchType: true,
    proxyCache: false,
    proxyReplaces: false,
    proxyAdvanced: false,
    rewritePresets: true,
    stat: false,
    defaultSite: false,
    rateLimit: false,
    realIP: false,
    lsCache: true,
    upstreamAlgos: [''],
    lang: 'plaintext',
  },
  caddy: {
    quic: false,
    ipv6Listen: false,
    resolver: false,
    matchType: true,
    proxyCache: false,
    proxyReplaces: true,
    proxyAdvanced: true,
    rewritePresets: true,
    stat: true,
    defaultSite: true,
    rateLimit: false,
    realIP: false,
    lsCache: false,
    upstreamAlgos: ['', 'least_conn', 'ip_hash', 'client_ip_hash', 'uri_hash', 'random', 'first'],
    lang: 'plaintext',
  },
}

const unknown: WebServerFeatures = {
  quic: false,
  ipv6Listen: false,
  resolver: false,
  matchType: false,
  proxyCache: false,
  proxyReplaces: false,
  proxyAdvanced: false,
  rewritePresets: false,
  stat: false,
  defaultSite: false,
  rateLimit: false,
  realIP: false,
  lsCache: false,
  upstreamAlgos: [''],
  lang: 'plaintext',
}

// 取 Web 服务器能力，未知类型全部降级
export function webserverFeatures(webserver: string): WebServerFeatures {
  return features[webserver] ?? unknown
}

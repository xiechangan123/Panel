// 配置调优预设生成
export interface TuneProfile {
  memoryMB: number
  cpuCores: number
  disk: 'hdd' | 'ssd' | 'nvme'
  scenario: string
  connections: number
}

export interface SizePair {
  num: number
  unit: string
}

// MB 数值转 num+unit
export const mbToPair = (mb: number, units: [string, string] = ['MB', 'GB']): SizePair => {
  if (mb >= 1024 && mb % 1024 === 0) {
    return { num: mb / 1024, unit: units[1] }
  }
  return { num: mb, unit: units[0] }
}

// kB 数值转 num+unit
export const kbToPair = (kb: number, units: [string, string] = ['kB', 'MB']): SizePair => {
  if (kb >= 1024 && kb % 1024 === 0) {
    return { num: kb / 1024, unit: units[1] }
  }
  return { num: kb, unit: units[0] }
}

// PostgreSQL
export const postgresqlPreset = (p: TuneProfile) => {
  const sharedBuffersMB = Math.floor(p.memoryMB / 4)
  const maintenanceDivisor = p.scenario === 'dw' ? 8 : 16
  const walSizes: Record<string, [number, number]> = {
    web: [1, 4],
    oltp: [2, 8],
    dw: [4, 16],
    mixed: [1, 4],
  }
  const [minWal, maxWal] = walSizes[p.scenario] ?? [1, 4]
  let workMemKB = Math.floor(((p.memoryMB - sharedBuffersMB) * 1024) / (p.connections * 3))
  if (p.scenario === 'dw') workMemKB = Math.floor(workMemKB / 2)

  return {
    maxConnections: p.connections,
    sharedBuffers: mbToPair(sharedBuffersMB),
    effectiveCacheSize: mbToPair(Math.floor((p.memoryMB * 3) / 4)),
    maintenanceWorkMem: mbToPair(Math.min(Math.floor(p.memoryMB / maintenanceDivisor), 2048)),
    workMemKB: Math.max(workMemKB, 64),
    walBuffersKB: Math.min(Math.floor((sharedBuffersMB * 1024 * 3) / 100), 16384),
    minWalSizeGB: minWal,
    maxWalSizeGB: maxWal,
    checkpointCompletionTarget: '0.9',
    defaultStatisticsTarget: p.scenario === 'dw' ? 500 : 100,
    randomPageCost: p.disk === 'hdd' ? '4.0' : '1.1',
    effectiveIoConcurrency: p.disk === 'hdd' ? 2 : p.disk === 'ssd' ? 200 : 300,
    hugePages: p.memoryMB >= 32768 ? 'try' : 'off',
  }
}

// MySQL
export const mysqlPreset = (p: TuneProfile) => {
  const mem = p.memoryMB
  const mg: [string, string] = ['M', 'G']
  const bufferPoolMB = Math.max(Math.floor((mem * 0.55) / 128) * 128, 128)
  const connections = Math.min(Math.max(Math.ceil(mem / 12 / 50) * 50, 100), 1000)
  // 小 / 中 / 大三档：<2G / <8G / 其余
  const pick = (small: number, medium: number, large: number) =>
    mem < 2048 ? small : mem < 8192 ? medium : large

  return {
    maxConnections: connections,
    tableOpenCache: pick(1024, 4000, 8000),
    threadCacheSize: Math.min(Math.max(Math.floor(connections / 8), 8), 100),
    keyBufferSize: mbToPair(pick(8, 16, 32), mg),
    sortBufferSizeK: pick(512, 1024, 2048),
    readBufferSizeK: pick(256, 512, 1024),
    readRndBufferSizeK: pick(512, 512, 1024),
    joinBufferSizeK: pick(512, 512, 2048),
    threadStackK: 512,
    myisamSortBufferSize: mbToPair(pick(8, 16, 32), mg),
    tmpTableSize: mbToPair(pick(32, 64, 256), mg),
    maxHeapTableSize: mbToPair(pick(32, 64, 256), mg),
    innodbBufferPoolSize: mbToPair(bufferPoolMB, mg),
    innodbLogBufferSize: mbToPair(pick(16, 32, 64), mg),
    innodbReadIoThreads: Math.max(Math.floor(p.cpuCores / 2), 1),
    innodbWriteIoThreads: Math.max(Math.floor(p.cpuCores / 2), 1),
  }
}

// Redis
export const redisPreset = (p: TuneProfile) => {
  const policies: Record<string, string> = {
    cache: 'allkeys-lru',
    mixed: 'volatile-lru',
    persistent: 'noeviction',
  }

  return {
    maxmemoryMB: Math.floor(p.memoryMB * 0.7),
    maxmemoryPolicy: policies[p.scenario] ?? 'noeviction',
    appendonly: p.scenario === 'cache' ? 'no' : 'yes',
    appendfsync: 'everysec',
  }
}

// Nginx
export const nginxPreset = (p: TuneProfile) => {
  return {
    workerProcesses: 'auto',
    workerConnections: p.memoryMB < 2048 ? 10240 : 65535,
    keepaliveTimeout: 60,
    gzip: 'on',
    gzipCompLevel: 5,
    brotli: 'on',
    brotliCompLevel: 5,
    brotliStatic: 'on',
    zstd: 'on',
    zstdCompLevel: 5,
    zstdStatic: 'on',
  }
}

// PHP-FPM
export const phpPreset = (p: TuneProfile) => {
  const perProcessMB = Number(p.scenario) || 50
  const maxChildren = Math.max(Math.floor(p.memoryMB / perProcessMB), 4)

  return {
    pm: 'dynamic',
    pmMaxChildren: maxChildren,
    pmStartServers: Math.max(Math.floor(maxChildren / 4), 2),
    pmMinSpareServers: Math.max(Math.floor(maxChildren / 8), 2),
    pmMaxSpareServers: Math.max(Math.floor(maxChildren / 2), 4),
    memoryLimitM: perProcessMB >= 80 ? 512 : 256,
  }
}

// Apache
export const apachePreset = (p: TuneProfile) => {
  const threadsPerChild = 25
  const workers = Math.max(
    Math.floor(p.memoryMB / 8 / threadsPerChild) * threadsPerChild,
    threadsPerChild * 2,
  )

  return {
    startServers: Math.min(Math.max(Math.floor(p.cpuCores / 2), 2), 4),
    threadsPerChild,
    maxRequestWorkers: Math.min(workers, 1600),
    minSpareThreads: threadsPerChild,
    maxSpareThreads: threadsPerChild * 3,
    maxConnectionsPerChild: 0,
    timeout: 60,
    keepAlive: 'On',
    maxKeepAliveRequests: 1000,
    keepAliveTimeout: 5,
  }
}

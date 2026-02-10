interface CpuInfoStat {
  cpu: number
  vendorId: string
  family: string
  model: string
  stepping: number
  physicalId: string
  coreId: string
  cores: number
  modelName: string
  mhz: number
  cacheSize: number
  flags: string[]
  microcode: string
}

interface LoadAvgStat {
  load1: number
  load5: number
  load15: number
}

interface HostInfoStat {
  hostname: string
  uptime: number
  bootTime: number
  procs: number
  os: string
  platform: string
  platformFamily: string
  platformVersion: string
  kernelVersion: string
  kernelArch: string
  virtualizationSystem: string
  virtualizationRole: string
  hostid: string
}

interface VirtualMemoryStat {
  total: number
  available: number
  used: number
  usedPercent: number
  free: number
  active: number
  inactive: number
  wired: number
  laundry: number
  buffers: number
  cached: number
  writeBack: number
  dirty: number
  writeBackTmp: number
  shared: number
  slab: number
  sreclaimable: number
  sunreclaim: number
  pageTables: number
  swapCached: number
  commitLimit: number
  committedAS: number
  highTotal: number
  highFree: number
  lowTotal: number
  lowFree: number
  swapTotal: number
  swapFree: number
  mapped: number
  vmallocTotal: number
  vmallocUsed: number
  vmallocChunk: number
  hugePagesTotal: number
  hugePagesFree: number
  hugePagesRsvd: number
  hugePagesSurp: number
  hugePageSize: number
  anonHugePages: number
}

interface SwapMemoryStat {
  total: number
  used: number
  free: number
  usedPercent: number
  sin: number
  sout: number
  pgin: number
  pgout: number
  pgfault: number
  pgmajfault: number
}

interface IOCountersStat {
  name: string
  bytesSent: number
  bytesRecv: number
  packetsSent: number
  packetsRecv: number
  errin: number
  errout: number
  dropin: number
  dropout: number
  fifoin: number
  fifoout: number
}

interface DiskIOCountersStat {
  readCount: number
  mergedReadCount: number
  writeCount: number
  mergedWriteCount: number
  readBytes: number
  writeBytes: number
  readTime: number
  writeTime: number
  iopsInProgress: number
  ioTime: number
  weightedIO: number
  name: string
  serialNumber: string
  label: string
}

interface PartitionStat {
  device: string
  mountpoint: string
  fstype: string
  opts: string
}

interface DiskUsageStat {
  path: string
  fstype: string
  total: number
  free: number
  used: number
  usedPercent: number
  inodesTotal: number
  inodesUsed: number
  inodesFree: number
  inodesUsedPercent: number
}

export interface Realtime {
  cpus: CpuInfoStat[]
  percent: number
  percents: number[]
  load: LoadAvgStat
  host: HostInfoStat
  mem: VirtualMemoryStat
  swap: SwapMemoryStat
  net: IOCountersStat[]
  disk_io: DiskIOCountersStat[]
  disk: PartitionStat[]
  disk_usage: DiskUsageStat[]
}

export interface ProcessStat {
  pid: number
  name: string
  username: string
  command: string
  value: number
  read?: number
  write?: number
}

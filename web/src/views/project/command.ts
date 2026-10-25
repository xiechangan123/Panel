// 各语言运行时的命令名，面板安装的版本带版本号后缀，如 go1.25、java21、node22
export const runtimes: Record<string, string> = {
  go: 'go',
  java: 'java',
  nodejs: 'node',
  php: 'php',
  python: 'python',
  dotnet: 'dotnet',
}

// 由选中的文件生成启动命令。runtime 为空表示文件本身可执行，systemd 要求写绝对路径；
// 否则交给运行时执行，工作目录内的文件写相对路径
export function buildCommand(type: string, runtime: string, file: string, workDir: string) {
  if (!runtime) return file

  const dir = workDir.replace(/\/+$/, '')
  const target = file.startsWith(dir + '/') ? file.slice(dir.length + 1) : file
  if (type === 'go') return `${runtime} run ${target}`
  if (type === 'java' && /\.(jar|war)$/.test(target)) return `${runtime} -jar ${target}`
  return `${runtime} ${target}`
}

// 编辑时不知道选的版本，沿用现有命令开头的运行时；Go 项目开头不是 go 时按二进制处理
export function currentRuntime(type: string, command: string) {
  const name = runtimes[type]
  if (!name) return ''

  const first = command.trim().split(/\s+/)[0] ?? ''
  if (new RegExp(`^${name}[\\d.]*$`).test(first)) return first
  return type === 'go' ? '' : name
}

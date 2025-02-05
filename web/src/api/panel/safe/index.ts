import { request } from '@/utils'

export default {
  // 获取SSH
  ssh: (): any => request.get('/safe/ssh'),
  // 设置SSH
  setSsh: (status: boolean, port: number): any => request.post('/safe/ssh', { status, port }),
  // 获取Ping状态
  pingStatus: (): any => request.get('/safe/ping'),
  // 设置Ping状态
  setPingStatus: (status: boolean): any => request.post('/safe/ping', { status })
}

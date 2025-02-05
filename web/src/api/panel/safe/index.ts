import { http } from '@/utils'

export default {
  ssh: (): any => http.Get('/safe/ssh'),
  updateSsh: (status: boolean, port: number): any => http.Post('/safe/ssh', { status, port }),
  pingStatus: (): any => http.Get('/safe/ping'),
  updatePingStatus: (status: boolean): any => http.Post('/safe/ping', { status })
}

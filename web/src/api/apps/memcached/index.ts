import { http } from '@/utils'

export default {
  getLoad: (): any => http.Get('/apps/memcached/load'),
  getConfig: (): any => http.Get('/apps/memcached/config'),
  updateConfig: (config: string): any => http.Post('/apps/memcached/config', { config })
}

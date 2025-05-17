import { http } from '@/utils'

export default {
  load: (): any => http.Get('/apps/memcached/load'),
  config: (): any => http.Get('/apps/memcached/config'),
  updateConfig: (config: string): any => http.Post('/apps/memcached/config', { config })
}

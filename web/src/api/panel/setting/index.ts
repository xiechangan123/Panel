import { http } from '@/utils'

export default {
  // 获取设置
  list: (): any => http.Get('/setting'),
  // 保存设置
  update: (settings: any): any => http.Post('/setting', settings)
}

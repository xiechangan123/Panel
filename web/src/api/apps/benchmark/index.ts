import { http } from '@/utils'

export default {
  // 运行评分
  test: (name: string, multi: boolean): any => http.Post('/apps/benchmark/test', { name, multi })
}

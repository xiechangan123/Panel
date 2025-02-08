import { resolveResError } from '@/utils/http/helpers'
import type { AlovaXHRResponse } from '@alova/adapter-xhr'
import { xhrRequestAdapter } from '@alova/adapter-xhr'
import { createAlova, Method } from 'alova'
import VueHook from 'alova/vue'

export const http = createAlova({
  id: 'panel',
  cacheFor: null,
  statesHook: VueHook,
  requestAdapter: xhrRequestAdapter(),
  baseURL: import.meta.env.VITE_BASE_API,
  responded: {
    onSuccess: async (response: AlovaXHRResponse, method: Method) => {
      const ct = response.headers['Content-Type'] || response.headers['content-type']
      let json
      try {
        if (ct && ct.includes('application/json')) {
          json = typeof response.data === 'string' ? JSON.parse(response.data) : response.data
        } else {
          json = { code: response.status, message: response.data }
        }
      } catch (error) {
        json = { code: response.status, message: 'JSON 解析失败' }
      }
      const { status, statusText } = response
      const { meta } = method
      if (status !== 200) {
        const code = json?.code ?? status
        console.log(json)
        const message = resolveResError(
          code,
          json?.message && json.message.trim() !== '' ? json.message : statusText
        )
        const noAlert = meta?.noAlert
        if (!noAlert) {
          if (code === 422) {
            window.$message.error(message)
          } else if (code !== 401) {
            window.$dialog.error({
              title: '错误',
              content: message,
              maskClosable: false
            })
          }
        }
        throw new Error(message)
      }
      return json.data
    },
    onError: (error: any, method: Method) => {
      const { code, message } = error
      const { meta } = method
      const errorMessage = resolveResError(code, message)
      const noAlert = meta?.noAlert

      if (!noAlert) {
        window.$dialog.error({
          title: '接口请求失败',
          content: errorMessage,
          maskClosable: false
        })
      }

      throw error
    }
  }
})

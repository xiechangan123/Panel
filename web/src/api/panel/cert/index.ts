import { request } from '@/utils'

export default {
  // CA 供应商列表
  caProviders: (): any => request.get('/cert/caProviders'),
  // DNS 供应商列表
  dnsProviders: (): any => request.get('/cert/dnsProviders'),
  // 证书算法列表
  algorithms: (): any => request.get('/cert/algorithms'),
  // ACME 账号列表
  accounts: (page: number, limit: number): any =>
    request.get('/cert/account', { params: { page, limit } }),
  // ACME 账号详情
  accountInfo: (id: number): any => request.get(`/cert/account/${id}`),
  // ACME 账号添加
  accountCreate: (data: any): any => request.post('/cert/account', data),
  // ACME 账号更新
  accountUpdate: (id: number, data: any): any => request.put(`/cert/account/${id}`, data),
  // ACME 账号删除
  accountDelete: (id: number): any => request.delete(`/cert/account/${id}`),
  // DNS 记录列表
  dns: (page: number, limit: number): any => request.get('/cert/dns', { params: { page, limit } }),
  // DNS 记录详情
  dnsInfo: (id: number): any => request.get(`/cert/dns/${id}`),
  // DNS 记录添加
  dnsCreate: (data: any): any => request.post('/cert/dns', data),
  // DNS 记录更新
  dnsUpdate: (id: number, data: any): any => request.put(`/cert/dns/${id}`, data),
  // DNS 记录删除
  dnsDelete: (id: number): any => request.delete(`/cert/dns/${id}`),
  // 证书列表
  certs: (page: number, limit: number): any =>
    request.get('/cert/cert', { params: { page, limit } }),
  // 证书详情
  certInfo: (id: number): any => request.get(`/cert/cert/${id}`),
  // 证书上传
  certUpload: (data: any): any => request.post('/cert/cert/upload', data),
  // 证书添加
  certCreate: (data: any): any => request.post('/cert/cert', data),
  // 证书更新
  certUpdate: (id: number, data: any): any => request.put(`/cert/cert/${id}`, data),
  // 证书删除
  certDelete: (id: number): any => request.delete(`/cert/cert/${id}`),
  // 证书自动签发
  obtainAuto: (id: number): any => request.post(`/cert/cert/${id}/obtainAuto`, { id }),
  // 证书手动签发
  obtainManual: (id: number): any => request.post(`/cert/cert/${id}/obtainManual`, { id }),
  // 证书自签名签发
  obtainSelfSigned: (id: number): any => request.post(`/cert/cert/${id}/obtainSelfSigned`, { id }),
  // 续签
  renew: (id: number): any => request.post(`/cert/cert/${id}/renew`, { id }),
  // 获取 DNS 记录
  manualDNS: (id: number): any => request.post(`/cert/cert/${id}/manualDNS`, { id }),
  // 部署
  deploy: (id: number, website_id: number): any =>
    request.post(`/cert/cert/${id}/deploy`, { id, website_id })
}

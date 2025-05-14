import { http } from '@/utils'

export default {
  // 获取数据库列表
  list: (page: number, limit: number) => http.Get(`/database`, { params: { page, limit } }),
  // 创建数据库
  create: (data: any) => http.Post(`/database`, data),
  // 删除数据库
  delete: (server_id: number, name: string) => http.Delete(`/database`, { server_id, name }),
  // 更新评论
  comment: (server_id: number, name: string, comment: string) =>
    http.Post(`/database/comment`, { server_id, name, comment }),
  // 获取数据库服务器列表
  serverList: (page: number, limit: number) =>
    http.Get('/database_server', { params: { page, limit } }),
  // 创建数据库服务器
  serverCreate: (data: any) => http.Post('/database_server', data),
  // 获取数据库服务器
  serverGet: (id: number) => http.Get(`/database_server/${id}`),
  // 更新数据库服务器
  serverUpdate: (id: number, data: any) => http.Put(`/database_server/${id}`, data),
  // 删除数据库服务器
  serverDelete: (id: number) => http.Delete(`/database_server/${id}`),
  // 同步数据库
  serverSync: (id: number) => http.Post(`/database_server/${id}/sync`),
  // 更新服务器备注
  serverRemark: (id: number, remark: string) =>
    http.Put(`/database_server/${id}/remark`, { remark }),
  // 获取数据库用户列表
  userList: (page: number, limit: number) =>
    http.Get('/database_user', { params: { page, limit } }),
  // 创建数据库用户
  userCreate: (data: any) => http.Post('/database_user', data),
  // 获取数据库用户
  userGet: (id: number) => http.Get(`/database_user/${id}`),
  // 更新数据库用户
  userUpdate: (id: number, data: any) => http.Put(`/database_user/${id}`, data),
  // 删除数据库用户
  userDelete: (id: number) => http.Delete(`/database_user/${id}`),
  // 更新用户备注
  userRemark: (id: number, remark: string) => http.Put(`/database_user/${id}/remark`, { remark })
}

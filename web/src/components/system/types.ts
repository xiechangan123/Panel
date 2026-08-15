// 分页取数契约，由 useTableQuery 使用
export interface PaginationFetchParams {
  page: number
  pageSize: number
  query: Record<string, any>
}

export interface PaginationFetchResult<T = any> {
  items: T[]
  total: number
}

export type FetchFn<T = any> = (params: PaginationFetchParams) => Promise<PaginationFetchResult<T>>

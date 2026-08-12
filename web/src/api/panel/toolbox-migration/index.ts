import { http } from '@/utils'

export type MigrationPanel = 'acepanel' | 'baota' | 'onepanel'
export type MigrationResourceType =
  | 'website'
  | 'database'
  | 'database_user'
  | 'project'
  | 'container'
  | 'compose'

export interface MigrationConnection {
  source_panel: MigrationPanel
  url: string
  token_id?: number
  token?: string
  api_key?: string
}

export interface MigrationResource {
  key: string
  type: MigrationResourceType
  subtype: string
  name: string
  status: string
  size: number
  supported: boolean
  blockers: string[]
  warnings: string[]
  features: string[]
  depends_on: string[]
  target_name: string
  target_path: string
  runtime_version: string
}

export interface MigrationSelection {
  key: string
  target_path?: string
  target_user?: string
}

export interface MigrationResult {
  key: string
  type: MigrationResourceType | string
  name: string
  status: 'pending' | 'running' | 'success' | 'partial' | 'failed' | 'skipped'
  stage:
    | 'preparing'
    | 'downloading'
    | 'validating'
    | 'importing'
    | 'configuring'
    | 'starting'
    | 'done'
  error: string
  warnings: string[]
  created_resources: string[]
  residual_resources: string[]
  started_at: string | null
  ended_at: string | null
  duration: number
}

export default {
  status: (): any => http.Get('/toolbox_migration/status'),
  precheck: (data: MigrationConnection): any => http.Post('/toolbox_migration/precheck', data),
  items: (): any => http.Get('/toolbox_migration/items'),
  start: (data: {
    items: MigrationSelection[]
    skip_incompatible_items: boolean
    stop_source_during_backup: boolean
  }): any => http.Post('/toolbox_migration/start', data),
  reset: (): any => http.Post('/toolbox_migration/reset'),
  results: (): any => http.Get('/toolbox_migration/results'),
  logUrl: '/api/toolbox_migration/log',
}

import { http } from '@/utils'

export type MigrationPanel = 'acepanel' | 'baota' | 'onepanel'
export type MigrationResourceType = 'website' | 'database' | 'database_user' | 'project'
export type MigrationStep = 'idle' | 'precheck' | 'select' | 'running' | 'done'
export type MigrationStatus = 'pending' | 'running' | 'success' | 'partial' | 'failed' | 'skipped'
export type MigrationStage = 'backup' | 'transfer' | 'import' | 'done'

export interface MigrationConnection {
  source_panel: MigrationPanel
  url: string
  token_id?: number
  token?: string
  api_key?: string
}

export interface MigrationSource {
  panel: string
  version: string
}

export interface MigrationResource {
  key: string
  type: MigrationResourceType
  subtype: string
  name: string
  status: string
  size: number
  blockers: string[]
  warnings: string[]
  depends_on: string[]
  target_name: string
  target_path: string
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
  status: MigrationStatus
  stage: MigrationStage
  error: string
  warnings: string[]
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
    skip_blocked: boolean
    stop_source: boolean
  }): any => http.Post('/toolbox_migration/start', data),
  reset: (): any => http.Post('/toolbox_migration/reset'),
  logUrl: '/api/toolbox_migration/log'
}

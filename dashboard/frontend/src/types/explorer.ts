import type { BrowseItem } from './models'

/**
 * Types mirroring Go backend models in internal/models/preview.go,
 * internal/models/api_explorer.go, and internal/models/operations.go.
 */

export interface FilePreviewResponse {
  remote: string
  path: string
  name: string
  size: number
  mime_type: string
  is_binary: boolean
  is_pdf?: boolean
  content?: string
  data_url?: string
  truncated: boolean
  error?: string
  is_duplicate_collision?: boolean
  collision_count?: number
}

export interface CompareItemRequest {
  remote: string
  path: string
  profile_id?: string
}

export interface CompareItemResult {
  remote: string
  path: string
  name: string
  size: number
  mod_time: string
  hashes?: Record<string, string>
  primary_hash?: string
  error?: string
}

export interface FileCompareResponse {
  status: string
  matches: boolean
  algorithm: string
  clusters: number[][]
  items: CompareItemResult[]
  error?: string
}

export interface CopyRequest {
  src_remote: string
  src_path: string
  dst_remote: string
  dst_path: string
  is_dir: boolean
  profile_id?: string
}

export interface CopyResponse {
  status: string
  msg?: string
  src?: string
  dst?: string
  error?: string
}

export interface MoveRequest {
  profile_id?: string
  src_profile_id?: string
  dst_profile_id?: string
  src_remote: string
  src_path: string
  dst_remote: string
  dst_path: string
  is_dir: boolean
}

export interface MoveResponse {
  status: string
  msg?: string
  src?: string
  dst?: string
  error?: string
}

export interface MkdirRequest {
  profile_id?: string
  remote: string
  path: string
}

export interface MkdirResponse {
  status: string
  msg?: string
  remote?: string
  path?: string
  error?: string
}

export interface DeleteRequest {
  remote: string
  path: string
  is_dir: boolean
  profile_id?: string
}

export interface DeleteResponse {
  status: string
  msg?: string
  remote?: string
  path?: string
  fs?: string
  error?: string
}

export interface UploadQueueItem {
  id: string
  profile_id: string
  remote: string
  path: string
  file_name: string
  size: number
  status: 'queued' | 'waiting_for_sync' | 'uploading' | 'completed' | 'failed' | string
  error?: string
  created_at?: string
}

export interface UploadDockStatus {
  pending_count: number
  is_sync_active: boolean
  items: UploadQueueItem[]
}

export const PREVIEWABLE_EXTENSIONS = new Set([
  '.txt', '.md', '.markdown', '.log', '.conf', '.ini', '.env', '.csv',
  '.json', '.yaml', '.yml', '.xml', '.html', '.htm', '.css', '.js', '.ts',
  '.go', '.sh', '.bash', '.py', '.rs', '.sql',
  '.png', '.jpg', '.jpeg', '.gif', '.webp', '.svg', '.bmp', '.ico', '.pdf',
])

export interface BreadcrumbItem {
  name: string
  path: string
  title?: string
  isEllipsis?: boolean
}

export type ExplorerSortField = 'name' | 'type' | 'mtime' | 'size'

export function isItemPreviewable(item: { Name?: string; IsDir?: boolean } | null | undefined): boolean {
  if (!item || item.IsDir) return false
  const name = (item.Name || '').toLowerCase()
  const dotIndex = name.lastIndexOf('.')
  if (dotIndex === -1) return false
  const ext = name.slice(dotIndex)
  return PREVIEWABLE_EXTENSIONS.has(ext)
}


export interface ExplorerDragPayload {
  item: BrowseItem
  sourcePane: 'left' | 'right'
  sourceRemote: string
  sourcePath: string
}

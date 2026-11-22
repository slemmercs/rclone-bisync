export interface HasherWrapRequest {
  name: string
  remote: string
  hashes?: string[] | string
  max_age?: string
  auto_size?: number
  create_profile?: boolean
  local_path?: string
  schedule?: string
}

export interface HasherWrapResponse {
  status: string
  name: string
  remote: string
  hashes?: string[] | string
  max_age?: string
  message?: string
  error?: string
}

export type HasherCommandName = 'dump' | 'fulldump' | 'drop' | 'import' | 'stickyimport'

export interface HasherCommandRequest {
  remote: string
  command: HasherCommandName | string
  args?: string[]
}

export interface HasherCommandResponse {
  status: string
  command?: string
  remote?: string
  result?: any
  output?: string
  error?: string
}

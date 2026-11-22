export interface ProviderOption {
  Name?: string
  Help?: string
  Provider?: string
  Type?: string
  Default?: any
  Required?: boolean
  IsPassword?: boolean
  Advanced?: boolean
  Hide?: number
  Examples?: Array<{ Value: string; Help: string }>
  // Lowercase alternatives matching Go internal/models/provider.go JSON tags
  name?: string
  help?: string
  provider?: string
  type?: string
  default?: any
  value?: string
  required?: boolean
  is_password?: boolean
  advanced?: boolean
  hide?: number
  examples?: Array<{ value: string; help: string }>
}

export interface ProviderCatalogItem {
  name: string
  prefix?: string
  description?: string
  category: 'popular' | 'cloud' | 'object' | 'protocol' | 'virtual' | 'other'
  options?: ProviderOption[]
  oauth_supported?: boolean
  supports_oauth?: boolean
}

export interface ProviderCatalogResponse {
  providers: ProviderCatalogItem[]
  categories: Record<string, string[]>
  popular: string[]
}

export interface CreateRemotePayload {
  name: string
  type: string
  parameters: Record<string, any>
  create_profile?: boolean
  local_path?: string
  profile_name?: string
  schedule?: string
  remote_subpath?: string
  // Virtual encrypted overlay support
  encrypt_remote?: boolean
  encrypted_overlay?: boolean
  crypt_password?: string
  crypt_password2?: string
  crypt_filename_encryption?: 'standard' | 'obfuscate' | 'off'
  crypt_directory_name_encryption?: boolean
  // Hasher acceleration overlay support
  accelerate_hasher?: boolean
  hasher_hashes?: string[]
}

export interface WizardConfigForm {
  remoteName: string
  authMode: 'browser' | 'custom'
  customClientId: string
  customClientSecret: string
  providerParams: Record<string, any>
  encryptRemote: boolean
  cryptPassword: string
  cryptPassword2: string
  cryptFilenameEncryption: 'standard' | 'obfuscate' | 'off'
  accelerateHasher: boolean
  hasherHashes: string[]
  createProfile: boolean
  localPath: string
  remoteSubpath: string
  schedule: string
}

export function isOAuthProvider(p: ProviderCatalogItem | null | undefined): boolean {
  if (!p) return false
  if (p.supports_oauth || (p as any).oauth_supported) return true
  const oauthTypes = ['drive', 'onedrive', 'dropbox', 'box', 'pcloud', 'hubic', 'yandex', 'mailru', 'google photos', 'zoho']
  const name = (p.name || '').toLowerCase()
  const prefix = (p.prefix || '').toLowerCase()
  return oauthTypes.includes(name) || oauthTypes.includes(prefix)
}

export function isVirtualProvider(p: ProviderCatalogItem | string | null | undefined): boolean {
  if (!p) return false
  const name = typeof p === 'string' ? p.toLowerCase() : (p.name || p.prefix || '').toLowerCase()
  return ['crypt', 'hasher', 'alias', 'chunker', 'compress', 'union', 'combine'].includes(name)
}

export function isBisyncIncompatible(p: ProviderCatalogItem | string | null | undefined): boolean {
  if (!p) return false
  const name = typeof p === 'string' ? p.toLowerCase() : (p.name || p.prefix || '').toLowerCase()
  return ['union', 'combine'].includes(name)
}

export function isBucketBasedProvider(p: ProviderCatalogItem | string | null | undefined): boolean {
  if (!p) return false
  const name = typeof p === 'string' ? p.toLowerCase() : (p.name || p.prefix || '').toLowerCase()
  return ['s3', 'b2', 'gcs', 'azureblob', 'swift'].includes(name)
}


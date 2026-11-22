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
}

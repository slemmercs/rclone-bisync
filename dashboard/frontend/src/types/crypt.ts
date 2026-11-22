export interface CryptWrapRequest {
  name: string
  remote: string
  password: string
  password2?: string
  filename_encryption?: 'standard' | 'obfuscate' | 'off'
  directory_name_encryption?: boolean
}

export interface CryptWrapResponse {
  status: string
  name: string
  remote: string
  filename_encryption?: string
  directory_name_encryption?: boolean
  message?: string
  error?: string
}

/**
 * Centralized API Client for Bisync Hub Cockpit
 * Guarantees CSRF header 'X-Dashboard-Request: 1' and consistent JSON handling
 */

export class ApiError extends Error {
  status: number
  data: any

  constructor(message: string, status: number, data?: any) {
    super(message)
    this.name = 'ApiError'
    this.status = status
    this.data = data
  }
}

async function request<T = any>(endpoint: string, options: RequestInit = {}): Promise<T> {
  const headers = new Headers(options.headers || {})
  headers.set('X-Dashboard-Request', '1')
  if (!headers.has('Content-Type') && options.body && typeof options.body === 'string') {
    headers.set('Content-Type', 'application/json')
  }

  const res = await fetch(endpoint, {
    ...options,
    headers
  })

  const contentType = res.headers.get('content-type') || ''
  let data: any = null

  if (contentType.includes('application/json')) {
    try {
      data = await res.json()
    } catch (_) {
      data = null
    }
  } else {
    try {
      data = await res.text()
    } catch (_) {
      data = null
    }
  }

  if (!res.ok) {
    const message =
      (data && typeof data === 'object' && (data.error || data.msg || data.message)) ||
      (typeof data === 'string' && data.trim()) ||
      `HTTP ${res.status}: ${res.statusText}`
    throw new ApiError(message, res.status, data)
  }

  return data as T
}

export const api = {
  get<T = any>(url: string, headers?: Record<string, string>): Promise<T> {
    return request<T>(url, { method: 'GET', headers })
  },

  post<T = any>(url: string, body?: any, headers?: Record<string, string>): Promise<T> {
    return request<T>(url, {
      method: 'POST',
      headers,
      body: body !== undefined ? JSON.stringify(body) : undefined
    })
  },

  put<T = any>(url: string, body?: any, headers?: Record<string, string>): Promise<T> {
    return request<T>(url, {
      method: 'PUT',
      headers,
      body: body !== undefined ? JSON.stringify(body) : undefined
    })
  },

  delete<T = any>(url: string, headers?: Record<string, string>): Promise<T> {
    return request<T>(url, { method: 'DELETE', headers })
  }
}

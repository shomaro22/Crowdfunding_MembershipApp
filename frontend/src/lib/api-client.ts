import {
  clearAccessToken,
  getAccessToken,
  setAccessToken,
} from '@/lib/token-store'
import { isApiErrorBody, isRecord } from '@/types/api'

export class ApiError extends Error {
  readonly status: number
  readonly code: string

  constructor(status: number, code: string, message: string) {
    super(message)
    this.name = 'ApiError'
    this.status = status
    this.code = code
  }
}

export interface ApiRequestOptions {
  method?: 'GET' | 'POST' | 'PUT' | 'PATCH' | 'DELETE'
  body?: unknown
  auth?: boolean
  /** 401 時の refresh 再試行をスキップ */
  skipRefresh?: boolean
  signal?: AbortSignal
}

const DEFAULT_TIMEOUT_MS = 30_000

export function apiBaseUrl(): string {
  const raw = import.meta.env.VITE_API_BASE_URL
  if (typeof raw !== 'string' || raw.trim() === '') {
    return ''
  }
  return raw.replace(/\/+$/, '')
}

async function parseJsonSafe(response: Response): Promise<unknown> {
  const text = await response.text()
  if (text === '') {
    return null
  }
  try {
    return JSON.parse(text) as unknown
  } catch {
    throw new ApiError(response.status, 'INVALID_JSON', 'Invalid JSON response')
  }
}

function errorFromResponse(status: number, payload: unknown): ApiError {
  if (isApiErrorBody(payload)) {
    return new ApiError(status, payload.code, payload.error)
  }
  return new ApiError(status, 'UNKNOWN', 'Request failed')
}

let refreshPromise: Promise<string | null> | null = null

async function refreshAccessToken(): Promise<string | null> {
  if (refreshPromise) {
    return refreshPromise
  }

  refreshPromise = (async () => {
    const controller = new AbortController()
    const timer = window.setTimeout(() => controller.abort(), DEFAULT_TIMEOUT_MS)

    try {
      const response = await fetch(`${apiBaseUrl()}/auth/refresh`, {
        method: 'POST',
        credentials: 'include',
        signal: controller.signal,
        headers: { Accept: 'application/json' },
      })

      if (!response.ok) {
        clearAccessToken()
        return null
      }

      if (response.status === 204) {
        clearAccessToken()
        return null
      }

      const payload = await parseJsonSafe(response)
      if (!isRecord(payload) || typeof payload.auth_token !== 'string') {
        clearAccessToken()
        return null
      }

      setAccessToken(payload.auth_token)
      return payload.auth_token
    } catch {
      clearAccessToken()
      return null
    } finally {
      window.clearTimeout(timer)
      refreshPromise = null
    }
  })()

  return refreshPromise
}

export async function apiRequest<T>(
  path: string,
  options: ApiRequestOptions = {},
): Promise<T> {
  const {
    method = 'GET',
    body,
    auth = false,
    skipRefresh = false,
    signal,
  } = options

  const controller = new AbortController()
  const timer = window.setTimeout(() => controller.abort(), DEFAULT_TIMEOUT_MS)

  const headers = new Headers({ Accept: 'application/json' })
  if (body !== undefined) {
    headers.set('Content-Type', 'application/json')
  }

  if (auth) {
    const token = getAccessToken()
    if (token) {
      headers.set('Authorization', `Bearer ${token}`)
    }
  }

  const abortFromCaller = (): void => {
    controller.abort()
  }
  signal?.addEventListener('abort', abortFromCaller)

  try {
    let response = await fetch(`${apiBaseUrl()}${path}`, {
      method,
      credentials: 'include',
      headers,
      body: body === undefined ? undefined : JSON.stringify(body),
      signal: controller.signal,
    })

    if (response.status === 401 && auth && !skipRefresh) {
      const newToken = await refreshAccessToken()
      if (newToken) {
        headers.set('Authorization', `Bearer ${newToken}`)
        response = await fetch(`${apiBaseUrl()}${path}`, {
          method,
          credentials: 'include',
          headers,
          body: body === undefined ? undefined : JSON.stringify(body),
          signal: controller.signal,
        })
      }
    }

    if (response.status === 204) {
      return undefined as T
    }

    const payload = await parseJsonSafe(response)
    if (!response.ok) {
      throw errorFromResponse(response.status, payload)
    }

    return payload as T
  } finally {
    window.clearTimeout(timer)
    signal?.removeEventListener('abort', abortFromCaller)
  }
}

export function googleOAuthStartUrl(fromPath: string): string {
  const from = encodeURIComponent(fromPath)
  return `${apiBaseUrl()}/auth/google?from=${from}`
}

export async function apiUpload<T>(
  path: string,
  formData: FormData,
  auth = true,
): Promise<T> {
  const controller = new AbortController()
  const timer = window.setTimeout(() => controller.abort(), DEFAULT_TIMEOUT_MS)
  const headers = new Headers({ Accept: 'application/json' })

  if (auth) {
    const token = getAccessToken()
    if (token) {
      headers.set('Authorization', `Bearer ${token}`)
    }
  }

  try {
    let response = await fetch(`${apiBaseUrl()}${path}`, {
      method: 'POST',
      credentials: 'include',
      headers,
      body: formData,
      signal: controller.signal,
    })

    if (response.status === 401 && auth) {
      const newToken = await refreshAccessToken()
      if (newToken) {
        headers.set('Authorization', `Bearer ${newToken}`)
        response = await fetch(`${apiBaseUrl()}${path}`, {
          method: 'POST',
          credentials: 'include',
          headers,
          body: formData,
          signal: controller.signal,
        })
      }
    }

    if (response.status === 204) {
      return undefined as T
    }

    const payload = await parseJsonSafe(response)
    if (!response.ok) {
      throw errorFromResponse(response.status, payload)
    }
    return payload as T
  } finally {
    window.clearTimeout(timer)
  }
}

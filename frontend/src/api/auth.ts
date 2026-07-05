import { apiRequest, ApiError, apiUpload } from '@/lib/api-client'
import { clearPending2FASession, getPending2FAToken } from '@/lib/pending-2fa-store'
import { setAccessToken } from '@/lib/token-store'
import { sanitizeEmail } from '@/lib/utils'
import type {
  AuthTokenResponse,
  LoginResponse,
  RefreshResponse,
  RegisterBuyerResponse,
  RegisterSellerResponse,
  Setup2FAResponse,
  User,
  Verify2FAResponse,
} from '@/types/api'

export async function login(email: string, password: string): Promise<LoginResponse> {
  return apiRequest<LoginResponse>('/auth/login', {
    method: 'POST',
    body: { email: sanitizeEmail(email), password: password.trim() },
    skipRefresh: true,
  })
}

export async function challenge2FA(code: string, pendingTokenOverride?: string): Promise<AuthTokenResponse> {
  const pendingToken = pendingTokenOverride?.trim() || getPending2FAToken()
  if (!pendingToken) {
    throw new ApiError(401, 'UNAUTHORIZED', '2fa session expired')
  }
  const res = await apiRequest<AuthTokenResponse>('/auth/2fa/challenge', {
    method: 'POST',
    body: { code, pending_token: pendingToken },
    skipRefresh: true,
  })
  clearPending2FASession()
  setAccessToken(res.auth_token)
  return res
}

export async function refreshSession(): Promise<string | null> {
  try {
    const res = await apiRequest<RefreshResponse | undefined>('/auth/refresh', {
      method: 'POST',
      skipRefresh: true,
    })
    if (!res || typeof res.auth_token !== 'string') {
      return null
    }
    setAccessToken(res.auth_token)
    return res.auth_token
  } catch {
    return null
  }
}

export async function logout(): Promise<void> {
  await apiRequest<void>('/auth/logout', { method: 'POST', skipRefresh: true })
}

export async function fetchMe(): Promise<User> {
  const res = await apiRequest<{ user: User }>('/me', { auth: true })
  return res.user
}

export async function registerBuyer(email: string, password: string): Promise<RegisterBuyerResponse> {
  return apiRequest<RegisterBuyerResponse>('/auth/register', {
    method: 'POST',
    body: { email, password },
    skipRefresh: true,
  })
}

export interface SellerRegistrationPayload {
  email: string
  password: string
  full_name: string
  birth_date: string
  address: string
  identity_document_front: File
  identity_document_back: File
}

export async function registerSeller(payload: SellerRegistrationPayload): Promise<RegisterSellerResponse> {
  const form = new FormData()
  form.append('registration', JSON.stringify({
    email: payload.email,
    password: payload.password,
    full_name: payload.full_name,
    birth_date: payload.birth_date,
    address: payload.address,
    photo_present: true,
  }))
  form.append('identity_document_front', payload.identity_document_front)
  form.append('identity_document_back', payload.identity_document_back)

  return apiUpload<RegisterSellerResponse>('/auth/register/seller', form, false)
}

export async function exchangeOAuthCode(code: string): Promise<AuthTokenResponse> {
  const res = await apiRequest<AuthTokenResponse>('/auth/oauth/exchange', {
    method: 'POST',
    body: { code },
    skipRefresh: true,
  })
  setAccessToken(res.auth_token)
  return res
}

export async function confirmOAuthLink(linkCode: string): Promise<AuthTokenResponse> {
  const res = await apiRequest<AuthTokenResponse>('/auth/oauth/link-confirm', {
    method: 'POST',
    body: { link_code: linkCode },
    skipRefresh: true,
  })
  setAccessToken(res.auth_token)
  return res
}

export async function setup2FA(): Promise<Setup2FAResponse> {
  return apiRequest<Setup2FAResponse>('/auth/2fa/setup', { method: 'POST', auth: true })
}

export async function verify2FA(code: string): Promise<Verify2FAResponse> {
  return apiRequest<Verify2FAResponse>('/auth/2fa/verify', {
    method: 'POST',
    body: { code },
    auth: true,
  })
}

export async function forgotPassword(email: string): Promise<void> {
  await apiRequest<void>('/auth/forgot-password', {
    method: 'POST',
    body: { email },
    skipRefresh: true,
  })
}

export async function resetPassword(token: string, password: string): Promise<void> {
  if (!token.trim()) {
    throw new Error('invalid token')
  }
  await apiRequest<void>('/auth/reset-password', {
    method: 'POST',
    body: { token: token.trim(), password },
    skipRefresh: true,
  })
}

export async function checkEmailAvailable(email: string): Promise<boolean> {
  const q = encodeURIComponent(email.trim().toLowerCase())
  const res = await apiRequest<{ available: boolean }>(`/auth/check-email?email=${q}`, {
    skipRefresh: true,
  })
  return res.available
}

export async function fetchProfile(): Promise<User> {
  return apiRequest<User>('/me/profile', { auth: true })
}

export async function updateProfile(body: { name?: string; avatar_url?: string }): Promise<User> {
  return apiRequest<User>('/me/profile', { method: 'PUT', auth: true, body })
}

export async function changePassword(
  currentPassword: string,
  newPassword: string,
  totpCode?: string,
): Promise<void> {
  await apiRequest<void>('/me/change-password', {
    method: 'POST',
    auth: true,
    body: {
      current_password: currentPassword,
      new_password: newPassword,
      totp_code: totpCode ?? '',
    },
  })
}

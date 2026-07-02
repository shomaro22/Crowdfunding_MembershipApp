import { authenticator } from 'otplib'
import { expect, type APIRequestContext } from '@playwright/test'

import { DEMO_PASSWORD } from './auth.js'

export async function apiLogin(
  request: APIRequestContext,
  email: string,
  password = DEMO_PASSWORD,
): Promise<string> {
  for (let attempt = 0; attempt < 5; attempt += 1) {
    const res = await request.post('/auth/login', { data: { email, password } })
    if (res.status() === 429) {
      const retryAfter = Number(res.headers()['retry-after'] ?? '2')
      await new Promise((resolve) => setTimeout(resolve, Math.max(1000, retryAfter * 1000)))
      continue
    }
    expect(res.ok()).toBeTruthy()
    const body = (await res.json()) as { auth_token?: string; requires_2fa?: boolean }
    if (body.requires_2fa) {
      throw new Error(`2FA required for ${email}; use apiLoginWith2FA`)
    }
    if (!body.auth_token) {
      throw new Error('missing auth_token')
    }
    return body.auth_token
  }
  throw new Error(`login rate limited for ${email}`)
}

export async function apiLoginWith2FA(
  request: APIRequestContext,
  email: string,
  password: string,
  totpSecret: string,
): Promise<string> {
  const loginRes = await request.post('/auth/login', { data: { email, password } })
  expect(loginRes.ok()).toBeTruthy()
  const pending = (await loginRes.json()) as { requires_2fa?: boolean }
  if (!pending.requires_2fa) {
    return apiLogin(request, email, password)
  }
  const code = authenticator.generate(totpSecret)
  const challengeRes = await request.post('/auth/2fa/challenge', { data: { code } })
  expect(challengeRes.ok()).toBeTruthy()
  const body = (await challengeRes.json()) as { auth_token: string }
  return body.auth_token
}

export async function fetchNotifications(request: APIRequestContext, token: string): Promise<number> {
  const res = await request.get('/me/notifications', {
    headers: { Authorization: `Bearer ${token}` },
  })
  expect(res.ok()).toBeTruthy()
  const body = (await res.json()) as { unread_messages: number }
  return body.unread_messages
}

export async function registerBuyer(
  request: APIRequestContext,
  email: string,
  password = DEMO_PASSWORD,
): Promise<void> {
  const res = await request.post('/auth/register', {
    data: { email, password, name: 'E2E User' },
  })
  expect(res.status()).toBe(201)
}

export async function setup2FAWithBackup(
  request: APIRequestContext,
  token: string,
): Promise<{ secret: string; backupCodes: string[] }> {
  const setupRes = await request.post('/auth/2fa/setup', {
    headers: { Authorization: `Bearer ${token}` },
  })
  expect(setupRes.ok()).toBeTruthy()
  const setup = (await setupRes.json()) as { secret: string }
  const code = authenticator.generate(setup.secret)
  const verifyRes = await request.post('/auth/2fa/verify', {
    headers: { Authorization: `Bearer ${token}` },
    data: { code },
  })
  expect(verifyRes.ok()).toBeTruthy()
  const verified = (await verifyRes.json()) as { backup_codes: string[] }
  return { secret: setup.secret, backupCodes: verified.backup_codes }
}

export async function listPublicProjects(
  request: APIRequestContext,
  token: string,
  search?: string,
): Promise<Array<{ id: string; title: string }>> {
  const params = new URLSearchParams({ limit: '10', offset: '0' })
  if (search) {
    params.set('search', search)
  }
  const res = await request.get(`/projects?${params.toString()}`, {
    headers: { Authorization: `Bearer ${token}` },
  })
  expect(res.ok()).toBeTruthy()
  const body = (await res.json()) as { projects: Array<{ id: string; title: string }> }
  return body.projects
}

import { authenticator } from 'otplib'
import { expect, type Page } from '@playwright/test'

export const DEMO_PASSWORD = 'password123'
export const DEMO_TOTP_SECRET = 'JBSWY3DPEHPK3PXP'

export const DEMO_ACCOUNTS = {
  buyer: 'buyer@example.com',
  seller: 'seller@example.com',
  admin: 'admin@example.com',
} as const

export async function loginAs(page: Page, email: string, password = DEMO_PASSWORD): Promise<void> {
  await page.goto('/signin')
  await page.locator('#email').fill(email)
  await page.locator('#password').fill(password)

  for (let attempt = 0; attempt < 5; attempt += 1) {
    const loginResponse = page.waitForResponse((res) => res.url().includes('/auth/login'))
    await page.getByRole('button', { name: 'ログイン', exact: true }).click()
    const res = await loginResponse
    if (res.status() === 429) {
      const retryAfter = Number(res.headers()['retry-after'] ?? '2')
      await page.waitForTimeout(Math.max(1000, retryAfter * 1000))
      continue
    }
    if (res.ok()) {
      break
    }
    if (attempt === 4) {
      throw new Error(`login failed for ${email}: ${res.status()} ${await res.text()}`)
    }
  }

  await expect(page).not.toHaveURL(/\/signin$/)
}

export async function complete2FAIfNeeded(page: Page, totpSecret = DEMO_TOTP_SECRET): Promise<void> {
  if (!page.url().includes('/auth/2fa')) {
    return
  }
  for (let attempt = 0; attempt < 3; attempt += 1) {
    const code = authenticator.generate(totpSecret)
    await page.locator('#code').fill(code)
    const challengeResponse = page.waitForResponse((res) => res.url().includes('/auth/2fa/challenge'))
    await page.getByRole('button', { name: 'ログインを完了する' }).click()
    const res = await challengeResponse
    if (res.ok()) {
      await expect(page).not.toHaveURL(/\/auth\/2fa/)
      return
    }
    if (res.status() === 429) {
      const retryAfter = Number(res.headers()['retry-after'] ?? '2')
      await page.waitForTimeout(Math.max(1000, retryAfter * 1000))
      continue
    }
    await page.waitForTimeout(1000)
  }
  throw new Error('2FA challenge failed after retries')
}

export async function loginBuyer(page: Page): Promise<void> {
  await loginAs(page, DEMO_ACCOUNTS.buyer)
}

export async function loginSeller(page: Page): Promise<void> {
  await loginAs(page, DEMO_ACCOUNTS.seller)
  await complete2FAIfNeeded(page)
}

export async function loginAdmin(page: Page): Promise<void> {
  await loginAs(page, DEMO_ACCOUNTS.admin)
  await complete2FAIfNeeded(page)
}

export async function waitForAppReady(page: Page): Promise<void> {
  const response = await page.request.get('/health')
  expect(response.ok()).toBeTruthy()
}

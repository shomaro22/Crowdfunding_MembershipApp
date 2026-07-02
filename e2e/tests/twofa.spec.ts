import { authenticator } from 'otplib'
import { expect, test } from '@playwright/test'

import { registerBuyer, setup2FAWithBackup } from '../helpers/api.js'
import { DEMO_ACCOUNTS, DEMO_PASSWORD, DEMO_TOTP_SECRET, loginSeller, waitForAppReady } from '../helpers/auth.js'

test.describe('二要素認証 (2FA)', () => {
  test('出品者: TOTP 成功でログイン完了', async ({ page }) => {
    await waitForAppReady(page)
    await page.goto('/signin')
    await page.locator('#email').fill(DEMO_ACCOUNTS.seller)
    await page.locator('#password').fill(DEMO_PASSWORD)
    await page.getByRole('button', { name: 'ログイン', exact: true }).click()
    await expect(page).toHaveURL(/\/auth\/2fa/)

    const code = authenticator.generate(DEMO_TOTP_SECRET)
    await page.locator('#code').fill(code)
    await page.getByRole('button', { name: 'ログインを完了する' }).click()
    await expect(page).not.toHaveURL(/\/auth\/2fa/)
  })

  test('出品者: 不正な TOTP で失敗する', async ({ page }) => {
    await waitForAppReady(page)
    await page.goto('/signin')
    await page.locator('#email').fill(DEMO_ACCOUNTS.seller)
    await page.locator('#password').fill(DEMO_PASSWORD)
    await page.getByRole('button', { name: 'ログイン', exact: true }).click()
    await expect(page).toHaveURL(/\/auth\/2fa/)

    await page.locator('#code').fill('000000')
    await page.getByRole('button', { name: 'ログインを完了する' }).click()
    await expect(page).toHaveURL(/\/auth\/2fa/)
    await expect(page.locator('body')).toContainText(/失敗|invalid|認証/)
  })

  test('2FAセットアップでリカバリーコードが10件発行される', async ({ request }) => {
    const email = `e2e-backup-${Date.now()}@test.local`
    await registerBuyer(request, email)
    const loginRes = await request.post('/auth/login', { data: { email, password: DEMO_PASSWORD } })
    expect(loginRes.ok()).toBeTruthy()
    const { auth_token: token } = (await loginRes.json()) as { auth_token: string }
    const { backupCodes } = await setup2FAWithBackup(request, token)
    expect(backupCodes.length).toBe(10)
    expect(backupCodes[0].length).toBeGreaterThan(4)
  })

  test('出品者ログイン: 不正なリカバリーコードを拒否する', async ({ request }) => {
    const loginRes = await request.post('/auth/login', {
      data: { email: DEMO_ACCOUNTS.seller, password: DEMO_PASSWORD },
    })
    expect(loginRes.ok()).toBeTruthy()
    const pending = (await loginRes.json()) as { requires_2fa?: boolean }
    expect(pending.requires_2fa).toBe(true)

    const challengeRes = await request.post('/auth/2fa/challenge', { data: { code: 'INVALID-CODE' } })
    expect(challengeRes.status()).toBeGreaterThanOrEqual(400)
  })

  test('loginSeller ヘルパーでダッシュボード到達', async ({ page }) => {
    await waitForAppReady(page)
    await loginSeller(page)
    await page.goto('/seller/dashboard')
    await expect(page.getByRole('heading', { name: '出品者ダッシュボード' })).toBeVisible()
  })
})

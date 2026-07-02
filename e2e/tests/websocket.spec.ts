import { expect, test } from '@playwright/test'

import { apiLogin } from '../helpers/api.js'
import { DEMO_ACCOUNTS, DEMO_TOTP_SECRET, loginBuyer, waitForAppReady } from '../helpers/auth.js'
import { authenticator } from 'otplib'

test.describe('WebSocket / リアルタイム', () => {
  test('メッセージ送信後にチャットに反映される', async ({ browser }) => {
    const buyerContext = await browser.newContext()
    const sellerContext = await browser.newContext()
    const buyerPage = await buyerContext.newPage()
    const sellerPage = await sellerContext.newPage()

    try {
      await waitForAppReady(buyerPage)
      await loginBuyer(buyerPage)
      await buyerPage.goto('/messages')

      const threadButton = buyerPage.locator('aside button').filter({ hasText: /.+/ }).first()
      if (await threadButton.count() === 0) {
        test.skip(true, 'メッセージスレッドなし')
        return
      }
      const threadText = await threadButton.textContent()
      await threadButton.click()

      await sellerPage.goto('/signin')
      await sellerPage.locator('#email').fill(DEMO_ACCOUNTS.seller)
      await sellerPage.locator('#password').fill('password123')
      await sellerPage.getByRole('button', { name: 'ログイン', exact: true }).click()
      if (sellerPage.url().includes('/auth/2fa')) {
        await sellerPage.locator('#code').fill(authenticator.generate(DEMO_TOTP_SECRET))
        await sellerPage.getByRole('button', { name: 'ログインを完了する' }).click()
      }
      await sellerPage.goto('/messages')
      const sellerThread = sellerPage.locator('aside button').filter({ hasText: threadText?.split('\n')[0] ?? '' }).first()
      if (await sellerThread.count() > 0) {
        await sellerThread.click()
      }

      const unique = `e2e-ws-${Date.now()}`
      const textarea = sellerPage.locator('section textarea')
      if (await textarea.count() === 0) {
        test.skip(true, '出品者側スレッドなし')
        return
      }
      await textarea.fill(unique)
      await sellerPage.getByRole('button', { name: '送信' }).click()

      await expect(buyerPage.locator('section').getByText(unique)).toBeVisible({ timeout: 20_000 })
    } finally {
      await buyerContext.close()
      await sellerContext.close()
    }
  })

  test('通知 API が WebSocket 接続後も取得できる', async ({ page, request }) => {
    await waitForAppReady(page)
    await loginBuyer(page)
    await page.goto('/messages')
    await page.waitForTimeout(2000)

    const token = await apiLogin(request, DEMO_ACCOUNTS.buyer)
    const res = await request.get('/me/notifications', {
      headers: { Authorization: `Bearer ${token}` },
    })
    expect(res.ok()).toBeTruthy()
    const body = (await res.json()) as { unread_messages: number }
    expect(body.unread_messages).toBeGreaterThanOrEqual(0)
  })
})

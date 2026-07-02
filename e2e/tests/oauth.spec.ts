import { expect, test } from '@playwright/test'

import { apiLogin } from '../helpers/api.js'
import { DEMO_ACCOUNTS, waitForAppReady } from '../helpers/auth.js'

test.describe('OAuth（モック）', () => {
  test('コールバックで exchange 成功時にホームへ遷移する', async ({ page, request }) => {
    await waitForAppReady(page)

    const token = await apiLogin(request, DEMO_ACCOUNTS.buyer)
    const meRes = await request.get('/me', {
      headers: { Authorization: `Bearer ${token}` },
    })
    expect(meRes.ok()).toBeTruthy()
    const meBody = (await meRes.json()) as {
      user: { id: string; email: string; role: string; name: string | null; is_active: boolean }
    }

    await page.route('**/auth/oauth/exchange', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          auth_token: token,
          user: meBody.user,
        }),
      })
    })

    await page.goto('/auth/callback?code=e2e-mock-code&from=/')
    await expect(page).not.toHaveURL(/\/auth\/callback/)
    await expect(page.locator('body')).not.toContainText('Google ログインに失敗')
  })

  test('OAuth エラー時にメッセージが表示される', async ({ page }) => {
    await waitForAppReady(page)
    await page.goto('/auth/callback?error=oauth_failed')
    await expect(page.locator('body')).toContainText(/Google ログインに失敗|失敗/)
  })

  test('リンク確認コード付きコールバックは link 画面へ遷移する', async ({ page }) => {
    await waitForAppReady(page)
    await page.goto('/auth/callback?link_code=e2e-link-code&account_email=a@b.com&google_email=c@d.com')
    await expect(page).toHaveURL(/\/auth\/oauth\/link/)
  })
})

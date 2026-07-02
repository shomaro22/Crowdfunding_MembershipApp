import { expect, test } from '@playwright/test'

import { loginSeller, waitForAppReady } from '../helpers/auth.js'

test.describe('Stripe Connect（出品者）', () => {
  test.beforeEach(async ({ page }) => {
    await waitForAppReady(page)
    await page.route('**/seller/connect/onboarding', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ url: 'http://localhost/seller/connect/return' }),
      })
    })
    await loginSeller(page)
  })

  test('出品者ダッシュボードで Connect セクションが表示される', async ({ page }) => {
    await page.goto('/seller/dashboard')
    await expect(page.getByRole('heading', { name: '出品者ダッシュボード' })).toBeVisible()
    await expect(page.getByRole('heading', { name: 'Stripe Connect' })).toBeVisible()
  })

  test('Connect 設定更新で成功または戻り画面が表示される', async ({ page }) => {
    await page.goto('/seller/dashboard')
    const connectButton = page.getByRole('button', { name: /Connect を開始|設定を更新/ })
    await expect(connectButton).toBeVisible()
    await connectButton.click()

    await expect(
      page.getByText(
        /Stripe Connect の設定を更新しました|Connect 状態を確認中|連携を更新しました|Stripe Connect 連携を更新しました/,
      ),
    ).toBeVisible({ timeout: 20_000 })
  })
})

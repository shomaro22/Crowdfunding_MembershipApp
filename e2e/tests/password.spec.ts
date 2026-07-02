import { expect, test } from '@playwright/test'

import { DEMO_ACCOUNTS, loginBuyer, waitForAppReady } from '../helpers/auth.js'

test.describe.configure({ mode: 'serial' })

test.describe('パスワード', () => {
  test('パスワード再設定リクエスト画面が動作する', async ({ page }) => {
    await waitForAppReady(page)
    await page.goto('/forgot-password')
    await page.locator('input[type="email"]').fill(DEMO_ACCOUNTS.buyer)
    await page.getByRole('button', { name: '送信' }).click()
    await expect(page.locator('body')).toContainText(/再設定リンクを送信/)
  })

  test('期限切れトークンで再設定に失敗する', async ({ page }) => {
    await waitForAppReady(page)
    await page.goto('/reset-password?token=expired-or-invalid-token')
    await page.locator('input[placeholder="新しいパスワード"]').first().fill('NewPassword123!')
    await page.locator('input[placeholder="確認用パスワード"]').fill('NewPassword123!')
    await page.getByRole('button', { name: 'パスワードを更新' }).click()
    await expect(page.locator('body')).toContainText(/失敗|無効|invalid/i)
  })

  test('ログイン済みユーザーがパスワード変更フォームを表示できる', async ({ page }) => {
    await waitForAppReady(page)
    await loginBuyer(page)
    await page.goto('/profile')
    await expect(page.getByRole('heading', { name: 'パスワード変更', level: 2 })).toBeVisible()
    await expect(page.locator('#current-password')).toBeVisible()
    await expect(page.locator('#new-password')).toBeVisible()
  })
})

import { expect, test } from '@playwright/test'

import { DEMO_ACCOUNTS, DEMO_PASSWORD, loginBuyer, waitForAppReady } from '../helpers/auth.js'

test.describe('認証フロー', () => {
  test.beforeEach(async ({ page }) => {
    await waitForAppReady(page)
  })

  test('購入者ログイン後に応援履歴へ遷移できる', async ({ page }) => {
    await loginBuyer(page)
    await page.goto('/purchases')
    await expect(page.getByRole('heading', { name: '応援履歴', level: 1 })).toBeVisible()
    await expect(page.getByRole('heading', { name: 'メンバーシップ', level: 2 })).toBeVisible()
  })

  test('不正なパスワードでログインできない', async ({ page }) => {
    await page.goto('/signin')
    await page.locator('#email').fill(DEMO_ACCOUNTS.buyer)
    await page.locator('#password').fill('wrong-password')
    await page.getByRole('button', { name: 'ログイン', exact: true }).click()
    await expect(page).toHaveURL(/\/signin/, { timeout: 10_000 })
  })

  test('新規購入者登録フォームが表示される', async ({ page }) => {
    await page.goto('/signup')
    await expect(page.getByRole('heading', { name: /登録|サインアップ|会員登録/ })).toBeVisible()
    await expect(page.locator('#email, input[type="email"]').first()).toBeVisible()
    await expect(page.locator('#password')).toBeVisible()
  })

  test('未ログイン時は保護ページからサインインへリダイレクトされる', async ({ page }) => {
    await page.goto('/purchases')
    await expect(page).toHaveURL(/\/signin/)
  })
})

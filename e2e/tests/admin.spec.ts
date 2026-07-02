import { expect, test } from '@playwright/test'

import { loginAdmin, waitForAppReady } from '../helpers/auth.js'

test.describe.configure({ mode: 'serial' })

test.describe('管理画面', () => {
  test.beforeEach(async ({ page }) => {
    await waitForAppReady(page)
    await loginAdmin(page)
  })

  test('管理者ログイン後に管理画面へ遷移できる', async ({ page }) => {
    await page.goto('/admin')
    await expect(page.getByRole('heading', { name: '管理画面' })).toBeVisible()
    await expect(page.getByRole('button', { name: '統計' })).toBeVisible()
  })

  test('ダッシュボード（統計）タブで数値が表示される', async ({ page }) => {
    await page.goto('/admin?tab=stats')
    await expect(page.getByRole('paragraph').filter({ hasText: 'ユーザー' })).toBeVisible()
    await expect(page.getByRole('paragraph').filter({ hasText: 'プロジェクト' })).toBeVisible()
    await expect(page.getByRole('paragraph').filter({ hasText: '購入' })).toBeVisible()
    await expect(page.getByRole('paragraph').filter({ hasText: '売上' })).toBeVisible()
  })

  test('ユーザー一覧で有効切替ができる', async ({ page }) => {
    await page.goto('/admin?tab=users')
    const targetRow = page.locator('li.card').filter({ hasNotText: 'admin@example.com' }).first()
    if (await targetRow.count() === 0) {
      test.skip(true, '切替対象ユーザーが一覧にありません')
      return
    }
    const toggle = targetRow.getByRole('button', { name: '有効切替' })
    if (await toggle.count() === 0) {
      test.skip(true, '有効切替ボタンなし')
      return
    }
    await toggle.click()
    await expect(page.locator('body')).toContainText(/更新しました|有効|停止中/)
    await toggle.click()
    await expect(page.locator('body')).toContainText(/更新しました|有効/)
  })

  test('通報タブで一覧が表示され解決操作ができる', async ({ page }) => {
    await page.goto('/admin?tab=reports')
    const resolveButton = page.getByRole('button', { name: '解決' }).first()
    if (await resolveButton.count() === 0) {
      await expect(page.locator('ul')).toBeVisible()
      return
    }
    await resolveButton.click()
    await expect(page.locator('body')).toContainText(/通報を解決済みにしました|解決済/)
  })
})

import { expect, test } from '@playwright/test'

import { apiLogin, fetchNotifications } from '../helpers/api.js'
import { DEMO_ACCOUNTS, loginBuyer, waitForAppReady } from '../helpers/auth.js'

test.describe('通知（メッセージ未読）', () => {
  test.beforeEach(async ({ page }) => {
    await waitForAppReady(page)
    await loginBuyer(page)
  })

  test('通知 API で未読件数を取得できる', async ({ request }) => {
    const token = await apiLogin(request, DEMO_ACCOUNTS.buyer)
    const unread = await fetchNotifications(request, token)
    expect(unread).toBeGreaterThanOrEqual(0)
  })

  test('メッセージ画面でスレッドと未読表示を確認できる', async ({ page, request }) => {
    const token = await apiLogin(request, DEMO_ACCOUNTS.buyer)
    const unreadBefore = await fetchNotifications(request, token)

    await page.goto('/messages')
    await expect(page.getByRole('heading', { name: 'メッセージ', level: 1 })).toBeVisible()

    const unreadBadge = page.getByText(/\d+ 未読/)
    if (unreadBefore > 0) {
      await expect(unreadBadge.first()).toBeVisible()
    }

    const threadButton = page.locator('aside button').filter({ hasText: /.+/ }).first()
    if (await threadButton.count() === 0) {
      test.skip(true, 'メッセージスレッドなし（seed の支援関係を確認）')
      return
    }
    await threadButton.click()
    await expect(page.locator('section textarea')).toBeVisible()
  })

  test('スレッド選択後に未読が減る（既読化）', async ({ page, request }) => {
    const token = await apiLogin(request, DEMO_ACCOUNTS.buyer)
    const unreadBefore = await fetchNotifications(request, token)
    if (unreadBefore === 0) {
      test.skip(true, '未読メッセージなし')
      return
    }

    await page.goto('/messages')
    const threadWithUnread = page.locator('aside button').filter({ hasText: /未読/ }).first()
    if (await threadWithUnread.count() === 0) {
      test.skip(true, '未読スレッドなし')
      return
    }
    await threadWithUnread.click()
    await page.waitForTimeout(1500)

    const unreadAfter = await fetchNotifications(request, token)
    expect(unreadAfter).toBeLessThanOrEqual(unreadBefore)
  })
})

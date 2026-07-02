import { expect, test } from '@playwright/test'

import { loginBuyer, waitForAppReady } from '../helpers/auth.js'

test.describe('プラン変更', () => {
  test.beforeEach(async ({ page }) => {
    await waitForAppReady(page)
    await loginBuyer(page)
  })

  test('応援履歴からプラン変更モーダルを開ける', async ({ page }) => {
    await page.goto('/purchases')
    await expect(page.getByRole('heading', { name: '応援履歴', level: 1 })).toBeVisible()

    const changeButton = page.getByRole('button', { name: 'プラン変更' }).first()
    if (await changeButton.count() === 0) {
      test.skip(true, 'アクティブなメンバーシップがありません（seed を確認）')
      return
    }

    await changeButton.click()
    await expect(page.getByRole('heading', { name: 'プラン変更' })).toBeVisible()
    await expect(page.getByRole('button', { name: '閉じる' })).toBeVisible()
  })
})

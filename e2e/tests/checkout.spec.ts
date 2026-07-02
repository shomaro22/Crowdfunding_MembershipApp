import { expect, test } from '@playwright/test'

import { loginBuyer, waitForAppReady } from '../helpers/auth.js'

test.describe('チェックアウト', () => {
  test.beforeEach(async ({ page }) => {
    await waitForAppReady(page)
    await loginBuyer(page)
  })

  test('チェックアウト画面が表示される', async ({ page }) => {
    await page.goto('/')
    const projectLink = page.locator('a[href^="/projects/"]').first()
    await expect(projectLink).toBeVisible()
    const href = await projectLink.getAttribute('href')
    if (!href) {
      test.skip()
      return
    }
    const projectId = href.replace('/projects/', '')
    await page.goto(`/checkout/${projectId}`)

    const checkoutHeading = page.getByRole('heading', { name: '応援する' })
    const alreadySupporting = page.getByText(/既に応援中/)
    await expect(checkoutHeading.or(alreadySupporting)).toBeVisible()
  })

  test('デモ決済フォームでカード番号を入力できる', async ({ page }) => {
    await page.goto('/')
    const projectLink = page.locator('a[href^="/projects/"]').first()
    const href = await projectLink.getAttribute('href')
    if (!href) {
      test.skip()
      return
    }
    const projectId = href.replace('/projects/', '')
    await page.goto(`/checkout/${projectId}`)

    const payButton = page.getByRole('button', { name: /応援する|決済|次へ/ })
    if (await payButton.count() > 0) {
      await payButton.first().click()
    }

    const cardInput = page.getByPlaceholder('4242 4242 4242 4242')
    if (await cardInput.count() === 0) {
      test.skip(true, 'デモ決済フォームが表示されない（既に応援中の可能性）')
      return
    }

    await cardInput.fill('4242424242424242')
    await expect(cardInput).toHaveValue('4242424242424242')
    await page.getByPlaceholder('MM / YY').fill('12 / 30')
    await page.getByPlaceholder('CVC').fill('123')
  })
})

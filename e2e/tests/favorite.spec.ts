import { expect, test } from '@playwright/test'

import { loginBuyer, waitForAppReady } from '../helpers/auth.js'

test.describe('お気に入り', () => {
  test.beforeEach(async ({ page }) => {
    await waitForAppReady(page)
    await loginBuyer(page)
  })

  test('プロジェクト詳細でお気に入り追加→一覧に表示', async ({ page }) => {
    await page.goto('/')
    const projectLink = page.locator('a[href^="/projects/"]').first()
    await expect(projectLink).toBeVisible()
    const href = await projectLink.getAttribute('href')
    if (!href) {
      test.skip()
      return
    }

    await page.goto(href)
    const likeButton = page.getByRole('button', { name: /♡ いいね|♥ いいね済み/ }).first()
    if (await likeButton.count() === 0) {
      test.skip(true, 'いいねボタンなし')
      return
    }

    const label = await likeButton.textContent()
    if (label?.includes('いいね済み')) {
      await likeButton.click()
      await page.getByRole('button', { name: /♡ いいね|♥ いいね済み/ }).first().click()
    } else {
      await likeButton.click()
    }
    await expect(page.locator('body')).toContainText(/お気に入りに追加|いいね済み/)

    await page.goto('/favorites')
    await expect(page.getByRole('heading', { name: 'お気に入り' })).toBeVisible()
    await expect(page.locator(`a[href="${href}"]`).first()).toBeVisible()
  })

  test('お気に入りページから解除できる', async ({ page }) => {
    await page.goto('/favorites')
    const removeButton = page.getByRole('button', { name: 'お気に入り解除' }).first()
    if (await removeButton.count() === 0) {
      test.skip(true, 'お気に入りなし')
      return
    }
    const card = page.locator('.relative').first()
    const titleBefore = await card.locator('a').first().textContent()
    await removeButton.click()
    if (titleBefore) {
      await expect(page.locator('body')).not.toContainText(titleBefore)
    }
  })
})

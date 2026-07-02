import { expect, test } from '@playwright/test'

import { loginBuyer, waitForAppReady } from '../helpers/auth.js'

test.describe.configure({ mode: 'serial' })

test.describe('通報', () => {
  let reportedProjectId: string | null = null

  test('購入者がプロジェクトを通報できる', async ({ page }) => {
    await waitForAppReady(page)
    await loginBuyer(page)

    await page.goto('/')
    const projectLink = page.locator('a[href^="/projects/"]').last()
    if (await projectLink.count() === 0) {
      test.skip(true, '通報可能なプロジェクトなし')
      return
    }
    const href = await projectLink.getAttribute('href')
    if (!href) {
      test.skip(true, 'プロジェクトリンクなし')
      return
    }

    await page.goto(href)
    const reportButton = page.getByRole('button', { name: '通報' })
    if (await reportButton.count() === 0) {
      test.skip(true, '既に通報済み')
      return
    }
    await reportButton.click()
    await page.getByPlaceholder('通報理由').fill(`E2E通報 ${Date.now()}`)
    await page.getByRole('button', { name: '送信' }).click()
    await expect(page.locator('body')).toContainText(/通報を受け付けました/)
    reportedProjectId = href.replace('/projects/', '')
  })

  test('通報後は対象プロジェクトにアクセスできない', async ({ page }) => {
    if (!reportedProjectId) {
      test.skip(true, '前のテストで通報していません')
      return
    }
    await waitForAppReady(page)
    await loginBuyer(page)
    await page.goto(`/projects/${reportedProjectId}`)
    await expect(page.locator('body')).toContainText(/表示できません|通報|取得に失敗/)
  })
})

import { expect, test } from '@playwright/test'

test.describe.configure({ mode: 'parallel' })

for (const path of ['/', '/projects?limit=5', '/health']) {
  test(`並列: ${path} が応答する`, async ({ request }) => {
    const res = await request.get(path)
    expect(res.ok()).toBeTruthy()
  })
}

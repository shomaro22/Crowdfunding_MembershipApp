interface ZipCloudResult {
  address1: string
  address2: string
  address3: string
}

import { digitsOnly } from '@/lib/validation'

export async function lookupJapaneseAddress(postalCode: string): Promise<string | null> {
  const digits = digitsOnly(postalCode, 7)
  if (digits.length !== 7) {
    return null
  }
  const res = await fetch(`https://zipcloud.ibsnet.co.jp/api/search?zipcode=${digits}`)
  if (!res.ok) {
    return null
  }
  const data = (await res.json()) as { results: ZipCloudResult[] | null }
  const row = data.results?.[0]
  if (!row) {
    return null
  }
  return `${row.address1}${row.address2}${row.address3}`
}

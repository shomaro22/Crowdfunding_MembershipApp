/** アップロード URL を現在のオリジンに合わせる（docker: API_PUBLIC_URL 差分の吸収） */
export function normalizeMediaUrl(url: string): string {
  const trimmed = url.trim()
  if (!trimmed) {
    return ''
  }
  try {
    const parsed = new URL(trimmed)
    if (typeof window !== 'undefined') {
      const local = window.location
      if (
        (parsed.hostname === 'localhost' || parsed.hostname === '127.0.0.1') &&
        parsed.port === '8080' &&
        (local.port === '' || local.port === '80')
      ) {
        return `${local.protocol}//${local.host}${parsed.pathname}${parsed.search}${parsed.hash}`
      }
    }
    return trimmed
  } catch {
    return trimmed
  }
}

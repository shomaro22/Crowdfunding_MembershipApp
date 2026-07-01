/** 入力検証（クライアント側防御 — 最終判定は常にサーバー） */

const UUID_RE = /^[0-9a-f]{8}-[0-9a-f]{4}-[1-5][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/i
const TOTP_RE = /^[0-9]{6}$/
const TOTP_BACKUP_RE = /^[0-9A-Z]{8,10}$/

export function isValidUUID(value: string): boolean {
  return UUID_RE.test(value.trim())
}

export function parseUUID(value: string | undefined | null): string | null {
  if (!value) {
    return null
  }
  const trimmed = value.trim()
  return isValidUUID(trimmed) ? trimmed : null
}

export function validateTOTPCode(code: string): boolean {
  const trimmed = code.trim()
  return TOTP_RE.test(trimmed) || TOTP_BACKUP_RE.test(trimmed)
}

export function validateMessageContent(content: string): { valid: boolean; error?: string } {
  const trimmed = content.trim()
  if (trimmed.length === 0) {
    return { valid: false, error: 'メッセージを入力してください' }
  }
  if (trimmed.length > 4000) {
    return { valid: false, error: 'メッセージは4000文字以内で入力してください' }
  }
  return { valid: true }
}

export function validateReportDetail(detail: string): { valid: boolean; error?: string } {
  const trimmed = detail.trim()
  if (trimmed.length > 2000) {
    return { valid: false, error: '詳細は2000文字以内で入力してください' }
  }
  return { valid: true }
}

export function clampInt(value: number, min: number, max: number): number {
  if (!Number.isFinite(value)) {
    return min
  }
  return Math.min(max, Math.max(min, Math.floor(value)))
}

export function parsePageParam(raw: string | null, defaultPage = 1): number {
  if (!raw) {
    return defaultPage
  }
  const n = Number.parseInt(raw, 10)
  if (!Number.isFinite(n) || n < 1) {
    return defaultPage
  }
  return Math.min(n, 10_000)
}

export function parseLimitParam(raw: string | null, defaultLimit = 20, max = 100): number {
  if (!raw) {
    return defaultLimit
  }
  const n = Number.parseInt(raw, 10)
  return clampInt(n, 1, max)
}

/** 同一オリジンの http(s) URL のみ許可（Checkout リダイレクト用） */
export function validateSameOriginUrl(url: string, origin: string): boolean {
  try {
    const parsed = new URL(url)
    const base = new URL(origin)
    if (parsed.origin !== base.origin) {
      return false
    }
    if (parsed.protocol !== 'http:' && parsed.protocol !== 'https:') {
      return false
    }
    return parsed.pathname.startsWith('/')
  } catch {
    return false
  }
}

export function validateProjectTitle(title: string): string | null {
  const trimmed = title.trim()
  if (trimmed.length === 0) {
    return 'タイトルを入力してください'
  }
  if (trimmed.length > 200) {
    return 'タイトルは200文字以内で入力してください'
  }
  return null
}

export function validatePositiveAmount(amount: number, label: string, max = 100_000_000): string | null {
  if (!Number.isInteger(amount) || amount < 0) {
    return `${label}は0以上の整数で入力してください`
  }
  if (amount > max) {
    return `${label}が大きすぎます`
  }
  return null
}

export function validatePlanPrice(price: number): string | null {
  if (!Number.isInteger(price) || price < 1 || price > 100_000) {
    return 'プラン価格は1〜100,000円で入力してください'
  }
  return null
}

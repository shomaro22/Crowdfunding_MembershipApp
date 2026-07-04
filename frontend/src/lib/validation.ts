/** 入力検証（クライアント側防御 — 最終判定は常にサーバー） */

import { digitsOnly, normalizeAsciiDigits } from '@/lib/digits'

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
  const trimmed = normalizeOtpCode(code)
  return TOTP_RE.test(trimmed) || TOTP_BACKUP_RE.test(trimmed)
}

/** 全角数字・空白を半角 OTP 用に正規化 */
export function normalizeOtpCode(value: string): string {
  return normalizeAsciiDigits(value)
    .replace(/[\uFF21-\uFF3A]/g, (ch) => String.fromCharCode(ch.charCodeAt(0) - 0xff21 + 0x41))
    .replace(/[\uFF41-\uFF5A]/g, (ch) => String.fromCharCode(ch.charCodeAt(0) - 0xff41 + 0x61))
    .replace(/\s+/g, '')
    .replace(/[^\dA-Za-z-]/g, '')
}

function luhnCheck(cardDigits: string): boolean {
  let sum = 0
  let alternate = false
  for (let i = cardDigits.length - 1; i >= 0; i -= 1) {
    let n = Number.parseInt(cardDigits[i] ?? '0', 10)
    if (alternate) {
      n *= 2
      if (n > 9) {
        n -= 9
      }
    }
    sum += n
    alternate = !alternate
  }
  return sum % 10 === 0
}

export function formatCardNumberInput(value: string): string {
  const digits = digitsOnly(value, 19)
  const groups = digits.match(/.{1,4}/g)
  return groups ? groups.join(' ') : ''
}

export function formatCardExpiryInput(value: string): string {
  const digits = digitsOnly(value, 4)
  if (digits.length <= 2) {
    return digits
  }
  return `${digits.slice(0, 2)} / ${digits.slice(2)}`
}

export function validateDemoCardNumber(cardNumber: string): string | null {
  const digits = digitsOnly(cardNumber)
  if (digits.length < 13) {
    return 'カード番号の桁数が足りません'
  }
  if (digits.length > 19) {
    return 'カード番号が長すぎます'
  }
  if (!luhnCheck(digits)) {
    return 'カード番号が正しくありません'
  }
  return null
}

export function validateDemoCardExpiry(expiry: string): string | null {
  const digits = digitsOnly(expiry)
  if (digits.length !== 4) {
    return '有効期限を MM / YY 形式で入力してください'
  }
  const month = Number.parseInt(digits.slice(0, 2), 10)
  const year = Number.parseInt(digits.slice(2, 4), 10)
  if (month < 1 || month > 12) {
    return '有効期限の月が正しくありません'
  }
  const now = new Date()
  const currentYear = now.getFullYear() % 100
  const currentMonth = now.getMonth() + 1
  if (year < currentYear || (year === currentYear && month < currentMonth)) {
    return '有効期限が過去の日付です'
  }
  return null
}

export function validateDemoCardCvc(cvc: string): string | null {
  const digits = digitsOnly(cvc, 3)
  if (digits.length !== 3) {
    return 'セキュリティコードは3桁で入力してください'
  }
  return null
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
  if (trimmed.length === 0) {
    return { valid: false, error: '通報理由を入力してください' }
  }
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

const EMAIL_RE = /^[^\s@]+@[^\s@]+\.[^\s@]+$/

export function validateEmail(email: string): string | null {
  const trimmed = email.trim().toLowerCase()
  if (!trimmed) {
    return 'メールアドレスを入力してください'
  }
  if (trimmed.length > 255) {
    return 'メールアドレスが長すぎます'
  }
  if (!EMAIL_RE.test(trimmed)) {
    return '有効なメールアドレスを入力してください'
  }
  return null
}

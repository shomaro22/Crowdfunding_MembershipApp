import { clsx, type ClassValue } from 'clsx'
import { twMerge } from 'tailwind-merge'

import { normalizeAsciiDigits } from '@/lib/digits'

export { digitsOnly, normalizeAsciiDigits, postalCodeInput } from '@/lib/digits'

export function cn(...inputs: ClassValue[]): string {
  return twMerge(clsx(inputs))
}

/** XSS 対策: 表示用テキストの長さ・危険文字を制限 */
export function sanitizeDisplayText(input: string, maxLength = 1000): string {
  return input.trim().slice(0, maxLength).replace(/[<>]/g, '')
}

export function sanitizeEmail(email: string): string {
  return email.trim().toLowerCase().slice(0, 255)
}

export function isValidEmail(email: string): boolean {
  return /^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(email)
}

export type PasswordStrength = 'weak' | 'medium' | 'strong' | 'very-strong'

export function evaluatePasswordStrength(password: string): PasswordStrength {
  if (password.length < 8) {
    return 'weak'
  }

  let score = 0
  if (password.length >= 8) score += 1
  if (password.length >= 12) score += 1
  if (password.length >= 16) score += 1
  if (/[a-z]/.test(password)) score += 1
  if (/[A-Z]/.test(password)) score += 1
  if (/[0-9]/.test(password)) score += 1
  if (/[^a-zA-Z0-9]/.test(password)) score += 1

  if (score >= 7) return 'very-strong'
  if (score >= 5) return 'strong'
  if (score >= 3) return 'medium'
  return 'weak'
}

export function validatePassword(password: string): { valid: boolean; errors: string[] } {
  const errors: string[] = []

  if (!password) {
    errors.push('パスワードを入力してください')
    return { valid: false, errors }
  }
  if (password.length < 8) {
    errors.push('パスワードは8文字以上で入力してください')
  }
  if (password.length > 128) {
    errors.push('パスワードは128文字以下で入力してください')
  }
  if (/\s/.test(password)) {
    errors.push('パスワードに空白文字は使用できません')
  }
  if (!/[a-z]/.test(password)) {
    errors.push('パスワードには小文字を含めてください')
  }
  if (!/[A-Z]/.test(password)) {
    errors.push('パスワードには大文字を含めてください')
  }
  if (!/[0-9]/.test(password)) {
    errors.push('パスワードには数字を含めてください')
  }
  if (!/[^a-zA-Z0-9]/.test(password)) {
    errors.push('パスワードには特殊文字を含めてください')
  }

  return { valid: errors.length === 0, errors }
}

/**
 * オープンリダイレクト防止: アプリ内相対パスのみ許可（バックエンド ValidateAppRedirectURL と同方針）
 */
export function sanitizeAppPath(input: string | null | undefined, fallback = '/'): string {
  if (!input || input === '/') {
    return fallback === '' ? '/' : fallback
  }
  if (!input.startsWith('/') || input.startsWith('//') || input.includes('://')) {
    return fallback
  }
  if (input.includes('\\')) {
    return fallback
  }
  for (const ch of input) {
    const code = ch.codePointAt(0) ?? 0
    if (code <= 0x1f || code === 0x7f) {
      return fallback
    }
  }
  return input
}

export function formatCurrencyYen(amount: number): string {
  return `${amount.toLocaleString('ja-JP')}円`
}

/** 目標金額など: 半角数字のみ受け付け、表示はカンマ区切り */
export function formatYenInput(raw: string): string {
  const digits = normalizeAsciiDigits(raw).replace(/\D/g, '').slice(0, 12)
  if (digits === '') {
    return ''
  }
  return Number(digits).toLocaleString('ja-JP')
}

export function parseYenInput(formatted: string): number {
  const digits = normalizeAsciiDigits(formatted).replace(/\D/g, '')
  if (digits === '') {
    return 0
  }
  return Number.parseInt(digits, 10)
}

export function fundingProgressPercent(current: number, target: number): number {
  if (target <= 0) {
    return 0
  }
  return Math.round((current / target) * 100)
}

export function fundingProgressBarWidth(percent: number): number {
  return Math.min(100, Math.max(0, percent))
}

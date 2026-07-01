import type { AdminRole } from '@/types/api'

const ADMIN_ROLES: ReadonlySet<AdminRole> = new Set([
  'SUPER_ADMIN',
  'REVIEW_ADMIN',
  'SUPPORT_ADMIN',
  'FINANCE_ADMIN',
])

interface ParsedJWTClaims {
  adminRole: AdminRole | null
  exp: number | null
}

/**
 * JWT payload を UI 表示用に読む（署名検証はサーバーのみ）。
 * admin メニュー出し分けに使用 — 権限の最終判定は API の 403 に委ねる。
 */
export function parseJWTClaims(token: string): ParsedJWTClaims {
  const parts = token.split('.')
  if (parts.length !== 3) {
    return { adminRole: null, exp: null }
  }
  try {
    const payload = parts[1]
    if (!payload) {
      return { adminRole: null, exp: null }
    }
    const padded = payload.replace(/-/g, '+').replace(/_/g, '/')
    const json = atob(padded.padEnd(Math.ceil(padded.length / 4) * 4, '='))
    const data = JSON.parse(json) as Record<string, unknown>
    let adminRole: AdminRole | null = null
    if (typeof data.admin_role === 'string' && ADMIN_ROLES.has(data.admin_role as AdminRole)) {
      adminRole = data.admin_role as AdminRole
    }
    let exp: number | null = null
    if (typeof data.exp === 'number') {
      exp = data.exp
    }
    return { adminRole, exp }
  } catch {
    return { adminRole: null, exp: null }
  }
}

export function isTokenExpired(exp: number | null, skewSeconds = 30): boolean {
  if (exp === null) {
    return false
  }
  return Date.now() / 1000 >= exp - skewSeconds
}

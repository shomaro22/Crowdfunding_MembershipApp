import type { User } from '@/types/api'

/** 2FA チャレンジ用（クロスオリジン Cookie 非依存。短期のみ sessionStorage） */
const TOKEN_KEY = 'pending_2fa_token'
const USER_KEY = 'pending_2fa_user'

let pendingToken: string | null = null
let pendingUser: User | null = null

function readStorage(): void {
  if (pendingToken) {
    return
  }
  try {
    const token = sessionStorage.getItem(TOKEN_KEY)
    const userRaw = sessionStorage.getItem(USER_KEY)
    if (token) {
      pendingToken = token
    }
    if (userRaw) {
      pendingUser = JSON.parse(userRaw) as User
    }
  } catch {
    /* private mode 等 */
  }
}

function writeStorage(token: string, user: User): void {
  try {
    sessionStorage.setItem(TOKEN_KEY, token)
    sessionStorage.setItem(USER_KEY, JSON.stringify(user))
  } catch {
    /* ignore */
  }
}

export function setPending2FASession(token: string, user: User): void {
  pendingToken = token
  pendingUser = user
  writeStorage(token, user)
}

export function getPending2FAToken(): string | null {
  readStorage()
  return pendingToken
}

export function getPending2FAUser(): User | null {
  readStorage()
  return pendingUser
}

export function clearPending2FASession(): void {
  pendingToken = null
  pendingUser = null
  try {
    sessionStorage.removeItem(TOKEN_KEY)
    sessionStorage.removeItem(USER_KEY)
  } catch {
    /* ignore */
  }
}

/** @deprecated use setPending2FASession */
export function setPending2FAToken(token: string): void {
  pendingToken = token
  try {
    sessionStorage.setItem(TOKEN_KEY, token)
  } catch {
    /* ignore */
  }
}

/** @deprecated use clearPending2FASession */
export function clearPending2FAToken(): void {
  clearPending2FASession()
}

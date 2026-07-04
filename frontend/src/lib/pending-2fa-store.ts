/** 2FA チャレンジ用の短期トークン（メモリのみ。クロスオリジン Cookie 非依存） */
let pendingToken: string | null = null

export function setPending2FAToken(token: string): void {
  pendingToken = token
}

export function getPending2FAToken(): string | null {
  return pendingToken
}

export function clearPending2FAToken(): void {
  pendingToken = null
}

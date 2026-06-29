import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useState,
  type ReactNode,
} from 'react'

import {
  challenge2FA,
  fetchMe,
  login as loginApi,
  logout as logoutApi,
  refreshSession,
} from '@/api/auth'
import { clearAccessToken, setAccessToken } from '@/lib/token-store'
import { isLogin2FAResponse, type User, type UserRole } from '@/types/api'

export type AuthStatus = 'loading' | 'anonymous' | 'authenticated' | 'pending_2fa'

interface AuthContextValue {
  status: AuthStatus
  user: User | null
  login: (email: string, password: string) => Promise<'ok' | '2fa'>
  complete2FA: (code: string) => Promise<void>
  logout: () => Promise<void>
  refreshUser: () => Promise<void>
  hasRole: (role: UserRole) => boolean
}

const AuthContext = createContext<AuthContextValue | null>(null)

export function AuthProvider({ children }: { children: ReactNode }) {
  const [status, setStatus] = useState<AuthStatus>('loading')
  const [user, setUser] = useState<User | null>(null)

  const refreshUser = useCallback(async () => {
    const me = await fetchMe()
    setUser(me)
    setStatus('authenticated')
  }, [])

  useEffect(() => {
    let cancelled = false

    void (async () => {
      const token = await refreshSession()
      if (cancelled) {
        return
      }
      if (!token) {
        setStatus('anonymous')
        return
      }
      try {
        const me = await fetchMe()
        if (!cancelled) {
          setUser(me)
          setStatus('authenticated')
        }
      } catch {
        clearAccessToken()
        if (!cancelled) {
          setUser(null)
          setStatus('anonymous')
        }
      }
    })()

    return () => {
      cancelled = true
    }
  }, [])

  const login = useCallback(async (email: string, password: string): Promise<'ok' | '2fa'> => {
    const res = await loginApi(email, password)
    if (isLogin2FAResponse(res)) {
      setUser(res.user)
      setStatus('pending_2fa')
      return '2fa'
    }
    setAccessToken(res.auth_token)
    setUser(res.user)
    setStatus('authenticated')
    return 'ok'
  }, [])

  const complete2FA = useCallback(async (code: string) => {
    const res = await challenge2FA(code)
    setUser(res.user)
    setStatus('authenticated')
  }, [])

  const logout = useCallback(async () => {
    try {
      await logoutApi()
    } finally {
      clearAccessToken()
      setUser(null)
      setStatus('anonymous')
    }
  }, [])

  const hasRole = useCallback(
    (role: UserRole) => user?.role === role,
    [user],
  )

  const value = useMemo<AuthContextValue>(
    () => ({
      status,
      user,
      login,
      complete2FA,
      logout,
      refreshUser,
      hasRole,
    }),
    [status, user, login, complete2FA, logout, refreshUser, hasRole],
  )

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>
}

export function useAuth(): AuthContextValue {
  const ctx = useContext(AuthContext)
  if (!ctx) {
    throw new Error('useAuth must be used within AuthProvider')
  }
  return ctx
}

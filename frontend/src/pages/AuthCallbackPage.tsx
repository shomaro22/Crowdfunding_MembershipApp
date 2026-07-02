import { useEffect, useRef, useState } from 'react'
import { useNavigate, useSearchParams } from 'react-router-dom'

import { exchangeOAuthCode, fetchMe } from '@/api/auth'
import { useAuth } from '@/auth/AuthProvider'
import { Loading } from '@/components/common/Loading'
import { ErrorMessage } from '@/components/common/ErrorMessage'
import { sanitizeAppPath } from '@/lib/utils'
import { postLoginPath } from '@/lib/auth-routes'

export function AuthCallbackPage() {
  const [searchParams] = useSearchParams()
  const navigate = useNavigate()
  const { refreshUser } = useAuth()
  const [error, setError] = useState('')
  const started = useRef(false)

  useEffect(() => {
    if (started.current) {
      return
    }
    started.current = true

    const oauthError = searchParams.get('error')
    if (oauthError) {
      if (oauthError === 'user_inactive') {
        setError('アカウントが有効化されていません。メール登録の場合は管理者の承認をお待ちください。')
      } else if (oauthError === 'oauth_failed') {
        setError('Google ログインに失敗しました。しばらくしてから再度お試しください。')
      } else if (oauthError === 'oauth_forbidden') {
        setError('この Google アカウントではログインできません（出品者アカウント等）。')
      } else {
        setError('Google ログインに失敗しました')
      }
      return
    }

    const code = searchParams.get('code')
    const from = sanitizeAppPath(searchParams.get('from'), '/')

    if (!code) {
      setError('認可コードがありません')
      return
    }

    void (async () => {
      try {
        await exchangeOAuthCode(code)
        await refreshUser()
        const me = await fetchMe()
        void navigate(postLoginPath(me, from), { replace: true })
      } catch {
        setError('ログインセッションの確立に失敗しました')
      }
    })()
  }, [navigate, refreshUser, searchParams])

  if (error) {
    return <ErrorMessage message={error} onRetry={() => { void navigate('/signin', { replace: true }) }} />
  }

  return <Loading label="ログイン処理中..." />
}

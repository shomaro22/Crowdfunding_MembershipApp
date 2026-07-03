import { useEffect, useRef, useState } from 'react'
import { useNavigate, useSearchParams } from 'react-router-dom'

import { exchangeOAuthCode } from '@/api/auth'
import { useAuth } from '@/auth/AuthProvider'
import { Loading } from '@/components/common/Loading'
import { ErrorMessage } from '@/components/common/ErrorMessage'
import { MainLayout } from '@/components/layout/MainLayout'
import { sanitizeAppPath } from '@/lib/utils'
import { postLoginPath } from '@/lib/auth-routes'

export function AuthCallbackPage() {
  const [searchParams] = useSearchParams()
  const navigate = useNavigate()
  const { acceptAuthenticatedUser } = useAuth()
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

    const linkCode = searchParams.get('link_code')
    if (linkCode) {
      const params = new URLSearchParams()
      params.set('link_code', linkCode)
      const accountEmail = searchParams.get('account_email')
      const googleEmail = searchParams.get('google_email')
      const from = searchParams.get('from')
      if (accountEmail) {
        params.set('account_email', accountEmail)
      }
      if (googleEmail) {
        params.set('google_email', googleEmail)
      }
      if (from) {
        params.set('from', from)
      }
      void navigate(`/auth/oauth/link?${params.toString()}`, { replace: true })
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
        const res = await exchangeOAuthCode(code)
        acceptAuthenticatedUser(res.user)
        void navigate(postLoginPath(res.user, from), { replace: true })
      } catch {
        setError('ログインセッションの確立に失敗しました')
      }
    })()
  }, [acceptAuthenticatedUser, navigate, searchParams])

  if (error) {
    return (
      <MainLayout>
        <ErrorMessage message={error} onRetry={() => { void navigate('/signin', { replace: true }) }} />
      </MainLayout>
    )
  }

  return (
    <MainLayout>
      <Loading label="ログイン処理中..." />
    </MainLayout>
  )
}

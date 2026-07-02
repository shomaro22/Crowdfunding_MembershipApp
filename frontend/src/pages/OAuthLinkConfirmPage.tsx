import { useState } from 'react'
import { Link, useNavigate, useSearchParams } from 'react-router-dom'

import { confirmOAuthLink } from '@/api/auth'
import { useAuth } from '@/auth/AuthProvider'
import { ApiError } from '@/lib/api-client'
import { postLoginPath } from '@/lib/auth-routes'
import { sanitizeAppPath } from '@/lib/utils'

export function OAuthLinkConfirmPage() {
  const [searchParams] = useSearchParams()
  const navigate = useNavigate()
  const { refreshUser } = useAuth()
  const [submitting, setSubmitting] = useState(false)
  const [error, setError] = useState('')

  const linkCode = searchParams.get('link_code') ?? ''
  const accountEmail = searchParams.get('account_email') ?? ''
  const googleEmail = searchParams.get('google_email') ?? ''
  const from = sanitizeAppPath(searchParams.get('from'), '/')

  if (!linkCode || !accountEmail) {
    return (
      <div className="card mx-auto max-w-md text-center">
        <p className="text-red-600">連携情報が無効です。もう一度 Google ログインをお試しください。</p>
        <Link to="/signin" className="btn-primary mt-4 inline-block rounded-full px-6 py-2">ログインへ</Link>
      </div>
    )
  }

  const onConfirm = async () => {
    setSubmitting(true)
    setError('')
    try {
      const res = await confirmOAuthLink(linkCode)
      await refreshUser()
      void navigate(postLoginPath(res.user, from), { replace: true })
    } catch (err) {
      setError(err instanceof ApiError ? err.message : 'アカウント連携に失敗しました')
    } finally {
      setSubmitting(false)
    }
  }

  return (
    <div className="card mx-auto max-w-md">
      <h1 className="mb-4 text-xl font-bold text-gray-900">Google アカウントの連携</h1>
      <p className="mb-4 text-sm leading-relaxed text-gray-700">
        Google アカウント
        {googleEmail ? (
          <strong className="mx-1">{googleEmail}</strong>
        ) : null}
        を、既存の Crowdfund アカウント
        <strong className="mx-1">{accountEmail}</strong>
        に連携しますか？
      </p>
      <p className="mb-6 text-xs text-gray-500">
        連携後は Google でも同じアカウントでログインできます。応援履歴やメッセージはそのまま引き継がれます。
      </p>
      {error ? <p className="mb-4 rounded-lg bg-red-50 px-3 py-2 text-sm text-red-600">{error}</p> : null}
      <div className="flex flex-col gap-3 sm:flex-row">
        <button
          type="button"
          disabled={submitting}
          onClick={() => void onConfirm()}
          className="btn-primary flex-1 rounded-full font-bold"
        >
          {submitting ? '連携中...' : '連携する'}
        </button>
        <Link to="/signin" className="btn-secondary flex-1 rounded-full text-center">
          キャンセル
        </Link>
      </div>
    </div>
  )
}

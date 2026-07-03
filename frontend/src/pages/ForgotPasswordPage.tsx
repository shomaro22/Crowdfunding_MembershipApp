import { useState, type FormEvent } from 'react'
import { Link } from 'react-router-dom'

import { forgotPassword } from '@/api/auth'
import { ApiError } from '@/lib/api-client'
import { validateEmail } from '@/lib/validation'
import { sanitizeEmail } from '@/lib/utils'

export function ForgotPasswordPage() {
  const [email, setEmail] = useState('')
  const [sent, setSent] = useState(false)
  const [error, setError] = useState('')
  const [loading, setLoading] = useState(false)

  const onSubmit = async (e: FormEvent) => {
    e.preventDefault()
    setError('')
    const emailValidationMessage = validateEmail(email)
    if (emailValidationMessage !== null) {
      setError(emailValidationMessage)
      return
    }
    setLoading(true)
    try {
      await forgotPassword(sanitizeEmail(email))
      setSent(true)
    } catch (err) {
      if (err instanceof ApiError && err.code === 'RATE_LIMITED') {
        setError('リクエストが多すぎます。しばらく待ってから再試行してください')
      } else {
        setSent(true)
      }
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="card mx-auto max-w-md">
      <h1 className="mb-4 text-2xl font-bold">パスワード再設定</h1>
      {sent ? (
        <div className="space-y-3 text-gray-700">
          <p>
            入力されたメールアドレスが登録されている場合、再設定リンクを送信しました。
          </p>
          <div className="rounded-lg border border-amber-100 bg-amber-50 p-3 text-sm text-amber-900">
            <p className="font-semibold">開発環境の場合</p>
            <p className="mt-1">
              メールは送信されません。backend コンテナのログに
              <code className="mx-1 rounded bg-white px-1">reset_url</code>
              が出力されます。
            </p>
            <pre className="mt-2 overflow-x-auto rounded bg-white p-2 text-xs">docker compose logs backend | grep reset_url</pre>
            <p className="mt-2 text-xs">
              リンクのホストは <code>PUBLIC_APP_URL</code>（Docker では <code>http://localhost</code>）に合わせてください。
            </p>
          </div>
        </div>
      ) : (
        <form onSubmit={(e) => void onSubmit(e)} className="space-y-4">
          {error ? <p className="text-sm text-red-600">{error}</p> : null}
          <input
            type="email"
            autoComplete="email"
            className="input-field"
            placeholder="メールアドレス"
            value={email}
            onChange={(ev) => setEmail(ev.target.value)}
          />
          <button type="submit" disabled={loading} className="btn-primary w-full rounded-full">
            {loading ? '送信中...' : '送信'}
          </button>
        </form>
      )}
      <p className="mt-6 text-center text-sm">
        <Link to="/signin" className="text-purple-700">ログインに戻る</Link>
      </p>
    </div>
  )
}

import { useState, type FormEvent } from 'react'
import { Link, useNavigate, useSearchParams } from 'react-router-dom'

import { resetPassword } from '@/api/auth'
import { ApiError } from '@/lib/api-client'
import { validatePassword } from '@/lib/utils'

export function ResetPasswordPage() {
  const [searchParams] = useSearchParams()
  const navigate = useNavigate()
  const token = searchParams.get('token') ?? ''
  const [password, setPassword] = useState('')
  const [confirm, setConfirm] = useState('')
  const [error, setError] = useState('')
  const [loading, setLoading] = useState(false)

  const onSubmit = async (e: FormEvent) => {
    e.preventDefault()
    setError('')
    if (!token || token.length > 256) {
      setError('再設定リンクが無効です')
      return
    }
    const v = validatePassword(password)
    if (!v.valid) {
      setError(v.errors[0] ?? 'パスワードが無効です')
      return
    }
    if (password !== confirm) {
      setError('確認用パスワードが一致しません')
      return
    }
    setLoading(true)
    try {
      await resetPassword(token, password)
      void navigate('/signin', { replace: true })
    } catch (err) {
      setError(err instanceof ApiError ? err.message : '再設定に失敗しました')
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="card mx-auto max-w-md">
      <h1 className="mb-4 text-2xl font-bold">新しいパスワード</h1>
      <form onSubmit={(e) => void onSubmit(e)} className="space-y-4">
        {error ? <p className="text-sm text-red-600">{error}</p> : null}
        <input
          type="password"
          autoComplete="new-password"
          className="input-field"
          placeholder="新しいパスワード"
          value={password}
          onChange={(ev) => setPassword(ev.target.value)}
        />
        <input
          type="password"
          autoComplete="new-password"
          className="input-field"
          placeholder="確認用パスワード"
          value={confirm}
          onChange={(ev) => setConfirm(ev.target.value)}
        />
        <button type="submit" disabled={loading} className="btn-primary w-full rounded-full">
          {loading ? '更新中...' : 'パスワードを更新'}
        </button>
      </form>
      <p className="mt-6 text-center text-sm">
        <Link to="/signin" className="text-purple-700">ログイン</Link>
      </p>
    </div>
  )
}

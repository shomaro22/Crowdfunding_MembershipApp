import { useState, type FormEvent } from 'react'
import { useLocation, useNavigate } from 'react-router-dom'

import { useAuth } from '@/auth/AuthProvider'
import { ApiError } from '@/lib/api-client'
import { sanitizeAppPath } from '@/lib/utils'

export function TwoFactorForm() {
  const { complete2FA, status } = useAuth()
  const navigate = useNavigate()
  const location = useLocation()
  const from = sanitizeAppPath(
    typeof location.state === 'object' && location.state !== null && 'from' in location.state
      ? String(location.state.from)
      : '/',
  )

  const [code, setCode] = useState('')
  const [error, setError] = useState('')
  const [isSubmitting, setIsSubmitting] = useState(false)

  if (status !== 'pending_2fa' && status !== 'authenticated') {
    return null
  }

  const onSubmit = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault()
    setError('')

    const normalized = code.trim().replace(/\s+/g, '')
    if (normalized.length < 6) {
      setError('認証コードを入力してください')
      return
    }

    setIsSubmitting(true)
    try {
      await complete2FA(normalized)
      void navigate(from, { replace: true })
    } catch (err) {
      if (err instanceof ApiError) {
        setError(err.message)
      } else {
        setError('認証に失敗しました')
      }
    } finally {
      setIsSubmitting(false)
    }
  }

  return (
    <div className="card mx-auto max-w-md">
      <div className="mb-8 text-center">
        <h2 className="mb-2 text-2xl font-bold text-gray-800">二段階認証</h2>
        <p className="text-gray-600">認証アプリの6桁コードを入力してください</p>
      </div>

      {error ? <p className="mb-4 text-sm text-red-600">{error}</p> : null}

      <form onSubmit={(e) => void onSubmit(e)} className="space-y-4">
        <div>
          <label htmlFor="code" className="mb-2 block text-sm font-medium text-gray-700">認証コード</label>
          <input
            id="code"
            inputMode="numeric"
            autoComplete="one-time-code"
            className="input-field tracking-widest"
            value={code}
            onChange={(e) => setCode(e.target.value.replace(/[^\dA-Za-z-]/g, '').slice(0, 12))}
            placeholder="000000"
          />
        </div>
        <button type="submit" disabled={isSubmitting} className="btn-primary w-full rounded-full font-bold">
          {isSubmitting ? '確認中...' : '確認する'}
        </button>
      </form>
    </div>
  )
}

import { useState, type FormEvent } from 'react'
import { useLocation, useNavigate } from 'react-router-dom'

import { useAuth } from '@/auth/AuthProvider'
import { ApiError } from '@/lib/api-client'
import { normalizeOtpCode, validateTOTPCode } from '@/lib/validation'
import { sanitizeAppPath } from '@/lib/utils'
import { postLoginPath } from '@/lib/auth-routes'

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
  const [errorMessage, setErrorMessage] = useState('')
  const [isSubmitting, setIsSubmitting] = useState(false)

  if (status !== 'pending_2fa' && status !== 'authenticated') {
    return null
  }

  const onSubmit = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault()
    setErrorMessage('')

    const normalized = normalizeOtpCode(code)
    if (!validateTOTPCode(normalized)) {
      setErrorMessage('認証コードを入力してください')
      return
    }

    setIsSubmitting(true)
    try {
      const loggedIn = await complete2FA(normalized)
      void navigate(postLoginPath(loggedIn, from), { replace: true })
    } catch (err) {
      if (err instanceof ApiError) {
        if (err.code === 'UNAUTHORIZED') {
          setErrorMessage('2FA の有効期限が切れました。再度ログインしてください。')
        } else if (err.code === 'INVALID_2FA_CODE') {
          setErrorMessage('認証コードが正しくありません。認証アプリのコードを確認してください。')
        } else {
          setErrorMessage(err.message)
        }
      } else {
        setErrorMessage('認証に失敗しました')
      }
    } finally {
      setIsSubmitting(false)
    }
  }

  return (
    <div className="card mx-auto max-w-md">
      <div className="mb-6 text-center">
        <h2 className="mb-2 text-2xl font-bold text-gray-800">二段階認証（2FA）</h2>
        <p className="text-gray-600">認証アプリに表示される6桁のコードを入力してください</p>
      </div>

      <div className="mb-6 rounded-xl border border-purple-100 bg-purple-50 p-4 text-sm text-gray-700">
        <p className="font-semibold text-purple-900">設定方法（初回のみ）</p>
        <ol className="mt-2 list-decimal space-y-1 pl-5">
          <li>スマホに Google Authenticator（Google認証システム）等をインストール</li>
          <li>「+」→「セットアップキーを入力」でシークレットキーを登録</li>
          <li>アプリに表示される6桁の数字を下の欄に入力</li>
        </ol>
        <p className="mt-3 text-xs text-gray-500">
          デモ出品者アカウント: シークレット <code className="rounded bg-white px-1">JBSWY3DPEHPK3PXP</code>
          （30秒ごとにコードが更新されます）
        </p>
      </div>

      {errorMessage ? <p className="mb-4 text-sm text-red-600">{errorMessage}</p> : null}

      <form onSubmit={(e) => void onSubmit(e)} className="space-y-4">
        <div>
          <label htmlFor="code" className="mb-2 block text-sm font-medium text-gray-700">認証コード（6桁）</label>
          <input
            id="code"
            inputMode="numeric"
            autoComplete="one-time-code"
            className="input-field text-center text-2xl tracking-[0.4em]"
            value={code}
            onChange={(e) => setCode(normalizeOtpCode(e.target.value).slice(0, 12))}
            placeholder="000000"
          />
        </div>
        <button type="submit" disabled={isSubmitting} className="btn-primary w-full rounded-full font-bold">
          {isSubmitting ? '確認中...' : 'ログインを完了する'}
        </button>
      </form>
    </div>
  )
}

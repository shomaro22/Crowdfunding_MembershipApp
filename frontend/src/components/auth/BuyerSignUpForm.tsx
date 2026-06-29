import { useState, type FormEvent } from 'react'
import { Link, useNavigate } from 'react-router-dom'

import { registerBuyer } from '@/api/auth'
import { ApiError } from '@/lib/api-client'
import {
  evaluatePasswordStrength,
  isValidEmail,
  sanitizeDisplayText,
  sanitizeEmail,
  validatePassword,
} from '@/lib/utils'

export function BuyerSignUpForm() {
  const navigate = useNavigate()
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [confirmPassword, setConfirmPassword] = useState('')
  const [error, setError] = useState('')
  const [success, setSuccess] = useState('')
  const [isSubmitting, setIsSubmitting] = useState(false)

  const strength = evaluatePasswordStrength(password)

  const onSubmit = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault()
    setError('')
    setSuccess('')

    if (!isValidEmail(email)) {
      setError('有効なメールアドレスを入力してください')
      return
    }

    const pwCheck = validatePassword(password)
    if (!pwCheck.valid) {
      setError(pwCheck.errors[0] ?? 'パスワードが無効です')
      return
    }

    if (password !== confirmPassword) {
      setError('確認用パスワードが一致しません')
      return
    }

    setIsSubmitting(true)
    try {
      const res = await registerBuyer(sanitizeEmail(email), password)
      setSuccess(sanitizeDisplayText(res.message))
      window.setTimeout(() => { void navigate('/signin') }, 2000)
    } catch (err) {
      if (err instanceof ApiError) {
        setError(err.message)
      } else {
        setError('登録に失敗しました')
      }
    } finally {
      setIsSubmitting(false)
    }
  }

  return (
    <div className="card mx-auto max-w-md">
      <div className="mb-8 text-center">
        <h2 className="mb-2 text-2xl font-bold text-gray-800">購入者 新規登録</h2>
        <p className="text-gray-600">管理者承認後にログインできます</p>
      </div>

      {error ? <p className="mb-4 text-sm text-red-600">{error}</p> : null}
      {success ? <p className="mb-4 text-sm text-green-700">{success}</p> : null}

      <form onSubmit={(e) => void onSubmit(e)} className="space-y-4">
        <div>
          <label htmlFor="email" className="mb-2 block text-sm font-medium text-gray-700">メールアドレス</label>
          <input id="email" type="email" autoComplete="email" className="input-field" value={email} onChange={(e) => setEmail(e.target.value)} />
        </div>
        <div>
          <label htmlFor="password" className="mb-2 block text-sm font-medium text-gray-700">パスワード</label>
          <input id="password" type="password" autoComplete="new-password" className="input-field" value={password} onChange={(e) => setPassword(e.target.value)} />
          <p className="mt-1 text-xs text-gray-500">強度: {strength}</p>
        </div>
        <div>
          <label htmlFor="confirm" className="mb-2 block text-sm font-medium text-gray-700">パスワード（確認）</label>
          <input id="confirm" type="password" autoComplete="new-password" className="input-field" value={confirmPassword} onChange={(e) => setConfirmPassword(e.target.value)} />
        </div>
        <button type="submit" disabled={isSubmitting} className="btn-primary w-full rounded-full font-bold">
          {isSubmitting ? '登録中...' : '登録する'}
        </button>
      </form>

      <p className="mt-6 text-center text-sm text-gray-600">
        出品者の方は <Link to="/signup/seller" className="text-purple-700">こちら</Link>
      </p>
    </div>
  )
}

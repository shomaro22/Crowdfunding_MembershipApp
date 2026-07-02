import { useState, type FormEvent } from 'react'
import { Link, useNavigate } from 'react-router-dom'

import { checkEmailAvailable, registerBuyer } from '@/api/auth'
import { googleOAuthStartUrl } from '@/lib/api-client'
import { PasswordInput } from '@/components/common/PasswordInput'
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
  const [emailTaken, setEmailTaken] = useState(false)

  const strength = evaluatePasswordStrength(password)

  const onEmailBlur = async () => {
    if (!isValidEmail(email)) {
      setEmailTaken(false)
      return
    }
    try {
      const available = await checkEmailAvailable(sanitizeEmail(email))
      setEmailTaken(!available)
    } catch {
      setEmailTaken(false)
    }
  }

  const onSubmit = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault()
    setError('')
    setSuccess('')

    if (!isValidEmail(email)) {
      setError('有効なメールアドレスを入力してください')
      return
    }

    if (emailTaken) {
      setError('このメールアドレスは既に登録されています')
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
    <div className="mx-auto grid max-w-4xl overflow-hidden rounded-2xl bg-white shadow-xl md:grid-cols-5">
      <div className="flex flex-col justify-center bg-gradient-to-br from-purple-700 via-purple-600 to-pink-600 p-8 text-white md:col-span-2 md:p-10">
        <p className="mb-2 text-sm font-semibold uppercase tracking-wider text-purple-200">FanFunding</p>
        <h2 className="mb-4 text-2xl font-extrabold md:text-3xl">ファンアカウント作成</h2>
        <p className="text-sm leading-relaxed text-purple-100">
          お気に入りのクリエイターを応援し、限定コンテンツやメンバー特典を楽しもう。
        </p>
        <ul className="mt-6 space-y-2 text-sm text-purple-100">
          <li>・ メンバーシッププランに応援</li>
          <li>・ お気に入り・メッセージ機能</li>
          <li>・ Google アカウントでも登録可能</li>
        </ul>
      </div>

      <div className="p-8 md:col-span-3 md:p-10">
        <div className="mb-6">
          <h3 className="text-xl font-bold text-gray-800">新規登録</h3>
          <p className="mt-1 text-sm text-gray-600">メールアドレスでアカウントを作成します</p>
        </div>

        {error ? <p className="mb-4 text-sm text-red-600">{error}</p> : null}
        {success ? <p className="mb-4 text-sm text-green-700">{success}</p> : null}

        <form onSubmit={(e) => void onSubmit(e)} className="space-y-4">
          <div>
            <label htmlFor="email" className="mb-2 block text-sm font-medium text-gray-700">メールアドレス</label>
            <input
              id="email"
              type="email"
              autoComplete="email"
              className="input-field"
              value={email}
              onChange={(e) => {
                setEmail(e.target.value)
                setEmailTaken(false)
              }}
              onBlur={() => void onEmailBlur()}
              placeholder="example@email.com"
            />
            {emailTaken ? (
              <p className="mt-1 text-sm text-amber-700">
                このメールアドレスは既に登録されています。
                <Link to="/signin" className="ml-1 font-medium text-purple-700 underline">ログインはこちら</Link>
              </p>
            ) : null}
          </div>
          <div>
            <label htmlFor="password" className="mb-2 block text-sm font-medium text-gray-700">パスワード</label>
            <PasswordInput
              id="password"
              value={password}
              onChange={setPassword}
              autoComplete="new-password"
              placeholder="8文字以上・英数字記号"
            />
            <p className="mt-1 text-xs text-gray-500">強度: {strength}</p>
          </div>
          <div>
            <label htmlFor="confirm" className="mb-2 block text-sm font-medium text-gray-700">パスワード（確認）</label>
            <PasswordInput
              id="confirm"
              value={confirmPassword}
              onChange={setConfirmPassword}
              autoComplete="new-password"
              placeholder="もう一度入力"
            />
          </div>
          <button type="submit" disabled={isSubmitting || emailTaken} className="btn-primary w-full rounded-full font-bold">
            {isSubmitting ? '登録中...' : 'アカウントを作成'}
          </button>
        </form>

        <div className="my-6 flex items-center gap-3">
          <div className="h-px flex-1 bg-gray-200" />
          <span className="text-xs text-gray-500">または</span>
          <div className="h-px flex-1 bg-gray-200" />
        </div>

        <button
          type="button"
          onClick={() => { window.location.assign(googleOAuthStartUrl('/')) }}
          className="btn-secondary w-full rounded-full font-bold"
        >
          Google でアカウント作成
        </button>

        <p className="mt-6 text-center text-sm text-gray-600">
          すでにアカウントをお持ちの方は{' '}
          <Link to="/signin" className="font-medium text-purple-700">ログイン</Link>
        </p>
        <p className="mt-2 text-center text-sm text-gray-600">
          クリエイター（出品者）の方は{' '}
          <Link to="/signup/seller" className="font-medium text-purple-700">こちら</Link>
        </p>
      </div>
    </div>
  )
}

import { useCallback, useState, type FormEvent } from 'react'
import { Link, useNavigate, useSearchParams } from 'react-router-dom'

import { useAuth } from '@/auth/AuthProvider'
import { ApiError } from '@/lib/api-client'
import { googleOAuthStartUrl } from '@/lib/api-client'
import { isValidEmail, sanitizeAppPath, sanitizeEmail } from '@/lib/utils'

export function SignInForm() {
  const navigate = useNavigate()
  const [searchParams] = useSearchParams()
  const { login } = useAuth()

  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [showPassword, setShowPassword] = useState(false)
  const [fieldErrors, setFieldErrors] = useState<{ email?: string; password?: string }>({})
  const [error, setError] = useState('')
  const [isSubmitting, setIsSubmitting] = useState(false)

  const callbackPath = sanitizeAppPath(searchParams.get('from'), '/')

  const validate = useCallback((): boolean => {
    const next: { email?: string; password?: string } = {}
    if (!email.trim()) {
      next.email = 'メールアドレスを入力してください'
    } else if (!isValidEmail(email)) {
      next.email = '有効なメールアドレスを入力してください'
    }
    if (!password) {
      next.password = 'パスワードを入力してください'
    }
    setFieldErrors(next)
    return Object.keys(next).length === 0
  }, [email, password])

  const onSubmit = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault()
    setError('')
    if (!validate()) {
      return
    }

    setIsSubmitting(true)
    try {
      const result = await login(sanitizeEmail(email), password)
      if (result === '2fa') {
        void navigate('/auth/2fa', { replace: true, state: { from: callbackPath } })
        return
      }
      void navigate(callbackPath, { replace: true })
    } catch (err) {
      if (err instanceof ApiError) {
        setError(err.message)
      } else {
        setError('ログインに失敗しました')
      }
    } finally {
      setIsSubmitting(false)
    }
  }

  const onGoogleSignIn = () => {
    window.location.assign(googleOAuthStartUrl(callbackPath))
  }

  return (
    <div className="card mx-auto max-w-md">
      <div className="mb-8 text-center">
        <h2 className="mb-2 text-2xl font-bold text-gray-800">ログイン</h2>
        <p className="text-gray-600">アカウントにアクセスしてください</p>
      </div>

      {error ? (
        <div className="mb-6 rounded-2xl border border-red-200 bg-red-50 p-4 text-red-700">
          <div className="flex items-center">
            <span className="mr-3 inline-block h-2 w-2 rounded-full bg-red-500" aria-hidden />
            {error}
          </div>
        </div>
      ) : null}

      <form onSubmit={(e) => void onSubmit(e)} className="space-y-6">
        <div>
          <label htmlFor="email" className="mb-2 block text-sm font-medium text-gray-700">
            メールアドレス
          </label>
          <input
            id="email"
            type="email"
            autoComplete="email"
            value={email}
            onChange={(e) => setEmail(e.target.value)}
            className={`input-field ${fieldErrors.email ? 'error' : ''}`}
            placeholder="example@email.com"
          />
          {fieldErrors.email ? (
            <p className="mt-2 flex items-center text-sm text-red-600">
              <span className="mr-2 inline-block h-1 w-1 rounded-full bg-red-500" aria-hidden />
              {fieldErrors.email}
            </p>
          ) : null}
        </div>

        <div>
          <label htmlFor="password" className="mb-2 block text-sm font-medium text-gray-700">
            パスワード
          </label>
          <div className="relative">
            <input
              id="password"
              type={showPassword ? 'text' : 'password'}
              autoComplete="current-password"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              className={`input-field pr-12 ${fieldErrors.password ? 'error' : ''}`}
              placeholder="パスワード"
            />
            <button
              type="button"
              onClick={() => setShowPassword((v) => !v)}
              className="absolute right-3 top-1/2 -translate-y-1/2 text-gray-500 hover:text-gray-700"
              aria-label={showPassword ? 'パスワードを隠す' : 'パスワードを表示'}
            >
              {showPassword ? '🙈' : '👁️'}
            </button>
          </div>
          {fieldErrors.password ? (
            <p className="mt-2 flex items-center text-sm text-red-600">
              <span className="mr-2 inline-block h-1 w-1 rounded-full bg-red-500" aria-hidden />
              {fieldErrors.password}
            </p>
          ) : null}
        </div>

        <button
          type="submit"
          disabled={isSubmitting}
          className={`btn-primary w-full rounded-full font-bold ${isSubmitting ? 'cursor-not-allowed opacity-50' : ''}`}
        >
          {isSubmitting ? 'ログイン中...' : 'ログイン'}
        </button>
      </form>

      <div className="mt-8">
        <div className="relative">
          <div className="absolute inset-0 flex items-center">
            <div className="w-full border-t border-purple-200" />
          </div>
          <div className="relative flex justify-center text-sm">
            <span className="bg-white px-4 text-gray-500">または</span>
          </div>
        </div>

        <button type="button" onClick={onGoogleSignIn} className="btn-secondary mt-6 w-full rounded-full font-medium">
          Googleでログイン
        </button>
      </div>

      <div className="mt-8 text-center">
        <p className="text-gray-600">
          アカウントをお持ちでない方は{' '}
          <Link to="/signup" className="rounded-full px-2 py-1 font-medium text-purple-700 hover:text-purple-800">
            新規登録
          </Link>
        </p>
      </div>
    </div>
  )
}

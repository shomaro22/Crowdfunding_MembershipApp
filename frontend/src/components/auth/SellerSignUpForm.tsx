import { useState, type FormEvent } from 'react'
import { Link, useNavigate } from 'react-router-dom'

import { registerSeller } from '@/api/auth'
import {
  isValidEmail,
  sanitizeDisplayText,
  sanitizeEmail,
  validatePassword,
} from '@/lib/utils'

const ALLOWED_IMAGE_TYPES = new Set(['image/jpeg', 'image/png', 'image/webp'])
const MAX_IMAGE_BYTES = 5 * 1024 * 1024

function validateImageFile(file: File | null, label: string): string | null {
  if (!file) {
    return `${label}を選択してください`
  }
  if (!ALLOWED_IMAGE_TYPES.has(file.type)) {
    return `${label}は JPEG / PNG / WebP のみ対応しています`
  }
  if (file.size > MAX_IMAGE_BYTES) {
    return `${label}は 5MB 以下にしてください`
  }
  return null
}

export function SellerSignUpForm() {
  const navigate = useNavigate()
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [fullName, setFullName] = useState('')
  const [birthDate, setBirthDate] = useState('')
  const [address, setAddress] = useState('')
  const [frontFile, setFrontFile] = useState<File | null>(null)
  const [backFile, setBackFile] = useState<File | null>(null)
  const [error, setError] = useState('')
  const [isSubmitting, setIsSubmitting] = useState(false)

  const onSubmit = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault()
    setError('')

    if (!isValidEmail(email)) {
      setError('有効なメールアドレスを入力してください')
      return
    }

    const pwCheck = validatePassword(password)
    if (!pwCheck.valid) {
      setError(pwCheck.errors[0] ?? 'パスワードが無効です')
      return
    }

    const frontErr = validateImageFile(frontFile, '身分証（表面）')
    if (frontErr) {
      setError(frontErr)
      return
    }
    const backErr = validateImageFile(backFile, '身分証（裏面）')
    if (backErr) {
      setError(backErr)
      return
    }

    if (!frontFile || !backFile) {
      setError('身分証画像が不足しています')
      return
    }

    setIsSubmitting(true)
    try {
      await registerSeller({
        email: sanitizeEmail(email),
        password,
        full_name: sanitizeDisplayText(fullName, 200),
        birth_date: birthDate,
        address: sanitizeDisplayText(address, 500),
        identity_document_front: frontFile,
        identity_document_back: backFile,
      })
      void navigate('/signin', { replace: true, state: { message: '出品者登録を受け付けました。審査完了までお待ちください。' } })
    } catch {
      setError('出品者登録に失敗しました')
    } finally {
      setIsSubmitting(false)
    }
  }

  return (
    <div className="card mx-auto max-w-lg">
      <div className="mb-8 text-center">
        <h2 className="mb-2 text-2xl font-bold text-gray-800">出品者 新規登録</h2>
        <p className="text-gray-600">KYC 書類の審査が必要です</p>
      </div>

      {error ? <p className="mb-4 text-sm text-red-600">{error}</p> : null}

      <form onSubmit={(e) => void onSubmit(e)} className="space-y-4">
        <div>
          <label htmlFor="email" className="mb-2 block text-sm font-medium text-gray-700">メールアドレス</label>
          <input id="email" type="email" autoComplete="email" className="input-field" value={email} onChange={(e) => setEmail(e.target.value)} />
        </div>
        <div>
          <label htmlFor="password" className="mb-2 block text-sm font-medium text-gray-700">パスワード</label>
          <input id="password" type="password" autoComplete="new-password" className="input-field" value={password} onChange={(e) => setPassword(e.target.value)} />
        </div>
        <div>
          <label htmlFor="fullName" className="mb-2 block text-sm font-medium text-gray-700">氏名</label>
          <input id="fullName" type="text" className="input-field" value={fullName} onChange={(e) => setFullName(e.target.value)} />
        </div>
        <div>
          <label htmlFor="birthDate" className="mb-2 block text-sm font-medium text-gray-700">生年月日</label>
          <input id="birthDate" type="date" className="input-field" value={birthDate} onChange={(e) => setBirthDate(e.target.value)} />
        </div>
        <div>
          <label htmlFor="address" className="mb-2 block text-sm font-medium text-gray-700">住所</label>
          <textarea id="address" className="input-field min-h-24" value={address} onChange={(e) => setAddress(e.target.value)} />
        </div>
        <div>
          <label htmlFor="front" className="mb-2 block text-sm font-medium text-gray-700">身分証（表面）</label>
          <input id="front" type="file" accept="image/jpeg,image/png,image/webp" onChange={(e) => setFrontFile(e.target.files?.[0] ?? null)} />
        </div>
        <div>
          <label htmlFor="back" className="mb-2 block text-sm font-medium text-gray-700">身分証（裏面）</label>
          <input id="back" type="file" accept="image/jpeg,image/png,image/webp" onChange={(e) => setBackFile(e.target.files?.[0] ?? null)} />
        </div>
        <button type="submit" disabled={isSubmitting} className="btn-primary w-full rounded-full font-bold">
          {isSubmitting ? '送信中...' : '登録する'}
        </button>
      </form>

      <p className="mt-6 text-center text-sm text-gray-600">
        購入者の方は <Link to="/signup" className="text-purple-700">こちら</Link>
      </p>
    </div>
  )
}

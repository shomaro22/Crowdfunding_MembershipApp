import { useEffect, useState, type FormEvent } from 'react'

import {
  changePassword,
  fetchProfile,
  setup2FA,
  updateProfile,
  verify2FA,
} from '@/api/auth'
import { uploadImage } from '@/api/media'
import { useAuth } from '@/auth/AuthProvider'
import { ApiError } from '@/lib/api-client'
import { Loading } from '@/components/common/Loading'
import { PasswordInput } from '@/components/common/PasswordInput'
import { UserAvatar } from '@/components/common/UserAvatar'
import { useToast } from '@/components/common/Toast'
import { validateImageFile } from '@/lib/upload'
import { normalizeMediaUrl } from '@/lib/media-url'
import { sanitizeDisplayText, validatePassword } from '@/lib/utils'
import { normalizeOtpCode, validateTOTPCode } from '@/lib/validation'

export function ProfilePage({ heading = 'プロフィール' }: { heading?: string }) {
  const { refreshUser } = useAuth()
  const { toast } = useToast()
  const [name, setName] = useState('')
  const [avatarUrl, setAvatarUrl] = useState('')
  const [loading, setLoading] = useState(true)
  const [uploading, setUploading] = useState(false)

  const [currentPw, setCurrentPw] = useState('')
  const [newPw, setNewPw] = useState('')
  const [totpForPw, setTotpForPw] = useState('')

  const [setupSecret, setSetupSecret] = useState('')
  const [setupUrl, setSetupUrl] = useState('')
  const [verifyCode, setVerifyCode] = useState('')
  const [backupCodes, setBackupCodes] = useState<string[]>([])
  const [totpEnabled, setTotpEnabled] = useState(false)

  useEffect(() => {
    void (async () => {
      try {
        const p = await fetchProfile()
        setName(p.name ?? '')
        setAvatarUrl(p.avatar_url ?? '')
        setTotpEnabled(Boolean(p.totp_enabled))
      } catch {
        toast('プロフィールの取得に失敗しました', 'error')
      } finally {
        setLoading(false)
      }
    })()
  }, [toast])

  const onAvatarUpload = async (file: File) => {
    const check = validateImageFile(file)
    if (!check.valid) {
      toast(check.message ?? '画像が無効です', 'error')
      return
    }
    setUploading(true)
    try {
      const res = await uploadImage(file)
      const url = normalizeMediaUrl(res.url)
      setAvatarUrl(url)
      const updated = await updateProfile({
        avatar_url: url.slice(0, 2048),
      })
      setAvatarUrl(updated.avatar_url ?? url)
      await refreshUser()
      toast('アバターを更新しました', 'success')
    } catch {
      toast('画像のアップロードに失敗しました', 'error')
    } finally {
      setUploading(false)
    }
  }

  const saveProfile = async (e: FormEvent) => {
    e.preventDefault()
    try {
      const updated = await updateProfile({
        name: sanitizeDisplayText(name, 100),
        avatar_url: normalizeMediaUrl(avatarUrl.trim()).slice(0, 2048),
      })
      setName(updated.name ?? '')
      setAvatarUrl(updated.avatar_url ?? '')
      await refreshUser()
      toast('プロフィールを更新しました', 'success')
    } catch (error) {
      toast(error instanceof ApiError ? error.message : '更新に失敗しました', 'error')
    }
  }

  const onChangePassword = async (e: FormEvent) => {
    e.preventDefault()
    const v = validatePassword(newPw)
    if (!v.valid) {
      toast(v.errors[0] ?? 'パスワードが無効です', 'error')
      return
    }
    try {
      await changePassword(currentPw, newPw, normalizeOtpCode(totpForPw) || undefined)
      setCurrentPw('')
      setNewPw('')
      setTotpForPw('')
      toast('パスワードを変更しました', 'success')
    } catch (error) {
      toast(error instanceof ApiError ? error.message : '変更に失敗しました', 'error')
    }
  }

  const onSetup2FA = async () => {
    if (totpEnabled) {
      toast('二要素認証は既に有効です', 'error')
      return
    }
    try {
      const res = await setup2FA()
      setSetupSecret(res.secret)
      setSetupUrl(res.otpauth_url)
    } catch (error) {
      if (error instanceof ApiError && error.code === '2FA_ALREADY_ENABLED') {
        setTotpEnabled(true)
        toast('二要素認証は既に有効です', 'error')
        return
      }
      toast(error instanceof ApiError ? error.message : '2FA 設定の開始に失敗しました', 'error')
    }
  }

  const onVerify2FA = async (e: FormEvent) => {
    e.preventDefault()
    if (!validateTOTPCode(verifyCode)) {
      toast('6桁の認証コードを入力してください', 'error')
      return
    }
    try {
      const res = await verify2FA(normalizeOtpCode(verifyCode))
      setBackupCodes(res.backup_codes)
      setTotpEnabled(true)
      setSetupSecret('')
      toast('2FA を有効化しました', 'success')
    } catch (error) {
      toast(error instanceof ApiError ? error.message : '2FA 確認に失敗しました', 'error')
    }
  }

  if (loading) {
    return <Loading />
  }

  return (
    <div className="mx-auto max-w-2xl space-y-8">
      <h1 className="text-2xl font-bold">{heading}</h1>

      <section className="card">
        <h2 className="mb-4 text-lg font-semibold">基本情報</h2>
        <form onSubmit={(e) => void saveProfile(e)} className="space-y-4">
          <div className="flex items-center gap-4">
            <UserAvatar name={name} avatarUrl={avatarUrl || undefined} size="lg" />
            <div>
              <label className="btn-secondary inline-block cursor-pointer rounded-full px-4 py-2 text-sm">
                {uploading ? 'アップロード中...' : '画像を選択'}
                <input
                  type="file"
                  accept="image/jpeg,image/png,image/webp,image/gif"
                  className="hidden"
                  disabled={uploading}
                  onChange={(e) => {
                    const f = e.target.files?.[0]
                    if (f) {
                      void onAvatarUpload(f)
                    }
                  }}
                />
              </label>
              <p className="mt-1 text-xs text-gray-500">JPEG / PNG / WebP / GIF（最大 5MB）</p>
            </div>
          </div>
          <input className="input-field" value={name} onChange={(e) => setName(e.target.value)} placeholder="表示名" />
          <button type="submit" className="btn-primary rounded-full">保存</button>
        </form>
      </section>

      <section className="card">
        <h2 className="mb-4 text-lg font-semibold">パスワード変更</h2>
        <form onSubmit={(e) => void onChangePassword(e)} className="space-y-4">
          <PasswordInput
            id="current-password"
            value={currentPw}
            onChange={setCurrentPw}
            autoComplete="current-password"
            placeholder="現在のパスワード"
          />
          <PasswordInput
            id="new-password"
            value={newPw}
            onChange={setNewPw}
            autoComplete="new-password"
            placeholder="新しいパスワード"
          />
          <input className="input-field" value={totpForPw} onChange={(e) => setTotpForPw(normalizeOtpCode(e.target.value))} placeholder="2FA コード（有効時）" inputMode="numeric" />
          <button type="submit" className="btn-primary rounded-full">パスワード変更</button>
        </form>
      </section>

      <section className="card">
        <h2 className="mb-4 text-lg font-semibold">二要素認証 (2FA)</h2>
        {totpEnabled ? (
          <p className="text-sm text-gray-600">二要素認証は有効です。認証アプリのコードがログイン時に必要になります。</p>
        ) : !setupSecret ? (
          <button type="button" onClick={() => void onSetup2FA()} className="btn-secondary rounded-full px-6 py-2 text-sm">
            2FA を設定
          </button>
        ) : (
          <div className="space-y-3">
            <p className="text-sm text-gray-600">Authenticator アプリで QR / シークレットを登録してください。</p>
            <p className="break-all font-mono text-sm">{setupSecret}</p>
            <a href={setupUrl} className="text-sm text-purple-700">otpauth リンク</a>
            <form onSubmit={(e) => void onVerify2FA(e)} className="flex flex-wrap items-end gap-2">
              <div className="min-w-[12rem] flex-1">
                <label htmlFor="verify-code" className="mb-1 block text-xs text-gray-500">6桁コード</label>
                <input id="verify-code" className="input-field w-full" value={verifyCode} onChange={(e) => setVerifyCode(normalizeOtpCode(e.target.value))} placeholder="000000" inputMode="numeric" maxLength={10} />
              </div>
              <button type="submit" className="btn-primary shrink-0 rounded-full px-6 py-2.5 text-sm">有効化</button>
            </form>
          </div>
        )}
        {backupCodes.length > 0 ? (
          <div className="mt-4 rounded-lg border border-amber-200 bg-amber-50 p-4">
            <p className="mb-2 text-sm font-semibold text-amber-900">バックアップコード（安全な場所に保管してください）</p>
            <div className="grid grid-cols-2 gap-2 font-mono text-sm text-gray-800 sm:grid-cols-3">
              {backupCodes.map((c) => (
                <span key={c} className="rounded bg-white px-2 py-1 text-center">{c}</span>
              ))}
            </div>
          </div>
        ) : null}
      </section>
    </div>
  )
}

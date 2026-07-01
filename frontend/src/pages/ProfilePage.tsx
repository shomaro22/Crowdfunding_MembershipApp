import { useEffect, useState, type FormEvent } from 'react'

import {
  changePassword,
  fetchProfile,
  setup2FA,
  updateProfile,
  verify2FA,
} from '@/api/auth'
import { useAuth } from '@/auth/AuthProvider'
import { ApiError } from '@/lib/api-client'
import { Loading } from '@/components/common/Loading'
import { useToast } from '@/components/common/Toast'
import { sanitizeDisplayText, validatePassword } from '@/lib/utils'
import { validateTOTPCode } from '@/lib/validation'

export function ProfilePage() {
  const { refreshUser } = useAuth()
  const { toast } = useToast()
  const [name, setName] = useState('')
  const [avatarUrl, setAvatarUrl] = useState('')
  const [loading, setLoading] = useState(true)

  const [currentPw, setCurrentPw] = useState('')
  const [newPw, setNewPw] = useState('')
  const [totpForPw, setTotpForPw] = useState('')

  const [setupSecret, setSetupSecret] = useState('')
  const [setupUrl, setSetupUrl] = useState('')
  const [verifyCode, setVerifyCode] = useState('')
  const [backupCodes, setBackupCodes] = useState<string[]>([])

  useEffect(() => {
    void (async () => {
      try {
        const p = await fetchProfile()
        setName(p.name ?? '')
        setAvatarUrl(p.avatar_url ?? '')
      } catch {
        toast('プロフィールの取得に失敗しました', 'error')
      } finally {
        setLoading(false)
      }
    })()
  }, [])

  const saveProfile = async (e: FormEvent) => {
    e.preventDefault()
    try {
      await updateProfile({
        name: sanitizeDisplayText(name, 100),
        avatar_url: avatarUrl.trim().slice(0, 2048),
      })
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
      await changePassword(currentPw, newPw, totpForPw || undefined)
      setCurrentPw('')
      setNewPw('')
      setTotpForPw('')
      toast('パスワードを変更しました', 'success')
    } catch (error) {
      toast(error instanceof ApiError ? error.message : '変更に失敗しました', 'error')
    }
  }

  const onSetup2FA = async () => {
    try {
      const res = await setup2FA()
      setSetupSecret(res.secret)
      setSetupUrl(res.otpauth_url)
    } catch (error) {
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
      const res = await verify2FA(verifyCode.trim())
      setBackupCodes(res.backup_codes)
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
      <h1 className="text-2xl font-bold">プロフィール</h1>

      <section className="card">
        <h2 className="mb-4 text-lg font-semibold">基本情報</h2>
        <form onSubmit={(e) => void saveProfile(e)} className="space-y-4">
          <input className="input-field" value={name} onChange={(e) => setName(e.target.value)} placeholder="表示名" />
          <input className="input-field" value={avatarUrl} onChange={(e) => setAvatarUrl(e.target.value)} placeholder="アバター URL（https）" />
          <button type="submit" className="btn-primary rounded-full">保存</button>
        </form>
      </section>

      <section className="card">
        <h2 className="mb-4 text-lg font-semibold">パスワード変更</h2>
        <form onSubmit={(e) => void onChangePassword(e)} className="space-y-4">
          <input type="password" autoComplete="current-password" className="input-field" value={currentPw} onChange={(e) => setCurrentPw(e.target.value)} placeholder="現在のパスワード" />
          <input type="password" autoComplete="new-password" className="input-field" value={newPw} onChange={(e) => setNewPw(e.target.value)} placeholder="新しいパスワード" />
          <input className="input-field" value={totpForPw} onChange={(e) => setTotpForPw(e.target.value)} placeholder="2FA コード（有効時）" inputMode="numeric" />
          <button type="submit" className="btn-primary rounded-full">パスワード変更</button>
        </form>
      </section>

      <section className="card">
        <h2 className="mb-4 text-lg font-semibold">二要素認証 (2FA)</h2>
        {!setupSecret ? (
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
                <input id="verify-code" className="input-field w-full" value={verifyCode} onChange={(e) => setVerifyCode(e.target.value)} placeholder="000000" inputMode="numeric" maxLength={10} />
              </div>
              <button type="submit" className="btn-primary shrink-0 rounded-full px-6 py-2.5 text-sm">有効化</button>
            </form>
          </div>
        )}
        {backupCodes.length > 0 ? (
          <ul className="mt-4 list-inside list-disc font-mono text-sm">
            {backupCodes.map((c) => <li key={c}>{c}</li>)}
          </ul>
        ) : null}
      </section>
    </div>
  )
}

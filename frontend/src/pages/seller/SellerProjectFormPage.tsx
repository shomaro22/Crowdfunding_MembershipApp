import { useEffect, useState, type FormEvent } from 'react'
import { Link, useNavigate, useParams } from 'react-router-dom'

import { createProject, getProject, updateProject } from '@/api/projects'
import { uploadImage } from '@/api/media'
import { Loading } from '@/components/common/Loading'
import { ApiError } from '@/lib/api-client'
import { normalizeMediaUrl } from '@/lib/media-url'
import { planTierLabel } from '@/lib/plans'
import { formatCurrencyYen, sanitizeDisplayText } from '@/lib/utils'
import {
  validatePlanPrice,
  validatePositiveAmount,
  validateProjectTitle,
  parseUUID,
} from '@/lib/validation'
import { validateImageFile } from '@/lib/upload'
import type { DurationType, MembershipType, UpdateProjectPayload } from '@/types/api'

const MEMBERSHIP_TYPES: MembershipType[] = ['BRONZE', 'SILVER', 'GOLD']
const STEPS = ['基本情報', '募集設定', 'メンバーシップ', '確認'] as const

interface PlanForm {
  membership_type: MembershipType
  price: string
  benefits: string
  max_supporters: string
}

export function SellerProjectFormPage() {
  const { projectId } = useParams<{ projectId: string }>()
  const navigate = useNavigate()
  const isEdit = projectId !== undefined && projectId !== 'new'
  const [step, setStep] = useState(0)
  const [loading, setLoading] = useState(isEdit)
  const [error, setError] = useState('')
  const [submitting, setSubmitting] = useState(false)

  const [title, setTitle] = useState('')
  const [description, setDescription] = useState('')
  const [targetAmount, setTargetAmount] = useState('100000')
  const [durationType, setDurationType] = useState<DurationType>('UNLIMITED')
  const [startDate, setStartDate] = useState('')
  const [endDate, setEndDate] = useState('')
  const [imageUrl, setImageUrl] = useState('')
  const [plans, setPlans] = useState<PlanForm[]>(
    MEMBERSHIP_TYPES.map((t) => ({ membership_type: t, price: '500', benefits: '', max_supporters: '' })),
  )

  useEffect(() => {
    if (!isEdit) {
      setStartDate(new Date().toISOString().slice(0, 16))
      return
    }
    const id = parseUUID(projectId)
    if (!id) {
      setError('ID が不正です')
      setLoading(false)
      return
    }
    void (async () => {
      try {
        const p = await getProject(id)
        setTitle(p.title)
        setDescription(p.description)
        setTargetAmount(String(p.target_amount))
        setDurationType(p.duration_type)
        setStartDate(p.start_date.slice(0, 16))
        setEndDate(p.end_date?.slice(0, 16) ?? '')
        setImageUrl(p.image_url ?? '')
        if (p.plans.length >= 3) {
          setPlans(p.plans.slice(0, 3).map((pl) => ({
            membership_type: pl.membership_type,
            price: String(pl.price),
            benefits: pl.benefits,
            max_supporters: pl.max_supporters ? String(pl.max_supporters) : '',
          })))
        }
      } catch {
        setError('プロジェクトの取得に失敗しました')
      } finally {
        setLoading(false)
      }
    })()
  }, [isEdit, projectId])

  const onUpload = async (file: File) => {
    const check = validateImageFile(file)
    if (!check.valid) {
      setError(check.error ?? '画像が無効です')
      return
    }
    try {
      const res = await uploadImage(file)
      setImageUrl(normalizeMediaUrl(res.url))
    } catch {
      setError('画像アップロードに失敗しました')
    }
  }

  const validateStep = (targetStep: number): string | null => {
    if (targetStep === 0) {
      const titleErr = validateProjectTitle(title)
      if (titleErr) {
        return titleErr
      }
      if (!sanitizeDisplayText(description, 10000)) {
        return '説明を入力してください'
      }
    }
    if (targetStep === 1) {
      const target = Number.parseInt(targetAmount, 10)
      const targetErr = validatePositiveAmount(target, '目標金額')
      if (targetErr) {
        return targetErr
      }
      if (!startDate) {
        return '開始日時を入力してください'
      }
      if (durationType === 'LIMITED' && !endDate) {
        return '終了日時を入力してください'
      }
    }
    if (targetStep === 2 && !isEdit) {
      for (const pl of plans) {
        const price = Number.parseInt(pl.price, 10)
        const priceErr = validatePlanPrice(price)
        if (priceErr) {
          return priceErr
        }
        if (!sanitizeDisplayText(pl.benefits, 2000)) {
          return `${planTierLabel(pl.membership_type)} の特典内容を入力してください`
        }
      }
    }
    return null
  }

  const goNext = () => {
    setError('')
    const err = validateStep(step)
    if (err) {
      setError(err)
      return
    }
    setStep((s) => Math.min(s + 1, STEPS.length - 1))
  }

  const goBack = () => {
    setError('')
    setStep((s) => Math.max(s - 1, 0))
  }

  const onSubmit = async (e: FormEvent) => {
    e.preventDefault()
    setError('')
    for (let i = 0; i < STEPS.length - 1; i++) {
      const err = validateStep(i)
      if (err) {
        setError(err)
        setStep(i)
        return
      }
    }

    const target = Number.parseInt(targetAmount, 10)
    setSubmitting(true)
    try {
      if (isEdit && projectId) {
        const updateBody: UpdateProjectPayload = {
          title: sanitizeDisplayText(title, 200),
          description: sanitizeDisplayText(description, 10000),
          target_amount: target,
          duration_type: durationType,
          start_date: new Date(startDate).toISOString(),
          end_date: durationType === 'LIMITED' && endDate ? new Date(endDate).toISOString() : null,
          image_url: imageUrl ? normalizeMediaUrl(imageUrl) : undefined,
          images: imageUrl ? [{ image_url: normalizeMediaUrl(imageUrl), is_main: true, order: 0 }] : undefined,
        }
        await updateProject(projectId, updateBody)
      } else {
        const planPayload = plans.map((pl) => {
          const price = Number.parseInt(pl.price, 10)
          const max = pl.max_supporters ? Number.parseInt(pl.max_supporters, 10) : undefined
          return {
            membership_type: pl.membership_type,
            price,
            benefits: sanitizeDisplayText(pl.benefits, 2000),
            max_supporters: max,
          }
        })
        await createProject({
          title: sanitizeDisplayText(title, 200),
          description: sanitizeDisplayText(description, 10000),
          target_amount: target,
          duration_type: durationType,
          start_date: new Date(startDate).toISOString(),
          end_date: durationType === 'LIMITED' && endDate ? new Date(endDate).toISOString() : null,
          image_url: imageUrl ? normalizeMediaUrl(imageUrl) : undefined,
          images: imageUrl ? [{ image_url: normalizeMediaUrl(imageUrl), is_main: true, order: 0 }] : undefined,
          plans: planPayload,
        })
      }
      void navigate('/seller/projects')
    } catch (err) {
      setError(err instanceof Error ? err.message : err instanceof ApiError ? err.message : '保存に失敗しました')
    } finally {
      setSubmitting(false)
    }
  }

  if (loading) {
    return <Loading />
  }

  return (
    <div className="mx-auto max-w-5xl">
      <div className="grid overflow-hidden rounded-2xl bg-white shadow-xl md:grid-cols-5">
        <div className="flex flex-col justify-between bg-gradient-to-br from-purple-700 via-purple-600 to-pink-600 p-8 text-white md:col-span-2">
          <div>
            <h1 className="mb-2 text-2xl font-bold">{isEdit ? '商品ページ編集' : '新規プロジェクト'}</h1>
            <p className="text-sm text-purple-100">
              {isEdit
                ? '公開ページのタイトル・説明・画像を更新できます。'
                : 'ステップに沿ってメンバーシップ付きプロジェクトを作成します。'}
            </p>
          </div>
          <ol className="mt-8 space-y-3">
            {STEPS.map((label, i) => (
              <li
                key={label}
                className={`flex items-center gap-3 text-sm ${i === step ? 'font-bold text-white' : i < step ? 'text-purple-200' : 'text-purple-300/70'}`}
              >
                <span className={`flex h-7 w-7 shrink-0 items-center justify-center rounded-full text-xs font-bold ${i <= step ? 'bg-white text-purple-700' : 'border border-purple-300/50'}`}>
                  {i + 1}
                </span>
                {label}
              </li>
            ))}
          </ol>
        </div>

        <div className="p-6 md:col-span-3 md:p-8">
          {error ? <p className="mb-4 text-sm text-red-600">{error}</p> : null}

          <form onSubmit={(e) => void onSubmit(e)} className="space-y-5">
            {step === 0 ? (
              <>
                <div>
                  <label className="mb-1 block text-sm font-medium">タイトル</label>
                  <input className="input-field" value={title} onChange={(e) => setTitle(e.target.value)} placeholder="プロジェクト名" />
                </div>
                <div>
                  <label className="mb-1 block text-sm font-medium">説明</label>
                  <textarea className="input-field min-h-32" value={description} onChange={(e) => setDescription(e.target.value)} placeholder="クリエイター活動の紹介、メンバー特典など" />
                </div>
                <div>
                  <label className="mb-1 block text-sm font-medium">メイン画像</label>
                  <input type="file" accept="image/jpeg,image/png,image/webp" onChange={(e) => {
                    const f = e.target.files?.[0]
                    if (f) {
                      void onUpload(f)
                    }
                  }} />
                  {imageUrl ? <img src={imageUrl} alt="" className="mt-2 h-40 w-full rounded-lg object-cover" /> : null}
                </div>
              </>
            ) : null}

            {step === 1 ? (
              <>
                <div className="grid gap-4 sm:grid-cols-2">
                  <div>
                    <label className="mb-1 block text-sm font-medium">目標金額（円）</label>
                    <input className="input-field" type="number" min={0} value={targetAmount} onChange={(e) => setTargetAmount(e.target.value)} />
                  </div>
                  <div>
                    <label className="mb-1 block text-sm font-medium">公開期間</label>
                    <select className="input-field" value={durationType} onChange={(e) => setDurationType(e.target.value as DurationType)}>
                      <option value="UNLIMITED">無期限</option>
                      <option value="LIMITED">期限あり</option>
                    </select>
                  </div>
                </div>
                <div>
                  <label className="mb-1 block text-sm font-medium">開始日時</label>
                  <input className="input-field" type="datetime-local" value={startDate} onChange={(e) => setStartDate(e.target.value)} />
                </div>
                {durationType === 'LIMITED' ? (
                  <div>
                    <label className="mb-1 block text-sm font-medium">終了日時</label>
                    <input className="input-field" type="datetime-local" value={endDate} onChange={(e) => setEndDate(e.target.value)} />
                  </div>
                ) : null}
              </>
            ) : null}

            {step === 2 ? (
              <fieldset className="space-y-3">
                <legend className="mb-2 text-sm font-semibold">メンバーシッププラン</legend>
                {isEdit ? (
                  <p className="mb-3 text-sm text-amber-700">公開後のプラン価格・特典の変更はできません。表示のみです。</p>
                ) : null}
                {plans.map((pl, i) => (
                  <div key={pl.membership_type} className={`card border-l-4 p-4 ${pl.membership_type === 'GOLD' ? 'border-yellow-500' : pl.membership_type === 'SILVER' ? 'border-gray-400' : 'border-orange-400'}`}>
                    <p className="mb-2 font-bold text-purple-800">{planTierLabel(pl.membership_type)}</p>
                    <div className="grid gap-2 sm:grid-cols-2">
                      <input className="input-field" type="number" value={pl.price} disabled={isEdit} onChange={(e) => {
                        setPlans((prev) => prev.map((row, idx) => idx === i ? { ...row, price: e.target.value } : row))
                      }} placeholder="月額（円）" />
                      <input className="input-field" type="number" value={pl.max_supporters} disabled={isEdit} onChange={(e) => {
                        setPlans((prev) => prev.map((row, idx) => idx === i ? { ...row, max_supporters: e.target.value } : row))
                      }} placeholder="上限人数（任意）" />
                      <input className="input-field sm:col-span-2" value={pl.benefits} disabled={isEdit} onChange={(e) => {
                        setPlans((prev) => prev.map((row, idx) => idx === i ? { ...row, benefits: e.target.value } : row))
                      }} placeholder="特典内容" />
                    </div>
                  </div>
                ))}
              </fieldset>
            ) : null}

            {step === 3 ? (
              <div className="space-y-3 rounded-xl border border-purple-100 bg-purple-50/50 p-4 text-sm">
                <p className="font-semibold text-purple-900">{title}</p>
                <p className="line-clamp-3 text-gray-600">{description}</p>
                <p>目標: {formatCurrencyYen(Number.parseInt(targetAmount, 10) || 0)} · {durationType === 'UNLIMITED' ? '無期限' : '期限あり'}</p>
                {!isEdit ? (
                  <ul className="space-y-1 border-t border-purple-100 pt-3">
                    {plans.map((pl) => (
                      <li key={pl.membership_type}>
                        {planTierLabel(pl.membership_type)}: {formatCurrencyYen(Number.parseInt(pl.price, 10) || 0)}/月 — {pl.benefits}
                      </li>
                    ))}
                  </ul>
                ) : null}
              </div>
            ) : null}

            <div className="flex gap-3 pt-2">
              {step > 0 ? (
                <button type="button" onClick={goBack} className="btn-secondary rounded-full px-6">戻る</button>
              ) : (
                <Link to="/seller/projects" className="btn-secondary rounded-full px-6">キャンセル</Link>
              )}
              {step < STEPS.length - 1 ? (
                <button type="button" onClick={goNext} className="btn-primary flex-1 rounded-full font-bold">次へ</button>
              ) : (
                <button type="submit" disabled={submitting} className="btn-primary flex-1 rounded-full font-bold">
                  {submitting ? '保存中...' : isEdit ? '保存する' : '公開申請する'}
                </button>
              )}
            </div>
          </form>
        </div>
      </div>
    </div>
  )
}

import { useEffect, useState, type FormEvent } from 'react'
import { Link, useNavigate, useParams } from 'react-router-dom'

import { createProject, getProject, updateProject } from '@/api/projects'
import { uploadImage } from '@/api/media'
import { Loading } from '@/components/common/Loading'
import { ApiError } from '@/lib/api-client'
import { sanitizeDisplayText } from '@/lib/utils'
import {
  validatePlanPrice,
  validatePositiveAmount,
  validateProjectTitle,
  parseUUID,
} from '@/lib/validation'
import { validateImageFile } from '@/lib/upload'
import type { DurationType, MembershipType } from '@/types/api'

const MEMBERSHIP_TYPES: MembershipType[] = ['BRONZE', 'SILVER', 'GOLD']

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
      setImageUrl(res.url)
    } catch {
      setError('画像アップロードに失敗しました')
    }
  }

  const onSubmit = async (e: FormEvent) => {
    e.preventDefault()
    setError('')
    const titleErr = validateProjectTitle(title)
    if (titleErr) {
      setError(titleErr)
      return
    }
    const target = Number.parseInt(targetAmount, 10)
    const targetErr = validatePositiveAmount(target, '目標金額')
    if (targetErr) {
      setError(targetErr)
      return
    }
    const planPayload = plans.map((pl) => {
      const price = Number.parseInt(pl.price, 10)
      const priceErr = validatePlanPrice(price)
      if (priceErr) {
        throw new Error(priceErr)
      }
      const max = pl.max_supporters ? Number.parseInt(pl.max_supporters, 10) : undefined
      return {
        membership_type: pl.membership_type,
        price,
        benefits: sanitizeDisplayText(pl.benefits, 2000),
        max_supporters: max,
      }
    })

    setSubmitting(true)
    try {
      const body = {
        title: sanitizeDisplayText(title, 200),
        description: sanitizeDisplayText(description, 10000),
        target_amount: target,
        duration_type: durationType,
        start_date: new Date(startDate).toISOString(),
        end_date: durationType === 'LIMITED' && endDate ? new Date(endDate).toISOString() : null,
        image_url: imageUrl || undefined,
        images: imageUrl ? [{ image_url: imageUrl, is_main: true, order: 0 }] : undefined,
        plans: planPayload,
      }
      if (isEdit && projectId) {
        await updateProject(projectId, body)
      } else {
        await createProject(body)
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
        <div className="bg-gradient-to-br from-purple-700 to-pink-600 p-8 text-white md:col-span-2">
          <h1 className="mb-2 text-2xl font-bold">{isEdit ? 'プロジェクト編集' : '新規プロジェクト'}</h1>
          <p className="text-sm text-purple-100">タイトル・説明・3つのメンバーシッププラン（BRONZE / SILVER / GOLD）を設定します。</p>
        </div>
        <div className="p-6 md:col-span-3 md:p-8">
      {error ? <p className="mb-4 text-sm text-red-600">{error}</p> : null}
      <form onSubmit={(e) => void onSubmit(e)} className="space-y-5">
        <div>
          <label className="mb-1 block text-sm font-medium">タイトル</label>
          <input className="input-field" value={title} onChange={(e) => setTitle(e.target.value)} placeholder="プロジェクト名" required />
        </div>
        <div>
          <label className="mb-1 block text-sm font-medium">説明</label>
          <textarea className="input-field min-h-32" value={description} onChange={(e) => setDescription(e.target.value)} placeholder="クリエイター活動の紹介、メンバー特典など" required />
        </div>
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
        <input className="input-field" type="datetime-local" value={startDate} onChange={(e) => setStartDate(e.target.value)} />
        {durationType === 'LIMITED' ? (
          <input className="input-field" type="datetime-local" value={endDate} onChange={(e) => setEndDate(e.target.value)} />
        ) : null}
        <div>
          <label className="mb-1 block text-sm font-medium">メイン画像</label>
          <input type="file" accept="image/jpeg,image/png,image/webp,image/gif" onChange={(e) => {
            const f = e.target.files?.[0]
            if (f) {
              void onUpload(f)
            }
          }} />
          {imageUrl ? <img src={imageUrl} alt="" className="mt-2 h-32 rounded object-cover" /> : null}
        </div>
        <fieldset className="space-y-3">
          <legend className="mb-2 text-sm font-semibold">メンバーシッププラン</legend>
          {plans.map((pl, i) => (
            <div key={pl.membership_type} className={`card border-l-4 p-4 ${pl.membership_type === 'GOLD' ? 'border-yellow-500' : pl.membership_type === 'SILVER' ? 'border-gray-400' : 'border-orange-400'}`}>
              <p className="mb-2 font-bold text-purple-800">{pl.membership_type}</p>
              <div className="grid gap-2 sm:grid-cols-2">
                <input className="input-field" type="number" value={pl.price} onChange={(e) => {
                  setPlans((prev) => prev.map((row, idx) => idx === i ? { ...row, price: e.target.value } : row))
                }} placeholder="月額（円）" />
                <input className="input-field sm:col-span-2" value={pl.benefits} onChange={(e) => {
                  setPlans((prev) => prev.map((row, idx) => idx === i ? { ...row, benefits: e.target.value } : row))
                }} placeholder="特典内容" />
              </div>
            </div>
          ))}
        </fieldset>
        <div className="flex gap-3 pt-2">
          <button type="submit" disabled={submitting} className="btn-primary rounded-full px-8">{submitting ? '保存中...' : '公開申請する'}</button>
          <Link to="/seller/projects" className="btn-secondary rounded-full px-6">キャンセル</Link>
        </div>
      </form>
        </div>
      </div>
    </div>
  )
}

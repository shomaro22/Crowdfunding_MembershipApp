import { useEffect, useState } from 'react'
import { Link, useParams } from 'react-router-dom'

import { getProject } from '@/api/projects'
import { createReport } from '@/api/reports'
import { checkFavorite, toggleFavorite } from '@/api/supporters'
import { useAuth } from '@/auth/AuthProvider'
import { ErrorMessage } from '@/components/common/ErrorMessage'
import { Loading } from '@/components/common/Loading'
import { ApiError } from '@/lib/api-client'
import { formatCurrencyYen, sanitizeDisplayText } from '@/lib/utils'
import { parseUUID } from '@/lib/validation'
import type { ProjectDetail } from '@/types/api'

export function ProjectDetailPage() {
  const { projectId } = useParams<{ projectId: string }>()
  const { status, user, hasRole } = useAuth()
  const [project, setProject] = useState<ProjectDetail | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')
  const [favorited, setFavorited] = useState(false)
  const [reportOpen, setReportOpen] = useState(false)
  const [reportReason, setReportReason] = useState('')
  const [actionMsg, setActionMsg] = useState('')

  useEffect(() => {
    const id = parseUUID(projectId)
    if (!id) {
      setError('プロジェクト ID が不正です')
      setLoading(false)
      return
    }

    void (async () => {
      try {
        const detail = await getProject(id)
        setProject(detail)
        if (status === 'authenticated' && hasRole('BUYER')) {
          const fav = await checkFavorite(id)
          setFavorited(fav.is_favorited)
        }
      } catch {
        setError('プロジェクトの取得に失敗しました')
      } finally {
        setLoading(false)
      }
    })()
  }, [projectId, status, hasRole])

  const onToggleFavorite = async () => {
    if (!project) {
      return
    }
    try {
      const res = await toggleFavorite(project.id, favorited ? 'remove' : 'add')
      setFavorited(res.is_favorited)
    } catch (err) {
      setActionMsg(err instanceof ApiError ? err.message : 'お気に入り操作に失敗しました')
    }
  }

  const onReport = async () => {
    if (!project) {
      return
    }
    const reason = sanitizeDisplayText(reportReason, 200)
    if (!reason) {
      setActionMsg('通報理由を入力してください')
      return
    }
    try {
      await createReport('PROJECT', project.id, reason)
      setReportOpen(false)
      setReportReason('')
      setActionMsg('通報を受け付けました')
    } catch (err) {
      setActionMsg(err instanceof ApiError ? err.message : '通報に失敗しました')
    }
  }

  if (loading) {
    return <Loading />
  }

  if (error || !project) {
    return <ErrorMessage message={error || 'プロジェクトが見つかりません'} />
  }

  const progress = project.target_amount > 0
    ? Math.min(100, Math.round((project.current_amount / project.target_amount) * 100))
    : 0

  const mainImage = project.images.find((img) => img.is_main)?.image_url ?? project.image_url

  return (
    <div className="mx-auto max-w-4xl">
      <div className="card overflow-hidden p-0">
        {mainImage ? (
          <img src={mainImage} alt="" className="h-72 w-full object-cover" referrerPolicy="no-referrer" />
        ) : (
          <div className="flex h-72 items-center justify-center bg-purple-100 text-purple-400">No image</div>
        )}
        {project.images.length > 1 ? (
          <div className="flex gap-2 overflow-x-auto p-4">
            {project.images.map((img) => (
              <img key={img.id} src={img.image_url} alt="" className="h-16 w-24 rounded object-cover" referrerPolicy="no-referrer" />
            ))}
          </div>
        ) : null}

        <div className="p-6">
          <div className="mb-4 flex flex-wrap items-start justify-between gap-3">
            <h1 className="text-3xl font-bold text-gray-900">{sanitizeDisplayText(project.title, 200)}</h1>
            {status === 'authenticated' && user?.role === 'BUYER' ? (
              <div className="flex gap-2">
                <button type="button" onClick={() => void onToggleFavorite()} className="btn-secondary rounded-full text-sm">
                  {favorited ? '★ お気に入り解除' : '☆ お気に入り'}
                </button>
                <button type="button" onClick={() => setReportOpen((v) => !v)} className="btn-secondary rounded-full text-sm">通報</button>
              </div>
            ) : null}
          </div>

          {project.seller ? (
            <p className="mb-4 text-sm text-gray-600">出品者: {sanitizeDisplayText(project.seller.name ?? project.seller.email, 100)}</p>
          ) : null}

          <p className="mb-6 whitespace-pre-wrap text-gray-700">{sanitizeDisplayText(project.description, 10000)}</p>

          <div className="mb-6">
            <div className="mb-2 flex justify-between text-sm text-gray-600">
              <span>達成率 {progress}% · 支援者 {project.supporter_count}</span>
              <span>{formatCurrencyYen(project.current_amount)} / {formatCurrencyYen(project.target_amount)}</span>
            </div>
            <div className="h-3 overflow-hidden rounded-full bg-purple-100">
              <div className="h-full bg-purple-600" style={{ width: `${progress}%` }} />
            </div>
          </div>

          {reportOpen ? (
            <div className="mb-6 rounded-lg border border-red-100 bg-red-50 p-4">
              <input className="input-field mb-2" value={reportReason} onChange={(e) => setReportReason(e.target.value)} placeholder="通報理由" maxLength={200} />
              <button type="button" onClick={() => void onReport()} className="btn-primary rounded-full text-sm">送信</button>
            </div>
          ) : null}
          {actionMsg ? <p className="mb-4 text-sm text-purple-800">{actionMsg}</p> : null}

          <h2 className="mb-3 text-xl font-semibold text-gray-900">メンバーシッププラン</h2>
          <div className="grid gap-4 md:grid-cols-3">
            {project.plans.filter((p) => p.is_active).map((plan) => (
              <div key={plan.id} className="rounded-xl border border-purple-100 p-4">
                <p className="font-bold text-purple-800">{plan.membership_type}</p>
                <p className="my-2 text-2xl font-extrabold">{formatCurrencyYen(plan.price)}</p>
                <p className="text-sm text-gray-600">{sanitizeDisplayText(plan.benefits, 2000)}</p>
                {status === 'authenticated' && user?.role === 'BUYER' ? (
                  <Link
                    to={`/checkout/${project.id}?plan_id=${encodeURIComponent(plan.id)}`}
                    className="btn-primary mt-4 inline-flex w-full justify-center rounded-full text-sm"
                  >
                    このプランで応援
                  </Link>
                ) : (
                  <Link to="/signin" className="btn-secondary mt-4 inline-flex w-full justify-center rounded-full text-sm">ログインして応援</Link>
                )}
              </div>
            ))}
          </div>
        </div>
      </div>
    </div>
  )
}

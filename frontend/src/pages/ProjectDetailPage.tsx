import { useEffect, useState } from 'react'
import { Link, useNavigate, useParams } from 'react-router-dom'

import { getProject, listPublicProjects } from '@/api/projects'
import { createReport } from '@/api/reports'
import { checkFavorite, toggleFavorite } from '@/api/supporters'
import { useAuth } from '@/auth/AuthProvider'
import { ErrorMessage } from '@/components/common/ErrorMessage'
import { Loading } from '@/components/common/Loading'
import { ApiError } from '@/lib/api-client'
import {
  formatCurrencyYen,
  fundingProgressBarWidth,
  fundingProgressPercent,
  sanitizeDisplayText,
} from '@/lib/utils'
import { parseUUID } from '@/lib/validation'
import type { ProjectDetail, ProjectSummary } from '@/types/api'

function membershipLabel(type: string): string {
  switch (type) {
    case 'BRONZE':
      return 'ブロンズ'
    case 'SILVER':
      return 'シルバー'
    case 'GOLD':
      return 'ゴールド'
    default:
      return type
  }
}

export function ProjectDetailPage() {
  const { projectId } = useParams<{ projectId: string }>()
  const navigate = useNavigate()
  const { status, user, hasRole } = useAuth()
  const [project, setProject] = useState<ProjectDetail | null>(null)
  const [related, setRelated] = useState<ProjectSummary[]>([])
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
        if (detail.seller?.id) {
          const list = await listPublicProjects({ limit: 50 })
          setRelated(
            list.projects.filter(
              (p) => p.id !== detail.id && p.seller?.id === detail.seller?.id,
            ),
          )
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
    if (status !== 'authenticated' || !hasRole('BUYER')) {
      void navigate('/signin', { state: { from: `/projects/${project.id}` } })
      return
    }
    try {
      const res = await toggleFavorite(project.id, favorited ? 'remove' : 'add')
      setFavorited(res.is_favorited)
      setActionMsg(res.is_favorited ? 'お気に入りに追加しました' : 'お気に入りを解除しました')
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

  const goCheckout = (planId: string) => {
    if (!project) {
      return
    }
    if (status === 'authenticated' && user?.role === 'BUYER') {
      void navigate(`/checkout/${project.id}?plan_id=${encodeURIComponent(planId)}`)
      return
    }
    void navigate('/signin', { state: { from: `/checkout/${project.id}?plan_id=${encodeURIComponent(planId)}` } })
  }

  if (loading) {
    return <Loading />
  }

  if (error || !project) {
    return <ErrorMessage message={error || 'プロジェクトが見つかりません'} />
  }

  const progressPercent = fundingProgressPercent(project.current_amount, project.target_amount)
  const barWidth = fundingProgressBarWidth(progressPercent)
  const mainImage = project.images.find((img) => img.is_main)?.image_url ?? project.image_url
  const canFavorite = status === 'authenticated' && hasRole('BUYER')

  return (
    <div className="mx-auto max-w-6xl">
      <div className="grid gap-8 lg:grid-cols-10">
        <div className="lg:col-span-7">
          <div className="card overflow-hidden p-0">
            {mainImage ? (
              <img src={mainImage} alt="" className="h-80 w-full object-cover" referrerPolicy="no-referrer" />
            ) : (
              <div className="flex h-80 items-center justify-center bg-purple-100 text-purple-400">画像なし</div>
            )}
            {project.images.length > 1 ? (
              <div className="flex gap-2 overflow-x-auto p-4">
                {project.images.map((img) => (
                  <img
                    key={img.id}
                    src={img.image_url}
                    alt=""
                    className="h-16 w-24 rounded object-cover"
                    referrerPolicy="no-referrer"
                  />
                ))}
              </div>
            ) : null}

            <div className="p-6">
              <div className="mb-4 flex flex-wrap items-start justify-between gap-3">
                <h1 className="text-3xl font-bold text-gray-900">
                  {sanitizeDisplayText(project.title, 200)}
                </h1>
                <div className="flex gap-2">
                  <button
                    type="button"
                    onClick={() => void onToggleFavorite()}
                    className={`rounded-full px-4 py-2 text-sm font-bold transition-colors ${
                      favorited
                        ? 'bg-pink-100 text-pink-700 ring-2 ring-pink-300'
                        : 'btn-secondary'
                    }`}
                  >
                    {favorited ? '♥ お気に入り済み' : '♡ お気に入り'}
                  </button>
                  {canFavorite ? (
                    <button
                      type="button"
                      onClick={() => setReportOpen((v) => !v)}
                      className="btn-secondary rounded-full text-sm"
                    >
                      通報
                    </button>
                  ) : null}
                </div>
              </div>

              {project.seller ? (
                <p className="mb-4 text-sm text-gray-600">
                  クリエイター:{' '}
                  <span className="font-semibold text-purple-800">
                    {sanitizeDisplayText(project.seller.name ?? project.seller.email, 100)}
                  </span>
                </p>
              ) : null}

              <p className="mb-6 whitespace-pre-wrap text-gray-700">
                {sanitizeDisplayText(project.description, 10000)}
              </p>

              <div className="mb-6">
                <div className="mb-2 flex justify-between text-sm text-gray-600">
                  <span>
                    達成率 {progressPercent}%
                    {progressPercent > 100 ? '（目標達成！）' : ''}
                    {' · '}
                    支援者 {project.supporter_count}人
                  </span>
                  <span>
                    {formatCurrencyYen(project.current_amount)} / {formatCurrencyYen(project.target_amount)}
                  </span>
                </div>
                <div className="h-3 overflow-hidden rounded-full bg-purple-100">
                  <div className="h-full bg-purple-600" style={{ width: `${barWidth}%` }} />
                </div>
              </div>

              {reportOpen ? (
                <div className="mb-6 rounded-lg border border-red-100 bg-red-50 p-4">
                  <input
                    className="input-field mb-2"
                    value={reportReason}
                    onChange={(e) => setReportReason(e.target.value)}
                    placeholder="通報理由"
                    maxLength={200}
                  />
                  <button type="button" onClick={() => void onReport()} className="btn-primary rounded-full text-sm">
                    送信
                  </button>
                </div>
              ) : null}
              {actionMsg ? <p className="mb-4 text-sm text-purple-800">{actionMsg}</p> : null}

              <h2 className="mb-4 text-xl font-semibold text-gray-900">メンバーシッププラン</h2>
              <div className="grid gap-4 sm:grid-cols-2">
                {project.plans.filter((p) => p.is_active).map((plan) => (
                  <button
                    key={plan.id}
                    type="button"
                    onClick={() => goCheckout(plan.id)}
                    className="rounded-xl border border-purple-100 p-5 text-left transition hover:border-purple-400 hover:shadow-md focus:outline-none focus:ring-2 focus:ring-purple-500"
                  >
                    <p className="font-bold text-purple-800">{membershipLabel(plan.membership_type)}</p>
                    <p className="my-2 text-2xl font-extrabold">{formatCurrencyYen(plan.price)}<span className="text-sm font-normal text-gray-500">/月</span></p>
                    <p className="text-sm text-gray-600">{sanitizeDisplayText(plan.benefits, 2000)}</p>
                    <p className="mt-4 text-sm font-bold text-purple-700">このプランで応援する →</p>
                  </button>
                ))}
              </div>
            </div>
          </div>
        </div>

        <aside className="lg:col-span-3">
          <div className="card sticky top-6 p-5">
            <h2 className="mb-4 text-lg font-bold text-gray-900">このクリエイターの他の出品</h2>
            {related.length === 0 ? (
              <p className="text-sm text-gray-500">他の公開プロジェクトはありません</p>
            ) : (
              <ul className="space-y-4">
                {related.slice(0, 5).map((p) => (
                  <li key={p.id}>
                    <Link to={`/projects/${p.id}`} className="block rounded-lg border border-purple-50 p-3 hover:bg-purple-50">
                      <p className="line-clamp-2 font-semibold text-purple-900">{p.title}</p>
                      <p className="mt-1 text-xs text-gray-500">
                        {fundingProgressPercent(p.current_amount, p.target_amount)}% ·{' '}
                        {formatCurrencyYen(p.current_amount)}
                      </p>
                    </Link>
                  </li>
                ))}
              </ul>
            )}
            {related.length > 5 ? (
              <p className="mt-3 text-center text-xs text-gray-500">他 {related.length - 5} 件</p>
            ) : null}
          </div>
        </aside>
      </div>
    </div>
  )
}

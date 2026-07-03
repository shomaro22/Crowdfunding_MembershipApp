import { useEffect, useState } from 'react'
import { Link, useLocation, useNavigate, useParams } from 'react-router-dom'

import { getProject, listPublicProjects } from '@/api/projects'
import { createReport } from '@/api/reports'
import { checkFavorite, toggleFavorite } from '@/api/supporters'
import { useAuth } from '@/auth/AuthProvider'
import { ErrorMessage } from '@/components/common/ErrorMessage'
import { Loading } from '@/components/common/Loading'
import { useToast } from '@/components/common/Toast'
import { ApiError } from '@/lib/api-client'
import { maxPlanSupporters, planBarClass, planHoverClass, planTierLabel, sortPlansByTier } from '@/lib/plans'
import {
  formatCurrencyYen,
  fundingProgressBarWidth,
  fundingProgressPercent,
  sanitizeDisplayText,
} from '@/lib/utils'
import { parseUUID, validateReportDetail } from '@/lib/validation'
import type { ProjectDetail, ProjectSummary } from '@/types/api'

const RELATED_LIMIT = 4

export function ProjectDetailPage() {
  const { projectId } = useParams<{ projectId: string }>()
  const navigate = useNavigate()
  const location = useLocation()
  const { status, hasRole } = useAuth()
  const { toast } = useToast()
  const [project, setProject] = useState<ProjectDetail | null>(null)
  const [related, setRelated] = useState<ProjectSummary[]>([])
  const [relatedTotal, setRelatedTotal] = useState(0)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')
  const [favorited, setFavorited] = useState(false)
  const [reportOpen, setReportOpen] = useState(false)
  const [reportReason, setReportReason] = useState('')

  const isAuthenticated = status === 'authenticated'
  const isRestrictedRole = isAuthenticated && (hasRole('SELLER') || hasRole('ADMIN'))
  const isBuyer = isAuthenticated && hasRole('BUYER')
  const sortedPlans = project ? sortPlansByTier(project.plans.filter((p) => p.is_active)) : []
  const maxSupporters = maxPlanSupporters(sortedPlans)

  const goSignIn = () => {
    const from = location.pathname + location.search
    void navigate(`/signin?from=${encodeURIComponent(from)}`, { state: { from } })
  }

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
        if (isBuyer) {
          const fav = await checkFavorite(id)
          setFavorited(fav.is_favorited)
        }
        if (detail.seller?.id) {
          const list = await listPublicProjects({ limit: 50, offset: 0 })
          const filtered = list.projects.filter(
            (p) => p.id !== id && p.seller?.id === detail.seller?.id,
          )
          setRelatedTotal(filtered.length)
          setRelated(filtered.slice(0, RELATED_LIMIT))
        }
      } catch (err) {
        if (err instanceof ApiError && err.status === 403) {
          setError('このプロジェクトは表示できません（通報済み、またはアクセス権がありません）')
        } else {
          setError('プロジェクトの取得に失敗しました')
        }
      } finally {
        setLoading(false)
      }
    })()
  }, [projectId, isBuyer])

  const onToggleFavorite = async () => {
    if (!project || isRestrictedRole) {
      return
    }
    if (!isAuthenticated) {
      goSignIn()
      return
    }
    try {
      const res = await toggleFavorite(project.id, favorited ? 'remove' : 'add')
      setFavorited(res.is_favorited)
      setProject((prev) => {
        if (!prev) {
          return prev
        }
        const delta = res.is_favorited ? 1 : -1
        return {
          ...prev,
          favorite_count: Math.max(0, (prev.favorite_count ?? 0) + delta),
        }
      })
      toast(res.is_favorited ? 'お気に入りに追加しました' : 'お気に入りを解除しました', 'success')
    } catch (err) {
      toast(err instanceof ApiError ? err.message : 'お気に入り操作に失敗しました', 'error')
    }
  }

  const onReport = async () => {
    if (!project || !isBuyer) {
      if (!isAuthenticated) {
        goSignIn()
      }
      return
    }
    const reason = sanitizeDisplayText(reportReason, 200)
    const reportCheck = validateReportDetail(reason)
    if (!reportCheck.valid) {
      toast(reportCheck.error ?? '通報理由が無効です', 'error')
      return
    }
    if (!reason) {
      toast('通報理由を入力してください', 'error')
      return
    }
    try {
      await createReport('PROJECT', project.id, reason)
      setReportOpen(false)
      setReportReason('')
      toast('通報を受け付けました。このプロジェクトは表示できなくなります', 'success')
      void navigate('/')
    } catch (err) {
      toast(err instanceof ApiError ? err.message : '通報に失敗しました', 'error')
    }
  }

  const goCheckout = (planId: string) => {
    if (!project || isRestrictedRole) {
      return
    }
    if (!isAuthenticated) {
      goSignIn()
      return
    }
    void navigate(`/checkout/${project.id}?plan_id=${encodeURIComponent(planId)}`)
  }

  if (loading) {
    return <Loading />
  }

  if (error || !project) {
    return <ErrorMessage message={error || 'プロジェクトが見つかりません'} />
  }

  const progressPercent = fundingProgressPercent(project.current_amount, project.target_amount)
  const barWidth = fundingProgressBarWidth(progressPercent)
  const isOverfunded = progressPercent > 100
  const mainImage = project.images.find((img) => img.is_main)?.image_url ?? project.image_url
  const favoriteCount = project.favorite_count ?? 0
  const sellerId = project.seller?.id

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

            <div className="p-6">
              <div className="mb-4 flex flex-wrap items-start justify-between gap-3">
                <h1 className="text-3xl font-bold text-gray-900">
                  {sanitizeDisplayText(project.title, 200)}
                </h1>
                <div className="flex items-center gap-3">
                  <span className="flex items-center gap-1 rounded-full bg-pink-50 px-3 py-1 text-sm font-bold text-pink-600">
                    ♥ {favoriteCount}
                  </span>
                  <button
                    type="button"
                    disabled={isRestrictedRole}
                    onClick={() => void onToggleFavorite()}
                    className={`rounded-full px-4 py-2 text-sm font-bold ${
                      isRestrictedRole
                        ? 'cursor-not-allowed bg-gray-100 text-gray-400'
                        : favorited
                          ? 'bg-pink-100 text-pink-700'
                          : 'btn-secondary'
                    }`}
                    title={isRestrictedRole ? '出品者・管理者は利用できません' : undefined}
                  >
                    {favorited ? '♥ いいね済み' : '♡ いいね'}
                  </button>
                  {isBuyer ? (
                    <button type="button" onClick={() => setReportOpen((v) => !v)} className="btn-secondary rounded-full text-sm">
                      通報
                    </button>
                  ) : null}
                </div>
              </div>

              {project.seller ? (
                <p className="mb-4 text-sm text-gray-600">
                  クリエイター:{' '}
                  {sellerId ? (
                    <Link to={`/creators/${sellerId}`} className="font-semibold text-purple-800 hover:underline">
                      {sanitizeDisplayText(project.seller.name ?? project.seller.email, 100)}
                    </Link>
                  ) : (
                    <span className="font-semibold text-purple-800">
                      {sanitizeDisplayText(project.seller.name ?? project.seller.email, 100)}
                    </span>
                  )}
                </p>
              ) : null}

              <p className="mb-6 whitespace-pre-wrap text-gray-700">
                {sanitizeDisplayText(project.description, 10000)}
              </p>

              <div className="mb-6">
                <div className="mb-2 flex justify-between text-sm text-gray-600">
                  <span className="flex items-center gap-2">
                    <span className={`text-lg font-extrabold ${isOverfunded ? 'text-amber-600' : 'text-purple-800'}`}>
                      {progressPercent}%
                    </span>
                    {isOverfunded ? (
                      <span className="rounded-full bg-amber-100 px-2 py-0.5 text-xs font-bold text-amber-800">
                        目標達成！
                      </span>
                    ) : null}
                    {' · '}
                    支援者 {project.supporter_count}人
                  </span>
                  <span>
                    {formatCurrencyYen(project.current_amount)} / {formatCurrencyYen(project.target_amount)}
                  </span>
                </div>
                <div className={`h-3 overflow-hidden rounded-full ${isOverfunded ? 'bg-amber-100' : 'bg-purple-100'}`}>
                  <div
                    className={`h-full ${isOverfunded ? 'bg-gradient-to-r from-amber-500 to-orange-500' : 'bg-purple-600'}`}
                    style={{ width: `${barWidth}%` }}
                  />
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

              <h2 className="mb-2 text-xl font-semibold text-gray-900">メンバーシッププラン</h2>
              <p className="mb-4 text-sm text-gray-500">プラン別の加入者数</p>
              <div className="mb-6 space-y-2">
                {sortedPlans.map((plan) => (
                  <div key={plan.id} className="flex items-center gap-3 text-sm">
                    <span className="w-16 shrink-0 font-bold">{planTierLabel(plan.membership_type)}</span>
                    <div className="h-3 flex-1 overflow-hidden rounded-full bg-gray-100">
                      <div
                        className={`h-full rounded-full ${planBarClass(plan.membership_type)}`}
                        style={{ width: `${Math.min(100, (plan.supporter_count / maxSupporters) * 100)}%` }}
                      />
                    </div>
                    <span className="w-12 text-right text-gray-600">{plan.supporter_count}人</span>
                  </div>
                ))}
              </div>

              <div className="grid gap-4 sm:grid-cols-2">
                {sortedPlans.map((plan) => (
                  <button
                    key={plan.id}
                    type="button"
                    disabled={isRestrictedRole}
                    onClick={() => goCheckout(plan.id)}
                    className={`rounded-xl border border-purple-100 p-5 text-left transition-all duration-200 ${planHoverClass(plan.membership_type)} ${
                      isRestrictedRole ? 'cursor-not-allowed opacity-60' : ''
                    }`}
                    title={isRestrictedRole ? '出品者・管理者は応援できません' : undefined}
                  >
                    <p className="font-bold text-purple-800">{planTierLabel(plan.membership_type)}</p>
                    <p className="my-2 text-2xl font-extrabold">
                      {formatCurrencyYen(plan.price)}
                      <span className="text-sm font-normal text-gray-500">/月</span>
                    </p>
                    <p className="text-sm text-gray-600">{sanitizeDisplayText(plan.benefits, 2000)}</p>
                    {isRestrictedRole ? (
                      <p className="mt-4 text-sm text-gray-400">出品者・管理者は応援できません</p>
                    ) : (
                      <p className="mt-4 text-sm font-bold text-purple-700">このプランで応援する →</p>
                    )}
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
              <ul className="space-y-3">
                {related.map((p) => (
                  <li key={p.id}>
                    <Link to={`/projects/${p.id}`} className="block rounded-lg border border-purple-50 p-3 hover:bg-purple-50">
                      <p className="line-clamp-2 font-semibold text-purple-900">{p.title}</p>
                      <p className="mt-1 text-xs text-gray-500">
                        {fundingProgressPercent(p.current_amount, p.target_amount)}% ·{' '}
                        ♥ {p.favorite_count ?? 0} · {formatCurrencyYen(p.current_amount)}
                      </p>
                    </Link>
                  </li>
                ))}
              </ul>
            )}
            {relatedTotal > RELATED_LIMIT && sellerId ? (
              <Link
                to={`/creators/${sellerId}`}
                className="mt-4 block text-center text-sm font-bold text-purple-700 hover:underline"
              >
                もっと見る（{relatedTotal}件）
              </Link>
            ) : null}
          </div>
        </aside>
      </div>
    </div>
  )
}

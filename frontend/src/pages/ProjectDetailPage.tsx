import { useCallback, useEffect, useState } from 'react'
import { Link, useNavigate, useParams } from 'react-router-dom'

import { getProject, listPublicProjects } from '@/api/projects'
import { createReport } from '@/api/reports'
import { checkFavorite, toggleFavorite } from '@/api/supporters'
import { useAuth } from '@/auth/AuthProvider'
import { ErrorMessage } from '@/components/common/ErrorMessage'
import { Loading } from '@/components/common/Loading'
import { useToast } from '@/components/common/Toast'
import { useInfiniteScroll } from '@/hooks/useInfiniteScroll'
import { ApiError } from '@/lib/api-client'
import { maxPlanSupporters, planBarClass, planHoverClass, planTierLabel, sortPlansByTier } from '@/lib/plans'
import {
  formatCurrencyYen,
  fundingProgressBarWidth,
  fundingProgressPercent,
  sanitizeDisplayText,
} from '@/lib/utils'
import { parseUUID } from '@/lib/validation'
import type { ProjectDetail, ProjectSummary } from '@/types/api'

const RELATED_PAGE = 6
const RELATED_MAX = 30

export function ProjectDetailPage() {
  const { projectId } = useParams<{ projectId: string }>()
  const navigate = useNavigate()
  const { status, hasRole } = useAuth()
  const { toast } = useToast()
  const [project, setProject] = useState<ProjectDetail | null>(null)
  const [related, setRelated] = useState<ProjectSummary[]>([])
  const [relatedTotal, setRelatedTotal] = useState(0)
  const [relatedLoading, setRelatedLoading] = useState(false)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')
  const [favorited, setFavorited] = useState(false)
  const [reportOpen, setReportOpen] = useState(false)
  const [reportReason, setReportReason] = useState('')

  const isBuyer = status === 'authenticated' && hasRole('BUYER')
  const canInteract = isBuyer
  const sortedPlans = project ? sortPlansByTier(project.plans.filter((p) => p.is_active)) : []
  const maxSupporters = maxPlanSupporters(sortedPlans)

  const hasMoreRelated = related.length < relatedTotal && related.length < RELATED_MAX

  const loadRelated = useCallback(async (sellerId: string, offset: number, append: boolean) => {
    setRelatedLoading(true)
    try {
      const list = await listPublicProjects({ limit: RELATED_PAGE, offset })
      const filtered = list.projects.filter(
        (p) => p.id !== projectId && p.seller?.id === sellerId,
      )
      setRelatedTotal(list.total)
      setRelated((prev) => (append ? [...prev, ...filtered] : filtered))
    } finally {
      setRelatedLoading(false)
    }
  }, [projectId])

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
          await loadRelated(detail.seller.id, 0, false)
        }
      } catch {
        setError('プロジェクトの取得に失敗しました')
      } finally {
        setLoading(false)
      }
    })()
  }, [projectId, isBuyer, loadRelated])

  const loadMoreRelated = useCallback(() => {
    if (!project?.seller?.id || relatedLoading || !hasMoreRelated) {
      return
    }
    void loadRelated(project.seller.id, related.length, true)
  }, [hasMoreRelated, loadRelated, project?.seller?.id, related.length, relatedLoading])

  const relatedSentinel = useInfiniteScroll(loadMoreRelated, hasMoreRelated, relatedLoading)

  const onToggleFavorite = async () => {
    if (!project || !canInteract) {
      return
    }
    try {
      const res = await toggleFavorite(project.id, favorited ? 'remove' : 'add')
      setFavorited(res.is_favorited)
      toast(res.is_favorited ? 'お気に入りに追加しました' : 'お気に入りを解除しました', 'success')
    } catch (err) {
      toast(err instanceof ApiError ? err.message : 'お気に入り操作に失敗しました', 'error')
    }
  }

  const onReport = async () => {
    if (!project || !canInteract) {
      return
    }
    const reason = sanitizeDisplayText(reportReason, 200)
    if (!reason) {
      toast('通報理由を入力してください', 'error')
      return
    }
    try {
      await createReport('PROJECT', project.id, reason)
      setReportOpen(false)
      setReportReason('')
      toast('通報を受け付けました', 'success')
    } catch (err) {
      toast(err instanceof ApiError ? err.message : '通報に失敗しました', 'error')
    }
  }

  const goCheckout = (planId: string) => {
    if (!project || !canInteract) {
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
  const mainImage = project.images.find((img) => img.is_main)?.image_url ?? project.image_url
  const favoriteCount = project.favorite_count ?? 0

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
                    disabled={!canInteract}
                    onClick={() => void onToggleFavorite()}
                    className={`rounded-full px-4 py-2 text-sm font-bold ${
                      !canInteract
                        ? 'cursor-not-allowed bg-gray-100 text-gray-400'
                        : favorited
                          ? 'bg-pink-100 text-pink-700'
                          : 'btn-secondary'
                    }`}
                    title={!canInteract ? '購入者アカウントでのみ利用できます' : undefined}
                  >
                    {favorited ? '♥ いいね済み' : '♡ いいね'}
                  </button>
                  {canInteract ? (
                    <button type="button" onClick={() => setReportOpen((v) => !v)} className="btn-secondary rounded-full text-sm">
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
                    disabled={!canInteract}
                    onClick={() => goCheckout(plan.id)}
                    className={`rounded-xl border border-purple-100 p-5 text-left transition-all duration-200 ${planHoverClass(plan.membership_type)} ${
                      !canInteract ? 'cursor-not-allowed opacity-60' : ''
                    }`}
                    title={!canInteract ? '購入者アカウントでのみ応援できます' : undefined}
                  >
                    <p className="font-bold text-purple-800">{planTierLabel(plan.membership_type)}</p>
                    <p className="my-2 text-2xl font-extrabold">
                      {formatCurrencyYen(plan.price)}
                      <span className="text-sm font-normal text-gray-500">/月</span>
                    </p>
                    <p className="text-sm text-gray-600">{sanitizeDisplayText(plan.benefits, 2000)}</p>
                    {canInteract ? (
                      <p className="mt-4 text-sm font-bold text-purple-700">このプランで応援する →</p>
                    ) : (
                      <p className="mt-4 text-sm text-gray-400">購入者のみ応援可能</p>
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
            {related.length === 0 && !relatedLoading ? (
              <p className="text-sm text-gray-500">他の公開プロジェクトはありません</p>
            ) : (
              <ul className="max-h-[32rem] space-y-3 overflow-y-auto">
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
            {relatedLoading ? <p className="mt-2 text-center text-xs text-gray-400">読み込み中...</p> : null}
            {hasMoreRelated ? <div ref={relatedSentinel} className="h-4" aria-hidden /> : null}
          </div>
        </aside>
      </div>
    </div>
  )
}

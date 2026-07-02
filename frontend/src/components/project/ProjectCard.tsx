import { Link } from 'react-router-dom'

import type { ProjectSummary } from '@/types/api'
import { planBarClass, sortPlansByTier } from '@/lib/plans'
import { cn, formatCurrencyYen, fundingProgressBarWidth, fundingProgressPercent } from '@/lib/utils'

interface ProjectCardProps {
  project: ProjectSummary
  className?: string
  hideFavoriteBadge?: boolean
  isFavorited?: boolean
}

export function ProjectCard({ project, className, hideFavoriteBadge = false, isFavorited = false }: ProjectCardProps) {
  const sortedPlans = sortPlansByTier(project.plans)
  const prices = sortedPlans.map((plan) => plan.price)
  const minPrice = prices.length > 0 ? Math.min(...prices) : 0
  const maxPrice = prices.length > 0 ? Math.max(...prices) : 0
  const progressPercent = fundingProgressPercent(project.current_amount, project.target_amount)
  const barWidth = fundingProgressBarWidth(progressPercent)
  const favoriteCount = project.favorite_count ?? 0
  const isOverfunded = progressPercent > 100

  return (
    <article
      className={cn(
        'flex h-full flex-col overflow-hidden rounded-lg bg-white shadow-md transition-shadow hover:shadow-lg',
        isOverfunded && 'ring-2 ring-amber-300',
        className,
      )}
    >
      <div className="relative flex h-64 items-center justify-center bg-gradient-to-br from-purple-100 to-purple-200">
        {project.image_url ? (
          <img
            src={project.image_url}
            alt=""
            className="h-full w-full object-cover"
            loading="lazy"
            referrerPolicy="no-referrer"
          />
        ) : (
          <svg className="h-12 w-12 text-purple-400" fill="none" stroke="currentColor" viewBox="0 0 24 24" aria-hidden>
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 16l4.586-4.586a2 2 0 012.828 0L16 16m-2-2l1.586-1.586a2 2 0 012.828 0L20 14m-6-6h.01M6 20h12a2 2 0 002-2V6a2 2 0 00-2-2H6a2 2 0 00-2 2v12a2 2 0 002 2z" />
          </svg>
        )}
        {isOverfunded ? (
          <span className="absolute left-3 top-3 rounded-full bg-amber-500 px-2.5 py-1 text-xs font-bold text-white shadow">
            目標達成
          </span>
        ) : null}
        {!hideFavoriteBadge ? (
          <div className={`absolute right-3 top-3 flex items-center gap-1 rounded-full px-2.5 py-1 text-sm font-bold shadow ${
            isFavorited ? 'bg-pink-600 text-white' : 'bg-white/90 text-pink-600'
          }`}>
            <span aria-hidden>{isFavorited ? '♥' : '♡'}</span>
            <span>{favoriteCount}</span>
            {isFavorited ? <span className="sr-only">いいね済み</span> : null}
          </div>
        ) : null}
      </div>

      <div className="flex flex-1 flex-col p-4">
        <h3 className="mb-2 line-clamp-2 text-lg font-bold text-gray-900">{project.title}</h3>
        <p className="mb-4 line-clamp-3 flex-1 text-sm text-gray-600">{project.description}</p>

        <div className="mb-3">
          <div className="mb-1 flex justify-between text-xs text-gray-500">
            <span className="flex items-center gap-1">
              <span className={`text-base font-extrabold ${isOverfunded ? 'text-amber-600' : 'text-purple-800'}`}>
                {progressPercent}%
              </span>
              · 支援者 {project.supporter_count}人
            </span>
            <span>{formatCurrencyYen(project.current_amount)} / {formatCurrencyYen(project.target_amount)}</span>
          </div>
          <div className={`h-2 overflow-hidden rounded-full ${isOverfunded ? 'bg-amber-100' : 'bg-purple-100'}`}>
            <div
              className={`h-full rounded-full ${isOverfunded ? 'bg-gradient-to-r from-amber-500 to-orange-500' : 'bg-purple-600'}`}
              style={{ width: `${barWidth}%` }}
            />
          </div>
        </div>

        {sortedPlans.length > 0 ? (
          <div className="mb-4 space-y-1">
            {sortedPlans.map((plan) => (
              <div key={plan.id} className="flex items-center gap-2 text-xs">
                <span className="w-14 shrink-0 font-semibold text-gray-700">{plan.membership_type}</span>
                <div className="h-1.5 flex-1 overflow-hidden rounded-full bg-gray-100">
                  <div
                    className={cn('h-full rounded-full', planBarClass(plan.membership_type))}
                    style={{ width: `${Math.min(100, (plan.supporter_count / Math.max(project.supporter_count, 1)) * 100)}%` }}
                  />
                </div>
                <span className="w-8 text-right text-gray-500">{plan.supporter_count}</span>
              </div>
            ))}
          </div>
        ) : null}

        <div className="mt-auto flex items-center justify-between">
          <span className="text-sm font-semibold text-purple-700">
            {minPrice === maxPrice
              ? formatCurrencyYen(minPrice)
              : `${formatCurrencyYen(minPrice)}〜${formatCurrencyYen(maxPrice)}`}
          </span>
          <Link
            to={`/projects/${project.id}`}
            className="btn-primary rounded-full px-4 py-2 text-sm font-bold"
          >
            詳細を見る
          </Link>
        </div>
      </div>
    </article>
  )
}

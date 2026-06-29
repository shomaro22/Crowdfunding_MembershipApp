import { Link } from 'react-router-dom'

import type { ProjectSummary } from '@/types/api'
import { cn, formatCurrencyYen } from '@/lib/utils'

interface ProjectCardProps {
  project: ProjectSummary
  className?: string
}

function membershipLabel(type: string): string {
  switch (type) {
    case 'BRONZE':
      return 'ブロンズ'
    case 'SILVER':
      return 'シルバー'
    case 'GOLD':
      return 'ゴールド'
    default:
      return 'メンバーシップ'
  }
}

function membershipColor(type: string): string {
  switch (type) {
    case 'BRONZE':
      return 'bg-gradient-to-r from-orange-400 to-orange-500 text-white'
    case 'SILVER':
      return 'bg-gradient-to-r from-gray-400 to-gray-500 text-white'
    case 'GOLD':
      return 'bg-gradient-to-r from-yellow-400 to-yellow-500 text-white'
    default:
      return 'bg-gradient-to-r from-purple-400 to-purple-500 text-white'
  }
}

export function ProjectCard({ project, className }: ProjectCardProps) {
  const prices = project.plans.map((plan) => plan.price)
  const minPrice = prices.length > 0 ? Math.min(...prices) : 0
  const maxPrice = prices.length > 0 ? Math.max(...prices) : 0
  const progress = project.target_amount > 0
    ? Math.min(100, Math.round((project.current_amount / project.target_amount) * 100))
    : 0

  return (
    <article
      className={cn(
        'flex h-full flex-col overflow-hidden rounded-lg bg-white shadow-md transition-shadow hover:shadow-lg',
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
      </div>

      <div className="flex flex-1 flex-col p-4">
        <h3 className="mb-2 line-clamp-2 text-lg font-bold text-gray-900">{project.title}</h3>
        <p className="mb-4 line-clamp-3 flex-1 text-sm text-gray-600">{project.description}</p>

        <div className="mb-3">
          <div className="mb-1 flex justify-between text-xs text-gray-500">
            <span>{progress}%</span>
            <span>{formatCurrencyYen(project.current_amount)} / {formatCurrencyYen(project.target_amount)}</span>
          </div>
          <div className="h-2 overflow-hidden rounded-full bg-purple-100">
            <div className="h-full rounded-full bg-purple-600" style={{ width: `${progress}%` }} />
          </div>
        </div>

        {project.plans.length > 0 ? (
          <div className="mb-4 flex flex-wrap gap-2">
            {project.plans.slice(0, 3).map((plan) => (
              <span
                key={plan.id}
                className={cn('rounded-full px-2 py-1 text-xs font-medium shadow', membershipColor(plan.membership_type))}
              >
                {membershipLabel(plan.membership_type)} {formatCurrencyYen(plan.price)}
              </span>
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

import type { ProjectPlan } from '@/types/api'

const TIER_ORDER: Record<string, number> = {
  BRONZE: 0,
  SILVER: 1,
  GOLD: 2,
}

export function sortPlansByTier<T extends { membership_type: string }>(plans: T[]): T[] {
  return [...plans].sort(
    (a, b) => (TIER_ORDER[a.membership_type] ?? 99) - (TIER_ORDER[b.membership_type] ?? 99),
  )
}

export function planTierLabel(type: string): string {
  return type
}

export function planHoverClass(type: string): string {
  switch (type) {
    case 'BRONZE':
      return 'hover:border-orange-400 hover:bg-gradient-to-br hover:from-orange-50 hover:to-orange-100 hover:shadow-orange-200/50'
    case 'SILVER':
      return 'hover:border-gray-400 hover:bg-gradient-to-br hover:from-gray-50 hover:to-gray-200 hover:shadow-gray-300/50'
    case 'GOLD':
      return 'hover:border-yellow-500 hover:bg-gradient-to-br hover:from-yellow-50 hover:to-amber-100 hover:shadow-yellow-300/50'
    default:
      return 'hover:border-purple-400 hover:bg-purple-50'
  }
}

export function planBarClass(type: string): string {
  switch (type) {
    case 'BRONZE':
      return 'bg-gradient-to-r from-orange-400 to-orange-500'
    case 'SILVER':
      return 'bg-gradient-to-r from-gray-400 to-gray-500'
    case 'GOLD':
      return 'bg-gradient-to-r from-yellow-400 to-amber-500'
    default:
      return 'bg-purple-500'
  }
}

export function maxPlanSupporters(plans: ProjectPlan[]): number {
  return Math.max(1, ...plans.map((p) => p.supporter_count))
}

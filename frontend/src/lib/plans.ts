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

export function planTierTextClass(type: string): string {
  switch (type) {
    case 'BRONZE':
      return 'text-orange-600 font-semibold'
    case 'SILVER':
      return 'text-slate-600 font-semibold'
    case 'GOLD':
      return 'text-amber-600 font-semibold'
    default:
      return 'text-purple-700 font-semibold'
  }
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

export function planSelectedClass(type: string): string {
  switch (type) {
    case 'BRONZE':
      return 'border-2 border-orange-500 bg-gradient-to-br from-orange-50 to-orange-100 ring-2 ring-orange-300'
    case 'SILVER':
      return 'border-2 border-gray-500 bg-gradient-to-br from-gray-50 to-gray-200 ring-2 ring-gray-400'
    case 'GOLD':
      return 'border-2 border-amber-500 bg-gradient-to-br from-yellow-50 to-amber-100 ring-2 ring-amber-400'
    default:
      return 'border-2 border-purple-600 bg-purple-50 ring-2 ring-purple-300'
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

const TIER_ORDER_LIST = ['BRONZE', 'SILVER', 'GOLD'] as const

/** ブロンズのみ / ブロンズ+シルバー / ブロンズ+シルバー+ゴールド のみ許可 */
export function validateMembershipTierCombo(enabled: Record<string, boolean>): string | null {
  const hasBronze = Boolean(enabled.BRONZE)
  const hasSilver = Boolean(enabled.SILVER)
  const hasGold = Boolean(enabled.GOLD)
  if (!hasBronze) {
    return 'ブロンズプランは必須です'
  }
  if (hasGold && !hasSilver) {
    return 'ゴールドを選ぶ場合はシルバーも有効にしてください'
  }
  if (hasSilver && !hasBronze) {
    return 'シルバーを選ぶ場合はブロンズも有効にしてください'
  }
  const active = TIER_ORDER_LIST.filter((t) => enabled[t])
  if (active.length === 0) {
    return '少なくとも1つのメンバーシッププランを選択してください'
  }
  for (let i = 0; i < active.length; i++) {
    if (active[i] !== TIER_ORDER_LIST[i]) {
      return 'プランはブロンズ → シルバー → ゴールドの順で連続して選択してください'
    }
  }
  return null
}

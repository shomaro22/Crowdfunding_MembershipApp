import type { AdminRole } from '@/types/api'

export type AdminTab =
  | 'stats'
  | 'users'
  | 'reviews-buyers'
  | 'reviews-sellers'
  | 'projects'
  | 'purchases'
  | 'reports'
  | 'activity'
  | 'security'
  | 'batches'
  | 'audit'

const TAB_ROLES: Record<AdminTab, readonly AdminRole[]> = {
  stats: ['SUPER_ADMIN', 'SUPPORT_ADMIN'],
  users: ['SUPER_ADMIN', 'SUPPORT_ADMIN'],
  'reviews-buyers': ['SUPER_ADMIN', 'REVIEW_ADMIN'],
  'reviews-sellers': ['SUPER_ADMIN', 'REVIEW_ADMIN'],
  projects: ['SUPER_ADMIN', 'REVIEW_ADMIN'],
  purchases: ['SUPER_ADMIN', 'FINANCE_ADMIN'],
  reports: ['SUPER_ADMIN', 'REVIEW_ADMIN', 'SUPPORT_ADMIN'],
  activity: ['SUPER_ADMIN', 'SUPPORT_ADMIN', 'FINANCE_ADMIN'],
  security: ['SUPER_ADMIN', 'SUPPORT_ADMIN'],
  batches: ['SUPER_ADMIN', 'FINANCE_ADMIN'],
  audit: ['SUPER_ADMIN'],
}

export function canAccessAdminTab(role: AdminRole | null, tab: AdminTab): boolean {
  if (!role) {
    return false
  }
  if (role === 'SUPER_ADMIN') {
    return true
  }
  return TAB_ROLES[tab].includes(role)
}

export function availableAdminTabs(role: AdminRole | null): AdminTab[] {
  const all: AdminTab[] = [
    'stats',
    'users',
    'reviews-buyers',
    'reviews-sellers',
    'projects',
    'purchases',
    'reports',
    'activity',
    'security',
    'batches',
    'audit',
  ]
  return all.filter((tab) => canAccessAdminTab(role, tab))
}

export const ADMIN_TAB_LABELS: Record<AdminTab, string> = {
  stats: '統計',
  users: 'ユーザー',
  'reviews-buyers': '購入者審査',
  'reviews-sellers': '出品者KYC',
  projects: 'プロジェクト',
  purchases: '購入',
  reports: '通報',
  activity: '活動ログ',
  security: 'セキュリティ',
  batches: 'バッチ',
  audit: '監査ログ',
}

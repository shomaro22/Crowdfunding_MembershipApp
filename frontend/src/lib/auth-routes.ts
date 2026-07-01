import type { User } from '@/types/api'

export function postLoginPath(user: User | null, fallback: string): string {
  if (!user) {
    return fallback
  }
  if (user.role === 'ADMIN') {
    return '/admin'
  }
  if (user.role === 'SELLER') {
    return '/seller/dashboard'
  }
  return fallback
}

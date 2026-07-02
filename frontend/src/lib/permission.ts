export type UserRole = 'BUYER' | 'SELLER' | 'ADMIN'

export type AdminRole = 'SUPER_ADMIN' | 'MODERATOR' | 'SUPPORT'

export function isAdmin(role: string | undefined | null): boolean {
  return role === 'ADMIN'
}

export function isSeller(role: string | undefined | null): boolean {
  return role === 'SELLER'
}

export function isBuyer(role: string | undefined | null): boolean {
  return role === 'BUYER'
}

export function canAccessAdmin(adminRole: string | undefined | null): boolean {
  return adminRole === 'SUPER_ADMIN' || adminRole === 'MODERATOR' || adminRole === 'SUPPORT'
}

export function canModerate(adminRole: string | undefined | null): boolean {
  return adminRole === 'SUPER_ADMIN' || adminRole === 'MODERATOR'
}

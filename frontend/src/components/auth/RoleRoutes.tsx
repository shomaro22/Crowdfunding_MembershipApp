import type { ReactNode } from 'react'
import { Navigate } from 'react-router-dom'

import { useAuth } from '@/auth/AuthProvider'
import { isAdmin, isBuyer, isSeller } from '@/lib/permission'

type RoleRouteProps = {
  children: ReactNode
  fallback?: string
}

export function AdminRoute({ children, fallback = '/signin' }: RoleRouteProps) {
  const { user, status } = useAuth()
  if (status === 'loading') {
    return null
  }
  if (!user || !isAdmin(user.role)) {
    return <Navigate replace to={fallback} />
  }
  return children
}

export function SellerRoute({ children, fallback = '/signin' }: RoleRouteProps) {
  const { user, status } = useAuth()
  if (status === 'loading') {
    return null
  }
  if (!user || !isSeller(user.role)) {
    return <Navigate replace to={fallback} />
  }
  return children
}

export function MemberRoute({ children, fallback = '/signin' }: RoleRouteProps) {
  const { user, status } = useAuth()
  if (status === 'loading') {
    return null
  }
  if (!user || !isBuyer(user.role)) {
    return <Navigate replace to={fallback} />
  }
  return children
}

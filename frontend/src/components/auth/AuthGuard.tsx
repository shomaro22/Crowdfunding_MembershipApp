import { Navigate, useLocation } from 'react-router-dom'

import { useAuth } from '@/auth/AuthProvider'
import { Loading } from '@/components/common/Loading'
import type { UserRole } from '@/types/api'

interface AuthGuardProps {
  children: React.ReactNode
  requiredRole?: UserRole
  redirectTo?: string
}

export function AuthGuard({
  children,
  requiredRole,
  redirectTo = '/signin',
}: AuthGuardProps) {
  const { status, user, hasRole } = useAuth()
  const location = useLocation()

  if (status === 'loading') {
    return <Loading label="認証状態を確認中..." />
  }

  if (status === 'pending_2fa') {
    return <Navigate to="/auth/2fa" replace state={{ from: location.pathname }} />
  }

  if (status !== 'authenticated' || !user) {
    return <Navigate to={redirectTo} replace state={{ from: location.pathname }} />
  }

  if (requiredRole && !hasRole(requiredRole)) {
    return <Navigate to="/" replace />
  }

  return children
}

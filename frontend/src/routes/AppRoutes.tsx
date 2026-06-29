import { BrowserRouter, Navigate, Route, Routes } from 'react-router-dom'

import { AuthProvider } from '@/auth/AuthProvider'
import { AuthGuard } from '@/components/auth/AuthGuard'
import { MainLayout } from '@/components/layout/MainLayout'
import { AuthCallbackPage } from '@/pages/AuthCallbackPage'
import { HomePage } from '@/pages/HomePage'
import { NotFoundPage } from '@/pages/NotFoundPage'
import { PlaceholderPage } from '@/pages/PlaceholderPage'
import { ProjectDetailPage } from '@/pages/ProjectDetailPage'
import { SellerSignUpPage } from '@/pages/SellerSignUpPage'
import { SignInPage } from '@/pages/SignInPage'
import { SignUpPage } from '@/pages/SignUpPage'
import { TwoFactorPage } from '@/pages/TwoFactorPage'

export function AppRoutes() {
  return (
    <BrowserRouter>
      <AuthProvider>
        <Routes>
          <Route element={<MainLayout><HomePage /></MainLayout>} path="/" />
          <Route element={<MainLayout><SignInPage /></MainLayout>} path="/signin" />
          <Route element={<MainLayout><SignUpPage /></MainLayout>} path="/signup" />
          <Route element={<MainLayout><SellerSignUpPage /></MainLayout>} path="/signup/seller" />
          <Route element={<MainLayout><TwoFactorPage /></MainLayout>} path="/auth/2fa" />
          <Route element={<AuthCallbackPage />} path="/auth/callback" />

          <Route element={<MainLayout><ProjectDetailPage /></MainLayout>} path="/projects/:projectId" />

          <Route
            element={(
              <MainLayout>
                <AuthGuard requiredRole="BUYER">
                  <PlaceholderPage title="プロフィール" description="プロフィール編集画面（実装予定）" />
                </AuthGuard>
              </MainLayout>
            )}
            path="/profile"
          />
          <Route
            element={(
              <MainLayout>
                <AuthGuard requiredRole="BUYER">
                  <PlaceholderPage title="お気に入り" description="お気に入り一覧（実装予定）" />
                </AuthGuard>
              </MainLayout>
            )}
            path="/favorites"
          />
          <Route
            element={(
              <MainLayout>
                <AuthGuard requiredRole="BUYER">
                  <PlaceholderPage title="応援履歴" description="購入履歴（実装予定）" />
                </AuthGuard>
              </MainLayout>
            )}
            path="/purchases"
          />
          <Route
            element={(
              <MainLayout>
                <AuthGuard>
                  <PlaceholderPage title="メッセージ" description="メッセージ / WebSocket（実装予定）" />
                </AuthGuard>
              </MainLayout>
            )}
            path="/messages"
          />
          <Route
            element={(
              <MainLayout>
                <AuthGuard requiredRole="SELLER">
                  <PlaceholderPage title="出品者ダッシュボード" description="統計・支援者管理（実装予定）" />
                </AuthGuard>
              </MainLayout>
            )}
            path="/seller/dashboard"
          />
          <Route
            element={(
              <MainLayout>
                <AuthGuard requiredRole="SELLER">
                  <PlaceholderPage title="プロジェクト管理" description="出品・編集（実装予定）" />
                </AuthGuard>
              </MainLayout>
            )}
            path="/seller/projects"
          />
          <Route
            element={(
              <MainLayout>
                <AuthGuard requiredRole="ADMIN">
                  <PlaceholderPage title="管理画面" description="審査・ユーザー管理（実装予定）" />
                </AuthGuard>
              </MainLayout>
            )}
            path="/admin"
          />
          <Route
            element={(
              <MainLayout>
                <AuthGuard requiredRole="BUYER">
                  <PlaceholderPage title="チェックアウト" description="Stripe Checkout 連携（実装予定）" />
                </AuthGuard>
              </MainLayout>
            )}
            path="/checkout/:projectId"
          />

          <Route element={<Navigate replace to="/" />} path="/auth/signin" />
          <Route element={<MainLayout><NotFoundPage /></MainLayout>} path="*" />
        </Routes>
      </AuthProvider>
    </BrowserRouter>
  )
}

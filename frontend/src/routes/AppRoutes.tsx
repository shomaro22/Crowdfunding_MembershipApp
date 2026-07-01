import { BrowserRouter, Navigate, Route, Routes } from 'react-router-dom'
import type { ReactNode } from 'react'

import { AuthProvider } from '@/auth/AuthProvider'
import { AuthGuard } from '@/components/auth/AuthGuard'
import { MainLayout } from '@/components/layout/MainLayout'
import { AdminPage } from '@/pages/admin/AdminPage'
import { AuthCallbackPage } from '@/pages/AuthCallbackPage'
import { CheckoutPage } from '@/pages/CheckoutPage'
import { FavoritesPage } from '@/pages/FavoritesPage'
import { ForgotPasswordPage } from '@/pages/ForgotPasswordPage'
import { CreatorProjectsPage } from '@/pages/CreatorProjectsPage'
import { HomePage } from '@/pages/HomePage'
import { MessagesPage } from '@/pages/MessagesPage'
import { NotFoundPage } from '@/pages/NotFoundPage'
import { ProfilePage } from '@/pages/ProfilePage'
import { ProjectDetailPage } from '@/pages/ProjectDetailPage'
import { PurchasesPage } from '@/pages/PurchasesPage'
import { ResetPasswordPage } from '@/pages/ResetPasswordPage'
import { SellerSignUpPage } from '@/pages/SellerSignUpPage'
import { SignInPage } from '@/pages/SignInPage'
import { SignUpPage } from '@/pages/SignUpPage'
import { SupportCancelPage, SupportSuccessPage } from '@/pages/SupportResultPages'
import { TwoFactorPage } from '@/pages/TwoFactorPage'
import { SellerConnectReturnPage } from '@/pages/seller/SellerConnectReturnPage'
import { SellerDashboardPage } from '@/pages/seller/SellerDashboardPage'
import { SellerProjectFormPage } from '@/pages/seller/SellerProjectFormPage'
import { SellerProjectsPage } from '@/pages/seller/SellerProjectsPage'

function buyerGuard(children: ReactNode) {
  return (
    <MainLayout>
      <AuthGuard requiredRole="BUYER">{children}</AuthGuard>
    </MainLayout>
  )
}

function sellerGuard(children: ReactNode) {
  return (
    <MainLayout>
      <AuthGuard requiredRole="SELLER">{children}</AuthGuard>
    </MainLayout>
  )
}

function adminGuard(children: ReactNode) {
  return (
    <MainLayout>
      <AuthGuard requiredRole="ADMIN">{children}</AuthGuard>
    </MainLayout>
  )
}

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
          <Route element={<MainLayout><ForgotPasswordPage /></MainLayout>} path="/forgot-password" />
          <Route element={<MainLayout><ResetPasswordPage /></MainLayout>} path="/reset-password" />

          <Route element={<MainLayout><ProjectDetailPage /></MainLayout>} path="/projects/:projectId" />
          <Route element={<MainLayout><CreatorProjectsPage /></MainLayout>} path="/creators/:sellerId" />

          <Route element={buyerGuard(<ProfilePage />)} path="/profile" />
          <Route element={buyerGuard(<FavoritesPage />)} path="/favorites" />
          <Route element={buyerGuard(<PurchasesPage />)} path="/purchases" />
          <Route element={buyerGuard(<CheckoutPage />)} path="/checkout/:projectId" />
          <Route element={buyerGuard(<SupportSuccessPage />)} path="/support/success" />
          <Route element={buyerGuard(<SupportCancelPage />)} path="/support/cancel" />

          <Route
            element={(
              <MainLayout>
                <AuthGuard><MessagesPage /></AuthGuard>
              </MainLayout>
            )}
            path="/messages"
          />

          <Route element={sellerGuard(<SellerDashboardPage />)} path="/seller/dashboard" />
          <Route element={sellerGuard(<SellerProjectsPage />)} path="/seller/projects" />
          <Route element={sellerGuard(<SellerProjectFormPage />)} path="/seller/projects/new" />
          <Route element={sellerGuard(<SellerProjectFormPage />)} path="/seller/projects/:projectId/edit" />
          <Route element={sellerGuard(<SellerConnectReturnPage />)} path="/seller/connect/return" />
          <Route element={sellerGuard(<SellerConnectReturnPage />)} path="/seller/connect/refresh" />

          <Route element={adminGuard(<AdminPage />)} path="/admin" />

          <Route element={<Navigate replace to="/signin" />} path="/auth/signin" />
          <Route element={<MainLayout><NotFoundPage /></MainLayout>} path="*" />
        </Routes>
      </AuthProvider>
    </BrowserRouter>
  )
}

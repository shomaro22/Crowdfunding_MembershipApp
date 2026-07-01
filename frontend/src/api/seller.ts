import { apiBaseUrl, apiRequest } from '@/lib/api-client'
import { getAccessToken } from '@/lib/token-store'
import type {
  ConnectAccountResponse,
  ConnectOnboardingResponse,
  ListSellerSupportersResponse,
  ProjectAnalysisResponse,
  SellerDashboardStats,
} from '@/types/api'

export async function getDashboardStats(): Promise<SellerDashboardStats> {
  return apiRequest<SellerDashboardStats>('/seller/dashboard/stats', { auth: true })
}

export async function listSellerSupporters(
  projectId?: string,
  limit = 50,
  offset = 0,
): Promise<ListSellerSupportersResponse> {
  const params = new URLSearchParams()
  params.set('limit', String(Math.min(100, Math.max(1, limit))))
  params.set('offset', String(Math.max(0, offset)))
  if (projectId) {
    params.set('project_id', projectId)
  }
  return apiRequest<ListSellerSupportersResponse>(`/seller/supporters?${params.toString()}`, {
    auth: true,
  })
}

export async function exportSupportersCsv(projectId?: string): Promise<Blob> {
  const params = new URLSearchParams()
  if (projectId) {
    params.set('project_id', projectId)
  }
  const qs = params.toString()
  const path = qs ? `/seller/supporters/export?${qs}` : '/seller/supporters/export'
  const token = getAccessToken()
  const headers = new Headers({ Accept: 'text/csv' })
  if (token) {
    headers.set('Authorization', `Bearer ${token}`)
  }
  const response = await fetch(`${apiBaseUrl()}${path}`, {
    credentials: 'include',
    headers,
  })
  if (!response.ok) {
    throw new Error('export failed')
  }
  return response.blob()
}

export async function getProjectAnalysis(): Promise<ProjectAnalysisResponse> {
  return apiRequest<ProjectAnalysisResponse>('/seller/projects/analysis', { auth: true })
}

export async function getConnectAccount(): Promise<ConnectAccountResponse> {
  return apiRequest<ConnectAccountResponse>('/seller/connect', { auth: true })
}

export async function startConnectOnboarding(): Promise<ConnectOnboardingResponse> {
  return apiRequest<ConnectOnboardingResponse>('/seller/connect/onboarding', {
    method: 'POST',
    auth: true,
  })
}

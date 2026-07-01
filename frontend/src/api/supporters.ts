import { apiRequest } from '@/lib/api-client'
import type {
  FavoriteStatus,
  ListFavoritesResponse,
  ListSupportersResponse,
  Supporter,
  UpgradePlanResponse,
} from '@/types/api'
import { isValidUUID } from '@/lib/validation'

export async function listSupporters(): Promise<ListSupportersResponse> {
  return apiRequest<ListSupportersResponse>('/supporters', { auth: true })
}

export async function upgradeSupporter(supporterId: string, planId: string): Promise<UpgradePlanResponse> {
  if (!isValidUUID(supporterId) || !isValidUUID(planId)) {
    throw new Error('invalid id')
  }
  return apiRequest<UpgradePlanResponse>(`/supporters/${supporterId}/upgrade`, {
    method: 'POST',
    auth: true,
    body: { plan_id: planId },
  })
}

export async function confirmUpgrade(supporterId: string, paymentIntentId: string): Promise<Supporter> {
  if (!isValidUUID(supporterId) || !paymentIntentId.trim()) {
    throw new Error('invalid id')
  }
  return apiRequest<Supporter>(`/supporters/${supporterId}/upgrade-confirm`, {
    method: 'POST',
    auth: true,
    body: { payment_intent_id: paymentIntentId.trim() },
  })
}

export async function cancelSupporter(supporterId: string): Promise<Supporter> {
  if (!isValidUUID(supporterId)) {
    throw new Error('invalid id')
  }
  return apiRequest<Supporter>(`/supporters/${supporterId}/cancel`, {
    method: 'POST',
    auth: true,
  })
}

export async function rejoinSupporter(supporterId: string): Promise<Supporter> {
  if (!isValidUUID(supporterId)) {
    throw new Error('invalid id')
  }
  return apiRequest<Supporter>(`/supporters/${supporterId}/rejoin`, {
    method: 'POST',
    auth: true,
  })
}

export async function changeBilling(
  supporterId: string,
  billingMode: 'SUBSCRIPTION' | 'ONE_TIME',
  paymentMethod: 'CARD' | 'KONBINI',
): Promise<Supporter> {
  if (!isValidUUID(supporterId)) {
    throw new Error('invalid id')
  }
  return apiRequest<Supporter>(`/supporters/${supporterId}/billing/change`, {
    method: 'POST',
    auth: true,
    body: { billing_mode: billingMode, payment_method: paymentMethod },
  })
}

export async function listFavorites(): Promise<ListFavoritesResponse> {
  return apiRequest<ListFavoritesResponse>('/favorites', { auth: true })
}

export async function toggleFavorite(projectId: string, action: 'add' | 'remove'): Promise<FavoriteStatus> {
  if (!isValidUUID(projectId)) {
    throw new Error('invalid project id')
  }
  return apiRequest<FavoriteStatus>('/favorites', {
    method: 'POST',
    auth: true,
    body: { project_id: projectId, action },
  })
}

export async function checkFavorite(projectId: string): Promise<FavoriteStatus> {
  if (!isValidUUID(projectId)) {
    throw new Error('invalid project id')
  }
  return apiRequest<FavoriteStatus>(`/favorites/check/${projectId}`, { auth: true })
}

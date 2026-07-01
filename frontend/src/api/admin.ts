import { apiRequest } from '@/lib/api-client'
import type {
  AdminStats,
  BatchJobRun,
  ListActivityLogsResponse,
  ListAuditLogsResponse,
  ListBatchJobRunsResponse,
  ListProjectsResponse,
  ListPurchasesResponse,
  ListSellerRegistrationsResponse,
  ListUsersResponse,
  ProjectDetail,
  Purchase,
  SecurityEventsResponse,
  SellerRegistrationDetail,
  User,
} from '@/types/api'
import { isValidUUID } from '@/lib/validation'
import { listAdminReports, patchAdminReport } from '@/api/reports'

export { listAdminReports, patchAdminReport }

export async function fetchAdminStats(): Promise<AdminStats> {
  return apiRequest<AdminStats>('/admin/stats', { auth: true })
}

export async function listAdminUsers(limit = 20, offset = 0): Promise<ListUsersResponse> {
  return apiRequest<ListUsersResponse>(
    `/admin/users?limit=${Math.min(100, limit)}&offset=${Math.max(0, offset)}`,
    { auth: true },
  )
}

export async function updateAdminUser(
  userId: string,
  body: { email?: string; name?: string; role?: string; is_active?: boolean },
): Promise<User> {
  if (!isValidUUID(userId)) {
    throw new Error('invalid user id')
  }
  return apiRequest<User>(`/admin/users/${userId}`, { method: 'PUT', auth: true, body })
}

export async function deleteAdminUser(userId: string): Promise<void> {
  if (!isValidUUID(userId)) {
    throw new Error('invalid user id')
  }
  await apiRequest<void>(`/admin/users/${userId}`, { method: 'DELETE', auth: true })
}

export async function resetUser2FA(userId: string): Promise<void> {
  if (!isValidUUID(userId)) {
    throw new Error('invalid user id')
  }
  await apiRequest<void>(`/admin/users/${userId}/reset-2fa`, { method: 'POST', auth: true })
}

export async function changeUserAdminRole(
  userId: string,
  adminRole: string,
  reason?: string,
): Promise<User> {
  if (!isValidUUID(userId)) {
    throw new Error('invalid user id')
  }
  const res = await apiRequest<{ user: User }>(`/admin/users/${userId}/admin-role`, {
    method: 'PATCH',
    auth: true,
    body: { admin_role: adminRole, reason: reason ?? null },
  })
  return res.user
}

export async function listAdminPurchases(limit = 20, offset = 0): Promise<ListPurchasesResponse> {
  return apiRequest<ListPurchasesResponse>(
    `/admin/purchases?limit=${Math.min(100, limit)}&offset=${Math.max(0, offset)}`,
    { auth: true },
  )
}

export async function patchAdminPurchase(purchaseId: string, status: string): Promise<Purchase> {
  if (!isValidUUID(purchaseId)) {
    throw new Error('invalid purchase id')
  }
  return apiRequest<Purchase>(`/admin/purchases/${purchaseId}`, {
    method: 'PATCH',
    auth: true,
    body: { status },
  })
}

export async function listBuyerPending(limit = 20, offset = 0): Promise<ListUsersResponse> {
  return apiRequest<ListUsersResponse>(
    `/admin/reviews/users?limit=${Math.min(100, limit)}&offset=${Math.max(0, offset)}`,
    { auth: true },
  )
}

export async function approveBuyer(userId: string): Promise<User> {
  if (!isValidUUID(userId)) {
    throw new Error('invalid user id')
  }
  return apiRequest<User>(`/admin/reviews/users/${userId}/approve`, {
    method: 'POST',
    auth: true,
  })
}

export async function rejectBuyer(userId: string, reason: string): Promise<User> {
  if (!isValidUUID(userId)) {
    throw new Error('invalid user id')
  }
  return apiRequest<User>(`/admin/reviews/users/${userId}/reject`, {
    method: 'POST',
    auth: true,
    body: { reason: reason.trim() },
  })
}

export async function listSellerRegistrations(
  limit = 20,
  offset = 0,
): Promise<ListSellerRegistrationsResponse> {
  return apiRequest<ListSellerRegistrationsResponse>(
    `/admin/seller-registrations?limit=${Math.min(100, limit)}&offset=${Math.max(0, offset)}`,
    { auth: true },
  )
}

export async function getSellerRegistration(id: string): Promise<SellerRegistrationDetail> {
  if (!isValidUUID(id)) {
    throw new Error('invalid id')
  }
  return apiRequest<SellerRegistrationDetail>(`/admin/seller-registrations/${id}`, { auth: true })
}

export async function approveSellerRegistration(id: string): Promise<void> {
  if (!isValidUUID(id)) {
    throw new Error('invalid id')
  }
  await apiRequest<void>(`/admin/seller-registrations/${id}/approve`, {
    method: 'POST',
    auth: true,
  })
}

export async function rejectSellerRegistration(id: string, reason: string): Promise<void> {
  if (!isValidUUID(id)) {
    throw new Error('invalid id')
  }
  await apiRequest<void>(`/admin/seller-registrations/${id}/reject`, {
    method: 'POST',
    auth: true,
    body: { reason: reason.trim() },
  })
}

export async function listAdminProjects(
  status?: string,
  limit = 20,
  offset = 0,
): Promise<ListProjectsResponse> {
  const params = new URLSearchParams()
  params.set('limit', String(Math.min(100, limit)))
  params.set('offset', String(Math.max(0, offset)))
  if (status) {
    params.set('status', status)
  }
  return apiRequest<ListProjectsResponse>(`/admin/projects?${params.toString()}`, { auth: true })
}

export async function patchAdminProject(
  projectId: string,
  status: string,
  reason?: string,
): Promise<ProjectDetail> {
  if (!isValidUUID(projectId)) {
    throw new Error('invalid project id')
  }
  return apiRequest<ProjectDetail>(`/admin/projects/${projectId}`, {
    method: 'PATCH',
    auth: true,
    body: { status, reason: reason ?? null },
  })
}

export async function deleteAdminProject(projectId: string): Promise<void> {
  if (!isValidUUID(projectId)) {
    throw new Error('invalid project id')
  }
  await apiRequest<void>(`/admin/projects/${projectId}`, { method: 'DELETE', auth: true })
}

export async function patchAdminDeleteRequest(
  projectId: string,
  action: string,
  reason?: string,
): Promise<ProjectDetail> {
  if (!isValidUUID(projectId)) {
    throw new Error('invalid project id')
  }
  return apiRequest<ProjectDetail>(`/admin/projects/${projectId}/delete-request`, {
    method: 'PATCH',
    auth: true,
    body: { action, reason: reason ?? null },
  })
}

export async function listActivityLogs(
  limit = 50,
  offset = 0,
): Promise<ListActivityLogsResponse> {
  return apiRequest<ListActivityLogsResponse>(
    `/admin/activity?limit=${Math.min(200, limit)}&offset=${Math.max(0, offset)}`,
    { auth: true },
  )
}

export async function fetchSecurityEvents(from?: string, to?: string): Promise<SecurityEventsResponse> {
  const params = new URLSearchParams()
  if (from) {
    params.set('from', from)
  }
  if (to) {
    params.set('to', to)
  }
  const qs = params.toString()
  return apiRequest<SecurityEventsResponse>(`/admin/security${qs ? `?${qs}` : ''}`, { auth: true })
}

export async function listBatchRuns(limit = 20): Promise<ListBatchJobRunsResponse> {
  return apiRequest<ListBatchJobRunsResponse>(`/admin/batches?limit=${Math.min(100, limit)}`, {
    auth: true,
  })
}

export async function runBatchJob(jobName: string, dryRun = false): Promise<BatchJobRun> {
  const allowed = new Set(['apply_plans', 'expire_supporters', 'purge_tokens'])
  if (!allowed.has(jobName)) {
    throw new Error('invalid job')
  }
  return apiRequest<BatchJobRun>(`/admin/batches/${jobName}/run`, {
    method: 'POST',
    auth: true,
    body: { dry_run: dryRun },
  })
}

export async function listAuditLogs(limit = 50, offset = 0): Promise<ListAuditLogsResponse> {
  return apiRequest<ListAuditLogsResponse>(
    `/admin/audit-logs?limit=${Math.min(200, limit)}&offset=${Math.max(0, offset)}`,
    { auth: true },
  )
}

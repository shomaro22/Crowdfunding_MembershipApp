import { apiRequest } from '@/lib/api-client'
import type { ListReportsResponse, Report, ReportTargetType } from '@/types/api'
import { isValidUUID, validateReportDetail } from '@/lib/validation'

export async function createReport(
  targetType: ReportTargetType,
  targetId: string,
  reason: string,
  detail?: string,
): Promise<Report> {
  if (!isValidUUID(targetId)) {
    throw new Error('invalid target id')
  }
  const trimmedReason = reason.trim()
  if (trimmedReason.length === 0 || trimmedReason.length > 200) {
    throw new Error('理由は1〜200文字で入力してください')
  }
  if (detail !== undefined) {
    const d = validateReportDetail(detail)
    if (!d.valid) {
      throw new Error(d.error ?? 'invalid detail')
    }
  }
  return apiRequest<Report>('/reports', {
    method: 'POST',
    auth: true,
    body: {
      target_type: targetType,
      target_id: targetId,
      reason: trimmedReason,
      detail: detail?.trim() ?? null,
    },
  })
}

export async function listAdminReports(
  status?: string,
  limit = 20,
  offset = 0,
): Promise<ListReportsResponse> {
  const params = new URLSearchParams()
  params.set('limit', String(Math.min(100, Math.max(1, limit))))
  params.set('offset', String(Math.max(0, offset)))
  if (status) {
    params.set('status', status)
  }
  return apiRequest<ListReportsResponse>(`/admin/reports?${params.toString()}`, { auth: true })
}

export async function patchAdminReport(reportId: string, status: string): Promise<Report> {
  if (!isValidUUID(reportId)) {
    throw new Error('invalid report id')
  }
  return apiRequest<Report>(`/admin/reports/${reportId}`, {
    method: 'PATCH',
    auth: true,
    body: { status },
  })
}

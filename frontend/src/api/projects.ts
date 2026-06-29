import { apiRequest } from '@/lib/api-client'
import type { ListProjectsResponse, ProjectDetail } from '@/types/api'

export interface ProjectSearchParams {
  search?: string
  limit?: number
  offset?: number
}

export async function listPublicProjects(params: ProjectSearchParams = {}): Promise<ListProjectsResponse> {
  const query = new URLSearchParams()
  if (params.search) {
    query.set('search', params.search)
  }
  if (params.limit !== undefined) {
    query.set('limit', String(params.limit))
  }
  if (params.offset !== undefined) {
    query.set('offset', String(params.offset))
  }

  const qs = query.toString()
  return apiRequest<ListProjectsResponse>(`/projects${qs ? `?${qs}` : ''}`)
}

export async function getProject(projectId: string): Promise<ProjectDetail> {
  return apiRequest<ProjectDetail>(`/projects/${encodeURIComponent(projectId)}`)
}

export async function listProjectPlans(projectId: string): Promise<{ plans: ProjectDetail['plans'] }> {
  return apiRequest<{ plans: ProjectDetail['plans'] }>(
    `/projects/${encodeURIComponent(projectId)}/plans`,
  )
}

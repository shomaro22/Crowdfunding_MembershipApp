import { apiRequest } from '@/lib/api-client'
import type {
  CreateProjectPayload,
  ListProjectsResponse,
  ProjectDetail,
  UpdateProjectPayload,
} from '@/types/api'
import { isValidUUID } from '@/lib/validation'

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
  if (!isValidUUID(projectId)) {
    throw new Error('invalid project id')
  }
  return apiRequest<{ plans: ProjectDetail['plans'] }>(
    `/projects/${encodeURIComponent(projectId)}/plans`,
  )
}

export async function listMyProjects(): Promise<ListProjectsResponse> {
  return apiRequest<ListProjectsResponse>('/projects/mine', { auth: true })
}

export async function createProject(payload: CreateProjectPayload): Promise<ProjectDetail> {
  return apiRequest<ProjectDetail>('/projects', {
    method: 'POST',
    auth: true,
    body: payload,
  })
}

export async function updateProject(projectId: string, payload: UpdateProjectPayload): Promise<ProjectDetail> {
  if (!isValidUUID(projectId)) {
    throw new Error('invalid project id')
  }
  return apiRequest<ProjectDetail>(`/projects/${projectId}`, {
    method: 'PUT',
    auth: true,
    body: payload,
  })
}

export async function requestProjectDelete(projectId: string): Promise<ProjectDetail> {
  if (!isValidUUID(projectId)) {
    throw new Error('invalid project id')
  }
  return apiRequest<ProjectDetail>(`/projects/${projectId}/delete-request`, {
    method: 'POST',
    auth: true,
  })
}

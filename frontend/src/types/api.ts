export type UserRole = 'BUYER' | 'SELLER' | 'ADMIN'

export type AdminRole = 'SUPER_ADMIN' | 'REVIEW_ADMIN' | 'SUPPORT_ADMIN' | 'FINANCE_ADMIN'

export type MembershipType = 'BRONZE' | 'SILVER' | 'GOLD'

export type ProjectStatus = 'PENDING' | 'ACTIVE' | 'REJECTED' | 'DELETED'

export type DurationType = 'LIMITED' | 'UNLIMITED'

export interface User {
  id: string
  email: string
  name: string | null
  role: UserRole
  is_active: boolean
  avatar_url: string | null
  created_at: string
  updated_at: string
}

export interface ApiErrorBody {
  error: string
  code: string
}

export interface AuthTokenResponse {
  auth_token: string
  user: User
}

export interface Login2FAResponse {
  requires_2fa: true
  user: User
}

export type LoginResponse = AuthTokenResponse | Login2FAResponse

export interface RefreshResponse {
  auth_token: string
}

export interface RegisterBuyerResponse {
  user: User
  message: string
}

export interface SellerRegistration {
  id: string
  user_id: string
  full_name: string
  birth_date: string
  address: string
  photo_present: boolean
  identity_document_type: string
  has_identity_document_front: boolean
  has_identity_document_back: boolean
  status: string
  rejection_reason: string | null
  created_at: string
  updated_at: string
}

export interface RegisterSellerResponse {
  user: User
  registration: SellerRegistration
}

export interface Verify2FAResponse {
  enabled: boolean
  backup_codes: string[]
}

export interface Setup2FAResponse {
  secret: string
  otpauth_url: string
}

export interface ProjectPlan {
  id: string
  project_id: string
  membership_type: MembershipType
  price: number
  max_supporters: number | null
  benefits: string
  is_active: boolean
  supporter_count: number
}

export interface ProjectSummary {
  id: string
  title: string
  description: string
  target_amount: number
  current_amount: number
  supporter_count: number
  duration_type: DurationType
  start_date: string
  end_date: string | null
  image_url: string | null
  status: ProjectStatus
  seller?: User
  plans: ProjectPlan[]
}

export interface ProjectImage {
  id: string
  image_url: string
  is_main: boolean
  order: number
}

export interface ProjectDetail extends ProjectSummary {
  images: ProjectImage[]
  delete_request: boolean
}

export interface ListProjectsResponse {
  projects: ProjectSummary[]
  total: number
  limit: number
  offset: number
}

export function isLogin2FAResponse(value: LoginResponse): value is Login2FAResponse {
  return 'requires_2fa' in value && value.requires_2fa
}

export function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === 'object' && value !== null
}

export function isApiErrorBody(value: unknown): value is ApiErrorBody {
  if (!isRecord(value)) {
    return false
  }
  return typeof value.error === 'string' && typeof value.code === 'string'
}

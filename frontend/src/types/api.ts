export type UserRole = 'BUYER' | 'SELLER' | 'ADMIN'

export type AdminRole = 'SUPER_ADMIN' | 'REVIEW_ADMIN' | 'SUPPORT_ADMIN' | 'FINANCE_ADMIN'

export type MembershipType = 'BRONZE' | 'SILVER' | 'GOLD'

export type ProjectStatus = 'PENDING' | 'ACTIVE' | 'CANCELLED' | 'DELETE_REQUESTED' | 'DELETED' | 'COMPLETED'

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
  reviewed_at?: string | null
  reviewed_by?: string | null
  created_at: string
  updated_at: string
  user?: User
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
  favorite_count?: number
  duration_type: DurationType
  start_date: string
  end_date: string | null
  image_url: string | null
  status: ProjectStatus
  delete_request?: boolean
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

// --- Extended domain types ---

export type SupporterStatus = 'ACTIVE' | 'CANCELLED' | 'EXPIRED' | 'PENDING'
export type BillingMode = 'SUBSCRIPTION' | 'ONE_TIME'
export type PurchaseStatus = 'PENDING' | 'COMPLETED' | 'FAILED' | 'REFUNDED' | 'CANCELLED'
export type PaymentMethod = 'CARD' | 'KONBINI'
export type RegistrationStatus = 'PENDING' | 'APPROVED' | 'REJECTED'
export type ReportStatus = 'PENDING' | 'IN_REVIEW' | 'RESOLVED' | 'DISMISSED'
export type ReportTargetType = 'PROJECT' | 'USER' | 'MESSAGE'

export interface UserProfile extends User {
  totp_enabled?: boolean
}

export interface Supporter {
  id: string
  project_id: string
  plan_id: string
  user_id: string
  status: SupporterStatus
  billing_mode: BillingMode
  start_date: string
  end_date: string
  pending_plan_id: string | null
  plan_change_date: string | null
  project?: ProjectSummary
  plan?: ProjectPlan
  pending_plan?: ProjectPlan
}

export interface ListSupportersResponse {
  supporters: Supporter[]
  total: number
}

export interface Purchase {
  id: string
  project_id: string
  supporter_id: string | null
  buyer_id: string
  quantity: number
  total_price: number
  status: PurchaseStatus
  payment_method: PaymentMethod
  created_at: string
  project?: ProjectSummary
  buyer?: User
}

export interface ListPurchasesResponse {
  purchases: Purchase[]
  total: number
}

export interface CheckoutSessionResponse {
  session_id: string
  url: string
}

export interface Message {
  id: string
  project_id: string
  sender_id: string
  receiver_id: string
  content: string
  is_read: boolean
  created_at: string
  sender?: User
  receiver?: User
}

export interface MessageThread {
  project_id: string
  project_title: string
  partner: User
  messages: Message[]
  unread_count: number
}

export interface ListMessageThreadsResponse {
  threads: MessageThread[]
  total: number
  page: number
  per_page: number
}

export interface ListFavoritesResponse {
  favorites: ProjectSummary[]
}

export interface FavoriteStatus {
  project_id: string
  is_favorited: boolean
}

export interface SellerDashboardStats {
  total_projects: number
  active_projects: number
  total_supporters: number
  total_revenue: number
}

export interface ConnectAccount {
  stripe_account_id: string
  charges_enabled: boolean
  payouts_enabled: boolean
  details_submitted: boolean
}

export interface ConnectAccountResponse {
  account: ConnectAccount | null
}

export interface ConnectOnboardingResponse {
  url: string
}

export interface ProjectAnalysisItem {
  project_id: string
  title: string
  supporter_count: number
  revenue: number
  plan_breakdown: Record<string, number>
}

export interface ProjectAnalysisResponse {
  projects: ProjectAnalysisItem[]
}

export interface MediaUploadResponse {
  url: string
}

export interface Report {
  id: string
  reporter_id: string
  target_type: ReportTargetType
  target_id: string
  reason: string
  detail: string | null
  status: ReportStatus
  created_at: string
  updated_at: string
}

export interface ListReportsResponse {
  reports: Report[]
  total: number
}

export interface AdminStats {
  total_users: number
  total_projects: number
  total_purchases: number
  total_revenue: number
  recent_users: User[]
}

export interface ListUsersResponse {
  users: User[]
  total: number
}

export interface SellerRegistrationDetail extends SellerRegistration {
  identity_document_front_url?: string
  identity_document_back_url?: string | null
}

export interface ListSellerRegistrationsResponse {
  registrations: SellerRegistration[]
  total: number
}

export interface ActivityLog {
  id: string
  user_id: string | null
  action: string
  resource: string
  resource_id: string
  ip_address: string
  user_agent: string
  request_id: string
  metadata: Record<string, unknown>
  created_at: string
  user?: User
}

export interface ListActivityLogsResponse {
  logs: ActivityLog[]
  total: number
  limit: number
  offset: number
}

export interface BatchJobRun {
  id: string
  job_name: 'apply_plans' | 'expire_supporters' | 'purge_tokens'
  status: 'RUNNING' | 'SUCCESS' | 'FAILED'
  started_at: string
  finished_at: string | null
  processed: number
  failed: number
  error_message: string | null
  triggered_by: string
}

export interface ListBatchJobRunsResponse {
  runs: BatchJobRun[]
  total: number
  limit: number
}

export interface SecurityEventsResponse {
  login_failed_count: number
  rate_limited_count: number
  top_ips: { ip: string; count: number }[]
  from: string
  to: string
}

export interface AuditLog {
  id: string
  admin_id: string
  admin_email: string
  action: string
  resource: string
  resource_id: string
  details: string | null
  reason: string | null
  created_at: string
}

export interface ListAuditLogsResponse {
  logs: AuditLog[]
  total: number
  limit: number
  offset: number
}

export interface NotificationsResponse {
  unread_messages: number
}

export interface UpgradePlanResponse {
  supporter: Supporter
  payment_intent_client_secret: string | null
}

export interface WSTokenResponse {
  ws_token: string
  expires_in: number
}

export interface CreateProjectPayload {
  title: string
  description: string
  target_amount: number
  duration_type: DurationType
  start_date: string
  end_date: string | null
  image_url?: string
  images?: { image_url: string; is_main: boolean; order: number }[]
  plans: { membership_type: MembershipType; price: number; max_supporters?: number; benefits: string }[]
}

export interface UpdateProjectPayload {
  title?: string
  description?: string
  target_amount?: number
  duration_type?: DurationType
  start_date?: string
  end_date?: string | null
  image_url?: string
  images?: { image_url: string; is_main: boolean; order: number }[]
  is_reapply?: boolean
}

export interface SellerSupporterRow {
  id: string
  project_id: string
  plan_id: string
  user_id: string
  status: string
  billing_mode: string
  start_date: string
  end_date: string
  buyer?: User
  project?: ProjectSummary
  plan?: ProjectPlan
}

export interface ListSellerSupportersResponse {
  supporters: SellerSupporterRow[]
  total: number
}

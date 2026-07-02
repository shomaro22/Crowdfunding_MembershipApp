import { apiRequest } from '@/lib/api-client'
import type {
  CheckoutSessionResponse,
  ListPurchasesResponse,
  PaymentMethod,
  Supporter,
} from '@/types/api'
import { isValidUUID } from '@/lib/validation'
import { sanitizeAppPath } from '@/lib/utils'

export interface CheckoutParams {
  projectId: string
  planId: string
  paymentMethod?: PaymentMethod
  successUrl?: string
  cancelUrl?: string
}

export async function createCheckout(params: CheckoutParams): Promise<CheckoutSessionResponse> {
  if (!isValidUUID(params.projectId) || !isValidUUID(params.planId)) {
    throw new Error('invalid ids')
  }
  const origin = window.location.origin
  const body: Record<string, string> = {
    project_id: params.projectId,
    plan_id: params.planId,
  }
  if (params.paymentMethod) {
    body.payment_method = params.paymentMethod
  }
  if (params.successUrl && params.successUrl.startsWith(origin)) {
    body.success_url = params.successUrl
  }
  if (params.cancelUrl && params.cancelUrl.startsWith(origin)) {
    body.cancel_url = params.cancelUrl
  }
  return apiRequest<CheckoutSessionResponse>('/payments/checkout', {
    method: 'POST',
    auth: true,
    body,
  })
}

export async function verifyCheckout(sessionId: string): Promise<Supporter> {
  if (!sessionId || sessionId.length > 256) {
    throw new Error('invalid session id')
  }
  return apiRequest<Supporter>('/payments/verify/checkout', {
    method: 'POST',
    auth: true,
    body: { session_id: sessionId },
  })
}

export interface PaymentIntentResponse {
  client_secret: string
  payment_intent_id: string
}

export async function createPaymentIntent(projectId: string, planId: string): Promise<PaymentIntentResponse> {
  if (!isValidUUID(projectId) || !isValidUUID(planId)) {
    throw new Error('invalid ids')
  }
  return apiRequest<PaymentIntentResponse>('/payments/intent', {
    method: 'POST',
    auth: true,
    body: { project_id: projectId, plan_id: planId },
  })
}

export async function verifyPaymentIntent(paymentIntentId: string): Promise<Supporter> {
  if (!paymentIntentId.trim()) {
    throw new Error('invalid payment intent id')
  }
  return apiRequest<Supporter>('/payments/verify/intent', {
    method: 'POST',
    auth: true,
    body: { payment_intent_id: paymentIntentId.trim() },
  })
}

export async function listPurchases(limit = 20, offset = 0): Promise<ListPurchasesResponse> {
  const safeLimit = Math.min(100, Math.max(1, limit))
  const safeOffset = Math.max(0, offset)
  return apiRequest<ListPurchasesResponse>(
    `/purchases?limit=${safeLimit}&offset=${safeOffset}`,
    { auth: true },
  )
}

export function checkoutSuccessPath(projectId: string): string {
  return sanitizeAppPath(
    `/support/success?project_id=${encodeURIComponent(projectId)}&session_id={CHECKOUT_SESSION_ID}`,
    '/purchases',
  )
}

export function checkoutCancelPath(projectId: string): string {
  return sanitizeAppPath(`/checkout/${encodeURIComponent(projectId)}`, '/')
}

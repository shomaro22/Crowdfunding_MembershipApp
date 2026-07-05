/** Stripe stub（ローカル docker / STRIPE_USE_STUB）の client secret 判定 */
export function isStubPaymentSecret(secret: string): boolean {
  return secret.includes('cs_test_secret') || secret.includes('pi_test_')
}

export function stripePublishableKey(): string {
  const raw = import.meta.env.VITE_STRIPE_PUBLISHABLE_KEY
  return typeof raw === 'string' ? raw.trim() : ''
}

/** 本番 API が返す real PaymentIntent には publishable key が必須 */
export function canUseLiveStripe(clientSecret: string): boolean {
  return stripePublishableKey() !== '' && !isStubPaymentSecret(clientSecret)
}

export function liveStripeConfigMissing(clientSecret: string): boolean {
  return !isStubPaymentSecret(clientSecret) && stripePublishableKey() === ''
}

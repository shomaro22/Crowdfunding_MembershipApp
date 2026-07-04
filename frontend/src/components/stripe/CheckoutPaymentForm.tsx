import { useState, type FormEvent } from 'react'
import { loadStripe } from '@stripe/stripe-js'
import { Elements, PaymentElement, useElements, useStripe } from '@stripe/react-stripe-js'

import { verifyPaymentIntent } from '@/api/payments'
import { digitsOnly } from '@/lib/digits'
import {
  formatCardExpiryInput,
  formatCardNumberInput,
  validateDemoCardCvc,
  validateDemoCardExpiry,
  validateDemoCardNumber,
} from '@/lib/validation'

const stripePublishableKey = import.meta.env.VITE_STRIPE_PUBLISHABLE_KEY as string | undefined

function isStubClientSecret(secret: string): boolean {
  return secret.includes('cs_test_secret') || secret.includes('pi_test_')
}

function canUseLiveStripe(clientSecret: string): boolean {
  return Boolean(stripePublishableKey?.trim()) && !isStubClientSecret(clientSecret)
}

interface CheckoutPaymentFormProps {
  clientSecret: string
  paymentIntentId: string
  amountLabel: string
  /** checkout: 初回応援の verify/intent。plan-upgrade: 呼び出し元が upgrade-confirm を行う */
  paymentPurpose?: 'checkout' | 'plan-upgrade'
  onSuccess: () => void
  onError: (message: string) => void
}

function LivePaymentForm({
  clientSecret,
  paymentIntentId,
  paymentPurpose = 'checkout',
  onSuccess,
  onError,
}: Pick<CheckoutPaymentFormProps, 'clientSecret' | 'paymentIntentId' | 'paymentPurpose' | 'onSuccess' | 'onError'>) {
  const stripe = useStripe()
  const elements = useElements()
  const [processing, setProcessing] = useState(false)
  const [errorMessage, setErrorMessage] = useState('')

  const onSubmit = async (e: FormEvent) => {
    e.preventDefault()
    if (!stripe || !elements) {
      return
    }
    setProcessing(true)
    setErrorMessage('')
    const submitResult = await elements.submit()
    if (submitResult.error) {
      setErrorMessage(submitResult.error.message ?? '決済情報の確認に失敗しました')
      setProcessing(false)
      return
    }
    const result = await stripe.confirmPayment({
      elements,
      clientSecret,
      confirmParams: {
        return_url: window.location.href,
      },
      redirect: 'if_required',
    })
    if (result.error) {
      setErrorMessage(result.error.message ?? '決済に失敗しました')
      setProcessing(false)
      return
    }
    if (paymentPurpose === 'plan-upgrade') {
      onSuccess()
      setProcessing(false)
      return
    }
    try {
      await verifyPaymentIntent(paymentIntentId)
      onSuccess()
    } catch {
      onError('決済の確認に失敗しました')
    } finally {
      setProcessing(false)
    }
  }

  return (
    <form onSubmit={(e) => void onSubmit(e)} className="space-y-4">
      <PaymentElement options={{ layout: 'tabs' }} />
      {errorMessage ? <p className="text-sm text-red-600">{errorMessage}</p> : null}
      <button
        type="submit"
        disabled={!stripe || !elements || processing}
        className="w-full rounded-full bg-purple-700 px-6 py-3 font-bold text-white hover:bg-purple-800 disabled:opacity-50"
      >
        {processing ? '処理中...' : '決済を完了する'}
      </button>
    </form>
  )
}

function DemoPaymentElement({
  paymentIntentId,
  paymentPurpose = 'checkout',
  onSuccess,
  onError,
}: Pick<CheckoutPaymentFormProps, 'paymentIntentId' | 'paymentPurpose' | 'onSuccess' | 'onError'>) {
  const [processing, setProcessing] = useState(false)
  const [cardNumber, setCardNumber] = useState('')
  const [expiry, setExpiry] = useState('')
  const [cvc, setCvc] = useState('')
  const [errorMessage, setErrorMessage] = useState('')

  const onPay = async () => {
    const cardErr = validateDemoCardNumber(cardNumber)
    const expiryErr = validateDemoCardExpiry(expiry)
    const cvcErr = validateDemoCardCvc(cvc)
    if (cardErr || expiryErr || cvcErr) {
      setErrorMessage(cardErr ?? expiryErr ?? cvcErr ?? 'カード情報を確認してください')
      return
    }

    setProcessing(true)
    setErrorMessage('')
    try {
      if (paymentPurpose === 'checkout') {
        await verifyPaymentIntent(paymentIntentId)
      }
      onSuccess()
    } catch {
      onError('決済の確認に失敗しました')
    } finally {
      setProcessing(false)
    }
  }

  return (
    <div className="space-y-4">
      <div className="rounded-lg border border-gray-200 bg-white p-4 shadow-sm">
        <div className="mb-3 flex gap-2 border-b border-gray-100 pb-2 text-sm">
          <span className="rounded-md bg-purple-100 px-3 py-1 font-medium text-purple-900">カード</span>
          <span className="rounded-md px-3 py-1 text-gray-400">コンビニ</span>
        </div>
        <label className="mb-1 block text-xs font-medium text-gray-600">カード番号</label>
        <input
          className="mb-3 w-full rounded-md border border-gray-300 px-3 py-2.5 text-sm"
          placeholder="4242 4242 4242 4242"
          value={cardNumber}
          onChange={(e) => {
            setCardNumber(formatCardNumberInput(e.target.value))
            setErrorMessage('')
          }}
          inputMode="numeric"
          autoComplete="cc-number"
        />
        <div className="grid grid-cols-2 gap-3">
          <div>
            <label className="mb-1 block text-xs font-medium text-gray-600">有効期限</label>
            <input
              className="w-full rounded-md border border-gray-300 px-3 py-2.5 text-sm"
              placeholder="MM / YY"
              value={expiry}
              onChange={(e) => {
                setExpiry(formatCardExpiryInput(e.target.value))
                setErrorMessage('')
              }}
              inputMode="numeric"
              autoComplete="cc-exp"
            />
          </div>
          <div>
            <label className="mb-1 block text-xs font-medium text-gray-600">セキュリティコード</label>
            <input
              className="w-full rounded-md border border-gray-300 px-3 py-2.5 text-sm tracking-widest"
              placeholder="•••"
              value={cvc}
              onChange={(e) => {
                setCvc(digitsOnly(e.target.value, 3))
                setErrorMessage('')
              }}
              type="password"
              inputMode="numeric"
              autoComplete="cc-csc"
              maxLength={3}
            />
          </div>
        </div>
        <p className="mt-3 text-xs text-gray-500">
          デモ環境: カード番号・有効期限・セキュリティコードを検証します。テスト番号は README を参照してください。
        </p>
      </div>
      {errorMessage ? <p className="text-sm text-red-600">{errorMessage}</p> : null}
      <button
        type="button"
        disabled={processing}
        onClick={() => void onPay()}
        className="w-full rounded-full bg-purple-700 px-6 py-3 font-bold text-white hover:bg-purple-800 disabled:opacity-50"
      >
        {processing ? '処理中...' : '決済を完了する'}
      </button>
    </div>
  )
}

export function CheckoutPaymentForm({
  clientSecret,
  paymentIntentId,
  amountLabel,
  paymentPurpose = 'checkout',
  onSuccess,
  onError,
}: CheckoutPaymentFormProps) {
  const [stripePromise] = useState(() =>
    stripePublishableKey?.trim() ? loadStripe(stripePublishableKey) : null,
  )
  const useLive = canUseLiveStripe(clientSecret)

  return (
    <div className="card space-y-4 border-2 border-purple-200">
      <h2 className="text-lg font-bold text-gray-900">お支払い</h2>
      <p className="text-sm text-gray-600">
        金額: <span className="font-bold text-purple-800">{amountLabel}</span>
      </p>
      {useLive && stripePromise ? (
        <Elements
          stripe={stripePromise}
          options={{
            clientSecret,
            appearance: { theme: 'stripe' },
          }}
        >
          <LivePaymentForm
            clientSecret={clientSecret}
            paymentIntentId={paymentIntentId}
            paymentPurpose={paymentPurpose}
            onSuccess={onSuccess}
            onError={onError}
          />
        </Elements>
      ) : (
        <DemoPaymentElement
          paymentIntentId={paymentIntentId}
          paymentPurpose={paymentPurpose}
          onSuccess={onSuccess}
          onError={onError}
        />
      )}
    </div>
  )
}

import { useState } from 'react'

import { verifyPaymentIntent } from '@/api/payments'

interface CheckoutPaymentFormProps {
  clientSecret: string
  paymentIntentId: string
  amountLabel: string
  onSuccess: () => void
  onError: (message: string) => void
}

/** デモ／スタブ環境向けのインライン決済（NewFF の Payment Element 相当の確認ステップ） */
export function CheckoutPaymentForm({
  clientSecret,
  paymentIntentId,
  amountLabel,
  onSuccess,
  onError,
}: CheckoutPaymentFormProps) {
  const [processing, setProcessing] = useState(false)

  const onPay = async () => {
    setProcessing(true)
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
    <div className="card space-y-4 border-2 border-purple-200">
      <h2 className="text-lg font-bold text-gray-900">お支払い</h2>
      <p className="text-sm text-gray-600">
        金額: <span className="font-bold text-purple-800">{amountLabel}</span>
      </p>
      <div className="rounded-lg border border-dashed border-purple-200 bg-purple-50/50 p-4 text-sm text-gray-600">
        <p className="mb-2 font-medium text-purple-900">カード情報（デモ）</p>
        <p>本番では Stripe Payment Element が表示されます。デモ環境では下のボタンで決済を完了できます。</p>
        <p className="mt-2 truncate text-xs text-gray-400">Intent: {paymentIntentId}</p>
        <p className="truncate text-xs text-gray-400">Secret: {clientSecret.slice(0, 24)}…</p>
      </div>
      <button
        type="button"
        disabled={processing}
        onClick={() => void onPay()}
        className="w-full rounded-full bg-purple-700 px-6 py-3 font-bold text-white hover:bg-purple-800 disabled:opacity-50"
      >
        {processing ? '処理中...' : 'デモ決済を完了する'}
      </button>
    </div>
  )
}

import { useEffect, useState } from 'react'
import { Link, useSearchParams } from 'react-router-dom'

import { verifyCheckout, verifyPaymentIntent } from '@/api/payments'
import { Loading } from '@/components/common/Loading'
import { ErrorMessage } from '@/components/common/ErrorMessage'
import { parseUUID } from '@/lib/validation'

const VERIFY_RETRIES = 5
const VERIFY_DELAY_MS = 1500

async function verifyWithRetry(sessionId: string): Promise<void> {
  let lastErr: unknown
  for (let attempt = 0; attempt < VERIFY_RETRIES; attempt++) {
    try {
      await verifyCheckout(sessionId)
      return
    } catch (err) {
      lastErr = err
      if (attempt < VERIFY_RETRIES - 1) {
        await new Promise((resolve) => window.setTimeout(resolve, VERIFY_DELAY_MS))
      }
    }
  }
  throw lastErr
}

export function SupportSuccessPage() {
  const [searchParams] = useSearchParams()
  const sessionId = searchParams.get('session_id') ?? ''
  const paymentIntentId = searchParams.get('payment_intent_id') ?? ''
  const projectId = parseUUID(searchParams.get('project_id'))
  const [status, setStatus] = useState<'loading' | 'ok' | 'error'>('loading')

  useEffect(() => {
    const hasSession = sessionId && sessionId.length <= 256 && !sessionId.includes('{')
    const hasIntent = paymentIntentId.trim().length > 0
    if (!hasSession && !hasIntent) {
      setStatus('error')
      return
    }
    void (async () => {
      try {
        if (hasIntent) {
          await verifyPaymentIntent(paymentIntentId)
        } else {
          await verifyWithRetry(sessionId)
        }
        setStatus('ok')
      } catch {
        setStatus('error')
      }
    })()
  }, [sessionId, paymentIntentId])

  if (status === 'loading') {
    return <Loading label="決済を確認中..." />
  }
  if (status === 'error') {
    return (
      <ErrorMessage
        message="決済の確認に失敗しました。Webhook 反映を待つか、応援履歴をご確認ください。"
        onRetry={() => window.location.reload()}
      />
    )
  }

  return (
    <div className="card mx-auto max-w-lg text-center">
      <h1 className="mb-4 text-2xl font-bold text-green-800">応援ありがとうございます</h1>
      <p className="mb-6 text-gray-700">決済が完了しました。</p>
      <Link to="/purchases" className="btn-primary inline-flex rounded-full">応援履歴を見る</Link>
      {projectId ? (
        <Link to={`/projects/${projectId}`} className="ml-3 text-sm text-purple-700">プロジェクトへ</Link>
      ) : null}
    </div>
  )
}

export function SupportCancelPage() {
  return (
    <div className="card mx-auto max-w-lg text-center">
      <h1 className="mb-4 text-2xl font-bold">決済がキャンセルされました</h1>
      <Link to="/" className="btn-primary inline-flex rounded-full">トップへ</Link>
    </div>
  )
}

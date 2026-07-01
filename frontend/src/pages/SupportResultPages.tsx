import { useEffect, useState } from 'react'
import { Link, useSearchParams } from 'react-router-dom'

import { verifyCheckout } from '@/api/payments'
import { Loading } from '@/components/common/Loading'
import { ErrorMessage } from '@/components/common/ErrorMessage'
import { parseUUID } from '@/lib/validation'

export function SupportSuccessPage() {
  const [searchParams] = useSearchParams()
  const sessionId = searchParams.get('session_id') ?? ''
  const projectId = parseUUID(searchParams.get('project_id'))
  const [status, setStatus] = useState<'loading' | 'ok' | 'error'>('loading')

  useEffect(() => {
    if (!sessionId || sessionId.length > 256) {
      setStatus('error')
      return
    }
    void (async () => {
      try {
        await verifyCheckout(sessionId)
        setStatus('ok')
      } catch {
        setStatus('error')
      }
    })()
  }, [sessionId])

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

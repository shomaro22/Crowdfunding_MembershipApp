import { useEffect, useState } from 'react'
import { Link } from 'react-router-dom'

import { getConnectAccount } from '@/api/seller'
import { Loading } from '@/components/common/Loading'

export function SellerConnectReturnPage() {
  const [status, setStatus] = useState<'loading' | 'ok' | 'error'>('loading')

  useEffect(() => {
    void (async () => {
      try {
        await getConnectAccount()
        setStatus('ok')
      } catch {
        setStatus('error')
      }
    })()
  }, [])

  if (status === 'loading') {
    return <Loading label="Connect 状態を確認中..." />
  }

  return (
    <div className="card mx-auto max-w-lg text-center">
      <h1 className="mb-4 text-xl font-bold">
        {status === 'ok' ? 'Stripe Connect 連携を更新しました' : 'Connect 状態の確認に失敗しました'}
      </h1>
      <Link to="/seller/dashboard" className="btn-primary inline-flex rounded-full">ダッシュボードへ</Link>
    </div>
  )
}

import { useEffect, useState } from 'react'
import { Link } from 'react-router-dom'

import { listPurchases } from '@/api/payments'
import {
  cancelSupporter,
  listSupporters,
  rejoinSupporter,
} from '@/api/supporters'
import { Loading } from '@/components/common/Loading'
import { ErrorMessage } from '@/components/common/ErrorMessage'
import { formatCurrencyYen } from '@/lib/utils'
import type { Purchase, Supporter } from '@/types/api'
import { ApiError } from '@/lib/api-client'

export function PurchasesPage() {
  const [purchases, setPurchases] = useState<Purchase[]>([])
  const [supporters, setSupporters] = useState<Supporter[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')
  const [actionMsg, setActionMsg] = useState('')

  const load = async () => {
    setLoading(true)
    setError('')
    try {
      const [p, s] = await Promise.all([listPurchases(), listSupporters()])
      setPurchases(p.purchases)
      setSupporters(s.supporters)
    } catch {
      setError('履歴の取得に失敗しました')
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    void load()
  }, [])

  const onCancel = async (id: string) => {
    if (!window.confirm('メンバーシップを解約しますか？')) {
      return
    }
    setActionMsg('')
    try {
      await cancelSupporter(id)
      await load()
      setActionMsg('解約手続きを受け付けました')
    } catch (err) {
      setActionMsg(err instanceof ApiError ? err.message : '操作に失敗しました')
    }
  }

  const onRejoin = async (id: string) => {
    setActionMsg('')
    try {
      await rejoinSupporter(id)
      await load()
      setActionMsg('再参加しました')
    } catch (err) {
      setActionMsg(err instanceof ApiError ? err.message : '操作に失敗しました')
    }
  }

  if (loading) {
    return <Loading />
  }
  if (error) {
    return <ErrorMessage message={error} onRetry={() => void load()} />
  }

  return (
    <div className="space-y-10">
      <h1 className="text-2xl font-bold">応援履歴</h1>
      {actionMsg ? <p className="text-sm text-purple-800">{actionMsg}</p> : null}

      <section>
        <h2 className="mb-4 text-lg font-semibold">メンバーシップ</h2>
        {supporters.length === 0 ? (
          <p className="text-gray-600">応援中のプロジェクトはありません</p>
        ) : (
          <ul className="space-y-4">
            {supporters.map((s) => (
              <li key={s.id} className="card flex flex-wrap items-center justify-between gap-4">
                <div>
                  <Link to={`/projects/${s.project_id}`} className="font-bold text-purple-800">
                    {s.project?.title ?? s.project_id}
                  </Link>
                  <p className="text-sm text-gray-600">
                    {s.plan?.membership_type} · {s.status} · 〜{new Date(s.end_date).toLocaleDateString('ja-JP')}
                  </p>
                </div>
                <div className="flex gap-2">
                  {s.status === 'ACTIVE' ? (
                    <button type="button" onClick={() => void onCancel(s.id)} className="btn-secondary rounded-full text-sm">解約</button>
                  ) : null}
                  {s.status === 'CANCELLED' || s.status === 'EXPIRED' ? (
                    <button type="button" onClick={() => void onRejoin(s.id)} className="btn-primary rounded-full text-sm">再参加</button>
                  ) : null}
                </div>
              </li>
            ))}
          </ul>
        )}
      </section>

      <section>
        <h2 className="mb-4 text-lg font-semibold">購入履歴</h2>
        {purchases.length === 0 ? (
          <p className="text-gray-600">購入履歴はありません</p>
        ) : (
          <ul className="space-y-3">
            {purchases.map((p) => (
              <li key={p.id} className="card text-sm">
                <span className="font-medium">{p.project?.title ?? p.project_id}</span>
                {' · '}
                {formatCurrencyYen(p.total_price)}
                {' · '}
                {p.status}
                {' · '}
                {new Date(p.created_at).toLocaleString('ja-JP')}
              </li>
            ))}
          </ul>
        )}
      </section>
    </div>
  )
}

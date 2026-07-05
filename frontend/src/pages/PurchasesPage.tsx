import { useEffect, useState } from 'react'
import { Link } from 'react-router-dom'

import { listPurchases } from '@/api/payments'
import { cancelSupporter, listSupporters, rejoinSupporter } from '@/api/supporters'
import { ConfirmDialog } from '@/components/common/ConfirmDialog'
import { Loading } from '@/components/common/Loading'
import { ErrorMessage } from '@/components/common/ErrorMessage'
import { useToast } from '@/components/common/Toast'
import { formatCurrencyYen } from '@/lib/utils'
import type { Purchase, Supporter } from '@/types/api'
import { ApiError } from '@/lib/api-client'

export function PurchasesPage() {
  const { toast } = useToast()
  const [purchases, setPurchases] = useState<Purchase[]>([])
  const [supporters, setSupporters] = useState<Supporter[]>([])
  const [loading, setLoading] = useState(true)
  const [errorMessage, setErrorMessage] = useState('')
  const [cancelTarget, setCancelTarget] = useState<string | null>(null)
  const [rejoinTarget, setRejoinTarget] = useState<string | null>(null)

  const load = async () => {
    setLoading(true)
    setErrorMessage('')
    try {
      const [p, s] = await Promise.all([listPurchases(), listSupporters()])
      setPurchases(p.purchases)
      setSupporters(s.supporters)
    } catch {
      setErrorMessage('履歴の取得に失敗しました')
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    void load()
  }, [])

  const onCancel = async (id: string) => {
    try {
      await cancelSupporter(id)
      await load()
      toast('解約手続きを受け付けました', 'success')
    } catch (err) {
      toast(err instanceof ApiError ? err.message : '操作に失敗しました', 'error')
    } finally {
      setCancelTarget(null)
    }
  }

  const onRejoin = async (id: string) => {
    try {
      await rejoinSupporter(id)
      await load()
      toast('再参加しました', 'success')
    } catch (err) {
      toast(err instanceof ApiError ? err.message : '操作に失敗しました', 'error')
    }
  }

  if (loading) {
    return <Loading />
  }
  if (errorMessage) {
    return <ErrorMessage message={errorMessage} onRetry={() => void load()} />
  }

  return (
    <div className="space-y-10">
      <h1 className="text-2xl font-bold">応援履歴</h1>

      <ConfirmDialog
        open={cancelTarget !== null}
        title="メンバーシップ解約"
        message="メンバーシップを解約しますか？"
        confirmLabel="解約する"
        danger
        onConfirm={() => {
          if (cancelTarget) {
            void onCancel(cancelTarget)
          }
        }}
        onCancel={() => setCancelTarget(null)}
      />

      <ConfirmDialog
        open={rejoinTarget !== null}
        title="再参加の確認"
        message="このプロジェクトに再参加しますか？月額料金が発生します。"
        confirmLabel="再参加する"
        onConfirm={() => {
          if (rejoinTarget) {
            void onRejoin(rejoinTarget)
          }
          setRejoinTarget(null)
        }}
        onCancel={() => setRejoinTarget(null)}
      />

      <section>
        <h2 className="mb-4 text-lg font-semibold">メンバーシップ</h2>
        {supporters.length === 0 ? (
          <p className="text-gray-600">応援中のプロジェクトはありません</p>
        ) : (
          <ul className="space-y-4">
            {supporters.map((s) => (
              <li key={s.id} className="card flex flex-wrap items-center justify-between gap-4">
                <div>
                  <Link to={`/projects/${s.project_id}`} className="font-bold text-purple-800 hover:underline">
                    {s.project?.title ?? s.project_id}
                  </Link>
                  <p className="text-sm text-gray-600">
                    {s.plan?.membership_type} · {s.status} · 〜{new Date(s.end_date).toLocaleDateString('ja-JP')}
                  </p>
                </div>
                <div className="flex flex-wrap gap-2">
                  {s.status === 'ACTIVE' ? (
                    <button type="button" onClick={() => setCancelTarget(s.id)} className="btn-secondary rounded-full text-sm">
                      解約
                    </button>
                  ) : null}
                  {s.status === 'CANCELLED' || s.status === 'EXPIRED' ? (
                    <button
                      type="button"
                      onClick={() => setRejoinTarget(s.id)}
                      className="rounded-full bg-purple-700 px-4 py-2 text-sm font-medium text-white hover:bg-purple-800"
                    >
                      再参加
                    </button>
                  ) : null}
                </div>
              </li>
            ))}
          </ul>
        )}
      </section>

      <section>
        <h2 className="mb-4 text-lg font-semibold">お支払い記録</h2>
        {purchases.length === 0 ? (
          <p className="text-gray-600">お支払い記録はありません</p>
        ) : (
          <ul className="space-y-3">
            {purchases.map((p) => (
              <li key={p.id} className="card text-sm">
                <Link to={`/projects/${p.project_id}`} className="font-medium text-purple-800 hover:underline">
                  {p.project?.title ?? p.project_id}
                </Link>
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

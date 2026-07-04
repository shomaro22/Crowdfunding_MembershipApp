import { useEffect, useState } from 'react'
import { Link } from 'react-router-dom'

import { getProject } from '@/api/projects'
import { listPurchases } from '@/api/payments'
import {
  cancelSupporter,
  confirmUpgrade,
  listSupporters,
  rejoinSupporter,
  upgradeSupporter,
} from '@/api/supporters'
import { ConfirmDialog } from '@/components/common/ConfirmDialog'
import { Loading } from '@/components/common/Loading'
import { ErrorMessage } from '@/components/common/ErrorMessage'
import { CheckoutPaymentForm } from '@/components/stripe/CheckoutPaymentForm'
import { useToast } from '@/components/common/Toast'
import { planSelectedClass, planTierLabel, sortPlansByTier } from '@/lib/plans'
import { formatCurrencyYen } from '@/lib/utils'
import type { ProjectDetail, Purchase, Supporter } from '@/types/api'
import { ApiError } from '@/lib/api-client'

export function PurchasesPage() {
  const { toast } = useToast()
  const [purchases, setPurchases] = useState<Purchase[]>([])
  const [supporters, setSupporters] = useState<Supporter[]>([])
  const [loading, setLoading] = useState(true)
  const [errorMessage, setErrorMessage] = useState('')
  const [cancelTarget, setCancelTarget] = useState<string | null>(null)
  const [planChangeTarget, setPlanChangeTarget] = useState<Supporter | null>(null)
  const [planChangeProject, setPlanChangeProject] = useState<ProjectDetail | null>(null)
  const [planChangeLoading, setPlanChangeLoading] = useState(false)
  const [pendingPlanId, setPendingPlanId] = useState<string | null>(null)
  const [rejoinTarget, setRejoinTarget] = useState<string | null>(null)
  const [upgradePayment, setUpgradePayment] = useState<{
    supporterId: string
    planId: string
    clientSecret: string
    paymentIntentId: string
    amountLabel: string
  } | null>(null)

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

  useEffect(() => {
    if (!planChangeTarget) {
      setPlanChangeProject(null)
      return
    }
    void (async () => {
      setPlanChangeLoading(true)
      try {
        const detail = await getProject(planChangeTarget.project_id)
        setPlanChangeProject(detail)
      } catch {
        toast('プラン情報の取得に失敗しました', 'error')
        setPlanChangeTarget(null)
      } finally {
        setPlanChangeLoading(false)
      }
    })()
  }, [planChangeTarget, toast])

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

  const onUpgradePlan = async (supporterId: string, planId: string) => {
    try {
      const res = await upgradeSupporter(supporterId, planId)
      if (res.payment_intent_id && res.payment_intent_client_secret) {
        const plan = planChangeProject?.plans.find((p) => p.id === planId)
        setUpgradePayment({
          supporterId,
          planId,
          clientSecret: res.payment_intent_client_secret,
          paymentIntentId: res.payment_intent_id,
          amountLabel: plan ? `${formatCurrencyYen(plan.price)}/月（差額）` : '差額',
        })
        setPlanChangeTarget(null)
        return
      }
      await load()
      toast('プラン変更を受け付けました', 'success')
      setPlanChangeTarget(null)
    } catch (err) {
      let msg = err instanceof ApiError ? err.message : 'プラン変更に失敗しました'
      if (err instanceof ApiError) {
        if (err.code === 'FORBIDDEN') {
          msg = '出品者の Stripe 設定が未完了のため、プラン変更できません'
        } else if (err.code === 'BAD_REQUEST' && err.message.includes('bad request')) {
          msg = 'プラン変更の処理中です。しばらく待ってから再度お試しください'
        }
      }
      toast(msg, 'error')
    }
  }

  const onUpgradePaymentSuccess = async () => {
    if (!upgradePayment) {
      return
    }
    try {
      await confirmUpgrade(upgradePayment.supporterId, upgradePayment.paymentIntentId)
      await load()
      toast('プラン変更が完了しました', 'success')
    } catch (err) {
      toast(err instanceof ApiError ? err.message : 'プラン変更の確定に失敗しました', 'error')
    } finally {
      setUpgradePayment(null)
    }
  }

  if (loading) {
    return <Loading />
  }
  if (errorMessage) {
    return <ErrorMessage message={errorMessage} onRetry={() => void load()} />
  }

  const changePlans = planChangeProject
    ? sortPlansByTier(planChangeProject.plans.filter((p) => p.is_active))
    : []

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
        open={pendingPlanId !== null && planChangeTarget !== null}
        title="プラン変更の確認"
        message="選択したプランに変更しますか？差額の決済が発生する場合があります。"
        confirmLabel="変更する"
        onConfirm={() => {
          if (planChangeTarget && pendingPlanId) {
            void onUpgradePlan(planChangeTarget.id, pendingPlanId)
          }
          setPendingPlanId(null)
        }}
        onCancel={() => setPendingPlanId(null)}
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

      {planChangeTarget ? (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/40 p-4">
          <div className="w-full max-w-lg rounded-2xl bg-white p-6 shadow-xl">
            <h2 className="mb-4 text-lg font-bold">プラン変更</h2>
            <p className="mb-4 text-sm text-gray-600">
              {planChangeTarget.project?.title ?? 'プロジェクト'} — 現在: {planChangeTarget.plan?.membership_type}
            </p>
            {planChangeLoading ? (
              <Loading label="プラン読み込み中..." />
            ) : (
              <ul className="mb-4 space-y-2">
                {changePlans.map((plan) => (
                  <li key={plan.id}>
                    <button
                      type="button"
                      disabled={plan.id === planChangeTarget.plan_id}
                      onClick={() => setPendingPlanId(plan.id)}
                      className={`w-full rounded-lg border p-3 text-left text-sm ${
                        plan.id === planChangeTarget.plan_id
                          ? 'cursor-not-allowed border-gray-200 bg-gray-50 text-gray-400'
                          : `border-purple-200 hover:bg-purple-50 ${planSelectedClass(plan.membership_type)}`
                      }`}
                    >
                      <span className="font-bold">{planTierLabel(plan.membership_type)}</span>
                      {' · '}
                      {formatCurrencyYen(plan.price)}/月
                    </button>
                  </li>
                ))}
              </ul>
            )}
            <button type="button" onClick={() => setPlanChangeTarget(null)} className="btn-secondary w-full rounded-full">
              閉じる
            </button>
          </div>
        </div>
      ) : null}

      {upgradePayment ? (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/40 p-4">
          <div className="max-h-[90vh] w-full max-w-lg overflow-y-auto rounded-2xl bg-white p-6 shadow-xl">
            <h2 className="mb-4 text-lg font-bold">プラン変更のお支払い</h2>
            <CheckoutPaymentForm
              clientSecret={upgradePayment.clientSecret}
              paymentIntentId={upgradePayment.paymentIntentId}
              amountLabel={upgradePayment.amountLabel}
              paymentPurpose="plan-upgrade"
              onSuccess={() => void onUpgradePaymentSuccess()}
              onError={(msg) => toast(msg, 'error')}
            />
            <button
              type="button"
              className="btn-secondary mt-4 w-full rounded-full"
              onClick={() => setUpgradePayment(null)}
            >
              キャンセル
            </button>
          </div>
        </div>
      ) : null}

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
                    <>
                      <button type="button" onClick={() => setPlanChangeTarget(s)} className="btn-primary rounded-full text-sm">
                        プラン変更
                      </button>
                      <button type="button" onClick={() => setCancelTarget(s.id)} className="btn-secondary rounded-full text-sm">
                        解約
                      </button>
                    </>
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

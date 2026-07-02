import { useEffect, useState } from 'react'
import { Link, useNavigate, useParams, useSearchParams } from 'react-router-dom'

import { getProject } from '@/api/projects'
import {
  checkoutCancelPath,
  checkoutSuccessPath,
  createCheckout,
  createPaymentIntent,
} from '@/api/payments'
import { CheckoutPaymentForm } from '@/components/stripe/CheckoutPaymentForm'
import { useAuth } from '@/auth/AuthProvider'
import { Loading } from '@/components/common/Loading'
import { ErrorMessage } from '@/components/common/ErrorMessage'
import { ApiError } from '@/lib/api-client'
import { planSelectedClass, planTierLabel, sortPlansByTier } from '@/lib/plans'
import { formatCurrencyYen } from '@/lib/utils'
import { isValidUUID, parseUUID } from '@/lib/validation'
import type { PaymentMethod, ProjectDetail } from '@/types/api'

function checkoutErrorMessage(err: unknown): string {
  if (err instanceof ApiError) {
    if (err.code === 'ALREADY_SUPPORTING' || (err.status === 409 && err.message.includes('already supporting'))) {
      return 'このプロジェクトは既に応援中です。応援履歴からプラン変更・解約ができます。'
    }
    if (err.code === 'PENDING_CHECKOUT') {
      return '未完了の決済があります。しばらく待ってから再度お試しください。'
    }
    if (err.status === 409) {
      return 'このプロジェクトは現在応援できません。応援履歴をご確認ください。'
    }
    if (err.status === 503) {
      return '決済サービスが一時的に利用できません。しばらくしてからお試しください。'
    }
    return err.message
  }
  return 'チェックアウトの開始に失敗しました'
}

export function CheckoutPage() {
  const { projectId } = useParams<{ projectId: string }>()
  const [searchParams] = useSearchParams()
  const navigate = useNavigate()
  const { user } = useAuth()
  const [project, setProject] = useState<ProjectDetail | null>(null)
  const [selectedPlanId, setSelectedPlanId] = useState<string | null>(null)
  const [paymentMethod, setPaymentMethod] = useState<PaymentMethod>('CARD')
  const [loading, setLoading] = useState(true)
  const [submitting, setSubmitting] = useState(false)
  const [error, setError] = useState('')
  const [clientSecret, setClientSecret] = useState<string | null>(null)
  const [paymentIntentId, setPaymentIntentId] = useState<string | null>(null)
  const [showPaymentForm, setShowPaymentForm] = useState(false)

  useEffect(() => {
    const id = parseUUID(projectId)
    if (!id) {
      setError('プロジェクト ID が不正です')
      setLoading(false)
      return
    }
    const planFromQuery = parseUUID(searchParams.get('plan_id'))
    void (async () => {
      try {
        const detail = await getProject(id)
        setProject(detail)
        if (planFromQuery && detail.plans.some((p) => p.id === planFromQuery)) {
          setSelectedPlanId(planFromQuery)
        } else {
          const first = detail.plans.find((p) => p.is_active)
          if (first) {
            setSelectedPlanId(first.id)
          }
        }
      } catch {
        setError('プロジェクトの取得に失敗しました')
      } finally {
        setLoading(false)
      }
    })()
  }, [projectId, searchParams])

  const selectedPlan = project?.plans.find((p) => p.id === selectedPlanId) ?? null

  const startCheckout = async () => {
    if (!project || !selectedPlanId || !user) {
      return
    }
    setSubmitting(true)
    setError('')
    setShowPaymentForm(false)
    setClientSecret(null)
    setPaymentIntentId(null)
    try {
      const origin = window.location.origin
      const success = `${origin}${checkoutSuccessPath(project.id)}`
      const cancel = `${origin}${checkoutCancelPath(project.id)}`

      if (paymentMethod === 'KONBINI') {
        const session = await createCheckout({
          projectId: project.id,
          planId: selectedPlanId,
          paymentMethod,
          successUrl: success,
          cancelUrl: cancel,
        })
        if (!session.url.startsWith('http://') && !session.url.startsWith('https://')) {
          setError('決済 URL が無効です')
          return
        }
        window.location.assign(session.url)
        return
      }

      const intent = await createPaymentIntent(project.id, selectedPlanId)
      if (!intent.client_secret || !intent.payment_intent_id) {
        setError('決済セッションの作成に失敗しました')
        return
      }
      setClientSecret(intent.client_secret)
      setPaymentIntentId(intent.payment_intent_id)
      setShowPaymentForm(true)
    } catch (err) {
      setError(checkoutErrorMessage(err))
    } finally {
      setSubmitting(false)
    }
  }

  const onPaymentSuccess = () => {
    if (!project || !paymentIntentId) {
      return
    }
    void navigate(
      `/support/success?project_id=${encodeURIComponent(project.id)}&payment_intent_id=${encodeURIComponent(paymentIntentId)}`,
      { replace: true },
    )
  }

  if (loading) {
    return <Loading />
  }
  if (error && !project) {
    return <ErrorMessage message={error} />
  }
  if (!project) {
    return <ErrorMessage message="プロジェクトが見つかりません" />
  }

  const activePlans = sortPlansByTier(project.plans.filter((p) => p.is_active))

  return (
    <div className="mx-auto max-w-2xl">
      <h1 className="mb-2 text-2xl font-bold">応援する</h1>
      <p className="mb-6 text-gray-600">{project.title}</p>
      {error ? <p className="mb-4 rounded-lg bg-red-50 px-4 py-2 text-sm text-red-600">{error}</p> : null}

      {!showPaymentForm ? (
        <>
          <div className="mb-6 space-y-3">
            {activePlans.map((plan) => {
              const selected = selectedPlanId === plan.id
              return (
                <label
                  key={plan.id}
                  className={`card flex cursor-pointer gap-3 transition-all ${
                    selected
                      ? planSelectedClass(plan.membership_type)
                      : 'border border-purple-100 hover:border-purple-300'
                  }`}
                >
                  <input
                    type="radio"
                    name="plan"
                    checked={selected}
                    onChange={() => setSelectedPlanId(plan.id)}
                    className="mt-1 accent-purple-700"
                  />
                  <div>
                    <p className={`font-bold ${selected ? 'text-purple-900' : 'text-gray-800'}`}>
                      {planTierLabel(plan.membership_type)}
                    </p>
                    <p className={`text-xl font-extrabold ${selected ? 'text-purple-700' : ''}`}>
                      {formatCurrencyYen(plan.price)}
                      <span className="text-sm font-normal text-gray-500">/月</span>
                    </p>
                    <p className="text-sm text-gray-600">{plan.benefits}</p>
                  </div>
                </label>
              )
            })}
          </div>

          <div className="mb-6">
            <label className="mb-2 block text-sm font-medium">支払い方法</label>
            <select
              className="input-field"
              value={paymentMethod}
              onChange={(e) => setPaymentMethod(e.target.value as PaymentMethod)}
            >
              <option value="CARD">クレジットカード</option>
              <option value="KONBINI">コンビニ払い</option>
            </select>
          </div>

          <button
            type="button"
            disabled={submitting || !selectedPlanId || !isValidUUID(selectedPlanId)}
            onClick={() => void startCheckout()}
            className="btn-primary w-full rounded-full font-bold"
          >
            {submitting ? '準備中...' : paymentMethod === 'KONBINI' ? 'コンビニ決済へ進む' : 'カード決済に進む'}
          </button>
        </>
      ) : null}

      {showPaymentForm && clientSecret && paymentIntentId && selectedPlan ? (
        <CheckoutPaymentForm
          clientSecret={clientSecret}
          paymentIntentId={paymentIntentId}
          amountLabel={`${formatCurrencyYen(selectedPlan.price)}/月`}
          onSuccess={onPaymentSuccess}
          onError={(msg) => setError(msg)}
        />
      ) : null}

      <p className="mt-4 text-center text-sm">
        <Link to={`/projects/${project.id}`} className="text-purple-700">プロジェクトに戻る</Link>
      </p>
    </div>
  )
}

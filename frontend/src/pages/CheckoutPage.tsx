import { useEffect, useState } from 'react'
import { Link, useParams, useSearchParams } from 'react-router-dom'

import { getProject } from '@/api/projects'
import {
  checkoutCancelPath,
  checkoutSuccessPath,
  createCheckout,
} from '@/api/payments'
import { useAuth } from '@/auth/AuthProvider'
import { Loading } from '@/components/common/Loading'
import { ErrorMessage } from '@/components/common/ErrorMessage'
import { ApiError } from '@/lib/api-client'
import { formatCurrencyYen } from '@/lib/utils'
import { isValidUUID, parseUUID } from '@/lib/validation'
import type { PaymentMethod, ProjectDetail, ProjectPlan } from '@/types/api'

export function CheckoutPage() {
  const { projectId } = useParams<{ projectId: string }>()
  const [searchParams] = useSearchParams()
  const { user } = useAuth()
  const [project, setProject] = useState<ProjectDetail | null>(null)
  const [selectedPlanId, setSelectedPlanId] = useState<string | null>(null)
  const [paymentMethod, setPaymentMethod] = useState<PaymentMethod>('CARD')
  const [loading, setLoading] = useState(true)
  const [submitting, setSubmitting] = useState(false)
  const [error, setError] = useState('')

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
          const first = detail.plans[0]
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

  const startCheckout = async () => {
    if (!project || !selectedPlanId || !user) {
      return
    }
    setSubmitting(true)
    setError('')
    try {
      const origin = window.location.origin
      const success = `${origin}${checkoutSuccessPath(project.id)}`
      const cancel = `${origin}${checkoutCancelPath(project.id)}`
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
    } catch (err) {
      if (err instanceof ApiError) {
        setError(err.message)
      } else {
        setError('チェックアウトの開始に失敗しました')
      }
    } finally {
      setSubmitting(false)
    }
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

  return (
    <div className="mx-auto max-w-2xl">
      <h1 className="mb-2 text-2xl font-bold">応援する</h1>
      <p className="mb-6 text-gray-600">{project.title}</p>
      {error ? <p className="mb-4 text-sm text-red-600">{error}</p> : null}

      <div className="mb-6 space-y-3">
        {project.plans.filter((p) => p.is_active).map((plan: ProjectPlan) => (
          <label key={plan.id} className={`card flex cursor-pointer gap-3 ${selectedPlanId === plan.id ? 'ring-2 ring-purple-500' : ''}`}>
            <input
              type="radio"
              name="plan"
              checked={selectedPlanId === plan.id}
              onChange={() => setSelectedPlanId(plan.id)}
            />
            <div>
              <p className="font-bold">{plan.membership_type}</p>
              <p>{formatCurrencyYen(plan.price)}</p>
              <p className="text-sm text-gray-600">{plan.benefits}</p>
            </div>
          </label>
        ))}
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
        {submitting ? 'Stripe へ移動中...' : 'Stripe で支払う'}
      </button>
      <p className="mt-4 text-center text-sm">
        <Link to={`/projects/${project.id}`} className="text-purple-700">プロジェクトに戻る</Link>
      </p>
    </div>
  )
}

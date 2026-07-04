import { useEffect, useState } from 'react'
import { Link } from 'react-router-dom'

import {
  exportSupportersCsv,
  getConnectAccount,
  getDashboardStats,
  getProjectAnalysis,
  listSellerSupporters,
  startConnectOnboarding,
} from '@/api/seller'
import { Loading } from '@/components/common/Loading'
import { ErrorMessage } from '@/components/common/ErrorMessage'
import { formatCurrencyYen } from '@/lib/utils'
import { planBarClass, planTierLabel, planTierTextClass } from '@/lib/plans'
import type { ConnectAccount, ProjectAnalysisItem, SellerDashboardStats, SellerSupporterRow } from '@/types/api'
import { ApiError } from '@/lib/api-client'

function normalizeConnectURL(url: string): string {
  try {
    const parsed = new URL(url, window.location.origin)
    if (parsed.pathname.startsWith('/seller/connect/')) {
      return `${window.location.origin}${parsed.pathname}${parsed.search}`
    }
    return parsed.toString()
  } catch {
    return url
  }
}

function isLocalConnectReturnURL(url: string): boolean {
  try {
    const parsed = new URL(url, window.location.origin)
    return parsed.pathname === '/seller/connect/return' || parsed.pathname === '/seller/connect/refresh'
  } catch {
    return false
  }
}

export function SellerDashboardPage() {
  const [stats, setStats] = useState<SellerDashboardStats | null>(null)
  const [connect, setConnect] = useState<ConnectAccount | null>(null)
  const [analysis, setAnalysis] = useState<ProjectAnalysisItem[]>([])
  const [supporterTotal, setSupporterTotal] = useState(0)
  const [supporterRows, setSupporterRows] = useState<SellerSupporterRow[]>([])
  const [supporterVisible, setSupporterVisible] = useState(10)
  const [loading, setLoading] = useState(true)
  const [errorMessage, setErrorMessage] = useState('')
  const [connectMsg, setConnectMsg] = useState('')

  const load = async () => {
    setLoading(true)
    setErrorMessage('')
    try {
      const [s, c, a, sup] = await Promise.all([
        getDashboardStats(),
        getConnectAccount(),
        getProjectAnalysis(),
        listSellerSupporters(undefined, 200, 0),
      ])
      setStats(s)
      setConnect(c.account)
      setAnalysis(a.projects)
      setSupporterTotal(sup.total)
      setSupporterRows(sup.supporters)
    } catch {
      setErrorMessage('ダッシュボードの取得に失敗しました')
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    void load()
  }, [])

  const onConnect = async () => {
    setConnectMsg('')
    try {
      const { url } = await startConnectOnboarding()
      const target = normalizeConnectURL(url)
      if (!target.startsWith('http://') && !target.startsWith('https://')) {
        setConnectMsg('Connect URL が無効です')
        return
      }
      if (isLocalConnectReturnURL(target)) {
        await load()
        setConnectMsg('Stripe Connect の設定を更新しました')
        return
      }
      window.location.assign(target)
    } catch (err) {
      const msg = err instanceof ApiError
        ? err.code === 'FORBIDDEN'
          ? '出品者アカウントが有効化されていないか、Connect を利用できません'
          : err.code === 'STRIPE_UNAVAILABLE'
            ? '決済サービスが利用できません'
            : err.message
        : 'Connect onboarding の開始に失敗しました'
      setConnectMsg(msg)
    }
  }

  const onExport = async () => {
    const blob = await exportSupportersCsv()
    const a = document.createElement('a')
    a.href = URL.createObjectURL(blob)
    a.download = 'supporters.csv'
    a.click()
    URL.revokeObjectURL(a.href)
  }

  if (loading) {
    return <Loading />
  }
  if (errorMessage || !stats) {
    return <ErrorMessage message={errorMessage || 'データがありません'} onRetry={() => void load()} />
  }

  return (
    <div className="space-y-8">
      <div className="flex flex-wrap items-center justify-between gap-4">
        <h1 className="text-2xl font-bold">出品者ダッシュボード</h1>
        <Link to="/seller/projects" className="btn-secondary rounded-full text-sm">プロジェクト管理</Link>
      </div>

      <section className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
        <div className="card"><p className="text-sm text-gray-600">プロジェクト</p><p className="text-2xl font-bold">{stats.total_projects}</p></div>
        <div className="card"><p className="text-sm text-gray-600">公開中</p><p className="text-2xl font-bold">{stats.active_projects}</p></div>
        <div className="card"><p className="text-sm text-gray-600">支援者</p><p className="text-2xl font-bold">{stats.total_supporters}</p></div>
        <div className="card"><p className="text-sm text-gray-600">売上</p><p className="text-2xl font-bold">{formatCurrencyYen(stats.total_revenue)}</p></div>
      </section>

      <section className="card">
        <h2 className="mb-3 text-lg font-semibold">Stripe Connect</h2>
        {connect?.stripe_account_id ? (
          <ul className="text-sm text-gray-700">
            <li>charges: {connect.charges_enabled ? '有効' : '未設定'}</li>
            <li>payouts: {connect.payouts_enabled ? '有効' : '未設定'}</li>
            <li>details: {connect.details_submitted ? '提出済' : '未提出'}</li>
          </ul>
        ) : (
          <p className="text-sm text-gray-600">Connect アカウント未連携</p>
        )}
        <button type="button" onClick={() => void onConnect()} className="btn-primary mt-4 rounded-full text-sm">
          {connect?.stripe_account_id ? '設定を更新' : 'Connect を開始'}
        </button>
        {connectMsg ? (
          <p className={`mt-2 text-sm ${connectMsg.includes('更新しました') ? 'text-green-700' : 'text-red-600'}`}>
            {connectMsg}
          </p>
        ) : null}
      </section>

      <section className="card">
        <div className="mb-3 flex items-center justify-between">
          <h2 className="text-lg font-semibold">支援者 ({supporterTotal})</h2>
          <button type="button" onClick={() => void onExport()} className="btn-secondary rounded-full text-sm">CSV 出力</button>
        </div>
        {supporterRows.length === 0 ? (
          <p className="text-sm text-gray-500">支援者はまだいません</p>
        ) : (
          <>
            <ul className="space-y-2 text-sm">
              {supporterRows.slice(0, supporterVisible).map((row) => (
                <li key={row.id} className="flex justify-between border-b border-purple-50 py-2">
                  <span>{row.project?.title ?? row.project_id}</span>
                  <span className="text-gray-600">{planTierLabel(row.plan?.membership_type ?? '')} · {row.status}</span>
                </li>
              ))}
            </ul>
            {supporterRows.length > supporterVisible ? (
              <button
                type="button"
                onClick={() => setSupporterVisible((n) => n + 20)}
                className="mt-3 text-sm font-medium text-purple-700 hover:underline"
              >
                さらに表示（残り {supporterRows.length - supporterVisible} 件）
              </button>
            ) : null}
          </>
        )}
      </section>

      <section className="card">
        <h2 className="mb-3 text-lg font-semibold">プロジェクト分析</h2>
        <ul className="space-y-4 text-sm">
          {analysis.map((p) => (
            <li key={p.project_id} className="border-b border-purple-50 pb-3">
              <div className="mb-2 flex justify-between font-medium">
                <Link to={`/projects/${p.project_id}`} className="text-purple-800 hover:underline">{p.title}</Link>
                <span>{p.supporter_count} 人 · {formatCurrencyYen(p.revenue)}</span>
              </div>
              {p.plan_breakdown ? (
                <div className="space-y-1 pl-2">
                  {(['BRONZE', 'SILVER', 'GOLD'] as const).map((tier) => (
                    <div key={tier} className="flex items-center gap-2 text-xs">
                      <span className={`w-14 ${planTierTextClass(tier)}`}>{planTierLabel(tier)}</span>
                      <div className="h-1.5 flex-1 rounded-full bg-gray-100">
                        <div
                          className={`h-full rounded-full ${planBarClass(tier)}`}
                          style={{
                            width: `${Math.min(100, ((p.plan_breakdown?.[tier] ?? 0) / Math.max(p.supporter_count, 1)) * 100)}%`,
                          }}
                        />
                      </div>
                      <span className="w-6 text-right">{p.plan_breakdown?.[tier] ?? 0}</span>
                    </div>
                  ))}
                </div>
              ) : null}
            </li>
          ))}
        </ul>
      </section>
    </div>
  )
}

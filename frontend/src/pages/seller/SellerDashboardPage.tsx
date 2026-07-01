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
import type { ConnectAccount, ProjectAnalysisItem, SellerDashboardStats } from '@/types/api'

export function SellerDashboardPage() {
  const [stats, setStats] = useState<SellerDashboardStats | null>(null)
  const [connect, setConnect] = useState<ConnectAccount | null>(null)
  const [analysis, setAnalysis] = useState<ProjectAnalysisItem[]>([])
  const [supporterTotal, setSupporterTotal] = useState(0)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')
  const [connectMsg, setConnectMsg] = useState('')

  const load = async () => {
    setLoading(true)
    setError('')
    try {
      const [s, c, a, sup] = await Promise.all([
        getDashboardStats(),
        getConnectAccount(),
        getProjectAnalysis(),
        listSellerSupporters(undefined, 5, 0),
      ])
      setStats(s)
      setConnect(c.account)
      setAnalysis(a.projects)
      setSupporterTotal(sup.total)
    } catch {
      setError('ダッシュボードの取得に失敗しました')
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
      if (!url.startsWith('https://')) {
        setConnectMsg('Connect URL が無効です')
        return
      }
      window.location.assign(url)
    } catch {
      setConnectMsg('Connect  onboarding の開始に失敗しました')
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
  if (error || !stats) {
    return <ErrorMessage message={error || 'データがありません'} onRetry={() => void load()} />
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
        {connect ? (
          <ul className="text-sm text-gray-700">
            <li>charges: {connect.charges_enabled ? '有効' : '未設定'}</li>
            <li>payouts: {connect.payouts_enabled ? '有効' : '未設定'}</li>
            <li>details: {connect.details_submitted ? '提出済' : '未提出'}</li>
          </ul>
        ) : (
          <p className="text-sm text-gray-600">Connect アカウント未連携</p>
        )}
        <button type="button" onClick={() => void onConnect()} className="btn-primary mt-4 rounded-full text-sm">
          {connect ? '設定を更新' : 'Connect を開始'}
        </button>
        {connectMsg ? <p className="mt-2 text-sm text-red-600">{connectMsg}</p> : null}
      </section>

      <section className="card">
        <div className="mb-3 flex items-center justify-between">
          <h2 className="text-lg font-semibold">支援者 ({supporterTotal})</h2>
          <button type="button" onClick={() => void onExport()} className="btn-secondary rounded-full text-sm">CSV 出力</button>
        </div>
      </section>

      <section className="card">
        <h2 className="mb-3 text-lg font-semibold">プロジェクト分析</h2>
        <ul className="space-y-2 text-sm">
          {analysis.map((p) => (
            <li key={p.project_id} className="flex justify-between border-b border-purple-50 py-2">
              <span>{p.title}</span>
              <span>{p.supporter_count} 人 · {formatCurrencyYen(p.revenue)}</span>
            </li>
          ))}
        </ul>
      </section>
    </div>
  )
}

import { useEffect, useMemo, useState } from 'react'
import { useSearchParams } from 'react-router-dom'

import {
  approveBuyer,
  approveSellerRegistration,
  deleteAdminProject,
  deleteAdminUser,
  fetchAdminStats,
  fetchSecurityEvents,
  getSellerRegistration,
  listActivityLogs,
  listAdminProjects,
  listAdminPurchases,
  listAdminReports,
  listAdminUsers,
  listAuditLogs,
  listBatchRuns,
  listBuyerPending,
  listSellerRegistrations,
  patchAdminDeleteRequest,
  patchAdminProject,
  patchAdminPurchase,
  patchAdminReport,
  rejectBuyer,
  rejectSellerRegistration,
  resetUser2FA,
  runBatchJob,
  updateAdminUser,
} from '@/api/admin'
import { useAuth } from '@/auth/AuthProvider'
import { Loading } from '@/components/common/Loading'
import { ErrorMessage } from '@/components/common/ErrorMessage'
import { useToast } from '@/components/common/Toast'
import {
  ADMIN_TAB_LABELS,
  availableAdminTabs,
  canAccessAdminTab,
  type AdminTab,
} from '@/lib/admin-access'
import { formatCurrencyYen, sanitizeDisplayText } from '@/lib/utils'
import { ApiError } from '@/lib/api-client'

export function AdminPage() {
  const { adminRole } = useAuth()
  const { toast } = useToast()
  const [searchParams, setSearchParams] = useSearchParams()
  const tabs = useMemo(() => availableAdminTabs(adminRole), [adminRole])
  const tabParam = searchParams.get('tab') as AdminTab | null
  const activeTab: AdminTab = tabParam && tabs.includes(tabParam) ? tabParam : (tabs[0] ?? 'reviews-buyers')

  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')

  useEffect(() => {
    if (!canAccessAdminTab(adminRole, activeTab)) {
      setError('この管理機能へのアクセス権がありません')
      setLoading(false)
      return
    }
    setLoading(false)
  }, [adminRole, activeTab])

  const setTab = (tab: AdminTab) => {
    setSearchParams({ tab })
    setError('')
  }

  if (!adminRole) {
    return <ErrorMessage message="管理者ロール情報を取得できません。再ログインしてください。" />
  }

  if (loading) {
    return <Loading />
  }

  return (
    <div>
      <h1 className="mb-6 text-2xl font-bold">管理画面</h1>
      <nav className="mb-6 flex flex-wrap gap-2">
        {tabs.map((t) => (
          <button
            key={t}
            type="button"
            onClick={() => setTab(t)}
            className={`rounded-full px-4 py-2 text-sm ${activeTab === t ? 'bg-purple-700 text-white' : 'bg-purple-100 text-purple-800'}`}
          >
            {ADMIN_TAB_LABELS[t]}
          </button>
        ))}
      </nav>
      {error ? <p className="mb-4 text-red-600">{error}</p> : null}
      <AdminTabPanel tab={activeTab} onToast={toast} onError={setError} />
    </div>
  )
}

function AdminTabPanel({
  tab,
  onToast,
  onError,
}: {
  tab: AdminTab
  onToast: (message: string, kind?: 'success' | 'error' | 'info') => void
  onError: (e: string) => void
}) {
  switch (tab) {
    case 'stats':
      return <AdminStatsPanel onError={onError} />
    case 'users':
      return <AdminUsersPanel onToast={onToast} onError={onError} />
    case 'reviews-buyers':
      return <AdminBuyerReviewsPanel onToast={onToast} onError={onError} />
    case 'reviews-sellers':
      return <AdminSellerReviewsPanel onToast={onToast} onError={onError} />
    case 'projects':
      return <AdminProjectsPanel onToast={onToast} onError={onError} />
    case 'purchases':
      return <AdminPurchasesPanel onToast={onToast} onError={onError} />
    case 'reports':
      return <AdminReportsPanel onToast={onToast} onError={onError} />
    case 'activity':
      return <AdminActivityPanel onError={onError} />
    case 'security':
      return <AdminSecurityPanel onError={onError} />
    case 'batches':
      return <AdminBatchesPanel onToast={onToast} onError={onError} />
    case 'audit':
      return <AdminAuditPanel onError={onError} />
    default:
      return null
  }
}

function AdminStatsPanel({ onError }: { onError: (e: string) => void }) {
  const [stats, setStats] = useState<Awaited<ReturnType<typeof fetchAdminStats>> | null>(null)
  useEffect(() => {
    void fetchAdminStats().then(setStats).catch(() => onError('統計の取得に失敗しました'))
  }, [onError])
  if (!stats) {
    return <Loading />
  }
  return (
    <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
      <div className="card"><p>ユーザー</p><p className="text-2xl font-bold">{stats.total_users}</p></div>
      <div className="card"><p>プロジェクト</p><p className="text-2xl font-bold">{stats.total_projects}</p></div>
      <div className="card"><p>購入</p><p className="text-2xl font-bold">{stats.total_purchases}</p></div>
      <div className="card"><p>売上</p><p className="text-2xl font-bold">{formatCurrencyYen(stats.total_revenue)}</p></div>
    </div>
  )
}

function AdminUsersPanel({ onToast, onError }: { onToast: (m: string, k?: 'success' | 'error' | 'info') => void; onError: (e: string) => void }) {
  const [users, setUsers] = useState<Awaited<ReturnType<typeof listAdminUsers>>['users']>([])
  useEffect(() => {
    void listAdminUsers().then((r) => setUsers(r.users)).catch(() => onError('ユーザー一覧の取得に失敗'))
  }, [onError])
  return (
    <ul className="space-y-2">
      {users.map((u) => (
        <li key={u.id} className="card flex flex-wrap items-center justify-between gap-2 text-sm">
          <span>{u.email} · {u.role} · {u.is_active ? 'active' : 'inactive'}</span>
          <div className="flex gap-2">
            <button type="button" className="btn-secondary rounded-full text-xs" onClick={() => void updateAdminUser(u.id, { is_active: !u.is_active }).then(() => onToast('更新しました')).catch(() => onError('更新失敗'))}>
              有効切替
            </button>
            <button type="button" className="btn-secondary rounded-full text-xs" onClick={() => void resetUser2FA(u.id).then(() => onToast('2FA リセット')).catch(() => onError('失敗'))}>2FA リセット</button>
            <button type="button" className="text-xs text-red-600" onClick={() => void deleteAdminUser(u.id).then(() => onToast('削除')).catch(() => onError('削除失敗'))}>削除</button>
          </div>
        </li>
      ))}
    </ul>
  )
}

function AdminBuyerReviewsPanel({ onToast, onError }: { onToast: (m: string, k?: 'success' | 'error' | 'info') => void; onError: (e: string) => void }) {
  const [users, setUsers] = useState<Awaited<ReturnType<typeof listBuyerPending>>['users']>([])
  const reload = () => listBuyerPending().then((r) => setUsers(r.users)).catch(() => onError('取得失敗'))
  useEffect(() => { void reload() }, [onError])
  return (
    <ul className="space-y-2">
      {users.map((u) => (
        <li key={u.id} className="card flex justify-between text-sm">
          <span>{u.email}</span>
          <div className="flex gap-2">
            <button type="button" className="btn-primary rounded-full text-xs" onClick={() => void approveBuyer(u.id).then(() => { onToast('承認'); void reload() }).catch(() => onError('失敗'))}>承認</button>
            <button type="button" className="btn-secondary rounded-full text-xs" onClick={() => {
              const reason = window.prompt('却下理由')
              if (reason) {
                void rejectBuyer(u.id, reason).then(() => { onToast('却下'); void reload() }).catch(() => onError('失敗'))
              }
            }}>却下</button>
          </div>
        </li>
      ))}
    </ul>
  )
}

function AdminSellerReviewsPanel({ onToast, onError }: { onToast: (m: string, k?: 'success' | 'error' | 'info') => void; onError: (e: string) => void }) {
  const [rows, setRows] = useState<Awaited<ReturnType<typeof listSellerRegistrations>>['registrations']>([])
  const reload = () => listSellerRegistrations().then((r) => setRows(r.registrations)).catch(() => onError('取得失敗'))
  useEffect(() => { void reload() }, [onError])
  return (
    <ul className="space-y-2">
      {rows.map((r) => (
        <li key={r.id} className="card text-sm">
          <p className="font-medium">{r.full_name} · {r.status}</p>
          <p className="text-gray-600">{r.user?.email}</p>
          <div className="mt-2 flex gap-2">
            <button type="button" className="btn-secondary rounded-full text-xs" onClick={() => void getSellerRegistration(r.id).then((d) => window.open(d.identity_document_front_url, '_blank', 'noopener,noreferrer'))}>KYC 表面</button>
            <button type="button" className="btn-primary rounded-full text-xs" onClick={() => void approveSellerRegistration(r.id).then(() => { onToast('承認'); void reload() }).catch(() => onError('失敗'))}>承認</button>
            <button type="button" className="btn-secondary rounded-full text-xs" onClick={() => {
              const reason = window.prompt('却下理由')
              if (reason) {
                void rejectSellerRegistration(r.id, reason).then(() => { onToast('却下'); void reload() }).catch(() => onError('失敗'))
              }
            }}>却下</button>
          </div>
        </li>
      ))}
    </ul>
  )
}

function AdminProjectsPanel({ onToast, onError }: { onToast: (m: string, k?: 'success' | 'error' | 'info') => void; onError: (e: string) => void }) {
  const [projects, setProjects] = useState<Awaited<ReturnType<typeof listAdminProjects>>['projects']>([])
  const reload = () => listAdminProjects().then((r) => setProjects(r.projects)).catch(() => onError('取得失敗'))
  useEffect(() => { void reload() }, [onError])
  return (
    <ul className="space-y-3">
      {projects.map((p) => (
        <li key={p.id} className="card text-sm">
          <p className="mb-2 font-medium">{p.title} · {p.status}{p.delete_request ? ' · 削除申請中' : ''}</p>
          <div className="flex flex-wrap gap-2">
            {p.status === 'PENDING' ? (
              <>
                <button type="button" className="btn-primary rounded-full text-xs" onClick={() => void patchAdminProject(p.id, 'ACTIVE').then(() => { onToast('プロジェクトを承認しました', 'success'); void reload() }).catch(() => onError('失敗'))}>承認</button>
                <button type="button" className="btn-secondary rounded-full text-xs" onClick={() => {
                  const reason = window.prompt('却下理由を入力してください')
                  if (reason) {
                    void patchAdminProject(p.id, 'CANCELLED', reason).then(() => { onToast('プロジェクトを却下しました', 'success'); void reload() }).catch(() => onError('却下に失敗しました'))
                  }
                }}>却下</button>
              </>
            ) : null}
            {p.delete_request ? (
              <div className="flex flex-col gap-1 sm:flex-row">
                <button type="button" className="rounded-full bg-red-600 px-3 py-1 text-xs text-white" onClick={() => void patchAdminDeleteRequest(p.id, 'approve').then(() => { onToast('削除申請を承認しました', 'success'); void reload() }).catch(() => onError('失敗'))}>削除申請を承認する</button>
                <button type="button" className="btn-secondary rounded-full text-xs" onClick={() => {
                  const reason = window.prompt('削除申請却下の理由')
                  if (reason) {
                    void patchAdminDeleteRequest(p.id, 'reject', reason).then(() => { onToast('削除申請を却下しました', 'success'); void reload() }).catch(() => onError('失敗'))
                  }
                }}>削除申請を却下する</button>
              </div>
            ) : null}
            {(p.status === 'CANCELLED' || p.status === 'PENDING') ? (
              <button type="button" className="text-xs text-red-600 underline" onClick={() => {
                if (window.confirm('このプロジェクトを強制削除しますか？')) {
                  void deleteAdminProject(p.id).then(() => { onToast('強制削除しました', 'success'); void reload() }).catch(() => onError('削除失敗'))
                }
              }}>強制削除する</button>
            ) : null}
          </div>
        </li>
      ))}
    </ul>
  )
}

function AdminPurchasesPanel({ onToast, onError }: { onToast: (m: string, k?: 'success' | 'error' | 'info') => void; onError: (e: string) => void }) {
  const [purchases, setPurchases] = useState<Awaited<ReturnType<typeof listAdminPurchases>>['purchases']>([])
  useEffect(() => {
    void listAdminPurchases().then((r) => setPurchases(r.purchases)).catch(() => onError('取得失敗'))
  }, [onError])
  return (
    <ul className="space-y-2 text-sm">
      {purchases.map((p) => (
        <li key={p.id} className="card flex justify-between">
          <span>{p.project?.title ?? p.id} · {formatCurrencyYen(p.total_price)} · {p.status}</span>
          <button type="button" className="btn-secondary rounded-full text-xs" onClick={() => void patchAdminPurchase(p.id, 'REFUNDED').then(() => onToast('更新')).catch(() => onError('失敗'))}>返金</button>
        </li>
      ))}
    </ul>
  )
}

function AdminReportsPanel({ onToast, onError }: { onToast: (m: string, k?: 'success' | 'error' | 'info') => void; onError: (e: string) => void }) {
  const [reports, setReports] = useState<Awaited<ReturnType<typeof listAdminReports>>['reports']>([])
  const reload = () => listAdminReports().then((r) => setReports(r.reports)).catch(() => onError('取得失敗'))
  useEffect(() => { void reload() }, [onError])
  return (
    <ul className="space-y-2 text-sm">
      {reports.map((r) => (
        <li key={r.id} className="card">
          <p>{r.target_type} · {r.reason} · {r.status}</p>
          <div className="mt-2 flex gap-2">
            <button type="button" className="btn-primary rounded-full text-xs" onClick={() => void patchAdminReport(r.id, 'RESOLVED').then(() => { onToast('解決'); void reload() }).catch(() => onError('失敗'))}>解決</button>
            <button type="button" className="btn-secondary rounded-full text-xs" onClick={() => void patchAdminReport(r.id, 'DISMISSED').then(() => { onToast('却下'); void reload() }).catch(() => onError('失敗'))}>却下</button>
          </div>
        </li>
      ))}
    </ul>
  )
}

function AdminActivityPanel({ onError }: { onError: (e: string) => void }) {
  const [logs, setLogs] = useState<Awaited<ReturnType<typeof listActivityLogs>>['logs']>([])
  useEffect(() => {
    void listActivityLogs().then((r) => setLogs(r.logs)).catch(() => onError('取得失敗'))
  }, [onError])
  return (
    <ul className="max-h-96 space-y-1 overflow-y-auto font-mono text-xs">
      {logs.map((l) => (
        <li key={l.id}>{l.created_at} {l.action} {l.resource}/{l.resource_id}</li>
      ))}
    </ul>
  )
}

function AdminSecurityPanel({ onError }: { onError: (e: string) => void }) {
  const [data, setData] = useState<Awaited<ReturnType<typeof fetchSecurityEvents>> | null>(null)
  useEffect(() => {
    void fetchSecurityEvents().then(setData).catch(() => onError('取得失敗'))
  }, [onError])
  if (!data) {
    return <Loading />
  }
  return (
    <div className="card text-sm">
      <p>ログイン失敗: {data.login_failed_count}</p>
      <p>Rate limit: {data.rate_limited_count}</p>
      <ul>{data.top_ips.map((ip) => <li key={ip.ip}>{ip.ip}: {ip.count}</li>)}</ul>
    </div>
  )
}

function AdminBatchesPanel({ onToast, onError }: { onToast: (m: string, k?: 'success' | 'error' | 'info') => void; onError: (e: string) => void }) {
  const [runs, setRuns] = useState<Awaited<ReturnType<typeof listBatchRuns>>['runs']>([])
  const reload = () => listBatchRuns().then((r) => setRuns(r.runs)).catch(() => onError('取得失敗'))
  useEffect(() => { void reload() }, [onError])
  const jobs = ['apply_plans', 'expire_supporters', 'purge_tokens'] as const
  return (
    <div>
      <div className="mb-4 flex flex-wrap gap-2">
        {jobs.map((j) => (
          <button key={j} type="button" className="btn-primary rounded-full text-xs" onClick={() => void runBatchJob(j, true).then(() => { onToast(`${j} dry-run 実行`); void reload() }).catch((e: unknown) => onError(e instanceof ApiError ? e.message : '失敗'))}>
            {j} (dry-run)
          </button>
        ))}
      </div>
      <ul className="space-y-1 text-sm">
        {runs.map((r) => (
          <li key={r.id}>{r.job_name} · {r.status} · processed={r.processed}</li>
        ))}
      </ul>
    </div>
  )
}

function AdminAuditPanel({ onError }: { onError: (e: string) => void }) {
  const [logs, setLogs] = useState<Awaited<ReturnType<typeof listAuditLogs>>['logs']>([])
  useEffect(() => {
    void listAuditLogs().then((r) => setLogs(r.logs)).catch(() => onError('取得失敗'))
  }, [onError])
  return (
    <ul className="max-h-96 space-y-1 overflow-y-auto text-xs">
      {logs.map((l) => (
        <li key={l.id}>{l.created_at} {l.admin_email} {l.action} {sanitizeDisplayText(l.resource, 64)}</li>
      ))}
    </ul>
  )
}

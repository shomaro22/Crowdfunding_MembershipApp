import { useEffect, useState } from 'react'
import { Link } from 'react-router-dom'

import { listMyProjects, requestProjectDelete } from '@/api/projects'
import { Loading } from '@/components/common/Loading'
import { ErrorMessage } from '@/components/common/ErrorMessage'
import type { ProjectSummary } from '@/types/api'
import { ApiError } from '@/lib/api-client'

export function SellerProjectsPage() {
  const [projects, setProjects] = useState<ProjectSummary[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')
  const [msg, setMsg] = useState('')

  const load = async () => {
    setLoading(true)
    setError('')
    try {
      const res = await listMyProjects()
      setProjects(res.projects)
    } catch {
      setError('プロジェクト一覧の取得に失敗しました')
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    void load()
  }, [])

  const onDeleteRequest = async (id: string) => {
    if (!window.confirm('削除申請を送信しますか？')) {
      return
    }
    try {
      await requestProjectDelete(id)
      setMsg('削除申請を送信しました')
      await load()
    } catch (err) {
      setMsg(err instanceof ApiError ? err.message : '申請に失敗しました')
    }
  }

  if (loading) {
    return <Loading />
  }
  if (error) {
    return <ErrorMessage message={error} onRetry={() => void load()} />
  }

  return (
    <div>
      <div className="mb-6 flex items-center justify-between">
        <h1 className="text-2xl font-bold">プロジェクト管理</h1>
        <Link to="/seller/projects/new" className="btn-primary rounded-full text-sm">新規作成</Link>
      </div>
      {msg ? <p className="mb-4 text-sm text-purple-800">{msg}</p> : null}
      <ul className="space-y-4">
        {projects.map((p) => (
          <li key={p.id} className="card flex flex-wrap items-center justify-between gap-3">
            <div>
              <Link to={`/seller/projects/${p.id}/edit`} className="font-bold text-purple-800">{p.title}</Link>
              <p className="text-sm text-gray-600">{p.status} · 支援 {p.supporter_count}</p>
            </div>
            <div className="flex gap-2">
              <Link to={`/projects/${p.id}`} className="btn-secondary rounded-full text-sm">公開ページ</Link>
              {p.status === 'ACTIVE' ? (
                <button type="button" onClick={() => void onDeleteRequest(p.id)} className="rounded-full border border-red-200 px-4 py-2 text-sm text-red-600">削除申請</button>
              ) : null}
            </div>
          </li>
        ))}
      </ul>
    </div>
  )
}

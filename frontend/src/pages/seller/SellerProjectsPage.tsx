import { useEffect, useState } from 'react'
import { Link } from 'react-router-dom'

import { listMyProjects, requestProjectDelete } from '@/api/projects'
import { ConfirmDialog } from '@/components/common/ConfirmDialog'
import { Loading } from '@/components/common/Loading'
import { ErrorMessage } from '@/components/common/ErrorMessage'
import { useToast } from '@/components/common/Toast'
import type { ProjectSummary } from '@/types/api'
import { ApiError } from '@/lib/api-client'

export function SellerProjectsPage() {
  const { toast } = useToast()
  const [projects, setProjects] = useState<ProjectSummary[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')
  const [deleteTarget, setDeleteTarget] = useState<string | null>(null)

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
    try {
      await requestProjectDelete(id)
      toast('削除申請を送信しました', 'success')
      await load()
    } catch (err) {
      toast(err instanceof ApiError ? err.message : '申請に失敗しました', 'error')
    } finally {
      setDeleteTarget(null)
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
      <ConfirmDialog
        open={deleteTarget !== null}
        title="削除申請"
        message="このプロジェクトの削除申請を管理者に送信します。よろしいですか？"
        confirmLabel="申請する"
        danger
        onConfirm={() => {
          if (deleteTarget) {
            void onDeleteRequest(deleteTarget)
          }
        }}
        onCancel={() => setDeleteTarget(null)}
      />

      <div className="mb-6 flex items-center justify-between">
        <h1 className="text-2xl font-bold">プロジェクト管理</h1>
        <Link to="/seller/projects/new" className="btn-primary rounded-full text-sm">新規作成</Link>
      </div>
      <ul className="space-y-4">
        {projects.map((p) => (
          <li key={p.id} className="card flex flex-wrap items-center justify-between gap-3">
            <div>
              <Link to={`/seller/projects/${p.id}/edit`} className="font-bold text-purple-800">{p.title}</Link>
              <p className="text-sm text-gray-600">
                {p.status === 'PENDING' ? '審査中' : p.status === 'ACTIVE' ? '公開中' : p.status === 'CANCELLED' ? '却下済' : p.status === 'DELETED' ? '削除済' : p.status}
                {' · '}
                支援 {p.supporter_count}
                {p.delete_request ? ' · 削除申請中' : ''}
              </p>
            </div>
            <div className="flex gap-2">
              <Link to={`/seller/projects/${p.id}/edit`} className="btn-primary rounded-full text-sm">
                商品ページを編集
              </Link>
              {(p.status === 'ACTIVE' || p.status === 'PENDING' || p.status === 'CANCELLED' || p.status === 'DELETED') ? (
                <Link to={`/projects/${p.id}`} className="btn-secondary rounded-full text-sm">公開ページ</Link>
              ) : null}
              {p.status === 'ACTIVE' ? (
                <button type="button" onClick={() => setDeleteTarget(p.id)} className="rounded-full border border-red-200 px-4 py-2 text-sm text-red-600">削除申請</button>
              ) : null}
            </div>
          </li>
        ))}
      </ul>
    </div>
  )
}

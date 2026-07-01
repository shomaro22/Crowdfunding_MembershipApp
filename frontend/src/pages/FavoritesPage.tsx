import { useEffect, useState } from 'react'
import { Link } from 'react-router-dom'

import { listFavorites, toggleFavorite } from '@/api/supporters'
import { ProjectCard } from '@/components/project/ProjectCard'
import { Loading } from '@/components/common/Loading'
import { ErrorMessage } from '@/components/common/ErrorMessage'
import type { ProjectSummary } from '@/types/api'

export function FavoritesPage() {
  const [items, setItems] = useState<ProjectSummary[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')

  const load = async () => {
    setLoading(true)
    setError('')
    try {
      const res = await listFavorites()
      setItems(res.favorites)
    } catch {
      setError('お気に入りの取得に失敗しました')
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    void load()
  }, [])

  const remove = async (projectId: string) => {
    await toggleFavorite(projectId, 'remove')
    setItems((prev) => prev.filter((p) => p.id !== projectId))
  }

  if (loading) {
    return <Loading />
  }
  if (error) {
    return <ErrorMessage message={error} onRetry={() => void load()} />
  }

  return (
    <div>
      <h1 className="mb-6 text-2xl font-bold">お気に入り</h1>
      {items.length === 0 ? (
        <p className="text-gray-600">お気に入りはまだありません。<Link to="/" className="text-purple-700">プロジェクトを探す</Link></p>
      ) : (
        <div className="grid gap-6 md:grid-cols-2 lg:grid-cols-3">
          {items.map((p) => (
            <div key={p.id} className="relative">
              <ProjectCard project={p} />
              <button
                type="button"
                onClick={() => void remove(p.id)}
                className="absolute right-3 top-3 rounded-full bg-white/90 px-3 py-1 text-xs text-red-600 shadow"
              >
                解除
              </button>
            </div>
          ))}
        </div>
      )}
    </div>
  )
}

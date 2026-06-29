import { useCallback, useEffect, useState } from 'react'

import { listPublicProjects } from '@/api/projects'
import { ErrorMessage } from '@/components/common/ErrorMessage'
import { Loading } from '@/components/common/Loading'
import { ProjectCard } from '@/components/project/ProjectCard'
import { sanitizeDisplayText } from '@/lib/utils'
import type { ProjectSummary } from '@/types/api'

export function HomePage() {
  const [search, setSearch] = useState('')
  const [query, setQuery] = useState('')
  const [projects, setProjects] = useState<ProjectSummary[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')

  const load = useCallback(async (term: string) => {
    setLoading(true)
    setError('')
    try {
      const res = await listPublicProjects({ search: term || undefined, limit: 24 })
      setProjects(res.projects)
    } catch {
      setError('プロジェクト一覧の取得に失敗しました')
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    void load(query)
  }, [load, query])

  const onSearch = (event: React.FormEvent<HTMLFormElement>) => {
    event.preventDefault()
    setQuery(sanitizeDisplayText(search, 100))
  }

  return (
    <div>
      <section className="mb-10 text-center">
        <h1 className="text-gradient mb-4 text-4xl font-extrabold">ファンと創作者をつなぐ</h1>
        <p className="mx-auto max-w-2xl text-gray-600">
          メンバーシップ型クラウドファンディングで、お気に入りのプロジェクトを応援しましょう。
        </p>
      </section>

      <form onSubmit={onSearch} className="mb-8 flex gap-3">
        <input
          type="search"
          value={search}
          onChange={(e) => setSearch(e.target.value)}
          placeholder="プロジェクトを検索..."
          className="input-field max-w-xl flex-1"
          maxLength={100}
        />
        <button type="submit" className="btn-primary rounded-full px-6">検索</button>
      </form>

      {loading ? <Loading label="プロジェクトを読み込み中..." /> : null}
      {error ? <ErrorMessage message={error} onRetry={() => void load(query)} /> : null}

      {!loading && !error ? (
        projects.length === 0 ? (
          <p className="text-center text-gray-500">公開中のプロジェクトがありません</p>
        ) : (
          <div className="grid grid-cols-1 gap-6 md:grid-cols-2 lg:grid-cols-3">
            {projects.map((project) => (
              <ProjectCard key={project.id} project={project} />
            ))}
          </div>
        )
      ) : null}
    </div>
  )
}

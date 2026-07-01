import { useCallback, useEffect, useState } from 'react'

import { listPublicProjects } from '@/api/projects'
import { ErrorMessage } from '@/components/common/ErrorMessage'
import { Loading } from '@/components/common/Loading'
import { Pagination } from '@/components/common/Pagination'
import { ProjectCard } from '@/components/project/ProjectCard'
import { sanitizeDisplayText } from '@/lib/utils'
import type { ProjectSummary } from '@/types/api'

const PAGE_SIZE = 6

export function HomePage() {
  const [search, setSearch] = useState('')
  const [query, setQuery] = useState('')
  const [page, setPage] = useState(1)
  const [projects, setProjects] = useState<ProjectSummary[]>([])
  const [total, setTotal] = useState(0)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')

  const totalPages = Math.max(1, Math.ceil(total / PAGE_SIZE))

  const load = useCallback(async (term: string, pageNum: number) => {
    setLoading(true)
    setError('')
    try {
      const offset = (pageNum - 1) * PAGE_SIZE
      const res = await listPublicProjects({
        search: term || undefined,
        limit: PAGE_SIZE,
        offset,
      })
      setProjects(res.projects)
      setTotal(res.total)
    } catch {
      setError('プロジェクト一覧の取得に失敗しました')
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    void load(query, page)
  }, [load, query, page])

  const onSearch = (event: React.FormEvent<HTMLFormElement>) => {
    event.preventDefault()
    setPage(1)
    setQuery(sanitizeDisplayText(search, 100))
  }

  return (
    <div>
      <section className="mb-10 text-center">
        <h1 className="text-gradient mb-4 text-4xl font-extrabold md:text-5xl">
          推しのクリエイターを、もっと近くに
        </h1>
        <p className="mx-auto max-w-2xl text-gray-600">
          月額メンバーシップで限定コンテンツ・制作裏話・コミュニティ特典を楽しもう。
          ファンとクリエイターをつなぐ、新しい応援の形です。
        </p>
      </section>

      <form onSubmit={onSearch} className="mb-8 flex justify-center">
        <div className="flex w-full max-w-xl gap-3">
          <input
            type="search"
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            placeholder="クリエイター・プロジェクトを検索..."
            className="input-field flex-1"
            maxLength={100}
          />
          <button type="submit" className="btn-primary shrink-0 rounded-full px-6">
            検索
          </button>
        </div>
      </form>

      {loading ? <Loading label="プロジェクトを読み込み中..." /> : null}
      {error ? <ErrorMessage message={error} onRetry={() => void load(query, page)} /> : null}

      {!loading && !error ? (
        projects.length === 0 ? (
          <p className="text-center text-gray-500">公開中のプロジェクトがありません</p>
        ) : (
          <>
            <p className="mb-4 text-center text-sm text-gray-500">
              {total}件中 {(page - 1) * PAGE_SIZE + 1}〜{Math.min(page * PAGE_SIZE, total)}件を表示
            </p>
            <div className="grid grid-cols-1 gap-6 md:grid-cols-2 lg:grid-cols-3">
              {projects.map((project) => (
                <ProjectCard key={project.id} project={project} />
              ))}
            </div>
            <Pagination page={page} totalPages={totalPages} onPageChange={setPage} />
          </>
        )
      ) : null}
    </div>
  )
}

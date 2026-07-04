import { useCallback, useEffect, useState } from 'react'

import { listPublicProjects } from '@/api/projects'
import { listFavorites } from '@/api/supporters'
import { useAuth } from '@/auth/AuthProvider'
import { ErrorMessage } from '@/components/common/ErrorMessage'
import { Loading } from '@/components/common/Loading'
import { HeroCarousel } from '@/components/layout/HeroCarousel'
import { ProjectCard } from '@/components/project/ProjectCard'
import { useInfiniteScroll } from '@/hooks/useInfiniteScroll'
import { sanitizeDisplayText } from '@/lib/utils'
import type { ProjectSummary } from '@/types/api'

const PAGE_SIZE = 10

export function HomePage() {
  const { status, hasRole } = useAuth()
  const [search, setSearch] = useState('')
  const [query, setQuery] = useState('')
  const [projects, setProjects] = useState<ProjectSummary[]>([])
  const [favoriteIds, setFavoriteIds] = useState<Set<string>>(new Set())
  const [total, setTotal] = useState(0)
  const [loading, setLoading] = useState(true)
  const [loadingMore, setLoadingMore] = useState(false)
  const [errorMessage, setErrorMessage] = useState('')

  useEffect(() => {
    if (status !== 'authenticated' || !hasRole('BUYER')) {
      setFavoriteIds(new Set())
      return
    }
    void listFavorites()
      .then((res) => setFavoriteIds(new Set(res.favorites.map((f) => f.id))))
      .catch(() => setFavoriteIds(new Set()))
  }, [hasRole, status])

  const hasMore = projects.length < total

  const loadPage = useCallback(async (term: string, offset: number, append: boolean) => {
    if (append) {
      setLoadingMore(true)
    } else {
      setLoading(true)
    }
    setErrorMessage('')
    try {
      const res = await listPublicProjects({
        search: term || undefined,
        limit: PAGE_SIZE,
        offset,
      })
      setTotal(res.total)
      setProjects((prev) => (append ? [...prev, ...res.projects] : res.projects))
    } catch {
      setErrorMessage('プロジェクト一覧の取得に失敗しました')
    } finally {
      setLoading(false)
      setLoadingMore(false)
    }
  }, [])

  useEffect(() => {
    void loadPage(query, 0, false)
  }, [loadPage, query])

  const loadMore = useCallback(() => {
    if (loadingMore || loading || !hasMore) {
      return
    }
    void loadPage(query, projects.length, true)
  }, [hasMore, loadPage, loading, loadingMore, projects.length, query])

  const sentinelRef = useInfiniteScroll(loadMore, hasMore, loading || loadingMore)

  const onSearch = (event: React.FormEvent<HTMLFormElement>) => {
    event.preventDefault()
    setQuery(sanitizeDisplayText(search, 100))
  }

  return (
    <div>
      <HeroCarousel />

      <section className="mb-10 text-center">
        <h1 className="text-gradient mb-4 text-4xl font-extrabold md:text-5xl">
          推しのクリエイターを、もっと近くに
        </h1>
        <p className="mx-auto max-w-2xl text-gray-600">
          月額メンバーシップで限定コンテンツ・制作裏話・コミュニティ特典を楽しもう。
          <br />
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

      {loading && projects.length === 0 ? <Loading label="プロジェクトを読み込み中..." /> : null}
      {errorMessage ? <ErrorMessage message={errorMessage} onRetry={() => void loadPage(query, 0, false)} /> : null}

      {!errorMessage && projects.length > 0 ? (
        <>
          <div className="grid grid-cols-1 gap-6 md:grid-cols-2 lg:grid-cols-3">
            {projects.map((project) => (
              <ProjectCard
                key={project.id}
                project={project}
                isFavorited={favoriteIds.has(project.id)}
              />
            ))}
          </div>
          {loadingMore ? <Loading label="さらに読み込み中..." /> : null}
          {hasMore ? <div ref={sentinelRef} className="h-8" aria-hidden /> : null}
        </>
      ) : null}

      {!loading && !errorMessage && projects.length === 0 ? (
        <p className="text-center text-gray-500">公開中のプロジェクトがありません</p>
      ) : null}
    </div>
  )
}

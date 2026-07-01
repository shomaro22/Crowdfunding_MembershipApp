import { useEffect, useState } from 'react'
import { Link, useParams } from 'react-router-dom'

import { listPublicProjects } from '@/api/projects'
import { ErrorMessage } from '@/components/common/ErrorMessage'
import { Loading } from '@/components/common/Loading'
import { ProjectCard } from '@/components/project/ProjectCard'
import { parseUUID } from '@/lib/validation'
import type { ProjectSummary } from '@/types/api'

export function CreatorProjectsPage() {
  const { sellerId } = useParams<{ sellerId: string }>()
  const [projects, setProjects] = useState<ProjectSummary[]>([])
  const [creatorName, setCreatorName] = useState('')
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')

  useEffect(() => {
    const id = parseUUID(sellerId)
    if (!id) {
      setError('クリエイター ID が不正です')
      setLoading(false)
      return
    }

    void (async () => {
      try {
        const res = await listPublicProjects({ limit: 100, offset: 0 })
        const filtered = res.projects.filter((p) => p.seller?.id === id)
        setProjects(filtered)
        const name = filtered[0]?.seller?.name ?? filtered[0]?.seller?.email ?? 'クリエイター'
        setCreatorName(name)
      } catch {
        setError('出品一覧の取得に失敗しました')
      } finally {
        setLoading(false)
      }
    })()
  }, [sellerId])

  if (loading) {
    return <Loading />
  }
  if (error) {
    return <ErrorMessage message={error} />
  }

  return (
    <div>
      <p className="mb-2 text-sm text-gray-500">
        <Link to="/" className="text-purple-700 hover:underline">トップ</Link>
        {' / '}
        クリエイター
      </p>
      <h1 className="mb-6 text-2xl font-bold">{creatorName} の出品</h1>
      {projects.length === 0 ? (
        <p className="text-gray-600">公開中のプロジェクトはありません</p>
      ) : (
        <div className="grid grid-cols-1 gap-6 md:grid-cols-2 lg:grid-cols-3">
          {projects.map((project) => (
            <ProjectCard key={project.id} project={project} />
          ))}
        </div>
      )}
    </div>
  )
}

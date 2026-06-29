import { useEffect, useState } from 'react'
import { Link, useParams } from 'react-router-dom'

import { getProject } from '@/api/projects'
import { ErrorMessage } from '@/components/common/ErrorMessage'
import { Loading } from '@/components/common/Loading'
import { formatCurrencyYen } from '@/lib/utils'
import type { ProjectDetail } from '@/types/api'

export function ProjectDetailPage() {
  const { projectId } = useParams<{ projectId: string }>()
  const [project, setProject] = useState<ProjectDetail | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')

  useEffect(() => {
    if (!projectId) {
      setError('プロジェクト ID が不正です')
      setLoading(false)
      return
    }

    void (async () => {
      try {
        const detail = await getProject(projectId)
        setProject(detail)
      } catch {
        setError('プロジェクトの取得に失敗しました')
      } finally {
        setLoading(false)
      }
    })()
  }, [projectId])

  if (loading) {
    return <Loading />
  }

  if (error || !project) {
    return <ErrorMessage message={error || 'プロジェクトが見つかりません'} />
  }

  const progress = project.target_amount > 0
    ? Math.min(100, Math.round((project.current_amount / project.target_amount) * 100))
    : 0

  return (
    <div className="mx-auto max-w-4xl">
      <div className="card overflow-hidden p-0">
        {project.image_url ? (
          <img src={project.image_url} alt="" className="h-72 w-full object-cover" referrerPolicy="no-referrer" />
        ) : (
          <div className="flex h-72 items-center justify-center bg-purple-100 text-purple-400">No image</div>
        )}
        <div className="p-6">
          <h1 className="mb-4 text-3xl font-bold text-gray-900">{project.title}</h1>
          <p className="mb-6 whitespace-pre-wrap text-gray-700">{project.description}</p>

          <div className="mb-6">
            <div className="mb-2 flex justify-between text-sm text-gray-600">
              <span>達成率 {progress}%</span>
              <span>{formatCurrencyYen(project.current_amount)} / {formatCurrencyYen(project.target_amount)}</span>
            </div>
            <div className="h-3 overflow-hidden rounded-full bg-purple-100">
              <div className="h-full bg-purple-600" style={{ width: `${progress}%` }} />
            </div>
          </div>

          <h2 className="mb-3 text-xl font-semibold text-gray-900">メンバーシッププラン</h2>
          <div className="grid gap-4 md:grid-cols-3">
            {project.plans.map((plan) => (
              <div key={plan.id} className="rounded-xl border border-purple-100 p-4">
                <p className="font-bold text-purple-800">{plan.membership_type}</p>
                <p className="my-2 text-2xl font-extrabold">{formatCurrencyYen(plan.price)}</p>
                <p className="text-sm text-gray-600">{plan.benefits}</p>
              </div>
            ))}
          </div>

          <div className="mt-8">
            <Link to={`/checkout/${project.id}`} className="btn-primary inline-flex rounded-full font-bold">
              応援する
            </Link>
          </div>
        </div>
      </div>
    </div>
  )
}

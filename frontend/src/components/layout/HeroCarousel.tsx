import { useEffect, useState } from 'react'
import { Link } from 'react-router-dom'

import { listFeaturedProjects } from '@/api/projects'
import { formatCurrencyYen, fundingProgressPercent } from '@/lib/utils'
import type { ProjectSummary } from '@/types/api'

function SideSlide({ project, onClick }: { project: ProjectSummary; onClick: () => void }) {
  const progress = fundingProgressPercent(project.current_amount, project.target_amount)
  const over = progress > 100

  return (
    <button
      type="button"
      onClick={onClick}
      className="group relative h-48 w-full overflow-hidden rounded-xl opacity-80 transition hover:opacity-100 md:h-56"
    >
      {project.image_url ? (
        <img
          src={project.image_url}
          alt=""
          className="h-full w-full object-cover transition group-hover:scale-105"
          referrerPolicy="no-referrer"
        />
      ) : (
        <div className="flex h-full items-center justify-center bg-purple-900/50" />
      )}
      <div className="absolute inset-0 bg-gradient-to-t from-black/70 to-transparent p-3">
        <p className="line-clamp-2 text-left text-xs font-bold text-white">{project.title}</p>
        <p className={`mt-1 text-left text-xs font-bold ${over ? 'text-amber-300' : 'text-purple-200'}`}>
          {progress}%
        </p>
      </div>
    </button>
  )
}

function MainSlide({ project }: { project: ProjectSummary }) {
  const progress = fundingProgressPercent(project.current_amount, project.target_amount)
  const over = progress > 100

  return (
    <div className="relative h-56 overflow-hidden rounded-2xl md:h-72">
      {project.image_url ? (
        <img
          src={project.image_url}
          alt=""
          className="h-full w-full object-cover"
          referrerPolicy="no-referrer"
        />
      ) : (
        <div className="flex h-full items-center justify-center bg-purple-900/50" />
      )}
      <div className="absolute inset-0 bg-gradient-to-t from-black/80 via-black/30 to-transparent p-6">
        <p className="mb-1 text-xs font-semibold uppercase tracking-wider text-purple-200">注目のプロジェクト</p>
        <h2 className="mb-2 line-clamp-2 text-xl font-bold text-white md:text-2xl">{project.title}</h2>
        <p className="mb-3 line-clamp-2 text-sm text-purple-100">{project.description}</p>
        <div className="flex flex-wrap items-center gap-3 text-sm">
          <span className={`text-2xl font-extrabold ${over ? 'text-amber-300' : 'text-white'}`}>
            {progress}%
          </span>
          {over ? (
            <span className="rounded-full bg-amber-500/90 px-2 py-0.5 text-xs font-bold text-white">目標達成</span>
          ) : null}
          <span className="text-purple-200">
            支援者 {project.supporter_count}人 · {formatCurrencyYen(project.current_amount)}
          </span>
        </div>
        <Link
          to={`/projects/${project.id}`}
          className="mt-4 inline-flex rounded-full bg-white px-5 py-2 text-sm font-bold text-purple-800 hover:bg-purple-50"
        >
          詳細を見る
        </Link>
      </div>
    </div>
  )
}

export function HeroCarousel() {
  const [slides, setSlides] = useState<ProjectSummary[]>([])
  const [index, setIndex] = useState(0)

  useEffect(() => {
    void listFeaturedProjects(10).then((res) => {
      if (res.projects.length > 0) {
        setSlides(res.projects)
      }
    }).catch(() => {
      /* ignore */
    })
  }, [])

  useEffect(() => {
    if (slides.length <= 1) {
      return
    }
    const timer = window.setInterval(() => {
      setIndex((i) => (i + 1) % slides.length)
    }, 6000)
    return () => window.clearInterval(timer)
  }, [slides.length])

  if (slides.length === 0) {
    return null
  }

  const len = slides.length
  const current = slides[index]
  const prev = slides[(index - 1 + len) % len]
  const next = slides[(index + 1) % len]

  if (!current) {
    return null
  }

  return (
    <section className="mb-8">
      {len >= 3 ? (
        <div className="grid grid-cols-1 items-center gap-3 md:grid-cols-10">
          <div className="hidden md:col-span-2 md:block">
            {prev ? <SideSlide project={prev} onClick={() => setIndex((index - 1 + len) % len)} /> : null}
          </div>
          <div className="md:col-span-6">
            <MainSlide project={current} />
          </div>
          <div className="hidden md:col-span-2 md:block">
            {next ? <SideSlide project={next} onClick={() => setIndex((index + 1) % len)} /> : null}
          </div>
        </div>
      ) : (
        <MainSlide project={current} />
      )}

      {len > 1 ? (
        <div className="mt-4 flex justify-center gap-2">
          {slides.map((s, i) => (
            <button
              key={s.id}
              type="button"
              aria-label={`スライド ${i + 1}`}
              onClick={() => setIndex(i)}
              className={`h-2 w-2 rounded-full transition ${i === index ? 'w-6 bg-purple-700' : 'bg-purple-300'}`}
            />
          ))}
        </div>
      ) : null}
    </section>
  )
}

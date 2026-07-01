import { useEffect, useState } from 'react'
import { Link } from 'react-router-dom'

import { listFeaturedProjects } from '@/api/projects'
import { formatCurrencyYen, fundingProgressPercent } from '@/lib/utils'
import type { ProjectSummary } from '@/types/api'

export function HeroCarousel() {
  const [slides, setSlides] = useState<ProjectSummary[]>([])
  const [index, setIndex] = useState(0)

  useEffect(() => {
    void listFeaturedProjects(5).then((res) => {
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
    }, 5000)
    return () => window.clearInterval(timer)
  }, [slides.length])

  if (slides.length === 0) {
    return null
  }

  const current = slides[index]
  if (!current) {
    return null
  }

  const progress = fundingProgressPercent(current.current_amount, current.target_amount)

  return (
    <section className="relative mb-8 overflow-hidden rounded-2xl bg-gradient-to-r from-purple-800 via-purple-700 to-pink-600 text-white shadow-lg">
      <div className="flex flex-col gap-4 p-6 md:flex-row md:items-center md:justify-between md:p-8">
        <div className="max-w-2xl">
          <p className="mb-1 text-xs font-semibold uppercase tracking-wider text-purple-200">注目のプロジェクト</p>
          <h2 className="mb-2 text-2xl font-bold md:text-3xl">{current.title}</h2>
          <p className="mb-4 line-clamp-2 text-sm text-purple-100">{current.description}</p>
          <p className="text-sm text-purple-100">
            達成率 {progress}% · 支援者 {current.supporter_count}人 · {formatCurrencyYen(current.current_amount)}
          </p>
          <Link
            to={`/projects/${current.id}`}
            className="mt-4 inline-flex rounded-full bg-white px-5 py-2 text-sm font-bold text-purple-800 hover:bg-purple-50"
          >
            詳細を見る
          </Link>
        </div>
        {current.image_url ? (
          <img
            src={current.image_url}
            alt=""
            className="h-40 w-full rounded-xl object-cover md:h-44 md:w-64"
            referrerPolicy="no-referrer"
          />
        ) : null}
      </div>
      {slides.length > 1 ? (
        <div className="flex justify-center gap-2 pb-4">
          {slides.map((s, i) => (
            <button
              key={s.id}
              type="button"
              aria-label={`スライド ${i + 1}`}
              onClick={() => setIndex(i)}
              className={`h-2 w-2 rounded-full ${i === index ? 'bg-white' : 'bg-white/40'}`}
            />
          ))}
        </div>
      ) : null}
    </section>
  )
}

interface PaginationProps {
  page: number
  totalPages: number
  onPageChange: (page: number) => void
}

export function Pagination({ page, totalPages, onPageChange }: PaginationProps) {
  if (totalPages <= 1) {
    return null
  }

  const pages = Array.from({ length: totalPages }, (_, i) => i + 1).slice(
    Math.max(0, page - 3),
    Math.min(totalPages, page + 2),
  )

  return (
    <nav className="mt-10 flex items-center justify-center gap-2" aria-label="ページネーション">
      <button
        type="button"
        disabled={page <= 1}
        onClick={() => onPageChange(page - 1)}
        className="btn-secondary rounded-full px-4 py-2 text-sm disabled:opacity-40"
      >
        前へ
      </button>
      {pages.map((p) => (
        <button
          key={p}
          type="button"
          onClick={() => onPageChange(p)}
          className={`min-w-10 rounded-full px-3 py-2 text-sm font-bold ${p === page ? 'bg-purple-700 text-white' : 'bg-purple-100 text-purple-800'}`}
        >
          {p}
        </button>
      ))}
      <button
        type="button"
        disabled={page >= totalPages}
        onClick={() => onPageChange(page + 1)}
        className="btn-secondary rounded-full px-4 py-2 text-sm disabled:opacity-40"
      >
        次へ
      </button>
    </nav>
  )
}

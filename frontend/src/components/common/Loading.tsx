interface LoadingProps {
  label?: string
}

export function Loading({ label = '読み込み中...' }: LoadingProps) {
  return (
    <div className="flex min-h-[40vh] flex-col items-center justify-center gap-4">
      <div
        className="h-10 w-10 animate-spin rounded-full border-4 border-purple-200 border-t-purple-700"
        role="status"
        aria-label={label}
      />
      <p className="text-sm text-gray-600">{label}</p>
    </div>
  )
}

interface ErrorMessageProps {
  message: string
  onRetry?: () => void
}

export function ErrorMessage({ message, onRetry }: ErrorMessageProps) {
  return (
    <div className="rounded-2xl border border-red-200 bg-red-50 p-4 text-red-700">
      <div className="flex items-start gap-3">
        <span className="mt-2 inline-block h-2 w-2 shrink-0 rounded-full bg-red-500" aria-hidden />
        <div className="flex-1">
          <p className="whitespace-pre-wrap text-sm">{message}</p>
          {onRetry ? (
            <button type="button" onClick={onRetry} className="btn-primary mt-4 rounded-full text-sm">
              再試行
            </button>
          ) : null}
        </div>
      </div>
    </div>
  )
}

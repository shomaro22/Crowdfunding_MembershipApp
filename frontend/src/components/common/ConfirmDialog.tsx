interface ConfirmDialogProps {
  open: boolean
  title: string
  message: string
  confirmLabel?: string
  cancelLabel?: string
  danger?: boolean
  onConfirm: () => void
  onCancel: () => void
}

export function ConfirmDialog({
  open,
  title,
  message,
  confirmLabel = '確認',
  cancelLabel = 'キャンセル',
  danger = false,
  onConfirm,
  onCancel,
}: ConfirmDialogProps) {
  if (!open) {
    return null
  }

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/40 p-4" role="dialog" aria-modal="true">
      <div className="w-full max-w-md rounded-2xl bg-white p-6 shadow-xl">
        <h2 className="mb-2 text-lg font-bold text-gray-900">{title}</h2>
        <p className="mb-6 text-sm text-gray-600">{message}</p>
        <div className="flex justify-end gap-3">
          <button type="button" onClick={onCancel} className="btn-secondary rounded-full px-4 py-2 text-sm">
            {cancelLabel}
          </button>
          <button
            type="button"
            onClick={onConfirm}
            className={`rounded-full px-4 py-2 text-sm font-bold text-white ${danger ? 'bg-red-600 hover:bg-red-700' : 'btn-primary'}`}
          >
            {confirmLabel}
          </button>
        </div>
      </div>
    </div>
  )
}

interface ReasonDialogProps {
  open: boolean
  title: string
  placeholder?: string
  confirmLabel?: string
  onConfirm: (reason: string) => void
  onCancel: () => void
}

export function ReasonDialog({
  open,
  title,
  placeholder = '理由を入力してください',
  confirmLabel = '送信',
  onConfirm,
  onCancel,
}: ReasonDialogProps) {
  if (!open) {
    return null
  }

  return (
    <form
      className="fixed inset-0 z-50 flex items-center justify-center bg-black/40 p-4"
      role="dialog"
      aria-modal="true"
      onSubmit={(e) => {
        e.preventDefault()
        const fd = new FormData(e.currentTarget)
        const raw = fd.get('reason')
        const reason = typeof raw === 'string' ? raw.trim() : ''
        if (reason) {
          onConfirm(reason)
        }
      }}
    >
      <div className="w-full max-w-md rounded-2xl bg-white p-6 shadow-xl">
        <h2 className="mb-4 text-lg font-bold text-gray-900">{title}</h2>
        <textarea
          name="reason"
          className="input-field mb-4 min-h-24"
          placeholder={placeholder}
          maxLength={500}
          required
          autoFocus
        />
        <div className="flex justify-end gap-3">
          <button type="button" onClick={onCancel} className="btn-secondary rounded-full px-4 py-2 text-sm">
            キャンセル
          </button>
          <button type="submit" className="btn-primary rounded-full px-4 py-2 text-sm">
            {confirmLabel}
          </button>
        </div>
      </div>
    </form>
  )
}

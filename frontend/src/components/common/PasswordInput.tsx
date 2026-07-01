import { useState } from 'react'

interface PasswordInputProps {
  id: string
  value: string
  onChange: (value: string) => void
  autoComplete?: string
  placeholder?: string
  className?: string
}

export function PasswordInput({
  id,
  value,
  onChange,
  autoComplete = 'current-password',
  placeholder = 'パスワード',
  className = 'input-field pr-12',
}: PasswordInputProps) {
  const [visible, setVisible] = useState(false)

  return (
    <div className="relative">
      <input
        id={id}
        type={visible ? 'text' : 'password'}
        autoComplete={autoComplete}
        className={className}
        value={value}
        onChange={(e) => onChange(e.target.value)}
        placeholder={placeholder}
      />
      <button
        type="button"
        onClick={() => setVisible((v) => !v)}
        className="absolute right-3 top-1/2 -translate-y-1/2 text-xs text-gray-500 hover:text-gray-800"
        aria-label={visible ? 'パスワードを隠す' : 'パスワードを表示'}
      >
        {visible ? '非表示' : '表示'}
      </button>
    </div>
  )
}

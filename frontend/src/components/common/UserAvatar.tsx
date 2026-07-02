import { cn } from '@/lib/utils'
import { normalizeMediaUrl } from '@/lib/media-url'

interface UserAvatarProps {
  name?: string | null
  email?: string | null
  avatarUrl?: string | null
  size?: 'sm' | 'md' | 'lg'
}

const sizeClass: Record<NonNullable<UserAvatarProps['size']>, string> = {
  sm: 'h-8 w-8 text-xs',
  md: 'h-10 w-10 text-sm',
  lg: 'h-12 w-12 text-base',
}

function initials(name: string | null | undefined, email: string | null | undefined): string {
  const source = (name ?? email ?? '?').trim()
  const parts = source.split(/\s+/).filter(Boolean)
  if (parts.length >= 2) {
    return `${parts[0]?.[0] ?? ''}${parts[1]?.[0] ?? ''}`.toUpperCase()
  }
  return source.slice(0, 2).toUpperCase()
}

export function UserAvatar({ name, email, avatarUrl, size = 'md' }: UserAvatarProps) {
  const src = avatarUrl ? normalizeMediaUrl(avatarUrl) : ''
  if (src) {
    return (
      <img
        src={src}
        alt=""
        className={cn('rounded-full object-cover', sizeClass[size])}
        referrerPolicy="no-referrer"
      />
    )
  }

  return (
    <div
      className={cn(
        'flex items-center justify-center rounded-full bg-gradient-to-br from-purple-500 to-purple-700 font-semibold text-white',
        sizeClass[size],
      )}
      aria-hidden
    >
      {initials(name, email)}
    </div>
  )
}

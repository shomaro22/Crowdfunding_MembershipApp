export const queryKeys = {
  auth: {
    me: ['auth', 'me'] as const,
    profile: ['auth', 'profile'] as const,
  },
  projects: {
    all: ['projects'] as const,
    list: (search: string, sort: string, page: number) => ['projects', 'list', search, sort, page] as const,
    featured: ['projects', 'featured'] as const,
    detail: (id: string) => ['projects', id] as const,
    plans: (id: string) => ['projects', id, 'plans'] as const,
    mine: ['projects', 'mine'] as const,
  },
  favorites: {
    all: ['favorites'] as const,
    check: (projectId: string) => ['favorites', 'check', projectId] as const,
  },
  supporters: {
    mine: ['supporters', 'mine'] as const,
  },
  purchases: {
    mine: ['purchases', 'mine'] as const,
  },
  messages: {
    threads: ['messages', 'threads'] as const,
    notifications: ['messages', 'notifications'] as const,
  },
  seller: {
    dashboard: ['seller', 'dashboard'] as const,
    supporters: (projectId?: string) => ['seller', 'supporters', projectId ?? 'all'] as const,
    projects: ['seller', 'projects'] as const,
  },
  admin: {
    stats: ['admin', 'stats'] as const,
    users: (role: string, search: string) => ['admin', 'users', role, search] as const,
    projects: (status: string) => ['admin', 'projects', status] as const,
  },
} as const

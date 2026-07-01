import { Link, NavLink } from 'react-router-dom'

import { useAuth } from '@/auth/AuthProvider'
import { UserAvatar } from '@/components/common/UserAvatar'

function navLinkClass({ isActive }: { isActive: boolean }): string {
  return isActive
    ? 'text-sm font-bold text-purple-900'
    : 'text-sm font-bold text-purple-700 transition-colors hover:text-purple-900'
}

export function AppHeader() {
  const { status, user, logout } = useAuth()
  const signedIn = status === 'authenticated' && user !== null

  return (
    <header className="border-b border-purple-100 bg-white/80 shadow-sm backdrop-blur-sm">
      <div className="container mx-auto flex items-center justify-between px-4 py-4">
        <Link to="/" className="flex items-center gap-3">
          <span className="flex h-10 w-10 items-center justify-center rounded-full bg-gradient-to-br from-purple-600 to-purple-800 text-sm font-bold text-white">
            FF
          </span>
          <span
            className="text-3xl font-extrabold"
            style={{
              background: 'linear-gradient(45deg, #9333ea, #ec4899, #dc2626)',
              WebkitBackgroundClip: 'text',
              WebkitTextFillColor: 'transparent',
              backgroundClip: 'text',
            }}
          >
            FanFunding
          </span>
        </Link>

        <nav className="flex items-center gap-4">
          {signedIn ? (
            <>
              <div className="hidden items-center gap-2 md:flex">
                <UserAvatar name={user.name} email={user.email} avatarUrl={user.avatar_url} size="sm" />
                <span className="max-w-[12rem] truncate text-sm text-gray-600">
                  {user.name ?? user.email}
                </span>
              </div>

              {user.role === 'BUYER' ? (
                <>
                  <NavLink to="/profile" className={navLinkClass}>プロフィール</NavLink>
                  <NavLink to="/favorites" className={navLinkClass}>お気に入り</NavLink>
                  <NavLink to="/purchases" className={navLinkClass}>応援履歴</NavLink>
                  <NavLink to="/messages" className={navLinkClass}>メッセージ</NavLink>
                </>
              ) : null}

              {user.role === 'SELLER' ? (
                <>
                  <NavLink to="/seller/dashboard" className={navLinkClass}>ダッシュボード</NavLink>
                  <NavLink to="/seller/projects" className={navLinkClass}>プロジェクト</NavLink>
                  <NavLink to="/messages" className={navLinkClass}>メッセージ</NavLink>
                </>
              ) : null}

              {user.role === 'ADMIN' ? (
                <NavLink to="/admin" className={navLinkClass}>管理</NavLink>
              ) : null}

              <button type="button" onClick={() => void logout()} className="btn-primary rounded-full text-sm">
                ログアウト
              </button>
            </>
          ) : (
            <>
              <Link to="/signin" className="btn-primary rounded-full text-sm">ログイン</Link>
              <Link to="/signup" className="btn-secondary rounded-full text-sm">新規登録</Link>
            </>
          )}
        </nav>
      </div>
    </header>
  )
}

export function AppFooter() {
  return (
    <footer className="mt-auto bg-gradient-to-r from-purple-800 to-purple-900 py-8 text-white">
      <div className="container mx-auto px-4 text-center text-sm text-purple-100">
        <p>FanFunding — クラウドファンディング・メンバーシップ</p>
      </div>
    </footer>
  )
}

export function MainLayout({ children }: { children: React.ReactNode }) {
  return (
    <div className="gradient-bg flex min-h-screen flex-col">
      <AppHeader />
      <main className="container mx-auto flex-1 px-4 py-8">{children}</main>
      <AppFooter />
    </div>
  )
}

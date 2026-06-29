import { Link } from 'react-router-dom'

export function NotFoundPage() {
  return (
    <div className="card mx-auto max-w-lg text-center">
      <h1 className="mb-2 text-3xl font-bold text-gray-900">404</h1>
      <p className="mb-6 text-gray-600">ページが見つかりません</p>
      <Link to="/" className="btn-primary inline-flex rounded-full font-bold">トップへ戻る</Link>
    </div>
  )
}

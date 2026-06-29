export function PlaceholderPage({ title, description }: { title: string; description: string }) {
  return (
    <div className="card mx-auto max-w-2xl text-center">
      <h1 className="mb-3 text-2xl font-bold text-gray-900">{title}</h1>
      <p className="text-gray-600">{description}</p>
    </div>
  )
}

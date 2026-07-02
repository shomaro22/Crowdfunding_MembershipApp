import { useCallback, useEffect, useRef, useState, type FormEvent } from 'react'

import {
  fetchWSToken,
  listMessageThreads,
  markMessageRead,
  sendMessage,
} from '@/api/messages'
import { Loading } from '@/components/common/Loading'
import { ErrorMessage } from '@/components/common/ErrorMessage'
import { MessageWebSocket } from '@/lib/ws-client'
import { sanitizeDisplayText } from '@/lib/utils'
import { validateMessageContent } from '@/lib/validation'
import type { Message, MessageThread } from '@/types/api'
import { useAuth } from '@/auth/AuthProvider'

function appendUniqueMessage(messages: Message[], msg: Message): Message[] {
  if (messages.some((m) => m.id === msg.id)) {
    return messages
  }
  return [...messages, msg]
}

export function MessagesPage() {
  const { user } = useAuth()
  const [threads, setThreads] = useState<MessageThread[]>([])
  const [active, setActive] = useState<MessageThread | null>(null)
  const [draft, setDraft] = useState('')
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')
  const [sendError, setSendError] = useState('')
  const wsRef = useRef<MessageWebSocket | null>(null)

  const load = useCallback(async () => {
    setLoading(true)
    setError('')
    try {
      const res = await listMessageThreads()
      setThreads(res.threads)
      if (res.threads.length > 0) {
        setActive((cur) => cur ?? res.threads[0] ?? null)
      }
    } catch {
      setError('メッセージの取得に失敗しました')
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    void load()
  }, [load])

  useEffect(() => {
    let ws: MessageWebSocket | null = null
    let unsub: (() => void) | null = null

    void (async () => {
      try {
        const { ws_token: token } = await fetchWSToken()
        ws = new MessageWebSocket()
        wsRef.current = ws
        unsub = ws.on((ev) => {
          if (ev.type === 'message_created') {
            setActive((cur) => {
              if (!cur || cur.project_id !== ev.project_id) {
                return cur
              }
              return { ...cur, messages: appendUniqueMessage(cur.messages, ev.message) }
            })
            void load()
          }
        })
        ws.connect(token)
      } catch {
        /* HTTP のみでも動作 */
      }
    })()

    return () => {
      unsub?.()
      ws?.disconnect()
      wsRef.current = null
    }
  }, [load])

  useEffect(() => {
    if (active) {
      wsRef.current?.subscribe(active.project_id)
    }
  }, [active])

  useEffect(() => {
    if (!active || !user) {
      return
    }
    for (const m of active.messages) {
      if (!m.is_read && m.receiver_id === user.id) {
        void markMessageRead(m.id)
      }
    }
  }, [active, user])

  const onSend = async (e: FormEvent) => {
    e.preventDefault()
    if (!active || !user) {
      return
    }
    const v = validateMessageContent(draft)
    if (!v.valid) {
      setSendError(v.error ?? '無効なメッセージ')
      return
    }
    setSendError('')
    try {
      const msg = await sendMessage(
        active.project_id,
        active.partner.id,
        sanitizeDisplayText(draft, 4000),
      )
      setDraft('')
      setActive((cur) => {
        if (!cur) {
          return cur
        }
        return { ...cur, messages: appendUniqueMessage(cur.messages, msg) }
      })
    } catch {
      setSendError('送信に失敗しました')
    }
  }

  if (loading) {
    return <Loading />
  }
  if (error) {
    return <ErrorMessage message={error} onRetry={() => void load()} />
  }

  return (
    <div className="grid gap-4 md:grid-cols-3">
      <aside className="card md:col-span-1">
        <h1 className="mb-4 text-lg font-bold">メッセージ</h1>
        {threads.length === 0 ? (
          <p className="text-sm text-gray-500">会話はまだありません。応援中のプロジェクトでクリエイターとやり取りできます。</p>
        ) : (
        <ul className="space-y-2">
          {threads.map((t) => (
            <li key={`${t.project_id}-${t.partner.id}`}>
              <button
                type="button"
                onClick={() => setActive(t)}
                className={`w-full rounded-lg px-3 py-2 text-left text-sm ${active?.project_id === t.project_id && active.partner.id === t.partner.id ? 'bg-purple-100' : 'hover:bg-purple-50'}`}
              >
                <span className="font-medium">{t.project_title}</span>
                <span className="block text-gray-600">{t.partner.name ?? t.partner.email}</span>
                {t.unread_count > 0 ? (
                  <span className="text-xs text-red-600">{t.unread_count} 未読</span>
                ) : null}
              </button>
            </li>
          ))}
        </ul>
        )}
      </aside>

      <section className="card md:col-span-2 flex min-h-[24rem] flex-col">
        {active ? (
          <>
            <h2 className="mb-4 font-semibold">{active.project_title}</h2>
            <div className="flex-1 space-y-3 overflow-y-auto">
              {active.messages.map((m: Message) => (
                <div
                  key={m.id}
                  className={`max-w-[85%] rounded-lg px-3 py-2 text-sm ${m.sender_id === user?.id ? 'ml-auto bg-purple-100' : 'bg-gray-100'}`}
                >
                  {sanitizeDisplayText(m.content, 4000)}
                </div>
              ))}
            </div>
            <form onSubmit={(e) => void onSend(e)} className="mt-4 flex gap-2">
              <textarea
                className="input-field min-h-[3rem] flex-1 resize-y"
                value={draft}
                maxLength={4000}
                onChange={(e) => setDraft(e.target.value)}
                placeholder="メッセージ（4000文字以内）"
              />
              <button type="submit" className="btn-primary rounded-full self-end">送信</button>
            </form>
            {sendError ? <p className="mt-2 text-sm text-red-600">{sendError}</p> : null}
          </>
        ) : (
          <p className="text-gray-600">スレッドを選択してください</p>
        )}
      </section>
    </div>
  )
}

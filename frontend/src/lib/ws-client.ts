import { apiBaseUrl } from '@/lib/api-client'
import { isValidUUID } from '@/lib/validation'
import type { Message } from '@/types/api'
import { isRecord } from '@/types/api'

export type WSClientEvent =
  | { type: 'session_ready'; user_id: string }
  | { type: 'subscribed'; project_id: string }
  | { type: 'unsubscribed'; project_id: string }
  | { type: 'message_created'; project_id: string; message: Message }
  | { type: 'read_updated'; project_id: string; message_id: string; reader_id: string }
  | { type: 'typing'; project_id: string; user_id: string; is_typing: boolean }
  | { type: 'pong' }
  | { type: 'error'; code: string; message: string; project_id?: string }

type WSListener = (event: WSClientEvent) => void

const SCHEMA_VERSION = 1

function wsBaseUrl(): string {
  const http = apiBaseUrl()
  if (!http) {
    const loc = window.location
    const proto = loc.protocol === 'https:' ? 'wss:' : 'ws:'
    return `${proto}//${loc.host}`
  }
  if (http.startsWith('https://')) {
    return http.replace('https://', 'wss://')
  }
  return http.replace('http://', 'ws://')
}

function parseServerFrame(raw: unknown): WSClientEvent | null {
  if (!isRecord(raw) || typeof raw.type !== 'string') {
    return null
  }
  if (raw.schema_version !== SCHEMA_VERSION) {
    return { type: 'error', code: 'SCHEMA_MISMATCH', message: 'unsupported schema' }
  }
  switch (raw.type) {
    case 'session_ready':
      if (typeof raw.user_id === 'string') {
        return { type: 'session_ready', user_id: raw.user_id }
      }
      break
    case 'subscribed':
    case 'unsubscribed':
      if (typeof raw.project_id === 'string' && isValidUUID(raw.project_id)) {
        return { type: raw.type, project_id: raw.project_id }
      }
      break
    case 'message_created':
      if (
        typeof raw.project_id === 'string'
        && isRecord(raw.message)
        && typeof raw.message.id === 'string'
      ) {
        return {
          type: 'message_created',
          project_id: raw.project_id,
          message: raw.message as unknown as Message,
        }
      }
      break
    case 'read_updated':
      if (
        typeof raw.project_id === 'string'
        && typeof raw.message_id === 'string'
        && typeof raw.reader_id === 'string'
      ) {
        return {
          type: 'read_updated',
          project_id: raw.project_id,
          message_id: raw.message_id,
          reader_id: raw.reader_id,
        }
      }
      break
    case 'typing':
      if (
        typeof raw.project_id === 'string'
        && typeof raw.user_id === 'string'
        && typeof raw.is_typing === 'boolean'
      ) {
        return {
          type: 'typing',
          project_id: raw.project_id,
          user_id: raw.user_id,
          is_typing: raw.is_typing,
        }
      }
      break
    case 'pong':
      return { type: 'pong' }
    case 'error':
      if (typeof raw.code === 'string' && typeof raw.message === 'string') {
        return {
          type: 'error',
          code: raw.code,
          message: raw.message,
          project_id: typeof raw.project_id === 'string' ? raw.project_id : undefined,
        }
      }
      break
    default:
      break
  }
  return null
}

export class MessageWebSocket {
  private socket: WebSocket | null = null
  private listeners = new Set<WSListener>()
  private pingTimer: number | null = null
  private reconnectAttempts = 0
  private closedByUser = false

  connect(wsToken: string): void {
    if (!wsToken || wsToken.length > 512) {
      throw new Error('invalid ws token')
    }
    this.closedByUser = false
    const url = `${wsBaseUrl()}/ws?ws_token=${encodeURIComponent(wsToken)}`
    this.socket = new WebSocket(url)

    this.socket.onmessage = (ev) => {
      try {
        const data = JSON.parse(String(ev.data)) as unknown
        const parsed = parseServerFrame(data)
        if (parsed) {
          this.emit(parsed)
        }
      } catch {
        this.emit({ type: 'error', code: 'PARSE_ERROR', message: 'invalid frame' })
      }
    }

    this.socket.onclose = () => {
      this.stopPing()
      if (!this.closedByUser && this.reconnectAttempts < 5) {
        this.reconnectAttempts += 1
      }
    }

    this.socket.onopen = () => {
      this.reconnectAttempts = 0
      this.startPing()
    }
  }

  disconnect(): void {
    this.closedByUser = true
    this.stopPing()
    this.socket?.close()
    this.socket = null
  }

  subscribe(projectId: string): void {
    if (!isValidUUID(projectId)) {
      return
    }
    this.send({ type: 'subscribe', schema_version: SCHEMA_VERSION, project_id: projectId })
  }

  unsubscribe(projectId: string): void {
    if (!isValidUUID(projectId)) {
      return
    }
    this.send({ type: 'unsubscribe', schema_version: SCHEMA_VERSION, project_id: projectId })
  }

  sendTyping(projectId: string, isTyping: boolean): void {
    if (!isValidUUID(projectId)) {
      return
    }
    this.send({
      type: 'typing',
      schema_version: SCHEMA_VERSION,
      project_id: projectId,
      is_typing: isTyping,
    })
  }

  on(listener: WSListener): () => void {
    this.listeners.add(listener)
    return () => this.listeners.delete(listener)
  }

  private emit(event: WSClientEvent): void {
    for (const listener of this.listeners) {
      listener(event)
    }
  }

  private send(payload: Record<string, unknown>): void {
    if (this.socket?.readyState === WebSocket.OPEN) {
      this.socket.send(JSON.stringify(payload))
    }
  }

  private startPing(): void {
    this.stopPing()
    this.pingTimer = window.setInterval(() => {
      this.send({ type: 'ping', schema_version: SCHEMA_VERSION })
    }, 25_000)
  }

  private stopPing(): void {
    if (this.pingTimer !== null) {
      window.clearInterval(this.pingTimer)
      this.pingTimer = null
    }
  }
}

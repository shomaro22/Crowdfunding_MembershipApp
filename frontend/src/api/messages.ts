import { apiRequest } from '@/lib/api-client'
import type { ListMessageThreadsResponse, Message, WSTokenResponse } from '@/types/api'
import { isValidUUID, validateMessageContent } from '@/lib/validation'

export async function listMessageThreads(
  page = 1,
  perPage = 20,
  messagesPerThread = 50,
): Promise<ListMessageThreadsResponse> {
  const p = Math.min(10_000, Math.max(1, page))
  const pp = Math.min(100, Math.max(1, perPage))
  const mpt = Math.min(100, Math.max(1, messagesPerThread))
  return apiRequest<ListMessageThreadsResponse>(
    `/messages?page=${p}&per_page=${pp}&messages_per_thread=${mpt}`,
    { auth: true },
  )
}

export async function sendMessage(
  projectId: string,
  receiverId: string,
  content: string,
): Promise<Message> {
  if (!isValidUUID(projectId) || !isValidUUID(receiverId)) {
    throw new Error('invalid id')
  }
  const v = validateMessageContent(content)
  if (!v.valid) {
    throw new Error(v.error ?? 'invalid message')
  }
  return apiRequest<Message>('/messages', {
    method: 'POST',
    auth: true,
    body: {
      project_id: projectId,
      receiver_id: receiverId,
      content: content.trim(),
    },
  })
}

export async function markMessageRead(messageId: string): Promise<void> {
  if (!isValidUUID(messageId)) {
    throw new Error('invalid message id')
  }
  await apiRequest<void>(`/messages/${messageId}/read`, {
    method: 'POST',
    auth: true,
  })
}

export async function fetchWSToken(): Promise<WSTokenResponse> {
  return apiRequest<WSTokenResponse>('/ws/token', { method: 'POST', auth: true })
}

export async function fetchNotifications(): Promise<{ unread_messages: number }> {
  return apiRequest<{ unread_messages: number }>('/me/notifications', { auth: true })
}

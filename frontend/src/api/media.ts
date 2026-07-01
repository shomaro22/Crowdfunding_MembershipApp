import { apiUpload } from '@/lib/api-client'
import type { MediaUploadResponse } from '@/types/api'
import { validateImageFile } from '@/lib/upload'

export async function uploadImage(file: File): Promise<MediaUploadResponse> {
  const check = validateImageFile(file)
  if (!check.valid) {
    throw new Error(check.error ?? 'invalid file')
  }
  const form = new FormData()
  form.append('file', file)
  return apiUpload<MediaUploadResponse>('/media/images', form, true)
}

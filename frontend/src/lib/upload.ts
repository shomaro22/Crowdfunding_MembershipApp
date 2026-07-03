/** アップロード前のクライアント検証（サーバー側でも再検証される） */

const ALLOWED_IMAGE_TYPES = new Set([
  'image/jpeg',
  'image/png',
  'image/webp',
  'image/gif',
])

const MAX_IMAGE_BYTES = 5 * 1024 * 1024

export interface FileValidationResult {
  valid: boolean
  message?: string
}

export function validateImageFile(file: File): FileValidationResult {
  if (!ALLOWED_IMAGE_TYPES.has(file.type)) {
    return { valid: false, message: 'JPEG / PNG / WebP / GIF のみアップロードできます' }
  }
  if (file.size > MAX_IMAGE_BYTES) {
    return { valid: false, message: '画像サイズは5MB以下にしてください' }
  }
  if (file.size === 0) {
    return { valid: false, message: '空のファイルはアップロードできません' }
  }
  const name = file.name.trim()
  if (name.includes('..') || name.includes('/') || name.includes('\\')) {
    return { valid: false, message: 'ファイル名が不正です' }
  }
  return { valid: true }
}

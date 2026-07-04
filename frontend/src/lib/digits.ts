/** 全角数字（０-９）を半角に変換 */
export function normalizeAsciiDigits(value: string): string {
  return value.replace(/[\uFF10-\uFF19]/g, (ch) =>
    String.fromCharCode(ch.charCodeAt(0) - 0xff10 + 0x30),
  )
}

/** 半角数字のみ抽出（全角数字は先に半角化） */
export function digitsOnly(value: string, maxLength?: number): string {
  let digits = normalizeAsciiDigits(value).replace(/\D/g, '')
  if (maxLength !== undefined) {
    digits = digits.slice(0, maxLength)
  }
  return digits
}

/** 郵便番号入力（7桁・ハイフンなし） */
export function postalCodeInput(value: string): string {
  return digitsOnly(value, 7)
}

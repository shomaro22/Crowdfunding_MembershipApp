package middleware

type contextKey string

const (
	keyRequestID     contextKey = "request_id"
	keyUserID        contextKey = "userID"
	keyUserRole      contextKey = "role"
	keyTwoFAVerified contextKey = "2fa_verified"
	keyAdminRole     contextKey = "admin_role"
)

// Echo コンテキストキー（typed contextKey の文字列表現）。
const (
	CtxRequestID     = string(keyRequestID)
	CtxUserID        = string(keyUserID)
	CtxUserRole      = string(keyUserRole)
	CtxTwoFAVerified = string(keyTwoFAVerified)
	CtxAdminRole     = string(keyAdminRole)
)

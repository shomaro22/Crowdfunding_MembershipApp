package usecase

import (
	"time"

	"crowdfund-app/model"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

const DefaultAuthTTL = 15 * time.Minute

type AuthClaims struct {
	UserID        uuid.UUID
	Role          model.UserRole
	TwoFAVerified bool
	AdminRole     *model.AdminRole
	TokenVersion  int64
}

type TokenService struct {
	secret []byte
	ttl    time.Duration
}

func NewTokenService(secret []byte, ttl time.Duration) *TokenService {
	if ttl <= 0 {
		ttl = DefaultAuthTTL
	}
	return &TokenService{secret: secret, ttl: ttl}
}

func (s *TokenService) Issue(claims AuthClaims) (string, error) {
	var (
		now time.Time
		mc  jwt.MapClaims
		t   *jwt.Token
	)
	if len(s.secret) == 0 {
		return "", ErrInvalidToken
	}
	now = time.Now()
	mc = jwt.MapClaims{
		"sub":          claims.UserID.String(),
		"role":         string(claims.Role),
		"2fa_verified": claims.TwoFAVerified,
		"tv":           claims.TokenVersion,
		"exp":          now.Add(s.ttl).Unix(),
		"iat":          now.Unix(),
	}
	if claims.Role == model.UserRoleAdmin && claims.AdminRole != nil {
		mc["admin_role"] = string(*claims.AdminRole)
	}
	t = jwt.NewWithClaims(jwt.SigningMethodHS256, mc)
	return t.SignedString(s.secret)
}

func (s *TokenService) Parse(raw string) (AuthClaims, error) {
	var (
		out     AuthClaims
		keyFunc jwt.Keyfunc
		tok     *jwt.Token
		mc      jwt.MapClaims
		ok      bool
		sub     string
		uid     uuid.UUID
		roleStr string
		v       bool
		tv      float64
		ar      string
		r       model.AdminRole
		err     error
	)
	if len(s.secret) == 0 || raw == "" {
		return out, ErrInvalidToken
	}
	keyFunc = jwt.Keyfunc(func(t *jwt.Token) (interface{}, error) {
		if t.Method != jwt.SigningMethodHS256 {
			return nil, ErrInvalidToken
		}
		return s.secret, nil
	})
	tok, err = jwt.Parse(raw, keyFunc)
	if err != nil || !tok.Valid {
		return out, ErrInvalidToken
	}
	mc, ok = tok.Claims.(jwt.MapClaims)
	if !ok {
		return out, ErrInvalidToken
	}
	sub, _ = mc["sub"].(string)
	uid, err = uuid.Parse(sub)
	if err != nil || uid == uuid.Nil {
		return out, ErrInvalidToken
	}
	roleStr, _ = mc["role"].(string)
	out.UserID = uid
	out.Role = model.UserRole(roleStr)
	if v, ok = mc["2fa_verified"].(bool); ok {
		out.TwoFAVerified = v
	}
	if tv, ok = mc["tv"].(float64); ok {
		out.TokenVersion = int64(tv)
	}
	if ar, ok = mc["admin_role"].(string); ok && ar != "" {
		r = model.AdminRole(ar)
		out.AdminRole = &r
	}
	return out, nil
}

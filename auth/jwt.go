package auth

import (
	"time"

	"crowdfund-app/entity"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

const DefaultAuthTTL = 15 * time.Minute

type AuthClaims struct {
	UserID        uuid.UUID
	Role          entity.UserRole
	TwoFAVerified bool
	AdminRole     *entity.AdminRole
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
	if len(s.secret) == 0 {
		return "", ErrInvalidToken
	}
	now := time.Now()
	mc := jwt.MapClaims{
		"sub":          claims.UserID.String(),
		"role":         string(claims.Role),
		"2fa_verified": claims.TwoFAVerified,
		"exp":          now.Add(s.ttl).Unix(),
		"iat":          now.Unix(),
	}
	if claims.Role == entity.UserRoleAdmin && claims.AdminRole != nil {
		mc["admin_role"] = string(*claims.AdminRole)
	}
	t := jwt.NewWithClaims(jwt.SigningMethodHS256, mc)
	return t.SignedString(s.secret)
}

func (s *TokenService) Parse(raw string) (AuthClaims, error) {
	var out AuthClaims
	if len(s.secret) == 0 || raw == "" {
		return out, ErrInvalidToken
	}
	tok, err := jwt.Parse(raw, func(t *jwt.Token) (any, error) {
		if t.Method != jwt.SigningMethodHS256 {
			return nil, ErrInvalidToken
		}
		return s.secret, nil
	})
	if err != nil || !tok.Valid {
		return out, ErrInvalidToken
	}
	mc, ok := tok.Claims.(jwt.MapClaims)
	if !ok {
		return out, ErrInvalidToken
	}
	sub, _ := mc["sub"].(string)
	uid, err := uuid.Parse(sub)
	if err != nil || uid == uuid.Nil {
		return out, ErrInvalidToken
	}
	roleStr, _ := mc["role"].(string)
	out.UserID = uid
	out.Role = entity.UserRole(roleStr)
	if v, ok := mc["2fa_verified"].(bool); ok {
		out.TwoFAVerified = v
	}
	if ar, ok := mc["admin_role"].(string); ok && ar != "" {
		r := entity.AdminRole(ar)
		out.AdminRole = &r
	}
	return out, nil
}

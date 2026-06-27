package controller

import (
	"time"

	"crowdfund-app/usecase"
)

type UserWire struct {
	ID        string  `json:"id"`
	Email     string  `json:"email"`
	Name      *string `json:"name"`
	Role      string  `json:"role"`
	IsActive  bool    `json:"is_active"`
	AvatarURL *string `json:"avatar_url"`
	CreatedAt string  `json:"created_at"`
	UpdatedAt string  `json:"updated_at"`
}

func UserWireFrom(v usecase.UserView) UserWire {
	return UserWire{
		ID:        v.ID.String(),
		Email:     v.Email,
		Name:      v.Name,
		Role:      v.Role,
		IsActive:  v.IsActive(),
		AvatarURL: v.AvatarURL,
		CreatedAt: v.CreatedAt.UTC().Format(time.RFC3339),
		UpdatedAt: v.UpdatedAt.UTC().Format(time.RFC3339),
	}
}

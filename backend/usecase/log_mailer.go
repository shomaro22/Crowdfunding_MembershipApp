package usecase

import (
	"context"
	"log/slog"
	"strings"
)

type LogMailer struct {
	AppPublicURL string
}

func NewLogMailer(appPublicURL string) *LogMailer {
	return &LogMailer{AppPublicURL: strings.TrimRight(appPublicURL, "/")}
}

func (m *LogMailer) SendPasswordReset(_ context.Context, email, plainToken string) error {
	resetURL := m.AppPublicURL + "/reset-password?token=" + plainToken
	slog.Info("password_reset_email", "email", email, "reset_url", resetURL)
	return nil
}

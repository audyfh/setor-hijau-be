package auth

import (
	"context"
	"log/slog"
)

// EmailSender abstracts email delivery for notifications and account recovery.
type EmailSender interface {
	SendPasswordReset(ctx context.Context, toEmail, resetToken string) error
}

// LogEmailSender is a development/testing implementation that logs the reset link.
type LogEmailSender struct {
	logger *slog.Logger
}

// NewLogEmailSender creates a new LogEmailSender.
func NewLogEmailSender(logger *slog.Logger) *LogEmailSender {
	return &LogEmailSender{logger: logger}
}

// SendPasswordReset logs the reset token securely for local development and test environments.
func (s *LogEmailSender) SendPasswordReset(ctx context.Context, toEmail, resetToken string) error {
	s.logger.InfoContext(ctx, "Password reset token generated",
		slog.String("email", toEmail),
		slog.String("token", resetToken),
	)
	return nil
}

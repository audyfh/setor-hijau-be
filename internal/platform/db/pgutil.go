package db

import (
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

// ToPGUUID converts a google/uuid.UUID to pgtype.UUID.
func ToPGUUID(u uuid.UUID) pgtype.UUID {
	return pgtype.UUID{Bytes: u, Valid: u != uuid.Nil}
}

// FromPGUUID converts a pgtype.UUID to google/uuid.UUID.
func FromPGUUID(u pgtype.UUID) uuid.UUID {
	if !u.Valid {
		return uuid.Nil
	}
	return uuid.UUID(u.Bytes)
}

// FromPGUUIDPtr converts a pgtype.UUID to *uuid.UUID (nil if !u.Valid or uuid.Nil).
func FromPGUUIDPtr(u pgtype.UUID) *uuid.UUID {
	if !u.Valid {
		return nil
	}
	id := uuid.UUID(u.Bytes)
	if id == uuid.Nil {
		return nil
	}
	return &id
}

// ToPGText converts a string to pgtype.Text (valid if non-empty).
func ToPGText(s string) pgtype.Text {
	return pgtype.Text{String: s, Valid: s != ""}
}

// FromPGText converts a pgtype.Text to string.
func FromPGText(t pgtype.Text) string {
	if !t.Valid {
		return ""
	}
	return t.String
}

// ToPGTimestamptz converts time.Time to pgtype.Timestamptz.
func ToPGTimestamptz(t time.Time) pgtype.Timestamptz {
	return pgtype.Timestamptz{Time: t, Valid: !t.IsZero()}
}

// FromPGTimestamptz converts pgtype.Timestamptz to time.Time.
func FromPGTimestamptz(t pgtype.Timestamptz) time.Time {
	if !t.Valid {
		return time.Time{}
	}
	return t.Time
}

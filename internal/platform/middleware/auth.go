package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/audyfh/setor-hijau-be/internal/platform/auth"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type contextKey string

const (
	tenantIDKey contextKey = "bank_sampah_id"
	userIDKey   contextKey = "user_id"
	userRoleKey contextKey = "user_role"
	usernameKey contextKey = "username"
)

// AuthMiddleware creates a Gin middleware that validates the JWT Bearer token
// and populates both Gin context and standard context with the user and tenant info.
func AuthMiddleware(jwtSecret []byte) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "Autentikasi diperlukan. Header Authorization tidak ditemukan.",
			})
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "Format Authorization tidak valid. Gunakan format 'Bearer <token>'.",
			})
			return
		}

		tokenString := strings.TrimSpace(parts[1])
		claims, err := auth.ParseAccessToken(jwtSecret, tokenString)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "Token tidak valid atau telah kedaluwarsa.",
			})
			return
		}

		// Set in Gin context
		c.Set(string(tenantIDKey), claims.BankSampahID)
		c.Set(string(userIDKey), claims.UserID)
		c.Set(string(userRoleKey), claims.Role)
		c.Set(string(usernameKey), claims.Username)

		// Also set in standard Go context for services and repositories
		ctx := c.Request.Context()
		ctx = context.WithValue(ctx, tenantIDKey, claims.BankSampahID)
		ctx = context.WithValue(ctx, userIDKey, claims.UserID)
		ctx = context.WithValue(ctx, userRoleKey, claims.Role)
		ctx = context.WithValue(ctx, usernameKey, claims.Username)
		c.Request = c.Request.WithContext(ctx)

		c.Next()
	}
}

// GetTenantID retrieves the bank_sampah_id from the context.
func GetTenantID(ctx context.Context) (uuid.UUID, bool) {
	val := ctx.Value(tenantIDKey)
	if id, ok := val.(uuid.UUID); ok && id != uuid.Nil {
		return id, true
	}
	return uuid.Nil, false
}

// GetUserID retrieves the user_id from the context.
func GetUserID(ctx context.Context) (uuid.UUID, bool) {
	val := ctx.Value(userIDKey)
	if id, ok := val.(uuid.UUID); ok && id != uuid.Nil {
		return id, true
	}
	return uuid.Nil, false
}

// GetUserRole retrieves the user role from the context.
func GetUserRole(ctx context.Context) (string, bool) {
	val := ctx.Value(userRoleKey)
	if role, ok := val.(string); ok && role != "" {
		return role, true
	}
	return "", false
}

// GetUsername retrieves the username from the context.
func GetUsername(ctx context.Context) (string, bool) {
	val := ctx.Value(usernameKey)
	if username, ok := val.(string); ok && username != "" {
		return username, true
	}
	return "", false
}

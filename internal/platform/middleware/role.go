package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// RequireRole creates a Gin middleware ensuring the authenticated user has one of the allowed roles.
func RequireRole(allowedRoles ...string) gin.HandlerFunc {
	allowedMap := make(map[string]struct{}, len(allowedRoles))
	for _, r := range allowedRoles {
		allowedMap[r] = struct{}{}
	}

	return func(c *gin.Context) {
		roleVal, exists := c.Get(string(userRoleKey))
		if !exists {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "Sesi tidak terautentikasi.",
			})
			return
		}

		userRole, ok := roleVal.(string)
		if !ok {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"error": "Peran pengguna tidak valid.",
			})
			return
		}

		if _, allowed := allowedMap[userRole]; !allowed {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"error": "Akses ditolak. Anda tidak memiliki izin untuk tindakan ini.",
			})
			return
		}

		c.Next()
	}
}

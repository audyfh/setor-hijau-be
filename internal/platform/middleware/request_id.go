package middleware

import (
	"context"

	"github.com/audyfh/setor-hijau-be/internal/platform/logger"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const HeaderXRequestID = "X-Request-ID"

// RequestID ensures every HTTP request has an X-Request-ID. If the incoming
// request already has one, it is preserved; otherwise, a new UUID v4 is generated.
// The ID is attached to the response header and to both Gin context and Go context.
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		reqID := c.GetHeader(HeaderXRequestID)
		if reqID == "" {
			reqID = uuid.NewString()
		}

		c.Header(HeaderXRequestID, reqID)
		c.Set(string(logger.RequestIDKey), reqID)

		ctx := context.WithValue(c.Request.Context(), logger.RequestIDKey, reqID)
		c.Request = c.Request.WithContext(ctx)

		c.Next()
	}
}

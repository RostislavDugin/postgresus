package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"databasus-backend/internal/util/logger"
)

const RequestIDHeader = "X-Request-Id"

// An inbound X-Request-Id is ignored: a client that picked its own ID could merge its actions into
// another user's trace.
func AssignRequestID() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		requestID := uuid.NewString()

		ctx.Header(RequestIDHeader, requestID)
		ctx.Request = ctx.Request.WithContext(logger.ContextWithRequestID(ctx.Request.Context(), requestID))

		ctx.Next()
	}
}

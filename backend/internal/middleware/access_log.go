package middleware

import (
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	verification_agents "databasus-backend/internal/features/verification/agents"
)

// Static assets and the SPA fallback match no route, so ctx.FullPath() is empty for them. A marker
// keeps that traffic filterable instead of looking like a routing bug.
const staticRouteName = "static"

// The query string is left out on purpose: download and restore tokens travel there.
func LogAccess(log *slog.Logger) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		startedAt := time.Now().UTC()

		ctx.Next()

		status := ctx.Writer.Status()
		message := fmt.Sprintf("%s %s -> %d in %dms",
			ctx.Request.Method,
			ctx.Request.URL.Path,
			status,
			time.Since(startedAt).Milliseconds(),
		)

		route := ctx.FullPath()
		if route == "" {
			route = staticRouteName
		}

		attributes := []any{
			"route", route,
			"client_ip", ctx.ClientIP(),
			"response_bytes", ctx.Writer.Size(),
		}

		// user_id already reaches every record from the request context, which AuthMiddleware
		// populates; an agent has no such carrier, so it is attached here. Runs after ctx.Next(),
		// so moving this middleware earlier in the chain would drop the agent.
		if agent, isAgent := verification_agents.GetAgentFromContext(ctx); isAgent {
			attributes = append(attributes, "agent_id", agent.ID)
		}

		if status >= http.StatusInternalServerError {
			// Handlers attach the cause with ctx.Error so the access line carries it, rather than
			// each of them logging its own near-duplicate of this record.
			if handlerErrors := collectHandlerErrors(ctx); handlerErrors != "" {
				attributes = append(attributes, "error", handlerErrors)
			}

			log.ErrorContext(ctx.Request.Context(), message, attributes...)

			return
		}

		log.InfoContext(ctx.Request.Context(), message, attributes...)
	}
}

func collectHandlerErrors(ctx *gin.Context) string {
	if len(ctx.Errors) == 0 {
		return ""
	}

	messages := make([]string, 0, len(ctx.Errors))
	for _, handlerError := range ctx.Errors {
		messages = append(messages, handlerError.Error())
	}

	return strings.Join(messages, "; ")
}

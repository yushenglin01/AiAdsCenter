package middleware

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

func RequestContext(logger *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		started := time.Now()
		requestID := c.GetHeader("X-Request-ID")
		if requestID == "" {
			requestID = uuid.NewString()
		}
		traceID := c.GetHeader("X-Trace-ID")
		if traceID == "" {
			traceID = uuid.NewString()
		}
		c.Set("request_id", requestID)
		c.Set("trace_id", traceID)
		c.Header("X-Request-ID", requestID)
		c.Header("X-Trace-ID", traceID)
		c.Next()
		logger.Info("http request",
			zap.String("request_id", requestID), zap.String("trace_id", traceID),
			zap.String("method", c.Request.Method), zap.String("path", c.FullPath()),
			zap.Int("status", c.Writer.Status()), zap.Int64("duration_ms", time.Since(started).Milliseconds()),
		)
	}
}

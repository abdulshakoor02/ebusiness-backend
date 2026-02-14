package middleware

import (
	"log/slog"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

func RequestLogger() fiber.Handler {
	return func(c *fiber.Ctx) error {
		start := time.Now()
		requestID := c.Get("X-Request-ID")
		if requestID == "" {
			requestID = uuid.New().String()
			c.Set("X-Request-ID", requestID)
		}

		// Process request
		err := c.Next()

		// Log request details
		duration := time.Since(start)
		statusCode := c.Response().StatusCode()

		// Extract fields
		fields := []any{
			slog.String("request_id", requestID),
			slog.String("method", c.Method()),
			slog.String("path", c.Path()),
			slog.Int("status", statusCode),
			slog.Duration("latency", duration),
			slog.String("ip", c.IP()),
		}

		// Add context fields (e.g. tenant_id, user_id) if set by AuthMiddleware
		if tenantID, ok := c.Locals("tenant_id").(string); ok {
			fields = append(fields, slog.String("tenant_id", tenantID))
		}
		if userID, ok := c.Locals("user_id").(string); ok {
			fields = append(fields, slog.String("user_id", userID))
		}

		if err != nil {
			fields = append(fields, slog.String("error", err.Error()))
			slog.Error("Request failed", fields...)
			return err
		}

		// Log based on status
		if statusCode >= 500 {
			slog.Error("Request failed", fields...)
		} else if statusCode >= 400 {
			slog.Warn("Request client error", fields...)
		} else {
			slog.Info("Request processed", fields...)
		}

		return nil
	}
}

package logging

import (
	"context"
	"time"

	"github.com/go-kratos/kratos/v2/errors"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/middleware"
	"github.com/go-kratos/kratos/v2/transport"
	"github.com/go-kratos/kratos/v2/transport/http"
)

// Server is a server logging middleware.
func Server(logger log.Logger) middleware.Middleware {
	return func(handler middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req interface{}) (reply interface{}, err error) {
			var (
				code      int32
				reason    string
				message   string
				kind      string
				operation string
			)
			startTime := time.Now()

			if info, ok := transport.FromServerContext(ctx); ok {
				kind = info.Kind().String()
				operation = info.Operation()
			}

			reply, err = handler(ctx, req)

			// Calculate latency
			latency := time.Since(startTime)

			// Extract error information if exists
			if err != nil {
				if se := errors.FromError(err); se != nil {
					code = se.Code
					reason = se.Reason
					message = se.Message
				}
			} else {
				code = 200
			}
			// Build log fields
			logFields := []interface{}{
				"kind", kind,
				"operation", operation,
				"latency", latency.Seconds(),
				"latency_human", latency.String(),
			}

			// Add HTTP specific fields
			if tr, ok := transport.FromServerContext(ctx); ok {
				if ht, ok := tr.(*http.Transport); ok {
					logFields = append(logFields,
						"method", ht.Request().Method,
						"uri", ht.Request().URL.String(),
						"user_agent", ht.Request().UserAgent(),
						"remote_ip", ht.Request().RemoteAddr,
					)

					// Get request ID from header
					requestID := ht.Request().Header.Get("X-Request-ID")
					if requestID != "" {
						logFields = append(logFields, "request_id", requestID)
					}

					// Get content length
					if ht.Request().ContentLength > 0 {
						logFields = append(logFields, "byte_in", ht.Request().ContentLength)
					}
				}
			}

			// Create log helper with logger
			logHelper := log.NewHelper(logger)

			// Add error information
			if err != nil {
				logFields = append(logFields,
					"code", code,
					"error_reason", reason,
					"error_message", message,
				)

				// Log based on error code
				if code >= 400 && code < 500 {
					logFields = append([]interface{}{"msg", "api call failed by bad request"}, logFields...)
					logHelper.Warnw(logFields...)
				} else if code >= 500 {
					logFields = append([]interface{}{"msg", "api call failed by internal error"}, logFields...)
					logHelper.Errorw(logFields...)
				} else {
					logFields = append([]interface{}{"msg", "api call failed"}, logFields...)
					logHelper.Errorw(logFields...)
				}
			} else {
				logFields = append(logFields, "code", code)
				logFields = append([]interface{}{"msg", "api call success"}, logFields...)
				logHelper.Infow(logFields...)
			}

			return
		}
	}
}

// Client is a client logging middleware.
func Client(logger log.Logger) middleware.Middleware {
	return func(handler middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req interface{}) (reply interface{}, err error) {
			var (
				code      int32
				reason    string
				kind      string
				operation string
			)
			startTime := time.Now()

			if info, ok := transport.FromClientContext(ctx); ok {
				kind = info.Kind().String()
				operation = info.Operation()
			}

			reply, err = handler(ctx, req)

			latency := time.Since(startTime)

			if err != nil {
				if se := errors.FromError(err); se != nil {
					code = se.Code
					reason = se.Reason
				}
			}

			logFields := []interface{}{
				"kind", kind,
				"operation", operation,
				"latency", latency.Seconds(),
				"latency_human", latency.String(),
			}

			// Create log helper with logger
			logHelper := log.NewHelper(logger)

			if err != nil {
				logFields = append(logFields,
					"error", err.Error(),
					"code", code,
					"reason", reason,
				)
				logFields = append([]interface{}{"msg", "client call failed"}, logFields...)
				logHelper.Errorw(logFields...)
			} else {
				logFields = append([]interface{}{"msg", "client call success"}, logFields...)
				logHelper.Infow(logFields...)
			}

			return
		}
	}
}

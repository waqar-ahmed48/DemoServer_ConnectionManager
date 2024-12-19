package helper

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/google/uuid"
)

// ContextKeyRequestLogger used for indexing in HTTP request context.
type ContextKeyRequestLogger struct{}

// PrepareContext used to prepare X-Request-Id tag in HTTP response and provides context aware logger.
func PrepareContext(r *http.Request, rw *http.ResponseWriter, l *slog.Logger) (string, *slog.Logger) {
	requestid := r.Header.Get("X-Request-Id")

	if requestid == "" {
		requestid = uuid.New().String()
		r.Header.Set("X-Request-Id", requestid)
		(*rw).Header().Set("X-Request-Id", requestid)
	}

	c := r.Context()

	if c != nil {
		obj := c.Value(ContextKeyRequestLogger{})

		if obj != nil {
			cl := obj.(*slog.Logger)

			return requestid, cl
		}

		if l == nil {
			l = slog.Default()
		}

		logAttrGroup := slog.Group(
			"request",
			"requestid", requestid,
			"endpoint", r.URL.EscapedPath(),
			"method", r.Method)

		cl := l.With(logAttrGroup)

		ctx := context.WithValue(r.Context(), ContextKeyRequestLogger{}, cl)
		(*r) = *r.WithContext(ctx)

		return requestid, cl
	}

	return requestid, nil
}

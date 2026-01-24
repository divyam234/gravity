package logger

import (
	"context"
	"net/http"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"github.com/go-chi/chi/v5/middleware"
)

var L *zap.Logger
var S *zap.SugaredLogger

func init() {
	// Initialize a default logger so we never have nil L
	L = zap.NewNop()
	S = L.Sugar()
}

func New(level string, isDev bool) *zap.Logger {
	var config zap.Config

	if isDev {
		config = zap.NewDevelopmentConfig()
		config.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	} else {
		config = zap.NewProductionConfig()
	}

	// Parse level
	var zapLevel zapcore.Level
	if err := zapLevel.UnmarshalText([]byte(level)); err != nil {
		zapLevel = zap.InfoLevel
	}
	config.Level = zap.NewAtomicLevelAt(zapLevel)
	config.DisableStacktrace = true

	logger, _ := config.Build(zap.AddCallerSkip(1))
	L = logger
	S = L.Sugar()
	return L
}

// Middleware returns a chi-compatible middleware for request logging
func Middleware(l *zap.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
			t1 := time.Now()
			
			defer func() {
				l.Info("request completed",
					zap.String("method", r.Method),
					zap.String("path", r.URL.Path),
					zap.Int("status", ww.Status()),
					zap.Int("size", ww.BytesWritten()),
					zap.Duration("duration", time.Since(t1)),
					zap.String("ip", r.RemoteAddr),
				)
			}()

			next.ServeHTTP(ww, r)
		})
	}
}

// WithContext returns a logger with context info
func WithContext(ctx context.Context) *zap.Logger {
	// In the future, we could extract request ID from context
	return L
}

type ZapWriter struct {
	Sugar *zap.SugaredLogger
}

func (w ZapWriter) Printf(fmt string, args ...interface{}) {
	w.Sugar.Debugf(fmt, args...)
}

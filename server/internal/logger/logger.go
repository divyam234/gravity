package logger

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-chi/chi/v5/middleware"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var L *zap.Logger
var S *zap.SugaredLogger

func init() {
	L = zap.NewNop()
	S = L.Sugar()
}

func New(level string, isDev bool, logFile string, useJSON bool) *zap.Logger {
	var zapLevel zapcore.Level
	if err := zapLevel.UnmarshalText([]byte(level)); err != nil {
		zapLevel = zap.InfoLevel
	}

	var core zapcore.Core

	// Console Core
	if useJSON {
		// JSON on console (Production style)
		config := zap.NewProductionConfig()
		config.Level = zap.NewAtomicLevelAt(zapLevel)
		config.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
		core = zapcore.NewCore(
			zapcore.NewJSONEncoder(config.EncoderConfig),
			zapcore.Lock(os.Stderr),
			zap.NewAtomicLevelAt(zapLevel),
		)
	} else {
		// Pretty on console (Development style)
		core = &prettyCore{
			level: zapLevel,
			out:   zapcore.Lock(os.Stderr),
		}
	}

	// File Core (Always JSON if logFile is provided)
	if logFile != "" {
		// Ensure directory exists
		_ = os.MkdirAll(filepath.Dir(logFile), 0755)

		f, err := os.OpenFile(logFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err == nil {
			fileEncoderConfig := zap.NewProductionEncoderConfig()
			fileEncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
			fileCore := zapcore.NewCore(
				zapcore.NewJSONEncoder(fileEncoderConfig),
				zapcore.AddSync(f),
				zap.NewAtomicLevelAt(zapLevel),
			)
			core = zapcore.NewTee(core, fileCore)
		} else {
			fmt.Printf("✗ failed to open log file %s: %v\n", logFile, err)
		}
	}

	L = zap.New(core)
	S = L.Sugar()
	return L
}

// prettyCore is a custom zapcore.Core for human-readable, colored, non-structured logs
type prettyCore struct {
	level  zapcore.Level
	out    zapcore.WriteSyncer
	fields []zapcore.Field
}

func (c *prettyCore) Enabled(l zapcore.Level) bool {
	return l >= c.level
}

func (c *prettyCore) With(fields []zapcore.Field) zapcore.Core {
	return &prettyCore{
		level:  c.level,
		out:    c.out,
		fields: append(c.fields[:len(c.fields):len(c.fields)], fields...),
	}
}

func (c *prettyCore) Check(ent zapcore.Entry, ce *zapcore.CheckedEntry) *zapcore.CheckedEntry {
	if c.Enabled(ent.Level) {
		return ce.AddCore(ent, c)
	}
	return ce
}

func (c *prettyCore) Write(ent zapcore.Entry, fields []zapcore.Field) error {
	var component string
	var otherFields []string

	// Combine baked-in fields and call-site fields
	allFields := append(c.fields[:len(c.fields):len(c.fields)], fields...)

	for _, f := range allFields {
		if f.Key == "component" {
			component = f.String
		} else {
			// Format other fields as key=value
			var val string
			switch f.Type {
			case zapcore.StringType:
				val = f.String
			case zapcore.Int64Type, zapcore.Int32Type:
				val = fmt.Sprintf("%d", f.Integer)
			case zapcore.ErrorType:
				val = f.Interface.(error).Error()
			case zapcore.BoolType:
				val = fmt.Sprintf("%v", f.Integer != 0)
			default:
				if f.Interface != nil {
					val = fmt.Sprintf("%v", f.Interface)
				} else {
					val = fmt.Sprintf("%v", f.Integer)
				}
			}
			otherFields = append(otherFields, fmt.Sprintf("\x1b[90m%s=\x1b[0m%v", f.Key, val))
		}
	}

	// Prepare Level Icon and Color
	var icon, color string
	switch ent.Level {
	case zapcore.DebugLevel:
		icon, color = "🐛", "\x1b[35m" // Magenta
	case zapcore.InfoLevel:
		icon, color = "✓", "\x1b[32m" // Green
	case zapcore.WarnLevel:
		icon, color = "⚠", "\x1b[33m" // Yellow
	case zapcore.ErrorLevel, zapcore.DPanicLevel, zapcore.PanicLevel, zapcore.FatalLevel:
		icon, color = "✗", "\x1b[31m" // Red
	default:
		icon, color = "·", "\x1b[37m" // White
	}

	// Format Component
	compStr := ""
	if component != "" {
		compStr = fmt.Sprintf("\x1b[36m[%s]\x1b[0m ", strings.ToUpper(component))
	}

	// Build the line
	// Format: HH:MM:SS  ICON LEVEL  [COMPONENT] MESSAGE   key=val key=val
	line := fmt.Sprintf("%s  %s %s%-5s\x1b[0m  %s%s",
		ent.Time.Format("15:04:05"),
		icon,
		color,
		strings.ToUpper(ent.Level.String()),
		compStr,
		ent.Message,
	)

	if len(otherFields) > 0 {
		line += "  " + strings.Join(otherFields, " ")
	}
	line += "\n"

	_, err := c.out.Write([]byte(line))
	return err
}

func (c *prettyCore) Sync() error {
	return c.out.Sync()
}

// Component returns a logger with a component tag
func Component(name string) *zap.Logger {
	return L.With(zap.String("component", name))
}

// SugaredComponent returns a sugared logger with a component tag
func SugaredComponent(name string) *zap.SugaredLogger {
	return S.With("component", name)
}

func Middleware(l *zap.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
			t1 := time.Now()

			defer func() {
				status := ww.Status()
				duration := time.Since(t1)

				var colorCode, icon string
				if status >= 500 {
					colorCode, icon = "\x1b[31m", "✗" // Red
				} else if status >= 400 {
					colorCode, icon = "\x1b[33m", "✗" // Yellow
				} else {
					colorCode, icon = "\x1b[32m", "→" // Green
				}
				reset := "\x1b[0m"
				cyan := "\x1b[36m"

				fmt.Printf("%s %s  %s[HTTP]%s %-4s %-25s %s%d %-5s%s %v %dB\n",
					icon,
					time.Now().Format("15:04:05"),
					cyan, reset,
					r.Method,
					r.URL.Path,
					colorCode,
					status,
					http.StatusText(status),
					reset,
					duration.Round(time.Millisecond),
					ww.BytesWritten(),
				)
			}()

			next.ServeHTTP(ww, r)
		})
	}
}

type ZapWriter struct {
	Sugar *zap.SugaredLogger
}

func (w ZapWriter) Printf(fmtStr string, args ...any) {
	w.Sugar.Debugf(fmtStr, args...)
}

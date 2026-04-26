package logger

import (
	"context"
	"os"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

var (
	// tracer OpenTelemetry追踪器
	tracer = otel.Tracer("verdant-common")
)

// Span 链路追踪Span包装
type Span struct {
	trace.Span
}

// End 结束Span
func (s *Span) End() {
	if s.Span != nil {
		s.Span.End()
	}
}

// CreateSpan 创建新的追踪Span
func CreateSpan(ctx context.Context, name string) (context.Context, *Span) {
	ctx, span := tracer.Start(ctx, name)
	return ctx, &Span{Span: span}
}

// InitLogger 初始化日志配置
func InitLogger(level string, format string) {
	// 设置日志级别
	logLevel, err := zerolog.ParseLevel(level)
	if err != nil {
		logLevel = zerolog.InfoLevel
	}
	zerolog.SetGlobalLevel(logLevel)

	// 设置日志格式
	if format == "console" {
		log.Logger = zerolog.New(zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.RFC3339}).With().Timestamp().Logger()
	} else {
		log.Logger = zerolog.New(os.Stderr).With().Timestamp().Logger()
	}
}

// WithTraceID 添加TraceID到日志
func WithTraceID(ctx context.Context) *zerolog.Logger {
	span := trace.SpanFromContext(ctx)
	if span.SpanContext().IsValid() {
		logger := log.With().
			Str("trace_id", span.SpanContext().TraceID().String()).
			Str("span_id", span.SpanContext().SpanID().String()).
			Logger()
		return &logger
	}
	return &log.Logger
}

// LogError 记录错误日志
func LogError(ctx context.Context, err error, msg string) {
	WithTraceID(ctx).Error().Err(err).Msg(msg)
}

// LogInfo 记录信息日志
func LogInfo(ctx context.Context, msg string) {
	WithTraceID(ctx).Info().Msg(msg)
}

// LogWarn 记录警告日志
func LogWarn(ctx context.Context, msg string) {
	WithTraceID(ctx).Warn().Msg(msg)
}

// LogDebug 记录调试日志
func LogDebug(ctx context.Context, msg string) {
	WithTraceID(ctx).Debug().Msg(msg)
}

// SetSpanAttribute 设置Span属性
func SetSpanAttribute(ctx context.Context, key string, value interface{}) {
	span := trace.SpanFromContext(ctx)
	if span.SpanContext().IsValid() {
		switch v := value.(type) {
		case string:
			span.SetAttributes(attribute.String(key, v))
		case int:
			span.SetAttributes(attribute.Int(key, v))
		case int64:
			span.SetAttributes(attribute.Int64(key, v))
		case float64:
			span.SetAttributes(attribute.Float64(key, v))
		case bool:
			span.SetAttributes(attribute.Bool(key, v))
		default:
			span.SetAttributes(attribute.String(key, "%+v"))
		}
	}
}

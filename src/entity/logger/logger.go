package logger

import (
	"context"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

func WithTraceId(ctx context.Context) zap.Field {
	return zap.String("trace_id", trace.SpanFromContext(ctx).SpanContext().TraceID().String())
}

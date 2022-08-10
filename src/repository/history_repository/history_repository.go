package history_repository

import "go.opentelemetry.io/otel"

var tracer = otel.Tracer("history_repository")

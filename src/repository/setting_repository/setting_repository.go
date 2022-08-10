package setting_repository

import "go.opentelemetry.io/otel"

var tracer = otel.Tracer("setting_repository")

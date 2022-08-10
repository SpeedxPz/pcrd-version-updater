package version_event_repository

import "go.opentelemetry.io/otel"

var tracer = otel.Tracer("version_event_repository")

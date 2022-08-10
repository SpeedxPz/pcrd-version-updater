package application_repository

import "go.opentelemetry.io/otel"

var tracer = otel.Tracer("application_repository")

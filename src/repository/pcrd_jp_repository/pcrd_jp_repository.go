package pcrd_jp_repository

import (
	"go.opentelemetry.io/otel"
)

var tracer = otel.Tracer("pcrd_jp_repository")

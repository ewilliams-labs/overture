package ports

import (
	"context"

	"github.com/ewilliams-labs/overture/backend/internal/core/domain"
)

type IntentCompiler interface {
	AnalyzeIntent(ctx context.Context, message string) (domain.IntentObject, error)
}

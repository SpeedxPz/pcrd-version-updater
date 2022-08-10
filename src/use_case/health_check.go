package use_case

import (
	"context"
	"fmt"
)

func (u UseCase) HealthCheck(ctx context.Context) error {

	err := u.applicationRepository.HealthCheck(ctx)
	if err != nil {
		return fmt.Errorf("applicationRepository.HealthCheck: %w", err)
	}

	return nil
}

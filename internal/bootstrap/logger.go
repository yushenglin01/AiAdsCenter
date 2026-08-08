package bootstrap

import (
	"fmt"

	"go.uber.org/zap"
)

func NewLogger(environment string) (*zap.Logger, error) {
	if environment == "development" {
		logger, err := zap.NewDevelopment()
		if err != nil {
			return nil, fmt.Errorf("create development logger: %w", err)
		}
		return logger, nil
	}
	logger, err := zap.NewProduction()
	if err != nil {
		return nil, fmt.Errorf("create production logger: %w", err)
	}
	return logger, nil
}

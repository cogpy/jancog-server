package cron

import (
	"context"

	"github.com/mileusna/crontab"
	inference_model_registry "menlo.ai/jan-api-gateway/app/domain/inference_model_registry"
	"menlo.ai/jan-api-gateway/config/environment_variables"
)

type CronService struct {
	InferenceModelRegistry *inference_model_registry.InferenceModelRegistry
}

func NewService(registry *inference_model_registry.InferenceModelRegistry) *CronService {
	return &CronService{
		InferenceModelRegistry: registry,
	}
}

func (cs *CronService) Start(ctx context.Context, ctab *crontab.Crontab) {
	// Run initial check
	cs.InferenceModelRegistry.CheckInferenceModels(ctx)

	ctab.AddJob("* * * * *", func() {
		cs.InferenceModelRegistry.CheckInferenceModels(ctx)

		// Reload environment variables
		environment_variables.EnvironmentVariables.LoadFromEnv()
	})
}

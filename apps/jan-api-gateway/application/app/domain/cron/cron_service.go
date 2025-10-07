package cron

import (
	"context"

	"github.com/mileusna/crontab"
	"menlo.ai/jan-api-gateway/app/utils/httpclients/chat"
	"menlo.ai/jan-api-gateway/app/utils/logger"
	"menlo.ai/jan-api-gateway/config/environment_variables"
)

type CronService struct {
	modelClient *chat.ChatModelClient
}

func NewService(modelClient *chat.ChatModelClient) *CronService {
	return &CronService{
		modelClient: modelClient,
	}
}

func (cs *CronService) Start(ctx context.Context, ctab *crontab.Crontab) {
	cs.refreshModels(ctx)

	ctab.AddJob("* * * * *", func() {
		cs.refreshModels(ctx)
		environment_variables.EnvironmentVariables.LoadFromEnv()
	})
}

func (cs *CronService) refreshModels(ctx context.Context) {
	if cs.modelClient == nil {
		return
	}
	if _, err := cs.modelClient.ListModels(ctx); err != nil {
		logger.GetLogger().Warnf("cron: unable to refresh models: %v", err)
	}
}

package infrastructure

import (
	"github.com/google/wire"
	inferencemodelregistry "menlo.ai/jan-api-gateway/app/domain/inference_model_registry"
	"menlo.ai/jan-api-gateway/app/infrastructure/cache"
	"menlo.ai/jan-api-gateway/app/infrastructure/inference"
)

var InfrastructureProvider = wire.NewSet(
	inference.NewJanRestyClient,
	inference.NewJanChatCompletionClient,
	inference.NewJanInferenceProvider,
	cache.NewRedisCacheService,
	inferencemodelregistry.NewInferenceModelRegistry,
)

package infrastructure

import (
	"github.com/google/wire"
	"menlo.ai/jan-api-gateway/app/infrastructure/cache"
	"menlo.ai/jan-api-gateway/app/infrastructure/inference"
)

var InfrastructureProvider = wire.NewSet(
	inference.NewInferenceProvider,
	cache.NewRedisCacheService,
)

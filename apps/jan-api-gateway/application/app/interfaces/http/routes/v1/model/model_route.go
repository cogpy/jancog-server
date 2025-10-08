package modelroute

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"menlo.ai/jan-api-gateway/app/domain/auth"
	domainmodel "menlo.ai/jan-api-gateway/app/domain/model"
	"menlo.ai/jan-api-gateway/app/domain/project"
	"menlo.ai/jan-api-gateway/app/infrastructure/inference"
	"menlo.ai/jan-api-gateway/app/interfaces/http/responses"
)

type ModelAPI struct {
	inferenceProvider    *inference.InferenceProvider
	authService          *auth.AuthService
	projectService       *project.ProjectService
	providerRegistry     *domainmodel.ProviderRegistryService
	providerModelService *domainmodel.ProviderModelService
}

func NewModelAPI(
	inferenceProvider *inference.InferenceProvider,
	authService *auth.AuthService,
	projectService *project.ProjectService,
	providerRegistry *domainmodel.ProviderRegistryService,
	providerModelService *domainmodel.ProviderModelService,
) *ModelAPI {
	return &ModelAPI{
		inferenceProvider:    inferenceProvider,
		authService:          authService,
		projectService:       projectService,
		providerRegistry:     providerRegistry,
		providerModelService: providerModelService,
	}
}

func (modelAPI *ModelAPI) RegisterRouter(router *gin.RouterGroup) {
	group := router.Group("",
		modelAPI.authService.AppUserAuthMiddleware(),
		modelAPI.authService.RegisteredUserMiddleware(),
	)
	group.GET("models", modelAPI.GetModels)
}

// ListModels
// @Summary List available models
// @Description Retrieves a list of available models that can be used for chat completions or other tasks.
// @Tags Chat Completions API
// @Security BearerAuth
// @Accept json
// @Produce json
// @Success 200 {object} ModelsResponse "Successful response"
// @Router /v1/models [get]
func (modelAPI *ModelAPI) GetModels(reqCtx *gin.Context) {
	ctx := reqCtx.Request.Context()
	includeProviderData := strings.EqualFold(reqCtx.GetHeader("X-PROVIDER-DATA"), "true")

	_, _, providers, ok := ResolveAccessibleProviders(reqCtx, modelAPI.authService, modelAPI.projectService, modelAPI.providerRegistry)
	if !ok {
		return
	}

	providerByID := make(map[uint]*domainmodel.Provider, len(providers))
	providerIDs := make([]uint, 0, len(providers))
	for _, provider := range providers {
		if provider == nil {
			continue
		}
		providerByID[provider.ID] = provider
		providerIDs = append(providerIDs, provider.ID)
	}

	providerModels, err := modelAPI.providerModelService.ListActiveByProviderIDs(ctx, providerIDs)
	if err != nil {
		reqCtx.AbortWithStatusJSON(http.StatusInternalServerError, responses.ErrorResponse{
			Code:          "f7f0f635-3f13-4c6f-b436-a78a5ccaa1af",
			ErrorInstance: err,
		})
		return
	}

	if includeProviderData {
		models := BuildModelsWithProvider(providerModels, providerByID)
		reqCtx.JSON(http.StatusOK, ModelsWithProviderResponse{
			Object: "list",
			Data:   models,
		})
		return
	}

	result := MergeModels(providerModels, providerByID)
	reqCtx.JSON(http.StatusOK, ModelsResponse{
		Object: "list",
		Data:   result,
	})
}

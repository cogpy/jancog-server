package organization

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"menlo.ai/jan-api-gateway/app/domain/auth"
	domainmodel "menlo.ai/jan-api-gateway/app/domain/model"
	"menlo.ai/jan-api-gateway/app/infrastructure/inference"
	"menlo.ai/jan-api-gateway/app/interfaces/http/responses"
	"menlo.ai/jan-api-gateway/app/utils/ptr"
)

type ModelProviderRoute struct {
	authService       *auth.AuthService
	providerRegistry  *domainmodel.ProviderRegistryService
	inferenceProvider *inference.InferenceProvider
}

func NewModelProviderRoute(
	authService *auth.AuthService,
	providerRegistry *domainmodel.ProviderRegistryService,
	inferenceProvider *inference.InferenceProvider,
) *ModelProviderRoute {
	return &ModelProviderRoute{
		authService:       authService,
		providerRegistry:  providerRegistry,
		inferenceProvider: inferenceProvider,
	}
}

func (route *ModelProviderRoute) RegisterRouter(router *gin.RouterGroup) {
	group := router.Group("/models/providers",
		route.authService.AdminUserAuthMiddleware(),
		route.authService.RegisteredUserMiddleware(),
		route.authService.OrganizationMemberRoleMiddleware(auth.OrganizationMemberRuleOwnerOnly),
	)
	group.POST("", route.registerProvider)
	group.PATCH("/:provider_public_id", route.updateProvider)
}

type registerProviderRequest struct {
	Name     string            `json:"name" binding:"required"`
	Vendor   string            `json:"vendor" binding:"required"`
	BaseURL  string            `json:"base_url" binding:"required"`
	APIKey   string            `json:"api_key"`
	Metadata map[string]string `json:"metadata"`
	Active   *bool             `json:"active"`
}

type registerProviderResponse struct {
	ID       string                         `json:"id"`
	Slug     string                         `json:"slug"`
	Name     string                         `json:"name"`
	Vendor   string                         `json:"vendor"`
	BaseURL  string                         `json:"base_url"`
	Active   bool                           `json:"active"`
	Metadata map[string]string              `json:"metadata,omitempty"`
	Models   []registerProviderModelSummary `json:"models"`
}

type registerProviderModelSummary struct {
	ID            string  `json:"id"`
	ModelKey      string  `json:"model_key"`
	DisplayName   string  `json:"display_name"`
	CatalogID     *string `json:"catalog_id,omitempty"`
	CatalogStatus *string `json:"catalog_status,omitempty"`
}

type updateProviderRequest struct {
	Name     *string            `json:"name"`
	BaseURL  *string            `json:"base_url"`
	APIKey   *string            `json:"api_key"`
	Metadata *map[string]string `json:"metadata"`
	Active   *bool              `json:"active"`
}

type providerDetailResponse struct {
	ID       string            `json:"id"`
	Slug     string            `json:"slug"`
	Name     string            `json:"name"`
	Vendor   string            `json:"vendor"`
	BaseURL  string            `json:"base_url"`
	Active   bool              `json:"active"`
	Metadata map[string]string `json:"metadata,omitempty"`
}

func (route *ModelProviderRoute) registerProvider(reqCtx *gin.Context) {
	ctx := reqCtx.Request.Context()
	orgEntity, ok := auth.GetAdminOrganizationFromContext(reqCtx)
	if !ok {
		return
	}

	var request registerProviderRequest
	if err := reqCtx.ShouldBindJSON(&request); err != nil {
		reqCtx.AbortWithStatusJSON(http.StatusBadRequest, responses.ErrorResponse{
			Code:          "4dfe7980-9d40-47fb-8cf1-1864dfd1e3eb",
			ErrorInstance: err,
		})
		return
	}

	active := true
	if request.Active != nil {
		active = *request.Active
	}

	result, err := route.providerRegistry.RegisterProvider(ctx, domainmodel.RegisterProviderInput{
		OrganizationID: orgEntity.ID,
		Name:           request.Name,
		Vendor:         request.Vendor,
		BaseURL:        request.BaseURL,
		APIKey:         request.APIKey,
		Metadata:       request.Metadata,
		Active:         active,
	})
	if err != nil {
		status := http.StatusBadRequest
		reqCtx.AbortWithStatusJSON(status, responses.ErrorResponse{
			Code:  err.GetCode(),
			Error: err.GetMessage(),
		})
		return
	}

	models, fetchErr := route.inferenceProvider.ListModels(ctx, result.Provider)
	if fetchErr != nil {
		reqCtx.AbortWithStatusJSON(http.StatusBadGateway, responses.ErrorResponse{
			Code:          "cbe9fb03-a434-4d57-8a59-7b1e6830f9e5",
			ErrorInstance: fetchErr,
		})
		return
	}

	syncResults, syncErr := route.providerRegistry.SyncProviderModels(ctx, result.Provider, models)
	if syncErr != nil {
		reqCtx.AbortWithStatusJSON(http.StatusInternalServerError, responses.ErrorResponse{
			Code:  syncErr.GetCode(),
			Error: syncErr.GetMessage(),
		})
		return
	}
	result.Models = syncResults

	resp := toRegisterProviderResponse(result)
	reqCtx.JSON(http.StatusOK, resp)
}

func toRegisterProviderResponse(result *domainmodel.ProviderRegistrationResult) registerProviderResponse {
	provider := result.Provider
	resp := registerProviderResponse{
		ID:       provider.PublicID,
		Slug:     provider.Slug,
		Name:     provider.DisplayName,
		Vendor:   strings.ToLower(string(provider.Kind)),
		BaseURL:  provider.BaseURL,
		Active:   provider.Active,
		Metadata: provider.Metadata,
	}

	for _, model := range result.Models {
		item := registerProviderModelSummary{
			ID:          model.ProviderModel.PublicID,
			ModelKey:    model.ProviderModel.ModelKey,
			DisplayName: model.ProviderModel.DisplayName,
		}
		if model.Catalog != nil {
			item.CatalogID = ptr.ToString(model.Catalog.PublicID)
			status := string(model.Catalog.Status)
			item.CatalogStatus = ptr.ToString(status)
		}
		resp.Models = append(resp.Models, item)
	}

	return resp
}

func (route *ModelProviderRoute) updateProvider(reqCtx *gin.Context) {
	ctx := reqCtx.Request.Context()
	orgEntity, ok := auth.GetAdminOrganizationFromContext(reqCtx)
	if !ok {
		return
	}
	publicID := strings.TrimSpace(reqCtx.Param("provider_public_id"))
	if publicID == "" {
		reqCtx.AbortWithStatusJSON(http.StatusBadRequest, responses.ErrorResponse{
			Code:  "28dd6e4a-b7df-4e75-bb70-2b7f2a44d8ec",
			Error: "provider id is required",
		})
		return
	}

	provider, err := route.providerRegistry.FindByPublicID(ctx, publicID)
	if err != nil {
		status := http.StatusBadRequest
		if err.GetCode() == "d16271bf-54f5-4b25-bbd2-2353f1d5265c" {
			status = http.StatusNotFound
		}
		reqCtx.AbortWithStatusJSON(status, responses.ErrorResponse{
			Code:  err.GetCode(),
			Error: err.GetMessage(),
		})
		return
	}
	if provider.OrganizationID == nil || *provider.OrganizationID != orgEntity.ID {
		reqCtx.AbortWithStatusJSON(http.StatusNotFound, responses.ErrorResponse{
			Code:  "a2b8c03f-4a15-4431-9a0f-0a5c8ef0e83d",
			Error: "provider not found",
		})
		return
	}
	if provider.ProjectID != nil {
		reqCtx.AbortWithStatusJSON(http.StatusBadRequest, responses.ErrorResponse{
			Code:  "4b4ff5ab-6a55-4aa7-842c-9a8d6fd8b061",
			Error: "only organization providers can be updated here",
		})
		return
	}

	var request updateProviderRequest
	if err := reqCtx.ShouldBindJSON(&request); err != nil {
		reqCtx.AbortWithStatusJSON(http.StatusBadRequest, responses.ErrorResponse{
			Code:          "f9be18d0-5eac-46e2-8fd6-779b272918aa",
			ErrorInstance: err,
		})
		return
	}

	input := domainmodel.UpdateProviderInput{
		Name:     request.Name,
		BaseURL:  request.BaseURL,
		APIKey:   request.APIKey,
		Metadata: request.Metadata,
		Active:   request.Active,
	}

	updated, updateErr := route.providerRegistry.UpdateProvider(ctx, provider, input)
	if updateErr != nil {
		reqCtx.AbortWithStatusJSON(http.StatusBadRequest, responses.ErrorResponse{
			Code:  updateErr.GetCode(),
			Error: updateErr.GetMessage(),
		})
		return
	}

	reqCtx.JSON(http.StatusOK, toProviderDetailResponse(updated))
}

func toProviderDetailResponse(provider *domainmodel.Provider) providerDetailResponse {
	return providerDetailResponse{
		ID:       provider.PublicID,
		Slug:     provider.Slug,
		Name:     provider.DisplayName,
		Vendor:   strings.ToLower(string(provider.Kind)),
		BaseURL:  provider.BaseURL,
		Active:   provider.Active,
		Metadata: provider.Metadata,
	}
}

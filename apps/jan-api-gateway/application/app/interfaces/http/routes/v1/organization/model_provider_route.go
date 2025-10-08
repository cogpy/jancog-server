package organization

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"menlo.ai/jan-api-gateway/app/domain/auth"
	domainmodel "menlo.ai/jan-api-gateway/app/domain/model"
	"menlo.ai/jan-api-gateway/app/interfaces/http/responses"
	"menlo.ai/jan-api-gateway/app/utils/ptr"
)

type ModelProviderRoute struct {
	authService      *auth.AuthService
	providerRegistry *domainmodel.ProviderRegistryService
}

func NewModelProviderRoute(authService *auth.AuthService, providerRegistry *domainmodel.ProviderRegistryService) *ModelProviderRoute {
	return &ModelProviderRoute{
		authService:      authService,
		providerRegistry: providerRegistry,
	}
}

func (route *ModelProviderRoute) RegisterRouter(router *gin.RouterGroup) {
	group := router.Group("/models/providers",
		route.authService.AdminUserAuthMiddleware(),
		route.authService.RegisteredUserMiddleware(),
		route.authService.OrganizationMemberRoleMiddleware(auth.OrganizationMemberRuleOwnerOnly),
	)
	group.POST("", route.registerProvider)
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

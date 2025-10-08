package modelroute

import (
	"net/http"
	"sort"
	"strings"

	"github.com/gin-gonic/gin"
	"menlo.ai/jan-api-gateway/app/domain/auth"
	domainmodel "menlo.ai/jan-api-gateway/app/domain/model"
	"menlo.ai/jan-api-gateway/app/domain/project"
	"menlo.ai/jan-api-gateway/app/utils/ptr"
)

type ProvidersAPI struct {
	authService      *auth.AuthService
	projectService   *project.ProjectService
	providerRegistry *domainmodel.ProviderRegistryService
}

func NewProvidersAPI(authService *auth.AuthService, projectService *project.ProjectService, providerRegistry *domainmodel.ProviderRegistryService) *ProvidersAPI {
	return &ProvidersAPI{
		authService:      authService,
		projectService:   projectService,
		providerRegistry: providerRegistry,
	}
}

func (api *ProvidersAPI) RegisterRouter(router *gin.RouterGroup) {
	group := router.Group("/models/providers",
		api.authService.AppUserAuthMiddleware(),
		api.authService.RegisteredUserMiddleware(),
	)
	group.GET("", api.listProviders)
}

type providerSummary struct {
	ID        string            `json:"id"`
	Slug      string            `json:"slug"`
	Name      string            `json:"name"`
	Vendor    string            `json:"vendor"`
	BaseURL   string            `json:"base_url"`
	Active    bool              `json:"active"`
	Metadata  map[string]string `json:"metadata,omitempty"`
	Scope     string            `json:"scope"`
	ProjectID *string           `json:"project_id,omitempty"`
}

type providersListResponse struct {
	Object string            `json:"object"`
	Data   []providerSummary `json:"data"`
}

func (api *ProvidersAPI) listProviders(reqCtx *gin.Context) {
	_, projectPublicIDs, providers, ok := ResolveAccessibleProviders(reqCtx, api.authService, api.projectService, api.providerRegistry)
	if !ok {
		return
	}

	resp := providersListResponse{
		Object: "list",
		Data:   make([]providerSummary, 0, len(providers)),
	}

	scopeOrder := map[string]int{
		"jan":          0,
		"organization": 1,
		"project":      2,
	}

	for _, provider := range providers {
		scope := "organization"
		var projectID *string
		if provider.ProjectID != nil {
			if publicID, exists := projectPublicIDs[*provider.ProjectID]; exists {
				projectID = ptr.ToString(publicID)
			}
			scope = "project"
		} else if provider.OrganizationID == nil {
			scope = "jan"
		}

		resp.Data = append(resp.Data, providerSummary{
			ID:        provider.PublicID,
			Slug:      provider.Slug,
			Name:      provider.DisplayName,
			Vendor:    strings.ToLower(string(provider.Kind)),
			BaseURL:   provider.BaseURL,
			Active:    provider.Active,
			Metadata:  provider.Metadata,
			Scope:     scope,
			ProjectID: projectID,
		})
	}

	getScopeOrder := func(scope string) int {
		if order, exists := scopeOrder[scope]; exists {
			return order
		}
		return len(scopeOrder)
	}

	sort.Slice(resp.Data, func(i, j int) bool {
		if resp.Data[i].Scope == resp.Data[j].Scope {
			return resp.Data[i].Name < resp.Data[j].Name
		}
		return getScopeOrder(resp.Data[i].Scope) < getScopeOrder(resp.Data[j].Scope)
	})

	reqCtx.JSON(http.StatusOK, resp)
}

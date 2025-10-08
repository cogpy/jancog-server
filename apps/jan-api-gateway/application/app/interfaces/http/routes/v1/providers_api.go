package v1

import (
	"net/http"
	"sort"
	"strings"

	"github.com/gin-gonic/gin"
	"menlo.ai/jan-api-gateway/app/domain/auth"
	domainmodel "menlo.ai/jan-api-gateway/app/domain/model"
	"menlo.ai/jan-api-gateway/app/domain/organization"
	"menlo.ai/jan-api-gateway/app/domain/project"
	"menlo.ai/jan-api-gateway/app/interfaces/http/responses"
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
	ctx := reqCtx.Request.Context()
	user, ok := auth.GetUserFromContext(reqCtx)
	if !ok || user == nil {
		reqCtx.AbortWithStatusJSON(http.StatusUnauthorized, responses.ErrorResponse{
			Code:  "b1ef40e7-9db9-477d-bb59-f3783585195d",
			Error: "user not found",
		})
		return
	}

	orgID := organization.DEFAULT_ORGANIZATION.ID
	orgIDPtr := ptr.ToUint(orgID)
	memberID := user.ID
	projects, err := api.projectService.Find(ctx, project.ProjectFilter{
		OrganizationID: orgIDPtr,
		MemberID:       &memberID,
	}, nil)
	if err != nil {
		reqCtx.AbortWithStatusJSON(http.StatusInternalServerError, responses.ErrorResponse{
			Code:          "d22f5fb5-7d09-4f61-8180-803f21722200",
			ErrorInstance: err,
		})
		return
	}

	projectIDs := make([]uint, 0, len(projects))
	projectPublicIDs := make(map[uint]string, len(projects))
	for _, proj := range projects {
		projectIDs = append(projectIDs, proj.ID)
		projectPublicIDs[proj.ID] = proj.PublicID
	}

	providers, err := api.providerRegistry.ListAccessibleProviders(ctx, orgID, projectIDs)
	if err != nil {
		reqCtx.AbortWithStatusJSON(http.StatusInternalServerError, responses.ErrorResponse{
			Code:          "7c88a4d8-d244-4f0d-8199-9851bc9f2df7",
			ErrorInstance: err,
		})
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
		} else if provider.OrganizationID != nil {
			if organization.DEFAULT_ORGANIZATION != nil &&
				*provider.OrganizationID == organization.DEFAULT_ORGANIZATION.ID &&
				organization.DEFAULT_ORGANIZATION.ID != orgID {
				scope = "jan"
			}
		} else {
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

package modelroute

import (
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"menlo.ai/jan-api-gateway/app/domain/auth"
	domainmodel "menlo.ai/jan-api-gateway/app/domain/model"
	"menlo.ai/jan-api-gateway/app/domain/organization"
	"menlo.ai/jan-api-gateway/app/domain/project"
	"menlo.ai/jan-api-gateway/app/interfaces/http/responses"
	"menlo.ai/jan-api-gateway/app/utils/ptr"
)

func ResolveAccessibleProviders(
	reqCtx *gin.Context,
	authService *auth.AuthService,
	projectService *project.ProjectService,
	providerRegistry *domainmodel.ProviderRegistryService,
) (uint, map[uint]string, []*domainmodel.Provider, bool) {
	ctx := reqCtx.Request.Context()
	user, ok := auth.GetUserFromContext(reqCtx)
	if !ok || user == nil {
		reqCtx.AbortWithStatusJSON(http.StatusUnauthorized, responses.ErrorResponse{
			Code:  "b1ef40e7-9db9-477d-bb59-f3783585195d",
			Error: "user not found",
		})
		return 0, nil, nil, false
	}

	orgID := organization.DEFAULT_ORGANIZATION.ID
	orgIDPtr := ptr.ToUint(orgID)
	memberID := user.ID
	projects, err := projectService.Find(ctx, project.ProjectFilter{
		OrganizationID: orgIDPtr,
		MemberID:       &memberID,
	}, nil)
	if err != nil {
		reqCtx.AbortWithStatusJSON(http.StatusInternalServerError, responses.ErrorResponse{
			Code:          "d22f5fb5-7d09-4f61-8180-803f21722200",
			ErrorInstance: err,
		})
		return 0, nil, nil, false
	}

	projectIDs := make([]uint, 0, len(projects))
	projectPublicIDs := make(map[uint]string, len(projects))
	for _, proj := range projects {
		projectIDs = append(projectIDs, proj.ID)
		projectPublicIDs[proj.ID] = proj.PublicID
	}

	providers, err := providerRegistry.ListAccessibleProviders(ctx, orgID, projectIDs)
	if err != nil {
		reqCtx.AbortWithStatusJSON(http.StatusInternalServerError, responses.ErrorResponse{
			Code:          "7c88a4d8-d244-4f0d-8199-9851bc9f2df7",
			ErrorInstance: err,
		})
		return 0, nil, nil, false
	}

	return orgID, projectPublicIDs, providers, true
}

type Model struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int    `json:"created"`
	OwnedBy string `json:"owned_by"`
}

type ModelsResponse struct {
	Object string  `json:"object"`
	Data   []Model `json:"data"`
}

type ModelWithProvider struct {
	ID             string `json:"id"`
	Object         string `json:"object"`
	ProviderID     string `json:"provider_id"`
	ProviderType   string `json:"provider_type"`
	ProviderVendor string `json:"provider_vendor"`
	ProviderName   string `json:"provider_name"`
}

type ModelsWithProviderResponse struct {
	Object string              `json:"object"`
	Data   []ModelWithProvider `json:"data"`
}

func BuildModelsWithProvider(
	providerModels []*domainmodel.ProviderModel,
	providerByID map[uint]*domainmodel.Provider,
) []ModelWithProvider {
	items := make([]ModelWithProvider, 0, len(providerModels))

	for _, pm := range providerModels {
		if pm == nil {
			continue
		}
		provider := providerByID[pm.ProviderID]
		if provider == nil {
			continue
		}
		scope := providerScope(provider)
		items = append(items, ModelWithProvider{
			ID:             pm.ModelKey,
			Object:         "model",
			ProviderID:     provider.PublicID,
			ProviderType:   scope,
			ProviderVendor: strings.ToLower(string(provider.Kind)),
			ProviderName:   provider.DisplayName,
		})
	}

	sort.SliceStable(items, func(i, j int) bool {
		pi := providerTypePriority(items[i].ProviderType)
		pj := providerTypePriority(items[j].ProviderType)
		if pi == pj {
			return items[i].ID < items[j].ID
		}
		return pi > pj
	})

	return items
}

func MergeModels(
	providerModels []*domainmodel.ProviderModel,
	providerByID map[uint]*domainmodel.Provider,
) []Model {
	result := map[string]Model{}
	priority := map[string]int{}

	for _, pm := range providerModels {
		if pm == nil {
			continue
		}
		provider := providerByID[pm.ProviderID]
		if provider == nil {
			continue
		}
		id := pm.ModelKey
		p := providerPriority(provider)
		if existingPriority, ok := priority[id]; ok && existingPriority >= p {
			continue
		}
		created := pm.UpdatedAt.Unix()
		if created == 0 {
			created = time.Now().Unix()
		}
		result[id] = Model{
			ID:      id,
			Object:  "model",
			Created: int(created),
			OwnedBy: provider.DisplayName,
		}
		priority[id] = p
	}

	list := make([]Model, 0, len(result))
	for _, model := range result {
		list = append(list, model)
	}

	sort.Slice(list, func(i, j int) bool {
		pi := priority[list[i].ID]
		pj := priority[list[j].ID]
		if pi == pj {
			return list[i].ID < list[j].ID
		}
		return pi > pj
	})

	return list
}

func providerScope(provider *domainmodel.Provider) string {
	if provider.ProjectID != nil {
		return "project"
	}
	return "organization"
}

func providerPriority(provider *domainmodel.Provider) int {
	if provider.ProjectID != nil {
		return 2
	}
	return 1
}

func providerTypePriority(scope string) int {
	switch scope {
	case "project":
		return 2
	case "organization":
		return 1
	default:
		return 0
	}
}

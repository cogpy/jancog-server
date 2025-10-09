package model

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"strings"
	"time"

	"gorm.io/gorm"
	"menlo.ai/jan-api-gateway/app/domain/common"
	"menlo.ai/jan-api-gateway/app/domain/organization"
	"menlo.ai/jan-api-gateway/app/domain/query"
	"menlo.ai/jan-api-gateway/app/utils/crypto"
	chatclient "menlo.ai/jan-api-gateway/app/utils/httpclients/chat"
	"menlo.ai/jan-api-gateway/app/utils/idgen"
	"menlo.ai/jan-api-gateway/app/utils/ptr"
	environment_variables "menlo.ai/jan-api-gateway/config/environment_variables"
)

type ProviderRegistryService struct {
	providerRepo         ProviderRepository
	providerModelService *ProviderModelService
	modelCatalogService  *ModelCatalogService
}

func NewProviderRegistryService(
	providerRepo ProviderRepository,
	providerModelService *ProviderModelService,
	modelCatalogService *ModelCatalogService,
) *ProviderRegistryService {
	return &ProviderRegistryService{
		providerRepo:         providerRepo,
		providerModelService: providerModelService,
		modelCatalogService:  modelCatalogService,
	}
}

type RegisterProviderInput struct {
	OrganizationID uint
	ProjectID      uint
	Name           string
	Vendor         string
	BaseURL        string
	APIKey         string
	Metadata       map[string]string
	Active         bool
}

type UpdateProviderInput struct {
	Name     *string
	BaseURL  *string
	APIKey   *string
	Metadata *map[string]string
	Active   *bool
}

type ProviderModelSyncResult struct {
	ProviderModel *ProviderModel
	Catalog       *ModelCatalog
}

type ProviderRegistrationResult struct {
	Provider *Provider
	Models   []ProviderModelSyncResult
}

func (s *ProviderRegistryService) RegisterProvider(ctx context.Context, input RegisterProviderInput) (*ProviderRegistrationResult, *common.Error) {
	name := strings.TrimSpace(input.Name)
	if name == "" {
		return nil, common.NewErrorWithMessage("provider name is required", "64f1d0d7-4a41-49e9-a4f5-61226c0b83c5")
	}

	baseURL := strings.TrimSpace(input.BaseURL)
	if baseURL == "" {
		return nil, common.NewErrorWithMessage("base_url is required", "9f0f7d62-4bbd-4d61-980e-dfc4d67a45f1")
	}
	if _, err := url.ParseRequestURI(baseURL); err != nil {
		return nil, common.NewError(err, "6c04d2f8-c39a-41a4-8d4a-0c2787b6ee2f")
	}

	kind := providerKindFromVendor(input.Vendor)

	orgIDValue := organization.DEFAULT_ORGANIZATION.ID
	if input.OrganizationID != 0 {
		orgIDValue = input.OrganizationID
	}
	organizationID := ptr.ToUint(orgIDValue)
	var projectID *uint
	if input.ProjectID != 0 {
		projectID = ptr.ToUint(input.ProjectID)
	}

	if kind != ProviderCustom {
		filter := ProviderFilter{Kind: &kind}
		filter.OrganizationID = organizationID
		if projectID != nil {
			filter.ProjectID = projectID
		} else {
			filter.WithoutProject = ptr.ToBool(true)
		}
		count, err := s.providerRepo.Count(ctx, filter)
		if err != nil {
			return nil, common.NewError(err, "5dc6de3c-d6df-410c-9329-48a306d0e4f7")
		}
		if count > 0 {
			return nil, common.NewErrorWithMessage("provider kind already exists", "323d2e23-4a8a-4f89-b090-4d49a0b0ca12")
		}
	}

	slug, err := s.generateUniqueSlug(ctx, slugCandidate(kind, name))
	if err != nil {
		return nil, common.NewError(err, "6df1386c-5aa0-4105-9366-74ad8637bd1a")
	}

	publicID, err := idgen.GenerateSecureID("prov", 24)
	if err != nil {
		return nil, common.NewError(err, "2d3d6c9a-5f36-4de2-8f5f-77f8401d5dd4")
	}

	plainAPIKey := strings.TrimSpace(input.APIKey)
	apiKeyHint := apiKeyHint(plainAPIKey)
	var encryptedAPIKey string
	if plainAPIKey != "" {
		secret := strings.TrimSpace(environment_variables.EnvironmentVariables.MODEL_PROVIDER_SECRET)
		if secret == "" {
			return nil, common.NewErrorWithMessage("model provider secret is not configured", "2f2a5cf4-5f2d-49ca-9e60-dfb09efc3a9e")
		}
		cipher, err := crypto.EncryptString(secret, plainAPIKey)
		if err != nil {
			return nil, common.NewError(err, "5d0d8f02-bf6f-4e1f-9f04-2a4dd21f4c81")
		}
		encryptedAPIKey = cipher
	}

	metadata := sanitizeMetadata(input.Metadata)

	provider := &Provider{
		PublicID:        publicID,
		Slug:            slug,
		OrganizationID:  organizationID,
		ProjectID:       projectID,
		DisplayName:     name,
		Kind:            kind,
		BaseURL:         normalizeURL(baseURL),
		EncryptedAPIKey: encryptedAPIKey,
		APIKeyHint:      apiKeyHint,
		IsModerated:     false,
		Active:          input.Active,
		Metadata:        metadata,
	}

	if err := s.providerRepo.Create(ctx, provider); err != nil {
		return nil, common.NewError(err, "5c1db208-0f8c-4c2b-90d9-5112e9cf2a47")
	}

	return &ProviderRegistrationResult{
		Provider: provider,
		Models:   []ProviderModelSyncResult{},
	}, nil
}

func providerKindFromVendor(vendor string) ProviderKind {
	switch strings.ToLower(strings.TrimSpace(vendor)) {
	case "jan":
		return ProviderJan
	case "openrouter":
		return ProviderOpenRouter
	case "openai":
		return ProviderOpenAI
	case "anthropic":
		return ProviderAnthropic
	case "gemini", "google", "googleai":
		return ProviderGemini
	case "mistral":
		return ProviderMistral
	case "groq":
		return ProviderGroq
	case "cohere":
		return ProviderCohere
	case "ollama":
		return ProviderOllama
	case "replicate":
		return ProviderReplicate
	case "azure_openai", "azure-openai":
		return ProviderAzureOpenAI
	case "aws_bedrock", "bedrock":
		return ProviderAWSBedrock
	case "perplexity":
		return ProviderPerplexity
	case "togetherai", "together":
		return ProviderTogetherAI
	case "huggingface":
		return ProviderHuggingFace
	case "vercel_ai", "vercel-ai", "vercel":
		return ProviderVercelAI
	case "deepinfra":
		return ProviderDeepInfra
	default:
		return ProviderCustom
	}
}

func (s *ProviderRegistryService) generateUniqueSlug(ctx context.Context, base string) (string, error) {
	candidate := slugify(base)
	if candidate == "" {
		candidate = "provider"
	}
	slug := candidate
	counter := 1
	for {
		filter := ProviderFilter{Slug: &slug}
		result, err := s.providerRepo.FindByFilter(ctx, filter, &query.Pagination{Limit: ptr.ToInt(1)})
		if err != nil {
			return "", err
		}
		if len(result) == 0 {
			return slug, nil
		}
		counter++
		slug = fmt.Sprintf("%s-%d", candidate, counter)
	}
}

func slugCandidate(kind ProviderKind, name string) string {
	return fmt.Sprintf("%s-%s", string(kind), name)
}

var slugRegex = regexp.MustCompile(`[^a-z0-9]+`)

func apiKeyHint(apiKey string) *string {
	key := strings.TrimSpace(apiKey)
	if len(key) < 4 {
		return nil
	}
	hint := key[len(key)-4:]
	return ptr.ToString(hint)
}

func (s *ProviderRegistryService) FindByPublicID(ctx context.Context, publicID string) (*Provider, *common.Error) {
	provider, err := s.providerRepo.FindByPublicID(ctx, publicID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, common.NewErrorWithMessage("provider not found", "d16271bf-54f5-4b25-bbd2-2353f1d5265c")
		}
		return nil, common.NewError(err, "1fcd6ba6-2c8e-4cca-bef7-799a1cf1c5d2")
	}
	return provider, nil
}

func (s *ProviderRegistryService) UpdateProvider(ctx context.Context, provider *Provider, input UpdateProviderInput) (*Provider, *common.Error) {
	if input.Name != nil {
		name := strings.TrimSpace(*input.Name)
		if name == "" {
			return nil, common.NewErrorWithMessage("provider name is required", "f65f5ec0-d9de-42da-8ae8-91f7f16c470a")
		}
		provider.DisplayName = name
	}
	if input.BaseURL != nil {
		baseURL := strings.TrimSpace(*input.BaseURL)
		if baseURL == "" {
			return nil, common.NewErrorWithMessage("base_url is required", "6eaf9ef7-281b-45f7-9b8d-668f6d2f5d8e")
		}
		if _, err := url.ParseRequestURI(baseURL); err != nil {
			return nil, common.NewError(err, "1fbfba8e-4fa9-4e06-8132-8d6754d88d5f")
		}
		provider.BaseURL = normalizeURL(baseURL)
	}
	if input.APIKey != nil {
		key := strings.TrimSpace(*input.APIKey)
		if key == "" {
			provider.EncryptedAPIKey = ""
			provider.APIKeyHint = nil
		} else {
			secret := strings.TrimSpace(environment_variables.EnvironmentVariables.MODEL_PROVIDER_SECRET)
			if secret == "" {
				return nil, common.NewErrorWithMessage("model provider secret is not configured", "ae950cb5-2f5a-4415-bc15-eec48c92610a")
			}
			cipher, err := crypto.EncryptString(secret, key)
			if err != nil {
				return nil, common.NewError(err, "b5bd5d1c-7811-4dd3-9f3c-43f0cb14e1f4")
			}
			provider.EncryptedAPIKey = cipher
			provider.APIKeyHint = apiKeyHint(key)
		}
	}
	if input.Metadata != nil {
		provider.Metadata = sanitizeMetadata(*input.Metadata)
	}
	if input.Active != nil {
		provider.Active = *input.Active
	}
	if err := s.providerRepo.Update(ctx, provider); err != nil {
		return nil, common.NewError(err, "3f3a055d-a4d7-4dd2-8795-2b5e9b6d7677")
	}
	return provider, nil
}

// ListAccessibleProviders returns providers accessible to the caller ordered by priority:
// project-scoped providers first, followed by organization-level and finally global providers.
func (s *ProviderRegistryService) ListAccessibleProviders(ctx context.Context, organizationID uint, projectIDs []uint) ([]*Provider, error) {
	result := []*Provider{}
	seen := map[uint]struct{}{}
	appendUnique := func(items []*Provider) {
		for _, provider := range items {
			if provider == nil {
				continue
			}
			if _, exists := seen[provider.ID]; exists {
				continue
			}
			seen[provider.ID] = struct{}{}
			result = append(result, provider)
		}
	}
	orgID := ptr.ToUint(organizationID)
	if len(projectIDs) > 0 {
		ids := projectIDs
		projectProviders, err := s.providerRepo.FindByFilter(ctx, ProviderFilter{
			OrganizationID: orgID,
			ProjectIDs:     &ids,
		}, nil)
		if err != nil {
			return nil, err
		}
		appendUnique(projectProviders)
	}
	orgProviders, err := s.providerRepo.FindByFilter(ctx, ProviderFilter{
		OrganizationID: orgID,
		WithoutProject: ptr.ToBool(true),
	}, nil)
	if err != nil {
		return nil, err
	}
	appendUnique(orgProviders)
	if organization.DEFAULT_ORGANIZATION != nil {
		globalProviders, err := s.providerRepo.FindByFilter(ctx, ProviderFilter{
			OrganizationID: ptr.ToUint(organization.DEFAULT_ORGANIZATION.ID),
			WithoutProject: ptr.ToBool(true),
		}, nil)
		if err != nil {
			return nil, err
		}
		appendUnique(globalProviders)
	}
	return result, nil
}

func (s *ProviderRegistryService) ListProviderModels(ctx context.Context, providerIDs []uint) ([]*ProviderModel, error) {
	return s.providerModelService.ListActiveByProviderIDs(ctx, providerIDs)
}

func (s *ProviderRegistryService) SyncProviderModels(ctx context.Context, provider *Provider, models []chatclient.Model) ([]ProviderModelSyncResult, *common.Error) {
	results := make([]ProviderModelSyncResult, 0, len(models))
	for _, model := range models {
		catalog, err := s.modelCatalogService.UpsertCatalog(ctx, provider.Kind, model)
		if err != nil {
			return nil, err
		}
		providerModel, err := s.providerModelService.UpsertProviderModel(ctx, provider, catalog, model)
		if err != nil {
			return nil, err
		}
		results = append(results, ProviderModelSyncResult{
			ProviderModel: providerModel,
			Catalog:       catalog,
		})
	}

	now := time.Now().UTC()
	provider.LastSyncedAt = &now
	if err := s.providerRepo.Update(ctx, provider); err != nil {
		return nil, common.NewError(err, "7fce47f4-67dd-47a3-93d6-3569b9d6d4f3")
	}

	return results, nil
}

func (s *ProviderRegistryService) GetProviderForModel(ctx context.Context, modelKey string, organizationID uint, projectIDs []uint) (*Provider, error) {
	if strings.TrimSpace(modelKey) == "" {
		return nil, errors.New("model key is required")
	}

	providers, err := s.ListAccessibleProviders(ctx, organizationID, projectIDs)
	if err != nil {
		return nil, err
	}

	if len(providers) == 0 {
		return nil, errors.New("no accessible providers found")
	}

	providerIDs := make([]uint, 0, len(providers))
	for _, provider := range providers {
		if provider == nil {
			continue
		}
		providerIDs = append(providerIDs, provider.ID)
	}

	if len(providerIDs) == 0 {
		return nil, errors.New("no accessible providers found")
	}

	providerModels, err := s.providerModelService.FindActiveByProviderIDsAndKey(ctx, providerIDs, modelKey)
	if err != nil {
		return nil, err
	}
	if len(providerModels) == 0 {
		return nil, fmt.Errorf("model '%s' not found in accessible providers", modelKey)
	}

	hasModel := make(map[uint]struct{}, len(providerModels))
	for _, pm := range providerModels {
		hasModel[pm.ProviderID] = struct{}{}
	}

	for _, provider := range providers {
		if provider == nil {
			continue
		}
		if _, ok := hasModel[provider.ID]; ok {
			return provider, nil
		}
	}

	return nil, fmt.Errorf("no valid provider found for model '%s'", modelKey)
}

func sanitizeMetadata(metadata map[string]string) map[string]string {
	if len(metadata) == 0 {
		return nil
	}
	result := make(map[string]string, len(metadata))
	for key, value := range metadata {
		k := strings.TrimSpace(key)
		v := strings.TrimSpace(value)
		if k == "" || v == "" {
			continue
		}
		result[k] = v
	}
	if len(result) == 0 {
		return nil
	}
	return result
}

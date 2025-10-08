package model

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"strings"
	"time"

	decimal "github.com/shopspring/decimal"
	"menlo.ai/jan-api-gateway/app/domain/common"
	"menlo.ai/jan-api-gateway/app/domain/query"
	"menlo.ai/jan-api-gateway/app/utils/crypto"
	"menlo.ai/jan-api-gateway/app/utils/httpclients"
	chatclient "menlo.ai/jan-api-gateway/app/utils/httpclients/chat"
	"menlo.ai/jan-api-gateway/app/utils/idgen"
	"menlo.ai/jan-api-gateway/app/utils/ptr"
	environment_variables "menlo.ai/jan-api-gateway/config/environment_variables"
)

type ProviderRegistryService struct {
	providerRepo      ProviderRepository
	providerModelRepo ProviderModelRepository
	modelCatalogRepo  ModelCatalogRepository
}

func NewProviderRegistryService(
	providerRepo ProviderRepository,
	providerModelRepo ProviderModelRepository,
	modelCatalogRepo ModelCatalogRepository,
) *ProviderRegistryService {
	return &ProviderRegistryService{
		providerRepo:      providerRepo,
		providerModelRepo: providerModelRepo,
		modelCatalogRepo:  modelCatalogRepo,
	}
}

type RegisterProviderInput struct {
	OrganizationID uint
	Name           string
	Vendor         string
	BaseURL        string
	APIKey         string
	Metadata       map[string]string
	Active         bool
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

	var organizationID *uint
	if input.OrganizationID != 0 {
		organizationID = ptr.ToUint(input.OrganizationID)
	}

	if kind != ProviderCustom {
		filter := ProviderFilter{Kind: &kind}
		if organizationID != nil {
			filter.OrganizationID = organizationID
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

	var metadata map[string]string
	if len(input.Metadata) > 0 {
		metadata = make(map[string]string, len(input.Metadata))
		for k, v := range input.Metadata {
			metadata[strings.TrimSpace(k)] = strings.TrimSpace(v)
		}
	}

	provider := &Provider{
		PublicID:        publicID,
		Slug:            slug,
		OrganizationID:  organizationID,
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

	models, err := s.fetchModels(ctx, provider.BaseURL, plainAPIKey)
	if err != nil {
		return nil, common.NewError(err, "cbe9fb03-a434-4d57-8a59-7b1e6830f9e5")
	}

	syncResults, syncErr := s.syncModels(ctx, provider, models)
	if syncErr != nil {
		return nil, syncErr
	}

	now := time.Now().UTC()
	provider.LastSyncedAt = &now
	if err := s.providerRepo.Update(ctx, provider); err != nil {
		return nil, common.NewError(err, "7fce47f4-67dd-47a3-93d6-3569b9d6d4f3")
	}

	return &ProviderRegistrationResult{
		Provider: provider,
		Models:   syncResults,
	}, nil
}

func (s *ProviderRegistryService) fetchModels(ctx context.Context, baseURL, apiKey string) ([]chatclient.Model, error) {
	client := httpclients.NewClient("ProviderModelDiscovery")
	client.SetBaseURL(normalizeURL(baseURL))
	if strings.TrimSpace(apiKey) != "" {
		client.SetHeader("Authorization", fmt.Sprintf("Bearer %s", strings.TrimSpace(apiKey)))
	}
	client.SetHeader("Content-Type", "application/json")

	modelClient := chatclient.NewChatModelClient(client, "provider_discovery", baseURL)
	resp, err := modelClient.ListModels(ctx)
	if err != nil {
		return nil, err
	}
	return resp.Data, nil
}

func (s *ProviderRegistryService) syncModels(ctx context.Context, provider *Provider, models []chatclient.Model) ([]ProviderModelSyncResult, *common.Error) {
	results := make([]ProviderModelSyncResult, 0, len(models))
	for _, model := range models {
		catalog, err := s.upsertCatalog(ctx, provider.Kind, model)
		if err != nil {
			return nil, err
		}
		providerModel, err := s.upsertProviderModel(ctx, provider, catalog, model)
		if err != nil {
			return nil, err
		}
		results = append(results, ProviderModelSyncResult{
			ProviderModel: providerModel,
			Catalog:       catalog,
		})
	}
	return results, nil
}

func (s *ProviderRegistryService) upsertCatalog(ctx context.Context, kind ProviderKind, model chatclient.Model) (*ModelCatalog, *common.Error) {
	publicID := catalogPublicID(model)
	existing, err := s.modelCatalogRepo.FindByPublicID(ctx, publicID)
	if err != nil {
		return nil, common.NewError(err, "35248ec0-0c17-4b73-b2ff-67955ad9b671")
	}

	catalog := buildModelCatalogFromModel(kind, model)
	catalog.PublicID = publicID
	now := time.Now().UTC()
	catalog.LastSyncedAt = &now

	if existing != nil {
		catalog.ID = existing.ID
		catalog.CreatedAt = existing.CreatedAt
		// Preserve filled/updated catalogs
		if existing.Status == ModelCatalogStatusFilled || existing.Status == ModelCatalogStatusUpdated {
			return existing, nil
		}
		if catalog.Status == ModelCatalogStatusFilled && existing.Status == ModelCatalogStatusUpdated {
			catalog.Status = existing.Status
		}
		if err := s.modelCatalogRepo.Update(ctx, catalog); err != nil {
			return nil, common.NewError(err, "9f5f9694-1a35-4cb4-b01e-0d531831df6e")
		}
		return catalog, nil
	}

	if err := s.modelCatalogRepo.Create(ctx, catalog); err != nil {
		return nil, common.NewError(err, "b3a1c6aa-0db5-4ef8-9f68-bebc56a149d9")
	}
	return catalog, nil
}

func (s *ProviderRegistryService) upsertProviderModel(ctx context.Context, provider *Provider, catalog *ModelCatalog, model chatclient.Model) (*ProviderModel, *common.Error) {
	modelKey := strings.TrimSpace(model.ID)
	if modelKey == "" {
		return nil, common.NewErrorWithMessage("model identifier missing", "1c5c6609-6df1-41b0-8fd9-2fa337eb0050")
	}

	filter := ProviderModelFilter{
		ProviderID: ptr.ToUint(provider.ID),
		ModelKey:   &modelKey,
	}
	existing, err := s.providerModelRepo.FindByFilter(ctx, filter, &query.Pagination{Limit: ptr.ToInt(1)})
	if err != nil {
		return nil, common.NewError(err, "5bcbced8-1a07-48cf-8b96-2d216af7ff58")
	}

	var catalogID *uint
	if catalog != nil {
		catalogID = &catalog.ID
	}

	if len(existing) > 0 {
		pm := existing[0]
		updateProviderModelFromRaw(pm, provider, catalogID, model)
		if err := s.providerModelRepo.Update(ctx, pm); err != nil {
			return nil, common.NewError(err, "19a79680-ae69-4b71-9be3-daa13cbbef16")
		}
		return pm, nil
	}

	publicID, err := idgen.GenerateSecureID("pmdl", 32)
	if err != nil {
		return nil, common.NewError(err, "62e9b0fb-a7f6-435c-9436-955f57843c73")
	}

	pm := buildProviderModelFromRaw(provider, catalogID, model)
	pm.PublicID = publicID
	if err := s.providerModelRepo.Create(ctx, pm); err != nil {
		return nil, common.NewError(err, "2f0d0864-d0b0-4f4c-90c5-5e4eb2c451e5")
	}
	return pm, nil
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

func slugify(input string) string {
	s := strings.ToLower(strings.TrimSpace(input))
	s = slugRegex.ReplaceAllString(s, "-")
	return strings.Trim(s, "-")
}

func normalizeURL(baseURL string) string {
	s := strings.TrimSpace(baseURL)
	s = strings.TrimRight(s, "/")
	return s
}

func providerKindFromVendor(vendor string) ProviderKind {
	switch strings.ToLower(strings.TrimSpace(vendor)) {
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

func apiKeyHint(apiKey string) *string {
	key := strings.TrimSpace(apiKey)
	if len(key) < 4 {
		return nil
	}
	hint := key[len(key)-4:]
	return ptr.ToString(hint)
}

func catalogPublicID(model chatclient.Model) string {
	if slug := slugify(model.CanonicalSlug); slug != "" {
		return slug
	}
	return slugify(model.ID)
}

func buildModelCatalogFromModel(kind ProviderKind, model chatclient.Model) *ModelCatalog {
	status := ModelCatalogStatusInit
	if kind == ProviderOpenRouter {
		status = ModelCatalogStatusFilled
	}

	var notes *string
	if desc, ok := getString(model.Raw, "description"); ok && desc != "" {
		notes = ptr.ToString(desc)
	}

	supportedParameters := SupportedParameters{
		Names:   extractStringSlice(model.Raw["supported_parameters"]),
		Default: extractDefaultParameters(model.Raw["default_parameters"]),
	}

	architecture := Architecture{}
	if archMap, ok := model.Raw["architecture"].(map[string]any); ok {
		architecture.Modality, _ = getString(archMap, "modality")
		architecture.InputModalities = extractStringSlice(archMap["input_modalities"])
		architecture.OutputModalities = extractStringSlice(archMap["output_modalities"])
		architecture.Tokenizer, _ = getString(archMap, "tokenizer")
		if instructType, ok := getString(archMap, "instruct_type"); ok && instructType != "" {
			architecture.InstructType = ptr.ToString(instructType)
		}
	}

	var isModerated *bool
	if topProvider, ok := model.Raw["top_provider"].(map[string]any); ok {
		if moderated, ok := topProvider["is_moderated"].(bool); ok {
			isModerated = ptr.ToBool(moderated)
		}
	}

	extras := copyMap(model.Raw)

	return &ModelCatalog{
		SupportedParameters: supportedParameters,
		Architecture:        architecture,
		Notes:               notes,
		IsModerated:         isModerated,
		Extras:              extras,
		Status:              status,
	}
}

func buildProviderModelFromRaw(provider *Provider, catalogID *uint, model chatclient.Model) *ProviderModel {
	pricing := extractPricing(model.Raw["pricing"])
	tokenLimits := extractTokenLimits(model.Raw)
	family := extractFamily(model.ID)
	supportsImages := containsString(extractStringSliceFromMap(model.Raw, "architecture", "input_modalities"), "image")
	supportsReasoning := containsString(extractStringSlice(model.Raw["supported_parameters"]), "include_reasoning")

	displayName := model.DisplayName
	if displayName == "" {
		displayName = model.ID
	}

	return &ProviderModel{
		ProviderID:         provider.ID,
		ModelCatalogID:     catalogID,
		ModelKey:           model.ID,
		DisplayName:        displayName,
		Pricing:            pricing,
		TokenLimits:        tokenLimits,
		Family:             family,
		SupportsImages:     supportsImages,
		SupportsEmbeddings: strings.Contains(strings.ToLower(model.ID), "embed"),
		SupportsReasoning:  supportsReasoning,
		Active:             provider.Active,
	}
}

func updateProviderModelFromRaw(pm *ProviderModel, provider *Provider, catalogID *uint, model chatclient.Model) {
	pm.ModelCatalogID = catalogID
	pm.DisplayName = model.DisplayName
	if pm.DisplayName == "" {
		pm.DisplayName = model.ID
	}
	pm.Pricing = extractPricing(model.Raw["pricing"])
	pm.TokenLimits = extractTokenLimits(model.Raw)
	pm.Family = extractFamily(model.ID)
	pm.SupportsImages = containsString(extractStringSliceFromMap(model.Raw, "architecture", "input_modalities"), "image")
	pm.SupportsEmbeddings = strings.Contains(strings.ToLower(model.ID), "embed")
	pm.SupportsReasoning = containsString(extractStringSlice(model.Raw["supported_parameters"]), "include_reasoning")
	pm.Active = provider.Active
}

func extractPricing(value any) Pricing {
	pricing := Pricing{}
	pricingMap, ok := value.(map[string]any)
	if !ok {
		return pricing
	}

	type priceMapping struct {
		Key  string
		Unit PriceUnit
	}

	mappings := []priceMapping{
		{Key: "prompt", Unit: Per1KPromptTokens},
		{Key: "completion", Unit: Per1KCompletionTokens},
		{Key: "request", Unit: PerRequest},
		{Key: "image", Unit: PerImage},
		{Key: "web_search", Unit: PerWebSearch},
		{Key: "internal_reasoning", Unit: PerInternalReasoning},
	}

	for _, mapping := range mappings {
		if amount, ok := pricingMap[mapping.Key]; ok {
			if micro, err := microUSDFromAny(amount); err == nil {
				pricing.Lines = append(pricing.Lines, PriceLine{
					Unit:     mapping.Unit,
					Amount:   micro,
					Currency: "USD",
				})
			}
		}
	}

	return pricing
}

func microUSDFromAny(value any) (MicroUSD, error) {
	switch v := value.(type) {
	case string:
		if strings.TrimSpace(v) == "" {
			return 0, errors.New("empty string")
		}
		d, err := decimal.NewFromString(v)
		if err != nil {
			return 0, err
		}
		return decimalToMicroUSD(d), nil
	case float64:
		return decimalToMicroUSD(decimal.NewFromFloat(v)), nil
	case float32:
		return decimalToMicroUSD(decimal.NewFromFloat32(v)), nil
	default:
		return 0, fmt.Errorf("unsupported pricing type %T", value)
	}
}

func decimalToMicroUSD(d decimal.Decimal) MicroUSD {
	micro := d.Mul(decimal.NewFromInt(1_000_000))
	return MicroUSD(micro.IntPart())
}

func extractTokenLimits(raw map[string]any) *TokenLimits {
	var contextLength, completionLength int
	if topProvider, ok := raw["top_provider"].(map[string]any); ok {
		contextLength = intFromAny(topProvider["context_length"])
		completionLength = intFromAny(topProvider["max_completion_tokens"])
	}
	if contextLength == 0 {
		contextLength = intFromAny(raw["context_length"])
	}
	if completionLength == 0 {
		completionLength = intFromAny(raw["max_completion_tokens"])
	}
	if contextLength == 0 && completionLength == 0 {
		return nil
	}
	return &TokenLimits{
		ContextLength:       contextLength,
		MaxCompletionTokens: completionLength,
	}
}

func extractFamily(modelID string) *string {
	if modelID == "" {
		return nil
	}
	if strings.Contains(modelID, "/") {
		parts := strings.Split(modelID, "/")
		if len(parts) > 0 {
			return ptr.ToString(parts[0])
		}
	}
	return nil
}

func extractDefaultParameters(value any) map[string]*decimal.Decimal {
	result := map[string]*decimal.Decimal{}
	params, ok := value.(map[string]any)
	if !ok {
		return result
	}
	for key, raw := range params {
		if raw == nil {
			result[key] = nil
			continue
		}
		switch v := raw.(type) {
		case string:
			if strings.TrimSpace(v) == "" {
				result[key] = nil
				continue
			}
			if d, err := decimal.NewFromString(v); err == nil {
				val := d
				result[key] = &val
			}
		case float64:
			d := decimal.NewFromFloat(v)
			result[key] = &d
		case float32:
			d := decimal.NewFromFloat32(v)
			result[key] = &d
		default:
			// ignore unsupported types
		}
	}
	return result
}

func extractStringSlice(value any) []string {
	list := []string{}
	switch arr := value.(type) {
	case []any:
		for _, item := range arr {
			if str, ok := item.(string); ok {
				list = append(list, strings.TrimSpace(str))
			}
		}
	case []string:
		for _, item := range arr {
			list = append(list, strings.TrimSpace(item))
		}
	}
	return list
}

func extractStringSliceFromMap(raw map[string]any, path ...string) []string {
	current := any(raw)
	for _, key := range path {
		m, ok := current.(map[string]any)
		if !ok {
			return nil
		}
		current = m[key]
	}
	return extractStringSlice(current)
}

func getString(raw map[string]any, key string) (string, bool) {
	if raw == nil {
		return "", false
	}
	if value, ok := raw[key]; ok {
		if str, ok := value.(string); ok {
			return strings.TrimSpace(str), true
		}
	}
	return "", false
}

func intFromAny(value any) int {
	switch v := value.(type) {
	case int:
		return v
	case int64:
		return int(v)
	case float64:
		return int(v)
	case float32:
		return int(v)
	case string:
		if strings.TrimSpace(v) == "" {
			return 0
		}
		if parsed, err := decimal.NewFromString(v); err == nil {
			return int(parsed.IntPart())
		}
	}
	return 0
}

func containsString(list []string, target string) bool {
	target = strings.ToLower(target)
	for _, item := range list {
		if strings.ToLower(item) == target {
			return true
		}
	}
	return false
}

func copyMap(source map[string]any) map[string]any {
	if source == nil {
		return nil
	}
	dest := make(map[string]any, len(source))
	for k, v := range source {
		dest[k] = v
	}
	return dest
}

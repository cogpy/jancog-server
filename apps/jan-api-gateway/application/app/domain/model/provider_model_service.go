package model

import (
	"context"
	"strings"
	"time"

	"menlo.ai/jan-api-gateway/app/domain/common"
	"menlo.ai/jan-api-gateway/app/domain/query"
	chatclient "menlo.ai/jan-api-gateway/app/utils/httpclients/chat"
	"menlo.ai/jan-api-gateway/app/utils/idgen"
	"menlo.ai/jan-api-gateway/app/utils/ptr"
)

// ProviderModelService encapsulates operations that persist provider models.
type ProviderModelService struct {
	providerModelRepo ProviderModelRepository
}

func NewProviderModelService(providerModelRepo ProviderModelRepository) *ProviderModelService {
	return &ProviderModelService{
		providerModelRepo: providerModelRepo,
	}
}

func (s *ProviderModelService) ListActiveByProviderIDs(ctx context.Context, providerIDs []uint) ([]*ProviderModel, error) {
	if len(providerIDs) == 0 {
		return nil, nil
	}
	ids := providerIDs
	active := ptr.ToBool(true)
	return s.providerModelRepo.FindByFilter(ctx, ProviderModelFilter{
		ProviderIDs: &ids,
		Active:      active,
	}, nil)
}

func (s *ProviderModelService) FindActiveByProviderIDsAndKey(ctx context.Context, providerIDs []uint, modelKey string) ([]*ProviderModel, error) {
	if strings.TrimSpace(modelKey) == "" {
		return nil, nil
	}
	ids := providerIDs
	key := modelKey
	active := ptr.ToBool(true)
	return s.providerModelRepo.FindByFilter(ctx, ProviderModelFilter{
		ProviderIDs: &ids,
		ModelKey:    &key,
		Active:      active,
	}, nil)
}

func (s *ProviderModelService) UpsertProviderModel(ctx context.Context, provider *Provider, catalog *ModelCatalog, model chatclient.Model) (*ProviderModel, *common.Error) {
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
	pm.UpdatedAt = time.Now().UTC()
}

func extractPricing(value any) Pricing {
	pricing := Pricing{}
	pricingMap, ok := value.(map[string]any)
	if !ok {
		return pricing
	}

	if lines, ok := pricingMap["lines"].([]any); ok {
		for _, line := range lines {
			lineMap, ok := line.(map[string]any)
			if !ok {
				continue
			}
			unitStr, _ := getString(lineMap, "unit")
			amount, ok := floatFromAny(lineMap["amount"])
			if !ok {
				continue
			}
			pricing.Lines = append(pricing.Lines, PriceLine{
				Unit:     PriceUnit(strings.ToLower(strings.TrimSpace(unitStr))),
				Amount:   MicroUSD(int64(amount * 1_000_000)),
				Currency: "USD",
			})
		}
	}

	return pricing
}

func extractTokenLimits(raw map[string]any) *TokenLimits {
	if raw == nil {
		return nil
	}
	limits := TokenLimits{}
	if contextLen, ok := floatFromAny(raw["context_length"]); ok {
		limits.ContextLength = int(contextLen)
	}
	if maxCompletion, ok := floatFromAny(raw["max_completion_tokens"]); ok {
		limits.MaxCompletionTokens = int(maxCompletion)
	}
	if limits.ContextLength == 0 && limits.MaxCompletionTokens == 0 {
		return nil
	}
	return &limits
}

func extractFamily(modelID string) *string {
	if strings.Contains(modelID, "/") {
		parts := strings.Split(modelID, "/")
		if len(parts) > 0 {
			return ptr.ToString(strings.TrimSpace(parts[0]))
		}
	}
	return nil
}

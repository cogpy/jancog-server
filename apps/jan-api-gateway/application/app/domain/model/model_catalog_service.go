package model

import (
	"context"

	"menlo.ai/jan-api-gateway/app/domain/common"
	chatclient "menlo.ai/jan-api-gateway/app/utils/httpclients/chat"
	"menlo.ai/jan-api-gateway/app/utils/ptr"
)

type ModelCatalogService struct {
	modelCatalogRepo ModelCatalogRepository
}

func NewModelCatalogService(modelCatalogRepo ModelCatalogRepository) *ModelCatalogService {
	return &ModelCatalogService{
		modelCatalogRepo: modelCatalogRepo,
	}
}

// UpsertCatalog ensures the catalog entry for the model exists and is up to date.
func (s *ModelCatalogService) UpsertCatalog(ctx context.Context, kind ProviderKind, model chatclient.Model) (*ModelCatalog, *common.Error) {
	publicID := catalogPublicID(model)
	existing, err := s.modelCatalogRepo.FindByPublicID(ctx, publicID)
	if err != nil {
		return nil, common.NewError(err, "35248ec0-0c17-4b73-b2ff-67955ad9b671")
	}

	catalog := buildModelCatalogFromModel(kind, model)
	catalog.PublicID = publicID

	if existing != nil {
		catalog.ID = existing.ID
		catalog.CreatedAt = existing.CreatedAt
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

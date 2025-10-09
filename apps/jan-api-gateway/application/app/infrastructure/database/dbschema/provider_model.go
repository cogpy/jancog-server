package dbschema

import (
	"encoding/json"

	"gorm.io/datatypes"

	domainmodel "menlo.ai/jan-api-gateway/app/domain/model"
	"menlo.ai/jan-api-gateway/app/infrastructure/database"
)

func init() {
	database.RegisterSchemaForAutoMigrate(ProviderModel{})
}

// ProviderModel represents the provider_models table.
type ProviderModel struct {
	BaseModel
	ProviderID         uint           `gorm:"not null;index;uniqueIndex:ux_provider_model_key,priority:1"`
	PublicID           string         `gorm:"size:64;not null;uniqueIndex"`
	ModelCatalogID     *uint          `gorm:"index"`
	ModelKey           string         `gorm:"size:128;not null;uniqueIndex:ux_provider_model_key,priority:2"`
	DisplayName        string         `gorm:"size:255;not null"`
	Pricing            datatypes.JSON `gorm:"type:jsonb;not null"`
	TokenLimits        datatypes.JSON `gorm:"type:jsonb"`
	Family             *string        `gorm:"size:128"`
	SupportsImages     bool           `gorm:"not null;default:false"`
	SupportsEmbeddings bool           `gorm:"not null;default:false"`
	SupportsReasoning  bool           `gorm:"not null;default:false"`
	Active             bool           `gorm:"not null;default:true"`
}

// TableName enforces snake_case table naming.
func (ProviderModel) TableName() string {
	return "provider_models"
}

// NewSchemaProviderModel converts a domain provider model into its database representation.
func NewSchemaProviderModel(m *domainmodel.ProviderModel) (*ProviderModel, error) {

	pricingJSON, err := json.Marshal(m.Pricing)
	if err != nil {
		return nil, err
	}

	var tokenLimitsJSON datatypes.JSON
	if m.TokenLimits != nil {
		data, err := json.Marshal(m.TokenLimits)
		if err != nil {
			return nil, err
		}
		tokenLimitsJSON = datatypes.JSON(data)
	}

	return &ProviderModel{
		BaseModel: BaseModel{
			ID:        m.ID,
			CreatedAt: m.CreatedAt,
			UpdatedAt: m.UpdatedAt,
		},
		ProviderID:         m.ProviderID,
		PublicID:           m.PublicID,
		ModelCatalogID:     m.ModelCatalogID,
		ModelKey:           m.ModelKey,
		DisplayName:        m.DisplayName,
		Pricing:            datatypes.JSON(pricingJSON),
		TokenLimits:        tokenLimitsJSON,
		Family:             m.Family,
		SupportsImages:     m.SupportsImages,
		SupportsEmbeddings: m.SupportsEmbeddings,
		SupportsReasoning:  m.SupportsReasoning,
		Active:             m.Active,
	}, nil
}

// EtoD converts a database provider model into its domain representation.
func (m *ProviderModel) EtoD() (*domainmodel.ProviderModel, error) {
	var pricing domainmodel.Pricing
	if len(m.Pricing) > 0 {
		if err := json.Unmarshal(m.Pricing, &pricing); err != nil {
			return nil, err
		}
	}

	var tokenLimits *domainmodel.TokenLimits
	if len(m.TokenLimits) > 0 {
		var limits domainmodel.TokenLimits
		if err := json.Unmarshal(m.TokenLimits, &limits); err != nil {
			return nil, err
		}
		tokenLimits = &limits
	}

	return &domainmodel.ProviderModel{
		ID:                 m.ID,
		ProviderID:         m.ProviderID,
		PublicID:           m.PublicID,
		ModelCatalogID:     m.ModelCatalogID,
		ModelKey:           m.ModelKey,
		DisplayName:        m.DisplayName,
		Pricing:            pricing,
		TokenLimits:        tokenLimits,
		Family:             m.Family,
		SupportsImages:     m.SupportsImages,
		SupportsEmbeddings: m.SupportsEmbeddings,
		SupportsReasoning:  m.SupportsReasoning,
		Active:             m.Active,
		CreatedAt:          m.CreatedAt,
		UpdatedAt:          m.UpdatedAt,
	}, nil
}

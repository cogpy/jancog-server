package dbschema

import (
	"encoding/json"
	"time"

	"gorm.io/datatypes"
	domainmodel "menlo.ai/jan-api-gateway/app/domain/model"
	"menlo.ai/jan-api-gateway/app/infrastructure/database"
)

func init() {
	database.RegisterSchemaForAutoMigrate(Provider{})
}

// Provider represents the providers table in the database.
type Provider struct {
	BaseModel
	PublicID        string         `gorm:"size:64;not null;uniqueIndex"`
	Slug            string         `gorm:"size:128;not null;uniqueIndex"`
	OrganizationID  *uint          `gorm:"index"`
	ProjectID       *uint          `gorm:"index"`
	DisplayName     string         `gorm:"size:255;not null"`
	Kind            string         `gorm:"size:64;not null;index"`
	BaseURL         string         `gorm:"size:512"`
	EncryptedAPIKey string         `gorm:"type:text"`
	APIKeyHint      *string        `gorm:"size:128"`
	IsModerated     bool           `gorm:"not null;default:false"`
	Active          bool           `gorm:"not null;default:true"`
	Metadata        datatypes.JSON `gorm:"type:jsonb"`
	LastSyncedAt    *time.Time
}

// TableName enforces snake_case table naming.
func (Provider) TableName() string {
	return "providers"
}

// NewSchemaProvider converts a domain provider into its database representation.
func NewSchemaProvider(p *domainmodel.Provider) *Provider {
	var metadataJSON datatypes.JSON
	if len(p.Metadata) > 0 {
		if data, err := json.Marshal(p.Metadata); err == nil {
			metadataJSON = datatypes.JSON(data)
		}
	}

	return &Provider{
		BaseModel: BaseModel{
			ID:        p.ID,
			CreatedAt: p.CreatedAt,
			UpdatedAt: p.UpdatedAt,
		},
		PublicID:        p.PublicID,
		Slug:            p.Slug,
		OrganizationID:  p.OrganizationID,
		ProjectID:       p.ProjectID,
		DisplayName:     p.DisplayName,
		Kind:            string(p.Kind),
		BaseURL:         p.BaseURL,
		EncryptedAPIKey: p.EncryptedAPIKey,
		APIKeyHint:      p.APIKeyHint,
		IsModerated:     p.IsModerated,
		Active:          p.Active,
		Metadata:        metadataJSON,
		LastSyncedAt:    p.LastSyncedAt,
	}
}

// EtoD converts a database provider into its domain representation.
func (p *Provider) EtoD() *domainmodel.Provider {
	var metadata map[string]string
	if len(p.Metadata) > 0 {
		_ = json.Unmarshal(p.Metadata, &metadata)
	}

	return &domainmodel.Provider{
		ID:              p.ID,
		PublicID:        p.PublicID,
		Slug:            p.Slug,
		OrganizationID:  p.OrganizationID,
		ProjectID:       p.ProjectID,
		DisplayName:     p.DisplayName,
		Kind:            domainmodel.ProviderKind(p.Kind),
		BaseURL:         p.BaseURL,
		EncryptedAPIKey: p.EncryptedAPIKey,
		APIKeyHint:      p.APIKeyHint,
		IsModerated:     p.IsModerated,
		Active:          p.Active,
		Metadata:        metadata,
		LastSyncedAt:    p.LastSyncedAt,
		CreatedAt:       p.CreatedAt,
		UpdatedAt:       p.UpdatedAt,
	}
}

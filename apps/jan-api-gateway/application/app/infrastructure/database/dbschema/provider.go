package dbschema

import (
	"time"

	domainmodel "menlo.ai/jan-api-gateway/app/domain/model"
	"menlo.ai/jan-api-gateway/app/infrastructure/database"
)

func init() {
	database.RegisterSchemaForAutoMigrate(Provider{})
}

// Provider represents the providers table in the database.
type Provider struct {
	BaseModel
	PublicID        string  `gorm:"size:64;not null;uniqueIndex"`
	Slug            string  `gorm:"size:128;not null;uniqueIndex"`
	OrganizationID  *uint   `gorm:"index"`
	DisplayName     string  `gorm:"size:255;not null"`
	Kind            string  `gorm:"size:64;not null;index"`
	BaseURL         string  `gorm:"size:512"`
	EncryptedAPIKey string  `gorm:"type:text"`
	APIKeyHint      *string `gorm:"size:128"`
	IsModerated     bool    `gorm:"not null;default:false"`
	Active          bool    `gorm:"not null;default:true"`
	LastSyncedAt    *time.Time
}

// TableName enforces snake_case table naming.
func (Provider) TableName() string {
	return "providers"
}

// NewSchemaProvider converts a domain provider into its database representation.
func NewSchemaProvider(p *domainmodel.Provider) *Provider {
	return &Provider{
		BaseModel: BaseModel{
			ID:        p.ID,
			CreatedAt: p.CreatedAt,
			UpdatedAt: p.UpdatedAt,
		},
		PublicID:        p.PublicID,
		Slug:            p.Slug,
		OrganizationID:  p.OrganizationID,
		DisplayName:     p.DisplayName,
		Kind:            string(p.Kind),
		BaseURL:         p.BaseURL,
		EncryptedAPIKey: p.EncryptedAPIKey,
		APIKeyHint:      p.APIKeyHint,
		IsModerated:     p.IsModerated,
		Active:          p.Active,
		LastSyncedAt:    p.LastSyncedAt,
	}
}

// EtoD converts a database provider into its domain representation.
func (p *Provider) EtoD() *domainmodel.Provider {
	return &domainmodel.Provider{
		ID:              p.ID,
		PublicID:        p.PublicID,
		Slug:            p.Slug,
		OrganizationID:  p.OrganizationID,
		DisplayName:     p.DisplayName,
		Kind:            domainmodel.ProviderKind(p.Kind),
		BaseURL:         p.BaseURL,
		EncryptedAPIKey: p.EncryptedAPIKey,
		APIKeyHint:      p.APIKeyHint,
		IsModerated:     p.IsModerated,
		Active:          p.Active,
		LastSyncedAt:    p.LastSyncedAt,
		CreatedAt:       p.CreatedAt,
		UpdatedAt:       p.UpdatedAt,
	}
}

package dbschema

import (
	"encoding/json"

	"gorm.io/datatypes"

	domainmodel "menlo.ai/jan-api-gateway/app/domain/model"
	"menlo.ai/jan-api-gateway/app/infrastructure/database"
)

func init() {
	database.RegisterSchemaForAutoMigrate(ModelCatalog{})
}

// ModelCatalog represents the model_catalogs table.
type ModelCatalog struct {
	BaseModel
	PublicID            string         `gorm:"size:64;not null;uniqueIndex"`
	SupportedParameters datatypes.JSON `gorm:"type:jsonb;not null"`
	Architecture        datatypes.JSON `gorm:"type:jsonb;not null"`
	Tags                datatypes.JSON `gorm:"type:jsonb"`
	Notes               *string        `gorm:"type:text"`
	IsModerated         *bool
	Status              string         `gorm:"size:32;not null;default:'init'"`
	Extras              datatypes.JSON `gorm:"type:jsonb"`
}

// TableName enforces snake_case table naming.
func (ModelCatalog) TableName() string {
	return "model_catalogs"
}

// NewSchemaModelCatalog converts a domain catalog into its database representation.
func NewSchemaModelCatalog(m *domainmodel.ModelCatalog) (*ModelCatalog, error) {
	if m == nil {
		return nil, nil
	}

	supportedParametersJSON, err := json.Marshal(m.SupportedParameters)
	if err != nil {
		return nil, err
	}

	architectureJSON, err := json.Marshal(m.Architecture)
	if err != nil {
		return nil, err
	}

	var tagsJSON datatypes.JSON
	if len(m.Tags) > 0 {
		data, err := json.Marshal(m.Tags)
		if err != nil {
			return nil, err
		}
		tagsJSON = datatypes.JSON(data)
	}

	var extrasJSON datatypes.JSON
	if len(m.Extras) > 0 {
		data, err := json.Marshal(m.Extras)
		if err != nil {
			return nil, err
		}
		extrasJSON = datatypes.JSON(data)
	}

	status := string(m.Status)
	if status == "" {
		status = string(domainmodel.ModelCatalogStatusInit)
	}

	return &ModelCatalog{
		BaseModel: BaseModel{
			ID:        m.ID,
			CreatedAt: m.CreatedAt,
			UpdatedAt: m.UpdatedAt,
		},
		PublicID:            m.PublicID,
		SupportedParameters: datatypes.JSON(supportedParametersJSON),
		Architecture:        datatypes.JSON(architectureJSON),
		Tags:                tagsJSON,
		Notes:               m.Notes,
		IsModerated:         m.IsModerated,
		Status:              status,
		Extras:              extrasJSON,
	}, nil
}

// EtoD converts a database catalog into its domain representation.
func (m *ModelCatalog) EtoD() (*domainmodel.ModelCatalog, error) {

	var supportedParameters domainmodel.SupportedParameters
	if len(m.SupportedParameters) > 0 {
		if err := json.Unmarshal(m.SupportedParameters, &supportedParameters); err != nil {
			return nil, err
		}
	}

	var architecture domainmodel.Architecture
	if len(m.Architecture) > 0 {
		if err := json.Unmarshal(m.Architecture, &architecture); err != nil {
			return nil, err
		}
	}

	var tags []string
	if len(m.Tags) > 0 {
		if err := json.Unmarshal(m.Tags, &tags); err != nil {
			return nil, err
		}
	}

	var extras map[string]any
	if len(m.Extras) > 0 {
		if err := json.Unmarshal(m.Extras, &extras); err != nil {
			return nil, err
		}
	}

	return &domainmodel.ModelCatalog{
		ID:                  m.ID,
		PublicID:            m.PublicID,
		SupportedParameters: supportedParameters,
		Architecture:        architecture,
		Tags:                tags,
		Notes:               m.Notes,
		IsModerated:         m.IsModerated,
		Extras:              extras,
		Status: func() domainmodel.ModelCatalogStatus {
			status := domainmodel.ModelCatalogStatus(m.Status)
			if status == "" {
				return domainmodel.ModelCatalogStatusInit
			}
			return status
		}(),
		CreatedAt: m.CreatedAt,
		UpdatedAt: m.UpdatedAt,
	}, nil
}

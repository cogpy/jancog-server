package model

import (
	"context"
	"time"

	decimal "github.com/shopspring/decimal"
	"menlo.ai/jan-api-gateway/app/domain/query"
)

// Parameters supported/defaults for a provider or model.
type SupportedParameters struct {
	Names   []string                    `json:"names"`   // e.g., ["include_reasoning","max_tokens",...]
	Default map[string]*decimal.Decimal `json:"default"` // temperature/top_p/frequency_penalty, null allowed
}

// Architecture metadata.
type Architecture struct {
	Modality         string   `json:"modality"` // "text+image->text"
	InputModalities  []string `json:"input_modalities"`
	OutputModalities []string `json:"output_modalities"`
	Tokenizer        string   `json:"tokenizer"`     // "GPT" / "SentencePiece" / etc.
	InstructType     *string  `json:"instruct_type"` // nullable
}

type ModelCatalogStatus string

const (
	ModelCatalogStatusInit    ModelCatalogStatus = "init"
	ModelCatalogStatusFilled  ModelCatalogStatus = "filled"
	ModelCatalogStatusUpdated ModelCatalogStatus = "updated"
)

// ModelCatalog centralizes rich metadata that can be shared across providers
// (if two providers resell the same base model) or kept per-model.
type ModelCatalog struct {
	ID                  uint                `json:"id"`
	PublicID            string              `json:"public_id"`
	SupportedParameters SupportedParameters `json:"supported_parameters"`
	Architecture        Architecture        `json:"architecture"`

	// Optional free-form metadata
	Tags  []string `json:"tags,omitempty"`
	Notes *string  `json:"notes,omitempty"`

	// Moderation at the model level (overrides provider if set)
	IsModerated *bool `json:"is_moderated,omitempty"`

	// Extensibility bucket for provider/model-specific extras
	Extras map[string]any `json:"extras,omitempty"`

	Status       ModelCatalogStatus `json:"status"`
	LastSyncedAt *time.Time
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// ModelCatalogFilter defines optional conditions for querying catalog entries.
type ModelCatalogFilter struct {
	IDs              *[]uint
	IsModerated      *bool
	Status           *ModelCatalogStatus
	LastSyncedAfter  *time.Time
	LastSyncedBefore *time.Time
}

// ModelCatalogRepository abstracts persistence for catalog metadata.
type ModelCatalogRepository interface {
	Create(ctx context.Context, catalog *ModelCatalog) error
	Update(ctx context.Context, catalog *ModelCatalog) error
	DeleteByID(ctx context.Context, id uint) error
	FindByID(ctx context.Context, id uint) (*ModelCatalog, error)
	FindByFilter(ctx context.Context, filter ModelCatalogFilter, p *query.Pagination) ([]*ModelCatalog, error)
	Count(ctx context.Context, filter ModelCatalogFilter) (int64, error)
}

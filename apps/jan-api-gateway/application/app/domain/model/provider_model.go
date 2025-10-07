package model

import (
	"context"
	"time"

	"menlo.ai/jan-api-gateway/app/domain/query"
)

// Money as fixed-precision in micro-USD to avoid float errors.
// If you prefer decimals, use shopspring/decimal directly.
type MicroUSD int64

// PerUnit conveys what a price applies to.
type PriceUnit string

const (
	Per1KPromptTokens     PriceUnit = "per_1k_prompt_tokens"
	Per1KCompletionTokens PriceUnit = "per_1k_completion_tokens"
	PerRequest            PriceUnit = "per_request"
	PerImage              PriceUnit = "per_image"
	PerWebSearch          PriceUnit = "per_web_search"
	PerInternalReasoning  PriceUnit = "per_internal_reasoning"
)

// PriceLine is a single line item (e.g., prompt token price).
type PriceLine struct {
	Unit     PriceUnit `json:"unit"`
	Amount   MicroUSD  `json:"amount_micro_usd"` // e.g., 15000 -> $0.0150
	Currency string    `json:"currency"`         // "USD" (fixed if you only bill in USD)
}

// Pricing groups price lines for a model.
type Pricing struct {
	Lines []PriceLine `json:"lines"` // flexible: add/remove units without schema churn
}

// TokenLimits for context and completion.
type TokenLimits struct {
	ContextLength       int `json:"context_length"`        // e.g., 400000
	MaxCompletionTokens int `json:"max_completion_tokens"` // e.g., 128000
}

// ProviderModel describes a specific model under a provider.
type ProviderModel struct {
	ID             uint         `json:"id"`
	PublicID       string       `json:"public_id"`
	ProviderID     uint         `json:"provider_id"`
	ModelCatalogID *uint        `json:"model_catalog_id"`
	ModelKey       string       `json:"model_key"` // provider's canonical id, e.g., "gpt-4o-mini"
	DisplayName    string       `json:"display_name"`
	Pricing        Pricing      `json:"pricing"`
	TokenLimits    *TokenLimits `json:"token_limits,omitempty"` // override provider top caps
	Family         *string      `json:"family,omitempty"`       // e.g., "gpt-4o", "llama-3.1"

	// Optional denormalized flags for quick filters
	SupportsImages     bool `json:"supports_images"`
	SupportsEmbeddings bool `json:"supports_embeddings"`
	SupportsReasoning  bool `json:"supports_reasoning"`

	// Lifecycle & audit
	Active    bool      `json:"active"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// ProviderModelFilter defines optional conditions for querying provider models.
type ProviderModelFilter struct {
	IDs                *[]uint
	ProviderIDs        *[]uint
	ProviderID         *uint
	ModelKey           *string
	ModelKeys          *[]string
	Active             *bool
	SupportsImages     *bool
	SupportsEmbeddings *bool
	SupportsReasoning  *bool
}

// ProviderModelRepository abstracts persistence for provider models.
type ProviderModelRepository interface {
	Create(ctx context.Context, model *ProviderModel) error
	Update(ctx context.Context, model *ProviderModel) error
	DeleteByID(ctx context.Context, id uint) error
	FindByID(ctx context.Context, id uint) (*ProviderModel, error)
	FindByFilter(ctx context.Context, filter ProviderModelFilter, p *query.Pagination) ([]*ProviderModel, error)
	Count(ctx context.Context, filter ProviderModelFilter) (int64, error)
}

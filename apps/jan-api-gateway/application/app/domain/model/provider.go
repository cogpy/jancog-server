package model

import (
	"context"
	"time"

	"menlo.ai/jan-api-gateway/app/domain/query"
)

type ProviderKind string

const (
	ProviderOpenAI      ProviderKind = "openai"
	ProviderOpenRouter  ProviderKind = "openrouter"
	ProviderAnthropic   ProviderKind = "anthropic"
	ProviderGemini      ProviderKind = "gemini"
	ProviderMistral     ProviderKind = "mistral"
	ProviderGroq        ProviderKind = "groq"
	ProviderCohere      ProviderKind = "cohere"
	ProviderOllama      ProviderKind = "ollama"
	ProviderReplicate   ProviderKind = "replicate"
	ProviderAzureOpenAI ProviderKind = "azure_openai"
	ProviderAWSBedrock  ProviderKind = "aws_bedrock"
	ProviderPerplexity  ProviderKind = "perplexity"
	ProviderTogetherAI  ProviderKind = "togetherai"
	ProviderHuggingFace ProviderKind = "huggingface"
	ProviderVercelAI    ProviderKind = "vercel_ai"
	ProviderDeepInfra   ProviderKind = "deepinfra"
	ProviderCustom      ProviderKind = "custom" // for any customer-provided API
)

// Provider is the aggregate root.
type Provider struct {
	ID              uint   `json:"id"`
	PublicID        string `json:"public_id"`
	Slug            string `json:"slug"` // unique, lowercase handle
	OrganizationID  *uint
	DisplayName     string       `json:"display_name"`
	Kind            ProviderKind `json:"kind"`
	BaseURL         string       `json:"base_url"` // e.g., https://api.openai.com/v1
	EncryptedAPIKey string
	APIKeyHint      *string `json:"api_key_hint,omitempty"` // last4 or source name, not the secret
	IsModerated     bool    `json:"is_moderated"`           // whether provider enforces moderation upstream
	Active          bool
	LastSyncedAt    *time.Time
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

// ProviderFilter defines optional conditions for querying providers.
type ProviderFilter struct {
	IDs              *[]uint
	PublicID         *string
	Slug             *string
	OrganizationID   *uint
	Kind             *ProviderKind
	Active           *bool
	IsModerated      *bool
	LastSyncedAfter  *time.Time
	LastSyncedBefore *time.Time
}

// ProviderRepository abstracts persistence for provider aggregate roots.
type ProviderRepository interface {
	Create(ctx context.Context, provider *Provider) error
	Update(ctx context.Context, provider *Provider) error
	DeleteByID(ctx context.Context, id uint) error
	FindByID(ctx context.Context, id uint) (*Provider, error)
	FindByPublicID(ctx context.Context, publicID string) (*Provider, error)
	FindBySlug(ctx context.Context, slug string) (*Provider, error)
	FindByFilter(ctx context.Context, filter ProviderFilter, p *query.Pagination) ([]*Provider, error)
	Count(ctx context.Context, filter ProviderFilter) (int64, error)
}

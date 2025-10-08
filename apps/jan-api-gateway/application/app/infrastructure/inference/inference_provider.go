package inference

import (
	"context"
	"fmt"
	"strings"

	domainmodel "menlo.ai/jan-api-gateway/app/domain/model"
	"menlo.ai/jan-api-gateway/app/utils/crypto"
	httpclients "menlo.ai/jan-api-gateway/app/utils/httpclients"
	chatclient "menlo.ai/jan-api-gateway/app/utils/httpclients/chat"
	"menlo.ai/jan-api-gateway/config/environment_variables"
	"resty.dev/v3"
)

// InferenceProvider provides chat completion and model clients for providers
type InferenceProvider struct{}

// NewInferenceProvider creates a new inference provider instance
func NewInferenceProvider() *InferenceProvider {
	return &InferenceProvider{}
}

// GetChatCompletionClient returns a chat completion client configured for the provider
func (ip *InferenceProvider) GetChatCompletionClient(provider *domainmodel.Provider) (*chatclient.ChatCompletionClient, error) {
	client, err := ip.createRestyClient(provider)
	if err != nil {
		return nil, err
	}

	clientName := provider.DisplayName
	return chatclient.NewChatCompletionClient(client, clientName, provider.BaseURL), nil
}

// GetChatModelClient returns a chat model client configured for the provider
func (ip *InferenceProvider) GetChatModelClient(provider *domainmodel.Provider) (*chatclient.ChatModelClient, error) {
	client, err := ip.createRestyClient(provider)
	if err != nil {
		return nil, err
	}

	clientName := provider.DisplayName
	return chatclient.NewChatModelClient(client, clientName, provider.BaseURL), nil
}

// ListModels retrieves the available models for the given provider.
func (ip *InferenceProvider) ListModels(ctx context.Context, provider *domainmodel.Provider) ([]chatclient.Model, error) {
	modelClient, err := ip.GetChatModelClient(provider)
	if err != nil {
		return nil, err
	}

	resp, err := modelClient.ListModels(ctx)
	if err != nil {
		return nil, err
	}

	return resp.Data, nil
}

// createRestyClient creates a configured resty client for the provider
func (ip *InferenceProvider) createRestyClient(provider *domainmodel.Provider) (*resty.Client, error) {
	clientName := fmt.Sprintf("%sClient", provider.DisplayName)
	client := httpclients.NewClient(clientName)
	client.SetBaseURL(provider.BaseURL)

	// Set authorization header if API key exists
	if provider.EncryptedAPIKey != "" {
		apiKey, err := ip.decryptAPIKey(provider.EncryptedAPIKey)
		if err != nil {
			return nil, fmt.Errorf("failed to decrypt API key: %w", err)
		}

		if strings.TrimSpace(apiKey) != "" && strings.ToLower(apiKey) != "none" {
			client.SetHeader("Authorization", fmt.Sprintf("Bearer %s", apiKey))
		}
	}

	return client, nil
}

// decryptAPIKey decrypts the provider's encrypted API key
func (ip *InferenceProvider) decryptAPIKey(encryptedAPIKey string) (string, error) {
	if encryptedAPIKey == "" {
		return "", nil
	}

	secret := strings.TrimSpace(environment_variables.EnvironmentVariables.MODEL_PROVIDER_SECRET)
	if secret == "" {
		return "", fmt.Errorf("MODEL_PROVIDER_SECRET not configured")
	}

	plainText, err := crypto.DecryptString(secret, encryptedAPIKey)
	if err != nil {
		return "", err
	}

	return plainText, nil
}

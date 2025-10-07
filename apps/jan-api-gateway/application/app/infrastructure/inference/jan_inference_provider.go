package inference

import (
	"context"
	"io"

	openai "github.com/sashabaranov/go-openai"
	"menlo.ai/jan-api-gateway/app/domain/inference"
	httpclients "menlo.ai/jan-api-gateway/app/utils/httpclients"
	chatclient "menlo.ai/jan-api-gateway/app/utils/httpclients/chat"
	"menlo.ai/jan-api-gateway/config/environment_variables"
	"resty.dev/v3"
)

func NewJanRestyClient() *resty.Client {
	client := httpclients.NewClient("JanInferenceClient")
	client.SetBaseURL(environment_variables.EnvironmentVariables.JAN_INFERENCE_MODEL_URL)
	return client
}

func NewJanChatCompletionClient(restyClient *resty.Client) *chatclient.ChatCompletionClient {
	return chatclient.NewChatCompletionClient(restyClient, "jan inference", environment_variables.EnvironmentVariables.JAN_INFERENCE_MODEL_URL)
}

func NewJanChatModelClient(restyClient *resty.Client) *chatclient.ChatModelClient {
	return chatclient.NewChatModelClient(restyClient, "jan inference models", environment_variables.EnvironmentVariables.JAN_INFERENCE_MODEL_URL)
}

// JanInferenceProvider implements InferenceProvider using Jan Inference service
type JanInferenceProvider struct {
	chatClient  *chatclient.ChatCompletionClient
	restyClient *resty.Client
}

// NewJanInferenceProvider creates a new JanInferenceProvider
func NewJanInferenceProvider(chatClient *chatclient.ChatCompletionClient, restyClient *resty.Client) inference.InferenceProvider {
	return &JanInferenceProvider{
		chatClient:  chatClient,
		restyClient: restyClient,
	}
}

// CreateCompletion creates a non-streaming chat completion
func (p *JanInferenceProvider) CreateCompletion(ctx context.Context, apiKey string, request openai.ChatCompletionRequest) (*openai.ChatCompletionResponse, error) {
	return p.chatClient.CreateChatCompletion(ctx, apiKey, request)
}

// CreateCompletionStream creates a streaming chat completion
func (p *JanInferenceProvider) CreateCompletionStream(ctx context.Context, apiKey string, request openai.ChatCompletionRequest) (io.ReadCloser, error) {
	return p.chatClient.CreateChatCompletionStream(ctx, apiKey, request)
}

func (p *JanInferenceProvider) GetModels(ctx context.Context) (*inference.ModelsResponse, error) {
	var modelsResponse struct {
		Object string `json:"object"`
		Data   []struct {
			ID      string `json:"id"`
			Object  string `json:"object"`
			Created int    `json:"created"`
			OwnedBy string `json:"owned_by"`
		} `json:"data"`
	}

	_, err := p.restyClient.R().
		SetContext(ctx).
		SetResult(&modelsResponse).
		Get("/v1/models")
	if err != nil {
		return nil, err
	}

	models := make([]inference.Model, len(modelsResponse.Data))
	for i, model := range modelsResponse.Data {
		models[i] = inference.Model{
			ID:      model.ID,
			Object:  model.Object,
			Created: model.Created,
			OwnedBy: model.OwnedBy,
		}
	}

	return &inference.ModelsResponse{
		Object: modelsResponse.Object,
		Data:   models,
	}, nil
}

// ValidateModel checks if a model is supported
func (p *JanInferenceProvider) ValidateModel(model string) error {
	// For now, assume all models are supported by Jan Inference
	// In the future, this could check against a list of supported models
	return nil
}

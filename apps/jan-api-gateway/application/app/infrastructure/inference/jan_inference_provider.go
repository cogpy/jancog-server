package inference

import (
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

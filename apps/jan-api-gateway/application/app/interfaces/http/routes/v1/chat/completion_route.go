package chat

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
	openai "github.com/sashabaranov/go-openai"
	"menlo.ai/jan-api-gateway/app/domain/auth"
	"menlo.ai/jan-api-gateway/app/domain/common"
	"menlo.ai/jan-api-gateway/app/interfaces/http/responses"
	chatclient "menlo.ai/jan-api-gateway/app/utils/httpclients/chat"
	"menlo.ai/jan-api-gateway/app/utils/logger"
)

// CompletionAPI handles chat completion requests with streaming support by delegating to the shared chat completion client.
type CompletionAPI struct {
	chatClient *chatclient.ChatCompletionClient
}

func NewCompletionAPI(chatClient *chatclient.ChatCompletionClient, authService *auth.AuthService) *CompletionAPI {
	return &CompletionAPI{
		chatClient: chatClient,
	}
}

func (completionAPI *CompletionAPI) RegisterRouter(router *gin.RouterGroup) {
	router.POST("/completions", completionAPI.PostCompletion)
}

// PostCompletion
// @Summary Create a chat completion
// @Description Generates a model response for the given chat conversation. This is a standard chat completion API that supports both streaming and non-streaming modes without conversation persistence.
// @Description
// @Description **Streaming Mode (stream=true):**
// @Description - Returns Server-Sent Events (SSE) with real-time streaming
// @Description - Streams completion chunks directly from the inference model
// @Description - Final event contains "[DONE]" marker
// @Description
// @Description **Non-Streaming Mode (stream=false or omitted):**
// @Description - Returns single JSON response with complete completion
// @Description - Standard OpenAI ChatCompletionResponse format
// @Description
// @Description **Features:**
// @Description - Supports all OpenAI ChatCompletionRequest parameters
// @Description - User authentication required
// @Description - Direct inference model integration
// @Description - No conversation persistence (stateless)
// @Tags Chat Completions API
// @Security BearerAuth
// @Accept json
// @Produce json
// @Produce text/event-stream
// @Param request body openai.ChatCompletionRequest true "Chat completion request with streaming options"
// @Success 200 {object} openai.ChatCompletionResponse "Successful non-streaming response (when stream=false)"
// @Success 200 {string} string "Successful streaming response (when stream=true) - SSE format with data: {json} events"
// @Failure 400 {object} responses.ErrorResponse "Invalid request payload, empty messages, or inference failure"
// @Failure 401 {object} responses.ErrorResponse "Unauthorized - missing or invalid authentication"
// @Failure 500 {object} responses.ErrorResponse "Internal server error"
// @Router /v1/chat/completions [post]
func (cApi *CompletionAPI) PostCompletion(reqCtx *gin.Context) {
	var request openai.ChatCompletionRequest
	if err := reqCtx.ShouldBindJSON(&request); err != nil {
		reqCtx.AbortWithStatusJSON(http.StatusBadRequest, responses.ErrorResponse{
			Code:          "0199600b-86d3-7339-8402-8ef1c7840475",
			ErrorInstance: err,
		})
		return
	}

	if len(request.Messages) == 0 {
		reqCtx.AbortWithStatusJSON(http.StatusBadRequest, responses.ErrorResponse{
			Code:  "0199600f-2cbe-7518-be5c-9989cce59472",
			Error: "messages cannot be empty",
		})
		return
	}

	var err *common.Error
	var response *openai.ChatCompletionResponse

	if request.Stream {
		err = cApi.StreamCompletionResponse(reqCtx, "", request)
	} else {
		response, err = cApi.CallCompletionAndGetRestResponse(reqCtx.Request.Context(), "", request)
	}

	if err != nil {
		logger.GetLogger().Errorf("completion failed: %v", err)
		reqCtx.AbortWithStatusJSON(
			http.StatusBadRequest,
			responses.ErrorResponse{
				Code:          err.GetCode(),
				ErrorInstance: err.GetError(),
			})
		return
	}

	if !request.Stream {
		reqCtx.JSON(http.StatusOK, response)
	}
}

// CallCompletionAndGetRestResponse calls the shared chat client and returns a complete non-streaming response.
func (cApi *CompletionAPI) CallCompletionAndGetRestResponse(ctx context.Context, apiKey string, request openai.ChatCompletionRequest) (*openai.ChatCompletionResponse, *common.Error) {
	response, err := cApi.chatClient.CreateChatCompletion(ctx, apiKey, request)
	if err != nil {
		logger.GetLogger().Errorf("inference failed: %v", err)
		return nil, common.NewError(err, "0199600c-3b65-7618-83ca-443a583d91c9")
	}

	return response, nil
}

// StreamCompletionResponse streams SSE events directly to the client via the shared chat client.
func (cApi *CompletionAPI) StreamCompletionResponse(reqCtx *gin.Context, apiKey string, request openai.ChatCompletionRequest) *common.Error {
	if _, err := cApi.chatClient.StreamChatCompletionToContext(reqCtx, apiKey, request); err != nil {
		return common.NewError(err, "bc82d69c-685b-4556-9d1f-2a4a80ae8ca4")
	}
	return nil
}

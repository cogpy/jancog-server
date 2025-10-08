package chat

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
	openai "github.com/sashabaranov/go-openai"
	"menlo.ai/jan-api-gateway/app/domain/auth"
	"menlo.ai/jan-api-gateway/app/domain/common"
	domainmodel "menlo.ai/jan-api-gateway/app/domain/model"
	"menlo.ai/jan-api-gateway/app/domain/organization"
	"menlo.ai/jan-api-gateway/app/domain/project"
	"menlo.ai/jan-api-gateway/app/infrastructure/inference"
	"menlo.ai/jan-api-gateway/app/interfaces/http/responses"
	"menlo.ai/jan-api-gateway/app/utils/logger"
	"menlo.ai/jan-api-gateway/app/utils/ptr"
)

// CompletionAPI handles chat completion requests with streaming support by delegating to the shared chat completion client.
type CompletionAPI struct {
	inferenceProvider *inference.InferenceProvider
	authService       *auth.AuthService
	projectService    *project.ProjectService
	providerRegistry  *domainmodel.ProviderRegistryService
}

func NewCompletionAPI(
	inferenceProvider *inference.InferenceProvider,
	authService *auth.AuthService,
	projectService *project.ProjectService,
	providerRegistry *domainmodel.ProviderRegistryService,
) *CompletionAPI {
	return &CompletionAPI{
		inferenceProvider: inferenceProvider,
		authService:       authService,
		projectService:    projectService,
		providerRegistry:  providerRegistry,
	}
}

func (completionAPI *CompletionAPI) RegisterRouter(router *gin.RouterGroup) {
	router.POST("/completions", completionAPI.PostCompletion)
}

// getProviderForModel resolves the provider for a given model key based on user's access
func (cApi *CompletionAPI) getProviderForModel(reqCtx *gin.Context, modelKey string) (*domainmodel.Provider, *common.Error) {
	ctx := reqCtx.Request.Context()

	// Get user from context
	user, ok := auth.GetUserFromContext(reqCtx)
	if !ok || user == nil {
		// If no user context, fall back to the default provider
		logger.GetLogger().Warnf("No user context found, using default provider as fallback")
		return cApi.providerRegistry.DefaultProvider(), nil
	}

	// Get organization ID (default organization for all users)
	orgID := organization.DEFAULT_ORGANIZATION.ID
	orgIDPtr := ptr.ToUint(orgID)
	memberID := user.ID

	// Get user's accessible projects
	var projectIDs []uint
	projects, err := cApi.projectService.Find(ctx, project.ProjectFilter{
		OrganizationID: orgIDPtr,
		MemberID:       &memberID,
	}, nil)
	if err != nil {
		logger.GetLogger().Warnf("Failed to get user projects: %v, using default provider as fallback", err)
		return cApi.providerRegistry.DefaultProvider(), nil
	}

	for _, proj := range projects {
		projectIDs = append(projectIDs, proj.ID)
	}

	// Find provider for the model (priority: Project → Organization → Global)
	provider, usedDefault, err := cApi.providerRegistry.GetProviderForModelOrDefault(ctx, modelKey, orgID, projectIDs)
	if err != nil && usedDefault {
		logger.GetLogger().Errorf("Failed to find provider for model '%s': %v, falling back to default provider", modelKey, err)
	}

	return provider, nil
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

	// Get provider based on the requested model
	provider, providerErr := cApi.getProviderForModel(reqCtx, request.Model)
	if providerErr != nil {
		reqCtx.AbortWithStatusJSON(http.StatusBadRequest, responses.ErrorResponse{
			Code:          providerErr.GetCode(),
			ErrorInstance: providerErr.GetError(),
		})
		return
	}

	var err *common.Error
	var response *openai.ChatCompletionResponse

	if request.Stream {
		err = cApi.StreamCompletionResponse(reqCtx, provider, "", request)
	} else {
		response, err = cApi.CallCompletionAndGetRestResponse(reqCtx.Request.Context(), provider, "", request)
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
func (cApi *CompletionAPI) CallCompletionAndGetRestResponse(ctx context.Context, provider *domainmodel.Provider, apiKey string, request openai.ChatCompletionRequest) (*openai.ChatCompletionResponse, *common.Error) {
	chatClient, err := cApi.inferenceProvider.GetChatCompletionClient(provider)
	if err != nil {
		logger.GetLogger().Errorf("failed to create chat client: %v", err)
		return nil, common.NewError(err, "0199600c-3b65-7618-83ca-443a583d91c8")
	}

	response, err := chatClient.CreateChatCompletion(ctx, apiKey, request)
	if err != nil {
		logger.GetLogger().Errorf("inference failed: %v", err)
		return nil, common.NewError(err, "0199600c-3b65-7618-83ca-443a583d91c9")
	}

	return response, nil
}

// StreamCompletionResponse streams SSE events directly to the client via the shared chat client.
func (cApi *CompletionAPI) StreamCompletionResponse(reqCtx *gin.Context, provider *domainmodel.Provider, apiKey string, request openai.ChatCompletionRequest) *common.Error {
	chatClient, err := cApi.inferenceProvider.GetChatCompletionClient(provider)
	if err != nil {
		return common.NewError(err, "bc82d69c-685b-4556-9d1f-2a4a80ae8ca3")
	}

	if _, err := chatClient.StreamChatCompletionToContext(reqCtx, apiKey, request); err != nil {
		return common.NewError(err, "bc82d69c-685b-4556-9d1f-2a4a80ae8ca4")
	}
	return nil
}

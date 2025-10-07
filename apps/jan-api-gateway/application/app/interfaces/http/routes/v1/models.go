package v1

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"menlo.ai/jan-api-gateway/app/interfaces/http/responses"
	"menlo.ai/jan-api-gateway/app/utils/functional"
	chatclient "menlo.ai/jan-api-gateway/app/utils/httpclients/chat"
)

type ModelAPI struct {
	modelClient *chatclient.ChatModelClient
}

func NewModelAPI(modelClient *chatclient.ChatModelClient) *ModelAPI {
	return &ModelAPI{
		modelClient: modelClient,
	}
}

func (modelAPI *ModelAPI) RegisterRouter(router *gin.RouterGroup) {
	router.GET("models", modelAPI.GetModels)
}

// ListModels
// @Summary List available models
// @Description Retrieves a list of available models that can be used for chat completions or other tasks.
// @Tags Chat Completions API
// @Security BearerAuth
// @Accept json
// @Produce json
// @Success 200 {object} ModelsResponse "Successful response"
// @Router /v1/models [get]
func (modelAPI *ModelAPI) GetModels(reqCtx *gin.Context) {
	ctx := reqCtx.Request.Context()
	modelsResp, err := modelAPI.modelClient.ListModels(ctx)
	if err != nil {
		reqCtx.AbortWithStatusJSON(http.StatusBadGateway, responses.ErrorResponse{
			Code:          "0199600b-86d3-7339-8402-8ef1c7840475",
			ErrorInstance: err,
		})
		return
	}

	reqCtx.JSON(http.StatusOK, ModelsResponse{
		Object: "list",
		Data: functional.Map(modelsResp.Data, func(model chatclient.Model) Model {
			return Model{
				ID:      model.ID,
				Object:  model.Object,
				Created: model.Created,
				OwnedBy: model.OwnedBy,
			}
		}),
	})
}

type Model struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int    `json:"created"`
	OwnedBy string `json:"owned_by"`
}

type ModelsResponse struct {
	Object string  `json:"object"`
	Data   []Model `json:"data"`
}

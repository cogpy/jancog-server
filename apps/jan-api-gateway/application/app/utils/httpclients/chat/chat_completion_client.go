package chat

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	openai "github.com/sashabaranov/go-openai"
	"menlo.ai/jan-api-gateway/app/utils/logger"
	"resty.dev/v3"
)

const (
	requestTimeout       = 120 * time.Second
	channelBufferSize    = 100
	errorBufferSize      = 10
	dataPrefix           = "data: "
	doneMarker           = "[DONE]"
	newlineChar          = "\n"
	scannerInitialBuffer = 12 * 1024   // 12KB
	scannerMaxBuffer     = 1024 * 1024 // 1MB
)

type StreamOption func(*resty.Request)

func WithHeader(key, value string) StreamOption {
	return func(r *resty.Request) {
		if strings.TrimSpace(key) == "" {
			return
		}
		if value == "" {
			r.SetHeader(key, "")
			return
		}
		r.SetHeader(key, value)
	}
}

func WithAcceptEncodingIdentity() StreamOption {
	return WithHeader("Accept-Encoding", "identity")
}

type ChatCompletionClient struct {
	client  *resty.Client
	baseURL string
	name    string
}

type functionCallAccumulator struct {
	Name      string
	Arguments string
	Complete  bool
}

type toolCallAccumulator struct {
	ID       string
	Type     string
	Index    int
	Function struct {
		Name      string
		Arguments string
	}
	Complete bool
}

func NewChatCompletionClient(client *resty.Client, name, baseURL string) *ChatCompletionClient {
	return &ChatCompletionClient{
		client:  client,
		baseURL: normalizeBaseURL(baseURL),
		name:    name,
	}
}

func (c *ChatCompletionClient) CreateChatCompletion(ctx context.Context, apiKey string, request openai.ChatCompletionRequest) (*openai.ChatCompletionResponse, error) {
	var respBody openai.ChatCompletionResponse
	resp, err := c.prepareRequest(ctx, apiKey).
		SetBody(request).
		SetResult(&respBody).
		Post(c.endpoint("/chat/completions"))
	if err != nil {
		return nil, err
	}
	if resp.IsError() {
		return nil, c.errorFromResponse(resp, "request failed")
	}
	return &respBody, nil
}

func (c *ChatCompletionClient) CreateChatCompletionStream(ctx context.Context, apiKey string, request openai.ChatCompletionRequest, opts ...StreamOption) (io.ReadCloser, error) {
	resp, err := c.doStreamingRequest(ctx, apiKey, request, opts...)
	if err != nil {
		return nil, err
	}

	reader, writer := io.Pipe()

	go func() {
		defer func() {
			if closeErr := resp.RawResponse.Body.Close(); closeErr != nil {
				logger.GetLogger().Errorf("%s: unable to close response body: %v", c.name, closeErr)
			}
		}()

		if _, copyErr := io.Copy(writer, resp.RawResponse.Body); copyErr != nil {
			_ = writer.CloseWithError(copyErr)
			return
		}
		_ = writer.Close()
	}()

	return reader, nil
}

// StreamChatCompletionToContext streams the completion to the provided Gin context while
// accumulating the complete response, mirroring the SSE handling found in the conversation
// completion flow.
func (c *ChatCompletionClient) StreamChatCompletionToContext(reqCtx *gin.Context, apiKey string, request openai.ChatCompletionRequest, opts ...StreamOption) (*openai.ChatCompletionResponse, error) {
	if reqCtx == nil {
		return nil, fmt.Errorf("%s: streaming request failed: nil gin context", c.name)
	}

	ctx, cancel := context.WithTimeout(reqCtx.Request.Context(), requestTimeout)
	defer cancel()

	c.SetupSSEHeaders(reqCtx)

	dataChan := make(chan string, channelBufferSize)
	errChan := make(chan error, errorBufferSize)

	var wg sync.WaitGroup
	wg.Add(1)

	go c.streamResponseToChannel(ctx, apiKey, request, dataChan, errChan, &wg, opts)

	var contentBuilder strings.Builder
	var reasoningBuilder strings.Builder
	functionCallAccumulator := make(map[int]*functionCallAccumulator)
	toolCallAccumulator := make(map[int]*toolCallAccumulator)

	streamingComplete := false

	for !streamingComplete {
		select {
		case line, ok := <-dataChan:
			if !ok {
				streamingComplete = true
				break
			}

			if err := c.writeSSELine(reqCtx, line); err != nil {
				cancel()
				wg.Wait()
				return nil, fmt.Errorf("%s: unable to write SSE line: %w", c.name, err)
			}

			if data, found := strings.CutPrefix(line, dataPrefix); found {
				if data == doneMarker {
					streamingComplete = true
					cancel()
					break
				}

				contentChunk, reasoningChunk, functionCallChunk, toolCallChunk := c.processStreamChunkForChannel(data)

				if contentChunk != "" {
					contentBuilder.WriteString(contentChunk)
				}

				if reasoningChunk != "" {
					reasoningBuilder.WriteString(reasoningChunk)
				}

				if functionCallChunk != nil {
					c.handleStreamingFunctionCall(functionCallChunk, functionCallAccumulator)
				}

				if toolCallChunk != nil {
					c.handleStreamingToolCall(toolCallChunk, toolCallAccumulator)
				}
			}

		case err, ok := <-errChan:
			if ok && err != nil {
				cancel()
				wg.Wait()
				return nil, fmt.Errorf("%s: streaming error: %w", c.name, err)
			}

		case <-ctx.Done():
			wg.Wait()
			return nil, fmt.Errorf("%s: streaming context cancelled: %w", c.name, ctx.Err())

		case <-reqCtx.Request.Context().Done():
			cancel()
			wg.Wait()
			return nil, fmt.Errorf("%s: client request cancelled: %w", c.name, reqCtx.Request.Context().Err())
		}
	}

	cancel()
	wg.Wait()

	close(dataChan)
	close(errChan)

	response := c.buildCompleteResponse(
		contentBuilder.String(),
		reasoningBuilder.String(),
		functionCallAccumulator,
		toolCallAccumulator,
		request.Model,
		request,
	)

	return &response, nil
}

// SetupSSEHeaders configures the Gin context for Server-Sent Events responses.
func (c *ChatCompletionClient) SetupSSEHeaders(reqCtx *gin.Context) {
	if reqCtx == nil {
		return
	}

	reqCtx.Header("Content-Type", "text/event-stream")
	reqCtx.Header("Cache-Control", "no-cache")
	reqCtx.Header("Connection", "keep-alive")
	reqCtx.Header("Access-Control-Allow-Origin", "*")
	reqCtx.Header("Access-Control-Allow-Headers", "Cache-Control")
	reqCtx.Header("Transfer-Encoding", "chunked")
	reqCtx.Writer.WriteHeaderNow()
}

func (c *ChatCompletionClient) prepareRequest(ctx context.Context, apiKey string) *resty.Request {
	req := c.client.R().SetContext(ctx)
	req.SetHeader("Content-Type", "application/json")
	if strings.TrimSpace(apiKey) != "" {
		req.SetHeader("Authorization", fmt.Sprintf("Bearer %s", apiKey))
	}
	return req
}

func (c *ChatCompletionClient) endpoint(path string) string {
	if path == "" {
		return c.baseURL
	}
	if strings.HasPrefix(path, "http://") || strings.HasPrefix(path, "https://") {
		return path
	}
	if c.baseURL == "" {
		return path
	}
	if strings.HasPrefix(path, "/") {
		return c.baseURL + path
	}
	return c.baseURL + "/" + path
}

func (c *ChatCompletionClient) errorFromResponse(resp *resty.Response, message string) error {
	if resp == nil || resp.RawResponse == nil || resp.RawResponse.Body == nil {
		return fmt.Errorf("%s: %s with status %d", c.name, message, statusCode(resp))
	}
	defer resp.RawResponse.Body.Close()
	body, err := io.ReadAll(resp.RawResponse.Body)
	if err != nil {
		return fmt.Errorf("%s: %s with status %d", c.name, message, statusCode(resp))
	}
	trimmed := strings.TrimSpace(string(body))
	if trimmed == "" {
		return fmt.Errorf("%s: %s with status %d", c.name, message, statusCode(resp))
	}
	return fmt.Errorf("%s: %s with status %d: %s", c.name, message, statusCode(resp), trimmed)
}

func (c *ChatCompletionClient) doStreamingRequest(ctx context.Context, apiKey string, request openai.ChatCompletionRequest, opts ...StreamOption) (*resty.Response, error) {
	req := c.prepareRequest(ctx, apiKey).
		SetBody(request).
		SetDoNotParseResponse(true)

	for _, opt := range opts {
		if opt == nil {
			continue
		}
		opt(req)
	}

	if req.Header.Get("Accept-Encoding") == "" {
		req.SetHeader("Accept-Encoding", "identity")
	}

	resp, err := req.Post(c.endpoint("/chat/completions"))
	if err != nil {
		return nil, err
	}
	if resp.IsError() {
		return nil, c.errorFromResponse(resp, "streaming request failed")
	}
	if resp.RawResponse == nil || resp.RawResponse.Body == nil {
		return nil, fmt.Errorf("%s: streaming request failed: empty response body", c.name)
	}

	return resp, nil
}

func (c *ChatCompletionClient) streamResponseToChannel(ctx context.Context, apiKey string, request openai.ChatCompletionRequest, dataChan chan<- string, errChan chan<- error, wg *sync.WaitGroup, opts []StreamOption) {
	defer wg.Done()

	resp, err := c.doStreamingRequest(ctx, apiKey, request, opts...)
	if err != nil {
		c.sendAsyncError(errChan, err)
		return
	}

	defer func() {
		if closeErr := resp.RawResponse.Body.Close(); closeErr != nil {
			logger.GetLogger().Errorf("%s: unable to close response body: %v", c.name, closeErr)
		}
	}()

	scanner := bufio.NewScanner(resp.RawResponse.Body)
	scanner.Buffer(make([]byte, 0, scannerInitialBuffer), scannerMaxBuffer)

	for scanner.Scan() {
		select {
		case <-ctx.Done():
			c.sendAsyncError(errChan, ctx.Err())
			return
		default:
		}

		line := scanner.Text()

		select {
		case dataChan <- line:
		case <-ctx.Done():
			c.sendAsyncError(errChan, ctx.Err())
			return
		}
	}

	if err := scanner.Err(); err != nil {
		c.sendAsyncError(errChan, err)
	}
}

func (c *ChatCompletionClient) writeSSELine(reqCtx *gin.Context, line string) error {
	if reqCtx == nil {
		return fmt.Errorf("%s: nil gin context provided", c.name)
	}
	_, err := reqCtx.Writer.Write([]byte(line + newlineChar))
	if err != nil {
		return err
	}
	reqCtx.Writer.Flush()
	return nil
}

func (c *ChatCompletionClient) processStreamChunkForChannel(data string) (string, string, *openai.FunctionCall, *openai.ToolCall) {
	var streamData struct {
		Choices []struct {
			Delta struct {
				Content          string               `json:"content"`
				ReasoningContent string               `json:"reasoning_content"`
				FunctionCall     *openai.FunctionCall `json:"function_call,omitempty"`
				ToolCalls        []openai.ToolCall    `json:"tool_calls,omitempty"`
			} `json:"delta"`
		} `json:"choices"`
	}

	if err := json.Unmarshal([]byte(data), &streamData); err != nil {
		logger.GetLogger().Errorf("%s: failed to parse stream chunk JSON: %v, data: %s", c.name, err, data)
		return "", "", nil, nil
	}

	var contentChunk string
	var reasoningChunk string
	var functionCall *openai.FunctionCall
	var toolCall *openai.ToolCall

	for _, choice := range streamData.Choices {
		if choice.Delta.Content != "" {
			contentChunk += choice.Delta.Content
		}

		if choice.Delta.ReasoningContent != "" {
			reasoningChunk += choice.Delta.ReasoningContent
		}

		if choice.Delta.FunctionCall != nil {
			functionCall = choice.Delta.FunctionCall
		}

		if len(choice.Delta.ToolCalls) > 0 {
			toolCall = &choice.Delta.ToolCalls[0]
		}
	}

	return contentChunk, reasoningChunk, functionCall, toolCall
}

func (c *ChatCompletionClient) handleStreamingFunctionCall(functionCall *openai.FunctionCall, accumulator map[int]*functionCallAccumulator) {
	if functionCall == nil {
		return
	}

	index := 0
	if accumulator[index] == nil {
		accumulator[index] = &functionCallAccumulator{}
	}

	if functionCall.Name != "" {
		accumulator[index].Name = functionCall.Name
	}
	if functionCall.Arguments != "" {
		accumulator[index].Arguments += functionCall.Arguments
	}

	if accumulator[index].Name != "" && accumulator[index].Arguments != "" && strings.HasSuffix(accumulator[index].Arguments, "}") {
		accumulator[index].Complete = true
	}
}

func (c *ChatCompletionClient) handleStreamingToolCall(toolCall *openai.ToolCall, accumulator map[int]*toolCallAccumulator) {
	if toolCall == nil {
		return
	}

	index := *toolCall.Index
	if accumulator[index] == nil {
		accumulator[index] = &toolCallAccumulator{
			ID:    toolCall.ID,
			Type:  string(toolCall.Type),
			Index: index,
		}
	}

	if toolCall.Function.Name != "" {
		accumulator[index].Function.Name = toolCall.Function.Name
	}
	if toolCall.Function.Arguments != "" {
		accumulator[index].Function.Arguments += toolCall.Function.Arguments
	}

	if accumulator[index].Function.Name != "" && accumulator[index].Function.Arguments != "" && strings.HasSuffix(accumulator[index].Function.Arguments, "}") {
		accumulator[index].Complete = true
	}
}

func (c *ChatCompletionClient) buildCompleteResponse(content string, reasoning string, functionCallAccumulator map[int]*functionCallAccumulator, toolCallAccumulator map[int]*toolCallAccumulator, model string, request openai.ChatCompletionRequest) openai.ChatCompletionResponse {
	message := openai.ChatCompletionMessage{
		Role:    openai.ChatMessageRoleAssistant,
		Content: content,
	}

	if reasoning != "" {
		message.ReasoningContent = reasoning
	}

	finishReason := openai.FinishReasonStop

	if len(functionCallAccumulator) > 0 {
		for _, acc := range functionCallAccumulator {
			if acc != nil && acc.Complete {
				message.FunctionCall = &openai.FunctionCall{
					Name:      acc.Name,
					Arguments: acc.Arguments,
				}
				finishReason = openai.FinishReasonFunctionCall
				break
			}
		}
	}

	if len(toolCallAccumulator) > 0 {
		var toolCalls []openai.ToolCall
		for _, acc := range toolCallAccumulator {
			if acc != nil && acc.Complete {
				toolCalls = append(toolCalls, openai.ToolCall{
					ID:   acc.ID,
					Type: openai.ToolType(acc.Type),
					Function: openai.FunctionCall{
						Name:      acc.Function.Name,
						Arguments: acc.Function.Arguments,
					},
				})
			}
		}

		if len(toolCalls) > 0 {
			message.ToolCalls = toolCalls
			finishReason = openai.FinishReasonToolCalls
		}
	}

	choices := []openai.ChatCompletionChoice{
		{
			Index:        0,
			Message:      message,
			FinishReason: finishReason,
		},
	}

	promptTokens := c.estimateTokens(request.Messages)
	completionTokens := c.estimateTokens([]openai.ChatCompletionMessage{message})
	totalTokens := promptTokens + completionTokens

	return openai.ChatCompletionResponse{
		ID:      "",
		Object:  "chat.completion",
		Created: time.Now().Unix(),
		Model:   model,
		Choices: choices,
		Usage: openai.Usage{
			PromptTokens:     promptTokens,
			CompletionTokens: completionTokens,
			TotalTokens:      totalTokens,
		},
	}
}

func (c *ChatCompletionClient) estimateTokens(messages []openai.ChatCompletionMessage) int {
	var allText strings.Builder

	for _, msg := range messages {
		allText.WriteString(msg.Content)
		allText.WriteString(" ")

		if msg.FunctionCall != nil {
			allText.WriteString(msg.FunctionCall.Name)
			allText.WriteString(" ")
			allText.WriteString(msg.FunctionCall.Arguments)
			allText.WriteString(" ")
		}

		for _, toolCall := range msg.ToolCalls {
			allText.WriteString(toolCall.ID)
			allText.WriteString(" ")
			allText.WriteString(toolCall.Function.Name)
			allText.WriteString(" ")
			allText.WriteString(toolCall.Function.Arguments)
			allText.WriteString(" ")
		}
	}

	normalized := strings.Join(strings.Fields(allText.String()), " ")
	words := strings.Fields(normalized)
	return len(words)
}

func (c *ChatCompletionClient) sendAsyncError(errChan chan<- error, err error) {
	if err == nil {
		return
	}

	select {
	case errChan <- err:
	default:
	}
}

func (c *ChatCompletionClient) BaseURL() string {
	return c.baseURL
}

func normalizeBaseURL(base string) string {
	trimmed := strings.TrimSpace(base)
	trimmed = strings.TrimRight(trimmed, "/")
	return trimmed
}

func statusCode(resp *resty.Response) int {
	if resp == nil {
		return 0
	}
	return resp.StatusCode()
}

package chat

import (
	"context"
	"fmt"
	"io"
	"strings"

	"resty.dev/v3"
)

type ChatModelClient struct {
	client  *resty.Client
	baseURL string
	name    string
}

type ModelsResponse struct {
	Object string  `json:"object"`
	Data   []Model `json:"data"`
}

type Model struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int    `json:"created"`
	OwnedBy string `json:"owned_by"`
}

func NewChatModelClient(client *resty.Client, name, baseURL string) *ChatModelClient {
	return &ChatModelClient{
		client:  client,
		baseURL: normalizeBaseURL(baseURL),
		name:    name,
	}
}

func (c *ChatModelClient) ListModels(ctx context.Context) (*ModelsResponse, error) {
	var respBody ModelsResponse
	resp, err := c.client.R().
		SetContext(ctx).
		SetResult(&respBody).
		Get(c.endpoint("/models"))
	if err != nil {
		return nil, err
	}
	if resp.IsError() {
		return nil, c.errorFromResponse(resp, "list models request failed")
	}
	return &respBody, nil
}

func (c *ChatModelClient) endpoint(path string) string {
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

func (c *ChatModelClient) errorFromResponse(resp *resty.Response, message string) error {
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

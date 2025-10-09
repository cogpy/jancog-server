package chat

import (
	"context"
	"encoding/json"
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
	ID            string         `json:"id"`
	Object        string         `json:"object"`
	OwnedBy       string         `json:"owned_by"`
	Created       int            `json:"created"`
	DisplayName   string         `json:"display_name"`
	Name          string         `json:"name"`
	CanonicalSlug string         `json:"canonical_slug"`
	Raw           map[string]any `json:"-"`
}

func (m *Model) UnmarshalJSON(data []byte) error {
	type Alias Model
	aux := Alias{}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	*m = Model(aux)
	m.Raw = raw
	if m.DisplayName == "" {
		if display, ok := raw["display_name"].(string); ok && display != "" {
			m.DisplayName = display
		} else if name, ok := raw["name"].(string); ok && name != "" {
			m.DisplayName = name
		} else {
			m.DisplayName = m.ID
		}
	}
	if m.Name == "" {
		if name, ok := raw["name"].(string); ok {
			m.Name = name
		}
	}
	if m.OwnedBy == "" {
		if ownedBy, ok := raw["owned_by"].(string); ok {
			m.OwnedBy = ownedBy
		}
	}
	if m.CanonicalSlug == "" {
		if slug, ok := raw["canonical_slug"].(string); ok {
			m.CanonicalSlug = slug
		}
	}
	if created, ok := raw["created"]; ok {
		if createdInt, castOK := created.(float64); castOK {
			m.Created = int(createdInt)
		} else if createdInt, castOK := created.(int); castOK {
			m.Created = createdInt
		}
	}
	return nil
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

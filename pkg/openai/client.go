package openai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const (
	OpenAIAPIURL = "https://api.openai.com/v1/responses"
	DefaultModel = "gpt-4.1"
)

type Client struct {
	apiKey     string
	httpClient *http.Client
	model      string
}

// ResponseRequest for the new Responses API
type ResponseRequest struct {
	Model       string  `json:"model"`
	Input       string  `json:"input"`
	Temperature float64 `json:"temperature,omitempty"`
}

// ResponseReply for the new Responses API
type ResponseReply struct {
	ID        string        `json:"id"`
	Object    string        `json:"object"`
	CreatedAt int64         `json:"created_at"`
	Status    string        `json:"status"`
	Model     string        `json:"model"`
	Output    []OutputItem  `json:"output"`
	Usage     ResponseUsage `json:"usage"`
}

type OutputItem struct {
	Type    string        `json:"type"`
	ID      string        `json:"id"`
	Status  string        `json:"status"`
	Role    string        `json:"role"`
	Content []ContentItem `json:"content"`
}

type ContentItem struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type ResponseUsage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
	TotalTokens  int `json:"total_tokens"`
}

type ErrorResponse struct {
	Error struct {
		Message string `json:"message"`
		Type    string `json:"type"`
		Code    string `json:"code"`
	} `json:"error"`
}

// NewClient creates a new OpenAI client
func NewClient(apiKey string) *Client {
	return &Client{
		apiKey: apiKey,
		httpClient: &http.Client{
			Timeout: 60 * time.Second,
		},
		model: DefaultModel,
	}
}

// WithModel sets a custom model (e.g., "gpt-4", "gpt-3.5-turbo")
func (c *Client) WithModel(model string) *Client {
	c.model = model
	return c
}

// CreateChatCompletion sends a chat completion request to OpenAI using the new Responses API
func (c *Client) CreateChatCompletion(ctx context.Context, prompt string) (string, error) {
	// Build input with system message and user prompt
	fullInput := "You are a professional career consultant and resume writer. Your task is to create compelling, professional CV descriptions based on candidate information.\n\n" + prompt

	reqBody := ResponseRequest{
		Model:       c.model,
		Input:       fullInput,
		Temperature: 0.7,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", OpenAIAPIURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		var errResp ErrorResponse
		if err := json.Unmarshal(body, &errResp); err != nil {
			return "", fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(body))
		}
		return "", fmt.Errorf("OpenAI API error: %s", errResp.Error.Message)
	}

	var responseReply ResponseReply
	if err := json.Unmarshal(body, &responseReply); err != nil {
		return "", fmt.Errorf("failed to unmarshal response: %w", err)
	}

	// Extract text from the response
	if len(responseReply.Output) == 0 {
		return "", fmt.Errorf("no output from OpenAI")
	}

	output := responseReply.Output[0]
	if len(output.Content) == 0 {
		return "", fmt.Errorf("no content in OpenAI output")
	}

	return output.Content[0].Text, nil
}

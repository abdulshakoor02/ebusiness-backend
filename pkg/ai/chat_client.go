package ai

import (
	"context"
	"fmt"

	openai "github.com/sashabaranov/go-openai"
)

// ChatClient wraps the go-openai client for tool-calling chat completions.
// It is separate from the existing Client (used by lead import) to avoid breaking changes.
type ChatClient struct {
	client *openai.Client
	model  string
}

func NewChatClient(baseURL, apiKey, model string) *ChatClient {
	config := openai.DefaultConfig(apiKey)
	if baseURL != "" {
		config.BaseURL = baseURL
	}
	return &ChatClient{
		client: openai.NewClientWithConfig(config),
		model:  model,
	}
}

// ChatWithToolsResult holds the result of a chat completion that may contain
// either a text response or tool calls to execute.
type ChatWithToolsResult struct {
	Content   string
	ToolCalls []openai.ToolCall
	FinishReason string
}

// HasToolCalls returns true if the LLM wants to call tools.
func (r *ChatWithToolsResult) HasToolCalls() bool {
	return len(r.ToolCalls) > 0
}

// CreateChatCompletion sends messages with optional tool definitions and returns
// either a text response or tool calls.
func (c *ChatClient) CreateChatCompletion(
	ctx context.Context,
	messages []openai.ChatCompletionMessage,
	tools []openai.Tool,
) (*ChatWithToolsResult, error) {
	req := openai.ChatCompletionRequest{
		Model:       c.model,
		Messages:    messages,
		Temperature: 0,
	}
	if len(tools) > 0 {
		req.Tools = tools
	}

	resp, err := c.client.CreateChatCompletion(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("AI chat completion failed: %w", err)
	}

	if len(resp.Choices) == 0 {
		return nil, fmt.Errorf("no response choices from AI")
	}

	choice := resp.Choices[0]
	result := &ChatWithToolsResult{
		Content:      choice.Message.Content,
		FinishReason: string(choice.FinishReason),
		ToolCalls:    choice.Message.ToolCalls,
	}

	return result, nil
}

// CreateFollowUpCompletion sends tool results back to the LLM for a final answer.
func (c *ChatClient) CreateFollowUpCompletion(
	ctx context.Context,
	messages []openai.ChatCompletionMessage,
) (*ChatWithToolsResult, error) {
	return c.CreateChatCompletion(ctx, messages, nil)
}

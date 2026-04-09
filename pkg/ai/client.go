package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"
)

type Client struct {
	baseURL    string
	apiKey     string
	model      string
	httpClient *http.Client
}

type chatRequest struct {
	Model       string    `json:"model"`
	Messages    []message `json:"messages"`
	Temperature float64   `json:"temperature"`
}

type message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatResponse struct {
	Choices []struct {
		Message message `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

func NewClient(baseURL, apiKey, model string) *Client {
	return &Client{
		baseURL: strings.TrimRight(baseURL, "/"),
		apiKey:  apiKey,
		model:   model,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (c *Client) Chat(ctx context.Context, systemPrompt, userPrompt string) (string, error) {
	req := chatRequest{
		Model:       c.model,
		Temperature: 0,
		Messages: []message{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: userPrompt},
		},
	}

	body, err := json.Marshal(req)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)

	var lastErr error
	for attempt := 0; attempt < 3; attempt++ {
		if attempt > 0 {
			time.Sleep(time.Duration(attempt) * time.Second)
		}

		resp, err := c.httpClient.Do(httpReq)
		if err != nil {
			lastErr = fmt.Errorf("request failed: %w", err)
			continue
		}

		bodyBytes, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			lastErr = fmt.Errorf("failed to read response: %w", err)
			continue
		}

		if resp.StatusCode == 429 {
			lastErr = errors.New("rate limited by AI provider")
			continue
		}

		if resp.StatusCode != 200 {
			lastErr = fmt.Errorf("AI API error %d: %s", resp.StatusCode, string(bodyBytes))
			continue
		}

		var chatResp chatResponse
		if err := json.Unmarshal(bodyBytes, &chatResp); err != nil {
			lastErr = fmt.Errorf("failed to parse response: %w", err)
			continue
		}

		if chatResp.Error != nil {
			lastErr = fmt.Errorf("AI error: %s", chatResp.Error.Message)
			continue
		}

		if len(chatResp.Choices) == 0 {
			lastErr = errors.New("no response choices from AI")
			continue
		}

		return chatResp.Choices[0].Message.Content, nil
	}

	return "", fmt.Errorf("AI request failed after 3 attempts: %w", lastErr)
}

func ParseJSONResponse(raw string) (string, error) {
	raw = strings.TrimSpace(raw)

	jsonRe := regexp.MustCompile("```(?:json)?\n([\\s\\S]*?)\n```")
	matches := jsonRe.FindStringSubmatch(raw)
	if len(matches) > 1 {
		return strings.TrimSpace(matches[1]), nil
	}

	start := strings.Index(raw, "{")
	end := strings.LastIndex(raw, "}")
	if start != -1 && end != -1 && end > start {
		return raw[start : end+1], nil
	}

	start = strings.Index(raw, "[")
	end = strings.LastIndex(raw, "]")
	if start != -1 && end != -1 && end > start {
		return raw[start : end+1], nil
	}

	return "", fmt.Errorf("no JSON found in response: %s", raw[:min(100, len(raw))])
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

package agent

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// OpenAIAgent implements Agent using any OpenAI-compatible chat completions API.
// Works with OpenAI (ChatGPT), Mistral, Llama via Groq, and other compatible providers.
type OpenAIAgent struct {
	baseURL   string
	apiKey    string
	modelID   string
	maxTokens int64
	verbose   bool
}

// NewOpenAIAgent returns an agent targeting the given OpenAI-compatible base URL.
// baseURL should be the root (e.g. "https://api.openai.com" — no trailing slash).
func NewOpenAIAgent(baseURL, apiKey, modelID string, maxTokens int64, verbose bool) *OpenAIAgent {
	return &OpenAIAgent{
		baseURL:   strings.TrimRight(baseURL, "/"),
		apiKey:    apiKey,
		modelID:   modelID,
		maxTokens: maxTokens,
		verbose:   verbose,
	}
}

// Run calls the provider's chat completions endpoint, parses the <files> block,
// and returns the generated files.
func (a *OpenAIAgent) Run(ctx context.Context, ac *Context) (*Result, error) {
	systemPrompt := SystemPrompt(ac.Task.Kind, ac.SkillDocs)
	userMsg, err := UserMessage(ac)
	if err != nil {
		return nil, fmt.Errorf("build user message: %w", err)
	}

	// o1 and o3 models use the "developer" role for system instructions and
	// "max_completion_tokens" instead of "max_tokens".
	isOSeries := strings.HasPrefix(a.modelID, "o1") || strings.HasPrefix(a.modelID, "o3")

	systemRole := "system"
	if isOSeries {
		systemRole = "developer"
	}

	type chatMessage struct {
		Role    string `json:"role"`
		Content string `json:"content"`
	}
	reqBody := map[string]any{
		"model": a.modelID,
		"messages": []chatMessage{
			{Role: systemRole, Content: systemPrompt},
			{Role: "user", Content: userMsg},
		},
	}
	if isOSeries {
		reqBody["max_completion_tokens"] = a.maxTokens
	} else {
		reqBody["max_tokens"] = a.maxTokens
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		a.baseURL+"/v1/chat/completions", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+a.apiKey)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("http request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response body: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("provider returned HTTP %d: %s",
			resp.StatusCode, truncateStr(string(respBody), 300))
	}

	var apiResp struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
			FinishReason string `json:"finish_reason"`
		} `json:"choices"`
		Usage struct {
			PromptTokens     int64 `json:"prompt_tokens"`
			CompletionTokens int64 `json:"completion_tokens"`
		} `json:"usage"`
	}
	if err := json.Unmarshal(respBody, &apiResp); err != nil {
		return nil, fmt.Errorf("parse response JSON: %w", err)
	}
	if len(apiResp.Choices) == 0 {
		return nil, fmt.Errorf("provider returned no choices")
	}

	choice := apiResp.Choices[0]
	if choice.FinishReason == "length" {
		return nil, fmt.Errorf("response truncated (finish_reason=length); increase maxTokens or split the task")
	}

	result := &Result{
		PromptTokens: apiResp.Usage.PromptTokens,
		OutputTokens: apiResp.Usage.CompletionTokens,
	}
	if a.verbose {
		fmt.Printf("[%s] tokens: in=%d out=%d\n", ac.Task.ID, result.PromptTokens, result.OutputTokens)
	}

	files, err := parseFilesBlock(choice.Message.Content)
	if err != nil {
		return nil, fmt.Errorf("parse <files> block for task %s: %w", ac.Task.ID, err)
	}
	result.Files = files
	return result, nil
}

// truncateStr limits s to n bytes for safe inclusion in error messages.
func truncateStr(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}

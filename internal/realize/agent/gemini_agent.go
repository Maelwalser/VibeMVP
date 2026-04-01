package agent

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

const geminiBaseURL = "https://generativelanguage.googleapis.com/v1beta/models"

// GeminiAgent implements Agent using the Google Gemini generative language API.
type GeminiAgent struct {
	apiKey    string
	modelID   string
	maxTokens int64
	verbose   bool
}

// NewGeminiAgent returns a GeminiAgent authenticated with the given API key.
func NewGeminiAgent(apiKey, modelID string, maxTokens int64, verbose bool) *GeminiAgent {
	return &GeminiAgent{
		apiKey:    apiKey,
		modelID:   modelID,
		maxTokens: maxTokens,
		verbose:   verbose,
	}
}

// Run calls the Gemini generateContent endpoint, parses the <files> block,
// and returns the generated files.
func (a *GeminiAgent) Run(ctx context.Context, ac *Context) (*Result, error) {
	systemPrompt := SystemPrompt(ac.Task.Kind, ac.SkillDocs)
	userMsg, err := UserMessage(ac)
	if err != nil {
		return nil, fmt.Errorf("build user message: %w", err)
	}

	reqBody := map[string]any{
		"system_instruction": map[string]any{
			"parts": []map[string]string{{"text": systemPrompt}},
		},
		"contents": []map[string]any{
			{
				"role":  "user",
				"parts": []map[string]string{{"text": userMsg}},
			},
		},
		"generationConfig": map[string]any{
			"maxOutputTokens": a.maxTokens,
		},
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	url := fmt.Sprintf("%s/%s:generateContent?key=%s", geminiBaseURL, a.modelID, a.apiKey)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

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
		return nil, fmt.Errorf("gemini returned HTTP %d: %s",
			resp.StatusCode, truncateStr(string(respBody), 300))
	}

	var apiResp struct {
		Candidates []struct {
			Content struct {
				Parts []struct {
					Text string `json:"text"`
				} `json:"parts"`
			} `json:"content"`
			FinishReason string `json:"finishReason"`
		} `json:"candidates"`
		UsageMetadata struct {
			PromptTokenCount     int64 `json:"promptTokenCount"`
			CandidatesTokenCount int64 `json:"candidatesTokenCount"`
		} `json:"usageMetadata"`
	}
	if err := json.Unmarshal(respBody, &apiResp); err != nil {
		return nil, fmt.Errorf("parse response JSON: %w", err)
	}
	if len(apiResp.Candidates) == 0 || len(apiResp.Candidates[0].Content.Parts) == 0 {
		return nil, fmt.Errorf("gemini returned no candidates")
	}

	candidate := apiResp.Candidates[0]
	if candidate.FinishReason == "MAX_TOKENS" {
		return nil, fmt.Errorf("response truncated (MAX_TOKENS); increase maxTokens or split the task")
	}

	result := &Result{
		PromptTokens: apiResp.UsageMetadata.PromptTokenCount,
		OutputTokens: apiResp.UsageMetadata.CandidatesTokenCount,
	}
	if a.verbose {
		fmt.Printf("[%s] tokens: in=%d out=%d\n", ac.Task.ID, result.PromptTokens, result.OutputTokens)
	}

	files, err := parseFilesBlock(candidate.Content.Parts[0].Text)
	if err != nil {
		return nil, fmt.Errorf("parse <files> block for task %s: %w", ac.Task.ID, err)
	}
	result.Files = files
	return result, nil
}

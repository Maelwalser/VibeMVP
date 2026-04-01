package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
	"github.com/vibe-mvp/internal/realize/dag"
)

// Result is what a task agent returns after one invocation.
type Result struct {
	Files        []dag.GeneratedFile
	ThinkingLog  string
	PromptTokens int64
	OutputTokens int64
}

// Agent executes one task and returns generated files.
type Agent interface {
	Run(ctx context.Context, ac *Context) (*Result, error)
}

// ClaudeAgent implements Agent using the Anthropic SDK with streaming.
type ClaudeAgent struct {
	client    *anthropic.Client
	model     string
	maxTokens int64
	verbose   bool
}

// NewClaudeAgent returns a ClaudeAgent. If authToken is non-empty it is used as a
// Bearer token (Claude Pro/Max session key); otherwise ANTHROPIC_API_KEY is used.
func NewClaudeAgent(model string, maxTokens int64, verbose bool, authToken string) *ClaudeAgent {
	opts := []option.RequestOption{option.WithMaxRetries(0)}
	if authToken != "" {
		opts = append(opts, option.WithAuthToken(authToken))
	}
	c := anthropic.NewClient(opts...)
	return &ClaudeAgent{
		client:    &c,
		model:     model,
		maxTokens: maxTokens,
		verbose:   verbose,
	}
}

// Run invokes Claude for the task, streams the response, parses the <files> block,
// and returns the generated files.
func (a *ClaudeAgent) Run(ctx context.Context, ac *Context) (*Result, error) {
	systemPrompt := SystemPrompt(ac.Task.Kind, ac.SkillDocs)
	userMsg, err := UserMessage(ac)
	if err != nil {
		return nil, fmt.Errorf("build user message: %w", err)
	}

	params := anthropic.MessageNewParams{
		Model:     anthropic.Model(a.model),
		MaxTokens: a.maxTokens,
		System: []anthropic.TextBlockParam{
			{Text: systemPrompt},
		},
		Messages: []anthropic.MessageParam{
			anthropic.NewUserMessage(anthropic.NewTextBlock(userMsg)),
		},
	}

	stream := a.client.Messages.NewStreaming(ctx, params)
	msg := anthropic.Message{}
	for stream.Next() {
		event := stream.Current()
		if err := msg.Accumulate(event); err != nil {
			return nil, fmt.Errorf("accumulate stream event: %w", err)
		}
	}
	if err := stream.Err(); err != nil {
		return nil, fmt.Errorf("streaming error: %w", err)
	}

	result := &Result{
		PromptTokens: msg.Usage.InputTokens,
		OutputTokens: msg.Usage.OutputTokens,
	}

	// Extract text and thinking content.
	var fullText strings.Builder
	var thinkingParts strings.Builder
	for _, block := range msg.Content {
		switch b := block.AsAny().(type) {
		case anthropic.TextBlock:
			fullText.WriteString(b.Text)
		case anthropic.ThinkingBlock:
			thinkingParts.WriteString(b.Thinking)
		}
	}
	result.ThinkingLog = thinkingParts.String()

	if a.verbose && result.ThinkingLog != "" {
		fmt.Printf("[%s] thinking: %s\n", ac.Task.ID, result.ThinkingLog)
	}
	if a.verbose {
		fmt.Printf("[%s] tokens: in=%d out=%d\n", ac.Task.ID, result.PromptTokens, result.OutputTokens)
	}

	files, err := parseFilesBlock(fullText.String())
	if err != nil {
		return nil, fmt.Errorf("parse <files> block for task %s: %w", ac.Task.ID, err)
	}
	result.Files = files
	return result, nil
}

// parseFilesBlock extracts the JSON array from between <files>...</files> tags.
func parseFilesBlock(text string) ([]dag.GeneratedFile, error) {
	start := strings.Index(text, "<files>")
	end := strings.Index(text, "</files>")
	if start == -1 || end == -1 || end <= start {
		return nil, fmt.Errorf("response does not contain a valid <files>...</files> block")
	}
	raw := strings.TrimSpace(text[start+len("<files>") : end])

	var files []dag.GeneratedFile
	if err := json.Unmarshal([]byte(raw), &files); err != nil {
		return nil, fmt.Errorf("unmarshal files JSON: %w", err)
	}
	if len(files) == 0 {
		return nil, fmt.Errorf("agent returned an empty file list")
	}
	return files, nil
}

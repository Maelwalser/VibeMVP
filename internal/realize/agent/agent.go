package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
	"github.com/vibe-menu/internal/realize/dag"
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
	client         *anthropic.Client
	model          string
	maxTokens      int64
	thinkingBudget int64 // 0 = disabled; >0 enables extended thinking
	verbose        bool
}

// NewClaudeAgent returns a ClaudeAgent authenticated via ANTHROPIC_API_KEY.
func NewClaudeAgent(model string, maxTokens, thinkingBudget int64, verbose bool) *ClaudeAgent {
	c := anthropic.NewClient(option.WithMaxRetries(0))
	return &ClaudeAgent{
		client:         &c,
		model:          model,
		maxTokens:      maxTokens,
		thinkingBudget: thinkingBudget,
		verbose:        verbose,
	}
}

// NewClaudeAgentWithKey returns a ClaudeAgent authenticated with the given API key.
// If apiKey is empty, falls back to ANTHROPIC_API_KEY from the environment.
func NewClaudeAgentWithKey(model string, maxTokens, thinkingBudget int64, verbose bool, apiKey string) *ClaudeAgent {
	opts := []option.RequestOption{option.WithMaxRetries(0)}
	if apiKey != "" {
		opts = append(opts, option.WithAPIKey(apiKey))
	}
	c := anthropic.NewClient(opts...)
	return &ClaudeAgent{
		client:         &c,
		model:          model,
		maxTokens:      maxTokens,
		thinkingBudget: thinkingBudget,
		verbose:        verbose,
	}
}

// Run invokes Claude for the task, streams the response, parses the <files> block,
// and returns the generated files.
func (a *ClaudeAgent) Run(ctx context.Context, ac *Context) (*Result, error) {
	systemPrompt := SystemPrompt(ac.Task.Kind, ac.SkillDocs, ac.DepsContext, ac.Language())
	userMsg, err := UserMessage(ac)
	if err != nil {
		return nil, fmt.Errorf("build user message: %w", err)
	}

	// Build system blocks. The role prompt is always cached. Cross-task reference
	// context (constructors, methods, sentinels, contracts) is stable across retries
	// and benefits from a separate cached block — saving ~30-40% on retry input costs.
	systemBlocks := []anthropic.TextBlockParam{
		{Text: systemPrompt, CacheControl: anthropic.NewCacheControlEphemeralParam()},
	}
	if refCtx := ReferenceContext(ac); refCtx != "" {
		systemBlocks = append(systemBlocks, anthropic.TextBlockParam{
			Text:         refCtx,
			CacheControl: anthropic.NewCacheControlEphemeralParam(),
		})
	}

	params := anthropic.MessageNewParams{
		Model:     anthropic.Model(a.model),
		MaxTokens: a.maxTokens,
		System:    systemBlocks,
		Messages: []anthropic.MessageParam{
			anthropic.NewUserMessage(anthropic.NewTextBlock(userMsg)),
		},
	}

	// Enable extended thinking when a budget is configured. Claude 4.6 models
	// (sonnet/opus) support adaptive thinking; older models use explicit budgets.
	if a.thinkingBudget > 0 {
		if isAdaptiveModel(a.model) {
			params.Thinking = anthropic.ThinkingConfigParamUnion{
				OfAdaptive: &anthropic.ThinkingConfigAdaptiveParam{},
			}
		} else {
			params.Thinking = anthropic.ThinkingConfigParamOfEnabled(a.thinkingBudget)
		}
	}

	// Estimate total token usage and warn if approaching context window limit.
	// Simple heuristic: ~4 chars per token for English text.
	estimatedInputTokens := int64(len(systemPrompt)+len(userMsg)) / 4
	estimatedTotal := estimatedInputTokens + a.maxTokens + a.thinkingBudget
	if a.verbose && estimatedTotal > 0 {
		fmt.Printf("[%s] estimated context: ~%dk input + %dk output + %dk thinking\n",
			ac.Task.ID, estimatedInputTokens/1000, a.maxTokens/1000, a.thinkingBudget/1000)
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

	// Log token usage immediately — before any early-return — so failures are
	// always visible in verbose mode (truncation used to hide the output count).
	if a.verbose {
		cacheRead := msg.Usage.CacheReadInputTokens
		cacheCreate := msg.Usage.CacheCreationInputTokens
		if cacheRead > 0 || cacheCreate > 0 {
			fmt.Printf("[%s] cache: read=%d create=%d\n", ac.Task.ID, cacheRead, cacheCreate)
		}
		fmt.Printf("[%s] tokens: in=%d out=%d\n", ac.Task.ID, result.PromptTokens, result.OutputTokens)
	}

	// Detect truncation before trying to parse — a max_tokens stop means the
	// </files> closing tag was never written and parsing will always fail.
	if msg.StopReason == anthropic.StopReasonMaxTokens {
		return nil, fmt.Errorf("response truncated (max_tokens=%d reached; out=%d tokens); task may be too large to fit in one generation", a.maxTokens, msg.Usage.OutputTokens)
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

	files, err := parseFilesBlock(fullText.String())
	if err != nil {
		return nil, fmt.Errorf("parse <files> block for task %s: %w", ac.Task.ID, err)
	}
	result.Files = files
	return result, nil
}

// isAdaptiveModel reports whether the model supports adaptive thinking (type: "adaptive")
// instead of explicit budget-based thinking. Claude 4.6 models (sonnet-4-6, opus-4-6)
// use adaptive; earlier models require an explicit BudgetTokens value.
func isAdaptiveModel(model string) bool {
	return strings.Contains(model, "sonnet-4-6") ||
		strings.Contains(model, "opus-4-6") ||
		strings.Contains(model, "sonnet-4-5") ||
		strings.Contains(model, "opus-4-5")
}

// parseFilesBlock extracts the JSON array from between <files>...</files> tags.
// Falls back to JSON repair and alternative extraction when the primary parse fails.
func parseFilesBlock(text string) ([]dag.GeneratedFile, error) {
	start := strings.Index(text, "<files>")
	end := strings.Index(text, "</files>")

	// Fallback 1: no <files> tags — try extracting the largest JSON array.
	if start == -1 || end == -1 || end <= start {
		if files := extractLargestJSONArray(text); len(files) > 0 {
			return files, nil
		}
		return nil, fmt.Errorf("response does not contain a valid <files>...</files> block")
	}

	raw := strings.TrimSpace(text[start+len("<files>") : end])

	var files []dag.GeneratedFile
	if err := json.Unmarshal([]byte(raw), &files); err != nil {
		// Fallback 2: try repairing common JSON malformations.
		repaired := repairJSON(raw)
		if err2 := json.Unmarshal([]byte(repaired), &files); err2 != nil {
			// Fallback 3: try extracting individual file objects.
			if extracted := extractFileObjects(raw); len(extracted) > 0 {
				return extracted, nil
			}
			return nil, fmt.Errorf("unmarshal files JSON: %w (repair also failed: %v)", err, err2)
		}
	}
	if len(files) == 0 {
		return nil, fmt.Errorf("agent returned an empty file list")
	}
	return files, nil
}

// repairJSON attempts to fix common LLM JSON malformations:
// - trailing commas before ] or }
// - unclosed arrays/objects at the end (from truncation)
func repairJSON(raw string) string {
	s := raw
	// Remove trailing commas before closing brackets.
	for {
		cleaned := strings.ReplaceAll(s, ",\n]", "\n]")
		cleaned = strings.ReplaceAll(cleaned, ",\n}", "\n}")
		cleaned = strings.ReplaceAll(cleaned, ", ]", "]")
		cleaned = strings.ReplaceAll(cleaned, ", }", "}")
		cleaned = strings.ReplaceAll(cleaned, ",]", "]")
		cleaned = strings.ReplaceAll(cleaned, ",}", "}")
		if cleaned == s {
			break
		}
		s = cleaned
	}
	// Close unclosed brackets at end (from truncation).
	s = strings.TrimRight(s, " \t\n\r")
	opens := strings.Count(s, "[") - strings.Count(s, "]")
	braces := strings.Count(s, "{") - strings.Count(s, "}")
	// If ends mid-string, try to close it.
	if braces > 0 && !strings.HasSuffix(s, "}") {
		s += "\"}"
		braces--
	}
	for braces > 0 {
		s += "}"
		braces--
	}
	for opens > 0 {
		s += "]"
		opens--
	}
	return s
}

// extractLargestJSONArray finds the largest [...] block in text and tries to parse
// it as a []GeneratedFile. Used when the LLM omits <files> tags entirely.
func extractLargestJSONArray(text string) []dag.GeneratedFile {
	bestStart, bestEnd := -1, -1
	for i := 0; i < len(text); i++ {
		if text[i] != '[' {
			continue
		}
		depth := 0
		inString := false
		escape := false
		for j := i; j < len(text); j++ {
			if escape {
				escape = false
				continue
			}
			ch := text[j]
			if ch == '\\' && inString {
				escape = true
				continue
			}
			if ch == '"' {
				inString = !inString
				continue
			}
			if inString {
				continue
			}
			if ch == '[' {
				depth++
			} else if ch == ']' {
				depth--
				if depth == 0 {
					// Found a complete array. Keep the largest.
					if bestStart == -1 || (j-i) > (bestEnd-bestStart) {
						bestStart, bestEnd = i, j+1
					}
					break
				}
			}
		}
	}
	if bestStart == -1 {
		return nil
	}
	raw := text[bestStart:bestEnd]
	var files []dag.GeneratedFile
	if err := json.Unmarshal([]byte(raw), &files); err != nil {
		repaired := repairJSON(raw)
		if err := json.Unmarshal([]byte(repaired), &files); err != nil {
			return nil
		}
	}
	return files
}

// extractFileObjects finds individual {"path":..., "content":...} objects in raw
// text and assembles them into a file list. Last-resort fallback when the JSON
// array is too malformed to parse as a whole.
func extractFileObjects(raw string) []dag.GeneratedFile {
	var files []dag.GeneratedFile
	search := raw
	for {
		idx := strings.Index(search, `"path"`)
		if idx == -1 {
			break
		}
		// Walk backward to find opening brace.
		braceIdx := strings.LastIndex(search[:idx], "{")
		if braceIdx == -1 {
			search = search[idx+6:]
			continue
		}
		// Walk forward to find matching closing brace.
		depth := 0
		inStr := false
		esc := false
		endIdx := -1
		for j := braceIdx; j < len(search); j++ {
			if esc {
				esc = false
				continue
			}
			ch := search[j]
			if ch == '\\' && inStr {
				esc = true
				continue
			}
			if ch == '"' {
				inStr = !inStr
				continue
			}
			if inStr {
				continue
			}
			if ch == '{' {
				depth++
			} else if ch == '}' {
				depth--
				if depth == 0 {
					endIdx = j + 1
					break
				}
			}
		}
		if endIdx == -1 {
			break
		}
		obj := search[braceIdx:endIdx]
		var f dag.GeneratedFile
		if err := json.Unmarshal([]byte(obj), &f); err == nil && f.Path != "" {
			files = append(files, f)
		}
		search = search[endIdx:]
	}
	return files
}

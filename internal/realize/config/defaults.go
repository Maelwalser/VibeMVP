package config

// DefaultModel is the Claude model used when no override is specified.
const DefaultModel = "claude-opus-4-6"

// DefaultMaxTokens is the maximum output token budget per agent call.
const DefaultMaxTokens = int64(64000)

// MaxSkillBytes is the maximum number of characters stored per skill document.
// Increased from 2000 to 6000 so critical API patterns, security rules, and
// library-specific usage docs are not truncated. With 2-4 skills per task,
// total skill injection stays within 12-24K chars — well within the context window,
// and system prompt caching amortizes the cost across retries.
const MaxSkillBytes = 6000

// MaxFileChars is the maximum characters included from a single dependency file.
const MaxFileChars = 4000

// MaxTotalChars is the total character budget across all dependency outputs (fallback).
const MaxTotalChars = 12000

// MaxTotalCharsByKind overrides the shared-memory budget for specific task kinds.
// Tasks that aggregate more upstream layers get a larger window so constructor
// signatures and type definitions are not truncated prematurely.
var MaxTotalCharsByKind = map[string]int{
	"backend.service.bootstrap":  30000, // sees repo + service + handler simultaneously
	"backend.service.handler":    20000, // sees repo + service + auth
	"backend.service.logic":      15000, // sees repo + data schemas
	"backend.service.repository": 15000, // sees data schemas + plan interfaces
	"backend.service.plan":       30000, // sees data schemas — must receive full domain structs + input types
	"backend.service.deps":       15000, // sees plan output (go.mod + interfaces)
	"backend.auth":               20000, // needs all service interfaces
	"backend.messaging":          15000, // needs domain + event definitions
	"backend.gateway":            20000, // needs full service surface + endpoints
	"backend.reconciliation":     40000, // reads entire module for cross-task repair
	"integration.repair":         40000, // reads entire module for cross-task repair
	"contracts":                  20000, // aggregates all service + data output
	"frontend":                   20000, // needs contracts + data types
	"crosscut.testing":           40000, // depends on ALL prior tasks — needs constructors + types from every layer
	"crosscut.docs":              25000, // depends on ALL prior tasks — needs endpoint + DTO definitions
}

// MaxTotalCharsFor returns the shared-memory character budget for the given task kind.
// Falls back to MaxTotalChars for unlisted kinds.
func MaxTotalCharsFor(kind string) int {
	if v, ok := MaxTotalCharsByKind[kind]; ok {
		return v
	}
	return MaxTotalChars
}

// ThinkingBudgetByTier maps abstract tier names to the extended thinking token
// budget for Claude agents. TierFast tasks skip thinking entirely; TierMedium
// tasks get moderate reasoning; TierSlow tasks (reconciliation, repair) get
// deep reasoning. Budget must be >= 1024 and < MaxTokens per Anthropic API.
var ThinkingBudgetByTier = map[int]int64{
	0: 0,     // TierFast — no thinking (boilerplate tasks)
	1: 8192,  // TierMedium — moderate reasoning
	2: 16384, // TierSlow — deep cross-task reasoning
}

// MaxRepairFilesPerCall is the maximum number of source files sent to the LLM
// per integration repair invocation. When a failing module has more files than
// this, only the error cluster (files mentioned in compiler errors + their
// direct imports) is sent — preventing context window overflow.
const MaxRepairFilesPerCall = 15

// RateLimitBackoffBase is the per-attempt multiplier in seconds for rate-limit backoff.
// Wait = (attempt+1) * RateLimitBackoffBase seconds.
const RateLimitBackoffBase = 60

package ui

import "strings"

// FieldKind enumerates the types of form fields.
type FieldKind int

const (
	KindText        FieldKind = iota // single-line text input
	KindSelect                      // cycle through a list of options
	KindTextArea                    // multi-line text input
	KindDataModel                   // sentinel: delegates to a sub-editor
	KindMultiSelect                 // select multiple options from a list
)

// Field represents a single form field within a section.
type Field struct {
	Key          string    // machine key (e.g. "arch_pattern")
	Label        string    // padded display label — must be exactly 14 chars
	Kind         FieldKind
	Value        string   // current string value
	Options      []string // KindSelect/KindMultiSelect: available choices
	SelIdx       int      // KindSelect: currently selected index
	SelectedIdxs []int    // KindMultiSelect: indices of selected options
	DDCursor     int      // KindMultiSelect: dropdown cursor position
	CustomText   string   // KindSelect: free-text value when "Custom"/"Other" is selected
}

// isCustomOption returns true for sentinel options that allow free-text entry.
func isCustomOption(opt string) bool {
	lower := strings.ToLower(opt)
	return lower == "custom" || lower == "other"
}

// CanEditAsText returns true when the field supports free-text entry in its current state.
// This is true for KindText, KindTextArea, KindSelect when the active option is
// "Custom"/"Other", and KindMultiSelect when a "custom"/"other" option is selected.
func (f Field) CanEditAsText() bool {
	if f.Kind == KindText || f.Kind == KindTextArea {
		return true
	}
	if f.Kind == KindSelect && len(f.Options) > 0 {
		return isCustomOption(f.Options[f.SelIdx])
	}
	if f.Kind == KindMultiSelect {
		for _, idx := range f.SelectedIdxs {
			if idx >= 0 && idx < len(f.Options) && isCustomOption(f.Options[idx]) {
				return true
			}
		}
	}
	return false
}

// TextInputValue returns the value to pre-populate a text input with when editing this field.
func (f Field) TextInputValue() string {
	if f.Kind == KindSelect && len(f.Options) > 0 && isCustomOption(f.Options[f.SelIdx]) {
		return f.CustomText
	}
	if f.Kind == KindMultiSelect {
		for _, idx := range f.SelectedIdxs {
			if idx >= 0 && idx < len(f.Options) && isCustomOption(f.Options[idx]) {
				return f.CustomText
			}
		}
	}
	return f.Value
}

// SaveTextInput saves the typed text back into the appropriate storage slot.
func (f *Field) SaveTextInput(val string) {
	if f.Kind == KindSelect && len(f.Options) > 0 && isCustomOption(f.Options[f.SelIdx]) {
		f.CustomText = val
		return
	}
	if f.Kind == KindMultiSelect {
		for _, idx := range f.SelectedIdxs {
			if idx >= 0 && idx < len(f.Options) && isCustomOption(f.Options[idx]) {
				f.CustomText = val
				return
			}
		}
	}
	f.Value = val
}

// PrepareCustomEntry clears CustomText so the text input starts blank when the
// user selects a "Custom"/"Other" option from a dropdown.
// Returns true when the field is now in custom-entry state (caller should enter insert mode).
func (f *Field) PrepareCustomEntry() bool {
	if f.Kind == KindSelect && len(f.Options) > 0 && isCustomOption(f.Options[f.SelIdx]) {
		f.CustomText = ""
		return true
	}
	return false
}

// DisplayValue returns the rendered value string for NORMAL mode.
func (f Field) DisplayValue() string {
	if f.Kind == KindSelect {
		if len(f.Options) == 0 {
			return f.Value // placeholder shown when dependent items not yet created
		}
		opt := f.Options[f.SelIdx]
		if isCustomOption(opt) && f.CustomText != "" {
			return f.CustomText
		}
		return opt
	}
	if f.Kind == KindMultiSelect {
		if len(f.Options) == 0 {
			return f.Value // placeholder shown when dependent items not yet created
		}
		if len(f.SelectedIdxs) == 0 {
			return ""
		}
		parts := make([]string, 0, len(f.SelectedIdxs))
		for _, idx := range f.SelectedIdxs {
			if idx >= 0 && idx < len(f.Options) {
				opt := f.Options[idx]
				if isCustomOption(opt) && f.CustomText != "" {
					parts = append(parts, f.CustomText)
				} else {
					parts = append(parts, opt)
				}
			}
		}
		return strings.Join(parts, ", ")
	}
	// Show a one-line preview for textarea fields.
	v := f.Value
	if f.Kind == KindTextArea && len(v) > 0 {
		lines := splitLines(v)
		if len(lines) > 1 {
			return lines[0] + " …"
		}
	}
	return v
}

// ToggleMultiSelect toggles the option at ddCursor in a KindMultiSelect field.
func (f *Field) ToggleMultiSelect(optIdx int) {
	if f.Kind != KindMultiSelect || optIdx < 0 || optIdx >= len(f.Options) {
		return
	}
	for i, idx := range f.SelectedIdxs {
		if idx == optIdx {
			// Remove it
			f.SelectedIdxs = append(f.SelectedIdxs[:i], f.SelectedIdxs[i+1:]...)
			return
		}
	}
	// Add it
	f.SelectedIdxs = append(f.SelectedIdxs, optIdx)
}

// IsMultiSelected returns whether optIdx is currently selected.
func (f Field) IsMultiSelected(optIdx int) bool {
	for _, idx := range f.SelectedIdxs {
		if idx == optIdx {
			return true
		}
	}
	return false
}

// CycleNext advances a KindSelect field to the next option.
func (f *Field) CycleNext() {
	if f.Kind != KindSelect || len(f.Options) == 0 {
		return
	}
	f.SelIdx = (f.SelIdx + 1) % len(f.Options)
	f.Value = f.Options[f.SelIdx]
}

// CyclePrev moves a KindSelect field to the previous option.
func (f *Field) CyclePrev() {
	if f.Kind != KindSelect || len(f.Options) == 0 {
		return
	}
	f.SelIdx = (f.SelIdx - 1 + len(f.Options)) % len(f.Options)
	f.Value = f.Options[f.SelIdx]
}

// Section groups related fields under a phase pillar.
// For the 6 main tabs, each section has a single KindDataModel sentinel field
// that triggers full delegation to the appropriate sub-editor.
type Section struct {
	ID     string  // short identifier (e.g. "backend")
	Abbr   string  // tab label
	Title  string  // full title
	Desc   string  // one-line description shown as a comment
	Fields []Field
}

// initSections returns the main tab section definitions.
func initSections() []Section {
	return []Section{
		{
			ID:    "describe",
			Abbr:  "✦ DESCRIBE",
			Title: "Describe",
			Desc:  "Describe your project in natural language.",
			Fields: []Field{
				{Key: "_describe", Kind: KindDataModel},
			},
		},
		{
			ID:    "backend",
			Abbr:  "⚡ BACKEND",
			Title: "Backend",
			Desc:  "Architecture pattern, environment, service units, communication, and auth.",
			Fields: []Field{
				{Key: "_backend", Kind: KindDataModel},
			},
		},
		{
			ID:    "data",
			Abbr:  "◈ DATA",
			Title: "Data",
			Desc:  "Databases, domains, caching, and file storage.",
			Fields: []Field{
				{Key: "_data", Kind: KindDataModel},
			},
		},
		{
			ID:    "contracts",
			Abbr:  "≋ CONTRACTS",
			Title: "Contracts",
			Desc:  "DTOs, API endpoints, and versioning strategy.",
			Fields: []Field{
				{Key: "_contracts", Kind: KindDataModel},
			},
		},
		{
			ID:    "frontend",
			Abbr:  "◉ FRONTEND",
			Title: "Frontend",
			Desc:  "Technologies, theming, pages, and navigation.",
			Fields: []Field{
				{Key: "_frontend", Kind: KindDataModel},
			},
		},
		{
			ID:    "infrastructure",
			Abbr:  "⊞ INFRA",
			Title: "Infrastructure",
			Desc:  "Networking, CI/CD, and observability.",
			Fields: []Field{
				{Key: "_infra", Kind: KindDataModel},
			},
		},
		{
			ID:    "crosscut",
			Abbr:  "⊕ CROSSCUT",
			Title: "Cross-Cutting",
			Desc:  "Testing strategy and documentation tooling.",
			Fields: []Field{
				{Key: "_crosscut", Kind: KindDataModel},
			},
		},
		{
			ID:    "realize",
			Abbr:  "▶ REALIZE",
			Title: "Realize",
			Desc:  "Output directory, app name, model, and realization options.",
			Fields: []Field{
				{Key: "_realize", Kind: KindDataModel},
			},
		},
	}
}

// splitLines splits a string into lines.
func splitLines(s string) []string {
	if s == "" {
		return nil
	}
	var lines []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			lines = append(lines, s[start:i])
			start = i + 1
		}
	}
	lines = append(lines, s[start:])
	return lines
}

// placeholderFor returns placeholder when opts is empty, or "" otherwise.
// Use as the Value of a KindSelect/KindMultiSelect field whose Options are
// populated dynamically — DisplayValue will show it when Options is empty,
// but it is never added to the options list so it cannot be "selected".
func placeholderFor(opts []string, placeholder string) string {
	if len(opts) == 0 {
		return placeholder
	}
	return ""
}

// noneOrPlaceholder handles fields where "(none)" is a valid explicit choice.
// When items is non-empty it returns options with "(none)" prepended and "(none)"
// as the default value.  When items is empty it returns an empty options slice
// and the placeholder string as the value, so DisplayValue shows the hint
// without adding a fake selectable entry.
func noneOrPlaceholder(items []string, placeholder string) (opts []string, defaultVal string) {
	if len(items) == 0 {
		return []string{}, placeholder
	}
	return append([]string{"(none)"}, items...), "(none)"
}

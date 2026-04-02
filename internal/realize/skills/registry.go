package skills

import "github.com/vibe-menu/internal/realize/dag"

// Doc is a skill markdown document loaded from the skills directory.
type Doc struct {
	Technology string
	Content    string
}

// Registry maps technology names to skill context documents.
type Registry interface {
	// Lookup returns the skill doc for a technology string, or ("", false).
	Lookup(technology string) (string, bool)
	// LookupAll returns all skill docs relevant to a task kind and technology list.
	LookupAll(kind dag.TaskKind, technologies []string) []Doc
}

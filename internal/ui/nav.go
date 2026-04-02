package ui

// nextEditableIdx returns the index of the next KindText or KindTextArea field
// after `from` in `fields`, wrapping around. Returns -1 if no such field exists.
func nextEditableIdx(fields []Field, from int) int {
	n := len(fields)
	if n == 0 {
		return -1
	}
	for i := 1; i <= n; i++ {
		idx := (from + i) % n
		if fields[idx].Kind == KindText || fields[idx].Kind == KindTextArea {
			return idx
		}
	}
	return -1
}

// prevEditableIdx returns the index of the previous KindText or KindTextArea field
// before `from` in `fields`, wrapping around. Returns -1 if no such field exists.
func prevEditableIdx(fields []Field, from int) int {
	n := len(fields)
	if n == 0 {
		return -1
	}
	for i := 1; i <= n; i++ {
		idx := (from - i + n) % n
		if fields[idx].Kind == KindText || fields[idx].Kind == KindTextArea {
			return idx
		}
	}
	return -1
}

// NavigateTab advances or retreats the active tab index based on a key string.
// Returns the new tab index clamped to [0, maxTabs-1].
// Handles "h", "left" (decrement) and "l", "right" (increment).
func NavigateTab(key string, active, maxTabs int) int {
	switch key {
	case "h", "left":
		if active > 0 {
			return active - 1
		}
	case "l", "right":
		if active < maxTabs-1 {
			return active + 1
		}
	}
	return active
}

// VimNav holds the state for vim-style field navigation with count prefixes
// (e.g. "3j" moves down 3) and gg/G motion. Embed or store by value; the
// zero value is ready to use.
//
// VimNav handles pure movement only (j/k/gg/G + digit prefix). Keys like
// enter, space, i, a — whose meaning depends on the field type — are left
// for the caller to handle after calling Reset.
type VimNav struct {
	CountBuf string
	GBuf     bool
}

// Reset clears the accumulated count and g prefix state. Call this in any
// switch case that VimNav does not own, so the state stays consistent.
func (v *VimNav) Reset() {
	v.CountBuf, v.GBuf = "", false
}

// Handle processes a navigation key and returns (newIdx, consumed).
// consumed is true when the key was fully handled by VimNav. When false, the
// caller should process the key and then call Reset to clear accumulated state.
func (v *VimNav) Handle(key string, idx, n int) (newIdx int, consumed bool) {
	// Digit prefix accumulation
	if len(key) == 1 && key[0] >= '1' && key[0] <= '9' {
		v.CountBuf += key
		v.GBuf = false
		return idx, true
	}
	if key == "0" && v.CountBuf != "" {
		v.CountBuf += "0"
		v.GBuf = false
		return idx, true
	}

	switch key {
	case "j", "down":
		count := parseVimCount(v.CountBuf)
		v.CountBuf, v.GBuf = "", false
		for i := 0; i < count; i++ {
			if idx < n-1 {
				idx++
			}
		}
		return idx, true

	case "k", "up":
		count := parseVimCount(v.CountBuf)
		v.CountBuf, v.GBuf = "", false
		for i := 0; i < count; i++ {
			if idx > 0 {
				idx--
			}
		}
		return idx, true

	case "g":
		v.CountBuf = ""
		if v.GBuf {
			v.GBuf = false
			return 0, true // gg → top
		}
		v.GBuf = true
		return idx, true

	case "G":
		v.CountBuf, v.GBuf = "", false
		if n > 0 {
			return n - 1, true
		}
		return idx, true
	}

	// Unknown key — caller handles, should call Reset when done.
	return idx, false
}

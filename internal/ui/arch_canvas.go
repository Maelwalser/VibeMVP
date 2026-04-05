package ui

import (
	"strings"
	"unicode/utf8"
)

// canvas is a 2D rune grid for drawing ASCII art diagrams.
// All draw operations are bounds-checked; out-of-bounds writes are silently ignored.
type canvas struct {
	cells [][]rune
	w, h  int
}

func newCanvas(w, h int) canvas {
	cells := make([][]rune, h)
	for i := range cells {
		row := make([]rune, w)
		for j := range row {
			row[j] = ' '
		}
		cells[i] = row
	}
	return canvas{cells: cells, w: w, h: h}
}

func (c *canvas) set(x, y int, r rune) {
	if x >= 0 && x < c.w && y >= 0 && y < c.h {
		c.cells[y][x] = r
	}
}

// writeStr writes a string left-to-right starting at (x,y), clipping at canvas edge.
func (c *canvas) writeStr(x, y int, s string) {
	for _, r := range s {
		c.set(x, y, r)
		x++
	}
}

// hLine draws a horizontal line of '─' from x to x+length-1 at row y.
func (c *canvas) hLine(x, y, length int) {
	for i := 0; i < length; i++ {
		c.set(x+i, y, '─')
	}
}

// vLine draws a vertical line of '│' from y to y+length-1 at column x.
func (c *canvas) vLine(x, y, length int) {
	for i := 0; i < length; i++ {
		c.set(x, y+i, '│')
	}
}

// drawBox draws a box at (x,y) with outer dimensions (w,h).
// title is written centered in the top border.
// lines are written inside the box, one per row, truncated to fit.
func (c *canvas) drawBox(x, y, w, h int, title string, lines []string) {
	if w < 4 || h < 2 {
		return
	}
	// Corners
	c.set(x, y, '┌')
	c.set(x+w-1, y, '┐')
	c.set(x, y+h-1, '└')
	c.set(x+w-1, y+h-1, '┘')
	// Top/bottom edges
	c.hLine(x+1, y, w-2)
	c.hLine(x+1, y+h-1, w-2)
	// Side edges
	c.vLine(x, y+1, h-2)
	c.vLine(x+w-1, y+1, h-2)

	// Title in top border
	if title != "" {
		inner := w - 4 // space between ┌─ and ─┐
		if inner > 0 {
			t := truncate(title, inner)
			// Center the title
			pad := (inner - utf8.RuneCountInString(t)) / 2
			c.writeStr(x+2+pad, y, t)
		}
	}

	// Interior lines
	innerW := w - 2
	for i, line := range lines {
		row := y + 1 + i
		if row >= y+h-1 {
			break
		}
		content := truncate(line, innerW)
		c.writeStr(x+1, row, content)
	}
}

// arrowRight writes '▶' at (x, y).
func (c *canvas) arrowRight(x, y int) {
	c.set(x, y, '▶')
}

// drawEnvBox draws a rounded-dashed environment container box.
// Uses ╭╌╮╎╰╯ to visually distinguish from solid node boxes (┌─┐│└─┘).
// The name is written left-aligned after the top-left corner.
func (c *canvas) drawEnvBox(x, y, w, h int, name string) {
	if w < 4 || h < 2 {
		return
	}
	// Corners
	c.set(x, y, '╭')
	c.set(x+w-1, y, '╮')
	c.set(x, y+h-1, '╰')
	c.set(x+w-1, y+h-1, '╯')
	// Top/bottom edges
	for i := 1; i < w-1; i++ {
		c.set(x+i, y, '╌')
		c.set(x+i, y+h-1, '╌')
	}
	// Side edges
	for i := 1; i < h-1; i++ {
		c.set(x, y+i, '╎')
		c.set(x+w-1, y+i, '╎')
	}
	// Name label after top-left corner (e.g. ╭╌ PROD ╌╌╌╮)
	if name != "" {
		inner := w - 4
		if inner > 0 {
			label := " " + truncate(name, inner-2) + " "
			c.writeStr(x+2, y, label)
		}
	}
}

// drawConfigGroupBox draws a double-border box around services sharing a stack config.
// Uses ╔═╗║╚═╝ to distinguish from env boxes (╭╌╮) and node boxes (┌─┐).
// The label is written left-aligned after the top-left corner.
func (c *canvas) drawConfigGroupBox(x, y, w, h int, name string) {
	if w < 4 || h < 2 {
		return
	}
	c.set(x, y, '╔')
	c.set(x+w-1, y, '╗')
	c.set(x, y+h-1, '╚')
	c.set(x+w-1, y+h-1, '╝')
	for i := 1; i < w-1; i++ {
		c.set(x+i, y, '═')
		c.set(x+i, y+h-1, '═')
	}
	for i := 1; i < h-1; i++ {
		c.set(x, y+i, '║')
		c.set(x+w-1, y+i, '║')
	}
	if name != "" {
		inner := w - 4
		if inner > 0 {
			label := " " + truncate(name, inner-2) + " "
			c.writeStr(x+2, y, label)
		}
	}
}

// render converts the canvas cells to a multi-line string.
func (c canvas) render() string {
	var sb strings.Builder
	for i, row := range c.cells {
		sb.WriteString(string(row))
		if i < len(c.cells)-1 {
			sb.WriteByte('\n')
		}
	}
	return sb.String()
}

// truncate clips s to maxRunes runes, appending '…' if it was cut.
func truncate(s string, maxRunes int) string {
	if maxRunes <= 0 {
		return ""
	}
	runes := []rune(s)
	if len(runes) <= maxRunes {
		return s
	}
	if maxRunes <= 1 {
		return "…"
	}
	return string(runes[:maxRunes-1]) + "…"
}

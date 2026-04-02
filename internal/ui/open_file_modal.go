package ui

import "strings"

// viewOpenFileModal renders the "Open Manifest" dialog as a lipgloss-styled
// string suitable for passing to placeOverlay.
func (m Model) viewOpenFileModal() string {
	const modalWidth = 62

	title := StyleNeonCyan.Bold(true).Render("Open Manifest")
	divider := StyleHelpDesc.Render(strings.Repeat("─", modalWidth-4))

	prompt := StyleHelpDesc.Render("Path: ") + m.openFile.input.View()

	var errLine string
	if m.openFile.errMsg != "" {
		errLine = "\n" + StyleMsgErr.Render("  "+m.openFile.errMsg)
	}

	hint := StyleHelpDesc.Render("  Enter: load   Esc: cancel")

	body := title + "\n" + divider + "\n\n  " + prompt + errLine + "\n\n" + hint

	return StyleModalBorder.Width(modalWidth).Render(body)
}

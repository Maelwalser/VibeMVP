package arch

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/vibe-menu/internal/ui/core"
)

// envColor returns a display color keyed on environment name convention.
func envColor(name string) string {
	lower := strings.ToLower(name)
	switch {
	case strings.Contains(lower, "prod"):
		return core.ClrRed
	case strings.Contains(lower, "stag"), strings.Contains(lower, "uat"):
		return core.ClrYellow
	case strings.Contains(lower, "dev"), strings.Contains(lower, "local"):
		return core.ClrGreen
	default:
		return core.ClrBlue
	}
}

// nodeLabelByID returns the display label for a node, falling back to stripping prefixes.
func nodeLabelByID(id string, nodes []archNode) string {
	for _, n := range nodes {
		if n.id == id {
			return n.label
		}
	}
	s := strings.TrimPrefix(id, "svc.")
	s = strings.TrimPrefix(s, "db.")
	s = strings.TrimPrefix(s, "ext.")
	if s == "frontend" {
		return "Frontend"
	}
	return s
}

// ── Raw diagram rendering ─────────────────────────────────────────────────────

func buildRawArchDiagram(nodes []archNode, edges []archEdge,
	nodeInfo map[string][]string, envInfoMap map[string]string,
	configLabelMap map[string]string,
	selectedID string, termW int) archDiagramData {

	if len(nodes) == 0 {
		return archDiagramData{rawLines: strings.Split(renderEmptyArch(), "\n")}
	}

	var frontendNodes, gatewayNodes, serviceNodes, brokerNodes, dbNodes, fsNodes, extNodes []archNode
	for _, n := range nodes {
		switch n.kind {
		case archFrontend:
			frontendNodes = append(frontendNodes, n)
		case archAPIGateway:
			gatewayNodes = append(gatewayNodes, n)
		case archService:
			serviceNodes = append(serviceNodes, n)
		case archBroker:
			brokerNodes = append(brokerNodes, n)
		case archDatabase:
			dbNodes = append(dbNodes, n)
		case archFileStorage:
			fsNodes = append(fsNodes, n)
		case archExternalAPI:
			extNodes = append(extNodes, n)
		}
	}

	col0W := maxBoxWidth(frontendNodes, 22)
	colGW := maxBoxWidth(gatewayNodes, 22)
	col1W := maxBoxWidth(serviceNodes, 22)
	colBW := maxBoxWidth(brokerNodes, 22)
	col4W := maxInt(maxInt(maxBoxWidth(dbNodes, 20), maxBoxWidth(fsNodes, 20)), maxBoxWidth(extNodes, 20))

	const gap = 10
	const margin = 3

	// Build column X positions as a forward chain: empty optional columns add no space.
	x := margin
	col0X := x
	x += col0W + gap

	colGX := x
	if len(gatewayNodes) > 0 {
		x += colGW + gap
	}

	col1X := x
	x += col1W + gap

	colBX := x
	if len(brokerNodes) > 0 {
		x += colBW + gap
	}

	col4X := x

	// Column 4 height: each present section (db, fs, ext) has a 1-row section header.
	col4H := 0
	if len(dbNodes) > 0 {
		col4H += 1 + columnHeight(dbNodes)
	}
	if len(fsNodes) > 0 {
		col4H += 1 + columnHeight(fsNodes)
	}
	if len(extNodes) > 0 {
		col4H += 1 + columnHeight(extNodes)
	}

	mainH := maxInt(
		columnHeight(frontendNodes),
		columnHeight(gatewayNodes),
		columnHeight(serviceNodes),
		columnHeight(brokerNodes),
		col4H,
	)

	// Info panel
	infoLines := nodeInfo[selectedID]
	infoPanelH := 0
	if len(infoLines) > 0 {
		infoPanelH = 3 + len(infoLines)
	}

	const headerRows = 1
	totalH := headerRows + mainH + infoPanelH + 4

	totalW := col1X + col1W + margin
	if len(brokerNodes) > 0 {
		totalW = colBX + colBW + margin
	}
	if len(dbNodes) > 0 || len(fsNodes) > 0 || len(extNodes) > 0 {
		totalW = col4X + col4W + margin
	}
	if totalW < termW {
		totalW = termW
	}

	cv := newCanvas(totalW, totalH)

	yOff := headerRows

	// Column header labels — colored amber via labelColorRanges.
	var labelColorRanges []colorRange
	addLabelRange := func(x, y int, text string) {
		labelColorRanges = append(labelColorRanges, colorRange{
			xStart: x, xEnd: x + len([]rune(text)),
			yStart: y, yEnd: y + 1,
			color: core.ClrYellow, priority: 2,
		})
	}
	labelRow := yOff - 1
	if len(frontendNodes) > 0 {
		cv.writeStr(col0X, labelRow, "FRONTEND")
		addLabelRange(col0X, labelRow, "FRONTEND")
	}
	if len(gatewayNodes) > 0 {
		cv.writeStr(colGX, labelRow, "API GATEWAY")
		addLabelRange(colGX, labelRow, "API GATEWAY")
	}
	if len(serviceNodes) > 0 {
		cv.writeStr(col1X, labelRow, "BACKEND LAYER")
		addLabelRange(col1X, labelRow, "BACKEND LAYER")
	}
	if len(brokerNodes) > 0 {
		cv.writeStr(colBX, labelRow, "MSG BROKER")
		addLabelRange(colBX, labelRow, "MSG BROKER")
	}

	// Step 1: compute positions
	nodePositions := map[string]nodePos{}
	computeColumnPositions(frontendNodes, col0X, col0W, yOff, nodePositions)
	if len(gatewayNodes) > 0 {
		computeColumnPositions(gatewayNodes, colGX, colGW, yOff, nodePositions)
	}
	computeColumnPositions(serviceNodes, col1X, col1W, yOff, nodePositions)
	if len(brokerNodes) > 0 {
		computeColumnPositions(brokerNodes, colBX, colBW, yOff, nodePositions)
	}
	// Column 4: stack db / fs / ext with a 1-row section header before each group.
	col4CurY := yOff
	if len(dbNodes) > 0 {
		cv.writeStr(col4X, col4CurY, "DATA SOURCES")
		addLabelRange(col4X, col4CurY, "DATA SOURCES")
		col4CurY++
		computeColumnPositions(dbNodes, col4X, col4W, col4CurY, nodePositions)
		col4CurY += columnHeight(dbNodes)
	}
	if len(fsNodes) > 0 {
		cv.writeStr(col4X, col4CurY, "OBJECT STORAGE")
		addLabelRange(col4X, col4CurY, "OBJECT STORAGE")
		col4CurY++
		computeColumnPositions(fsNodes, col4X, col4W, col4CurY, nodePositions)
		col4CurY += columnHeight(fsNodes)
	}
	if len(extNodes) > 0 {
		cv.writeStr(col4X, col4CurY, "EXTERNAL APIS")
		addLabelRange(col4X, col4CurY, "EXTERNAL APIS")
		col4CurY++
		computeColumnPositions(extNodes, col4X, col4W, col4CurY, nodePositions)
	}

	// Step 2: environment boxes — outermost layer, spans all columns sharing an env.
	allEnvNodes := make([]archNode, 0,
		len(frontendNodes)+len(gatewayNodes)+len(serviceNodes)+
			len(brokerNodes)+len(dbNodes)+len(fsNodes)+len(extNodes))
	allEnvNodes = append(allEnvNodes, frontendNodes...)
	allEnvNodes = append(allEnvNodes, gatewayNodes...)
	allEnvNodes = append(allEnvNodes, serviceNodes...)
	allEnvNodes = append(allEnvNodes, brokerNodes...)
	allEnvNodes = append(allEnvNodes, dbNodes...)
	allEnvNodes = append(allEnvNodes, fsNodes...)
	allEnvNodes = append(allEnvNodes, extNodes...)
	envGroups := computeEnvGroups(allEnvNodes, nodePositions, envInfoMap)
	for _, eg := range envGroups {
		cv.drawEnvBox(eg.x, eg.y, eg.w, eg.h, eg.label)
	}

	// Step 3: config group boxes — middle layer, groups same-stack services within an env.
	configGroups := computeConfigGroups(serviceNodes, nodePositions, configLabelMap)
	for _, cg := range configGroups {
		cv.drawConfigGroupBox(cg.x, cg.y, cg.w, cg.h, cg.label)
	}

	// Step 4: node boxes (solid, drawn last — innermost layer)
	drawColumnBoxes(&cv, frontendNodes, nodePositions)
	drawColumnBoxes(&cv, gatewayNodes, nodePositions)
	drawColumnBoxes(&cv, serviceNodes, nodePositions)
	drawColumnBoxes(&cv, brokerNodes, nodePositions)
	drawColumnBoxes(&cv, dbNodes, nodePositions)
	drawColumnBoxes(&cv, fsNodes, nodePositions)
	drawColumnBoxes(&cv, extNodes, nodePositions)

	// Step 5: arrows + collect edge bounds
	edgeBoundsMap := map[string]edgeBounds{}
	edgePathMap := map[string][]edgeBounds{}

	// drawEdgeArrow draws a left-to-right (→) arrow between two nodes in different columns.
	// The vertical connector is placed at toX-gap/2 (near the target) so it avoids
	// crossing through any intermediate column boxes.
	drawEdgeArrow := func(e archEdge) {
		from, ok1 := nodePositions[e.fromID]
		to, ok2 := nodePositions[e.toID]
		if !ok1 || !ok2 {
			return
		}
		fromX := from.x + from.w - 1
		toX := to.x
		arrowY := midY(from)
		toY := midY(to)

		var bx1, bx2, by1, by2 int
		var pathSegs []edgeBounds
		if arrowY == toY {
			cv.hLine(fromX, arrowY, toX-fromX)
			cv.arrowRight(toX, arrowY)
			bx1, bx2, by1, by2 = fromX, toX+1, arrowY, arrowY+1
			pathSegs = []edgeBounds{{x1: fromX, x2: toX + 1, y1: arrowY, y2: arrowY + 1}}
		} else {
			// Route near the target column: vertical connector sits in the gap just
			// before the target, avoiding any intermediate column boxes.
			midX := toX - gap/2
			if midX <= fromX {
				midX = fromX + 1 // fallback: adjacent columns
			}
			cv.hLine(fromX, arrowY, midX-fromX)
			if arrowY < toY {
				cv.vLine(midX, arrowY, toY-arrowY+1)
			} else {
				cv.vLine(midX, toY, arrowY-toY+1)
			}
			cv.hLine(midX, toY, toX-midX)
			cv.arrowRight(toX, toY)
			bx1, bx2 = fromX, toX+1
			vTop, vBot := arrowY, toY
			if arrowY < toY {
				by1, by2 = arrowY, toY+1
			} else {
				by1, by2 = toY, arrowY+1
				vTop, vBot = toY, arrowY
			}
			pathSegs = []edgeBounds{
				{x1: fromX, x2: midX + 1, y1: arrowY, y2: arrowY + 1},
				{x1: midX, x2: midX + 1, y1: vTop, y2: vBot + 1},
				{x1: midX, x2: toX + 1, y1: toY, y2: toY + 1},
			}
		}
		edgeBoundsMap[e.id] = edgeBounds{x1: bx1, x2: bx2, y1: by1, y2: by2}
		edgePathMap[e.id] = pathSegs
	}

	// Frontend → Gateway or Service arrows
	for _, e := range edges {
		if e.fromID == "frontend" {
			drawEdgeArrow(e)
		}
	}
	// Gateway → Service arrows
	for _, e := range edges {
		if e.fromID == "gateway" && strings.HasPrefix(e.toID, "svc.") {
			drawEdgeArrow(e)
		}
	}
	// Service → Database arrows
	for _, e := range edges {
		if strings.HasPrefix(e.fromID, "svc.") && strings.HasPrefix(e.toID, "db.") {
			drawEdgeArrow(e)
		}
	}
	// Service → Service arrows: routed on the right side of the services column.
	// Path: exit source's right edge → 3 chars right (bypass) → vertical to target row
	//       → 3 chars back left to target's right edge → ◀ at target entry.
	for _, e := range edges {
		if !strings.HasPrefix(e.fromID, "svc.") || !strings.HasPrefix(e.toID, "svc.") {
			continue
		}
		from, ok1 := nodePositions[e.fromID]
		to, ok2 := nodePositions[e.toID]
		if !ok1 || !ok2 || e.fromID == e.toID {
			continue
		}
		fromY := midY(from)
		toY := midY(to)
		exitX := from.x + from.w // first cell right of source box
		bypassX := exitX + 3     // vertical connector column

		// Source: horizontal exit
		cv.hLine(exitX, fromY, 3)
		// Vertical connector
		lo, hi := fromY, toY
		if lo > hi {
			lo, hi = hi, lo
		}
		cv.vLine(bypassX, lo, hi-lo+1)
		// Target: horizontal return (leave room for arrow at target exit cell)
		cv.hLine(to.x+to.w+1, toY, bypassX-(to.x+to.w))
		// Arrow pointing left into target from the right
		cv.arrowLeft(to.x+to.w, toY)
		// Bidirectional: also mark source with a left-pointing arrow (traffic returns)
		if e.direction == dirBidirectional {
			cv.arrowLeft(exitX, fromY)
		}

		bx1, bx2 := exitX, bypassX+1
		by1, by2 := lo, hi+1
		edgeBoundsMap[e.id] = edgeBounds{x1: bx1, x2: bx2, y1: by1, y2: by2}
		// Precise path: three segments instead of the bounding rectangle.
		edgePathMap[e.id] = []edgeBounds{
			{x1: exitX, x2: bypassX + 1, y1: fromY, y2: fromY + 1},   // horizontal exit
			{x1: bypassX, x2: bypassX + 1, y1: lo, y2: hi + 1},       // vertical connector
			{x1: to.x + to.w, x2: bypassX + 1, y1: toY, y2: toY + 1}, // horizontal entry + arrow
		}
	}

	// Service → Broker arrows (left-to-right using drawEdgeArrow).
	for _, e := range edges {
		if strings.HasPrefix(e.fromID, "svc.") && e.toID == "broker" {
			drawEdgeArrow(e)
		}
	}

	// drawRtoLArrow draws a right-to-left (←) arrow: broker → service.
	// The broker is to the right; the arrow enters the service's right edge with ◀.
	drawRtoLArrow := func(e archEdge) {
		from, ok1 := nodePositions[e.fromID] // broker
		to, ok2 := nodePositions[e.toID]     // service
		if !ok1 || !ok2 {
			return
		}
		fromX := from.x    // left edge of broker (arrow exits here)
		toX := to.x + to.w // right exit of service (just outside the box)
		fromY := midY(from)
		toY := midY(to)

		var bx1, bx2, by1, by2 int
		var pathSegs []edgeBounds
		if fromY == toY {
			cv.hLine(toX, fromY, fromX-toX)
			cv.arrowLeft(toX, fromY)
			bx1, bx2, by1, by2 = toX, fromX, fromY, fromY+1
			pathSegs = []edgeBounds{{x1: toX, x2: fromX + 1, y1: fromY, y2: fromY + 1}}
		} else {
			midX := toX + (fromX-toX)/2 // midpoint in the gap
			cv.hLine(midX, fromY, fromX-midX)
			lo, hi := fromY, toY
			if lo > hi {
				lo, hi = hi, lo
			}
			cv.vLine(midX, lo, hi-lo+1)
			cv.hLine(toX, toY, midX-toX)
			cv.arrowLeft(toX, toY)
			bx1, bx2 = toX, fromX
			if fromY < toY {
				by1, by2 = fromY, toY+1
			} else {
				by1, by2 = toY, fromY+1
			}
			pathSegs = []edgeBounds{
				{x1: midX, x2: fromX + 1, y1: fromY, y2: fromY + 1},
				{x1: midX, x2: midX + 1, y1: lo, y2: hi + 1},
				{x1: toX, x2: midX + 1, y1: toY, y2: toY + 1},
			}
		}
		edgeBoundsMap[e.id] = edgeBounds{x1: bx1, x2: bx2, y1: by1, y2: by2}
		edgePathMap[e.id] = pathSegs
	}

	// Broker → Service (consumer) arrows using drawRtoLArrow.
	for _, e := range edges {
		if e.fromID == "broker" && strings.HasPrefix(e.toID, "svc.") {
			drawRtoLArrow(e)
		}
	}

	// Service → FileStorage arrows.
	for _, e := range edges {
		if strings.HasPrefix(e.fromID, "svc.") && strings.HasPrefix(e.toID, "fs.") {
			drawEdgeArrow(e)
		}
	}

	// Service → ExternalAPI arrows.
	for _, e := range edges {
		if strings.HasPrefix(e.fromID, "svc.") && strings.HasPrefix(e.toID, "ext.") {
			drawEdgeArrow(e)
		}
	}

	// Step 6: info panel for selected node or edge — compute content and colors.
	var infoColorRanges []colorRange
	if len(infoLines) > 0 {
		panelY := yOff + mainH + 2

		var header string
		if strings.HasPrefix(selectedID, "edge.") {
			for _, e := range edges {
				if e.id != selectedID {
					continue
				}
				fromLabel := nodeLabelByID(e.fromID, nodes)
				toLabel := nodeLabelByID(e.toID, nodes)
				header = "  COMMUNICATION: " + fromLabel + " → " + toLabel
				break
			}
		} else {
			for _, n := range nodes {
				if n.id == selectedID {
					header = "  SELECTED: " + n.label
					break
				}
			}
		}
		if header == "" {
			header = "  INFO: " + selectedID
		}

		cv.writeStr(0, panelY, header)
		cv.hLine(0, panelY+1, totalW-1)
		for i, line := range infoLines {
			cv.writeStr(0, panelY+2+i, line)
		}

		// Color the header line (SELECTED / COMMUNICATION).
		infoColorRanges = append(infoColorRanges, colorRange{
			xStart: 0, xEnd: len([]rune(header)),
			yStart: panelY, yEnd: panelY + 1,
			color: core.ClrYellow, priority: 4,
		})
		// Color individual info lines by content pattern.
		infoColorRanges = append(infoColorRanges, buildInfoLineColors(infoLines, panelY+2)...)
	}

	return archDiagramData{
		rawLines:         strings.Split(cv.render(), "\n"),
		nodePositions:    nodePositions,
		edgeBoundsMap:    edgeBoundsMap,
		edgePathMap:      edgePathMap,
		envGroups:        envGroups,
		frontendNodes:    frontendNodes,
		gatewayNodes:     gatewayNodes,
		serviceNodes:     serviceNodes,
		brokerNodes:      brokerNodes,
		dbNodes:          dbNodes,
		fsNodes:          fsNodes,
		extNodes:         extNodes,
		selectedID:       selectedID,
		infoColorRanges:  infoColorRanges,
		labelColorRanges: labelColorRanges,
	}
}

// buildInfoLineColors returns color ranges for lines in the info panel.
// - Section headers like "  Endpoints:" → ClrBlue
// - Deeply indented items (4+ spaces) → ClrFgDim
// baseY is the absolute canvas Y of the first info line.
func buildInfoLineColors(lines []string, baseY int) []colorRange {
	var ranges []colorRange
	for i, line := range lines {
		y := baseY + i
		trimmed := strings.TrimLeft(line, " ")
		indent := len(line) - len(trimmed)
		switch {
		case indent == 2 && strings.HasSuffix(trimmed, ":"):
			// Section header: "  Endpoints:", "  Protocol:", etc.
			ranges = append(ranges, colorRange{
				xStart: 0, xEnd: len([]rune(line)),
				yStart: y, yEnd: y + 1,
				color: core.ClrBlue, priority: 4,
			})
		case indent >= 4:
			// Indented list items and sub-values.
			ranges = append(ranges, colorRange{
				xStart: 0, xEnd: len([]rune(line)),
				yStart: y, yEnd: y + 1,
				color: core.ClrFgDim, priority: 4,
			})
		}
	}
	return ranges
}

func renderEmptyArch() string {
	return strings.Join([]string{
		"",
		"  Fill in sections to see the architecture diagram.",
		"",
		"  Configure backend services, data sources, and frontend pages to visualize connections.",
		"",
	}, "\n")
}

// ── Layout helpers ────────────────────────────────────────────────────────────

type nodePos struct {
	x, y, w, h int
}

func midY(p nodePos) int { return p.y + p.h/2 }

func computeColumnPositions(nodes []archNode, colX, boxW, startY int, positions map[string]nodePos) {
	y := startY
	for _, n := range nodes {
		boxH := 2 + len(n.details())
		if boxH < 3 {
			boxH = 3
		}
		positions[n.id] = nodePos{x: colX, y: y, w: boxW, h: boxH}
		y += boxH + 1
	}
}

func drawColumnBoxes(cv *canvas, nodes []archNode, positions map[string]nodePos) {
	for _, n := range nodes {
		if p, ok := positions[n.id]; ok {
			cv.drawBox(p.x, p.y, p.w, p.h, n.label, nil)
		}
	}
}

// details returns the node's interior lines. Nodes are black boxes — no details.
func (n archNode) details() []string { return nil }

func computeEnvGroups(nodes []archNode, positions map[string]nodePos, envInfoMap map[string]string) []archEnvGroup {
	order := []string{}
	byEnv := map[string][]nodePos{}
	for _, n := range nodes {
		if n.environment == "" {
			continue
		}
		if _, exists := byEnv[n.environment]; !exists {
			order = append(order, n.environment)
		}
		if p, ok := positions[n.id]; ok {
			byEnv[n.environment] = append(byEnv[n.environment], p)
		}
	}

	var groups []archEnvGroup
	for _, envName := range order {
		poses := byEnv[envName]
		if len(poses) == 0 {
			continue
		}
		minX, minY := poses[0].x, poses[0].y
		maxX, maxY := poses[0].x+poses[0].w, poses[0].y+poses[0].h
		for _, p := range poses[1:] {
			if p.x < minX {
				minX = p.x
			}
			if p.y < minY {
				minY = p.y
			}
			if p.x+p.w > maxX {
				maxX = p.x + p.w
			}
			if p.y+p.h > maxY {
				maxY = p.y + p.h
			}
		}
		label := strings.ToUpper(envName)
		if info, ok := envInfoMap[envName]; ok && info != "" {
			label += " · " + info
		}
		groups = append(groups, archEnvGroup{
			label: label,
			color: envColor(envName),
			x:     minX - 2,
			y:     minY - 2,
			w:     maxX - minX + 4,
			h:     maxY - minY + 4,
		})
	}
	return groups
}

// computeConfigGroups builds containers for services sharing the same ConfigRef.
// Only draws a box when 2+ services share the same config (single services need no box).
func computeConfigGroups(nodes []archNode, positions map[string]nodePos, configLabelMap map[string]string) []archEnvGroup {
	countByConfig := map[string]int{}
	for _, n := range nodes {
		if n.configRef != "" {
			countByConfig[n.configRef]++
		}
	}

	order := []string{}
	byConfig := map[string][]nodePos{}
	for _, n := range nodes {
		if n.configRef == "" || countByConfig[n.configRef] < 2 {
			continue
		}
		if _, exists := byConfig[n.configRef]; !exists {
			order = append(order, n.configRef)
		}
		if p, ok := positions[n.id]; ok {
			byConfig[n.configRef] = append(byConfig[n.configRef], p)
		}
	}

	var groups []archEnvGroup
	for _, configRef := range order {
		poses := byConfig[configRef]
		if len(poses) == 0 {
			continue
		}
		minX, minY := poses[0].x, poses[0].y
		maxX, maxY := poses[0].x+poses[0].w, poses[0].y+poses[0].h
		for _, p := range poses[1:] {
			if p.x < minX {
				minX = p.x
			}
			if p.y < minY {
				minY = p.y
			}
			if p.x+p.w > maxX {
				maxX = p.x + p.w
			}
			if p.y+p.h > maxY {
				maxY = p.y + p.h
			}
		}
		label := strings.ToUpper(configRef)
		if techLabel, ok := configLabelMap[configRef]; ok && techLabel != "" {
			label += " · " + techLabel
		}
		groups = append(groups, archEnvGroup{
			label: label,
			x:     minX - 1,
			y:     minY - 1,
			w:     maxX - minX + 2,
			h:     maxY - minY + 2,
		})
	}
	return groups
}

func columnHeight(nodes []archNode) int {
	total := 0
	for _, n := range nodes {
		h := 2 + len(n.details())
		if h < 3 {
			h = 3
		}
		total += h + 1
	}
	return total
}

func maxBoxWidth(nodes []archNode, minW int) int {
	w := minW
	for _, n := range nodes {
		if l := len([]rune(n.label)) + 4; l > w {
			w = l
		}
	}
	return w
}

func maxInt(vals ...int) int {
	m := 0
	for _, v := range vals {
		if v > m {
			m = v
		}
	}
	return m
}

// ── Colorization ──────────────────────────────────────────────────────────────

type colorRange struct {
	xStart, xEnd int
	yStart, yEnd int
	color        string
	bold         bool
	priority     int
}

// colorizeClipped applies ANSI colors to already-clipped plain-text lines.
// Positions are adjusted by (scrollX, scrollY) to map into the viewport.
func colorizeClipped(lines []string, data archDiagramData, scrollX, scrollY int) string {
	var ranges []colorRange

	// Env box regions — priority 1 (background)
	for _, eg := range data.envGroups {
		ranges = append(ranges, colorRange{
			xStart:   eg.x - scrollX,
			xEnd:     eg.x + eg.w - scrollX,
			yStart:   eg.y - scrollY,
			yEnd:     eg.y + eg.h - scrollY,
			color:    eg.color,
			priority: 1,
		})
	}

	// Label color ranges — priority 2 (column/section header labels)
	for _, cr := range data.labelColorRanges {
		ranges = append(ranges, colorRange{
			xStart:   cr.xStart - scrollX,
			xEnd:     cr.xEnd - scrollX,
			yStart:   cr.yStart - scrollY,
			yEnd:     cr.yEnd - scrollY,
			color:    cr.color,
			priority: cr.priority,
		})
	}

	// Node box regions — priority 2 (foreground)
	addRanges := func(nodes []archNode, color string) {
		for _, n := range nodes {
			if p, ok := data.nodePositions[n.id]; ok {
				ranges = append(ranges, colorRange{
					xStart:   p.x - scrollX,
					xEnd:     p.x + p.w - scrollX,
					yStart:   p.y - scrollY,
					yEnd:     p.y + p.h - scrollY,
					color:    color,
					priority: 2,
				})
			}
		}
	}
	addRanges(data.frontendNodes, core.ClrCyan)
	addRanges(data.gatewayNodes, core.ClrBlue)
	addRanges(data.serviceNodes, core.ClrGreen)
	addRanges(data.brokerNodes, core.ClrRed)
	addRanges(data.dbNodes, core.ClrYellow)
	addRanges(data.fsNodes, core.ClrYellow)
	addRanges(data.extNodes, core.ClrMagenta)

	// Unselected edge paths — dim so the selected edge pops (priority 2).
	for edgeID, segs := range data.edgePathMap {
		if edgeID == data.selectedID {
			continue // selected edge gets bright highlight below
		}
		for _, seg := range segs {
			ranges = append(ranges, colorRange{
				xStart:   seg.x1 - scrollX,
				xEnd:     seg.x2 - scrollX,
				yStart:   seg.y1 - scrollY,
				yEnd:     seg.y2 - scrollY,
				color:    core.ClrFgDim,
				priority: 2,
			})
		}
	}

	// Info panel colors — priority 4 (above node colors)
	for _, cr := range data.infoColorRanges {
		ranges = append(ranges, colorRange{
			xStart:   cr.xStart - scrollX,
			xEnd:     cr.xEnd - scrollX,
			yStart:   cr.yStart - scrollY,
			yEnd:     cr.yEnd - scrollY,
			color:    cr.color,
			priority: cr.priority,
		})
	}

	// Selected item highlight — priority 3.
	// Blinks between pure white (bold) and bright gold (bold) for nodes,
	// and between bright sky-blue (bold) and white (bold) for edges,
	// so the selection always pops clearly against the muted palette.
	const selColorA = "#FFFFFF" // bright white — "on" frame for nodes, "off" for edges
	const selColorB = "#F0CC6A" // bright gold  — "off" frame for nodes
	const selEdgeA = "#6DD9D9"  // bright teal  — "on" frame for edges
	if data.selectedID != "" {
		if strings.HasPrefix(data.selectedID, "edge.") {
			// Use precise path segments so only actual edge pixels blink.
			edgeColor := selEdgeA
			if animFrame()%2 == 0 {
				edgeColor = selColorA
			}
			if segs, ok := data.edgePathMap[data.selectedID]; ok && len(segs) > 0 {
				for _, seg := range segs {
					ranges = append(ranges, colorRange{
						xStart:   seg.x1 - scrollX,
						xEnd:     seg.x2 - scrollX,
						yStart:   seg.y1 - scrollY,
						yEnd:     seg.y2 - scrollY,
						color:    edgeColor,
						bold:     true,
						priority: 3,
					})
				}
			} else if eb, ok := data.edgeBoundsMap[data.selectedID]; ok {
				// Fallback to bounding box if path segments are unavailable.
				ranges = append(ranges, colorRange{
					xStart:   eb.x1 - scrollX,
					xEnd:     eb.x2 - scrollX,
					yStart:   eb.y1 - scrollY,
					yEnd:     eb.y2 - scrollY,
					color:    edgeColor,
					bold:     true,
					priority: 3,
				})
			}
		} else {
			// Highlight the selected node box with bold so it pops.
			if p, ok := data.nodePositions[data.selectedID]; ok {
				blinkColor := selColorA
				if animFrame()%2 == 0 {
					blinkColor = selColorB
				}
				ranges = append(ranges, colorRange{
					xStart:   p.x - scrollX,
					xEnd:     p.x + p.w - scrollX,
					yStart:   p.y - scrollY,
					yEnd:     p.y + p.h - scrollY,
					color:    blinkColor,
					bold:     true,
					priority: 3,
				})
			}
		}
	}

	result := make([]string, len(lines))
	for y, line := range lines {
		result[y] = applyColorRanges(line, y, ranges)
	}
	return strings.Join(result, "\n")
}

// applyColorRanges colorizes one line using the per-character priority system.
func applyColorRanges(line string, y int, ranges []colorRange) string {
	runes := []rune(line)
	if len(runes) == 0 {
		return line
	}

	colorAt := make([]string, len(runes))
	boldAt := make([]bool, len(runes))
	prioAt := make([]int, len(runes))

	for _, cr := range ranges {
		if y < cr.yStart || y >= cr.yEnd {
			continue
		}
		start := cr.xStart
		if start < 0 {
			start = 0
		}
		end := cr.xEnd
		if end > len(runes) {
			end = len(runes)
		}
		for x := start; x < end; x++ {
			if cr.priority >= prioAt[x] {
				colorAt[x] = cr.color
				boldAt[x] = cr.bold
				prioAt[x] = cr.priority
			}
		}
	}

	var sb strings.Builder
	i := 0
	for i < len(runes) {
		col := colorAt[i]
		bld := boldAt[i]
		j := i + 1
		for j < len(runes) && colorAt[j] == col && boldAt[j] == bld {
			j++
		}
		chunk := string(runes[i:j])
		if col != "" || bld {
			style := lipgloss.NewStyle()
			if col != "" {
				style = style.Foreground(lipgloss.Color(col))
			}
			if bld {
				style = style.Bold(true)
			}
			chunk = style.Render(chunk)
		}
		sb.WriteString(chunk)
		i = j
	}
	return sb.String()
}

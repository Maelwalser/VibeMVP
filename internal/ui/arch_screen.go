package ui

import (
	"fmt"
	"strconv"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/vibe-menu/internal/manifest"
)

// ── Node / Edge types ─────────────────────────────────────────────────────────

type archNodeKind int

const (
	archFrontend archNodeKind = iota
	archService
	archDatabase
	archExternalAPI
)

type archNode struct {
	id          string
	kind        archNodeKind
	label       string
	environment string
	configRef   string // references BackendPillar.StackConfigs[*].Name
}

// archEdge direction constants — mirror CommLink.Direction values.
const (
	dirUnidirectional = "unidirectional"
	dirBidirectional  = "bidirectional"
)

type archEdge struct {
	id        string // "edge.N" — unique index-based ID
	fromID    string
	toID      string
	label     string
	direction string
}

// edgeBounds is the bounding rectangle of a drawn arrow on the canvas.
type edgeBounds struct {
	x1, x2, y1, y2 int
}

// archEnvGroup is a bounding rect drawn around nodes sharing an environment or config.
type archEnvGroup struct {
	label string
	color string
	x, y  int
	w, h  int
}

// archDiagramData holds the raw render output plus metadata for colorization.
type archDiagramData struct {
	rawLines      []string
	nodePositions map[string]nodePos
	edgeBoundsMap map[string]edgeBounds
	envGroups     []archEnvGroup
	frontendNodes []archNode
	serviceNodes  []archNode
	dbNodes       []archNode
	extNodes      []archNode
	selectedID    string
}

// ── Screen state ──────────────────────────────────────────────────────────────

// ArchScreen is the full-screen architecture overview overlay.
type ArchScreen struct {
	nodes          []archNode
	edges          []archEdge
	nodeInfo       map[string][]string // node or edge id → detail lines shown in info panel
	envInfoMap     map[string]string   // env name → "ComputeEnv · CloudProvider"
	configLabelMap map[string]string   // ConfigRef name → "Language · Framework"
	selectedID     string
	scrollY        int
	scrollX        int
	wantsQuit      bool
}

func newArchScreen() ArchScreen { return ArchScreen{} }

// Open rebuilds the graph from the given manifest and resets scroll.
func (s ArchScreen) Open(mf *manifest.Manifest) ArchScreen {
	nodes, edges := buildArchGraph(mf)
	firstID := ""
	if len(nodes) > 0 {
		firstID = nodes[0].id
	}
	return ArchScreen{
		nodes:          nodes,
		edges:          edges,
		nodeInfo:       buildAllInfo(mf, nodes, edges),
		envInfoMap:     buildEnvInfoMap(mf),
		configLabelMap: buildConfigLabelMap(mf),
		selectedID:     firstID,
	}
}

func (s ArchScreen) WantsQuit() bool { return s.wantsQuit }

func (s ArchScreen) HintLine() string {
	return lipgloss.NewStyle().Foreground(lipgloss.Color(clrFgDim)).
		Render("  h/l navigate · j/k cycle · H/L pan · g reset · q close")
}

func (s ArchScreen) Update(msg tea.Msg) (ArchScreen, tea.Cmd) {
	key, ok := msg.(tea.KeyMsg)
	if !ok {
		return s, nil
	}
	switch key.String() {
	case "q", "esc", "P":
		s.wantsQuit = true
	case "l":
		s = s.navigateRight()
	case "h":
		s = s.navigateLeft()
	case "j", "down":
		s = s.navigateDown()
	case "k", "up":
		s = s.navigateUp()
	case "L": // Shift+L — pan right
		s.scrollX++
	case "H": // Shift+H — pan left
		if s.scrollX > 0 {
			s.scrollX--
		}
	case "g":
		s.scrollY = 0
		s.scrollX = 0
	}
	return s, nil
}

// View renders the diagram. Pipeline: build raw canvas → clip plain text → colorize.
// Clipping on plain text avoids ANSI-code rune inflation that hides right-hand columns.
func (s ArchScreen) View(w, h int) string {
	data := buildRawArchDiagram(s.nodes, s.edges, s.nodeInfo, s.envInfoMap, s.configLabelMap, s.selectedID, w)
	rawLines := data.rawLines
	total := len(rawLines)

	sy := s.autoScrollY(data, s.scrollY, h)
	sx := s.scrollX

	maxScrollY := total - h
	if maxScrollY < 0 {
		maxScrollY = 0
	}
	if sy > maxScrollY {
		sy = maxScrollY
	}

	// Clip raw lines — rune count is accurate on plain text.
	clipped := make([]string, 0, h)
	for i := sy; i < sy+h && i < total; i++ {
		runes := []rune(rawLines[i])
		if sx < len(runes) {
			runes = runes[sx:]
		} else {
			runes = nil
		}
		if len(runes) > w {
			runes = runes[:w]
		}
		clipped = append(clipped, string(runes))
	}
	for len(clipped) < h {
		clipped = append(clipped, "")
	}

	return colorizeClipped(clipped, data, sx, sy)
}

// autoScrollY returns a scrollY that keeps the selected item inside the viewport.
func (s ArchScreen) autoScrollY(data archDiagramData, currentScroll, viewH int) int {
	if strings.HasPrefix(s.selectedID, "edge.") {
		if eb, ok := data.edgeBoundsMap[s.selectedID]; ok {
			midYPos := (eb.y1 + eb.y2) / 2
			if midYPos < currentScroll {
				return midYPos
			}
			if midYPos >= currentScroll+viewH {
				v := midYPos - viewH/2
				if v < 0 {
					v = 0
				}
				return v
			}
		}
		return currentScroll
	}
	if p, ok := data.nodePositions[s.selectedID]; ok {
		nodeBottom := p.y + p.h
		if nodeBottom > currentScroll+viewH {
			return nodeBottom - viewH
		}
		if p.y < currentScroll {
			return p.y
		}
	}
	return currentScroll
}

// ── Navigation helpers ────────────────────────────────────────────────────────

func nodeColumn(kind archNodeKind) int {
	switch kind {
	case archFrontend:
		return 0
	case archService:
		return 1
	case archDatabase:
		return 2
	case archExternalAPI:
		return 3
	}
	return -1
}

func (s ArchScreen) nodeColumnByID(id string) int {
	for _, n := range s.nodes {
		if n.id == id {
			return nodeColumn(n.kind)
		}
	}
	return 0
}

func (s ArchScreen) nodesInColumn(col int) []archNode {
	var result []archNode
	for _, n := range s.nodes {
		if nodeColumn(n.kind) == col {
			result = append(result, n)
		}
	}
	return result
}

func (s ArchScreen) selectedEdge() (archEdge, bool) {
	for _, e := range s.edges {
		if e.id == s.selectedID {
			return e, true
		}
	}
	return archEdge{}, false
}

// siblingEdges returns all edges that share the same column-pair as e.
func (s ArchScreen) siblingEdges(e archEdge) []archEdge {
	fromCol := s.nodeColumnByID(e.fromID)
	toCol := s.nodeColumnByID(e.toID)
	var result []archEdge
	for _, edge := range s.edges {
		if s.nodeColumnByID(edge.fromID) == fromCol && s.nodeColumnByID(edge.toID) == toCol {
			result = append(result, edge)
		}
	}
	return result
}

// navigateRight: node → select outgoing edge; edge → go to target node.
func (s ArchScreen) navigateRight() ArchScreen {
	if e, ok := s.selectedEdge(); ok {
		s.selectedID = e.toID
		return s
	}
	col := s.nodeColumnByID(s.selectedID)
	for _, e := range s.edges {
		if e.fromID == s.selectedID && s.nodeColumnByID(e.toID) > col {
			s.selectedID = e.id
			return s
		}
	}
	// No direct edge — jump to first node in next column
	nextCol := col + 1
	for _, n := range s.nodes {
		if nodeColumn(n.kind) == nextCol {
			s.selectedID = n.id
			return s
		}
	}
	return s
}

// navigateLeft: node → select incoming edge; edge → go to source node.
func (s ArchScreen) navigateLeft() ArchScreen {
	if e, ok := s.selectedEdge(); ok {
		s.selectedID = e.fromID
		return s
	}
	col := s.nodeColumnByID(s.selectedID)
	for _, e := range s.edges {
		if e.toID == s.selectedID && s.nodeColumnByID(e.fromID) < col {
			s.selectedID = e.id
			return s
		}
	}
	prevCol := col - 1
	if prevCol < 0 {
		return s
	}
	for _, n := range s.nodes {
		if nodeColumn(n.kind) == prevCol {
			s.selectedID = n.id
			return s
		}
	}
	return s
}

// navigateDown: cycle to next node in column, or next sibling edge.
func (s ArchScreen) navigateDown() ArchScreen {
	if e, ok := s.selectedEdge(); ok {
		siblings := s.siblingEdges(e)
		for i, sib := range siblings {
			if sib.id == e.id {
				s.selectedID = siblings[(i+1)%len(siblings)].id
				return s
			}
		}
		return s
	}
	col := s.nodeColumnByID(s.selectedID)
	nodes := s.nodesInColumn(col)
	for i, n := range nodes {
		if n.id == s.selectedID {
			s.selectedID = nodes[(i+1)%len(nodes)].id
			return s
		}
	}
	return s
}

// navigateUp: cycle to previous node in column, or previous sibling edge.
func (s ArchScreen) navigateUp() ArchScreen {
	if e, ok := s.selectedEdge(); ok {
		siblings := s.siblingEdges(e)
		for i, sib := range siblings {
			if sib.id == e.id {
				s.selectedID = siblings[(i-1+len(siblings))%len(siblings)].id
				return s
			}
		}
		return s
	}
	col := s.nodeColumnByID(s.selectedID)
	nodes := s.nodesInColumn(col)
	for i, n := range nodes {
		if n.id == s.selectedID {
			s.selectedID = nodes[(i-1+len(nodes))%len(nodes)].id
			return s
		}
	}
	return s
}

// ── Graph building ────────────────────────────────────────────────────────────

func buildEnvInfoMap(mf *manifest.Manifest) map[string]string {
	m := map[string]string{}
	for _, env := range mf.Infra.Environments {
		if env.Name == "" {
			continue
		}
		var parts []string
		if env.ComputeEnv != "" {
			parts = append(parts, env.ComputeEnv)
		}
		if env.CloudProvider != "" {
			parts = append(parts, env.CloudProvider)
		}
		if env.Orchestrator != "" && env.Orchestrator != env.ComputeEnv {
			parts = append(parts, env.Orchestrator)
		}
		m[env.Name] = strings.Join(parts, " · ")
	}
	return m
}

// buildConfigLabelMap maps each StackConfig name to its "Language · Framework" label.
func buildConfigLabelMap(mf *manifest.Manifest) map[string]string {
	m := map[string]string{}
	for _, sc := range mf.Backend.StackConfigs {
		if sc.Name == "" {
			continue
		}
		label := sc.Language
		if sc.Framework != "" {
			label += " · " + sc.Framework
		}
		if sc.LanguageVersion != "" {
			label += " " + sc.LanguageVersion
		}
		m[sc.Name] = label
	}
	return m
}

func buildArchGraph(mf *manifest.Manifest) ([]archNode, []archEdge) {
	var nodes []archNode
	var edges []archEdge

	// Frontend — label only
	if mf.Frontend.Tech != nil && mf.Frontend.Tech.Language != "" {
		tech := mf.Frontend.Tech
		label := tech.Language
		if tech.Framework != "" {
			label += " / " + tech.Framework
		}
		nodes = append(nodes, archNode{id: "frontend", kind: archFrontend, label: label})
	}

	// Monolith arch: all services share the pillar-level MonolithEnvironment when
	// the individual service has no Environment set.
	isMonolith := mf.Backend.ArchPattern == manifest.ArchMonolith ||
		mf.Backend.ArchPattern == manifest.ArchModularMonolith

	// Services — label + configRef for grouping
	for _, svc := range mf.Backend.Services {
		if svc.Name == "" {
			continue
		}
		env := svc.Environment
		if env == "" && isMonolith {
			env = mf.Backend.MonolithEnvironment
		}
		nodes = append(nodes, archNode{
			id:          "svc." + svc.Name,
			kind:        archService,
			label:       svc.Name,
			environment: env,
			configRef:   svc.ConfigRef,
		})
	}

	// Databases — label only
	for _, db := range mf.Data.Databases {
		if db.Alias == "" {
			continue
		}
		nodes = append(nodes, archNode{
			id:          "db." + db.Alias,
			kind:        archDatabase,
			label:       db.Alias,
			environment: db.Environment,
		})
	}

	// External APIs — label only
	for _, api := range mf.Contracts.ExternalAPIs {
		if api.Provider == "" {
			continue
		}
		nodes = append(nodes, archNode{
			id:    "ext." + api.Provider,
			kind:  archExternalAPI,
			label: api.Provider,
		})
	}

	// Edges: Frontend → Services via component actions
	seen := map[string]bool{}
	for _, comp := range mf.Frontend.Components {
		for _, action := range comp.Actions {
			if action.Endpoint == "" {
				continue
			}
			ep := findEndpoint(mf, action.Endpoint)
			if ep == nil || ep.ServiceUnit == "" {
				continue
			}
			key := "frontend|svc." + ep.ServiceUnit
			if seen[key] {
				continue
			}
			seen[key] = true
			eID := "edge." + strconv.Itoa(len(edges))
			edges = append(edges, archEdge{
				id:        eID,
				fromID:    "frontend",
				toID:      "svc." + ep.ServiceUnit,
				direction: dirUnidirectional,
			})
		}
	}
	// Implied edges when frontend is configured but has no explicit actions
	if len(edges) == 0 && mf.Frontend.Tech != nil && mf.Frontend.Tech.Language != "" {
		for _, ep := range mf.Contracts.Endpoints {
			if ep.ServiceUnit == "" {
				continue
			}
			key := "frontend|svc." + ep.ServiceUnit
			if seen[key] {
				continue
			}
			seen[key] = true
			eID := "edge." + strconv.Itoa(len(edges))
			edges = append(edges, archEdge{
				id:        eID,
				fromID:    "frontend",
				toID:      "svc." + ep.ServiceUnit,
				direction: dirUnidirectional,
			})
		}
	}

	// Edges: Service → Service (direction-aware)
	for _, link := range mf.Backend.CommLinks {
		if link.From == "" || link.To == "" {
			continue
		}
		dir := dirUnidirectional
		if strings.EqualFold(link.Direction, "bidirectional") {
			dir = dirBidirectional
		}
		eID := "edge." + strconv.Itoa(len(edges))
		edges = append(edges, archEdge{
			id:        eID,
			fromID:    "svc." + link.From,
			toID:      "svc." + link.To,
			label:     link.Protocol,
			direction: dir,
		})
	}

	// Edges: Service → Database (via RepositoryDef.TargetDB)
	dbEdgeSeen := map[string]bool{}
	for _, svc := range mf.Backend.Services {
		if svc.Name == "" {
			continue
		}
		for _, repo := range svc.Repositories {
			if repo.TargetDB == "" {
				continue
			}
			key := "svc." + svc.Name + "|db." + repo.TargetDB
			if dbEdgeSeen[key] {
				continue
			}
			dbEdgeSeen[key] = true
			eID := "edge." + strconv.Itoa(len(edges))
			edges = append(edges, archEdge{
				id:        eID,
				fromID:    "svc." + svc.Name,
				toID:      "db." + repo.TargetDB,
				direction: dirUnidirectional,
			})
		}
	}
	return nodes, edges
}

func findEndpoint(mf *manifest.Manifest, namePath string) *manifest.EndpointDef {
	for i := range mf.Contracts.Endpoints {
		if mf.Contracts.Endpoints[i].NamePath == namePath {
			return &mf.Contracts.Endpoints[i]
		}
	}
	return nil
}

// buildAllInfo constructs detail lines for every node AND edge, stored in one map.
func buildAllInfo(mf *manifest.Manifest, nodes []archNode, edges []archEdge) map[string][]string {
	outEdges := map[string][]archEdge{}
	inEdges := map[string][]archEdge{}
	for _, e := range edges {
		outEdges[e.fromID] = append(outEdges[e.fromID], e)
		inEdges[e.toID] = append(inEdges[e.toID], e)
	}

	info := map[string][]string{}
	for _, n := range nodes {
		info[n.id] = buildSingleNodeInfo(mf, n, outEdges[n.id], inEdges[n.id])
	}
	for _, e := range edges {
		info[e.id] = buildEdgeInfo(mf, e)
	}
	return info
}

// epRow holds one endpoint's display columns for aligned formatting.
type epRow struct {
	method string
	path   string
	reqDTO string
	resDTO string
}

// formatEpRows formats a slice of endpoint rows as aligned table lines.
// If any row has DTOs, all rows include the DTO column (using "—" for absent values).
// Columns: method (dynamic width)  path (dynamic width, capped)  reqDTO → resDTO
func formatEpRows(rows []epRow) []string {
	if len(rows) == 0 {
		return nil
	}
	methodW := 0
	pathW := 0
	hasDTOs := false
	for _, r := range rows {
		if len(r.method) > methodW {
			methodW = len(r.method)
		}
		if len(r.path) > pathW {
			pathW = len(r.path)
		}
		if r.reqDTO != "" || r.resDTO != "" {
			hasDTOs = true
		}
	}
	const maxPathW = 40
	if pathW > maxPathW {
		pathW = maxPathW
	}

	var out []string
	for _, r := range rows {
		if !hasDTOs {
			out = append(out, fmt.Sprintf("    %-*s  %s", methodW, r.method, r.path))
			continue
		}
		req := r.reqDTO
		if req == "" {
			req = "—"
		}
		res := r.resDTO
		if res == "" {
			res = "—"
		}
		out = append(out, fmt.Sprintf("    %-*s  %-*s  %s → %s", methodW, r.method, pathW, r.path, req, res))
	}
	return out
}

// buildEdgeInfo builds the detail lines for a selected communication path.
func buildEdgeInfo(mf *manifest.Manifest, e archEdge) []string {
	var lines []string

	if e.fromID == "frontend" && strings.HasPrefix(e.toID, "svc.") {
		svcName := strings.TrimPrefix(e.toID, "svc.")
		var rows []epRow
		for _, ep := range mf.Contracts.Endpoints {
			if ep.ServiceUnit != svcName {
				continue
			}
			method := ep.HTTPMethod
			if method == "" {
				method = "[" + ep.Protocol + "]"
			}
			rows = append(rows, epRow{
				method: method,
				path:   ep.NamePath,
				reqDTO: ep.RequestDTO,
				resDTO: ep.ResponseDTO,
			})
		}
		if len(rows) > 0 {
			lines = append(lines, "  Endpoints:")
			lines = append(lines, formatEpRows(rows)...)
		} else {
			lines = append(lines, "  No endpoints defined for this service.")
		}
	} else if strings.HasPrefix(e.fromID, "svc.") && strings.HasPrefix(e.toID, "svc.") {
		fromSvc := strings.TrimPrefix(e.fromID, "svc.")
		toSvc := strings.TrimPrefix(e.toID, "svc.")
		found := false
		for _, link := range mf.Backend.CommLinks {
			if link.From != fromSvc || link.To != toSvc {
				continue
			}
			found = true
			if link.Protocol != "" {
				lines = append(lines, "  Protocol:    "+link.Protocol)
			}
			if link.Direction != "" {
				lines = append(lines, "  Direction:   "+link.Direction)
			}
			if link.Trigger != "" {
				lines = append(lines, "  Trigger:     "+link.Trigger)
			}
			if string(link.SyncAsync) != "" {
				lines = append(lines, "  Sync/Async:  "+string(link.SyncAsync))
			}
			if len(link.DTOs) > 0 {
				lines = append(lines, "  Request DTOs:")
				for _, dto := range link.DTOs {
					lines = append(lines, "    → "+dto)
				}
			}
			if len(link.ResponseDTOs) > 0 {
				lines = append(lines, "  Response DTOs:")
				for _, dto := range link.ResponseDTOs {
					lines = append(lines, "    ← "+dto)
				}
			}
			if len(link.ResiliencePatterns) > 0 {
				lines = append(lines, "  Resilience:  "+strings.Join(link.ResiliencePatterns, ", "))
			}
		}
		if !found {
			lines = append(lines, "  No communication details available.")
		}
	} else if strings.HasPrefix(e.fromID, "svc.") && strings.HasPrefix(e.toID, "db.") {
		svcName := strings.TrimPrefix(e.fromID, "svc.")
		dbAlias := strings.TrimPrefix(e.toID, "db.")
		var repos []manifest.RepositoryDef
		for _, svc := range mf.Backend.Services {
			if svc.Name != svcName {
				continue
			}
			for _, repo := range svc.Repositories {
				if repo.TargetDB == dbAlias {
					repos = append(repos, repo)
				}
			}
			break
		}
		if len(repos) == 0 {
			lines = append(lines, "  Implied connection (no repositories defined).")
		} else {
			lines = append(lines, fmt.Sprintf("  Repositories: (%d)", len(repos)))
			for _, repo := range repos {
				repoHeader := "    · " + repo.Name
				if repo.EntityRef != "" {
					repoHeader += "  [" + repo.EntityRef + "]"
				}
				lines = append(lines, repoHeader)
				for _, op := range repo.Operations {
					opLine := "        " + op.OpType + ": " + op.Name
					if op.ResultShape != "" {
						opLine += " → " + op.ResultShape
					}
					lines = append(lines, opLine)
				}
			}
		}
	}

	if len(lines) == 0 {
		lines = []string{"  No contract details available."}
	}
	return lines
}

// splitTrimComma splits a comma-separated string and trims whitespace from each part.
func splitTrimComma(s string) []string {
	var result []string
	for _, p := range strings.Split(s, ",") {
		p = strings.TrimSpace(p)
		if p != "" {
			result = append(result, p)
		}
	}
	return result
}

func buildSingleNodeInfo(mf *manifest.Manifest, n archNode, outs, ins []archEdge) []string {
	var lines []string

	switch n.kind {
	case archFrontend:
		pages := mf.Frontend.Pages
		if len(pages) == 0 {
			lines = append(lines, "  No pages configured.")
		} else {
			// Compute max route width for aligned columns.
			maxRouteW := 0
			for _, page := range pages {
				if len(page.Route) > maxRouteW {
					maxRouteW = len(page.Route)
				}
			}
			const maxRouteCap = 35
			if maxRouteW > maxRouteCap {
				maxRouteW = maxRouteCap
			}

			lines = append(lines, fmt.Sprintf("  Pages: (%d)", len(pages)))
			for _, page := range pages {
				var pageLabel string
				if page.Name != "" && page.Name != page.Route {
					pageLabel = fmt.Sprintf("%-*s  %s", maxRouteW, page.Route, page.Name)
				} else {
					pageLabel = page.Route
				}
				if page.AuthRequired == "true" || page.AuthRequired == "yes" {
					pageLabel += "  [auth]"
				}
				lines = append(lines, "    "+pageLabel)
				if page.ComponentRefs != "" {
					for _, ref := range splitTrimComma(page.ComponentRefs) {
						lines = append(lines, "      · "+ref)
					}
				}
				if page.LinkedPages != "" {
					for _, link := range splitTrimComma(page.LinkedPages) {
						lines = append(lines, "      → "+link)
					}
				}
			}
		}

	case archService:
		svcName := strings.TrimPrefix(n.id, "svc.")
		// Collect rows first to compute dynamic column width.
		var rows []epRow
		for _, ep := range mf.Contracts.Endpoints {
			if ep.ServiceUnit != svcName {
				continue
			}
			method := ep.HTTPMethod
			if method == "" {
				method = "[" + ep.Protocol + "]"
			}
			rows = append(rows, epRow{method: method, path: ep.NamePath})
		}
		if len(rows) > 0 {
			lines = append(lines, fmt.Sprintf("  Endpoints: (%d)", len(rows)))
			lines = append(lines, formatEpRows(rows)...)
		} else {
			lines = append(lines, "  No endpoints configured.")
		}

	case archDatabase:
		alias := strings.TrimPrefix(n.id, "db.")
		var domainNames []string
		for _, domain := range mf.Data.Domains {
			if domain.Databases == "" {
				continue
			}
			for _, dbName := range splitTrimComma(domain.Databases) {
				if dbName == alias {
					domainNames = append(domainNames, domain.Name)
					break
				}
			}
		}
		if len(domainNames) > 0 {
			lines = append(lines, fmt.Sprintf("  Domains: (%d)", len(domainNames)))
			for _, name := range domainNames {
				lines = append(lines, "    · "+name)
			}
		} else {
			lines = append(lines, "  No domains assigned to this database.")
		}

	case archExternalAPI:
		provider := strings.TrimPrefix(n.id, "ext.")
		for _, api := range mf.Contracts.ExternalAPIs {
			if api.Provider != provider {
				continue
			}
			if api.Protocol != "" {
				lines = append(lines, "  Protocol:     "+api.Protocol)
			}
			if api.AuthMechanism != "" {
				lines = append(lines, "  Auth:         "+api.AuthMechanism)
			}
			if api.BaseURL != "" {
				lines = append(lines, "  Base URL:     "+api.BaseURL)
			}
			if api.RateLimit != "" {
				lines = append(lines, "  Rate limit:   "+api.RateLimit)
			}
			if api.FailureStrategy != "" {
				lines = append(lines, "  On failure:   "+api.FailureStrategy)
			}
			if cnt := len(api.Interactions); cnt > 0 {
				lines = append(lines, fmt.Sprintf("  Interactions: %d configured", cnt))
				for i, ia := range api.Interactions {
					if i >= 4 {
						lines = append(lines, fmt.Sprintf("    … +%d more", cnt-4))
						break
					}
					lines = append(lines, "    "+ia.Name)
				}
			}
			break
		}
	}

	// Connection summary — only for external API nodes.
	// Frontend and service connections are navigated directly via the edge selection.
	if n.kind == archExternalAPI && len(outs)+len(ins) > 0 {
		lines = append(lines, "")
		lines = append(lines, "  Connections:")
		for _, e := range outs {
			arrow := "──▶"
			if e.direction == dirBidirectional {
				arrow = "◀──▶"
			}
			target := strings.TrimPrefix(e.toID, "svc.")
			target = strings.TrimPrefix(target, "db.")
			target = strings.TrimPrefix(target, "ext.")
			l := "    " + arrow + " " + target
			if e.label != "" {
				l += "  [" + e.label + "]"
			}
			lines = append(lines, l)
		}
		for _, e := range ins {
			if e.direction == dirBidirectional {
				continue
			}
			src := strings.TrimPrefix(e.fromID, "svc.")
			src = strings.TrimPrefix(src, "frontend")
			if src == "" {
				src = "frontend"
			}
			lines = append(lines, "    ◀── "+src)
		}
	}

	if len(lines) == 0 {
		lines = []string{"  No details available."}
	}
	return lines
}

// envColor returns a display color keyed on environment name convention.
func envColor(name string) string {
	lower := strings.ToLower(name)
	switch {
	case strings.Contains(lower, "prod"):
		return clrRed
	case strings.Contains(lower, "stag"), strings.Contains(lower, "uat"):
		return clrYellow
	case strings.Contains(lower, "dev"), strings.Contains(lower, "local"):
		return clrGreen
	default:
		return clrBlue
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

	var frontendNodes, serviceNodes, dbNodes, extNodes []archNode
	for _, n := range nodes {
		switch n.kind {
		case archFrontend:
			frontendNodes = append(frontendNodes, n)
		case archService:
			serviceNodes = append(serviceNodes, n)
		case archDatabase:
			dbNodes = append(dbNodes, n)
		case archExternalAPI:
			extNodes = append(extNodes, n)
		}
	}

	col0W := maxBoxWidth(frontendNodes, 22)
	col1W := maxBoxWidth(serviceNodes, 22)
	col2W := maxBoxWidth(dbNodes, 20)
	col3W := maxBoxWidth(extNodes, 20)

	const gap = 10
	const margin = 3

	col0X := margin
	col1X := col0X + col0W + gap
	col2X := col1X + col1W + gap
	col3X := col2X + col2W + gap

	mainH := maxInt(
		columnHeight(frontendNodes),
		columnHeight(serviceNodes),
		columnHeight(dbNodes),
		columnHeight(extNodes),
	)

	// Info panel
	infoLines := nodeInfo[selectedID]
	infoPanelH := 0
	if len(infoLines) > 0 {
		infoPanelH = 3 + len(infoLines)
	}

	const headerRows = 3
	totalH := headerRows + mainH + infoPanelH + 4

	totalW := col1X + col1W + margin
	if len(dbNodes) > 0 {
		totalW = col2X + col2W + margin
	}
	if len(extNodes) > 0 {
		totalW = col3X + col3W + margin
	}
	if totalW < termW {
		totalW = termW
	}

	cv := newCanvas(totalW, totalH)

	cv.writeStr(0, 0, "  Architecture Overview")
	cv.hLine(0, 1, totalW-1)

	yOff := headerRows

	if len(frontendNodes) > 0 {
		cv.writeStr(col0X, yOff-1, "FRONTEND")
	}
	if len(serviceNodes) > 0 {
		cv.writeStr(col1X, yOff-1, "BACKEND LAYER")
	}
	if len(dbNodes) > 0 {
		cv.writeStr(col2X, yOff-1, "DATA SOURCES")
	}
	if len(extNodes) > 0 {
		cv.writeStr(col3X, yOff-1, "EXTERNAL APIS")
	}

	// Step 1: compute positions
	nodePositions := map[string]nodePos{}
	computeColumnPositions(frontendNodes, col0X, col0W, yOff, nodePositions)
	computeColumnPositions(serviceNodes, col1X, col1W, yOff, nodePositions)
	computeColumnPositions(dbNodes, col2X, col2W, yOff, nodePositions)
	computeColumnPositions(extNodes, col3X, col3W, yOff, nodePositions)

	// Step 2: environment boxes — outermost layer, spans all columns sharing an env.
	// All node types are included so a service and its database share one env box.
	allEnvNodes := make([]archNode, 0, len(frontendNodes)+len(serviceNodes)+len(dbNodes)+len(extNodes))
	allEnvNodes = append(allEnvNodes, frontendNodes...)
	allEnvNodes = append(allEnvNodes, serviceNodes...)
	allEnvNodes = append(allEnvNodes, dbNodes...)
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
	drawColumnBoxes(&cv, serviceNodes, nodePositions)
	drawColumnBoxes(&cv, dbNodes, nodePositions)
	drawColumnBoxes(&cv, extNodes, nodePositions)

	// Step 5: arrows + collect edge bounds
	edgeBoundsMap := map[string]edgeBounds{}

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
		if arrowY == toY {
			cv.hLine(fromX, arrowY, toX-fromX)
			cv.arrowRight(toX, arrowY)
			bx1, bx2, by1, by2 = fromX, toX+1, arrowY, arrowY+1
		} else {
			midX := fromX + (toX-fromX)/2
			cv.hLine(fromX, arrowY, midX-fromX)
			if arrowY < toY {
				cv.vLine(midX, arrowY, toY-arrowY+1)
			} else {
				cv.vLine(midX, toY, arrowY-toY+1)
			}
			cv.hLine(midX, toY, toX-midX)
			cv.arrowRight(toX, toY)
			bx1, bx2 = fromX, toX+1
			if arrowY < toY {
				by1, by2 = arrowY, toY+1
			} else {
				by1, by2 = toY, arrowY+1
			}
		}
		edgeBoundsMap[e.id] = edgeBounds{x1: bx1, x2: bx2, y1: by1, y2: by2}
	}

	// Frontend → Service arrows
	for _, e := range edges {
		if e.fromID == "frontend" {
			drawEdgeArrow(e)
		}
	}
	// Service → Database arrows
	for _, e := range edges {
		if strings.HasPrefix(e.fromID, "svc.") && strings.HasPrefix(e.toID, "db.") {
			drawEdgeArrow(e)
		}
	}

	// Step 6: info panel for selected node or edge
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
	}

	return archDiagramData{
		rawLines:      strings.Split(cv.render(), "\n"),
		nodePositions: nodePositions,
		edgeBoundsMap: edgeBoundsMap,
		envGroups:     envGroups,
		frontendNodes: frontendNodes,
		serviceNodes:  serviceNodes,
		dbNodes:       dbNodes,
		extNodes:      extNodes,
		selectedID:    selectedID,
	}
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
	addRanges(data.frontendNodes, clrCyan)
	addRanges(data.serviceNodes, clrGreen)
	addRanges(data.dbNodes, clrYellow)
	addRanges(data.extNodes, clrMagenta)

	// Selected item blink — priority 3
	if data.selectedID != "" {
		if strings.HasPrefix(data.selectedID, "edge.") {
			// Color the selected edge path
			if eb, ok := data.edgeBoundsMap[data.selectedID]; ok {
				edgeColor := clrCyan
				if AnimFrame%2 == 0 {
					edgeColor = clrFg
				}
				ranges = append(ranges, colorRange{
					xStart:   eb.x1 - scrollX,
					xEnd:     eb.x2 - scrollX,
					yStart:   eb.y1 - scrollY,
					yEnd:     eb.y2 - scrollY,
					color:    edgeColor,
					priority: 3,
				})
			}
		} else {
			// Color the selected node box
			if p, ok := data.nodePositions[data.selectedID]; ok {
				blinkColor := clrFg
				if AnimFrame%2 == 0 {
					blinkColor = clrYellow
				}
				ranges = append(ranges, colorRange{
					xStart:   p.x - scrollX,
					xEnd:     p.x + p.w - scrollX,
					yStart:   p.y - scrollY,
					yEnd:     p.y + p.h - scrollY,
					color:    blinkColor,
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
				prioAt[x] = cr.priority
			}
		}
	}

	var sb strings.Builder
	i := 0
	for i < len(runes) {
		col := colorAt[i]
		j := i + 1
		for j < len(runes) && colorAt[j] == col {
			j++
		}
		chunk := string(runes[i:j])
		if col != "" {
			chunk = lipgloss.NewStyle().Foreground(lipgloss.Color(col)).Render(chunk)
		}
		sb.WriteString(chunk)
		i = j
	}
	return sb.String()
}

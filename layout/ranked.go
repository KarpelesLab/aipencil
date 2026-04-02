package layout

import (
	"math"
	"sort"

	"github.com/KarpelesLab/aipencil/scene"
)

// layoutRanked implements a Sugiyama-style layered graph layout.
// It produces clean top-to-bottom (or left-to-right) directed graph layouts
// with minimized edge crossings and proper node spacing.
//
// Algorithm:
// 1. Build adjacency from arrows
// 2. Assign ranks (layers) via longest-path
// 3. Order nodes within ranks to minimize crossings (barycenter method)
// 4. Assign X/Y coordinates with proper spacing
type rankedNode struct {
	el    *scene.Element
	id    string
	rank  int
	order int // position within rank
}

func layoutRanked(el *scene.Element, allElements []*scene.Element) {
	if len(el.Children) == 0 {
		return
	}

	// Determine direction from layout config
	direction := "TB" // top-to-bottom (default)
	if el.Layout != nil && el.Layout.Align == "LR" {
		direction = "LR"
	}

	var nodes []*rankedNode
	nodeByID := make(map[string]*rankedNode)

	for _, child := range el.Children {
		if child.Type == "arrow" {
			continue
		}
		id := child.ID
		if id == "" {
			continue // skip unnamed elements
		}
		n := &rankedNode{el: child, id: id, rank: -1}
		nodes = append(nodes, n)
		nodeByID[id] = n
	}

	if len(nodes) == 0 {
		return
	}

	// Build edges from arrows
	type edge struct {
		from, to *rankedNode
	}
	var edges []edge
	outgoing := make(map[string][]*rankedNode) // node ID → targets
	incoming := make(map[string][]*rankedNode) // node ID → sources

	collectArrowEdges := func(elements []*scene.Element) {
		for _, e := range elements {
			if e.Type != "arrow" {
				continue
			}
			fid := baseID(e.From)
			tid := baseID(e.To)
			fn, fok := nodeByID[fid]
			tn, tok := nodeByID[tid]
			if fok && tok {
				edges = append(edges, edge{fn, tn})
				outgoing[fid] = append(outgoing[fid], tn)
				incoming[tid] = append(incoming[tid], fn)
			}
		}
	}
	collectArrowEdges(el.Children)
	if allElements != nil {
		collectArrowEdges(allElements)
	}

	// Step 1: Assign ranks using longest-path from sources
	// Find source nodes (no incoming edges)
	var sources []*rankedNode
	for _, n := range nodes {
		if len(incoming[n.id]) == 0 {
			sources = append(sources, n)
		}
	}
	// If no clear sources (cycle), pick first node
	if len(sources) == 0 {
		sources = []*rankedNode{nodes[0]}
	}

	// BFS/DFS longest path ranking
	assignRanks(nodes, sources, outgoing)

	// Step 2: Group nodes by rank
	maxRank := 0
	for _, n := range nodes {
		if n.rank > maxRank {
			maxRank = n.rank
		}
	}
	ranks := make([][]*rankedNode, maxRank+1)
	for _, n := range nodes {
		ranks[n.rank] = append(ranks[n.rank], n)
	}

	// Sort nodes within each rank alphabetically for initial order
	for _, rank := range ranks {
		sort.SliceStable(rank, func(i, j int) bool {
			return rank[i].id < rank[j].id
		})
		for i, n := range rank {
			n.order = i
		}
	}

	// Step 3: Minimize crossings using barycenter heuristic (multiple passes)
	for pass := 0; pass < 24; pass++ {
		if pass%2 == 0 {
			// Forward sweep: order each rank based on connected nodes in previous rank
			for r := 1; r <= maxRank; r++ {
				for _, n := range ranks[r] {
					n.order = barycenter(n, incoming[n.id])
				}
				sort.SliceStable(ranks[r], func(i, j int) bool {
					return ranks[r][i].order < ranks[r][j].order
				})
				for i, n := range ranks[r] {
					n.order = i
				}
			}
		} else {
			// Backward sweep: order each rank based on connected nodes in next rank
			for r := maxRank - 1; r >= 0; r-- {
				for _, n := range ranks[r] {
					n.order = barycenter(n, outgoing[n.id])
				}
				sort.SliceStable(ranks[r], func(i, j int) bool {
					return ranks[r][i].order < ranks[r][j].order
				})
				for i, n := range ranks[r] {
					n.order = i
				}
			}
		}
	}

	// Step 4: Assign coordinates
	gap := 40.0
	rankGap := 80.0
	if el.Layout != nil {
		if el.Layout.Gap > 0 {
			rankGap = el.Layout.Gap
		}
	}

	// Compute max node size per rank
	rankWidths := make([]float64, maxRank+1)
	rankHeights := make([]float64, maxRank+1)
	for r, rank := range ranks {
		var totalW, maxH float64
		for _, n := range rank {
			if n.el.ComputedWidth > 0 {
				totalW += n.el.ComputedWidth + gap
			}
			if n.el.ComputedHeight > maxH {
				maxH = n.el.ComputedHeight
			}
		}
		rankWidths[r] = totalW - gap
		rankHeights[r] = maxH
	}

	// Find the widest rank for centering
	maxRankWidth := 0.0
	for _, w := range rankWidths {
		if w > maxRankWidth {
			maxRankWidth = w
		}
	}

	if direction == "TB" {
		// Top-to-bottom layout
		y := 0.0
		for r, rank := range ranks {
			// Center this rank horizontally
			totalW := rankWidths[r]
			startX := (maxRankWidth - totalW) / 2
			x := startX

			for _, n := range rank {
				n.el.ComputedX = x
				n.el.ComputedY = y
				n.el.Positioned = true
				x += n.el.ComputedWidth + gap
			}
			y += rankHeights[r] + rankGap
		}

		el.ComputedWidth = maxRankWidth
		el.ComputedHeight = y - rankGap
	} else {
		// Left-to-right layout
		x := 0.0
		maxRankHeight := 0.0
		for _, h := range rankWidths { // swap width/height semantics
			if h > maxRankHeight {
				maxRankHeight = h
			}
		}

		for r, rank := range ranks {
			totalH := 0.0
			maxW := 0.0
			for _, n := range rank {
				totalH += n.el.ComputedHeight + gap
				if n.el.ComputedWidth > maxW {
					maxW = n.el.ComputedWidth
				}
			}
			totalH -= gap
			startY := (maxRankHeight - totalH) / 2

			y := startY
			for _, n := range rank {
				n.el.ComputedX = x + (maxW-n.el.ComputedWidth)/2
				n.el.ComputedY = y
				n.el.Positioned = true
				y += n.el.ComputedHeight + gap
			}
			_ = rankHeights[r]
			x += maxW + rankGap
		}

		el.ComputedWidth = x - rankGap
		el.ComputedHeight = maxRankHeight
	}
}

// assignRanks uses BFS longest-path to assign rank levels.
func assignRanks(nodes []*rankedNode, sources []*rankedNode, outgoing map[string][]*rankedNode) {
	// Initialize all ranks to -1
	for _, n := range nodes {
		n.rank = -1
	}

	// BFS from sources, assigning longest path rank
	type qEntry struct {
		node *rankedNode
		rank int
	}
	queue := make([]qEntry, 0, len(sources))
	for _, s := range sources {
		queue = append(queue, qEntry{s, 0})
	}

	for len(queue) > 0 {
		entry := queue[0]
		queue = queue[1:]

		if entry.rank <= entry.node.rank {
			continue // already have a longer path
		}
		entry.node.rank = entry.rank

		for _, target := range outgoing[entry.node.id] {
			queue = append(queue, qEntry{target, entry.rank + 1})
		}
	}

	// Any unvisited nodes (disconnected) get rank 0
	for _, n := range nodes {
		if n.rank < 0 {
			n.rank = 0
		}
	}
}

// barycenter computes the average position of connected nodes.
func barycenter(n *rankedNode, connected []*rankedNode) int {
	if len(connected) == 0 {
		return n.order
	}
	sum := 0.0
	for _, c := range connected {
		sum += float64(c.order)
	}
	return int(math.Round(sum / float64(len(connected))))
}

package layout

import (
	"math"
	"sort"

	"github.com/KarpelesLab/aipencil/scene"
)

const (
	graphIterations = 120
	graphCoolStart  = 1.0
	graphCoolEnd    = 0.005
)

type graphNode struct {
	el     *scene.Element
	x, y   float64
	fx, fy float64 // accumulated force
	w, h   float64 // element size
}

type graphEdge struct {
	from, to int // indices into nodes slice
}

// layoutGraph applies a force-directed (Fruchterman-Reingold) layout
// to position the children of a group.
//
// It discovers edges by scanning sibling arrow elements that reference
// children by ID. If no arrows are found among siblings, it scans the
// group's own children for arrows.
func layoutGraph(el *scene.Element, allElements []*scene.Element) {
	if len(el.Children) == 0 {
		return
	}

	// Build node list from non-arrow children, sorted by ID for determinism
	var nodes []*graphNode
	idIndex := make(map[string]int) // element ID → node index

	// Collect non-arrow children
	type sortable struct {
		key string
		el  *scene.Element
	}
	var sorted []sortable
	for i, child := range el.Children {
		if child.Type == "arrow" {
			continue
		}
		key := child.ID
		if key == "" {
			key = string(rune('a'+i)) + "_unnamed"
		}
		sorted = append(sorted, sortable{key, child})
	}
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].key < sorted[j].key
	})

	for _, s := range sorted {
		idx := len(nodes)
		idIndex[s.el.ID] = idx
		nodes = append(nodes, &graphNode{
			el: s.el,
			w:  s.el.ComputedWidth,
			h:  s.el.ComputedHeight,
		})
	}

	if len(nodes) == 0 {
		return
	}

	// Build edge list from arrows
	var edges []graphEdge
	collectEdges := func(elements []*scene.Element) {
		for _, e := range elements {
			if e.Type != "arrow" {
				continue
			}
			fi, fok := idIndex[baseID(e.From)]
			ti, tok := idIndex[baseID(e.To)]
			if fok && tok {
				edges = append(edges, graphEdge{fi, ti})
			}
		}
	}

	// Check group's own children for arrows
	collectEdges(el.Children)
	// Also check sibling elements (arrows at the same level as this group)
	if allElements != nil {
		collectEdges(allElements)
	}

	// Initialize positions in a circle
	n := len(nodes)

	// Compute average node size for spacing
	avgSize := 0.0
	for _, node := range nodes {
		avgSize += math.Max(node.w, node.h)
	}
	avgSize /= float64(n)

	// Initial circle radius based on node count and size
	radius := math.Max(avgSize*2, avgSize*float64(n)*0.6)
	for i, node := range nodes {
		angle := 2 * math.Pi * float64(i) / float64(n)
		node.x = radius + radius*math.Cos(angle)
		node.y = radius + radius*math.Sin(angle)
	}

	// Optimal spacing between nodes: proportional to average node size
	k := avgSize * 1.5

	// Iterate
	for iter := range graphIterations {
		// Cooling schedule: linear from coolStart to coolEnd
		t := graphCoolStart - (graphCoolStart-graphCoolEnd)*float64(iter)/float64(graphIterations-1)
		temp := t * k

		// Reset forces
		for _, node := range nodes {
			node.fx = 0
			node.fy = 0
		}

		// Repulsive forces between all pairs
		for i := 0; i < n; i++ {
			for j := i + 1; j < n; j++ {
				dx := nodes[i].x - nodes[j].x
				dy := nodes[i].y - nodes[j].y

				// Include node size in distance calculation
				minDist := (nodes[i].w+nodes[j].w)/2 + 20
				dist := math.Sqrt(dx*dx + dy*dy)
				if dist < 1 {
					dist = 1
					dx = 1 // nudge apart deterministically
				}

				// Repulsive force: k^2 / dist, boosted when overlapping
				force := (k * k) / dist
				if dist < minDist {
					force *= 3 // stronger repulsion when too close
				}

				fx := (dx / dist) * force
				fy := (dy / dist) * force

				nodes[i].fx += fx
				nodes[i].fy += fy
				nodes[j].fx -= fx
				nodes[j].fy -= fy
			}
		}

		// Attractive forces along edges
		for _, edge := range edges {
			ni, nj := nodes[edge.from], nodes[edge.to]
			dx := ni.x - nj.x
			dy := ni.y - nj.y
			dist := math.Sqrt(dx*dx + dy*dy)
			if dist < 1 {
				dist = 1
			}

			// Attractive force (FR standard): dist^2 / k
			force := (dist * dist) / k

			fx := (dx / dist) * force
			fy := (dy / dist) * force

			ni.fx -= fx
			ni.fy -= fy
			nj.fx += fx
			nj.fy += fy
		}

		// Gravity: pull all nodes toward center to prevent drift
		cx, cy := 0.0, 0.0
		for _, node := range nodes {
			cx += node.x
			cy += node.y
		}
		cx /= float64(n)
		cy /= float64(n)
		gravity := 0.1
		for _, node := range nodes {
			node.fx -= (node.x - cx) * gravity
			node.fy -= (node.y - cy) * gravity
		}

		// Apply forces with temperature limiting
		for _, node := range nodes {
			dist := math.Sqrt(node.fx*node.fx + node.fy*node.fy)
			if dist > 0 {
				scale := math.Min(dist, temp) / dist
				node.x += node.fx * scale
				node.y += node.fy * scale
			}
		}
	}

	// Normalize: shift so minimum is at (0, 0), then correct aspect ratio
	minX, minY := math.Inf(1), math.Inf(1)
	rawMaxX, rawMaxY := math.Inf(-1), math.Inf(-1)
	for _, node := range nodes {
		if node.x < minX {
			minX = node.x
		}
		if node.y < minY {
			minY = node.y
		}
		if node.x+node.w > rawMaxX {
			rawMaxX = node.x + node.w
		}
		if node.y+node.h > rawMaxY {
			rawMaxY = node.y + node.h
		}
	}

	spanX := rawMaxX - minX
	spanY := rawMaxY - minY

	// Correct extreme aspect ratios toward a more balanced shape.
	// Target ratio is ~1.4:1 (landscape). Compress the long axis toward target.
	scaleX, scaleY := 1.0, 1.0
	if spanX > 0 && spanY > 0 {
		ratio := spanX / spanY
		targetRatio := 1.4
		if ratio < 0.7 {
			// Too tall: compress Y and stretch X toward target
			scaleX = math.Sqrt(targetRatio / ratio)
			scaleY = 1.0 / scaleX
		} else if ratio > 2.0 {
			// Too wide: compress X and stretch Y toward target
			scaleY = math.Sqrt(ratio / targetRatio)
			scaleX = 1.0 / scaleY
		}
	}

	var maxX, maxY float64
	for _, node := range nodes {
		node.x = (node.x - minX) * scaleX
		node.y = (node.y - minY) * scaleY

		node.el.ComputedX = node.x
		node.el.ComputedY = node.y
		node.el.Positioned = true

		right := node.x + node.w
		bottom := node.y + node.h
		if right > maxX {
			maxX = right
		}
		if bottom > maxY {
			maxY = bottom
		}
	}

	el.ComputedWidth = maxX
	el.ComputedHeight = maxY
}

// baseID extracts the element ID from an anchor reference like "nodeA.right".
func baseID(ref string) string {
	for i := len(ref) - 1; i >= 0; i-- {
		if ref[i] == '.' {
			suffix := ref[i+1:]
			switch suffix {
			case "top", "bottom", "left", "right", "center":
				return ref[:i]
			}
			break
		}
	}
	return ref
}

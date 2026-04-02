package layout

import (
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/KarpelesLab/aipencil/scene"
)

const maxSolverIterations = 100

// ConstraintOp is the comparison operator.
type ConstraintOp int

const (
	OpEq ConstraintOp = iota
	OpGte
	OpLte
)

// Strength ordering for constraint priority.
type Strength int

const (
	StrengthRequired Strength = iota
	StrengthStrong
	StrengthMedium
	StrengthWeak
)

// solverVar represents a single solvable variable.
type solverVar struct {
	element *scene.Element
	attr    string
	value   float64
}

// solverConstraint is a parsed, ready-to-evaluate constraint.
type solverConstraint struct {
	target   *solverVar
	op       ConstraintOp
	expr     *expression
	strength Strength
	source   string // for error messages
}

// expression represents the right-hand side of a constraint.
type expression struct {
	refVar  *solverVar // nil for literal
	mathOp  byte       // 0, '+', '-', '*'
	literal float64
}

func (e *expression) eval() float64 {
	if e.refVar == nil {
		return e.literal
	}
	base := e.refVar.value
	switch e.mathOp {
	case '+':
		return base + e.literal
	case '-':
		return base - e.literal
	case '*':
		return base * e.literal
	default:
		return base
	}
}

// SolveConstraints resolves all constraint-based positioning.
// Returns any error/warning messages.
func SolveConstraints(elements []*scene.Element, idMap *IDMap) []string {
	var msgs []string

	// Collect all elements with constraints
	var constrained []*scene.Element
	walkElements(elements, func(el *scene.Element) {
		if len(el.Constraints) > 0 {
			constrained = append(constrained, el)
		}
	})

	if len(constrained) == 0 {
		return nil
	}

	// Build variable map: "elementID.attr" → *solverVar
	vars := make(map[string]*solverVar)

	// Create variables for all constrained elements and their references
	for _, el := range constrained {
		ensureVars(el, vars)
		for _, c := range el.Constraints {
			expr := c.Eq
			if expr == "" {
				expr = c.Gte
			}
			if expr == "" {
				expr = c.Lte
			}
			// Parse the expression to find referenced elements
			refID, _ := parseExprRef(expr)
			if refID != "" && refID != "parent" {
				if refEl, ok := idMap.Elements[refID]; ok {
					ensureVars(refEl, vars)
				}
			}
			// Handle parent reference
			if refID == "parent" {
				if parent, ok := idMap.Parent[el]; ok {
					ensureVars(parent, vars)
				}
			}
		}
	}

	// Build solver constraints
	var constraints []*solverConstraint
	for _, el := range constrained {
		for i, c := range el.Constraints {
			sc, err := buildConstraint(el, c, i, vars, idMap)
			if err != nil {
				msgs = append(msgs, err.Error())
				continue
			}
			constraints = append(constraints, sc)
		}
	}

	// Sort by strength (required first)
	sortConstraints(constraints)

	// Iterative solve
	for iter := range maxSolverIterations {
		_ = iter
		changed := false

		// Recompute derived attributes (right/bottom/centerX/centerY)
		// from primaries (left/top/width/height) for all known elements
		seen := make(map[*scene.Element]bool)
		for _, v := range vars {
			if !seen[v.element] {
				recomputeDerived(v.element, vars)
				seen[v.element] = true
			}
		}

		for _, sc := range constraints {
			target := sc.target
			desired := sc.expr.eval()

			switch sc.op {
			case OpEq:
				if math.Abs(target.value-desired) > 0.01 {
					target.value = desired
					changed = true
					applyToRelated(target, vars)
				}
			case OpGte:
				if target.value < desired-0.01 {
					target.value = desired
					changed = true
					applyToRelated(target, vars)
				}
			case OpLte:
				if target.value > desired+0.01 {
					target.value = desired
					changed = true
					applyToRelated(target, vars)
				}
			}
		}

		if !changed {
			break
		}
	}

	// Write back solved values
	for _, el := range constrained {
		writeBack(el, vars)
	}

	return msgs
}

// ensureVars creates the 8 standard variables for an element.
func ensureVars(el *scene.Element, vars map[string]*solverVar) {
	id := el.ID
	if id == "" {
		id = fmt.Sprintf("_anon_%p", el)
	}

	attrs := []string{"left", "right", "top", "bottom", "width", "height", "centerX", "centerY"}
	for _, attr := range attrs {
		key := id + "." + attr
		if _, exists := vars[key]; exists {
			continue
		}
		v := &solverVar{element: el, attr: attr}
		// Initialize from current computed values
		switch attr {
		case "left":
			v.value = el.EffectiveX()
		case "top":
			v.value = el.EffectiveY()
		case "width":
			v.value = el.ComputedWidth
		case "height":
			v.value = el.ComputedHeight
		case "right":
			v.value = el.EffectiveX() + el.ComputedWidth
		case "bottom":
			v.value = el.EffectiveY() + el.ComputedHeight
		case "centerX":
			v.value = el.EffectiveX() + el.ComputedWidth/2
		case "centerY":
			v.value = el.EffectiveY() + el.ComputedHeight/2
		}
		vars[key] = v
	}
}

// recomputeDerived updates right/bottom/centerX/centerY from left/top/width/height.
func recomputeDerived(el *scene.Element, vars map[string]*solverVar) {
	id := varID(el)
	left, _ := vars[id+".left"]
	top, _ := vars[id+".top"]
	width, _ := vars[id+".width"]
	height, _ := vars[id+".height"]
	right, _ := vars[id+".right"]
	bottom, _ := vars[id+".bottom"]
	cx, _ := vars[id+".centerX"]
	cy, _ := vars[id+".centerY"]

	if left != nil && width != nil {
		if right != nil {
			right.value = left.value + width.value
		}
		if cx != nil {
			cx.value = left.value + width.value/2
		}
	}
	if top != nil && height != nil {
		if bottom != nil {
			bottom.value = top.value + height.value
		}
		if cy != nil {
			cy.value = top.value + height.value/2
		}
	}
}

// applyToRelated propagates a change to a derived attribute back to primaries.
// For example, if "right" was just set, update "width" or "left".
func applyToRelated(v *solverVar, vars map[string]*solverVar) {
	id := varID(v.element)
	switch v.attr {
	case "right":
		if left, ok := vars[id+".left"]; ok {
			if w, ok := vars[id+".width"]; ok {
				w.value = v.value - left.value
			}
		}
	case "bottom":
		if top, ok := vars[id+".top"]; ok {
			if h, ok := vars[id+".height"]; ok {
				h.value = v.value - top.value
			}
		}
	case "centerX":
		if w, ok := vars[id+".width"]; ok {
			if left, ok := vars[id+".left"]; ok {
				left.value = v.value - w.value/2
			}
		}
	case "centerY":
		if h, ok := vars[id+".height"]; ok {
			if top, ok := vars[id+".top"]; ok {
				top.value = v.value - h.value/2
			}
		}
	case "width":
		if left, ok := vars[id+".left"]; ok {
			if right, ok := vars[id+".right"]; ok {
				right.value = left.value + v.value
			}
		}
	case "height":
		if top, ok := vars[id+".top"]; ok {
			if bottom, ok := vars[id+".bottom"]; ok {
				bottom.value = top.value + v.value
			}
		}
	}
}

func writeBack(el *scene.Element, vars map[string]*solverVar) {
	id := varID(el)
	if v, ok := vars[id+".left"]; ok {
		el.ComputedX = v.value
		// Clear explicit X so EffectiveX uses ComputedX
		el.X = nil
	}
	if v, ok := vars[id+".top"]; ok {
		el.ComputedY = v.value
		el.Y = nil
	}
	if v, ok := vars[id+".width"]; ok {
		el.ComputedWidth = v.value
		el.Width = nil
	}
	if v, ok := vars[id+".height"]; ok {
		el.ComputedHeight = v.value
		el.Height = nil
	}
	el.Positioned = true
}

func varID(el *scene.Element) string {
	if el.ID != "" {
		return el.ID
	}
	return fmt.Sprintf("_anon_%p", el)
}

func buildConstraint(el *scene.Element, c *scene.Constraint, idx int, vars map[string]*solverVar, idMap *IDMap) (*solverConstraint, error) {
	id := varID(el)
	source := fmt.Sprintf("element %q constraint[%d]", id, idx)

	targetKey := id + "." + c.Attr
	target, ok := vars[targetKey]
	if !ok {
		return nil, fmt.Errorf("%s: unknown attribute %q", source, c.Attr)
	}

	var op ConstraintOp
	var exprStr string
	if c.Eq != "" {
		op = OpEq
		exprStr = c.Eq
	} else if c.Gte != "" {
		op = OpGte
		exprStr = c.Gte
	} else if c.Lte != "" {
		op = OpLte
		exprStr = c.Lte
	} else {
		return nil, fmt.Errorf("%s: must specify eq, gte, or lte", source)
	}

	expr, err := parseExpression(exprStr, el, vars, idMap)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", source, err)
	}

	strength := StrengthRequired
	switch c.Strength {
	case "strong":
		strength = StrengthStrong
	case "medium":
		strength = StrengthMedium
	case "weak":
		strength = StrengthWeak
	}

	return &solverConstraint{
		target:   target,
		op:       op,
		expr:     expr,
		strength: strength,
		source:   source,
	}, nil
}

// parseExpression parses: "150", "parent.width", "parent.width * 0.5", "sidebar.right + 20"
func parseExpression(s string, el *scene.Element, vars map[string]*solverVar, idMap *IDMap) (*expression, error) {
	s = strings.TrimSpace(s)
	parts := strings.Fields(s)

	switch len(parts) {
	case 1:
		// Literal number or reference
		if v, err := strconv.ParseFloat(parts[0], 64); err == nil {
			return &expression{literal: v}, nil
		}
		// Reference: "parent.width" or "sidebar.right"
		v, err := resolveRef(parts[0], el, vars, idMap)
		if err != nil {
			return nil, err
		}
		return &expression{refVar: v}, nil

	case 3:
		// "ref op literal": "parent.width * 0.5"
		v, err := resolveRef(parts[0], el, vars, idMap)
		if err != nil {
			return nil, err
		}
		op := parts[1]
		if len(op) != 1 || !strings.ContainsRune("+-*", rune(op[0])) {
			return nil, fmt.Errorf("unsupported operator %q (use +, -, or *)", op)
		}
		lit, err := strconv.ParseFloat(parts[2], 64)
		if err != nil {
			return nil, fmt.Errorf("expected number after %q, got %q", op, parts[2])
		}
		return &expression{refVar: v, mathOp: op[0], literal: lit}, nil

	default:
		return nil, fmt.Errorf("invalid expression %q: expected 'value', 'ref.attr', or 'ref.attr op number'", s)
	}
}

// resolveRef resolves "parent.width" or "elementID.attr" to a solver variable.
func resolveRef(ref string, el *scene.Element, vars map[string]*solverVar, idMap *IDMap) (*solverVar, error) {
	dot := strings.LastIndex(ref, ".")
	if dot < 0 {
		return nil, fmt.Errorf("invalid reference %q: expected 'id.attr'", ref)
	}
	elemRef := ref[:dot]
	attr := ref[dot+1:]

	var key string
	if elemRef == "parent" {
		if parent, ok := idMap.Parent[el]; ok {
			key = varID(parent) + "." + attr
		} else {
			return nil, fmt.Errorf("element has no parent for 'parent.%s'", attr)
		}
	} else {
		key = elemRef + "." + attr
	}

	v, ok := vars[key]
	if !ok {
		return nil, fmt.Errorf("unknown reference %q", ref)
	}
	return v, nil
}

// parseExprRef extracts the element ID from an expression string (for pre-scanning).
func parseExprRef(s string) (string, string) {
	parts := strings.Fields(strings.TrimSpace(s))
	if len(parts) == 0 {
		return "", ""
	}
	ref := parts[0]
	dot := strings.LastIndex(ref, ".")
	if dot < 0 {
		return "", ""
	}
	return ref[:dot], ref[dot+1:]
}

func sortConstraints(constraints []*solverConstraint) {
	// Stable sort by strength (required first)
	for i := 0; i < len(constraints); i++ {
		for j := i + 1; j < len(constraints); j++ {
			if constraints[j].strength < constraints[i].strength {
				constraints[i], constraints[j] = constraints[j], constraints[i]
			}
		}
	}
}

// walkElements calls fn for every element in the tree (including layers).
func walkElements(elements []*scene.Element, fn func(*scene.Element)) {
	for _, el := range elements {
		fn(el)
		walkElements(el.Children, fn)
		for _, layer := range el.Layers {
			walkElements(layer.Elements, fn)
		}
	}
}

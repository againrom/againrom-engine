package game

import (
	"fmt"
	"image"
	"strings"

	"againrom/pkg/formats/alm"
	"againrom/pkg/ui"
)

// Operation captions describe only the pinned vocabulary's established arms
// (TRIG-COND-003, TRIG-ACT-004, TRIG-DIST-014, TRIG-SACK-022). An authored label
// never supplies semantics, and an unlisted operation remains explicitly raw.
func inspectionOperation(kind string, opcode uint32) string {
	var names map[uint32]string
	if kind == "Condition" {
		names = map[uint32]string{1: "Group count", 2: "Unit in box", 3: "Unit in distance square", 4: "Unit field (selector 6)", 5: "Unit alive", 6: "Distance between units", 7: "Point to unit distance", 8: "Player population", 10: "Player relation", 11: "No-write arm", 12: "Item presence", 13: "No-write arm", 14: "Sack at cell", 15: "Nearest player unit distance", 16: "Item to point distance", 17: "Item presence", 18: "Lose if unit dies", 19: "Read variable", 21: "Structure field", 22: "Move object check", 0x10002: "Constant / variable preset"}
	} else {
		names = map[uint32]string{2: "Send message", 3: "Set variable", 4: "Win", 5: "Lose", 6: "Group command", 8: "Increment variable", 9: "No-op arm", 10: "Set relation", 16: "Take unit off map", 17: "Return unit to map", 18: "Replace unit on map", 19: "Give unit to player", 22: "Give group to player", 26: "Set structure field", 27: "Move unit", 29: "Set cell effect duration", 30: "Set unit effect duration", 32: "Take group off map", 33: "Return group to map", 34: "Set unit field", 0x10002: "Drop location (build time)", 0x10003: "Object form (build time)"}
	}
	if name := names[opcode]; name != "" {
		return name
	}
	if kind == "Action" {
		// TRIG-ADDITEM-027, TRIG-GIVEALL-025, TRIG-MONEY-028.
		switch opcode {
		case 12:
			return "Create item"
		case 23:
			return "Give money to player"
		case 28:
			return "Give entire inventory"
		}
	}
	return "Unknown operation"
}

type inspectionScript struct {
	doc *ui.InspectionDocument
	m   *alm.Map
	// Slice-valued maps preserve duplicate identities rather than choosing one.
	conditions, actions       map[uint32][]alm.ScriptNode
	units, groups, structures map[uint32][]ui.InspectionTarget
}

func inspectScript(d *ui.InspectionDocument, m *alm.Map, script alm.Script) {
	s := inspectionScript{doc: d, m: m, conditions: make(map[uint32][]alm.ScriptNode), actions: make(map[uint32][]alm.ScriptNode), units: make(map[uint32][]ui.InspectionTarget), groups: make(map[uint32][]ui.InspectionTarget), structures: make(map[uint32][]ui.InspectionTarget)}
	for _, n := range script.Conditions {
		s.conditions[n.ID] = append(s.conditions[n.ID], n)
	}
	for _, n := range script.Actions {
		s.actions[n.ID] = append(s.actions[n.ID], n)
	}
	for _, r := range d.Records {
		t := ui.InspectionTarget{Cell: r.Cell, Size: r.Size}
		switch r.Kind {
		case "Unit":
			u := m.Units[r.Index]
			s.units[uint32(u.UnitID)] = append(s.units[uint32(u.UnitID)], t)
			s.groups[u.GroupID] = append(s.groups[u.GroupID], t)
		case "Structure":
			id := uint32(m.Objects[r.Index].Field12)
			s.structures[id] = append(s.structures[id], t)
		}
	}
	for _, family := range []struct {
		kind  string
		nodes []alm.ScriptNode
	}{{"Action", script.Actions}, {"Condition", script.Conditions}} {
		for i, n := range family.nodes {
			r := ui.InspectionRecord{Kind: family.kind, Section: 7, Index: i, Label: n.Label}
			s.node(&r, family.kind, n)
			d.Records = append(d.Records, r)
		}
	}
	for i, t := range script.Triggers {
		r := ui.InspectionRecord{Kind: "Trigger", Section: 7, Index: i, Label: t.Name}
		once := "repeat while conditions hold"
		if t.Once != 0 {
			once = "at most once per session"
		}
		s.line(&r, fmt.Sprintf("Once %d: %s", t.Once, once), "")
		s.line(&r, "Conditions / AND", "condition")
		for p := range t.Left {
			comparison := "unknown"
			if t.Cmp[p] < 6 {
				comparison = []string{"==", "!=", ">", "<", ">=", "<="}[t.Cmp[p]]
			}
			status := ""
			if t.Left[p] == 0 && t.Right[p] == 0 {
				status = " / empty"
			} else if t.Left[p] == 0 || t.Right[p] == 0 {
				status = " / incomplete pair"
			}
			s.line(&r, fmt.Sprintf("%d: C%d %s C%d [cmp %d]%s", p+1, t.Left[p], comparison, t.Right[p], t.Cmp[p], status), "condition")
			if status == " / incomplete pair" {
				s.issue(&r, fmt.Sprintf("Pair %d incomplete; raw sides retained", p+1))
			}
			if t.Cmp[p] > 5 {
				s.issue(&r, fmt.Sprintf("Pair %d unknown comparator %#x", p+1, t.Cmp[p]))
			}
		}
		s.line(&r, "Actions / authored order", "action")
		for p, id := range t.Acts {
			label := "empty"
			if id != 0 {
				label = s.nodeCaption("Action", id)
			}
			s.line(&r, fmt.Sprintf("%d: A%d / %s", p+1, id, label), "action")
		}
		s.line(&r, "Live state unknown; not executed", "")
		if t.Left[0] == 0 {
			s.line(&r, "First Left 0: omitted by runtime builder", "")
		}
		// Preserve every reference in the summary; expand each node identity only
		// once, so C1==C1 does not bury the next clause in duplicate arguments.
		seen := make(map[string]bool)
		for _, family := range []struct {
			kind string
			ids  []uint32
		}{{"Condition", []uint32{t.Left[0], t.Right[0], t.Left[1], t.Right[1], t.Left[2], t.Right[2]}}, {"Action", t.Acts[:]}} {
			for _, id := range family.ids {
				key := fmt.Sprintf("%s:%d", family.kind, id)
				if id == 0 || seen[key] {
					continue
				}
				seen[key] = true
				nodes := s.nodeMap(family.kind)[id]
				if len(nodes) != 1 {
					s.issue(&r, s.nodeCaption(family.kind, id))
					continue
				}
				s.node(&r, family.kind, nodes[0])
			}
		}
		d.Records = append(d.Records, r)
	}
}

func (s *inspectionScript) nodeMap(kind string) map[uint32][]alm.ScriptNode {
	if kind == "Condition" {
		return s.conditions
	}
	return s.actions
}

func (s *inspectionScript) nodeCaption(kind string, id uint32) string {
	nodes := s.nodeMap(kind)[id]
	if len(nodes) == 0 {
		return fmt.Sprintf("Unresolved %s ID %d", kind, id)
	}
	if len(nodes) > 1 {
		return fmt.Sprintf("Ambiguous %s ID %d (%d records)", kind, id, len(nodes))
	}
	return inspectionOperation(kind, nodes[0].Opcode)
}

func (s *inspectionScript) line(r *ui.InspectionRecord, text, role string) {
	r.Lines = append(r.Lines, ui.InspectionLine{Text: text, Role: role})
}

func (s *inspectionScript) issue(r *ui.InspectionRecord, text string) {
	r.Warning = joinInspectionWarning(r.Warning, text)
	s.line(r, "! "+text, "warning")
}

func (s *inspectionScript) link(r *ui.InspectionRecord, label, role string, targets []ui.InspectionTarget) {
	ref := 0
	if len(targets) > 0 {
		r.References = append(r.References, ui.InspectionReference{Label: label, Role: role, Targets: targets})
		ref = len(r.References)
	}
	r.Lines = append(r.Lines, ui.InspectionLine{Text: label, Role: role, Reference: ref})
}

func (s *inspectionScript) node(r *ui.InspectionRecord, kind string, n alm.ScriptNode) {
	role := strings.ToLower(kind)
	op := inspectionOperation(kind, n.Opcode)
	s.line(r, fmt.Sprintf("%s ID %d: %s", kind, n.ID, op), role)
	s.line(r, fmt.Sprintf("Label: %s / opcode %d", n.Label, n.Opcode), "")
	if len(s.nodeMap(kind)[n.ID]) > 1 {
		s.issue(r, fmt.Sprintf("Ambiguous %s ID %d", kind, n.ID))
	}
	if op == "Unknown operation" {
		s.issue(r, fmt.Sprintf("Unknown %s opcode %d (%#x)", kind, n.Opcode, n.Opcode))
	}
	for i, tag := range n.Type {
		if tag == 0 && n.Value[i] == 0 {
			continue
		}
		name := "Unknown"
		if tag <= 9 {
			name = []string{"None", "Int/Enum", "Group", "Player", "Unit", "X", "Y", "Const", "Item", "Structure"}[tag]
		}
		label := fmt.Sprintf("Par%d %s [tag %d] = %d", i, name, tag, n.Value[i])
		s.line(r, label, "")
		if tag > 9 {
			s.issue(r, fmt.Sprintf("Par%d unknown type tag %d", i, tag))
		}
		s.parameter(r, role, tag, n.Value[i])
	}
	// All ten positions remain visible, even tag-0 values and trailing zeros.
	s.line(r, fmt.Sprintf("Raw values: %v", n.Value), "")
	s.line(r, fmt.Sprintf("Raw tags: %v", n.Type), "")
	// Coordinates come from actual typed slots, never from a node's label or
	// opcode's expected schema (the mismatched PalNearChief is a real example).
	for i := 0; i < len(n.Type)-1; i++ {
		if n.Type[i] != 5 || n.Type[i+1] != 6 {
			continue
		}
		label := fmt.Sprintf("Cell %d,%d (Par%d/%d)", n.Value[i], n.Value[i+1], i, i+1)
		if uint64(n.Value[i]) >= uint64(s.m.Width) || uint64(n.Value[i+1]) >= uint64(s.m.Height) {
			s.issue(r, label+": outside map; no target")
			continue
		}
		x, y := int(n.Value[i]), int(n.Value[i+1])
		s.link(r, label, role, []ui.InspectionTarget{{Cell: image.Pt(x, y), Size: image.Pt(1, 1)}})
	}
	// TRIG-DIST-014: this exact authored signature bounds a Chebyshev square.
	// The helper masks coordinates to bytes. Wide maps can therefore alias
	// distinct authored cells even when this node's center is byte-sized. Keep
	// their raw point, but withhold a semantic map region instead of inventing
	// an unmasked square or emulating the runtime's coordinate aliases.
	if kind == "Condition" && n.Opcode == 3 && n.Type[0] == 4 && n.Type[1] == 5 && n.Type[2] == 6 && n.Type[3] == 1 {
		switch {
		case n.Value[1] > 255 || n.Value[2] > 255 || s.m.Width > 256 || s.m.Height > 256:
			s.issue(r, "Distance region withheld: byte-masked coordinates; authored point only")
		case n.Value[3] > 255:
			s.issue(r, "Distance region withheld: radius outside 0..255")
		case n.Value[1] < uint32(s.m.Width) && n.Value[2] < uint32(s.m.Height):
			x, y, radius := int(n.Value[1]), int(n.Value[2]), int(n.Value[3])
			s.link(r, fmt.Sprintf("Distance square: %d,%d radius %d", x, y, radius), role, []ui.InspectionTarget{{Cell: image.Pt(x-radius, y-radius), Size: image.Pt(2*radius+1, 2*radius+1)}})
		}
	}
	if kind == "Condition" && n.Opcode == 2 {
		s.line(r, "Box endpoints authored; edge rules unknown", "")
	}
}

func (s *inspectionScript) parameter(r *ui.InspectionRecord, role string, tag, id uint32) {
	var targets []ui.InspectionTarget
	label, unique := "", true
	switch tag {
	case 2:
		label, targets, unique = fmt.Sprintf("Group %d", id), s.groups[id], false
	case 3:
		if id == 0 || uint64(id) > uint64(len(s.m.Groups)) {
			s.issue(r, fmt.Sprintf("Unresolved player slot %d", id))
			return
		}
		s.line(r, fmt.Sprintf("Player slot %d: %s (no map anchor)", id, s.m.Groups[id-1].Name), role)
		return
	case 4:
		if id >= 10001 {
			if id <= 11000 {
				s.line(r, fmt.Sprintf("External hero ordinal %d; no placed target", id-10000), role)
			} else {
				s.line(r, fmt.Sprintf("External static unit %d; no placed target", id), role)
			}
			return
		}
		label, targets = fmt.Sprintf("Unit ID %d", id), s.units[id]
	case 8:
		s.line(r, fmt.Sprintf("Item offset %d / code %#04x; no map anchor", id, uint16(id+0x0e18)), role)
		return
	case 9:
		label, targets = fmt.Sprintf("Structure ID %d", id), s.structures[id]
	default:
		return
	}
	if len(targets) == 0 {
		s.issue(r, "Unresolved "+label)
		return
	}
	if unique && len(targets) != 1 {
		s.issue(r, fmt.Sprintf("Ambiguous %s (%d placements)", label, len(targets)))
		return
	}
	var visible []ui.InspectionTarget
	for _, t := range targets {
		if !t.Cell.In(image.Rect(0, 0, s.m.Width, s.m.Height)) {
			s.issue(r, fmt.Sprintf("%s anchor %d,%d outside map", label, t.Cell.X, t.Cell.Y))
			continue
		}
		visible = append(visible, t)
	}
	s.link(r, fmt.Sprintf("%s / %d placed target(s)", label, len(targets)), role, visible)
}

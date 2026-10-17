package game

import (
	"fmt"
	"image"
	"strings"

	"againrom/pkg/formats/alm"
	"againrom/pkg/mapload"
	"againrom/pkg/ui"
)

func inspectionOperation(kind string, opcode uint32) string {
	return mapload.ScriptOperationName(kind == "Condition", opcode)
}

type inspectionScript struct {
	doc  *ui.InspectionDocument
	m    *alm.Map
	text func(string) string // UTF-8 to the font's bytes
	// Slice-valued maps preserve duplicate identities rather than choosing one.
	conditions, actions       map[uint32][]alm.ScriptNode
	units, groups, structures map[uint32][]ui.InspectionTarget
}

// inspectScript adds the script's records and model to d. Records and lines
// carry the font's bytes through text; the model's names stay UTF-8, the text
// the name fields edit.
func inspectScript(d *ui.InspectionDocument, m *alm.Map, script alm.Script, text func(string) string) {
	s := inspectionScript{doc: d, m: m, text: text, conditions: make(map[uint32][]alm.ScriptNode), actions: make(map[uint32][]alm.ScriptNode), units: make(map[uint32][]ui.InspectionTarget), groups: make(map[uint32][]ui.InspectionTarget), structures: make(map[uint32][]ui.InspectionTarget)}
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
	model := scriptModel(d, m, text)
	for _, family := range []struct {
		kind  string
		nodes []alm.ScriptNode
		out   *[]ui.InspectionScriptNode
	}{{"Action", script.Actions, &model.Actions}, {"Condition", script.Conditions, &model.Conditions}} {
		for i, n := range family.nodes {
			r := ui.InspectionRecord{Kind: family.kind, Section: 7, Index: i, Label: text(n.Label)}
			s.node(&r, family.kind, n)
			*family.out = append(*family.out, ui.InspectionScriptNode{Record: len(d.Records), ID: n.ID, Opcode: n.Opcode, Label: n.Label, Types: n.Type, Values: n.Value})
			d.Records = append(d.Records, r)
		}
	}
	for i, t := range script.Triggers {
		model.Triggers = append(model.Triggers, ui.InspectionTrigger{Record: len(d.Records), Name: t.Name, Left: t.Left, Right: t.Right, Cmp: t.Cmp, Acts: t.Acts, Once: t.Once})
		r := ui.InspectionRecord{Kind: "Trigger", Section: 7, Index: i, Label: text(t.Name)}
		once := "repeat while conditions hold"
		if t.Once != 0 {
			once = "at most once per session"
		}
		s.line(&r, fmt.Sprintf("Once %d: %s", t.Once, once), "")
		s.line(&r, "Conditions / AND", "condition")
		for p := range t.Left {
			comparison := "unknown"
			if t.Cmp[p] < 6 {
				comparison = mapload.ScriptComparisons[t.Cmp[p]]
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
	areas, _ := mapload.ScriptAreas(m, script)
	for _, a := range areas {
		node := model.Actions
		if a.Condition {
			node = model.Conditions
		}
		radius := -1
		if a.RadiusNode >= 0 {
			radius = model.Conditions[a.RadiusNode].Record
		}
		model.Areas = append(model.Areas, ui.InspectionArea{Trigger: a.Trigger, Node: node[a.Node].Record, Shape: int(a.Shape), Rect: a.Rect, Centre: a.Centre, Radius: a.Radius,
			Slots: a.Slots, RadiusNode: radius, RadiusSlot: a.RadiusSlot, RadiusBias: a.RadiusBias})
	}
	model.Catalog = scriptCatalog()
	d.Script = model
}

// scriptCatalog is the decoder's knowledge of the operations, for the trigger
// editor: each opcode with whether a claim names it, the group commands, and
// the signatures the claims state.
func scriptCatalog() ui.InspectionScriptCatalog {
	family := func(condition bool) []ui.InspectionScriptOperation {
		var out []ui.InspectionScriptOperation
		for _, op := range mapload.ScriptOperations(condition) {
			out = append(out, ui.InspectionScriptOperation{Opcode: op.Opcode, Named: mapload.ScriptOperationNamed(condition, op.Opcode)})
		}
		return out
	}
	return ui.InspectionScriptCatalog{Checks: family(true), Actions: family(false), GroupCommands: mapload.ScriptGroupCommands,
		Signature: func(condition bool, opcode, command uint32) ([10]ui.InspectionScriptSlot, bool) {
			sig, ok := mapload.ScriptSignature(condition, opcode, command)
			var out [10]ui.InspectionScriptSlot
			for i, slot := range sig {
				out[i] = ui.InspectionScriptSlot{Type: slot.Type, Role: slot.Role}
			}
			return out, ok
		}}
}

// scriptModel is the trigger editor's view of what the map's parameters can
// name: each unit id, group id, player slot and structure id the map holds,
// with the record it stands for.
func scriptModel(d *ui.InspectionDocument, m *alm.Map, text func(string) string) *ui.InspectionScriptModel {
	model := &ui.InspectionScriptModel{Targets: map[uint32][]ui.InspectionScriptTarget{}}
	units, structures, players := map[int]int{}, map[int]int{}, map[int]int{}
	for i, r := range d.Records {
		switch r.Kind {
		case "Unit":
			units[r.Index] = i
		case "Structure":
			structures[r.Index] = i
		case "Player":
			players[r.Index] = i
		}
	}
	groups := map[uint32]int{}
	var order []uint32
	for i, u := range m.Units {
		if u.UnitID > 0 {
			model.Targets[mapload.ScriptUnit] = append(model.Targets[mapload.ScriptUnit], ui.InspectionScriptTarget{Value: uint32(u.UnitID), Record: units[i], Name: d.Records[units[i]].Label})
		}
		if _, ok := groups[u.GroupID]; !ok {
			order = append(order, u.GroupID)
		}
		groups[u.GroupID]++
	}
	for _, g := range order {
		model.Targets[mapload.ScriptGroup] = append(model.Targets[mapload.ScriptGroup], ui.InspectionScriptTarget{Value: g, Record: -1, Members: groups[g]})
	}
	for i, g := range m.Groups {
		t := ui.InspectionScriptTarget{Value: uint32(i + 1), Record: -1, Name: text(alm.PlayerNameText.Decode([]byte(g.Name)))}
		if r, ok := players[i]; ok {
			t.Record, t.Name = r, d.Records[r].Label
		}
		model.Targets[mapload.ScriptPlayer] = append(model.Targets[mapload.ScriptPlayer], t)
	}
	for i, o := range m.Objects {
		if o.Field12 > 0 {
			model.Targets[mapload.ScriptStructure] = append(model.Targets[mapload.ScriptStructure], ui.InspectionScriptTarget{Value: uint32(o.Field12), Record: structures[i], Name: d.Records[structures[i]].Label})
		}
	}
	return model
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
	s.line(r, fmt.Sprintf("Label: %s / opcode %d", s.text(n.Label), n.Opcode), "")
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
			name = mapload.ScriptParameterNames[tag]
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
		s.line(r, fmt.Sprintf("Player slot %d: %s (no map anchor)", id, s.text(alm.PlayerNameText.Decode([]byte(s.m.Groups[id-1].Name)))), role)
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

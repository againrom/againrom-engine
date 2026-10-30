package ui

import (
	"encoding/json"
	"fmt"
	"image"
	"slices"
)

// The command table's edits of several records at once, a group, a shop, a
// message, a trigger area and a script goal. Each apply method makes the host
// call the window's route made and adopts what the host returns as that
// route did. The window passes the state it holds (the selection, the copy,
// the message's cards, the chosen goal) as the command's values; MCP passes
// the same values as named arguments.

// recordsMove is records moved by Offset cells.
type recordsMove struct {
	Refs   []RecordRef
	Offset image.Point
}

// areaMove is records and the objects of Box moved by Offset cells.
type areaMove struct {
	Refs   []RecordRef
	Box    image.Rectangle
	Offset image.Point
}

// recordsPaste is a copy pasted with its top-left on Cell.
type recordsPaste struct {
	Clip InspectionClip
	Cell image.Point
}

// adoptedGroup takes a group edit as the window takes it: its document, the
// records it leaves selected and its note.
func (e *MapEditor) adoptedGroup(r SelectionEdit, err error) (editorChange, error) {
	if err := e.adoptSelection(r, err); err != nil {
		return editorChange{}, err
	}
	return editorChange{doc: r.Doc, group: &r}, nil
}

func (e *MapEditor) commandMoveRecords(m recordsMove) (editorChange, error) {
	h, ok := e.selectionHost()
	if !ok {
		return editorChange{}, errNoEditHost
	}
	return e.adoptedGroup(h.MoveRecords(m.Refs, m.Offset))
}

func (e *MapEditor) commandDeleteRecords(refs []RecordRef) (editorChange, error) {
	h, ok := e.selectionHost()
	if !ok {
		return editorChange{}, errNoEditHost
	}
	return e.adoptedGroup(h.DeleteRecords(refs))
}

func (e *MapEditor) commandPaste(p recordsPaste) (editorChange, error) {
	h, ok := e.selectionHost()
	if !ok {
		return editorChange{}, errNoEditHost
	}
	return e.adoptedGroup(h.PasteRecords(p.Clip, p.Cell))
}

func (e *MapEditor) commandMoveArea(m areaMove) (editorChange, error) {
	h, _ := e.selectionHost()
	area, ok := h.(InspectionAreaCopy)
	if !ok {
		return editorChange{}, errNoEditHost
	}
	return e.adoptedGroup(area.MoveArea(m.Refs, m.Box, m.Offset))
}

// copyOf is what Ctrl+C keeps: the records refs, or with a box the box's
// tiles, heights and objects with the records refs.
func (e *MapEditor) copyOf(refs []RecordRef, box image.Rectangle) (InspectionClip, string, error) {
	h, ok := e.selectionHost()
	if !ok {
		return nil, "", errNoEditHost
	}
	if box.Empty() {
		return h.CopyRecords(refs)
	}
	area, ok := h.(InspectionAreaCopy)
	if !ok {
		return nil, "", errNoEditHost
	}
	return area.CopyArea(refs, box)
}

// unitsGroup is units given group Group, 0 taking them out of their groups.
type unitsGroup struct {
	Units []RecordRef
	Group int
}

func (e *MapEditor) commandSetUnitsGroup(g unitsGroup) (editorChange, error) {
	h, ok := e.groupsHost()
	if !ok {
		return editorChange{}, errNoEditHost
	}
	next, err := h.SetUnitsGroup(g.Units, g.Group)
	if err != nil {
		return editorChange{}, err
	}
	return e.adopted(next, nil)
}

func (e *MapEditor) commandAddStock(unit int) (editorChange, error) {
	host, ok := e.doc.Edits.(InspectionRecords)
	if !ok {
		return editorChange{}, errNoEditHost
	}
	next, err := host.AddStock(unit)
	if err != nil {
		return editorChange{}, err
	}
	var ref *RecordRef
	if next != nil {
		ref = newestOf(next, "Stock")
	}
	return e.adopted(next, ref)
}

// itemRecipe is item Item's recipe added to a stock or sack.
type itemRecipe struct {
	Kind  string
	Index int
	Item  int64
}

func (e *MapEditor) commandAddItemRecipe(r itemRecipe) (editorChange, error) {
	host, ok := e.doc.Edits.(InspectionRecords)
	if !ok {
		return editorChange{}, errNoEditHost
	}
	next, err := host.AddItemRecipe(r.Kind, r.Index, r.Item)
	if err != nil {
		return editorChange{}, err
	}
	var ref *RecordRef
	if next != nil {
		ref = newestOf(next, "Enchantment")
	}
	return e.adopted(next, ref)
}

// messageWrite is a message's parts written on Page. With Owner the owner is
// pointed at Page in the same edit, or at a new page when Page is
// NewMessagePage.
type messageWrite struct {
	Owner *InspectionMessageOwner
	Page  uint32
	Parts []InspectionMessagePart
}

func (e *MapEditor) commandWriteMessage(w messageWrite) (editorChange, error) {
	host, ok := e.doc.Edits.(InspectionMessages)
	if !ok {
		return editorChange{}, errNoEditHost
	}
	page := w.Page
	var next *InspectionDocument
	var err error
	switch {
	case w.Owner == nil:
		next, err = host.SetMessageParts(w.Page, w.Parts)
	case w.Page == NewMessagePage:
		next, page, err = host.CreateMessageParts(*w.Owner, w.Parts)
	default:
		next, err = host.PointMessageParts(*w.Owner, w.Page, w.Parts)
	}
	if err != nil {
		return editorChange{}, err
	}
	var ref *RecordRef
	switch {
	case next == nil || w.Owner == nil:
	case w.Owner.Kind == "Message":
		ref = newestOf(next, "Action")
	default:
		ref = &RecordRef{Kind: w.Owner.Kind, Index: w.Owner.Index}
	}
	ch, _ := e.adopted(next, ref)
	ch.page = &page
	return ch, nil
}

// boxArea is a box condition over Box's cells joined to trigger Trigger, or
// to a new trigger when Trigger is -1.
type boxArea struct {
	Box     image.Rectangle
	Trigger int
}

func (e *MapEditor) commandAddBoxArea(a boxArea) (editorChange, error) {
	host, ok := e.doc.Edits.(InspectionAreaAuthor)
	if !ok {
		return editorChange{}, errNoEditHost
	}
	next, err := host.AddBoxArea(a.Box, a.Trigger)
	if err != nil {
		return editorChange{}, err
	}
	var ref *RecordRef
	if next != nil && next.Script != nil {
		t := a.Trigger
		if t < 0 {
			t = len(next.Script.Triggers) - 1
		}
		ref = &RecordRef{Kind: "Trigger", Index: t}
	}
	return e.adopted(next, ref)
}

// goalTarget is script goal Goal pointed at Target.
type goalTarget struct {
	Goal   int
	Target RecordRef
}

func (e *MapEditor) commandSetGoalTarget(g goalTarget) (editorChange, error) {
	s, ok := e.script()
	if !ok {
		return editorChange{}, errNoEditHost
	}
	next, err := s.SetGoalTarget(g.Goal, g.Target.Kind, g.Target.Index)
	if err != nil {
		return editorChange{}, err
	}
	return e.adopted(next, &g.Target)
}

// addValue is what add_record adds: a record through the Add window's
// assembly, or a message action with its new page.
type addValue struct {
	record  InspectionNewRecord
	message *messageWrite
}

func (e *MapEditor) commandAdd(v addValue) (editorChange, error) {
	if v.message != nil {
		return e.commandWriteMessage(*v.message)
	}
	return e.commandAddRecord(v.record)
}

// The decoders of these commands.

func recordRefsArg(help string) map[string]any {
	return map[string]any{"type": "array", "description": help, "minItems": 1,
		"items": map[string]any{"type": "object", "additionalProperties": false, "required": []string{"kind", "index"},
			"properties": map[string]any{"kind": textArg("Record kind"), "index": intArg("Record index")}}}
}

func areaArg(help string) map[string]any {
	return pairArg(help+": x0,y0,x1,y1, both corners included", 4)
}

func messagePartsArg() map[string]any {
	return map[string]any{"type": "array", "description": "The message's parts in order, as the message editor's cards: a part the page has by its source (messages lists them), a new part without; a part of the page left out is struck", "minItems": 1,
		"items": map[string]any{"type": "object", "additionalProperties": false, "required": []string{"body"}, "properties": map[string]any{
			"source": intArg("The page's part, from messages; omitted or -1 for a new part"), "body": textArg("The part's text"),
			"speaker": intArg("Speaker, from the speakers messages lists; default -1, none")}}}
}

// groupRefs checks refs as a selection takes them: each a unit, structure or
// sack on the map, named once.
func (e *MapEditor) groupRefs(refs []RecordRef) ([]RecordRef, error) {
	if len(refs) == 0 {
		return nil, fmt.Errorf("records must name at least one record")
	}
	for i, ref := range refs {
		r, err := e.recordOf(ref.Kind, ref.Index)
		if err != nil {
			return nil, err
		}
		if !selectable(r.Kind) || !r.Spatial {
			return nil, fmt.Errorf("a selection takes no %s record", ref.Kind)
		}
		if slices.Contains(refs[:i], ref) {
			return nil, fmt.Errorf("%s %d is named twice", ref.Kind, ref.Index)
		}
	}
	return refs, nil
}

// boxOf is the cell rectangle of x0,y0,x1,y1, both corners on the map.
func (e *MapEditor) boxOf(c []int) (image.Rectangle, error) {
	if len(c) != 4 || c[2] < c[0] || c[3] < c[1] {
		return image.Rectangle{}, fmt.Errorf("an area is x0,y0,x1,y1 with x0 <= x1 and y0 <= y1")
	}
	for _, p := range [][2]int{{c[0], c[1]}, {c[2], c[3]}} {
		if _, err := e.cellOn(p[0], p[1]); err != nil {
			return image.Rectangle{}, err
		}
	}
	return image.Rect(c[0], c[1], c[2]+1, c[3]+1), nil
}

func (e *MapEditor) decodeMoveRecords(raw json.RawMessage) (recordsMove, error) {
	a, err := decodeArgs[struct {
		Records []RecordRef
		DX, DY  int
	}](raw, editorCommandArgs("move_records"))
	if err != nil {
		return recordsMove{}, err
	}
	if _, ok := e.selectionHost(); !ok {
		return recordsMove{}, errNoEditHost
	}
	refs, err := e.groupRefs(a.Records)
	if err != nil {
		return recordsMove{}, err
	}
	return recordsMove{Refs: refs, Offset: image.Pt(a.DX, a.DY)}, nil
}

func (e *MapEditor) decodeDeleteRecords(raw json.RawMessage) ([]RecordRef, error) {
	a, err := decodeArgs[struct{ Records []RecordRef }](raw, editorCommandArgs("delete_records"))
	if err != nil {
		return nil, err
	}
	if _, ok := e.selectionHost(); !ok {
		return nil, errNoEditHost
	}
	return e.groupRefs(a.Records)
}

func (e *MapEditor) decodeCopyRecords(raw json.RawMessage) (recordsPaste, error) {
	a, err := decodeArgs[struct {
		Records []RecordRef
		Area    []int
		X, Y    int
	}](raw, editorCommandArgs("copy_records"))
	if err != nil {
		return recordsPaste{}, err
	}
	if (a.Records == nil) == (a.Area == nil) {
		return recordsPaste{}, fmt.Errorf("copy_records takes records or an area")
	}
	var refs []RecordRef
	var box image.Rectangle
	if a.Area != nil {
		if _, ok := e.doc.Edits.(InspectionAreaCopy); !ok {
			return recordsPaste{}, fmt.Errorf("this map copies records, not areas")
		}
		if box, err = e.boxOf(a.Area); err != nil {
			return recordsPaste{}, err
		}
		refs = e.cellRecords(box, nil)
	} else if refs, err = e.groupRefs(a.Records); err != nil {
		return recordsPaste{}, err
	}
	cell, err := e.cellOn(a.X, a.Y)
	if err != nil {
		return recordsPaste{}, err
	}
	clip, _, err := e.copyOf(refs, box)
	if err != nil {
		return recordsPaste{}, err
	}
	if clip == nil || clip.Len() == 0 {
		return recordsPaste{}, fmt.Errorf("the copy holds no record to paste")
	}
	return recordsPaste{Clip: clip, Cell: cell}, nil
}

func (e *MapEditor) decodeMoveArea(raw json.RawMessage) (areaMove, error) {
	a, err := decodeArgs[struct {
		Area    []int
		DX, DY  int
		Records []RecordRef
	}](raw, editorCommandArgs("move_area"))
	if err != nil {
		return areaMove{}, err
	}
	if _, ok := e.doc.Edits.(InspectionAreaCopy); !ok {
		return areaMove{}, errNoEditHost
	}
	box, err := e.boxOf(a.Area)
	if err != nil {
		return areaMove{}, err
	}
	m := areaMove{Box: box, Offset: image.Pt(a.DX, a.DY), Refs: e.cellRecords(box, nil)}
	if a.Records != nil {
		if m.Refs, err = e.groupRefs(a.Records); err != nil {
			return m, err
		}
		for _, ref := range m.Refs {
			if r, _ := e.recordOf(ref.Kind, ref.Index); !r.Cell.In(box) {
				return m, fmt.Errorf("%s %d does not stand in the area", ref.Kind, ref.Index)
			}
		}
	}
	if moved := box.Add(m.Offset); !moved.In(image.Rect(0, 0, e.doc.Width, e.doc.Height)) {
		return m, fmt.Errorf("the area moved by %d,%d leaves the map", a.DX, a.DY)
	}
	return m, nil
}

func (e *MapEditor) decodeSetUnitsGroup(raw json.RawMessage) (unitsGroup, error) {
	a, err := decodeArgs[struct {
		Units []int
		Group int
	}](raw, editorCommandArgs("set_units_group"))
	if err != nil {
		return unitsGroup{}, err
	}
	h, ok := e.groupsHost()
	if !ok {
		return unitsGroup{}, errNoEditHost
	}
	if len(a.Units) == 0 {
		return unitsGroup{}, fmt.Errorf("units must name at least one unit")
	}
	g := unitsGroup{Group: a.Group}
	for i, u := range a.Units {
		if _, err := e.recordOf("Unit", u); err != nil {
			return g, err
		}
		if slices.Contains(a.Units[:i], u) {
			return g, fmt.Errorf("unit %d is named twice", u)
		}
		g.Units = append(g.Units, RecordRef{Kind: "Unit", Index: u})
	}
	switch {
	case a.Group == -1:
		g.Group = h.FreeGroup()
	case a.Group < 0:
		return g, fmt.Errorf("group is a number, 0 to take the units out or -1 for a new group")
	}
	return g, nil
}

func (e *MapEditor) decodeAddStock(raw json.RawMessage) (int, error) {
	a, err := decodeArgs[struct{ Unit int }](raw, editorCommandArgs("add_stock"))
	if err != nil {
		return 0, err
	}
	if _, ok := e.doc.Edits.(InspectionRecords); !ok {
		return 0, errNoEditHost
	}
	_, err = e.recordOf("Unit", a.Unit)
	return a.Unit, err
}

func (e *MapEditor) decodeAddItemRecipe(raw json.RawMessage) (itemRecipe, error) {
	a, err := decodeArgs[itemRecipe](raw, editorCommandArgs("add_item_recipe"))
	if err != nil {
		return a, err
	}
	if _, ok := e.doc.Edits.(InspectionRecords); !ok {
		return a, errNoEditHost
	}
	if a.Kind != "Stock" && a.Kind != "Sack" {
		return a, fmt.Errorf("a recipe goes in a Stock or a Sack")
	}
	_, err = e.recordOf(a.Kind, a.Index)
	return a, err
}

type messagePartArgs struct {
	Source  *int
	Body    string
	Speaker *int64
}

// messageParts are the parts args name on page as the message editor's
// cards hold them: a part of the page keeps its source, number and tag; the
// first new part of a page that has no parts yet takes the page's first card;
// every other new part is numbered as the editor's "+" numbers a card.
func (e *MapEditor) messageParts(host InspectionMessages, page uint32, args []messagePartArgs) ([]InspectionMessagePart, error) {
	if len(args) == 0 {
		return nil, fmt.Errorf("parts must name at least one part")
	}
	current := host.MessageParts(page)
	if len(current) == 0 {
		return nil, fmt.Errorf("the map has no message page %d", page)
	}
	next := 0
	for _, c := range current {
		next = min(next, c.Number)
	}
	var out []InspectionMessagePart
	used := map[int]bool{}
	for i, p := range args {
		part := InspectionMessagePart{Source: -1, Body: p.Body, Speaker: -1}
		if p.Speaker != nil {
			part.Speaker = *p.Speaker
		}
		switch {
		case p.Source != nil && *p.Source >= 0:
			at := slices.IndexFunc(current, func(c InspectionMessagePart) bool { return c.Source == *p.Source })
			if at < 0 || used[*p.Source] {
				if page == NewMessagePage {
					return nil, fmt.Errorf("part source %d: a new page has no parts yet", *p.Source)
				}
				return nil, fmt.Errorf("part source %d is not a part of page %d, or is named twice", *p.Source, page)
			}
			used[*p.Source] = true
			part.Source, part.Number, part.Tag = current[at].Source, current[at].Number, current[at].Tag
		case p.Source != nil && *p.Source < -1:
			return nil, fmt.Errorf("part source %d is not a part", *p.Source)
		case i == 0 && current[0].Source < 0:
			part.Number = current[0].Number
		default:
			next--
			part.Number = next
		}
		out = append(out, part)
	}
	return out, nil
}

func (e *MapEditor) decodeWriteMessage(raw json.RawMessage) (messageWrite, error) {
	a, err := decodeArgs[struct {
		Page   *int64
		Action *int
		Parts  []messagePartArgs
	}](raw, editorCommandArgs("write_message"))
	if err != nil {
		return messageWrite{}, err
	}
	host, ok := e.doc.Edits.(InspectionMessages)
	if !ok {
		return messageWrite{}, errNoEditHost
	}
	if a.Page == nil && a.Action == nil {
		return messageWrite{}, fmt.Errorf("write_message needs a page, an action, or both")
	}
	w := messageWrite{Page: NewMessagePage}
	if a.Action != nil {
		if _, err := e.recordOf("Action", *a.Action); err != nil {
			return w, err
		}
		w.Owner = &InspectionMessageOwner{Kind: "Action", Index: *a.Action, Key: "value0"}
	}
	if a.Page != nil {
		if *a.Page <= 0 || *a.Page >= int64(NewMessagePage) {
			return w, fmt.Errorf("the map has no message page %d", *a.Page)
		}
		w.Page = uint32(*a.Page)
	}
	w.Parts, err = e.messageParts(host, w.Page, a.Parts)
	return w, err
}

func (e *MapEditor) decodeAddBoxArea(raw json.RawMessage) (boxArea, error) {
	a, err := decodeArgs[struct {
		Area    []int
		Trigger *int
	}](raw, editorCommandArgs("add_box_area"))
	if err != nil {
		return boxArea{}, err
	}
	if _, ok := e.doc.Edits.(InspectionAreaAuthor); !ok || e.doc.Script == nil {
		return boxArea{}, errNoEditHost
	}
	b := boxArea{Trigger: -1}
	if b.Box, err = e.boxOf(a.Area); err != nil {
		return b, err
	}
	if a.Trigger != nil {
		b.Trigger = *a.Trigger
	}
	triggers := e.doc.Script.Triggers
	switch {
	case b.Trigger < -1 || b.Trigger >= len(triggers):
		return b, fmt.Errorf("the map has no trigger %d", b.Trigger)
	case b.Trigger >= 0 && !freePair(triggers[b.Trigger]):
		return b, fmt.Errorf("%s", e.mcpDisplay(fmt.Sprintf(e.word("editor.area.full"), b.Trigger+1)))
	}
	return b, nil
}

func (e *MapEditor) decodeSetGoalTarget(raw json.RawMessage) (goalTarget, error) {
	a, err := decodeArgs[struct {
		Goal  int
		Kind  string
		Index int
	}](raw, editorCommandArgs("set_goal_target"))
	if err != nil {
		return goalTarget{}, err
	}
	s, ok := e.script()
	if !ok {
		return goalTarget{}, errNoEditHost
	}
	goals := s.Goals()
	if a.Goal < 0 || a.Goal >= len(goals) || goals[a.Goal].Pick == "" {
		return goalTarget{}, fmt.Errorf("goal %d takes no target; goals lists those that do", a.Goal)
	}
	if pick := goals[a.Goal].Pick; a.Kind != pick {
		return goalTarget{}, fmt.Errorf("goal %d takes a %s", a.Goal, pick)
	}
	_, err = e.recordOf(a.Kind, a.Index)
	return goalTarget{Goal: a.Goal, Target: RecordRef{Kind: a.Kind, Index: a.Index}}, err
}

// forestArgs is paint's forest: the object classes, the share of the
// covered cells in per cent, and thinning.
type forestArgs struct {
	Kinds []int
	Share *int
	Thin  bool
}

// forestOf is the forest of a stroke as the Forest grid sends it: the
// classes in the grid's order, each one the grid offers.
func (e *MapEditor) forestOf(a forestArgs) (BrushForest, error) {
	f := BrushForest{Share: forestShares[1], Thin: a.Thin}
	if a.Share != nil {
		if *a.Share < 1 || *a.Share > 100 {
			return f, fmt.Errorf("forest share is 1..100 per cent")
		}
		f.Share = *a.Share
	}
	offered := e.forestKinds()
	for i, k := range a.Kinds {
		if !slices.ContainsFunc(offered, func(o BrushObject) bool { return o.Code == k }) || slices.Contains(a.Kinds[:i], k) {
			return f, fmt.Errorf("forest kind %d is not an object class the Forest grid offers, or is named twice", k)
		}
	}
	for _, o := range offered {
		if slices.Contains(a.Kinds, o.Code) {
			f.Kinds = append(f.Kinds, o.Code)
		}
	}
	if len(f.Kinds) == 0 {
		return f, fmt.Errorf("forest kinds must name at least one class")
	}
	return f, nil
}

func forestArg() map[string]any {
	return map[string]any{"type": "object", "additionalProperties": false, "required": []string{"kinds"},
		"description": "Objects plane: a Forest stroke, which plants the classes over a share of the covered cells, or with thin removes them; value is then left out",
		"properties": map[string]any{"kinds": map[string]any{"type": "array", "items": map[string]any{"type": "integer"}, "minItems": 1, "description": "Object codes from brushes"},
			"share": intArg("Share of the covered cells, 1..100 per cent; default 25"), "thin": boolArg("Remove the classes instead of planting them")}}
}

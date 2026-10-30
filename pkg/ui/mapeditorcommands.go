package ui

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"reflect"
	"strconv"
	"strings"

	"againrom/pkg/render/terrain"
)

// The editor command table. A command is one typed edit of the open map: its
// apply method makes the edit through the host operation the window's own
// route uses and adopts the document the host returns, so the edit enters the
// one undo history. The window's routes call these apply methods with the
// values their controls hold; MCP decodes a command's named arguments with
// the checks the window's controls make and calls the same method. No other
// editor file calls a host operation a command owns (internal/archtest).

// editorChange is what an applied command did: the document it adopted, nil
// when the host changed nothing, the record it names, a group edit's outcome
// and the message page a message command wrote.
type editorChange struct {
	doc    *InspectionDocument
	record *RecordRef
	group  *SelectionEdit
	page   *uint32
	// removed marks a record the command deleted: the reference names
	// where it stood.
	removed bool
}

// commandArg is one named argument: its JSON schema and whether it is
// required.
type commandArg struct {
	name     string
	schema   map[string]any
	required bool
}

// editorCommand is a row of the table. run decodes MCP arguments and applies
// the command; opens marks the command that opens a map, which needs no
// editable one.
type editorCommand struct {
	name, help string
	args       []commandArg
	opens      bool
	run        func(*MapEditor, json.RawMessage) (editorChange, error)
}

// editorReader is a typed read of the open map through the window's readers.
type editorReader struct {
	name, help string
	args       []commandArg
	read       func(*MapEditor, json.RawMessage) (any, error)
}

// argumentError is a refusal of the arguments, before anything is applied.
type argumentError struct{ error }

func typedCommand[A any](name, help string, args []commandArg, decode func(*MapEditor, json.RawMessage) (A, error), apply func(*MapEditor, A) (editorChange, error)) editorCommand {
	return editorCommand{name: name, help: help, args: args, run: func(e *MapEditor, raw json.RawMessage) (editorChange, error) {
		a, err := decode(e, raw)
		if err != nil {
			return editorChange{}, argumentError{err}
		}
		return apply(e, a)
	}}
}

func intArg(help string) map[string]any {
	return map[string]any{"type": "integer", "description": help}
}
func boolArg(help string) map[string]any {
	return map[string]any{"type": "boolean", "description": help}
}
func textArg(help string, enum ...string) map[string]any {
	s := map[string]any{"type": "string", "description": help}
	if len(enum) > 0 {
		s["enum"] = enum
	}
	return s
}

// pairArg is a list of n integers, such as a cell's x,y.
func pairArg(help string, n int) map[string]any {
	return map[string]any{"type": "array", "description": help, "items": map[string]any{"type": "integer"}, "minItems": n, "maxItems": n}
}

func cellsArg(help string) map[string]any {
	return map[string]any{"type": "array", "description": help, "items": pairArg("x,y", 2), "minItems": 1}
}

func fieldsArg() map[string]any {
	return map[string]any{"type": "array", "description": "Fields to write: key and value for a number field, key and text for a text field", "minItems": 1,
		"items": map[string]any{"type": "object", "additionalProperties": false, "required": []string{"key"}, "properties": map[string]any{
			"key": textArg("Field key, as the record reader names it"), "value": intArg("Number value"), "text": textArg("Text value")}}}
}

func req(name string, schema map[string]any) commandArg {
	return commandArg{name: name, schema: schema, required: true}
}
func opt(name string, schema map[string]any) commandArg {
	return commandArg{name: name, schema: schema}
}

// commandSchema is the JSON schema of an argument list.
func commandSchema(args []commandArg) map[string]any {
	props := map[string]any{}
	required := []string{}
	for _, a := range args {
		props[a.name] = a.schema
		if a.required {
			required = append(required, a.name)
		}
	}
	return map[string]any{"type": "object", "properties": props, "required": required, "additionalProperties": false}
}

// decodeArgs reads raw into D: an object whose keys args name, with no null at
// any depth and every required key present.
func decodeArgs[D any](raw json.RawMessage, args []commandArg) (D, error) {
	var d D
	var top map[string]json.RawMessage
	if err := json.Unmarshal(raw, &top); err != nil || top == nil {
		return d, fmt.Errorf("arguments must be an object")
	}
	for key := range top {
		known := false
		for _, a := range args {
			known = known || a.name == key
		}
		if !known {
			return d, fmt.Errorf("unknown argument %q", key)
		}
	}
	for _, a := range args {
		if a.required && top[a.name] == nil {
			return d, fmt.Errorf("missing argument %q", a.name)
		}
	}
	var tree any
	if err := json.Unmarshal(raw, &tree); err != nil {
		return d, err
	}
	if hasNull(tree) {
		return d, fmt.Errorf("arguments must not hold null")
	}
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&d); err != nil {
		return d, err
	}
	return d, nil
}

func hasNull(v any) bool {
	switch v := v.(type) {
	case nil:
		return true
	case map[string]any:
		for _, x := range v {
			if hasNull(x) {
				return true
			}
		}
	case []any:
		for _, x := range v {
			if hasNull(x) {
				return true
			}
		}
	}
	return false
}

// editorCommandRows is the table, in the order a map is built from nothing.
func editorCommandRows() []editorCommand {
	newMap := typedCommand("new_map", "Open a new empty map of width x height cells, as the catalogue's new-map question does; unsaved changes still ask unless discard. The result names the playable area inside the 8-cell border ring",
		[]commandArg{req("width", intArg("Cells across")), req("height", intArg("Cells down")),
			opt("altitude", intArg("Starting height, 0..126; default 64")), opt("multiplayer", boolArg("A multiplayer map; default single-player")),
			opt("discard", boolArg("Drop unsaved changes, as the unsaved-changes question's open button does; default false, the question stays open"))},
		(*MapEditor).decodeNewMap, func(e *MapEditor, o mapOpening) (editorChange, error) { return e.commandNewMap(o.source, o.discard) })
	newMap.opens = true
	return []editorCommand{
		newMap,
		typedCommand("paint", "Paint one brush stroke over cells as one edit: terrain class, height change or set height, object code, or passability",
			[]commandArg{req("plane", textArg("Grid layer", "terrain", "height", "objects", "passability")), req("cells", cellsArg("Stroke centres in order")),
				opt("value", intArg("Terrain class or object code from brushes; height change, or the height with set; passability 1 closes, 0 opens")),
				opt("radius", intArg("Cells around each centre; default 0, the centre alone")), opt("shape", textArg("Brush shape; default circle", "circle", "square")),
				opt("fill", textArg("Terrain centre fill; default pattern", "pattern", "random", "variant")), opt("variant", intArg("Terrain variant for fill variant, from 0")),
				opt("set", boolArg("Height: value is the height the first centre takes, the radius blending toward it")), opt("flat", boolArg("Height with set: every covered cell takes value")),
				opt("mode", textArg("Height change: how it is laid; default step", "step", "level", "smooth")), opt("forest", forestArg()),
				opt("verbose", verboseArg())},
			(*MapEditor).decodePaint, (*MapEditor).commandPaint),
		typedCommand("add_record", "Add one record as the Add window does: values by step key (add_steps lists the steps and choices), x,y for a record on a cell; a Message takes parts and adds an action showing a new message page",
			[]commandArg{req("kind", textArg("Record kind", addCommandKinds()...)), opt("x", intArg("Cell x")), opt("y", intArg("Cell y")),
				opt("values", map[string]any{"type": "object", "description": "Step key to value: an integer, a text for a name step, a list of integers for a many step; a step left out takes its default",
					"additionalProperties": map[string]any{"type": []string{"integer", "string", "array"}}}),
				opt("parts", messagePartsArg())},
			(*MapEditor).decodeAddRecord, (*MapEditor).commandAdd),
		typedCommand("set_field", "Set one field of a record, as the field prompt does",
			[]commandArg{req("kind", textArg("Record kind")), req("index", intArg("Record index")), req("key", textArg("Field key")),
				opt("value", intArg("Number value")), opt("text", textArg("Text value"))},
			(*MapEditor).decodeSetField, (*MapEditor).commandSetField),
		typedCommand("set_fields", "Set several fields of one record as one edit, as a script record's popup Apply does",
			[]commandArg{req("kind", textArg("Record kind")), req("index", intArg("Record index")), req("fields", fieldsArg())},
			(*MapEditor).decodeSetFields, (*MapEditor).commandSetFields),
		typedCommand("set_object", "Set fields of a record and optionally move it, as one edit, as the object popup's Apply does",
			[]commandArg{req("kind", textArg("Record kind")), req("index", intArg("Record index")), opt("fields", fieldsArg()), opt("cell", pairArg("New cell x,y", 2))},
			(*MapEditor).decodeSetObject, (*MapEditor).commandSetObject),
		typedCommand("move_record", "Move a record on the map to cell x,y, as a drag does",
			[]commandArg{req("kind", textArg("Record kind")), req("index", intArg("Record index")), req("x", intArg("Cell x")), req("y", intArg("Cell y"))},
			(*MapEditor).decodeMoveRecord, (*MapEditor).commandMove),
		typedCommand("clone_record", "Copy a unit, structure or sack to cell x,y, as Clone does",
			[]commandArg{req("kind", textArg("Record kind")), req("index", intArg("Record index")), req("x", intArg("Cell x")), req("y", intArg("Cell y"))},
			(*MapEditor).decodeCloneRecord, (*MapEditor).commandClone),
		typedCommand("delete_record", "Delete one record, as Del does",
			[]commandArg{req("kind", textArg("Record kind")), req("index", intArg("Record index"))},
			(*MapEditor).decodeDeleteRecord, (*MapEditor).commandDelete),
		typedCommand("set_drop_cell", "Set the cell the party lands on, as the drop tool does on a map without drop zones",
			[]commandArg{req("x", intArg("Cell x")), req("y", intArg("Cell y"))},
			(*MapEditor).decodeDropCell, (*MapEditor).commandSetDropCell),
		typedCommand("add_drop_zone", "Add a drop zone on cell x,y, as the drop tool's click on an empty cell does",
			[]commandArg{req("x", intArg("Cell x")), req("y", intArg("Cell y"))},
			(*MapEditor).decodeAddDropZone, (*MapEditor).commandAddDropZone),
		typedCommand("move_drop_zone", "Move drop zone zone (from 0, in the map reader's drops order) to cell x,y",
			[]commandArg{req("zone", intArg("Zone index")), req("x", intArg("Cell x")), req("y", intArg("Cell y"))},
			(*MapEditor).decodeMoveDropZone, (*MapEditor).commandMoveDropZone),
		typedCommand("delete_drop_zone", "Delete drop zone zone (from 0)",
			[]commandArg{req("zone", intArg("Zone index"))},
			(*MapEditor).decodeDeleteDropZone, (*MapEditor).commandDeleteDropZone),
		typedCommand("write_trigger", "Write a trigger from the wizard's ready items (trigger_items lists them): conditions and actions, each an item and its parameters",
			[]commandArg{opt("index", intArg("Trigger index to rewrite; default -1 adds a trigger")), req("name", textArg("Trigger name")),
				opt("once", boolArg("Run once; default true")), opt("conditions", triggerRowsArg(true)), req("actions", triggerRowsArg(false))},
			(*MapEditor).decodeWriteTrigger, (*MapEditor).commandWriteTrigger),
		typedCommand("move_records", "Move several units, structures and sacks by dx,dy cells as one edit, as a drag of a selection does",
			[]commandArg{req("records", recordRefsArg("The records, as a selection holds them")), req("dx", intArg("Cells right")), req("dy", intArg("Cells down"))},
			(*MapEditor).decodeMoveRecords, (*MapEditor).commandMoveRecords),
		typedCommand("delete_records", "Delete several units, structures and sacks as one edit, as Del on a selection does",
			[]commandArg{req("records", recordRefsArg("The records, as a selection holds them"))},
			(*MapEditor).decodeDeleteRecords, (*MapEditor).commandDeleteRecords),
		typedCommand("copy_records", "Copy records, or an area with its tiles, heights, objects and records, and paste the copy with its top-left on cell x,y as one edit, as Ctrl+C and a Ctrl+V click do",
			[]commandArg{opt("records", recordRefsArg("The records to copy")), opt("area", areaArg("The area to copy")), req("x", intArg("Paste cell x")), req("y", intArg("Paste cell y"))},
			(*MapEditor).decodeCopyRecords, (*MapEditor).commandPaste),
		typedCommand("move_area", "Move the records and the objects of an area by dx,dy cells as one edit, as a drag of a selection box does; the tiles and heights stay",
			[]commandArg{req("area", areaArg("The area")), req("dx", intArg("Cells right")), req("dy", intArg("Cells down")),
				opt("records", recordRefsArg("The records to move with it; default every unit, structure and sack in the area"))},
			(*MapEditor).decodeMoveArea, (*MapEditor).commandMoveArea),
		typedCommand("set_units_group", "Put units in a group, or take them out with group 0, as one edit, as the group keys and the To group question do",
			[]commandArg{req("units", map[string]any{"type": "array", "items": map[string]any{"type": "integer"}, "minItems": 1, "description": "Unit indices, one player's"}),
				req("group", intArg("Group number; 0 takes the units out of their groups; -1 is a new group, the lowest free number"))},
			(*MapEditor).decodeSetUnitsGroup, (*MapEditor).commandSetUnitsGroup),
		typedCommand("add_stock", "Create a unit's shop stock, as K on a selected unit does",
			[]commandArg{req("unit", intArg("Unit index"))},
			(*MapEditor).decodeAddStock, (*MapEditor).commandAddStock),
		typedCommand("add_item_recipe", "Add an item recipe to a stock or a sack, as I and its item prompt do",
			[]commandArg{req("kind", textArg("Record kind", "Stock", "Sack")), req("index", intArg("Record index")), req("item", intArg("Item number, from 1"))},
			(*MapEditor).decodeAddItemRecipe, (*MapEditor).commandAddItemRecipe),
		typedCommand("write_message", "Write a message's parts as the message editor's Apply does: page alone rewrites that page; action alone gives the action a new page; both point the action at the page and rewrite it",
			[]commandArg{opt("page", intArg("Message page, from messages")), opt("action", intArg("Action record index whose message parameter takes the page")), req("parts", messagePartsArg())},
			(*MapEditor).decodeWriteMessage, (*MapEditor).commandWriteMessage),
		typedCommand("add_box_area", "Add a box area over cells to a trigger as one more condition pair, or to a new trigger, as the drawarea (L) tool does",
			[]commandArg{req("area", areaArg("The box")), opt("trigger", intArg("Trigger index; default -1, a new trigger"))},
			(*MapEditor).decodeAddBoxArea, (*MapEditor).commandAddBoxArea),
		typedCommand("set_goal_target", "Point a script goal at a unit or structure, as a pick from the Goals (G) report does; goals lists the goals and the kind each takes",
			[]commandArg{req("goal", intArg("Goal index, from goals")), req("kind", textArg("Record kind", "Unit", "Structure")), req("index", intArg("Record index"))},
			(*MapEditor).decodeSetGoalTarget, (*MapEditor).commandSetGoalTarget),
		typedCommand("set_map_field", "Set a map property, as the Metadata (M) and Pages (X) prompts and the mod package window (W) do; map_fields lists the keys",
			[]commandArg{req("scope", textArg("Field group", "metadata", "pages", "package")), req("key", textArg("Field key")), req("value", textArg("Value as typed"))},
			(*MapEditor).decodeMapField, (*MapEditor).commandSetMapField),
		typedCommand("randomize", "Redraw the whole map's terrain or heights from seed, as Ctrl+E and Ctrl+Shift+R do",
			[]commandArg{req("plane", textArg("What to redraw", "terrain", "heights")), req("seed", intArg("Seed, 0..4294967295")), opt("verbose", verboseArg())},
			(*MapEditor).decodeRandomize, (*MapEditor).commandRandomize),
		typedCommand("undo", "Undo the last edit of the history the window shares", nil, (*MapEditor).decodeUndo, func(e *MapEditor, _ struct{}) (editorChange, error) { return e.commandHistory(false) }),
		typedCommand("redo", "Redo the next edit of the history the window shares", nil, (*MapEditor).decodeRedo, func(e *MapEditor, _ struct{}) (editorChange, error) { return e.commandHistory(true) }),
	}
}

// adopted takes the document a host operation returned.
func (e *MapEditor) adopted(next *InspectionDocument, ref *RecordRef) (editorChange, error) {
	_ = e.adoptEdit(next, nil)
	return editorChange{doc: next, record: ref}, nil
}

// adoptedTerrain takes a document whose terrain the operation redrew.
func (e *MapEditor) adoptedTerrain(next *InspectionDocument) (editorChange, error) {
	_ = e.adoptTerrainEdit(next, nil)
	return editorChange{doc: next}, nil
}

var errNoEditHost = errors.New("this map cannot take that edit")

// commandNewMap opens the new map of source; with discard it answers the
// unsaved-changes question as its open button does.
func (e *MapEditor) commandNewMap(source string, discard bool) (editorChange, error) {
	before := e.doc
	err := e.Open(source)
	if err == nil && discard {
		err = e.discardLeave()
	}
	ch := editorChange{}
	if e.doc != before {
		ch.doc = e.doc
	}
	return ch, err
}

func (e *MapEditor) commandPaint(op BrushStroke) (editorChange, error) {
	host := e.brushHost()
	if host == nil {
		return editorChange{}, errNoEditHost
	}
	next, err := host.Paint(op)
	if err != nil {
		return editorChange{}, err
	}
	return e.adoptedTerrain(next)
}

func (e *MapEditor) commandAddRecord(rec InspectionNewRecord) (editorChange, error) {
	host, ok := e.addHost()
	if !ok {
		return editorChange{}, errNoEditHost
	}
	next, err := host.AddRecord(rec)
	if err != nil {
		return editorChange{}, err
	}
	var ref *RecordRef
	if next != nil {
		ref = &next.AddedRecord
	}
	return e.adopted(next, ref)
}

// fieldEdit is one field of a record: a number, or a text when Text.
type fieldEdit struct {
	Kind  string
	Index int
	Value InspectionFieldValue
}

func (e *MapEditor) commandSetField(f fieldEdit) (editorChange, error) {
	var next *InspectionDocument
	var err error
	if f.Value.Text {
		host, ok := e.doc.Edits.(InspectionTextFields)
		if !ok {
			return editorChange{}, fmt.Errorf("This host cannot edit text fields.")
		}
		next, err = host.SetTextField(f.Kind, f.Index, f.Value.Key, f.Value.TextValue)
	} else {
		next, err = e.doc.Edits.SetField(f.Kind, f.Index, f.Value.Key, f.Value.Value)
	}
	if err != nil {
		return editorChange{}, err
	}
	return e.adopted(next, &RecordRef{Kind: f.Kind, Index: f.Index})
}

// recordFields is several fields of one record and, for the object popup, the
// cell it moves to.
type recordFields struct {
	Kind   string
	Index  int
	Values []InspectionFieldValue
	Move   *image.Point
}

func (e *MapEditor) commandSetFields(o recordFields) (editorChange, error) {
	next, err := e.setFields(o.Kind, o.Index, o.Values)
	if err != nil {
		return editorChange{}, err
	}
	return e.adopted(next, &RecordRef{Kind: o.Kind, Index: o.Index})
}

func (e *MapEditor) commandSetObject(o recordFields) (editorChange, error) {
	var next *InspectionDocument
	var err error
	if host, ok := e.doc.Edits.(InspectionObjectSet); ok {
		next, err = host.SetObject(o.Kind, o.Index, o.Values, o.Move)
	} else if o.Move != nil {
		next, err = e.doc.Edits.Move(o.Kind, o.Index, *o.Move)
	} else {
		next, err = e.setFields(o.Kind, o.Index, o.Values)
	}
	if err != nil {
		return editorChange{}, err
	}
	return e.adopted(next, &RecordRef{Kind: o.Kind, Index: o.Index})
}

// setFields writes values through the host's one-edit seam, or field by
// field on a host without it.
func (e *MapEditor) setFields(kind string, index int, values []InspectionFieldValue) (*InspectionDocument, error) {
	if host, ok := e.doc.Edits.(InspectionFieldSet); ok {
		return host.SetFields(kind, index, values)
	}
	var next *InspectionDocument
	for _, v := range values {
		var d *InspectionDocument
		var err error
		if v.Text {
			host, ok := e.doc.Edits.(InspectionTextFields)
			if !ok {
				return nil, fmt.Errorf("This host cannot edit text fields.")
			}
			d, err = host.SetTextField(kind, index, v.Key, v.TextValue)
		} else {
			d, err = e.doc.Edits.SetField(kind, index, v.Key, v.Value)
		}
		if err != nil {
			return next, err
		}
		if d != nil {
			next = d
		}
	}
	return next, nil
}

// recordAt is a record and the cell a command takes it to.
type recordAt struct {
	Ref  RecordRef
	Cell image.Point
}

func (e *MapEditor) commandMove(m recordAt) (editorChange, error) {
	next, err := e.doc.Edits.Move(m.Ref.Kind, m.Ref.Index, m.Cell)
	if err != nil {
		return editorChange{}, err
	}
	return e.adopted(next, &m.Ref)
}

func (e *MapEditor) commandClone(m recordAt) (editorChange, error) {
	next, err := e.doc.Edits.Clone(m.Ref.Kind, m.Ref.Index, m.Cell)
	if err != nil {
		return editorChange{}, err
	}
	var ref *RecordRef
	if next != nil {
		ref = newestOf(next, m.Ref.Kind)
	}
	return e.adopted(next, ref)
}

// newestOf is the record of kind with the highest index in d.
func newestOf(d *InspectionDocument, kind string) *RecordRef {
	var out *RecordRef
	for _, r := range d.Records {
		if r.Kind == kind && (out == nil || r.Index > out.Index) {
			out = &RecordRef{Kind: kind, Index: r.Index}
		}
	}
	return out
}

func (e *MapEditor) commandDelete(ref RecordRef) (editorChange, error) {
	next, err := e.doc.Edits.Delete(ref.Kind, ref.Index)
	if err != nil {
		return editorChange{}, err
	}
	ch, err := e.adopted(next, &ref)
	ch.removed = true
	return ch, err
}

func (e *MapEditor) commandSetDropCell(cell image.Point) (editorChange, error) {
	s, ok := e.script()
	if !ok {
		return editorChange{}, errNoEditHost
	}
	next, err := s.SetDropCell(cell)
	if err != nil {
		return editorChange{}, err
	}
	return e.adopted(next, nil)
}

// zoneAt is drop zone Zone and the cell it takes.
type zoneAt struct {
	Zone int
	Cell image.Point
}

func (e *MapEditor) commandAddDropZone(cell image.Point) (editorChange, error) {
	z, ok := e.dropZones()
	if !ok {
		return editorChange{}, errNoEditHost
	}
	next, err := z.AddDropZone(cell)
	if err != nil {
		return editorChange{}, err
	}
	return e.adopted(next, nil)
}

func (e *MapEditor) commandMoveDropZone(m zoneAt) (editorChange, error) {
	z, ok := e.dropZones()
	if !ok {
		return editorChange{}, errNoEditHost
	}
	next, err := z.MoveDropZone(m.Zone, m.Cell)
	if err != nil {
		return editorChange{}, err
	}
	return e.adopted(next, nil)
}

func (e *MapEditor) commandDeleteDropZone(zone int) (editorChange, error) {
	z, ok := e.dropZones()
	if !ok {
		return editorChange{}, errNoEditHost
	}
	next, err := z.DeleteDropZone(zone)
	if err != nil {
		return editorChange{}, err
	}
	return e.adopted(next, nil)
}

func (e *MapEditor) commandWriteTrigger(t WizardTrigger) (editorChange, error) {
	host, ok := e.doc.Edits.(InspectionTriggerWizard)
	if !ok {
		return editorChange{}, errNoEditHost
	}
	next, err := host.WriteTrigger(t)
	if err != nil {
		return editorChange{}, err
	}
	var ref *RecordRef
	if next != nil && next.Script != nil {
		index := t.Index
		if index < 0 {
			index = len(next.Script.Triggers) - 1
		}
		if index >= 0 && index < len(next.Script.Triggers) {
			ref = &RecordRef{Kind: "Trigger", Index: index}
		}
	}
	return e.adopted(next, ref)
}

// mapField is one map property: its field group, key and typed value.
type mapField struct {
	Scope, Key, Value string
}

func (e *MapEditor) commandSetMapField(f mapField) (editorChange, error) {
	host, ok := e.workspace()
	if !ok {
		return editorChange{}, errNoEditHost
	}
	next, err := host.SetWorkspaceField(f.Scope, f.Key, f.Value)
	if err != nil {
		return editorChange{}, err
	}
	return e.adopted(next, nil)
}

// wholeMap is a whole-map redraw: heights when Heights, else terrain.
type wholeMap struct {
	Heights bool
	Seed    uint32
}

func (e *MapEditor) commandRandomize(w wholeMap) (editorChange, error) {
	h, ok := e.wholeMapHost()
	if !ok {
		return editorChange{}, errNoEditHost
	}
	run := h.RandomTerrain
	if w.Heights {
		run = h.RandomHeights
	}
	next, err := run(w.Seed)
	if err != nil {
		return editorChange{}, err
	}
	return e.adoptedTerrain(next)
}

// commandHistory is the window's Undo, or its Redo: it leaves a drag or a
// pick, takes one history step when there is one, and keeps the open message
// page on a page the map still holds.
func (e *MapEditor) commandHistory(redo bool) (editorChange, error) {
	e.leave(leaveDrag | leavePick)
	if !e.editable() || !redo && !e.doc.Edits.CanUndo() || redo && !e.doc.Edits.CanRedo() {
		return editorChange{}, nil
	}
	var next *InspectionDocument
	var err error
	if redo {
		next, err = e.doc.Edits.Redo()
	} else {
		next, err = e.doc.Edits.Undo()
	}
	if err != nil {
		return editorChange{}, err
	}
	ch, _ := e.adopted(next, nil)
	e.refreshMessagePage()
	return ch, nil
}

// The decoders: MCP arguments to a command's values, with the checks the
// window's controls make before they call the command.

// mapOpening is a map to open and whether its open drops unsaved changes.
type mapOpening struct {
	source  string
	discard bool
}

func (e *MapEditor) decodeNewMap(raw json.RawMessage) (mapOpening, error) {
	a, err := decodeArgs[struct {
		Width, Height int
		Altitude      *int
		Multiplayer   bool
		Discard       bool
	}](raw, editorCommandArgs("new_map"))
	if err != nil {
		return mapOpening{}, err
	}
	if a.Width <= 0 || a.Height <= 0 {
		return mapOpening{}, fmt.Errorf("width and height must be positive")
	}
	h := NewMapDefaultAltitude
	if a.Altitude != nil {
		var ok bool
		if h, ok = newAltitude(strconv.Itoa(*a.Altitude)); !ok {
			return mapOpening{}, fmt.Errorf("altitude must be 0..%d", NewMapTopAltitude)
		}
	}
	source := newMapAltitudeSource(fmt.Sprintf("new/%dx%d", a.Width, a.Height), h)
	if a.Multiplayer {
		source += NewMapMultiplayer
	}
	return mapOpening{source: source, discard: a.Discard}, nil
}

// editorCommandArgs is the argument list of the command named name.
func editorCommandArgs(name string) []commandArg {
	for _, c := range editorCommandRows() {
		if c.name == name {
			return c.args
		}
	}
	for _, r := range editorReaders() {
		if r.name == name {
			return r.args
		}
	}
	return nil
}

// cellOn is x,y as a cell of the open map.
func (e *MapEditor) cellOn(x, y int) (image.Point, error) {
	c := image.Pt(x, y)
	if e.doc == nil || !c.In(image.Rect(0, 0, e.doc.Width, e.doc.Height)) {
		return c, fmt.Errorf("cell %d,%d is outside the open map", x, y)
	}
	return c, nil
}

func (e *MapEditor) decodePaint(raw json.RawMessage) (BrushStroke, error) {
	a, err := decodeArgs[struct {
		Plane, Shape, Fill, Mode string
		Cells                    [][]int
		Value, Radius, Variant   int
		Set, Flat                bool
		Forest                   *forestArgs
	}](raw, editorCommandArgs("paint"))
	if err != nil {
		return BrushStroke{}, err
	}
	host := e.brushHost()
	if host == nil {
		return BrushStroke{}, errNoEditHost
	}
	op := BrushStroke{Value: a.Value, Radius: a.Radius, Set: a.Set, Flat: a.Flat}
	if a.Radius < 0 || a.Radius > host.BrushMaxRadius() {
		return op, fmt.Errorf("radius must be 0..%d", host.BrushMaxRadius())
	}
	for _, c := range a.Cells {
		if len(c) != 2 {
			return op, fmt.Errorf("a cell is two integers x,y")
		}
		cell, err := e.cellOn(c[0], c[1])
		if err != nil {
			return op, err
		}
		op.Cells = append(op.Cells, cell)
	}
	if len(op.Cells) == 0 {
		return op, fmt.Errorf("cells must name at least one cell")
	}
	shapes := map[string]BrushShape{"": BrushCircle, "circle": BrushCircle, "square": BrushSquare}
	shape, ok := shapes[a.Shape]
	if !ok {
		return op, fmt.Errorf("shape must be circle or square")
	}
	op.Shape = shape
	if (a.Fill != "" || a.Variant != 0) && a.Plane != "terrain" || (a.Set || a.Flat || a.Mode != "") && a.Plane != "height" || a.Forest != nil && a.Plane != "objects" {
		return op, fmt.Errorf("fill and variant are terrain arguments; set, flat and mode are height arguments; forest is an objects argument")
	}
	switch a.Plane {
	case "terrain":
		op.Plane = BrushPlaneTerrain
		known := false
		for _, t := range host.BrushTerrains() {
			known = known || t.Class == a.Value
		}
		if !known {
			return op, fmt.Errorf("terrain class %d is not in the brushes list", a.Value)
		}
		fills := map[string]BrushFillMode{"": BrushFillPattern, "pattern": BrushFillPattern, "random": BrushFillRandom, "variant": BrushFillVariant}
		mode, ok := fills[a.Fill]
		if !ok {
			return op, fmt.Errorf("fill must be pattern, random or variant")
		}
		op.Fill = BrushFill{Mode: mode, Variant: a.Variant}
		if mode == BrushFillVariant {
			n := 1
			if f, ok := e.doc.Edits.(InspectionBrushFill); ok {
				n = f.BrushVariants(a.Value)
			}
			if a.Variant < 0 || a.Variant >= n {
				return op, fmt.Errorf("variant must be 0..%d", n-1)
			}
		}
	case "height":
		op.Plane = BrushPlaneHeight
		modes := map[string]BrushHeightMode{"": HeightModeStep, "step": HeightModeStep, "level": HeightModeLevel, "smooth": HeightModeSmooth}
		if op.Mode, ok = modes[a.Mode]; !ok {
			return op, fmt.Errorf("mode must be step, level or smooth")
		}
		if a.Flat && !a.Set || a.Mode != "" && a.Set {
			return op, fmt.Errorf("flat needs set; mode is for a height change without set")
		}
		if a.Set && (a.Value < 0 || a.Value > 255) || !a.Set && (a.Value == 0 || a.Value < -brushMaxStrength || a.Value > brushMaxStrength) {
			return op, fmt.Errorf("a set height is 0..255; a height change is -%d..%d and not 0", brushMaxStrength, brushMaxStrength)
		}
	case "objects":
		op.Plane = BrushPlaneObjects
		known := false
		for _, o := range host.BrushObjects() {
			known = known || o.Code == a.Value
		}
		if !known {
			return op, fmt.Errorf("object code %d is not in the brushes list", a.Value)
		}
		if a.Forest != nil {
			if a.Value != 0 {
				return op, fmt.Errorf("a forest stroke takes forest kinds, not value")
			}
			if op.Forest, err = e.forestOf(*a.Forest); err != nil {
				return op, err
			}
		}
	case "passability":
		op.Plane = BrushPlanePassability
		if a.Value != 0 && a.Value != 1 {
			return op, fmt.Errorf("passability value is 1 to close a cell or 0 to open it")
		}
	default:
		return op, fmt.Errorf("plane must be terrain, height, objects or passability")
	}
	return op, nil
}

// addCommandKinds are the Add window kinds a command adds: each kind whose
// steps the window asks before one host add.
func addCommandKinds() []string {
	var out []string
	for _, k := range addKinds {
		if len(k.Steps) > 0 && k.Kind != "Script" {
			out = append(out, k.Kind)
		}
	}
	return append(out, "Action", "Condition", "Message")
}

// decodeAddRecord is a record the Add window's walk assembles, or for a
// Message the message editor's first Apply: an action showing a new page.
func (e *MapEditor) decodeAddRecord(raw json.RawMessage) (addValue, error) {
	a, err := decodeArgs[struct {
		Kind   string
		X, Y   *int
		Values map[string]json.RawMessage
		Parts  []messagePartArgs
	}](raw, editorCommandArgs("add_record"))
	if err != nil {
		return addValue{}, err
	}
	if a.Kind != "Message" {
		if a.Parts != nil {
			return addValue{}, fmt.Errorf("parts are a Message's")
		}
		rec, err := e.newRecordOf(a.Kind, a.X, a.Y, a.Values, nil)
		return addValue{record: rec}, err
	}
	host, ok := e.doc.Edits.(InspectionMessages)
	if !ok {
		return addValue{}, errNoEditHost
	}
	if a.X != nil || a.Y != nil || a.Values != nil || a.Parts == nil {
		return addValue{}, fmt.Errorf("a Message takes parts alone")
	}
	parts, err := e.messageParts(host, NewMessagePage, a.Parts)
	return addValue{message: &messageWrite{Owner: &InspectionMessageOwner{Kind: "Message"}, Page: NewMessagePage, Parts: parts}}, err
}

// addStepSeen is one step as newRecordOf walked it: the host's offer, whether
// the window leaves the step out, and the host's refusal of its value.
type addStepSeen func(st addStep, offer InspectionNewStep, skipped bool, refused error)

// newRecordOf is the record the Add window adds for kind with the given
// values. It walks an Add run through the window's own steps: addOffer gives
// each step's offer and skip, each asked step takes its value or the window's
// default, each value passes the host's check of its step as the window's
// Enter checks it, and addRecord assembles the record. With seen it reads
// instead of refusing: a missing cell or many value is left out and a
// refused value is reported to seen.
func (e *MapEditor) newRecordOf(kind string, x, y *int, values map[string]json.RawMessage, seen addStepSeen) (InspectionNewRecord, error) {
	row, ok := addKindOf(kind)
	host, hok := e.addHost()
	if !ok || !hok || kind == "Script" || kind == "Message" {
		return InspectionNewRecord{}, fmt.Errorf("add_record adds no %q record", kind)
	}
	r := &addRun{kind: kind, at: -1}
	r.setSteps(row.Steps)
	used, onCell := map[string]bool{}, false
	check := func(i int, offer InspectionNewStep) error {
		err := host.CheckNewRecord(e.addRecord(r, i+1), r.steps[i].Key)
		if seen != nil {
			seen(r.steps[i], offer, false, err)
			return nil
		}
		return err
	}
	for i, st := range r.steps {
		if st.Control == addCell {
			onCell = true
			if x == nil || y == nil {
				if seen != nil {
					seen(st, InspectionNewStep{}, false, nil)
					continue
				}
				return InspectionNewRecord{}, fmt.Errorf("a %s needs x and y", kind)
			}
			c, err := e.cellOn(*x, *y)
			if err != nil {
				return InspectionNewRecord{}, err
			}
			r.texts[i] = fmt.Sprintf("%d,%d", c.X, c.Y)
			if err := check(i, InspectionNewStep{}); err != nil {
				return InspectionNewRecord{}, err
			}
			continue
		}
		var offer InspectionNewStep
		offer, r.skip[i] = e.addOffer(host, r, i)
		given, has := values[st.Key]
		used[st.Key] = true
		if r.skip[i] {
			if has {
				return InspectionNewRecord{}, fmt.Errorf("step %q is not asked for this %s", st.Key, kind)
			}
			if seen != nil {
				seen(st, offer, true, nil)
			}
			continue
		}
		r.texts[i] = addDefault(st, offer)
		switch st.Control {
		case addName:
			if has && json.Unmarshal(given, &r.texts[i]) != nil {
				return InspectionNewRecord{}, fmt.Errorf("step %q takes a text", st.Key)
			}
		case addMany:
			var many []int64
			var one int64
			switch {
			case !has && seen != nil:
				seen(st, offer, false, nil)
				continue
			case !has:
				return InspectionNewRecord{}, fmt.Errorf("step %q needs at least one value", st.Key)
			case json.Unmarshal(given, &one) == nil:
				many = []int64{one}
			case json.Unmarshal(given, &many) != nil || len(many) == 0:
				return InspectionNewRecord{}, fmt.Errorf("step %q takes an integer or a list of integers", st.Key)
			}
			for _, v := range many {
				r.many[i] = append(r.many[i], v)
				if err := check(i, offer); err != nil {
					return InspectionNewRecord{}, err
				}
			}
			continue
		default:
			v := offer.Value
			if has && json.Unmarshal(given, &v) != nil {
				return InspectionNewRecord{}, fmt.Errorf("step %q takes an integer", st.Key)
			}
			r.texts[i] = strconv.FormatInt(v, 10)
		}
		if err := check(i, offer); err != nil {
			return InspectionNewRecord{}, err
		}
	}
	for key := range values {
		if !used[key] {
			return InspectionNewRecord{}, fmt.Errorf("a %s has no step %q", kind, key)
		}
	}
	if (x != nil || y != nil) && !onCell {
		return InspectionNewRecord{}, fmt.Errorf("a %s stands on no cell", kind)
	}
	return e.addRecord(r, len(r.steps)), nil
}

// recordOf is the open document's record ref names.
func (e *MapEditor) recordOf(kind string, index int) (InspectionRecord, error) {
	if e.doc != nil {
		for _, r := range e.doc.Records {
			if r.Kind == kind && r.Index == index {
				return r, nil
			}
		}
	}
	return InspectionRecord{}, fmt.Errorf("the map has no %s record %d", kind, index)
}

// fieldValue is the value of field key of a record: a number, or a text for
// a text field, checked against the record's fields as the popup offers them.
// A number passes the window's fieldNumber as the field prompt's text: no
// value in an inheriting field is its empty text and inherits, and a 0 that
// the field stores as inherited is refused with the prompt's words.
func (e *MapEditor) fieldValue(kind string, index int, key string, value *int64, text *string) (InspectionFieldValue, error) {
	if _, err := e.recordOf(kind, index); err != nil {
		return InspectionFieldValue{}, err
	}
	for _, f := range e.doc.Edits.Fields(kind, index) {
		if f.Key != key || f.Action {
			continue
		}
		switch {
		case f.ReadOnly:
			return InspectionFieldValue{}, fmt.Errorf("field %q is read-only", key)
		case f.Text && text == nil, f.Text && value != nil:
			return InspectionFieldValue{}, fmt.Errorf("field %q takes text", key)
		case !f.Text && value == nil && !f.Inherit, !f.Text && text != nil:
			return InspectionFieldValue{}, fmt.Errorf("field %q takes a number value", key)
		case f.Text:
			return InspectionFieldValue{Key: key, Text: true, TextValue: *text}, nil
		case value != nil && *value == NewMessageChoice:
			return InspectionFieldValue{}, fmt.Errorf("a new message is written in the window's message editor")
		}
		typed := ""
		if value != nil {
			typed = strconv.FormatInt(*value, 10)
		}
		n, err := fieldNumber(f, typed)
		if err != nil {
			return InspectionFieldValue{}, errors.New(e.fieldNumberNote(f, err))
		}
		return InspectionFieldValue{Key: key, Value: n}, nil
	}
	return InspectionFieldValue{}, fmt.Errorf("%s %d has no field %q", kind, index, key)
}

type fieldArgs struct {
	Key   string
	Value *int64
	Text  *string
}

func (e *MapEditor) decodeSetField(raw json.RawMessage) (fieldEdit, error) {
	a, err := decodeArgs[struct {
		Kind  string
		Index int
		fieldArgs
	}](raw, editorCommandArgs("set_field"))
	if err != nil {
		return fieldEdit{}, err
	}
	v, err := e.fieldValue(a.Kind, a.Index, a.Key, a.Value, a.Text)
	return fieldEdit{Kind: a.Kind, Index: a.Index, Value: v}, err
}

func (e *MapEditor) objectValues(kind string, index int, fields []fieldArgs) (recordFields, error) {
	o := recordFields{Kind: kind, Index: index}
	for _, f := range fields {
		v, err := e.fieldValue(kind, index, f.Key, f.Value, f.Text)
		if err != nil {
			return o, err
		}
		o.Values = append(o.Values, v)
	}
	return o, nil
}

func (e *MapEditor) decodeSetFields(raw json.RawMessage) (recordFields, error) {
	a, err := decodeArgs[struct {
		Kind   string
		Index  int
		Fields []fieldArgs
	}](raw, editorCommandArgs("set_fields"))
	if err != nil {
		return recordFields{}, err
	}
	if len(a.Fields) == 0 {
		return recordFields{}, fmt.Errorf("fields must name at least one field")
	}
	return e.objectValues(a.Kind, a.Index, a.Fields)
}

func (e *MapEditor) decodeSetObject(raw json.RawMessage) (recordFields, error) {
	a, err := decodeArgs[struct {
		Kind   string
		Index  int
		Fields []fieldArgs
		Cell   []int
	}](raw, editorCommandArgs("set_object"))
	if err != nil {
		return recordFields{}, err
	}
	o, err := e.objectValues(a.Kind, a.Index, a.Fields)
	if err != nil {
		return o, err
	}
	if a.Cell != nil {
		r, _ := e.recordOf(a.Kind, a.Index)
		if !editableKind(a.Kind) || !r.Spatial {
			return o, fmt.Errorf("a %s does not move", a.Kind)
		}
		if len(a.Cell) != 2 {
			return o, fmt.Errorf("cell is two integers x,y")
		}
		c, err := e.cellOn(a.Cell[0], a.Cell[1])
		if err != nil {
			return o, err
		}
		o.Move = &c
	}
	if len(o.Values) == 0 && o.Move == nil {
		return o, fmt.Errorf("set_object needs fields or a cell")
	}
	return o, nil
}

type recordArgs struct {
	Kind  string
	Index int
	X, Y  int
}

func (e *MapEditor) recordAtOf(name string, raw json.RawMessage) (recordAt, error) {
	a, err := decodeArgs[recordArgs](raw, editorCommandArgs(name))
	if err != nil {
		return recordAt{}, err
	}
	r, err := e.recordOf(a.Kind, a.Index)
	if err != nil {
		return recordAt{}, err
	}
	if !r.Spatial || name == "clone_record" && !editableKind(a.Kind) {
		return recordAt{}, fmt.Errorf("a %s record does not stand on the map here", a.Kind)
	}
	c, err := e.cellOn(a.X, a.Y)
	return recordAt{Ref: RecordRef{Kind: a.Kind, Index: a.Index}, Cell: c}, err
}

func (e *MapEditor) decodeMoveRecord(raw json.RawMessage) (recordAt, error) {
	return e.recordAtOf("move_record", raw)
}

func (e *MapEditor) decodeCloneRecord(raw json.RawMessage) (recordAt, error) {
	return e.recordAtOf("clone_record", raw)
}

func (e *MapEditor) decodeDeleteRecord(raw json.RawMessage) (RecordRef, error) {
	a, err := decodeArgs[recordArgs](raw, editorCommandArgs("delete_record"))
	if err != nil {
		return RecordRef{}, err
	}
	_, err = e.recordOf(a.Kind, a.Index)
	return RecordRef{Kind: a.Kind, Index: a.Index}, err
}

type zoneArgs struct{ Zone, X, Y int }

func (e *MapEditor) decodeDropCell(raw json.RawMessage) (image.Point, error) {
	a, err := decodeArgs[zoneArgs](raw, editorCommandArgs("set_drop_cell"))
	if err != nil {
		return image.Point{}, err
	}
	if _, ok := e.dropZones(); ok {
		return image.Point{}, fmt.Errorf("this map keeps drop zones: use add_drop_zone or move_drop_zone")
	}
	return e.cellOn(a.X, a.Y)
}

func (e *MapEditor) decodeAddDropZone(raw json.RawMessage) (image.Point, error) {
	a, err := decodeArgs[zoneArgs](raw, editorCommandArgs("add_drop_zone"))
	if err != nil {
		return image.Point{}, err
	}
	c, err := e.cellOn(a.X, a.Y)
	if err == nil {
		for _, d := range e.doc.Drops {
			if d == c {
				return c, fmt.Errorf("a drop zone already stands on %d,%d", c.X, c.Y)
			}
		}
	}
	return c, err
}

func (e *MapEditor) zoneOf(zone int) error {
	if zone < 0 || zone >= len(e.doc.Drops) {
		return fmt.Errorf("the map has no drop zone %d", zone)
	}
	return nil
}

func (e *MapEditor) decodeMoveDropZone(raw json.RawMessage) (zoneAt, error) {
	a, err := decodeArgs[zoneArgs](raw, editorCommandArgs("move_drop_zone"))
	if err != nil {
		return zoneAt{}, err
	}
	if err := e.zoneOf(a.Zone); err != nil {
		return zoneAt{}, err
	}
	c, err := e.cellOn(a.X, a.Y)
	return zoneAt{Zone: a.Zone, Cell: c}, err
}

func (e *MapEditor) decodeDeleteDropZone(raw json.RawMessage) (int, error) {
	a, err := decodeArgs[zoneArgs](raw, editorCommandArgs("delete_drop_zone"))
	if err != nil {
		return 0, err
	}
	return a.Zone, e.zoneOf(a.Zone)
}

// triggerRowsArg is the schema of a write_trigger row list.
func triggerRowsArg(condition bool) map[string]any {
	props := map[string]any{"item": textArg("Item key from trigger_items"),
		"params": map[string]any{"type": "array", "description": "One entry per item parameter: an integer, or a list of integers for a cell x,y or an area x1,y1,x2,y2; missing trailing ones take their defaults",
			"items": map[string]any{"type": []string{"integer", "array"}}}}
	help, most := "Actions, up to 4", 4
	if condition {
		props["comparison"] = intArg("For an item whose comparison the row chooses: 0..5")
		help, most = "Conditions, up to 3", 3
	}
	return map[string]any{"type": "array", "description": help, "maxItems": most,
		"items": map[string]any{"type": "object", "additionalProperties": false, "required": []string{"item"}, "properties": props}}
}

// wizItemKey is an item's key: its label word without the wizard prefix.
func wizItemKey(it wizItem) string { return strings.TrimPrefix(it.labelWord, "editor.wizard.") }

type triggerRowArgs struct {
	Item       string
	Params     []json.RawMessage
	Comparison *int
}

// commandWizardRow is the wizard row an MCP row names, its parameter texts as the
// row's fields would hold them.
func (e *MapEditor) commandWizardRow(condition bool, a triggerRowArgs) (wizRow, error) {
	items := wizItems(condition)
	at := -1
	for i, it := range items {
		if wizItemKey(it) == a.Item {
			at = i
		}
	}
	if at < 0 {
		return wizRow{}, fmt.Errorf("no ready item %q; trigger_items lists them", a.Item)
	}
	it := items[at]
	vals := e.wizardDefaults(condition, at)
	if len(a.Params) > len(it.params) {
		return wizRow{}, fmt.Errorf("item %q takes %d parameters", a.Item, len(it.params))
	}
	for k, p := range a.Params {
		var one int64
		var many []int64
		if json.Unmarshal(p, &one) == nil {
			vals[k+1] = strconv.FormatInt(one, 10)
			continue
		}
		if json.Unmarshal(p, &many) != nil || len(many) == 0 {
			return wizRow{}, fmt.Errorf("item %q parameter %d is an integer or a list of integers", a.Item, k)
		}
		parts := make([]string, len(many))
		for i, v := range many {
			parts[i] = strconv.FormatInt(v, 10)
		}
		vals[k+1] = strings.Join(parts, ",")
	}
	if a.Comparison != nil {
		if !condition || it.cmp >= 0 {
			return wizRow{}, fmt.Errorf("item %q has a fixed comparison", a.Item)
		}
		vals[len(it.params)+1] = strconv.Itoa(*a.Comparison)
	}
	for k, p := range it.params {
		if p.kind != wpUnit && p.kind != wpGroup {
			continue
		}
		v, err := strconv.ParseInt(strings.TrimSpace(vals[k+1]), 10, 64)
		if err != nil || !e.wizardTargetAllowed(p.kind, v) {
			return wizRow{}, fmt.Errorf("item %q parameter %d names %s %s, which the map does not hold; trigger_items lists the choices", a.Item, k, wizParamNames[p.kind], vals[k+1])
		}
	}
	return wizRow{condition: condition, item: at, origItem: -2, vals: vals}, nil
}

func (e *MapEditor) decodeWriteTrigger(raw json.RawMessage) (WizardTrigger, error) {
	a, err := decodeArgs[struct {
		Index      *int
		Name       string
		Once       *bool
		Conditions []triggerRowArgs
		Actions    []triggerRowArgs
	}](raw, editorCommandArgs("write_trigger"))
	if err != nil {
		return WizardTrigger{}, err
	}
	if _, ok := e.doc.Edits.(InspectionTriggerWizard); !ok || e.doc.Script == nil {
		return WizardTrigger{}, errNoEditHost
	}
	t := WizardTrigger{Index: -1, Name: a.Name, Once: 1}
	if a.Index != nil {
		t.Index = *a.Index
		if t.Index < -1 || t.Index >= len(e.doc.Script.Triggers) {
			return t, fmt.Errorf("the map has no trigger %d", t.Index)
		}
	}
	if a.Once != nil && !*a.Once {
		t.Once = 0
	}
	var rows []wizRow
	for _, list := range []struct {
		condition bool
		rows      []triggerRowArgs
	}{{true, a.Conditions}, {false, a.Actions}} {
		for _, r := range list.rows {
			row, err := e.commandWizardRow(list.condition, r)
			if err != nil {
				return t, err
			}
			rows = append(rows, row)
		}
	}
	if len(a.Actions) == 0 {
		return t, fmt.Errorf("a trigger needs at least one action")
	}
	t, err = e.rowsPlan(t, rows)
	if err != nil {
		return t, errors.New(e.mcpDisplay(err.Error()))
	}
	return t, nil
}

func (e *MapEditor) decodeMapField(raw json.RawMessage) (mapField, error) {
	a, err := decodeArgs[mapField](raw, editorCommandArgs("set_map_field"))
	if err != nil {
		return a, err
	}
	if a.Scope != "metadata" && a.Scope != "pages" && a.Scope != "package" {
		return a, fmt.Errorf("scope must be metadata, pages or package")
	}
	host, ok := e.workspace()
	if !ok {
		return a, errNoEditHost
	}
	if a.Scope == "package" && packageWindowOnly[a.Key] {
		return a, fmt.Errorf("package field %q opens a folder or a mission; the mod package window (W) does that", a.Key)
	}
	for _, f := range host.WorkspaceFields(a.Scope) {
		if f.Key == a.Key && !f.ReadOnly && !f.Action {
			return a, nil
		}
	}
	return a, fmt.Errorf("%s has no field %q", a.Scope, a.Key)
}

// packageWindowOnly are the mod package fields set_map_field leaves to the
// window: the working folder, the output folder and the project's mission
// choice open or switch what the window edits, not a value of the package.
var packageWindowOnly = map[string]bool{"folder": true, "folder-open": true, "output": true, "source": true}

func (e *MapEditor) decodeRandomize(raw json.RawMessage) (wholeMap, error) {
	a, err := decodeArgs[struct {
		Plane string
		Seed  int64
	}](raw, editorCommandArgs("randomize"))
	if err != nil {
		return wholeMap{}, err
	}
	if a.Plane != "terrain" && a.Plane != "heights" {
		return wholeMap{}, fmt.Errorf("plane must be terrain or heights")
	}
	if a.Seed < 0 || a.Seed > 1<<32-1 {
		return wholeMap{}, fmt.Errorf("seed must be 0..4294967295")
	}
	if _, ok := e.wholeMapHost(); !ok {
		return wholeMap{}, errNoEditHost
	}
	return wholeMap{Heights: a.Plane == "heights", Seed: uint32(a.Seed)}, nil
}

func (e *MapEditor) decodeUndo(raw json.RawMessage) (struct{}, error) {
	if _, err := decodeArgs[struct{}](raw, nil); err != nil {
		return struct{}{}, err
	}
	if !e.doc.Edits.CanUndo() {
		return struct{}{}, fmt.Errorf("there is nothing to undo")
	}
	return struct{}{}, nil
}

func (e *MapEditor) decodeRedo(raw json.RawMessage) (struct{}, error) {
	if _, err := decodeArgs[struct{}](raw, nil); err != nil {
		return struct{}{}, err
	}
	if !e.doc.Edits.CanRedo() {
		return struct{}{}, fmt.Errorf("there is nothing to redo")
	}
	return struct{}{}, nil
}

// mcpRun applies the command for an MCP call. It refuses while a window
// prompt waits for the person, as the path tools do; a host refusal shows on
// the status line as the window's own routes show it; the result names what
// changed.
func (c editorCommand) mcpRun(e *MapEditor, raw json.RawMessage) (any, error) {
	if e.pathDialog.pending != nil || e.prompt.open() || e.objectTyping() {
		return nil, fmt.Errorf("finish or cancel the current window prompt first")
	}
	if !c.opens && (!e.editable() || e.catalog) {
		return nil, fmt.Errorf("no editable map is open")
	}
	before := e.doc
	if c.opens {
		before = nil
	}
	cells := true
	if commandTakes(c.args, "verbose") {
		var err error
		if raw, cells, err = takeVerbose(raw); err != nil {
			return nil, err
		}
	}
	ch, err := c.run(e, raw)
	if err != nil {
		var bad argumentError
		if !errors.As(err, &bad) {
			e.message = err.Error()
		}
		return nil, errors.New(e.mcpDisplay(err.Error()))
	}
	return e.changeReport(before, ch, cells), nil
}

func verboseArg() map[string]any {
	return boolArg("List every changed cell; default false, the count, the bounding cells and the planes alone")
}

// commandTakes reports that args has an argument named name.
func commandTakes(args []commandArg, name string) bool {
	for _, a := range args {
		if a.name == name {
			return true
		}
	}
	return false
}

// takeVerbose is raw without its verbose argument, and that argument's value.
func takeVerbose(raw json.RawMessage) (json.RawMessage, bool, error) {
	var top map[string]json.RawMessage
	if err := json.Unmarshal(raw, &top); err != nil || top == nil {
		return raw, false, fmt.Errorf("arguments must be an object")
	}
	given, ok := top["verbose"]
	if !ok {
		return raw, false, nil
	}
	var verbose bool
	if err := json.Unmarshal(given, &verbose); err != nil || string(given) == "null" {
		return raw, false, fmt.Errorf("verbose is true or false")
	}
	delete(top, "verbose")
	out, err := json.Marshal(top)
	return out, verbose, err
}

// The most cells and records a change report lists; the totals are given.
const (
	reportCells   = 1024
	reportRecords = 256
)

// InspectionStore is implemented by a host that reads what a document's map
// stores, for the change report.
type InspectionStore interface {
	Stored(d *InspectionDocument) (StoredMap, error)
}

// StoredMap is what a map stores: each cell's tile word with its passability
// bit, its height and its object byte, row-major, and the stored bytes of
// each record the host reads by reference.
type StoredMap struct {
	Tiles     []uint16
	Altitudes []uint8
	Overlay   []uint8
	Records   map[RecordRef][]byte
}

// storedMap is what d's map stores, through the host; a host without the
// seam gives the planes the view draws and no record bytes.
func storedMap(d *InspectionDocument) StoredMap {
	if h, ok := d.Edits.(InspectionStore); ok {
		if s, err := h.Stored(d); err == nil {
			return s
		}
	}
	g := d.Viewer.grid
	return StoredMap{Tiles: g.Tiles, Altitudes: g.Altitudes, Overlay: g.Overlay}
}

// cellPlanes are the planes a cell stores, each with its test of change.
var cellPlanes = []struct {
	name    string
	changed func(a, b StoredMap, i int) bool
}{
	{"terrain", func(a, b StoredMap, i int) bool {
		return differs(a.Tiles, b.Tiles, i, func(w uint16) uint16 { return w &^ terrain.ImpassableBit })
	}},
	{"passability", func(a, b StoredMap, i int) bool {
		return differs(a.Tiles, b.Tiles, i, func(w uint16) uint16 { return w & terrain.ImpassableBit })
	}},
	{"height", func(a, b StoredMap, i int) bool { return differs(a.Altitudes, b.Altitudes, i, nil) }},
	{"objects", func(a, b StoredMap, i int) bool { return differs(a.Overlay, b.Overlay, i, nil) }},
}

// changeReport is what a command changed, compared on what the map stores:
// whether the host changed the map, the record the command names with its
// script id, the cells whose stored words differ (listed when cells) with
// their count, bounding cells and count per plane, the records added,
// removed or with other stored bytes, by reference, and the map check's
// findings on those records. A map newly opened, before nil, or one of
// another size names its size and playable area instead.
func (e *MapEditor) changeReport(before *InspectionDocument, ch editorChange, cells bool) map[string]any {
	out := map[string]any{"changed": ch.doc != nil, "record": ch.record, "dirty": e.Dirty(),
		"message": e.mcpDisplay(e.message), "prompt_open": e.prompt.open()}
	if r := ch.record; r != nil && !ch.removed && (r.Kind == "Unit" || r.Kind == "Structure") {
		if rec, err := e.recordOf(r.Kind, r.Index); err == nil {
			out["script_id"] = rec.ScriptID
		}
	}
	if e.editable() {
		out["can_undo"], out["can_redo"] = e.doc.Edits.CanUndo(), e.doc.Edits.CanRedo()
	}
	if g := ch.group; g != nil {
		out["selected"], out["note"], out["refusals"] = g.Selected, g.Note, g.Refusals
	}
	if ch.page != nil {
		out["page"] = *ch.page
	}
	if ch.doc == nil {
		return out
	}
	after := e.doc
	if before == nil || before.Width != after.Width || before.Height != after.Height || before.Viewer == nil || after.Viewer == nil {
		out["width"], out["height"], out["playable"] = after.Width, after.Height, PlayableText(after.Playable)
		return out
	}
	a, b := storedMap(before), storedMap(after)
	list := [][]int{}
	planes := map[string]int{}
	total := 0
	var box image.Rectangle
	for i := 0; i < after.Width*after.Height; i++ {
		changed := false
		for _, p := range cellPlanes {
			if p.changed(a, b, i) {
				planes[p.name]++
				changed = true
			}
		}
		if changed {
			c := image.Pt(i%after.Width, i/after.Width)
			box = box.Union(image.Rectangle{Min: c, Max: c.Add(image.Pt(1, 1))})
			if total++; len(list) < reportCells {
				list = append(list, []int{c.X, c.Y})
			}
		}
	}
	if cells {
		out["cells"] = list
	}
	out["cells_total"], out["planes"] = total, planes
	if total > 0 {
		out["cells_box"] = []int{box.Min.X, box.Min.Y, box.Max.X - 1, box.Max.Y - 1}
	}
	refs := changedRecords(before.Records, after.Records, a.Records, b.Records)
	out["records_total"] = len(refs)
	out["records"] = refs[:min(len(refs), reportRecords)]
	out["warnings"] = e.editWarnings(refs, ch.record)
	return out
}

// editWarnings are the map check's findings on the records an edit touched:
// the records it changed and the one it names.
func (e *MapEditor) editWarnings(refs []RecordRef, named *RecordRef) []map[string]any {
	out := []map[string]any{}
	touched := map[RecordRef]bool{}
	for _, r := range refs {
		touched[r] = true
	}
	if named != nil {
		touched[*named] = true
	}
	c, ok := e.checker()
	if !ok || len(touched) == 0 {
		return out
	}
	for _, f := range c.Report() {
		if f.Kind != "" && touched[RecordRef{Kind: f.Kind, Index: f.Index}] {
			out = append(out, e.findingRow(f))
		}
	}
	return out
}

// PlayableText names a playable area as the editor's results give it: its
// corner cells x0,y0..x1,y1, both included, or none.
func PlayableText(r image.Rectangle) string {
	if r.Empty() {
		return "none"
	}
	return fmt.Sprintf("%d,%d..%d,%d", r.Min.X, r.Min.Y, r.Max.X-1, r.Max.Y-1)
}

// differs is whether cell i differs between two planes, each value seen
// through part when part is set.
func differs[T comparable](a, b []T, i int, part func(T) T) bool {
	if i >= len(a) || i >= len(b) {
		return len(a) != len(b)
	}
	if part != nil {
		return part(a[i]) != part(b[i])
	}
	return a[i] != b[i]
}

// changedRecords are the references whose record differs between two
// documents, in the order of the later one, then the removed ones: a record
// the host stores by reference differs when its stored bytes do, any other
// when its list entry does.
func changedRecords(before, after []InspectionRecord, was, now map[RecordRef][]byte) []RecordRef {
	old := map[RecordRef]InspectionRecord{}
	for _, r := range before {
		r.Preview = nil
		old[RecordRef{Kind: r.Kind, Index: r.Index}] = r
	}
	out := []RecordRef{}
	for _, r := range after {
		ref := RecordRef{Kind: r.Kind, Index: r.Index}
		r.Preview = nil
		prior, ok := old[ref]
		delete(old, ref)
		storedWas, okWas := was[ref]
		storedNow, okNow := now[ref]
		switch {
		case !ok:
			out = append(out, ref)
		case okWas && okNow:
			if !bytes.Equal(storedWas, storedNow) {
				out = append(out, ref)
			}
		case !reflect.DeepEqual(prior, r):
			out = append(out, ref)
		}
	}
	for _, r := range before {
		if ref := (RecordRef{Kind: r.Kind, Index: r.Index}); old[ref].Kind != "" {
			out = append(out, ref)
			delete(old, ref)
		}
	}
	return out
}

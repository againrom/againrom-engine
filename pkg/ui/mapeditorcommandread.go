package ui

import (
	"encoding/json"
	"fmt"
	"image"
)

// The typed readers an agent needs beside the command table: a record by
// reference, the records in a cell range, the Add window's kinds and steps,
// the brushes, the map report, the trigger wizard's items and the map's
// property fields. Each reads through the reader the window itself uses, and
// text in the font's encoding is given as UTF-8.

func editorReaders() []editorReader {
	return []editorReader{
		{name: "record", help: "Read one record by kind and index: its map record and every field with its key, value, choices and whether it is read-only",
			args: []commandArg{req("kind", textArg("Record kind")), req("index", intArg("Record index"))}, read: (*MapEditor).readRecord},
		{name: "records", help: "List the records whose footprint meets the cell range x0,y0..x1,y1, both corners included",
			args: []commandArg{req("x0", intArg("Left cell")), req("y0", intArg("Top cell")), req("x1", intArg("Right cell")), req("y1", intArg("Bottom cell"))}, read: (*MapEditor).readRecords},
		{name: "add_steps", help: "Without kind, the kinds add_record adds; with kind, its steps in order with the host's choices and defaults, given the values so far",
			args: []commandArg{opt("kind", textArg("Record kind")), opt("x", intArg("Cell x")), opt("y", intArg("Cell y")),
				opt("values", map[string]any{"type": "object", "description": "Values so far, as add_record takes them", "additionalProperties": map[string]any{"type": []string{"integer", "string", "array"}}})},
			read: (*MapEditor).readAddSteps},
		{name: "brushes", help: "The paint planes, terrain classes with names and variant counts, object codes with names, and the largest radius", read: (*MapEditor).readBrushes},
		{name: "check", help: "The map report: every error and warning the map's check finds, with its record and cell", read: (*MapEditor).readCheck},
		{name: "trigger_items", help: "The trigger wizard's ready conditions and actions: item key, label, parameters with their kinds and defaults, and the comparison", read: (*MapEditor).readTriggerItems},
		{name: "map_fields", help: "The map property fields of a group, as the Metadata (M) and Pages (X) prompts and the mod package window (W) show them",
			args: []commandArg{req("scope", textArg("Field group", "metadata", "pages", "package"))}, read: (*MapEditor).readMapFields},
		{name: "goals", help: "The script's goals as the Goals (G) report lists them: goal index, arm, text, the record kind a target takes (pick), the script record and the target's cell", read: (*MapEditor).readGoals},
		{name: "messages", help: "Without page, the map's message pages and their text; with page, its parts in order (source, number, body, speaker) and the speakers a part takes",
			args: []commandArg{opt("page", intArg("Message page"))}, read: (*MapEditor).readMessages},
		{name: "unit_settings", help: "The unit settings window's preview: every field of a unit as it would read with the given number fields set; nothing is written",
			args: []commandArg{req("index", intArg("Unit index")), opt("fields", fieldsArg())}, read: (*MapEditor).readUnitSettings},
	}
}

func (e *MapEditor) readGoals(raw json.RawMessage) (any, error) {
	if _, err := decodeArgs[struct{}](raw, nil); err != nil {
		return nil, err
	}
	s, ok := e.script()
	if !ok {
		return nil, fmt.Errorf("this map has no script goals")
	}
	out := []map[string]any{}
	for i, g := range s.Goals() {
		row := map[string]any{"goal": i, "arm": g.Arm, "text": e.mcpDisplay(e.goalText(g)), "pick": g.Pick}
		if g.Kind != "" {
			row["record"] = RecordRef{Kind: g.Kind, Index: g.Index}
		}
		if g.HasCell {
			row["cell"] = []int{g.Cell.X, g.Cell.Y}
		}
		out = append(out, row)
	}
	return out, nil
}

func (e *MapEditor) readMessages(raw json.RawMessage) (any, error) {
	a, err := decodeArgs[struct{ Page *int64 }](raw, editorCommandArgs("messages"))
	if err != nil {
		return nil, err
	}
	if !e.editable() || e.doc.Script == nil {
		return nil, fmt.Errorf("this map has no message pages")
	}
	host, ok := e.doc.Edits.(InspectionMessages)
	if !ok {
		return nil, fmt.Errorf("this map has no message pages")
	}
	if a.Page == nil {
		out := []map[string]any{}
		for _, t := range e.doc.Script.Targets[ScriptMessageParameter] {
			if !t.NewMessage {
				out = append(out, map[string]any{"page": t.Value, "text": e.mcpDisplay(t.Text)})
			}
		}
		return out, nil
	}
	var parts []InspectionMessagePart
	if *a.Page > 0 && *a.Page < int64(NewMessagePage) {
		parts = host.MessageParts(uint32(*a.Page))
	}
	if len(parts) == 0 {
		return nil, fmt.Errorf("the map has no message page %d", *a.Page)
	}
	rows := []map[string]any{}
	var speakers []InspectionChoice
	for _, p := range parts {
		rows = append(rows, map[string]any{"source": p.Source, "number": p.Number, "body": p.Body, "speaker": p.Speaker})
		if f := host.MessageFields(uint32(*a.Page), p.Source); speakers == nil && len(f) == 2 {
			speakers = e.plainChoices(f[1].Choices)
		}
	}
	return map[string]any{"page": *a.Page, "parts": rows, "speakers": speakers}, nil
}

func (e *MapEditor) readUnitSettings(raw json.RawMessage) (any, error) {
	a, err := decodeArgs[struct {
		Index  int
		Fields []fieldArgs
	}](raw, editorCommandArgs("unit_settings"))
	if err != nil {
		return nil, err
	}
	if !e.editable() {
		return nil, fmt.Errorf("no editable map is open")
	}
	host, ok := e.doc.Edits.(InspectionUnitSettings)
	if !ok {
		return nil, fmt.Errorf("this map has no unit settings window")
	}
	if _, err := e.recordOf("Unit", a.Index); err != nil {
		return nil, err
	}
	o, err := e.objectValues("Unit", a.Index, a.Fields)
	if err != nil {
		return nil, err
	}
	for i := range o.Values {
		if o.Values[i].Text {
			return nil, fmt.Errorf("unit_settings previews number fields")
		}
		o.Values[i].Step = 1
	}
	_, fields, _, err := host.UnitSettings(a.Index, o.Values)
	if err != nil {
		return nil, err
	}
	return map[string]any{"unit": a.Index, "fields": e.plainFields(fields)}, nil
}

// plainRecord is r with its text as UTF-8 and without its picture.
func (e *MapEditor) plainRecord(r InspectionRecord) InspectionRecord {
	r.Preview = nil
	r.Label = e.mcpDisplay(r.Label)
	r.Hover = e.mcpDisplay(r.Hover)
	r.Warning = e.mcpDisplay(r.Warning)
	r.Details = e.mcpLines(r.Details)
	r.Lines = append([]InspectionLine(nil), r.Lines...)
	for i := range r.Lines {
		r.Lines[i].Text = e.mcpDisplay(r.Lines[i].Text)
	}
	return r
}

// plainFields are fields with their text as UTF-8 and without their icons.
func (e *MapEditor) plainFields(fields []InspectionField) []InspectionField {
	fields = append([]InspectionField(nil), fields...)
	for i := range fields {
		fields[i].Icons = nil
		fields[i].Label = e.mcpDisplay(fields[i].Label)
		fields[i].Meaning = e.mcpDisplay(fields[i].Meaning)
		fields[i].Section = e.mcpDisplay(fields[i].Section)
		fields[i].Group = e.mcpDisplay(fields[i].Group)
		fields[i].Inherited = e.mcpDisplay(fields[i].Inherited)
		fields[i].Lines = append([]InspectionLine(nil), fields[i].Lines...)
		for j := range fields[i].Lines {
			fields[i].Lines[j].Text = e.mcpDisplay(fields[i].Lines[j].Text)
		}
		fields[i].Choices = e.plainChoices(fields[i].Choices)
	}
	return fields
}

func (e *MapEditor) plainChoices(choices []InspectionChoice) []InspectionChoice {
	out := append([]InspectionChoice(nil), choices...)
	for j := range out {
		out[j].Label = e.mcpDisplay(out[j].Label)
	}
	return out
}

func (e *MapEditor) readRecord(raw json.RawMessage) (any, error) {
	a, err := decodeArgs[recordArgs](raw, editorCommandArgs("record"))
	if err != nil {
		return nil, err
	}
	r, err := e.recordOf(a.Kind, a.Index)
	if err != nil {
		return nil, err
	}
	var fields []InspectionField
	if e.editable() {
		fields = e.plainFields(e.doc.Edits.Fields(a.Kind, a.Index))
	}
	return map[string]any{"record": e.plainRecord(r), "fields": fields}, nil
}

func (e *MapEditor) readRecords(raw json.RawMessage) (any, error) {
	a, err := decodeArgs[struct{ X0, Y0, X1, Y1 int }](raw, editorCommandArgs("records"))
	if err != nil {
		return nil, err
	}
	if e.doc == nil {
		return nil, fmt.Errorf("no map is open")
	}
	box := image.Rect(a.X0, a.Y0, a.X1+1, a.Y1+1)
	if a.X1 < a.X0 || a.Y1 < a.Y0 {
		return nil, fmt.Errorf("the range needs x0 <= x1 and y0 <= y1")
	}
	out := []InspectionRecord{}
	for _, r := range e.doc.Records {
		if r.Spatial && r.footprint().Overlaps(box) {
			out = append(out, e.plainRecord(r))
		}
	}
	return out, nil
}

type addStepRead struct {
	Key      string             `json:"key"`
	Label    string             `json:"label"`
	Control  string             `json:"control"`
	Optional bool               `json:"optional"`
	Asked    bool               `json:"asked"`
	Choices  []InspectionChoice `json:"choices"`
	Default  any                `json:"default,omitempty"`
	Refused  string             `json:"refused,omitempty"`
}

var addControlNames = map[addControl]string{addChoice: "choice", addSprites: "choice", addNumber: "number", addName: "text", addCell: "cell", addMany: "many", addKind: "kind"}

func (e *MapEditor) readAddSteps(raw json.RawMessage) (any, error) {
	a, err := decodeArgs[struct {
		Kind   string
		X, Y   *int
		Values map[string]json.RawMessage
	}](raw, editorCommandArgs("add_steps"))
	if err != nil {
		return nil, err
	}
	if a.Kind == "" {
		kinds := []map[string]string{}
		for _, k := range addCommandKinds() {
			row, _ := addKindOf(k)
			kinds = append(kinds, map[string]string{"kind": k, "label": e.mcpDisplay(e.word(row.Word))})
		}
		return map[string]any{"kinds": kinds}, nil
	}
	if a.Kind == "Message" {
		return map[string]any{"kind": a.Kind, "steps": []addStepRead{}, "parts": "a Message has no steps: add_record takes its parts, as write_message does"}, nil
	}
	var steps []addStepRead
	at := map[string]int{}
	_, err = e.newRecordOf(a.Kind, a.X, a.Y, a.Values, func(st addStep, offer InspectionNewStep, skipped bool, refused error) {
		s := addStepRead{Key: st.Key, Label: e.mcpDisplay(e.word(st.Word)), Control: addControlNames[st.Control], Optional: st.Optional, Asked: !skipped, Choices: e.plainChoices(offer.Choices)}
		switch st.Control {
		case addName:
			s.Default = offer.Text
		case addChoice, addSprites, addNumber:
			s.Default = offer.Value
		}
		if refused != nil {
			s.Refused = e.mcpDisplay(refused.Error())
		}
		if i, ok := at[st.Key]; ok {
			if s.Refused == "" {
				s.Refused = steps[i].Refused
			}
			steps[i] = s
			return
		}
		at[st.Key] = len(steps)
		steps = append(steps, s)
	})
	if err != nil {
		return nil, err
	}
	return map[string]any{"kind": a.Kind, "steps": steps}, nil
}

func (e *MapEditor) readBrushes(raw json.RawMessage) (any, error) {
	if _, err := decodeArgs[struct{}](raw, nil); err != nil {
		return nil, err
	}
	host := e.brushHost()
	if host == nil {
		return nil, fmt.Errorf("this map has no brushes")
	}
	fill, _ := e.doc.Edits.(InspectionBrushFill)
	terrains := []map[string]any{}
	for _, t := range host.BrushTerrains() {
		variants := 1
		if fill != nil {
			variants = fill.BrushVariants(t.Class)
		}
		terrains = append(terrains, map[string]any{"class": t.Class, "name": e.mcpDisplay(t.Name), "variants": variants})
	}
	objects := []map[string]any{}
	for _, o := range host.BrushObjects() {
		objects = append(objects, map[string]any{"code": o.Code, "name": e.mcpDisplay(o.Name)})
	}
	return map[string]any{"planes": []string{"terrain", "height", "objects", "passability"}, "terrains": terrains, "objects": objects,
		"max_radius": host.BrushMaxRadius(), "max_height_change": brushMaxStrength}, nil
}

func (e *MapEditor) readCheck(raw json.RawMessage) (any, error) {
	if _, err := decodeArgs[struct{}](raw, nil); err != nil {
		return nil, err
	}
	c, ok := e.checker()
	if !ok {
		return nil, fmt.Errorf("this map has no check")
	}
	out := []map[string]any{}
	for _, f := range c.Report() {
		row := map[string]any{"error": f.Error, "rule": f.Rule, "text": e.mcpDisplay(f.Text)}
		if f.Kind != "" {
			row["record"] = RecordRef{Kind: f.Kind, Index: f.Index}
		}
		if f.HasCell {
			row["cell"] = []int{f.Cell.X, f.Cell.Y}
		}
		out = append(out, row)
	}
	return out, nil
}

var wizParamNames = map[wizParamKind]string{wpUnit: "unit", wpPlayer: "player", wpGroup: "group", wpNumber: "number", wpCell: "cell", wpBox: "area", wpMessage: "message", wpCommand: "command", wpRelation: "relation"}

func (e *MapEditor) readTriggerItems(raw json.RawMessage) (any, error) {
	if _, err := decodeArgs[struct{}](raw, nil); err != nil {
		return nil, err
	}
	if e.doc == nil {
		return nil, fmt.Errorf("no map is open")
	}
	list := func(condition bool) []map[string]any {
		out := []map[string]any{}
		for i, it := range wizItems(condition) {
			defaults := e.wizardDefaults(condition, i)
			params := []map[string]any{}
			for k, p := range it.params {
				params = append(params, map[string]any{"kind": wizParamNames[p.kind], "label": e.mcpDisplay(e.word(p.labelWord)), "default": defaults[k+1]})
			}
			row := map[string]any{"item": wizItemKey(it), "label": e.mcpDisplay(e.word(it.labelWord)), "params": params}
			if condition && it.cmp < 0 {
				row["comparison"] = "chosen, 0..5"
			} else if condition {
				row["comparison"] = it.cmp
			}
			if it.onePerTrigger {
				row["one_per_trigger"] = true
			}
			out = append(out, row)
		}
		return out
	}
	return map[string]any{"conditions": list(true), "actions": list(false)}, nil
}

func (e *MapEditor) readMapFields(raw json.RawMessage) (any, error) {
	a, err := decodeArgs[struct{ Scope string }](raw, editorCommandArgs("map_fields"))
	if err != nil {
		return nil, err
	}
	if a.Scope != "metadata" && a.Scope != "pages" && a.Scope != "package" {
		return nil, fmt.Errorf("scope must be metadata, pages or package")
	}
	host, ok := e.workspace()
	if !ok {
		return nil, fmt.Errorf("this map has no property fields")
	}
	return e.plainFields(host.WorkspaceFields(a.Scope)), nil
}

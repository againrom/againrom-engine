package ui

import (
	"encoding/json"
	"image"
	"reflect"
	"strconv"
	"strings"
	"testing"

	"againrom/pkg/editormcp"
)

func TestEditorCommandToolsComeFromTheTable(t *testing.T) {
	byName := map[string]editorTool{}
	for _, r := range editorTools() {
		if _, twice := byName[r.Name]; twice {
			t.Fatalf("tool %s listed twice", r.Name)
		}
		byName[r.Name] = r
	}
	check := func(name, help string, args []commandArg) {
		t.Helper()
		r, ok := byName[name]
		if !ok || r.typed == nil || r.Description != help || !reflect.DeepEqual(r.InputSchema, commandSchema(args)) {
			t.Fatalf("tool %s is not its table row", name)
		}
		props := r.InputSchema["properties"].(map[string]any)
		if len(props) != len(args) || r.InputSchema["additionalProperties"] != false {
			t.Fatalf("tool %s schema %v", name, r.InputSchema)
		}
		for _, a := range args {
			if props[a.name] == nil {
				t.Fatalf("tool %s lacks argument %s", name, a.name)
			}
		}
	}
	for _, c := range editorCommandRows() {
		check(c.name, c.help, c.args)
	}
	for _, r := range editorReaders() {
		check(r.name, r.help, r.args)
	}
}

// unit_settings answers the unit settings window's preview of draft values
// and writes nothing; a text value and a missing unit are refused.
func TestEditorUnitSettingsToolPreviewsWithoutWriting(t *testing.T) {
	e, host := popupEditor(t)
	_ = e.Open("popups")
	e.doc.Edits = &settingsPreviewProbe{popupHost: host, readOnly: true}
	p := newMCPPipe(t, e)
	raw := string(e.doc.SourceBytes)
	var out struct {
		Unit   int
		Fields []InspectionField
	}
	if err := json.Unmarshal([]byte(p.tool(t, "unit_settings", map[string]any{"index": 0, "fields": []any{map[string]any{"key": "a", "value": 42}}}).Content[0].Text), &out); err != nil {
		t.Fatal(err)
	}
	if out.Unit != 0 || len(out.Fields) == 0 || len(out.Fields[len(out.Fields)-1].Lines) == 0 || out.Fields[len(out.Fields)-1].Lines[0].Text != "Carrier 42" {
		t.Fatalf("unit_settings answers %+v", out)
	}
	for _, bad := range []map[string]any{{"index": 99}, {"index": 0, "fields": []any{map[string]any{"key": "a", "text": "x"}}}} {
		b := p.request(t, "tools/call", map[string]any{"name": "unit_settings", "arguments": bad})
		var r editormcp.Result
		if json.Unmarshal(b, &r) != nil || !r.IsError {
			t.Fatalf("unit_settings %v accepted: %s", bad, b)
		}
	}
	if string(e.doc.SourceBytes) != raw || len(host.calls) != 0 || e.Dirty() {
		t.Fatal("unit_settings wrote the map")
	}
}

func TestEditorCommandToolsRefuseBadArgumentsAndKeepTheMap(t *testing.T) {
	e := newMapEditorFixture(t)
	e.Layout(800, 600)
	probe := &mcpWitness{editorMoveProbe: editorMoveProbe{doc: e.doc, index: -1}, before: e.doc}
	e.doc.Edits = probe
	p := newMCPPipe(t, e)
	doc := e.doc
	unit := func(extra map[string]any) map[string]any {
		a := map[string]any{"kind": "Unit", "index": 3, "x": 10, "y": 30}
		for k, v := range extra {
			if v == "drop" {
				delete(a, k)
			} else {
				a[k] = v
			}
		}
		return a
	}
	for _, tc := range []struct {
		name string
		args any
	}{
		{"move_record", map[string]any{}}, {"move_record", unit(map[string]any{"y": "drop"})}, {"move_record", unit(map[string]any{"typo": 1})},
		{"move_record", unit(map[string]any{"x": nil})}, {"move_record", unit(map[string]any{"x": "one"})}, {"move_record", unit(map[string]any{"x": 50})},
		{"move_record", unit(map[string]any{"index": 99})}, {"paint", map[string]any{"plane": "lava", "cells": [][]int{{1, 1}}}},
		{"new_map", map[string]any{"width": 0, "height": 32}}, {"undo", map[string]any{"steps": 1}}, {"randomize", map[string]any{"plane": "terrain", "seed": -1}},
		{"set_map_field", map[string]any{"scope": "package", "key": "folder", "value": "x"}}, {"record", map[string]any{"kind": "Unit", "index": 99}},
		{"move_records", map[string]any{"records": []any{}, "dx": 1, "dy": 0}}, {"set_units_group", map[string]any{"units": []int{3}, "group": -2}},
		{"write_message", map[string]any{"parts": []any{map[string]any{"body": "x"}}}}, {"add_box_area", map[string]any{"area": []int{3, 3, 1, 1}}},
	} {
		var result editormcp.Result
		b := p.request(t, "tools/call", map[string]any{"name": tc.name, "arguments": tc.args})
		if json.Unmarshal(b, &result) != nil || !result.IsError {
			t.Fatalf("%s %v accepted: %s", tc.name, tc.args, b)
		}
		if e.doc != doc || probe.index != -1 || e.Dirty() {
			t.Fatalf("%s %v changed the map", tc.name, tc.args)
		}
	}
	var report struct {
		Changed bool       `json:"changed"`
		Record  *RecordRef `json:"record"`
		Dirty   bool       `json:"dirty"`
	}
	if err := json.Unmarshal([]byte(p.tool(t, "move_record", unit(nil)).Content[0].Text), &report); err != nil {
		t.Fatal(err)
	}
	if !report.Changed || report.Record == nil || *report.Record != (RecordRef{Kind: "Unit", Index: 3}) || !report.Dirty {
		t.Fatalf("move report %+v", report)
	}
	if probe.index != 3 || e.Document().Records[3].Cell != image.Pt(10, 30) {
		t.Fatal("move_record did not move the unit")
	}
	e.pathFallback(0, nil)
	var result editormcp.Result
	b := p.request(t, "tools/call", map[string]any{"name": "move_record", "arguments": unit(map[string]any{"x": 11})})
	if json.Unmarshal(b, &result) != nil || !result.IsError || !strings.Contains(string(b), "prompt") || e.Document().Records[3].Cell != image.Pt(10, 30) {
		t.Fatalf("a command ran through the open path prompt: %s", b)
	}
}

// walkHost offers a tier only to class 2, one cast kind and no range, so
// the walk asks some optional steps and leaves others out, and keeps each
// record it is asked to add.
type walkHost struct {
	*modesFull
	added []InspectionNewRecord
}

func (h *walkHost) NewRecordStep(r InspectionNewRecord, key string) InspectionNewStep {
	switch key {
	case "tier":
		for _, v := range r.Values {
			if v.Key == "class" && v.Value == 2 {
				return InspectionNewStep{Choices: []InspectionChoice{{Value: 1, Label: "Tier 1"}, {Value: 2, Label: "Tier 2"}}, Value: 1}
			}
		}
		return InspectionNewStep{}
	case "castkind":
		return InspectionNewStep{Choices: []InspectionChoice{{Value: 4, Label: "Only"}}, Value: 4}
	case "range":
		return InspectionNewStep{Absent: true}
	case "link", "power":
		return InspectionNewStep{Value: 3}
	}
	return h.modesFull.NewRecordStep(r, key)
}

func (h *walkHost) AddRecord(r InspectionNewRecord) (*InspectionDocument, error) {
	h.added = append(h.added, r)
	return h.modesFull.AddRecord(r)
}

func newWalkEditor(t *testing.T) (*MapEditor, *walkHost) {
	t.Helper()
	e, _ := newModesEditor(t, "full")
	host := &walkHost{modesFull: e.doc.Edits.(*modesFull)}
	host.self, e.doc.Edits = host, host
	return e, host
}

// The Add window and add_record build one record from the same answers for
// every kind add_record adds: the window asked step by step with Enter, a
// many step value by value, a script kind through the Script kind step.
func TestEditorAddRecordCommandBuildsTheAddWindowsRecord(t *testing.T) {
	for _, c := range []struct {
		kind   string
		values map[string]any
	}{
		{"Unit", nil}, {"Unit", map[string]any{"class": 2, "tier": 2, "owner": 2}},
		{"Structure", nil}, {"Structure", map[string]any{"kind": 2, "owner": 2}},
		{"Sack", nil}, {"Sack", map[string]any{"item": []int{2, 1}}},
		{"Player", nil}, {"Player", map[string]any{"name": "Rivals", "colour": 4, "control": 2}},
		{"Trigger", nil}, {"Trigger", map[string]any{"name": "Gate", "once": 2}},
		{"Enchantment", nil}, {"Enchantment", map[string]any{"template": 2, "link": 7, "spell": 2, "power": 9}},
		{"Action", nil}, {"Condition", nil},
	} {
		w, wh := newWalkEditor(t)
		if c.kind == "Action" || c.kind == "Condition" {
			w.openAdd("Script")
			w.prompt.path = map[string]string{"Action": "0", "Condition": "1"}[c.kind]
			_ = w.Key("enter")
		} else {
			w.openAdd(c.kind)
		}
		onCell := false
		for n := 0; n < 16; n++ {
			r, ok := w.addRunning()
			if !ok {
				break
			}
			st := r.steps[r.at]
			v, given := c.values[st.Key]
			switch {
			case st.Control == addCell:
				onCell, w.prompt.path = true, "25,20"
			case st.Control == addMany && given:
				for _, item := range v.([]int) {
					w.prompt.path = strconv.Itoa(item)
					_ = w.Key("enter")
				}
				w.prompt.path = ""
			case st.Control == addMany:
				w.addTake(0)
				_ = w.Key("enter")
			case given && st.Control == addName:
				w.prompt.path = v.(string)
			case given:
				w.prompt.path = strconv.Itoa(v.(int))
			}
			_ = w.Key("enter")
			if r.note != "" {
				t.Fatalf("%s %v: the window refused step %s: %s", c.kind, c.values, st.Key, r.note)
			}
		}
		if len(wh.added) != 1 {
			t.Fatalf("%s %v: the window added %v", c.kind, c.values, wh.added)
		}
		m, _ := newWalkEditor(t)
		values := map[string]json.RawMessage{}
		for k, v := range c.values {
			b, _ := json.Marshal(v)
			values[k] = b
		}
		if c.kind == "Sack" && c.values == nil {
			values["item"] = json.RawMessage("1")
		}
		var x, y *int
		if onCell {
			x, y = new(int), new(int)
			*x, *y = 25, 20
		}
		got, err := m.newRecordOf(c.kind, x, y, values, nil)
		if err != nil || !reflect.DeepEqual(got, wh.added[0]) {
			t.Fatalf("%s %v: add_record builds %+v (%v), the Add window %+v", c.kind, c.values, got, err, wh.added[0])
		}
	}
}

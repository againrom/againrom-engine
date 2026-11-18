package ui

import (
	"encoding/json"
	"fmt"
	"image"
	"reflect"
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

// walkHost offers one cast kind and no range, so the walk leaves those
// optional steps out, and keeps each record it is asked to add.
type walkHost struct {
	*modesFull
	added []InspectionNewRecord
}

func (h *walkHost) NewRecordStep(r InspectionNewRecord, key string) InspectionNewStep {
	switch key {
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

func TestEditorAddRecordAndAddStepsUseTheRecordBuilder(t *testing.T) {
	e, host := newWalkEditor(t)
	pipe := newMCPPipe(t, e)
	steps := pipe.tool(t, "add_steps", map[string]any{"kind": "Sack"})
	if steps.IsError || !strings.Contains(fmt.Sprint(steps), "item") {
		t.Fatalf("add_steps: %+v", steps)
	}
	res := pipe.tool(t, "add_record", map[string]any{
		"kind": "Sack", "x": 25, "y": 20, "values": map[string]any{"item": []int{2, 1}},
	})
	want := InspectionNewRecord{Kind: KindSack, HasCell: true, Cell: image.Pt(25, 20),
		Values: []InspectionFieldValue{{Key: "item", Value: 2}, {Key: "item", Value: 1}}}
	if res.IsError || len(host.added) != 1 || !reflect.DeepEqual(host.added[0], want) {
		t.Fatalf("add_record: %+v, wrote %+v", res, host.added)
	}
}

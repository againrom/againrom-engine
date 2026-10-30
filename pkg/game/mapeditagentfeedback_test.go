package game

import (
	"bytes"
	"encoding/json"
	"image"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"againrom/internal/synth"
	"againrom/pkg/editormcp"
	"againrom/pkg/mapedit"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

// agentEditor is an editor over the overlap tables, whose unit rows the map
// check reads up to slot 37, laid out for the typed tools.
func agentEditor(t *testing.T) *ui.MapEditor {
	t.Helper()
	x := overlapInspector()
	units := x.table.Units.(dbCollection)
	for i, row := range units {
		if len(row.params) > 0 {
			units[i].params = append(row.params, make([]int32, 0x26-len(row.params))...)
		}
	}
	e := x.Editor()
	e.Layout(1280, 800)
	return e
}

// checkRows are the map check's findings as the check tool gives them.
func checkRows(t *testing.T, r editormcp.Result) []struct {
	Error  bool
	Rule   string
	Text   string
	Record *ui.RecordRef
} {
	t.Helper()
	var rows []struct {
		Error  bool
		Rule   string
		Text   string
		Record *ui.RecordRef
	}
	if err := json.Unmarshal([]byte(r.Content[0].Text), &rows); err != nil {
		t.Fatalf("check gives %s: %v", r.Content[0].Text, err)
	}
	return rows
}

// An agent builds the village of docs/MAPEDIT-MCP.md: a hut and units, the
// second of them the chief. It reads the chief's script ID from add_record
// and from record, finds it among the unit choices trigger_items lists, and
// writes a victory on that unit's death; the script names that placement. A
// unit value no placement holds is refused, and one planted in the file is
// named by the map check. Each unit added without a group takes the default
// group of its own; a group given in the call wins.
func TestMapEditorMCPVictoryTriggerNamesTheUnitAddRecordReports(t *testing.T) {
	e := agentEditor(t)
	tool, must := editorMCP(t, e)
	if r := commandReport(t, must("new_map", map[string]any{"width": 32, "height": 32})); r["playable"] != "8,8..23,23" {
		t.Fatalf("new_map reports %v", r)
	}
	if text := must("map", map[string]any{"x": 16, "y": 16}).Content[0].Text; !strings.Contains(text, `"playable":"8,8..23,23"`) {
		t.Fatalf("map gives %s", text)
	}
	if hut := commandReport(t, must("add_record", map[string]any{"kind": "Structure", "x": 16, "y": 9, "values": map[string]any{"kind": 5}})); hut["script_id"] == nil {
		t.Fatalf("add_record of a structure reports %v", hut)
	}
	first := commandReport(t, must("add_record", map[string]any{"kind": "Unit", "x": 10, "y": 10, "values": map[string]any{"class": 40}}))
	second := commandReport(t, must("add_record", map[string]any{"kind": "Unit", "x": 14, "y": 14, "values": map[string]any{"class": 40}}))
	id, ok := second["script_id"].(float64)
	if !ok || id <= 0 || id == first["script_id"] {
		t.Fatalf("add_record reports script ids %v and %v", first["script_id"], second["script_id"])
	}
	var read struct {
		Record struct{ ScriptID int }
		Fields []ui.InspectionField
	}
	if err := json.Unmarshal([]byte(must("record", map[string]any{"kind": "Unit", "index": 1}).Content[0].Text), &read); err != nil || read.Record.ScriptID != int(id) {
		t.Fatalf("record gives script id %d, add_record %v: %v", read.Record.ScriptID, id, err)
	}
	readonly := false
	for _, f := range read.Fields {
		readonly = readonly || f.Key == "script_id" && f.ReadOnly && f.Value == int64(id)
	}
	if !readonly {
		t.Fatal("record does not expose the script ID as a read-only field")
	}
	if r := tool("set_field", map[string]any{"kind": "Unit", "index": 1, "key": "script_id", "value": 77}); !r.IsError {
		t.Fatal("set_field wrote the script ID")
	}
	must("add_record", map[string]any{"kind": "Unit", "x": 18, "y": 18, "values": map[string]any{"class": 40, "group": 1}})
	if m := mustMap(t, e.Document().SourceBytes); len(m.Units) != 3 || m.Units[0].GroupID != 1 || m.Units[1].GroupID != 2 || m.Units[2].GroupID != 1 {
		t.Fatalf("the units' groups %+v", m.Units)
	}

	var items struct {
		Conditions []struct {
			Item   string
			Params []struct {
				Kind    string
				Choices []ui.InspectionChoice
			}
		}
	}
	if err := json.Unmarshal([]byte(must("trigger_items", map[string]any{}).Content[0].Text), &items); err != nil {
		t.Fatal(err)
	}
	listed, heroTop, group := false, false, false
	for _, it := range items.Conditions {
		for _, p := range it.Params {
			for _, c := range p.Choices {
				listed = listed || it.Item == "c.unitdead" && c.Value == int64(id)
				heroTop = heroTop || it.Item == "c.unitdead" && c.Value == 11000
				group = group || p.Kind == "group" && c.Value == 2
			}
		}
	}
	if !listed || !heroTop || !group {
		t.Fatalf("trigger_items does not list unit %v for c.unitdead: %+v", id, items.Conditions)
	}
	victory := func(unit int64) map[string]any {
		return map[string]any{"name": "Chief",
			"conditions": []any{map[string]any{"item": "c.unitdead", "params": []any{unit}}},
			"actions":    []any{map[string]any{"item": "a.win"}}}
	}
	before := bytes.Clone(e.Document().SourceBytes)
	if r := tool("write_trigger", victory(20001)); !r.IsError || !strings.Contains(r.Content[0].Text, "trigger_items") || !bytes.Equal(before, e.Document().SourceBytes) {
		t.Fatalf("write_trigger took unit 20001: %+v", r)
	}
	if r := tool("write_trigger", map[string]any{"name": "Missing group", "conditions": []any{map[string]any{"item": "c.groupcount", "params": []any{99, 0}, "comparison": 0}}, "actions": []any{map[string]any{"item": "a.win"}}}); !r.IsError || !bytes.Equal(before, e.Document().SourceBytes) {
		t.Fatalf("write_trigger took group 99: %+v", r)
	}
	must("write_trigger", victory(int64(id)))
	compiledMap := mustMap(t, e.Document().SourceBytes)
	compiled, report, err := mapload.CompileScript(compiledMap, mapload.ScriptRefs{Units: mapload.ScriptUnits(compiledMap, nil)})
	if err != nil || len(report.Unresolved) != 0 {
		t.Fatalf("compile the victory: %+v, %v", report, err)
	}
	bound := false
	for _, c := range compiled.Checks() {
		bound = bound || c.Op == sim.ScriptCheckAlive && c.HasUnit && c.Unit == 1
	}
	if !bound || compiledMap.Units[1].X>>8 != 14 || compiledMap.Units[1].Y>>8 != 14 {
		t.Fatal("the compiled death condition does not bind the second placement on 14,14")
	}
	named := false
	for _, r := range e.Document().Records {
		for _, ref := range r.References {
			for _, target := range ref.Targets {
				named = named || r.Kind == "Condition" && target.Cell == image.Pt(14, 14)
			}
		}
	}
	if !named {
		t.Fatal("no condition of the script names the unit on 14,14")
	}
	for _, f := range checkRows(t, must("check", map[string]any{})) {
		if f.Rule == "script-reference" {
			t.Fatalf("the check finds %+v", f)
		}
	}

	// A unit value above the hero band, planted in the saved file.
	m, err := mapedit.New(e.Document().SourceBytes)
	if err != nil {
		t.Fatal(err)
	}
	conditions := mustMap(t, m.Bytes())
	sc, err := conditions.Script()
	if err != nil {
		t.Fatal(err)
	}
	at := -1
	for i, n := range sc.Conditions {
		if n.Opcode == 5 {
			at = i
		}
	}
	if err := m.SetScriptNode(mapedit.KindCondition, at, "", 5, mapedit.ScriptParam{Type: 4, Value: 12345}); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "planted.alm")
	if err := os.WriteFile(path, m.Bytes(), 0600); err != nil {
		t.Fatal(err)
	}
	must("paint", map[string]any{"plane": "height", "value": 4, "radius": 1, "cells": [][]int{{12, 12}}})
	must("open", map[string]any{"path": path, "discard": true})
	if e.Dirty() || !bytes.Equal(e.Document().SourceBytes, m.Bytes()) {
		t.Fatal("open with discard left the unsaved map")
	}
	found := false
	for _, f := range checkRows(t, must("check", map[string]any{})) {
		found = found || f.Rule == "script-reference" && f.Error && strings.Contains(f.Text, "12345")
	}
	if !found {
		t.Fatal("the check does not name unit 12345")
	}
	must("paint", map[string]any{"plane": "height", "value": 2, "cells": [][]int{{12, 12}}})
	must("new_map", map[string]any{"width": 64, "height": 64, "discard": true})
	if e.Dirty() || e.Document().Width != 64 || len(mustMap(t, e.Document().SourceBytes).Units) != 0 {
		t.Fatal("new_map with discard kept the previous map")
	}
}

// On a 64x64 map the playable area is 8,8..55,55: a unit, a structure and a
// drop zone on 4,4 are refused with that area in the refusal, as a move into
// the ring is, while a height and a passability stroke take the cell. A
// paint names its cells only when asked.
func TestMapEditorMCPRefusesTheBorderRingForPlacements(t *testing.T) {
	e := agentEditor(t)
	tool, must := editorMCP(t, e)
	if r := commandReport(t, must("new_map", map[string]any{"width": 64, "height": 64})); r["playable"] != "8,8..55,55" {
		t.Fatalf("new_map reports %v", r)
	}
	before := bytes.Clone(e.Document().SourceBytes)
	for _, c := range []commandCall{
		{"add_record", map[string]any{"kind": "Unit", "x": 4, "y": 4, "values": map[string]any{"class": 40}}},
		{"add_record", map[string]any{"kind": "Structure", "x": 4, "y": 4, "values": map[string]any{"kind": 5}}},
		{"add_drop_zone", map[string]any{"x": 4, "y": 4}},
	} {
		r := tool(c.name, c.args)
		if !r.IsError || !strings.Contains(r.Content[0].Text, "border ring") || !strings.Contains(r.Content[0].Text, "8,8..55,55") {
			t.Fatalf("%s %v: %+v", c.name, c.args, r)
		}
		if !bytes.Equal(before, e.Document().SourceBytes) {
			t.Fatalf("%s %v changed the map", c.name, c.args)
		}
	}
	must("add_record", map[string]any{"kind": "Unit", "x": 20, "y": 20, "values": map[string]any{"class": 40}})
	if r := tool("move_record", map[string]any{"kind": "Unit", "index": 0, "x": 60, "y": 20}); !r.IsError || !strings.Contains(r.Content[0].Text, "8,8..55,55") {
		t.Fatalf("move_record into the ring: %+v", r)
	}
	r := commandReport(t, must("paint", map[string]any{"plane": "height", "value": 6, "radius": 1, "cells": [][]int{{4, 4}}}))
	if r["changed"] != true || r["cells"] != nil || r["cells_box"] == nil || r["cells_total"] == float64(0) {
		t.Fatalf("a height stroke on the ring reports %v", r)
	}
	r = commandReport(t, must("paint", map[string]any{"plane": "passability", "value": 1, "radius": 1, "cells": [][]int{{5, 5}}, "verbose": true}))
	if cells, _ := r["cells"].([]any); r["changed"] != true || len(cells) == 0 || float64(len(cells)) != r["cells_total"] {
		t.Fatalf("a verbose passability stroke on the ring reports %v", r)
	}
	if bad := tool("paint", map[string]any{"plane": "height", "value": 2, "cells": [][]int{{4, 4}}, "verbose": nil}); !bad.IsError {
		t.Fatal("paint accepted a null verbose flag")
	}
	r = commandReport(t, must("randomize", map[string]any{"plane": "heights", "seed": 17}))
	if r["changed"] != true || r["cells"] != nil || r["cells_box"] == nil {
		t.Fatalf("randomize without verbose reports %v", r)
	}
}

// A structure planted in the ring is named by the map check, and an edit of
// it carries that finding in its warnings; an edit of a structure inside the
// playable area carries none.
func TestMapEditorMCPWarnsOfAStructureInTheBorderRing(t *testing.T) {
	raw := synth.ALM(synth.ALMOptions{Width: 64, Height: 64, Type7Payload: make([]byte, 12), Objects: []synth.ALMObject{
		{X: 3*256 + 128, Y: 3*256 + 128, Kind: 5, Field12: 7},
		{X: 20*256 + 128, Y: 20*256 + 128, Kind: 5, Field12: 8},
	}})
	path := filepath.Join(t.TempDir(), "ring.alm")
	if err := os.WriteFile(path, raw, 0600); err != nil {
		t.Fatal(err)
	}
	e := agentEditor(t)
	_, must := editorMCP(t, e)
	must("open", map[string]any{"path": path})
	var ring []string
	for _, f := range checkRows(t, must("check", map[string]any{})) {
		if f.Rule == "structure-border" {
			if f.Record == nil || *f.Record != (ui.RecordRef{Kind: "Structure", Index: 0}) {
				t.Fatalf("the border finding names %+v", f.Record)
			}
			ring = append(ring, f.Text)
		}
	}
	if len(ring) != 1 || !strings.Contains(ring[0], "8,8..55,55") {
		t.Fatalf("the check's border findings %q", ring)
	}
	// The fixture's structures have no owner in the roster, which each
	// edit's warnings also name; the border finding is structure 0's alone.
	warned := func(index int) bool {
		t.Helper()
		r := commandReport(t, must("set_field", map[string]any{"kind": "Structure", "index": index, "key": "kind", "value": 1}))
		if r["changed"] != true {
			t.Fatalf("set_field on structure %d reports %v", index, r)
		}
		w, _ := r["warnings"].([]any)
		border := false
		for _, f := range w {
			row := f.(map[string]any)
			if rec := row["record"].(map[string]any); rec["Kind"] != "Structure" || rec["Index"] != float64(index) {
				t.Fatalf("the edit of structure %d warns of %v", index, row)
			}
			border = border || row["rule"] == "structure-border"
		}
		return border
	}
	if !warned(0) {
		t.Fatal("the ring structure's edit does not warn of the ring")
	}
	if warned(1) {
		t.Fatal("the playable structure's edit warns of the ring")
	}
}

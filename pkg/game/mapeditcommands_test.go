package game

import (
	"bytes"
	"encoding/json"
	"image"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"testing"

	"againrom/pkg/editormcp"
	"againrom/pkg/formats/alm"
	"againrom/pkg/mapedit"
	"againrom/pkg/render/terrain"
	"againrom/pkg/ui"
)

// commandCall is one typed tool call.
type commandCall struct {
	name string
	args map[string]any
}

// commandReport is the change report of a typed command.
func commandReport(t *testing.T, r editormcp.Result) map[string]any {
	t.Helper()
	var out map[string]any
	if len(r.Content) == 0 || json.Unmarshal([]byte(r.Content[0].Text), &out) != nil {
		t.Fatalf("no change report: %+v", r)
	}
	return out
}

// newMapWindow is an editor over a new w x w map of host x, the Map panel
// closed and cell (w/2, w/2) in view.
func newMapWindow(t *testing.T, x *MapInspector, w int) *ui.MapEditor {
	t.Helper()
	e := x.Editor()
	e.Layout(1280, 800)
	if err := e.Open(NewMapSource(strconv.Itoa(w) + "x" + strconv.Itoa(w))); err != nil {
		t.Fatal(err)
	}
	closeMapPanel(t, e)
	e.ShowCell(image.Pt(w/2, w/2))
	return e
}

// typeSeed answers the open seed prompt with seed.
func typeSeed(t *testing.T, e *ui.MapEditor, key string, seed int) {
	t.Helper()
	if err := e.Key(key); err != nil {
		t.Fatal(err)
	}
	for range 12 {
		_ = e.Key("backspace")
	}
	e.TextInput(strconv.Itoa(seed))
	if err := e.Key("enter"); err != nil {
		t.Fatal(err)
	}
}

// Each window route and the typed command its table row holds give the same
// map bytes from the same map, the same undo steps back to the bytes they
// started from, and the same redo.
func TestMapEditorCommandsMatchTheWindowRoutes(t *testing.T) {
	var must func(name string, args any) editormcp.Result
	recordArgs := func(e *ui.MapEditor, row int, extra map[string]any) map[string]any {
		r := e.Document().Records[row]
		out := map[string]any{"kind": r.Kind, "index": r.Index}
		for k, v := range extra {
			out[k] = v
		}
		return out
	}
	moved := func(t *testing.T) *ui.MapEditor {
		_, e, _ := openTestMapEditor(t, inspectionFixtureBytes(t))
		return e
	}
	authoring := func(t *testing.T) *ui.MapEditor { _, e := openAuthoring(t); return e }
	players := func(t *testing.T) *ui.MapEditor {
		_, e, _ := playerWindow(t)
		if err := e.Key("players"); err != nil {
			t.Fatal(err)
		}
		return e
	}
	goal := func(zones ...image.Point) func(t *testing.T) *ui.MapEditor {
		return func(t *testing.T) *ui.MapEditor {
			e, _ := openGoalMap(t, emptyScriptMap(t, 40))
			e.Fit()
			if len(zones) > 0 {
				_ = e.Key("drop")
				for _, c := range zones {
					clickCell(e, c)
				}
				_ = e.Key("escape")
			}
			return e
		}
	}
	plain := func(t *testing.T) *ui.MapEditor { return newMapWindow(t, &MapInspector{tiles: &terrain.Tileset{}}, 48) }
	built := func(t *testing.T) *ui.MapEditor { return newMapWindow(t, overlapInspector(), 32) }
	// A terrain stroke's default fill in the window draws a fresh seed per
	// stroke, so the stroke cases are the height and passability planes.
	stroke := []image.Point{{X: 23, Y: 24}, {X: 24, Y: 24}, {X: 25, Y: 24}}
	strokeCells := [][]int{{23, 24}, {24, 24}, {25, 24}}
	cases := []struct {
		name   string
		open   func(t *testing.T) *ui.MapEditor
		window func(t *testing.T, e *ui.MapEditor)
		calls  func(t *testing.T, e *ui.MapEditor) []commandCall
	}{
		{"move_record", moved, func(t *testing.T, e *ui.MapEditor) { dragInspectionRecord(e, image.Pt(11, 12), image.Pt(13, 14)) },
			func(t *testing.T, e *ui.MapEditor) []commandCall {
				return []commandCall{{"move_record", recordArgs(e, 0, map[string]any{"x": 13, "y": 14})}}
			}},
		{"add unit", authoring, func(t *testing.T, e *ui.MapEditor) {
			addWindow(t, e, "unit", map[string]string{"class": "40", "cell": "9,10"})
		}, func(t *testing.T, e *ui.MapEditor) []commandCall {
			return []commandCall{{"add_record", map[string]any{"kind": "Unit", "x": 9, "y": 10, "values": map[string]any{"class": 40}}}}
		}},
		{"add structure", built, func(t *testing.T, e *ui.MapEditor) {
			addWindow(t, e, "structure", map[string]string{"kind": "5", "cell": "10,12"})
		}, func(t *testing.T, e *ui.MapEditor) []commandCall {
			return []commandCall{{"add_record", map[string]any{"kind": "Structure", "x": 10, "y": 12, "values": map[string]any{"kind": 5}}}}
		}},
		{"add player", players, func(t *testing.T, e *ui.MapEditor) { addWindow(t, e, "player", nil) },
			func(t *testing.T, e *ui.MapEditor) []commandCall {
				return []commandCall{{"add_record", map[string]any{"kind": "Player"}}}
			}},
		{"set_field text", players, func(t *testing.T, e *ui.MapEditor) {
			if err := enterField(e, 0, "Copper"); err != nil {
				t.Fatal(err)
			}
		}, func(t *testing.T, e *ui.MapEditor) []commandCall {
			key := e.Document().Edits.Fields("Player", 0)[0].Key
			return []commandCall{{"set_field", map[string]any{"kind": "Player", "index": 0, "key": key, "text": "Copper"}}}
		}},
		{"set_field number", players, func(t *testing.T, e *ui.MapEditor) {
			if err := enterField(e, 1, "9"); err != nil {
				t.Fatal(err)
			}
		}, func(t *testing.T, e *ui.MapEditor) []commandCall {
			key := e.Document().Edits.Fields("Player", 0)[1].Key
			return []commandCall{{"set_field", map[string]any{"kind": "Player", "index": 0, "key": key, "value": 9}}}
		}},
		{"delete_record", authoring, func(t *testing.T, e *ui.MapEditor) {
			e.Select(1, true)
			_ = e.Key("delete")
		}, func(t *testing.T, e *ui.MapEditor) []commandCall {
			return []commandCall{{"delete_record", recordArgs(e, 1, nil)}}
		}},
		{"clone_record", authoring, func(t *testing.T, e *ui.MapEditor) {
			e.Select(0, true)
			_ = e.Key("clone")
			clickCell(e, image.Pt(9, 10))
		}, func(t *testing.T, e *ui.MapEditor) []commandCall {
			return []commandCall{{"clone_record", recordArgs(e, 0, map[string]any{"x": 9, "y": 10})}}
		}},
		{"add_drop_zone", goal(), func(t *testing.T, e *ui.MapEditor) {
			_ = e.Key("drop")
			clickCell(e, image.Pt(12, 12))
		}, func(t *testing.T, e *ui.MapEditor) []commandCall {
			return []commandCall{{"add_drop_zone", map[string]any{"x": 12, "y": 12}}}
		}},
		{"move_drop_zone", goal(image.Pt(12, 12)), func(t *testing.T, e *ui.MapEditor) {
			_ = e.Key("drop")
			clickCell(e, image.Pt(12, 12))
			dragCell(e, image.Pt(12, 12), image.Pt(9, 20))
		}, func(t *testing.T, e *ui.MapEditor) []commandCall {
			return []commandCall{{"move_drop_zone", map[string]any{"zone": 0, "x": 9, "y": 20}}}
		}},
		{"delete_drop_zone", goal(image.Pt(12, 12), image.Pt(30, 8)), func(t *testing.T, e *ui.MapEditor) {
			_ = e.Key("drop")
			clickCell(e, image.Pt(30, 8))
			_ = e.Key("delete")
		}, func(t *testing.T, e *ui.MapEditor) []commandCall {
			return []commandCall{{"delete_drop_zone", map[string]any{"zone": 1}}}
		}},
		{"write_trigger", goal(image.Pt(12, 12)), func(t *testing.T, e *ui.MapEditor) {
			wizNewTrigger(t, e, "Treasure")
			wizPress(t, e, "", 0)
			wizPress(t, e, "c0.p1", 0)
			dragCell(e, image.Pt(10, 12), image.Pt(14, 15))
			wizPress(t, e, "+action", 0)
			wizPress(t, e, "+action", 0)
			trigChoose(t, e, wizField(t, e, "a1.item"), "Win the mission")
			wizApply(t, e)
		}, func(t *testing.T, e *ui.MapEditor) []commandCall {
			index := len(e.Document().Script.Triggers)
			hero, err := strconv.Atoi(heroDefault(t, must))
			if err != nil {
				t.Fatal(err)
			}
			return []commandCall{{"add_record", map[string]any{"kind": "Trigger", "values": map[string]any{"name": "Treasure"}}},
				{"write_trigger", map[string]any{"index": index, "name": "Treasure",
					"conditions": []any{map[string]any{"item": "c.heroarea", "params": []any{hero, []int{10, 12, 14, 15}}}},
					"actions":    []any{map[string]any{"item": "a.gold"}, map[string]any{"item": "a.win"}}}}}
		}},
		{"paint height", plain, func(t *testing.T, e *ui.MapEditor) {
			brushWindow(t, e, "height", 0)
			pointerStroke(e, stroke)
		}, func(t *testing.T, e *ui.MapEditor) []commandCall {
			return []commandCall{{"paint", map[string]any{"plane": "height", "value": 8, "cells": strokeCells}}}
		}},
		{"paint passability", plain, func(t *testing.T, e *ui.MapEditor) {
			brushWindow(t, e, "passability", 0)
			pointerStroke(e, stroke)
		}, func(t *testing.T, e *ui.MapEditor) []commandCall {
			return []commandCall{{"paint", map[string]any{"plane": "passability", "value": 1, "cells": strokeCells}}}
		}},
		{"randomize terrain", plain, func(t *testing.T, e *ui.MapEditor) { typeSeed(t, e, "randomterrain", 7) },
			func(t *testing.T, e *ui.MapEditor) []commandCall {
				return []commandCall{{"randomize", map[string]any{"plane": "terrain", "seed": 7}}}
			}},
		{"randomize heights", plain, func(t *testing.T, e *ui.MapEditor) { typeSeed(t, e, "randomheights", 9) },
			func(t *testing.T, e *ui.MapEditor) []commandCall {
				return []commandCall{{"randomize", map[string]any{"plane": "heights", "seed": 9}}}
			}},
		{"set_map_field", func(t *testing.T) *ui.MapEditor { _, e := workspaceFixture(t); return e }, func(t *testing.T, e *ui.MapEditor) {
			if err := workspaceField(e, "metadata", 0, "Карта"); err != nil {
				t.Fatal(err)
			}
		}, func(t *testing.T, e *ui.MapEditor) []commandCall {
			return []commandCall{{"set_map_field", map[string]any{"scope": "metadata", "key": "name", "value": "Карта"}}}
		}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			w := c.open(t)
			start := bytes.Clone(w.Document().SourceBytes)
			c.window(t, w)
			got := bytes.Clone(w.Document().SourceBytes)
			if bytes.Equal(got, start) {
				t.Fatalf("the window route changed nothing: %q", w.Message())
			}
			m := c.open(t)
			if !bytes.Equal(m.Document().SourceBytes, start) {
				t.Fatal("the two openings differ")
			}
			_, must = editorMCP(t, m)
			for _, call := range c.calls(t, m) {
				if r := commandReport(t, must(call.name, call.args)); r["changed"] != true || r["dirty"] != true {
					t.Fatalf("%s reports %v", call.name, r)
				}
			}
			if !bytes.Equal(m.Document().SourceBytes, got) {
				t.Fatalf("the command and the window route wrote different bytes: %s", tileDiff(t, got, m.Document().SourceBytes))
			}
			steps := 0
			for ; !bytes.Equal(w.Document().SourceBytes, start); steps++ {
				if steps == 8 || !w.Document().Edits.CanUndo() {
					t.Fatal("window undo does not reach the start")
				}
				_ = w.Key("undo")
				must("undo", map[string]any{})
				if !bytes.Equal(m.Document().SourceBytes, w.Document().SourceBytes) {
					t.Fatalf("undo step %d differs", steps+1)
				}
			}
			for range steps {
				_ = w.Key("redo")
				must("redo", map[string]any{})
				if !bytes.Equal(m.Document().SourceBytes, w.Document().SourceBytes) {
					t.Fatal("redo differs")
				}
			}
			if !bytes.Equal(m.Document().SourceBytes, got) {
				t.Fatal("redo did not give the edit back")
			}
		})
	}
}

// tileDiff names the first cells whose tile or height differ between two
// maps of one size.
func tileDiff(t *testing.T, a, b []byte) string {
	t.Helper()
	ma, errA := alm.Open(a)
	mb, errB := alm.Open(b)
	if errA != nil || errB != nil || len(ma.Tiles) != len(mb.Tiles) || len(ma.Altitudes) != len(mb.Altitudes) {
		return "the maps do not compare cell by cell"
	}
	out := ""
	for i := range ma.Tiles {
		if ma.Tiles[i] != mb.Tiles[i] || ma.Altitudes[i] != mb.Altitudes[i] {
			out += " " + strconv.Itoa(i%int(ma.Width)) + "," + strconv.Itoa(i/int(ma.Width)) + ":" + strconv.Itoa(int(ma.Tiles[i])) + "/" + strconv.Itoa(int(mb.Tiles[i]))
			if len(out) > 400 {
				break
			}
		}
	}
	return "cells" + out
}

// heroDefault is the unit the wizard's hero-area condition names first.
func heroDefault(t *testing.T, must func(name string, args any) editormcp.Result) string {
	t.Helper()
	var items struct {
		Conditions []struct {
			Item   string
			Params []struct{ Default string }
		}
	}
	if err := json.Unmarshal([]byte(must("trigger_items", map[string]any{}).Content[0].Text), &items); err != nil {
		t.Fatal(err)
	}
	for _, it := range items.Conditions {
		if it.Item == "c.heroarea" {
			return it.Params[0].Default
		}
	}
	t.Fatal("no hero-area condition")
	return ""
}

// The new-map command opens the map the window's New question makes.
func TestMapEditorNewMapCommandMatchesTheWindow(t *testing.T) {
	_, w := workspaceFixture(t)
	f := newTestPlayFixture(t)
	f.x.tiles = &terrain.Tileset{}
	m := f.x.Editor()
	_, must := editorMCP(t, m)
	r := commandReport(t, must("new_map", map[string]any{"width": 40, "height": 56}))
	if r["changed"] != true || r["width"] != float64(40) || r["height"] != float64(56) {
		t.Fatalf("new_map reports %v", r)
	}
	if !bytes.Equal(m.Document().SourceBytes, w.Document().SourceBytes) || m.Document().Source != w.Document().Source {
		t.Fatal("new_map and the window's New question opened different maps")
	}
}

// An agent builds a small playable map from nothing through the typed tools
// alone: size, terrain, a player, units and structures of two sides, the
// drop zone, a victory trigger; it saves, and the saved file reopens to the
// same bytes and records.
func TestMapEditorMCPBuildsAPlayableMapFromEmpty(t *testing.T) {
	e := agentEditor(t)
	tool, must := editorMCP(t, e)
	must("new_map", map[string]any{"width": 32, "height": 32})
	if d := e.Document(); d == nil || d.Width != 32 || d.Height != 32 {
		t.Fatal("new_map did not open a 32 x 32 map")
	}
	start := bytes.Clone(e.Document().SourceBytes)
	if r := commandReport(t, must("paint", map[string]any{"plane": "terrain", "value": int(mapedit.ClassSand), "radius": 4, "shape": "square", "cells": [][]int{{16, 16}}})); r["cells_total"] == nil {
		t.Fatalf("paint named no cells: %v", r)
	}
	must("paint", map[string]any{"plane": "height", "value": 6, "radius": 2, "cells": [][]int{{8, 24}}})
	must("add_record", map[string]any{"kind": "Player", "values": map[string]any{"name": "Rivals"}})
	var steps struct {
		Steps []struct {
			Key     string
			Choices []ui.InspectionChoice
		}
	}
	if err := json.Unmarshal([]byte(must("add_steps", map[string]any{"kind": "Unit", "x": 12, "y": 12, "values": map[string]any{"class": 40}}).Content[0].Text), &steps); err != nil {
		t.Fatal(err)
	}
	var owners []ui.InspectionChoice
	for _, s := range steps.Steps {
		if s.Key == "owner" {
			owners = s.Choices
		}
	}
	if len(owners) < 2 {
		t.Fatalf("the unit's owner step offers %v", owners)
	}
	a, b := owners[0].Value, owners[len(owners)-1].Value
	for _, add := range []map[string]any{
		{"kind": "Unit", "x": 12, "y": 12, "values": map[string]any{"class": 40, "owner": a}},
		{"kind": "Unit", "x": 20, "y": 20, "values": map[string]any{"class": 40, "owner": b}},
		{"kind": "Structure", "x": 10, "y": 17, "values": map[string]any{"kind": 5, "owner": a}},
		{"kind": "Structure", "x": 21, "y": 12, "values": map[string]any{"kind": 1, "owner": b}},
	} {
		if r := commandReport(t, must("add_record", add)); r["record"] == nil {
			t.Fatalf("add_record %v named no record: %v", add, r)
		}
	}
	must("add_drop_zone", map[string]any{"x": 13, "y": 14})
	if r := tool("set_drop_cell", map[string]any{"x": 13, "y": 15}); !r.IsError {
		t.Fatal("set_drop_cell took a cell on a map with drop zones")
	}
	hero, err := strconv.Atoi(heroDefault(t, must))
	if err != nil {
		t.Fatal(err)
	}
	must("write_trigger", map[string]any{"name": "Victory",
		"conditions": []any{map[string]any{"item": "c.heroarea", "params": []any{hero, []int{18, 18, 22, 22}}}},
		"actions":    []any{map[string]any{"item": "a.win"}}})
	report := must("check", map[string]any{}).Content[0].Text
	var found []struct{ Error bool }
	if err := json.Unmarshal([]byte(report), &found); err != nil {
		t.Fatal(err)
	}
	for _, f := range found {
		if f.Error {
			t.Fatalf("the map's check finds an error: %s", report)
		}
	}
	t.Logf("check: %s", report)
	path := filepath.Join(t.TempDir(), "built.alm")
	must("save_as", map[string]any{"path": path})
	saved, err := os.ReadFile(path)
	if err != nil || !bytes.Equal(saved, e.Document().SourceBytes) || bytes.Equal(saved, start) {
		t.Fatalf("the saved file is not the built map: %v", err)
	}
	m, err := alm.Open(saved)
	if err != nil {
		t.Fatal(err)
	}
	sc, err := m.Script()
	if err != nil {
		t.Fatal(err)
	}
	won := false
	for _, tr := range sc.Triggers {
		won = won || tr.Name == "Victory"
	}
	if len(m.Units) != 2 || len(m.Objects) != 2 || !won {
		t.Fatalf("the saved map holds %d units, %d structures, Victory %v", len(m.Units), len(m.Objects), won)
	}
	again := overlapInspector().Editor()
	if err := again.Open(path); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(again.Document().SourceBytes, saved) || len(again.Document().Records) != len(e.Document().Records) {
		t.Fatal("the saved map does not reopen equal")
	}
}

// storedCellChanges counts, per plane, the cells whose stored words differ
// between two maps of one size: the tile word without and with its
// passability bit, the height and the object byte.
func storedCellChanges(t *testing.T, a, b []byte) map[string]int {
	t.Helper()
	ma, errA := alm.Open(a)
	mb, errB := alm.Open(b)
	if errA != nil || errB != nil || len(ma.Tiles) != len(mb.Tiles) {
		t.Fatal("the maps do not compare cell by cell", errA, errB)
	}
	out := map[string]int{}
	for i := range ma.Tiles {
		if (ma.Tiles[i]^mb.Tiles[i])&^terrain.ImpassableBit != 0 {
			out["terrain"]++
		}
		if (ma.Tiles[i]^mb.Tiles[i])&terrain.ImpassableBit != 0 {
			out["passability"]++
		}
		if ma.Altitudes[i] != mb.Altitudes[i] {
			out["height"]++
		}
		if ma.Overlay[i] != mb.Overlay[i] {
			out["objects"]++
		}
	}
	return out
}

// The change report names what the map holds: the cells of each plane whose
// stored words differ, a passability stroke and its undo included, and the
// record whose stored fields set_field changed; a call that changes nothing
// names nothing.
func TestMapEditorCommandReportNamesWhatTheMapHolds(t *testing.T) {
	m := newMapWindow(t, &MapInspector{tiles: &terrain.Tileset{}}, 48)
	_, must := editorMCP(t, m)
	for _, c := range []commandCall{
		{"paint", map[string]any{"plane": "passability", "value": 1, "radius": 1, "cells": [][]int{{10, 10}}, "verbose": true}},
		{"undo", map[string]any{}},
		{"paint", map[string]any{"plane": "height", "value": 8, "radius": 2, "cells": [][]int{{24, 24}}}},
	} {
		before := bytes.Clone(m.Document().SourceBytes)
		r := commandReport(t, must(c.name, c.args))
		want := storedCellChanges(t, before, m.Document().SourceBytes)
		total := 0
		planes := map[string]any{}
		for p, n := range want {
			total += n
			planes[p] = float64(n)
		}
		// A paint lists the cells themselves only when the call asks for
		// them; every report names their count, planes and bounding box.
		cells, _ := r["cells"].([]any)
		listed := total
		if c.name == "paint" && c.args["verbose"] != true {
			listed = 0
		}
		if total == 0 || r["cells_total"] != float64(total) || len(cells) != listed || r["cells_box"] == nil || !reflect.DeepEqual(r["planes"], planes) {
			t.Fatalf("%s %v: the map changed %v, the report says %v", c.name, c.args, want, r)
		}
	}
	_, a := openAuthoring(t)
	_, must = editorMCP(t, a)
	body := map[string]any{"kind": "Unit", "index": 0, "key": "body", "value": 5}
	r := commandReport(t, must("set_field", body))
	unit := []any{map[string]any{"Kind": "Unit", "Index": float64(0)}}
	if r["changed"] != true || r["records_total"] != float64(1) || !reflect.DeepEqual(r["records"], unit) || r["cells_total"] != float64(0) {
		t.Fatalf("set_field reports %v", r)
	}
	if r := commandReport(t, must("set_field", body)); r["changed"] != false || r["records"] != nil || r["cells"] != nil {
		t.Fatalf("a set_field that changes nothing reports %v", r)
	}
}

// A typed 0 in a field that stores 0 as "inherit" is refused as the window's
// field prompt refuses it, with the window's words, and the map keeps its
// bytes, through set_field, set_fields and set_object alike.
func TestMapEditorCommandsRefuseZeroInAnInheritingField(t *testing.T) {
	_, w := openAuthoring(t)
	at := -1
	for i, f := range w.Document().Edits.Fields("Unit", 0) {
		if f.Key == "body" && f.Inherit && f.Absent == 0 {
			at = i
		}
	}
	if at < 0 {
		t.Fatal("the fixture's unit has no inheriting body field")
	}
	start := bytes.Clone(w.Document().SourceBytes)
	w.Select(0, true)
	if err := enterField(w, at, "0"); err == nil || !bytes.Equal(start, w.Document().SourceBytes) {
		t.Fatal("the window's field prompt took the 0")
	}
	want := w.Message()
	if !strings.Contains(want, "inherit") {
		t.Fatalf("the window says %q", want)
	}
	zero := []any{map[string]any{"key": "body", "value": 0}}
	for _, c := range []commandCall{
		{"set_field", map[string]any{"kind": "Unit", "index": 0, "key": "body", "value": 0}},
		{"set_fields", map[string]any{"kind": "Unit", "index": 0, "fields": zero}},
		{"set_object", map[string]any{"kind": "Unit", "index": 0, "fields": zero}},
	} {
		_, m := openAuthoring(t)
		tool, must := editorMCP(t, m)
		must("set_field", map[string]any{"kind": "Unit", "index": 0, "key": "body", "value": 5})
		five := bytes.Clone(m.Document().SourceBytes)
		r := tool(c.name, c.args)
		if !r.IsError || len(r.Content) == 0 || r.Content[0].Text != want || !bytes.Equal(five, m.Document().SourceBytes) {
			t.Fatalf("%s took the 0 or said another reason: %+v", c.name, r)
		}
		// No value is the prompt's empty text: the field inherits again.
		must("set_field", map[string]any{"kind": "Unit", "index": 0, "key": "body"})
		if !bytes.Equal(start, m.Document().SourceBytes) {
			t.Fatal("set_field without a value did not inherit")
		}
	}
}

// The typed undo is the window's Undo: after undoing a message just made on
// an Action's open message page, the window shows what its own Undo shows.
func TestMapEditorTypedUndoLeavesTheWindowAsItsUndo(t *testing.T) {
	type state struct {
		Popup    []string
		Open     bool
		Selected int
		Message  string
		Restored bool
	}
	run := func(typed bool) state {
		e, s := messagePopupFixture(t, 77)
		before := bytes.Clone(s.model.Bytes())
		messageField(t, e, 2)
		r, ok := e.DropdownRowRect(0)
		messageClick(t, e, r, ok)
		messageField(t, e, 0)
		e.TextInput("First")
		messageButton(t, e, 0)
		if bytes.Equal(before, s.model.Bytes()) {
			t.Fatal("the message page made no message")
		}
		if typed {
			_, must := editorMCP(t, e)
			must("undo", map[string]any{})
		} else if err := e.Key("undo"); err != nil {
			t.Fatal(err)
		}
		_, open := e.PopupRect()
		return state{Popup: e.PopupLines(), Open: open, Selected: e.Selected(), Message: e.Message(), Restored: bytes.Equal(before, s.model.Bytes())}
	}
	window, typed := run(false), run(true)
	if !window.Restored || !window.Open || len(window.Popup) == 0 {
		t.Fatalf("the window's Undo leaves %+v", window)
	}
	if !reflect.DeepEqual(window, typed) {
		t.Fatalf("the typed undo leaves %+v, the window's Undo %+v", typed, window)
	}
}

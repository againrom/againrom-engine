package game

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"image"
	"os"
	"path/filepath"
	"testing"
	"time"

	"againrom/pkg/formats/alm"
	"againrom/pkg/render/terrain"
	"againrom/pkg/ui"
)

// editorState is what an edit writes: the map's bytes and the workspace
// beside them, the message pages and the mod package.
func editorState(e *ui.MapEditor) []byte {
	src := e.Document().SourceBytes
	out := binary.AppendUvarint(nil, uint64(len(src)))
	out = append(out, src...)
	return append(out, e.Document().Edits.(*mapInspectionEdits).workspaceImage...)
}

// refArgs are refs as a tool's records argument names them.
func refArgs(refs ...ui.RecordRef) []any {
	out := make([]any, 0, len(refs))
	for _, r := range refs {
		out = append(out, map[string]any{"kind": r.Kind, "index": r.Index})
	}
	return out
}

// newestOf is the index of the last record of kind in e's document.
func newestOf(e *ui.MapEditor, kind string) int {
	n := -1
	for _, r := range e.Document().Records {
		if r.Kind == kind {
			n = max(n, r.Index)
		}
	}
	return n
}

// The documented example session over the in-process MCP client: a selection
// move, a group, a shop stock, a message and a goal target, each one undo
// step, then save_as; the saved map and its message page reopen equal.
func TestMapEditorMCPExampleSessionSavesAndReopens(t *testing.T) {
	e, _ := openGoalMap(t, goalMap(t, true))
	_, must := editorMCP(t, e)
	start := editorState(e)
	must("map", map[string]any{"x": 18, "y": 14})
	var goals []struct {
		Goal int
		Pick string
	}
	if err := json.Unmarshal([]byte(must("goals", map[string]any{}).Content[0].Text), &goals); err != nil || len(goals) == 0 || goals[0].Pick != "Unit" {
		t.Fatalf("goals reads %+v: %v", goals, err)
	}
	steps := []commandCall{
		{"move_records", map[string]any{"records": refArgs(ui.RecordRef{Kind: "Unit", Index: 2}, ui.RecordRef{Kind: "Unit", Index: 3}), "dx": 0, "dy": 4}},
		{"set_units_group", map[string]any{"units": []int{2, 3}, "group": -1}},
		{"add_stock", map[string]any{"unit": 0}},
		{"add_record", map[string]any{"kind": "Message", "parts": []any{map[string]any{"body": "Hold the bridge"}, map[string]any{"body": "Then go north"}}}},
		{"set_goal_target", map[string]any{"goal": 0, "kind": "Unit", "index": 3}},
	}
	var page float64
	for i, call := range steps {
		r := commandReport(t, must(call.name, call.args))
		if r["changed"] != true {
			t.Fatalf("%s reports %v", call.name, r)
		}
		if p, ok := r["page"].(float64); ok {
			page = p
		}
		if call.name == "move_records" && r["selected"] == nil {
			t.Fatalf("the move report names no selection: %v", r)
		}
		if i == 0 {
			before := editorState(e)
			off := commandReport(t, must("move_records", map[string]any{"records": refArgs(ui.RecordRef{Kind: "Unit", Index: 2}), "dx": 0, "dy": 40}))
			if off["changed"] == true || off["refusals"] == nil || !bytes.Equal(editorState(e), before) {
				t.Fatalf("a move off the map was taken or not reported: %v", off)
			}
		}
	}
	if page < 1 {
		t.Fatal("the message reports no page")
	}
	var read struct {
		Page  int
		Parts []struct{ Source, Number int }
	}
	if err := json.Unmarshal([]byte(must("messages", map[string]any{"page": int(page)}).Content[0].Text), &read); err != nil {
		t.Fatal(err)
	}
	if read.Page != int(page) || len(read.Parts) != 2 || read.Parts[0].Source < 0 {
		t.Fatalf("messages reads %+v", read)
	}
	path := filepath.Join(t.TempDir(), "session.alm")
	must("save_as", map[string]any{"path": path})
	// The save writes the map in the form the game loads it (alm.SaveForm):
	// this map has no player records, so its units are written as player 1's.
	saved, err := os.ReadFile(path)
	want, werr := alm.SaveForm(e.Document().SourceBytes)
	if err != nil || werr != nil || !bytes.Equal(saved, want) {
		t.Fatalf("the saved file is not the edited map in its save form: %v %v", err, werr)
	}
	again := (&MapInspector{tiles: &terrain.Tileset{}, rules: []mapRule{}}).Editor()
	if err := again.Open(path); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(again.Document().SourceBytes, saved) {
		t.Fatal("the saved map does not reopen equal")
	}
	host := again.Document().Edits.(*mapInspectionEdits)
	parts := host.MessageParts(uint32(page))
	if len(parts) != 2 || parts[0].Body != "Hold the bridge" || parts[1].Body != "Then go north" {
		t.Fatalf("the reopened page holds %+v", parts)
	}
	for range steps {
		must("undo", map[string]any{})
	}
	if !bytes.Equal(editorState(e), start) {
		t.Fatal("five undos do not give the opened map back")
	}
}

// Each typed tool refuses bad arguments with isError over a real host, and
// the refusal writes nothing: the map, the workspace and the history stay.
func TestMapEditorCommandEditsRefuseBadArguments(t *testing.T) {
	unit := func(i int) map[string]any { return map[string]any{"kind": "Unit", "index": i} }
	part := []any{map[string]any{"body": "x"}}
	type call struct {
		name string
		args map[string]any
	}
	for _, f := range []struct {
		name  string
		open  func(t *testing.T) *ui.MapEditor
		calls []call
	}{
		{"selection", func(t *testing.T) *ui.MapEditor { return groupEditor(t, groupMap(t, 2)) }, []call{
			{"move_records", map[string]any{"records": []any{}, "dx": 1, "dy": 0}},
			{"move_records", map[string]any{"records": []any{unit(0), unit(0)}, "dx": 1, "dy": 0}},
			{"move_records", map[string]any{"records": []any{unit(9)}, "dx": 1, "dy": 0}},
			{"move_records", map[string]any{"records": []any{unit(0)}, "dx": 1}},
			{"move_records", map[string]any{"records": []any{unit(0)}, "dx": 1, "dy": 0, "typo": 1}},
			{"delete_records", map[string]any{"records": []any{}}},
			{"delete_records", map[string]any{"records": []any{map[string]any{"kind": "Player", "index": 0}}}},
			{"copy_records", map[string]any{"x": 1, "y": 1}},
			{"copy_records", map[string]any{"records": []any{unit(0)}, "area": []int{1, 1, 2, 2}, "x": 1, "y": 1}},
			{"copy_records", map[string]any{"area": []int{5, 5, 2, 2}, "x": 1, "y": 1}},
			{"copy_records", map[string]any{"area": []int{0, 0, 50, 50}, "x": 1, "y": 1}},
			{"copy_records", map[string]any{"records": []any{unit(0)}, "x": 99, "y": 1}},
			{"move_area", map[string]any{"area": []int{10, 10, 15, 15}, "dx": 30, "dy": 0}},
			{"move_area", map[string]any{"area": []int{11, 11, 13, 13}, "dx": 1, "dy": 0, "records": []any{map[string]any{"kind": "Structure", "index": 0}}}},
			{"move_area", map[string]any{"area": []int{10, 10, 15}, "dx": 1, "dy": 0}},
			{"set_units_group", map[string]any{"units": []int{}, "group": 3}},
			{"set_units_group", map[string]any{"units": []int{0, 0}, "group": 3}},
			{"set_units_group", map[string]any{"units": []int{9}, "group": 3}},
			{"set_units_group", map[string]any{"units": []int{0}, "group": -2}},
			{"set_units_group", map[string]any{"units": []int{0, 1}, "group": 3}},
			{"add_stock", map[string]any{"unit": 9}},
			{"add_item_recipe", map[string]any{"kind": "Unit", "index": 0, "item": 1}},
			{"add_item_recipe", map[string]any{"kind": "Sack", "index": 9, "item": 1}},
			{"set_goal_target", map[string]any{"goal": 0, "kind": "Unit", "index": 0}},
			{"unit_settings", map[string]any{"index": 9}},
		}},
		{"message", func(t *testing.T) *ui.MapEditor { e, _ := messagePopupFixture(t, 77); return e }, []call{
			{"write_message", map[string]any{"parts": part}},
			{"write_message", map[string]any{"page": 5, "parts": part}},
			{"write_message", map[string]any{"action": 9, "parts": part}},
			{"write_message", map[string]any{"action": 0, "parts": []any{}}},
			{"write_message", map[string]any{"action": 0, "parts": []any{map[string]any{"source": 3, "body": "x"}}}},
			{"write_message", map[string]any{"action": 0, "parts": []any{map[string]any{"body": "x", "speaker": 9999}}}},
			{"write_message", map[string]any{"action": 0, "parts": []any{map[string]any{"text": "x"}}}},
			{"add_record", map[string]any{"kind": "Message", "parts": []any{}}},
			{"add_record", map[string]any{"kind": "Message"}},
			{"add_record", map[string]any{"kind": "Action", "parts": part}},
			{"messages", map[string]any{"page": 5}},
		}},
		{"area", func(t *testing.T) *ui.MapEditor { e, _ := openGoalMap(t, areaEditMap(t, true)); return e }, []call{
			{"add_box_area", map[string]any{"area": []int{6, 33, 2, 30}}},
			{"add_box_area", map[string]any{"area": []int{2, 30, 6, 99}}},
			{"add_box_area", map[string]any{"area": []int{2, 30, 6, 33}, "trigger": 99}},
			{"add_box_area", map[string]any{"area": []int{2, 30, 6, 33}, "trigger": -2}},
		}},
		{"goal", func(t *testing.T) *ui.MapEditor { e, _ := openGoalMap(t, goalMap(t, true)); return e }, []call{
			{"set_goal_target", map[string]any{"goal": 99, "kind": "Unit", "index": 0}},
			{"set_goal_target", map[string]any{"goal": 0, "kind": "Structure", "index": 0}},
			{"set_goal_target", map[string]any{"goal": 0, "kind": "Unit", "index": 99}},
			{"set_goal_target", map[string]any{"goal": 0, "kind": "Sack", "index": 0}},
		}},
		{"forest", func(t *testing.T) *ui.MapEditor { e, _ := forestEditor(t, forestMap(t)); return e }, []call{
			{"paint", map[string]any{"plane": "objects", "cells": [][]int{{20, 12}}, "forest": map[string]any{"kinds": []int{}}}},
			{"paint", map[string]any{"plane": "objects", "cells": [][]int{{20, 12}}, "forest": map[string]any{"kinds": []int{999}}}},
			{"paint", map[string]any{"plane": "objects", "cells": [][]int{{20, 12}}, "forest": map[string]any{"kinds": []int{forestCodes[0], forestCodes[0]}}}},
			{"paint", map[string]any{"plane": "objects", "cells": [][]int{{20, 12}}, "forest": map[string]any{"kinds": []int{forestCodes[0]}, "share": 0}}},
			{"paint", map[string]any{"plane": "objects", "cells": [][]int{{20, 12}}, "forest": map[string]any{"kinds": []int{forestCodes[0]}, "share": 101}}},
			{"paint", map[string]any{"plane": "objects", "value": 3, "cells": [][]int{{20, 12}}, "forest": map[string]any{"kinds": []int{forestCodes[0]}}}},
			{"paint", map[string]any{"plane": "terrain", "value": 1, "cells": [][]int{{20, 12}}, "forest": map[string]any{"kinds": []int{forestCodes[0]}}}},
		}},
		{"package", func(t *testing.T) *ui.MapEditor { _, e := workspaceFixture(t); return e }, []call{
			{"set_map_field", map[string]any{"scope": "package", "key": "folder", "value": "x"}},
			{"set_map_field", map[string]any{"scope": "package", "key": "source", "value": "1"}},
			{"set_map_field", map[string]any{"scope": "package", "key": "output", "value": "x"}},
			{"set_map_field", map[string]any{"scope": "package", "key": "nosuch", "value": "x"}},
			{"set_map_field", map[string]any{"scope": "mod", "key": "gold", "value": "1"}},
		}},
	} {
		t.Run(f.name, func(t *testing.T) {
			e := f.open(t)
			tool, _ := editorMCP(t, e)
			before, undo := editorState(e), e.Document().Edits.CanUndo()
			for _, c := range f.calls {
				r := tool(c.name, c.args)
				if !r.IsError || len(r.Content) == 0 {
					t.Errorf("%s %v accepted: %+v", c.name, c.args, r.Content)
				} else {
					t.Logf("%s: %s", c.name, r.Content[0].Text)
				}
				if !bytes.Equal(editorState(e), before) || e.Document().Edits.CanUndo() != undo {
					t.Fatalf("%s %v changed the map, the workspace or the history", c.name, c.args)
				}
			}
		})
	}
}

// The selection, group, stock, recipe, message, box area, goal, forest and
// mod package routes of the window and the typed command each holds give the
// same map and workspace from the same map, the same undo steps back to
// where they started, and the same redo.
func TestMapEditorCommandEditsMatchTheWindowRoutes(t *testing.T) {
	group := func(players int) func(t *testing.T, _ string) *ui.MapEditor {
		return func(t *testing.T, _ string) *ui.MapEditor { return groupEditor(t, groupMap(t, players)) }
	}
	box := func(t *testing.T, _ string) *ui.MapEditor {
		e, _ := openGoalMap(t, boxMoveMap(t))
		e.Fit()
		return e
	}
	records := func(t *testing.T, _ string) *ui.MapEditor { _, e := openRecords(t); return e }
	unit := func(t *testing.T, _ string) *ui.MapEditor {
		_, e := openRecords(t)
		recordAddInput(t, e, "unit", image.Pt(11, 11), 40)
		return e
	}
	message := func(t *testing.T, _ string) *ui.MapEditor { e, _ := messagePopupFixture(t, 77); return e }
	newPage := func(t *testing.T, e *ui.MapEditor) {
		messageField(t, e, 2)
		r, ok := e.DropdownRowRect(0)
		messageClick(t, e, r, ok)
		messageField(t, e, 0)
		e.TextInput("First")
		_ = e.Key("newline")
		e.TextInput("Second")
		messageButton(t, e, 0)
	}
	written := func(t *testing.T, _ string) *ui.MapEditor {
		e, _ := messagePopupFixture(t, 77)
		newPage(t, e)
		messageButton(t, e, 2)
		return e
	}
	areas := func(t *testing.T, _ string) *ui.MapEditor { e, _ := openGoalMap(t, areaEditMap(t, true)); return e }
	goals := func(t *testing.T, _ string) *ui.MapEditor {
		e, _ := openGoalMap(t, goalMap(t, true))
		e.Layout(1600, 900)
		return e
	}
	forest := func(t *testing.T, _ string) *ui.MapEditor {
		forestSeed(t, 3)
		d, _ := newForestDriver(t, forestMap(t))
		return d.e
	}
	pack := func(t *testing.T, dir string) *ui.MapEditor {
		_, e := workspaceFixture(t)
		if err := workspaceField(e, "package", 0, filepath.Join(dir, "authored-map")); err != nil {
			t.Fatal(err)
		}
		_ = e.Key("escape")
		return e
	}
	fixed := func(calls ...commandCall) func(*ui.MapEditor) []commandCall {
		return func(*ui.MapEditor) []commandCall { return calls }
	}
	cases := []struct {
		name   string
		open   func(t *testing.T, dir string) *ui.MapEditor
		window func(t *testing.T, e *ui.MapEditor)
		calls  func(m *ui.MapEditor) []commandCall
	}{
		{"move_records", group(2), func(t *testing.T, e *ui.MapEditor) {
			selectArea(e, image.Pt(11, 11), image.Pt(21, 21))
			dragCell(e, image.Pt(12, 12), image.Pt(15, 14))
		}, fixed(commandCall{"move_records", map[string]any{"records": refArgs(groupRefs...), "dx": 3, "dy": 2}})},
		{"move_records past the edge", group(2), func(t *testing.T, e *ui.MapEditor) {
			selectArea(e, image.Pt(11, 11), image.Pt(21, 21))
			dragCell(e, image.Pt(14, 12), image.Pt(1, 12))
		}, fixed(commandCall{"move_records", map[string]any{"records": refArgs(groupRefs...), "dx": -13, "dy": 0}})},
		{"delete_records", group(2), func(t *testing.T, e *ui.MapEditor) {
			selectArea(e, image.Pt(11, 11), image.Pt(21, 21))
			if err := e.Key("delete"); err != nil {
				t.Fatal(err)
			}
		}, fixed(commandCall{"delete_records", map[string]any{"records": refArgs(groupRefs...)}})},
		{"copy_records", group(2), func(t *testing.T, e *ui.MapEditor) {
			selectArea(e, image.Pt(11, 11), image.Pt(21, 21))
			if err := e.Key("copy"); err != nil {
				t.Fatal(err)
			}
			pasteAt(t, e, image.Pt(9, 22))
		}, fixed(commandCall{"copy_records", map[string]any{"records": refArgs(groupRefs...), "x": 9, "y": 22}})},
		{"copy_records area", box, func(t *testing.T, e *ui.MapEditor) {
			dragCell(e, image.Pt(10, 10), image.Pt(15, 15))
			if err := e.Key("copy"); err != nil {
				t.Fatal(err)
			}
			pasteAt(t, e, image.Pt(20, 20))
		}, fixed(commandCall{"copy_records", map[string]any{"area": []int{10, 10, 15, 15}, "x": 20, "y": 20}})},
		{"move_area", box, func(t *testing.T, e *ui.MapEditor) {
			dragCell(e, image.Pt(10, 10), image.Pt(15, 15))
			dragCell(e, image.Pt(11, 11), image.Pt(21, 14))
		}, fixed(commandCall{"move_area", map[string]any{"area": []int{10, 10, 15, 15}, "dx": 10, "dy": 3}})},
		{"set_units_group", group(1), func(t *testing.T, e *ui.MapEditor) {
			selectArea(e, image.Pt(11, 11), image.Pt(21, 21))
			if err := e.Key("togroup7"); err != nil {
				t.Fatal(err)
			}
		}, fixed(commandCall{"set_units_group", map[string]any{"units": []int{0, 1}, "group": 7}})},
		{"add_stock", unit, func(t *testing.T, e *ui.MapEditor) {
			if err := e.Key("stock"); err != nil {
				t.Fatal(err)
			}
		}, func(m *ui.MapEditor) []commandCall {
			return []commandCall{{"add_stock", map[string]any{"unit": newestOf(m, "Unit")}}}
		}},
		{"add_item_recipe", records, func(t *testing.T, e *ui.MapEditor) {
			e.Select(recordIndex(t, e, "Stock", 0), true)
			_ = e.Key("recipe")
			e.TextInput("1")
			if err := e.Key("enter"); err != nil {
				t.Fatal(err)
			}
		}, fixed(commandCall{"add_item_recipe", map[string]any{"kind": "Stock", "index": 0, "item": 1}})},
		{"write_message new page", message, newPage, fixed(commandCall{"write_message", map[string]any{"action": 0, "parts": []any{map[string]any{"body": "First\nSecond"}}}})},
		{"write_message page", written, func(t *testing.T, e *ui.MapEditor) {
			opened := false
			for i, line := range e.PopupLines() {
				if len(line) >= 12 && line[:12] == "Open Message" {
					r, ok := e.PopupRowRect(i)
					messageClick(t, e, r, ok)
					opened = true
					break
				}
			}
			if !opened {
				t.Fatalf("no Open Message link: %q", e.PopupLines())
			}
			messageField(t, e, 0)
			_ = e.Key("clear")
			e.TextInput("Rewritten")
			messageButton(t, e, 0)
		}, fixed(commandCall{"write_message", map[string]any{"page": 1, "parts": []any{map[string]any{"source": 0, "body": "Rewritten"}}}})},
		{"add_record message", message, func(t *testing.T, e *ui.MapEditor) {
			if err := e.Key("scripts"); err != nil {
				t.Fatal(err)
			}
			addWindow(t, e, "player", map[string]string{"kind": "3"})
			messageField(t, e, 0)
			e.TextInput("Hello")
			messageButton(t, e, 0)
		}, fixed(commandCall{"add_record", map[string]any{"kind": "Message", "parts": []any{map[string]any{"body": "Hello"}}}})},
		{"add_box_area new trigger", areas, func(t *testing.T, e *ui.MapEditor) {
			areaDraw(t, e, image.Pt(6, 33), image.Pt(2, 30))
			_ = e.Key("enter")
		}, fixed(commandCall{"add_box_area", map[string]any{"area": []int{2, 30, 6, 33}}})},
		{"add_box_area trigger", areas, func(t *testing.T, e *ui.MapEditor) {
			areaDraw(t, e, image.Pt(2, 30), image.Pt(3, 31))
			_ = e.Key("down")
			_ = e.Key("down")
			_ = e.Key("enter")
		}, fixed(commandCall{"add_box_area", map[string]any{"area": []int{2, 30, 3, 31}, "trigger": 1}})},
		{"set_goal_target", goals, func(t *testing.T, e *ui.MapEditor) {
			_ = e.Key("goals")
			_ = e.Key("enter")
			e.ShowCell(image.Pt(21, 14))
			clickCell(e, image.Pt(21, 14))
		}, fixed(commandCall{"set_goal_target", map[string]any{"goal": 0, "kind": "Unit", "index": 3}})},
		{"paint forest", forest, func(t *testing.T, e *ui.MapEditor) {
			d := &forestDriver{t: t, e: e, c: &editorClicker{t: t, now: time.Unix(1, 0)}, now: time.Unix(1, 0)}
			d.forest(2, 2)
			d.button(4, 3, "Dense")
			d.stroke(false, image.Pt(24, 16))
		}, fixed(commandCall{"paint", map[string]any{"plane": "objects", "radius": 2, "shape": "square", "cells": [][]int{{24, 16}},
			"forest": map[string]any{"kinds": []int{forestCodes[0], forestCodes[1]}, "share": 50}}})},
		{"paint forest thin", forest, func(t *testing.T, e *ui.MapEditor) {
			d := &forestDriver{t: t, e: e, c: &editorClicker{t: t, now: time.Unix(1, 0)}, now: time.Unix(1, 0)}
			d.forest(1, 16)
			d.button(4, 3, "Dense")
			d.stroke(true, image.Pt(20, 12))
		}, fixed(commandCall{"paint", map[string]any{"plane": "objects", "radius": 16, "shape": "square", "cells": [][]int{{20, 12}},
			"forest": map[string]any{"kinds": []int{forestCodes[0]}, "share": 50, "thin": true}}})},
		{"set_map_field package", pack, func(t *testing.T, e *ui.MapEditor) {
			if err := workspaceField(e, "package", 8, "73"); err != nil {
				t.Fatal(err)
			}
		}, fixed(commandCall{"set_map_field", map[string]any{"scope": "package", "key": "gold", "value": "73"}})},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			dir := t.TempDir()
			w := c.open(t, dir)
			start := editorState(w)
			c.window(t, w)
			got := editorState(w)
			if bytes.Equal(got, start) {
				t.Fatalf("the window route changed nothing: %q", w.Message())
			}
			m := c.open(t, dir)
			if !bytes.Equal(editorState(m), start) {
				t.Fatal("the two openings differ")
			}
			_, must := editorMCP(t, m)
			for _, call := range c.calls(m) {
				if r := commandReport(t, must(call.name, call.args)); r["changed"] != true || r["dirty"] != true {
					t.Fatalf("%s reports %v", call.name, r)
				}
			}
			if !bytes.Equal(editorState(m), got) {
				same := bytes.Equal(m.Document().SourceBytes, w.Document().SourceBytes)
				t.Fatalf("the command and the window route wrote different states (map bytes equal: %v): %s", same, tileDiff(t, w.Document().SourceBytes, m.Document().SourceBytes))
			}
			steps := 0
			for ; !bytes.Equal(editorState(w), start); steps++ {
				if steps == 8 || !w.Document().Edits.CanUndo() {
					t.Fatal("window undo does not reach the start")
				}
				_ = w.Key("undo")
				must("undo", map[string]any{})
				if !bytes.Equal(editorState(m), editorState(w)) {
					t.Fatalf("undo step %d differs", steps+1)
				}
			}
			for range steps {
				_ = w.Key("redo")
				must("redo", map[string]any{})
				if !bytes.Equal(editorState(m), editorState(w)) {
					t.Fatal("redo differs")
				}
			}
			if !bytes.Equal(editorState(m), got) {
				t.Fatal("redo did not give the edit back")
			}
		})
	}
}

// A move that takes some records past the map edge moves the rest and lists
// the refused ones in the change report, as the window's drag does; a move
// that takes every record past the edge changes nothing.
func TestMapEditorMoveRecordsPastTheEdgeReportsRefusals(t *testing.T) {
	e := groupEditor(t, groupMap(t, 2))
	_, must := editorMCP(t, e)
	r := commandReport(t, must("move_records", map[string]any{"records": refArgs(groupRefs...), "dx": -13, "dy": 0}))
	refusals, _ := r["refusals"].([]any)
	if r["changed"] != true || r["note"] != "Moved 1 of 4" || len(refusals) != 3 {
		t.Fatalf("the partial move reports %v", r)
	}
	before, undo := editorState(e), e.Document().Edits.CanUndo()
	r = commandReport(t, must("move_records", map[string]any{"records": refArgs(groupRefs[0]), "dx": 0, "dy": -40}))
	if r["changed"] == true || len(r["refusals"].([]any)) != 1 {
		t.Fatalf("the move of one record off the map reports %v", r)
	}
	if !bytes.Equal(editorState(e), before) || e.Document().Edits.CanUndo() != undo {
		t.Fatal("a move off the map changed the map or the history")
	}
}

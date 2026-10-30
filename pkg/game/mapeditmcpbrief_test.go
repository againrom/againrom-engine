package game

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"image"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"testing"
	"time"

	"againrom/internal/synth"
	"againrom/pkg/editormcp"
	"againrom/pkg/mapedit"
	"againrom/pkg/ui"
)

// readHeights is the heights tool's answer over x0,y0..x1,y1.
func readHeights(t *testing.T, must func(string, any) editormcp.Result, x0, y0, x1, y1 int) (rows [][]int, low, high int) {
	t.Helper()
	var got struct {
		Range     []int
		Rows      [][]int
		Min, Max  int
		LabelZero int `json:"label_zero"`
	}
	if err := json.Unmarshal([]byte(must("heights", map[string]int{"x0": x0, "y0": y0, "x1": x1, "y1": y1}).Content[0].Text), &got); err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(got.Range, []int{x0, y0, x1, y1}) || got.LabelZero != ui.HeightLabelZero || len(got.Rows) != y1-y0+1 {
		t.Fatalf("heights answered range %v, label_zero %d, %d rows", got.Range, got.LabelZero, len(got.Rows))
	}
	return got.Rows, got.Min, got.Max
}

// The heights tool reads back, after a stroke of each height mode, the
// height plane the map's bytes store.
func TestMapEditorMCPHeightsReadTheStoredPlane(t *testing.T) {
	e := agentEditor(t)
	tool, must := editorMCP(t, e)
	must("new_map", map[string]any{"width": 32, "height": 32})
	for _, stroke := range []map[string]any{
		{"value": 5, "radius": 3, "cells": [][]int{{10, 10}, {12, 11}}},
		{"mode": "level", "value": 30, "radius": 2, "cells": [][]int{{20, 12}}},
		{"mode": "smooth", "value": -12, "radius": 4, "cells": [][]int{{14, 20}, {15, 20}}},
		{"set": true, "value": 90, "radius": 2, "cells": [][]int{{22, 22}, {5, 5}}},
		{"set": true, "flat": true, "value": 40, "radius": 1, "cells": [][]int{{8, 24}, {9, 25}}},
	} {
		stroke["plane"] = "height"
		must("paint", stroke)
		rows, low, high := readHeights(t, must, 0, 0, 31, 31)
		plane := mustMap(t, e.Document().SourceBytes).Altitudes
		for y, row := range rows {
			for x, h := range row {
				if h != int(plane[y*32+x]) {
					t.Fatalf("%v: heights gives %d at %d,%d; the map stores %d", stroke, h, x, y, plane[y*32+x])
				}
			}
		}
		if low != int(slices.Min(plane)) || high != int(slices.Max(plane)) {
			t.Fatalf("%v: min/max %d/%d, plane %d/%d", stroke, low, high, slices.Min(plane), slices.Max(plane))
		}
	}
	if r := tool("heights", map[string]int{"x0": 0, "y0": 0, "x1": 31, "y1": 31}); r.IsError {
		t.Fatal(r.Content[0].Text)
	}
	for _, bad := range []map[string]int{{"x0": 0, "y0": 0, "x1": 32, "y1": 0}, {"x0": 5, "y0": 0, "x1": 4, "y1": 0}} {
		if r := tool("heights", bad); !r.IsError {
			t.Fatalf("heights took %v", bad)
		}
	}
	e2 := agentEditor(t)
	tool2, must2 := editorMCP(t, e2)
	must2("new_map", map[string]any{"width": 80, "height": 80})
	if r := tool2("heights", map[string]int{"x0": 0, "y0": 0, "x1": 64, "y1": 63}); !r.IsError || !strings.Contains(r.Content[0].Text, "at most 4096") {
		t.Fatalf("a 65x64 read: %+v", r)
	}
}

// A height stroke's report gives each covered cell's height before and
// after only with verbose; without it the count, the box and the least and
// greatest change.
func TestMapEditorMCPPaintReportsHeightsBeforeAndAfter(t *testing.T) {
	e := agentEditor(t)
	_, must := editorMCP(t, e)
	must("new_map", map[string]any{"width": 32, "height": 32})
	before, _, _ := readHeights(t, must, 0, 0, 31, 31)
	rep := commandReport(t, must("paint", map[string]any{"plane": "height", "mode": "smooth", "value": 12, "radius": 4, "cells": [][]int{{16, 16}}, "verbose": true}))
	after, _, _ := readHeights(t, must, 0, 0, 31, 31)
	var cells [][]int
	b, _ := json.Marshal(rep["heights"])
	if err := json.Unmarshal(b, &cells); err != nil || len(cells) != 49 {
		t.Fatalf("verbose heights %v", rep["heights"])
	}
	for _, c := range cells {
		x, y := c[0], c[1]
		if c[2] != before[y][x] || c[3] != after[y][x] {
			t.Fatalf("cell %d,%d reported %d -> %d, read %d -> %d", x, y, c[2], c[3], before[y][x], after[y][x])
		}
	}
	// The centre takes the whole change; a neighbour 12*63/64.
	if after[16][16]-before[16][16] != 12 || after[16][17]-before[16][17] != 11 {
		t.Fatalf("smooth 12: centre +%d, neighbour +%d", after[16][16]-before[16][16], after[16][17]-before[16][17])
	}
	brief := commandReport(t, must("paint", map[string]any{"plane": "height", "value": 3, "radius": 1, "cells": [][]int{{5, 5}}}))
	if _, listed := brief["heights"]; listed {
		t.Fatal("a stroke without verbose listed heights")
	}
	change, _ := brief["height_change"].(map[string]any)
	if change["cells"] != 5.0 || change["changed"] != 5.0 || change["min"] != 3.0 || change["max"] != 3.0 || fmt.Sprint(change["box"]) != "[4 4 6 6]" {
		t.Fatalf("height_change %v", change)
	}
}

// A stroke of three centres covers three circles, each cell once, as the
// paint description says: the line between centres is not filled.
func TestMapEditorMCPStrokeOfThreeCentres(t *testing.T) {
	e := agentEditor(t)
	_, must := editorMCP(t, e)
	must("new_map", map[string]any{"width": 40, "height": 40})
	before, _, _ := readHeights(t, must, 0, 0, 39, 39)
	centres := [][]int{{8, 16}, {17, 16}, {26, 16}}
	rep := commandReport(t, must("paint", map[string]any{"plane": "height", "value": 5, "radius": 4, "cells": centres}))
	after, _, _ := readHeights(t, must, 0, 0, 39, 39)
	if change, _ := rep["height_change"].(map[string]any); change["cells"] != 147.0 || change["changed"] != 147.0 {
		t.Fatalf("three radius-4 centres: %v", rep["height_change"])
	}
	for y := range after {
		for x := range after[y] {
			covered := false
			for _, c := range centres {
				covered = covered || (x-c[0])*(x-c[0])+(y-c[1])*(y-c[1]) <= 16
			}
			want := before[y][x]
			if covered {
				want += 5
			}
			if after[y][x] != want {
				t.Fatalf("cell %d,%d: %d, want %d (covered %v)", x, y, after[y][x], want, covered)
			}
		}
	}
	// Level sets every covered cell to the first centre's height plus value:
	// 64 + 30 on a new map.
	must("paint", map[string]any{"plane": "height", "mode": "level", "value": 30, "radius": 2, "cells": [][]int{{20, 30}, {24, 30}}})
	rows, _, _ := readHeights(t, must, 18, 30, 26, 30)
	for x, h := range rows[0] {
		if h != 94 {
			t.Fatalf("level cell %d,30 is %d, want 94", 18+x, h)
		}
	}
}

// save, save_as and key answer what the call concerns; verbose and state
// answer the whole window state.
func TestMapEditorMCPShortResults(t *testing.T) {
	e := agentEditor(t)
	tool, must := editorMCP(t, e)
	must("new_map", map[string]any{"width": 32, "height": 32})
	if r := tool("save", map[string]any{}); !r.IsError || !strings.Contains(r.Content[0].Text, "save_as") {
		t.Fatalf("save on a new map: %+v", r)
	}
	if e.PromptNotes() != nil {
		t.Fatal("save left its prompt open")
	}
	path := filepath.Join(t.TempDir(), "brief.alm")
	saved := must("save_as", map[string]any{"path": path})
	must("paint", map[string]any{"plane": "height", "value": 4, "cells": [][]int{{10, 10}}})
	again := must("save", map[string]any{})
	for _, r := range []editormcp.Result{saved, again} {
		var got map[string]any
		if err := json.Unmarshal([]byte(r.Content[0].Text), &got); err != nil {
			t.Fatal(err)
		}
		if len(r.Content[0].Text) > 1200 || got["saved"] != path || got["dirty"] != false || got["object_fields"] != nil || got["help"] != nil {
			t.Fatalf("save result (%d chars): %s", len(r.Content[0].Text), r.Content[0].Text)
		}
	}
	key := must("key", map[string]any{"name": "help"}).Content[0].Text
	full := must("key", map[string]any{"name": "help", "verbose": true}).Content[0].Text
	state := must("state", map[string]any{}).Content[0].Text
	if strings.Contains(key, `"mode_help"`) || !strings.Contains(full, `"mode_help"`) || !strings.Contains(state, `"mode_help"`) || len(key)*3 > len(full) {
		t.Fatalf("key %d chars, verbose %d, state %d", len(key), len(full), len(state))
	}
	if m := must("map", map[string]any{"x": 3, "y": 3}).Content[0].Text; strings.Contains(m, `"records"`) || !strings.Contains(m, `"record_counts"`) {
		t.Fatalf("map without verbose: %.300s", m)
	}
	if m := must("map", map[string]any{"x": 3, "y": 3, "verbose": true}).Content[0].Text; !strings.Contains(m, `"records"`) {
		t.Fatalf("map with verbose: %.300s", m)
	}
	items := must("trigger_items", map[string]any{}).Content[0].Text
	every := must("trigger_items", map[string]any{"verbose": true}).Content[0].Text
	if len(items)*10 > len(every) || !strings.Contains(items, `"hero_band"`) {
		t.Fatalf("trigger_items %d chars, verbose %d", len(items), len(every))
	}
}

// plantedError is a saved map of x whose condition names a unit the map
// does not hold: an error the save asks about.
func plantedError(t *testing.T, e *ui.MapEditor, must func(string, any) editormcp.Result) string {
	t.Helper()
	must("new_map", map[string]any{"width": 32, "height": 32, "discard": true})
	rep := commandReport(t, must("add_record", map[string]any{"kind": "Unit", "x": 10, "y": 10, "values": map[string]any{"class": 40}}))
	must("write_trigger", map[string]any{"name": "Chief", "conditions": []any{map[string]any{"item": "c.unitdead", "params": []any{rep["script_id"]}}}, "actions": []any{map[string]any{"item": "a.win"}}})
	m, err := mapedit.New(e.Document().SourceBytes)
	if err != nil {
		t.Fatal(err)
	}
	sc, err := mustMap(t, m.Bytes()).Script()
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
	must("open", map[string]any{"path": path, "discard": true})
	return path
}

// save_as writes into the source map's folder and answers no error; a map
// the errors question stops answers the question, not an error; a missing
// folder is named and leaves the window free for the next call.
func TestMapEditorMCPSaveAsAnswers(t *testing.T) {
	_, e, source := openTestMapEditor(t, inspectionFixtureBytes(t))
	tool, must := editorMCP(t, e)
	must("paint", map[string]any{"plane": "height", "value": 6, "radius": 2, "cells": [][]int{{8, 24}}})
	target := filepath.Join(filepath.Dir(source), "other.alm")
	got := commandReport(t, must("save_as", map[string]any{"path": target}))
	if _, err := os.Stat(target); err != nil || got["saved"] != target {
		t.Fatalf("save_as beside the source: %v %v", err, got)
	}
	if r := tool("save_as", map[string]any{"path": target}); !r.IsError || !strings.Contains(r.Content[0].Text, "Save writes over") {
		t.Fatalf("save_as onto the map's own file: %+v", r)
	}
	missing := filepath.Join(t.TempDir(), "missing", "c.alm")
	r := tool("save_as", map[string]any{"path": missing})
	if !r.IsError || !strings.Contains(r.Content[0].Text, "does not exist") || !strings.Contains(r.Content[0].Text, filepath.Dir(missing)) {
		t.Fatalf("missing folder: %+v", r)
	}
	must("paint", map[string]any{"plane": "height", "value": 2, "cells": [][]int{{14, 14}}})

	e = agentEditor(t)
	tool, must = editorMCP(t, e)
	plantedError(t, e, must)
	must("paint", map[string]any{"plane": "height", "value": 4, "radius": 1, "cells": [][]int{{12, 12}}})
	written := filepath.Join(t.TempDir(), "x.alm")
	asked := commandReport(t, must("save_as", map[string]any{"path": written}))
	if _, err := os.Stat(written); err == nil || asked["saved"] != "" || asked["report_asks"] != true {
		t.Fatalf("save_as on a map with errors: %v", asked)
	}
	answered := commandReport(t, must("key", map[string]any{"name": "enter"}))
	if _, err := os.Stat(written); err != nil || answered["saved"] != written {
		t.Fatalf("enter on the errors question: %v %v", err, answered)
	}
}

// mcpTogether sends calls to e in one write, before any answer, and
// answers them in the order the server sent its answers.
func mcpTogether(t *testing.T, e *ui.MapEditor, calls ...map[string]any) []editormcp.Result {
	t.Helper()
	server := e.EnableMCP(0)
	inRead, inWrite := io.Pipe()
	outRead, outWrite := io.Pipe()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	done := make(chan error, 1)
	go func() { done <- server.Serve(ctx, inRead, outWrite) }()
	defer func() {
		e.CloseMCP()
		cancel()
		_ = inWrite.Close()
		_ = outWrite.Close()
		_ = outRead.Close()
		<-done
	}()
	var batch []byte
	for i, c := range calls {
		b, err := json.Marshal(map[string]any{"jsonrpc": "2.0", "id": i + 1, "method": "tools/call", "params": c})
		if err != nil {
			t.Fatal(err)
		}
		batch = append(append(batch, b...), '\n')
	}
	go func() { _, _ = inWrite.Write(batch) }()
	lines := make(chan []byte, len(calls))
	go func() {
		scan := bufio.NewReader(outRead)
		for range calls {
			b, err := scan.ReadBytes('\n')
			if err != nil {
				return
			}
			lines <- b
		}
	}()
	var out []editormcp.Result
	for len(out) < len(calls) {
		select {
		case b := <-lines:
			var rsp struct {
				ID     int
				Result editormcp.Result
			}
			if err := json.Unmarshal(b, &rsp); err != nil || rsp.ID != len(out)+1 {
				t.Fatalf("answer %d: %s", len(out)+1, b)
			}
			out = append(out, rsp.Result)
		case <-ctx.Done():
			t.Fatal(ctx.Err())
		default:
			if err := e.Update(); err != nil {
				t.Fatal(err)
			}
			e.CompleteMCPFrame(func() (image.Image, error) { return image.NewRGBA(image.Rect(0, 0, 1280, 800)), nil })
			runtime.Gosched()
		}
	}
	return out
}

// Calls sent together run one after another in arrival order, each through
// its own route; none is refused because another runs.
func TestMapEditorMCPCallsSentTogether(t *testing.T) {
	e := agentEditor(t)
	if r := mcpTogether(t, e, map[string]any{"name": "new_map", "arguments": map[string]any{"width": 32, "height": 32}}); r[0].IsError {
		t.Fatal(r[0].Content[0].Text)
	}
	got := mcpTogether(t, e,
		map[string]any{"name": "write_trigger", "arguments": map[string]any{"name": "Talk", "actions": []any{map[string]any{"item": "a.dialog", "params": []any{0}}}}},
		map[string]any{"name": "write_message", "arguments": map[string]any{"action": 1, "parts": []any{map[string]any{"body": "Hello"}}}},
		map[string]any{"name": "paint", "arguments": map[string]any{"plane": "height", "value": 3, "cells": [][]int{{9, 9}}}},
	)
	for i, r := range got {
		if r.IsError {
			t.Fatalf("call %d: %s", i+1, r.Content[0].Text)
		}
	}
	if page := commandReport(t, got[1])["page"]; page == nil {
		t.Fatalf("write_message wrote no page: %s", got[1].Content[0].Text)
	}
}

// write_trigger writes a dialogue in one call: the trigger and its new page
// are one edit, through the message builder write_message uses.
func TestMapEditorMCPDialogueInOneCall(t *testing.T) {
	e := agentEditor(t)
	tool, must := editorMCP(t, e)
	must("new_map", map[string]any{"width": 32, "height": 32})
	rep := commandReport(t, must("write_trigger", map[string]any{"name": "Talk", "actions": []any{
		map[string]any{"item": "a.dialog", "parts": []any{map[string]any{"body": "Hello"}, map[string]any{"body": "Again"}}}}}))
	page, _ := rep["page"].(float64)
	record, _ := rep["record"].(map[string]any)
	if page <= 0 || record["Kind"] != "Trigger" || record["Index"] != 0.0 {
		t.Fatalf("dialogue report: %v", rep)
	}
	var parts struct{ Parts []struct{ Body string } }
	if err := json.Unmarshal([]byte(must("messages", map[string]any{"page": page}).Content[0].Text), &parts); err != nil || len(parts.Parts) != 2 || parts.Parts[1].Body != "Again" {
		t.Fatalf("page %v parts %+v", page, parts)
	}
	must("undo", map[string]any{})
	if e.Document().Script != nil && len(e.Document().Script.Triggers) != 0 {
		t.Fatal("one undo left the dialogue trigger")
	}
	for _, bad := range []map[string]any{
		{"name": "T", "actions": []any{map[string]any{"item": "a.win", "parts": []any{map[string]any{"body": "x"}}}}},
		{"name": "T", "actions": []any{map[string]any{"item": "a.dialog", "params": []any{0}, "parts": []any{map[string]any{"body": "x"}}}}},
	} {
		if r := tool("write_trigger", bad); !r.IsError {
			t.Fatalf("write_trigger took %v", bad)
		}
	}
}

// The message list holds every page of a map with 14 pages on disk.
func TestMapEditorMCPMessageListHoldsEveryPage(t *testing.T) {
	e := agentEditor(t)
	_, must := editorMCP(t, e)
	must("new_map", map[string]any{"width": 32, "height": 32})
	must("write_trigger", map[string]any{"name": "Talk", "actions": []any{map[string]any{"item": "a.dialog", "params": []any{0}}}})
	dir := t.TempDir()
	path := filepath.Join(dir, "m.alm")
	must("save_as", map[string]any{"path": path})
	if err := os.MkdirAll(filepath.Join(dir, "m.text"), 0700); err != nil {
		t.Fatal(err)
	}
	for i := 1; i <= 14; i++ {
		if err := os.WriteFile(filepath.Join(dir, "m.text", messageFilename(uint32(i))), []byte(fmt.Sprintf("<part=1>\nPage %d\n", i)), 0600); err != nil {
			t.Fatal(err)
		}
	}
	must("open", map[string]any{"path": path, "discard": true})
	var list []struct{ Page int }
	if err := json.Unmarshal([]byte(must("messages", map[string]any{}).Content[0].Text), &list); err != nil || len(list) != 14 {
		t.Fatalf("messages lists %d pages: %v", len(list), err)
	}
	var rec struct {
		Fields []struct {
			Key     string
			Choices []struct{ Value int }
		}
	}
	if err := json.Unmarshal([]byte(must("record", map[string]any{"kind": "Action", "index": 1, "verbose": true}).Content[0].Text), &rec); err != nil {
		t.Fatal(err)
	}
	pages := 0
	for _, f := range rec.Fields {
		if f.Key == "value0" {
			for _, c := range f.Choices {
				if c.Value >= 1 && c.Value <= 14 {
					pages++
				}
			}
		}
	}
	if pages != 14 {
		t.Fatalf("the dialogue action's page field offers %d of 14 pages", pages)
	}
}

// A unit whose placement sets a definition is named with it in the trigger
// unit choices and in its record.
func TestMapEditorMCPUnitChoiceNamesItsDefinition(t *testing.T) {
	e := agentEditor(t)
	_, must := editorMCP(t, e)
	path := filepath.Join(t.TempDir(), "defined.alm")
	b := synth.ALM(synth.ALMOptions{Width: 32, Height: 32, Type7Payload: make([]byte, 12), Units: []synth.ALMUnit{
		{X: 10 << 8, Y: 10 << 8, ClassID: 5, DefID: 1234, UnitID: 7},
		{X: 12 << 8, Y: 10 << 8, ClassID: 5, DefID: 999, UnitID: 8},
		{X: 14 << 8, Y: 10 << 8, ClassID: 5, UnitID: 9},
	}})
	if err := os.WriteFile(path, b, 0600); err != nil {
		t.Fatal(err)
	}
	must("open", map[string]any{"path": path, "discard": true})
	var items struct{ Units []ui.InspectionChoice }
	if err := json.Unmarshal([]byte(must("trigger_items", map[string]any{}).Content[0].Text), &items); err != nil {
		t.Fatal(err)
	}
	want := map[int64]string{7: "; definition: Humans row 1, row", 8: "; definition 999: no Humans row", 9: ""}
	for _, c := range items.Units {
		suffix, ok := want[c.Value]
		if !ok {
			continue
		}
		if suffix == "" && strings.Contains(c.Label, "definition") || suffix != "" && !strings.HasSuffix(c.Label, suffix) {
			t.Fatalf("unit %d choice %q, want suffix %q", c.Value, c.Label, suffix)
		}
		delete(want, c.Value)
	}
	if len(want) != 0 {
		t.Fatalf("no choice for units %v in %+v", want, items.Units)
	}
	var rec struct{ Record ui.InspectionRecord }
	if err := json.Unmarshal([]byte(must("record", map[string]any{"kind": "Unit", "index": 0}).Content[0].Text), &rec); err != nil || rec.Record.Definition != "definition: Humans row 1, row" {
		t.Fatalf("unit record definition %q: %v", rec.Record.Definition, err)
	}
}

func TestMapEditorMCPSaveLeavesAnOpenPrompt(t *testing.T) {
	promptState := func(must func(string, any) editormcp.Result) (bool, string) {
		var s map[string]any
		if err := json.Unmarshal([]byte(must("state", map[string]any{}).Content[0].Text), &s); err != nil {
			t.Fatal(err)
		}
		open, _ := s["prompt_open"].(bool)
		text, _ := s["prompt_text"].(string)
		return open, text
	}
	e := agentEditor(t)
	tool, must := editorMCP(t, e)
	must("new_map", map[string]any{"width": 32, "height": 32})
	if err := e.Key("saveas"); err != nil {
		t.Fatal(err)
	}
	e.TextInput("typed-by-the-person")
	if open, text := promptState(must); !open || text != "typed-by-the-person" {
		t.Fatalf("typed prompt: open %v, text %q", open, text)
	}
	r := tool("save", map[string]any{})
	if !r.IsError || !strings.Contains(r.Content[0].Text, "prompt") || strings.Contains(r.Content[0].Text, "no file of its own") {
		t.Fatalf("save over an open prompt: %+v", r)
	}
	if open, text := promptState(must); !open || text != "typed-by-the-person" {
		t.Fatalf("save changed the open prompt: open %v, text %q", open, text)
	}

	n := agentEditor(t)
	ntool, nmust := editorMCP(t, n)
	nmust("new_map", map[string]any{"width": 32, "height": 32})
	if r := ntool("save", map[string]any{}); !r.IsError || !strings.Contains(r.Content[0].Text, "save_as") {
		t.Fatalf("save on a map with no file: %+v", r)
	}
	if open, _ := promptState(nmust); open {
		t.Fatal("save left the prompt it opened")
	}
}

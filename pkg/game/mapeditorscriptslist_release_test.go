package game

import (
	"bytes"
	"image"
	"os"
	"path/filepath"
	"sort"
	"testing"

	"againrom/pkg/formats/alm"
	"againrom/pkg/ui"
)

func expectedScriptSections(d *ui.InspectionDocument, sc alm.Script) [4][]int {
	var out [4][]int
	used := [2]map[uint32]bool{{}, {}}
	registers := map[uint32]bool{}
	for _, tr := range sc.Triggers {
		for _, id := range append(append([]uint32{}, tr.Left[:]...), tr.Right[:]...) {
			used[0][id] = id != 0
		}
		for _, id := range tr.Acts {
			used[1][id] = id != 0
		}
	}
	for family, nodes := range [][]alm.ScriptNode{sc.Conditions, sc.Actions} {
		for _, n := range nodes {
			if family == 0 && n.Opcode == 19 || family == 1 && (n.Opcode == 3 || n.Opcode == 8) {
				registers[scriptVariable(n)] = true
			}
		}
	}
	for n := range sc.Triggers {
		out[0] = append(out[0], recordOf(d, "Trigger", n))
	}
	var ordered []uint32
	for register := range registers {
		ordered = append(ordered, register)
	}
	sort.Slice(ordered, func(i, j int) bool { return ordered[i] < ordered[j] })
	for i := range ordered {
		out[1] = append(out[1], -2-i)
	}
	for n, node := range sc.Conditions {
		if node.Opcode == 0x10002 && registers[uint32(n)] {
			continue
		} else if !used[0][node.ID] {
			out[2] = append(out[2], recordOf(d, "Condition", n))
		}
	}
	for n, node := range sc.Actions {
		if !used[1][node.ID] {
			out[3] = append(out[3], recordOf(d, "Action", n))
		}
	}
	return out
}

func scriptsListRow(t *testing.T, e *ui.MapEditor, row int) image.Rectangle {
	t.Helper()
	for pages := 0; pages < 200; pages++ {
		if r, ok := e.ListRowRect(row); ok {
			return r
		}
		_ = e.Key("pagedown")
	}
	t.Fatalf("Scripts row %d is not reachable", row)
	return image.Rectangle{}
}

func TestReleaseMapEditorScriptsListUntouchedBytes(t *testing.T) {
	x := installedInspector(t)
	keys, sources := installedMapBytes(t, x)
	totals := [4]int{}
	popups, variableMaps := 0, 0
	for n, key := range keys {
		t.Run(key, func(t *testing.T) {
			e := x.Editor()
			e.Layout(1280, 800)
			if err := e.Open(key); err != nil {
				t.Fatal(err)
			}
			closeMapPanel(t, e)
			m, err := alm.Open(sources[n])
			if err != nil {
				t.Fatal(err)
			}
			sc, err := m.Script()
			if err != nil {
				t.Fatal(err)
			}
			sections := expectedScriptSections(e.Document(), sc)
			if len(sections[1]) > 0 {
				variableMaps++
			}
			_ = e.Key("scripts")
			if e.Selected() != -1 {
				_ = e.Key("escape")
			}
			_, _ = e.ListRowRect(0)
			_ = e.Key("home")
			wantRows := 4
			for _, ids := range sections {
				wantRows += len(ids)
			}
			if got := e.ListRowCount(); got != wantRows {
				t.Fatalf("list rows=%d, want %d", got, wantRows)
			}
			row, counts := 0, [4]int{}
			for section, records := range sections {
				counts[section], totals[section] = len(records), totals[section]+len(records)
				row++
				for _, id := range records {
					r := scriptsListRow(t, e, row)
					trigClick(t, e, r, true, "Scripts row")
					if e.Selected() != id {
						t.Fatalf("row %d selected %d, want %d", row, e.Selected(), id)
					}
					if _, ok := e.PopupButtonRect(0); !ok {
						t.Fatalf("row %d did not open its popup", row)
					}
					_ = e.Key("escape")
					popups++
					row++
				}
			}
			_ = e.Key("down")
			_ = e.Key("up")
			if e.Dirty() || !bytes.Equal(e.Document().SourceBytes, sources[n]) {
				t.Fatal("opening, navigating or cancelling changed the map")
			}
			_, _, saved := areaSaveAs(t, e, "untouched.alm")
			if !bytes.Equal(saved, sources[n]) {
				t.Fatal("Save As changed untouched installed bytes")
			}
			t.Logf("headings Triggers=%d Variables=%d Conditions=%d Actions=%d; saved byte-equal (%d bytes)", counts[0], counts[1], counts[2], counts[3], len(saved))
		})
	}
	t.Logf("root=%s maps=%d Triggers=%d Variables=%d variableMaps=%d Conditions=%d Actions=%d popups=%d", os.Getenv("AGAINROM_ASSETS"), len(keys), totals[0], totals[1], variableMaps, totals[2], totals[3], popups)
}

func TestReleaseMapEditorScriptsListFrames(t *testing.T) {
	x := installedInspector(t)
	dir, lang := os.Getenv("AGAINROM_EDITOR_FRAMES"), installLanguage(x)
	e := x.Editor()
	e.Layout(1280, 800)
	if err := e.Open("scenario/151.alm"); err != nil {
		t.Fatal(err)
	}
	closeMapPanel(t, e)
	_ = e.Key("scripts")
	_ = e.Key("escape")
	row := 2 + len(e.Document().Script.Triggers)
	r := scriptsListRow(t, e, row)
	writeEditorFrame(t, e, filepath.Join(dir, "scripts-variables-list-"+lang+".png"), dir != "", image.Rectangle{})
	trigClick(t, e, r, true, "variable row")
	writeEditorFrame(t, e, filepath.Join(dir, "scripts-variable-popup-"+lang+".png"), dir != "", image.Rectangle{})
	if e.Dirty() || e.Selected() != -2 {
		t.Fatal("variable frame did not open a read-only variable")
	}
	for _, key := range editorKeys(x) {
		e := x.Editor()
		e.Layout(1280, 800)
		if err := e.Open(key); err != nil {
			t.Fatal(err)
		}
		m, err := alm.Open(e.Document().SourceBytes)
		if err != nil {
			t.Fatal(err)
		}
		sc, err := m.Script()
		if err != nil {
			t.Fatal(err)
		}
		sections := expectedScriptSections(e.Document(), sc)
		count := len(sections[0]) + len(sections[1]) + len(sections[2]) + len(sections[3])
		if len(sections[0]) == 0 || len(sections[2]) == 0 || len(sections[3]) == 0 || count > 14 {
			continue
		}
		closeMapPanel(t, e)
		_ = e.Key("scripts")
		_ = e.Key("escape")
		shot := func(name string) {
			writeEditorFrame(t, e, filepath.Join(dir, name+"-"+lang+".png"), dir != "", image.Rectangle{})
		}
		shot("scripts-list")
		_ = e.Key("home")
		r := scriptsListRow(t, e, 1)
		trigClick(t, e, r, true, "trigger row")
		shot("scripts-trigger")
		_ = e.Key("escape")
		_ = e.Key("player")
		if _, ok := e.PromptRect(); !ok {
			t.Fatal("Scripts A did not open Add")
		}
		shot("scripts-add")
		_ = e.Key("escape")
		if e.Dirty() {
			t.Fatal("frames changed the map")
		}
		t.Logf("all four headings: %s counts=%v", key, [4]int{len(sections[0]), len(sections[1]), len(sections[2]), len(sections[3])})
		return
	}
	t.Fatal("no installed map fits all four headings with trigger, condition and action rows")
}

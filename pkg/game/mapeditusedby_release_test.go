package game

import (
	"bytes"
	"fmt"
	"image"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"

	"againrom/internal/firstgame"
	"againrom/pkg/ui"
)

func expectedObjectUses(d *ui.InspectionDocument, kind, value uint32) []ui.InspectionScriptUse {
	var out []ui.InspectionScriptUse
	m := d.Script
	for family, nodes := range [][]ui.InspectionScriptNode{m.Conditions, m.Actions} {
		for _, node := range nodes {
			for slot, v := range node.Values {
				if scriptParameterKind(family == 0, node.Opcode, node.Types, slot) != kind || v != value || m.OpenTarget(kind, v) == nil {
					continue
				}
				use := ui.InspectionScriptUse{Node: node.Record, Slot: slot}
				for _, trigger := range m.Triggers {
					ids := trigger.Acts[:]
					if family == 0 {
						ids = append(append([]uint32{}, trigger.Left[:]...), trigger.Right[:]...)
					}
					if node.ID != 0 && slices.Contains(ids, node.ID) {
						use.Triggers = append(use.Triggers, trigger.Record)
					}
				}
				out = append(out, use)
			}
		}
	}
	return out
}

func expectedUseLinks(m *ui.InspectionScriptModel, uses []ui.InspectionScriptUse) []int {
	var out []int
	for _, trigger := range m.Triggers {
		if slices.ContainsFunc(uses, func(u ui.InspectionScriptUse) bool { return slices.Contains(u.Triggers, trigger.Record) }) {
			out = append(out, trigger.Record)
		}
	}
	for _, use := range uses {
		if len(use.Triggers) == 0 && !slices.Contains(out, use.Node) {
			out = append(out, use.Node)
		}
	}
	return out
}

func popupUseLinks(e *ui.MapEditor) []int {
	var out []int
	for _, id := range e.PopupRecordLinks() {
		if id >= 0 {
			kind := e.Document().Records[id].Kind
			if kind == "Trigger" || kind == "Action" || kind == "Condition" {
				out = append(out, id)
			}
		}
	}
	return out
}

func openUsedByGroup(t *testing.T, e *ui.MapEditor, row int) {
	t.Helper()
	r, ok := e.BarRect("groups")
	trigClick(t, e, r, ok, "Groups")
	r = scriptsListRow(t, e, row)
	trigClick(t, e, r, true, "group")
}

func TestReleaseMapEditorUsedByCensusPreservesBytes(t *testing.T) {
	firstgame.SkipOnSecondGame(t)
	x := installedInspector(t)
	totals, rows, unattached, popupTotals := map[uint32]int{}, map[uint32]int{}, map[uint32]int{}, map[uint32]int{}
	maps, popups := 0, 0
	keys, sources := installedMapBytes(t, x)
	for index, key := range keys {
		before := bytes.Clone(sources[index])
		e := x.Editor()
		if err := e.Open(key); err != nil {
			t.Fatal(key, err)
		}
		e.Layout(1280, 800)
		closeMapPanel(t, e)
		d := e.Document()
		if !bytes.Equal(d.SourceBytes, before) {
			t.Fatal(key, "Open changed installed ALM bytes")
		}
		m := d.Script
		counts, links, popupLinks := map[uint32]int{}, map[uint32]int{}, map[uint32]int{}
		if m != nil {
			for _, kind := range []uint32{4, 9, 2, 3} {
				for _, target := range m.Targets[kind] {
					if m.OpenTarget(kind, target.Value) == nil {
						continue
					}
					want := expectedObjectUses(d, kind, target.Value)
					if !reflect.DeepEqual(target.Uses, want) {
						t.Fatalf("%s kind%d value%d reverse=%+v forward=%+v", key, kind, target.Value, target.Uses, want)
					}
					counts[kind] += len(want)
					wantLinks := expectedUseLinks(m, want)
					links[kind] += len(wantLinks)
					for _, id := range wantLinks {
						if d.Records[id].Kind != "Trigger" {
							unattached[kind]++
						}
					}
				}
			}
		}
		for id, record := range d.Records {
			if record.Kind != "Unit" && record.Kind != "Structure" && record.Kind != "Player" {
				continue
			}
			e.Select(id, false)
			var want []int
			if m != nil {
				for _, kind := range []uint32{4, 9, 3} {
					for _, target := range m.Targets[kind] {
						if target.Record == id {
							want = expectedUseLinks(m, expectedObjectUses(d, kind, target.Value))
						}
					}
				}
			}
			got := popupUseLinks(e)
			if !slices.Equal(got, want) {
				t.Fatalf("%s %s[%d] popup links=%v want=%v", key, record.Kind, record.Index, got, want)
			}
			popupLinks[map[string]uint32{"Unit": 4, "Structure": 9, "Player": 3}[record.Kind]] += len(got)
			popups++
		}
		r, ok := e.BarRect("groups")
		trigClick(t, e, r, ok, "Groups")
		groups := e.ListRowCount()
		_ = e.Key("escape")
		for group := 0; group < groups; group++ {
			openUsedByGroup(t, e, group)
			members := e.GroupMembers()
			if len(members) == 0 {
				t.Fatal(key, "empty group popup")
			}
			value := uint32(d.Records[members[0]].Group)
			var want []int
			if m != nil {
				want = expectedUseLinks(m, expectedObjectUses(d, 2, value))
			}
			got := popupUseLinks(e)
			if !slices.Equal(got, want) {
				t.Fatalf("%s group%d popup links=%v want=%v", key, value, got, want)
			}
			popupLinks[2] += len(got)
			popups++
		}
		if !bytes.Equal(before, e.Document().SourceBytes) || e.Dirty() {
			t.Fatal(key, "opening popups wrote ALM")
		}
		path := filepath.Join(t.TempDir(), "untouched.alm")
		if _, err := e.Document().Edits.SaveAs(path); err != nil {
			t.Fatal(key, err)
		}
		saved, err := os.ReadFile(path)
		if err != nil || !bytes.Equal(saved, before) {
			t.Fatal(key, "Save As differs", err)
		}
		for _, kind := range []uint32{4, 9, 2, 3} {
			totals[kind] += counts[kind]
			rows[kind] += links[kind]
			popupTotals[kind] += popupLinks[kind]
		}
		t.Logf("map=%s parameters Unit=%d Structure=%d Group=%d Player=%d; target-links Unit=%d Structure=%d Group=%d Player=%d; popup-links Unit=%d Structure=%d Group=%d Player=%d; SaveAs byte-equal bytes=%d", key, counts[4], counts[9], counts[2], counts[3], links[4], links[9], links[2], links[3], popupLinks[4], popupLinks[9], popupLinks[2], popupLinks[3], len(saved))
		maps++
	}
	for _, kind := range []uint32{4, 9, 2, 3} {
		if totals[kind] == 0 {
			t.Errorf("kind%d has no installed subject", kind)
		}
		t.Logf("root=%s kind=%s parameters=%d target-links=%d popup-links=%d trigger-links=%d unattached-links=%d", os.Getenv("AGAINROM_ASSETS"), map[uint32]string{4: "Unit", 9: "Structure", 2: "Group", 3: "Player"}[kind], totals[kind], rows[kind], popupTotals[kind], rows[kind]-unattached[kind], unattached[kind])
	}
	t.Logf("root=%s maps=%d popups=%d", os.Getenv("AGAINROM_ASSETS"), maps, popups)
}

func TestReleaseMapEditorUsedByFrames(t *testing.T) {
	firstgame.SkipOnSecondGame(t)
	x := installedInspector(t)
	dir := os.Getenv("AGAINROM_USED_BY_FRAMES")
	for _, key := range installedMapKeys(x) {
		e := x.Editor()
		if err := e.Open(key); err != nil {
			t.Fatal(err)
		}
		e.Layout(1280, 800)
		closeMapPanel(t, e)
		if !strings.HasPrefix(key, "scenario/") || e.Document().Script == nil {
			continue
		}
		for _, target := range e.Document().Script.Targets[4] {
			want := expectedUseLinks(e.Document().Script, target.Uses)
			if target.Record < 0 || len(want) == 0 || e.Document().Records[want[0]].Kind != "Trigger" {
				continue
			}
			e.Select(target.Record, true)
			row := slices.Index(e.PopupRecordLinks(), want[0])
			if row < 0 {
				t.Fatal("missing Used by link")
			}
			r, ok := e.PopupRowRect(row)
			for k := 0; !ok && k < 50; k++ {
				e.ScrollDetails(1)
				r, ok = e.PopupRowRect(row)
			}
			if !ok {
				t.Fatal("Used by row never visible")
			}
			e.ScrollDetails(3)
			r, ok = e.PopupRowRect(row)
			if !ok {
				t.Fatal("Used by link scrolled out of view")
			}
			if dir != "" {
				if err := os.MkdirAll(dir, 0o755); err != nil {
					t.Fatal(err)
				}
			}
			writeEditorFrame(t, e, filepath.Join(dir, "used-by-unit.png"), dir != "", image.Rectangle{})
			trigClick(t, e, r, true, "Used by")
			if e.Selected() != want[0] {
				t.Fatal("wrong trigger")
			}
			writeEditorFrame(t, e, filepath.Join(dir, "used-by-trigger.png"), dir != "", image.Rectangle{})
			t.Logf("root=%s map=%s unit=%d trigger-record=%d", os.Getenv("AGAINROM_ASSETS"), key, target.Value, want[0])
			if dir != "" {
				index := fmt.Sprintf("# Used by frames\n\nRoot: `%s`. Map: `%s`. Unit: %d. Trigger record: %d.\n\n- `used-by-unit.png`: unit popup, Used by section.\n- `used-by-trigger.png`: trigger opened by its first Used by link.\n", os.Getenv("AGAINROM_ASSETS"), key, target.Value, want[0])
				if err := os.WriteFile(filepath.Join(dir, "INDEX.md"), []byte(index), 0o644); err != nil {
					t.Fatal(err)
				}
			}
			return
		}
	}
	t.Fatal("no installed campaign unit with a trigger link")
}

package game

import (
	"fmt"
	"image"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"againrom/internal/firstgame"
	"againrom/pkg/ui"
)

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
			r, ok := e.FindControl(ui.ControlQuery{Role: ui.ControlPopupRow, N: row})
			for k := 0; !ok && k < 50; k++ {
				e.ScrollDetails(1)
				r, ok = e.FindControl(ui.ControlQuery{Role: ui.ControlPopupRow, N: row})
			}
			if !ok {
				t.Fatal("Used by row never visible")
			}
			from := r
			for step := 0; step < 3; step++ {
				e.ScrollDetails(1)
				if _, ok = e.FindControl(ui.ControlQuery{Role: ui.ControlPopupRow, N: row}); !ok {
					e.ScrollDetails(-1)
					break
				}
			}
			r, ok = e.FindControl(ui.ControlQuery{Role: ui.ControlPopupRow, N: row})
			if !ok || r.Min.Y >= from.Min.Y {
				t.Fatal("Used by link did not stay in view of a scrolled popup")
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

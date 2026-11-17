package ui

import (
	"fmt"
	"image"
	"slices"
	"strings"
	"testing"
)

// In the unit window, in both views and at three sizes, every Used by label
// stays inside its place and a press at the middle of a Used by row opens
// the object that row names.
func TestUnitWindowUsedByLinksFitAndOpenTheirOwnObject(t *testing.T) {
	for _, size := range []image.Point{{640, 480}, {800, 600}, {1920, 1080}} {
		for _, plain := range []bool{false, true} {
			name := fmt.Sprintf("%v plain=%v", size, plain)
			e, host := popupEditor(t)
			_ = e.Open("popups")
			e.doc.Edits = &personSettingsProbe{&settingsPreviewProbe{popupHost: host}}
			m := &InspectionScriptModel{Targets: map[uint32][]InspectionScriptTarget{scriptUnit: {{Record: 0}}}}
			var records []int
			for i := range 7 {
				id := len(e.doc.Records)
				e.doc.Records = append(e.doc.Records, InspectionRecord{Kind: KindAction, Index: i})
				m.Actions = append(m.Actions, InspectionScriptNode{Record: id, Label: strings.Repeat("Give the healing potion to the guard ", 3)})
				target := &m.Targets[scriptUnit][0]
				target.Uses = append(target.Uses, InspectionScriptUse{Node: id})
				records = append(records, id)
			}
			e.doc.Script = m
			e.props.plain = plain
			e.Layout(size.X, size.Y)
			// open shows the unit with its Used by fold open, scrolled by top.
			open := func(top int) (popupSpec, []int, popupLayout) {
				e.Select(0, false)
				e.objectSpec()
				e.obj.usesTurned = true
				e.ScrollDetails(top)
				s, links, _ := e.objectSpec()
				return s, links, layoutPopup(s, e.font, e.bodyFont())
			}
			if open(0); !e.nativeUnitWindow() {
				t.Fatalf("%s: the person window is not open", name)
			}
			done := map[int]bool{}
			for top := 0; top < 100; top++ {
				s, links, l := open(top)
				for _, r := range l.Rows {
					if s.Rows[r.Index].Kind != popupLink || r.Index >= len(links) || links[r.Index] >= 0 || done[r.Index] {
						continue
					}
					done[r.Index] = true
					if w := e.bodyFont().Advance(s.Rows[r.Index].Label); w > r.Rect.Dx() {
						t.Errorf("%s: row %d label is %d wide in a %d wide place", name, r.Index, w, r.Rect.Dx())
					}
					want := -links[r.Index] - 1
					if !slices.Contains(records, want) {
						t.Fatalf("%s: row %d names record %d, not a use", name, r.Index, want)
					}
					chromeClick(e, center(r.Rect))
					if e.selected != want {
						t.Errorf("%s: a press on row %d opened record %d, want %d", name, r.Index, e.selected, want)
					}
					s, links, l = open(top)
					break
				}
			}
			if len(done) < len(records) {
				t.Errorf("%s: %d Used by rows reached, want at least %d", name, len(done), len(records))
			}
		}
	}
}

package ui

import (
	"fmt"
	"image"
	"slices"
	"strings"
	"testing"
)

// shownRows walks the window from its first row to its last through the
// popup's own scroll, one row a step, and calls visit at each place.
func shownRows(e *MapEditor, visit func(s popupSpec, l popupLayout)) {
	e.objectTop = 0
	last := -1
	for range 400 {
		s, _, ok := e.objectSpec()
		if !ok {
			return
		}
		visit(s, layoutPopup(s, e.font, e.bodyFont()))
		if e.objectTop == last {
			return
		}
		last = e.objectTop
		e.objectTop++
	}
}

// openBlocks opens each closed block of the plain view by a press on its
// row, scrolled to through the popup's scroll.
func openBlocks(t *testing.T, e *MapEditor) {
	t.Helper()
	for range 20 {
		pressed := false
		shownRows(e, func(s popupSpec, l popupLayout) {
			if pressed {
				return
			}
			for _, rr := range l.Rows {
				id, ok := e.obj.rowBlock[rr.Index]
				if ok && !e.blockOpen(plainBlock{id: id, fold: true}) {
					viewsClick(t, e, rr.Rect, true, "block "+id)
					pressed = true
					return
				}
			}
		})
		if !pressed {
			return
		}
	}
}

// reachedFields names each field of the shown window that a scroll brings
// on screen: an edited field by its control's rectangle, a read-only field
// by its first line.
func reachedFields(e *MapEditor, reached map[string]bool) {
	shownRows(e, func(s popupSpec, l popupLayout) {
		shown := map[int]image.Rectangle{}
		labels := map[string]bool{}
		for _, rr := range l.Rows {
			shown[rr.Index] = rr.Control
			labels[strings.TrimSpace(s.Rows[rr.Index].Label)] = true
		}
		for i, f := range e.obj.fields {
			if f.ReadOnly && len(f.Lines) > 0 {
				first := editorTextLines(e.bodyFont(), f.Lines[0].Text, 2000)[0]
				for label := range labels {
					if label != "" && strings.HasPrefix(first, label) {
						reached[f.Key] = true
					}
				}
				continue
			}
			if r, ok := shown[e.fieldRow(i)]; ok && !r.Empty() && r.In(l.Rect) {
				reached[f.Key] = true
			}
		}
	})
}

// Every field of a human, a structure and a sack is in reach in the right
// panel at its least, default and greatest width and floating, in both
// views and both languages: after its tabs, blocks and slot details open,
// the popup's scroll brings each field's control or line on screen.
func TestMapEditorWindowReachesEveryField(t *testing.T) {
	for _, lang := range []string{"en", "ru"} {
		for _, width := range wideWidths {
			for _, view := range []string{"plain", "technical"} {
				for _, record := range []int{viewsUnit, viewsStructure, viewsSack} {
					e := wideEditor(t, lang, width, view, nil)
					basicUnit := view == "plain" && record == viewsUnit
					if basicUnit {
						// The unit's basic view is laid out for the least
						// screen the editor takes.
						e.Layout(1920, 1080)
					}
					e.Select(record, false)
					e.objectSpec()
					blocks := e.PlainFieldBlocks()
					reached := map[string]bool{}
					for _, section := range wideSections(e) {
						openSection(t, e, section)
						openBlocks(t, e)
						reachedFields(e, reached)
						for _, f := range e.obj.fields {
							if f.Parent == "" || reached[f.Key] {
								continue
							}
							if parent := slices.IndexFunc(e.obj.fields, func(p InspectionField) bool { return p.Key == f.Parent }); parent >= 0 && reached[f.Parent] {
								// The slot's details show once its row is chosen.
								e.focusObjectField(parent)
								reachedFields(e, reached)
								e.obj.focus = -1
							}
						}
					}
					var missed []string
					for _, f := range e.obj.fields {
						// A field of the unit's sack is the sack's own
						// window's. The unit's basic view reaches the
						// fields it places and no other: the rest stand in
						// the detailed view.
						if sackField(e.doc.Records[record], f) {
							continue
						}
						if basicUnit && blocks[f.Key] == "" {
							if reached[f.Key] {
								t.Errorf("%s panel %d: the basic view shows %s, which it leaves to the detailed view", lang, width, f.Key)
							}
							continue
						}
						if !reached[f.Key] {
							missed = append(missed, f.Key)
						}
					}
					if len(missed) > 0 {
						t.Errorf("%s panel %d %s %s: out of reach %v", lang, width, view, e.doc.Records[record].Kind, missed)
					}
				}
			}
		}
	}
}

// plainWordsOf are the plain view's own words a record's window shows, as
// shown: each "%d" word for the fixture's numbers, the placement word's
// line for its value and the meaning for its type and value.
func plainWordsOf(e *MapEditor, record int, placement, kind int64) []string {
	w := e.word
	switch record {
	case viewsUnit:
		var out []string
		for _, id := range []string{"position", "name", "player", "group", "kind", "tier"} {
			out = append(out, w("editor.plain."+id))
		}
		return out
	case viewsStructure:
		out := []string{w("editor.plain.position"), w("editor.plain.type"), w("editor.plain.owner"), w("editor.plain.target"),
			w("editor.plain.width"), w("editor.plain.height"), w("editor.plain.placement"), w("editor.plain.word")}
		var bits []string
		for b := 0; b < 16; b++ {
			if placement&(1<<b) != 0 {
				bits = append(bits, fmt.Sprint(b))
			}
		}
		set := w("editor.plain.bits.none")
		if len(bits) > 0 {
			set = strings.Join(bits, ", ")
		}
		out = append(out, fmt.Sprintf(w("editor.plain.bits"), placement, set))
		switch {
		case kind == 34 || kind == 35:
			out = append(out, w("editor.plain.placement.shop"))
		case placement == 0:
			out = append(out, w("editor.plain.placement.zero"))
		default:
			out = append(out, w("editor.plain.placement.health"))
		}
		return out
	}
	out := []string{w("editor.plain.position"), w("editor.plain.gold"), w("editor.plain.additem"), w("editor.plain.details")}
	for n := 1; n <= 3; n++ {
		for _, id := range []string{"item", "code", "recipe"} {
			out = append(out, fmt.Sprintf(w("editor.plain."+id), n))
		}
	}
	return out
}

// Every word of the plain view and of the view switch shows whole in the
// right panel at each width and floating, in English and Russian, in a
// font as wide as the installed ones: a label in its column, a heading in
// its band, a line of text, split by the line breaker, in its row.
func TestMapEditorPlainWordsShowWhole(t *testing.T) {
	type variant struct {
		record          int
		set             map[string]int64
		placement, kind int64
	}
	variants := []variant{
		{viewsUnit, nil, 0, 0},
		{viewsStructure, nil, 5, 51},
		{viewsStructure, map[string]int64{"Structure/placement": 0}, 0, 51},
		{viewsStructure, map[string]int64{"Structure/kind": 34}, 5, 34},
		{viewsSack, nil, 0, 0},
	}
	for _, lang := range []string{"en", "ru"} {
		for _, width := range wideWidths {
			for _, v := range variants {
				e := wideEditor(t, lang, width, "plain", v.set)
				e.Select(v.record, false)
				e.objectSpec()
				where := fmt.Sprintf("%s panel %d %s", lang, width, e.doc.Records[v.record].Kind)
				openBlocks(t, e)
				want := plainWordsOf(e, v.record, v.placement, v.kind)
				seen := map[string]bool{}
				shownRows(e, func(s popupSpec, l popupLayout) {
					run, cut := "", false
					flush := func() {
						if slices.Contains(want, run) {
							seen[run] = true
							if cut {
								t.Errorf("%s: %q is cut", where, run)
							}
						}
						run, cut = "", false
					}
					for _, rr := range l.Rows {
						row := s.Rows[rr.Index]
						label := strings.TrimSpace(strings.TrimPrefix(strings.TrimPrefix(row.Label, "+ "), "- "))
						if row.Kind == popupText && row.Value == "" {
							// A line of text: consecutive lines join into the
							// text the line breaker split.
							lineCut := e.bodyFont().Advance(row.Label) > rr.Label.Dx()-2
							run, cut = strings.TrimSpace(run+" "+label), cut || lineCut
							if !slices.ContainsFunc(want, func(w string) bool { return strings.HasPrefix(w, run) }) {
								run, cut = label, lineCut
							}
							if slices.Contains(want, run) {
								flush()
							}
							continue
						}
						flush()
						if slices.Contains(want, label) {
							seen[label] = true
							_, wholeCut := rowFull(e.font, e.bodyFont(), row, rr)
							labelCut := row.Kind != popupHeading && e.bodyFont().Advance(row.Label) > rr.Label.Dx()-2
							if s.Rows[rr.Index].Kind == popupHeading && wholeCut || labelCut {
								t.Errorf("%s: %q is cut", where, label)
							}
						}
					}
					flush()
				})
				for _, word := range want {
					if !seen[word] {
						t.Errorf("%s: %q never shows whole", where, word)
					}
				}
				// The view switch's caption fits its button inside the window.
				s, _, _ := e.objectSpec()
				l := layoutPopup(s, e.font, e.bodyFont())
				caption := e.viewCaption()
				r, ok := e.ViewSwitchRect()
				if !ok || !r.In(l.Rect) || e.bodyFont().Advance(caption)+8 > r.Dx() {
					t.Errorf("%s: the switch %q in %v does not fit the window %v", where, caption, r, l.Rect)
				}
				// It stands out of the button row, whose room the base's
				// buttons keep, and clear of the close button.
				for _, other := range append(slices.Clone(l.Buttons), l.Close) {
					if r.Overlaps(other) {
						t.Errorf("%s: the switch %v stands on the button row or the close button %v", where, r, other)
					}
				}
			}
		}
	}
}

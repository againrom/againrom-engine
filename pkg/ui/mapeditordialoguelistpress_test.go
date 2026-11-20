package ui

import (
	"fmt"
	"image"
	"slices"
	"testing"
)

// dialogueListEdits hosts one dialogue of many plain lines.
type dialogueListEdits struct {
	*speakerPortraitEdits
	lines int
}

func (d *dialogueListEdits) DialogueNames() []string { return []string{"talk"} }

func (d *dialogueListEdits) ReadDialogue(string) (DialogueData, error) {
	var data DialogueData
	for i := 1; i <= d.lines; i++ {
		n := DialogueNodeData{ID: i, Name: fmt.Sprintf("n%d", i), Text: map[string]string{"en": fmt.Sprintf("line %d", i), "ru": fmt.Sprintf("line %d", i)}}
		if i < d.lines {
			n.Next = i + 1
		}
		data.Nodes = append(data.Nodes, n)
	}
	return data, nil
}

func (d *dialogueListEdits) WriteDialogue(string, DialogueData) (*InspectionDocument, error) {
	return nil, errNoEditHost
}

func (d *dialogueListEdits) WriteDialogueFor(InspectionMessageOwner, string, DialogueData) (*InspectionDocument, error) {
	return nil, errNoEditHost
}

func (d *dialogueListEdits) RemoveDialogue(string) (*InspectionDocument, error) {
	return nil, errNoEditHost
}

func (d *dialogueListEdits) DialogueUsers(string) []string               { return nil }
func (d *dialogueListEdits) DialogueIssues(DialogueData) []DialogueIssue { return nil }
func (d *dialogueListEdits) FreeMark(DialogueData) int                   { return -1 }

// A press anywhere on a drawn line row of the Dialog window's list, grip
// included, shows that line, at every scroll position; a press outside the
// rows shows no other line.
func TestDialogueListRowTakesAPressAnywhereOnIt(t *testing.T) {
	e, _ := speakerPortraitEditor(t, "en")
	e.doc.Edits = &dialogueListEdits{speakerPortraitEdits: e.doc.Edits.(*speakerPortraitEdits), lines: 40}
	if err := e.openDialogue("talk", false, nil); err != nil {
		t.Fatal(err)
	}
	live := e.liveNumbers()
	if len(live) != 40 {
		t.Fatalf("the window lists %d lines, want 40", len(live))
	}
	tried := 0
	for _, top := range []int{0, 7, 20, 100} {
		e.objectTop = top
		s, _, _ := e.objectSpec()
		first := e.objectTop
		_ = s
		for rank := 1; rank <= len(live); rank++ {
			e.objectTop = first
			r, ok := e.DialogueListRow(rank)
			if !ok {
				continue
			}
			rr := r
			for _, y := range []int{rr.Min.Y, (rr.Min.Y + rr.Max.Y) / 2, rr.Max.Y - 1} {
				for _, x := range []int{rr.Min.X, (rr.Min.X + rr.Max.X) / 2, rr.Max.X - 1} {
					other := live[(rank)%len(live)]
					e.obj.sel, e.objectTop = other, first
					s, _, _ := e.objectSpec()
					l := layoutPopup(s, e.font, e.bodyFont())
					p := image.Pt(x, y)
					if part, _ := l.hit(p); part == popupRowHelp || part == popupRowButton {
						continue
					}
					feedClick(t, e, p)
					tried++
					if !slices.Equal(e.liveNumbers(), live) {
						t.Fatalf("top %d: a press at %v on the row of line %d moved a card", first, p, live[rank-1])
					}
					if e.obj.sel != live[rank-1] {
						t.Fatalf("top %d: a press at %v on the row of line %d shows line %d", first, p, live[rank-1], e.obj.sel)
					}
				}
			}
		}
	}
	if tried < 100 {
		t.Fatalf("only %d presses tried", tried)
	}
	e.obj.sel, e.objectTop = live[3], 0
	s, _, _ := e.objectSpec()
	l := layoutPopup(s, e.font, e.bodyFont())
	feedClick(t, e, l.Title.Min.Add(image.Pt(2, 2)))
	if e.obj.sel != live[3] {
		t.Fatalf("a press outside the rows shows line %d", e.obj.sel)
	}
}

// A pick keeps the list where it is scrolled, so the next press lands on the
// row drawn under it.
func TestDialogueListKeepsItsScrollAfterAPick(t *testing.T) {
	e, _ := speakerPortraitEditor(t, "en")
	e.doc.Edits = &dialogueListEdits{speakerPortraitEdits: e.doc.Edits.(*speakerPortraitEdits), lines: 40}
	if err := e.openDialogue("talk", false, nil); err != nil {
		t.Fatal(err)
	}
	live := e.liveNumbers()
	for _, top := range []int{7, 20} {
		e.objectTop = top
		e.objectSpec()
		first := e.objectTop
		if first == 0 {
			t.Fatalf("the list cannot scroll to %d", top)
		}
		for _, pair := range [][2]int{{first + 1, first + 3}, {first + 2, first + 1}} {
			e.objectTop = first
			a, okA := e.DialogueListRow(pair[0] + 1)
			b, okB := e.DialogueListRow(pair[1] + 1)
			if !okA || !okB {
				t.Fatalf("top %d: rows %v are not drawn", first, pair)
			}
			feedClick(t, e, a.Min.Add(a.Size().Div(2)))
			if e.obj.sel != live[pair[0]] || e.objectTop != first {
				t.Fatalf("top %d: a pick showed line %d at scroll %d, want line %d at %d", first, e.obj.sel, e.objectTop, live[pair[0]], first)
			}
			b, _ = e.DialogueListRow(pair[1] + 1)
			feedClick(t, e, b.Min.Add(b.Size().Div(2)))
			if e.obj.sel != live[pair[1]] || e.objectTop != first {
				t.Fatalf("top %d: the next press showed line %d at scroll %d, want line %d at %d", first, e.obj.sel, e.objectTop, live[pair[1]], first)
			}
		}
	}
}

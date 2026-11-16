package ui

import (
	"fmt"
	"image"
	"slices"
	"testing"

	"againrom/pkg/words"
)

// ownerHost is the modes host whose Add owner step lists the map's players
// as "slot: name", as the game's does, whose player records keep one key
// through a rename and a delete, and which logs each added unit's owner.
type ownerHost struct {
	*modesFull
	keys   []uint64
	owners []int64
}

func (h *ownerHost) NewRecordStep(r InspectionNewRecord, key string) InspectionNewStep {
	if key != addOwner.Key {
		return h.modesFull.NewRecordStep(r, key)
	}
	var out InspectionNewStep
	for i, name := range h.players {
		out.Choices = append(out.Choices, InspectionChoice{Value: int64(i + 1), Label: fmt.Sprintf("%d: %s", i+1, name)})
	}
	return out
}

func (h *ownerHost) AddRecord(r InspectionNewRecord) (*InspectionDocument, error) {
	switch r.Kind {
	case KindPlayer:
		h.keys = append(h.keys, uint64(1001+len(h.keys)))
	case KindUnit:
		for _, v := range r.Values {
			if v.Key == addOwner.Key {
				h.owners = append(h.owners, v.Value)
			}
		}
	}
	return h.modesFull.AddRecord(r)
}

func (h *ownerHost) Delete(k RecordKind, i int) (*InspectionDocument, error) {
	if k == KindPlayer && i > 0 && i < len(h.keys) {
		h.keys = slices.Delete(h.keys, i, i+1)
	}
	return h.modesFull.Delete(k, i)
}

// ownerEditor is the full modes editor on ownerHost in lang, its map's
// documents keeping the first one's source.
func ownerEditor(t *testing.T, lang string) (*MapEditor, *ownerHost) {
	t.Helper()
	e, _ := newModesEditor(t, "full")
	e.SetWords(words.For(lang))
	f := e.doc.Edits.(*modesFull)
	h := &ownerHost{modesFull: f, keys: []uint64{1001, 1002}}
	source, build := e.doc.Source, f.doc
	keyed := func(d *InspectionDocument) *InspectionDocument {
		d.Source = source
		for i := range d.Records {
			if r := &d.Records[i]; r.Kind == KindPlayer && r.Index < len(h.keys) {
				r.Key = h.keys[r.Index]
			}
		}
		return d
	}
	f.self = h
	f.doc = func(x InspectionEdits) *InspectionDocument { return keyed(build(x)) }
	e.doc.Edits = h
	keyed(e.doc)
	e.places[popList].at = image.Pt(8, e.barHeight()+8)
	return e, h
}

// ownerRow is the Units tab's player row as the palette lays it out: its
// button captions, the pressed one, and the buttons' window rectangles.
func ownerRow(t *testing.T, e *MapEditor) ([]string, int, []image.Rectangle) {
	t.Helper()
	s, ok := e.listSpec()
	if !ok {
		t.Fatal("the palette is not shown")
	}
	l := layoutPopup(s, e.font, e.bodyFont())
	for _, rr := range l.Rows {
		row := s.Rows[rr.Index]
		if row.Label != e.word("editor.place.player") {
			continue
		}
		pressed := -1
		for k, on := range row.Armed {
			if on {
				pressed = k
			}
		}
		return row.Buttons, pressed, rr.Buttons
	}
	t.Fatal("the palette shows no player row")
	return nil, 0, nil
}

// paletteSays reports whether a row of the shown palette reads text.
func paletteSays(e *MapEditor, text string) bool {
	s, _ := e.listSpec()
	return slices.ContainsFunc(s.Rows, func(r popupRow) bool { return r.Label == text })
}

// The Units tab shows a player row in place of the placing note; a player
// chosen there owns each unit a cell press and map clicks place, one copy a
// click, and no window opens. The Structures tab keeps its note and has no
// player row.
func TestMapEditorUnitsTabPlacesForTheChosenPlayer(t *testing.T) {
	for _, lang := range []string{"en", "ru"} {
		t.Run(lang, func(t *testing.T) {
			e, h := ownerEditor(t, lang)
			chromeClick(e, barPoint(t, e, "units"))
			if paletteSays(e, e.word("editor.place.hint")) {
				t.Fatal("the Units tab still shows the placing note")
			}
			captions, pressed, boxes := ownerRow(t, e)
			if fmt.Sprint(captions) != "[1: Self 2: Enemy]" || pressed != 0 {
				t.Fatalf("the player row reads %v, pressed %d", captions, pressed)
			}
			chromeClick(e, center(boxes[1]))
			if _, pressed, _ = ownerRow(t, e); pressed != 1 {
				t.Fatalf("a press on Enemy left button %d pressed", pressed)
			}
			r, ok := e.FindControl(ControlQuery{Role: ControlPlaceGridCell, N: 1})
			if !ok {
				t.Fatal("grid cell 1 is not shown")
			}
			chromeClick(e, center(r))
			if _, ok := e.promptSpec(); ok {
				t.Fatal("a press on a unit cell opened the Add window")
			}
			e.Fit()
			modesClick(e, e.CellPoint(image.Pt(40, 20)))
			modesClick(e, e.CellPoint(image.Pt(41, 20)))
			if fmt.Sprint(h.owners) != "[2 2]" {
				t.Fatalf("two clicks placed units owned by %v, want [2 2]", h.owners)
			}
			if _, ok := e.promptSpec(); ok || e.pick != pickAddCell {
				t.Fatal("placing opened a window or ended the pick")
			}
			if _, pressed, _ = ownerRow(t, e); pressed != 1 {
				t.Fatalf("placing moved the player row to %d", pressed)
			}
			chromeClick(e, barPoint(t, e, "structures"))
			if !paletteSays(e, e.word("editor.place.hint")) || paletteSays(e, e.word("editor.place.player")) {
				t.Fatal("the Structures tab lost its note or shows the player row")
			}
		})
	}
}

// The player row reads the map's players each time: a rename shows at once,
// deleting another player keeps the chosen one at its new slot, and
// deleting the chosen one returns the row to the first player.
func TestMapEditorUnitsTabPlayerFollowsRenameAndDelete(t *testing.T) {
	e, h := ownerEditor(t, "en")
	if err := e.adoptEdit(h.AddRecord(InspectionNewRecord{Kind: KindPlayer})); err != nil {
		t.Fatal(err)
	}
	chromeClick(e, barPoint(t, e, "units"))
	_, _, boxes := ownerRow(t, e)
	chromeClick(e, center(boxes[2]))
	if err := e.adoptEdit(h.SetTextField(KindPlayer, 2, "name", "Third")); err != nil {
		t.Fatal(err)
	}
	captions, pressed, _ := ownerRow(t, e)
	if fmt.Sprint(captions) != "[1: Self 2: Enemy 3: Third]" || pressed != 2 {
		t.Fatalf("after the rename the row reads %v, pressed %d", captions, pressed)
	}
	if err := e.adoptEdit(h.Delete(KindPlayer, 1)); err != nil {
		t.Fatal(err)
	}
	if captions, pressed, _ = ownerRow(t, e); fmt.Sprint(captions) != "[1: Self 2: Third]" || pressed != 1 {
		t.Fatalf("after deleting Enemy the row reads %v, pressed %d", captions, pressed)
	}
	if err := e.adoptEdit(h.Delete(KindPlayer, 1)); err != nil {
		t.Fatal(err)
	}
	if captions, pressed, _ = ownerRow(t, e); fmt.Sprint(captions) != "[1: Self]" || pressed != 0 {
		t.Fatalf("after deleting the chosen player the row reads %v, pressed %d", captions, pressed)
	}
}

package game

import (
	"fmt"
	"image"
	"slices"
	"strings"
	"testing"
	"time"

	"againrom/internal/synth"
	"againrom/pkg/ui"
	"againrom/pkg/words"
)

// speakerPlacesMap is the speaker pick map's four placements with a trigger
// whose one action starts the dialogue of page 1.
func speakerPlacesMap() []byte {
	acts := []scriptNode{{label: "Talk", opcode: 2, id: 1, tags: []uint32{1}, values: []uint32{1}}}
	checks := []scriptNode{{label: "One", opcode: 0x10002, id: 1, tags: []uint32{1}, values: []uint32{1}}}
	triggers := []goalTrigger{{name: "Talk here", left: 1, right: 1, actions: []uint32{1}}}
	return synth.ALM(synth.ALMOptions{Width: 64, Height: 64, Type7Payload: goalScript(acts, checks, triggers), Units: []synth.ALMUnit{
		pickUnit(pickPaladinCell, 1, 25, 1, 1),
		pickUnit(pickFighterCell, 3, 0, 0, 2),
		pickUnit(pickLadyCell, 4, 0, 0, 3),
		pickUnit(pickGoblinCell, pickGoblinClass, 2, 0, 4),
	}})
}

// openMessageLink presses the shown node popup's Open link of its message
// field.
func openMessageLink(t *testing.T, e *ui.MapEditor, book words.Book) {
	t.Helper()
	link := e.Document().Edits.(*mapInspectionEdits).inspector.inspectionUTF8(book.Text("editor.param.open") + " ")
	for i, line := range e.PopupLines() {
		if strings.HasPrefix(line, link) {
			r, ok := e.FindControl(ui.ControlQuery{Role: ui.ControlPopupRow, N: i})
			messageClick(t, e, r, ok)
			return
		}
	}
	t.Fatalf("the node popup shows no %q link:\n%s", link, trigLines(e))
}

// speakerRoutes are the editor's ways into a dialogue speaker field. Each
// opens the message popup, the one place a speaker is chosen; fresh routes
// open it on a new page.
var speakerRoutes = []struct {
	name  string
	fresh bool
	open  func(t *testing.T, e *ui.MapEditor, book words.Book)
}{
	{"the action's Open Message link", false, func(t *testing.T, e *ui.MapEditor, book words.Book) {
		e.Select(recordIndex(t, e, "Action", 0), false)
		openMessageLink(t, e, book)
	}},
	{"the trigger wizard's Configure", false, func(t *testing.T, e *ui.MapEditor, _ words.Book) {
		e.Select(recordIndex(t, e, "Trigger", 0), false)
		configure(t, e, "a0.item")
	}},
	{"Details, the action row, Open Message", false, func(t *testing.T, e *ui.MapEditor, book words.Book) {
		e.Select(recordIndex(t, e, "Trigger", 0), false)
		messageButton(t, e, 4)
		r, ok := e.FindControl(ui.ControlQuery{Role: ui.ControlRowButton, N: 11, K: 0, Flag: false})
		trigClick(t, e, r, ok, "the action row's button")
		openMessageLink(t, e, book)
	}},
	{"the message field's New message", true, func(t *testing.T, e *ui.MapEditor, book words.Book) {
		e.Select(recordIndex(t, e, "Action", 0), false)
		messageField(t, e, 2)
		choices := e.DropdownChoices() // the host's pages, then New message
		messageChoiceClick(t, e, choices[len(choices)-1])
	}},
}

// Every way into a dialogue speaker field reaches the message popup's
// speaker row, whose On the map pick is the one speaker pick: in English
// and Russian, it refuses the woman, whom no record's dialogue shows, names
// the cause beside card 1, stays armed, and then takes the man fighter,
// whose record 12 Apply writes as the card's speaker.
func TestEverySpeakerPlaceOffersTheOnePick(t *testing.T) {
	for _, lang := range []string{"", "russian"} {
		book := words.For(lang)
		for _, route := range speakerRoutes {
			t.Run(fmt.Sprintf("%s/%s", book.Text("editor.message.new"), route.name), func(t *testing.T) {
				e, s := speakerPickEditorOn(t, speakerPlacesMap())
				if lang != "" {
					e.SetWords(book)
				}
				route.open(t, e, book)
				if !slices.ContainsFunc(e.PopupLines(), func(l string) bool { return strings.HasPrefix(l, "1 ") }) {
					t.Fatalf("the route shows no message popup:\n%s", trigLines(e))
				}
				r, ok := e.FindControl(ui.ControlQuery{Role: ui.ControlRowButton, N: 1, K: 0, Flag: false})
				if !ok {
					t.Fatalf("the speaker row offers no pick:\n%s", trigLines(e))
				}
				messageClick(t, e, r, ok)
				hover := e.CellPoint(pickLadyCell)
				e.Pointer(ui.Input{CursorX: hover.X, CursorY: hover.Y}, time.Time{})
				click := func(c image.Point) {
					p := e.CellPoint(c)
					messageClick(t, e, image.Rectangle{Min: p, Max: p.Add(image.Pt(1, 1))}, true)
				}
				click(pickLadyCell)
				// Record 3 fits her, but a dialogue shows the paladin, the
				// first unit it fits.
				// The paladin is named by the one unit label.
				m, _ := s.model.Map()
				paladin := ""
				for _, u := range m.AuthoredUnits {
					if unitCell(u) == pickPaladinCell {
						paladin = s.inspector.unitLabel(u, s.mission())
					}
				}
				enc := s.inspector.inspectionUTF8
				why := fmt.Sprintf(enc(book.Text("editor.speaker.why.notfirst")), enc("3: speaker"), paladin)
				want := fmt.Sprintf(enc(book.Text("editor.speaker.why.card")), "1", why)
				if got := strings.Join(strings.Fields(strings.Join(e.PopupLines(), " ")), " "); !strings.Contains(got, want) {
					t.Fatalf("the refused woman does not name the cause %q:\n%s", want, trigLines(e))
				}
				click(pickFighterCell)
				messageButton(t, e, 0)
				written := ""
				for name, page := range s.workspace.Pages {
					if strings.Contains(string(page), "npc=12") {
						written = name
					}
				}
				if written == "" {
					t.Fatalf("Apply wrote no page with speaker 12: %q", s.workspace.Pages)
				}
				if !route.fresh && written != "event01.txt" {
					t.Fatalf("the pick wrote %s, not page 1", written)
				}
			})
		}
	}
}

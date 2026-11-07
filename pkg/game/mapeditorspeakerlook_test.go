package game

import (
	"bytes"
	"image"
	"strings"
	"testing"
	"time"

	"againrom/pkg/ui"
	"againrom/pkg/words"
)

// A click on the goblin makes its speaker the card's speaker, and that
// speaker's face is the picture the goblin's info window shows. A click on the
// woman no speaker shows leaves the speaker as it was and says why.
func TestSpeakerPickedOnTheMapWearsTheInfoWindowLook(t *testing.T) {
	e, s := speakerPickEditor(t)
	openPageOne(t, e)
	press := func() {
		r, ok := e.RowButtonRect(1, 0, false)
		messageClick(t, e, r, ok)
	}
	click := func(cell image.Point) {
		p := e.CellPoint(cell)
		e.Pointer(ui.Input{CursorX: p.X, CursorY: p.Y}, time.Time{})
		p = e.CellPoint(cell)
		messageClick(t, e, image.Rectangle{Min: p, Max: p.Add(image.Pt(1, 1))}, true)
	}
	card := func() string { return e.PopupLines()[0] }

	press()
	before := card()
	click(pickLadyCell)
	if card() != before {
		t.Fatalf("the refused click changed the card from %q to %q", before, card())
	}
	if !strings.Contains(e.StatusLine(), words.For("en").Text("editor.param.nospeakerunit")) {
		t.Fatalf("the refused click is silent: status %q", e.StatusLine())
	}
	click(pickGoblinCell)
	if !strings.HasPrefix(card(), "1 41:") {
		t.Fatalf("the goblin's click left the card %q, want speaker 41", card())
	}
	mw := missionStart(t, s)
	info := mw.unitPicture(3, pickGoblinClass)
	face, _, ok := mw.SpeakerFace(41)
	if info == nil || !ok || !bytes.Equal(face.Pix, info.Pix) {
		t.Fatal("the picked speaker's face is not the goblin's info picture")
	}
}

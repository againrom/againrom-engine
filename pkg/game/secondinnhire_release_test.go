package game

import (
	"image"
	"strings"
	"testing"

	"againrom/pkg/ui"
)

// secondInnRows is the visible list rows of the current town screen.
func secondInnRows(s *secondCampaignScreen) []string {
	var rows []string
	for _, r := range s.Rows() {
		rows = append(rows, r.Text)
	}
	return rows
}

// secondInnSeatPaint paints the inn page and returns seat 0's pixels.
func secondInnSeatPaint(s *secondCampaignScreen) *image.RGBA {
	dst := image.NewRGBA(image.Rect(0, 0, 640, 480))
	s.TownSquareView().Scene.Paint(dst)
	return dst.SubImage(image.Rect(176, 416, 224, 480)).(*image.RGBA)
}

func secondInnSeatEqual(a, b *image.RGBA) bool {
	for y := a.Rect.Min.Y; y < a.Rect.Max.Y; y++ {
		for x := a.Rect.Min.X; x < a.Rect.Max.X; x++ {
			if a.RGBAAt(x, y) != b.RGBAAt(x, y) {
				return false
			}
		}
	}
	return true
}

// secondInnNotice pages through an open inn dialogue.
func secondInnNotice(t *testing.T, app *ui.App) {
	t.Helper()
	for page := 0; page < 64; page++ {
		if err := app.HeadlessActivate("notice"); err != nil {
			return
		}
	}
	t.Fatal("inn dialogue exceeded 64 pages")
}

// In the Kaarg inn after mission 20's departure stores (hire 1 available,
// in town 2), the inn page seats hire 1 first; Hire shows npc1accept30 and
// selects it in bank slot 512 with the hired mark on its seat; SAVE and
// cold LOAD keep the selection, and Fire clears it with no dialogue
// (R2-ENGINE-495, R2-ENGINE-496, R2-ENGINE-497, R2-SESSION-191). The
// mission entry of the selection is DIV-3111.
func TestReleaseSecondTownInnHireSaveAndFire(t *testing.T) {
	f, app, s, _ := secondTownTwo(t)
	c := f.Town.second
	c.bank[532], c.bank[552] = 1, 2
	if err := app.HeadlessActivate("TAVERN"); err != nil || c.room != secondTownInn {
		t.Fatalf("TAVERN: %v room %v", err, c.room)
	}
	s.TownSquareActive(true)
	rows := secondInnRows(s)
	if len(rows) < 2 || rows[0] != "HIRE 1" || !strings.HasPrefix(rows[1], "TALK ") {
		t.Fatalf("inn rows %v", rows)
	}
	words := f.Words.Hover
	if len(words) <= secondInnFireCaption || words[secondInnHiredMark] == "" || words[secondInnHireCaption] == "" || words[secondInnFireCaption] == "" {
		t.Fatalf("main text keys 257..259 absent: %d keys", len(words))
	}
	unhired := secondInnSeatPaint(s)
	if err := app.HeadlessActivate("HIRE 1"); err != nil {
		t.Fatal(err)
	}
	body, ok := s.dialogueBody()
	if !ok || body == "" || c.bank[512] != 1 || !c.hired(1) {
		t.Fatalf("Hire: dialogue %v %q, slot 512 = %d", ok, body, c.bank[512])
	}
	t.Logf("npc1accept30 shows %d bytes", len(body))
	secondInnNotice(t, app)
	if rows := secondInnRows(s); rows[0] != "FIRE 1" {
		t.Fatalf("rows after Hire %v", rows)
	}
	hired := secondInnSeatPaint(s)
	if secondInnSeatEqual(unhired, hired) {
		t.Fatal("the hired mark changed no pixel of the hire seat")
	}

	out := t.TempDir()
	secondTownNamedSave(t, f, app, out, "Kaarg inn hire")
	cold, coldApp := secondTownCold(t, out, secondSaveFile(t, out, "Kaarg inn hire"))
	cs := cold.TownScreen().(*secondCampaignScreen)
	coldApp.SetTown(cs)
	cc := cold.Town.second
	if cc.current != (secondLocation{2, 2}) || cc.room != secondTownInn || cc.bank[512] != 1 || cc.bank[532] != 1 || cc.bank[552] != 2 {
		t.Fatalf("cold LOAD current %v room %v slots 512/532/552 = %d/%d/%d", cc.current, cc.room, cc.bank[512], cc.bank[532], cc.bank[552])
	}
	cs.TownSquareActive(true)
	if rows := secondInnRows(cs); rows[0] != "FIRE 1" {
		t.Fatalf("cold rows %v", rows)
	}
	if err := coldApp.HeadlessActivate("FIRE 1"); err != nil {
		t.Fatal(err)
	}
	if _, open := cs.dialogueBody(); open || cc.bank[512] != 0 || cc.hired(1) {
		t.Fatalf("Fire: dialogue %v, slot 512 = %d", open, cc.bank[512])
	}
	if rows := secondInnRows(cs); rows[0] != "HIRE 1" {
		t.Fatalf("rows after Fire %v", rows)
	}
	// Loss control: the fired seat paints as the unhired one did.
	if fired := secondInnSeatPaint(cs); !secondInnSeatEqual(unhired, fired) {
		t.Error("the fired seat still differs from the unhired seat")
	}
}

// Hire on a kind 2 hire (availability 2, after a departure that took it)
// shows npc<ID>reject<stage> and selects nothing (R2-ENGINE-495,
// R2-SESSION-191).
func TestReleaseSecondTownInnHireRejectsKindTwo(t *testing.T) {
	f, app, s, _ := secondTownTwo(t)
	c := f.Town.second
	c.bank[532], c.bank[552] = 2, 2
	if err := app.HeadlessActivate("TAVERN"); err != nil {
		t.Fatal(err)
	}
	s.TownSquareActive(true)
	if rows := secondInnRows(s); rows[0] != "HIRE 1" {
		t.Fatalf("inn rows %v", rows)
	}
	before := c.bank
	if err := app.HeadlessActivate("HIRE 1"); err != nil {
		t.Fatal(err)
	}
	body, ok := s.dialogueBody()
	if !ok || body == "" || c.bank != before || c.hired(1) {
		t.Fatalf("Hire on kind 2: dialogue %v %q, bank changed %v", ok, body, c.bank != before)
	}
	t.Logf("npc1reject30 shows %d bytes", len(body))
}

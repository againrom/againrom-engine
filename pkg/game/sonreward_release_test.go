package game

import (
	"bytes"
	"strings"
	"testing"

	"againrom/internal/firstgame"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

// sonRewardRun opens mission 131 through the map driver, raises the farmer's
// son when heal is set, and steps until the purse rises or limit steps pass.
// Every page that opens before then is paged through. It returns the step of
// the rise (or limit), the purse gain, the message lines standing on that
// step, whether the thank-you page (event 5) opened on it, and the gold line
// the installed words state for 10000.
func sonRewardRun(t *testing.T, heal bool, limit int) (int, uint32, []ui.MessageLine, bool, string) {
	t.Helper()
	f := releaseFront(t)
	f.SetDeterministicFrames(true)
	app := f.App("son reward")
	app.Layout(1024, 768)
	if err := app.OpenMission(f.MissionOpener(131)); err != nil {
		t.Fatal(err)
	}
	mw := f.live
	m := mw.mission
	thanks, ok := m.files().eventText(m.src, m.edition(), m.number, 5)
	if !ok {
		t.Fatal("mission 131 has no event 5 text")
	}
	if heal {
		var son sim.EntityID
		for _, e := range mw.world.Entities() {
			if e.MapUnitID == 32 && e.Owner != sim.SelfSlot {
				son = e.ID
			}
		}
		hero, found := mw.primaryPartyID()
		if son == 0 || !found {
			t.Fatalf("son %d hero %d", son, hero)
		}
		if err := mw.world.HeadlessHeal(son); err != nil {
			t.Fatal(err)
		}
		near := false
		for _, c := range [][2]int32{{78, 65}, {79, 65}, {80, 65}, {78, 66}, {80, 66}, {78, 67}, {79, 67}, {80, 67}} {
			if mw.world.HeadlessPlace(hero, c[0], c[1]) != nil {
				continue
			}
			if e, _ := mw.world.Entity(hero); e.X >= 78 && e.X <= 80 && e.Y >= 65 && e.Y <= 67 {
				near = true
				break
			}
		}
		if !near {
			t.Fatal("no cell beside the son takes the hero")
		}
	}
	words := mw.view.Words()
	want := grantGoldLine(&words, 10000)
	if lit := releaseRootText(mw, "Received 10000 gold", "Получено: 10000 монет"); want != lit {
		t.Fatalf("the script gold line reads %q, want %q", want, lit)
	}
	start := mw.world.Purse(sim.SelfSlot)
	for step := 0; step < limit; step++ {
		mw.tickStep(nil, false)
		if now := mw.world.Purse(sim.SelfSlot); now != start {
			shown := m.open && m.kind == ui.NoticeDialogue && bytes.Equal(m.payload, thanks)
			return step, now - start, mw.view.MessageLines(), shown, want
		}
		for k := 0; m.open && k < 32; k++ {
			mw.advanceNotice()
		}
		if m.open {
			t.Fatalf("step %d: a page does not close", step)
		}
	}
	return limit, 0, mw.view.MessageLines(), false, want
}

// TestReleaseDragonClanSonRewardPaysWithThePickupLine: the son raised and
// walked home pays 10000 gold, opens the thank-you page and states the gold
// on the message line as received, naming nobody (TRIG-MONEY-028,
// DIV-3767). The loss control leaves the son fallen: over the same
// steps nothing is paid, no gold line stands and the page never opens.
func TestReleaseDragonClanSonRewardPaysWithThePickupLine(t *testing.T) {
	firstgame.SkipOnSecondGame(t)
	const limit = 6000
	gold := func(lines []ui.MessageLine) []string {
		var out []string
		for _, l := range lines {
			if strings.Contains(l.Text, "10000") {
				out = append(out, l.Text)
			}
		}
		return out
	}
	step, gain, lines, shown, want := sonRewardRun(t, true, limit)
	if gain != 10000 {
		t.Fatalf("by step %d the purse gained %d, want 10000", step, gain)
	}
	if !shown {
		t.Fatalf("step %d: the thank-you page did not open", step)
	}
	if got := gold(lines); len(got) != 1 || got[0] != want || lines[len(lines)-1].Ink != ui.MessageWhite {
		t.Fatalf("step %d: message lines %+v, want one %q", step, lines, want)
	}
	t.Logf("step %d: %q", step, want)
	step, gain, lines, shown, _ = sonRewardRun(t, false, limit)
	if gain != 0 || shown || len(gold(lines)) != 0 {
		t.Fatalf("loss control: step %d gain %d page %t lines %+v", step, gain, shown, lines)
	}
}

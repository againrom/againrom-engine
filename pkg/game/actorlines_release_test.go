package game

import (
	"slices"
	"testing"

	"againrom/internal/firstgame"
	"againrom/pkg/data"
	"againrom/pkg/render/text"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

// releaseRootText is en on an English root and ru, in the map font's CP866
// bytes, on a Russian one: a line's literal as the root's player reads it.
func releaseRootText(mw *mapWorld, en, ru string) string {
	if sel := mw.view.TextSelector(); sel == text.SelectorConverting {
		return EncodeInstallText(ru, sel)
	}
	return en
}

// Mission 30's script gives the hero the Cure for Plague (code 0x0e1e) by an
// instant-12 node within its first steps. Played through the map driver, the
// grant states the received line naming the hero, the player's name on its
// card (DIV-3766). The loss control advances the same world without the
// driver's step report: the Item still arrives and no line posts, so the line
// is the driver's reading of the grant.
func TestReleaseScriptItemGrantStatesTheReceivedLine(t *testing.T) {
	firstgame.SkipOnSecondGame(t)
	const grant = data.ItemCode(0x0e1e)
	for _, observed := range []bool{true, false} {
		f := releaseFront(t)
		f.SetDeterministicFrames(true)
		f.SetTipsOff(true)
		a := f.App("script item grant")
		a.SetCutscenes(nil)
		party := helpWitnessParty(false)
		party[0].Name = "Gord"
		if err := a.OpenMission(f.MissionOpenerWith(30, party)); err != nil {
			t.Fatal(err)
		}
		mw := f.live
		hero := mw.mission.ids[0]
		holds := func() bool {
			stacks, _ := mw.world.CarriedStacks(hero)
			return slices.ContainsFunc(stacks, func(s sim.ItemStack) bool { return data.ItemCode(s.Code) == grant })
		}
		if holds() || len(mw.view.MessageLines()) != 0 {
			t.Fatalf("observed=%v: the hero holds the Item or a line stands before the grant", observed)
		}
		for step := 0; step < 400 && !mw.world.ScriptLatched(7); step++ {
			if observed {
				mw.tickStep(nil, false)
			} else {
				sim.Step(mw.world, nil)
			}
		}
		if !mw.world.ScriptLatched(7) || !holds() {
			t.Fatalf("observed=%v: the grant did not run (latch 7 %v)", observed, mw.world.ScriptLatched(7))
		}
		lines := mw.view.MessageLines()
		if !observed {
			if len(lines) != 0 {
				t.Fatalf("loss control: lines %+v without the driver's report", lines)
			}
			continue
		}
		name := itemName(grant, mw.invParty.table)
		if holder := messageHolder(t, mw, hero); holder != "Gord" {
			t.Fatalf("the hero's card shows %q, want the player's name \"Gord\"", holder)
		}
		want := announced(releaseRootText(mw, "Received ", "Получено: ") + name + " (Gord)")
		if got := slices.DeleteFunc(slices.Clone(lines), func(l ui.MessageLine) bool { return l.Text != want[0].Text }); len(got) != 1 || got[0] != want[0] {
			t.Fatalf("the grant posted %+v, want one %q", lines, messageShown(t, f.Font.Value(), want[0].Text))
		}
		requireInkedNoticeGlyphs(t, f.Font.Value(), want[0].Text)
		t.Logf("script item grant: %q", messageShown(t, f.Font.Value(), want[0].Text))
	}
}

package game

import (
	"testing"

	"againrom/pkg/base"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

// TestObjectivesAndOutcomeAnswersPerGame records what the objectives panel
// and the outcome text answer in both games over fixture data. The answers
// were recorded before the one producer replaced the per-game hooks.
func TestObjectivesAndOutcomeAnswersPerGame(t *testing.T) {
	w := secondGameScriptWorld(t,
		[]sim.ScriptInstant{{Op: 35, Args: [10]int32{752, 3}}, {Op: 35, Args: [10]int32{753, 5}}},
		[]sim.ScriptTrigger{missionTrigger(3, 1, true, 0, 1)})
	for i := 0; i < 48; i++ {
		sim.Step(w, nil)
	}
	labels := []string{"first", "second", "hidden", "open"}
	failures := []string{"dead hero", "", "lost caravan"}
	panels := []struct {
		name     string
		game     base.Game
		briefing string
		want     string
	}{
		{"first plain", base.GameROM1, "briefing", "briefing"},
		{"first padded", base.GameROM1, "  briefing\n", "  briefing\n"},
		{"first empty", base.GameROM1, "", ""},
		{"second plain", base.GameROM2, "briefing", "briefing\n\n[+] first\n[-] second"},
		{"second padded", base.GameROM2, "  briefing\n", "briefing\n\n\n[+] first\n[-] second"},
		{"second empty", base.GameROM2, "", "[+] first\n[-] second"},
	}
	for _, tc := range panels {
		t.Run("panel "+tc.name, func(t *testing.T) {
			mw := &mapWorld{world: w, mission: &missionNotices{
				table: readUnder(tc.game, &mapload.Table{}), objectiveLabels: labels,
			}}
			if got := gameMenuContext(mw, true, tc.briefing).Objective; got != tc.want {
				t.Fatalf("objectives = %q, want %q", got, tc.want)
			}
		})
	}
	outcomes := []struct {
		name    string
		game    base.Game
		outcome sim.Outcome
		reason  int32
		want    string
	}{
		{"first won", base.GameROM1, sim.OutcomeWon, 0, "generic"},
		{"first lost reason 2", base.GameROM1, sim.OutcomeLost, 2, "generic"},
		{"second won", base.GameROM2, sim.OutcomeWon, 2, "generic"},
		{"second lost reason 2", base.GameROM2, sim.OutcomeLost, 2, "dead hero"},
		{"second lost empty entry", base.GameROM2, sim.OutcomeLost, 3, "generic"},
		{"second lost reason 4", base.GameROM2, sim.OutcomeLost, 4, "lost caravan"},
		{"second lost past entries", base.GameROM2, sim.OutcomeLost, 5, "generic"},
		{"second lost reason 1", base.GameROM2, sim.OutcomeLost, 1, "generic"},
		{"second lost reason 0", base.GameROM2, sim.OutcomeLost, 0, "generic"},
	}
	for _, tc := range outcomes {
		t.Run("outcome "+tc.name, func(t *testing.T) {
			v := &ui.Viewer{}
			mw := &mapWorld{view: v, mission: &missionNotices{
				table: readUnder(tc.game, &mapload.Table{}), outcome: tc.outcome, lossReason: tc.reason, failureText: failures,
			}}
			mw.showOutcome()
			want := tc.want
			if want == "generic" {
				want = missionOutcomeText(v, tc.outcome)
			}
			if body, _, shown := v.NoticeState(); !shown || body != want {
				t.Fatalf("notice %q/%v, want %q", body, shown, want)
			}
		})
	}
}

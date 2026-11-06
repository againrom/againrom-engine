package game

import (
	"reflect"
	"testing"

	"againrom/pkg/base"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

// TestObjectivesAndOutcomeAnswersPerGame records what the objectives panel
// and the outcome text answer in both games over fixture data: the briefing
// as given, and in the second game a row with its mark frame per opened
// sub-objective.
func TestObjectivesAndOutcomeAnswersPerGame(t *testing.T) {
	w := secondGameScriptWorld(t,
		[]sim.ScriptInstant{{Op: 35, Args: [10]int32{752, 3}}, {Op: 35, Args: [10]int32{753, 5}}},
		[]sim.ScriptTrigger{missionTrigger(3, 1, true, 0, 1)})
	for i := 0; i < 48; i++ {
		sim.Step(w, nil)
	}
	labels := []string{"first", "second", "hidden", "open"}
	failures := []string{"dead hero", "", "lost caravan"}
	opened := []ui.QuestObjective{{Text: "first", Mark: 1}, {Text: "second", Mark: 2}}
	panels := []struct {
		name     string
		game     base.Game
		briefing string
		rows     []ui.QuestObjective
	}{
		{"first plain", base.GameROM1, "briefing", nil},
		{"first padded", base.GameROM1, "  briefing\n", nil},
		{"first empty", base.GameROM1, "", nil},
		{"second plain", base.GameROM2, "briefing", opened},
		{"second padded", base.GameROM2, "  briefing\n", opened},
		{"second empty", base.GameROM2, "", opened},
	}
	for _, tc := range panels {
		t.Run("panel "+tc.name, func(t *testing.T) {
			mw := &mapWorld{world: w, mission: &missionNotices{
				table: readUnder(tc.game, &mapload.Table{}), objectiveLabels: labels,
			}}
			ctx := gameMenuContext(mw, true, tc.briefing)
			if ctx.Objective != tc.briefing || !reflect.DeepEqual(ctx.Objectives, tc.rows) {
				t.Fatalf("objectives = %q %v, want %q %v", ctx.Objective, ctx.Objectives, tc.briefing, tc.rows)
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

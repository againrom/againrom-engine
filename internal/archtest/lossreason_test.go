package archtest

import (
	"os"
	"strings"
	"testing"
)

// TestCheckLossReasonNamesTheProducer drives the scan with synthetic
// sources.
func TestCheckLossReasonNamesTheProducer(t *testing.T) {
	cases := []struct {
		name, file, src, want string
	}{
		{"the latch", "pkg/game/world.go",
			"package game\nfunc (mw *mapWorld) latchOutcome(o sim.Outcome, c lossCause) { mw.mission.lossReason = mw.missionLossReason(c); mw.mission.announced, mw.mission.outcome = true, o }\n", ""},
		{"a second outcome writer", "pkg/game/a.go",
			"package game\nfunc lose(m *missionNotices) { m.announced, m.outcome = true, sim.OutcomeLost }\n", "lose writes the mission's outcome"},
		{"a second reason writer", "pkg/game/a.go",
			"package game\nfunc reason(m *missionNotices) { m.lossReason = 3 }\n", "reason writes the mission's lossReason"},
		{"the latch from the counters", "pkg/game/world.go",
			"package game\nfunc (mw *mapWorld) latchOutcome(o sim.Outcome, c lossCause) { _, l := mw.world.ScriptCounters(); mw.mission.lossReason = int32(l) }\n", "something other than missionLossReason"},
		{"the notice from the counters", "pkg/game/campaignsecond.go",
			"package game\nfunc (secondCampaignRules) outcomeText(mw *mapWorld, body string) string { _, r := mw.world.ScriptCounters(); _ = r; return body }\n", "outcomeText reads the script counters"},
		{"the party reason", "pkg/game/world.go",
			"package game\nfunc partyFailureReason(e base.Edition, row base.PartyIdentity) int32 { return e.HeroFailureReason + row.FailureReason }\n", ""},
		{"a second edition reader", "pkg/game/a.go",
			"package game\nfunc fall(row base.PartyIdentity) int32 { return row.FailureReason }\n", "fall reads the edition's FailureReason"},
		{"the edition's own rows", "pkg/base/edition.go",
			"package base\nfunc (e Edition) r(p PartyIdentity) int32 { return p.FailureReason }\n", ""},
		{"a test", "pkg/game/a_test.go",
			"package game\nfunc f(m *missionNotices) { m.lossReason = 3 }\n", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var got []string
			for _, v := range CheckLossReason(map[string]string{tc.file: tc.src}) {
				got = append(got, v.Reason)
			}
			if tc.want == "" {
				if len(got) != 0 {
					t.Fatalf("CheckLossReason = %v, want none", got)
				}
				return
			}
			if len(got) == 0 || !strings.Contains(strings.Join(got, "\n"), tc.want) {
				t.Fatalf("CheckLossReason = %v, want one naming %q", got, tc.want)
			}
		})
	}
}

// TestLossReasonHasOneProducer is the live-tree half.
func TestLossReasonHasOneProducer(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	root, err := FindModuleRoot(wd)
	if err != nil {
		t.Fatal(err)
	}
	files, err := LoadDrawnTextSources(root)
	if err != nil {
		t.Fatal(err)
	}
	src := files["pkg/game/world.go"]
	if !strings.Contains(src, "func (mw *mapWorld) latchOutcome(") || !strings.Contains(src, "func (mw *mapWorld) missionLossReason(") {
		t.Fatal("pkg/game/world.go holds no latchOutcome or missionLossReason; the scan guards nothing")
	}
	for _, v := range CheckLossReason(files) {
		t.Errorf("%s: %s", v.From, v.Reason)
	}
}

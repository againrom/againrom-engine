package archtest

import (
	"os"
	"strings"
	"testing"
)

// winRouteTree is the smallest tree the win route scan accepts: the seam's
// finishWon, the finisher and the identity file's own reads.
func winRouteTree() map[string]string {
	return map[string]string{
		"pkg/game/missiontransition.go": "package game\nfunc (t frontTransitions) finishWon(n int, ms *Mission) (int, string) { return t.f.finishWin(winInput{}) }\n",
		"pkg/game/winfinish.go":         "package game\nfunc (f *FrontEnd) finishWin(in winInput) (int, string) { return 0, \"\" }\n",
		"pkg/game/sessionmission.go": "package game\nfunc (t *Town) missionIdentity(n int, m mod.MissionData) missionIdentity {\n" +
			"\tg := t.guestMission(); _ = g.playing(n); _ = g.returnMission(n); _, _ = m.ByNumber(n); _, _ = m.Replacing(n)\n\treturn missionIdentity{}\n}\n",
		"pkg/game/campaignservice.go": "package game\ntype campaignService interface {\n\tbeforeAdvance(f *FrontEnd, n int) string\n}\n",
	}
}

func TestCheckWinRoute(t *testing.T) {
	cases := []struct {
		name, file, src, want string
	}{
		{"the tree as it stands", "", "", ""},
		{"a guestMission read", "pkg/game/missionleave.go",
			"package game\nfunc f(fe *FrontEnd) bool { return fe.Town.guestMission() != nil }\n", "guestMission call outside the session mission identity"},
		{"a playing branch", "pkg/game/campaignsession.go",
			"package game\nfunc f(s *CampaignSession, n int) bool { return s.Town.guest.playing(n) }\n", "playing call outside the session mission identity"},
		{"a return mission read", "pkg/game/campaignfirst.go",
			"package game\nfunc f(g *guestMission, n int) int { return g.returnMission(n) }\n", "returnMission call outside the session mission identity"},
		{"a guest entry read", "pkg/game/frontend.go",
			"package game\nfunc f(g *guestMission, n int) { g.guestEntry(n) }\n", "guestEntry call outside the session mission identity"},
		{"a mod mission by number", "pkg/game/originalrestore.go",
			"package game\nfunc f(m mapload.ModContext, n int) bool { _, ok := m.Missions.ByNumber(n); return ok }\n", "ByNumber call outside the session mission identity"},
		{"a replaced map lookup", "pkg/game/missionports.go",
			"package game\nfunc f(m mod.MissionData, n int) bool { _, ok := m.Replacing(n); return ok }\n", "Replacing call outside the session mission identity"},
		{"a campaign mission lookup", "pkg/game/missionports.go",
			"package game\nfunc f(m mod.MissionData, n int) bool { _, ok := m.CampaignMission(n); return ok }\n", "CampaignMission call outside the session mission identity"},
		{"a per-game finishWon", "pkg/game/campaignsecond.go",
			"package game\nfunc (r secondCampaignRules) finishWon(f *FrontEnd, n int, ms *Mission) (int, string) { return 0, \"\" }\n", "finishWon on secondCampaignRules"},
		{"a second finishWon", "pkg/game/campaignfirst.go",
			"package game\nfunc (firstCampaignRules) finishWon(f *FrontEnd, n int, ms *Mission) (int, string) { return 0, \"\" }\n", "finishWon declared 2 times"},
		{"a finisher in another file", "pkg/game/modmission.go",
			"package game\nfunc (f *FrontEnd) finishWin(in winInput) (int, string) { return 0, \"\" }\n", "finishWin outside pkg/game/winfinish.go"},
		{"a win step in the campaign service", "pkg/game/campaignservice.go",
			"package game\ntype campaignService interface {\n\troute(f *FrontEnd, n, successor int) winRoute\n}\n", "campaignService declares the win step route"},
		{"a return mission field", "pkg/game/worldmap.go",
			"package game\nfunc f(s *worldMapState) bool { return s.returnMission != 0 }\n", ""},
		{"a successor arm in the town", "pkg/game/town.go",
			"package game\nfunc (t *Town) Won(n int) { t.camp.AutoAdvance(n) }\n", "win successor read outside Campaign.after"},
		{"a successor arm in the record", "pkg/game/campaignprogress.go",
			"package game\nfunc (p *campaignProgress) complete(c Campaign, n int) { _ = c.TownAfter[n] }\n", "win successor read outside Campaign.after"},
		{"a successor arm in the session", "pkg/game/campaignsession.go",
			"package game\nfunc (s *CampaignSession) recordWin(c Campaign, n int) { c.NextMission(n) }\n", "win successor read outside Campaign.after"},
		{"a test file", "pkg/game/modmission_test.go",
			"package game\nfunc f(g *guestMission) bool { return g.playing(25) }\n", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			files := winRouteTree()
			if tc.file != "" {
				files[tc.file] = tc.src
			}
			var got []string
			for _, v := range CheckWinRoute(files) {
				got = append(got, v.From+": "+v.Reason)
			}
			if tc.want == "" {
				if len(got) != 0 {
					t.Fatalf("CheckWinRoute = %v, want none", got)
				}
				return
			}
			if len(got) == 0 || !strings.Contains(strings.Join(got, "\n"), tc.want) {
				t.Fatalf("CheckWinRoute = %v, want one naming %q", got, tc.want)
			}
		})
	}
}

func TestCheckWinRouteRefusesAnEmptyScan(t *testing.T) {
	if vs := CheckWinRoute(map[string]string{"pkg/game/x_test.go": "package game\n"}); len(vs) != 1 || !strings.Contains(vs[0].Reason, "no sources scanned") {
		t.Fatalf("CheckWinRoute over no production file = %v, want the empty-scan refusal", vs)
	}
}

func TestWinRouteHoldsInTheTree(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	root, err := FindModuleRoot(wd)
	if err != nil {
		t.Fatal(err)
	}
	files, err := LoadTreeSources(root, "pkg/game")
	if err != nil {
		t.Fatal(err)
	}
	for _, v := range CheckWinRoute(files) {
		t.Errorf("%s: %s", v.From, v.Reason)
	}
}

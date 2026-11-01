package archtest

import (
	"os"
	"strings"
	"testing"
)

// TestCheckObjectivesNamesEachShape plants one instance of each shape of a
// second objectives builder.
func TestCheckObjectivesNamesEachShape(t *testing.T) {
	cases := []struct {
		name, file, src, want string
	}{
		{"the home", "pkg/game/questobjectives.go",
			"package game\nfunc (mw *mapWorld) missionObjectives(b string) string { return mw.mission.objectiveLabels[0] + b }\nfunc outcomeText(mw *mapWorld, b string) string { return mw.mission.failureText[0] + b }\nfunc loadMissionText() {}\n", ""},
		{"a first-game hook", "pkg/game/campaignfirst.go",
			"package game\nfunc (firstCampaignRules) objectives(_ *mapWorld, b string) string { return b }\n", "firstCampaignRules.objectives is a per-game objectives hook"},
		{"a second-game outcome hook", "pkg/game/campaignsecond.go",
			"package game\nfunc (secondCampaignRules) outcomeText(mw *mapWorld, b string) string { return b }\n", "secondCampaignRules.outcomeText is a per-game objectives hook"},
		{"a mission text hook", "pkg/game/campaignsecond.go",
			"package game\nfunc (r *secondCampaignRules) loadMissionText(mw *mapWorld) {}\n", "secondCampaignRules.loadMissionText is a per-game objectives hook"},
		{"a service method", "pkg/game/campaignservice.go",
			"package game\ntype campaignService interface {\n\tobjectives(mw *mapWorld, b string) string\n}\n", "interface method objectives"},
		{"a second panel producer", "pkg/game/a.go",
			"package game\nfunc (mw *mapWorld) secondGameObjectivePanel(b string) string { return b }\n", "second objectives panel producer"},
		{"a second panel producer in the home", "pkg/game/questobjectives.go",
			"package game\nfunc (mw *mapWorld) thirdObjectivesPanel(b string) string { return b }\n", "beside missionObjectives"},
		{"a second outcome function", "pkg/game/a.go",
			"package game\nfunc outcomeText(mw *mapWorld, b string) string { return b }\n", "second objectives builder"},
		{"the state read elsewhere", "pkg/game/a.go",
			"package game\nfunc panel(mw *mapWorld) int { return len(mw.mission.objectiveLabels) }\n", "reads the mission's objectiveLabels"},
		{"the failure text read elsewhere", "pkg/game/a.go",
			"package game\nfunc loss(mw *mapWorld) string { return mw.mission.failureText[0] }\n", "reads the mission's failureText"},
		{"a game branch in the home", "pkg/game/questobjectives.go",
			"package game\nfunc outcomeText(mw *mapWorld, b string) string { if mw.mission.campaign() == nil { return b }; return b }\n", "outcomeText reads campaign"},
		{"a game identity in the home", "pkg/game/questobjectives.go",
			"package game\nfunc (mw *mapWorld) missionObjectives(b string) string { if g.GameOf() == base.GameROM2 { return b }; return b }\n", "missionObjectives reads GameOf"},
		{"a test", "pkg/game/a_test.go",
			"package game\nfunc (secondCampaignRules) objectives(mw *mapWorld, b string) string { return b }\n", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var got []string
			for _, v := range CheckObjectives(map[string]string{tc.file: tc.src}) {
				got = append(got, v.Reason)
			}
			if tc.want == "" {
				if len(got) != 0 {
					t.Fatalf("CheckObjectives = %v, want none", got)
				}
				return
			}
			if len(got) == 0 || !strings.Contains(strings.Join(got, "\n"), tc.want) {
				t.Fatalf("CheckObjectives = %v, want one naming %q", got, tc.want)
			}
		})
	}
}

// TestObjectivesHaveOneBuilder is the live-tree half.
func TestObjectivesHaveOneBuilder(t *testing.T) {
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
	if !strings.Contains(files[objectivesHome], "func (mw *mapWorld) missionObjectives(") {
		t.Fatal(objectivesHome + " holds no missionObjectives; the scan guards nothing")
	}
	for _, v := range CheckObjectives(files) {
		t.Errorf("%s: %s", v.From, v.Reason)
	}
}

package archtest

import (
	"os"
	"strings"
	"testing"
)

func TestCheckMissionMessages(t *testing.T) {
	for _, tc := range []struct{ name, file, src, want string }{
		{"one pipeline", "", "", ""},
		{"second settle", "other.go", "func (mw *mapWorld) settleDeferredMessages() {}", "second mission message settle function: settleDeferredMessages"},
		{"campaign hook", "other.go", "func (secondCampaignRules) scriptMessages() {}", "per-game mission message hook: scriptMessages"},
		{"queue writer", "other.go", "func f(m *missionNotices) { m.pendingMessages = append(m.pendingMessages, 4) }", "pendingMessages outside"},
		{"second queue", "world.go", "type missionNotices struct { deferred []int32 }", "second mission message queue: deferred"},
		{"latch caller", "other.go", "func f(mw *mapWorld) { mw.latchOutcome(1, lossCause{}) }", "latchOutcome outside"},
		{"latch sampler", "other.go", "func f(m *missionNotices) { m.ann.Sample(nil) }", "Sample outside"},
		{"service hook", "campaignservice.go", "type campaignService interface { acknowledgeNotice() }", "campaignService mission delivery hook"},
		{"special-number switch", "other.go", "func f(event int) { switch event { case 250: } }", "special-number dispatch"},
		{"second special table", "other.go", "var messageRules = []int{250, 255}", "special-number table messageRules"},
		{"reserved branch", "other.go", "func f(event int) { if event == ReservedMessageNumber {} }", "ReservedMessageNumber outside"},
		{"game identity", "secondgamenotices.go", "func (mw *mapWorld) settleNotices() { if base.GameROM2 == g {} }", "game identity branch"},
		{"carrier port", "residuecarrier.go", "func f(m *missionNotices) { m.pendingMessages = nil }", ""},
		{"observer port", "scenario.go", "func f(p *PlayWorld) { p.ann.Sample(nil) }", ""},
		{"test ignored", "other_test.go", "func (mw *mapWorld) settleOtherNotices() {}", ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			files := map[string]string{missionMessagesFile: "package game\nfunc (mw *mapWorld) settleNotices() {}"}
			if tc.file != "" {
				files["pkg/game/"+tc.file] = "package game\n" + tc.src
			}
			var got []string
			for _, v := range CheckMissionMessages(files) {
				got = append(got, v.From+": "+v.Reason)
			}
			if tc.want == "" && len(got) != 0 || tc.want != "" && !strings.Contains(strings.Join(got, "\n"), tc.want) {
				t.Fatalf("CheckMissionMessages = %v, want %q", got, tc.want)
			}
		})
	}
}

func TestMissionMessagesGuardRefusesMissingSettle(t *testing.T) {
	for _, files := range []map[string]string{nil, {"pkg/game/world.go": "package game"}} {
		if len(CheckMissionMessages(files)) == 0 {
			t.Fatal("accepted a missing mission message pipeline")
		}
	}
}

func TestMissionMessagesHoldInTheTree(t *testing.T) {
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
	for _, v := range CheckMissionMessages(files) {
		t.Errorf("%s: %s", v.From, v.Reason)
	}
}

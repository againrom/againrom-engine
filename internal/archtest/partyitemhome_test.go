package archtest

import (
	"os"
	"strings"
	"testing"
)

// TestCheckPartyItemHomeNamesEachShape plants each conversion and pack
// replacement outside the home and a second graph follower inside it.
func TestCheckPartyItemHomeNamesEachShape(t *testing.T) {
	home := "package game\nfunc followHeroPack() {}\nfunc a() { followHeroPack() }\n"
	cases := []struct {
		name  string
		files map[string]string
		want  string
	}{
		{"a clean tree", map[string]string{PartyItemHomeFile: home, "pkg/game/x.go": "package game\n"}, ""},
		{"a kept conversion outside", map[string]string{PartyItemHomeFile: home,
			"pkg/game/x.go": "package game\nfunc a() { secondshop.KeptUnit(x, y) }\n"}, "secondshop.KeptUnit outside"},
		{"a mission conversion passed as a value", map[string]string{PartyItemHomeFile: home,
			"pkg/game/x.go": "package game\nvar f = secondshop.MissionUnit\n"}, "secondshop.MissionUnit outside"},
		{"a pack replacement outside", map[string]string{PartyItemHomeFile: home,
			"pkg/game/x.go": "package game\nfunc a() { memberWithPack(m, nil, nil, true) }\n"}, "memberWithPack( outside"},
		{"a graph follower outside", map[string]string{PartyItemHomeFile: home,
			"pkg/game/x.go": "package game\nfunc a() { s.followHeroPack(b, p, 0, nil) }\n"}, "followHeroPack( outside"},
		{"a second graph call in the home", map[string]string{PartyItemHomeFile: home + "func b() { followHeroPack() }\n"}, "3 times"},
	}
	for _, c := range cases {
		got := strings.Join(CheckPartyItemHome(c.files), "\n")
		if (c.want == "") != (got == "") || !strings.Contains(got, c.want) {
			t.Errorf("%s: findings %q, want %q", c.name, got, c.want)
		}
	}
}

func TestPartyItemHomeHoldsEveryCrossing(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	root, err := FindModuleRoot(wd)
	if err != nil {
		t.Fatal(err)
	}
	files, err := LoadSackPickupSources(root)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := files[PartyItemHomeFile]; !ok {
		t.Fatalf("the party item home %s is missing", PartyItemHomeFile)
	}
	for _, finding := range CheckPartyItemHome(files) {
		t.Error(finding)
	}
}

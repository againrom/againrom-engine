package archtest

import (
	"os"
	"strings"
	"testing"
)

// TestCheckPartyItemHomeNamesEachShape plants each conversion and pack
// replacement outside the home, in every spelling that reaches the function,
// and a second graph follower inside it.
func TestCheckPartyItemHomeNamesEachShape(t *testing.T) {
	const shop = "package secondshop\nfunc KeptUnit(a, b int) int { return a }\nfunc MissionUnit(a int) int { return a }\n" +
		"func ownImplementation() int { return MissionUnit(1) }\n"
	const homeHead = "package game\nimport \"againrom/pkg/secondshop\"\ntype CampaignSession struct{}\n" +
		"func memberWithPack(m int) int { return m }\n" +
		"func (s *CampaignSession) followHeroPack() {}\n" +
		"var kept = secondshop.KeptUnit\n"
	const homeCall = "func (s *CampaignSession) a() { s.followHeroPack() }\n"
	cases := []struct {
		name  string
		other string
		home  string
		want  string
	}{
		{"a clean tree", "package game\n", homeCall, ""},
		{"a kept conversion outside", "package game\nimport \"againrom/pkg/secondshop\"\nfunc a() { secondshop.KeptUnit(1, 2) }\n", homeCall, "secondshop.KeptUnit outside"},
		{"a conversion through an import alias", "package game\nimport ss \"againrom/pkg/secondshop\"\nfunc a() { ss.KeptUnit(1, 2) }\n", homeCall, "secondshop.KeptUnit outside"},
		{"a mission conversion passed as a value", "package game\nimport \"againrom/pkg/secondshop\"\nvar f = secondshop.MissionUnit\n", homeCall, "secondshop.MissionUnit outside"},
		{"a pack replacement outside", "package game\nfunc a() { memberWithPack(1) }\n", homeCall, "memberWithPack outside"},
		{"a pack replacement as a function value", "package game\nvar g = memberWithPack\n", homeCall, "memberWithPack outside"},
		{"a graph follower outside", "package game\nfunc (s *CampaignSession) b() { s.followHeroPack() }\n", homeCall, "followHeroPack outside"},
		{"a graph follower as a method value", "package game\nfunc (s *CampaignSession) b() { f := s.followHeroPack; f() }\n", homeCall, "followHeroPack outside"},
		{"a second graph call in the home", "package game\n", homeCall + "func (s *CampaignSession) b() { s.followHeroPack() }\n", "2 times"},
		{"no graph call in the home", "package game\n", "", "0 times"},
	}
	for _, c := range cases {
		module := checkedFixture(t,
			fixturePackage{path: "againrom/pkg/secondshop", files: map[string]string{"pkg/secondshop/shop.go": shop}},
			fixturePackage{path: "againrom/pkg/game", files: map[string]string{PartyItemHomeFile: homeHead + c.home, "pkg/game/x.go": c.other}})
		got := strings.Join(CheckPartyItemHome(module), "\n")
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
	module, err := loadTypeCheckedModule(root)
	if err != nil {
		t.Fatal(err)
	}
	for _, finding := range CheckPartyItemHome(module) {
		t.Error(finding)
	}
}

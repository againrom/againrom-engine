package archtest

import (
	"os"
	"strings"
	"testing"
)

// TestCheckHireNamesEveryShape drives the scan with one synthetic source per
// shape and its admitted twin.
func TestCheckHireNamesEveryShape(t *testing.T) {
	cases := []struct {
		name, file, src, want string
	}{
		{"a bank hire slot index", "pkg/game/a.go",
			"package game\nfunc f(c *secondCampaign, id int) bool { return c.bank[511+id] > 0 }\n", "f indexes a hire slot of the bank"},
		{"a dereferenced bank slot", "pkg/game/a.go",
			"package game\nfunc f(bank *[1024]int32) int32 { return (*bank)[552] }\n", "f indexes a hire slot of the bank"},
		{"a bank slot outside the range", "pkg/game/a.go",
			"package game\nfunc f(c *secondCampaign) int32 { return c.bank[768] + c.bank[510] + c.bank[572] }\n", ""},
		{"a hire slot write", "pkg/game/a.go",
			"package game\nfunc f(c *secondCampaign) { sim.SetScenario(&c.bank, 532, 1) }\n", "f sets a hire slot of the bank"},
		{"another scenario write", "pkg/game/a.go",
			"package game\nfunc f(c *secondCampaign) { sim.SetScenario(&c.bank, 772, 1) }\n", ""},
		{"a hire slot constant", "pkg/game/a.go",
			"package game\nfunc f(c *secondCampaign, id int) int32 { return c.bank[secondHireSelection+id] }\n", "f names the hire slot secondHireSelection"},
		{"the hire rows", "pkg/game/hire.go",
			"package game\nconst secondHireTown = 552\nfunc f(c *secondCampaign) int32 { return c.bank[511] }\n", ""},
		{"a retired producer", "pkg/game/a.go",
			"package game\nfunc buildMercenarySquad() {}\n", "buildMercenarySquad is a retired second hire producer"},
		{"a retired producer in the hire rows", "pkg/game/hire.go",
			"package game\nfunc (c *secondCampaign) hireOrFire(npc int) string { return \"\" }\n", "hireOrFire is a retired second hire producer"},
		{"a hire flag write", "pkg/game/a.go",
			"package game\nfunc (t *Town) hire(typ int) { t.mercHired[typ] = true }\n", "hire writes a first-game hire flag"},
		{"a hire flag address", "pkg/game/a.go",
			"package game\nfunc (t *Town) hire(typ int) *bool { return &t.mercHired[typ] }\n", "hire takes a first-game hire flag's address"},
		{"the travel restore", "pkg/game/towntravel.go",
			"package game\nfunc (t *Town) restoreTowns() { t.mercHired[1] = true }\n", ""},
		{"the squad deal", "pkg/game/tavern.go",
			"package game\nfunc (t *Town) squadDeal(typ int) hireDeal { var d hireDeal; d.held = &t.mercHired[typ]; return d }\n", ""},
		{"a hire flag read", "pkg/game/a.go",
			"package game\nfunc (t *Town) hired(typ int) bool { return t.mercHired[typ] }\n", ""},
		{"a hire spec", "pkg/game/a.go",
			"package game\nvar s = mapload.CampaignActorSpec{Identity: 1, Hire: true}\n", "package scope sets Hire of a campaign actor spec"},
		{"a mercenary spec", "pkg/game/a.go",
			"package game\nfunc f() { _ = mapload.CampaignActorSpec{MercenaryType: 3} }\n", "f sets MercenaryType of a campaign actor spec"},
		{"a hire spec field", "pkg/game/a.go",
			"package game\nfunc f(s mapload.CampaignActorSpec) { s.Hire = true }\n", "f sets Hire of a campaign actor spec"},
		{"a character spec", "pkg/mapload/joined.go",
			"package mapload\nfunc f() { _ = CampaignActorSpec{Profile: CampaignActorProfile{PlayerCharacter: true}} }\n", ""},
		{"a second transaction method", "pkg/game/hire.go",
			"package game\nfunc (d hireDeal) refund() {}\n", "hireDeal.refund is a second hire transaction"},
		{"the transaction elsewhere", "pkg/game/a.go",
			"package game\nfunc (d *hireDeal) run() {}\n", "hireDeal.run is a second hire transaction"},
		{"the transaction", "pkg/game/hire.go",
			"package game\nfunc (d hireDeal) run() (string, bool, []string) { return \"\", false, nil }\n", ""},
		{"a test", "pkg/game/a_test.go",
			"package game\nfunc f(c *secondCampaign) { c.bank[512] = 1 }\n", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var got []string
			for _, v := range CheckHire(map[string]string{tc.file: tc.src}) {
				got = append(got, v.Reason)
			}
			if tc.want == "" {
				if len(got) != 0 {
					t.Fatalf("CheckHire = %v, want none", got)
				}
				return
			}
			if len(got) == 0 || !strings.Contains(strings.Join(got, "\n"), tc.want) {
				t.Fatalf("CheckHire = %v, want one naming %q", got, tc.want)
			}
		})
	}
}

// TestHireHasOneProducerAndOneTransaction is the live-tree half.
func TestHireHasOneProducerAndOneTransaction(t *testing.T) {
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
	for _, want := range []string{"func hireMembers(", "func (d hireDeal) run("} {
		if !strings.Contains(files[hireFile], want) {
			t.Fatalf("%s holds no %q; the scan guards nothing", hireFile, want)
		}
	}
	for _, v := range CheckHire(files) {
		t.Errorf("%s: %s", v.From, v.Reason)
	}
}

package archtest

import (
	"os"
	"strings"
	"testing"
)

const sleepRunner = `package game
func (t *townScreen) sleep() bool { return t.campaignTown().townShop().restock(restockInput{}) }
func sleepButton(name string) bool { return name == "sleep" }
`

func wantSleepFindings(t *testing.T, file, src string, allow map[string]int, want ...string) {
	t.Helper()
	var got []string
	for _, v := range sleepSitesAllowing(map[string]string{SleepFile: sleepRunner, file: src}, allow) {
		got = append(got, v.From)
	}
	if strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Fatalf("findings = %q, want %q", got, want)
	}
}

func TestSleepRestockCallIsFound(t *testing.T) {
	wantSleepFindings(t, "pkg/game/a.go", `package game
func (t *townScreen) a(s *Shop, c *secondCampaign) {
	s.RestockWith(nil, 1, nil)
	c.shop.restock(restockInput{})
	t.restockShop()
}
`, map[string]int{}, "pkg/game/a.go:3 restock-call", "pkg/game/a.go:4 restock-call")
}

func TestSleepStockDrawIsFound(t *testing.T) {
	src := `package game
func a(c *secondCampaign) {
	c.shop.drawStock(secondshop.Tables{}, 1, nil)
	c.shop.enter(secondshop.Tables{}, 1, nil, nil)
}
`
	wantSleepFindings(t, "pkg/game/a.go", src, map[string]int{}, "pkg/game/a.go:3 stock-draw")
	wantSleepFindings(t, "pkg/game/a.go", src, map[string]int{"pkg/game/a.go stock-draw": 1})
}

func TestSleepControlNameIsFound(t *testing.T) {
	wantSleepFindings(t, "pkg/game/a.go", `package game
func a(name string) int {
	switch name {
	case "sleep":
		return 0
	case "Sleep", "hire":
		return 1
	}
	if name == `+"`sleep`"+` {
		return 2
	}
	return 3
}
`, map[string]int{}, "pkg/game/a.go:4 sleep-name", "pkg/game/a.go:9 sleep-name")
}

func TestSleepRunnerAndOtherTreesAreExempt(t *testing.T) {
	wantSleepFindings(t, "cmd/a/main.go", `package main
func main() { shop.restock(); _ = "sleep" }
`, map[string]int{})
	if got := sleepSitesAllowing(map[string]string{SleepFile: sleepRunner}, map[string]int{}); len(got) != 0 {
		t.Fatalf("the runner itself = %v, want none", got)
	}
	if got := sleepSitesAllowing(map[string]string{"pkg/game/a.go": "package game\n"}, map[string]int{}); len(got) != 1 ||
		!strings.Contains(got[0].Reason, "missing") {
		t.Fatalf("no runner = %v, want one finding naming it missing", got)
	}
}

// TestSleepIsTheOneRunner is the live-tree half.
func TestSleepIsTheOneRunner(t *testing.T) {
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
	for _, v := range SleepSites(files) {
		t.Errorf("%s", v)
	}
}

package archtest

import (
	"os"
	"strings"
	"testing"
)

// castRequestBoundary is a synthetic boundary file declaring two refusals.
const castRequestBoundary = `package sim
const (
	refusalCasterBusy = "caster action busy"
	refusalNoMana     = "insufficient mana"
)
func (w *World) actorCastBusy(i int) bool { return w.orderStateAt(i).castBusy() }
func isManaRefusal(r string) bool { return r == refusalNoMana }
`

func castRequestFindings(t *testing.T, file, src string, allow map[string]int) []string {
	t.Helper()
	var got []string
	for _, v := range castRequestSitesAllowing(map[string]string{CastRequestFile: castRequestBoundary, file: src}, allow) {
		got = append(got, v.From)
	}
	return got
}

func wantCastRequestFindings(t *testing.T, file, src string, want ...string) {
	t.Helper()
	got := castRequestFindings(t, file, src, map[string]int{})
	if strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Fatalf("findings = %q, want %q", got, want)
	}
}

func TestCastRequestFoldCallIsFound(t *testing.T) {
	wantCastRequestFindings(t, "pkg/sim/a.go", `package sim
func (w *World) a(i int, s orderState) bool {
	return w.actorCastBusy(i) || w.strikeHoldsCast(i, "") || s.castBusy() ||
		w.bookSpellAdmissionRange(i, 0, 1, false, 0, false) == "" || w.castHolds(i, 0)
}
`, "pkg/sim/a.go:3 cast-fold", "pkg/sim/a.go:3 cast-fold", "pkg/sim/a.go:3 cast-fold", "pkg/sim/a.go:4 cast-fold")
}

func TestCastRequestRefusalCompareIsFound(t *testing.T) {
	wantCastRequestFindings(t, "pkg/game/a.go", `package game
func a(r string) bool {
	return r == "insufficient mana" || refusalCasterBusy != r || r == "" || r == "some other text"
}
`, "pkg/game/a.go:3 refusal-test", "pkg/game/a.go:3 refusal-test")
}

func TestCastRequestRefusalCaseIsFound(t *testing.T) {
	wantCastRequestFindings(t, "pkg/sim/a.go", `package sim
func a(r string) int {
	switch r {
	case "":
		return 0
	case refusalNoMana, "caster action busy":
		return 1
	}
	return 2
}
`, "pkg/sim/a.go:6 refusal-test", "pkg/sim/a.go:6 refusal-test")
}

func TestCastRequestBoundaryAndOtherPackagesAreExempt(t *testing.T) {
	wantCastRequestFindings(t, "pkg/game/a.go", `package game
func (f *FrontEnd) a(i int) bool { return f.actorCastBusy(i) }
`)
	if got := castRequestSitesAllowing(map[string]string{CastRequestFile: castRequestBoundary}, map[string]int{}); len(got) != 0 {
		t.Fatalf("the boundary itself = %v, want none", got)
	}
	if got := castRequestSitesAllowing(map[string]string{"pkg/sim/a.go": "package sim\n"}, map[string]int{}); len(got) != 1 ||
		!strings.Contains(got[0].Reason, "missing") {
		t.Fatalf("no boundary = %v, want one finding naming it missing", got)
	}
}

func TestCastRequestAllowListAdmitsItsCount(t *testing.T) {
	src := `package sim
func (w *World) a(i int) bool { return w.actorCastBusy(i) || w.actorActionBusy(i) }
`
	got := castRequestFindings(t, "pkg/sim/a.go", src, map[string]int{"pkg/sim/a.go cast-fold": 1})
	if strings.Join(got, "\n") != "pkg/sim/a.go:2 cast-fold" {
		t.Fatalf("findings = %q, want the second call only", got)
	}
}

// TestCastRequestIsTheOneBusyBoundary is the live-tree half.
func TestCastRequestIsTheOneBusyBoundary(t *testing.T) {
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
	for _, v := range CastRequestSites(files) {
		t.Errorf("%s", v)
	}
}

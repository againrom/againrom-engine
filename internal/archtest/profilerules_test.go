package archtest

import (
	"os"
	"regexp"
	"strings"
	"testing"
)

// A rule the game profile picks a body for is a rule row: no pkg/sim source
// names a flag that chose between two bodies under a branch.
func TestProfileRulesAreRows(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	root, err := FindModuleRoot(wd)
	if err != nil {
		t.Fatal(err)
	}
	files, err := LoadRandomnessSources(root)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := files["pkg/sim/gameprofile.go"]; !ok {
		t.Fatal("pkg/sim/gameprofile.go is missing; the scan guards nothing")
	}
	flag := regexp.MustCompile(`\b(stopRefusedFlee|classRetreatMode|roundUphillTowardZero|countedOutcome|autoCastSideSight|savedProjectileAim|withdrawMoverPoint|victimHolds|raiseAtNext|packedStep|effectsFirst)\b`)
	for path, src := range files {
		if strings.HasPrefix(path, "pkg/sim/") {
			if m := flag.FindString(src); m != "" {
				t.Errorf("%s names %s, a flag that chose a rule's body; hold the rule as a row", path, m)
			}
		}
	}
}

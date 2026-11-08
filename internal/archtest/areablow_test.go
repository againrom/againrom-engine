package archtest

import (
	"os"
	"strings"
	"testing"
)

// An area blow's answer on its victim has one builder, answerAreaBlow in
// pkg/sim/celleffect.go: the live area path and the saved replay both call
// it, and only it reads the profile's blowOrdersAttack.
func TestAreaBlowsAnswerThroughOneHelper(t *testing.T) {
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
	for _, f := range []string{"pkg/sim/celleffect.go", "pkg/sim/savedcontinuation.go"} {
		src, ok := files[f]
		if !ok {
			t.Fatalf("%s is missing; the scan guards nothing", f)
		}
		if n := strings.Count(src, "w.answerAreaBlow("); n != 1 {
			t.Errorf("%s calls answerAreaBlow %d times, want once", f, n)
		}
	}
	for f, src := range files {
		n := strings.Count(src, "game.blowOrdersAttack")
		if f == "pkg/sim/celleffect.go" && n != 1 || f != "pkg/sim/celleffect.go" && n != 0 {
			t.Errorf("%s reads blowOrdersAttack %d times; only answerAreaBlow reads it", f, n)
		}
	}
}

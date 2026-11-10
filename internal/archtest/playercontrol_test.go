package archtest

import (
	"os"
	"regexp"
	"strings"
	"testing"
)

// The player-unit test's builder, pkg/sim/playerparticipants.go, states the
// owner-0 policy once; no other pkg/sim line adds an owner-0 test to a call.
var (
	playerUnitBuilder = "pkg/sim/playerparticipants.go"
	playerUnitCall    = regexp.MustCompile(`\b(playerControlledUnit|computerControlledUnit|playerControlledSlot|humanParticipantUnit|humanParticipantSlot)\(`)
	ownerZeroTest     = regexp.MustCompile(`Owner\s*(?:[!=]=|>)\s*0\b|\b0\s*[!=]=\s*[\w.()\[\]*]*Owner\b`)
)

func TestThePlayerUnitTestStatesItsOwnerZeroPolicyOnce(t *testing.T) {
	for _, line := range []string{"e.Owner != 0 && w.playerControlledUnit(e)", "if e.Owner == 0 || !w.playerControlledUnit(e) {"} {
		if !ownerZeroTest.MatchString(line) || !playerUnitCall.MatchString(line) {
			t.Fatalf("the scan misses %q", line)
		}
	}
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
	if !strings.Contains(files[playerUnitBuilder], "func (w *World) slotControl(") {
		t.Fatalf("%s holds no slotControl; the scan guards nothing", playerUnitBuilder)
	}
	for f, src := range files {
		if !strings.HasPrefix(f, "pkg/sim/") || f == playerUnitBuilder {
			continue
		}
		for n, line := range strings.Split(src, "\n") {
			if strings.Contains(line, "humanParticipant") {
				t.Errorf("%s:%d asks a retired player-unit test: %s", f, n+1, strings.TrimSpace(line))
			}
			if ownerZeroTest.MatchString(line) && playerUnitCall.MatchString(line) {
				t.Errorf("%s:%d composes an owner-0 test beside the player-unit test: %s", f, n+1, strings.TrimSpace(line))
			}
		}
	}
}

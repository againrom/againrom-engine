// Package firstgame lets a release test that reads the first game's data skip
// itself when the install under test is a second-game root.
package firstgame

import (
	"os"

	"againrom/pkg/base"
)

// Skipper is the part of testing.TB the skip needs.
type Skipper interface {
	Helper()
	Skip(args ...any)
}

// SkipOnSecondGame skips t when AGAINROM_ASSETS names a second-game install.
// An unset variable or a root that does not detect leaves the test to its own
// gate or failure.
func SkipOnSecondGame(t Skipper) {
	t.Helper()
	root := os.Getenv("AGAINROM_ASSETS")
	if root == "" {
		return
	}
	m, err := base.Detect(root, nil)
	if err == nil && !m.Profile.HasFirstData() {
		t.Skip("reads the first game's data; the root is a second-game install")
	}
}

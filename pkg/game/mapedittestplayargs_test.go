package game

import (
	"slices"
	"testing"

	"againrom/pkg/launch"
)

// A test play starts the game with the chicken launch flag.
func TestTestPlayArgsCarryTheChickenFlag(t *testing.T) {
	args, err := TestPlayArgs(t.TempDir(), t.TempDir(), 1, "male-fighter")
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Contains(args, "-"+launch.Chicken.Flag) && !slices.Contains(args, "--"+launch.Chicken.Flag) {
		t.Fatalf("args %q lack -%s", args, launch.Chicken.Flag)
	}
}

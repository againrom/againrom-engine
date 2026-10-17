package game

import (
	"strings"
	"testing"

	"againrom/pkg/ui"
)

// addSteps drives the open Add window to its end: a step named in values
// takes that text in place of its own, every step then takes Enter. A many
// step takes each of its "|"-separated values with an Enter and ends with
// the next. It fails when a step refuses its value.
func addSteps(t *testing.T, e *ui.MapEditor, values map[string]string) {
	t.Helper()
	if addStepsN(t, e, values, -1) {
		t.Fatal("the Add window did not end")
	}
}

// addStepsN answers the first n steps of the open Add window as addSteps
// does, all of them when n is negative, and reports whether the window is
// still open.
func addStepsN(t *testing.T, e *ui.MapEditor, values map[string]string, n int) bool {
	t.Helper()
	given := map[string]bool{}
	done := 0
	enter := func(key string) {
		t.Helper()
		if err := e.Key("enter"); err != nil {
			t.Fatal(err)
		}
		if _, note, ok := e.AddStep(); ok && note != "" {
			t.Fatalf("the Add window refused step %s: %s", key, note)
		}
	}
	for i := 0; i < 32; i++ {
		key, _, ok := e.AddStep()
		if !ok {
			return false
		}
		if done == n {
			return true
		}
		if v, ok := values[key]; ok && !given[key] {
			given[key] = true
			parts := strings.Split(v, "|")
			for _, p := range parts[:len(parts)-1] {
				_ = e.Key("clear")
				e.TextInput(p)
				enter(key)
			}
			_ = e.Key("clear")
			e.TextInput(parts[len(parts)-1])
		}
		enter(key)
		if next, _, ok := e.AddStep(); !ok || next != key {
			done++
		}
	}
	t.Fatal("the Add window did not end")
	return false
}

// addWindow opens the Add window with key and drives it to its end.
func addWindow(t *testing.T, e *ui.MapEditor, key string, values map[string]string) {
	t.Helper()
	if err := e.Key(key); err != nil {
		t.Fatal(err)
	}
	if _, _, ok := e.AddStep(); !ok {
		t.Fatalf("%s opened no Add window", key)
	}
	addSteps(t, e, values)
}

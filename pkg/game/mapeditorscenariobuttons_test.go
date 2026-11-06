package game

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"image"
	"slices"
	"strings"
	"testing"

	"againrom/pkg/ui"
)

// scenarioButtonClick presses the Scenario palette's button key.
func scenarioButtonClick(t *testing.T, e *ui.MapEditor, key string) {
	t.Helper()
	scenarioOpen(t, e)
	r, ok := e.ListButtonKeyRect(key)
	trigClick(t, e, r, ok, "palette button "+key)
}

// The hash of the map scriptAddMap builds, and of the bytes the script Add
// window wrote for a trigger with every proposal accepted, taken from the
// window before it was removed.
const (
	scenarioMapSum        = "8d0f9bb5c2a07f0c0580ea3a8412713b825577ccb1a749213f08205f57406135"
	scenarioTriggerAddSum = "9b396a9a5ecbf126db4ccbed199e8f94006eca23ae0ce02328e7ae8c080e2251"
)

func sumOf(b []byte) string { return fmt.Sprintf("%x", sha256.Sum256(b)) }

// The Scenario palette lists the rows of the tree's Scenario branch in the
// tree's order and words, one step less indented, from row 0; no New trigger
// row stands first. Every row shows at 1920x1080, and a click on a row
// selects and opens what a click on its tree row does. EN and RU.
func TestMapEditorScenarioPaletteListsOnlyTheRecordRows(t *testing.T) {
	b := scriptAddMap(t)
	window := image.Rect(0, 0, 1920, 1080)
	for _, l := range scriptAddLangs {
		t.Run(l.name, func(t *testing.T) {
			e, _, w := scriptAddEditor(t, l, b)
			_, tree := scenarioBranch(t, e)
			var want []string
			for _, line := range tree[1:] {
				want = append(want, strings.TrimPrefix(line, "  "))
			}
			scenarioOpen(t, e)
			got := e.ListLines()
			if !slices.Equal(got, want) || slices.Contains(got, w("editor.tool.newtrigger")) {
				t.Fatalf("the palette lists\n%s\nwant\n%s", strings.Join(got, "\n"), strings.Join(want, "\n"))
			}
			for n := range want {
				if r, ok := e.ListRowRect(n); !ok || !r.In(window) {
					t.Fatalf("palette row %d shows %v at %v", n, ok, r)
				}
			}
			members := 0
			for k := 0; ; k++ {
				x, _, _ := scriptAddEditor(t, l, b)
				showTree(x)
				row := x.TreeMemberRow("scripts", k)
				if row < 0 {
					break
				}
				members++
				branch := x.TreeBranchRow("scripts")
				r, ok := x.TreeRowRect(row)
				trigClick(t, x, r, ok, "tree row")
				y, _, _ := scriptAddEditor(t, l, b)
				scenarioOpen(t, y)
				r, ok = y.ListRowRect(row - branch - 1)
				trigClick(t, y, r, ok, "palette row")
				if y.Selected() != x.Selected() || y.Selected() < 0 || y.WizardShown() != x.WizardShown() || trigLines(y) != trigLines(x) {
					t.Fatalf("palette row %d opens %d\n%s\nthe tree row opens %d\n%s", row-branch-1, y.Selected(), trigLines(y), x.Selected(), trigLines(x))
				}
			}
			if members == 0 {
				t.Fatal("the Scenario branch holds no member row")
			}
		})
	}
}

// The palette's buttons are the drop zone switch and New trigger, in
// that row; Add is gone.
func TestMapEditorScenarioPaletteButtonKeys(t *testing.T) {
	b := scriptAddMap(t)
	window := image.Rect(0, 0, 1920, 1080)
	for _, l := range scriptAddLangs {
		e, _, _ := scriptAddEditor(t, l, b)
		scenarioOpen(t, e)
		var rects []image.Rectangle
		for _, k := range []string{"drop", "newtrigger"} {
			r, ok := e.ListButtonKeyRect(k)
			if !ok || !r.In(window) {
				t.Fatalf("%s: button %s shows %v at %v", l.name, k, ok, r)
			}
			rects = append(rects, r)
		}
		if _, ok := e.ListButtonKeyRect("add"); ok {
			t.Fatalf("%s: the palette still has Add", l.name)
		}
		if _, ok := e.ListButtonKeyRect("newvariable"); ok {
			t.Fatalf("%s: the palette still has New variable", l.name)
		}
		for i := 1; i < len(rects); i++ {
			if rects[i].Overlaps(rects[i-1]) || rects[i].Min.X < rects[i-1].Max.X || rects[i].Min.Y != rects[i-1].Min.Y {
				t.Fatalf("%s: buttons %d and %d at %v and %v", l.name, i-1, i, rects[i-1], rects[i])
			}
		}
	}
}

// One click on New trigger writes the bytes the script Add window wrote for
// a trigger with every proposal accepted, and opens the trigger in the
// wizard, its name field focused with the name selected. EN and RU.
func TestMapEditorScenarioNewTriggerButtonAddsInOneClick(t *testing.T) {
	b := scriptAddMap(t)
	if sumOf(b) != scenarioMapSum {
		t.Fatal("the test map changed")
	}
	triggers := len(wizScript(t, b).Triggers)
	for _, l := range scriptAddLangs {
		t.Run(l.name, func(t *testing.T) {
			e, _, _ := scriptAddEditor(t, l, b)
			scenarioButtonClick(t, e, "newtrigger")
			if got := sumOf(e.Document().SourceBytes); got != scenarioTriggerAddSum {
				t.Fatalf("the button wrote %s, the Add window wrote %s", got, scenarioTriggerAddSum)
			}
			freshTriggerShown(t, e, triggers+1, true)
			if _, _, open := e.AddStep(); open {
				t.Fatal("the button opened an Add window")
			}
			r, ok := e.PopupButtonRect(1)
			trigClick(t, e, r, ok, "Cancel")
			if err := e.Key("undo"); err != nil || !bytes.Equal(e.Document().SourceBytes, b) {
				t.Fatalf("one undo left the trigger: %v", err)
			}
		})
	}
}

// No route opens the script Add window: the A key on the palette, the
// wizard's New (which adds the one-click trigger) and F4.
func TestMapEditorNoRouteOpensTheScriptAddWindow(t *testing.T) {
	b := scriptAddMap(t)
	triggers := len(wizScript(t, b).Triggers)
	for _, l := range scriptAddLangs {
		e, _, _ := scriptAddEditor(t, l, b)
		scenarioOpen(t, e)
		_ = e.Key("player")
		if _, _, open := e.AddStep(); open {
			t.Fatalf("%s: the A key opened the Add window", l.name)
		}
		e.Select(trigRecord(t, e, "Trigger", 0), true)
		r, ok := e.PopupButtonRect(2)
		trigClick(t, e, r, ok, "New")
		if _, _, open := e.AddStep(); open {
			t.Fatalf("%s: the wizard's New opened the Add window", l.name)
		}
		freshTriggerShown(t, e, triggers+1, true)
		pressF4(t, e)
		if _, _, open := e.AddStep(); open {
			t.Fatalf("%s: F4 opened the Add window", l.name)
		}
	}
}

func TestMapEditorScenarioWindowsReturnAfterSwitch(t *testing.T) {
	b := scriptAddMap(t)
	for _, l := range scriptAddLangs {
		t.Run(l.name, func(t *testing.T) {
			e, _, _ := scriptAddEditor(t, l, b)
			scenarioButtonClick(t, e, "newtrigger")
			before := slices.Clone(e.Document().SourceBytes)
			e.TextInput("discard me")
			r, ok := e.BarRect("units")
			trigClick(t, e, r, ok, "Units")
			r, ok = e.BarRect("scripts")
			trigClick(t, e, r, ok, "Scenario")
			if !bytes.Equal(e.Document().SourceBytes, before) || len(e.ListLines()) == 0 || strings.Contains(trigLines(e), "discard me") {
				t.Fatal("return lost the palette, kept a draft or wrote the map")
			}
			if !e.WizardShown() {
				t.Fatal("the trigger wizard did not return")
			}
		})
	}
}

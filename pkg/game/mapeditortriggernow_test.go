package game

import (
	"bytes"
	"image"
	"slices"
	"strings"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"

	"againrom/pkg/ui"
)

// scenarioOpen opens the Scenario palette as F2 does.
func scenarioOpen(t *testing.T, e *ui.MapEditor) {
	t.Helper()
	if err := e.Key("scripts"); err != nil {
		t.Fatal(err)
	}
	if len(e.ListLines()) == 0 {
		t.Fatalf("the Scenario palette lists no row:\n%s", trigLines(e))
	}
}

// scenarioBranch is the row of the tree's Scenario branch and the lines of
// that branch, its own first, the left panel shown.
func scenarioBranch(t *testing.T, e *ui.MapEditor) (int, []string) {
	t.Helper()
	showTree(e)
	lines := e.TreeLines()
	b := e.TreeBranchRow("scripts")
	if b < 0 || e.TreeMemberRow("scripts", 0) < 0 {
		t.Fatalf("the tree shows no open Scenario branch:\n%s", strings.Join(lines, "\n"))
	}
	end := b + 1
	for end < len(lines) && strings.HasPrefix(lines[end], "  ") {
		end++
	}
	return b, lines[b:end]
}

// newTriggerClick presses the Scenario palette's New trigger row.
func newTriggerClick(t *testing.T, e *ui.MapEditor) {
	t.Helper()
	scenarioOpen(t, e)
	r, ok := e.ListRowRect(0)
	trigClick(t, e, r, ok, "New trigger")
}

// pressF4 runs one editor frame in which F4 goes down.
func pressF4(t *testing.T, e *ui.MapEditor) {
	t.Helper()
	ui.ProbeSetInput(e, func() ui.ProbeInput { return ui.ProbeInput{Key: ebiten.KeyF4, HasKey: true} })
	if err := e.Update(); err != nil {
		t.Fatal(err)
	}
}

// freshTriggerShown checks that the newest trigger's wizard is the one
// window open, its name field focused with the whole name selected.
func freshTriggerShown(t *testing.T, e *ui.MapEditor, triggers int) {
	t.Helper()
	sc := wizScript(t, e.Document().SourceBytes)
	if len(sc.Triggers) != triggers {
		t.Fatalf("the map holds %d triggers, want %d", len(sc.Triggers), triggers)
	}
	if !e.WizardShown() {
		t.Fatalf("the new trigger's wizard is not shown:\n%s", trigLines(e))
	}
	if w := e.OpenWindows(); !slices.Equal(w, []string{"object"}) {
		t.Fatalf("windows open %q, want the wizard alone", w)
	}
	name := string(sc.Triggers[len(sc.Triggers)-1].Name)
	if field, from, to := e.FieldFocus(); field != wizField(t, e, "name") || from != 0 || to != len([]rune(name)) {
		t.Fatalf("focus field %d selection %d-%d, want the name field %d with %q selected", field, from, to, wizField(t, e, "name"), name)
	}
}

// The Scenario palette lists New trigger, then the rows of the tree's
// Scenario branch in the tree's order and words, one step less indented;
// every row shows at 1920x1080, and a click on a record row selects and
// opens what a click on its tree row does. EN and RU.
func TestMapEditorScenarioPaletteListsTheTreesRows(t *testing.T) {
	b := scriptAddMap(t)
	window := image.Rect(0, 0, 1920, 1080)
	for _, l := range scriptAddLangs {
		t.Run(l.name, func(t *testing.T) {
			e, _, w := scriptAddEditor(t, l, b)
			branch, tree := scenarioBranch(t, e)
			want := []string{w("editor.tool.newtrigger")}
			for _, line := range tree[1:] {
				want = append(want, strings.TrimPrefix(line, "  "))
			}
			scenarioOpen(t, e)
			if got := e.ListLines(); !slices.Equal(got, want) {
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
				r, ok := x.TreeRowRect(row)
				trigClick(t, x, r, ok, "tree row")
				y, _, _ := scriptAddEditor(t, l, b)
				scenarioOpen(t, y)
				r, ok = y.ListRowRect(row - branch)
				trigClick(t, y, r, ok, "palette row")
				if y.Selected() != x.Selected() || y.Selected() < 0 || y.WizardShown() != x.WizardShown() || trigLines(y) != trigLines(x) {
					t.Fatalf("palette row %d opens %d %v\n%s\nthe tree row opens %d %v\n%s", row-branch, y.Selected(), y.WizardShown(), trigLines(y), x.Selected(), x.WizardShown(), trigLines(x))
				}
			}
			if members == 0 {
				t.Fatal("the Scenario branch holds no member row")
			}
		})
	}
}

// One click on New trigger writes the bytes the Add window's trigger steps
// write with their proposals accepted, and opens the trigger at once in the
// wizard, the one window open, its name field focused with the name
// selected. The Add window still asks its trigger steps. EN and RU.
func TestMapEditorNewTriggerTakesOneClick(t *testing.T) {
	b := scriptAddMap(t)
	triggers := len(wizScript(t, b).Triggers)
	for _, l := range scriptAddLangs {
		t.Run(l.name, func(t *testing.T) {
			add, _, _ := scriptAddEditor(t, l, b)
			openScriptAdd(t, add)
			if keys := addWalk(t, add, map[string]string{"kind": "2"}, ""); !slices.Equal(keys, []string{"kind", "name", "once"}) {
				t.Fatalf("the Add window's trigger asked %q", keys)
			}
			e, _, _ := scriptAddEditor(t, l, b)
			newTriggerClick(t, e)
			if got := e.Document().SourceBytes; bytes.Equal(got, b) || !bytes.Equal(got, add.Document().SourceBytes) {
				t.Fatalf("one click wrote %d bytes, the Add window %d, the map %d", len(got), len(add.Document().SourceBytes), len(b))
			}
			freshTriggerShown(t, e, triggers+1)
			if !e.Dirty() {
				t.Fatal("the map is not dirty after the one-click trigger")
			}
		})
	}
}

// Typing replaces the selected name and Apply writes it; one undo takes the
// named trigger out and one redo puts it back. Cancel leaves the trigger in
// the map; one undo takes it out and one redo puts it back. The palette
// stays closed when the wizard closes. EN and RU.
func TestMapEditorOneClickTriggerUndoesInOneStep(t *testing.T) {
	b := scriptAddMap(t)
	triggers := len(wizScript(t, b).Triggers)
	for _, l := range scriptAddLangs {
		t.Run(l.name, func(t *testing.T) {
			e, _, _ := scriptAddEditor(t, l, b)
			newTriggerClick(t, e)
			e.TextInput("Gate")
			wizApply(t, e)
			named := slices.Clone(e.Document().SourceBytes)
			sc := wizScript(t, named)
			if len(sc.Triggers) != triggers+1 || string(sc.Triggers[triggers].Name) != "Gate" || !e.Dirty() {
				t.Fatalf("Apply wrote %d triggers, the last %q, dirty %v", len(sc.Triggers), sc.Triggers[len(sc.Triggers)-1].Name, e.Dirty())
			}
			if w := e.OpenWindows(); slices.Contains(w, "list") {
				t.Fatalf("Apply reopened the palette: %q", w)
			}
			step := func(key string, want []byte) {
				t.Helper()
				if err := e.Key(key); err != nil {
					t.Fatal(err)
				}
				if !bytes.Equal(e.Document().SourceBytes, want) {
					t.Fatalf("one %s left %d triggers", key, len(wizScript(t, e.Document().SourceBytes).Triggers))
				}
			}
			step("undo", b)
			step("redo", named)

			c, _, _ := scriptAddEditor(t, l, b)
			newTriggerClick(t, c)
			added := slices.Clone(c.Document().SourceBytes)
			r, ok := c.PopupButtonRect(1)
			trigClick(t, c, r, ok, "Cancel")
			if len(c.OpenWindows()) != 0 || !bytes.Equal(c.Document().SourceBytes, added) {
				t.Fatalf("Cancel left windows %q, the trigger kept %v", c.OpenWindows(), bytes.Equal(c.Document().SourceBytes, added))
			}
			e = c
			step("undo", b)
			step("redo", added)
		})
	}
}

// F4 from the map view does what the click does. While the name field has
// focus F4 adds nothing and leaves the field as it stands, and typing goes
// to the field. EN and RU.
func TestMapEditorNewTriggerKeyWorksFromTheMap(t *testing.T) {
	b := scriptAddMap(t)
	triggers := len(wizScript(t, b).Triggers)
	for _, l := range scriptAddLangs {
		t.Run(l.name, func(t *testing.T) {
			e, _, _ := scriptAddEditor(t, l, b)
			if w := e.OpenWindows(); len(w) != 0 {
				t.Fatalf("the map view shows %q", w)
			}
			pressF4(t, e)
			freshTriggerShown(t, e, triggers+1)
			click, _, _ := scriptAddEditor(t, l, b)
			newTriggerClick(t, click)
			if !bytes.Equal(e.Document().SourceBytes, click.Document().SourceBytes) {
				t.Fatal("F4 wrote other bytes than the click")
			}
			// The Map panel a map opens with closes as the wizard opens.
			p := openHeroLabelMap(t, heroLabelInspector(t, l.profile, l.sel), b)
			p.Layout(1920, 1080)
			if !p.PanelOpen() {
				t.Fatal("the Map panel did not open with the map")
			}
			pressF4(t, p)
			freshTriggerShown(t, p, triggers+1)
			if p.PanelOpen() {
				t.Fatal("the Map panel stays open over the new trigger's wizard")
			}
			field, from, to := e.FieldFocus()
			pressF4(t, e)
			if !bytes.Equal(e.Document().SourceBytes, click.Document().SourceBytes) {
				t.Fatal("F4 in the name field added a trigger")
			}
			if f, a, z := e.FieldFocus(); f != field || a != from || z != to {
				t.Fatalf("F4 in the name field moved focus %d %d-%d to %d %d-%d", field, from, to, f, a, z)
			}
			e.TextInput("Gate")
			wizApply(t, e)
			sc := wizScript(t, e.Document().SourceBytes)
			if len(sc.Triggers) != triggers+1 || string(sc.Triggers[triggers].Name) != "Gate" {
				t.Fatalf("typing after F4 wrote %d triggers, the last %q", len(sc.Triggers), sc.Triggers[len(sc.Triggers)-1].Name)
			}
		})
	}
}

// The join with the add ends with the wizard session the click opened. A
// save between the add and the rename, and a Cancel followed by a later
// opening of the same trigger, each leave the rename its own history entry.
// EN and RU.
func TestMapEditorOneClickJoinEndsWithTheWizardSession(t *testing.T) {
	b := scriptAddMap(t)
	for _, l := range scriptAddLangs {
		t.Run(l.name, func(t *testing.T) {
			step := func(e *ui.MapEditor, key string, want []byte, what string) {
				t.Helper()
				if err := e.Key(key); err != nil {
					t.Fatal(err)
				}
				if !bytes.Equal(e.Document().SourceBytes, want) {
					t.Fatalf("%s: %s left %d triggers", what, key, len(wizScript(t, e.Document().SourceBytes).Triggers))
				}
			}
			// Save, then rename and Apply.
			e, _, _ := scriptAddEditor(t, l, b)
			newTriggerClick(t, e)
			added := slices.Clone(e.Document().SourceBytes)
			sr, sok := e.BarRect("save")
			trigClick(t, e, sr, sok, "Save")
			if e.Dirty() {
				t.Fatalf("Save left the map dirty (%s)", e.Message())
			}
			wizType(t, e, "name", "Gate")
			wizApply(t, e)
			renamed := slices.Clone(e.Document().SourceBytes)
			if bytes.Equal(renamed, added) {
				t.Fatal("Apply after the save wrote nothing")
			}
			step(e, "undo", added, "after a save")
			if e.Dirty() {
				t.Fatal("the saved map's undo state is dirty")
			}
			step(e, "undo", b, "after a save")
			step(e, "redo", added, "after a save")
			step(e, "redo", renamed, "after a save")

			// Cancel, then the same trigger reopened, renamed and applied.
			c, _, _ := scriptAddEditor(t, l, b)
			newTriggerClick(t, c)
			added = slices.Clone(c.Document().SourceBytes)
			sel := c.Selected()
			r, ok := c.PopupButtonRect(1)
			trigClick(t, c, r, ok, "Cancel")
			if len(c.OpenWindows()) != 0 {
				t.Fatalf("Cancel left windows %q", c.OpenWindows())
			}
			shown := false
			for row := 1; row < 64 && !shown; row++ {
				scenarioOpen(t, c)
				r, ok = c.ListRowRect(row)
				if !ok {
					break
				}
				trigClick(t, c, r, ok, "palette row")
				shown = c.Selected() == sel && c.WizardShown() && slices.Contains(c.OpenWindows(), "object")
				if !shown {
					closePalette(c)
					_ = c.Key("escape")
				}
			}
			if !shown {
				t.Fatalf("no palette row reopens the new trigger:\n%q", c.ListLines())
			}
			wizType(t, c, "name", "Later")
			wizApply(t, c)
			renamed = slices.Clone(c.Document().SourceBytes)
			if bytes.Equal(renamed, added) {
				t.Fatal("Apply after the reopening wrote nothing")
			}
			step(c, "undo", added, "after Cancel")
			step(c, "undo", b, "after Cancel")
		})
	}
}

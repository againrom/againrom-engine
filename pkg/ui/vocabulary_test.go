package ui

// 0146: the player's order vocabulary, driven through the SHIPPED front-end
// path — a whole App on the map screen, real key frames and real presses, with
// the seams the loader handed over as the only place an order can be observed.
//
// Nothing here calls a seam directly. What this story adds is which key does
// what to which selection and which seam a press leaves by, and both are
// properties of App.step: a test that reached past it would witness the closure
// and not the press.

import (
	"go/ast"
	"go/parser"
	"go/token"
	"reflect"
	"strings"
	"testing"
)

// vocOnMap is atOnMap's own harness, reused whole: an App parked on the map
// screen over a recording seam, with the camera at the origin and the same
// snapshot the attack cases use.
func vocOnMap(t *testing.T) (*App, *Viewer, *mapSeam) {
	t.Helper()
	return atOnMap(t)
}

// vocKey presses one of the four keys with the cursor over empty ground, so the
// key is the only thing the frame carries.
func vocKey(a *App, v *Viewer, set func(*appInput)) {
	x, y := cellPoint(v, atEmptyCol, atEmptyRow)
	in := atFrame(x, y)
	set(&in)
	a.step(in, atAt)
}

func TestTheStanceKeysOrderTheWholeSelection(t *testing.T) {
	for _, tc := range []struct {
		name  string
		set   func(*appInput)
		guard bool
	}{
		{"guard", func(in *appInput) { in.Guard = true }, true},
		{"stand ground", func(in *appInput) { in.StandGround = true }, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			a, v, seam := vocOnMap(t)
			v.sel = selection{atLoID, atDeadID, atHiID, 999}
			vocKey(a, v, tc.set)

			want := []stood{{entity: atLoID, guard: tc.guard}, {entity: atHiID, guard: tc.guard}}
			if !reflect.DeepEqual(seam.stances, want) {
				t.Fatalf("the seam took %+v, want %+v", seam.stances, want)
			}
			if len(seam.orders) != 0 || len(seam.attacks) != 0 || len(seam.marches) != 0 {
				t.Fatalf("a stance key also issued %d moves, %d attacks and %d marches",
					len(seam.orders), len(seam.attacks), len(seam.marches))
			}
		})
	}
}

// TestAStanceKeyWithNothingSelectedIssuesNothing is the stance keys' own
// empty case — the one shape every key on this screen shares.
func TestAStanceKeyWithNothingSelectedIssuesNothing(t *testing.T) {
	a, v, seam := vocOnMap(t)
	vocKey(a, v, func(in *appInput) { in.Guard = true })
	if len(seam.stances) != 0 {
		t.Fatalf("the seam took %+v with nothing selected, want nothing", seam.stances)
	}
}

func TestAnArmedAimedOrderSpendsOnTheNextSecondaryPress(t *testing.T) {
	for _, tc := range []struct {
		name   string
		set    func(*appInput)
		patrol bool
	}{
		{"patrol", func(in *appInput) { in.Patrol = true }, true},
		{"march", func(in *appInput) { in.March = true }, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			a, v, seam := vocOnMap(t)
			v.sel = selection{atLoID, atHiID}
			vocKey(a, v, tc.set)
			atPress(a, v, atEmptyCol, atEmptyRow)

			want := []marched{
				{entity: atLoID, patrol: tc.patrol, x: atEmptyCol, y: atEmptyRow},
				{entity: atHiID, patrol: tc.patrol, x: atEmptyCol, y: atEmptyRow},
			}
			if !reflect.DeepEqual(seam.marches, want) {
				t.Fatalf("the march seam took %+v, want %+v", seam.marches, want)
			}
			if len(seam.orders) != 0 {
				t.Fatalf("the same press also issued %d plain moves", len(seam.orders))
			}
		})
	}
}

func TestAnUnarmedPressIsStillThePlainMove(t *testing.T) {
	a, v, seam := vocOnMap(t)
	v.sel = selection{atLoID}
	atPress(a, v, atEmptyCol, atEmptyRow)

	if len(seam.marches) != 0 {
		t.Fatalf("an unarmed press reached the march seam: %+v", seam.marches)
	}
	want := []issued{{entity: atLoID, x: atEmptyCol, y: atEmptyRow}}
	if !reflect.DeepEqual(seam.orders, want) {
		t.Fatalf("the move seam took %+v, want %+v", seam.orders, want)
	}
}

// TestAnAimedOrderIsSpentByThePressWhateverItProduced is the arm's own spend
// rule, borrowed whole from the attack key: a tap that fell outside the map
// lowers it too, so the NEXT tap is a plain move rather than a march the
// player has forgotten he armed.
func TestAnAimedOrderIsSpentByThePressWhateverItProduced(t *testing.T) {
	a, v, seam := vocOnMap(t)
	v.sel = selection{atLoID}
	vocKey(a, v, func(in *appInput) { in.March = true })

	// A tap far outside the camera's extent. The point is off the map surface
	// altogether, so the hover cascade puts up no cursor at all and the click
	// reaches no arm (`AI-CLICK-050`): nothing is issued, and the arm is still
	// spent because it is the TAP that lowers it and not the order it produced.
	in := atFrame(-4000, -4000)
	in.PrimaryPressed, in.PrimaryReleased = true, true
	a.step(in, atAt)
	if len(seam.marches) != 0 || len(seam.orders) != 0 {
		t.Fatalf("a tap outside the extent issued %+v / %+v", seam.marches, seam.orders)
	}
	if v.aimedOrder() != commandNone {
		t.Fatal("the arm survived a tap that issued nothing")
	}

	atPress(a, v, atEmptyCol, atEmptyRow)
	if len(seam.marches) != 0 {
		t.Fatalf("the next press was still a march: %+v", seam.marches)
	}
	if len(seam.orders) != 1 {
		t.Fatalf("the next press issued %d plain moves, want 1", len(seam.orders))
	}
}

// TestAnAimedKeyTogglesAndReplaces is the arm's own key rule: the same key
// disarms, the other replaces.
func TestAnAimedKeyTogglesAndReplaces(t *testing.T) {
	a, v, _ := vocOnMap(t)
	v.sel = selection{atLoID}

	vocKey(a, v, func(in *appInput) { in.Patrol = true })
	if v.aimedOrder() != commandPatrol {
		t.Fatalf("aimed order is %d after the patrol key, want patrol", v.aimedOrder())
	}
	vocKey(a, v, func(in *appInput) { in.Patrol = true })
	if v.aimedOrder() != commandNone {
		t.Fatalf("aimed order is %d after a second patrol key, want none", v.aimedOrder())
	}
	vocKey(a, v, func(in *appInput) { in.Patrol = true })
	vocKey(a, v, func(in *appInput) { in.March = true })
	if v.aimedOrder() != commandSwarm {
		t.Fatalf("aimed order is %d after patrol then march, want march", v.aimedOrder())
	}
}

func TestTheArmsAreExclusive(t *testing.T) {
	t.Run("arming a command lowers the attack mode", func(t *testing.T) {
		a, v, _ := vocOnMap(t)
		v.sel = selection{atLoID}
		atArm(a, v)
		if !v.AttackArmed() {
			t.Fatal("setup: the attack did not arm")
		}
		vocKey(a, v, func(in *appInput) { in.March = true })
		if v.AttackArmed() {
			t.Error("the attack mode survived arming a command")
		}
	})

	t.Run("arming an attack lowers the command", func(t *testing.T) {
		a, v, _ := vocOnMap(t)
		v.sel = selection{atLoID}
		vocKey(a, v, func(in *appInput) { in.Patrol = true })
		atArm(a, v)
		if v.aimedOrder() != commandNone {
			t.Error("the command arm survived arming an attack")
		}
	})

	t.Run("a held modifier does not take a press the command armed", func(t *testing.T) {
		a, v, seam := vocOnMap(t)
		v.sel = selection{atLoID}
		vocKey(a, v, func(in *appInput) { in.March = true })

		// The modifier goes down and stays down — the level writer, which
		// re-raises the attack mode on every frame it is held.
		x, y := cellPoint(v, atFoeCol, atFoeRow)
		in := atFrame(x, y)
		in.AttackHeld = true
		in.PrimaryPressed, in.PrimaryReleased = true, true
		a.step(in, atAt)

		if len(seam.attacks) != 0 {
			t.Fatalf("the modifier took the press: %+v", seam.attacks)
		}
		if len(seam.marches) != 1 || seam.marches[0].patrol {
			t.Fatalf("the march seam took %+v, want one march", seam.marches)
		}
	})
}

// -------------------------------------------------------------------- AC-7

// TestTheFourKeysAreBoundOnceEach is AC-7: each of the order vocabulary's
// letters is named by exactly ONE binding in this package, so none of them
// is bound twice.
//
// IT READS THE SOURCE, which is the only way to ask the question: the
// bindings are Ebitengine calls and no test may open a window.
//
// DIV-237
func TestTheFourKeysAreBoundOnceEach(t *testing.T) {
	fset := token.NewFileSet()
	seen := map[string]int{}
	for _, name := range []string{"app.go", "viewer.go"} {
		f, err := parser.ParseFile(fset, name, nil, 0)
		if err != nil {
			t.Fatalf("parse %s: %v", name, err)
		}
		ast.Inspect(f, func(n ast.Node) bool {
			sel, ok := n.(*ast.SelectorExpr)
			if !ok {
				return true
			}
			pkg, ok := sel.X.(*ast.Ident)
			if !ok || pkg.Name != "ebiten" || !strings.HasPrefix(sel.Sel.Name, "Key") {
				return true
			}
			seen[strings.TrimPrefix(sel.Sel.Name, "Key")]++
			return true
		})
	}
	for _, key := range []string{"G", "T", "P", "S", "M", "E", "B", "Q", "J", "X"} {
		if seen[key] != 1 {
			t.Errorf("ebiten.Key%s is named %d times, want exactly 1 — it was bound before this story or is bound twice",
				key, seen[key])
		}
	}
}

func TestTheReadoutStatesWhichOrderIsArmed(t *testing.T) {
	for _, tc := range []struct {
		aimed uint8
		want  string
	}{
		{commandNone, readoutDisarmed},
		{commandPatrol, readoutPatrol},
		{commandSwarm, readoutSwarm},
		{commandMove, readoutMove},
	} {
		got, ok := readoutText(readoutSubject{Aimed: tc.aimed}, PanelFieldOrder)
		if !ok {
			t.Fatalf("the readout has no value for aimed order %d — the row would vanish", tc.aimed)
		}
		if got != tc.want {
			t.Errorf("aimed order %d reads %q, want %q", tc.aimed, got, tc.want)
		}
	}
}

// -------------------------------------------------------------------- AC-8

// TestAMapWithNoStanceSeamTakesEveryKey is AC-8: a loader that installs neither
// seam still drives the whole map screen, and the four keys reach nothing.
func TestAMapWithNoStanceSeamTakesEveryKey(t *testing.T) {
	a := newTestApp(t, appRows(3), okLoader(t))
	a.flow.screen = ScreenPicker
	a.step(appInput{Enter: true}, atAt)
	if a.Screen() != ScreenMap {
		t.Fatalf("setup: screen = %v, want ScreenMap", a.Screen())
	}
	mapAtScaleOne(a)
	v := a.flow.viewer
	v.SetEntities(atEntities())
	v.sel = selection{atLoID}

	for _, set := range []func(*appInput){
		func(in *appInput) { in.Guard = true },
		func(in *appInput) { in.StandGround = true },
		func(in *appInput) { in.Patrol = true },
		func(in *appInput) { in.March = true },
	} {
		vocKey(a, v, set)
		atPress(a, v, atEmptyCol, atEmptyRow)
	}
	if a.Screen() != ScreenMap {
		t.Fatalf("screen = %v after the four keys, want ScreenMap", a.Screen())
	}
}

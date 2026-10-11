package ui

import (
	"math"
	"slices"
	"strings"
	"testing"
	"time"

	"againrom/pkg/render/camera"
	"againrom/pkg/render/terrain"
)

const qolOwner = 3

// qolApp is a running map screen holding three units of the local player and
// one of another, with a two-spell book; every unit knows both spells.
func qolApp(t *testing.T) (*App, *Viewer) {
	t.Helper()
	a := spellbookTestApp(t)
	mapAtScaleOne(a)
	v := a.flow.viewer
	v.SetLocalOwner(qolOwner)
	v.SetEntities(qolEntities(0, 0, 0))
	v.SetSpellbook(1, sbBook())
	return a, v
}

// qolEntities are units 1, 2 and 3 of the local player, autocasting the three
// given spells, and unit 4 of player 9; every one knows spells 1 and 6.
func qolEntities(a1, a2, a3 uint32) []MapEntity {
	known := uint32(1)<<1 | uint32(1)<<6
	unit := func(id, owner, auto uint32) MapEntity {
		return MapEntity{ID: id, Life: LifeAlive, Owner: owner, SpellStateKnown: true, KnownSpells: known,
			CastCapable: true, AutoSpell: auto}
	}
	return []MapEntity{unit(1, qolOwner, a1), unit(2, qolOwner, a2), unit(3, qolOwner, a3), unit(4, 9, 0)}
}

func qolAutocastSink(v *Viewer) *[][2]uint32 {
	var sent [][2]uint32
	v.SetAutocastSink(func(entity, spell uint32) { sent = append(sent, [2]uint32{entity, spell}) })
	return &sent
}

// Ctrl+A over a mixed selection: any caster not on the selected spell turns
// the spell on for every caster; all on turns it off for all. The loss
// control is the primary-only toggle this replaces, which sent one command.
func TestCtrlATogglesAutocastForTheWholeSelection(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	for _, tc := range []struct {
		name       string
		auto       [3]uint32
		sel        selection
		spell      uint32
		want       [][2]uint32
		knowsOnly1 bool
	}{
		{"one off turns all on", [3]uint32{1, 0, 1}, selection{1, 2, 3}, 1, [][2]uint32{{1, 1}, {2, 1}, {3, 1}}, false},
		{"another spell counts as off", [3]uint32{1, 1, 6}, selection{1, 2, 3}, 1, [][2]uint32{{1, 1}, {2, 1}, {3, 1}}, false},
		{"all on turns all off", [3]uint32{1, 1, 1}, selection{1, 2, 3}, 1, [][2]uint32{{1, 0}, {2, 0}, {3, 0}}, false},
		{"another player's unit is left alone", [3]uint32{0, 0, 0}, selection{1, 4}, 1, [][2]uint32{{1, 1}}, false},
		{"a unit that does not know the spell is left alone", [3]uint32{0, 0, 0}, selection{1, 2}, 6, [][2]uint32{{1, 6}}, true},
		{"no spell selected and all on turns all off", [3]uint32{1, 6, 1}, selection{1, 2, 3}, 0, [][2]uint32{{1, 0}, {2, 0}, {3, 0}}, false},
		{"no spell selected and one off does nothing", [3]uint32{1, 0, 1}, selection{1, 2, 3}, 0, nil, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			a, v := qolApp(t)
			ents := qolEntities(tc.auto[0], tc.auto[1], tc.auto[2])
			if tc.knowsOnly1 {
				ents[1].KnownSpells = 1 << 1
			}
			v.SetEntities(ents)
			v.sel = tc.sel
			v.selectedSpell = tc.spell
			sent := qolAutocastSink(v)
			a.step(appInput{Autocast: true, Viewer: Input{Ctrl: true}}, now)
			if !slices.Equal(*sent, tc.want) {
				t.Errorf("Ctrl+A sent %v, want %v", *sent, tc.want)
			}
		})
	}
}

// A dead unit in the selection takes no autocast; the control is the same
// selection alive, which does.
func TestCtrlALeavesADeadUnitAlone(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	for _, dead := range []bool{false, true} {
		a, v := qolApp(t)
		ents := qolEntities(0, 0, 0)
		if dead {
			ents[1].Life, ents[1].Selectable = LifeDead, true
		}
		v.SetEntities(ents)
		v.sel = selection{1, 2}
		v.selectedSpell = 1
		sent := qolAutocastSink(v)
		a.step(appInput{Autocast: true, Viewer: Input{Ctrl: true}}, now)
		want := [][2]uint32{{1, 1}, {2, 1}}
		if dead {
			want = want[:1]
		}
		if !slices.Equal(*sent, want) {
			t.Errorf("dead=%t: Ctrl+A sent %v, want %v", dead, *sent, want)
		}
	}
}

// W names every living selected unit of the local player to the swap sink,
// once per press; Alt+W, the debug range, names none.
func TestWeaponSwapKeyNamesTheOwnedSelection(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	a, v := qolApp(t)
	ents := qolEntities(0, 0, 0)
	ents[2].Life, ents[2].Selectable = LifeDead, true
	v.SetEntities(ents)
	v.sel = selection{1, 2, 3, 4}
	var calls [][]uint32
	v.SetWeaponSwapSink(func(ids []uint32) { calls = append(calls, ids) })

	a.step(appInput{}, now)
	if len(calls) != 0 {
		t.Fatalf("a frame without the key named %v", calls)
	}
	a.step(appInput{WeaponSwap: true}, now)
	if len(calls) != 1 || !slices.Equal(calls[0], []uint32{1, 2}) {
		t.Fatalf("W named %v, want one call naming [1 2]", calls)
	}
	in := appInput{WeaponSwap: true, Viewer: Input{Alt: true}}
	in.suppressAltLetters()
	a.step(in, now)
	if len(calls) != 1 {
		t.Errorf("Alt+W reached the swap: %v", calls)
	}
}

// Home returns the zoom to the opening zoom and keeps the view's centre to
// within the camera's own one-cell clamp; the control is the zoom the wheel
// left, which differs.
func TestHomeRestoresTheOpeningZoomAboutTheCentre(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	a, v := qolApp(t)
	v.cam.SetZoom(defaultMapZoom())
	v.cam.CenterOn(900, 900)
	a.step(appInput{Viewer: Input{WheelY: 1}}, now)
	a.step(appInput{Viewer: Input{WheelY: 1}}, now)
	if v.cam.Zoom == defaultMapZoom() {
		t.Fatal("control: the wheel left the opening zoom")
	}
	cx, cy := v.cam.ScreenToWorld(float64(v.cam.ViewW)/2, float64(v.cam.ViewH)/2)
	a.step(appInput{ZoomReset: true}, now)
	if v.cam.Zoom != defaultMapZoom() {
		t.Errorf("Home left zoom %v, want %v", v.cam.Zoom, defaultMapZoom())
	}
	nx, ny := v.cam.ScreenToWorld(float64(v.cam.ViewW)/2, float64(v.cam.ViewH)/2)
	if math.Abs(nx-cx) > camera.CellSize || math.Abs(ny-cy) > camera.CellSize {
		t.Errorf("Home moved the view centre from (%v,%v) to (%v,%v)", cx, cy, nx, ny)
	}
}

// End returns the speed to the shipped default rung, stores it as the
// preference and leaves the unlimited loop with its phase reset; a popup
// refuses it. The control is the rung the faster key left.
func TestEndRestoresTheDefaultGameSpeed(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	a, v := qolApp(t)
	type call struct {
		period          int
		unpaced, resets bool
	}
	var calls []call
	a.flow.cadence = func(period int, _, unpaced, reset bool) { calls = append(calls, call{period, unpaced, reset}) }
	var stored []int
	a.flow.persistRung = func(rung int) { stored = append(stored, rung) }

	a.step(appInput{Faster: true}, now)
	a.step(appInput{Faster: true}, now)
	a.step(appInput{Unpaced: true}, now)
	if a.flow.rung == terrain.DefaultCadenceRung || !a.flow.unpaced {
		t.Fatalf("control: rung %d unpaced %t after two faster presses and the unlimited loop", a.flow.rung, a.flow.unpaced)
	}
	calls = nil
	a.step(appInput{SpeedReset: true}, now)
	if a.flow.rung != terrain.DefaultCadenceRung || a.flow.unpaced {
		t.Errorf("End left rung %d unpaced %t, want %d paced", a.flow.rung, a.flow.unpaced, terrain.DefaultCadenceRung)
	}
	if len(stored) == 0 || stored[len(stored)-1] != terrain.DefaultCadenceRung {
		t.Errorf("End stored %v, want the default rung last", stored)
	}
	want := call{terrain.CadencePeriod(terrain.DefaultCadenceRung), false, true}
	if len(calls) != 1 || calls[0] != want {
		t.Errorf("End told the far side %v, want [%v]", calls, want)
	}

	a.step(appInput{Faster: true}, now)
	v.OpenHelp("help")
	a.step(appInput{SpeedReset: true}, now)
	if a.flow.rung == terrain.DefaultCadenceRung {
		t.Error("End acted under the help panel")
	}
}

// The help panel lists the added keys after the install text, in the install
// text's line break; an empty install text opens no panel.
func TestHelpBodyListsTheAddedKeys(t *testing.T) {
	f := newFlow(nil, nil)
	if got := f.helpBody(""); got != "" {
		t.Errorf("empty install text gave %q", got)
	}
	got := f.helpBody("Line one\r\nLine two\r\n")
	want := "Line one\r\nLine two\r\n\r\nAdditional keys:\r\nW - take the previous weapon back from the backpack\r\n" +
		"Ctrl+A - autocast on or off for every selected caster\r\nHome - default zoom\r\nEnd - default game speed"
	if got != want {
		t.Errorf("help body = %q, want %q", got, want)
	}
	if lf := f.helpBody("One\nTwo"); strings.Contains(lf, "\r") {
		t.Errorf("a LF install text gained a CR: %q", lf)
	}
}

// The three keys are bound as the story names them, each without Ctrl.
func TestQualityOfLifeKeyBindings(t *testing.T) {
	got := bindingSource(t)
	for field, key := range map[string]string{"WeaponSwap": "ebiten.KeyW)", "ZoomReset": "ebiten.KeyHome)", "SpeedReset": "ebiten.KeyEnd)"} {
		expr := got[field]
		if !strings.Contains(expr, key) || !strings.Contains(expr, "!ctrlHeld()") {
			t.Errorf("%s is bound to %q, want %s without Ctrl", field, expr, key)
		}
	}
}

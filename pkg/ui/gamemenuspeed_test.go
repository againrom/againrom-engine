package ui

import (
	"reflect"
	"strings"
	"testing"

	"againrom/pkg/render/terrain"
)

func TestMenuSpeedChangesPausedDriverAndPersists(t *testing.T) {
	var seam *cadenceSeam
	var persisted []int
	a := newTestApp(t, appRows(3), cadenceLoader(t, &seam))
	a.SetMapCadencePreference(terrain.DefaultCadenceRung, func(r int) { persisted = append(persisted, r) })
	a.flow.screen = ScreenPicker
	a.step(appInput{Enter: true}, cadenceAt(0))
	a.step(appInput{Escape: true}, cadenceAt(1))
	if err := a.HeadlessGameMenuAction("game-options"); err != nil {
		t.Fatal(err)
	}
	if rows := a.HeadlessRows(); len(rows) != 6 || rows[1].Text != "GAME SPEED: 1x" {
		t.Fatal("missing current speed", rows)
	}
	if err := a.HeadlessGameMenuAction("speed-up"); err != nil {
		t.Fatal(err)
	}
	want := terrain.DefaultCadenceRung + 1
	if a.MapCadencePreference() != want || !reflect.DeepEqual(persisted, []int{want}) {
		t.Fatal("menu did not persist speed", a.MapCadencePreference(), persisted)
	}
	if call := mustLast(t, seam); !call.stopped || call.unpaced || call.periodUS != terrain.CadencePeriod(want) {
		t.Fatal("menu speed resumed the paused world or missed the driver", call)
	}
	if err := a.HeadlessKey("escape"); err != nil {
		t.Fatal(err)
	}
	if err := a.HeadlessKey("escape"); err != nil {
		t.Fatal(err)
	}
	if a.Screen() != ScreenMap {
		t.Fatal("Escape did not return to the map")
	}
	// Escape consumes the held interval; the next map frame resumes cadence.
	a.step(cadenceInput(false, false, false), a.headlessAt())
	if call := mustLast(t, seam); call.stopped || call.periodUS != terrain.CadencePeriod(want) {
		t.Fatal("closing the menu lost the chosen speed", call)
	}
	a.step(cadenceInput(false, false, true), a.headlessAt())
	a.step(appInput{Escape: true}, a.headlessAt())
	if err := a.HeadlessGameMenuAction("game-options"); err != nil {
		t.Fatal(err)
	}
	if a.HeadlessRows()[1].Text != "GAME SPEED: 1x" {
		t.Fatal("menu did not reflect the keyboard speed")
	}
}

func TestMenuSpeedBoundsAndUnpacedExit(t *testing.T) {
	for _, rung := range []int{terrain.CadenceRungMin, terrain.CadenceRungMax} {
		for _, unpaced := range []bool{false, true} {
			f := openMissionMenu(t)
			f.rung, f.unpaced, f.stopped = rung, unpaced, true
			f.rebuildGameMenu(gameMenuGameOptionsPage, 0)
			rows := f.menuRows()
			if rows[2].Enabled != (rung > terrain.CadenceRungMin || unpaced) || rows[3].Enabled != (rung < terrain.CadenceRungMax || unpaced) {
				t.Fatal("wrong endpoint availability", rung, unpaced)
			}
			if unpaced {
				if !strings.Contains(rows[1].text(), "UNLIMITED") {
					t.Fatal("hidden unpaced state")
				}
				selectGameMenuAction(f, gameMenuSpeedDown)
				f.chooseGameMenu()
				if f.unpaced || f.stopped || f.rung != terrain.ClampCadenceRung(rung-1) {
					t.Fatal("menu did not return to the chosen normal speed", f.rung, f.unpaced, f.stopped)
				}
			}
		}
	}
}

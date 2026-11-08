package ui

import (
	"image"
	"reflect"
	"testing"
	"time"
)

type buttonSquareTown struct {
	*fakeSquareArtTown
	buttons []int
}

func (s *buttonSquareTown) TownSquareButton(i int) TownAction {
	s.buttons = append(s.buttons, i)
	return TownAction{Msg: "talk"}
}

func TestTownSurfaceInnButtonUsesItsAction(t *testing.T) {
	town := &buttonSquareTown{fakeSquareArtTown: &fakeSquareArtTown{scene: &fakeSquareScene{controls: map[image.Point]TownSquareControl{
		image.Pt(20, 20): {Kind: TownSquareControlButton, Door: 2},
	}}}}
	a := newTestApp(t, appRows(3), okLoader(t))
	a.SetTown(town)
	if !a.flow.showTown("") {
		t.Fatal("town did not open")
	}
	a.stepTownAt(appInput{CursorX: 20, CursorY: 20, PrimaryReleased: true}, time.Unix(1700000000, 0))
	if !reflect.DeepEqual(town.buttons, []int{2}) || len(town.chosen) != 0 || a.flow.msg != "talk" {
		t.Fatalf("buttons %v rows %v message %q", town.buttons, town.chosen, a.flow.msg)
	}
}

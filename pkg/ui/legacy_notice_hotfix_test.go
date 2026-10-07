package ui

import (
	"reflect"
	"testing"
)

type localizedHeadlessTavern struct{ fakeSurfaceDialogueTown }

func (f *localizedHeadlessTavern) TownSurface() TownSurfaceView {
	view := f.fakeSurfaceDialogueTown.TownSurface()
	view.Buttons[1].Label = "Localized Talk"
	return view
}

func TestHeadlessNPCActivationUsesTheInstalledTalkWord(t *testing.T) {
	town := &localizedHeadlessTavern{}
	app := newTestApp(t, appRows(3), okLoader(t))
	words := AuthoredWords()
	words.TavernTalk = "Localized Talk"
	app.SetWords(words, nil, nil)
	app.SetTown(town)
	if !app.flow.showTown("") {
		t.Fatal("town did not open")
	}
	if err := app.HeadlessActivate("NPC 22"); err != nil {
		t.Fatal(err)
	}
	want := []TownSurfaceControl{
		{Kind: TownSurfaceControlCell, Index: 0},
		{Kind: TownSurfaceControlButton, Index: 1},
	}
	if !reflect.DeepEqual(town.surfaceClicks, want) {
		t.Fatalf("localized NPC activation=%+v, want %+v", town.surfaceClicks, want)
	}
}

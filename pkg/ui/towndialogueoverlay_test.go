package ui

import (
	"image"
	"image/color"
	"testing"
)

// dialogueOverMapTown answers true to BOTH AtWorldMap and TownDialogue. No
// other town model in this repository does, and pkg/game's own townScreen
// cannot: AtWorldMap is room == roomGates and townDialogueLayout is
// room == roomTalk, one field on one struct. That is why overlayTownDialogue's
// world-map guard is unreachable through production today, and it is also why
// a test has to build the state by hand to witness the guard at all.
type dialogueOverMapTown struct {
	view     WorldMapView
	dialogue *image.RGBA
}

func (f *dialogueOverMapTown) Header() string        { return "world map" }
func (f *dialogueOverMapTown) Rows() []TownRow       { return nil }
func (f *dialogueOverMapTown) Footer() []string      { return nil }
func (f *dialogueOverMapTown) Choose(int) TownAction { return TownAction{} }
func (f *dialogueOverMapTown) Back() bool            { return true }

func (f *dialogueOverMapTown) AtWorldMap() bool                     { return true }
func (f *dialogueOverMapTown) WorldMapView() WorldMapView           { return f.view }
func (f *dialogueOverMapTown) WorldMapHover(image.Point)            {}
func (f *dialogueOverMapTown) WorldMapMove(int)                     {}
func (f *dialogueOverMapTown) WorldMapChoose() TownAction           { return TownAction{} }
func (f *dialogueOverMapTown) WorldMapClick(image.Point) TownAction { return TownAction{} }
func (f *dialogueOverMapTown) WorldMapTick() TownAction             { return TownAction{} }

func (f *dialogueOverMapTown) TownDialogue() (*image.RGBA, bool) {
	return f.dialogue, f.dialogue != nil
}
func (f *dialogueOverMapTown) TownDialogueButton() (image.Rectangle, bool) {
	return image.Rectangle{}, false
}
func (f *dialogueOverMapTown) AdvanceTownDialogue() TownAction { return TownAction{} }

// TestTheWorldMapTakesNoDialogueOverlay pins the shape drawTown had before the
// composer seam existed: its world-map branch returned before its own dialogue
// check, so no path could paint a dialogue over the map. The two wrappers that
// replaced it apply the blit after ANY successful compose, so the invariant now
// lives in overlayTownDialogue's own guard rather than in control flow.
//
// The expectation is independent of the guard: it is the frame ComposeWorldMap
// itself produces, composed separately in this test, compared pixel for pixel.
// Removing the guard makes the two frames differ wherever the dialogue lands.
func TestTheWorldMapTakesNoDialogueOverlay(t *testing.T) {
	dialogue := image.NewRGBA(image.Rect(0, 0, 200, 120))
	for y := dialogue.Bounds().Min.Y; y < dialogue.Bounds().Max.Y; y++ {
		for x := dialogue.Bounds().Min.X; x < dialogue.Bounds().Max.X; x++ {
			dialogue.SetRGBA(x, y, color.RGBA{R: 0xff, G: 0x00, B: 0xff, A: 0xff})
		}
	}
	town := &dialogueOverMapTown{
		view:     WorldMapView{Hovered: -1, Selected: -1, Problem: "no world map art in a synthetic test"},
		dialogue: dialogue,
	}

	got, err := ComposeTownScreen(town, "")
	if err != nil {
		t.Fatalf("ComposeTownScreen over the world map: %v", err)
	}

	view := town.view
	view.Message = ""
	want := ComposeWorldMap(view)
	if got.Bounds() != want.Bounds() {
		t.Fatalf("bounds = %v, want the world map's own %v", got.Bounds(), want.Bounds())
	}

	differing := 0
	first := image.Point{X: -1, Y: -1}
	for y := want.Bounds().Min.Y; y < want.Bounds().Max.Y; y++ {
		for x := want.Bounds().Min.X; x < want.Bounds().Max.X; x++ {
			if got.RGBAAt(x, y) != want.RGBAAt(x, y) {
				differing++
				if first.X < 0 {
					first = image.Pt(x, y)
				}
			}
		}
	}
	if differing != 0 {
		t.Errorf("the world map frame differs from ComposeWorldMap's own at %d pixel(s), first %v: a dialogue was blitted over the map", differing, first)
	}

	// This test pins the negative half only: the map, and only the map, is
	// exempt. The positive half - that a dialogue over a room that is NOT the
	// map still composes the room behind it - is TestTheShopComposesBehindItsOwnDialogue
	// (town_test.go:400), which drives drawTown with a shop model whose own
	// dialogue is open and asserts the shop composed rather than the row list.
	// That test predates this seam and still passes through it.
	if _, onMap := townWorldMapScreen(town); !onMap {
		t.Fatal("the fixture stopped answering AtWorldMap, so this test no longer witnesses the guard")
	}
}

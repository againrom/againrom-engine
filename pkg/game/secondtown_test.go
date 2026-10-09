package game

import (
	"encoding/json"
	"image"
	"image/color"
	"reflect"
	"testing"

	"againrom/pkg/ui"
)

// secondTownSquareCodes places each square mask code on its own 10x10 block.
var secondTownSquareCodes = map[uint8]image.Point{0x80: {10, 10}, 0x90: {30, 10}, 0xa0: {50, 10}, 0xb0: {70, 10}, 0xc0: {90, 10}}

func secondTownSquareFixture(t *testing.T) (*FrontEnd, *ui.App, *secondCampaignScreen) {
	t.Helper()
	f, app, screen := secondCampaignFixture(t, true)
	mask := image.NewPaletted(image.Rect(0, 0, 640, 480), make(color.Palette, 256))
	for code, at := range secondTownSquareCodes {
		for y := at.Y; y < at.Y+10; y++ {
			for x := at.X; x < at.X+10; x++ {
				mask.SetColorIndex(x, y, code)
			}
		}
	}
	background := image.NewRGBA(image.Rect(0, 0, 640, 480))
	for i := range background.Pix {
		background.Pix[i] = 0x40
	}
	f.TownSquareArt = resolved(&ui.TownSquareArt{Background: background, Mask: mask}, nil)
	app.Layout(640, 480)
	if err := app.HeadlessActivate("new game"); err != nil {
		t.Fatal(err)
	}
	return f, app, screen
}

func secondTownPress(t *testing.T, app *ui.App, code uint8, edges ...string) {
	t.Helper()
	p := secondTownSquareCodes[code].Add(image.Pt(5, 5))
	for _, edge := range edges {
		if err := app.HeadlessPointer(edge, p.X, p.Y); err != nil {
			t.Fatal(err)
		}
	}
}

func TestSecondFirstTownOpensTheSquare(t *testing.T) {
	f, app, screen := secondTownSquareFixture(t)
	if f.Town.second.current != firstSecondTown || !screen.AtTownSquare() || screen.AtTownSurface() {
		t.Fatal("the first town did not open its square")
	}
	want := []ui.TownRow{{Text: "TAVERN", Choosable: true}, {Text: "SHOP"}, {Text: "SCHOOL"}, {Text: "GATES"}}
	if !reflect.DeepEqual(screen.Rows(), want) {
		t.Fatal("square doors", screen.Rows())
	}
	pix, _, err := app.HeadlessFrame()
	if err != nil || pix.RGBAAt(320, 240) != (color.RGBA{0x40, 0x40, 0x40, 0x40}) {
		t.Fatal("the square is not drawn from its picture", err)
	}
}

func TestSecondFirstTownDoorHoverAndClick(t *testing.T) {
	f, app, screen := secondTownSquareFixture(t)
	c := f.Town.second
	for _, door := range []struct {
		code     uint8
		selector int
	}{{0x80, 2}, {0x90, 1}, {0xc0, -1}, {0xa0, 8}} {
		secondTownPress(t, app, door.code, "hover")
		if got := screen.TownSquareView().Selector; got != door.selector {
			t.Fatalf("code %#x selector %d, want %d", door.code, got, door.selector)
		}
	}
	if screen.view.exterior.school {
		t.Fatal("the school hover armed its motion")
	}
	for _, code := range []uint8{0x90, 0xc0, 0xa0} {
		secondTownPress(t, app, code, "hover", "press", "release")
		if c.current != firstSecondTown || c.room != secondTownSquare || app.Screen() != ui.ScreenTown {
			t.Fatalf("closed door %#x opened something", code)
		}
	}
	secondTownPress(t, app, 0x80, "hover", "press", "release")
	if c.room != secondTownInn || !screen.AtTownSurface() || screen.AtTownSquare() {
		t.Fatal("the tavern click did not open the tavern")
	}
}

func TestSecondFirstTownTavernTalksThenGatesLeave(t *testing.T) {
	f, app, screen := secondTownSquareFixture(t)
	c := f.Town.second
	if screen.view.exteriorGateAvailable() {
		t.Fatal("the gate is open before mission 10 is available")
	}
	secondTownPress(t, app, 0x80, "hover", "press", "release")
	v := screen.TownSurface()
	var names []string
	for _, cell := range v.Cells {
		names = append(names, cell.Semantic)
	}
	if !reflect.DeepEqual(names, []string{"NPC 207", "NPC 2108", "NPC 517"}) || !v.RosterUnpainted || v.Buttons[tavernButtonTalk].Enabled || v.Buttons[tavernButtonHire].Enabled || v.Buttons[tavernButtonSleep].Enabled {
		t.Fatal("tavern entry", names, v.RosterUnpainted, v.Buttons)
	}
	screen.TownSurfaceClick(ui.TownSurfaceControl{Kind: ui.TownSurfaceControlCell, Index: 0}, false)
	if v := screen.TownSurface(); !v.Cells[0].Selected || v.RosterUnpainted || !v.Buttons[tavernButtonTalk].Enabled || c.payload != nil {
		t.Fatal("a cell click did not only select its speaker")
	}
	screen.TownSurfaceClick(ui.TownSurfaceControl{Kind: ui.TownSurfaceControlCell, Index: 2}, false)
	screen.TownSurfaceClick(ui.TownSurfaceControl{Kind: ui.TownSurfaceControlButton, Index: tavernButtonTalk}, false)
	if !c.has(secondLocation{kind: 1, id: 10}) || c.payload == nil || c.bank[769] != 1 {
		t.Fatal("Talk did not run TalkTo and AddMission")
	}
	for c.payload != nil {
		if err := app.HeadlessActivate("notice"); err != nil {
			t.Fatal(err)
		}
	}
	screen.TownSurfaceClick(ui.TownSurfaceControl{Kind: ui.TownSurfaceControlCell, Index: 2}, true)
	if c.payload == nil || len(c.available) != 2 {
		t.Fatal("a double click did not talk once")
	}
	for c.payload != nil {
		if err := app.HeadlessActivate("notice"); err != nil {
			t.Fatal(err)
		}
	}
	if err := app.HeadlessActivate(f.Words.TavernExit); err != nil {
		t.Fatal(err)
	}
	if !screen.AtTownSquare() || !screen.view.exteriorGateAvailable() || !screen.Rows()[3].Choosable {
		t.Fatal("Exit did not reach the square with its gate open")
	}
	secondTownPress(t, app, 0xa0, "hover", "press", "release")
	if c.current != (secondLocation{}) || screen.AtTownSquare() || !reflect.DeepEqual(c.available, []secondLocation{{1, 10}}) {
		t.Fatal("the gates did not take the first-town departure", c.current, c.available)
	}
	if rows := screen.Rows(); len(rows) != 1 || rows[0].Text != "mission 10" {
		t.Fatal("destination list", rows)
	}
}

func TestSecondFirstTownRowsKeepTheTalkSeam(t *testing.T) {
	f, _, screen := secondTownSquareFixture(t)
	c := f.Town.second
	screen.Choose(0)
	want := []ui.TownRow{{Text: "TALK 207", Choosable: true}, {Text: "TALK 2108", Choosable: true}, {Text: "TALK 517", Choosable: true}}
	if !reflect.DeepEqual(screen.Rows(), want) {
		t.Fatal("inn rows", screen.Rows())
	}
	screen.Choose(2)
	if !c.has(secondLocation{kind: 1, id: 10}) || c.payload == nil {
		t.Fatal("the TALK row did not run TalkTo")
	}
}

func TestSecondFirstTownRoomSurvivesTheSaveDocument(t *testing.T) {
	for _, room := range []secondTownRoom{secondTownSquare, secondTownInn} {
		f, _, _ := secondTownSquareFixture(t)
		f.Town.second.room = room
		raw, err := json.Marshal(captureSecondCampaign(f.Town.second))
		if err != nil {
			t.Fatal(err)
		}
		var loaded currentSecondCampaign
		if err := json.Unmarshal(raw, &loaded); err != nil {
			t.Fatal(err)
		}
		f.Town.second = loaded.restore()
		screen := f.TownScreen().(*secondCampaignScreen)
		if screen.AtTownSquare() != (room == secondTownSquare) || screen.AtTownSurface() != (room == secondTownInn) {
			t.Fatalf("room %d restored square=%v tavern=%v", room, screen.AtTownSquare(), screen.AtTownSurface())
		}
	}
}

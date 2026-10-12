package game

import (
	"image"
	"strings"
	"testing"

	"againrom/pkg/ui"
)

// secondMapAtTownOne starts a new game and talks in the first inn, so town 1
// is left for mission 10 next.
func secondMapAtTownOne(t *testing.T) (*FrontEnd, *ui.App, *secondCampaignScreen) {
	t.Helper()
	f := secondGameFront(t)
	app := f.App("second")
	app.Layout(1024, 768)
	for _, target := range []string{"new game", "TAVERN", "TALK 517"} {
		if err := app.HeadlessActivate(target); err != nil {
			t.Fatal(err)
		}
	}
	for pages := 0; pages < 64; pages++ {
		if err := app.HeadlessActivate("notice"); err != nil {
			break
		}
	}
	if err := app.HeadlessActivate("EXIT"); err != nil {
		t.Fatal(err)
	}
	screen, ok := f.TownScreen().(*secondCampaignScreen)
	if !ok || screen.AtWorldMap() {
		t.Fatal("the first town is not the second campaign's screen")
	}
	return f, app, screen
}

// secondMapChanged counts the pixels two compositions differ in, inside and
// outside r.
func secondMapChanged(a, b *image.RGBA, r image.Rectangle) (inside, outside int) {
	for y := a.Rect.Min.Y; y < a.Rect.Max.Y; y++ {
		for x := a.Rect.Min.X; x < a.Rect.Max.X; x++ {
			if a.RGBAAt(x, y) != b.RGBAAt(x, y) {
				if image.Pt(x, y).In(r) {
					inside++
				} else {
					outside++
				}
			}
		}
	}
	return inside, outside
}

// Leaving town 1 opens the installed map in route mode: Umoir under the
// party's Flag at (x-10,y-24) from town 1's point, no record marker and no
// text, travelling at once to mission 10; the travel's end opens mission 10
// (R2-ENGINE-272, R2-ENGINE-428, R2-ENGINE-430, R2-ENGINE-432).
func TestReleaseSecondCampaignWorldMapRouteMode(t *testing.T) {
	f, app, screen := secondMapAtTownOne(t)
	if err := app.HeadlessActivate("GATES"); err != nil {
		t.Fatal(err)
	}
	if app.Screen() != ui.ScreenTown || !screen.AtWorldMap() {
		t.Fatal("leaving town 1 did not open the map", app.Screen())
	}
	v := screen.WorldMapView()
	if v.Problem != "" || v.Background == nil || v.Background.Bounds().Size() != image.Pt(640, 480) || v.Flag == nil || v.Cross == nil {
		t.Fatalf("route-mode art: problem %q background %v", v.Problem, v.Background)
	}
	if len(v.Missions) != 1 || v.Missions[0] != (ui.WorldMapMission{Number: 10, Kind: 1, Anchor: image.Pt(302, 158), Enabled: true}) {
		t.Fatalf("route-mode records %+v", v.Missions)
	}
	if v.Position != image.Pt(215, 234) || v.FlagAt != (ui.WorldMapPlace{Offset: image.Pt(-10, -24), TopLeft: true}) || v.Selected != 0 {
		t.Fatalf("party at %v placed %+v, travel %d", v.Position, v.FlagAt, v.Selected)
	}
	if len(v.Route) < 2 || v.Route[0] != v.Position || v.Route[len(v.Route)-1] != image.Pt(302, 158) || v.Text != nil {
		t.Fatalf("route %d points, text %v", len(v.Route), v.Text)
	}
	frame := ui.ComposeWorldMap(v)
	noFlag := v
	// A transparent sheet of the same size, so no fallback dot stands in.
	noFlag.Flag = image.NewRGBA(v.Flag.Bounds())
	flag := image.Rectangle{Min: v.Position.Add(v.FlagAt.Offset), Max: v.Position.Add(v.FlagAt.Offset).Add(v.Flag.Bounds().Size())}
	if in, out := secondMapChanged(frame, ui.ComposeWorldMap(noFlag), flag); in == 0 || out != 0 {
		t.Fatalf("the party Flag changed %d pixels in its place and %d outside", in, out)
	}
	ticks := 0
	for ; ticks < 400 && app.Screen() == ui.ScreenTown; ticks++ {
		if err := app.HeadlessStep(); err != nil {
			t.Fatal(err)
		}
	}
	if app.Screen() != ui.ScreenMap || f.liveMission != 10 {
		t.Fatalf("the travel's end opened %v mission %d after %d steps: %q", app.Screen(), f.liveMission, ticks, app.HeadlessMessage())
	}
	if minimum := (len(v.Route) + 7) / 8; ticks < minimum {
		t.Fatalf("the travel ended after %d steps, before its %d reveal paints", ticks, minimum)
	}
	t.Logf("route of %d points ended after %d steps", len(v.Route), ticks)
}

// A press during a travel skips it: the next paint ends it (R2-ENGINE-432).
func TestReleaseSecondCampaignWorldMapSkip(t *testing.T) {
	f, app, screen := secondMapAtTownOne(t)
	if err := app.HeadlessActivate("GATES"); err != nil {
		t.Fatal(err)
	}
	screen.WorldMapTick()
	if a := screen.WorldMapClick(image.Pt(5, 5)); a.Open != nil || !screen.AtWorldMap() {
		t.Fatal("the skip entered at once")
	}
	if a := screen.WorldMapTick(); a.Open == nil {
		t.Fatal("the paint after a skip did not end the travel", a.Msg)
	}
	if f.Town.second.mapOpen != nil && f.Town.second.current != (secondLocation{}) {
		t.Fatal("the entry kept the opening")
	}
}

// In select mode the pointer targets the nearest record, its route shows
// before travel, MissionFlag marks it, FlagOnMap the others, YourFlag the
// party, and the target's globalmap.txt text draws in its block; a press
// travels and the end enters the record at the target (R2-ENGINE-430,
// R2-ENGINE-431, R2-ENGINE-432, R2-ENGINE-434). Records without a public
// point are disabled (DIV-3073).
func TestReleaseSecondCampaignWorldMapSelectMode(t *testing.T) {
	f, app, screen := secondMapAtTownOne(t)
	c := f.Town.second
	c.current, c.available = secondLocation{}, []secondLocation{{2, 1}, {1, 93}, {1, 94}, {1, 21}}
	c.openMap(secondLocation{1, 10})
	if app.Screen() != ui.ScreenTown || !screen.AtWorldMap() {
		t.Fatal("the map is not open")
	}
	screen.WorldMapHover(image.Pt(215, 230))
	town := screen.WorldMapView()
	if town.Problem != "" || town.Position != image.Pt(302, 158) || town.FlagAt != (ui.WorldMapPlace{Offset: image.Pt(-20, -44), TopLeft: true}) {
		t.Fatalf("select-mode party at %v placed %+v: %q", town.Position, town.FlagAt, town.Problem)
	}
	if town.Hovered != 0 || town.Missions[3].Enabled || town.Missions[3].Problem == "" {
		t.Fatalf("target %d; record 21 %+v", town.Hovered, town.Missions[3])
	}
	if len(town.Route) < 2 || town.Route[len(town.Route)-1] != image.Pt(215, 234) || town.RouteShown != len(town.Route) {
		t.Fatalf("town 1 preview %d of %d points", town.RouteShown, len(town.Route))
	}
	target := ui.WorldMapPlace{Offset: image.Pt(-5, -41), TopLeft: true}
	other := ui.WorldMapPlace{Offset: image.Pt(-5, -37), TopLeft: true}
	if town.Missions[0].Marker == nil || town.Missions[0].MarkerAt != target || town.Missions[1].Marker == nil || town.Missions[1].MarkerAt != other || town.Missions[3].Marker != nil {
		t.Fatalf("markers %+v", town.Missions)
	}
	if town.Text != nil {
		t.Fatal("town 1 has no text key and drew text", town.Text.Name)
	}

	screen.WorldMapHover(image.Pt(300, 150))
	mission := screen.WorldMapView()
	if mission.Hovered != 1 || mission.Route[len(mission.Route)-1] != image.Pt(306, 148) || mission.Missions[0].MarkerAt != other || mission.Missions[2].MarkerAt != target {
		t.Fatalf("second target %d, markers %+v", mission.Hovered, mission.Missions)
	}
	if mission.Text == nil || mission.Text.Name == "" || mission.Text.Body == "" || strings.Contains(mission.Text.Name+mission.Text.Body, "#") {
		t.Fatalf("mission 93 text %+v", mission.Text)
	}
	t.Logf("mission 93 text: %q / %d characters", mission.Text.Name, len(mission.Text.Body))
	noText := mission
	noText.Text = nil
	block := image.Rect(220, 7, 230+330, 220)
	if in, out := secondMapChanged(ui.ComposeWorldMap(mission), ui.ComposeWorldMap(noText), block); in == 0 || out != 0 {
		t.Fatalf("the text block changed %d pixels in its block and %d outside", in, out)
	}
	if in, _ := secondMapChanged(ui.ComposeWorldMap(town), ui.ComposeWorldMap(mission), image.Rect(0, 0, 640, 480)); in == 0 {
		t.Fatal("two targets drew the same frame")
	}

	screen.WorldMapClick(image.Pt(215, 230))
	travel := screen.WorldMapView()
	if travel.Selected != 0 || travel.Missions[1].Marker != nil || travel.Text != nil {
		t.Fatal("a travel kept the markers or the text")
	}
	for tick := 0; tick < 400 && screen.AtWorldMap(); tick++ {
		if a := screen.WorldMapTick(); a.Open != nil {
			t.Fatal("the town opened a mission")
		}
	}
	if c.current != (secondLocation{2, 1}) || c.room != secondTownSquare || c.mapOpen != nil || app.Screen() != ui.ScreenTown {
		t.Fatalf("the travel's end entered %v room %v", c.current, c.room)
	}
}

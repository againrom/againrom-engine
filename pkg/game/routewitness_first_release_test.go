package game

import (
	"fmt"
	"image"
	"os"
	"path/filepath"
	"strings"
	"time"

	"againrom/pkg/render/menu"
	"againrom/pkg/ui"
)

// routeTown is the town screen the route witness reads, whichever game's.
type routeTown interface {
	AtWorldMap() bool
	AtTownSquare() bool
	AtTownShop() bool
	TownSquareView() ui.TownSquareView
	ShopScreen() ui.ShopScreenView
	WorldMapView() ui.WorldMapView
	Rows() []ui.TownRow
	TownDialogue() (*image.RGBA, bool)
	TownDialogueButton() (image.Rectangle, bool)
	routeRoom() string
	dialogueBody() (string, bool)
}

func (t *secondCampaignScreen) routeRoom() string { return t.roomName() }

// routeRoom names the open room; a talk names the building it opened in.
func (t *townScreen) routeRoom() string {
	if t.room == roomTalk {
		return "talk"
	}
	return townRoomName(t.room)
}

// dialogueBody is the open conversation's page, as the second game's screen
// answers it.
func (t *townScreen) dialogueBody() (string, bool) {
	if t.room != roomTalk {
		return "", false
	}
	return t.dialogue.text, true
}

// firstRouteStages is the first game's step table, in route order. A new
// game opens the first mission from the generator, so the first town comes
// after the missions before it, which the walk wins with the chat cheat.
var firstRouteStages = []routeStage{
	{"menu", "menu", (*routeWitness).firstMenu},
	{"generator-precreate", "chargen/precreate", (*routeWitness).preCreate},
	{"generator-detail", "chargen/detailed", (*routeWitness).detail},
	{"mission-dialogue", "map", (*routeWitness).missionDialogue},
	{"mission-screen", "map", func(w *routeWitness) { w.missionScreen(false) }},
	{"mission-hero", "map", (*routeWitness).missionHero},
	{"to-town", "map", (*routeWitness).winToTown},
	{"town-square", "town", (*routeWitness).firstSquare},
	{"tavern", "town/square", func(w *routeWitness) { w.surfaceRoom("TAVERN", "tavern") }},
	{"school", "town/square", func(w *routeWitness) { w.surfaceRoom("SCHOOL", "school") }},
	{"shop", "town/square", (*routeWitness).firstShop},
	{"way-out", "town/square", (*routeWitness).firstWayOut},
	{"world-map", "town/worldmap", func(w *routeWitness) { w.worldMapWith(w.chooseMission) }},
	{"next-mission-dialogue", "map", (*routeWitness).missionDialogue},
}

// menuRegions is the menu buttons the mask answers.
func (w *routeWitness) menuRegions() []routeRegion {
	return routeRegions(image.Pt(menu.FrameW, menu.FrameH), 2, func(p image.Point) (string, bool) {
		b := w.f.Assets.ButtonAt(p)
		if b == 0 {
			return "", false
		}
		return fmt.Sprintf("%d %s", b, w.f.Assets.Binding(b).Name), true
	})
}

// startupMovies plays the startup movies as the App's Run does, records
// frame routeLaunchFrame of each movie and leaves it with Esc. The movie
// track goes to no device.
func (w *routeWitness) startupMovies() {
	w.app.SetCutsceneAudio(nil)
	if !w.app.PlayStartupCutscenes() {
		w.record("startup movies", image.Rectangle{}, "wait", fmt.Sprintf("none plays: %v", w.app.CutsceneError()))
		return
	}
	for i := 1; w.app.Screen() == ui.ScreenCutscene && i <= 8; i++ {
		name := w.app.CutsceneName()
		deadline := time.Now().Add(2 * time.Minute)
		for w.app.Screen() == ui.ScreenCutscene && w.app.CutsceneName() == name && w.app.CutsceneFrameNumber() < routeLaunchFrame && time.Now().Before(deadline) {
			if _, _, err := w.app.HeadlessFrame(); err != nil {
				w.t.Fatalf("menu: startup movie frame: %v", err)
			}
			w.run(1)
			time.Sleep(time.Millisecond)
		}
		if w.app.Screen() != ui.ScreenCutscene {
			break
		}
		w.record(fmt.Sprintf("startup movie %d", i), image.Rectangle{}, "wait", fmt.Sprintf("%s frame %d", w.app.CutsceneName(), w.app.CutsceneFrameNumber()))
		w.escape(fmt.Sprintf("startup movie %d", i))
	}
	w.expect("menu", "the startup movies")
}

// The first game's side screens and the button each is left by, at the
// button's middle: the Load window's Cancel, the Hall of Fame's art button
// and its Back, and the movie library's Cancel. Credits end on any press.
var routeFirstSideButtons = map[string][]struct {
	what string
	at   image.Point
}{
	"load":      {{"cancel", image.Pt(452, 392)}},
	"ending":    {{"hall art button", image.Pt(575, 435)}, {"back", image.Pt(98, 446)}},
	"cutscenes": {{"cancel", image.Pt(404, 372)}},
	"credits":   {{"press", image.Pt(320, 240)}},
}

// sideScreen leaves the side screen r opened by Esc, opens it again and
// leaves it by its button.
func (w *routeWitness) sideScreen(r routeRegion) {
	side := w.screen()
	settle := 2
	if side == "credits" {
		w.waitFor("credits roll", routeCreditsWait)
		settle = int(routeCreditsShot / (20 * time.Millisecond))
	}
	w.escape(side + ", leave by Esc")
	if w.screen() != "menu" {
		w.t.Errorf("menu: Esc leaves %s on %s", side, w.screen())
		return
	}
	w.click(r.key+" again", r.rect, r.at, settle, nil)
	for _, b := range routeFirstSideButtons[side] {
		if w.screen() != side {
			break
		}
		w.hover(side+" "+b.what, image.Rectangle{}, b.at)
		w.click(side+" "+b.what, image.Rectangle{}, b.at, 2, nil)
	}
	if w.screen() != "menu" {
		w.t.Errorf("menu: no button leaves %s; it shows %s", side, w.screen())
		w.escape(side + ", after its buttons")
	}
}

// firstMenu records the startup movies, hovers each button the menu's mask
// answers, clicks every button but New Game and Exit and leaves each screen
// it opens by Esc and by its button, then presses New Game.
func (w *routeWitness) firstMenu() {
	w.startupMovies()
	regions := w.menuRegions()
	if len(regions) == 0 {
		w.t.Fatal("menu: the mask answers no button")
	}
	for _, r := range regions {
		w.hover(r.key, r.rect, r.at)
	}
	newGame := -1
	for i, r := range regions {
		b := 0
		fmt.Sscanf(r.key, "%d", &b)
		switch w.f.Assets.Binding(b).Action {
		case menu.ActionNewGame:
			newGame = i
			continue
		case menu.ActionExit:
			w.record(r.key, r.rect, "wait", "not clicked: Exit ends the program")
			continue
		}
		w.click(r.key, r.rect, r.at, 2, nil)
		if w.screen() != "menu" {
			w.sideScreen(r)
		}
	}
	if newGame < 0 {
		w.t.Fatal("menu: no New Game button")
	}
	r := regions[newGame]
	w.click(r.key, r.rect, r.at, 2, nil)
	if !strings.HasPrefix(w.screen(), "chargen/") {
		w.t.Fatalf("menu: New Game shows %s", w.screen())
	}
}

// missionHero opens the hero's inventory and character pane by their keys
// and closes them.
func (w *routeWitness) missionHero() {
	for _, k := range []string{"i", "i"} {
		before := w.screen()
		if err := w.app.HeadlessKey(k); err != nil {
			w.t.Errorf("mission: key %s: %v", k, err)
		}
		w.run(2)
		w.record("inventory key "+k, image.Rectangle{}, "key", w.result(before, ""))
	}
	for _, k := range []string{"tab", "tab"} {
		before := w.screen()
		if err := w.app.HeadlessKey(k); err != nil {
			w.t.Errorf("mission: key %s: %v", k, err)
		}
		w.run(2)
		w.record("hero pane key "+k, image.Rectangle{}, "key", w.result(before, ""))
	}
}

// chat types line into the mission's chat through the App's keys and
// records it.
func (w *routeWitness) chat(line string) {
	before := w.screen()
	if err := w.app.HeadlessKey("enter"); err != nil {
		w.t.Errorf("%s: chat open: %v", w.stage, err)
	}
	if err := w.app.HeadlessType(line, false); err != nil {
		w.t.Errorf("%s: chat %s: %v", w.stage, line, err)
	}
	if err := w.app.HeadlessKey("enter"); err != nil {
		w.t.Errorf("%s: chat send: %v", w.stage, err)
	}
	w.run(2)
	w.record("chat "+line, image.Rectangle{}, "key", w.result(before, ""))
}

// winToTown wins each mission before the first town with the chat cheat,
// takes Victory on its notice, records each movie between and leaves it
// with Esc, until the town shows.
func (w *routeWitness) winToTown() {
	for n := 0; n < 8 && w.app.Screen() != ui.ScreenTown; n++ {
		switch w.app.Screen() {
		case ui.ScreenCutscene:
			w.wait("movie " + w.app.CutsceneName())
			w.escape("movie")
		case ui.ScreenMap:
			w.missionPages()
			w.wait(fmt.Sprintf("mission %d", w.f.liveMission))
			w.chat("#Chicken")
			w.missionPages()
			w.chat("#victory")
			before := w.screen()
			if err := w.app.HeadlessActivate("victory"); err != nil {
				w.t.Errorf("to town: Victory: %v", err)
			}
			w.run(5)
			w.record("victory", image.Rectangle{}, "click", w.result(before, ""))
		default:
			w.t.Fatalf("to town: the route shows %s", w.screen())
		}
		w.until(400, func() bool {
			s := w.app.Screen()
			return s == ui.ScreenTown || s == ui.ScreenCutscene || s == ui.ScreenMap && w.screen() == "map/notice"
		})
	}
	if w.app.Screen() != ui.ScreenTown {
		w.t.Fatalf("to town: no town after the missions; %s", w.screen())
	}
	w.dialoguePages("town entry", nil)
}

// firstSquare closes the town's tip panel and hovers every control and tip
// area of the square.
func (w *routeWitness) firstSquare() {
	w.wait("town")
	w.closeRoomTip()
	if w.screen() == "town/worldmap" {
		x, y, err := w.app.HeadlessWorldMapTownPoint()
		if err != nil {
			w.t.Fatalf("town: the world map has no town point: %v", err)
		}
		w.hover("world map town", image.Rectangle{}, w.nativePoint(x, y))
		w.click("world map town", image.Rectangle{}, w.nativePoint(x, y), 2, nil)
		w.until(200, func() bool { return w.screen() != "town/worldmap" })
		w.dialoguePages("town", nil)
		w.closeRoomTip()
	}
	w.expect("town/square", "the town's first screen")
	for _, r := range w.squareRegions() {
		w.hover(r.key, r.rect, r.at)
	}
}

// surfaceRegions is the tavern or school page's controls by its own hit test.
func (w *routeWitness) surfaceRegions() []routeRegion {
	v := w.chapter.TownSurface()
	return routeRegions(image.Pt(640, 480), 2, func(p image.Point) (string, bool) {
		if c, ok := ui.TownSurfaceControlAt(v, p); ok {
			return fmt.Sprintf("control %d/%d", c.Kind, c.Index), true
		}
		return "", false
	})
}

// surfaceRoom enters the square's door target, hovers every control of its
// page and each picker arrow once by click, and leaves by Esc.
func (w *routeWitness) surfaceRoom(target, room string) {
	w.enter(target, room)
	w.closeRoomTip()
	regions := w.surfaceRegions()
	if len(regions) == 0 {
		w.record(room, image.Rectangle{}, "wait", "no control answers")
	}
	for _, r := range regions {
		w.hover(r.key, r.rect, r.at)
	}
	for _, r := range regions {
		var kind, index int
		fmt.Sscanf(r.key, "control %d/%d", &kind, &index)
		switch ui.TownSurfaceControlKind(kind) {
		case ui.TownSurfaceControlCell, ui.TownSurfaceControlMode, ui.TownSurfaceControlBook:
			w.click(r.key, r.rect, r.at, 2, nil)
			w.dialoguePages(r.key, nil)
		}
	}
	var exit routeRegion
	for _, r := range w.surfaceRegions() {
		if strings.HasPrefix(r.key, fmt.Sprintf("control %d/", ui.TownSurfaceControlButton)) {
			exit = r
		}
	}
	w.leaveRoom(room, exit.key, exit.rect, exit.at)
}

// closeRoomTip closes a showing tip panel at the App's own close point.
func (w *routeWitness) closeRoomTip() {
	x, y, err := w.app.HeadlessTipClosePoint()
	if err != nil {
		return
	}
	w.wait("tip panel")
	w.hover("tip close", image.Rectangle{}, w.nativePoint(x, y))
	w.click("tip close", image.Rectangle{}, w.nativePoint(x, y), 2, nil)
}

// leaveRoom sends Esc, returns from a game menu it opens, and leaves the
// room by its last button at p.
func (w *routeWitness) leaveRoom(room, control string, rect image.Rectangle, p image.Point) {
	w.key("escape", 2)
	if w.screen() == "gamemenu" {
		before := w.screen()
		if err := w.app.HeadlessGameMenuAction("return"); err != nil {
			w.t.Errorf("%s: return: %v", room, err)
		}
		w.run(2)
		w.record("return", image.Rectangle{}, "click", w.result(before, ""))
	}
	w.dialoguePages(room+" Esc", nil)
	if w.screen() != "town/square" && control != "" {
		w.hover(control, rect, p)
		w.click(control, rect, p, 2, nil)
		w.dialoguePages(room+" exit", nil)
	}
	w.until(200, func() bool { return w.screen() == "town/square" })
	w.expect("town/square", room+" exit")
}

// firstShop hovers every control of the shop page, opens each shelf, buys
// the cheapest item and sells it back, and leaves by Esc.
func (w *routeWitness) firstShop() {
	w.enter("SHOP", "shop")
	w.closeRoomTip()
	v := w.place.ShopScreen()
	l := v.Layout
	if l == nil {
		fl := ui.FirstShopLayout()
		l = &fl
	}
	controls := shopControls(v)
	for _, c := range controls {
		w.hover(fmt.Sprintf("%s %d", c.kind, c.index), c.rect, w.shopPoint(c))
	}
	bestShelf, bestCell, bestPrice := -1, -1, int32(0)
	for _, c := range controls {
		if c.kind != "shelf_pick" {
			continue
		}
		w.click(fmt.Sprintf("shelf_pick %d", c.index), c.rect, w.shopPoint(c), 2, func() string {
			return fmt.Sprintf("shelf=%d", w.place.ShopScreen().Chosen)
		})
		for i, cell := range w.place.ShopScreen().Shelf {
			if cell.Icon != nil && (bestShelf < 0 || cell.Price < bestPrice) {
				bestShelf, bestCell, bestPrice = c.index, i, cell.Price
			}
		}
	}
	if w.save != "" {
		w.record("shop", image.Rectangle{}, "wait", "no buy: a town save walk changes nothing")
	} else if bestShelf < 0 {
		w.record("shop", image.Rectangle{}, "wait", "no item on any shelf")
	} else {
		pick := shopControlRow{"shelf_pick", bestShelf, l.ShelfPick[bestShelf]}
		w.click(fmt.Sprintf("shelf_pick %d", bestShelf), pick.rect, w.shopPoint(pick), 2, nil)
		gold := w.f.Town.Gold()
		cell := shopControlRow{"shelf", bestCell, l.ShelfCellRect(bestCell)}
		table := shopControlRow{"table", 0, l.TableCellRect(0)}
		staged := func() string {
			v := w.place.ShopScreen()
			return fmt.Sprintf("buy=%d sell=%d gold=%d", v.Buy, v.Sell, w.f.Town.Gold())
		}
		w.drag(fmt.Sprintf("shelf %d cheapest (%d) to table 0", bestCell, bestPrice), cell.rect, w.shopPoint(cell), w.shopPoint(table), 2, staged)
		for i := range l.Button {
			b := shopControlRow{"button", i, l.Button[i]}
			if w.f.Town.Gold() == gold && w.place.ShopScreen().Buy != 0 && i > 0 && i < len(l.Button)-1 {
				w.click(fmt.Sprintf("button %d", i), b.rect, w.shopPoint(b), 2, staged)
			}
		}
		if w.f.Town.Gold() != gold-int(bestPrice) {
			w.t.Errorf("shop: buying the %d item left gold %d from %d", bestPrice, w.f.Town.Gold(), gold)
		}
		gold = w.f.Town.Gold()
		mine := -1
		for i, c := range w.place.ShopScreen().Pack {
			if c.Icon != nil && !c.Money {
				mine = i
			}
		}
		if mine >= 0 {
			pack := shopControlRow{"pack", mine, l.PackCellRect(mine)}
			w.drag(fmt.Sprintf("pack %d bought, to table 0", mine), pack.rect, w.shopPoint(pack), w.shopPoint(table), 2, staged)
			for i := range l.Button {
				b := shopControlRow{"button", i, l.Button[i]}
				if w.f.Town.Gold() == gold && w.place.ShopScreen().Sell != 0 && i > 0 && i < len(l.Button)-1 {
					w.click(fmt.Sprintf("button %d", i), b.rect, w.shopPoint(b), 2, staged)
				}
			}
			if w.f.Town.Gold() <= gold {
				w.t.Errorf("shop: selling back left gold %d from %d", w.f.Town.Gold(), gold)
			}
		} else {
			w.t.Error("shop: the bought item is not in the pack")
		}
	}
	last := shopControlRow{"button", len(l.Button) - 1, l.Button[len(l.Button)-1]}
	w.leaveRoom("shop", fmt.Sprintf("button %d", last.index), last.rect, w.shopPoint(last))
}

// firstWayOut takes the gates to the world map.
func (w *routeWitness) firstWayOut() {
	r := w.sceneControl("GATES")
	w.click("GATES", r.rect, r.at, 2, nil)
	if !w.until(600, func() bool { return w.screen() == "town/worldmap" }) {
		w.dialoguePages("GATES", nil)
		w.t.Fatalf("way out: the gates show %s", w.screen())
	}
	w.wait("world map")
}

// routeTownStages are the town rows of the first game's table, the walk a
// town save takes after its load.
var routeTownStages = func() []routeStage {
	var out []routeStage
	for _, s := range firstRouteStages {
		switch s.name {
		case "town-square", "tavern", "school", "shop":
			out = append(out, s)
		}
	}
	return out
}()

// townSaves loads each .sav of AGAINROM_ROUTE_SAVES, in name order, cold
// through the menu's Load window and walks the town rows from there, with
// the file name leading each row and its frames in saves/<file>/. The walk
// buys, hires and saves nothing.
func (w *routeWitness) townSaves() {
	dir := os.Getenv("AGAINROM_ROUTE_SAVES")
	if dir == "" {
		w.rows = append(w.rows, strings.Join([]string{fmt.Sprint(len(w.rows) + 1), "town-saves", w.screen(), "", "", "wait", "none: AGAINROM_ROUTE_SAVES names no town save set", "", "", ""}, "\t"))
		w.writeRoute()
		return
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		w.t.Fatal(err)
	}
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || !strings.EqualFold(filepath.Ext(name), ".sav") {
			continue
		}
		payload, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			w.t.Fatal(err)
		}
		originals := filepath.Join(w.t.TempDir(), "originals")
		if err := os.MkdirAll(originals, 0o700); err != nil {
			w.t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(originals, name), payload, 0o600); err != nil {
			w.t.Fatal(err)
		}
		s, _ := newRouteWitness(w.t, false, originals)
		if s.dir != "" {
			s.dir = filepath.Join(s.dir, "saves", name)
			if err := os.MkdirAll(s.dir, 0o700); err != nil {
				w.t.Fatal(err)
			}
		}
		s.save, s.stage = name, "load"
		s.wait("menu")
		r, ok := routeFind(s.menuRegions(), "")
		for _, m := range s.menuRegions() {
			b := 0
			fmt.Sscanf(m.key, "%d", &b)
			if s.f.Assets.Binding(b).Action == menu.ActionLoad {
				r, ok = m, true
			}
		}
		if !ok {
			w.t.Fatal("town saves: the menu has no Load button")
		}
		s.click(r.key, r.rect, r.at, 2, nil)
		s.expect("load", "Load")
		before := s.screen()
		if err := s.app.HeadlessActivate("@first"); err != nil {
			w.t.Errorf("town saves: %s: %v", name, err)
		}
		s.until(400, func() bool { return s.app.Screen() == ui.ScreenTown })
		s.record("load "+name, image.Rectangle{}, "activate", s.result(before, ""))
		if s.app.Screen() != ui.ScreenTown {
			w.t.Errorf("town saves: %s loads to %s: %q", name, s.screen(), s.app.HeadlessMessage())
			s.writeRoute()
			continue
		}
		s.dialoguePages("town entry", nil)
		s.walk(routeTownStages, name)
		w.t.Logf("%s: %d steps", name, len(s.rows))
	}
}

// chooseMission clicks the first mission the world map offers, and again
// while the map stays on it, as a choice and its confirmation.
func (w *routeWitness) chooseMission() {
	v := w.place.WorldMapView()
	if len(v.Missions) == 0 {
		w.t.Fatal("world map: no mission offered")
	}
	n := v.Missions[0].Number
	for i := 0; i < 2 && w.screen() == "town/worldmap"; i++ {
		x, y, err := w.app.HeadlessWorldMapMissionPoint(n)
		if err != nil {
			w.t.Fatalf("world map: mission %d: %v", n, err)
		}
		w.click(fmt.Sprintf("mission %d", n), v.Missions[0].Region, w.nativePoint(x, y), 2, nil)
	}
}

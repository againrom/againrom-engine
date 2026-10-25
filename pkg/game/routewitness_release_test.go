package game

import (
	"fmt"
	"image"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"againrom/pkg/formats/res"
	"againrom/pkg/render/menu"
	"againrom/pkg/town"
	"againrom/pkg/ui"
)

// The second game's route witness walks one App from the main menu to the
// first mission's first ticks through the App's own pointer and key input,
// and records each step: the screen, the control and its rectangle, the
// action, the result, the text the step shows and the state the game holds.
// With AGAINROM_ROUTE_WITNESS_DIR set it writes one PNG frame and one
// route.tsv row per step under <dir>/<edition>/.

// routeStage is one row of the step table: the stage the rows carry, the
// screen it opens on, and its walk.
type routeStage struct {
	name   string
	screen string
	walk   func(*routeWitness)
}

// secondRouteStages is the step table, in route order. Each walk reads its
// controls from the screen's own hit test or layout.
var secondRouteStages = []routeStage{
	{"menu", "menu", (*routeWitness).menu},
	{"generator-precreate", "chargen/precreate", (*routeWitness).preCreate},
	{"generator-detail", "chargen/detailed", (*routeWitness).detail},
	{"town-square", "town/square", (*routeWitness).square},
	{"town-statue", "town/square", (*routeWitness).statue},
	{"tavern", "town/square", (*routeWitness).tavern},
	{"shop", "town/square", (*routeWitness).shop},
	{"way-out", "town/square", (*routeWitness).wayOut},
	{"world-map", "town/worldmap", (*routeWitness).worldMap},
	{"mission-dialogue", "map", (*routeWitness).missionDialogue},
	{"mission-screen", "map", (*routeWitness).missionScreen},
}

// routeRegion is one control a hit test answers over a screen: its key, the
// bounding rectangle of its sampled pixels and the middle sample in raster
// order.
type routeRegion struct {
	key  string
	rect image.Rectangle
	at   image.Point
}

// routeRegions samples size at step and groups the pixels at answers by
// key, in key order.
func routeRegions(size image.Point, step int, at func(p image.Point) (string, bool)) []routeRegion {
	hits := map[string][]image.Point{}
	for y := 0; y < size.Y; y += step {
		for x := 0; x < size.X; x += step {
			p := image.Pt(x, y)
			if k, ok := at(p); ok {
				hits[k] = append(hits[k], p)
			}
		}
	}
	keys := make([]string, 0, len(hits))
	for k := range hits {
		keys = append(keys, k)
	}
	slices.Sort(keys)
	out := make([]routeRegion, 0, len(keys))
	for _, k := range keys {
		ps := hits[k]
		r := image.Rectangle{Min: ps[0], Max: ps[0].Add(image.Pt(1, 1))}
		for _, p := range ps {
			r = r.Union(image.Rectangle{Min: p, Max: p.Add(image.Pt(step, step))})
		}
		out = append(out, routeRegion{key: k, rect: r, at: ps[len(ps)/2]})
	}
	return out
}

func routeFind(regions []routeRegion, key string) (routeRegion, bool) {
	for _, r := range regions {
		if r.key == key {
			return r, true
		}
	}
	return routeRegion{}, false
}

func routeMiddle(r image.Rectangle) image.Point { return r.Min.Add(r.Size().Div(2)) }

// routeText is installed text as the TSV carries it: the font's code page
// decoded, on one line.
func routeText(s string) string {
	s = res.DecodeCP866([]byte(s))
	return strings.NewReplacer("\t", " ", "\r", "", "\n", `\n`).Replace(s)
}

func routeRect(r image.Rectangle) string {
	if r.Empty() {
		return ""
	}
	return fmt.Sprintf("%d,%d-%d,%d", r.Min.X, r.Min.Y, r.Max.X, r.Max.Y)
}

type routeWitness struct {
	t       *testing.T
	f       *FrontEnd
	app     *ui.App
	town    *secondCampaignScreen
	chargen *ui.Chargen
	dir     string
	stage   string
	rows    []string
	steps   map[string]int
	frames  int
	opening ui.ChargenResult
}

// screen names what the App shows: its screen, the generator page, the town
// room or world map, and an open town dialogue or mission notice.
func (w *routeWitness) screen() string {
	s := w.app.Screen()
	switch s {
	case ui.ScreenChargen:
		st, _ := w.app.HeadlessChargenState()
		return "chargen/" + string(st.Stage)
	case ui.ScreenTown:
		place := w.town.roomName()
		if w.town.AtWorldMap() {
			place = "worldmap"
		}
		if _, open := w.town.dialogueBody(); open {
			place += "/dialogue"
		}
		return "town/" + place
	case ui.ScreenMap:
		if w.f.live != nil && w.f.live.view != nil {
			if _, _, shown := w.f.live.view.NoticeState(); shown {
				return "map/notice"
			}
		}
	}
	return s.String()
}

// text is what the step shows as text: the hover box, the tip panel, the
// town dialogue page or the mission notice, the menu rows and the message
// line.
func (w *routeWitness) text() string {
	var parts []string
	add := func(k, v string) {
		if v = strings.TrimSpace(v); v != "" {
			parts = append(parts, k+"="+routeText(v))
		}
	}
	add("tooltip", strings.Join(w.app.HeadlessTooltipLines(), "#"))
	switch w.app.Screen() {
	case ui.ScreenChargen:
		if v := w.chargen.TipPanel(); v.Showing() {
			add("tip", v.Text)
		}
	case ui.ScreenTown:
		if body, open := w.town.dialogueBody(); open {
			add("dialogue", body)
		}
		if w.town.AtTownShop() {
			v := w.town.ShopScreen()
			if v.TipPanel.Showing() {
				add("tip", v.TipPanel.Text)
			}
			add("shop", v.Msg)
		} else if w.town.AtTownSquare() {
			if v := w.town.TownSquareView(); v.Tip.Showing() {
				add("tip", v.Tip.Text)
			}
		}
	case ui.ScreenMap:
		if w.f.live != nil && w.f.live.view != nil {
			if body, _, shown := w.f.live.view.NoticeState(); shown {
				add("notice", body)
			}
		}
	case ui.ScreenGameMenu:
		var rows []string
		for _, r := range w.app.HeadlessRows() {
			rows = append(rows, r.Text)
		}
		add("rows", strings.Join(rows, "|"))
	}
	add("message", w.app.HeadlessMessage())
	return strings.Join(parts, " ; ")
}

// state is what the game holds: the generator's draft, or the party, the
// gold and the mission.
func (w *routeWitness) state() string {
	switch w.app.Screen() {
	case ui.ScreenMenu:
		return ""
	case ui.ScreenChargen:
		st, _ := w.app.HeadlessChargenState()
		res, _ := w.chargen.Result()
		return fmt.Sprintf("draft name=%s picture=%d difficulty=%d stats=%v choices=%v free=%d",
			routeText(st.Name), w.chargen.PreChoice(), st.Difficulty, res.Stats, res.Choices, w.chargen.Remaining())
	}
	var party []string
	for _, p := range w.f.Carried {
		h := p.Hero
		party = append(party, fmt.Sprintf("%s body=%d reaction=%d mind=%d spirit=%d skills=%v",
			routeText(p.Name), h.Body, h.Reaction, h.Mind, h.Spirit, h.Skill))
	}
	gold := 0
	if w.f.Town != nil {
		gold = w.f.Town.Gold()
	}
	s := fmt.Sprintf("gold=%d party=[%s] difficulty=%d", gold, strings.Join(party, "; "), w.f.Difficulty)
	if w.app.Screen() == ui.ScreenMap && w.f.live != nil {
		var units []string
		for _, id := range w.f.live.mission.ids {
			if e, ok := w.f.live.world.Entity(id); ok {
				units = append(units, fmt.Sprintf("%d hp=%d/%d", id, e.HP, e.MaxHP))
			}
		}
		s += fmt.Sprintf(" mission=%d tick=%d units=[%s]", w.f.liveMission, w.f.live.world.Tick(), strings.Join(units, "; "))
	}
	return s
}

// frame composes the App's frame and names the part it shows. The mission
// screen has no CPU composite (HeadlessFrame refuses it), so a mission step
// records the part it shows: the open notice, the visible hover box, the
// character pane, or else the map window around the party's first member.
func (w *routeWitness) frame(control string) (*image.RGBA, string) {
	if w.app.Screen() != ui.ScreenMap {
		pix, _, err := w.app.HeadlessFrame()
		if err != nil {
			w.t.Errorf("%s: frame: %v", w.stage, err)
		}
		return pix, ""
	}
	if w.screen() == "map/notice" {
		pix, err := w.app.HeadlessNoticeFrame()
		if err != nil {
			w.t.Errorf("%s: notice frame: %v", w.stage, err)
		}
		return pix, "-notice"
	}
	if st, pix := w.app.HeadlessTooltip(); st.Visible && pix != nil {
		return pix, "-tooltip"
	}
	if strings.HasPrefix(control, "corner/") || strings.HasPrefix(control, "stat/") || control == "hero and party" {
		if pix, _, err := w.app.HeadlessCharacterPane(); err == nil && pix != nil {
			return pix, "-pane"
		}
	}
	at := image.Point{}
	if len(w.f.live.mission.ids) > 0 {
		if e, ok := w.f.live.world.Entity(w.f.live.mission.ids[0]); ok {
			at = image.Pt(int(e.X), int(e.Y))
		}
	}
	pix, _, err := w.f.live.view.HeadlessMapFrame(at, 640, 480)
	if err != nil {
		w.t.Errorf("%s: map frame: %v", w.stage, err)
	}
	return pix, "-map"
}

// record writes one row and its frame.
func (w *routeWitness) record(control string, rect image.Rectangle, action, result string) {
	w.steps[w.stage]++
	n := len(w.rows) + 1
	file := ""
	if pix, part := w.frame(control); pix == nil {
		w.t.Errorf("%s %s %s: no frame", w.stage, control, action)
	} else if w.dir != "" {
		name := fmt.Sprintf("%04d-%s%s", n, w.stage, part)
		writeTipRender(w.t, w.dir, name, pix)
		file = name + ".png"
		w.frames++
	}
	w.rows = append(w.rows, strings.Join([]string{fmt.Sprint(n), w.stage, w.screen(), routeText(control), routeRect(rect),
		action, result, w.text(), file, w.state()}, "\t"))
}

// run runs n App steps.
func (w *routeWitness) run(n int) {
	for range n {
		if err := w.app.HeadlessStep(); err != nil {
			w.t.Fatalf("%s: step: %v", w.stage, err)
		}
	}
}

// until runs App steps until done answers true, at most max.
func (w *routeWitness) until(max int, done func() bool) bool {
	for range max {
		if done() {
			return true
		}
		w.run(1)
	}
	return done()
}

func (w *routeWitness) pointer(action string, p image.Point) {
	if err := w.app.HeadlessPointer(action, p.X, p.Y); err != nil {
		w.t.Fatalf("%s: pointer %s at %v: %v", w.stage, action, p, err)
	}
}

func (w *routeWitness) result(before, extra string) string {
	r := "same"
	if after := w.screen(); after != before {
		r = after
	}
	if extra != "" {
		r += " " + extra
	}
	return r
}

// hover rests the pointer on p past the hover box's delay on the headless
// clock and records the step.
func (w *routeWitness) hover(control string, rect image.Rectangle, p image.Point) {
	before := w.screen()
	w.pointer("hover", p)
	w.run(w.app.TooltipDelay()/20 + 5)
	w.record(control, rect, "hover", w.result(before, ""))
}

// click presses and releases at p, composing a frame after each edge, runs
// settle steps and records the step; note adds the change the caller reads.
func (w *routeWitness) click(control string, rect image.Rectangle, p image.Point, settle int, note func() string) {
	before := w.screen()
	for _, edge := range []string{"hover", "press", "release"} {
		w.pointer(edge, p)
		if _, _, err := w.app.HeadlessFrame(); err != nil && w.app.Screen() != ui.ScreenMap {
			w.t.Errorf("%s %s: frame after %s: %v", w.stage, control, edge, err)
		}
	}
	w.run(settle)
	extra := ""
	if note != nil {
		extra = note()
	}
	w.record(control, rect, "click", w.result(before, extra))
}

// key sends one key, runs settle steps and records the step.
func (w *routeWitness) key(name string, settle int) {
	before := w.screen()
	if err := w.app.HeadlessKey(name); err != nil {
		w.t.Errorf("%s: key %s: %v", w.stage, name, err)
	}
	w.run(settle)
	w.record("key "+name, image.Rectangle{}, "key", w.result(before, ""))
}

// wait records the screen as it stands.
func (w *routeWitness) wait(control string) {
	w.record(control, image.Rectangle{}, "wait", "same")
}

// expect stops the walk when the screen is not want.
func (w *routeWitness) expect(want, what string) {
	if got := w.screen(); got != want {
		w.t.Fatalf("%s: %s shows %s, want %s; message %q", w.stage, what, got, want, w.app.HeadlessMessage())
	}
}

// menu hovers each button the menu's mask answers, then presses New Game.
func (w *routeWitness) menu() {
	w.wait("menu")
	regions := routeRegions(image.Pt(menu.FrameW, menu.FrameH), 2, func(p image.Point) (string, bool) {
		b := w.f.Assets.ButtonAt(p)
		if b == 0 {
			return "", false
		}
		return fmt.Sprintf("%d %s", b, w.f.Assets.Binding(b).Name), true
	})
	if len(regions) != menu.ButtonCount {
		w.t.Errorf("menu: the mask answers %d buttons, want %d", len(regions), menu.ButtonCount)
	}
	newGame := -1
	for i, r := range regions {
		w.hover(r.key, r.rect, r.at)
		if strings.HasSuffix(r.key, " new game") {
			newGame = i
		}
	}
	if newGame < 0 {
		w.t.Fatal("menu: no New Game button")
	}
	r := regions[newGame]
	w.click(r.key, r.rect, r.at, 2, nil)
	w.expect("chargen/precreate", "New Game")
}

// closeChargenTip hovers the generator's tip panel controls and closes it.
func (w *routeWitness) closeChargenTip() {
	v := w.chargen.TipPanel()
	if !v.Showing() {
		w.wait("no tip panel")
		return
	}
	w.wait("tip panel")
	toggle := ui.TipPanelToggleRect(v.Rect)
	w.hover("tip toggle", toggle, routeMiddle(toggle))
	closeRect := ui.TipPanelCloseRect(v.Rect)
	w.hover("tip close", closeRect, routeMiddle(closeRect))
	w.click("tip close", closeRect, routeMiddle(closeRect), 2, nil)
	if w.chargen.TipPanel().Showing() {
		w.t.Errorf("%s: Close left the tip panel showing", w.stage)
	}
}

// chargenRegions is the generator page's controls by its own hit test,
// outside a showing tip panel.
func (w *routeWitness) chargenRegions(at func(*ui.Chargen, image.Point) (string, bool)) []routeRegion {
	tip := w.chargen.TipPanel()
	return routeRegions(image.Pt(640, 480), 2, func(p image.Point) (string, bool) {
		if tip.Covers(p) {
			return "", false
		}
		return at(w.chargen, p)
	})
}

// chargenClick clicks the control key of the page's hit test.
func (w *routeWitness) chargenClick(at func(*ui.Chargen, image.Point) (string, bool), key, want string) {
	r, ok := routeFind(w.chargenRegions(at), key)
	if !ok {
		w.t.Fatalf("%s: no %s control", w.stage, key)
	}
	w.click(key, r.rect, r.at, 2, nil)
	w.expect(want, key)
}

// preCreate hovers and clicks every pre-create control once, then leaves
// for the menu and opens a fresh generator, whose opening values the hero
// keeps, and goes forward.
func (w *routeWitness) preCreate() {
	w.closeChargenTip()
	regions := w.chargenRegions(ui.PreCreateControlAt)
	for _, r := range regions {
		w.hover(r.key, r.rect, r.at)
	}
	for _, r := range regions {
		if r.key == "back" || r.key == "forward" {
			continue
		}
		w.click(r.key, r.rect, r.at, 2, nil)
		w.expect("chargen/precreate", r.key)
	}
	w.chargenClick(ui.PreCreateControlAt, "back", "menu")
	w.click("new game", image.Rectangle{}, secondButtonPoint(w.t, w.f, menu.ActionNewGame), 2, nil)
	w.expect("chargen/precreate", "New Game again")
	if v := w.chargen.TipPanel(); v.Showing() {
		closeRect := ui.TipPanelCloseRect(v.Rect)
		w.click("tip close", closeRect, routeMiddle(closeRect), 2, nil)
	}
	w.chargenClick(ui.PreCreateControlAt, "forward", "chargen/detailed")
	w.opening, _ = w.chargen.Result()
}

// detail hovers and clicks every detail-page control once, goes back to
// pre-create and forward again to the opening values, and plays them.
func (w *routeWitness) detail() {
	w.closeChargenTip()
	regions := w.chargenRegions(ui.DetailedControlAt)
	for _, r := range regions {
		w.hover(r.key, r.rect, r.at)
	}
	for _, r := range regions {
		if r.key == "back" || r.key == "play" {
			continue
		}
		w.click(r.key, r.rect, r.at, 2, nil)
		w.expect("chargen/detailed", r.key)
	}
	w.chargenClick(ui.DetailedControlAt, "back", "chargen/precreate")
	w.chargenClick(ui.PreCreateControlAt, "forward", "chargen/detailed")
	if v := w.chargen.TipPanel(); v.Showing() {
		closeRect := ui.TipPanelCloseRect(v.Rect)
		w.click("tip close", closeRect, routeMiddle(closeRect), 2, nil)
	}
	if got, _ := w.chargen.Result(); !slices.Equal(got.Stats, w.opening.Stats) || !slices.Equal(got.Choices, w.opening.Choices) || got.Name != w.opening.Name {
		w.t.Errorf("detail: back and forward gave %+v, want the opening %+v", got, w.opening)
	}
	r, ok := routeFind(w.chargenRegions(ui.DetailedControlAt), "play")
	if !ok {
		w.t.Fatal("detail: no play control")
	}
	w.click("play", r.rect, r.at, 5, nil)
	w.expect("town/square", "Play")
}

// squareRegions is the shown room's controls by its scene's own hit and
// tip tests, outside a showing tip panel.
func (w *routeWitness) squareRegions() []routeRegion {
	v := w.town.TownSquareView()
	if v.Scene == nil {
		w.t.Fatalf("%s: the room has no scene", w.stage)
	}
	// A room page names its seats and buttons from the description, the
	// rectangles its hit test reads.
	var page *town.RoomSpec
	seats := 0
	if d, room, ok := w.town.composedRoom(); ok {
		page = d.Room(room)
		list, _ := w.town.state().innSeats()
		seats = len(list)
	}
	return routeRegions(v.Scene.Size(), 2, func(p image.Point) (string, bool) {
		if v.Tip.Covers(p) {
			return "", false
		}
		c, ok := v.Scene.ControlAt(p)
		tip, tok := v.Scene.TipAt(p)
		part := ""
		if page != nil && page.Seats != nil {
			if i := page.Seats.SeatAt(p, seats); i >= 0 {
				part = fmt.Sprintf(" seat %d", i)
			}
		}
		if page != nil && part == "" {
			if i := town.ButtonAt(page.Buttons, p); i >= 0 {
				part = " button " + page.Buttons[i].Name
			}
		}
		switch {
		case ok && tok:
			return fmt.Sprintf("control %d/%d tip %d%s", c.Kind, c.Door, tip, part), true
		case ok:
			return fmt.Sprintf("control %d/%d%s", c.Kind, c.Door, part), true
		case tok:
			return fmt.Sprintf("tip %d%s", tip, part), true
		}
		return "", false
	})
}

// sceneControl is the room region whose control is the menu, or a door
// whose row the screen lists as target.
func (w *routeWitness) sceneControl(target string) routeRegion {
	rows := w.town.Rows()
	for _, r := range w.squareRegions() {
		var kind, door int
		if _, err := fmt.Sscanf(r.key, "control %d/%d", &kind, &door); err != nil {
			continue
		}
		switch k := ui.TownSquareControlKind(kind); {
		case target == "menu" && k == ui.TownSquareControlMenu:
			return r
		case target != "menu" && k == ui.TownSquareControlDoor && door < len(rows) && rows[door].Text == target:
			return r
		}
	}
	w.t.Fatalf("%s: the room has no %s control", w.stage, target)
	return routeRegion{}
}

// closeTownTip hovers and closes a showing room tip panel.
func (w *routeWitness) closeTownTip(v ui.TipPanelView) {
	if !v.Showing() {
		return
	}
	w.wait("tip panel")
	closeRect := ui.TipPanelCloseRect(v.Rect)
	w.hover("tip close", closeRect, routeMiddle(closeRect))
	w.click("tip close", closeRect, routeMiddle(closeRect), 2, nil)
}

// square hovers every control and tip area of the town square.
func (w *routeWitness) square() {
	w.wait("town square")
	w.closeTownTip(w.town.TownSquareView().Tip)
	for _, r := range w.squareRegions() {
		w.hover(r.key, r.rect, r.at)
	}
}

// statue opens the square's menu, records its rows and returns.
func (w *routeWitness) statue() {
	r := w.sceneControl("menu")
	w.click("menu", r.rect, r.at, 2, nil)
	w.expect("gamemenu", "the statue")
	before := w.screen()
	if err := w.app.HeadlessGameMenuAction("return"); err != nil {
		w.t.Fatalf("statue: return: %v", err)
	}
	w.run(2)
	w.record("return", image.Rectangle{}, "click", w.result(before, ""))
	w.expect("town/square", "return")
}

// dialoguePages turns every page of the open town dialogue with its button
// and answers the page count.
func (w *routeWitness) dialoguePages(what string) int {
	for pages := 0; pages < 64; pages++ {
		if _, open := w.town.dialogueBody(); !open {
			return pages
		}
		pic, showing := w.town.TownDialogue()
		if !showing || pic == nil {
			w.t.Fatalf("%s %s: the dialogue is open but draws no window", w.stage, what)
		}
		r, ok := w.town.TownDialogueButton()
		if !ok {
			w.t.Fatalf("%s %s: the dialogue has no button", w.stage, what)
		}
		// The window's button is in window pixels; the App centres the
		// window in the 640x480 frame.
		r = r.Add(image.Pt((640-pic.Bounds().Dx())/2, (480-pic.Bounds().Dy())/2))
		w.hover("dialogue button", r, routeMiddle(r))
		w.click("dialogue button", r, routeMiddle(r), 2, nil)
	}
	w.t.Fatalf("%s %s: more than 64 pages", w.stage, what)
	return 0
}

// enter clicks the square's door named target and waits for room.
func (w *routeWitness) enter(target, room string) {
	r := w.sceneControl(target)
	w.click(target, r.rect, r.at, 2, nil)
	if !w.until(400, func() bool { return w.screen() != "town/square" }) {
		w.t.Fatalf("%s: %s opened nothing", w.stage, target)
	}
	w.dialoguePages(target)
	w.expect("town/"+room, target)
	w.wait(room)
}

// tavern hovers the inn page's seats and buttons, talks to each person
// present page by page, reports the hire offer and leaves by Exit.
func (w *routeWitness) tavern() {
	w.enter("TAVERN", "inn")
	w.closeTownTip(w.town.TownSquareView().Tip)
	for _, r := range w.squareRegions() {
		w.hover(r.key, r.rect, r.at)
	}
	seats, hires := w.town.state().innSeats()
	if hires == 0 {
		w.record("hire offer", image.Rectangle{}, "wait", "none offered: the inn seats no hire actor")
	}
	for i := range seats {
		seat, ok := w.innSeat(i)
		if !ok {
			w.t.Fatalf("tavern: seat %d answers no pixel", i)
		}
		label := w.town.Rows()[i].Text
		w.click(label, seat.rect, seat.at, 2, nil)
		if n := w.dialoguePages(label); n == 0 && i >= hires {
			w.t.Errorf("tavern: %s opened no dialogue", label)
		}
	}
	r := w.sceneControl(secondInnExit)
	w.click(secondInnExit, r.rect, r.at, 2, nil)
	w.until(200, func() bool { return w.screen() == "town/square" })
	w.expect("town/square", "Exit")
}

// innSeat is the inn page's seat i by the page's own seat rectangles.
func (w *routeWitness) innSeat(i int) (routeRegion, bool) {
	return routeFind(w.squareRegions(), fmt.Sprintf("control %d/%d seat %d", ui.TownSquareControlDoor, i, i))
}

// shopControlRow is one control of the shop page's layout.
type shopControlRow struct {
	kind  string
	index int
	rect  image.Rectangle
}

// shopControls lists every control of the shop page's own layout.
func shopControls(v ui.ShopScreenView) []shopControlRow {
	l := v.Layout
	if l == nil {
		fl := ui.FirstShopLayout()
		l = &fl
	}
	var out []shopControlRow
	for i, r := range l.Button {
		out = append(out, shopControlRow{"button", i, r})
	}
	for i, r := range l.ShelfPick {
		if !r.Empty() {
			out = append(out, shopControlRow{"shelf_pick", i, r})
		}
	}
	out = append(out, shopControlRow{"merchant", 0, l.Merchant},
		shopControlRow{"arrow up", 0, l.ArrowUp}, shopControlRow{"arrow down", 0, l.ArrowDown},
		shopControlRow{"pack left", 0, l.PackLeft}, shopControlRow{"pack right", 0, l.PackRight})
	for i := range v.Shelf {
		out = append(out, shopControlRow{"shelf", i, l.ShelfCellRect(i)})
	}
	for i := range v.Table {
		out = append(out, shopControlRow{"table", i, l.TableCellRect(i)})
	}
	for i := range v.Pack {
		out = append(out, shopControlRow{"pack", i, l.PackCellRect(i)})
	}
	return out
}

// shopPoint is the pixel the production hit test answers for a control, or
// its rectangle's middle where the headless point names no such surface.
func (w *routeWitness) shopPoint(c shopControlRow) image.Point {
	switch c.kind {
	case "button", "shelf_pick", "merchant", "shelf", "table", "pack":
		x, y, err := w.app.HeadlessShopPoint(c.kind, c.index)
		if err != nil {
			w.t.Errorf("shop: %s %d: %v", c.kind, c.index, err)
			return routeMiddle(c.rect)
		}
		return image.Pt(x, y)
	}
	return routeMiddle(c.rect)
}

// shop hovers every control of the shop page, opens each shelf and hovers
// its first item, buys the cheapest item and sells it back, and leaves.
func (w *routeWitness) shop() {
	w.enter("SHOP", "shop")
	w.closeTownTip(w.town.ShopScreen().TipPanel)
	v := w.town.ShopScreen()
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
			return fmt.Sprintf("shelf=%d", w.town.ShopScreen().Chosen)
		})
		v = w.town.ShopScreen()
		if v.Chosen != c.index {
			w.t.Errorf("shop: the shelf %d click shows shelf %d", c.index, v.Chosen)
		}
		items := 0
		for i, cell := range v.Shelf {
			if cell.Icon == nil {
				continue
			}
			if items++; items == 1 {
				w.hover(fmt.Sprintf("shelf %d first item", i), v.Layout.ShelfCellRect(i), w.shopPoint(shopControlRow{kind: "shelf", index: i}))
			}
			if bestShelf < 0 || cell.Price < bestPrice {
				bestShelf, bestCell, bestPrice = c.index, i, cell.Price
			}
		}
		if items == 0 {
			w.record(fmt.Sprintf("shelf_pick %d", c.index), c.rect, "wait", "empty shelf")
		}
	}
	if bestShelf < 0 {
		w.t.Fatal("shop: no item on any shelf")
	}
	button := func(i int) shopControlRow { return shopControlRow{"button", i, v.Layout.Button[i]} }
	pick := shopControlRow{"shelf_pick", bestShelf, v.Layout.ShelfPick[bestShelf]}
	w.click(fmt.Sprintf("shelf_pick %d", bestShelf), pick.rect, w.shopPoint(pick), 2, nil)
	gold := w.f.Town.Gold()
	cell := shopControlRow{"shelf", bestCell, v.Layout.ShelfCellRect(bestCell)}
	w.click(fmt.Sprintf("shelf %d cheapest", bestCell), cell.rect, w.shopPoint(cell), 2, func() string {
		return fmt.Sprintf("buy=%d", w.town.ShopScreen().Buy)
	})
	w.click("button 1 buy", button(1).rect, w.shopPoint(button(1)), 2, func() string {
		return fmt.Sprintf("gold %d->%d", gold, w.f.Town.Gold())
	})
	if w.f.Town.Gold() != gold-int(bestPrice) {
		w.t.Errorf("shop: buying the %d item left gold %d from %d", bestPrice, w.f.Town.Gold(), gold)
	}
	v = w.town.ShopScreen()
	mine := -1
	for i, c := range v.Pack {
		if c.Icon != nil && !c.Money {
			mine = i
		}
	}
	if mine < 0 {
		w.t.Fatal("shop: the bought item is not in the pack")
	}
	gold = w.f.Town.Gold()
	pack := shopControlRow{"pack", mine, v.Layout.PackCellRect(mine)}
	w.click(fmt.Sprintf("pack %d bought", mine), pack.rect, w.shopPoint(pack), 2, func() string {
		return fmt.Sprintf("sell=%d", w.town.ShopScreen().Sell)
	})
	w.click("button 2 sell", button(2).rect, w.shopPoint(button(2)), 2, func() string {
		return fmt.Sprintf("gold %d->%d", gold, w.f.Town.Gold())
	})
	if w.f.Town.Gold() <= gold {
		w.t.Errorf("shop: selling back left gold %d from %d", w.f.Town.Gold(), gold)
	}
	w.click("button 3 leave", button(3).rect, w.shopPoint(button(3)), 2, nil)
	w.until(200, func() bool { return w.screen() == "town/square" })
	w.expect("town/square", "Leave")
}

// wayOut takes the gates to the world map.
func (w *routeWitness) wayOut() {
	r := w.sceneControl("GATES")
	w.click("GATES", r.rect, r.at, 2, nil)
	if !w.until(600, func() bool { return w.screen() == "town/worldmap" }) {
		w.dialoguePages("GATES")
		w.t.Fatalf("way out: the gates show %s", w.screen())
	}
	w.wait("world map")
}

// worldMap hovers every card and site the world map's hit tests answer
// while it shows, and records the travel until the first mission opens.
func (w *routeWitness) worldMap() {
	v := w.town.WorldMapView()
	regions := routeRegions(image.Pt(640, 480), 2, func(p image.Point) (string, bool) {
		if i, ok := ui.WorldMapCardAt(v, p); ok && i < 0 {
			return "town card", true
		} else if ok {
			return fmt.Sprintf("card %d mission %d", i, v.Missions[i].Number), true
		}
		if i, ok := ui.WorldMapRegionAt(v, p); ok {
			return fmt.Sprintf("site %d mission %d", i, v.Missions[i].Number), true
		}
		return "", false
	})
	if len(regions) == 0 {
		w.record("world map", image.Rectangle{}, "wait", "no card or site answers")
	}
	for _, r := range regions {
		if w.screen() == "town/worldmap" {
			w.hover(r.key, r.rect, r.at)
		}
	}
	for i := 1; w.app.Screen() != ui.ScreenMap; i++ {
		if i > 4000 {
			w.t.Fatalf("world map: no mission opened: %s %q", w.screen(), w.app.HeadlessMessage())
		}
		w.run(1)
		if i%40 == 0 && w.screen() == "town/worldmap" {
			w.wait(fmt.Sprintf("travel step %d", i))
		}
	}
	w.wait("mission opened")
}

// missionPages pages every open mission notice with Enter.
func (w *routeWitness) missionPages() {
	for pages := 0; pages < 64; pages++ {
		if w.screen() != "map/notice" {
			return
		}
		w.wait(fmt.Sprintf("notice page %d", pages+1))
		w.key("enter", 2)
	}
	w.t.Fatal("mission: more than 64 notice pages")
}

// missionDialogue runs the first ticks and pages each notice that opens.
func (w *routeWitness) missionDialogue() {
	if w.f.liveMission != w.f.Base().Profile.Mission() {
		w.t.Errorf("mission: opened %d, want %d", w.f.liveMission, w.f.Base().Profile.Mission())
	}
	for range 4 {
		if !w.until(100, func() bool { return w.screen() == "map/notice" }) {
			break
		}
		w.missionPages()
	}
	w.wait("first ticks")
}

// missionTargets hovers the frame at step and groups the points by the
// hover target the mission screen's own hover test answers.
func (w *routeWitness) missionTargets(step int) []routeRegion {
	place := w.f.live.view.Placement()
	size := place.FrameSize()
	regions := map[string]routeRegion{}
	var order []string
	for y := step / 2; y < size.Y; y += step {
		for x := step / 2; x < size.X; x += step {
			wx, wy, ok := place.FrameToWindow(image.Pt(x, y))
			if !ok {
				continue
			}
			w.pointer("hover", image.Pt(wx, wy))
			st, _ := w.app.HeadlessTooltip()
			if st.Target == "" {
				continue
			}
			cell := image.Rect(x-step/2, y-step/2, x+step/2, y+step/2)
			key := st.Target
			if strings.HasPrefix(key, "item/") {
				w.run(w.app.TooltipDelay()/20 + 5)
				key += "\x00" + strings.Join(w.app.HeadlessTooltipLines(), "\x00")
			}
			r, seen := regions[key]
			if !seen {
				r = routeRegion{key: st.Target, rect: cell, at: image.Pt(wx, wy)}
				order = append(order, key)
			}
			r.rect = r.rect.Union(cell)
			regions[key] = r
		}
	}
	out := make([]routeRegion, 0, len(order))
	seen := map[string]int{}
	for _, k := range order {
		r := regions[k]
		if strings.HasPrefix(r.key, "item/") {
			seen[r.key]++
			r.key = fmt.Sprintf("%s#%d", r.key, seen[r.key])
		}
		out = append(out, r)
	}
	return out
}

// missionScreen pauses the mission, selects the hero, hovers every control
// the mission screen's hover test answers, presses the pack corner twice,
// and records the hero and the party.
func (w *routeWitness) missionScreen() {
	w.key("0", 1)
	if len(w.f.live.mission.ids) == 0 {
		w.t.Fatal("mission: no party unit")
	}
	hero := w.f.live.mission.ids[0]
	if x, y, err := w.app.HeadlessEntityPoint(uint32(hero)); err != nil {
		w.t.Errorf("mission: the hero answers no pixel: %v", err)
	} else {
		w.click("hero", image.Rectangle{}, image.Pt(x, y), 1, func() string {
			return fmt.Sprintf("selection=%v", w.app.HeadlessSelection())
		})
	}
	targets := w.missionTargets(8)
	if len(targets) == 0 {
		w.t.Error("mission: no control answers a hover")
	}
	hovered := map[string]string{}
	for _, r := range targets {
		w.hover(r.key, r.rect, r.at)
		hovered[r.key] = strings.Join(w.app.HeadlessTooltipLines(), " ")
		if hovered[r.key] == "" {
			w.t.Errorf("mission: %s shows no hover text", r.key)
		}
	}
	// The pack corner closes and reopens the hero's inventory; its hover
	// text names the next press.
	for _, r := range targets {
		if !strings.HasPrefix(r.key, "corner/0/") {
			continue
		}
		var after [2]string
		for i := range after {
			w.click(r.key, r.rect, r.at, w.app.TooltipDelay()/20+5, func() string {
				after[i] = strings.Join(w.app.HeadlessTooltipLines(), " ")
				return "hover=" + routeText(after[i])
			})
		}
		if after[0] == hovered[r.key] || after[1] != hovered[r.key] {
			w.t.Errorf("mission: the pack corner reads %q, then %q and %q", hovered[r.key], after[0], after[1])
		}
	}
	w.wait("hero and party")
	w.key("0", 1)
}

// TestReleaseSecondGameRouteWitness walks the second game's route from the
// main menu to the first mission's first ticks through the App's own input,
// asserts every step reaches its screen, and records each step.
func TestReleaseSecondGameRouteWitness(t *testing.T) {
	f := secondGameFront(t)
	if f.ChargenAssets == nil {
		t.Fatal("the second game's install loaded no generator")
	}
	f.Options = OptionsStore{}
	app := f.App("route witness")
	app.Layout(640, 480)
	f.TownAnimationNow = app.HeadlessSceneNow
	f.ConfigureSaveSeams(app, SaveStore{Dir: t.TempDir()}, OriginalStore{}, func() time.Time { return time.Unix(1700000000, 0).UTC() })
	// The witness holds the App's own town screen: the second campaign's
	// builder makes a fresh screen at each call.
	town, ok := f.TownScreen().(*secondCampaignScreen)
	if !ok {
		t.Fatal("the second game's town is not the second campaign's screen")
	}
	app.SetTown(town)
	w := &routeWitness{t: t, f: f, app: app, town: town, steps: map[string]int{}}
	app.SetNewGameChargen(func() *ui.ChargenEntry {
		w.chargen = ui.NewChargen(f.ChargenSetup())
		return &ui.ChargenEntry{Model: w.chargen, Begin: f.NewGameBegin(f.Base().Profile.Mission())}
	})
	if out := os.Getenv("AGAINROM_ROUTE_WITNESS_DIR"); out != "" {
		root, err := editorPhysicalDirectory(f.Archives.Root)
		if err != nil || !filepath.IsAbs(out) {
			t.Fatal("absolute install and witness roots required", err)
		}
		if rel, err := filepath.Rel(root, filepath.Clean(out)); err == nil && !strings.HasPrefix(rel, "..") {
			t.Fatal("the witness output is inside the install")
		}
		w.dir = filepath.Join(out, f.Base().Profile.ID)
		if err := os.MkdirAll(w.dir, 0o700); err != nil {
			t.Fatal(err)
		}
	}
	start := time.Now()
	for _, s := range secondRouteStages {
		w.stage = s.name
		// A stage opens on its screen; a dialogue or notice already open
		// over it is the stage's first page.
		if got := w.screen(); got != s.screen && !strings.HasPrefix(got, s.screen+"/") {
			t.Fatalf("%s: the stage opens on %s, want %s", s.name, got, s.screen)
		}
		s.walk(w)
	}
	counts := make([]string, 0, len(secondRouteStages))
	for _, s := range secondRouteStages {
		counts = append(counts, fmt.Sprintf("%s=%d", s.name, w.steps[s.name]))
	}
	t.Logf("%s: %d steps, %d frames in %v; %s", f.Base().Profile.ID, len(w.rows), w.frames, time.Since(start).Round(time.Millisecond), strings.Join(counts, " "))
	if w.dir != "" {
		head := strings.Join([]string{"n", "step", "screen", "control", "rect", "action", "result", "text", "frame", "state"}, "\t")
		if err := os.WriteFile(filepath.Join(w.dir, "route.tsv"), []byte(head+"\n"+strings.Join(w.rows, "\n")+"\n"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
}

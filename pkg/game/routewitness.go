package game

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"time"

	"againrom/pkg/random"
	"againrom/pkg/render/frame"
	"againrom/pkg/render/menu"
	"againrom/pkg/sim"
	"againrom/pkg/town"
	"againrom/pkg/ui"
	"againrom/pkg/vfs"
)

// The route witness walks one App of either game from the main menu to the
// first mission's first ticks through the App's own pointer and key input,
// and records each step: the screen, the control and its rectangle, the
// action, the result, the text the step shows and the state the game holds.
// With AGAINROM_ROUTE_WITNESS_DIR set it writes one PNG frame and one
// route.tsv row per step under <dir>/<edition>/; on the first game,
// AGAINROM_ROUTE_SAVES names town saves whose walks go under saves/<file>/.

// routeStage is one row of a game's step table: the stage the rows carry,
// the screen it opens on, the stage kind that walks it and the row's data.
type routeStage struct {
	name   string
	screen string
	kind   routeKind
	data   routeData
}

// walk runs the row's kind on its data.
func (s routeStage) walk(w *routeWitness) { routeKindWalks[s.kind](w, s.data) }

// routeKind is a stage kind. Each has one walk in routeKindWalks, which
// serves every row of the kind in either game's table.
type routeKind int

const (
	routeKindMenu routeKind = iota
	routeKindPreCreate
	routeKindDetail
	routeKindSquare
	routeKindStatue
	routeKindRoom
	routeKindShop
	routeKindBlank
	routeKindWayOut
	routeKindWorldMap
	routeKindMissionDialogue
	routeKindMissionScreen
	routeKindMissionHero
	routeKindToTown
	routeKindMissionMenu
	routeKindMissionPack
	routeKindSignpost
	routeKindMissionExit
	routeKindCount
)

// routeKindWalks is each stage kind's one walk.
var routeKindWalks = [routeKindCount]func(*routeWitness, routeData){
	routeKindMenu:            (*routeWitness).menu,
	routeKindPreCreate:       (*routeWitness).preCreate,
	routeKindDetail:          (*routeWitness).detail,
	routeKindSquare:          (*routeWitness).square,
	routeKindStatue:          (*routeWitness).statue,
	routeKindRoom:            (*routeWitness).room,
	routeKindShop:            (*routeWitness).shop,
	routeKindBlank:           (*routeWitness).squareBlank,
	routeKindWayOut:          (*routeWitness).wayOut,
	routeKindWorldMap:        (*routeWitness).worldMap,
	routeKindMissionDialogue: (*routeWitness).missionDialogue,
	routeKindMissionScreen:   (*routeWitness).missionScreen,
	routeKindMissionHero:     (*routeWitness).missionHero,
	routeKindToTown:          (*routeWitness).winToTown,
	routeKindMissionMenu:     (*routeWitness).missionMenu,
	routeKindMissionPack:     (*routeWitness).missionPack,
	routeKindSignpost:        (*routeWitness).signpost,
	routeKindMissionExit:     (*routeWitness).missionExit,
}

// routeData is what a row's kind does on that row; a kind reads only its
// own fields, and a zero field is the reduced walk.
type routeData struct {
	// menu is the main menu walk's data.
	menu routeMenuData
	// play is the screen the generator's Play opens.
	play string
	// arrive is the wait the square's walk records on arrival.
	arrive string
	// tip is how a square, room or shop tip panel closes.
	tip routeTipClose
	// room is the room walk's data, shop the shop walk's.
	room routeRoomData
	shop routeShopData
	// choose clicks the first mission the world map offers before the
	// travel.
	choose bool
	// newGame states that the mission is the one a new game opens.
	newGame bool
	// spellbook clicks the mission's spellbook icon twice; heroShown
	// states that the hero answers a window pixel at the mission's start.
	spellbook, heroShown bool
	// town marks a row a town save's walk takes after its load.
	town bool
}

// routeMenuData is the menu walk's data.
type routeMenuData struct {
	// launch is the startup movie the walk records; zero records every
	// startup movie and leaves each by Esc.
	launch routeLaunch
	// buttons is the button count the menu's mask must answer, 0 for any.
	buttons int
	// trips are the side trips at the original's points, in its order;
	// nil visits every button but New Game and Exit at its region.
	trips []routeTrip
	// notes are waits recorded after New Game.
	notes []routeNote
}

// routeLaunch is a startup movie the walk records at routeLaunchFrame,
// named shot, and whose Esc, named esc, ends the startup sequence.
type routeLaunch struct{ movie, shot, esc string }

// routeNote is a wait the walk records with its result.
type routeNote struct{ control, result string }

// routeTrip is one side trip from the menu: the press of the button bound
// to action at at, a wait named roll on the screen it opens, Esc, the press
// again where Esc left that screen, the screen's own buttons while it shows,
// and Esc named leave where they leave it showing. A trip found on the mask
// is explore: a press that opens nothing ends it, Esc must reach the menu,
// and routeSideScreens gives the rest by the screen it opened.
type routeTrip struct {
	action      menu.Action
	at          image.Point
	press       string
	settle      int
	roll        string
	esc         string
	again       string
	againSettle int
	buttons     []routeSideButton
	leave       string
	explore     bool
}

// routeSideButton is a side screen's own button: its hover and click
// controls and its point.
type routeSideButton struct {
	hover, click string
	at           image.Point
}

// routeTipClose is how the walk closes a tip panel.
type routeTipClose int

const (
	// routeTipCloseButton clicks the middle of the close button, whose
	// rectangle the row records.
	routeTipCloseButton routeTipClose = iota
	// routeTipClosePoint clicks the close point the App answers, with no
	// rectangle.
	routeTipClosePoint
)

// routeRoomData is the room walk's data.
type routeRoomData struct {
	// door is the square's door the walk enters by, room the room it opens.
	door, room string
	// surface reads the room as a surface page: its controls by the page's
	// hit test, each pick (cell, mode, book) clicked once, left by Esc and
	// the page's last button. Otherwise the room is a scene page left by
	// its exit control.
	surface bool
	exit    string
	// pane walks the room's character pane; talks selects each seat and
	// talks to its person.
	pane, talks bool
}

// routeShopData is the shop walk's data.
type routeShopData struct {
	// pane hovers the character pane and toggles its book twice.
	pane bool
	// inspect asserts each shelf's choice and the stock, hovers each
	// shelf's first item and records an empty shelf.
	inspect bool
	// named presses the trade buttons once each by name, Buy and Sell, the
	// sale staged by a click on the pack, and leaves by Leave. Otherwise
	// each middle button is pressed while the trade stays staged, the sale
	// is staged by a drag to the table, and the walk leaves by Esc and the
	// last button.
	named bool
	// undo stages the original's drag buy (s3.64) and takes it back.
	undo bool
}

// secondRouteStages is the second game's step table, in route order. Each
// walk reads its controls from the screen's own hit test or layout.
var secondRouteStages = []routeStage{
	{"menu", "menu", routeKindMenu, routeData{menu: routeMenuData{
		// The original's s1.01 shows the Nival logo movie's opening
		// frost, which starts after the publisher's logo.
		launch:  routeLaunch{"logos/nival.smk", "intro movie (s1.01)", "s1.02"},
		buttons: menu.ButtonCount,
		trips:   routeOriginalTrips,
		notes:   []routeNote{{"intro movie (s1.42)", "none: New Game opens the hero creation with no movie; the original's s1.42 is the launch intro of a second run"}},
	}}},
	{"generator-precreate", "chargen/precreate", routeKindPreCreate, routeData{}},
	{"generator-detail", "chargen/detailed", routeKindDetail, routeData{play: "town/square"}},
	{"town-square", "town/square", routeKindSquare, routeData{arrive: "town square"}},
	{"town-statue", "town/square", routeKindStatue, routeData{}},
	{"tavern", "town/square", routeKindRoom, routeData{room: routeRoomData{door: "TAVERN", room: "inn", exit: secondInnExit, pane: true, talks: true}}},
	{"shop", "town/square", routeKindShop, routeData{shop: routeShopData{pane: true, inspect: true, named: true, undo: true}}},
	{"town-blank", "town/square", routeKindBlank, routeData{}},
	{"way-out", "town/square", routeKindWayOut, routeData{}},
	{"world-map", "town/worldmap", routeKindWorldMap, routeData{}},
	{"mission-dialogue", "map", routeKindMissionDialogue, routeData{newGame: true}},
	{"mission-screen", "map", routeKindMissionScreen, routeData{spellbook: true, heroShown: true}},
	{"mission-menu", "map", routeKindMissionMenu, routeData{}},
	{"mission-pack", "map", routeKindMissionPack, routeData{}},
	{"mission-signpost", "map", routeKindSignpost, routeData{}},
	{"mission-exit", "map", routeKindMissionExit, routeData{}},
}

// routeOriginalTrips are the original's side trips from the menu (s1.30 to
// s1.40), at its points and in its order.
var routeOriginalTrips = []routeTrip{
	{action: menu.ActionLoad, at: routeMenuLoad, press: "load game (s1.30)", settle: 2, esc: "s1.31"},
	{action: menu.ActionHallOfFame, at: routeMenuHall, press: "hall of fame (s1.32)", settle: 2, esc: "s1.33",
		again: "hall of fame again, to reach s1.34", againSettle: 2,
		buttons: []routeSideButton{{"hall seal ok (s1.34)", "hall seal ok (s1.35)", routeHallSeal}}, leave: "after s1.35"},
	{action: menu.ActionCredits, at: routeMenuCredits, press: "credits (s1.36)", settle: int(routeCreditsShot / (20 * time.Millisecond)),
		roll: "credits roll (s1.37)", esc: "s1.38"},
	{action: menu.ActionCutscenes, at: routeMenuCutscenes, press: "cut scenes (s1.39)", settle: 2, esc: "s1.40"},
}

// routeTownRows are the rows of stages a town save's walk takes after its
// load, in route order.
func routeTownRows(stages []routeStage) []routeStage {
	var out []routeStage
	for _, s := range stages {
		if s.data.town {
			out = append(out, s)
		}
	}
	return out
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
	s = vfs.DecodeText([]byte(s))
	return strings.NewReplacer("\t", " ", "\r", "", "\n", `\n`).Replace(s)
}

func routeRect(r image.Rectangle) string {
	if r.Empty() {
		return ""
	}
	return fmt.Sprintf("%d,%d-%d,%d", r.Min.X, r.Min.Y, r.Max.X, r.Max.Y)
}

type routeWitness struct {
	t       routeReport
	f       *FrontEnd
	app     *ui.App
	town    *townScreen
	chargen *ui.Chargen
	dir     string
	stage   string
	rows    []string
	steps   map[string]int
	frames  int
	size    image.Point
	opening ui.ChargenResult
	// save is the town save a walk loaded, or empty.
	save string
	// render, when set, draws the whole window over the running mission;
	// without it a mission step records what the App composes on the CPU.
	render func(app *ui.App, size image.Point) *image.RGBA
	// emit writes a step's frame under dir; frames go out only with dir set.
	emit func(dir, name string, pix *image.RGBA)
}

// routeRoom names the open room; a talk names the building it opened in.
func (t *townScreen) routeRoom() string { return t.roomName() }

// dialogueBody is the open conversation's page as the seat read it.
func (t *townScreen) dialogueBody() (string, bool) {
	if !t.view().Open {
		return "", false
	}
	return t.dialogue.text, true
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
		place := w.town.routeRoom()
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

// routeFrameRow is one way the route asks for a frame. The first row whose
// when holds and whose accept takes the result answers: compose, or the App
// when compose is nil, supplies the picture of the row's part, a non-empty
// report names a composition error, and refuse reports a screen with no frame.
type routeFrameRow struct {
	when    func(w *routeWitness, control string) bool
	part    ui.HeadlessPart
	compose func(w *routeWitness) (*image.RGBA, ui.HeadlessExtras, error)
	suffix  string
	report  string
	accept  func(pix *image.RGBA, extras ui.HeadlessExtras, err error) bool
	refuse  string
}

func routeMissionPlay(w *routeWitness, _ string) bool {
	return w.app.HeadlessGameplayScreen() == ui.ScreenMap
}

func routeOverMission(w *routeWitness, control string) bool {
	return routeMissionPlay(w, control) && w.app.Screen() != ui.ScreenMap
}

func routeMissionFrame(w *routeWitness, control string) bool {
	return routeMissionPlay(w, control) && w.app.Screen() == ui.ScreenMap
}

func routeComposed(pix *image.RGBA, _ ui.HeadlessExtras, err error) bool {
	return err == nil && pix != nil
}

// routeFrameRows selects the frame of a route step. A mission reads through
// App.Draw when a window is allowed, and every other screen through the CPU
// composite; a menu or a page over the mission is its own composite where one
// exists, else the menu's panel.
var routeFrameRows = []routeFrameRow{
	{when: func(w *routeWitness, _ string) bool { return w.app.HeadlessGameplayScreen() != ui.ScreenMap }, part: ui.HeadlessPartFrame, report: "frame"},
	{when: func(w *routeWitness, control string) bool { return w.render != nil && routeMissionPlay(w, control) }, compose: func(w *routeWitness) (*image.RGBA, ui.HeadlessExtras, error) {
		return w.render(w.app, w.size), ui.HeadlessExtras{}, nil
	}, suffix: "-mission"},
	{when: routeOverMission, part: ui.HeadlessPartFrame, accept: func(_ *image.RGBA, _ ui.HeadlessExtras, err error) bool { return err == nil }},
	{when: routeOverMission, part: ui.HeadlessPartGameMenuPanel, suffix: "-menu", accept: routeComposed},
	{when: routeOverMission, refuse: "over the mission composes no frame"},
	{when: func(w *routeWitness, control string) bool {
		return routeMissionFrame(w, control) && w.screen() == "map/notice"
	}, part: ui.HeadlessPartNotice, suffix: "-notice", report: "notice frame"},
	{when: func(w *routeWitness, control string) bool {
		return routeMissionFrame(w, control) && control == "mission spellbook icon"
	}, part: ui.HeadlessPartBottomHUD, suffix: "-hud", report: "spellbook frame", accept: func(pix *image.RGBA, _ ui.HeadlessExtras, _ error) bool {
		return pix != nil && !pix.Bounds().Empty()
	}},
	{when: routeMissionFrame, part: ui.HeadlessPartTooltip, suffix: "-tooltip", accept: func(pix *image.RGBA, extras ui.HeadlessExtras, _ error) bool {
		return extras.Tooltip.Visible && pix != nil
	}},
	{when: func(w *routeWitness, control string) bool {
		return routeMissionFrame(w, control) && strings.HasPrefix(control, "inventory")
	}, part: ui.HeadlessPartInventoryPack, suffix: "-pack", accept: routeComposed},
	{when: func(w *routeWitness, control string) bool {
		if !routeMissionFrame(w, control) {
			return false
		}
		return strings.HasPrefix(control, "corner/") || strings.HasPrefix(control, "stat/") || control == "hero and party" || strings.HasPrefix(control, "hero pane")
	}, part: ui.HeadlessPartCharacterPane, suffix: "-pane", accept: routeComposed},
	{when: routeMissionFrame, compose: routeMapFrame, suffix: "-map", report: "map frame"},
}

// routeMapFrame composes the mission around the first unit of the mission.
func routeMapFrame(w *routeWitness) (*image.RGBA, ui.HeadlessExtras, error) {
	at := image.Point{}
	if len(w.f.live.mission.ids) > 0 {
		if e, ok := w.f.live.world.Entity(w.f.live.mission.ids[0]); ok {
			at = image.Pt(int(e.X), int(e.Y))
		}
	}
	pix, _, err := w.f.live.view.HeadlessMapFrame(at, w.size.X, w.size.Y)
	return pix, ui.HeadlessExtras{}, err
}

// frame captures the route step's frame through the first row that answers.
func (w *routeWitness) frame(control string) (*image.RGBA, string) {
	for _, row := range routeFrameRows {
		if !row.when(w, control) {
			continue
		}
		if row.refuse != "" {
			w.t.Errorf("%s: %s %s", w.stage, w.screen(), row.refuse)
			return nil, ""
		}
		compose := row.compose
		if compose == nil {
			compose = func(w *routeWitness) (*image.RGBA, ui.HeadlessExtras, error) {
				return w.app.HeadlessPicture(ui.HeadlessRequest{Part: row.part})
			}
		}
		pix, extras, err := compose(w)
		if err != nil && row.report != "" {
			w.t.Errorf("%s: %s: %v", w.stage, row.report, err)
		}
		if row.accept == nil || row.accept(pix, extras, err) {
			return pix, row.suffix
		}
	}
	return nil, ""
}

func (w *routeWitness) fitFrame(src *image.RGBA) *image.RGBA {
	if src.Bounds().Size() == w.size {
		return src
	}
	dst := image.NewRGBA(image.Rectangle{Max: w.size})
	draw.Draw(dst, dst.Bounds(), image.NewUniform(color.Black), image.Point{}, draw.Src)
	place := frame.Fit(src.Bounds().Dx(), src.Bounds().Dy(), w.size.X, w.size.Y)
	for y := 0; y < w.size.Y; y++ {
		for x := 0; x < w.size.X; x++ {
			if p, ok := place.WindowToFrame(x, y); ok {
				dst.SetRGBA(x, y, src.RGBAAt(src.Rect.Min.X+p.X, src.Rect.Min.Y+p.Y))
			}
		}
	}
	return dst
}

func (w *routeWitness) windowPoint(p image.Point) image.Point {
	if w.app.Screen() == ui.ScreenMap {
		return p
	}
	x, y, ok := w.app.HeadlessNativePoint(p)
	if !ok {
		w.t.Fatalf("%s: native point %v has no window pixel at %v", w.stage, p, w.size)
	}
	return image.Pt(x, y)
}

func (w *routeWitness) nativePoint(x, y int) image.Point {
	p, ok := frame.Fit(frame.W, frame.H, w.size.X, w.size.Y).WindowToFrame(x, y)
	if !ok {
		w.t.Fatalf("%s: window point %d,%d has no native pixel", w.stage, x, y)
	}
	return p
}

// record writes one row and its frame.
func (w *routeWitness) record(control string, rect image.Rectangle, action, result string) {
	w.steps[w.stage]++
	n := len(w.rows) + 1
	file := ""
	if pix, part := w.frame(control); pix == nil {
		w.t.Errorf("%s %s %s: no frame", w.stage, control, action)
	} else if w.dir != "" && w.emit != nil {
		name := fmt.Sprintf("%04d-%s%s", n, w.stage, part)
		w.emit(w.dir, name, w.fitFrame(pix))
		file = name + ".png"
		w.frames++
	}
	w.rows = append(w.rows, strings.Join([]string{fmt.Sprint(n), strings.TrimSpace(w.save + " " + w.stage), w.screen(), routeText(control), routeRect(rect),
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
	w.hoverNote(control, rect, p, nil)
}

// hoverNote is hover with the change the caller reads after the rest.
func (w *routeWitness) hoverNote(control string, rect image.Rectangle, p image.Point, note func() string) {
	p = w.windowPoint(p)
	before := w.screen()
	w.pointer("hover", p)
	w.run(w.app.TooltipDelay()/20 + 5)
	extra := ""
	if note != nil {
		extra = note()
	}
	w.record(control, rect, "hover", w.result(before, extra))
}

// click presses and releases at p, composing a frame after each edge, runs
// settle steps and records the step; note adds the change the caller reads.
func (w *routeWitness) click(control string, rect image.Rectangle, p image.Point, settle int, note func() string) {
	p = w.windowPoint(p)
	before := w.screen()
	for _, edge := range []string{"hover", "press", "release"} {
		w.pointer(edge, p)
		if _, _, err := w.app.HeadlessPicture(ui.HeadlessRequest{Part: ui.HeadlessPartFrame}); err != nil && w.app.HeadlessGameplayScreen() != ui.ScreenMap {
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

// drag presses at from, moves to to in steps, releases there, composing a
// frame after each edge, runs settle steps and records the step.
func (w *routeWitness) drag(control string, rect image.Rectangle, from, to image.Point, settle int, note func() string) {
	from, to = w.windowPoint(from), w.windowPoint(to)
	before := w.screen()
	w.pointer("hover", from)
	w.pointer("press", from)
	for i := 1; i <= 4; i++ {
		w.pointer("move", from.Add(to.Sub(from).Mul(i).Div(4)))
		if _, _, err := w.app.HeadlessPicture(ui.HeadlessRequest{Part: ui.HeadlessPartFrame}); err != nil {
			w.t.Errorf("%s %s: frame during drag: %v", w.stage, control, err)
		}
	}
	w.pointer("release", to)
	w.run(settle)
	extra := ""
	if note != nil {
		extra = note()
	}
	w.record(control, rect, "drag", w.result(before, extra))
}

// key sends one key, runs settle steps and records the step.
func (w *routeWitness) key(name string, settle int) { w.keyNote(name, settle, nil) }

// keyNote is key with note adding the change the caller reads.
func (w *routeWitness) keyNote(name string, settle int, note func() string) {
	before := w.screen()
	if err := w.app.HeadlessKey(name); err != nil {
		w.t.Errorf("%s: key %s: %v", w.stage, name, err)
	}
	w.run(settle)
	extra := ""
	if note != nil {
		extra = note()
	}
	w.record("key "+name, image.Rectangle{}, "key", w.result(before, extra))
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

// The original's menu points, Hall of Fame seal and credits wait.
var (
	routeMenuLoad      = image.Pt(385, 95)
	routeMenuHall      = image.Pt(458, 298)
	routeMenuCredits   = image.Pt(250, 380)
	routeMenuCutscenes = image.Pt(170, 295)
	routeHallSeal      = image.Pt(575, 435)
	routeCreditsShot   = 4200 * time.Millisecond // the original shot s1.36 4.2 s after its click
	routeCreditsWait   = 10 * time.Second
)

// menuButton is the menu region at p, which must bind action.
func (w *routeWitness) menuButton(regions []routeRegion, p image.Point, action menu.Action) image.Rectangle {
	b := w.f.Assets.ButtonAt(p)
	if got := w.f.Assets.Binding(b).Action; b == 0 || got != action {
		w.t.Errorf("menu: the original's point %v answers button %d action %d, want action %d", p, b, got, action)
	}
	for _, r := range regions {
		if strings.HasPrefix(r.key, fmt.Sprintf("%d ", b)) {
			return r.rect
		}
	}
	return image.Rectangle{}
}

// escape sends the original's Esc of step and records it; on the menu the
// App answers an exit request, which the row states.
func (w *routeWitness) escape(step string) {
	before := w.screen()
	extra := ""
	if err := w.app.HeadlessKey("escape"); err != nil {
		extra = "exit requested: " + err.Error()
	}
	w.run(2)
	w.record("key escape ("+step+")", image.Rectangle{}, "key", w.result(before, extra))
}

// waitFor runs the headless clock for d and records the screen.
func (w *routeWitness) waitFor(control string, d time.Duration) {
	before := w.screen()
	w.run(int(d / (20 * time.Millisecond)))
	w.record(control, image.Rectangle{}, "wait", w.result(before, fmt.Sprintf("after %v", d)))
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

// menuAction is the action the button of menu region r binds.
func (w *routeWitness) menuAction(r routeRegion) menu.Action {
	b := 0
	fmt.Sscanf(r.key, "%d", &b)
	return w.f.Assets.Binding(b).Action
}

// menuTrip takes trip t from the menu, its button at rect.
func (w *routeWitness) menuTrip(t routeTrip, rect image.Rectangle) {
	w.click(t.press, rect, t.at, t.settle, nil)
	side := w.screen()
	if t.explore {
		if side == "menu" {
			return
		}
		t = routeExploredTrip(t, side)
	}
	if t.roll != "" {
		w.waitFor(t.roll, routeCreditsWait)
	}
	w.escape(t.esc)
	if t.explore && w.screen() != "menu" {
		w.t.Errorf("menu: Esc leaves %s on %s", side, w.screen())
		return
	}
	if t.again != "" && w.screen() != side {
		w.click(t.again, rect, t.at, t.againSettle, nil)
	}
	for _, b := range t.buttons {
		if w.screen() != side {
			break
		}
		w.hover(b.hover, image.Rectangle{}, b.at)
		w.click(b.click, image.Rectangle{}, b.at, 2, nil)
	}
	if w.screen() == "menu" {
		return
	}
	if t.leave == "" {
		w.expect("menu", t.esc)
		return
	}
	w.t.Errorf("menu: no button leaves %s; it shows %s", side, w.screen())
	w.escape(t.leave)
}

// routeExploredTrip is trip t found on the mask, after its press opened
// side: the rest of the trip by routeSideScreens.
func routeExploredTrip(t routeTrip, side string) routeTrip {
	s := routeSideScreens[side]
	t.esc = side + ", leave by Esc"
	t.again, t.againSettle = t.press+" again", 2
	if s.roll {
		t.roll = side + " roll"
		t.againSettle = int(routeCreditsShot / (20 * time.Millisecond))
	}
	for _, b := range s.buttons {
		t.buttons = append(t.buttons, routeSideButton{side + " " + b.what, side + " " + b.what, b.at})
	}
	t.leave = side + ", after its buttons"
	return t
}

// menu records the startup movies, hovers each button the menu's mask
// answers, takes the side trips, then presses New Game.
func (w *routeWitness) menu(d routeData) {
	m := d.menu
	w.startupMovies(m.launch)
	regions := w.menuRegions()
	if m.buttons > 0 && len(regions) != m.buttons {
		w.t.Errorf("menu: the mask answers %d buttons, want %d", len(regions), m.buttons)
	}
	if len(regions) == 0 {
		w.t.Fatal("menu: the mask answers no button")
	}
	newGame := -1
	for i, r := range regions {
		w.hover(r.key, r.rect, r.at)
		if w.menuAction(r) == menu.ActionNewGame {
			newGame = i
		}
	}
	if newGame < 0 {
		w.t.Fatal("menu: no New Game button")
	}
	for _, t := range m.trips {
		w.menuTrip(t, w.menuButton(regions, t.at, t.action))
	}
	if m.trips == nil {
		for _, r := range regions {
			switch w.menuAction(r) {
			case menu.ActionNewGame:
			case menu.ActionExit:
				w.record(r.key, r.rect, "wait", "not clicked: Exit ends the program")
			default:
				w.menuTrip(routeTrip{press: r.key, at: r.at, settle: 2, explore: true}, r.rect)
			}
		}
	}
	w.expect("menu", "the side trips")
	r := regions[newGame]
	w.click(r.key, r.rect, r.at, 2, nil)
	w.expect("chargen/precreate", "New Game")
	for _, n := range m.notes {
		w.record(n.control, image.Rectangle{}, "wait", n.result)
	}
}

// routeLaunchFrame is the frame of a startup movie the walk records.
const routeLaunchFrame = 5

// startupMovies plays the startup movies as the App's Run does and records
// frame routeLaunchFrame: of the launch movie, whose Esc ends the sequence,
// or of each movie, each left by Esc. The movie track goes to no device.
func (w *routeWitness) startupMovies(launch routeLaunch) {
	w.app.SetCutsceneAudio(nil)
	if !w.app.PlayStartupCutscenes() {
		if launch.movie != "" {
			w.t.Fatalf("menu: no startup movie plays: %v", w.app.CutsceneError())
		}
		w.record("startup movies", image.Rectangle{}, "wait", fmt.Sprintf("none plays: %v", w.app.CutsceneError()))
		return
	}
	for i := 1; w.app.Screen() == ui.ScreenCutscene && i <= 8; i++ {
		name := w.app.CutsceneName()
		shown := func() bool {
			if launch.movie != "" {
				return w.app.CutsceneName() == launch.movie && w.app.CutsceneFrameNumber() >= routeLaunchFrame
			}
			return w.app.CutsceneName() != name || w.app.CutsceneFrameNumber() >= routeLaunchFrame
		}
		deadline := time.Now().Add(2 * time.Minute)
		for w.app.Screen() == ui.ScreenCutscene && !shown() && time.Now().Before(deadline) {
			// A movie's last frame waits for its presentation, which a
			// composition makes.
			if _, _, err := w.app.HeadlessPicture(ui.HeadlessRequest{Part: ui.HeadlessPartFrame}); err != nil {
				w.t.Fatalf("menu: startup movie frame: %v", err)
			}
			w.run(1)
			time.Sleep(time.Millisecond)
		}
		shot, esc := fmt.Sprintf("startup movie %d", i), fmt.Sprintf("startup movie %d", i)
		if launch.movie != "" {
			if w.app.Screen() != ui.ScreenCutscene || !shown() || w.app.CutsceneFrameNumber() != routeLaunchFrame {
				w.t.Fatalf("menu: the startup movie %q shows frame %d on %v", w.app.CutsceneName(), w.app.CutsceneFrameNumber(), w.app.Screen())
			}
			shot, esc = launch.shot, launch.esc
		}
		if w.app.Screen() != ui.ScreenCutscene {
			break
		}
		w.record(shot, image.Rectangle{}, "wait", fmt.Sprintf("%s frame %d", w.app.CutsceneName(), w.app.CutsceneFrameNumber()))
		w.escape(esc)
		if launch.movie != "" {
			break
		}
	}
	w.expect("menu", "the startup movies")
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
func (w *routeWitness) preCreate(routeData) {
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
// pre-create and forward again to the opening values, and plays them into
// the row's play screen.
func (w *routeWitness) detail(d routeData) {
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
	w.expect(d.play, "Play")
}

// squareRegions is the shown room's controls by its scene's own hit and
// tip tests, outside a showing tip panel.
func (w *routeWitness) squareRegions() []routeRegion {
	size, at := w.squareAt()
	return routeRegions(size, 2, at)
}

// squareAt is the shown room's size and the key its hit and tip tests
// answer at a point.
func (w *routeWitness) squareAt() (image.Point, func(image.Point) (string, bool)) {
	v := w.town.TownSquareView()
	if v.Scene == nil {
		w.t.Fatalf("%s: the room has no scene", w.stage)
	}
	// A room page names its seats and buttons from the description, the
	// rectangles its hit test reads.
	var page *town.RoomSpec
	seats := 0
	if d, room, ok := w.town.composedRoom(); ok {
		if page = d.Room(room); page != nil {
			list, _ := w.town.innPageSeats()
			seats = len(list)
		}
	}
	return v.Scene.Size(), func(p image.Point) (string, bool) {
		if v.Tip.Covers(p) {
			return "", false
		}
		c, ok := v.Scene.ControlAt(p)
		tip, tok := v.Scene.TipAt(p)
		part := ""
		layout := buildRoomPage(page, nil, seats, 0)
		if i := layout.SeatAt(p); i >= 0 {
			part = fmt.Sprintf(" seat %d", i)
		}
		if page != nil && part == "" {
			if i := layout.ButtonAt(p); i >= 0 {
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
	}
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
		case target != "menu" && k == ui.TownSquareControlDoor && door < len(rows) && (rows[door].Text == target || strings.HasPrefix(rows[door].Text, target+" ")):
			return r
		}
	}
	w.t.Fatalf("%s: the room has no %s control", w.stage, target)
	return routeRegion{}
}

// closeTip hovers and closes a showing tip panel the row's way: at the
// middle of the close rectangle of the panel view answers, or at the close
// point the App answers.
func (w *routeWitness) closeTip(how routeTipClose, view func() ui.TipPanelView) {
	var rect image.Rectangle
	var p image.Point
	if how == routeTipClosePoint {
		x, y, err := w.app.HeadlessTipClosePoint()
		if err != nil {
			return
		}
		p = w.nativePoint(x, y)
	} else {
		v := view()
		if !v.Showing() {
			return
		}
		rect = ui.TipPanelCloseRect(v.Rect)
		p = routeMiddle(rect)
	}
	w.wait("tip panel")
	w.hover("tip close", rect, p)
	w.click("tip close", rect, p, 2, nil)
}

// squareTip is the square's tip panel.
func (w *routeWitness) squareTip() ui.TipPanelView { return w.town.TownSquareView().Tip }

// routeSquareBlank is the original's town points that show no caption, with
// their step ids (s3.04, s3.09 to s3.14).
var routeSquareBlank = []struct {
	step, what string
	at         image.Point
}{
	{"s3.04", "hanging sign on the left building", image.Pt(115, 370)},
	{"s3.09", "horse and cart", image.Pt(510, 440)},
	{"s3.10", "man at the statue base", image.Pt(345, 445)},
	{"s3.11", "woman at the statue base", image.Pt(410, 445)},
	{"s3.12", "blue shield sign", image.Pt(385, 250)},
	{"s3.13", "right building upper wall", image.Pt(560, 250)},
	{"s3.14", "left building upper wall", image.Pt(40, 200)},
}

// square records the arrival, closes the town's tip panel, enters the town
// from the world map where the town opens on it, and hovers every control
// and tip area of the square.
func (w *routeWitness) square(d routeData) {
	w.wait(d.arrive)
	w.closeTip(d.tip, w.squareTip)
	if w.screen() == "town/worldmap" {
		x, y, err := w.app.HeadlessWorldMapTownPoint()
		if err != nil {
			w.t.Fatalf("town: the world map has no town point: %v", err)
		}
		w.hover("world map town", image.Rectangle{}, w.nativePoint(x, y))
		w.click("world map town", image.Rectangle{}, w.nativePoint(x, y), 2, nil)
		w.until(200, func() bool { return w.screen() != "town/worldmap" })
		w.dialoguePages("town", nil)
		w.closeTip(d.tip, w.squareTip)
	}
	w.expect("town/square", "the town's first screen")
	for _, r := range w.squareRegions() {
		w.hover(r.key, r.rect, r.at)
	}
}

// squareBlank closes a showing tip panel and hovers the original's
// no-caption points with the region our hit test answers at each. It runs
// after the last room visit: a square actor keeps its phase across visits.
func (w *routeWitness) squareBlank(routeData) {
	if v := w.town.TownSquareView().Tip; v.Showing() {
		closeRect := ui.TipPanelCloseRect(v.Rect)
		w.click("tip close, before s3.04", closeRect, routeMiddle(closeRect), 2, nil)
	}
	regions := w.squareRegions()
	_, at := w.squareAt()
	for _, b := range routeSquareBlank {
		control, rect := fmt.Sprintf("%s (%s): no region", b.what, b.step), image.Rectangle{}
		if k, ok := at(b.at); ok {
			r, _ := routeFind(regions, k)
			control, rect = fmt.Sprintf("%s (%s): %s", b.what, b.step, k), r.rect
		}
		w.hover(control, rect, b.at)
	}
}

// statue opens the square's menu, records its rows and returns.
func (w *routeWitness) statue(routeData) {
	r := w.sceneControl("menu")
	w.click("menu", r.rect, r.at, 2, nil)
	w.expect("gamemenu", "the statue")
	w.menuChoose("abort", "abort game (m1.32)", nil)
	w.expect("gamemenu", "abort game")
	w.menuChoose("page-return", "return to game, abort page", nil)
	if w.screen() == "gamemenu" {
		w.menuChoose("return", "return", nil)
	}
	w.expect("town/square", "return")
}

// menuChoose chooses the game menu's row by its action through the App's
// arrow and Enter dispatch, settles and records the step.
func (w *routeWitness) menuChoose(action, control string, note func() string) {
	before := w.screen()
	if err := w.app.HeadlessGameMenuAction(action); err != nil {
		w.t.Fatalf("%s: %s: %v", w.stage, control, err)
	}
	w.run(2)
	extra := ""
	if note != nil {
		extra = note()
	}
	w.record(control, image.Rectangle{}, "click", w.result(before, extra))
}

// dialoguePages turns every page of the open town dialogue with its button
// and answers the page count; before, when set, acts on each page first.
func (w *routeWitness) dialoguePages(what string, before func(page int)) int {
	for pages := 0; pages < 64; pages++ {
		if _, open := w.town.dialogueBody(); !open {
			return pages
		}
		if before != nil {
			before(pages)
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
	w.dialoguePages(target, nil)
	w.expect("town/"+room, target)
	w.wait(room)
}

// room enters the square's door the row names, closes the room's tip panel
// and hovers every control of its page; walks the character pane and talks
// to each seat where the row says; clicks each pick of a surface page once;
// and leaves by the scene's exit control, or by Esc and the surface page's
// last button.
func (w *routeWitness) room(d routeData) {
	r := d.room
	w.enter(r.door, r.room)
	w.closeTip(d.tip, w.squareTip)
	regions := w.roomRegions(r.surface)
	if len(regions) == 0 {
		w.record(r.room, image.Rectangle{}, "wait", "no control answers")
	}
	for _, c := range regions {
		w.hover(c.key, c.rect, c.at)
	}
	if r.pane {
		w.innPane()
	}
	if r.talks {
		w.innTalks()
	}
	if r.surface {
		w.surfacePicks(regions)
		var exit routeRegion
		for _, c := range w.surfaceRegions() {
			if strings.HasPrefix(c.key, fmt.Sprintf("control %d/", ui.TownSurfaceControlButton)) {
				exit = c
			}
		}
		w.leaveRoom(r.room, exit.key, exit.rect, exit.at)
		return
	}
	c := w.sceneControl(r.exit)
	w.click(r.exit, c.rect, c.at, 2, nil)
	w.until(200, func() bool { return w.screen() == "town/square" })
	w.expect("town/square", "Exit")
}

// roomRegions is the open room's controls: a surface page's by its own hit
// test, a scene page's as the square's.
func (w *routeWitness) roomRegions(surface bool) []routeRegion {
	if surface {
		return w.surfaceRegions()
	}
	return w.squareRegions()
}

// surfaceRegions is the tavern or school page's controls by its own hit test.
func (w *routeWitness) surfaceRegions() []routeRegion {
	v := w.town.TownSurface()
	return routeRegions(image.Pt(640, 480), 2, func(p image.Point) (string, bool) {
		if c, ok := ui.TownSurfaceControlAt(v, p); ok {
			return fmt.Sprintf("control %d/%d", c.Kind, c.Index), true
		}
		return "", false
	})
}

// surfacePicks clicks each cell, mode and book control of a surface page
// once and pages the dialogue each opens.
func (w *routeWitness) surfacePicks(regions []routeRegion) {
	for _, r := range regions {
		var kind, index int
		fmt.Sscanf(r.key, "control %d/%d", &kind, &index)
		switch ui.TownSurfaceControlKind(kind) {
		case ui.TownSurfaceControlCell, ui.TownSurfaceControlMode, ui.TownSurfaceControlBook:
			w.click(r.key, r.rect, r.at, 2, nil)
			w.dialoguePages(r.key, nil)
		}
	}
}

// innTalks selects each inn seat with a click, which opens no talk, talks
// to each selected person page by page by the button that acts on the seat,
// and reports the hire offer. The talk with patron 2 takes the original's
// modality steps.
func (w *routeWitness) innTalks() {
	seats, hires := w.town.innPageSeats()
	if hires == 0 {
		w.record("hire offer", image.Rectangle{}, "wait", "none offered: the inn seats no hire actor")
	}
	for i := range seats {
		seat, ok := w.innSeat(i)
		if !ok {
			w.t.Fatalf("tavern: seat %d answers no pixel", i)
		}
		label := w.town.Rows()[i+1].Text
		w.click(fmt.Sprintf("seat %d", i), seat.rect, seat.at, 2, nil)
		if w.town.composed.seat != i {
			w.t.Errorf("tavern: a click on seat %d selects seat %d", i, w.town.composed.seat)
		}
		if _, open := w.town.dialogueBody(); open {
			w.t.Errorf("tavern: a click on seat %d opened a dialogue", i)
		}
		r := w.sceneControl(label)
		w.click(label, r.rect, r.at, 2, nil)
		var modal func(int)
		if i == routeTalkSeat {
			modal = w.talkModality
		}
		if n := w.dialoguePages(label, modal); n == 0 && i >= hires {
			w.t.Errorf("tavern: %s opened no dialogue", label)
		}
	}
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

// The original's talk-box modality steps on patron 2's talk.
var (
	routeTalkSeat    = 2
	routeTalkWait    = 10 * time.Second
	routeTalkButton  = image.Pt(553, 112)
	routeTalkPatrons = map[int]struct {
		step   string
		patron int
		at     image.Point
	}{
		1: {"s3.43", 1, image.Pt(247, 445)},
		2: {"s3.45", 0, image.Pt(200, 445)},
	}
)

// talkModality takes the original's step on the open talk box's page.
func (w *routeWitness) talkModality(page int) {
	if page == 0 {
		w.waitFor("talk box (s3.41)", routeTalkWait)
		return
	}
	p, ok := routeTalkPatrons[page]
	if !ok {
		return
	}
	body, _ := w.town.dialogueBody()
	note := func() string {
		now, _ := w.town.dialogueBody()
		return fmt.Sprintf("seat=%d page kept=%v", w.town.composed.seat, now == body)
	}
	w.click(fmt.Sprintf("patron %d outside the talk box (%s)", p.patron, p.step), image.Rectangle{}, p.at, 2, note)
	w.click(fmt.Sprintf("talk outside the talk box (%s)", p.step), image.Rectangle{}, routeTalkButton, 2, note)
}

// The pane points the original's route used (R2-SESSION-203, R2-SESSION-204,
// R2-SESSION-214, R2-SESSION-217).
var (
	routePaneEye      = image.Pt(622, 253)
	routePaneTopLeft  = image.Pt(490, 255)
	routePanePrevious = image.Pt(490, 457)
	routePaneNext     = image.Pt(612, 457)
	routePanePortrait = image.Pt(555, 330)
	routeInnOval      = image.Pt(80, 330)
)

// paneCorner is corner c's rectangle on the town pane.
func paneCorner(c ui.CharacterPaneCorner) image.Rectangle {
	return ui.CharacterPaneCornerRect(ui.TownCharacterRegion, c)
}

// paneClick clicks p and waits out the hover delay, so the frame shows the
// caption the pointer then stands under.
func (w *routeWitness) paneClick(control string, rect image.Rectangle, p image.Point, note func() string) {
	w.click(control, rect, p, w.app.TooltipDelay()/20+5, note)
}

// rightClick presses and releases the right button at p and records the
// step.
func (w *routeWitness) rightClick(control string, rect image.Rectangle, p image.Point) {
	p = w.windowPoint(p)
	before := w.screen()
	for _, edge := range []string{"hover", "right-press", "right-release"} {
		w.pointer(edge, p)
	}
	w.run(2)
	w.record(control, rect, "right-click", w.result(before, ""))
}

// innPane hovers the inn's character pane corners, the hero's portrait and
// the patron's oval, swaps the portrait and the statistics card by the eye
// and by Tab twice each, and clicks both picker arrows and the portrait.
func (w *routeWitness) innPane() {
	if !w.town.TownSquareView().Pane {
		w.t.Fatal("tavern: the inn shows no character pane")
	}
	w.hover("pane eye", paneCorner(ui.CharacterPaneMode), routePaneEye)
	w.hover("pane top left", paneCorner(ui.CharacterPaneBook), routePaneTopLeft)
	w.hover("pane previous", paneCorner(ui.CharacterPanePrev), routePanePrevious)
	w.hover("pane next", paneCorner(ui.CharacterPaneNext), routePaneNext)
	w.hover("pane portrait", ui.TownCharacterRegion, routePanePortrait)
	w.hover("patron oval", image.Rect(0, 238, 160, 480), routeInnOval)
	stats := func() bool { return w.town.TownSquareView().Hero.Statistics }
	note := func() string { return fmt.Sprintf("statistics=%v", stats()) }
	for i := range 2 {
		w.paneClick("pane eye", paneCorner(ui.CharacterPaneMode), routePaneEye, note)
		if stats() != (i == 0) {
			w.t.Errorf("tavern: eye click %d shows statistics %v", i+1, stats())
		}
	}
	for i := range 2 {
		w.key("tab", 2)
		if stats() != (i == 0) {
			w.t.Errorf("tavern: Tab %d shows statistics %v", i+1, stats())
		}
	}
	member := func() string {
		h := w.town.TownSquareView().Hero
		return fmt.Sprintf("member=%d/%d %s", h.Member, h.MemberCount, routeText(h.Subject.Char.Name))
	}
	before := member()
	w.paneClick("pane previous", paneCorner(ui.CharacterPanePrev), routePanePrevious, member)
	w.paneClick("pane next", paneCorner(ui.CharacterPaneNext), routePaneNext, member)
	if got := member(); got != before {
		w.t.Errorf("tavern: the picker arrows move the pane from %s to %s", before, got)
	}
	w.paneClick("pane portrait", ui.TownCharacterRegion, routePanePortrait, nil)
	w.rightClick("pane portrait", ui.TownCharacterRegion, routePanePortrait)
}

// innSeat is the inn page's seat i by the page's own seat rectangles.
func (w *routeWitness) innSeat(i int) (routeRegion, bool) {
	return routeFind(w.squareRegions(), fmt.Sprintf("control %d/%d seat %d", ui.TownSquareControlSelect, i+1, i))
}

// shopControlRow is one control of the shop page's layout.
type shopControlRow struct {
	kind  string
	index int
	rect  image.Rectangle
}

// shopControls lists every control of the shop page's own layout.
func shopControls(v ui.ShopScreenView) []shopControlRow {
	l := shopLayout(v)
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
		return w.nativePoint(x, y)
	}
	return routeMiddle(c.rect)
}

// The original's buy by drag (s3.64): shelf item 0 to the table.
var (
	routeShopShelfItem = image.Pt(43, 70)
	routeShopTable     = image.Pt(100, 330)
)

// shopDrag drags the first shelf's item at the original's point to the
// table at its point, as the original stages a buy, and takes the staging
// back with undo.
func (w *routeWitness) shopDrag() {
	v := w.town.ShopScreen()
	undo := shopControlRow{"button", 0, v.Layout.Button[0]}
	if v.Chosen != 0 {
		pick := shopControlRow{"shelf_pick", 0, v.Layout.ShelfPick[0]}
		w.click("shelf_pick 0 (s3.64)", pick.rect, w.shopPoint(pick), 2, nil)
	}
	gold := w.f.Town.Gold()
	staged := func() string {
		v := w.town.ShopScreen()
		return fmt.Sprintf("buy=%d sell=%d gold=%d table0=%v", v.Buy, v.Sell, w.f.Town.Gold(), v.Table[0].Icon != nil)
	}
	w.drag("shelf item 0 to the table (s3.64)", v.Layout.ShelfCellRect(0), routeShopShelfItem, routeShopTable, 2, staged)
	if v := w.town.ShopScreen(); v.Table[0].Icon == nil || v.Buy == 0 {
		w.t.Errorf("shop: the drag from %v to %v staged buy %d", routeShopShelfItem, routeShopTable, v.Buy)
	}
	w.click("button 0 undo, after s3.64", undo.rect, w.shopPoint(undo), 2, staged)
	if v := w.town.ShopScreen(); v.Buy != 0 || w.f.Town.Gold() != gold {
		w.t.Errorf("shop: undo left buy %d and gold %d from %d", v.Buy, w.f.Town.Gold(), gold)
	}
}

// shopLayout is the shop page's layout, the first game's where the view
// names none.
func shopLayout(v ui.ShopScreenView) *ui.ShopLayout {
	if v.Layout == nil {
		fl := ui.FirstShopLayout()
		return &fl
	}
	return v.Layout
}

// shop hovers every control of the shop page and, where the row says, the
// character pane and its book; opens each shelf; buys the cheapest item and
// sells it back, which a town save walk does not; stages the original's
// drag buy and takes it back where the row says; and leaves.
func (w *routeWitness) shop(d routeData) {
	s := d.shop
	w.enter("SHOP", "shop")
	w.closeTip(d.tip, func() ui.TipPanelView { return w.town.ShopScreen().TipPanel })
	v := w.town.ShopScreen()
	l := shopLayout(v)
	controls := shopControls(v)
	for _, c := range controls {
		w.hover(fmt.Sprintf("%s %d", c.kind, c.index), c.rect, w.shopPoint(c))
	}
	if s.pane {
		w.shopPane(l)
	}
	shelf, cell, price := w.shopShelves(l, controls, s.inspect)
	switch {
	case w.save != "":
		w.record("shop", image.Rectangle{}, "wait", "no buy: a town save walk changes nothing")
	case shelf < 0 && s.inspect:
		w.t.Fatal("shop: no item on any shelf")
	case shelf < 0:
		w.record("shop", image.Rectangle{}, "wait", "no item on any shelf")
	default:
		w.shopTrade(l, shelf, cell, price, s)
	}
	if s.undo {
		w.shopDrag()
	}
	last := shopControlRow{"button", len(l.Button) - 1, l.Button[len(l.Button)-1]}
	if s.named {
		w.click(fmt.Sprintf("button %d leave", last.index), last.rect, w.shopPoint(last), 2, nil)
		w.until(200, func() bool { return w.screen() == "town/square" })
		w.expect("town/square", "Leave")
		return
	}
	w.leaveRoom("shop", fmt.Sprintf("button %d", last.index), last.rect, w.shopPoint(last))
}

// shopPane hovers the shop's character pane book, eye and portrait and
// toggles the book twice.
func (w *routeWitness) shopPane(l *ui.ShopLayout) {
	if !l.Character || !l.Book {
		w.t.Error("shop: the shop shows no character pane with its book")
	}
	w.hover("pane book", paneCorner(ui.CharacterPaneBook), routePaneTopLeft)
	w.hover("pane eye", paneCorner(ui.CharacterPaneMode), routePaneEye)
	w.hover("pane portrait", ui.TownCharacterRegion, routePanePortrait)
	book := func() string { return fmt.Sprintf("book=%v", w.town.ShopScreen().Book) }
	for i := range 2 {
		w.paneClick("pane book", paneCorner(ui.CharacterPaneBook), routePaneTopLeft, book)
		if w.town.ShopScreen().Book != (i == 0) {
			w.t.Errorf("shop: book click %d shows the book %v", i+1, w.town.ShopScreen().Book)
		}
	}
}

// shopShelves opens each shelf and answers the cheapest item's shelf, cell
// and price, shelf -1 for none; inspect checks each shelf's choice, hovers
// its first item and records an empty shelf.
func (w *routeWitness) shopShelves(l *ui.ShopLayout, controls []shopControlRow, inspect bool) (shelf, cell int, price int32) {
	shelf, cell = -1, -1
	for _, c := range controls {
		if c.kind != "shelf_pick" {
			continue
		}
		w.click(fmt.Sprintf("shelf_pick %d", c.index), c.rect, w.shopPoint(c), 2, func() string {
			return fmt.Sprintf("shelf=%d", w.town.ShopScreen().Chosen)
		})
		v := w.town.ShopScreen()
		if inspect && v.Chosen != c.index {
			w.t.Errorf("shop: the shelf %d click shows shelf %d", c.index, v.Chosen)
		}
		items := 0
		for i, it := range v.Shelf {
			if it.Icon == nil {
				continue
			}
			if items++; items == 1 && inspect {
				w.hover(fmt.Sprintf("shelf %d first item", i), l.ShelfCellRect(i), w.shopPoint(shopControlRow{kind: "shelf", index: i}))
			}
			if shelf < 0 || it.Price < price {
				shelf, cell, price = c.index, i, it.Price
			}
		}
		if items == 0 && inspect {
			w.record(fmt.Sprintf("shelf_pick %d", c.index), c.rect, "wait", "empty shelf")
		}
	}
	return shelf, cell, price
}

// shopTrade buys the item at cell of shelf for price by a drag to the table
// and sells it back, by the named buttons or by each middle button while
// the trade stays staged.
func (w *routeWitness) shopTrade(l *ui.ShopLayout, shelf, at int, price int32, s routeShopData) {
	pick := shopControlRow{"shelf_pick", shelf, l.ShelfPick[shelf]}
	w.click(fmt.Sprintf("shelf_pick %d", shelf), pick.rect, w.shopPoint(pick), 2, nil)
	gold := w.f.Town.Gold()
	cell := shopControlRow{"shelf", at, l.ShelfCellRect(at)}
	table := shopControlRow{"table", 0, l.TableCellRect(0)}
	staged := func() string {
		v := w.town.ShopScreen()
		return fmt.Sprintf("buy=%d sell=%d gold=%d", v.Buy, v.Sell, w.f.Town.Gold())
	}
	if s.named {
		w.drag(fmt.Sprintf("shelf %d cheapest to table 0", at), cell.rect, w.shopPoint(cell), w.shopPoint(table), 2, func() string {
			return fmt.Sprintf("buy=%d", w.town.ShopScreen().Buy)
		})
		if v := w.town.ShopScreen(); v.Buy != price || v.Table[0].Icon == nil {
			w.t.Errorf("shop: the drag to the table staged buy %d, want %d", v.Buy, price)
		}
		w.shopPress(l, 1, "buy", gold)
	} else {
		w.drag(fmt.Sprintf("shelf %d cheapest (%d) to table 0", at, price), cell.rect, w.shopPoint(cell), w.shopPoint(table), 2, staged)
		w.shopPressWhile(l, gold, func(v ui.ShopScreenView) bool { return v.Buy != 0 }, staged)
	}
	if w.f.Town.Gold() != gold-int(price) {
		w.t.Errorf("shop: buying the %d item left gold %d from %d", price, w.f.Town.Gold(), gold)
	}
	gold = w.f.Town.Gold()
	mine := -1
	for i, c := range w.town.ShopScreen().Pack {
		if c.Icon != nil && !c.Money {
			mine = i
		}
	}
	if mine < 0 {
		if s.inspect {
			w.t.Fatal("shop: the bought item is not in the pack")
		}
		w.t.Error("shop: the bought item is not in the pack")
		return
	}
	pack := shopControlRow{"pack", mine, l.PackCellRect(mine)}
	if s.named {
		w.click(fmt.Sprintf("pack %d bought", mine), pack.rect, w.shopPoint(pack), 2, func() string {
			return fmt.Sprintf("sell=%d", w.town.ShopScreen().Sell)
		})
		w.shopPress(l, 2, "sell", gold)
	} else {
		w.drag(fmt.Sprintf("pack %d bought, to table 0", mine), pack.rect, w.shopPoint(pack), w.shopPoint(table), 2, staged)
		w.shopPressWhile(l, gold, func(v ui.ShopScreenView) bool { return v.Sell != 0 }, staged)
	}
	if w.f.Town.Gold() <= gold {
		w.t.Errorf("shop: selling back left gold %d from %d", w.f.Town.Gold(), gold)
	}
}

// shopPress presses button i, named name, once and notes the gold from
// gold.
func (w *routeWitness) shopPress(l *ui.ShopLayout, i int, name string, gold int) {
	b := shopControlRow{"button", i, l.Button[i]}
	w.click(fmt.Sprintf("button %d %s", i, name), b.rect, w.shopPoint(b), 2, func() string {
		return fmt.Sprintf("gold %d->%d", gold, w.f.Town.Gold())
	})
}

// shopPressWhile presses each middle button while the gold stays gold and
// the trade stays staged, noting staged.
func (w *routeWitness) shopPressWhile(l *ui.ShopLayout, gold int, staging func(ui.ShopScreenView) bool, staged func() string) {
	for i := range l.Button {
		b := shopControlRow{"button", i, l.Button[i]}
		if w.f.Town.Gold() == gold && staging(w.town.ShopScreen()) && i > 0 && i < len(l.Button)-1 {
			w.click(fmt.Sprintf("button %d", i), b.rect, w.shopPoint(b), 2, staged)
		}
	}
}

// wayOut takes the gates to the world map.
func (w *routeWitness) wayOut(routeData) {
	r := w.sceneControl("GATES")
	w.click("GATES", r.rect, r.at, 2, nil)
	if !w.until(600, func() bool { return w.screen() == "town/worldmap" }) {
		w.dialoguePages("GATES", nil)
		w.t.Fatalf("way out: the gates show %s", w.screen())
	}
	w.wait("world map")
}

// worldMap hovers every card and site the world map's hit tests answer
// while it shows, chooses the first mission where the row says, and records
// the travel until the mission opens.
func (w *routeWitness) worldMap(d routeData) {
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
	if d.choose {
		w.missionChoice()
	}
	w.travel()
}

// travel records the travel until the first mission opens.
func (w *routeWitness) travel() {
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

// missionDialogue runs the first ticks and pages each notice that opens;
// on a row of the new game's mission it checks the mission opened.
func (w *routeWitness) missionDialogue(d routeData) {
	if d.newGame && w.f.liveMission != w.f.NewGameMission() {
		w.t.Errorf("mission: opened %d, want %d", w.f.liveMission, w.f.NewGameMission())
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
	camera := *w.f.live.view.Camera()
	defer func() { *w.f.live.view.Camera() = camera }()
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
			_, st, _ := w.app.HeadlessPicture(ui.HeadlessRequest{Part: ui.HeadlessPartTooltip})
			if st.Tooltip.Target == "" {
				continue
			}
			cell := image.Rect(x-step/2, y-step/2, x+step/2, y+step/2)
			key := st.Tooltip.Target
			if strings.HasPrefix(key, "item/") {
				w.run(w.app.TooltipDelay()/20 + 5)
				key += "\x00" + strings.Join(w.app.HeadlessTooltipLines(), "\x00")
			}
			r, seen := regions[key]
			if !seen {
				r = routeRegion{key: st.Tooltip.Target, rect: cell, at: image.Pt(wx, wy)}
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

// packHovers hovers every control the opened pack row adds: a new mission
// opens with the row closed (DIV-4020), so its arrows, items and background
// answer only after the first press of the pack corner.
func (w *routeWitness) packHovers(hovered map[string]string) {
	w.newHovers(hovered, func(key string) bool {
		return strings.HasPrefix(key, "pack-") || strings.HasPrefix(key, "item/")
	})
}

// newHovers hovers every control the mission screen's hover test answers
// that hovered does not hold yet and keep admits, and adds its text.
func (w *routeWitness) newHovers(hovered map[string]string, keep func(key string) bool) {
	for _, r := range w.missionTargets(8) {
		if _, seen := hovered[r.key]; seen || !keep(r.key) {
			continue
		}
		w.hover(r.key, r.rect, r.at)
		hovered[r.key] = strings.Join(w.app.HeadlessTooltipLines(), " ")
		if hovered[r.key] == "" {
			w.t.Errorf("mission: %s shows no hover text", r.key)
		}
	}
}

// missionScreen pauses the mission, selects the hero, hovers every control
// the mission screen's hover test answers, presses the pack corner twice,
// records the hero and the party, and with the row's spellbook clicks the
// spellbook icon twice.
func (w *routeWitness) missionScreen(d routeData) {
	w.key("0", 1)
	if len(w.f.live.mission.ids) == 0 {
		w.t.Fatal("mission: no party unit")
	}
	hero := w.f.live.mission.ids[0]
	if x, y, err := w.app.HeadlessEntityPoint(uint32(hero)); err != nil {
		if d.heroShown {
			w.t.Errorf("mission: the hero answers no pixel: %v", err)
		}
		w.record("hero", image.Rectangle{}, "wait", "no click: the hero answers no window pixel: "+err.Error())
	} else {
		w.click("hero", image.Rectangle{}, image.Pt(x, y), 1, func() string {
			return fmt.Sprintf("selection=%v", w.app.HeadlessSelection())
		})
	}
	camera := *w.f.live.view.Camera()
	targets := w.missionTargets(8)
	if after := w.f.live.view.Camera(); after.X != camera.X || after.Y != camera.Y || after.Zoom != camera.Zoom {
		w.t.Error("mission: hover discovery moved the camera")
	}
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
			if i == 0 {
				w.packHovers(hovered)
			}
		}
		if after[0] == hovered[r.key] || after[1] != hovered[r.key] {
			w.t.Errorf("mission: the pack corner reads %q, then %q and %q", hovered[r.key], after[0], after[1])
		}
	}
	w.wait("hero and party")
	w.key("0", 1)
	if d.spellbook {
		book, ok := routeFind(targets, fmt.Sprintf("corner/%d/%d/8", ui.CharacterPaneBook, hero))
		if !ok {
			w.t.Fatal("mission: no spellbook icon")
		}
		before, _, err := w.app.HeadlessPicture(ui.HeadlessRequest{Part: ui.HeadlessPartBottomHUD})
		if err != nil {
			w.t.Fatal(err)
		}
		for i := range 2 {
			w.click("mission spellbook icon", book.rect, book.at, w.app.TooltipDelay()/20+5, func() string {
				after, _, err := w.app.HeadlessPicture(ui.HeadlessRequest{Part: ui.HeadlessPartBottomHUD})
				if err != nil {
					w.t.Fatal(err)
				}
				opened := after.Bounds().Dy() > before.Bounds().Dy()
				if opened != (i == 0) {
					w.t.Errorf("mission: spellbook open=%v after click %d", opened, i+1)
				}
				return fmt.Sprintf("spellbook open=%v HUD=%v", opened, after.Bounds().Size())
			})
		}
	}
}

// routeCorner is the character pane corner c of unit id among regions.
func routeCorner(regions []routeRegion, c ui.CharacterPaneCorner, id any) (routeRegion, bool) {
	prefix := fmt.Sprintf("corner/%d/%d/", c, id)
	for _, r := range regions {
		if strings.HasPrefix(r.key, prefix) {
			return r, true
		}
	}
	return routeRegion{}, false
}

// pageRows is what the open menu page, save page or load page lists, and
// the save page's name and question.
func (w *routeWitness) pageRows() string {
	var rows []string
	for _, r := range w.app.HeadlessRows() {
		rows = append(rows, r.Text)
	}
	s := "rows=" + routeText(strings.Join(rows, "|"))
	if h, ok := w.app.HeadlessSaveState(); ok {
		s += fmt.Sprintf(" name=%s confirm=%v", routeText(h.Request.Label), h.Confirmation)
	}
	return s
}

// saveAction takes the save page's control action and records the step.
func (w *routeWitness) saveAction(action, control string) {
	before := w.screen()
	if err := w.app.HeadlessSaveAction(action); err != nil {
		w.t.Errorf("%s: %s: %v", w.stage, control, err)
	}
	w.run(2)
	w.record(control, image.Rectangle{}, "click", w.result(before, w.pageRows()))
}

// saveSelect selects the save page's row labelled label and records it.
func (w *routeWitness) saveSelect(label, control string) {
	before := w.screen()
	row := slices.IndexFunc(w.app.HeadlessRows(), func(r ui.PickerRow) bool { return r.Text == label })
	if row < 0 {
		w.t.Errorf("%s: the save page lists no %q: %s", w.stage, label, w.pageRows())
		return
	}
	if err := w.app.HeadlessSaveSelect(row); err != nil {
		w.t.Errorf("%s: %s: %v", w.stage, control, err)
	}
	w.run(2)
	w.record(control, image.Rectangle{}, "click", w.result(before, w.pageRows()))
}

// The original's save name and its points over the spellbook and the
// pack, in mission window pixels, and its signpost against its hero.
var (
	routeSaveName  = "exp75"
	routeBookHover = []struct {
		what, step string
		at         image.Point
	}{
		{"pack left gem area", "m3.02", image.Pt(60, 705)},
		{"pack left gem", "m3.03", image.Pt(15, 720)},
		{"pack right gem", "m3.04", image.Pt(832, 720)},
		{"spell cell 1", "m3.05", image.Pt(215, 616)},
		{"spellbook left ornament", "m3.06", image.Pt(95, 630)},
		{"spellbook right ornament", "m3.07", image.Pt(750, 630)},
	}
	routeSignpostFromHero = image.Pt(207-110, 325-362)
)

// missionMenu opens the game menu by the pane's menu corner and takes the
// original's rows: Save with a name, Ok, the list, a row, Delete with No
// and Yes, Cancel; Load; Game Options; Sound Options; Quest Objectives;
// End Quest and its return; then Esc twice.
func (w *routeWitness) missionMenu(routeData) {
	hero := w.f.live.mission.ids[0]
	corner, ok := routeCorner(w.missionTargets(8), ui.CharacterPaneMenu, hero)
	if !ok {
		w.t.Fatal("mission: no main menu corner")
	}
	open := func(step string) {
		if w.screen() == "gamemenu" {
			return
		}
		w.click("main menu ("+step+")", corner.rect, corner.at, 2, nil)
		w.expect("gamemenu", "the main menu corner")
	}
	open("m1.01")
	w.menuChoose("save", "save game (m1.02)", w.pageRows)
	w.expect("save", "save game")
	before := w.screen()
	if err := w.app.HeadlessSaveEdit("", routeSaveName, ui.SaveSAV); err != nil {
		w.t.Errorf("mission: save name: %v", err)
	}
	w.run(2)
	w.record("save name (m1.03)", image.Rectangle{}, "type "+routeSaveName, w.result(before, w.pageRows()))
	w.saveAction("save", "save ok (m1.05)")
	open("m1.06")
	w.menuChoose("save", "save game (m1.06)", w.pageRows)
	w.saveSelect(routeSaveName, "save row (m1.07)")
	w.saveAction("delete", "save delete (m1.08)")
	w.saveAction("cancel", "delete no (m1.09)")
	w.saveAction("save", "save ok on the row (m1.10)")
	open("m1.11")
	w.menuChoose("save", "save game (m1.11)", w.pageRows)
	w.saveSelect(routeSaveName, "save row (m1.11)")
	w.saveAction("delete", "save delete (m1.11)")
	w.saveAction("confirm", "delete yes (m1.11)")
	w.saveAction("cancel", "save cancel (m1.12)")
	open("m1.13")
	w.menuChoose("load", "load game (m1.13)", w.pageRows)
	w.key("escape", 2)
	open("m1.15")
	w.menuChoose("game-options", "game options (m1.15)", nil)
	w.menuChoose("options-cancel", "game options cancel (m1.16)", nil)
	open("m1.17")
	w.menuChoose("sound-options", "sound options (m1.17)", nil)
	w.key("escape", 2)
	open("m1.19")
	w.menuChoose("objectives", "quest objectives (m1.19)", nil)
	w.menuChoose("page-return", "quest objectives ok (m1.20)", nil)
	open("m1.21")
	w.menuChoose("end", "end quest (m1.21)", nil)
	w.menuChoose("page-return", "end quest return (m1.23)", nil)
	if w.screen() == "gamemenu" {
		w.menuChoose("return", "return to game", nil)
	}
	w.expect("map", "the menu's return")
	w.key("escape", 2)
	if w.screen() != "gamemenu" {
		w.t.Errorf("mission: Esc opens %s, want the game menu", w.screen())
	}
	w.key("escape", 2)
	w.expect("map", "the second Esc")
}

// bars names the bottom bars the mission shows: the pack and the band the
// spellbook and the pack take.
func (w *routeWitness) bars() string {
	hud, _, err := w.app.HeadlessPicture(ui.HeadlessRequest{Part: ui.HeadlessPartBottomHUD})
	if err != nil {
		return "bars: " + err.Error()
	}
	_, _, packErr := w.app.HeadlessPicture(ui.HeadlessRequest{Part: ui.HeadlessPartInventoryPack})
	return fmt.Sprintf("pack=%v band=%d", packErr == nil, hud.Bounds().Dy())
}

// missionPack opens the pack by its corner and hovers its controls, opens
// the spellbook over it by its corner, hovers the original's points, and
// toggles the spellbook by Q and B and the pack by I and the back quote,
// each twice.
func (w *routeWitness) missionPack(routeData) {
	hero := w.f.live.mission.ids[0]
	targets := w.missionTargets(8)
	pack, okPack := routeCorner(targets, ui.CharacterPaneBackpack, hero)
	book, okBook := routeCorner(targets, ui.CharacterPaneBook, hero)
	if !okPack || !okBook {
		w.t.Fatal("mission: no pack or spellbook corner")
	}
	settle := w.app.TooltipDelay()/20 + 5
	w.click("open inventory (m2.10)", pack.rect, pack.at, settle, w.bars)
	if _, _, err := w.app.HeadlessPicture(ui.HeadlessRequest{Part: ui.HeadlessPartInventoryPack}); err != nil {
		w.t.Errorf("mission: the pack corner opens no pack: %v", err)
	}
	w.packHovers(map[string]string{})
	w.click("open spellbook (m3.01)", book.rect, book.at, settle, w.bars)
	for _, h := range routeBookHover {
		w.hover(fmt.Sprintf("%s (%s)", h.what, h.step), image.Rectangle{}, h.at)
	}
	for _, k := range []struct{ key, step string }{
		{"q", "m3.09"}, {"q", "m3.10"}, {"b", "m3.11"}, {"b", "m3.12"},
		{"i", "m2.22"}, {"i", "m2.23"}, {"backquote", "m2.24"}, {"backquote", "m2.25"},
	} {
		before := w.bars()
		w.keyNote(k.key, 2, func() string { return "(" + k.step + ") " + w.bars() })
		if w.bars() == before {
			w.t.Errorf("mission: key %s (%s) leaves the bars at %s", k.key, k.step, before)
		}
	}
}

// signpost hovers and clicks the mission's signpost, waits as the original
// did and pages a notice it opens.
func (w *routeWitness) signpost(routeData) {
	p, on, err := w.signpostPoint()
	if err != nil {
		w.record("signpost", image.Rectangle{}, "wait", "no click: "+err.Error())
		return
	}
	cursor := func() string {
		c, _ := w.app.HeadlessMapCursor()
		return fmt.Sprintf("at=%d,%d on=%s cursor=%s", p.X, p.Y, on, c)
	}
	w.hoverNote("signpost (m4.04)", image.Rectangle{}, p, cursor)
	w.click("signpost (m4.05)", image.Rectangle{}, p, int(4*time.Second/(20*time.Millisecond)), cursor)
	w.waitFor("signpost wait (m4.06)", 10*time.Second)
	w.missionPages()
}

// signpostPoint is a shown pixel of the mission's usable Pointer; without
// one in view it is the original's signpost offset from the hero's first hit
// pixel, which lies above and left of the hero's figure.
func (w *routeWitness) signpostPoint() (image.Point, string, error) {
	if s, ok := routeSignpost(w.f.live.world.Structures()); ok {
		x, y, err := w.f.live.view.InspectionPoint(ui.InspectionSubject{Kind: ui.InspectionStructure, ID: uint32(s.ID)})
		if err == nil {
			return image.Pt(x, y), fmt.Sprintf("structure %d", s.ID), nil
		}
	}
	x, y, err := w.app.HeadlessEntityPoint(uint32(w.f.live.mission.ids[0]))
	if err != nil {
		return image.Point{}, "", fmt.Errorf("the hero answers no window pixel: %w", err)
	}
	return image.Pt(x, y).Add(routeSignpostFromHero), "hero offset", nil
}

// routeSignpost is the first second-game Pointer the hero can use.
func routeSignpost(all []sim.Structure) (sim.Structure, bool) {
	for _, s := range all {
		if sim.PointerKind(s.Kind) && s.Usable() {
			return s, true
		}
	}
	return sim.Structure{}, false
}

// missionExit leaves the mission by Esc, End Quest and Exit to Main Menu.
func (w *routeWitness) missionExit(routeData) {
	w.key("escape", 2)
	w.expect("gamemenu", "Esc")
	w.menuChoose("end", "end quest (m1.26)", nil)
	w.menuChoose("exit-main", "exit to main menu (m1.26)", nil)
	w.expect("menu", "exit to main menu")
}

// missionHero opens the hero's inventory and character pane by their keys
// and closes them.
func (w *routeWitness) missionHero(routeData) {
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
func (w *routeWitness) winToTown(routeData) {
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

// missionChoice clicks the first mission the world map offers, and again
// while the map stays on it, as a choice and its confirmation.
func (w *routeWitness) missionChoice() {
	v := w.town.WorldMapView()
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

// walk runs the stage table, writing route.tsv under the output directory
// when it ends, a failed walk's rows included. save, when set, is the town
// save the walk loaded and leads each row's step.
func (w *routeWitness) walk(stages []routeStage, save string) {
	w.save = save
	defer w.writeRoute()
	for _, s := range stages {
		w.stage = s.name
		// A stage opens on its screen; a dialogue or notice already open
		// over it is the stage's first page.
		if got := w.screen(); got != s.screen && !strings.HasPrefix(got, s.screen+"/") {
			w.t.Fatalf("%s: the stage opens on %s, want %s", s.name, got, s.screen)
		}
		s.walk(w)
	}
}

func (w *routeWitness) writeRoute() {
	if w.dir == "" {
		return
	}
	head := strings.Join([]string{"n", "step", "screen", "control", "rect", "action", "result", "text", "frame", "state"}, "\t")
	if err := os.WriteFile(filepath.Join(w.dir, "route.tsv"), []byte(head+"\n"+strings.Join(w.rows, "\n")+"\n"), 0o600); err != nil {
		w.t.Error(err)
	}
}

// newRouteWitnessOn opens f's App for the walk at the window size, with
// originals, when set, as the original saves' directory and out, when set,
// as the parent of the edition's frame directory, and answers the witness
// and its campaign service's stage table.
func newRouteWitnessOn(t routeReport, f *FrontEnd, originals string, size image.Point, out string) (*routeWitness, []routeStage) {
	if f.ChargenAssets == nil {
		t.Fatal("the install loaded no generator")
	}
	routeWitnessProfile(f, t.TempDir())
	app := f.App("route witness")
	width, height := app.Layout(size.X, size.Y)
	routeWitnessSeams(f, app, t.TempDir(), originals)
	// The witness holds the App's own town screen; the entry policy can
	// construct a fresh screen at each call.
	w := &routeWitness{t: t, f: f, app: app, steps: map[string]int{}, size: image.Pt(width, height)}
	stages := f.campaign().routeStages()
	town, ok := f.TownScreen().(*townScreen)
	if !ok {
		t.Fatal("the game's town is not the common town screen")
	}
	app.SetTown(town)
	w.town = town
	app.SetNewGameChargen(func() *ui.ChargenEntry {
		w.chargen = ui.NewChargen(f.ChargenSetup())
		return &ui.ChargenEntry{Model: w.chargen, Begin: f.NewGameBegin(f.NewGameMission())}
	})
	if out != "" {
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
	return w, stages
}

// WriteRouteWitness walks the install at root as TestReleaseRouteWitness
// does, on the CPU alone: a step over the running mission records the map
// view, a pane, the pack, a tooltip or the menu panel the App composes
// headlessly, never the GPU's whole window. It writes route.tsv and one PNG
// per step through png under out/<edition>/ at the window size, logs
// through logf, and answers the stop that ended the walk or the count of
// failed steps.
func WriteRouteWitness(root, out string, window image.Point, logf func(format string, args ...any), png func(path string, pix *image.RGBA) error) error {
	f, err := NewFrontEnd(root)
	if err != nil {
		return err
	}
	witnessSeeds(f)
	f.SetDeterministicFrames(true)
	tmp, err := os.MkdirTemp("", "route-witness-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmp)
	r := &routeLog{logf: logf, tmp: tmp}
	err = r.guard(func() {
		w, stages := newRouteWitnessOn(r, f, "", window, out)
		w.emit = func(dir, name string, pix *image.RGBA) {
			if err := png(filepath.Join(dir, name+".png"), pix); err != nil {
				r.Error(err)
			}
		}
		w.walk(stages, "")
		logf("%s: %d steps, %d frames", f.Base().Profile.ID, len(w.rows), w.frames)
	})
	if err == nil && r.errs > 0 {
		err = fmt.Errorf("route witness: %d failed steps", r.errs)
	}
	return err
}

// secondButtonPoint is a frame point the mask gives to the button bound to
// action.
func secondButtonPoint(t routeReport, f *FrontEnd, action menu.Action) image.Point {
	t.Helper()
	for y := 0; y < menu.FrameH; y++ {
		for x := 0; x < menu.FrameW; x++ {
			if b := f.Assets.ButtonAt(image.Pt(x, y)); b != 0 && f.Assets.Binding(b).Action == action {
				return image.Pt(x, y)
			}
		}
	}
	t.Fatalf("no mask point for action %d", action)
	return image.Point{}
}

// routeWitnessProfile gives the walk a fresh profile under dir: the options
// file records the seen cutscenes the Cut Scenes library lists.
func routeWitnessProfile(f *FrontEnd, dir string) {
	f.Options = OptionsStore{Path: filepath.Join(dir, "options.txt")}
	f.Markers = Markers{}
}

// routeWitnessSeams points the App's town animation at its own scene clock
// and its saves at a scratch store under dir, with originals as the original
// saves' directory and a fixed save time.
func routeWitnessSeams(f *FrontEnd, app *ui.App, dir, originals string) {
	f.TownAnimationNow = app.HeadlessSceneNow
	f.ConfigureSaveSeams(app, SaveStore{Dir: dir}, OriginalStore{Dir: originals}, func() time.Time { return time.Unix(1700000000, 0).UTC() })
}

// witnessSeeds fixes the presentation streams a witness's town frames read,
// so the wildlife and the birds stand the same from run to run.
func witnessSeeds(f *FrontEnd) {
	f.randomService().SetStreamSeed(random.TownWildlife, 1)
	f.randomService().SetStreamSeed(random.AmbientBirds, 1)
}

// routeReport is what the walk reports to: a test's *testing.T, or the
// command's routeLog.
type routeReport interface {
	Helper()
	Error(args ...any)
	Errorf(format string, args ...any)
	Fatal(args ...any)
	Fatalf(format string, args ...any)
	Logf(format string, args ...any)
	TempDir() string
}

// routeLog reports a command's walk: an error is logged and counted, and a
// fatal one is logged and stops the walk through guard.
type routeLog struct {
	logf func(format string, args ...any)
	tmp  string
	errs int
	dirs int
}

type routeStop struct{ msg string }

func (r *routeLog) Helper() {}

func (r *routeLog) Logf(format string, args ...any) { r.logf(format, args...) }

func (r *routeLog) Error(args ...any) { r.Errorf("%s", fmt.Sprint(args...)) }

func (r *routeLog) Errorf(format string, args ...any) {
	r.errs++
	r.logf("error: "+format, args...)
}

func (r *routeLog) Fatal(args ...any) { r.Fatalf("%s", fmt.Sprint(args...)) }

func (r *routeLog) Fatalf(format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	r.logf("stop: %s", msg)
	panic(routeStop{msg})
}

// TempDir answers a fresh directory under the command's temporary root.
func (r *routeLog) TempDir() string {
	r.dirs++
	dir := filepath.Join(r.tmp, strconv.Itoa(r.dirs))
	if err := os.MkdirAll(dir, 0o700); err != nil {
		r.Fatal(err)
	}
	return dir
}

// guard runs walk and answers the stop a Fatal raised in it, if any.
func (r *routeLog) guard(walk func()) (err error) {
	defer func() {
		if v := recover(); v != nil {
			stop, ok := v.(routeStop)
			if !ok {
				panic(v)
			}
			err = fmt.Errorf("route witness: %s", stop.msg)
		}
	}()
	walk()
	return nil
}

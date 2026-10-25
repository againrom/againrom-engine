package game

import (
	"fmt"
	"image"
	"image/draw"
	"time"

	"againrom/pkg/audio"
	"againrom/pkg/render/text"
	"againrom/pkg/screenart"
	"againrom/pkg/town"
	"againrom/pkg/ui"
)

// secondTownPages are the page kinds the second game's town screen draws.
var secondTownPages = []string{"inn", "shop"}

// secondTownRooms binds the description's room names to the campaign's
// rooms, the room the SAV records.
var secondTownRooms = map[string]secondTownRoom{"square": secondTownSquare, "inn": secondTownInn, "shop": secondTownShop}

// secondTownRowText is the list fallback's text for a square hotspot's row.
var secondTownRowText = map[string]string{"inn": "TAVERN", "gate": "GATES",
	"girl1": "GIRL 1", "girl2": "GIRL 2", "guard": "GUARD", "dervish": "DERVISH",
	"innkeeper": "INNKEEPER", "shopkeeper": "SHOPKEEPER", "shop": "SHOP"}

// secondTownConditions answers the conditions the second game's town
// descriptions name.
var secondTownConditions = map[string]func(c *secondCampaign) bool{
	"gate-open": (*secondCampaign).gateOpen,
	"shelf-0":   func(c *secondCampaign) bool { return c.shop.category == 0 },
	"shelf-1":   func(c *secondCampaign) bool { return c.shop.category == 1 },
	"shelf-2":   func(c *secondCampaign) bool { return c.shop.category == 2 },
	"shelf-3":   func(c *secondCampaign) bool { return c.shop.category == 3 },
	// bank-770-set is the druid shop's fourth-category gate, Scenario query
	// 0x302 (R2-ENGINE-259).
	"bank-770-set": func(c *secondCampaign) bool { return c.bank[770] != 0 },
}

// secondTownValues answers the values the second game's town descriptions
// name: the stage is Scenario variable 0x300, bank slot 768 (R2-ENGINE-234).
var secondTownValues = map[string]func(c *secondCampaign) int{
	"stage": func(c *secondCampaign) int { return int(c.bank[768]) },
	"shelf": func(c *secondCampaign) int { return c.shop.category },
}

// secondTownHooks are the second game's campaign hooks, each run for the
// room its step belongs to.
var secondTownHooks = map[string]func(t *secondCampaignScreen, room string){
	"set-room": func(t *secondCampaignScreen, room string) {
		if r, ok := secondTownRooms[room]; ok {
			t.state().room = r
		}
	},
	"gate-closed": func(t *secondCampaignScreen, _ string) { t.composed.action = t.openTownText(secondGateText) },
	"navigate": func(t *secondCampaignScreen, _ string) {
		c := t.state()
		c.leaveTown()
		c.room = secondTownSquare
	},
	"shop-open":  secondShopOpen,
	"tip":        func(t *secondCampaignScreen, room string) { t.loadRoomTip(room) },
	"shop-leave": func(t *secondCampaignScreen, _ string) { c := t.state(); c.shop.leave(c.heroBag()) },
}

// secondGateText is the town text section the closed gate shows
// (R2-ENGINE-241).
const secondGateText = "plagatguard"

// secondInnExit is the list fallback's row that leaves the composed inn.
const secondInnExit = "EXIT"

// secondTownVocabulary is the hook, condition, value and page names the
// second game's campaign answers.
func secondTownVocabulary() town.Vocabulary {
	v := town.Vocabulary{Pages: secondTownPages, Events: secondShopEvents}
	for name := range secondTownHooks {
		v.Hooks = append(v.Hooks, name)
	}
	for name := range secondTownConditions {
		v.Conditions = append(v.Conditions, name)
	}
	for name := range secondTownValues {
		v.Values = append(v.Values, name)
	}
	return v
}

// secondTownState is the composed town's presentation: its scenes and art,
// its room audio, the inn's seat selection and the action a hook left. It
// lives with the screen; the campaign state holds the room.
type secondTownState struct {
	process *town.Process
	sound   townAudio
	draws   townDraws

	scenes map[string]*town.Scene
	art    map[string]*secondTownArt

	roomAudio *ui.AudioScope
	audioRoom string
	loop      audio.Voice

	seat       int
	seatFrames []int
	// patrons are the seated people the pages have shown, built on first
	// show.
	patrons   map[secondInnPatronKey]*secondInnPatron
	seatStamp time.Time
	innShown  bool
	// pointer is the last pointer update the shown room had; (-1,-1) off
	// the screen.
	pointer image.Point

	action ui.TownAction
}

// secondTownArt is one room's loaded art: its scene's and its page's.
type secondTownArt struct {
	scene *town.Art
	page  *screenart.Set
	err   error
	// shop is the shop page's pictures from page, made on first use.
	shop *ui.ShopScreenArt
}

func newSecondTownState(f *FrontEnd) *secondTownState {
	return &secondTownState{process: &f.townProcess, sound: runtimeTownAudio{&f.RuntimeServices},
		draws: runtimeTownDraws{&f.RuntimeServices}, scenes: map[string]*town.Scene{},
		art: map[string]*secondTownArt{}, seat: -1, pointer: image.Pt(-1, -1)}
}

// composedTown answers the description of the current town when the
// install profile's catalog composes it under the second game's hook set. A
// town the catalog does not list keeps the row list.
func (t *secondCampaignScreen) composedTown() (*town.Description, bool) {
	c := t.state()
	if c == nil || c.current.kind != placeTown || t.composed == nil {
		return nil, false
	}
	d := townDescriptionOf(TownCatalog(t.install.profile()), c.current.id, townHooksDestinations)
	return d, d != nil
}

// roomName is the description's name of the campaign's current room.
func (t *secondCampaignScreen) roomName() string {
	for name, r := range secondTownRooms {
		if r == t.state().room {
			return name
		}
	}
	return ""
}

func (t *secondCampaignScreen) scene(d *town.Description, room string) *town.Scene {
	key := d.Town + "/" + room
	if s := t.composed.scenes[key]; s != nil {
		return s
	}
	s := town.NewScene(d, room, townSceneHost{t: t, room: room}, t.composed.process)
	t.composed.scenes[key] = s
	return s
}

// roomArt loads a room's art once for the screen through the install's art
// library: the scene's, and for a room the page's own.
func (t *secondCampaignScreen) roomArt(d *town.Description, room string) *secondTownArt {
	key := d.Town + "/" + room
	if a := t.composed.art[key]; a != nil {
		return a
	}
	a := &secondTownArt{}
	t.composed.art[key] = a
	src := t.install.Archives.artSource()
	if src == nil {
		a.err = fmt.Errorf("town %s: no install art", d.Town)
		return a
	}
	spec := d.Scene(room)
	if spec == nil {
		a.err = fmt.Errorf("town %s: room %s has no scene", d.Town, room)
		return a
	}
	a.scene, a.err = town.LoadArt(spec, screenArtOf(src))
	if a.err != nil {
		return a
	}
	if r := d.Room(room); r != nil {
		a.page, a.err = screenArtOf(src).Load(room, r.Art)
		if a.err != nil {
			a.scene = nil
		}
	}
	return a
}

// composedRoom answers the composed description and room shown now, and
// whether that room's art is loaded.
func (t *secondCampaignScreen) composedRoom() (*town.Description, string, bool) {
	d, ok := t.composedTown()
	if !ok {
		return nil, "", false
	}
	room := t.roomName()
	return d, room, t.roomArt(d, room).scene != nil
}

// AtTownSquare reports whether the composed town is shown: its square or its
// inn page, with that room's art loaded. The shop room is the shop page's.
func (t *secondCampaignScreen) AtTownSquare() bool {
	_, _, ok := t.composedRoom()
	return ok && !t.AtTownShop()
}

// TownSquareView answers the composed room the ui draws and hit-tests.
func (t *secondCampaignScreen) TownSquareView() ui.TownSquareView {
	d, room, ok := t.composedRoom()
	if !ok {
		return ui.TownSquareView{}
	}
	return ui.TownSquareView{Scene: secondTownView{t: t, d: d, room: room}, Font: t.install.Font.Value(),
		Tip: t.roomTipView(d.Town, room)}
}

// TownSquareActive runs the shown room's scene and pauses and hides every
// other; leaving the composed town hides them all. The shop page runs its
// room through ShopInteriorActive.
func (t *secondCampaignScreen) TownSquareActive(active bool) {
	if t.composed == nil || t.state() == nil || t.AtTownShop() {
		return
	}
	t.roomActive(active)
}

// roomActive runs the shown room's scene and pauses and hides every other.
func (t *secondCampaignScreen) roomActive(active bool) {
	d, room, shown := t.composedRoom()
	t.arriveRoomTip(shown, room)
	for key, s := range t.composed.scenes {
		show := shown && key == d.Town+"/"+room
		if show {
			continue
		}
		s.SetActive(false, false)
	}
	if !shown {
		t.composed.innShown = false
		return
	}
	s := t.scene(d, room)
	if room != d.Square.Name && !s.Present() {
		// A room page is built afresh at each entry (R2-ENGINE-248).
		s.Reset()
	}
	s.SetActive(active, true)
	if r := d.Room(room); r != nil && r.Seats != nil {
		if s.Present() && !t.composed.innShown {
			t.enterSeats()
		}
		t.composed.innShown = s.Present()
	} else {
		t.composed.innShown = false
	}
}

// TownSquarePointer delivers one pointer update to the shown room's scene.
func (t *secondCampaignScreen) TownSquarePointer(p image.Point) {
	d, room, ok := t.composedRoom()
	if !ok || t.AtTownShop() {
		return
	}
	t.TownSquareActive(true)
	t.composed.pointer = p
	t.scene(d, room).Pointer(p)
}

// AdvanceTownSquareAnimation is one paint of the shown room: its scene's
// steps, then the selected seat's frame.
func (t *secondCampaignScreen) AdvanceTownSquareAnimation() {
	if !t.AtTownShop() {
		t.advanceRoom()
	}
}

// advanceRoom is one paint of the shown room's scene and its seats.
func (t *secondCampaignScreen) advanceRoom() {
	d, room, ok := t.composedRoom()
	if !ok {
		return
	}
	s := t.scene(d, room)
	s.Advance()
	if r := d.Room(room); r != nil && r.Seats != nil && s.Active() {
		var frames []image.Image
		if list, _ := t.state().innSeats(); t.composed.seat >= 0 && t.composed.seat < len(list) {
			frames = t.innPatron(r, t.roomArt(d, room).page, list[t.composed.seat].npc).frames
		}
		t.advanceSeats(r.Seats, frames)
	}
}

// SelectTownSquare selects the seat of inn row i and requests the seat
// grid's selection sound.
func (t *secondCampaignScreen) SelectTownSquare(i int) {
	d, room, ok := t.composedRoom()
	if !ok {
		return
	}
	r := d.Room(room)
	if seats, _ := t.state().innSeats(); r == nil || r.Seats == nil || i < 0 || i >= len(seats) {
		return
	}
	t.selectSeat(t.scene(d, room), r.Seats, i)
}

func (t *secondCampaignScreen) selectSeat(s *town.Scene, seats *town.SeatSpec, i int) {
	t.composed.seat = i
	if seats.Select != "" {
		s.Sound(seats.Select)
	}
}

// enterSeats is the seat state a page entry sets: the first seat selected
// when the roster is not empty, every frame at zero (R2-ENGINE-247,
// R2-ENGINE-248).
func (t *secondCampaignScreen) enterSeats() {
	c := t.state()
	c.enterInn()
	t.composed.seat = -1
	if seats, _ := c.innSeats(); len(seats) > 0 {
		t.composed.seat = 0
	}
	t.composed.seatFrames = nil
	t.composed.seatStamp = t.sceneNow()
}

// advanceSeats advances the selected seat's actor one frame once its period
// has passed; every other seat keeps its frame (R2-ENGINE-248).
func (t *secondCampaignScreen) advanceSeats(seats *town.SeatSpec, frames []image.Image) {
	st := t.composed
	if st.seat < 0 || len(frames) == 0 {
		return
	}
	now := t.sceneNow()
	since, period := now.Sub(st.seatStamp), time.Duration(seats.FrameMS)*time.Millisecond
	if since < period || since == period && seats.Compare == "greater" {
		return
	}
	st.seatStamp = now
	for len(st.seatFrames) <= st.seat {
		st.seatFrames = append(st.seatFrames, 0)
	}
	st.seatFrames[st.seat] = (st.seatFrames[st.seat] + 1) % len(frames)
}

// seatFrame is the frame a seat's actor shows.
func (t *secondCampaignScreen) seatFrame(i int) int {
	if i < len(t.composed.seatFrames) {
		return t.composed.seatFrames[i]
	}
	return 0
}

// composedRows are the composed town's list rows: on the square one per
// hotspot with a row, in row order; in a page with seats one row per seat,
// then EXIT. The shop room has none: the shop page takes its input.
func (t *secondCampaignScreen) composedRows(d *town.Description, c *secondCampaign) []ui.TownRow {
	if r := d.Room(t.roomName()); r != nil && r.Page == "shop" {
		return nil
	} else if r != nil {
		return append(c.innRows(), ui.TownRow{Text: secondInnExit, Choosable: true})
	}
	s := t.scene(d, d.Square.Name)
	var rows []ui.TownRow
	for i := 0; ; i++ {
		h := s.HotspotForRow(i)
		if h == nil {
			return rows
		}
		rows = append(rows, ui.TownRow{Text: secondTownRowText[h.Name], Choosable: true})
	}
}

// chooseComposed acts on row i of the composed town: a square row runs its
// hotspot's click program; an inn row selects its seat and runs its action,
// or leaves.
func (t *secondCampaignScreen) chooseComposed(d *town.Description, c *secondCampaign, i int) ui.TownAction {
	square := t.scene(d, d.Square.Name)
	room := t.roomName()
	if r := d.Room(room); r != nil && r.Page == "shop" {
		return ui.TownAction{}
	} else if r != nil {
		if seats, _ := c.innSeats(); i < len(seats) {
			if r.Seats != nil {
				t.selectSeat(t.scene(d, room), r.Seats, i)
			}
			return t.innAct(c, i)
		}
		square.LeaveRoom(room)
		return ui.TownAction{}
	}
	h := square.HotspotForRow(i)
	if h == nil {
		return ui.TownAction{}
	}
	t.composed.action = ui.TownAction{}
	square.Click(h)
	action := t.composed.action
	t.composed.action = ui.TownAction{}
	return action
}

// backComposed leaves a composed room for the square, as Exit and Escape do
// (R2-ENGINE-250).
func (t *secondCampaignScreen) backComposed(d *town.Description) bool {
	room := t.roomName()
	if d.Room(room) == nil {
		return false
	}
	t.scene(d, d.Square.Name).LeaveRoom(room)
	return true
}

// openTownText opens a town text section as the town dialogue; a missing
// section opens nothing.
func (t *secondCampaignScreen) openTownText(key string) ui.TownAction {
	payload, err := readSecondTownTalk(t.install, key)
	if err != nil {
		return ui.TownAction{Msg: err.Error()}
	}
	if payload != nil {
		t.setTownDialogue(payload)
	}
	return ui.TownAction{}
}

// secondTownView is a composed room as ui paints and hit-tests it.
type secondTownView struct {
	t    *secondCampaignScreen
	d    *town.Description
	room string
}

func (v secondTownView) Size() image.Point { return v.t.scene(v.d, v.room).Size() }

// Paint draws the room: the square's layers, or a page's back layers, its
// seated speakers, its front layers and its buttons.
func (v secondTownView) Paint(dst *image.RGBA) {
	s := v.t.scene(v.d, v.room)
	r := v.d.Room(v.room)
	if r == nil {
		s.Paint(dst, "")
		return
	}
	page := v.t.roomArt(v.d, v.room).page
	s.Paint(dst, "back")
	words, font := v.t.install.Words, v.t.install.Font.Value()
	if r.Seats != nil {
		v.paintSeats(dst, r, page, words, font)
		v.paintPatron(dst, r, page)
	}
	s.Paint(dst, "front")
	var buttons []ui.RoomButton
	for _, b := range s.ShownButtons(r.Buttons) {
		rect := b.Rect.Rectangle()
		rb := ui.RoomButton{Rect: rect, Off: page.Picture(b.Off), Hover: v.t.composed.pointer.In(rect)}
		if b.On != "" {
			rb.On = page.Picture(b.On)
		}
		caption := b.Caption
		if b.Name == "hire" {
			caption = v.hireCaption()
		}
		if caption != nil && *caption < len(words.Hover) {
			rb.Caption = words.Hover[*caption]
		}
		buttons = append(buttons, rb)
	}
	ui.DrawRoomButtons(dst, font, v.t.install.roomButtonStyle(v.d), buttons)
}

// roomButtonStyle is the description's room button style in the generator
// font it names; nil when the description names none.
func (in *InstallResources) roomButtonStyle(d *town.Description) *ui.RoomButtonStyle {
	if d == nil || d.RoomButtons == nil {
		return nil
	}
	b := d.RoomButtons
	s := &ui.RoomButtonStyle{Ink: ui.GeneratorInk(b.Ink).RGBA(), Hover: ui.GeneratorInk(b.HoverInk).RGBA()}
	if a := in.ChargenAssets; a != nil && a.Presentation != nil {
		s.Font = a.Presentation.Font
		if b.Font == "name" {
			s.Font = a.Presentation.NameFont
		}
	}
	return s
}

// Main-text keys of the inn page: the hired mark (R2-ENGINE-497) and the
// hire button's Hire and Fire captions (R2-ENGINE-246).
const (
	secondInnHiredMark   = 257
	secondInnHireCaption = 258
	secondInnFireCaption = 259
)

// paintSeats draws the seats: each hire actor, then each talk actor, its
// back and then its person's frames; a hire actor whose copied word has the high bit
// also shows the hired mark over its seat. No hire actor is drawn while no
// seat is selected (R2-ENGINE-497, DIV-2964).
func (v secondTownView) paintSeats(dst *image.RGBA, r *town.RoomSpec, page *screenart.Set, words ui.Words, font *text.Font) {
	seats := r.Seats
	c := v.t.state()
	list, hires := c.innSeats()
	for i, o := range list {
		at := seats.Seat(i)
		if at.Empty() {
			break
		}
		back := seats.Back
		if i < hires {
			if v.t.composed.seat < 0 {
				continue
			}
			if seats.HireBack != "" {
				back = seats.HireBack
			}
		}
		drawAt(dst, page.Picture(back), at.Min, draw.Src)
		if actor := v.t.innPatron(r, page, o.npc).frames; len(actor) > 0 {
			drawAt(dst, actor[v.t.seatFrame(i)%len(actor)], at.Min, draw.Over)
		}
		if i < hires && c.hired(o.npc) && secondInnHiredMark < len(words.Hover) && font != nil {
			ui.DrawTownCaption(dst, font, words.Hover[secondInnHiredMark], at)
		}
	}
}

// hireCaption is the hire button's caption: Fire on a selected hire actor
// whose copied word has the high bit, Hire on any other selected hire
// actor, none on a talk actor or without a selection (R2-ENGINE-246).
func (v secondTownView) hireCaption() *int {
	c := v.t.state()
	list, hires := c.innSeats()
	seat := v.t.composed.seat
	if seat < 0 || seat >= hires {
		return nil
	}
	key := secondInnHireCaption
	if c.hired(list[seat].npc) {
		key = secondInnFireCaption
	}
	return &key
}

// drawAt draws pic with its top-left corner at at.
func drawAt(dst *image.RGBA, pic image.Image, at image.Point, op draw.Op) {
	if pic == nil {
		return
	}
	b := pic.Bounds()
	draw.Draw(dst, b.Add(at.Sub(b.Min)), pic, b.Min, op)
}

// ControlAt answers what a click at p reaches: on the square a hotspot's row
// or the menu; on a page a seat to select, or the row its talk or exit button
// acts on.
func (v secondTownView) ControlAt(p image.Point) (ui.TownSquareControl, bool) {
	r := v.d.Room(v.room)
	if r == nil {
		h := v.t.scene(v.d, v.room).HotspotAt(p)
		switch {
		case h == nil:
			return ui.TownSquareControl{}, false
		case h.Row != nil:
			return ui.TownSquareControl{Kind: ui.TownSquareControlDoor, Door: *h.Row}, true
		case town.Menu(h.Click) != "":
			return ui.TownSquareControl{Kind: ui.TownSquareControlMenu}, true
		}
		return ui.TownSquareControl{}, false
	}
	if r.Page == "shop" {
		return ui.TownSquareControl{}, false
	}
	list, hires := v.t.state().innSeats()
	seats := len(list)
	if r.Seats != nil {
		if i := r.Seats.SeatAt(p, seats); i >= 0 {
			// A seat click selects the seat and acts on nothing; the
			// buttons act on the selection (R2-SESSION-203, R2-SESSION-213).
			return ui.TownSquareControl{Kind: ui.TownSquareControlSelect, Door: i}, true
		}
	}
	if i := town.ButtonAt(r.Buttons, p); i >= 0 {
		seat := v.t.composed.seat
		switch r.Buttons[i].Name {
		case "hire":
			if seat >= 0 && seat < hires {
				return ui.TownSquareControl{Kind: ui.TownSquareControlDoor, Door: seat}, true
			}
		case "talk":
			if seat >= hires && seat < seats {
				return ui.TownSquareControl{Kind: ui.TownSquareControlDoor, Door: seat}, true
			}
		case "exit":
			return ui.TownSquareControl{Kind: ui.TownSquareControlDoor, Door: seats}, true
		}
	}
	return ui.TownSquareControl{}, false
}

// TipAt answers the square hotspot's tip slot; a page has no hover tips.
func (v secondTownView) TipAt(p image.Point) (int, bool) {
	if v.d.Room(v.room) != nil {
		return 0, false
	}
	h := v.t.scene(v.d, v.room).HotspotAt(p)
	if h == nil {
		return 0, false
	}
	return h.TipSlot()
}

// The second game's campaign screen answers the one scene adapter
// (townSceneHost) for the rooms of its composed towns.

func (t *secondCampaignScreen) sceneArt(room string) *town.Art {
	d, ok := t.composedTown()
	if !ok {
		return nil
	}
	return t.roomArt(d, room).scene
}

func (t *secondCampaignScreen) sceneNow() time.Time {
	if now := t.composed.draws.animationClock(); now != nil {
		return now()
	}
	return time.Now()
}

func (t *secondCampaignScreen) sceneDraw(source string, n int) int {
	return sceneDrawFrom(t.composed.draws, source, n)
}

func (t *secondCampaignScreen) sceneSeed() int64 { return t.composed.sound.wildlifeSeed() }

func (t *secondCampaignScreen) sceneCondition(name string) bool {
	if c := secondTownConditions[name]; c != nil && t.state() != nil {
		return c(t.state())
	}
	return false
}

func (t *secondCampaignScreen) sceneValue(name string) int {
	if v := secondTownValues[name]; v != nil && t.state() != nil {
		return v(t.state())
	}
	return -1
}

// sceneDialogue opens a town text section in place, as a click on a person
// of the square does (R2-ENGINE-234).
func (t *secondCampaignScreen) sceneDialogue(key string) { t.composed.action = t.openTownText(key) }

// roomPlayer is the room's audio scope, built for the room on first use and
// replaced when another room plays.
func (t *secondCampaignScreen) roomPlayer(room string) audio.Player {
	st := t.composed
	device := st.sound.soundDevice()
	if device == nil {
		return nil
	}
	if st.roomAudio != nil && (st.audioRoom != room || st.roomAudio.Owner() != ui.DeliveryOwner(device)) {
		st.roomAudio.Destroy()
		st.roomAudio = nil
	}
	if st.roomAudio == nil {
		st.roomAudio, st.audioRoom = ui.NewAudioScope(device), room
	}
	if st.roomAudio != nil {
		return st.roomAudio.Player(audio.EffectsChannel)
	}
	return device
}

func (t *secondCampaignScreen) scenePlay(room string, r town.SoundRequest) town.Voice {
	source, key, loop := r.Source, r.Key, r.Loop
	sample, ok := t.install.SoundBank.namedSample(key)
	if !ok {
		return nil
	}
	player := t.roomPlayer(room)
	if player == nil {
		return nil
	}
	v := audio.Dispatch(player, sample, audio.FixedRequest(source, key, audio.EffectsChannel, townSoundPriority(r), loop,
		audio.Placement{Left: audio.GainUnit, Right: audio.GainUnit}))
	if v == nil {
		return nil
	}
	return v
}

func (t *secondCampaignScreen) sceneStartLoop(key string) bool {
	st := t.composed
	ambient := st.sound.ambientDevice()
	if ambient == nil {
		return false
	}
	sample, ok := t.install.SoundBank.namedSample(key)
	if !ok {
		return false
	}
	request := audio.FixedRequest("town-crowd", key, audio.EffectsChannel, 128, true,
		audio.Placement{Left: audio.GainUnit, Right: audio.GainUnit})
	if ui.DeliveryOwner(st.sound.soundDevice()) != nil {
		st.loop = audio.Dispatch(t.roomPlayer(t.roomName()), sample, request)
	} else {
		st.loop = ui.RequestAmbient(ambient, ui.AmbientTownCrowd, sample, request)
	}
	return true
}

func (t *secondCampaignScreen) sceneStopLoop() {
	st := t.composed
	if ambient := st.sound.ambientDevice(); ambient != nil {
		if ui.DeliveryOwner(st.sound.soundDevice()) != nil {
			audio.StopReset(st.loop)
		} else {
			ambient.StopLoop(ui.AmbientTownCrowd)
		}
	}
	st.loop = nil
}

func (t *secondCampaignScreen) sceneLeave(room string) {
	if st := t.composed; st.roomAudio != nil && st.audioRoom == room {
		st.roomAudio.Destroy()
		st.roomAudio, st.audioRoom = nil, ""
	}
}

func (t *secondCampaignScreen) sceneHook(name, room string) {
	if hook := secondTownHooks[name]; hook != nil {
		hook(t, room)
	}
}

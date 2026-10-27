package game

import (
	"fmt"
	"image"
	"image/draw"
	"time"

	"againrom/pkg/render/text"
	"againrom/pkg/screenart"
	"againrom/pkg/town"
	"againrom/pkg/ui"
)

// secondTownRooms binds the description's room names to the campaign's
// rooms, the room the SAV records.
var secondTownRooms = map[string]secondTownRoom{"square": secondTownSquare, "inn": secondTownInn, "shop": secondTownShop}

// townRowText is the list fallback's text for a square hotspot's row.
var townRowText = map[string]string{"inn": "TAVERN", "gate": "GATES",
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

// secondGateText is the town text section the closed gate shows
// (R2-ENGINE-241).
const secondGateText = "plagatguard"

// secondInnExit is the list fallback's row that leaves the composed inn.
const secondInnExit = "EXIT"

// townSceneState is the composed town's presentation: its scenes and art,
// its room audio, the inn's seat selection and the action a hook left. It
// lives with the screen; the campaign state holds the room.
type townSceneState struct {
	scenes map[string]*town.Scene
	art    map[string]*townRoomArt

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

	// heroStats is the room panes' mode, the statistics card when set;
	// hero is the shown member's face, composed again after a room entry.
	heroStats bool
	hero      *secondHeroFace
}

// townRoomArt is one room's loaded art: its scene's and its page's.
type townRoomArt struct {
	scene *town.Art
	page  *screenart.Set
	err   error
	// shop is the shop page's pictures from page, made on first use.
	shop *ui.ShopScreenArt
}

func newTownSceneState(_ *FrontEnd) *townSceneState {
	return &townSceneState{scenes: map[string]*town.Scene{}, art: map[string]*townRoomArt{}, seat: -1, pointer: image.Pt(-1, -1)}
}

// composedTown answers the description of the current town when the
// install profile's catalog composes it under the second game's hook set. A
// town the catalog does not list keeps the row list.
func (t *townScreen) composedTown() (*town.Description, bool) {
	if t.composed == nil && t.hostRules().Pages == "entry" || !t.campaignTown().present() {
		return nil, false
	}
	d := t.townDescription()
	return d, d != nil
}

// roomName is the description's name of the campaign's current room.

func (t *townScreen) scene(d *town.Description, room string) *town.Scene {
	if t.composed == nil {
		t.composed = newTownSceneState(nil)
	}
	if t.hostRules().Pages == "retained" {
		if room == d.Square.Name && t.square != nil {
			return t.square
		}
		if room != d.Square.Name && t.pages[room] != nil {
			return t.pages[room]
		}
	}
	key := d.Town + "/" + room
	if s := t.composed.scenes[key]; s != nil && t.hostRules().Pages != "retained" {
		return s
	}
	s := town.NewScene(d, room, townSceneHost{t: t, room: room}, t.townProcess)
	t.composed.scenes[key] = s
	if room == d.Square.Name {
		t.square = s
	} else {
		if t.pages == nil {
			t.pages = map[string]*town.Scene{}
		}
		t.pages[room] = s
	}
	return s
}

// roomArt loads a room's art once for the screen through the install's art
// library: the scene's, and for a room the page's own.
func (t *townScreen) roomArt(d *town.Description, room string) *townRoomArt {
	key := d.Town + "/" + room
	if a := t.composed.art[key]; a != nil {
		return a
	}
	a := &townRoomArt{}
	t.composed.art[key] = a
	src := t.in.Archives.artSource()
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
func (t *townScreen) composedRoom() (*town.Description, string, bool) {
	d, ok := t.composedTown()
	if !ok {
		return nil, "", false
	}
	room := t.displayRoom()
	return d, room, t.sceneArt(room) != nil
}

// roomActive runs the shown room's scene and pauses and hides every other.
func (t *townScreen) roomActive(active bool) {
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

// advanceRoom is one paint of the shown room's scene and its seats.
func (t *townScreen) advanceRoom() {
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
func (t *townScreen) SelectTownSquare(i int) {
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

func (t *townScreen) selectSeat(s *town.Scene, seats *town.SeatSpec, i int) {
	t.composed.seat = i
	if seats.Select != "" {
		s.Sound(seats.Select)
	}
}

// enterSeats is the seat state a page entry sets: the first seat selected
// when the roster is not empty, every frame at zero (R2-ENGINE-247,
// R2-ENGINE-248).
func (t *townScreen) enterSeats() {
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
func (t *townScreen) advanceSeats(seats *town.SeatSpec, frames []image.Image) {
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
func (t *townScreen) seatFrame(i int) int {
	if i < len(t.composed.seatFrames) {
		return t.composed.seatFrames[i]
	}
	return 0
}

// openTownText opens a town text section as the town dialogue; a missing
// section opens nothing.
func (t *townScreen) openTownText(key string) ui.TownAction {
	payload, err := readSecondTownTalk(t.in, key)
	if err != nil {
		return ui.TownAction{Msg: err.Error()}
	}
	if payload != nil {
		t.setTownDialogue(payload)
	}
	return ui.TownAction{}
}

// townRoomView is a composed room as ui paints and hit-tests it.
type townRoomView struct {
	t    *townScreen
	d    *town.Description
	room string
}

func (v townRoomView) Size() image.Point { return v.t.viewScene(v.d, v.room).Size() }

// Paint draws the room: the square's layers, or a page's back layers, its
// seated speakers, its front layers and its buttons.
func (v townRoomView) Paint(dst *image.RGBA) {
	s := v.t.viewScene(v.d, v.room)
	r := v.d.Room(v.room)
	if r == nil {
		s.Paint(dst, "")
		return
	}
	page := v.t.roomArt(v.d, v.room).page
	s.Paint(dst, "back")
	words, font := v.t.in.Words, v.t.in.Font.Value()
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
	ui.DrawRoomButtons(dst, font, v.t.in.roomButtonStyle(v.d), buttons)
}

// roomButtonStyle is the description's room button style in the generator
// font it names; nil when the description names none.
func (in *InstallResources) roomButtonStyle(d *town.Description) *ui.RoomButtonStyle {
	if d == nil || d.RoomButtons == nil {
		return nil
	}
	b := d.RoomButtons
	s := &ui.RoomButtonStyle{Ink: ui.GeneratorInk(b.Ink).RGBA(), Hover: ui.GeneratorInk(b.HoverInk).RGBA(), Lit: b.Lit,
		LabelShift: b.LabelShift.Pt(), ValueShift: b.ValueShift.Pt()}
	for _, i := range b.OneLine {
		if i >= 0 && i < len(s.OneLine) {
			s.OneLine[i] = true
		}
	}
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
func (v townRoomView) paintSeats(dst *image.RGBA, r *town.RoomSpec, page *screenart.Set, words ui.Words, font *text.Font) {
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
func (v townRoomView) hireCaption() *int {
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
func (v townRoomView) ControlAt(p image.Point) (ui.TownSquareControl, bool) {
	r := v.d.Room(v.room)
	if r == nil {
		h := v.t.viewScene(v.d, v.room).HotspotAt(p)
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
func (v townRoomView) TipAt(p image.Point) (int, bool) {
	if v.d.Room(v.room) != nil {
		return 0, false
	}
	h := v.t.viewScene(v.d, v.room).HotspotAt(p)
	if h == nil {
		return 0, false
	}
	return h.TipSlot()
}

// sceneDialogue opens a town text section in place, as a click on a person
// of the square does (R2-ENGINE-234).

// roomPlayer is the room's audio scope, built for the room on first use and
// replaced when another room plays.

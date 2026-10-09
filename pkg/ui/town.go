package ui

import (
	"image"

	"againrom/pkg/render/frame"
)

// TownRow is one line of whatever room of the town is open: the text to show,
// and whether choosing it does anything.
//
// It is PickerRow's shape and deliberately a second type rather than a reuse.
// A picker row is a map the front-end listed; a town row is a door, a sentence
// about the state of the game, a shelf item or a line of dialogue, and the two
// lists have nothing in common except that both are text with a choosable flag.
// Sharing the type would make a later change to one silently a change to the
// other.
type TownRow struct {
	Text      string
	Choosable bool
}

// TownAction is what choosing a town row led to: a mission to open, a line to
// state, or neither.
//
// BOTH FIELDS MAY BE EMPTY, and that is the ordinary answer. Walking into a
// room, backing out of one, and paging a conversation all change what Rows
// reports and say nothing here — this type carries only what the SCREEN has to
// act on, which is exactly two things: leaving for a map, and a sentence.
//
// Open IS A MapOpener AND NOT A MISSION NUMBER, for the reason the
// won-mission advance already carries one: this package cannot name a
// mission, and a MapOpener is a shape three doors here already accept. So
// the town crosses the seam without this package gaining any way to describe
// a simulation value.
type TownAction struct {
	Open MapOpener
	Msg  string
	// Info marks Msg as a status echo of a success, which the player's view
	// does not show; an error leaves Info false and is shown.
	Info bool
}

// TownInfo is a success echo: the caller sees the text, the screen does not.
func TownInfo(msg string) TownAction { return TownAction{Msg: msg, Info: true} }

// TownScreen is the town as this package can see it: a header, some rows,
// some lines under them, the effect of choosing a row, and the way back out
// of whatever room is open.
//
// FIVE METHODS OVER STRINGS, INTS, BOOLS AND ONE CALLBACK, and that is the
// whole seam. This package may import the render tier and no other, so it must
// not be able to name a simulation type; nothing in this interface can name
// one. Which rooms exist, what a row means, what a conversation says and what a
// building hands out are all questions on the far side, exactly as MapTick's
// own doc says of what it advances: this tier holds the screen, not the town.
//
// Choose AND Back ARE THE ONLY MUTATIONS, and the front-end relies on it: it
// rebuilds its row list after those two calls and after nothing else. An
// implementation that changed what Rows reports at some other moment would
// show the player a list one press out of date.
type TownScreen interface {
	// Header is the line above the list — which room is open, and where.
	Header() string

	// Rows is what the list holds now, top to bottom.
	Rows() []TownRow

	// Footer is the lines under the list: what the player has, stated
	// wherever he is. It may be empty.
	Footer() []string

	// Choose acts on row i of what Rows last reported.
	Choose(i int) TownAction

	// Back leaves the open room and reports whether it left one.
	Back() bool
}

func townCanSave(t TownScreen) bool {
	if s, ok := t.(interface{ CanSave() bool }); ok {
		return s.CanSave()
	}
	return true
}

// TownSquareScreen is the optional presentation seam for the town's four-door
// square. Other rooms keep the row-button layout. A TownScreen that does not
// implement this interface keeps the row-button layout too.
type TownSquareScreen interface {
	AtTownSquare() bool
}

// TownShopScreen is the optional trade seam for the shop room.
//
// THE VIEW IS PIXELS AND NUMBERS AND NOTHING ELSE, exactly as
// ChargenPresentation is: what an ownership stamp means, what a price is and
// which shelf an item came off are all on the far side.
type TownShopScreen interface {
	AtTownShop() bool
	ShopScreen() ShopScreenView
	ShopClick(c ShopControl) TownAction
	ShopScroll(region ShopWheelRegion, rows int)

	// ShopDrag completes a drag between two of the screen's own surfaces —
	// the doll, the shelf grid and the pack strip — reached exactly as
	// ShopClick is (1005 round 2: "all of it works during a mission and in
	// the shop"). A pair it does not recognise is a no-op and a nil
	// TownAction, on ShopClick's own refusal shape.
	ShopDrag(from, to ShopControl) TownAction

	// ShopSuppressDoll is the shop's own refreshDollDrag (pkg/game's
	// world.go): pushed once a frame with the live doll-drag slot, 1-based,
	// 0 for none, so the figure the screen composes next redraws without
	// that one layer for exactly as long as the drag lifts it (spec bullet
	// 2, "the doll redraws without that layer for the duration of the
	// drag").
	ShopSuppressDoll(slot int)
}

// TownSquareArtScreen is the optional presentation seam for the town
// square's own picture. A TownScreen that does not implement it, or whose
// view carries no scene, keeps the row-button layout drawTown has always
// drawn for the square.
type TownSquareArtScreen interface {
	TownSquareView() TownSquareView
}

// TownSquareAnimator separates pointer delivery, lifecycle and paint. Client
// updates deliver stationary pointers too; original physical message delivery
// remains Unknown (TOWN-399, DIV-834).
type TownSquareAnimator interface {
	TownSquarePointer(image.Point)
	TownSquareActive(bool)
	AdvanceTownSquareAnimation()
}

// TownSurfaceScreen is the shared-shell seam used by the tavern and school.
// The view carries only resolved presentation values; mutations cross through
// the same TownAction door as the older row and shop seams.
type TownSurfaceScreen interface {
	AtTownSurface() bool
	TownSurface() TownSurfaceView
	TownSurfaceClick(c TownSurfaceControl, double bool) TownAction
}

// TownSurfaceAnimator advances presentation state once per own-surface paint.
// It is separate from TownSurface so hit tests and snapshots remain read-only.
type TownSurfaceAnimator interface {
	AdvanceTownSurfaceAnimation()
}

// SchoolTrainingLifecycle freezes the school's private clocks whenever its
// live room is not focused. It stays separate from TownSurfaceAnimator so
// synthetic surface implementations need not invent lifecycle state.
type SchoolTrainingLifecycle interface {
	SchoolTrainingActive(bool)
}

// TownSurfacePresser hears a left press on a tavern or school control before
// its release acts: the school requests a skill member on the press
// (VIDEO-SFX-059).
type TownSurfacePresser interface {
	TownSurfacePress(TownSurfaceControl)
}

// TavernInteriorLifecycle is the tavern's independent focus/menu/room seam.
// It is separate from TownSquareAnimator: the exterior and interior own no
// shared clocks, frames, directions or retained voices.
type TavernInteriorLifecycle interface {
	TavernInteriorActive(bool)
}

// ShopInteriorAnimator owns the shop's independent paint progression and
// focus/menu lifecycle. ShopScreen remains a read-only snapshot; only the live
// App composition route calls AdvanceShopInteriorAnimation.
type ShopInteriorAnimator interface {
	ShopInteriorActive(bool)
	AdvanceShopInteriorAnimation()
}

// TownDialogueScreen optionally supplies the already-composed portrait window
// for the current town dialogue part.
type TownDialogueScreen interface {
	TownDialogue() (*image.RGBA, bool)
	TownDialogueButton() (image.Rectangle, bool)
	AdvanceTownDialogue() TownAction
}

func atTownSquare(t TownScreen) bool {
	s, ok := t.(TownSquareScreen)
	return ok && s.AtTownSquare()
}

func townDialogueButton(t TownScreen) (image.Rectangle, bool) {
	d, ok := t.(TownDialogueScreen)
	if !ok {
		return image.Rectangle{}, false
	}
	button, ok := d.TownDialogueButton()
	if !ok {
		return image.Rectangle{}, false
	}
	pic, ok := d.TownDialogue()
	if !ok || pic == nil {
		return image.Rectangle{}, false
	}
	origin := image.Pt((frame.W-pic.Bounds().Dx())/2, (frame.H-pic.Bounds().Dy())/2)
	return button.Add(origin), true
}

func advanceTownDialogue(t TownScreen) (TownAction, bool) {
	d, ok := t.(TownDialogueScreen)
	if !ok {
		return TownAction{}, false
	}
	if _, shown := d.TownDialogue(); !shown {
		return TownAction{}, false
	}
	return d.AdvanceTownDialogue(), true
}

func townDialogue(t TownScreen) (*image.RGBA, bool) {
	d, ok := t.(TownDialogueScreen)
	if !ok {
		return nil, false
	}
	return d.TownDialogue()
}

// townShopScreen is the shop's own screen, and whether the shop room is the
// one open. Whether the dialogue itself is ALSO drawn, on top, is drawTown's
// own dialogueOpen test and no part of this function.
func townShopScreen(t TownScreen) (ShopScreenView, bool) {
	s, ok := t.(TownShopScreen)
	if !ok || !s.AtTownShop() {
		return ShopScreenView{}, false
	}
	return s.ShopScreen(), true
}

func atTownShop(t TownScreen) bool {
	s, ok := t.(TownShopScreen)
	return ok && s.AtTownShop()
}

// townSquareView answers the square's own view and whether it can be drawn:
// the screen must be at the square, implement TownSquareArtScreen, and
// supply a scene. A game supplies no scene when its square's art did not
// load.
func townSquareView(t TownScreen) (TownSquareView, bool) {
	s, ok := t.(TownSquareArtScreen)
	if !ok || !atTownSquare(t) {
		return TownSquareView{}, false
	}
	v := s.TownSquareView()
	if v.Scene == nil {
		return TownSquareView{}, false
	}
	return v, true
}

func townSurfaceScreen(t TownScreen) (TownSurfaceView, bool) {
	s, ok := t.(TownSurfaceScreen)
	if !ok || !s.AtTownSurface() {
		return TownSurfaceView{}, false
	}
	return s.TownSurface(), true
}

func atTownSurface(t TownScreen) bool {
	s, ok := t.(TownSurfaceScreen)
	return ok && s.AtTownSurface()
}

func (f *flow) clickTownSurface(c TownSurfaceControl, double bool) bool {
	if f.screen != ScreenTown || f.town == nil {
		return false
	}
	s, ok := f.town.(TownSurfaceScreen)
	if !ok || !s.AtTownSurface() {
		return false
	}
	f.applyTownAction(s.TownSurfaceClick(c, double))
	return true
}

// clickShop sends a click on one control across the seam and applies whatever it
// answered, which is the same path a row press takes.
func (f *flow) clickShop(c ShopControl) bool {
	if f.screen != ScreenTown || f.town == nil {
		return false
	}
	if c.Kind == ShopControlMerchant {
		// The painted merchant has no click test (TOWN-478): a press on him
		// reaches no handler and leaves the message line as it was.
		return false
	}
	s, ok := f.town.(TownShopScreen)
	if !ok || !s.AtTownShop() {
		return false
	}
	f.applyTownAction(s.ShopClick(c))
	return true
}

// dragShop sends a completed drag between two shop surfaces across the seam,
// clickShop's own shape restated for two controls (1005 round 2).
func (f *flow) dragShop(from, to ShopControl) bool {
	if f.screen != ScreenTown || f.town == nil {
		return false
	}
	s, ok := f.town.(TownShopScreen)
	if !ok || !s.AtTownShop() {
		return false
	}
	f.applyTownAction(s.ShopDrag(from, to))
	return true
}

// TipScreen is the tip panel's own mutation seam (1018 spec behaviours 3, 4):
// one screen implementation covers every room's own panel, since the panel
// is the same widget everywhere and the room it is showing on is read on the
// far side of this seam (townScreen.room), not passed in from here.
type TipScreen interface {
	// CloseTip dismisses whichever room's panel is open, for the rest of
	// the session (spec behaviour 3).
	CloseTip()
	// ToggleTips flips the permanent suppression store, read again at every
	// room's own next entry (spec behaviour 4).
	ToggleTips()
}

// closeTip and toggleTips are ClickTownSurface's own shape restated with no
// TownAction to apply: a tip mutation never opens a mission or states a
// line, so there is nothing for flow.applyTownAction to do afterward.
func (f *flow) closeTip() bool {
	if f.screen != ScreenTown || f.town == nil {
		return false
	}
	s, ok := f.town.(TipScreen)
	if !ok {
		return false
	}
	s.CloseTip()
	return true
}

func (f *flow) toggleTips() bool {
	if f.screen != ScreenTown || f.town == nil {
		return false
	}
	s, ok := f.town.(TipScreen)
	if !ok {
		return false
	}
	s.ToggleTips()
	return true
}

// suppressShopDoll pushes this frame's live doll-drag slot across the seam,
// clickShop's own shape restated for a value with no TownAction to apply
// (1005 round 2).
func (f *flow) suppressShopDoll(slot int) bool {
	if f.screen != ScreenTown || f.town == nil {
		return false
	}
	s, ok := f.town.(TownShopScreen)
	if !ok || !s.AtTownShop() {
		return false
	}
	s.ShopSuppressDoll(slot)
	return true
}

func (f *flow) scrollShop(at image.Point, rows int) bool {
	if f.screen != ScreenTown || f.town == nil {
		return false
	}
	s, ok := f.town.(TownShopScreen)
	if !ok || !s.AtTownShop() {
		return false
	}
	region := ShopWheelAt(at)
	if region == ShopWheelNone {
		return false
	}
	s.ShopScroll(region, rows)
	return true
}

// townRows converts the far side's rows into the picker's own, so the town's
// list is scrolled, marked, hit-tested and clipped by the map list's own
// model rather than by a second copy of it.
func townRows(rows []TownRow) []PickerRow {
	out := make([]PickerRow, len(rows))
	for i, r := range rows {
		out[i] = PickerRow{Text: r.Text, Choosable: r.Choosable}
	}
	return out
}

// showTown adopts the town screen and builds the list its current room holds.
//
// IT IS THE ONE PLACE f.screen BECOMES ScreenTown, which is what lets every
// method on the town arm assume the list is not nil once it is showing — the
// same single-assignment rule armChargen states for the generation screen, and
// for the same reason: a town screen showing over a nil list turns the next
// frame into the panic this package's design refuses to have.
//
// A NIL TownScreen IS REFUSED and reported, rather than showing an empty town:
// a front end with no town has nowhere to send a finished mission, and finding
// that out on the map list with the reason stated is better than finding it out
// on a screen with no rows and no way to know why.
func (f *flow) showTown(msg string) bool {
	if f.showCampaignEnding() {
		return true
	}
	if f.town == nil {
		return false
	}
	f.townList = NewPicker(townRows(f.town.Rows())).SetWindow(townListWindow(f.town))
	// The surface transition is setScreen's (flow.go): this build's one town
	// screen stands for the shop, inn, school and town-square routines alike,
	// which TOWN-372 names individually by address and music path (R1315
	// music\shop.wav, R1318 music\inn.wav, R1319 music\schoolm.wav,
	// R1320 music\Town.wav) and grades Medium on the step from a pushed
	// music path to a surface identity. One screen standing for four surfaces
	// is the divergence, not an absence of decoded detail.
	f.setScreen(ScreenTown)
	f.msg = msg
	return true
}

// townListWindow is how many rows the town's list may occupy: the whole
// frame. The shop room has no list at all, so no room needs a window any
// more; the function is kept as the one place the answer is decided.
func townListWindow(TownScreen) int { return 0 }

// refreshTown rebuilds the list after the model was mutated.
//
// THE SELECTION RESETS TO THE FIRST ROW. Choose and Back are the only callers
// and both of them mean the room may have changed; keeping a place across a
// room change would be keeping a place in a different list, and the one case
// where the room did NOT change — a conversation paging — is a list the player
// is reading from the top anyway.
func (f *flow) refreshTown() {
	if f.town == nil {
		return
	}
	f.townList = NewPicker(townRows(f.town.Rows())).SetWindow(townListWindow(f.town))
}

// chooseTown acts on the town's selected row and does whatever the far side
// says (FR-8a).
//
// A MISSION OPENS THROUGH THE SAME enter EVERY OTHER DOOR USES. The gates are
// a fourth door into a map screen beside the picker's own choice, the mission
// door and the won-mission advance, and it goes through enter for the reason
// each of those does: the cadence ladder and the command mode are established
// there, and a door that assigned the seams itself would open a mission whose
// first speed key slams the clock.
//
// AN OPENER THAT FAILS LEAVES THE PLAYER IN THE TOWN holding the failure's own
// words, which is choose()'s own answer to the same event on the map list. It
// does NOT mark the row unusable: the town's rows are rebuilt from the model on
// the next mutation, so a mark here would be erased without ever having meant
// anything.
func (f *flow) chooseTown() {
	if f.screen != ScreenTown || f.town == nil || f.townList == nil {
		return
	}
	i, ok := f.townList.Choose()
	if !ok {
		return
	}
	act := f.town.Choose(i)
	f.applyTownAction(act)
}

func (f *flow) advanceTownDialogue() bool {
	if f.screen != ScreenTown || f.town == nil {
		return false
	}
	act, ok := advanceTownDialogue(f.town)
	if !ok {
		return false
	}
	f.applyTownAction(act)
	return true
}

func (f *flow) applyTownAction(act TownAction) {
	f.refreshTown()
	f.msg = act.Msg
	if act.Info {
		f.msg = ""
	}
	if act.Open == nil {
		return
	}
	v, tick, order, cadence, affect, advance, attack, grab, stance, march, err := act.Open()
	if err != nil {
		f.msg = err.Error()
		return
	}
	f.enter(v, tick, order, cadence, affect, advance, attack, grab, stance, march)
}

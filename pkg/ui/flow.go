package ui

import (
	"image"
	"time"

	"againrom/pkg/render/terrain"
	"againrom/pkg/render/text"
)

// Screen identifies which of the front-end's screens is showing. Exactly one
// shows at a time.
type Screen int

const (
	// ScreenMenu is the root screen. The application opens here, and Esc here is
	// the only way out of the program.
	ScreenMenu Screen = iota
	ScreenPicker
	ScreenMap
	// ScreenChargen is the generation screen. It is APPENDED here, after
	// ScreenMap and not before it, so no existing constant's value moves — a
	// Screen is compared and switched on by value throughout this package, and
	// a shifted constant would silently retarget every one of those sites.
	//
	// It is reached through App.OpenChargen, called before Run, and — since
	// 0140 — through a picker row a ChargenGate claims. Both go through
	// flow.armChargen and nothing else assigns this value, so "the screen is
	// armed whenever it shows" stays a property of one statement rather than of
	// a list of transitions. A front-end that neither calls OpenChargen nor sets
	// a gate never reaches this value at all.
	ScreenChargen
	// ScreenTown is the town — the screen the player lives on between
	// missions. It is APPENDED here, after ScreenChargen, for the reason
	// ScreenChargen itself was: a Screen is compared and switched on by value
	// throughout this package, and a shifted constant would silently retarget
	// every one of those sites.
	//
	// IT IS ONE VALUE AND THE TOWN HAS FIVE ROOMS. Which room is open lives in
	// the model behind the seam, not here, so this package cannot tell the
	// tavern from the gates and cannot grow a rule about either. Five values
	// would be five step arms and five draw arms for one list drawn five ways.
	//
	// It is reached through flow.showTown and nothing else assigns it, so
	// "the list is not nil whenever the screen is showing" stays a property
	// of one statement.
	ScreenTown
	// ScreenGameMenu is the mini-menu — RETURN / SAVE / LOAD / EXIT — shown
	// over a running mission and over the town, and ScreenLoad is the LOAD GAME
	// window. Both are APPENDED here, after ScreenTown, for the reason
	// ScreenChargen and ScreenTown themselves were: a Screen is compared and
	// switched on by value throughout this package, and a shifted constant
	// would silently retarget every one of those sites.
	//
	// THE GAME BEHIND THE MINI-MENU DOES NOT ADVANCE, and that falls out of
	// its being a screen rather than an overlay: the tick is issued on the
	// map arm alone, so a menu that is showing is a mission that is not
	// running, with no statement anywhere that had to arrange it.
	//
	// EACH REMEMBERS THE SCREEN IT WAS ARMED FROM (menuBack, loadBack) and
	// neither returns to a constant: the mini-menu opens from two screens
	// and the load window from two, and a constant would send the player
	// somewhere he was not. ScreenGameMenu is reached through openGameMenu
	// and ScreenLoad through openLoad, and nothing else assigns either, so
	// "the list is not nil whenever the screen is showing" stays a property
	// of one statement each.
	ScreenGameMenu
	ScreenLoad
	// ScreenDocuments is the campaign documents panel, APPENDED here for the
	// reason every value above it was.
	//
	// IT IS A SCREEN AND NOT AN OVERLAY OVER THE MISSION, which is what makes
	// the mission behind it stop: the tick is issued on the map arm alone, as
	// ScreenGameMenu's own note above records. The original runs one 640x480
	// surface and its panel covers the whole of it, so a screen and a
	// full-screen window draw the same picture there; this build's mission
	// frame is 1024x768, so the two are not the same picture here and the
	// choice had to be made (DIV-299).
	//
	// It remembers the screen it was armed from (docBack). This build has
	// ONE production entry point, App.step's ScreenMap arm, which arms it
	// with ScreenMap; DIV-298 records why, and the original's own gate
	// (campaign+0x3dc == 1, TOWN-352's bit 0 alone) is the mission frame
	// with nothing over it, so there is no town route to build. The field
	// is a parameter rather than a constant because closeDocuments must
	// return where openDocuments was called from, and a constant would be
	// a second place to change if a later story adds a route.
	// It is reached through openDocuments and nothing else assigns it.
	ScreenDocuments
	// ScreenCutscene is a presentation overlay. flow retains its destination;
	// only App.Screen reports this value while a decoder is active.
	ScreenCutscene
	// ScreenSave holds the paused session while the player chooses save targets.
	ScreenSave
	ScreenEnding
	ScreenCutsceneLibrary
	ScreenCredits
)

func (s Screen) String() string {
	switch s {
	case ScreenCutsceneLibrary:
		return "cutscenes"
	case ScreenCredits:
		return "credits"
	case ScreenEnding:
		return "ending"
	case ScreenMenu:
		return "menu"
	case ScreenPicker:
		return "picker"
	case ScreenMap:
		return "map"
	case ScreenChargen:
		return "chargen"
	case ScreenTown:
		return "town"
	case ScreenGameMenu:
		return "gamemenu"
	case ScreenLoad:
		return "load"
	case ScreenSave:
		return "save"
	case ScreenDocuments:
		return "documents"
	case ScreenCutscene:
		return "cutscene"
	case ScreenMod:
		return "mod"
	}
	return "unknown"
}

// MapTick advances whatever the loader put under the open map screen by one
// tick. The map screen calls it once per tick and nothing else calls it at all.
//
// It takes no argument and returns nothing ON PURPOSE. This package may
// import the render tier and no other, so it must not be able to name a
// simulation type — and a parameterless function names none. What is
// behind it, how it decides what to apply, and whether there is anything
// behind it at all are questions this tier cannot ask and does not need to:
// it holds the seam, not the world.
type MapTick func()

// MapOrder issues one move order into whatever the loader put under the open map
// screen: the entity to move, and the map cell to move it to. The map screen
// calls it at most once per tick of that screen — never on any other screen —
// and nothing else calls it at all.
//
// THREE SCALARS ARE THE WHOLE PAYLOAD, for the same reason MapTick takes
// none: this package may import the render tier and no other, so it must not
// be able to name a simulation type. A uint32 and two ints name none, and no
// new value type crosses out of here for the order either.
//
// WHEN the order is applied, whether an advance is due, and what it is
// applied to are questions on the far side of the seam. This tier issues; it
// holds no queue and counts nothing. Widening MapTick to take the frame's
// orders and return how many it applied was rejected exactly there: the
// queue would then sit in this package while whether a logic tick fired is
// decided beyond it, so an advance firing no tick would drop the order and
// turn exactly-one into none.
type MapOrder func(entity uint32, x, y int)

// MapCadence re-rates whatever the loader put under the open map screen,
// sets or clears its stop, and selects its paced or unpaced owner-loop arm.
// The map screen calls it when its own cadence changes or an explicit phase
// reset is requested — never per tick, and never on any other screen —
// and nothing else calls it at all.
//
// FOUR SCALARS ARE THE WHOLE PAYLOAD, for the same reason MapTick takes none:
// this package may import the render tier and no other, so it must not be able
// to name a simulation type. An int and three bools name none. reset is an
// edge, not stored state: Ctrl+numpad minus resets phase even when already paced.
//
// IT CARRIES A PERIOD AND NOT A RATE. A tick length in microseconds is what
// both consumers physically hold, and it is computed ONCE, on the statement
// that changes the cadence, from the one ladder in pkg/render/terrain.
// Carrying the ladder's position instead — a rate, a speed index, a rung
// — would leave each side of the seam to divide for itself, and two
// divisions of one number is how the two came to disagree in the first
// place: the game's own speeds truncate to a whole millisecond and our own
// extension does not, so no single quotient serves both and the SHIPPED
// cadence is the one a rate loses. What the far side does with the period
// — which clock it re-rates, what it declines to run while the flag is set
// — is a question this tier cannot ask: it holds the seam, not the world.
//
// Widening MapTick to carry them instead was rejected: the once-per-tick
// call takes nothing and returns nothing on purpose, and a cadence that
// crossed on it would cross at the tick rate rather than at the rate it
// changes.
type MapCadence func(periodUS int, stopped, unpaced, reset bool)

// MapAffect issues one BLOW into whatever the loader put under the open map
// screen: the entity to hit, and whether the blow is a kill or a chip. The map
// screen calls it at most once per marked unit per key press — never on any
// other screen — and nothing else calls it at all.
//
// TWO SCALARS ARE THE WHOLE PAYLOAD, for the same reason MapTick takes none: a
// uint32 and a bool name no simulation type.
//
// NO AMOUNT CROSSES. What a chip takes off is decided on the far side, by the
// tier that owns the world and can read the unit's own maximum — this tier would
// otherwise hold a damage rule, and the tier that applies it would take a number
// it has no way to check. A bool rather than a kind byte for the same reason
// there are two seams and not one: this side names the two things it can ask
// for, not the command stream that answers them.
//
// It is a SECOND seam beside MapOrder rather than a widening of it. Widened,
// every existing call site would have to say it is not a blow, and a move would
// carry a field it never means.
type MapAffect func(entity uint32, kill bool)

// MapAttack issues one ATTACK ORDER into whatever the loader put under the open
// map screen: the entity ordered, and the entity it is ordered onto. The map
// screen calls it at most once per selected unit per consuming press — never on
// any other screen — and nothing else calls it at all.
//
// TWO ENTITY IDS ARE THE WHOLE PAYLOAD, for the same reason MapTick takes none:
// two uint32 name no simulation type, and both were minted on the far side and
// arrived in a MapEntity, so this package still cannot CONSTRUCT an id.
//
// IT IS AN ORDER AND NOT A BLOW, which is the whole of why it is not MapAffect.
// A blow is applied and is over; an order names a victim and leaves what happens
// to the world's own cycle — how long a blow takes, whether it reaches, what it
// costs. So nothing about damage crosses here, in either direction, and the far
// side is free to decide that an order at a distance walks first.
//
// It is a THIRD seam beside MapOrder rather than a widening of it, and the
// argument is MapAffect's with one clause added. Widened, every existing call
// site would have to say it is not an attack and every move would carry a victim
// it never means — and the two payloads are not merely disjoint but of different
// KINDS, since an attack names an entity where a move names a cell. A widened
// seam would carry two dead fields rather than one dead flag.
//
// The far side may make the order MEAN more than this tier can say — that is the
// point of a seam — but it may not need more: an attacker, a victim, and no
// third thing.
//
// SPELL IS THE OWNER'S RULING AND ONLY THE RULING'S SHAPE: "a click selects
// any spell from the book, and the cursor casts it at a target" — that
// sentence is the whole of what is decoded here, and it is why the third
// argument is a spell id and not, say, a target kind or an effect.
//
// NOT A NINTH SEAM ON MapOpener. Widening the loader's tuple for one more
// value it hands over once was rejected in favour of widening a seam that
// already fires on every press this feature cares about — see command.go's
// own gesture.spell and order.spell for where the id is carried from the
// click to here.
// With spell zero, cell distinguishes a structure handle from a unit handle.
// With a spell, it retains the existing cell-cast meaning. No mere hover calls it.
type MapAttack func(entity, victim, spell uint32, x, y int, cell bool)

// MapGrab performs one PICK-UP into whatever the loader put under the open
// map screen — the container transfer alone, with no walk, no path and no
// order behind it. The map screen calls it at most once per press of the key
// it is bound to — never on any other screen — and nothing else calls it
// at all.
//
// IT TAKES NO PAYLOAD, which is MapTick's own shape and for a related but
// different reason. Like every seam here, this package may import the render
// tier and no other, so it must not be able to name a simulation type — a
// parameterless function names none, exactly as MapTick's own doc says. But
// there is a SECOND reason this one never had a choice about: WHICH
// CHARACTER the key acts for is the WIRING TIER's own question, and that
// tier already answers it once, for the inventory window's own subject. A
// seam carrying an entity id would let this key act on a character the
// window does not show — a second answer to a question one tier already
// settled. A seam that cannot name one is what keeps the key acting on
// exactly the character the open window is about, whatever this package is
// ever handed.
//
// It is a FOURTH seam beside MapOrder, MapAffect and MapAttack rather than a
// widening of any of them, for the same reason those three are not one
// another: an all-or-nothing container transfer with no distance test, no
// ownership test and no capacity test is neither a move, a blow nor an
// attack order, and widening any of the three would give it a payload it
// never means.
//
//   - aimed FALSE is the KEY. entity, col and row are unread. The far side acts
//     for the inventory window's subject at that subject's own cell.
//   - aimed TRUE is the CLICK. entity is the unit `AI-CURSOR-242`'s gate named,
//     col and row are the sack's cell, and the far side issues the standing
//     order rather than transferring anything now.
//
// A BOOL RATHER THAN A SECOND SEAM keeps MapOpener's tuple at eleven values.
// Adding a twelfth reaches MapOpener, flow.enter, seven production call sites
// and the headless harness, for a distinction the far side answers in one
// branch.
type MapGrab func(entity uint32, col, row int, aimed bool)

// MapStance issues one CELL-FREE STANDING ORDER into whatever the loader put
// under the open map screen: the entity ordered, and which of the two the
// order is. The map screen calls it at most once per selected unit per press
// of the key it is bound to — never on any other screen — and nothing
// else calls it at all.
//
// A uint32 AND A BOOL ARE THE WHOLE PAYLOAD, for the same reason MapTick takes
// none: this package may import the render tier and no other, so it must not be
// able to name a simulation type. The id was minted on the far side and arrived
// in a MapEntity, so this package still cannot CONSTRUCT one.
//
// THE BOOL IS THE WHOLE CHOICE, and it is a bool rather than a kind byte for
// MapAffect's own reason: this side names the two things it can ask for, not the
// command stream that answers them. TRUE is "hold this ground, and you may leave
// it to meet what you notice — and come back to it afterwards"; FALSE is "hold
// exactly this cell". Which of the law's own order bytes each becomes is decided
// wholly on the far side, and neither number appears anywhere in this package.
//
// IT IS A FIFTH SEAM beside MapOrder, MapAffect, MapAttack and MapGrab rather
// than a widening of any of them. Widened onto MapOrder, every existing call
// site would have to say it is not a stance, and every move would carry a flag
// it never means; widened onto MapAttack, a standing order would carry a victim.
type MapStance func(entity uint32, guard bool)

// MapMarch issues one AIMED STANDING ORDER into whatever the loader put
// under the open map screen: the entity ordered, which of the two the order
// is, and the map cell it is aimed at. The map screen calls it at most once
// per selected unit per consuming press — never on any other screen —
// and nothing else calls it at all.
//
// A uint32, A BOOL AND TWO ints ARE THE WHOLE PAYLOAD, for MapStance's own
// reasons. TRUE is "walk between where you stand and this cell, and keep doing
// it"; FALSE is "go to this cell, and fight what you meet on the way". Again
// neither of the law's own numbers appears here.
//
// IT IS A SIXTH SEAM beside MapStance rather than the same one widened, and the
// split is exactly the one real difference in payload: these two orders NAME A
// CELL and the other two do not. A single seam would give a stance an x and a y
// it never means — and a zero pair is not an absence here, it is the map's own
// corner, so there would be no sentinel to read it as one either. That is the
// same argument every widening rejected in this file was rejected on.
type MapMarch func(entity uint32, patrol bool, x, y int)

// MapLoader turns a chosen row index into a running map viewer, the tick that
// advances what runs under it, the seam an order leaves through, the seam its
// cadence is set through and the seam a blow reaches it by, or reports why it
// could not.
//
// The blow seam is LAST in the tuple, which is the convention every seam added
// since the tick has followed: appended, so that no existing position moves and
// a reader of an older call site is not silently reading a different value.
//
// It is injected rather than reached for, which is what lets the whole screen
// flow — including the arm where a chosen map passes its metadata read and then
// fails to decode — be exercised without a game file or a window.
//
// The tick, the order, the cadence and the blow may all be nil: a loader that
// puts nothing under the map screen says so by handing back none, and the map
// screen then advances nothing, issues nothing, re-rates nothing and hits
// nothing. That is not an error, and it is what the standalone-viewer path and a
// bare test loader do.
type MapLoader func(index int) (*Viewer, MapTick, MapOrder, MapCadence, MapAffect, MapAdvance, MapAttack, MapGrab, MapStance, MapMarch, error)

// MapAdvance applies the selected notice action and reports where the
// front-end should go next, what to say when it gets there, and — for one
// destination alone — what to open once it is there. The map screen calls
// it once per press of the three inputs that advance a notice — never on
// any other screen — and nothing else calls it at all.
//
// THE ACTION AND THREE RESULTS ARE THE WHOLE PAYLOAD, for the same reason
// MapTick takes none: this package may import the render tier and no other,
// so it must not be able to name a simulation type. An integer destination,
// a string and a MapOpener name none — the opener is the same shape
// OpenMission and the generation screen's own door already hand this
// package, so this seam widening to carry one is not the first payload here
// that could name a simulation type, it is the same one two doors already
// accept. In particular an OUTCOME does not cross — what the mission came
// to is decided on the far side, and this tier is told a screen to show, a
// line to show there, and — where the screen is a successor's own map —
// how to open it.
//
// IT IS APPENDED to the loader's tuple, which is the convention every seam added
// since the tick has followed: no existing position moves, so a reader of an
// older call site is not silently reading a different value. The cost is not
// hidden — every destructure of that tuple changes — and it is accepted because
// appending buys that no existing member changes MEANING.
//
// THE THIRD RESULT WAS ADDED TO THIS SEAM, NOT A NINTH ONE ONTO MapLoader.
// Every producer built before this story hands back nil for it and every
// existing caller keeps compiling and drawing the same frame, because a nil
// opener is read only on NoticeToMission — the one destination nothing
// produced before this story.
//
// The far side decides EVERYTHING about what an advance does: whether the
// notice pages to another part or closes, whether the mission is over, which
// of the two endings it was, and — new to 0131 — what a won mission's
// successor is and whether it can be opened at all. This tier holds only the
// press and the answer.
type MapAdvance func(action ...NoticeAction) (NoticeDest, string, MapOpener)

// flow is the front-end's screen state machine: which screen is showing, what the
// picker holds, and the message a failed load left behind.
//
// It is pure. Every transition is a function of the events it is handed, so the
// transitions are decidable in a test and the engine layer above it holds no
// branching of its own.
type flow struct {
	endingSeams                                     endingSeams
	hallFromMenu                                    bool
	menuArt                                         *MenuPanelArt
	helpScroll                                      *helpScrollArt
	questPress                                      bool
	questTop                                        int
	questBar                                        scrollBarInput
	ending                                          EndingView
	endingPage, endingTop, endingFocus, endingPress int
	screen                                          Screen
	picker                                          *Picker
	load                                            MapLoader
	viewer                                          *Viewer

	cursor *CursorManager

	// words is the resolved program-chosen word set. The in-game menu's rows
	// are built from it, and enter copies it onto every viewer this flow opens,
	// so the menu's words and the notices' words are one value and cannot
	// disagree.
	//
	// It is the AUTHORED set until App.SetWords replaces it, which is what the
	// standalone developer viewer and every hand-assembled test flow keep.
	words Words

	// menuFont is what the in-game menu's rows are drawn with. It arrives with
	// the words and for their sake: the install's bytes are CP866 on a Russian
	// root, and this font carries that root's language selector.
	//
	// A NIL FONT DRAWS THE ROWS WITH THE DEBUG FONT, which is what this package
	// did before 0168 and what cmd/mapview and every test flow still do. That
	// path is ASCII only, so it is correct exactly where the labels are the
	// authored English.
	menuFont *text.Font

	encodeMenuKey func(rune) (byte, bool)

	// tick advances what the loader put under the open map screen, and is held
	// beside the viewer rather than on it: Viewer.step is the standalone
	// viewer's path too, so hanging the advance there would advance a world in
	// cmd/mapview, which owns none.
	//
	// It lives and dies with viewer — set together by choose, dropped together
	// by escape — so whatever it closes over is released on the same statement
	// the map screen is left on.
	//
	// order is the other half of that seam: the way out for a move order the
	// map screen's own frame produced. It is held HERE and beside the tick for
	// exactly the tick's reasons — Viewer.step is the standalone viewer's
	// path too, so hanging an order path there would give cmd/mapview a world
	// to order about, which it does not own — and it is set and dropped in
	// the SAME STATEMENT the tick is, so no transition can leave the front-end
	// holding one without the other.
	//
	// cadence is the third member of that seam, set and dropped in those same
	// two statements for those same reasons: a transition that left the
	// front-end holding a cadence without a tick would re-rate a world it could
	// not advance.
	//
	// affect is the fourth and last, set and dropped in those same two
	// statements for those same reasons: a front-end holding a way to hit a
	// world it could not advance would leave a blow queued for a tick that never
	// comes.
	//
	// advance is the fifth and last, set and dropped in those same two
	// statements for those same reasons: a front-end holding a way to dismiss a
	// notice over a map it could not advance would be answering for a mission
	// that had already been left.
	//
	// attack is the sixth and last, set and dropped in those same two
	// statements for those same reasons: a front-end holding a way to order an
	// attack into a world it could not advance would leave an order queued for
	// a tick that never comes.
	//
	// stance and march are the EIGHTH and NINTH, set and dropped in those same
	// two statements for those same reasons: a front-end holding a way to
	// change a group's standing order in a world it could not advance would
	// leave that order queued for a tick that never comes.
	tick    MapTick
	order   MapOrder
	cadence MapCadence
	affect  MapAffect
	advance MapAdvance
	attack  MapAttack
	grab    MapGrab
	stance  MapStance
	march   MapMarch

	// rung and stopped are the FRONT-END'S OWN cadence: a position on the
	// terrain package's cadence ladder and a switch beside it. They live here
	// rather than on the viewer because the stop belongs to the world and the
	// viewer is also the standalone viewer's — which owns no world and must
	// gain no stop.
	//
	// A RUNG AND NOT A RATE. What the two keys move is a position in an ordered
	// set of cadences, and the game's own nine speeds are nine of them; a rate
	// cannot hold that position, because the shipped periods are a truncated
	// whole millisecond and the rate that names one does not compute it back
	// — rate 16 is 62500 us where the game's speed 4 is 62000. Holding the
	// rate is what made the opening cadence unreachable after one press.
	//
	// They are born in choose at the persisted normal rung, or the historical
	// map-load default when no preference exists. The world and viewer are
	// built at that default; enter crosses a different restored rung once
	// before the first frame and explicitly clears the map-only unpaced
	// selector. The first speed press therefore steps from what the player last
	// selected instead of replacing it.
	//
	// stopped IS THE PLAYER'S OWN SWITCH AND NOTHING ELSE WRITES IT. A notice
	// suspending the world does NOT write here — it contributes a second
	// disjunct at syncCadence, where the two are resolved into the one stop the
	// far side is told about. That is what makes "a notice restores whatever
	// the player had" a property of there being no statement that could do
	// otherwise, rather than a save-and-restore pair that can run twice, run
	// without its partner, or not run at all because the mission ended first.
	rung    int
	stopped bool
	unpaced bool

	// preferredRung is the normal deadline-paced rung the NEXT map opens on;
	// preferredRungSet distinguishes the valid slowest rung 0 from a hand-built
	// flow that never installed a preference. persistRung is the optional local
	// settings sink. Only a real bare +/- move updates this trio: pause and the
	// Ctrl+numpad owner-loop selector are session state and never enter it.
	preferredRung    int
	preferredRungSet bool
	persistRung      func(int)

	// farRung, farStopped and farUnpaced are the cadence the far side was LAST
	// TOLD ABOUT — not the cadence the player asked for.
	//
	// The distinction is the whole of why they exist. Before this story the
	// front-end decided whether to write by comparing against rung and stopped,
	// which was sound while the player was the only thing that could move the
	// cadence. A notice is a SECOND CAUSE: it changes what the far side ought to
	// be holding without changing either of the player's fields, so a comparison
	// against them would see nothing to write and the suspension would never
	// cross. There is no way to notice that by reading the write; it is only
	// visible by asking what the compared value MEANS.
	//
	// They are born in enter at what both consumers were constructed holding
	// — the map-load default, running and paced. If the preference differs,
	// syncCadence crosses it once and updates these fields before the first map
	// frame; the default retains the historical no-call entry path.
	farRung    int
	farStopped bool
	farUnpaced bool

	msg string

	// chargen, chargenBegin, chargenBack and chargenGate are the generation
	// screen's own state. Nothing above this comment concerns that screen at
	// all, so these fields are APPENDED here rather than woven in beside picker
	// or viewer, and every existing field above keeps exactly the meaning it
	// already had.
	//
	// chargen is nil until the screen is armed. Every method reached from the
	// chargen arm — stepChargen, drawChargen — is written to assume it is not
	// nil once f.screen is ScreenChargen, which is what armChargen's own single
	// gate buys: there is exactly ONE statement in this package that assigns
	// this field, and it refuses a nil pointer, so both arming doors — startup
	// through App.OpenChargen and a picker row through choose() — inherit that
	// refusal instead of each restating it.
	//
	// chargenBegin turns a CONFIRMED ChargenResult into whatever the wiring
	// tier decides: a MapOpener to enter through the SAME flow.enter the
	// picker's own choose() and OpenMission already use — a third entry path
	// would drift from the cadence rung and the command mode both of those
	// already establish — or the reason it declined.
	//
	// chargenBack is the screen Escape returns this one to (0140): the screen it
	// was ARMED FROM, not a constant. Generation is no longer only a startup
	// screen — a picker row can arm it mid-run — and a player who backs out of
	// generating a character must land back on the map list with his place in it
	// kept, not at the main menu with the list unwound. ScreenMenu is iota 0, so
	// the zero value is startup's own destination and the door that arms this
	// screen before Run needs no special case to keep behaving as it did.
	//
	// chargenGate is asked, in choose() AND NOWHERE ELSE, whether the picker row
	// just chosen opens generation instead of its map. It is nil on every
	// front-end that does not set one, and a nil gate is the picker this package
	// has always had. It answers in THIS package's own vocabulary — a row index
	// in, a model and a begin callback out — because what makes a row want
	// generation is a fact of the wiring tier (a campaign mission, in the shipped
	// game) and this package may not import the tier that knows it.
	//
	// "IN choose() AND NOWHERE ELSE" IS LOAD-BEARING, not an implementation
	// note. A map screen entered any other way — the mission door, and above
	// all advanceNotice's NoticeToMission arm, which is how a won mission opens
	// its successor — reaches enter() directly and never passes this field.
	// That is what makes "generation belongs to a campaign being started, never
	// to one being continued" a property of the control flow rather than a
	// condition somebody has to keep writing (0140, owner).
	chargen      *Chargen
	chargenBegin func(ChargenResult) (MapOpener, error)
	chargenBack  Screen
	chargenGate  ChargenGate

	// newGameChargen is what activateNewGame calls to arm generation directly
	// instead of opening the map picker (owner reversal). It is installed by
	// cmd/againrom for an ordinary launch and left nil only under -picker,
	// which is what makes the map picker the debug route: nil is the picker
	// this package has always had, exactly as chargenGate's own nil is "no
	// generation on this row".
	//
	// CALLED FRESH ON EVERY PRESS, and for chargenGate's own reason: a player
	// who backs out of generating a character and presses NEW GAME again is
	// starting again, not resuming a spread he abandoned, and a shared model
	// would hand him the second attempt with the first one's half-spent
	// points already on it.
	newGameChargen NewGameChargen

	// newGameDirect, when set, is what NEW GAME opens instead of generation or
	// the picker: the first mission of a base that ships no generation art,
	// opened with the default party. App.activateNewGame calls it; a nil value,
	// a nil opener or a failed open falls back to what newGameChargen arms.
	newGameDirect NewGameDirect

	// town and townList are the town screen's own state. Nothing above this
	// comment concerns that screen, so they are APPENDED here for the reason
	// the chargen block above them was, and every existing field keeps exactly
	// the meaning it had.
	//
	// town is the seam and it is INSTALLED ONCE AND NEVER CLEARED — unlike
	// the map screen's seven, which live and die with the viewer. That is
	// the difference between a screen and a session: the map seams answer
	// for one running mission and must not outlive it, while the town is
	// what the missions happen between, and a front end that dropped it on
	// leaving would have nowhere to send the next mission's ending.
	//
	// townList is the rows of whatever room is open, held as the MAP LIST'S
	// OWN MODEL. Reusing *Picker rather than writing a second list model is
	// what makes the scroll window, the selection marker, the hit test and
	// the rune clip one implementation — so a row that is drawn is a row
	// that can be clicked, which is the property picker.go's own header
	// states and this screen inherits rather than re-earns.
	//
	// It is rebuilt on Choose and on Back and at no other time, which is sound
	// because those two are the only mutations the seam has.
	town     TownScreen
	townList *Picker

	// save, list and load are the save/load seam, and menuList, menuBack,
	// loadList, loadBack and saves are the two screens over them. Nothing above
	// this comment concerns either screen, so they are APPENDED here for the
	// reason the town block above them was, and every existing field keeps
	// exactly the meaning it had.
	//
	// ALL THREE MAY BE NIL, and a front end that installs none of them has
	// mission rows with the decoded store gates. Town keeps its researched
	// all-enabled population, and the action reports an unavailable seam rather
	// than making a nil call.
	//
	// saves is what list() last answered, held BESIDE loadList because the
	// picker carries a row's text and not its name: the label is what is
	// drawn and the name is what the far side reads back by, and the two are
	// different strings.
	saveGame        SaveGame
	saveDialogSeams SaveDialogSeams
	saveDialog      *saveDialog
	saveList        SaveList
	loadGame        LoadGame
	menuList        *Picker
	menuBack        Screen
	loadList        *Picker
	loadUI          loadWindow
	loadBack        Screen
	saves           []SaveEntry

	// These fields are the in-game menu's own state: which panel and nested page
	// are up, the session projection, and the decoded gates as they answered
	// when the root was built.
	//
	// THE ROW LIST IS NOT STORED. It is derived by menuRows(),
	// so what a row MEANS and what menuList DRAWS cannot come to disagree — and
	// so this struct holds no sequence mapping a row to what it does, which is
	// the shape DD14 refuses.
	//
	// The predicates are cached rather than re-asked because they read the save
	// store, and a row that changed under the player's hand between the frame
	// he aimed at and the frame he clicked would be picked wrong once.
	//
	// The surface is derived from menuBack at open time rather than read from
	// it at every use: menuBack is where closing returns to, and the surface is
	// which panel was built, and those are different questions.
	menuSurface gameMenuSurface
	menuPage    gameMenuPage
	// menuPress is the menu buttons' press latch (MENU-116).
	menuPress           buttonLatch
	menuContext         GameMenuContext
	menuCanSave         bool
	menuCanLoad         bool
	menuCanSound        bool
	menuTips            GameMenuTipsSource
	setMenuTips         GameMenuTipsSink
	menuSound           GameMenuSoundSource
	setMenuSound        GameMenuSoundSink
	gameOptions         GameOptionControls
	soundOptions        SoundOptionControls
	soundPointer        soundOptionPointer
	tooltip             *tooltipController
	persistTooltipDelay func(int) error
	menuExit            bool

	// docSrc, docArt, docFont, docPanel and docBack are the campaign documents
	// panel's whole seam and state. Nothing above this comment concerns that
	// screen, so they are APPENDED here for the reason the two blocks above
	// them were, and every existing field keeps exactly the meaning it had.
	//
	// docSrc IS ASKED AT EVERY OPEN and the panel is rebuilt from what it
	// answers. The collection grows as the campaign advances, so a panel
	// built once when the front end was assembled would show the collection
	// as it stood then; the original re-reads its own list on the same
	// event, which is the node the panel's entry point posts.
	//
	// ALL FOUR MAY BE NIL, and a front end that installs none of them has
	// the panel's entry point answering that there is nothing to show —
	// never a nil call. That is what keeps every existing test in this
	// package, none of which installs a document source, driving unchanged.
	//
	// docBack is the screen the panel was armed from, held for the reason
	// menuBack and loadBack are: closeDocuments must return where
	// openDocuments was called from. Unlike those two it has one
	// production call site today, which arms it with ScreenMap.
	docSrc   DocumentSource
	docArt   *DocumentPanelArt
	docFont  *text.Font
	docPanel *documentPanel
	docBack  Screen

	// modUI is the state of the screens mods declare (modscreens.go).
	modUI modScreenState

	// selfNotice says the notice standing over the map was raised by THIS tier
	// and not by the far side (round 3).
	//
	// IT DECIDES WHO CLOSES THE NOTICE, which is the whole reason it exists.
	// advanceNotice hands every dismissal to the MapAdvance seam, and the
	// driver behind that seam answers NoticeStay for a notice it did not open
	// — its own mission record's open flag is false — so nothing would ever
	// clear one, and the player would be held by a box that takes their
	// dismissal and does not go away. A notice this tier raised is therefore
	// closed here, without consulting the seam at all: the driver holds no
	// page for it and cannot be asked for one.
	//
	// Its zero value is "the far side's", which is what every notice in this
	// build was before this story.
	selfNotice        bool
	completedCutscene string // presentation request retained across leaveMap
}

// showTextNotice opens a notice over the map carrying s, owned by this tier
// (round 3). It reports whether one was raised.
//
// IT IS THE `Pause` KEY'S WHOLE DESTINATION. `keyboard.tsv` row 20 gives that
// key "show modal text main.txt[119]", and this build already has the modal
// text window the row names: the dialogue notice, which draws the words, draws
// the button word `main.txt[77]` the shipped line itself tells the player to
// click, and holds the world through popupOpen. So the key reaches the window
// this build already has rather than a second display path built for it.
//
// IT REFUSES WHILE ANY NOTICE IS OPEN, which is the row's own "repeat blocked
// by modal state" and also what stops this tier from replacing a mission's
// dialogue with its own.
//
// AN EMPTY STRING RAISES NOTHING. That is the case of an install whose
// `main.txt` does not state the line at all; a notice with no words is a box
// the player must dismiss to learn nothing.
func (f *flow) showTextNotice(s string) bool {
	if s == "" || f.screen != ScreenMap || f.viewer == nil || f.viewer.NoticeOpen() {
		return false
	}
	f.viewer.SetNotice(s, NoticeDialogue)
	f.selfNotice = true
	return true
}

// showHelp opens the help panel over the map carrying s (MENU-051). Like the
// Pause key's notice it is owned by this tier, so its OK button and Esc close
// it without asking the mission driver. It refuses while any notice is open.
func (f *flow) showHelp(s string) bool {
	if f.screen != ScreenMap || f.viewer == nil || !f.viewer.OpenHelp(s) {
		return false
	}
	f.selfNotice = true
	return true
}

// openDocuments builds the campaign documents panel over what the source
// answers NOW and shows it, remembering the screen it was armed from.
//
// IT REFUSES AN EMPTY COLLECTION rather than showing an empty sheet (story
// 1035 B2). The original's own entry point is a mission-side control, and a
// player who has collected nothing has nothing to page through; a screen that
// can only be left again is a screen that should not have opened. The refusal
// is reported so the caller can leave the player where he was.
func (f *flow) openDocuments(back Screen) bool {
	if f.docSrc == nil {
		return false
	}
	pages := f.docSrc.Documents()
	if len(pages) == 0 {
		return false
	}
	f.docPanel = newDocumentPanel(f.docArt, f.docFont, pages)
	f.docBack = back
	f.setScreen(ScreenDocuments)
	f.msg = ""
	f.setDocUp(true)
	return true
}

// closeDocuments returns to the screen the panel was armed from and drops the
// panel.
//
// THE PANEL IS DROPPED RATHER THAN KEPT for the reason openDocuments rebuilds
// it: a kept panel is a stale collection, and a kept page position would be a
// second piece of state to reason about. The original's OK arm posts 0x445 and
// tears its own window down (MENU-DOC-009).
func (f *flow) closeDocuments() {
	f.docPanel = nil
	f.setScreen(f.docBack)
	f.msg = ""
	f.setDocUp(false)
}

func (f *flow) setDocUp(up bool) {
	if f.viewer != nil {
		f.viewer.docUp = up
	}
}

// ChargenEntry is a generation screen ready to be shown: the model to drive and
// the callback a legal confirm reaches. It is the exact pair App.OpenChargen
// takes, named as a type so a ChargenGate can answer both at once — or answer
// nil, which is the whole of "this row does not want generation".
type ChargenEntry struct {
	Model *Chargen
	Begin func(ChargenResult) (MapOpener, error)
}

// ChargenGate answers whether picker row i opens the generation screen before
// its map, and with what (0140).
//
// IT IS ASKED WHEN A ROW IS CHOSEN FROM THE LIST, which in this front-end is
// what -picker's own debug NEW GAME route leads to — so a gate is a statement
// about starting something, not about entering a map. Nothing else in this
// package consults it: the mission door and the won-mission advance both enter
// a map screen without passing here, deliberately (see flow.chargenGate's own
// field comment).
//
// A NIL RETURN IS THE ORDINARY ROW and not an error: the picker loads it and
// shows it exactly as it always has. That is the shape rather than a (entry,
// bool) pair because there is nothing for a caller to distinguish — "no entry"
// and "no generation" are one answer.
//
// IT TAKES A ROW INDEX AND NOTHING ELSE. The gate is installed by the wiring
// tier, which built the row list and therefore already knows what row i is; a
// richer argument here would be this package describing rows it deliberately
// knows nothing about beyond their text and whether they can be chosen.
type ChargenGate func(row int) *ChargenEntry

// NewGameChargen answers what the menu's NEW GAME arms directly: a fresh
// generation screen for the front end's own default campaign entry, ready
// exactly as a ChargenGate's answer is, or nil to fall back to the map
// picker (owner reversal; SetNewGameChargen's own doc on App carries why the
// picker is the fallback and not the default).
//
// IT TAKES NO ROW, unlike ChargenGate: NEW GAME opens no list before asking
// it, so there is exactly one destination to arm rather than one per row.
//
// A FRESH VALUE PER CALL, deliberately, on newGameChargen's own field
// comment's reasoning: a player who backs out of generating a character and
// presses NEW GAME again is starting again, not resuming a spread he
// abandoned.
type NewGameChargen func() *ChargenEntry

// NewGameDirect answers the opener NEW GAME enters straight away, with no
// generation screen, or nil when there is none to enter. It is called fresh on
// every press.
type NewGameDirect func() MapOpener

// newFlow starts the front-end on the MENU screen.
//
// That is worth stating rather than leaving to the zero value: the application
// opening on the main menu is a requirement, and an earlier draft of this
// story's contract assumed the picker was the root screen. Nothing else in the
// automated suite would notice if it silently opened on the wrong one.
func newFlow(picker *Picker, load MapLoader) *flow {
	return &flow{screen: ScreenMenu, picker: picker, load: load, words: AuthoredWords(),
		cursor: NewCursorManager(), modUI: modScreenState{open: -1}}
}

func defaultCadenceRung() int {
	return terrain.DefaultCadenceRung
}

// cadencePreference is the normal rung to use at the next map entry. The
// explicit set bit keeps rung zero usable while preserving the old default for
// tests and embedders that construct a flow directly.
func (f *flow) cadencePreference() int {
	if f == nil || !f.preferredRungSet {
		return defaultCadenceRung()
	}
	return terrain.ClampCadenceRung(f.preferredRung)
}

// setCadencePreference installs the process-local opening rung and its optional
// persistence sink. It selects no owner-loop arm and writes nothing merely by
// being installed; the sink runs only after a later keyboard or menu speed step
// changes the normal rung.
func (f *flow) setCadencePreference(rung int, persist func(int)) {
	if f == nil {
		return
	}
	f.preferredRung = terrain.ClampCadenceRung(rung)
	f.preferredRungSet = true
	f.persistRung = persist
}

func (f *flow) surfaceTransition(exit string) {
	f.cursor.SetCursor("wait")
	f.cursor.SetCursor(exit)
}

// screenExitCursor is the cursor each of this build's screens ends its own
// surface transition on, and whether that screen runs one at all. It is the
// whole of the screen-to-cursor mapping: one table, read by setScreen and by
// nothing else.
//
// THE MAPPING IS AUTHORED where this build's screen set does not correspond
// one to one with the original's seventeen transition routines. Menu, chargen,
// map and town are the four with a decoded counterpart (MENU-CURSOR-046 for
// the menu's `select` exit; TOWN-372 for the rest, at Medium on which routine
// is which surface). Picker, load, the in-game menu and the documents panel
// run no transition,
// which stands in for TOWN-372's five-routine "sets no cursor" family without
// claiming correspondence to any one of the five: B2's persistence rule is
// what makes that safe, since the previous screen's cursor stays current.
func screenExitCursor(s Screen) (string, bool) {
	switch s {
	case ScreenMenu:
		return "select", true
	case ScreenChargen, ScreenTown, ScreenMap, ScreenEnding, ScreenCutsceneLibrary, ScreenCredits:
		return "default", true
	}
	return "", false
}

// setScreen is THE ONE SITE THAT ASSIGNS f.screen, and that is what makes the
// surface transition a property of arriving at a screen rather than of the
// path taken to it.
//
// WHY IT IS ONE SITE. Each transition used to run its own cursor pair beside
// its own assignment, and three paths assigned f.screen without one: Escape
// from chargen restores f.chargenBack, Escape from the load window restores
// f.loadBack, and the in-game menu's exit restores f.menuBack (save.go). The
// main menu reached by backing out of generation then drew `default` instead
// of `select`, because toMenu was not on that path. A per-path pair is a rule
// every future path has to remember; a table read at the assignment is a rule
// no path can miss. cursorlifecycle_test.go asserts by source scan that this
// stays the only production assignment.
//
// A SCREEN ARRIVED AT FROM ITSELF still runs its pair, which is the original's
// own shape: the transition is what the routine does, not a diff it takes.
func (f *flow) setScreen(s Screen) {
	if s == ScreenMap {
		f.ending = EndingView{}
		f.endingPage, f.endingTop, f.endingFocus, f.endingPress = 0, 0, 0, -1
	}
	if exit, ok := screenExitCursor(s); ok {
		f.surfaceTransition(exit)
	}
	f.screen = s
}

// pointerWanted reports whether the system pointer must be hidden on the frame
// being composed: exactly when this build draws a pointer of its own.
//
// IT IS ONE ANSWER FOR THE WHOLE WINDOW. The map screen and every other screen
// draw their pointers from two different places -- Viewer.Draw's attack
// pointer and App.drawCursor's manager picture -- and each used to keep its
// own "last told" cache of one piece of global window state. Neither saw the
// other's writes, so entering the map left the engine on Hidden with nothing
// drawn: no pointer at all on the primary gameplay surface. Taking the wanted
// state from one function, once per frame, whatever branch Draw takes, makes
// that state unreachable rather than unlikely, which is pointerModeChange's
// own argument (cursor.go) applied one level up.
//
// THE MAP BRANCH NOW READS THREE ANSWERS, NOT ONE (1031 B5): the attack
// pointer, mapCursorPresent and the drag machine's held-item picture, in the
// SAME order Viewer.drawPointer draws them in, first true wins. Before this
// story the map branch read only the attack pointer, so hovering outside
// attack mode left the system pointer showing over a map that drew none of
// its own -- the defect this story's own Result closes. A drag is checked
// here too: dragItemPresent already drew its own picture on every map screen
// before this story, under whatever the system pointer happened to be doing,
// and folding it into this same answer is what makes "hides one and draws
// neither" unreachable for it as well.
func (f *flow) pointerWanted() bool {
	if f.mapShowing() {
		if f.viewer == nil {
			return false
		}
		if _, _, shown := f.viewer.attackPointerPresent(); shown {
			return true
		}
		if _, _, shown := f.viewer.mapCursorPresent(); shown {
			return true
		}
		_, _, shown := f.viewer.dragItemPresent()
		return shown
	}
	_, _, ok := f.cursor.Current()
	return ok
}

// syncPointerMode brings the shared cache to this frame's wanted state and
// reports whether the engine has to be told. The engine call itself is
// App.applyPointerMode's, so what a test with no window asserts is this: the
// cache equals pointerWanted on every frame, on every screen.
func (f *flow) syncPointerMode() bool {
	return f.cursor.SetPointerHidden(f.pointerWanted())
}

// cursorPresent is what App.Draw paints for the front-end's own cursor: the
// picture, the frame pixel its top-left goes at, and whether one is drawn at
// all. tip is the frame pixel the pointer is on.
//
// THE HOTSPOT IS SUBTRACTED HERE, IN FRAME PIXELS. Each registration carries
// the pixel within its own frame that names the cursor point (SPR16A-CURSOR-046,
// SPR256-CURSOR-046), read out of the executable rather than chosen:
// `default` (5,5), `select` (3,4), `cantput` (38,36). Placing the picture's
// top-left at the tip instead puts the drawn pointer's own tip down-and-right of
// where the click lands, by the hotspot. The original blits the picture at the
// position minus that origin on its 640x480 surface (AI-CURSOR-218), and the
// mission's own pointer subtracts in the same frame space (mapCursorPresent).
func (f *flow) cursorPresent(tip image.Point) (*image.RGBA, image.Point, bool) {
	pic, hot, ok := f.cursor.Current()
	if !ok {
		return nil, image.Point{}, false
	}
	return pic, tip.Sub(hot), true
}

// toMenu shows the main menu (0142's own destination, the game-menu abort
// path, and the picker's own Escape). It is the one surface-transition site
// this package's main menu goes through: `select` on exit rather than
// `default`, the one member of TOWN-372's eleven-routine family that differs
// (MENU-CURSOR-046).
func (f *flow) toMenu() {
	f.setScreen(ScreenMenu)
}

// activateNewGame arms character generation directly when newGameChargen is
// set, and opens the map picker when it is not. It is the only activation the
// flow exposes.
//
// THE DEFAULT IS GENERATION, NOT THE PICKER (owner reversal: launching the
// game must open the main menu and nothing else, and the picker the player
// never asked for must not appear without an explicit debug flag). The
// wiring tier — cmd/againrom — installs newGameChargen for an ordinary
// launch and leaves it nil only under -picker, which is what makes "the
// picker" the debug route rather than the shipped one. newGameChargen
// returning nil, or a nil model inside what it returns, is treated as "no
// generation to arm" and falls back to the picker rather than doing
// nothing — the same degrade-rather-than-panic rule armChargen's own refusal
// already carries.
//
// There is deliberately no per-button command table here. Only one brooch button
// has a bound action; a release on any of the other seven is consumed by the menu
// screen and changes nothing. A dispatch table "ready for the other buttons"
// would be the shape that invites binding them, and what they do is not decoded.
func (f *flow) activateNewGame() {
	if f.screen != ScreenMenu {
		return
	}
	if f.newGameChargen != nil {
		if e := f.newGameChargen(); e != nil && f.armChargen(e.Model, e.Begin, ScreenMenu) {
			return
		}
	}
	f.setScreen(ScreenPicker)
	f.msg = ""
}

// choose loads the picker's selected map and shows it.
//
// On failure it stays in the picker, records why, and marks that row unusable so
// the same choice cannot be made again — it never exits and never panics. A map
// can list cleanly and still fail here, because listing reads only the map's own
// metadata while showing it decodes the whole thing.
//
// THE GATE IS ASKED BEFORE THE LOADER (0140), and that order is the whole
// mechanism: a row that opens generation must not be decoded first, because the
// map it decodes to depends on a character that does not exist yet. A gate
// answering non-nil therefore RETURNS — nothing below runs, f.load is never
// called for that row, and the map is opened later, by the begin callback, out
// of the confirmed result.
//
// A GATE THAT ANSWERS AN ENTRY WITH NO MODEL IS REFUSED ON THE PICKER, on the
// message line, with the row left choosable. It is the same shape a loader
// failure already takes here — this method never exits and never panics — and
// the row is NOT marked unusable, because an entry the wiring tier could not
// build says nothing about whether the map behind it would decode.
func (f *flow) choose() {
	if f.screen != ScreenPicker {
		return
	}
	i, ok := f.picker.Choose()
	if !ok {
		return
	}
	if f.chargenGate != nil {
		if e := f.chargenGate(i); e != nil {
			if !f.armChargen(e.Model, e.Begin, ScreenPicker) {
				f.msg = "cannot generate a character for this row"
			}
			return
		}
	}
	if f.load == nil {
		f.msg = "cannot load maps: no loader configured"
		f.picker.SetUnusable(i)
		return
	}
	v, tick, order, cadence, affect, advance, attack, grab, stance, march, err := f.load(i)
	if err != nil {
		f.msg = err.Error()
		f.picker.SetUnusable(i)
		return
	}
	f.enter(v, tick, order, cadence, affect, advance, attack, grab, stance, march)
}

// armChargen shows the generation screen over the model c, with begin as the
// callback a legal confirm reaches and back as the screen Escape returns to. It
// reports whether the screen was armed (0140).
//
// IT IS THE ONE ASSIGNMENT OF f.chargen IN THIS PACKAGE, and that is why it
// exists rather than three lines written twice. Generation has three doors
// now — App.OpenChargen before Run (an explicit -mission's own direct entry),
// NEW GAME's own direct arm mid-run, and a gated picker row mid-run under
// -picker — and the rule every method on the chargen arm is written against
// is that f.chargen is not nil once f.screen is ScreenChargen. One assignment,
// one nil check, and the rule holds for all three doors and for any door
// added later; separate copies would be separate chances for one of them to
// arm the screen over a nil model and turn the next tick into the panic this
// package's whole design refuses to have.
//
// begin IS NOT CHECKED. A nil callback is a screen that can be driven and
// refuses to confirm, which stepChargen already reports on the message line as
// the recoverable state it is — unlike a nil model, which nothing on the screen
// can survive reading.
//
// THE MESSAGE LINE IS CLEARED. Whatever the previous screen last had to say —
// a load failure on the picker, a refused confirm — is about that screen and
// not about the character now being generated.
func (f *flow) armChargen(c *Chargen, begin func(ChargenResult) (MapOpener, error), back Screen) bool {
	if c == nil {
		return false
	}
	f.chargen = c
	f.chargenBegin = begin
	f.chargenBack = back
	// wait/default (B4): TOWN-372 names R0909, pushing music\chrgen.wav, as
	// this exact surface — the character-generation screen — among its ten
	// wait/default routines.
	f.setScreen(ScreenChargen)
	f.msg = ""
	return true
}

// enter shows a loaded map screen: it adopts the viewer and every seam, opens
// the cadence ladder and puts the viewer into command mode.
//
// IT IS FACTORED OUT OF choose RATHER THAN COPIED, and that is the whole
// reason it exists. The mission door opens a map screen no picker row names,
// and everything below happens to be load-bearing rather than incidental:
// the cadence rung establishes the ladder at the period the world's clock
// actually started at — the ladder's zero value is its SLOWEST rung, so a
// door that skipped this would make the first speed key slam a 62 ms tick
// down to the slowest one — and command mode is what makes the map screen
// box-select and take its pan on Shift rather than behaving like the
// developer viewer. A second copy of this statement would be a second chance
// for a door to differ from a pick in a way nobody could see until they
// pressed a key.
func (f *flow) enter(v *Viewer, tick MapTick, order MapOrder, cadence MapCadence,
	affect MapAffect, advance MapAdvance, attack MapAttack, grab MapGrab,
	stance MapStance, march MapMarch) {
	f.resetTimedAutosave()
	v.entryCampaignStart = f.screen == ScreenChargen && v.missionCutscene > 0
	f.viewer = v
	v.beginSoundEntry()
	// THE WORDS TRAVEL WITH THE VIEWER, on the one statement the map screen is
	// entered on. A second place that told a viewer its words would be a second
	// chance for a door to open a map whose notices speak a different language
	// from the menu over them.
	v.SetWords(f.words)
	v.SetDialogFrame(f.menuArt)
	if f.helpScroll != nil {
		v.setHelpScrollArt(f.helpScroll.frames)
	}
	v.setFailureLoadCheck(func() bool {
		return f.loadGame != nil && f.saveList != nil && len(f.saveList()) != 0
	})
	f.tick, f.order, f.cadence, f.affect, f.advance, f.attack, f.grab, f.stance, f.march =
		tick, order, cadence, affect, advance, attack, grab, stance, march
	// The cadence this screen starts on is the last selected normal rung. A
	// missing preference answers the old shipped default. The owner-loop arm is
	// deliberately reset to paced on every entry and is never restored from the
	// preference beside it.
	defaultRung := defaultCadenceRung()
	f.rung, f.stopped, f.unpaced = f.cadencePreference(), false, false
	// What the far side is holding, established on the SAME STATEMENT as the
	// ladder rather than left at a zero value. Both consumers are constructed
	// at the shipped default. A different preference therefore compares unequal
	// and crosses once below; the default retains the historical no-call entry
	// path.
	f.farRung, f.farStopped, f.farUnpaced = defaultRung, false, false
	// The viewer is now the front-end's map screen, so its left drag draws a
	// selection rectangle and Shift takes the pan. It is set HERE, beside the
	// two halves of the seam and on the same transition, so a viewer in command
	// mode and a front-end holding a seam are one state rather than two kept in
	// step; and it is set unconditionally, because a loader that put nothing
	// under the map still hands back a screen whose gestures are the map
	// screen's — a world holding no unit must not silently lose its pan
	// either way.
	if v != nil {
		v.commandMode = true
		// THE SHARED MANAGER TRAVELS WITH THE VIEWER, on the same statement as
		// commandMode above, so the animated attack pointer (B3) reads the
		// SAME current-cursor state every other screen this session shows
		// does, rather than a second manager nobody set (docs/1030-cursor-
		// lifecycle B2 "one manager, one current cursor").
		v.SetCursorManager(f.cursor)
		// AND THE POINTER IS THIS App's TO COMPOSE from here on: it paints the
		// in-game menu over the frame this viewer returns, so the viewer must not
		// put a pointer inside that frame (Viewer.DeferPointer, App.Draw).
		v.DeferPointer(true)
		if f.adoptRestoredApplication(v) {
			// The prepared save owns the loaded cadence; no profile write.
		} else if f.rung == defaultRung {
			// The owner-loop arm is map/session state. A reused viewer must not
			// inherit it even when the persisted normal rung needs no far-side
			// re-rate.
			v.SetCadenceMode(terrain.CadencePeriod(f.rung), false, false)
		} else {
			// The loader built both readers at the shipped default. Cross the
			// restored normal rung once before the first map frame.
			f.syncCadence(false)
		}
	}
	// wait/default (B4): TOWN-372's R1317, pushing music\map.wav, is this
	// exact surface among its ten wait/default routines.
	f.setScreen(ScreenMap)
	f.msg = ""
}

// noticeOpen reports whether the open map screen is showing a notice that takes
// input. It is false on every other screen, which is what makes "a notice only
// ever intercepts the map screen's keys" a property of this one test.
func (f *flow) noticeOpen() bool {
	return f.screen == ScreenMap && f.viewer != nil && f.viewer.NoticeOpen()
}

func (f *flow) advanceNotice(actions ...NoticeAction) {
	if !f.noticeOpen() {
		return
	}
	action := NoticeAdvance
	if len(actions) > 0 {
		action = actions[0]
	}
	if f.advance == nil {
		f.viewer.ClearNotice()
		return
	}
	// A NOTICE THIS TIER RAISED closes here and the seam is never asked (round
	// 3). The driver behind MapAdvance answers NoticeStay for a notice its own
	// mission record does not know about, so consulting it would leave the box
	// up for every dismissal the player made. See the selfNotice field's own
	// doc.
	if f.selfNotice {
		f.selfNotice = false
		f.viewer.ClearNotice()
		return
	}
	f.takeNoticeAction(action)
}

// takeNoticeAction navigates one decision already admitted by its surface.
// The open notice and the later End Quest Victory row deliberately converge
// here, so both reach the same MapAdvance/continuity completion boundary.
func (f *flow) takeNoticeAction(action NoticeAction) {
	if action == NoticeLoadGame && f.viewer != nil && f.viewer.noticeLayout().SecondaryDisabled {
		return
	}
	dest, msg, open := f.advance(action)
	// Only an accepted completion navigates to these destinations. Continue,
	// a rejected Victory, defeat and ordinary dialogue leave no movie request.
	// Capture the completed viewer before a successor replaces it.
	switch dest {
	case NoticeToEnding, NoticeToMission, NoticeToTown, NoticeToMapList:
		if f.viewer != nil {
			f.completedCutscene = f.viewer.completionCutscene
		}
	}
	f.goNoticeDest(dest, msg, open)
}

// goNoticeDest takes the screen change an advance answered with.
func (f *flow) goNoticeDest(dest NoticeDest, msg string, open MapOpener) {
	switch dest {
	case NoticeStay:
		if msg != "" {
			f.msg = msg
			if f.viewer != nil {
				f.viewer.PostMessageUnlessNewest(msg, MessageWhite, 5*time.Second)
			}
		}
	case NoticeToEnding:
		f.leaveMap()
		if !f.showCampaignEnding() {
			f.toMenu()
		}
	case NoticeToLoad:
		// Do not close the failure panel or release its seams. Cancellation
		// and failed loads must return to the same terminal state.
		f.openLoad(ScreenMap)
	case NoticeToMenu:
		f.leaveMap()
		f.toMenu()
		f.msg = ""
	case NoticeToMapList:
		f.leaveMap()
		f.setScreen(ScreenPicker)
		f.msg = msg
	case NoticeToMission:
		f.leaveMap()
		if open == nil {
			f.setScreen(ScreenPicker)
			f.msg = msg
			return
		}
		v, tick, order, cadence, affect, advance, attack, grab, stance, march, err := open()
		if err != nil {
			f.setScreen(ScreenPicker)
			f.msg = err.Error()
			return
		}
		f.enter(v, tick, order, cadence, affect, advance, attack, grab, stance, march)
	case NoticeToTown:
		// The map screen is torn down through the SAME leaveMap the other three
		// arms use, and then the town is shown — with the seam's own sentence
		// written after the teardown, because leaveMap clears the message field
		// the town draws.
		//
		// A FRONT END WITH NO TOWN LANDS ON THE MAP LIST holding the same
		// sentence, rather than on a torn-down map screen. That is the arm
		// above's own answer to a nil opener applied to a nil seam, and it
		// cannot be reached through the shipped front end — which installs
		// a town before it can ever produce this destination — so it is
		// written rather than assumed.
		f.leaveMap()
		if !f.showTown(msg) {
			f.setScreen(ScreenPicker)
			f.msg = msg
		}
	}
}

// finishContinuedMission takes the campaign End Quest panel's Victory row.
// Its enable gate was projected when the menu opened; the far-side latch is
// still the authority and can answer NoticeStay if the state changed.
func (f *flow) finishContinuedMission() {
	if f.screen != ScreenGameMenu || f.menuBack != ScreenMap || f.advance == nil {
		return
	}
	f.setMenuUp(false)
	f.setScreen(ScreenMap)
	f.menuPage = gameMenuRoot
	f.menuList = nil
	f.menuContext = GameMenuContext{}
	f.msg = ""
	f.takeNoticeAction(NoticeVictory)
}

// leaveMap drops everything the map screen was holding: the viewer, all five
// seams, the command mode and the message.
//
// It is factored out of escape for advanceNotice's sake, and the factoring is
// what keeps "a transition cannot leave the front-end holding one seam without
// the others" true of BOTH ways off this screen rather than of the one that
// existed first.
func (f *flow) leaveMap() {
	// Cleared on the statement that drops the seam, for the reason enter sets
	// it there: the mode is the map screen's, so a viewer that outlives the
	// screen — one a caller still holds a pointer to — is a plain terrain
	// viewer again and pans by a plain drag.
	if f.viewer != nil {
		f.viewer.DestroyAudio()
		// A viewer retained by its caller after map teardown becomes an ordinary
		// paced standalone viewer again, just as commandMode is cleared below.
		f.viewer.SetCadenceMode(f.viewer.anim.Period(), false, false)
		f.viewer.commandMode = false
	}
	f.viewer = nil
	f.tick, f.order, f.cadence, f.affect, f.advance, f.attack, f.grab, f.stance, f.march =
		nil, nil, nil, nil, nil, nil, nil, nil, nil
	f.msg = ""
	// Dropped with the viewer that was carrying the notice: a flag left set
	// across a teardown would make the NEXT map's first mission dialogue close
	// on one press without paging (round 3).
	f.selfNotice = false
}

// stepCadence resolves one frame's five cadence keys and, when they moved
// the cadence or requested a phase reset, writes the result to BOTH
// consumers of the rate.
//
// The three arrive as bare bools rather than as the input snapshot, which is the
// same rule the seam types keep: this method decides a cadence and needs to know
// nothing else about the frame that produced it.
//
// THE KEYS STEP THE LADDER BY ONE RUNG, and what a rung is lives in
// pkg/render/terrain and nowhere else. Across the nine rungs the game itself
// offers, that is the original's own control: its `+`/`-` step the speed
// index by one (TERR-ANIM-008). Past either end of that set the ladder
// continues into OUR extension, doubling and halving as 0041's did, so
// stepping by one is still a control at 1024 ticks a second.
//
// This method therefore holds NO arithmetic of its own — no table, no division,
// no bound. It adds one, subtracts one, and clamps through
// terrain.ClampCadenceRung, the very clamp CadencePeriod is taken over, so the
// control cannot stop one step short of what the ladder accepts and a press at
// either end computes the period already in force.
//
// ALL THREE ARE RESOLVED INTO ONE CADENCE before anything is written, so a frame
// that saw two keys writes once rather than twice; and NOTHING IS WRITTEN WHEN
// NOTHING CHANGED, which is what makes "on a change, never per tick" a property
// of this comparison rather than of how often the caller calls.
//
// THE WRITE IS ONE STATEMENT, and that statement is the coupling this story
// exists to decide. SetPeriod takes the period, adopts it, and REPORTS what
// it adopted; the seam is handed that reported value. So there is one period
// value and two readers of it, and no path through this package re-rates the
// water without re-rating the world in the same expression. Two independent
// tickers drifting apart is not something a later story has to remember not
// to do — there is nowhere to write one of them alone.
//
// The period is computed HERE and only here. Before this story each consumer
// was handed a rate and divided it for itself, so one number became two
// quotients — and where the game's own arithmetic truncates, the quotient
// the far sides computed was not the one the world's clock had opened
// holding.
//
// The stop reaches the seam and NOT the viewer: the viewer is the standalone
// viewer's type too, and the player's own pause still leaves the map's
// ambient animation running (and 0077's disclosed narrowing of it — a
// POPUP now holds that animation, and the pause key does not).
//
// A map screen the loader put nothing under holds neither consumer, and then
// neither is written — the front-end's own rate still moves, so the ladder
// is where the player left it, and there is simply nothing on the far side
// of it. THE THREE KEYS ARE INERT WHILE A POPUP IS OPEN. The gate stands
// HERE, beside the state it protects, rather than in the input snapshot: the
// snapshot is a record of what the player pressed and stays true, and the
// popup's own three inputs keep the single interception point they already
// have.
//
// THIS IS NO LONGER THE ONLY GATE ON THE MAP ARM. Until 0077 the camera, the
// selection, the orders, the blows and the two diagnostics were exactly what
// 0066 left them; a popup now takes all of them, at one early return on that
// arm. This gate stays here rather than folding into that one, because the
// arm's gate stands AFTER the advance and this must stand before it: a frame
// that pressed pause under a popup must not write the setting the dismissal
// will restore.
//
// The pause key is the case that decides it. Pressed under a notice the world
// is already stopped, so a toggle has no effect anyone can see — and it would
// then take effect on DISMISSAL, unannounced, leaving a player who tapped Space
// during a cutscene on a paused world with nothing to connect the two. A key
// that does nothing is better than a key that arms something.
func (f *flow) stepCadence(pause, faster, slower, unpaced, paced bool) {
	reset := false
	if !f.popupOpen() {
		oldRung := f.rung
		rung, stopped, ownerUnpaced := oldRung, f.stopped, f.unpaced
		if faster {
			rung++
		}
		if slower {
			rung--
		}
		if pause {
			stopped = !stopped
		}
		// These two commands SET and CLEAR one bit; they do not toggle it. A
		// snapshot carrying both has lost the host event order, so the two cancel
		// just as simultaneous faster/slower do above rather than inventing one.
		if unpaced != paced {
			ownerUnpaced = unpaced
			reset = paced
		}
		f.rung, f.stopped, f.unpaced = terrain.ClampCadenceRung(rung), stopped, ownerUnpaced
		if f.rung != oldRung {
			// The stored preference is the normal rung even while the separate
			// unpaced arm is selected. Ctrl+plus/minus themselves never reach
			// this write because neither moves rung.
			f.rememberCadenceRung(f.rung)
		}
	}
	f.syncCadence(reset)
}

// THE SUSPENSION PREDICATE IS NOW popupOpen, IN popup.go. What was written
// here about the ruling, and the claim that now backs it, is written there.

// syncCadence tells both consumers the cadence actually in force, and only
// when it changed from what they were last told.
//
// THE STOP IN FORCE IS A DISJUNCTION, and this is the only place the two causes
// meet: the switch the player set, OR a notice standing over the map. The
// player's own field is never written from here, so dismissing a notice restores
// exactly what the player had — including a pause they set before it opened —
// without a restore step existing to be got wrong.
//
// THE COMPARISON IS AGAINST WHAT CROSSED, NOT AGAINST WHAT WAS ASKED FOR. This
// is the correction 0073 makes to the shipped write rule: the old comparison
// against the player's fields was sound while the player was the only cause,
// and a notice opening moves the answer without moving those fields, so that
// comparison would have found nothing to write and the suspension would never
// have crossed. Nothing about the write LOOKS wrong; only the meaning of the
// compared value does.
//
// It is the ONE write site. A second one for the suspension would be a second
// writer of a seam whose whole point is that one number has two readers.
//
// The far-side record is updated even where there is no far side to tell, so a
// map screen the loader put nothing under keeps a meaningful comparison rather
// than a conditional memory; there is simply nothing to hand it to.
func (f *flow) syncCadence(reset bool) {
	if f.viewer != nil {
		label := f.pauseLabel()
		if f.viewer.playerPauseLabel != label {
			f.viewer.playerPauseLabel = label
			f.viewer.messages.serial++
		}
		f.viewer.setPlayerPaused(f.stopped)
	}
	rung, stopped, unpaced := f.rung, f.stopped || f.popupOpen(), f.unpaced
	if rung == f.farRung && stopped == f.farStopped && unpaced == f.farUnpaced && !reset {
		return
	}
	f.farRung, f.farStopped, f.farUnpaced = rung, stopped, unpaced

	if f.viewer == nil {
		return
	}
	period := f.viewer.SetCadenceMode(terrain.CadencePeriod(rung), unpaced, reset)
	if f.cadence != nil {
		f.cadence(period, stopped, unpaced, reset)
	}
}

// escape handles Esc and reports whether the program should exit.
//
// Esc unwinds one screen at a time — map to picker, picker to menu — and only at
// the menu does it exit. Nothing else in the front-end exits the program, which
// is what keeps a mis-click in the map screen from dropping the user out of the
// game.
//
// AN OPEN NOTICE IS ANSWERED BEFORE THIS IS REACHED (AC-9). The interception
// is at the app's own Escape arm rather than here, because Escape is read
// there before any per-screen dispatch and this method is what it dispatches
// to; a test inside here would be a second reading of the same key. So
// "Escape with a notice open advances it and does not leave the map screen"
// is a statement about where this is CALLED FROM, and this method's own
// contract is unchanged.
func (f *flow) escape() (exit bool) {
	switch f.screen {
	case ScreenMap:
		// ESCAPE OPENS THE MINI-MENU AND NO LONGER LEAVES. Leaving is that menu's
		// EXIT row, which runs exactly the two statements this arm used to. The
		// alternative — a second key for the menu, with Escape still leaving —
		// was rejected: the menu's four entries include EXIT, so a menu Escape
		// does not reach is a menu with a redundant row and a player who still
		// leaves his mission by reflex.
		f.openGameMenu(ScreenMap)
		return false
	case ScreenGameMenu:
		if f.menuPage == gameMenuRoot {
			f.closeGameMenu()
		} else {
			f.rebuildGameMenu(gameMenuRoot, 0)
		}
		return false
	case ScreenLoad:
		if f.loadUI.confirm {
			f.loadUI.confirm = false
			f.msg = ""
		} else {
			f.closeLoad()
		}
		return false
	case ScreenSave:
		f.cancelSaveDialog()
		return false
	case ScreenDocuments:
		// ESCAPE IS THIS BUILD'S OWN and not the original's (DIV-304).
		// MENU-DOC-009 gives the panel one way out, the OK button, and
		// says nothing about a key. Escape unwinds one screen at a time
		// everywhere else in this method, and a panel that a stuck
		// pointer cannot leave is worse than a key the original did not
		// have; it reaches the same statement OK does.
		f.closeDocuments()
		return false
	case ScreenPicker:
		f.toMenu()
		f.msg = ""
		return false
	case ScreenChargen:
		// The screen holds no window and no map of its own to tear down —
		// unlike leaveMap, there is nothing here to release — so switching
		// screens is the whole of it.
		//
		// IT RETURNS TO THE SCREEN IT WAS ARMED FROM, not to a constant (0140). A
		// player who reached generation from a picker row instead gets the map
		// list back with his place in it kept: unwinding him two screens for
		// backing out of one would lose the row he was on, and the picker is as
		// intact behind this screen as the menu is.
		f.setScreen(f.chargenBack)
		f.msg = ""
		return false
	case ScreenTown:
		if f.advanceTownDialogue() {
			return false
		}
		// ESCAPE UNWINDS A ROOM AT A TIME AND THEN THE TOWN, which is the same
		// one-screen-at-a-time rule this method has always had, applied one level
		// down: the town is five rooms behind one Screen value, so the far side
		// gets first refusal on the press and only answers false when it is
		// already at the square.
		//
		// THE TOWN IS LEFT FOR THE MENU AND NOT FOR THE MAP LIST. The map list is
		// where a mission was picked from, and the town is not a mission; the menu
		// is the screen the town sits under exactly as the map list does.
		//
		// THE SEAM IS NOT DROPPED. town outlives the screen deliberately
		// (see its own field comment): the player may leave the town and a
		// mission may still end into it.
		if f.town != nil && f.town.Back() {
			f.refreshTown()
			f.msg = ""
			return false
		}
		// AT THE SQUARE IT OPENS THE MINI-MENU, which is the map screen's own
		// change applied here: the room unwind keeps first refusal on the press,
		// and leaving the town for the menu is now that menu's EXIT row.
		f.openGameMenu(ScreenTown)
		return false
	case ScreenEnding:
		f.endingBack()
		return false
	case ScreenMod:
		f.closeModScreen()
		return false
	default:
		return true
	}
}

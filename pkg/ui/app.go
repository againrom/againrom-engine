package ui

import (
	"errors"
	"fmt"
	"image"
	"image/color"
	"runtime"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/vector"

	"againrom/pkg/audio"
	"againrom/pkg/render/frame"
	"againrom/pkg/render/menu"
	"againrom/pkg/render/text"
	"againrom/pkg/video"
)

// Window geometry for the front-end.
//
// MenuWindowW x MenuWindowH is an INTEGER multiple of the virtual frame, and
// that is the part that matters: at an integer scale nearest-neighbour
// reproduces the brooch art pixel for pixel, while any other factor resamples
// the one screen this project renders exactly. The viewer's own 1024x768 default
// would put the frame at 1.6x, i.e. uneven pixel doubling.
//
// The multiple is 2. One is the smallest window that shows the whole frame, and
// picking it made the game open at a quarter of a 1080p desktop, which is too
// small to read a map list in. Three would need 1440 lines and would not fit
// under a 1080-line desktop with its title bar; two needs 960 and does.
//
// IT IS NO LONGER THE SIZE THE GAME OPENS AT (owner) — startupWindow is,
// and this pair is what it falls back to. The window is resizable and the
// letterbox handles every other size; nothing about the mapping depends on
// this constant. The mission screen alone expands its logical width.
const (
	MenuWindowScale = 2

	MenuWindowW = frame.W * MenuWindowScale
	MenuWindowH = frame.H * MenuWindowScale
)

// startupWindow uses the monitor's size, or a movable menu-sized window when
// the monitor reports no usable extent. Frame.Fit scales the menu uniformly;
// the mission screen expands to the resulting aspect ratio.
func startupWindow(monW, monH int) (w, h int, whole bool) {
	if monW <= 0 || monH <= 0 {
		return MenuWindowW, MenuWindowH, false
	}
	return monW, monH, true
}

func configureStartupWindow(monW, monH int, goos string) {
	w, h, whole := startupWindow(monW, monH)
	// macOS needs native fullscreen to keep the Dock and menu bar off the game.
	// Retain decorations for its windowed return; other desktops use borderless.
	nativeFullscreen := whole && goos == "darwin"
	ebiten.SetWindowDecorated(!whole || nativeFullscreen)
	if whole && !nativeFullscreen {
		ebiten.SetWindowPosition(0, 0)
	}
	ebiten.SetWindowSize(w, h)
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)
	ebiten.SetFullscreen(nativeFullscreen)
}

// monitorSize reports the current monitor's size in the device-independent
// pixels SetWindowSize is spelled in — the pairing Ebitengine's own Size
// documents for "the perfectly fit fullscreen", so a scaled desktop needs no
// arithmetic of ours.
//
// Monitor may answer nil before the main loop starts, and a platform with no
// monitor concept answers a zero size; both arrive at startupWindow as the same
// "nothing usable" and take its fallback.
func monitorSize() (int, int) {
	m := ebiten.Monitor()
	if m == nil {
		return 0, 0
	}
	return m.Size()
}

// appInput is one tick of everything the front-end reads.
//
// Gathering it into a value is what makes the whole dispatch — every screen
// transition, every menu latch event, and the map screen's camera call —
// reachable in a test with no window. Update stays a gatherer with no logic of
// its own.
type appInput struct {
	// AnyKey is used only by the cutscene overlay (VIDEO-036). Other screens
	// continue to consume their named controls rather than arbitrary keys.
	AnyKey  bool
	AnyHeld bool
	Close   bool
	Viewer  Input // the camera input, used only on the map screen

	CursorX, CursorY int
	PrimaryPressed   bool // primary mouse button went down this tick
	PrimaryReleased  bool // ...and came up this tick

	// SecondaryPressed is the secondary mouse button's PRESS EDGE. Its zero
	// value is "not pressed", so every existing literal that does not name it
	// means what it always meant.
	SecondaryPressed bool

	// SecondaryReleased is the secondary button's RELEASE EDGE, added by
	// docs/1028-command-panel for the command panel's own "right up cancels"
	// (contract B2). Its zero value is "not released".
	SecondaryReleased bool

	// WheelY is this tick's vertical wheel movement, positive away from the
	// user. It is the picker's own read of the wheel: the map screen zooms from
	// Viewer.WheelY, and keeping the two separate is what stops one screen's
	// gesture leaking into the other's.
	WheelY float64

	Escape     bool
	Up         bool
	Down       bool
	Enter      bool
	Typed      string
	AltLetter  byte
	Backspace  bool
	Delete     bool
	Home       bool
	End        bool
	SelectText bool
	// AI-KEY-125: F2 Save and F3 Load/Diplomacy. These are distinct from
	// the main menu's existing L shortcut. Press-edge policy is DIV-526.
	SaveGame   bool
	LoadGame   bool
	QuickSave  bool
	QuickLoad  bool
	QuickSpell [4]bool // physical F5–F8 press edges; Ctrl comes from Viewer.Ctrl

	// Pause, Faster and Slower are the map screen's three paced-cadence keys.
	// Unpaced and Paced select the original's separate owner-loop arm
	// (`AI-KEY-125`, `SESS-CLOCK-005`, `SESS-IDLE-007`). They are PRESS EDGES,
	// like every other key in this snapshot and unlike Viewer.Input's levels: a
	// held key acts once, where a level would toggle the stop on every frame it
	// was down.
	//
	// They are read on the MAP ARM ALONE. "On every other screen all five do
	// nothing" is therefore a property of where the read stands, not a rule
	// someone keeps — the menu and picker arms cannot reach them.
	//
	// Their zero value is "not pressed", so every existing literal that does not
	// name them means what it always meant.
	Pause   bool
	Faster  bool
	Slower  bool
	Unpaced bool
	Paced   bool

	// Kill and Chip are the map screen's two DEBUG BLOW keys. They are press
	// edges like the three above and unlike Viewer.Input's levels: a held key
	// acts once, where a level would empty a company in a quarter of a second.
	//
	// They are read on the MAP ARM ALONE, beside the cadence keys and for the
	// same reason: "on every other screen both do nothing" is then a property of
	// where the read stands rather than a rule someone keeps.
	//
	// Their zero value is "not pressed", so every existing literal that does not
	// name them means what it always meant.
	Kill bool
	Chip bool

	// Load is the MAIN MENU's own key for the LOAD GAME window. It is the same
	// physical key Chip reads and there is no conflict: Chip is read on the map
	// arm alone and this is read on the menu arm alone, so neither screen can
	// reach the other's meaning. Its zero value is "not pressed", so every
	// existing literal that does not name it means what it always meant.
	Load bool

	// Grab is the map screen's PICK-UP key. It is a press edge like every other
	// key in this snapshot and unlike Viewer.Input's levels: a held key acts
	// once, where a level would try the transfer at the frame rate.
	//
	// It is on the LETTER register, beside Kill and Chip: it moves the
	// world's own container, exactly as they move a unit's own health, so it
	// takes an action's letter and not a diagnostic's function key.
	//
	// It is read on the MAP ARM ALONE, beside the two blow keys and for the
	// same reason: "on every other screen it does nothing" is a property of
	// where the read stands rather than a rule someone keeps.
	//
	// Its zero value is "not pressed", so every existing literal that does
	// not name it means what it always meant.
	Grab bool

	// Numerals is the damage-numeral display's toggle, and it is the ONE key in
	// this snapshot whose binding is decoded rather than ours: Ctrl+L, solved
	// by scanning the original's own vkey-to-arm byte table for every key
	// reaching the flip, 1 of 161 (ANIM-NUM-020, AI-KEYMOD-059, MENU-057).
	//
	// IT SHARES ITS LETTER WITH Chip ABOVE AND THE TWO ARE EXCLUSIVE. L was
	// bound to the chip on this project's own authority, before anything was
	// decoded about the letter; the decoded key needs the same letter under a
	// modifier. So the split is made at the binding, by the same Ctrl read
	// AttackHeld below already makes: a press of L reaches exactly one of them,
	// and Ctrl+L no longer chips.
	//
	// Split at the CONSUMER instead, this snapshot would carry a Chip that is
	// true on a frame that did not chip, and every existing literal that builds
	// it by hand would be asserting something false.
	//
	// It is a press edge and is read on the MAP ARM ALONE, beside the two above
	// and for their reasons. Its zero value is "not pressed", which is NOT the
	// same as the display being off — the display is on from the moment a map
	// opens and the flag that says so lives on the viewer.
	Numerals bool

	// Attack is the map screen's ARMING key and the command panel's own cell 0
	// (docs/1028-command-panel contract B3). It is a press edge like every
	// other key in this snapshot and unlike Viewer.Input's levels: a held key
	// toggles once, where a level would flip the mode at the frame rate and
	// leave which way up it ended a function of frame timing.
	//
	// It is read on the MAP ARM ALONE, beside the blow keys, so "on every other
	// screen it does nothing" is a property of where the read stands.
	//
	// Its zero value is "not pressed", so every existing literal that does not
	// name it means what it always meant.
	Attack bool

	// AttackHeld is the map screen's attack MODIFIER, and it is the one member
	// of this snapshot that is a LEVEL rather than a press edge.
	//
	// The exception is the point. What is being reconstructed gates its attack
	// cursor on a global set when a modifier key goes down and cleared when that
	// same key comes up — a key-held latch, and a latch needs a level. An edge
	// here would need a toggle behind it, and a toggle is the thing this
	// replaces.
	//
	// IT IS IN THIS SNAPSHOT AND NOT IN Viewer.Input, which is where the other
	// modifier level lives. Input is read by the standalone developer viewer too,
	// so a modifier there would give that viewer a mode; here, read on the map
	// arm alone, "the developer viewer has no attack pointer" is a property of
	// where the read stands rather than a rule someone keeps.
	//
	// Its zero value is "not held", so every existing literal that does not name
	// it means what it always meant.
	AttackHeld bool

	// HighlightHeld is the level of the sack highlight key (Z, with neither
	// group modifier down). It only changes how the map is drawn.
	HighlightHeld bool

	// Unfocused is whether the front-end does NOT hold the window's focus this
	// tick.
	//
	// IT IS STORED NEGATED, and that is the readout flag's own inversion for the
	// readout flag's own reason: FALSE has to be the ordinary state, or every
	// snapshot literal in the suite that does not name it would describe an
	// unfocused tick and would silently lower the attack mode. A default that IS
	// the zero value is one nothing has to know about.
	//
	// It is read on NO screen's arm — it is read before the dispatch — because
	// focus is not a map-screen question and a latch that survives an alt-tab is
	// a bug wherever it lives.
	Unfocused bool

	// Grid is the cell lattice's toggle, and it is the first member of a
	// register this front-end did not have.
	//
	// THE CONVENTION, AUTHORED HERE BECAUSE THERE WAS NONE: an ACTION the map
	// screen performs takes a LETTER (Kill is K, Chip is L), and a DIAGNOSTIC
	// the map screen SHOWS takes a FUNCTION KEY. The two registers are worth
	// keeping apart because they differ in consequence rather than in degree —
	// a letter changes the world, an F-key changes only what is drawn over it —
	// and because a diagnostic pressed by accident should be visibly harmless.
	// It is a seam and not a decoded fact: nothing published says what the
	// original bound, and this asserts nothing about it.
	//
	// It is a PRESS EDGE like every other key in this snapshot: a held key
	// toggles once, where a level would flicker the lattice at the frame rate.
	// It is read on the MAP ARM ALONE, so "on every other screen it does
	// nothing" is a property of where the read stands.
	//
	// Its zero value is "not pressed", so every existing literal that does not
	// name it means what it always meant.
	Grid bool

	// Readout is the debug readout's toggle, and the SECOND member of the
	// diagnostic register above.
	//
	// It takes F1, which that register deliberately left free for it: the
	// readout is a diagnostic — it changes what is drawn over the world and
	// never the world — so it belongs to the F-keys and not to the letters.
	// F2 is the lattice.
	//
	// It is a PRESS EDGE like every other key in this snapshot: a held key
	// toggles once, where a level would flicker the box at the frame rate. It
	// is read on the MAP ARM ALONE, so "on every other screen it does nothing"
	// is a property of where the read stands.
	//
	// Its zero value is "not pressed" — which is NOT the same as the readout
	// being off. The box is shown by default and the flag that says so lives on
	// the viewer, so every existing literal that does not name this means "no
	// key was pressed this frame" exactly as it always did.
	Readout bool

	// FPS is F12, the frame-rate readout's toggle (MENU-060). A press edge on
	// the map arm below the popup gate; off at load.
	FPS bool

	// TimeFlow is the day/night cycle's switch. It takes a LETTER under Ctrl,
	// which is the counterexample the register comment above is narrowed for:
	// `Ctrl`+`N` is the ORIGINAL's binding, decoded, and a decoded binding
	// outranks a convention of ours.
	//
	// IT WAS BARE `N` UNTIL THIS STORY'S ROUND 3, under a comment claiming that
	// bare letter was the decoded one. `keyboard.tsv` row 50 gives day/night
	// only as `Ctrl`+`N` ("Ctrl held", `main.txt[104..105]`) and names no bare
	// `N` at all, so bare `N` was a key of this build's occupying a letter the
	// original spends under a modifier. Bare `N` is now unbound.
	//
	// It is a PRESS EDGE, so a held key flips the cycle once rather than at the
	// frame rate — the original's is an edge too, and a level here would force a
	// relight every frame. It is read on the MAP ARM ALONE.
	TimeFlow    bool
	Smoothing   bool
	AutoHealing bool

	// ShowHealth is the show-health setting's switch (round 3), `keyboard.tsv`
	// row 48's `Ctrl`+`H`. It changes only what is drawn, so it sits beside
	// TimeFlow above and under its rules exactly: a PRESS EDGE, and read on the
	// MAP ARM ALONE.
	//
	// Its zero value is "not pressed", which is NOT the same as the bars being
	// hidden — they are drawn from the moment a map opens and the flag that
	// says so lives on the viewer — so every existing literal that does not
	// name this means what it always meant.
	ShowHealth bool

	// Formation is `AI-KEY-125`'s Ctrl+F player command (`MENU-057`). It is a PRESS EDGE;
	// Ctrl itself remains the level latch read by ctrlHeld. The field is read
	// on the MAP ARM alone and below the popup gate, so town, text-entry and
	// modal surfaces consume it without a second list of exclusions here.
	//
	// The front end stores no mode. The press leaves through Viewer's formation
	// sink into the world's ordinary command queue, where opcode 0x46 selector
	// 2 performs the decoded remap and writes the existing canonical byte.
	Formation bool

	// Retreat is `AI-KEY-125`'s Ctrl+W player command (`MENU-057`). Like
	// Formation it is a map-only press edge below the popup gate and leaves
	// through a Viewer-installed sink into the ordinary world-command queue.
	Retreat bool

	// PauseText is `keyboard.tsv` row 20's `Pause` key: show the modal text at
	// `main.txt[119]`. It is NOT this build's own clock pause, which is
	// bare zero and arrives in the Pause field above (`DIV-2263`); the two
	// are different subjects on different keys and the names differ by one
	// word for that reason.
	//
	// A press edge, read on the MAP ARM ALONE and BELOW the popup gate, which
	// is this build's own spelling of the row's "repeat blocked by modal
	// state": a press made while any popup stands never reaches the raise.
	//
	// Its zero value is "not pressed", so every existing literal that does not
	// name it means what it always meant.
	PauseText bool

	// Help is F1's press edge (MENU-051): the help panel opens on the map arm
	// below the popup gate, so a press over a notice, menu or documents panel
	// is ignored, and nothing reads it in town.
	Help bool

	// PageUp and PageDown are the help panel's page scroll keys.
	PageUp, PageDown bool

	// LightStep is the unbound lighting-clock diagnostic input.
	LightStep bool

	// Reveal is the unbound fog-of-war diagnostic input.
	Reveal bool

	// DIV-232
	Inventory bool
	Book      bool
	Doll      bool
	Worn      bool

	// PaneMode carries Tab. Town character panes use it for their mode switch
	// (TOWN-355); dialogs use it for focus. The mission keeps its figure and
	// separate statistics card visible together (owner direction, DIV-344).
	PaneMode bool

	// Panels is the original's own Space (`AI-KEY-125`, `keyboard.tsv`): with
	// the inventory and the spellbook both closed it opens both, and with
	// either open it closes each open one. It is NOT the pair of toggles above
	// pressed together -- that would open the closed half of a mixed pair --
	// so it is a field of its own with its own arm in the map branch.
	Panels bool

	// SelectAll is the original's own E (`AI-KEY-125`, `AI-SELECT-122`):
	// select every owned unit, replacing the selection.
	SelectAll bool

	// GroupKey is true on the frame one of the ten group digits was pressed,
	// and GroupDigit is which of the ten it was. The Ctrl, Alt and Shift
	// latches read alongside them decide which of the four actions the press
	// is (`AI-KEY-125`'s three digit rows; Viewer.groupKey).
	//
	// GroupDigit IS A NUMBER RATHER THAN TEN BOOLS because the original's own
	// handler takes one virtual key per message and the actions are mutually
	// exclusive; ten flags would state a frame with two groups pressed, which
	// has no meaning here.
	//
	// THE SEPARATE BOOL IS WHAT MAKES THE ZERO VALUE A NEUTRAL FRAME. Group 0
	// is a real group, so a single int cannot say "no digit" and "group 0"
	// apart at its own zero, and every `appInput{}` literal in this package --
	// there are hundreds -- is a frame with no key pressed.
	GroupKey   bool
	GroupDigit int

	// Left and Right are the generation screen's ADJUST keys: a press decreases
	// or increases whatever the focused chargen row holds — a choice row
	// cycles one option back or forward, a statistic row steps by one, both
	// through Chargen.Adjust's own single argument. Up and Down above already
	// carry this screen's FOCUS move; these two exist because nothing before
	// this screen needed a second axis of navigation to read.
	//
	// They are read on the CHARGEN ARM ALONE. "On every other screen they do
	// nothing" is a property of where the read stands, not a rule someone
	// keeps — the menu, picker and map arms have no statement that reaches
	// them.
	//
	// Their zero value is "not pressed", so every existing literal that does
	// not name them means what it always meant.
	Left  bool
	Right bool

	// Guard, StandGround, Patrol and March are the order vocabulary's four
	// keys, joined by Move (docs/1028-command-panel, `AI-PANEL- 053` table
	// entry 2) at this same story. Guard and StandGround ACT on the press;
	// Patrol, March and Move ARM, and the next secondary press aims them —
	// which is the attack key's own shape, reused rather than a second one
	// invented.
	//
	// They are APPENDED here, so every existing literal that does not name them
	// means "no key was pressed this frame" exactly as it always did. Autocast
	// is the autocast toggle's own combination: Ctrl and A together. It is a
	// COMBINATION and not a bare letter because the owner asked for one, and
	// because every bare letter in this build toggles something visible while
	// this one spends a unit's mana — so an accidental press here costs more
	// than an accidental press anywhere else on the keyboard.
	//
	// DIV-231
	Autocast bool

	Guard         bool
	PlayerRetreat bool
	StandGround   bool
	Patrol        bool
	March         bool
	Defend        bool
	// Move is the command panel's own Move cell and its keyboard mirror
	// (docs/1028-command-panel contract B3/B4). It arms `commandMove`
	// exactly the way Patrol arms `commandPatrol` and March arms
	// `commandSwarm`; the next secondary press aims it. Its zero value is
	// "not pressed".
	Move bool
	// Cast is the panel's Cast cell's own key, C (`MENU-054`, `MENU-055`). A
	// press edge on the map arm below the popup gate. Its zero value is "not
	// pressed".
	Cast bool

	// ShiftHeld is the shop screen's quantity modifier (owner, DIV-046,
	// DIV-047, DIV-1463): a click or a drag from a shelf, table or pack cell
	// moves one unit of the stack, and with ShiftHeld the whole stack.
	//
	// IT IS A LEVEL LIKE AttackHeld ABOVE AND UNLIKE EVERY PRESS-EDGE KEY IN
	// THIS SNAPSHOT, for AttackHeld's own reason: the release is what acts, and
	// this states only whether the physical key is down when it lands.
	//
	// It is read on the TOWN ARM'S SHOP BRANCH ALONE. Viewer.Input already
	// carries its own Shift for the map screen's box-select-or-pan choice
	// (readInput, viewer.go); this is the same physical key read a second
	// time, because the map's Input is not reached from the town arm.
	//
	// Its zero value is "not held", so every existing literal that does not
	// name it means what it always meant.
	ShiftHeld bool
}

// enterKeys are the two keys that are Enter: the main one and the numpad's.
// Windows posts the same virtual key for both and marks the numpad one only with
// the extended-key flag, which the engine's key layer turns into the second key.
// The original's key handlers (TOWN-211, TOWN-207, DLG-KEYS-040) test the
// virtual key alone and no claim has it read that flag, so the two act alike.
var enterKeys = [...]ebiten.Key{ebiten.KeyEnter, ebiten.KeyNumpadEnter}

// enterPressed reports whether either Enter key answers down this tick.
// readAppInput asks the engine's press edge; a headless key edge asks onlyKey.
func enterPressed(down func(ebiten.Key) bool) bool {
	for _, key := range enterKeys {
		if down(key) {
			return true
		}
	}
	return false
}

// onlyKey is the key state in which the one key is down, which is what a headless
// key edge presents to the bindings a windowed tick asks.
func onlyKey(key ebiten.Key) func(ebiten.Key) bool {
	return func(k ebiten.Key) bool { return k == key }
}

// readAppInput samples this tick's front-end input from the engine.
//
// The bindings live here and nowhere else: the primary mouse button is the left
// one, the secondary the right one, Esc leaves a screen, and the picker takes
// the arrow keys, Enter and the wheel. The menu screen deliberately reads no
// keys but Esc — it is driven by the pointer, and keyboard navigation of the
// brooch is not part of this front-end.
func readAppInput() appInput {
	cx, cy := ebiten.CursorPosition()
	_, wy := ebiten.Wheel()
	shiftHeld := ebiten.IsKeyPressed(ebiten.KeyShiftLeft) || ebiten.IsKeyPressed(ebiten.KeyShiftRight)
	return appInput{
		AnyKey:            len(inpututil.AppendJustPressedKeys(nil)) != 0,
		AnyHeld:           len(inpututil.AppendPressedKeys(nil)) != 0,
		Close:             ebiten.IsWindowBeingClosed(),
		Viewer:            readInput(),
		CursorX:           cx,
		CursorY:           cy,
		WheelY:            wy,
		PrimaryPressed:    inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft),
		PrimaryReleased:   inpututil.IsMouseButtonJustReleased(ebiten.MouseButtonLeft),
		SecondaryPressed:  inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonRight),
		SecondaryReleased: inpututil.IsMouseButtonJustReleased(ebiten.MouseButtonRight),
		Escape:            inpututil.IsKeyJustPressed(ebiten.KeyEscape),
		Up:                inpututil.IsKeyJustPressed(ebiten.KeyArrowUp),
		Down:              inpututil.IsKeyJustPressed(ebiten.KeyArrowDown),
		Enter:             enterPressed(inpututil.IsKeyJustPressed),
		Typed:             string(ebiten.AppendInputChars(nil)),
		AltLetter:         readAltLetter(altHeld(), inpututil.IsKeyJustPressed),
		Backspace:         inpututil.IsKeyJustPressed(ebiten.KeyBackspace),
		Delete:            inpututil.IsKeyJustPressed(ebiten.KeyDelete),
		Home:              inpututil.IsKeyJustPressed(ebiten.KeyHome),
		End:               inpututil.IsKeyJustPressed(ebiten.KeyEnd),
		SelectText:        inpututil.IsKeyJustPressed(ebiten.KeyA) && ctrlHeld(),
		// Left and Right are the chargen screen's own adjust keys, read here
		// beside Up/Down/Enter for the same reason those three are: every screen's
		// keys are gathered in one snapshot regardless of which arm ever reads
		// them.
		Left:  inpututil.IsKeyJustPressed(ebiten.KeyArrowLeft),
		Right: inpututil.IsKeyJustPressed(ebiten.KeyArrowRight),
		// SPACE IS THE ORIGINAL'S PANEL-SET TOGGLE (`AI-KEY-125`); the clock
		// pause is the bare zero key (DIV-2263).
		Panels: inpututil.IsKeyJustPressed(ebiten.KeySpace) && !altHeld() &&
			!ebiten.IsKeyPressed(ebiten.KeyShiftLeft) && !ebiten.IsKeyPressed(ebiten.KeyShiftRight),
		Pause: barePauseKey(),
		// CTRL IS GUARDED OFF BOTH (round 3). `keyboard.tsv` rows 21 and 22 give
		// the speed step "Ctrl NOT held", and rows 23 and 24 give `Ctrl`+numpad
		// plus and minus a different mechanism entirely: the unpaced/max-speed
		// owner idle loop, and its restore with a phase/epoch reset.
		Faster: inpututil.IsKeyJustPressed(ebiten.KeyEqual) ||
			(inpututil.IsKeyJustPressed(ebiten.KeyNumpadAdd) && !ctrlHeld()),
		Slower: inpututil.IsKeyJustPressed(ebiten.KeyMinus) ||
			(inpututil.IsKeyJustPressed(ebiten.KeyNumpadSubtract) && !ctrlHeld()),
		// The modified NUMPAD pair selects and clears a separate loop; it does
		// not name the main keyboard's =/- keys and it is not an alias for the
		// ladder ceiling. Repeated Windows keydown messages are not exposed by
		// Ebitengine's snapshot. Clear's phase reset therefore runs once on the
		// press edge; a held frame does not invent a frame-rate repeat cadence.
		Unpaced: inpututil.IsKeyJustPressed(ebiten.KeyNumpadAdd) && ctrlHeld(),
		Paced:   inpututil.IsKeyJustPressed(ebiten.KeyNumpadSubtract) && ctrlHeld(),
		Kill:    inpututil.IsKeyJustPressed(ebiten.KeyK),
		// DIV-236
		Grab: inpututil.IsKeyJustPressed(ebiten.KeyF) && !ctrlHeld(),
		// L IS SPLIT BY THE MODIFIER. Bare L chips; Ctrl+L flips the damage
		// numerals, which is the decoded binding. The same Ctrl read AttackHeld
		// below makes decides both, so the two cannot come to overlap or to leave
		// a press unclaimed.
		Chip: inpututil.IsKeyJustPressed(ebiten.KeyL) && !ctrlHeld(),
		Load: inpututil.IsKeyJustPressed(ebiten.KeyL) && !ctrlHeld(),
		// CTRL+L IS FLYING DAMAGE and Ctrl+O is smoothing (`MENU-057`,
		// `ANIM-NUM-020`); the earlier keyboard table named them the other way
		// round. Bare L stays the chip, split by the Ctrl read.
		Numerals: inpututil.IsKeyJustPressed(ebiten.KeyL) && ctrlHeld(),
		// A, THE ORIGINAL'S OWN BINDING (`MENU-COMBAT-019` cell 0, Attack/A): this
		// line used to bind F, on the premise that nothing decoded names the
		// panel's own accelerators, which was an absence claim about the corpus
		// written without reading it (`DIV-201`).
		Attack: inpututil.IsKeyJustPressed(ebiten.KeyA) && !ctrlHeld(),
		// THE MODIFIER, and Ctrl is a BINDING rather than a reading. What is
		// decoded is the latch and which of three globals gates the attack cursor;
		// naming that global's virtual key Ctrl is the Windows platform contract,
		// which the claim carrying it labels as such and says every clause
		// survives the withdrawal of. So this line is the replaceable part, and it
		// is the only line that has to change to move the modifier elsewhere.
		//
		// EITHER Ctrl key, the two being one modifier and not two — the same
		// reading Viewer.Input already gives Shift.
		AttackHeld:    ctrlHeld(),
		HighlightHeld: ebiten.IsKeyPressed(ebiten.KeyZ) && !ctrlHeld() && !altHeld(),
		// Negated at the binding, so the field's zero value is a focused tick.
		// This is the one read of the engine's focus in the front-end.
		Unfocused: !ebiten.IsFocused(),
		SaveGame:  inpututil.IsKeyJustPressed(ebiten.KeyF2),
		LoadGame:  inpututil.IsKeyJustPressed(ebiten.KeyF3),
		QuickSave: inpututil.IsKeyJustPressed(ebiten.KeyF4) && !shiftHeld,
		Reveal:    inpututil.IsKeyJustPressed(ebiten.KeyF4) && shiftHeld,
		QuickLoad: inpututil.IsKeyJustPressed(ebiten.KeyF9),
		QuickSpell: [4]bool{
			inpututil.IsKeyJustPressed(ebiten.KeyF5),
			inpututil.IsKeyJustPressed(ebiten.KeyF6),
			inpututil.IsKeyJustPressed(ebiten.KeyF7),
			inpututil.IsKeyJustPressed(ebiten.KeyF8),
		},
		// Diagnostics retain F10/F11.
		Grid:    inpututil.IsKeyJustPressed(ebiten.KeyF10),
		Readout: inpututil.IsKeyJustPressed(ebiten.KeyF11),
		FPS:     inpututil.IsKeyJustPressed(ebiten.KeyF12),
		// CTRL+N, NOT BARE N (round 3). `keyboard.tsv` row 50 gives "Ctrl+N | map
		// | Ctrl held | toggle day/night changes | main.txt[104..105]" and no row
		// of the file names a bare `N` on the mission map. The old comment here
		// called bare `N` the original's own decoded binding, which was true of
		// the ACTION and wrong about the KEY. It is Ctrl+O's and Ctrl+A's own
		// precedent in this file: the modifier is part of the binding.
		TimeFlow:    inpututil.IsKeyJustPressed(ebiten.KeyN) && ctrlHeld(),
		Smoothing:   inpututil.IsKeyJustPressed(ebiten.KeyO) && ctrlHeld(),
		AutoHealing: inpututil.IsKeyJustPressed(ebiten.KeyU) && ctrlHeld(),
		// CTRL+H, THE SHOW-HEALTH SETTING (round 3). `keyboard.tsv` row 48 gives
		// "Ctrl+H | map | Ctrl held | toggle show-health setting |
		// main.txt[100..101]", and those two lines read "Show Health Off" / "Show
		// Health On" on the EN root. What this build draws under that name is the
		// per-unit health bar, so the switch is this build's reading of the
		// caption and the reading is disclosed (`DIV-333`). Bare `H` stays
		// unbound.
		ShowHealth: inpututil.IsKeyJustPressed(ebiten.KeyH) && ctrlHeld(),
		// CTRL+F, THE FORMATION PLAYER COMMAND (`AI-KEY-125`, `MENU-057`,
		// `AI-FORM-037`). The negated guard on Grab above leaves plain F's
		// pick-up meaning intact and prevents one chord from issuing both. The
		// press-edge read keeps a held physical key from cycling at the
		// render-frame rate while Ctrl remains a level latch.
		Formation: inpututil.IsKeyJustPressed(ebiten.KeyF) && ctrlHeld(),
		// CTRL+W, THE RETREAT PLAYER COMMAND (`AI-KEY-125`, `MENU-057`,
		// `SESS-PARAM-017`). W alone remains inert.
		Retreat: inpututil.IsKeyJustPressed(ebiten.KeyW) && ctrlHeld(),
		// THE PAUSE KEY'S OWN MODAL TEXT (round 3). `keyboard.tsv` row 20 gives
		// "Pause | map | campaign phase 2 | show modal text main.txt[119] | repeat
		// blocked by modal state", and line 119 reads "Game paused. Click OK to
		// continue." on the EN root. It carries NO modifier and no Ctrl guard,
		// because the file names no modified form of this key.
		PauseText: inpututil.IsKeyJustPressed(ebiten.KeyPause),
		Help:      inpututil.IsKeyJustPressed(ebiten.KeyF1),
		PageUp:    inpututil.IsKeyJustPressed(ebiten.KeyPageUp),
		PageDown:  inpututil.IsKeyJustPressed(ebiten.KeyPageDown),
		// I is the letter this project authors for the inventory (0110 plan
		// D-10); since 0140 it switches the PACK BAR rather than opening the
		// doll window — the field's own doc says what moved.
		// I OR THE OEM BACKTICK, which are the original's own two keys for this
		// one toggle (`keyboard.tsv`: "I or OEM backtick | map | toggle
		// inventory"). The backtick was unbound here.
		Inventory: inpututil.IsKeyJustPressed(ebiten.KeyI) ||
			inpututil.IsKeyJustPressed(ebiten.KeyBackquote),
		// X, NOT E. `keyboard.tsv` gives E the whole-army selection and
		// `AI-SELECT-122` states the same clause, so E is the original's and
		// SelectAll below takes it. The worn-set switch is this project's own and
		// moves to X, which the original names in no form -- plain or Ctrl -- on
		// the mission map. Recorded as `DIV-296`.
		Worn: inpututil.IsKeyJustPressed(ebiten.KeyX),
		// E IS THE ORIGINAL'S OWN ARMY SELECTION (`AI-KEY-125`,
		// `AI-SELECT-122`): every owned unit, replacing the selection.
		SelectAll: inpututil.IsKeyJustPressed(ebiten.KeyE),
		// THE ORDER VOCABULARY'S KEYS (spec A-2, docs/1028- command-panel contract
		// B3/B4 for Move). `MENU-COMBAT-019` names the original's eight cells and
		// accelerators — Attack/A, Move/M, Guard/G, Defend/D, Cast/C, Swarm/S,
		// Stand Ground/T, Retreat/R — and Guard, Stand Ground and Move take
		// exactly those three letters. Patrol matches no cell of the eight and
		// keeps the free-letter assignment this project already gave it, disclosed
		// as an addition (`DIV- 231`).
		//
		// DIV-237
		Guard:         inpututil.IsKeyJustPressed(ebiten.KeyG),
		PlayerRetreat: inpututil.IsKeyJustPressed(ebiten.KeyR),
		StandGround:   inpututil.IsKeyJustPressed(ebiten.KeyT),
		Patrol:        inpututil.IsKeyJustPressed(ebiten.KeyP),
		// S, NOT H. `keyboard.tsv` gives S "arm Swarm | mode 6" and `AI-PANEL-123`
		// gives the panel's Swarm cell the same mode and click opcode `0x1a`;
		// `commandSwarm` -- which this field raises -- IS that order, so S is not
		// a free letter this project is taking but the original's own binding
		// arriving at the order it names. The field keeps the name March, which is
		// what this project called the order before the row was read; the order
		// itself is unchanged.
		March:  inpututil.IsKeyJustPressed(ebiten.KeyS),
		Defend: inpututil.IsKeyJustPressed(ebiten.KeyD),
		Move:   inpututil.IsKeyJustPressed(ebiten.KeyM),
		// Reserved and inert at this story — see the Cast field's own doc.
		Cast: inpututil.IsKeyJustPressed(ebiten.KeyC),
		// B OR Q, NOT S: `keyboard.tsv` gives "B or Q | map or spellbook state |
		// toggle/close spellbook | 0x40f", so the spellbook switch lands on the
		// original's own two keys and S goes to the Swarm arm above.
		Book: inpututil.IsKeyJustPressed(ebiten.KeyB) ||
			inpututil.IsKeyJustPressed(ebiten.KeyQ),
		// J is the authored paperdoll key (DIV-296); D arms Defend above.
		Doll: inpututil.IsKeyJustPressed(ebiten.KeyJ),
		// Tab, the character pane's own mode key — see the PaneMode field.
		PaneMode: inpututil.IsKeyJustPressed(ebiten.KeyTab),
		// Ctrl+A, on Ctrl+L's own precedent above (Numerals): the modifier
		// changes what the letter means rather than the letter being spent on
		// one of the two.
		Autocast: inpututil.IsKeyJustPressed(ebiten.KeyA) && ctrlHeld(),
		// EITHER Shift key, the two being one modifier and not two — the same
		// reading Viewer.Input already gives Shift (readInput, viewer.go).
		ShiftHeld: shiftHeld,
		// THE DIGIT ROW AND THE NUMPAD'S OWN DIGITS, which `keyboard.tsv`
		// treats as one input ("0..9 or Numpad 0..9"). GroupDigit is -1 for a
		// frame with no digit pressed, which is every frame but the press.
		GroupKey:   groupDigit() >= 0 && !barePauseKey(),
		GroupDigit: max(groupDigit(), 0),
	}
}

// barePauseKey is the clock pause: the digit row's or the numpad's zero with
// no Ctrl, Alt or Shift held. A modified zero stays group 0's recall and
// assignment (DIV-2263).
func barePauseKey() bool {
	return (inpututil.IsKeyJustPressed(ebiten.KeyDigit0) || inpututil.IsKeyJustPressed(ebiten.KeyNumpad0)) &&
		!ctrlHeld() && !altHeld() &&
		!ebiten.IsKeyPressed(ebiten.KeyShiftLeft) && !ebiten.IsKeyPressed(ebiten.KeyShiftRight)
}

// groupDigit is which of the ten group keys was pressed this frame, or -1.
//
// THE TWO ROWS ARE ONE INPUT (`keyboard.tsv`: "0..9 or Numpad 0..9"), so the
// numpad digit answers the same number as the top row's. The lowest pressed
// digit wins a frame carrying two, which no player produces and which keeps the
// answer a single number rather than a set.
func groupDigit() int {
	for n := 0; n <= 9; n++ {
		if inpututil.IsKeyJustPressed(ebiten.Key(int(ebiten.KeyDigit0)+n)) ||
			inpututil.IsKeyJustPressed(ebiten.Key(int(ebiten.KeyNumpad0)+n)) {
			return n
		}
	}
	return -1
}

// ctrlHeld is the Ctrl modifier: EITHER Ctrl key, the two being one modifier and
// not two — the same reading Viewer.Input already gives Shift.
//
// It is a function rather than a repeated expression because several bindings
// depend on the same modifier reading. A split pair such as bare/modified
// numpad +/- must not drift and leave a press claimed by both arms or neither.
func ctrlHeld() bool {
	return ebiten.IsKeyPressed(ebiten.KeyControlLeft) || ebiten.IsKeyPressed(ebiten.KeyControlRight)
}

// altHeld is the Alt modifier: either Alt key.
func altHeld() bool {
	return ebiten.IsKeyPressed(ebiten.KeyAltLeft) || ebiten.IsKeyPressed(ebiten.KeyAltRight)
}

type dialoguePointerOwner struct {
	surface  Screen
	viewer   *Viewer
	picture  *image.RGBA
	revision uint64
}

// dialoguePointerPress captures the pointer for the dialogue that a press
// began on; the kit latch holds the press when it landed on the enabled
// advance button.
type dialoguePointerPress struct {
	owner    dialoguePointerOwner
	captured bool
	press    buttonLatch
}

type townDialogueRevision interface {
	TownDialogueRevision() uint64
}

// App is the game's front-end: an Ebitengine game that shows one screen at a
// time and composes the menu and town family in a native 640x480 frame.
//
// The native source and expanded presentation are two offscreen images. Native
// art is moved in whole segments, never resampled independently; only the
// finished logical frame receives the window's uniform fit. The mission family
// owns a separate height-768 frame and expands its map viewport directly.
type App struct {
	cheatInput     cheatInputState
	screenshot     screenshotState
	tooltip        tooltipController
	tooltipPoint   image.Point
	tooltipTexture *ebiten.Image
	tooltipSurface *TownSurfaceView // current Update's already-built input view
	title          string
	assets         *menu.Assets
	flow           *flow
	mapEntry       func(*Viewer)
	// backgroundFlush waits for work the front end started off the frame
	// thread, such as an automatic save being written.
	backgroundFlush func()
	mapEntryViewer  *Viewer
	sel             menu.Selection
	menuLabel       string

	// Interface sounds share the production device and install-backed bank
	// already used by every Viewer. They live on App because menu and town
	// controls have no map viewer, while the fixed selectors have no map cell.
	soundPlayer            audio.Player
	speechPlayer           audio.Player
	soundBank              SoundBank
	music                  *MusicController
	musicLoads             uint32 // flow.loadUI.completed as of the last music sync
	ambientDevice          AmbientDevice
	cutscene               *video.Player
	cutsceneSource         CutsceneSource
	cutsceneStartupOff     bool
	media                  mediaUI
	cutsceneError          error
	cutsceneCanvas         *ebiten.Image
	cutsceneQueue          []string
	cutsceneName           string
	cutsceneDrain          bool
	cutsceneFinalPresented bool
	// cutsceneAudio streams the current movie's own decoded track, if any.
	// cutsceneAudioStarted/cutsceneAudioFrame are pumpCutsceneAudio's own
	// per-movie latches: Start fires once, the instant the format is known, and
	// a chunk is pushed once per FrameNumber change.
	cutsceneAudio        CutsceneAudioDevice
	cutsceneAudioStarted bool
	cutsceneAudioFrame   uint32

	canvas *ebiten.Image // the unchanged native 640x480 composition frame
	// wideCanvas is the 640x480 non-mission presentation actually fitted into
	// the window.
	wideCanvas *ebiten.Image
	// wideDialogue is the current town modal detached from the native room
	// before a wide column layout separates that room. It is painted intact
	// after the room has been widened and cleared at the start of every Draw.
	wideDialogue      *image.RGBA
	wideDialogueCalls [2]int
	dialogueBackdrop  dialogueBackdropState

	// textSmoothingEnabled is owner decision method C's own on/off switch
	// (DIV-1385): NewApp sets it true, so the presentation-layer overlay is
	// the default, and SetTextSmoothing is the only way to turn it off.
	textSmoothingEnabled bool
	// frameSmoothingOff is the FrameSmoothing switch negated, so an App that
	// never receives it uses the Catmull-Rom scaler (drawFinalFrame).
	frameSmoothingOff bool
	textOverlay       textOverlay
	textEraser        textEraser
	// canvasLog, presentLog and menuLog record what canvas, wideCanvas and
	// menuCanvas were drawn with (pixelLog).
	canvasLog, presentLog, menuLog pixelLog
	// settledBuf/settledTex hold the native composite read back with its
	// visible glyphs erased (settleTextFrame), on the frames the logs cannot
	// decide; textSettleFallbacks counts those frames.
	settledBuf          *image.RGBA
	settledTex          *ebiten.Image
	textSettleFallbacks int
	textCaptured        int
	textKept            int
	// The in-game menu over a map screen settles and smooths its own layer:
	// the map's overlay belongs to the viewer.
	menuOverlay            textOverlay
	menuEraser             textEraser
	menuBuf                *image.RGBA
	menuTex                *ebiten.Image
	menuCaptured, menuKept int
	// textLayers is what TextFates reads: each settled layer of the last
	// frame, its glyphs as captured and the log that decided them.
	textLayers []textLayer
	// menuText is the glyphs the cached menu canvas was drawn with.
	menuText []text.DrawCall

	cursorTex cursorTexture
	// pointer is the window position the latest step was given, which is what
	// App.Draw places the cursor picture at (cursorPlacement).
	pointer image.Point
	// menuCanvas is a SECOND virtual frame, transparent except where the
	// in-game menu's panel stands. It exists because the map screen draws at
	// window resolution and never touches canvas, so a panel in frame
	// coordinates over a map needs a frame of its own to be placed from. It is
	// allocated on the first frame the menu opens over a map and never
	// otherwise.
	menuCanvas *ebiten.Image
	winW       int
	winH       int
	place      frame.Placement

	// composed is the menu state the canvas currently holds, so the CPU
	// composite runs only when the selection actually changes rather than every
	// frame.
	composed menu.State
	hasMenu  bool

	// townCursor is the pointer in the 640x480 town frame. It drives the shop
	// table's hover popup and the click that sends a place home, and has no
	// meaning on another screen.
	townCursor          image.Point
	headlessPointer     image.Point
	hasHeadlessPointer  bool
	headlessUnfocused   bool
	hasTownCursor       bool
	townSurfacePress    buttonLatch
	townSurfaceClick    TownSurfaceControl
	townSurfaceAt       time.Time
	townSurfaceKey      string
	townSurfaceReleased bool
	townSurfaceRevision uint64
	dialoguePress       dialoguePointerPress
	// noticePress is an outcome panel's button press latch (MENU-116).
	noticePress buttonLatch
	// noticePressSerial is the notice serial the outcome latch was taken on.
	noticePressSerial int
	// blink is the edit fields' caret phase (MENU-126).
	blink              caretBlink
	townTipPress       buttonLatch
	townTipPressOwner  townTipOwner
	townPaintAdmission func() bool
	// townSurfaceAnimationTick is presentation time only. Six client ticks
	// make one shipped miniature frame; TownSurfaceView applies it only to
	// the selected card, so no simulation clock or hash consumes it.
	townSurfaceAnimationTick int

	// worldMapTickAt is the wall-clock time WorldMapTick last fired
	// (`DIV-136`, authored): the zero value fires on the world map's first
	// driven frame. `stepTownAt` clears it whenever the gates are not the
	// world map, so the next entry always fires its first tick at once
	// instead of waiting out a stale interval from an earlier visit.
	worldMapTickAt time.Time

	// headlessClock is this App's own advancing timestamp for headless
	// dispatch (headlessAt, headless.go), zero until the first headless call.
	// worldMapTickInterval is the first headless-reachable gate that needs
	// real elapsed time between calls to make progress at all, rather than
	// only to debounce a double click within one still instant — see
	// headlessAt's own doc for why every earlier headless consumer tolerated
	// a frozen clock and this one cannot.
	headlessClock time.Time

	// The shop's own drag machine (1005 round 2), command.go's dragCandKind
	// family held here instead of on a Viewer: the shop has none. Armed by a
	// press on the doll, the shelf grid or the pack strip; shopDragMoved is
	// the Manhattan distance from the press point to wherever the cursor
	// stands THIS FRAME, a high-water mark rather than dragIntent's own
	// per-frame accumulation (command.go, viewer.go) — the shop has no camera
	// pan to share the same measurement with, and a straight distance from
	// one fixed point answers the one question this machine asks (has this
	// gesture crossed TapSlop) exactly as well.
	shopDragArmed bool
	// shopButtonPress latches the shop's command buttons by index.
	shopButtonPress buttonLatch
	shopUseTap      *shopUseTap
	shopDragOrigin  ShopControl
	// shopDragOriginBase is the shelf/pack list base at the press. The
	// visible-cell index may change under a held wheel gesture; retaining the
	// base keeps the origin bound to that one absolute source record.
	shopDragOriginBase int
	shopDragMoved      int
	shopPressX         int
	shopPressY         int

	// shopDragIcon is the item carried on the cursor while a shop drag is past
	// TapSlop (round-2 adversarial review, item 2, `DIV-090` retired):
	// command.go's own dragIcon, restated here since the shop's own drag state
	// lives on App rather than on a Viewer. It is read once at the press from
	// the SAME ShopScreenView cell the grid draws — never a second icon load
	// of its own — while shopDragItemPresent keeps it hidden until TapSlop.
	// Capturing the alias at the press means scrolling before TapSlop cannot
	// substitute a newly visible cell's picture. It is cleared with the rest of
	// the drag state on every release and Escape.
	shopDragIcon *image.RGBA
	// shopStars is presentation state owned by actual shop paints. Each
	// eligible visible cell advances once before ComposeShopScreen, including
	// while no simulation map is running (ITEM-STARPHASE-099).
	shopStars shopStarState

	// suppressPrimaryRelease consumes the release half of a press that changed
	// screens. Without it, pressing Mission Complete enters the town and the
	// same physical click's later release activates a town door.
	suppressPrimaryRelease   bool
	suppressSecondaryRelease bool
	chargenPress             buttonLatch
	chargenHover             chargenControl
	chargenHoverText         string
	chargenChoiceClick       chargenControl
	chargenChoiceAt          time.Time
	// Each surface's own chrgen instances (VIDEO-SFX-058..060). The
	// pre-create page's close stops its own.
	preCreateSounds SFXVoices
	detailedSounds  SFXVoices
	menuSounds      SFXVoices
	hallSounds      SFXVoices
	chargenRepeat   chargenRepeat
	// tipCycles are the pre-create and detailed guided cycles. Like the
	// original's statics they outlive every generator page (TOWN-519).
	tipCycles [2]guidedCycle
	// chargenHeld is whether the left button is down this frame, for the
	// stat buttons' held-and-hovered art (MENU-138).
	chargenHeld bool
}

// NewApp builds the front-end over a validated menu asset set, the picker's rows
// and the loader that turns a chosen row into a map viewer.
func NewApp(title string, assets *menu.Assets, rows []PickerRow, load MapLoader) *App {
	a := &App{
		title:                title,
		assets:               assets,
		flow:                 newFlow(NewPicker(rows), load),
		winW:                 MenuWindowW,
		winH:                 MenuWindowH,
		textSmoothingEnabled: true,
	}
	a.place = frame.Fit(frame.W, frame.H, a.winW, a.winH)
	a.SetTooltipDelayPreference(DefaultTooltipDelay, nil)
	return a
}

// SetMapCadencePreference installs the normal deadline-paced rung a map opens
// on and the optional sink for later +/- or Game Options changes. The separate unpaced
// owner-loop selector is intentionally absent: it starts clear in every new App
// and at every map entry.
func (a *App) SetMapCadencePreference(rung int, persist func(int)) {
	if a == nil {
		return
	}
	a.flow.setCadencePreference(rung, persist)
}

// MapCadencePreference reports the normal rung the next map will open on. It is
// presentation/driver state, not a simulation field or save-form input.
func (a *App) MapCadencePreference() int {
	if a == nil {
		return defaultCadenceRung()
	}
	return a.flow.cadencePreference()
}

// Screen reports which screen is showing.
func (a *App) Screen() Screen {
	if a.cutscene != nil {
		return ScreenCutscene
	}
	return a.flow.screen
}

// MapOpener opens a map screen that NO PICKER ROW NAMES — the campaign
// mission the command line asked for. It is MapLoader's tuple without the
// index, because there is no row to index.
type MapOpener func() (*Viewer, MapTick, MapOrder, MapCadence, MapAffect, MapAdvance, MapAttack, MapGrab, MapStance, MapMarch, error)

// OpenMission shows a mission's map screen at startup, or reports why it
// could not.
//
// IT RUNS THE PICKER'S OWN ENTRY and differs only in where the map came from.
// That is not decoration: the entry establishes the cadence rung from the period
// the world's clock actually started at, and puts the viewer into command mode.
// A door that assigned the seams itself would open a mission whose first speed
// key slammed the tick to the ladder's slowest rung and whose left drag panned
// like the developer viewer, and nothing automated would notice either.
//
// THE MENU AND THE MAP LIST ARE LEFT INTACT BEHIND IT.
//
// It is called BEFORE Run and reports its failure as a value, so a mission that
// cannot be started never opens a window.
func (a *App) OpenMission(open MapOpener) error {
	v, tick, order, cadence, affect, advance, attack, grab, stance, march, err := open()
	if err != nil {
		return err
	}
	a.flow.enter(v, tick, order, cadence, affect, advance, attack, grab, stance, march)
	a.syncViewerLayout()
	a.startPendingCutscene()
	a.syncMusic()
	return nil
}

// OpenChargen arms the generation screen at startup, before any window frame
// is drawn — called BEFORE Run, exactly where OpenMission is called, and
// for the same reason: a caller that gets an error back here never opens a
// window over a screen with nothing to show.
//
// UNLIKE OpenMission IT OPENS NOTHING. So this method's whole job is to arm
// the two things stepChargen and drawChargen read — the model to drive and
// the callback a legal confirm reaches — and switch the screen, nothing
// more.
//
// A NIL MODEL IS REFUSED. Every method stepChargen and drawChargen reach
// through the flow's own chargen field assumes it is not nil once this screen
// is showing — chargen.go's own contract is that a malformed SETUP must
// never panic, not that a missing MODEL must not — so arming the screen over
// one would turn the first tick into exactly the nil pointer dereference this
// package's whole design otherwise refuses to have. Reporting that here,
// before the screen ever shows, is what lets every later read assume it is
// set. The refusal is armChargen's, read here rather than restated: this door
// and the picker's own gated row arm one field through one statement (0140).
//
// ESCAPE FROM THE SCREEN THIS DOOR ARMS STILL LEAVES FOR THE MAIN MENU,
// which is what ScreenMenu names below. It is written out rather than left
// to armChargen's zero value so that the destination is stated where the
// door is, and so a later door choosing a different one reads as a choice.
func (a *App) OpenChargen(c *Chargen, begin func(ChargenResult) (MapOpener, error)) error {
	if !a.flow.armChargen(c, begin, ScreenMenu) {
		return errors.New("OpenChargen: nil model")
	}
	a.chargenPress, a.chargenHover, a.chargenHoverText, a.chargenChoiceClick = buttonLatch{}, chargenNone, "", chargenNone
	a.chargenChoiceAt = time.Time{}
	a.chargenRepeat = chargenRepeat{}
	a.syncMusic()
	return nil
}

// SetChargenGate installs the gate choose() asks before it loads a picker row
// (0140): a row the gate claims opens the generation screen instead, and the
// map behind it is opened later, out of the confirmed character.
//
// IT GATES CHOOSING A ROW AND NOTHING ELSE. OpenMission and a won mission's own
// NoticeToMission advance both enter a map screen without consulting it, which
// is what keeps a campaign transition free of a generation screen it must never
// show — ChargenGate's own doc carries the reason.
//
// IT IS A SETTER AND NOT A NewApp ARGUMENT. NewApp already takes four, every
// existing caller passes them, and a fifth would make every front-end that
// wants no generation at all — cmd/mapview's developer viewer among them —
// spell a nil for it. The gate is also the one part of this screen's wiring
// that a front-end may legitimately not have, which is exactly what an optional
// setter says and a constructor parameter does not.
//
// IT MAY BE CALLED WITH NIL to remove one; nil is the picker this package has
// always had.
func (a *App) SetChargenGate(g ChargenGate) {
	a.flow.chargenGate = g
}

// SetNewGameChargen installs what the menu's NEW GAME arms directly, in place
// of the map picker (owner reversal: a normal launch must open the main menu
// and NEW GAME there must reach generation without a debug flag). g is called
// fresh on every press, on NewGameChargen's own "starting again" reasoning.
//
// IT MAY BE CALLED WITH NIL to remove one; nil is what makes NEW GAME open the
// map picker instead — the debug route -picker installs by leaving this
// unset.
func (a *App) SetNewGameChargen(g NewGameChargen) {
	a.flow.newGameChargen = g
}

// SetNewGameDirect installs what NEW GAME enters without generation, for a base
// that ships no generation art. A nil g, a nil opener, or an opener that fails
// leaves NEW GAME to what SetNewGameChargen arms (the picker when nothing is
// armed), with the failure on the picker's message line.
func (a *App) SetNewGameDirect(g NewGameDirect) {
	a.flow.newGameDirect = g
}

// activateNewGame is NEW GAME's one activation: the direct entry when one is
// installed and opens, else the flow's own arm.
func (a *App) activateNewGame() {
	if a.flow.screen == ScreenMenu {
		if town, ok := a.flow.town.(interface{ StartNewGame() error }); ok {
			// A campaign that starts in town opens generation first when the
			// wiring installed it; its accept starts the campaign.
			if a.flow.newGameChargen != nil {
				if e := a.flow.newGameChargen(); e != nil && a.flow.armChargen(e.Model, e.Begin, ScreenMenu) {
					return
				}
			}
			if err := town.StartNewGame(); err != nil {
				a.flow.msg = err.Error()
				return
			}
			a.flow.showTown("")
			return
		}
	}
	if a.flow.screen != ScreenMenu || a.flow.newGameDirect == nil {
		a.flow.activateNewGame()
		return
	}
	open := a.flow.newGameDirect()
	if open == nil {
		a.flow.activateNewGame()
		return
	}
	v, tick, order, cadence, affect, advance, attack, grab, stance, march, err := open()
	if err != nil {
		a.flow.activateNewGame()
		a.flow.msg = err.Error()
		return
	}
	a.flow.enter(v, tick, order, cadence, affect, advance, attack, grab, stance, march)
	if v != nil {
		v.entryCampaignStart = v.missionCutscene > 0
	}
	a.syncViewerLayout()
}

// SetAudio installs the same optional production device and shipped registry
// bank that FrontEnd gives each map Viewer. A nil half keeps every interface
// event silent, matching Viewer's existing degradation rule.
func (a *App) SetAudio(p audio.Player, b SoundBank) {
	a.soundPlayer, a.soundBank = p, b
}

func (a *App) SetSpeechAudio(p audio.Player) {
	a.speechPlayer = p
}

func (a *App) playUISound(slot UISoundSlot) {
	playUISound(a.soundPlayer, a.soundBank, slot)
}

// namedSounds is the installed bank's member lookup, or nil when the bank has
// none.
func (a *App) namedSounds() NamedSoundBank {
	b, _ := a.soundBank.(NamedSoundBank)
	return b
}

// SetCursorRegistry installs the resolved 28-slot cursor table this session
// draws from. NIL is the ordinary state for a front-end whose install could
// not resolve one, and for every test flow built before this story: every
// screen then draws no cursor of its own and the operating system's arrow
// shows throughout, exactly as it did before this story.
//
// IT ALSO RUNS THE CURRENT SCREEN'S OWN TRANSITION (B4). Every transition run
// before a registry arrives was a no-op — SetCursor cannot resolve a name
// against a registry that is not there — so the screen showing at the moment
// one does arrive would otherwise carry no current cursor at all. In the
// shipped front-end that screen is always the boot menu, which newFlow opens
// on; running the table's own entry for whatever screen is up costs the same
// and does not depend on which one it is.
func (a *App) SetCursorRegistry(reg *CursorRegistry) {
	a.flow.cursor.SetRegistry(reg)
	if exit, ok := screenExitCursor(a.flow.screen); ok {
		a.flow.surfaceTransition(exit)
	}
}

// Layout adopts the window size and forwards it to every screen that has to be
// placed in it.
//
// a.place is the TOWN FAMILY's native 640x480 frame: the main menu, picker,
// chargen pages, town, shop, tavern, school and in-game menu. It remains 4:3
// and is fitted uniformly. The map screen alone uses Viewer's height-768 frame
// and expands horizontally to the window aspect.
func (a *App) Layout(outsideWidth, outsideHeight int) (int, int) {
	if outsideWidth != a.winW || outsideHeight != a.winH {
		a.tooltip.reset()
	}
	if outsideWidth > 0 && outsideHeight > 0 {
		a.winW, a.winH = outsideWidth, outsideHeight
		a.place = frame.Fit(frame.W, frame.H, a.winW, a.winH)
		a.syncViewerLayout()
	}
	return outsideWidth, outsideHeight
}

// Update gathers one tick of input and hands it to step. It holds no logic, so
// nothing the front-end decides is out of a test's reach.
func (a *App) Update() error {
	if a.step(readAppInput(), time.Now()) {
		return ebiten.Termination
	}
	return nil
}

func (a *App) townDialoguePointerOwner(picture *image.RGBA) dialoguePointerOwner {
	owner := dialoguePointerOwner{surface: ScreenTown, picture: picture}
	if town, ok := a.flow.town.(townDialogueRevision); ok {
		owner.picture, owner.revision = nil, town.TownDialogueRevision()
	}
	return owner
}

func (a *App) currentDialoguePointerOwner() (dialoguePointerOwner, bool) {
	switch a.flow.screen {
	case ScreenTown:
		picture, open := townDialogue(a.flow.town)
		return a.townDialoguePointerOwner(picture), open
	case ScreenMap:
		v := a.flow.viewer
		if v != nil && v.NoticeOpen() && v.noticeKind == NoticeDialogue {
			_, _, _, visible := v.noticePresent()
			return dialoguePointerOwner{surface: ScreenMap, viewer: v, revision: uint64(v.noticeSerial)}, visible
		}
	}
	return dialoguePointerOwner{}, false
}

// validateWidgetLatches drops every held widget press when the window loses
// focus or closes, and the outcome latch when its notice was replaced, so a
// release can only activate the button its own press latched (MENU-116).
func (a *App) validateWidgetLatches(in appInput) {
	f := a.flow
	if in.Unfocused || in.Close {
		a.dropWidgetLatches()
		return
	}
	if v := f.viewer; a.noticePress.Holds() && (v == nil || !v.NoticeOpen() || v.noticeSerial != a.noticePressSerial) {
		a.noticePress.Clear()
	}
}

// dropWidgetLatches clears every screen's press latch. A latch belongs to
// the controls of one screen and goes with them: it is dropped when the
// window loses focus and when the screen or menu page it was pressed on is
// left, so a press held across a key that changes screens cannot swallow
// the next press on the screen it returns to (MENU-116).
func (a *App) dropWidgetLatches() {
	f := a.flow
	a.noticePress.Clear()
	f.menuPress.Clear()
	f.loadUI.press.Clear()
	a.media.press.Clear()
	a.sel.Clear()
	a.chargenPress.Clear()
	f.questPress.Clear()
	f.endingPress.Clear()
	f.modUI.backPress.Clear()
	f.modUI.press.Clear()
	if f.saveDialog != nil {
		f.saveDialog.press.Clear()
	}
	if f.docPanel != nil {
		f.docPanel.press.Clear()
	}
	if f.viewer != nil {
		f.viewer.gold.press.Clear()
	}
	f.soundPointer = soundOptionPointer{}
	if d := f.gameOptions.draft; d != nil {
		d.radio = 0
	}
}

// stepNoticeButtons drives an outcome panel's buttons through the shared
// latch: a press latches, a release inside the latched button activates it
// (MENU-116), and the pointer writes each button's hover and pressed look.
func (a *App) stepNoticeButtons(in *appInput) bool {
	v := a.flow.viewer
	id, over := v.noticeButtonIDAt(in.CursorX, in.CursorY)
	if in.PrimaryPressed {
		a.noticePress.Press(id, over)
		a.noticePressSerial = v.noticeSerial
		if over {
			in.PrimaryPressed = false
		}
	}
	activated := false
	if in.PrimaryReleased {
		if _, activated = a.noticePress.Release(id, over); activated {
			in.PrimaryReleased = false
		}
	}
	state := func(b int) DialogueButtonState {
		inside := over && id == b
		return DialogueButtonState{Hover: inside, Pressed: a.noticePress.Pressed(b), Inside: inside}
	}
	v.setNoticeButtonStates(state(noticePrimaryButton), state(noticeSecondButton))
	return activated
}

func (a *App) dialogueButtonDisabled(owner dialoguePointerOwner) bool {
	if owner.viewer != nil {
		return owner.viewer.noticeLayout().ButtonState.Disabled
	}
	if town, ok := a.flow.town.(interface{ TownDialogueButtonState() DialogueButtonState }); ok {
		return town.TownDialogueButtonState().Disabled
	}
	return false
}

func (a *App) applyDialogueButtonState(owner dialoguePointerOwner, state DialogueButtonState) {
	if owner.viewer != nil {
		state.Disabled = a.dialogueButtonDisabled(owner)
		owner.viewer.SetDialogueButtonState(state)
		return
	}
	if owner.surface == ScreenTown {
		state.Disabled = a.dialogueButtonDisabled(owner)
		if town, ok := a.flow.town.(interface {
			TownDialogueVisual(DialogueButtonState, DialogueBackdrop)
		}); ok {
			town.TownDialogueVisual(state, a.dialogueBackdrop.policy)
		}
	}
}

func (a *App) cancelDialoguePointer() {
	owner := a.dialoguePress.owner
	if !a.dialoguePress.captured {
		owner, _ = a.currentDialoguePointerOwner()
	}
	if a.dialoguePress.captured {
		a.suppressPrimaryRelease = true
	}
	a.dialoguePress = dialoguePointerPress{}
	a.applyDialogueButtonState(owner, DialogueButtonState{})
}

func (a *App) validateDialoguePointer(in appInput) {
	if a.cutscene != nil || in.Close || in.Unfocused || in.Enter || in.Escape || in.SecondaryPressed || in.SecondaryReleased {
		a.cancelDialoguePointer()
		return
	}
	if !a.dialoguePress.captured {
		return
	}
	owner, open := a.currentDialoguePointerOwner()
	if !open || owner != a.dialoguePress.owner {
		a.cancelDialoguePointer()
	}
}

func (a *App) stepDialoguePointer(in appInput, owner dialoguePointerOwner, inside bool) bool {
	if in.Close || in.Unfocused || in.Enter || in.Escape || in.SecondaryPressed || in.SecondaryReleased {
		a.cancelDialoguePointer()
		return false
	}
	if a.dialoguePress.captured && a.dialoguePress.owner != owner {
		a.cancelDialoguePointer()
	}
	if in.PrimaryPressed {
		if !a.dialoguePress.captured {
			a.dialoguePress = dialoguePointerPress{owner: owner, captured: true}
		}
		a.dialoguePress.press.Press(0, inside && !a.dialogueButtonDisabled(owner))
	}
	if in.PrimaryReleased {
		_, released := a.dialoguePress.press.Release(0, inside && a.dialoguePress.owner == owner && !a.dialogueButtonDisabled(owner))
		advance := a.dialoguePress.captured && released
		a.dialoguePress = dialoguePointerPress{}
		a.applyDialogueButtonState(owner, DialogueButtonState{Hover: inside, Inside: inside})
		return advance
	}
	a.applyDialogueButtonState(owner, DialogueButtonState{Hover: inside, Inside: inside, Pressed: a.dialoguePress.captured})
	return false
}

// step advances the front-end one tick and reports whether the program should
// exit.
//
// Esc is handled first on every screen, so leaving is never blocked by whatever
// else the screen is doing. Everything after it is per-screen dispatch.
func (a *App) step(in appInput, now time.Time) (exit bool) {
	viewerBefore := a.flow.viewer
	defer func() {
		if v := a.flow.viewer; v != nil {
			if v != viewerBefore && v.animationRestored || a.flow.screen != ScreenMap || in.SaveGame || in.LoadGame || in.QuickSave || in.QuickLoad {
				v.last = now
			}
			if v != viewerBefore {
				v.animationRestored = false
			}
		}
	}()
	defer func() {
		if !in.QuickLoad {
			a.pollTimedAutosave(exit, in.Unfocused)
		}
	}()
	squareBefore, squareRevision := a.townEntryState()
	screenBefore := a.flow.screen
	a.pointer = image.Pt(in.CursorX, in.CursorY)
	a.blink.tick(now)
	a.tooltipSurface = nil
	a.validateDialoguePointer(in)
	a.validateWidgetLatches(in)
	defer func(screen Screen, page gameMenuPage) {
		if a.flow.screen != screen || a.flow.menuPage != page {
			a.dropWidgetLatches()
		}
	}(a.flow.screen, a.flow.menuPage)
	a.validateTownTipPointer(in)
	defer func(before Screen, viewer *Viewer, tooltipInput appInput) {
		a.updateTooltip(tooltipInput, now, exit || before != a.flow.screen || viewer != a.flow.viewer)
	}(a.flow.screen, a.flow.viewer, in)
	defer func() {
		if screenBefore == ScreenGameMenu && a.flow.screen == ScreenTown {
			if town, ok := a.flow.town.(interface{ ResumeTown() }); ok {
				town.ResumeTown()
			}
		}
		if town, ok := a.flow.town.(TownDialogueLifecycle); ok {
			town.TownDialogueActive(!exit && !in.Unfocused && a.cutscene == nil && a.flow.screen == ScreenTown)
		}
		if town, ok := a.flow.town.(TownSquareAnimator); ok {
			town.TownSquareActive(!exit && !in.Unfocused && a.cutscene == nil && a.flow.screen == ScreenTown && atTownSquare(a.flow.town))
		}
		if town, ok := a.flow.town.(TavernInteriorLifecycle); ok {
			town.TavernInteriorActive(!exit && !in.Unfocused && a.cutscene == nil && a.flow.screen == ScreenTown && atTownSurface(a.flow.town))
		}
		if town, ok := a.flow.town.(ShopInteriorAnimator); ok {
			town.ShopInteriorActive(!exit && !in.Unfocused && a.cutscene == nil && a.flow.screen == ScreenTown && atTownShop(a.flow.town))
		}
		if town, ok := a.flow.town.(SchoolTrainingLifecycle); ok {
			town.SchoolTrainingActive(!exit && !in.Unfocused && a.cutscene == nil && a.flow.screen == ScreenTown && atTownSurface(a.flow.town))
		}
		a.paintTownEntry(squareBefore, squareRevision, !exit && !in.Unfocused && a.cutscene == nil)
		a.validateTownTipPointer(in)
		if in.Unfocused || a.flow.screen != ScreenTown || !atTownSurface(a.flow.town) {
			a.resetTownSurfacePair()
		}
	}()
	defer a.validateShopUseTap()
	if in.Escape || in.SecondaryPressed {
		a.shopUseTap = nil
	}
	defer a.paintChargenCaret(now)
	defer a.updateMusic()
	defer a.startPendingCutscene()
	if a.cutscene != nil {
		a.stepCutscene(in, now)
		return false
	}
	if in.Close {
		return true
	}
	if a.cutsceneDrain {
		// The key/button which skipped belongs wholly to the overlay. Drain
		// held levels as well as edges before the map may see another gesture.
		if !in.AnyHeld && !in.Viewer.PrimaryDown && !in.Viewer.SecondaryDown {
			a.cutsceneDrain = false
			a.suppressPrimaryRelease, a.suppressSecondaryRelease = false, false
			a.flow.syncCadence(true)
		}
		return false
	}
	// THE CURSOR MANAGER'S CLOCK, advanced on every tick regardless of screen:
	// a no-op until a registry names a current cursor with more than one frame
	// (CursorManager.Advance's own guard).
	a.flow.cursor.Advance(now.UnixMilli())
	if a.flow.screen == ScreenChargen && a.flow.chargen != nil {
		a.flow.chargen.advancePresentation(now, !in.Unfocused)
	}
	if a.flow.screen == ScreenCutsceneLibrary || a.flow.screen == ScreenCredits {
		a.stepMedia(in, now)
		return false
	}
	if a.flow.screen == ScreenTown {
		if atTownSurface(a.flow.town) {
			a.townSurfaceAnimationTick++
		} else {
			a.townSurfaceAnimationTick = 0
		}
	} else {
		a.townSurfaceAnimationTick = 0
	}
	shop, atShop := a.flow.town.(TownShopScreen)
	if a.flow.screen != ScreenTown || !atShop || !shop.AtTownShop() {
		a.shopStars.reset()
	}

	// FOCUS FIRST, ABOVE EVERYTHING. What is being reconstructed clears all
	// three of its modifier gates from one routine reached off the window's
	// focus-loss message, and it is right to: a mode held by a key the window
	// can no longer see the release of is a mode that never comes down.
	//
	// It stands ABOVE the Escape branch and above the screen switch, so "on every
	// screen and whatever else the tick is doing" is where the statement is
	// rather than a case list somebody maintains — Escape returns, and a clear
	// below it would be skipped on the tick it fires.
	//
	// It does NOT return: an unfocused tick is otherwise an ordinary tick, and
	// the engine keeps calling Update while the window is in the background. The
	// world must go on advancing.
	//
	// IT ALSO DROPS THE MODIFIER OUT OF THE SNAPSHOT, and that second line is
	// not belt and braces — without it the map arm below reads the level again
	// and raises the mode straight back, one statement later, on the very tick
	// this one lowered it. What a keyboard reports while the window cannot see
	// its key-ups is precisely what must not be believed, so the frame is made
	// to carry no modifier rather than the clear being made to fight it. It is
	// the notice's own consume of a press, applied to a level.
	if v := a.flow.viewer; v != nil {
		v.playerRetreatBlocked = in.Unfocused
		v.setSackHighlight(false)
	}
	if in.Unfocused {
		in.AttackHeld = false
		in.Defend = false
		if v := a.flow.viewer; v != nil {
			v.clearAttackMode()
			if v.aimed == commandDefend {
				v.aimed = commandNone
			}
		}
	}

	// A press that navigated to another screen owns its eventual release too.
	// Keep the latch through every held frame, consume the one release edge,
	// then permit the next physical press/release gesture normally.
	if a.suppressPrimaryRelease {
		a.suppressPrimaryRelease = !in.PrimaryReleased
		in.PrimaryPressed, in.PrimaryReleased, in.Viewer.PrimaryDown = false, false, false
	}
	if a.suppressSecondaryRelease {
		a.suppressSecondaryRelease = !in.SecondaryReleased
		in.SecondaryPressed, in.SecondaryReleased, in.Viewer.SecondaryDown = false, false, false
	}

	if a.flow.missionCheatInput() && in.Viewer.Alt && in.AltLetter != 0 && !in.Unfocused {
		a.dispatchCheatAlt(in.AltLetter)
		in.AltLetter = 0
		in.Typed = ""
	}
	in = a.stepCheatChat(in)
	if in.Escape {
		// The open Drop Gold editor owns Escape as its cancel.
		if v := a.flow.viewer; v != nil && a.flow.screen == ScreenMap && v.goldModalOpen() && !in.Unfocused {
			v.cancelGold()
			return false
		}
		if a.flow.screen == ScreenSave || a.flow.screen == ScreenMod && a.flow.modUI.back == ScreenGameMenu {
			a.holdMapUnderMenu(in, now)
		}
		if a.flow.screen == ScreenGameMenu {
			// Load can return here and immediately receive Escape, without
			// a neutral menu frame. Consume the held interval before lowering
			// the popup, just as the menu's Return action already does.
			a.holdMapUnderMenu(in, now)
		}
		// THE SHOP'S OWN DRAG MACHINE IS DROPPED FIRST (round-2 adversarial
		// review, item 2): Escape can unwind the shop room with the primary button
		// still held, and the eventual release lands outside stepTown's own inShop
		// gate once the screen has changed, so nothing else clears it. Harmless
		// where no shop drag is armed — clearShopDrag then writes the same zero
		// values already there.
		a.clearShopDrag()
		// Detailed-stage Escape is its own Back transition. Pre-create Escape
		// keeps the existing flow unwind to the screen that armed generation.
		// A page whose Escape does nothing swallows it.
		if c := a.flow.chargen; a.flow.screen == ScreenChargen && c != nil {
			switch c.keys().Escape {
			case "none":
				return false
			case "back":
				if c.Back() {
					a.chargenPress, a.chargenHover, a.chargenHoverText, a.chargenChoiceClick = buttonLatch{}, chargenNone, "", chargenNone
					a.chargenChoiceAt = time.Time{}
					return false
				}
			}
		}
		if a.flow.noticeOpen() {
			a.flow.advanceNotice(a.flow.viewer.noticeEscapeAction())
			a.syncViewerLayout()
			return false
		}
		// Pre-create Escape requests the page's escape sound unless it plays,
		// then closes the page, and the close stops it (VIDEO-SFX-058).
		preCreate := a.flow.screen == ScreenChargen && a.flow.chargen != nil && a.flow.chargen.Stage() == PreCreateStage
		if preCreate {
			a.preCreateSounds.RequestFor("character-precreate", a.soundPlayer, a.namedSounds(), a.flow.chargen.keys().EscapeSound)
		}
		exit := a.flow.escape()
		if preCreate {
			a.preCreateSounds.Stop()
		}
		if exit {
			return true
		}
		a.syncViewerLayout()
		return false
	}

	if a.stepGameFunctionKeys(in) {
		return false
	}
	if h, ok := screenHandlers[a.flow.screen]; ok && h.Step != nil {
		return h.Step(a, in, now)
	}
	switch a.flow.screen {
	case ScreenMap:
		quickBlocked := in.Unfocused || a.flow.popupOpen()
		// THE FRAME GUARD'S OWN BASELINE. Captured before the advance below runs,
		// so the comparison after it can ask the one question that stays true
		// across every destination a notice can choose: did THIS statement change
		// what viewer is adopted.
		viewer := a.flow.viewer
		if viewer != nil && viewer.goldModalOpen() {
			if a.flow.noticeOpen() {
				viewer.cancelGold()
			} else {
				viewer.blink.tick(now)
				in = a.stepGoldModal(viewer, in)
			}
		}

		// Dialogue owns its complete pointer gesture before the map sees input.
		// The outcome panels retain their press actions and release suppression.
		// Return selects the default action; Escape is handled above.
		if a.flow.noticeOpen() {
			if hv := a.flow.viewer; hv.help != nil {
				hv.helpTab(in.PaneMode)
				hv.helpKeys(in.Up, in.Down, in.PageUp, in.PageDown)
				if hv.helpPointer(in) {
					in.PrimaryPressed = false
				}
			}
			mouseAction, mouseAdvance := a.flow.viewer.noticeActionAt(in.CursorX, in.CursorY)
			dialogue := a.flow.viewer.noticeKind == NoticeDialogue
			if dialogue {
				owner, _ := a.currentDialoguePointerOwner()
				mouseAdvance = a.stepDialoguePointer(in, owner, a.flow.viewer.noticeDialogueInside(in.CursorX, in.CursorY))
				in.PrimaryPressed, in.PrimaryReleased = false, false
			} else {
				mouseAdvance = a.stepNoticeButtons(&in)
			}
			if in.Enter || mouseAdvance {
				in.PrimaryPressed = false
				action := a.flow.viewer.noticeDefaultAction()
				if mouseAdvance {
					action = mouseAction
				}
				a.flow.advanceNotice(action)
			}
		}
		// A POPUP TAKES EVERY OTHER MAP-SCREEN INPUT. The product's author has now
		// taken the rest. The gate is ONE early return below, after the advance;
		// the three inputs above are the whole of what still answers while a popup
		// is up.
		//
		// THE GUARD IS THE VIEWER, NOT THE SCREEN. Until this story every
		// destination a notice could choose left the map screen — the menu, or
		// the map list — so "did the screen change" and "did the advance
		// navigate" were the same question. AN ADVANCE ONTO A SUCCESSOR IS NOT:
		// enter() sets f.screen back to ScreenMap on the very statement that
		// adopts the new viewer, so a screen test here would read ScreenMap before
		// the advance and ScreenMap after it and see no change — and the rest of
		// this arm would go on to tick, step and command a mission that press had
		// no part in (AC-9, R-2). The viewer identity is what actually moved:
		// leaveMap nils it and enter replaces it, on every one of the seam's
		// destinations, and a notice that only paged is the one case that touches
		// neither.
		if a.flow.viewer != viewer || a.flow.screen != ScreenMap {
			// A destination was taken. Whatever it left showing — the menu, the map
			// list, or a freshly adopted successor — is laid out here and nowhere
			// else on this arm, which is what every other route that adopts a viewer
			// already does on the statement it adopts one (OpenMission, both choose()
			// sites, the chargen door). Skipping it for this one route would leave a
			// successor's camera at whatever size its Viewer happened to be built
			// with and its armed start view unapplied — the one door into a map
			// screen that never sized itself to its window (AC-13).
			a.syncViewerLayout()
			return false
		}

		// The five cadence keys, resolved BEFORE the advance below. A press
		// therefore takes effect on the frame it is made on: the stop this frame
		// set is already set when the advance asks whether to run, and the rate
		// this frame selected is already the rate it runs at.
		//
		// It is the ONLY call to it and it is on THIS ARM, which is the whole of
		// "on every other screen all five do nothing" — the menu and picker
		// arms have no statement to reach a cadence through.
		//
		// It moves no camera and changes no selection: those are v.step's and
		// v.command's below, and this call touches neither.
		stepped := !a.flow.popupOpen()
		a.flow.stepCadence(in.Pause && !quickBlocked, in.Faster, in.Slower, in.Unpaced, in.Paced)
		if stepped && in.Faster != in.Slower && !in.AttackHeld && a.flow.viewer != nil {
			a.flow.viewer.postSpeedNotice(a.flow.rung)
		}

		// ONE advance, here and nowhere else in the front-end. The call is
		// unconditional on this arm and absent from every other, so "once per
		// map-screen tick" is a property of there being a single call site on a
		// single arm rather than a rule someone has to keep.
		//
		// IT STAYS UNCONDITIONAL UNDER AN OPEN NOTICE, and that is deliberate
		// rather than something 0073 forgot. A suspended world is one that has
		// been TOLD it is stopped, by the cadence call above — not one that is
		// no longer asked. The far side consumes its pacing baseline on every call
		// including the stopped one, so the span a notice covers is spent as it
		// passes. Not calling would leave that span owed, and the first call after
		// a dismissal would pay it up to its catch-up bound: a quarter-second of
		// world time in one frame, every walking unit jumping. It would freeze the
		// world's own readout for the same span too, because that push rides this
		// call.
		//
		// Nil is the ordinary case for a loader that put nothing under the map.
		if tick := a.flow.tick; tick != nil {
			tick()
		}
		if v := a.flow.viewer; v != nil {
			// The same method the standalone viewer's Update calls. Routing the
			// map screen through it is what makes the two entry points' camera
			// behaviour one thing rather than two that must be kept in step.
			//
			// ITS ORDER AGAINST THE ADVANCE ABOVE IS NOW LOAD-BEARING, and until 0077
			// it was not: this comment said the two were independent because neither
			// read what the other wrote, and that was true while the step only moved
			// a camera. A POPUP IS RAISED INSIDE THE ADVANCE — the far side settles
			// its notices inside the tick loop — so the step's own gate below sees
			// it on the very frame it is raised on. Reversing these two statements
			// would leave a selection rectangle latched into the first frame the box
			// is drawn on, and nothing else in the tree would fail.
			//
			// It is also why it stands FIRST among the viewer's statements now,
			// where the lattice used to: the popup gate below has to be one
			// statement, and the step must run under it rather than beside it.
			v.step(in.Viewer, now)

			if a.flow.popupOpen() {
				return false
			}
			in.suppressAltLetters()
			// Backspace empties the message line and does nothing else (MENU-061).
			if in.Backspace {
				v.ClearMessages()
			}
			if in.FPS {
				v.ToggleFPS()
			}
			if !quickBlocked {
				for slot, pressed := range in.QuickSpell {
					if pressed {
						v.quickSpell(slot, in.Viewer.Ctrl, in.CursorX, in.CursorY)
					}
				}
			}

			// THE CAMPAIGN DOCUMENTS PANEL'S OWN DOOR. The request was raised inside
			// the advance above, by the tier that reads item codes, and it is drained
			// HERE — on the map arm, below the popup gate — because that is this
			// build's reading of the original's own gate on the panel's only shower:
			// `campaign+0x3dc == 1`, which TOWN-352 reads as bit 0 alone, the mission
			// frame up with no other surface over it. Below the popup gate is the
			// same statement in this build's terms (DIV-298).
			//
			// It returns, because the panel is a screen: every statement
			// below this one on this arm reads a map screen that is no
			// longer showing. A refusal — nothing collected — falls through
			// and the rest of the arm runs as it always did, which is what
			// makes an unusable item do nothing rather than open an empty
			// sheet.
			if v.TakeDocumentsRequest() && a.flow.openDocuments(ScreenMap) {
				a.syncViewerLayout()
				return false
			}

			// THE LATTICE. It moves no camera, no selection and no world — it
			// changes what the frame draws and nothing else — so its position among
			// the viewer's statements is not observable, and it is early only so that
			// the frame this press lands on is already drawn with the answer.
			//
			// It is the ONLY call to it and it is on THIS ARM, which is the
			// whole of "on every other screen it does nothing": the menu and
			// picker arms have no statement to reach a viewer through.
			if in.Grid {
				v.ToggleGrid()
			}

			// THE READOUT, beside the lattice and under its rules exactly. It moves
			// no camera, no selection and no world; its position on this arm is
			// therefore not observable, and it is early only so that the frame this
			// press lands on is already drawn with the answer.
			//
			// It is the ONLY call to it and it is on THIS ARM, which is the
			// whole of "on every other screen it does nothing": the menu and
			// picker arms have no statement to reach a viewer through.
			//
			// It is a HIDE, not a show. The box is on from the moment a map
			// opens — a readout that had to be switched on could not report the
			// key that switched it on — so this key exists for the frame the
			// owner wants clean.
			if in.Readout {
				v.ToggleReadout()
			}

			// THE DAMAGE NUMERALS, beside the lattice and the readout and under their
			// rules exactly. It moves no camera, no selection and no world, so its
			// position on this arm is not observable, and it is here only so that the
			// frame this press lands on is already composed with the answer.
			//
			// It is the ONLY call to it and it is on THIS ARM, which is the
			// whole of "on every other screen it does nothing".
			//
			// It is a HIDE, not a show: figures are drawn from the moment a map
			// opens, which is the original's own default, so this key exists for
			// the frame the owner wants clean. And it gates CREATION rather than
			// drawing, so turning it back on reveals nothing that accrued while
			// it was off.
			if in.Numerals {
				if !a.flow.cycleGameOption(GameOptionDamage) {
					v.ToggleDamageNumerals()
					on, _ := v.DamageNumerals()
					v.postSettingNotice(noticeSlotFlyingDamage, noticeState(on))
				} else {
					a.postOptionNotice(v, GameOptionDamage, noticeSlotFlyingDamage)
				}
			}

			// THE DAY/NIGHT SWITCH AND THE LIGHTING-CLOCK SCRUB, beside the three
			// above and under their rules exactly. Neither moves a camera, a
			// selection or a world — both rebuild the light the frame is drawn by
			// and nothing else — so their position on this arm is not observable,
			// and they are here only so that the frame each press lands on is already
			// lit with the answer.
			//
			// They are the ONLY calls to either and they are on THIS ARM, which
			// is the whole of "on every other screen they do nothing".
			//
			// THE SWITCH IS ON WHEN A MAP OPENS, which is the original's
			// compiled default, so N is the key for the fixed sun rather than
			// for the moving one — the reverse of the lattice and the readout,
			// and stated because the asymmetry is easy to read as a bug.
			if in.TimeFlow {
				if !a.flow.cycleGameOption(GameOptionDayNight) {
					v.ToggleTimeFlow()
					v.postSettingNotice(noticeSlotDayNight, noticeState(v.TimeFlow()))
				} else {
					a.postOptionNotice(v, GameOptionDayNight, noticeSlotDayNight)
				}
			}
			if in.Smoothing {
				if !a.flow.cycleGameOption(GameOptionSmoothing) {
					options := v.GraphicsOptions()
					options.Smoothing = !options.Smoothing
					v.SetGraphicsOptions(options)
					v.postSettingNotice(noticeSlotSmoothing, noticeState(options.Smoothing))
				} else {
					a.postOptionNotice(v, GameOptionSmoothing, noticeSlotSmoothing)
				}
			}
			if in.AutoHealing && a.flow.cycleGameOption(GameOptionAutoHealing) {
				a.postOptionNotice(v, GameOptionAutoHealing, noticeSlotAutoHealing)
			}
			if in.LightStep {
				v.StepLightClock()
			}

			// THE SHOW-HEALTH SETTING, beside the four above and under their rules
			// exactly (round 3). It moves no camera, no selection and no world — it
			// decides which units statusBars answers for — so its position
			// on this arm is not observable.
			//
			// It is the ONLY call to it and it is on THIS ARM, which is the
			// whole of "on every other screen it does nothing".
			//
			// It is a HIDE, not a show: bars are drawn from the moment a map
			// opens, so the switch's first press is the "Show Health Off" the
			// shipped caption names (`DIV-333`).
			if in.ShowHealth {
				if !a.flow.cycleGameOption(GameOptionHealth) {
					v.ToggleShowHealth()
					v.postSettingNotice(noticeSlotShowHealth, noticeState(v.HealthBarsShown()))
				} else {
					a.postOptionNotice(v, GameOptionHealth, noticeSlotShowHealth)
				}
			}

			// CTRL+F'S FORMATION PLAYER COMMAND. It is below the popup gate and on
			// the map arm alone, so modal/text-entry surfaces and every town surface
			// consume it. The viewer holds only the injected seam: the existing
			// formation byte remains simulation state and this press reaches it only
			// after the world's ordinary queued advance.
			if in.Formation {
				if !a.flow.cycleGameOption(GameOptionFormation) {
					v.cycleFormation()
				} else {
					a.postOptionNotice(v, GameOptionFormation, noticeSlotFormation)
				}
			}

			// CTRL+W'S RETREAT PLAYER COMMAND. It shares Formation's map/popup
			// boundaries and queued-command timing. Plain F stays the pick-up, so
			// the same press cannot also pick up a sack through Grab below.
			if in.Retreat {
				if !a.flow.cycleGameOption(GameOptionRetreat) {
					v.cycleRetreat()
				} else {
					a.postOptionNotice(v, GameOptionRetreat, noticeSlotRetreat)
				}
			}

			// THE PAUSE KEY'S MODAL TEXT (`keyboard.tsv` row 20, round 3). It stands
			// BELOW the popup gate, which is this build's own spelling of that row's
			// "repeat blocked by modal state": a press made while a notice, the
			// in-game menu or the documents panel is up never reaches this line, so
			// the key cannot stack one notice on another or re-raise its own.
			//
			// It falls through rather than returning. The notice takes effect
			// on the frame after this one, exactly as a notice the far side
			// raises inside the advance does, and the popup gate holds the
			// world from that frame on.
			if in.PauseText {
				a.flow.showTextNotice(v.Words().PauseNotice)
			}

			// F1'S HELP PANEL (MENU-051): below the popup gate, so every popup
			// ignores the key, and on the map arm alone, so the town ignores it.
			if in.Help {
				a.flow.showHelp(v.Words().HelpText)
			}

			// THE FOG-OF-WAR DEBUG REVEAL, beside the four diagnostics above and
			// under their rules exactly. It moves no camera, no selection and no
			// world — it changes only whether fogAt answers visible for every cell
			// — so its position on this arm is not observable, and it is here only
			// so that the frame this press lands on is already composed with the
			// answer.
			//
			// It is the ONLY call to it and it is on THIS ARM, which is the
			// whole of "on every other screen it does nothing".
			if in.Reveal && !quickBlocked {
				v.ToggleFogReveal()
			}

			// THE AUTOCAST TOGGLE, beside the display switches and under the same
			// fence: it moves no camera, no selection and no world of this package's,
			// so its position on this arm is not observable, and it sits ABOVE the
			// command path below so the press that flips it cannot also be read as
			// part of a gesture.
			//
			// toggleAutocast DECIDES EVERYTHING (spellbook.go): no sink, no
			// unit, no spell and an unowned unit are all no-ops there, so
			// there is no second gate written out here.
			if in.Autocast {
				v.toggleAutocast()
			}

			if in.Inventory {
				v.toggleHudPanel(hudPanelPack)
			}
			if in.Worn {
				v.toggleHudPanel(hudPanelWorn)
			}
			if in.Book {
				v.toggleHudPanel(hudPanelBook)
			}
			if in.Doll {
				v.toggleHudPanel(hudPanelDoll)
			}

			// SPACE, THE ORIGINAL'S OWN PAIR TOGGLE (`keyboard.tsv`): both
			// closed opens both, and either open closes each open one. It is
			// deliberately NOT `in.Inventory && in.Book` -- that reading opens
			// the closed half of a mixed pair, which is the one case the two
			// rows of the table exist to separate.
			if in.Panels {
				pack, book := v.hudShown(hudPanelPack), v.hudShown(hudPanelBook)
				switch {
				case !pack && !book:
					v.toggleHudPanel(hudPanelPack)
					v.toggleHudPanel(hudPanelBook)
				default:
					if pack {
						v.toggleHudPanel(hudPanelPack)
					}
					if book {
						v.toggleHudPanel(hudPanelBook)
					}
				}
			}

			// THE TWO SELECTION KEYS, beside the switches and above the command path
			// for the switches' own reason: a key that writes the selection must not
			// also be read as part of a gesture. Both are pure viewer state and reach
			// no world seam; the selection reply below hears what they selected.
			if in.SelectAll {
				v.selectAllOwnedUnits()
			}
			if in.GroupKey {
				v.groupKey(in.GroupDigit, in.Viewer.Ctrl, in.Viewer.Alt, in.Viewer.Shift)
			}

			// The frame's buttons, AFTER the camera step, and the accumulator is why:
			// command judges a release by how far this gesture has travelled, and
			// this tick's own delta is raised inside dragIntent, which v.step above
			// has just run. Judging the release first would judge it on everything
			// the gesture did EXCEPT the movement that ended it.
			//
			// So this arm's call order IS observable, and the disclosed cost is that
			// an order formed on this frame reaches the NEXT advance — one frame
			// plus at most one tick period later — rather than the one already run
			// above. The advance stays first, unconditional and single, which is what
			// keeps it one per map-screen tick whether an order was issued or not.
			//
			// A nil seam is the ordinary case for a loader that put nothing
			// under the map, exactly as a nil tick is: the gesture is still
			// resolved and the selection still moves, and the orders it
			// produced have nowhere to go.
			//
			// k ORDERS ARE k CALLS, in the ascending id decide emitted them in. The
			// seam keeps its three scalars: widening it to take a collection would
			// put a collection in this tier for the far side to consume, and give the
			// far side a second append path beside the one statement that both queues
			// an order and marks its entity commanded. They are all appended before
			// the next advance truncates the queue, which is what makes "one frame's
			// orders reach ONE advance, in issue order" that truncation's position
			// rather than a new rule. THE ARMING KEY, read BEFORE the gesture and on
			// this arm alone. Before, so that a key and a press arriving in one frame
			// arm and then spend, in that order — which is the two- click shape
			// collapsed into one frame, and the only reading under which "the next
			// press is the consuming press" is true of every frame rather than of
			// most of them.
			//
			// It reads the selection the PREVIOUS frame ended holding, which is
			// the selection the player was looking at when he pressed the key.
			if in.Attack {
				v.armAttack()
			}

			// THE THREE AIMING KEYS, beside the attack key and under both of its
			// rules (docs/1028-command-panel contract B3/B4 for Move): read BEFORE
			// the gesture, so a key and a press arriving in one frame arm and then
			// spend in that order, and on this arm alone. Each toggles its own order,
			// and raising any one lowers the other two — armCommand's own
			// statement, so no ordering here can leave two arms up.
			//
			// PATROL IS READ FIRST, then March, then Move, so a frame that
			// somehow saw all three ends with Move armed. That is arbitrary
			// and it is written down as arbitrary: what matters is that it is
			// a defined single outcome rather than three flags in an
			// undefined set.
			if in.Patrol {
				v.armCommand(commandPatrol)
			}
			if in.March {
				v.armCommand(commandSwarm)
			}
			if in.Move {
				v.armCommand(commandMove)
			}
			if in.Defend {
				v.armDefend()
			}
			// C arms Cast mode over a nonempty selection that can cast and is
			// consumed silently over one that cannot (MENU-054, MENU-055).
			if in.Cast {
				v.castKey()
			}

			// THE MODIFIER, beside the key and under the same two rules: read before
			// the gesture, and on this arm alone.
			//
			// Before, so that a modifier pressed and a press made in one frame
			// arm and then spend in that order — the same reading the key's own
			// position rests on.
			//
			// On this arm alone, and BELOW the popup gate above, which is the whole
			// of "a popup takes it too": the viewer's step has already lowered the
			// mode by the time that gate returns, and this line is never reached to
			// raise it again.
			//
			// It ASSIGNS the level rather than testing it, so a tick with the
			// modifier up is what lowers the mode. That is why there is no `if`
			// here and why there must not be one: a guarded call would make the
			// latch a one-way raise.
			v.setAttackHeld(in.AttackHeld)
			v.setSackHighlight(in.HighlightHeld && !in.Unfocused)

			// THE CHARACTER PANE'S MENU CORNER IS DRAINED HERE, immediately after the
			// gesture that could have raised it. Rect F posts `0x416`, which
			// `MENU-ESC-010` (High for the handler identity and both rects) reads on
			// the frame window's own `VK_ESCAPE` arm under `campaign+0x3dc == 1` —
			// so the corner and the key reach one surface, and this calls the same
			// `escape()` arm rather than a second opener beside it. The corner's own
			// tooltip in the original is `text/main.txt` line 15, `Main Menu <ESC>`,
			// which names the pairing outright.
			//
			// IT RUNS BEFORE THE ORDERS ARE ISSUED, not after: the menu holds
			// the mission still (`holdMapUnderMenu`), so a frame that opens it
			// must not also walk the party.
			ords, ordsOK := v.command(in)
			// The E key, a group recall, a click and a marquee each record what
			// they selected; one reply for the frame follows the gesture, before
			// the menu door can leave the map.
			v.replySelection(now)
			if v.TakeCharacterPaneMenu() {
				a.flow.escape()
				a.syncViewerLayout()
				return
			}
			if ordsOK {
				// TWO SEAMS, ONE WALK. The frame decided one list, in ascending id, and
				// each member goes out through the seam of its own kind — so k orders
				// are k calls whichever kind they are, and the emission order is the
				// list's rather than two lists' interleaving.
				//
				// A nil seam is the ordinary case for a loader that put nothing
				// under the map, exactly as a nil tick is, and an order of that
				// kind then reaches nothing.
				issue, strike, march := a.flow.order, a.flow.attack, a.flow.march
				pick := a.flow.grab
				var spoken []uint32
				var gesture VoiceGesture
				speak := func(g VoiceGesture, id uint32) {
					if v.commandAcknowledgment != nil {
						gesture = g
						spoken = append(spoken, id)
					}
				}
				for _, ord := range ords {
					// Dispatch each admitted order to its typed sink in selection order.
					switch ord.kind {
					case orderKindDefend:
						if v.defendSink != nil {
							v.defendSink(ord.entity, ord.victim)
							speak(VoiceDefend, ord.entity)
						}
					case orderKindTown:
						if v.structureUseSink != nil {
							v.structureUseSink(ord.entity, ord.victim)
							speak(VoiceTown, ord.entity)
						}
					case orderKindAttack, orderKindCast:
						if strike != nil {
							strike(ord.entity, ord.victim, ord.spell, ord.x, ord.y, ord.cell)
							// A cast order asks for no reply (VIDEO-067).
							if ord.kind == orderKindAttack {
								speak(VoiceAttack, ord.entity)
							}
						}
					case orderKindSwarm, orderKindPatrol:
						if march != nil {
							march(ord.entity, ord.kind == orderKindPatrol, ord.x, ord.y)
							if ord.kind == orderKindPatrol {
								speak(VoicePatrol, ord.entity)
							} else {
								speak(VoiceSwarm, ord.entity)
							}
						}
					case orderKindPickup:
						// THE MAP'S PICK-UP CURSOR REACHES THE SAME SEAM THE
						// PICK-UP KEY DOES (`AI-CLICK-050` arm `0x21`) and
						// names what the key cannot: the unit the cursor's own
						// gate selected, and the cell the sack stands on. That
						// is `ITEM-PICK-016`'s own payload for `0x21`, and it
						// is what closed `DIV-292` -- until round 4 this call
						// passed nothing and the transfer acted for the
						// inventory window's subject at ITS cell, so a click
						// on a sack the selected hero was not already standing
						// on did nothing at all.
						if pick != nil {
							pick(ord.entity, ord.x, ord.y, true)
							speak(VoicePickup, ord.entity)
						}
					case orderKindMove:
						if issue != nil {
							issue(ord.entity, ord.x, ord.y)
							speak(VoiceMove, ord.entity)
						}
					}
				}
				if len(spoken) != 0 {
					v.commandAcknowledgment(gesture, spoken, now)
				}
			}

			// THE TWO BLOW KEYS, read AFTER the gesture and on this arm alone. After,
			// so that a unit selected by the very tap of this frame can be hit by a
			// key pressed with it — the selection this reads is the one the frame
			// ended holding, not the one it began with.
			//
			// The marked set is taken ONCE and both keys walk it, so a frame that
			// saw both issues a kill and a chip for the same units rather than a
			// chip for whatever the kill left. Neither writes to the selection:
			// they read the same filter the marks and the orders read, through the
			// viewer's own method, and hand ids to the seam.
			//
			// A nil seam is the ordinary case for a loader that put nothing under
			// the map, exactly as a nil tick is, and the keys then reach nothing.
			if in.Kill || in.Chip {
				if affect := a.flow.affect; affect != nil {
					marked := v.marked()
					// Each key's own emission is ascending id, which is what makes
					// a frame's blows a function of the set and not of the walk.
					if in.Kill {
						for _, id := range marked {
							affect(id, true)
						}
					}
					if in.Chip {
						for _, id := range marked {
							affect(id, false)
						}
					}
				}
			}

			// THE PICK-UP KEY, beside the two blow keys and under the same rule: a
			// nil seam is the ordinary case for a loader that put nothing under the
			// map, exactly as a nil tick is, and the key then reaches nothing. It
			// carries no id — MapGrab's own doc says why — so there is no marked
			// set to walk here: the far side already knows which character it acts
			// for. THE FOUR ARGUMENTS ARE THE SEAM'S CURSOR ARM'S and the key reads
			// none of them, which `aimed` false is what says.
			if in.Grab {
				if grab := a.flow.grab; grab != nil {
					grab(0, 0, 0, false)
				}
			}

			// THE TWO CELL-FREE STANDING ORDERS, read AFTER the gesture and on this
			// arm alone, on the two blow keys' own grounds: after, so that a unit
			// selected by the very tap of this frame can be ordered by a key pressed
			// with it — the selection this reads is the one the frame ENDED
			// holding.
			//
			// The marked set is taken ONCE and both keys walk it, so a frame
			// that saw both issues both, each over the same units — and the far
			// side resolves the second, because the last order applied inside
			// one advance is the one that stands. Neither writes to the
			// selection: they read the same filter the marks, the orders and the
			// blows read.
			//
			// A nil seam is the ordinary case for a loader that put nothing
			// under the map, exactly as a nil tick is, and the keys then reach
			// nothing.
			//
			// PENDING GUARD AND STAND GROUND (docs/1028-command-panel contract
			// B2) join the two keys here rather than opening a second call to
			// a.flow.stance: a press on the panel's own Guard or Stand Ground
			// cell sets one of v's two one-shot flags (commandpanel.go), and
			// consumeCommandStancePending reads and clears both every tick
			// whether or not either was set, so a tick with neither pressed
			// costs one pair of false reads and nothing else.
			pendingGuard, pendingStandGround := v.consumeCommandStancePending()
			if in.PlayerRetreat {
				v.issuePlayerRetreat()
			}
			v.replyRetreat(now)
			if in.Guard || in.StandGround || pendingGuard || pendingStandGround {
				if stance := a.flow.stance; stance != nil {
					marked := v.marked()
					if in.Guard || pendingGuard {
						for _, id := range marked {
							stance(id, true)
						}
						v.replyStance(VoiceGuard, marked, now)
					}
					if in.StandGround || pendingStandGround {
						for _, id := range marked {
							stance(id, false)
						}
						v.replyStance(VoiceStandGround, marked, now)
					}
				}
			}
		}
	case ScreenChargen:
		a.stepChargen(in, now)
	}
	return false
}

// stepMenu drives the brooch: the cursor selects at most one button through the
// hit mask, and a press-and-release on the same button activates it. It reports
// whether the activated button ends the program.
//
// New Game, Load, Hall of Fame and Exit use their installed buttons.
func (a *App) stepMenu(in appInput) (exit bool) {
	if a.stepModMenu(in) {
		return false
	}
	hit := a.buttonAt(in.CursorX, in.CursorY)

	if in.Load {
		a.flow.openLoad(ScreenMenu)
		return false
	}

	switch {
	case in.PrimaryPressed:
		// A press on any of the eight buttons requests the menu's own ok.wav
		// unless it plays (VIDEO-SFX-060).
		if hit != 0 {
			a.menuSounds.RequestFor("main-menu", a.soundPlayer, a.namedSounds(), ChargenSoundOK)
		}
		a.sel.Press(hit)
	case in.PrimaryReleased:
		released := a.sel.Release(hit)
		if released != 0 {
			a.playUISound(UISoundCommonControl)
		}
		switch released {
		case menu.NewGameButton:
			a.activateNewGame()
			a.PlayCutsceneSequence(numberedCutscenes("newgame"))
		case menu.LoadGameButton:
			a.flow.openLoad(ScreenMenu)
		case menu.HallOfFameButton:
			a.flow.showHallOfFame()
		case menu.CutscenesButton:
			a.openCutsceneLibrary()
		case menu.CreditsButton:
			a.openCredits(ScreenMenu)
		case menu.ExitButton:
			return true
		}
	default:
		a.sel.Move(hit)
	}
	return false
}

// buttonAt maps a window position to the brooch button under it, or 0.
//
// A position in the letterbox maps to no frame pixel and therefore to no button,
// which is the same answer as a position over the brooch's background.
func (a *App) buttonAt(x, y int) int {
	p, ok := a.windowToNativeFrame(x, y)
	if !ok {
		return 0
	}
	return a.assets.ButtonAt(p)
}

// stepPicker drives the map list: arrows move the selection, the wheel moves it
// in larger steps, Enter chooses it, and a primary release inside the list
// chooses the row under the cursor.
//
// The wheel moves the SELECTION rather than an independent view offset. The
// picker keeps one invariant the whole screen rests on — top is whatever keeps
// sel visible, and Visible() is the single value the draw path, the hit test and
// the header all derive from. A view that scrolled on its own would break it: the
// selection could leave the screen, and a drawn row would no longer be the row
// RowAt names. Moving the selection reuses Move's clamping and minimal scrolling
// unchanged.
//
// Only the sign of WheelY is read. Its magnitude is platform-dependent — one per
// notch on a Windows mouse, fractional on some trackpads — so a step scaled by it
// would be a step nobody could state.
func (a *App) stepPicker(in appInput) {
	switch {
	case in.WheelY > 0:
		a.flow.picker.Move(-pickerWheelRows) // away from the user: towards row 0
	case in.WheelY < 0:
		a.flow.picker.Move(+pickerWheelRows)
	case in.Up:
		a.flow.picker.Move(-1)
	case in.Down:
		a.flow.picker.Move(+1)
	case in.Enter:
		a.activatePicker(a.flow.picker, a.flow.choose)
		a.syncViewerLayout()
		// See OpenMission's own comment: a successful choose() hands the
		// screen a BRAND NEW *Viewer, closed and holding no subject by
		// construction, and a failed one leaves a.flow.viewer nil — either
		// way there is no inventory state reachable here for a statement of
		// this file's own to reset.
	case in.PrimaryReleased:
		if p, ok := a.windowToNativeFrame(in.CursorX, in.CursorY); ok {
			if row, ok := a.flow.picker.RowAt(p); ok {
				a.flow.picker.Select(row)
				a.activatePicker(a.flow.picker, a.flow.choose)
				a.syncViewerLayout()
			}
		}
	}
}

// activatePicker is the clean-room common-control boundary. Picker.Choose is
// pure and answers whether the non-zero activation will run; click01 therefore
// plays after the disabled-row gate and before the supplied action, for Enter
// and pointer routes alike (VIDEO-SFX-016).
func (a *App) activatePicker(list *Picker, choose func()) {
	if list != nil {
		if _, ok := list.Choose(); ok {
			a.playUISound(UISoundCommonControl)
		}
	}
	choose()
}

// stepTown drives the town.
//
// IT IS stepPicker'S OWN DISPATCH over the town's list, and that is deliberate
// down to the wheel: this screen is a text list drawn through the map list's
// own model, so the presses that move it are the same presses, and a player who
// has learned one list has learned both. Writing a second dispatch would be a
// second chance for the two to differ in a way nobody could see until they
// pressed a key.
//
// ESCAPE IS NOT HERE. It is read above, before any per-screen dispatch,
// exactly as it is for every other screen, and flow.escape's own ScreenTown
// arm is what unwinds a room. THE SHOP DRAG MACHINE IS CLEARED THERE TOO
// (App.step's own Escape branch, clearShopDrag) — Escape can leave the
// shop room with the primary button still physically held (round-2
// adversarial review, item 2), and the release that eventually follows lands
// outside this function's own inShop gate, so nothing here would ever see it
// and a.shopDragArmed would stay true into whatever the player does next.
// clearShopDrag drops any candidate the shop's own drag machine is holding,
// with no release ever having resolved it — App.step's Escape branch calls
// this before flow.escape() unwinds the room, so a stale press point and a
// stale origin cannot survive into a later frame or a later room (round-2
// adversarial review, item 2).
func (a *App) clearShopDrag() {
	a.shopUseTap = nil
	a.shopDragArmed, a.shopDragOrigin, a.shopDragMoved = false, ShopControl{}, 0
	a.shopButtonPress.Clear()
	a.shopDragOriginBase = 0
	a.shopDragIcon = nil
}

func shopDragGridBase(v ShopScreenView, origin ShopControl) int {
	switch origin.Kind {
	case ShopControlShelfCell:
		return v.ShelfOffset
	case ShopControlPackCell:
		return v.PackOffset
	default:
		return 0
	}
}

// shopDragOriginAt projects the press-bound absolute source record back into
// the grid's current visible-index vocabulary. The result may be negative or
// past the visible array when the selected item has scrolled offscreen; that
// is intentional. Render omission then touches no replacement cell, while the
// model's existing base+index readers still recover the original record.
func shopDragOriginAt(v ShopScreenView, origin ShopControl, sourceBase int) ShopControl {
	origin.Index += sourceBase - shopDragGridBase(v, origin)
	return origin
}

// shopDragOriginIcon is origin's own picture out of v — ShopCell.Icon for
// a shelf or pack cell, SlotIcon for a doll slot — or nil for an origin
// naming none of the three (round-2 adversarial review, item 2).
func shopDragOriginIcon(v ShopScreenView, origin ShopControl) *image.RGBA {
	switch origin.Kind {
	case ShopControlDoll:
		if origin.Index >= 0 && origin.Index < len(v.SlotIcon) {
			return v.SlotIcon[origin.Index]
		}
	case ShopControlShelfCell:
		if origin.Index >= 0 && origin.Index < len(v.Shelf) {
			return v.Shelf[origin.Index].Icon
		}
	case ShopControlPackCell:
		if origin.Index >= 0 && origin.Index < len(v.Pack) {
			return v.Pack[origin.Index].Icon
		}
	case ShopControlTableCell:
		if origin.Index >= 0 && origin.Index < len(v.Table) {
			return v.Table[origin.Index].Icon
		}
	}
	return nil
}

// shopDragCursorName is `cantput` while a shelf-stamped object is held in the panel (DIV-1667).
func (a *App) shopDragCursorName(v ShopScreenView, in appInput) string {
	if !a.shopDragArmed || a.shopDragMoved < TapSlop || !a.shopDragHoldsShelfStamp(v) {
		return a.shopCursorRestore()
	}
	if p, ok := a.windowToNativeFrame(in.CursorX, in.CursorY); ok && shopPaneTakesItemAt(p) {
		return "cantput"
	}
	return a.shopCursorRestore()
}

func (a *App) shopCursorRestore() string {
	if a.flow.cursor.CurrentName() == "cantput" {
		return "default"
	}
	return ""
}

func (a *App) shopDragHoldsShelfStamp(v ShopScreenView) bool {
	switch o := a.shopDragOrigin; o.Kind {
	case ShopControlShelfCell:
		return true
	case ShopControlTableCell:
		return o.Index >= 0 && o.Index < len(v.Table) && !v.Table[o.Index].Mine
	}
	return false
}

// shopDragItemPresent is Viewer.dragItemPresent's own contract (pkg/ui/
// inventory.go), restated for the App-level shop drag machine: the carried
// item's own picture for this frame, or false for a frame carrying nothing —
// a drag not yet past TapSlop, one whose origin resolved to no icon, or one
// with no cursor position to carry it to.
func (a *App) shopDragItemPresent() (*image.RGBA, bool) {
	if !a.shopDragArmed || a.shopDragMoved < TapSlop || a.shopDragIcon == nil {
		return nil, false
	}
	return a.shopDragIcon, true
}

// worldMapTickInterval gates how often WorldMapTick fires (`DIV-136`,
// authored). `TOWN-121` states the known paint driver cannot advance more
// than once every 99 ms and grades its own real-time cadence Unknown; this is
// the closest defensible reading — just above that floor, the fastest a
// message timer gated at "more than 99 ms" could fire.
const worldMapTickInterval = 100 * time.Millisecond

func (a *App) stepTown(in appInput) { a.stepTownAt(in, time.Now()) }

func (a *App) clickTownSurface(c TownSurfaceControl, double bool) bool {
	switch c.Kind {
	case TownSurfaceControlPrevious, TownSurfaceControlNext, TownSurfaceControlMode, TownSurfaceControlBook:
		// Campaign-pane controls D/E/C play click00 before their transition.
		a.playUISound(UISoundCampaignPanel)
	case TownSurfaceControlButton:
		a.playUISound(UISoundCommonControl)
	}
	return a.flow.clickTownSurface(c, double)
}

func (a *App) clickShop(c ShopControl) bool {
	if c.Kind == ShopControlBook {
		// Child 3's cue follows the mutation, unlike panel/control clicks.
		ok := a.flow.clickShop(c)
		if ok {
			a.playUISound(UISoundBookToggle)
		}
		return ok
	}
	switch c.Kind {
	case ShopControlPickerPrev, ShopControlPickerNext, ShopControlCharacterMode:
		a.playUISound(UISoundCampaignPanel)
	case ShopControlButton, ShopControlArrowUp, ShopControlArrowDown,
		ShopControlPackLeft, ShopControlPackRight,
		ShopControlShelfPick:
		a.playUISound(UISoundCommonControl)
	}
	return a.flow.clickShop(c)
}

func (a *App) stepTownAt(in appInput, now time.Time) {
	if p, ok := a.windowToNativeFrame(in.CursorX, in.CursorY); ok {
		a.townCursor, a.hasTownCursor = p, true
	} else {
		a.hasTownCursor = false
	}
	if town, ok := a.flow.town.(TownSquareAnimator); ok && atTownSquare(a.flow.town) && !in.Unfocused {
		p := image.Pt(-1, -1)
		if a.hasTownCursor {
			p = a.townCursor
		}
		town.TownSquarePointer(p)
	}
	world, onMap := townWorldMapScreen(a.flow.town)
	if !onMap {
		// The next entry's first tick must fire at once rather than waiting
		// out an interval left over from an earlier visit.
		a.worldMapTickAt = time.Time{}
	}
	if onMap {
		if a.hasTownCursor {
			world.WorldMapHover(a.townCursor)
		} else {
			world.WorldMapHover(image.Pt(-1, -1))
		}
		if a.worldMapTickAt.IsZero() || now.Sub(a.worldMapTickAt) >= worldMapTickInterval {
			a.worldMapTickAt = now
			a.flow.applyTownAction(world.WorldMapTick())
			if !world.AtWorldMap() || a.flow.screen != ScreenTown {
				a.syncViewerLayout()
				a.worldMapTickAt = time.Time{}
				return // Arrival consumes this frame; stale map input cannot select a new trip.
			}
		}
		switch {
		case in.WheelY > 0 || in.Up:
			world.WorldMapMove(-1)
		case in.WheelY < 0 || in.Down:
			world.WorldMapMove(+1)
		case in.Enter:
			a.flow.applyTownAction(world.WorldMapChoose())
			a.syncViewerLayout()
		case in.PrimaryReleased && a.hasTownCursor:
			a.flow.applyTownAction(world.WorldMapClick(a.townCursor))
			a.syncViewerLayout()
		}
		return
	}
	if a.flow.townList == nil {
		return
	}
	shopView, inShop := townShopScreen(a.flow.town)
	if inShop {
		a.expireShopUseTap(now)
	}
	surface, inSurface := townSurfaceScreen(a.flow.town)
	if inSurface {
		a.tooltipSurface = &surface
	}
	if picture, dialogue := townDialogue(a.flow.town); dialogue {
		owner := a.townDialoguePointerOwner(picture)
		button, shown := townDialogueButton(a.flow.town)
		p, insideFrame := a.windowToNativeFrame(in.CursorX, in.CursorY)
		mouseAdvance := a.stepDialoguePointer(in, owner, shown && insideFrame && p.In(button))
		if in.Enter {
			a.flow.advanceTownDialogue()
			a.syncViewerLayout()
			return
		}
		if mouseAdvance {
			a.flow.advanceTownDialogue()
			a.syncViewerLayout()
		}
		return
	}
	var tip TipPanelView
	if inSurface {
		tip = surface.Tip
	} else if inShop {
		tip = shopView.TipPanel
	} else if square, ready := townSquareView(a.flow.town); ready {
		tip = square.Tip
	}
	if tip.Showing() || a.townTipPress.Holds() {
		p, inside := a.windowToNativeFrame(in.CursorX, in.CursorY)
		if a.stepTownTipPointer(in, tip, p, inside) {
			a.resetTownSurfacePair()
			a.clearShopDrag()
			return
		}
	}
	// Advertised panel shortcuts share the same action as the painted corner.
	// Dialogue and focus gates above retain ownership of their input.
	if !in.Unfocused && (in.PaneMode || in.Book) {
		if in.Viewer.PrimaryDown {
			a.suppressPrimaryRelease = true
		}
		if inShop {
			kind := ShopControlBook
			if in.PaneMode {
				kind = ShopControlCharacterMode
			}
			a.clearShopDrag()
			a.clickShop(ShopControl{Kind: kind})
			a.syncViewerLayout()
			return
		}
		if inSurface && surface.Hero.HasSubject && (in.PaneMode || !surface.Hero.NoBook) {
			kind := TownSurfaceControlBook
			if in.PaneMode {
				kind = TownSurfaceControlMode
			}
			a.townSurfacePress.Clear()
			a.clickTownSurface(TownSurfaceControl{Kind: kind}, false)
			a.syncViewerLayout()
			return
		}
	}
	if inSurface {
		if in.PrimaryPressed {
			// A new press replaces whatever the latch held.
			a.townSurfacePress.Clear()
			if p, ok := a.windowToNativeFrame(in.CursorX, in.CursorY); ok {
				if c, hit := TownSurfaceControlAt(surface, p); hit {
					a.townSurfacePress.Press(townSurfaceLatchID(c), true)
					if presser, ok := a.flow.town.(TownSurfacePresser); ok {
						presser.TownSurfacePress(c)
					}
					if c.Kind == TownSurfaceControlCell {
						key := surface.Cells[c.Index].Key
						double := surface.Kind == TownSurfaceTavern && a.townSurfaceReleased && surface.Tip.Revision == a.townSurfaceRevision && key == a.townSurfaceKey && c == a.townSurfaceClick && !a.townSurfaceAt.IsZero() && now.Sub(a.townSurfaceAt) >= 0 && now.Sub(a.townSurfaceAt) <= 350*time.Millisecond
						a.clickTownSurface(c, double)
						a.townSurfaceAnimationTick = 0
						a.townSurfaceKey, a.townSurfaceReleased = key, false
						a.townSurfaceRevision = surface.Tip.Revision
						if double {
							a.townSurfaceClick, a.townSurfaceAt = TownSurfaceControl{}, time.Time{}
						} else {
							a.townSurfaceClick, a.townSurfaceAt = c, now
						}
						a.syncViewerLayout()
					} else {
						a.townSurfaceClick, a.townSurfaceAt = TownSurfaceControl{}, time.Time{}
					}
				}
				if !a.townSurfacePress.Holds() {
					a.townSurfaceClick, a.townSurfaceAt = TownSurfaceControl{}, time.Time{}
				}
			}
		}
		if in.PrimaryReleased {
			if p, ok := a.windowToNativeFrame(in.CursorX, in.CursorY); ok {
				c, hit := TownSurfaceControlAt(surface, p)
				_, fire := a.townSurfacePress.Release(townSurfaceLatchID(c), hit)
				if fire && c.Kind != TownSurfaceControlCell {
					a.clickTownSurface(c, false)
					a.syncViewerLayout()
				} else if fire {
					a.townSurfaceReleased = true
				} else {
					a.townSurfaceClick, a.townSurfaceAt = TownSurfaceControl{}, time.Time{}
					a.townSurfaceReleased = false
				}
			} else {
				a.resetTownSurfacePair()
			}
			a.townSurfacePress.Clear()
		}
		return
	}
	// Mouse gestures retain the shelf, pack and trade-table routes. Panel
	// shortcuts are dispatched above; Escape is handled before screen input.
	if inShop {
		// THE DRAG MACHINE'S OWN PER-FRAME ARM (1005 round 2, command.go's
		// own "runs on EVERY SWALLOWED FRAME" doc restated): a press already
		// armed keeps its own high-water distance current, and the doll's
		// own suppression is pushed every frame a doll origin has crossed
		// TapSlop — including the frame a release clears a.shopDragArmed,
		// which is why this runs BEFORE the release case below rather than
		// only in the press arm.
		if a.shopDragArmed {
			// THIS SUMS FRAME PIXELS, AFTER WindowToFrame's OWN SCALE DIVISION —
			// command.go's mission-side accumulator sums raw window pixels instead,
			// so the same TapSlop reads as a different real distance on the two
			// screens (round-2 adversarial review, eleventh pass, judged and left
			// open — command.go's own TapSlop doc, closure.md).
			if p, ok := a.windowToNativeFrame(in.CursorX, in.CursorY); ok {
				if d := absInt(p.X-a.shopPressX) + absInt(p.Y-a.shopPressY); d > a.shopDragMoved {
					a.shopDragMoved = d
				}
			}
		}
		a.flow.cursor.SetCursor(a.shopDragCursorName(shopView, in))
		suppressSlot := 0
		if a.shopDragArmed && a.shopDragOrigin.Kind == ShopControlDoll && a.shopDragMoved >= TapSlop {
			suppressSlot = a.shopDragOrigin.Index + 1
		}
		a.flow.suppressShopDoll(suppressSlot)

		// THE WHEEL TURNS THE REGION UNDER THE POINTER (owner, DIV-005). A wheel
		// event whose pointer is not over the frame at all,
		wheelAt, onFrame := a.windowToNativeFrame(in.CursorX, in.CursorY)
		switch {
		case in.WheelY > 0 && onFrame:
			a.flow.scrollShop(wheelAt, -1)
		case in.WheelY < 0 && onFrame:
			a.flow.scrollShop(wheelAt, +1)
		case in.PrimaryPressed:
			// THE DRAG MACHINE'S OWN ARM (1005 round 2): every press over the
			// doll, the shelf grid or the pack strip is a candidate origin,
			// resolved and held rather than acted on — a press that never
			// crosses TapSlop is judged as a tap below, on release, exactly
			// as command.go's own dragCandKind is.
			//
			// a.shopDragIcon = nil HERE IS DEFENSIVE, NOT INDEPENDENTLY WITNESSED
			// (round-2 adversarial review, item 2's own mutation run):
			// shopDragItemPresent gates on a.shopDragArmed, which the release arm
			// below always clears first, so no reachable
			// PrimaryPressed-without-a-prior-PrimaryReleased sequence exists in this
			// build for it to guard against. It mirrors the release arm's own reset
			// on the same terms dollSuppressOwner's guard (viewer.go) already stands
			// on.
			if p, ok := a.windowToNativeFrame(in.CursorX, in.CursorY); ok {
				if c, hit := shopGridControlAt(shopView, p); a.shopUseTap != nil && (!hit || c != a.shopUseTap.control) {
					a.shopUseTap = nil
				}
				a.shopPressX, a.shopPressY = p.X, p.Y
				a.shopDragMoved = 0
				a.shopButtonPress.Clear()
				a.shopDragOrigin, a.shopDragArmed = shopGridControlAt(shopView, p)
				if !a.shopDragArmed {
					if c, hit := shopScreenControlAt(shopView, p); hit && c.Kind == ShopControlButton {
						a.shopButtonPress.Press(c.Index, shopView.Live[c.Index])
					}
				}
				a.shopDragOriginBase = shopDragGridBase(shopView, a.shopDragOrigin)
				a.shopDragIcon = shopDragOriginIcon(shopView, a.shopDragOrigin)
			}
		case in.PrimaryReleased:
			origin := shopDragOriginAt(shopView, a.shopDragOrigin, a.shopDragOriginBase)
			armed, moved := a.shopDragArmed, a.shopDragMoved
			a.shopDragArmed = false
			a.flow.cursor.SetCursor(a.shopCursorRestore())
			// The release ends the preview on this frame, even when it cancels
			// without sending a model action. Leaving the suppressed slot on the
			// town screen made the next press read yesterday's undressed mask and
			// miss the item the player had just returned to the doll.
			a.flow.suppressShopDoll(0)
			// a.shopDragIcon = nil HERE outlives its own observable effect:
			// shopDragItemPresent already answers false the instant
			// a.shopDragArmed above does, on the same line. It is kept so
			// a.shopDragIcon itself — read nowhere but here and the press
			// arm — never holds a stale pointer between two gestures for a
			// caller that comes to read it directly.
			a.shopDragIcon = nil
			p, onFrame := a.windowToNativeFrame(in.CursorX, in.CursorY)
			switch {
			case a.shopButtonPress.Holds():
				c, hit := shopScreenControlAt(shopView, p)
				if _, fire := a.shopButtonPress.Release(c.Index, onFrame && hit && c.Kind == ShopControlButton); fire {
					a.clickShop(c)
					a.syncViewerLayout()
				}
			case armed && moved >= TapSlop:
				a.shopUseTap = nil
				// THE DRAG MACHINE'S OWN RELEASE: a candidate that crossed
				// TapSlop is judged by where it LANDS, never by the ordinary
				// click geometry — a release naming no recognised surface at
				// all (dest, ok := ...; !ok) is `ITEM-DROP-008`'s counterpart
				// for a town screen, DIV-088's own "no ground on a town
				// screen": nothing is sent across the seam, so nothing moves
				// and the item stays exactly where the drag began.
				//
				// held is the origin with this release frame's Shift level: a
				// drag reads the modifier when it lands, as a click does
				// (DIV-046, DIV-047, DIV-1463).
				held := origin
				held.Shift = in.ShiftHeld
				if onFrame {
					// THE ORIGIN'S OWN IDENTITY IS ASKED FIRST, AND SEPARATELY FROM
					// shopGridControlAt (round-2 adversarial review, tenth pass,
					// counterexample 1): a doll origin whose drag has crossed TapSlop
					// already carries a suppressed SlotMask (ShopSuppressDoll's own
					// per-frame push, above), with the origin's own slot cleared —
					// shopGridControlAt, reading that mask, can never answer the origin's
					// own index again for as long as the drag stays armed, so a release
					// back on the exact pixel the press began on found no recognised
					// surface at all (`ok` false below, before this fix) and the whole
					// gesture — ShopClick AND ShopDrag alike — was silently dropped.
					// shopReleaseIsOrigin reads OrdinaryDollMask for a doll origin, the
					// mask in force before this drag suppressed anything, so the origin's
					// identity does not change because the drawn picture did.
					if shopReleaseIsOrigin(shopView, origin, p) {
						// A SAME-CELL TREMOR remains a click for the three
						// rectangular grids. A DOLL DRAG is different: once it
						// crossed TapSlop the item is visibly in hand and a
						// release anywhere back on its original painted layer
						// cancels the move. Treating that release as ShopClick
						// unequipped the item even though the player dropped it
						// back on the doll. An actual doll tap still reaches the
						// uncrossed arm below and keeps tap-to-unequip.
						if origin.Kind != ShopControlDoll {
							a.flow.clickShop(held)
							a.syncViewerLayout()
						}
					} else if dest, ok := shopGridControlAt(shopView, p); ok {
						// A release naming no recognised surface at all
						// (`ok` false) is `ITEM-DROP-008`'s counterpart for a
						// town screen, DIV-088's own "no ground on a town
						// screen": nothing is sent across the seam, and the
						// item stays exactly where the drag began.
						//
						// dest.Kind == origin.Kind ALONE — the check this
						// replaced, before shopReleaseIsOrigin's own fix —
						// also matched a GENUINE drag between two DIFFERENT
						// cells of one family (shelf 0 to shelf 3, or any
						// doll slot to another, since every doll pixel is its
						// own cell), which shopReleaseIsOrigin's exact-cell
						// comparison above already excludes: reaching this
						// branch means dest, whatever it is, is not origin,
						// so every release here is a genuine cross-cell drag.
						// ShopDrag's own switch (shopview.go) carries no case
						// for a same-family pair, so a same-family,
						// different-cell release is a no-op there — nothing
						// further is asked of it here.
						a.flow.dragShop(held, dest)
						a.syncViewerLayout()
					} else if origin.Kind != ShopControlDoll && shopPaneTakesItemAt(p) {
						// THE PANE TAKES THE ITEM ANYWHERE IN ITS RECTANGLE, corners and
						// statistics card included (TOWN-348, DIV-315): shopGridControlAt
						// answered !ok above because no slot's layer is painted at p, or
						// because p is under a corner, and neither is a reason to keep the
						// item. ShopDrag's doll-destination cases match on Kind alone and
						// never consult Index, so a member wearing nothing can still be
						// dressed by a drag; Index 0 costs nothing here. Hidden equipment
						// stays unavailable as a drag origin or a click-to-unequip target.
						a.flow.dragShop(held, ShopControl{Kind: ShopControlDoll})
						a.syncViewerLayout()
					}
				}
			case armed && origin.Kind == ShopControlDoll:
				// A DOLL PRESS THAT NEVER CROSSED TapSlop is round 1's own
				// tap-to-unequip, restated for the shop (spec bullet 2):
				// ShopClick's own new ShopControlDoll arm, so this gesture
				// and a genuine click share the one mutation door.
				a.clickShop(origin)
				a.syncViewerLayout()
			case onFrame:
				if c, hit := shopScreenControlAt(shopView, p); hit {
					if c.Kind == ShopControlButton {
						break
					}
					// THE MODIFIER IS FILLED IN HERE, ONE STATEMENT AFTER THE HIT TEST
					// (owner, DIV-046, DIV-047). ShopControlAt is a pure geometry test with
					// no notion of a held key; this is the one place a table or pack click
					// learns whether it moves one unit or the whole stack.
					c.Shift = in.ShiftHeld
					if !a.useShopTap(shopView, c, now) {
						a.clickShop(c)
					}
					a.syncViewerLayout()
				}
			}
		}
		return
	}

	// THE SQUARE'S OWN PICTURE IS DRIVEN WITH THE MOUSE ALONE, ONCE ITS ART IS
	// READY (1016 round-2 review, the shop's own precedent, the comment above
	// `inShop`). ComposeTownSquare (drawTown) replaces the row-button grid with
	// the shipped picture and draws no selection cursor over it — townList
	// still HOLDS a selection, but nothing on screen shows it moving — so
	// before this guard the wheel, the arrows and Enter silently moved that
	// invisible selection and Enter silently chose it, on every install that
	// ships the picture, which is the install this story is for. Escape is
	// unaffected: it is read before every per-screen dispatch (flow.go's own
	// escape(), never here), exactly as the shop's own comment states for
	// itself. An install missing the picture is not ready here, so it falls
	// through to the switch below unchanged — the row-button layout's own
	// keyboard path is the only way to use such an install at all.
	if v, ready := townSquareView(a.flow.town); ready {
		if in.PrimaryReleased {
			if p, ok := a.windowToNativeFrame(in.CursorX, in.CursorY); ok {
				if button, shown := townDialogueButton(a.flow.town); shown && p.In(button) {
					a.activatePicker(a.flow.townList, a.flow.chooseTown)
					a.syncViewerLayout()
					return
				}
				// THE SQUARE'S OWN PICTURE IS HIT-TESTED BY ITS SCENE, which reads
				// the install's own mask rather than a rectangle this build authored.
				if c, hit := v.Scene.ControlAt(p); hit {
					switch c.Kind {
					case TownSquareControlDoor:
						// THE DOOR INDEX FEEDS THE SAME SEAM THE ROW-BUTTON GRID ALWAYS HAS
						// (Select then chooseTown), so Choose(i) — and every headless
						// scenario driving it — is unchanged by this story.
						a.flow.townList.Select(c.Door)
						a.flow.chooseTown()
						a.syncViewerLayout()
					case TownSquareControlMenu:
						// THE STATUE OPENS THE SAME MINI-MENU ESCAPE ALREADY DOES AT THE
						// SQUARE (flow.go's escape, ScreenTown arm). No new save/load
						// plumbing: 0143's seam already carries SAVE and LOAD once the menu is
						// up.
						a.flow.openGameMenu(ScreenTown)
					}
				}
			}
		}
		return
	}

	switch {
	case in.WheelY > 0:
		a.flow.townList.Move(-pickerWheelRows)
	case in.WheelY < 0:
		a.flow.townList.Move(+pickerWheelRows)
	case in.Up:
		a.flow.townList.Move(-1)
	case in.Down:
		a.flow.townList.Move(+1)
	case in.Enter:
		a.activatePicker(a.flow.townList, a.flow.chooseTown)
		a.syncViewerLayout()
	case in.PrimaryReleased:
		if p, ok := a.windowToNativeFrame(in.CursorX, in.CursorY); ok {
			if button, shown := townDialogueButton(a.flow.town); shown && p.In(button) {
				a.activatePicker(a.flow.townList, a.flow.chooseTown)
				a.syncViewerLayout()
				return
			}
			row, hit := a.flow.townList.RowAt(p)
			if atTownSquare(a.flow.town) {
				row, hit = townSquareRowAt(p.X, p.Y)
			}
			if hit {
				a.flow.townList.Select(row)
				a.activatePicker(a.flow.townList, a.flow.chooseTown)
				a.syncViewerLayout()
			}
		}
	}
}

// stepList is stepTown's own dispatch over any list-and-choose screen: the
// wheel and the arrows move the selection, Enter chooses it, and a primary
// release inside the list chooses the row under the cursor.
//
// IT IS FACTORED OUT RATHER THAN COPIED A THIRD AND FOURTH TIME. The mini-menu
// and the load window are two more lists over the same *Picker the map list and
// the town already use, and a copy of this dispatch per screen is four chances
// for one of them to differ from the others in a way nobody can see until they
// press a key.
//
// syncViewerLayout RIDES BOTH CHOICES for stepTown's own reason: either screen
// can enter a map, and a viewer adopted this frame has not been told the window
// size yet.
func (a *App) stepList(in appInput, list *Picker, choose func()) {
	if list == nil {
		return
	}
	switch {
	case in.WheelY > 0:
		list.Move(-pickerWheelRows)
	case in.WheelY < 0:
		list.Move(+pickerWheelRows)
	case in.Up:
		list.Move(-1)
	case in.Down:
		list.Move(+1)
	case in.Enter:
		a.activatePicker(list, choose)
		a.syncViewerLayout()
	case in.PrimaryReleased:
		if p, ok := a.windowToNativeFrame(in.CursorX, in.CursorY); ok {
			if row, ok := list.RowAt(p); ok {
				list.Select(row)
				a.activatePicker(list, choose)
				a.syncViewerLayout()
			}
		}
	}
}

// holdMapUnderMenu runs the three statements the map arm runs above its own
// popup gate, on the frames the in-game menu is standing over a map.
//
// EACH OF THE THREE IS HERE FOR THE REASON THE MAP ARM RECORDS FOR IT. The
// cadence call is what DECLARES the stop, so skipping it would leave the world
// never told; the advance consumes the far side's pacing baseline on every call
// including the stopped ones, so skipping it would leave the held span owed and
// pay it in one jump on dismissal; and the viewer's step takes the ambient
// clock's baseline the same way, where the debt is worse because that clock's
// catch-up is unbounded.
//
// THE FIVE CADENCE KEYS ARE NAILED TO FALSE. The menu takes every input the
// surface behind it would have had, and stepCadence already refuses to move
// either field while a popup is open — so this passes what the player cannot
// press rather than relying on that refusal alone.
//
// The viewer's own input IS passed through, exactly as the map arm passes it.
// Its popup gate keeps the cursor position live and drops everything else,
// which is what leaves the readout resolving a cell behind the dim.
func (a *App) holdMapUnderMenu(in appInput, now time.Time) {
	a.holdMapUnder(a.flow.menuBack, in, now)
}

func (a *App) holdMapUnderDocuments(in appInput, now time.Time) {
	a.holdMapUnder(a.flow.docBack, in, now)
}

// holdMapUnder is the body holdMapUnderMenu and holdMapUnderDocuments share:
// the three statements the map arm runs above its own popup gate, gated on
// the screen the caller was armed from being the map. One implementation
// rather than two copies with different field names, because the two
// screens' own "back" fields (menuBack, docBack) are not the same field and
// must not be conflated at the call site.
func (a *App) holdMapUnder(back Screen, in appInput, now time.Time) {
	if back != ScreenMap {
		return
	}
	a.flow.stepCadence(false, false, false, false, false)
	if tick := a.flow.tick; tick != nil {
		tick()
	}
	if v := a.flow.viewer; v != nil {
		v.step(in.Viewer, now)
	}
}

func (a *App) beforeGameMenuAction(action gameMenuAction) {
	a.playUISound(UISoundCommonControl)
	if action == gameMenuTestSound {
		// Options action 7 is itself the fixed sword-sample request.
		a.playUISound(UISoundOptionsTest)
	}
}

func (a *App) chooseGameMenu() {
	rows := a.flow.menuRows()
	if a.flow.menuList != nil {
		if i, ok := a.flow.menuList.Choose(); ok && i >= 0 && i < len(rows) {
			a.beforeGameMenuAction(rows[i].Action)
		}
	}
	a.flow.chooseGameMenu()
}

// stepGameMenu is the in-game menu's own input.
//
// IT IS NOT stepList. The panel has its own geometry, so a pointer release is
// hit-tested against the row rectangles the paint draws rather than against the
// map list's line pitch — what is clickable is then what is drawn. It also has
// accelerators, which no other list in this front-end has.
//
// A RELEASE THAT FALLS ON NO ROW DOES NOTHING, inside the panel or outside
// it (spec AU-6). MENU-INPUT-016: the panel is the root's capture object, so a
// click outside it reaches no other child; nothing behind this panel is
// reachable while it stands, because the map arm never runs.
func (a *App) stepGameMenu(in appInput) bool {
	if a.flow.menuPage == gameMenuSoundOptionsPage && a.flow.soundOptions.Read != nil {
		return a.stepSoundOptions(in)
	}
	if a.flow.menuPage == gameMenuQuestObjectivesPage {
		a.stepQuestObjectives(in)
		return false
	}
	list := a.flow.menuList
	if list == nil {
		return false
	}
	if a.flow.menuPage == gameMenuGameOptionsPage && a.flow.gameOptions.Read != nil && a.stepGameOptionsPointer(in) {
		return false
	}
	options := a.flow.menuPage == gameMenuGameOptionsPage && a.flow.gameOptions.Read != nil
	if in.PrimaryPressed {
		if options {
			a.pressGameOptions(a.windowToNativeFrame(in.CursorX, in.CursorY))
		} else {
			a.flow.menuPress.Press(a.gameMenuButtonAt(in))
		}
	}
	if in.PrimaryReleased && (in.Up || in.Down || in.Enter || in.Typed != "") {
		// A key in the release's tick takes the tick; the release
		// activates nothing and drops the latch.
		a.flow.menuPress.Clear()
	}
	switch {
	case in.Up:
		list.Move(-1)
	case in.Down:
		list.Move(+1)
	case in.Enter:
		a.chooseGameMenu()
		a.syncViewerLayout()
	case in.Typed != "":
		// RANGED AS RUNES, NOT BYTES (1014, MENU-KEY-013). in.Typed is
		// ebiten.AppendInputChars' own UTF-8, and a Cyrillic character is
		// more than one byte there; chooseGameMenuAccelerator takes the whole
		// rune and encodes it to the install's own byte itself, the same
		// split EditName already uses for a typed character name.
		for _, r := range in.Typed {
			if a.flow.chooseGameMenuAccelerator(r, a.beforeGameMenuAction) {
				a.syncViewerLayout()
				break
			}
		}
	case in.PrimaryReleased:
		if options {
			a.releaseGameOptions(a.windowToNativeFrame(in.CursorX, in.CursorY))
			return false
		}
		at, inside := a.gameMenuButtonAt(in)
		row, activated := a.flow.menuPress.Release(at, inside)
		if !activated {
			return false
		}
		list.Select(row)
		a.chooseGameMenu()
		a.syncViewerLayout()
	}
	return a.flow.takeMenuExit()
}

// gameMenuButtonAt is the enabled menu button under the pointer, as a list
// index. Status rows and disabled rows are not buttons a press can latch.
func (a *App) gameMenuButtonAt(in appInput) (int, bool) {
	p, ok := a.windowToNativeFrame(in.CursorX, in.CursorY)
	list := a.flow.menuList
	if !ok || list == nil {
		return 0, false
	}
	top, count := list.Visible()
	row, ok := gameMenuRowAt(a.flow.menuPanelSurface(), count, p)
	rows := a.flow.menuRows()
	if !ok || top+row >= len(rows) || rows[top+row].Status || !rows[top+row].Enabled {
		return 0, false
	}
	return top + row, true
}

// stepDocuments drives the campaign documents panel: hover, press and the
// three controls (MENU-DOC-009).
//
// THE PRESS IS LATCHED because this snapshot carries the primary button's two
// EDGES and no level (appInput's own fields). A control is taken on the
// RELEASE that lands on the control the press began on, which is this
// package's own rule everywhere else a button is clicked, and it is what makes
// a press dragged off a control cancel rather than fire.
//
// A POINTER OUTSIDE THE FRAME clears the hover and keeps the press: the
// letterbox is not a control, and a player whose pointer crossed the black
// border mid-press has not released anything.
func (a *App) stepDocuments(in appInput) {
	p := a.flow.docPanel
	if p == nil {
		return
	}
	pt, inFrame := a.windowToNativeFrame(in.CursorX, in.CursorY)
	control := docNoControl
	if inFrame {
		control, _ = documentControlAt(pt.X, pt.Y)
	}
	p.hover = control
	if in.PrimaryPressed {
		p.press.Press(control, control != docNoControl)
	}
	if !in.PrimaryReleased {
		return
	}
	pressed, activated := p.press.Release(control, control != docNoControl)
	if !activated {
		return
	}
	a.playUISound(UISoundCommonControl)
	switch pressed {
	case docControlLeft:
		p.step(-1)
	case docControlRight:
		p.step(+1)
	case docControlOK:
		// OK IS THE ORIGINAL'S ONE WAY OUT (MENU-DOC-009: the arm posts
		// 0x445 and tears the window down). closeDocuments is the same
		// statement Escape reaches.
		a.flow.closeDocuments()
		a.syncViewerLayout()
	}
}

// SetDocuments installs the campaign documents seam: where the collection
// comes from, the panel's eleven bitmaps, and the font its text pages are
// drawn with.
//
// IT IS CALLED ONCE, BEFORE RUN, and it is the only writer. Every argument may
// be nil, and an application told none of them has the panel's entry point
// answering that there is nothing to show — which is what cmd/mapview and
// every hand-assembled test flow get.
func (a *App) SetDocuments(src DocumentSource, art *DocumentPanelArt, font *text.Font) {
	a.flow.docSrc, a.flow.docArt, a.flow.docFont = src, art, font
}

// SetSaveSeams installs the save/load seam. All three may be nil and a front
// end that calls this with nils is the front end this story inherits: the
// mini-menu still opens, and SAVE and LOAD report that there is no store.
func (a *App) SetSaveSeams(save SaveGame, list SaveList, load LoadGame) {
	a.flow.saveGame, a.flow.saveList, a.flow.loadGame = save, list, load
}

// SetGameMenuSettings installs the two settings destinations' seams. The tips
// callbacks reach the existing persistent preference. The sound callbacks
// apply audio state and persist it through the configured game preference store.
// Any callback may be nil; its page then names
// the setting as unavailable and still provides a return row.
func (a *App) SetGameMenuSettings(tips GameMenuTipsSource, setTips GameMenuTipsSink,
	sound GameMenuSoundSource, setSound GameMenuSoundSink) {
	a.flow.menuTips, a.flow.setMenuTips = tips, setTips
	a.flow.menuSound, a.flow.setMenuSound = sound, setSound
}

// SetWords replaces the program-chosen words this application draws.
//
// IT IS CALLED ONCE, BEFORE RUN, and it is the only writer. The front end
// resolves the set from the install at construction and hands it over here; the
// flow carries it to every viewer it opens, and to a viewer already open, so
// the call cannot land at a moment where half the application has the old set.
//
// An application never told any words draws this build's authored English,
// which is what cmd/mapview and every hand-assembled test flow do.
// THE FONT COMES WITH THE WORDS. An install word is bytes in that install's own
// code page, and the font is what converts them; handing them over separately
// would allow an application holding Russian words and an ASCII font, which
// draws a row of replacement glyphs. A nil font keeps the debug-font path, which
// is correct for the authored English.
//
// ENCODE COMES WITH THEM TOO (1014, MENU-KEY-013). A keyboard rune has to
// become the SAME install byte a label's own `~` mark carries before the two
// can be compared, and that conversion is pkg/formats/textinput's — a formats
// leaf pkg/ui may not import (internal/archtest's DAG). pkg/game already
// crosses that exact conversion as a plain function value for chargen's name
// field (chargen.go's EncodeName); this is the same seam for the menu. A nil
// encode keeps the ASCII-only behaviour this package had before this story,
// which is what cmd/mapview and every hand-assembled test flow do.
func (a *App) SetWords(w Words, f *text.Font, encode func(rune) (byte, bool)) {
	a.flow.words, a.flow.menuFont, a.flow.encodeMenuKey = w, f, encode
	if a.flow.viewer != nil {
		a.flow.viewer.SetWords(w)
	}
}

// stepChargen drives the generation screen: Up/Down move the model's focus,
// Left/Right adjust whatever row is focused, and Enter confirms.
//
// A.flow.chargen IS ASSUMED SET. It is nil only when this arm is reached
// without OpenChargen having armed the screen first — unreachable through the
// application's own transitions, since ScreenChargen is never assigned any
// other way — but this method still checks and returns rather than trust
// that: the same "a row nothing can make sense of is a row that never moves"
// rule chargen.go's own header states for a malformed setup applies here to a
// malformed CALLER.
//
// CONFIRMING IS THE ONE BRANCH THAT CAN LEAVE THIS SCREEN, and it does so
// through the SAME flow.enter the picker's own choose() and OpenMission
// already use — a third entry path would drift from the cadence rung and
// the command mode both of those already establish. Result()'s own refusal
// (AC-6) is read here rather than duplicated: an illegal spread answers
// (ChargenResult{}, false), and this method's whole job on that branch is to
// say so on the message line and change nothing else — not the screen, not
// a call to begin.
//
// begin AND THE OPENER IT RETURNS ARE THE TWO THINGS THAT CAN STILL FAIL
// AFTER Result() SAYS YES: the wiring tier can decline a legal spread (no
// base row resolves to anything usable, say) and a MapOpener can fail exactly
// as OpenMission's own open() can. Either error is reported on the message
// line and the chargen screen is left showing, unadvanced — never a partial
// transition, never a panic.
//
// A NIL OPENER WITH NO ERROR IS NOT A FAILURE. It is the same "nothing to
// enter" answer a nil MapLoader tick or order already means elsewhere in
// this file: begin ran, named no failure, and simply had nothing to open.
func (a *App) stepChargen(in appInput, now time.Time) {
	c := a.flow.chargen
	if c == nil {
		return
	}
	if c.Stage() == PreCreateStage {
		a.chargenRepeat = chargenRepeat{}
		a.stepPreCreate(c, in, now)
		return
	}
	if c.setup.PreCreate != nil {
		a.stepChargenDetailed(c, in, now)
		return
	}
	c.EditName(in.Typed, in.Backspace)
	if in.Up {
		c.Move(-1)
	}
	if in.Down {
		c.Move(1)
	}
	if in.Left {
		c.Adjust(-1)
	}
	if in.Right {
		c.Adjust(1)
	}
	if !in.Enter {
		return
	}
	result, ok := c.Result()
	if !ok {
		a.flow.msg = "cannot confirm: the spread is not legal"
		return
	}
	if a.flow.chargenBegin == nil {
		a.flow.msg = "cannot begin: no callback configured"
		return
	}
	open, err := a.flow.chargenBegin(result)
	if err != nil {
		a.flow.msg = err.Error()
		return
	}
	if open == nil {
		if a.flow.screen == ScreenTown {
			a.flow.resetTimedAutosave()
		}
		return
	}
	v, tick, order, cadence, affect, advance, attack, grab, stance, march, err := open()
	if err != nil {
		a.flow.msg = err.Error()
		return
	}
	a.flow.enter(v, tick, order, cadence, affect, advance, attack, grab, stance, march)
	a.syncViewerLayout()
}

// stepChargenDetailed gives the pictorial stage one arbitration point. A
// pointer press or completed release, Enter, movement, and text input are
// mutually exclusive in this order, so a frame cannot both edit and launch.
func (a *App) stepChargenDetailed(c *Chargen, in appInput, now time.Time) {
	a.observeChargenTipPress(in)
	a.flow.cursor.SetCursor(c.pageCursor())
	a.chargenHeld = in.Viewer.PrimaryDown || in.PrimaryPressed
	defer a.paintSkillCycle(c, in, now)
	repeat := false
	if l := c.layout(); l != nil && l.Detail.Repeat != nil {
		repeat = a.chargenRepeat.tickEvery(in, l.Detail.Repeat.DelayTicks, l.Detail.Repeat.IntervalTicks)
	}
	// THE SHOWING PANEL SWALLOWS EVERY PRESS AND RELEASE INSIDE ITS OWN RECT
	// BEFORE THE PAGE (1022 spec B5, stepPreCreate's own precedent above):
	// The detail tip rectangle reaches over the skill column, so a press meant for the
	// panel's own close/toggle controls must not fall through to
	// detailedControlAt below and land on a skill or stat cell instead.
	if p, ok := a.windowToNativeFrame(in.CursorX, in.CursorY); ok && (in.PrimaryPressed || in.PrimaryReleased) {
		if kind, consumed := TipPanelControlAt(c.TipPanel(), p); consumed {
			if in.PrimaryReleased {
				switch kind {
				case TipControlClose:
					c.CloseTip()
				case TipControlToggle:
					c.ToggleTips()
				}
			}
			a.chargenPress.Clear()
			if in.PrimaryPressed {
				a.chargenChoiceClick, a.chargenChoiceAt = chargenNone, time.Time{}
			}
			return
		}
	}
	hit := chargenNone
	if p, ok := a.windowToNativeFrame(in.CursorX, in.CursorY); ok {
		hit = detailedControlAt(c, p)
	}
	a.chargenHover = hit
	a.chargenHoverText = ""
	if c.setup.Detailed != nil && hit >= chargenSkill0 && hit <= chargenSkill4 && !a.chargenPress.Holds() {
		a.chargenHoverText = c.setup.Detailed.SkillHover[c.columnClass()][int(hit-chargenSkill0)]
	}
	if !a.chargenPress.Holds() {
		if hit >= chargenStatMinus0 && hit <= chargenStatMinus3 {
			a.chargenHoverText, _ = c.StatStepText(int(hit-chargenStatMinus0), false)
		} else if hit >= chargenStatPlus0 && hit <= chargenStatPlus3 {
			a.chargenHoverText, _ = c.StatStepText(int(hit-chargenStatPlus0), true)
		}
	}
	// The statistic and skill panels act on the left press (VIDEO-SFX-059).
	// The statistic panel also takes a double-click's second click and the
	// held-button repeat as presses; the skill panel ignores both.
	if repeat && chargenStatControl(hit) {
		a.activateChargenDetailed(c, hit, true)
	}
	if in.PrimaryPressed {
		a.chargenPress.Press(int(hit), hit != chargenNone)
		double := a.chargenDoubleClick(c, hit, now)
		if chargenStatControl(hit) || hit >= chargenSkill0 && hit <= chargenSkill4 && !double {
			a.activateChargenDetailed(c, hit, true)
		}
		return
	}
	if in.PrimaryReleased {
		_, released := a.chargenPress.Release(int(hit), hit != chargenNone)
		activate := released && !chargenStatControl(hit) && (hit < chargenSkill0 || hit > chargenSkill4)
		if activate {
			a.activateChargenDetailed(c, hit, true)
		}
		return
	}
	keys := c.keys()
	if in.Enter {
		if keys.Enter == "play" {
			a.playChargen(c)
		} else {
			a.activateChargenDetailed(c, c.detailedFocusControl(), false)
		}
		return
	}
	if keys.FocusKeys && in.Up {
		c.Move(-1)
		return
	}
	if keys.FocusKeys && in.Down {
		c.Move(1)
	}
}

// activateChargenDetailed runs one detailed control. A pointer activation is a
// press and requests the control's chrgen member (VIDEO-SFX-059); keyboard
// focus activation requests nothing (DIV-1493).
func (a *App) activateChargenDetailed(c *Chargen, id chargenControl, pointer bool) {
	switch {
	case id >= chargenSkill0 && id <= chargenSkill4:
		skill := int(id - chargenSkill0)
		if c.SelectSkill(skill) {
			a.flow.msg = ""
		}
		if pointer {
			c.tipSkillClicked()
		}
		// No comparison with the previous selection guards the request.
		if pointer && len(c.choiceIndex) > 2 && skill < len(c.choiceOptions(2)) {
			a.detailedSounds.RequestFor("character-detail", a.soundPlayer, a.namedSounds(), c.skillSound(skill))
		}
	case id >= chargenStatMinus0 && id <= chargenStatMinus3:
		a.stepChargenStat(c, int(id-chargenStatMinus0), -1, pointer)
	case id >= chargenStatPlus0 && id <= chargenStatPlus3:
		a.stepChargenStat(c, int(id-chargenStatPlus0), 1, pointer)
	case id == chargenBack:
		c.Back()
		a.flow.msg, a.chargenHoverText = "", ""
		a.chargenChoiceClick, a.chargenChoiceAt = chargenNone, time.Time{}
	case id == chargenReset:
		c.Reset()
		a.flow.msg, a.chargenHoverText = "", ""
	case id == chargenPlay:
		a.playChargen(c)
	}
}

// stepChargenStat applies one statistic step. An applied pointer step restarts
// +_-.wav; a refused step is silent (VIDEO-SFX-059).
func (a *App) stepChargenStat(c *Chargen, stat, delta int, pointer bool) {
	if !c.AdjustStat(stat, delta) {
		return
	}
	a.flow.msg = ""
	if pointer {
		a.detailedSounds.RestartFor("character-detail", a.soundPlayer, a.namedSounds(), c.statSound())
	}
}

func chargenStatControl(id chargenControl) bool {
	return id >= chargenStatMinus0 && id <= chargenStatPlus3
}

// chargenDoubleClick classifies one pointer press on the showing page and
// records it. A press on the control the previous single press chose, inside
// the page's double-click window, is the second click of a double-click; the press
// after it starts a new pair (DIV-1493).
// chargenPressed is the generation page control the press latched, or
// chargenNone.
func (a *App) chargenPressed() chargenControl {
	if c, ok := a.chargenPress.Latched(); ok {
		return chargenControl(c)
	}
	return chargenNone
}

func (a *App) chargenDoubleClick(c *Chargen, hit chargenControl, now time.Time) bool {
	double := hit != chargenNone && a.chargenChoiceClick == hit && !a.chargenChoiceAt.IsZero() && now.Sub(a.chargenChoiceAt) <= ms(c.keys().DoubleClickMS)
	if double || hit == chargenNone {
		a.chargenChoiceClick, a.chargenChoiceAt = chargenNone, time.Time{}
	} else {
		a.chargenChoiceClick, a.chargenChoiceAt = hit, now
	}
	return double
}

// Hover explanations use the shared overlay; the message strip keeps the
// durable refusal until an accepted edit clears it.
func (a *App) chargenDetailMessage() string {
	return a.flow.msg
}

func chargenReservedName(name string, reservedNames []string) bool {
	if len(name) == 0 {
		return false
	}
	fold := func(b byte) byte {
		if b >= 'A' && b <= 'Z' {
			return b + ('a' - 'A')
		}
		return b
	}
	for _, reserved := range reservedNames {
		if len(name) != len(reserved) {
			continue
		}
		match := true
		for i := range name {
			if fold(name[i]) != reserved[i] {
				match = false
				break
			}
		}
		if match {
			return true
		}
	}
	return false
}

func (a *App) playChargen(c *Chargen) {
	name := c.NameText()
	if name == "" {
		if c.setup.Detailed != nil {
			a.flow.msg = c.setup.Detailed.EmptyName
		} else {
			a.flow.msg = "cannot confirm: name is empty"
		}
		return
	}
	if chargenReservedName(name, c.reservedNames()) {
		if c.setup.Detailed != nil {
			a.flow.msg = c.setup.Detailed.ReservedName
		} else {
			a.flow.msg = "cannot confirm: reserved name"
		}
		return
	}
	result, ok := c.Result()
	if !ok {
		a.flow.msg = "cannot confirm: the spread is not legal"
		return
	}
	if a.flow.chargenBegin == nil {
		a.flow.msg = "cannot begin: no callback configured"
		return
	}
	open, err := a.flow.chargenBegin(result)
	if err != nil {
		a.flow.msg = err.Error()
		return
	}
	if open == nil && c.setup.AcceptShowsTown {
		a.preCreateSounds.Stop()
		a.detailedSounds.Stop()
		a.flow.showTown("")
		return
	}
	if open == nil {
		if a.flow.screen == ScreenTown {
			a.flow.resetTimedAutosave()
		}
		return
	}
	v, tick, order, cadence, affect, advance, attack, grab, stance, march, err := open()
	if err != nil {
		a.flow.msg = err.Error()
		return
	}
	a.flow.enter(v, tick, order, cadence, affect, advance, attack, grab, stance, march)
	a.syncViewerLayout()
}

func (a *App) stepPreCreate(c *Chargen, in appInput, now time.Time) {
	a.observeChargenTipPress(in)
	a.flow.cursor.SetCursor(c.pageCursor())
	defer a.paintPreCreateCycle(c, in, now)
	// THE SHOWING PANEL SWALLOWS EVERY PRESS AND RELEASE INSIDE ITS OWN RECT
	// BEFORE THE PAGE (1018 spec behaviour 1, restored in round 3 / DIV-162),
	// on the town screens' own precedent (stepShop/stepTownSurface /square
	// dispatch above): a press on the panel's own painted body used to fall
	// through to preControlAt below and could commit a portrait the player
	// never clicked (round-3 review) — the researched-width rect's own small
	// residual overlap against the first fighter portrait relies on this
	// swallow.
	if p, ok := a.windowToNativeFrame(in.CursorX, in.CursorY); ok && (in.PrimaryPressed || in.PrimaryReleased) {
		if kind, consumed := TipPanelControlAt(c.TipPanel(), p); consumed {
			if in.PrimaryReleased {
				switch kind {
				case TipControlClose:
					c.CloseTip()
				case TipControlToggle:
					c.ToggleTips()
				}
			}
			// A SWALLOWED PRESS OR RELEASE CLEARS THE PAGE'S OWN PRESS
			// LATCH (hotfix, tip-latch): the block below this one only
			// arms or resolves a.chargenPress when the panel does NOT
			// consume the event, so a gesture crossing the panel's own
			// boundary in either direction used to leave the latch exactly
			// as the last un-swallowed event left it, letting a release on
			// a live control this press never touched call
			// activatePreCreate — which, for a choice control, also arms
			// the double-click window and can make a later, genuine click
			// read as the second half of a double click it was never part
			// of. Clearing it here mirrors what the un-swallowed release
			// path already does unconditionally at its own end (below).
			a.chargenPress.Clear()
			if in.PrimaryPressed {
				a.chargenChoiceClick, a.chargenChoiceAt = chargenNone, time.Time{}
			}
			return
		}
	}
	if a.chargenChoiceClick != chargenNone && (a.chargenChoiceAt.IsZero() || now.Sub(a.chargenChoiceAt) > ms(c.keys().DoubleClickMS)) {
		a.chargenChoiceClick, a.chargenChoiceAt = chargenNone, time.Time{}
	}
	hit := chargenNone
	if p, ok := a.windowToNativeFrame(in.CursorX, in.CursorY); ok {
		hit = preControlAt(c, p)
	}
	a.chargenHover = hit
	if in.PrimaryPressed {
		a.chargenPress.Press(int(hit), hit != chargenNone)
		// Difficulty, hero, OK and the amulet act on the left press
		// (VIDEO-SFX-058); the name field keeps its completed release.
		if double := a.chargenDoubleClick(c, hit, now); hit != chargenNone && hit != chargenName {
			a.activatePreCreate(c, hit, true, double)
			if a.flow.screen != ScreenChargen || c.Stage() != PreCreateStage {
				a.chargenPress.Clear()
				return
			}
		}
	}
	if in.PrimaryReleased {
		_, released := a.chargenPress.Release(int(hit), hit != chargenNone)
		activate := released && hit == chargenName
		if activate {
			a.activatePreCreate(c, hit, true, false)
		}
		return
	}
	keys := c.keys()
	if in.Enter {
		if keys.Enter == "forward" {
			a.preCreateForward(c, true)
		} else {
			a.activatePreCreate(c, c.preFocusControl(), false, false)
		}
		return
	}
	if keys.FocusKeys && in.Up {
		c.Move(-1)
		return
	}
	if keys.FocusKeys && in.Down {
		c.Move(1)
		return
	}
	if (keys.Typing == "always" || c.preFocusControl() == chargenName) && (in.Backspace || in.Typed != "") {
		c.EditName(in.Typed, in.Backspace)
	}
}

// paintPreCreateCycle runs the pre-create guided cycle once per frame while
// TipsMode is set and the popup exists (TOWN-519). Its hovered region is the
// Mask.bmp code under the pointer.
func (a *App) paintPreCreateCycle(c *Chargen, in appInput, now time.Time) {
	if a.flow.screen != ScreenChargen || c.Stage() != PreCreateStage || !c.setup.TipsOn || !c.preTip {
		c.cycleDraw = -1
		return
	}
	hovered := -1
	if p, ok := a.windowToNativeFrame(in.CursorX, in.CursorY); ok {
		hovered = c.preHoverRegion(preControlAt(c, p))
	}
	l := c.layout()
	if l == nil {
		c.cycleDraw = -1
		return
	}
	c.cycleDraw = a.tipCycles[0].paint(now, l.Tips.PreCreateCycle, c.preStep, hovered, ms(l.Tips.HoverWaitMS), ms(l.Tips.StepGapMS))
}

// paintSkillCycle runs the detailed skill cycle while the popup exists and
// its step is 0; it reads no TipsMode (TOWN-522, MENU-136).
func (a *App) paintSkillCycle(c *Chargen, in appInput, now time.Time) {
	if a.flow.screen != ScreenChargen || c.Stage() != DetailedStage || !c.detailTip || c.detailStep != 0 {
		c.cycleDraw = -1
		return
	}
	hovered := -1
	if p, ok := a.windowToNativeFrame(in.CursorX, in.CursorY); ok {
		if id := detailedControlAt(c, p); id >= chargenSkill0 && id <= chargenSkill4 {
			hovered = int(id - chargenSkill0)
		}
	}
	l := c.layout()
	if l == nil {
		c.cycleDraw = -1
		return
	}
	skills := make([]int, c.selectableSkills())
	for i := range skills {
		skills[i] = i
	}
	c.cycleDraw = a.tipCycles[1].paint(now, [][]int{skills}, 0, hovered, ms(l.Tips.HoverWaitMS), ms(l.Tips.StepGapMS))
}

// activatePreCreate runs one pre-create control. A pointer activation is the
// control's left press and requests its chrgen member; double marks the
// second click of a pointer double-click, which requests none and turns a
// hero press into Forward (VIDEO-SFX-058). Keyboard focus activation requests
// nothing except the continue (DIV-1493), so Enter stays a single focused
// action.
func (a *App) activatePreCreate(c *Chargen, id chargenControl, pointer, double bool) {
	switch id {
	case chargenLevel0, chargenLevel1, chargenLevel2:
		if double {
			return
		}
		c.SelectDifficulty(int(id - chargenLevel0))
		if pointer {
			c.tipLevelClicked()
			a.preCreateSounds.RestartFor("character-precreate", a.soundPlayer, a.namedSounds(), c.levelSound(int(id-chargenLevel0)))
		}
	case chargenName:
		c.focus = c.focusIndex(chargenName)
	case chargenChoice0, chargenChoice1, chargenChoice2, chargenChoice3:
		c.SelectPreChoice(int(id - chargenChoice0))
		if double && c.keys().DoubleClickForward {
			a.preCreateForward(c, false)
			return
		}
		if pointer && !double {
			c.tipPortraitClicked()
			a.preCreateSounds.RestartFor("character-precreate", a.soundPlayer, a.namedSounds(), c.heroSound())
		}
	case chargenForward:
		a.preCreateForward(c, true)
	case chargenBack:
		a.preCreateBack(c, pointer)
	}
}

// preCreateForward leaves pre-create for the detailed page. The OK control
// continues only with a non-empty name (TEXT-CHARGEN-028); with an empty one
// OK and Enter request nothing and send nothing (VIDEO-SFX-058), so the page
// stays as it is. The hero double-click is the same transition and takes the
// same test (DIV-1507). OK and Enter request the page's ok.wav unless it
// plays; the page's close then stops every instance it requested.
func (a *App) preCreateForward(c *Chargen, continued bool) {
	if c.NameText() == "" {
		return
	}
	if continued {
		a.preCreateSounds.RequestFor("character-precreate", a.soundPlayer, a.namedSounds(), c.buttonSound(chargenForward))
	}
	c.Forward()
	a.chargenChoiceClick, a.chargenChoiceAt = chargenNone, time.Time{}
	a.flow.msg, a.chargenHoverText = "", ""
	a.preCreateSounds.Stop()
}

// preCreateBack is the amulet: a press requests the page's ok.wav unless it
// plays before the page closes, and the close stops it (VIDEO-SFX-058).
func (a *App) preCreateBack(c *Chargen, pressed bool) {
	if pressed {
		a.preCreateSounds.RequestFor("character-precreate", a.soundPlayer, a.namedSounds(), c.buttonSound(chargenBack))
	}
	a.chargenChoiceClick, a.chargenChoiceAt = chargenNone, time.Time{}
	a.flow.escape()
	a.preCreateSounds.Stop()
}

// syncViewerLayout sizes a freshly loaded map screen's camera to the window.
//
// The engine calls Layout only when the window changes, so a viewer created
// mid-frame would otherwise keep the default view size until the user resized.
func (a *App) syncViewerLayout() {
	if v := a.flow.viewer; v != nil {
		if a.dialogueBackdrop.policySet && a.dialogueBackdrop.appliedViewer != v {
			v.SetDialogueBackdrop(a.dialogueBackdrop.policy)
			a.dialogueBackdrop.appliedViewer = v
		}
		v.Layout(a.winW, a.winH)
		if a.flow.screen == ScreenMap && a.mapEntry != nil && a.mapEntryViewer != v {
			a.mapEntryViewer = v
			a.mapEntry(v)
		}
	}
}

// composeScreen returns the CPU RGBA composite for a.flow.screen's current
// state, or an error naming the screen (and, where the refusal is about a
// substate rather than the screen itself, the substate) when this build has
// no CPU composite for it.
//
// THIS IS THE ONE SITE THAT DECIDES WHICH COMPOSER A SCREEN USES. Six
// developer tools under cmd/ (plaquescreens, townsquarecheck, schoolcheck,
// shopdump, paneldump, plaqueseams) each re-implemented that decision by
// hand before this function existed, one Compose* call site per tool, so a
// later change to Draw's own selection had six other places it could
// silently stop matching. drawMenu, drawChargen and drawTown call this
// function for every state it can compose; HeadlessFrame (headless.go),
// cmd/screenshot's own seam, calls nothing else. A screen this function
// refuses is refused identically by both callers, and a screen this
// function composes is composed identically by both.
//
// It composes ScreenMenu (menu.Assets.Compose), ScreenChargen once the
// active model carries pre-create art (composeChargenScreen), and
// ScreenTown or ScreenGameMenu once the town's current room has a shipped
// picture (composeTownScreen — Draw's own comment on the ScreenGameMenu
// case says "the only surface that reaches here is the town", so the two
// screens share one case). It refuses every screen this build paints only
// through ebitenutil (ScreenPicker) and any screen value this
// package does not know, by the same default arm.
//
// It does not refuse or compose the gameplay/mission screen (ScreenMap):
// that state never reaches a.flow.screen inside Draw's own switch, because
// Draw returns from its mapShowing() branch first. A caller that needs to
// tell "gameplay" apart from "this build cannot compose the current state"
// checks mapShowing (HeadlessFrame does, through a.Screen() and its own
// early check) before calling this function, not after.
func (a *App) composeScreen() (*image.RGBA, error) {
	if h, ok := screenHandlers[a.flow.screen]; ok && h.Compose != nil {
		return h.Compose(a)
	}
	return nil, fmt.Errorf("draws only through ebitenutil, no CPU composite")
}

// Draw paints the current screen.
//
// Town-family screens retain their native 640x480 composition and are fitted
// uniformly. Mission screens alone expand their height-768 logical frame.
func (a *App) Draw(screen *ebiten.Image) {
	defer a.captureScreenshot(screen)
	if a.cutscene != nil {
		a.drawCutscene(screen)
		return
	}
	// THE MAP SCREEN AND THE MENU STANDING OVER ONE take the same path. The map
	// is drawn exactly as it was, at window resolution and without the canvas,
	// and the panel is composed over it afterwards. The dim between the two is
	// the VIEWER'S, already drawn by the call below under the popup answer the
	// menu raises — there is no dim statement here for the map path, and that
	// is the inheritance working. Both branches draw a pointer of their own on
	// some frames, and each used to keep its own "last told" cache of the one
	// piece of global window state; the map branch returns before drawCursor,
	// so entering the map left the engine hidden with nothing drawn. One call
	// here, on every frame, is what makes "hidden exactly when something is
	// drawn" a property of the frame rather than of the branch.
	a.applyPointerMode()

	if a.flow.mapShowing() {
		if v := a.flow.viewer; v != nil {
			v.Draw(screen)
			a.drawCheatChat(screen)
			a.textLayers = append(a.textLayers[:0], textLayer{v.textCalls, v.canvas.Bounds(), &v.canvasLog})
			a.menuCaptured, a.menuKept = 0, 0
			a.drawGameMenuOverMap(screen)
			// THE POINTER IS COMPOSED LAST ON THIS BRANCH TOO (owner). It is the
			// whole reason the viewer defers it: the in-game menu is painted onto the
			// window above, after the frame the viewer returned, so a pointer drawn
			// inside that frame is a pointer under the pause menu. This branch's
			// return used to sit where this call is, which is also why drawCursor at
			// the foot of Draw — the non-map screens' own pointer, already last in
			// its branch — never covered the map.
			//
			// The mission placement is the VIEWER's and not the App's: the
			// map uses a height-768 logical frame while the other screens use
			// height 480, with independently expanded widths, scales and origins
			// (DIV-249, DIV-211).
			ox, oy := v.place.Origin()
			v.drawPointer(screen, v.place.Scale(), ox, oy)
			return
		}
		a.textLayers, a.menuCaptured, a.menuKept = a.textLayers[:0], 0, 0
		a.drawGameMenuOverMap(screen)
		return
	}

	if a.canvas == nil {
		a.canvas = ebiten.NewImage(frame.W, frame.H)
		a.canvasLog.stale = true
	}
	a.canvasLog.begin(a.canvas.Bounds())
	a.wideDialogue = nil
	if a.textSmoothingEnabled {
		// OWNER DECISION METHOD C (DIV-1385): every glyph this switch, the
		// dialogue overlay and the tooltip below place is captured here, in
		// present's own 640x480 coordinate space (prepareWideCanvas's base
		// layout is always that exact identity mapping — wideframe.go), and
		// still rasterised. settleTextFrame then keeps only the glyphs that
		// survived into present and erases their raster before the
		// sharp-bilinear blit, so the smoothed overlay repaints exactly the
		// visible text: a tooltip or dialogue pasted over a glyph hides it.
		text.ResetCapture()
		text.SetCapture(false)
	}
	if h, ok := screenHandlers[a.flow.screen]; ok && h.Draw != nil {
		h.Draw(a)
	}

	a.canvasLog.end()
	present := a.prepareWideCanvas(a.baseWideFrameLayout())
	if a.wideDialogue != nil {
		calls, span := text.Captured(), a.wideDialogueCalls
		shows := townDialogueShows(a.flow.town)
		a.dialogueBackdrop.drawWithCalls(present, &a.presentLog, shows, calls[:span[0]])
		remapCaptured(calls[span[1]:], a.dialogueBackdrop.rect(present.Bounds()), a.dialogueBackdrop.table(), shows)
		art, body := townDialogueFrame(a.flow.town)
		at := image.Pt((present.Bounds().Dx()-a.wideDialogue.Bounds().Dx())/2, (frame.H-a.wideDialogue.Bounds().Dy())/2)
		a.dialogueBackdrop.drawFrameShadows(present, &a.presentLog, art, body, at, 1, calls[:span[0]])
		if mask := a.dialogueBackdrop.shadowMask; mask != nil && art.valid() {
			remapCapturedMask(calls[span[1]:], mask.Rect, a.dialogueBackdrop.shadowTable(), mask, mask.Rect.Min, true)
		}
	}
	paintDetachedTownDialogue(present, a.wideDialogue, &a.presentLog, a.dialogueBackdrop.policy)
	if a.flow.screen == ScreenGameMenu {
		a.paintGameMenuOnWideCanvas(present)
	}
	a.drawTooltip(present)
	a.presentLog.end()
	var capturedText []text.DrawCall
	if a.textSmoothingEnabled {
		text.StopCapture()
		capturedText = append(capturedText, text.Captured()...)
	}
	a.textLayers = append(a.textLayers[:0], textLayer{capturedText, present.Bounds(), &a.presentLog})
	if a.place.Valid() {
		ox, oy := a.place.Origin()
		var op ebiten.DrawImageOptions
		op.GeoM.Scale(a.place.Scale(), a.place.Scale())
		op.GeoM.Translate(ox, oy)
		op.Filter = ebiten.FilterNearest
		// The final blit of the native composite to the window, through the
		// owner's FrameSmoothing scaler. See drawFinalFrame.
		a.textCaptured = len(capturedText)
		present, capturedText = settleText(present, &a.presentLog, capturedText, &a.textEraser,
			&a.settledBuf, &a.settledTex, &a.textSettleFallbacks)
		a.textKept = len(capturedText)
		drawFinalFrame(screen, present, &op, &townCompositeSharpBuf, a.frameSmoothingOff)
		if len(capturedText) > 0 {
			a.textOverlay.draw(screen, capturedText, a.place.Scale(), ox, oy)
		}
	}
	a.drawCursor(screen)
}

// prepareWideCanvas places the unchanged native 640x480 non-mission
// composition into its fixed logical interface. The shared layout's segments
// are also consumed by wideFrameLayout.compose, so windowed and headless
// witnesses use one set of source and destination rectangles.
func (a *App) prepareWideCanvas(layout wideFrameLayout) *ebiten.Image {
	bounds := layout.bounds()
	if a.wideCanvas == nil || a.wideCanvas.Bounds().Size() != bounds.Size() {
		if a.wideCanvas != nil {
			a.wideCanvas.Dispose()
		}
		a.wideCanvas = ebiten.NewImage(bounds.Dx(), bounds.Dy())
	}
	a.wideCanvas.Fill(wideFrameFill)
	blitNativeIntoWide(a.wideCanvas, a.canvas, layout)
	a.presentLog.reset(a.wideCanvas.Bounds())
	a.presentLog.fill(a.wideCanvas.Bounds(), wideFrameFill)
	logNativeIntoWide(&a.presentLog, &a.canvasLog, layout)
	return a.wideCanvas
}

// logNativeIntoWide records blitNativeIntoWide: a segment copied at scale
// one is a layer of the source's own log, any other is unknown.
func logNativeIntoWide(dst, src *pixelLog, layout wideFrameLayout) {
	for _, segment := range layout.segments() {
		destination := segment.destination()
		if destination.Size() == segment.Source.Size() {
			dst.overLayer(src, destination, destination.Min.Sub(segment.Source.Min))
		} else {
			dst.unknown(destination)
		}
	}
}

func blitNativeIntoWide(dst, src *ebiten.Image, layout wideFrameLayout) {
	if dst == nil || src == nil {
		return
	}
	for _, segment := range layout.segments() {
		part := src.SubImage(segment.Source).(*ebiten.Image)
		destination := segment.destination()
		var op ebiten.DrawImageOptions
		op.GeoM.Scale(float64(destination.Dx())/float64(segment.Source.Dx()),
			float64(destination.Dy())/float64(segment.Source.Dy()))
		op.GeoM.Translate(float64(destination.Min.X), float64(destination.Min.Y))
		op.Filter = ebiten.FilterNearest
		dst.DrawImage(part, &op)
	}
}

// paintDetachedTownDialogue keeps a town conversation one intact modal after
// the underlying native room has separated its main surface and right column.
// Painting it into the 640x480 source first would cut every 580-pixel dialogue
// at the town seam and move its right fragment with the HUD.
func paintDetachedTownDialogue(dst *ebiten.Image, dialogue *image.RGBA, log *pixelLog, policies ...DialogueBackdrop) {
	if dst == nil || dialogue == nil {
		return
	}
	pic := ebiten.NewImageFromImage(dialogue)
	at := image.Pt((dst.Bounds().Dx()-dialogue.Bounds().Dx())/2, (frame.H-dialogue.Bounds().Dy())/2)
	var op ebiten.DrawImageOptions
	op.GeoM.Translate(float64(at.X), float64(at.Y))
	policy := DialogueBackdrop{}
	if len(policies) > 0 {
		policy = policies[0]
	}
	clip := dialoguePolicyClip(policy, dst.Bounds())
	if clip.Empty() {
		return
	}
	dst.SubImage(clip).(*ebiten.Image).DrawImage(pic, &op)
	log.overClipped(dialogue, at, dialoguePolicyClip(policy, dst.Bounds()))
}

// paintGameMenuOnWideCanvas leaves the widened town visible under one dim and
// keeps the native menu panel centred. Newly inserted pixels therefore belong
// to the paused interface, while no installed panel bitmap is stretched.
func (a *App) paintGameMenuOnWideCanvas(dst *ebiten.Image) {
	if dst == nil {
		return
	}
	vector.DrawFilledRect(dst, 0, 0, float32(dst.Bounds().Dx()), float32(frame.H),
		AuthoredNoticeBackdrop(), false)
	a.presentLog.overSolid(image.Rect(0, 0, dst.Bounds().Dx(), frame.H), AuthoredNoticeBackdrop())
	if a.menuCanvas == nil {
		a.menuCanvas = ebiten.NewImage(frame.W, frame.H)
	}
	a.menuCanvas.Clear()
	a.menuLog.reset(a.menuCanvas.Bounds())
	a.menuLog.fill(a.menuCanvas.Bounds(), color.RGBA{})
	a.paintCurrentGameMenu(a.menuCanvas)
	if a.flow.msg != "" && a.gameMenuAcknowledgement() == "" && !(a.flow.menuPage == gameMenuSoundOptionsPage && a.flow.soundOptions.Read != nil) {
		msg := clipRunes(a.flow.msg, pickerCols)
		drawDebugText(a.menuCanvas, &a.menuLog, msg, pickerLeft, pickerMessageY)
	}
	a.menuLog.end()
	layout := newWideFrameLayout(dst.Bounds().Dx(), wideFrameCentered)
	blitNativeIntoWide(dst, a.menuCanvas, layout)
	logNativeIntoWide(&a.presentLog, &a.menuLog, layout)
}

// applyPointerMode tells the engine what flow.pointerWanted decided for this
// frame, and is THE ONLY ebiten.SetCursorMode call site an App session makes.
//
// The engine is told on a CHANGE only, which is the cache's whole job — not
// for cost, but so a program that never draws a cursor of its own never asks
// the engine for anything, and so the standalone developer viewer's frame is
// unchanged down to the calls it makes (pointerModeChange's own argument,
// cursor.go). The cache lives on the shared CursorManager, so both draw sites
// read and write one record rather than two that never met.
func (a *App) applyPointerMode() {
	if !a.flow.syncPointerMode() {
		return
	}
	mode := ebiten.CursorModeVisible
	if a.flow.cursor.PointerHidden() {
		mode = ebiten.CursorModeHidden
	}
	ebiten.SetCursorMode(mode)
}

// CursorPlacement is where App.Draw puts the front-end's own cursor picture on
// one frame, on every screen but the map.
type CursorPlacement struct {
	// Pic is the registered picture. It is the registry's own, not a copy.
	Pic *image.RGBA
	// Hot is the picture's registered hotspot, in picture pixels.
	Hot image.Point
	// Tip is the frame pixel under the hotspot: the pixel the hit tests resolve
	// for the pointer, continued past the frame's edge into the letterbox.
	Tip image.Point
	// X and Y are the window position of the picture's top-left corner.
	X, Y float64
	// Scale is window pixels per frame pixel.
	Scale float64
}

// cursorPlacement is the one placement Draw and the headless witness share.
// The position it reads is the latest step's, so the picture stands where the
// step that acted on the pointer saw it, on the frame the step precedes.
//
// THE PICTURE STANDS ON THE FRAME'S PIXEL LATTICE, NOT ON THE WINDOW'S. The
// original blits its cursor at an integer position of its 640x480 surface
// (AI-CURSOR-218), the hit tests read that same surface (MENU-MASK-004), and the
// mission pointer is placed from the frame pixel the same way. The picture goes
// at the frame pixel under the pointer, minus the hotspot, and only then is
// scaled and offset by the placement, so it moves by whole frame pixels and is
// resampled as the frame under it is. Placed at the raw window pixel it slid by
// window pixels over art that only has frame pixels, and each phase between two
// frame pixels drew a differently resampled picture.
//
// Before a window size is known the placement is invalid and the window pixel is
// the frame pixel at scale 1, which is where the picture stood before there was
// a placement to consult.
func (a *App) cursorPlacement() (CursorPlacement, bool) {
	tip, scale, ox, oy := a.pointer, 1.0, 0.0, 0.0
	if p, ok := a.place.WindowToFrameExtended(a.pointer.X, a.pointer.Y); ok {
		tip, scale = p, a.place.Scale()
		ox, oy = a.place.Origin()
	}
	pic, at, ok := a.flow.cursorPresent(tip)
	if !ok {
		return CursorPlacement{}, false
	}
	return CursorPlacement{
		Pic: pic, Hot: tip.Sub(at), Tip: tip, Scale: scale,
		X: ox + float64(at.X)*scale, Y: oy + float64(at.Y)*scale,
	}, true
}

// drawCursor paints the shared cursor manager's current picture at the engine's
// own cursor position. It decides nothing about the system pointer:
// applyPointerMode above already did, for the whole frame.
//
// IT IS CALLED ONLY FROM THE NON-MAP BRANCH of Draw.
//
// NO PICTURE IS THE ORDINARY STATE before a registry is installed. It is not
// the ordinary state on the picker or the load list: those screens run no
// surface transition (flow.screenExitCursor), and B2's persistence rule then
// leaves whatever the previous screen set current, which is what is drawn.
//
// THE PICTURE IS SCALED BY THE CANVAS PLACEMENT. Every other pixel of these
// screens is 640x480 art blown up by a.place.Scale(), so a pointer drawn at
// 1:1 window pixels is about a third of the size it occupies in the original's
// own frame on a 1920x1080 monitor. The hotspot is subtracted in frame pixels
// (flow.cursorPresent), so the registration's own point still lands on the
// pixel a click names. The map screen uses the same rule over its distinct
// height-768 logical frame: the mission pointer is drawn 1:1 in mission-frame
// pixels and that frame receives its own uniform fit (see DIV-249, DIV-211).
func (a *App) drawCursor(screen *ebiten.Image) {
	c, ok := a.cursorPlacement()
	if !ok {
		return
	}
	var op ebiten.DrawImageOptions
	op.Filter = ebiten.FilterNearest
	op.GeoM.Scale(c.Scale, c.Scale)
	op.GeoM.Translate(c.X, c.Y)
	// The cursor is scaled by the same a.place.Scale() as the composite it
	// is drawn over, so it shares that composite's scaler (owner direction).
	drawFinalFrame(screen, a.cursorTex.upload(c.Pic), &op, &cursorSharpBuf, a.frameSmoothingOff)
}

// drawGameMenuOverMap composes the in-game menu's panel over a map screen
// already drawn to the window.
//
// THE PANEL IS PAINTED AT NATIVE FRAME COORDINATES AND FITTED through the
// native 640x480 placement, which keeps its decoded rectangles and input
// inverse together without stretching the panel. The map behind it uses the
// mission placement and is unchanged.
//
// It draws nothing when the menu is down, so the map screen's own frame is
// byte-identical to the frame before this story.
func (a *App) drawGameMenuOverMap(screen *ebiten.Image) {
	if a.flow.screen != ScreenGameMenu || a.flow.menuList == nil || !a.place.Valid() {
		return
	}
	if a.menuCanvas == nil {
		a.menuCanvas = ebiten.NewImage(frame.W, frame.H)
	}
	a.menuCanvas.Clear()
	a.menuLog.reset(a.menuCanvas.Bounds())
	a.menuLog.fill(a.menuCanvas.Bounds(), color.RGBA{})
	// The panel is the one layer the menu composes, so its glyphs are captured
	// for the overlay in the menu canvas's own coordinates.
	captureAt := beginTextCapture(a.textSmoothingEnabled)
	a.paintCurrentGameMenu(a.menuCanvas)
	if a.flow.msg != "" && a.gameMenuAcknowledgement() == "" {
		msg := clipRunes(a.flow.msg, pickerCols)
		drawDebugText(a.menuCanvas, &a.menuLog, msg, pickerLeft, pickerMessageY)
	}
	a.menuLog.end()
	var captured []text.DrawCall
	if captureAt >= 0 {
		text.StopCapture()
		captured = append(captured, text.Captured()[captureAt:]...)
	}
	a.textLayers = append(a.textLayers, textLayer{captured, a.menuCanvas.Bounds(), &a.menuLog})
	menuCanvas, kept := settleText(a.menuCanvas, &a.menuLog, captured, &a.menuEraser,
		&a.menuBuf, &a.menuTex, &a.textSettleFallbacks)
	a.menuCaptured, a.menuKept = len(captured), len(kept)
	ox, oy := a.place.Origin()
	menuAt, _ := newWideFrameLayout(a.wideFrameWidth(), wideFrameCentered).nativeToWide(image.Point{})
	scale := a.place.Scale()
	var op ebiten.DrawImageOptions
	op.GeoM.Scale(scale, scale)
	op.GeoM.Translate(ox+float64(menuAt.X)*scale, oy)
	op.Filter = ebiten.FilterNearest
	frameSmoothingOff = a.frameSmoothingOff
	blitMenuOverMap(screen, menuCanvas, &op)
	if len(kept) > 0 {
		a.menuOverlay.draw(screen, kept, scale, ox+float64(menuAt.X)*scale, oy)
	}
}

// blitMenuOverMap places the in-game menu's own canvas on the window over a
// map screen.
//
// THE DEFAULT DRAWS THROUGH drawFinalFrame: op carries the same a.place scale
// and origin the town composite above does — this is that composite's own
// native 640x480 menu canvas, scaled to the window the identical way — so it
// shares its scaler. Production never replaces it; missioncursor_test.go
// replaces it only to record ordering, which op's realization does not
// change.
var blitMenuOverMap = func(dst, img *ebiten.Image, op *ebiten.DrawImageOptions) {
	drawFinalFrame(dst, img, op, &menuOverMapSharpBuf, frameSmoothingOff)
}

// drawMenu refreshes the frame from the menu assets, but only when the selection
// changed: the composite is a 1.2 MB CPU copy and the brooch changes only when
// the pointer moves onto or off a button.
func (a *App) drawMenu() {
	state := a.sel.State()
	if !a.hasMenu || state != a.composed {
		// composeScreen, not a.assets.Compose(state) directly: ScreenMenu is
		// one of its cases, and HeadlessFrame reads the same case for a
		// headless capture of this screen (cmd/screenshot). The error return
		// is unreachable here — composeScreen never refuses ScreenMenu — so
		// it is discarded rather than threaded through Draw's own signature.
		var pix *image.RGBA
		a.menuText = text.Record(func() { pix, _ = a.composeScreen() })
		a.writeCanvas(pix)
		a.composed, a.hasMenu = state, true
	}
	// The canvas keeps the composed menu across frames; its glyphs are
	// re-captured each frame so the overlay smooths them every frame.
	text.Append(a.menuText, 0, 0)
	// The key hint that used to be painted here is GONE (owner). The brooch's
	// own top-right button opens the load window now, so a line of debug text
	// advertising a keyboard shortcut over the game's own art was telling the
	// player the wrong thing about his own screen.
}

// drawList paints a header, a list and the message line, through drawPicker's
// own constants (0143 plan D-6) — so the mini-menu, the load window and the map
// list agree about where a row is by construction rather than by three pieces of
// code sharing a taste, and a drawn row is a clickable one here for the reason
// it is there.
func (a *App) drawList(header string, list *Picker) {
	a.hasMenu = false // the canvas no longer holds a composed menu frame
	if pix, err := a.composeLoadList(header, list); err == nil {
		a.writeCanvas(pix)
		return
	}
	a.fillCanvas(pickerBackground)
	if list == nil {
		return
	}
	a.printCanvas(header, pickerLeft, pickerHeaderY)
	top, n := list.Visible()
	for k := 0; k < n; k++ {
		a.printCanvas(list.RowText(top+k), pickerLeft, pickerTop+k*pickerLine)
	}
	if message := a.loadMessage(); message != "" {
		a.printCanvas(clipRunes(message, pickerCols), pickerLeft, pickerMessageY)
	}
}

// loadHeader names the window and how many saves it found.
func (a *App) loadHeader() string {
	return fmt.Sprintf("LOAD GAME  (%d saved)", len(a.flow.saves))
}

// drawPicker paints the map list: a header carrying the list's extent, the
// visible rows with the selection marked, and — when a chosen map failed to
// load — the reason, on its own line.
//
// That message line is where a failed load is actually reported. Recording the
// reason without putting it in front of the user would satisfy nothing. The
// header's counts come from the picker for the same reason: a list longer than
// the frame has to say so, or it reads as a complete short list.
func (a *App) drawPicker() {
	a.hasMenu = false // the canvas no longer holds a composed menu frame
	a.fillCanvas(pickerBackground)

	a.printCanvas(a.flow.picker.HeaderText(), pickerLeft, pickerHeaderY)

	top, n := a.flow.picker.Visible()
	for k := 0; k < n; k++ {
		row := top + k
		entry := a.flow.picker.Rows()[row]
		if !entry.HasColumns() {
			a.printCanvas(a.flow.picker.RowText(row), pickerLeft, pickerTop+k*pickerLine)
			continue
		}
		// The description column ends where the size column starts.
		a.printCanvas(clipRunes(a.flow.picker.RowText(row), pickerColumnSize/pickerAdvance-1), pickerLeft, pickerTop+k*pickerLine)
		columns := entry.ColumnTexts()
		for i, edge := range [3]int{pickerColumnSize, pickerColumnWord, pickerColumnLast} {
			cell := columns[i]
			if i == 1 {
				cell = clipRunes(cell, (pickerColumnLast-pickerColumnWord)/pickerAdvance)
			}
			a.printCanvas(cell, pickerLeft+edge, pickerTop+k*pickerLine)
		}
	}
	if a.flow.msg != "" {
		a.printCanvas(clipRunes(a.flow.msg, pickerCols), pickerLeft, pickerMessageY)
	}
}

// drawTown paints the town: the room's header, its rows with the selection
// marked, whatever the far side states under them, and — when something
// was refused or reported — the message line.
//
// IT IS drawPicker WITH A FOOTER, through the picker's own layout constants,
// so this screen and the map list agree about where a row is by construction
// rather than by two pieces of code sharing a taste. The rows come from the
// same *Picker the hit test reads, which is what makes a drawn row a clickable
// one on this screen for the same reason it is on that one.
//
// THE FOOTER IS CLAMPED TO WHAT FITS. It is drawn from the bottom of the list
// window down, and lines past the message line are dropped rather than painted
// over it — the budget is measured here, once, for the reason chargenDerivedFit
// exists: "six lines is what fits" was written down once without being measured
// and was never true.
const (
	townSquareLeft = 30
	townSquareTop  = 72
	townSquareW    = 280
	townSquareH    = 112
	townSquareGap  = 20
)

var (
	townButtonFill     = color.RGBA{R: 34, G: 39, B: 48, A: 255}
	townButtonSelected = color.RGBA{R: 67, G: 78, B: 96, A: 255}
	townButtonBorder   = color.RGBA{R: 205, G: 191, B: 143, A: 255}
)

func townSquareRect(i int) (x, y, w, h int) {
	return townSquareLeft + (i%2)*(townSquareW+townSquareGap),
		townSquareTop + (i/2)*(townSquareH+townSquareGap), townSquareW, townSquareH
}

func townSquareRowAt(x, y int) (int, bool) {
	for i := 0; i < 4; i++ {
		rx, ry, rw, rh := townSquareRect(i)
		if x >= rx && x < rx+rw && y >= ry && y < ry+rh {
			return i, true
		}
	}
	return 0, false
}

// townButton draws a row-list button on the canvas.
func (a *App) townButton(x, y, w, h int, selected bool) {
	drawTownButton(a.canvas, x, y, w, h, selected)
	a.canvasLog.unknown(image.Rect(x, y, x+w, y+h).Inset(-1))
}

// writeCanvas uploads pic over the whole canvas.
func (a *App) writeCanvas(pic *image.RGBA) {
	a.canvas.WritePixels(pic.Pix)
	a.canvasLog.upload(pic)
}

// fillCanvas fills the whole canvas with c.
func (a *App) fillCanvas(c color.Color) {
	a.canvas.Fill(c)
	a.canvasLog.fill(a.canvas.Bounds(), c)
}

func (a *App) printCanvas(s string, x, y int) {
	drawDebugText(a.canvas, &a.canvasLog, s, x, y)
}

func drawTownButton(dst *ebiten.Image, x, y, w, h int, selected bool) {
	fill := townButtonFill
	if selected {
		fill = townButtonSelected
	}
	ebitenutil.DrawRect(dst, float64(x), float64(y), float64(w), float64(h), fill)
	ebitenutil.DrawRect(dst, float64(x), float64(y), float64(w), 2, townButtonBorder)
	ebitenutil.DrawRect(dst, float64(x), float64(y+h-2), float64(w), 2, townButtonBorder)
	ebitenutil.DrawRect(dst, float64(x), float64(y), 2, float64(h), townButtonBorder)
	ebitenutil.DrawRect(dst, float64(x+w-2), float64(y), 2, float64(h), townButtonBorder)
}

// dialogueOrigin is where a composed dialogue picture is placed within the
// 640x480 frame: centered, the one position every town room draws it at
// (0157). Both drawTown's ebiten path and composeTownScreen's image/draw
// path call this, so the position agrees even though the two blit through
// different libraries.
func dialogueOrigin(dialogue *image.RGBA) (x, y int) {
	b := dialogue.Bounds()
	return (frame.W - b.Dx()) / 2, (frame.H - b.Dy()) / 2
}

var errTownRowList = errors.New("the row-list room has no shipped picture")

// composeTownRoom returns the CPU composite for t's current room — the
// world map, the shop, a surface (school/tavern/temple/…), or the square
// — with no dialogue overlay.
//
// IT IS A FREE FUNCTION AND NOT AN App METHOD, so a caller holding a
// TownScreen with no App around it can reach the same selection. Two callers
// need exactly that: App itself, through the (a *App) composeTownRoom
// wrapper below (live cursor/press/drag state, App's own message line), and
// ComposeTownScreen, exported for game.FrontEnd's own TownScreen() — the
// seam cmd/shopdump and cmd/plaquescreens already read — reached through
// FinishMission with no App ever constructed, because there is no
// lightweight production path from a FrontEnd to a live App's own
// ScreenTown. It has two production callers: flow.go, the mission-end notice
// a real session only reaches by finishing a mission through the sim, and
// save.go:325, chooseLoad, reached by loading a save recorded as a town
// save. Neither is lightweight for a screenshot tool - the first runs the
// sim, and the second needs a town save, which neither lawful install ships
// (all four shipped slots on both roots are mid-mission map saves). An
// earlier revision of this comment named only the first (pass-1 adversarial
// review). cursor/hasCursor/press are the surface room's own live pointer
// state and dragIcon/hasDrag the shop's own held item; a caller with none of
// that live App state (ComposeTownScreen) passes the zero values, which is
// what an App with no session interaction yet would also hold.
func composeTownRoom(t TownScreen, msg string, cursor image.Point, hasCursor bool, press TownSurfaceControl, dragIcon *image.RGBA, hasDrag bool, animationFrame int, stars *shopStarState, dragOrigin ShopControl, dragOriginBase int, liveShopAnimation bool, pointerState ...townRoomPointerState) (*image.RGBA, error) {
	if t == nil {
		return nil, fmt.Errorf("no town model installed")
	}
	var pointer townRoomPointerState
	if len(pointerState) != 0 {
		pointer = pointerState[0]
	}
	if world, onMap := townWorldMapScreen(t); onMap {
		view := world.WorldMapView()
		view.Message = msg
		return ComposeWorldMap(view), nil
	}

	// THE SHOP ROOM IS ITS OWN SCREEN AND REPLACES THE LIST ENTIRELY. It is
	// composed into an RGBA, which the caller uploads or blits, same as every
	// other composed room here.
	//
	// IT COMPOSES WHETHER OR NOT ITS OWN CONVERSATION IS OPEN (hotfix, owner:
	// entering the shop after a mission — e.g. mission 20 — can open a
	// dialogue immediately, and the shop must show behind it). AtTownShop
	// already answers true through that dialogue (pkg/game's
	// townScreen.AtTownShop: room == roomShop OR room == roomTalk with the shop
	// as the dialogue's building) — inShop alone decides whether the shop
	// composes; the dialogue overlay this function's own caller applies is a
	// separate decision.
	shop, inShop := townShopScreen(t)
	if inShop && liveShopAnimation {
		if animator, ok := t.(ShopInteriorAnimator); ok {
			animator.AdvanceShopInteriorAnimation()
			// The controller publishes its selected pictures through ShopScreen.
			// Reacquire after advancement so this paint never draws the stale
			// read-only view captured before the eligible step.
			shop, inShop = townShopScreen(t)
		}
	}
	if atTownSurface(t) {
		if animator, ok := t.(TownSurfaceAnimator); ok {
			animator.AdvanceTownSurfaceAnimation()
		}
	}
	surface, inSurface := townSurfaceScreen(t)
	switch {
	case inSurface:
		surface.Tip = tipPanelWithPointer(surface.Tip, uint8(surface.Kind)+1, cursor, hasCursor, pointer.tipPress, pointer.tipOwner)
		surface.SuppressHover = true
		surface.Message = msg
		surface.AnimationFrame = animationFrame
		if hasCursor {
			surface.Hover = cursor
			surface.HasHover = true
			if c, ok := TownSurfaceControlAt(surface, cursor); ok {
				if c.Kind == TownSurfaceControlCell {
					surface.HoverCell = c.Index
				}
				// A button shows its own pressed art only while the cursor
				// is still over the control the press began on — the same
				// control TownSurfaceClick requires at release for the
				// press to act at all (1017).
				if c.Kind == TownSurfaceControlButton && c == press {
					surface.Press = c
				}
			}
		}
		return ComposeTownSurface(surface), nil
	case inShop:
		shop.TipPanel = tipPanelWithPointer(shop.TipPanel, 3, cursor, hasCursor, pointer.tipPress, pointer.tipOwner)
		shop.SuppressHover = true
		shop.Msg = msg
		if len(pointerState) > 0 {
			shop.Press = ShopControl{}
			if i, held := pointer.shopPress.Latched(); held {
				shop.Press = ShopControl{Kind: ShopControlButton, Index: i}
			}
		}
		if stars != nil {
			visibleOrigin := shopDragOriginAt(shop, dragOrigin, dragOriginBase)
			stars.prepare(&shop, visibleOrigin, hasDrag)
			stars.advance(&shop)
		}
		frame := ComposeShopScreen(shop, cursor, hasCursor, dragIcon, hasDrag)
		return frame, nil
	default:
		// THE SQUARE'S OWN PICTURE REPLACES THE FOUR-BUTTON GRID. squareView is
		// only "ready" once Background is non-nil, so this arm never fires over a
		// front end whose install failed to ship the picture — the caller's own
		// row-list fallback still draws in that case, exactly as it always has.
		if squareView, ready := townSquareView(t); ready {
			if animator, ok := t.(TownSquareAnimator); ok {
				animator.AdvanceTownSquareAnimation()
				squareView, _ = townSquareView(t)
			}
			squareView.Message = msg
			squareView.Tip = tipPanelWithPointer(squareView.Tip, 4, cursor, hasCursor, pointer.tipPress, pointer.tipOwner)
			return ComposeTownSquare(squareView), nil
		}
		return nil, errTownRowList
	}
}

// townRoomMessage is the line the room under the game menu may draw: none, as
// the menu draws the line itself and a second copy would stand beside it.
func (a *App) townRoomMessage() string {
	if a.flow.screen == ScreenGameMenu {
		return ""
	}
	return a.flow.msg
}

// composeTownRoom is the package-level composeTownRoom applied to a's own
// live town model and interaction state.
func (a *App) composeTownRoom() (*image.RGBA, error) {
	if a.flow.town == nil || a.flow.townList == nil {
		return nil, fmt.Errorf("no town model installed")
	}
	msg := a.townRoomMessage()
	if atTownSquare(a.flow.town) && !a.townPaintAllowed() {
		if v, ready := townSquareView(a.flow.town); ready {
			v.Message = msg
			v.Tip = tipPanelWithPointer(v.Tip, 4, a.townCursor, a.hasTownCursor, a.townTipPress, a.townTipPressOwner)
			return ComposeTownSquare(v), nil
		}
	}
	dragIcon, hasDrag := a.shopDragItemPresent()
	pix, err := composeTownRoom(a.flow.town, msg, a.townCursor, a.hasTownCursor, a.townSurfacePressed(), dragIcon, hasDrag, a.townSurfaceAnimationTick/6, &a.shopStars, a.shopDragOrigin, a.shopDragOriginBase, true,
		townRoomPointerState{shopPress: a.shopButtonPress, tipPress: a.townTipPress, tipOwner: a.townTipPressOwner})
	if errors.Is(err, errTownRowList) {
		return composeTownList(a.flow.town, a.flow.townList, a.flow.msg), nil
	}
	return pix, err
}

// composeTownScreen places the open dialogue over the current room.
// overlayTownDialogue blits t's own open dialogue over an already-composed
// room, at dialogueOrigin.
//
// THE WORLD MAP TAKES NO OVERLAY. Before this seam existed, drawTown
// returned from its world-map branch before reaching its own dialogue check
// (16177e9:pkg/ui/app.go:2769), so no pre-refactor path could paint a
// dialogue over the map; the two wrappers below would have, since both apply
// the blit after any successful compose. The two states are mutually
// exclusive in pkg/game today - AtWorldMap is room == roomGates
// (worldmap.go:421) and townDialogueLayout is room == roomTalk
// (townscreen.go:552), one field on one struct - so this guard changes
// nothing this build can reach. It exists so a later change to that field
// cannot silently introduce a composition no pre-refactor path had.
// Witnessed by TestTheWorldMapTakesNoDialogueOverlay, which is the only
// place in this repository a town model answers true to both (pass-1
// adversarial review).
// markCapture/shiftCapture below record and re-express the dialogue's own
// glyph draws in pix's coordinate space — DIV-1385, owner decision method C.
// The dialogue is composed on its own small picture, so a glyph Draw places
// while composing it carries THAT picture's local coordinates, not pix's;
// the offset this function already computes to paste the picture's pixels
// is the exact same one that puts its captured glyphs in the right place,
// the coordinate-space fix round-1's own prototype found necessary for this
// exact site. Both calls are no-ops whenever no outer capture window is
// open (text smoothing off, or a headless/CPU caller that never opens one),
// so this function's own behaviour is otherwise unchanged.
func overlayTownDialogue(pix *image.RGBA, t TownScreen) {
	overlayTownDialogueWithPolicy(pix, t, &dialogueBackdropState{})
}

func overlayTownDialogueWithPolicy(pix *image.RGBA, t TownScreen, state *dialogueBackdropState) {
	if _, onMap := townWorldMapScreen(t); onMap {
		return
	}
	start := markCapture()
	dialogue, open := townDialogue(t)
	if !open || dialogue == nil {
		return
	}
	state.apply(pix, townDialogueShows(t))
	remapCaptured(text.Captured()[:start], state.rect(pix.Bounds()), state.table(), townDialogueShows(t))
	ox, oy := dialogueOrigin(dialogue)
	art, body := townDialogueFrame(t)
	state.applyFrameShadows(pix, art, body, image.Pt(ox, oy), text.Captured()[:start])
	shiftCapture(start, image.Pt(ox, oy))
	composeDialogueImageWithPolicy(pix, dialogue, image.Pt(ox, oy), state.policy)
}

func (a *App) composeTownScreen() (*image.RGBA, error) {
	pix, err := a.composeTownRoom()
	if err != nil {
		return nil, err
	}
	overlayTownDialogueWithPolicy(pix, a.flow.town, &a.dialogueBackdrop)
	return pix, nil
}

// ComposeTownScreen is composeTownRoom (package-level, above) with the open
// dialogue blitted over it, exported for a caller holding a TownScreen with
// no App around it — see composeTownRoom's own header for why that caller
// exists and cmd/screenshot for its use. cursor, press and drag state are
// the zero values: a bare TownScreen carries none of a live App's pointer
// interaction, which is what an unvisited, freshly opened room also shows.
//
// It returns composeTownRoom's own error, unchanged, for the row-list room.
func ComposeTownScreen(t TownScreen, msg string) (*image.RGBA, error) {
	return ComposeTownScreenFrame(t, msg, 0)
}

// ComposeTownScreenFrame is ComposeTownScreen at one explicit client-only
// tavern animation phase. It exists for headless production captures; the
// live App supplies the same phase from townSurfaceAnimationTick. The school
// diamond instead advances once per invocation of the room compositor.
func ComposeTownScreenFrame(t TownScreen, msg string, animationFrame int) (*image.RGBA, error) {
	pix, err := composeTownRoom(t, msg, image.Point{}, false, TownSurfaceControl{}, nil, false, animationFrame, nil, ShopControl{}, 0, false)
	if err != nil {
		return nil, err
	}
	overlayTownDialogue(pix, t)
	return pix, nil
}

func (a *App) drawTown() {
	a.hasMenu = false // the canvas no longer holds a composed menu frame
	a.fillCanvas(pickerBackground)

	wideDialogueCapture := markCapture()
	var pix *image.RGBA
	var err error
	if dialogue, detached := a.detachedTownDialogue(); detached {
		a.wideDialogueCalls = [2]int{wideDialogueCapture, markCapture()}
		shiftCapture(wideDialogueCapture, image.Pt((a.baseWideFrameLayout().width-dialogue.Bounds().Dx())/2, (frame.H-dialogue.Bounds().Dy())/2))
		pix, err = a.composeTownRoom()
		if err == nil {
			a.wideDialogue = dialogue
		}
	} else {
		pix, err = a.composeTownScreen()
	}
	if err == nil {
		a.writeCanvas(pix)
		return
	}
	if a.flow.town == nil || a.flow.townList == nil {
		return
	}

	// The row-list uses the same layout as its CPU capture.
	dialogueCapture := markCapture()
	dialogue, dialogueOpen := townDialogue(a.flow.town)
	if dialogueOpen && dialogue != nil {
		ox, oy := dialogueOrigin(dialogue)
		shiftCapture(dialogueCapture, image.Pt(ox, oy))
	}
	paintTownList(a.flow.town, a.flow.townList, a.flow.msg, a.townButton, a.printCanvas)

	if dialogueOpen && dialogue != nil {
		a.dialogueBackdrop.drawWithCalls(a.canvas, &a.canvasLog, townDialogueShows(a.flow.town), text.Captured()[:dialogueCapture])
		art, body := townDialogueFrame(a.flow.town)
		sx, sy := dialogueOrigin(dialogue)
		a.dialogueBackdrop.drawFrameShadows(a.canvas, &a.canvasLog, art, body, image.Pt(sx, sy), 1, text.Captured()[:dialogueCapture])
		if dialoguePolicyClip(a.dialogueBackdrop.policy, a.canvas.Bounds()).Empty() {
			return
		}
		pic := ebiten.NewImageFromImage(dialogue)
		var op ebiten.DrawImageOptions
		ox, oy := dialogueOrigin(dialogue)
		op.GeoM.Translate(float64(ox), float64(oy))
		a.canvas.SubImage(dialoguePolicyClip(a.dialogueBackdrop.policy, a.canvas.Bounds())).(*ebiten.Image).DrawImage(pic, &op)
		a.canvasLog.overClipped(dialogue, image.Pt(ox, oy), dialoguePolicyClip(a.dialogueBackdrop.policy, a.canvas.Bounds()))
	}
}

// SetTown installs the town screen the front-end shows between missions, and
// shows nothing itself.
//
// IT IS A SETTER AND NOT A NewApp ARGUMENT, on SetChargenGate's own grounds:
// NewApp already takes four, and the town is exactly the part of a front end
// that a caller may legitimately not have — cmd/mapview's developer viewer has
// no campaign to have a town for. It may be called with nil to remove one.
//
// INSTALLING IT DOES NOT ENTER IT.
func (a *App) SetTown(t TownScreen) {
	a.cancelTownTipPointer()
	a.resetTownSurfacePair()
	a.cancelDialoguePointer()
	a.flow.town = t
}

// Chargen screen layout, in frame pixels of the 640x480 virtual frame,
// beside the picker's own block in picker.go.
//
// The margin, the header line, the line pitch, the message line and the
// column limit are the SAME NUMBERS as the picker's own, named separately
// rather than reused bare: both screens paint through the same DebugPrintAt
// call, into the same frame, with the same fixed 6x16 debug-font glyph cell,
// so a second magic number for one geometry would be two constants that have
// to be kept equal by hand instead of one read twice. They are given their
// own names anyway, because this screen's rows are its own concern — a later
// change to the picker's own layout must not silently move this screen's
// rows with it.
//
// THERE IS NO chargenVisible AND NO SCROLL WINDOW.
const (
	chargenLeft     = pickerLeft
	chargenHeaderY  = pickerHeaderY
	chargenTop      = pickerTop
	chargenLine     = pickerLine
	chargenMessageY = pickerMessageY
	chargenCols     = pickerCols

	// chargenDerivedGap is how many line pitches below the LAST ROW the
	// consequence block starts: 2, so the footer sits at the first and the
	// block at the second, with one blank line between them.
	//
	// HOW MANY LINES THE BLOCK CAN HOLD, since there is no scroll window to
	// absorb an extra one. Seven rows put the footer at y=152 and block line k
	// at y = 184 + 16k. A line occupies [y, y+16) — the picker's own stated
	// convention, pickerVisible's 25 rows "occupy y in [40, 440)" — so the last
	// line clear of the message line at 452 is k=15, at y=424, exactly where the
	// picker's own last row sits. SIXTEEN LINES, title included.
	//
	// It is written down because it was got wrong once: pkg/game's own label
	// block excluded four values on the stated ground that "six lines is what
	// fits under the footer", which was never measured and was never true (0140).
	// A screen with a different row count moves the whole block, which is why
	// chargenDerivedGap is counted off the row count and not off a y of its own.
	chargenDerivedGap = 2
)

// chargenDerivedFit is how many consequence lines fit under a screen of the
// given row count without reaching the message line (0140).
//
// IT EXISTS SO THE BUDGET IS MEASURED RATHER THAN REMEMBERED. What is in the
// block is the wiring tier's answer and this package will not second-guess it —
// but where the block ENDS is this package's own layout, and the one time that
// budget was written as a sentence instead of an expression it was wrong by
// ten lines and kept four values off the screen for two stories. A line
// occupies [y, y+16), so a line whose bottom would pass chargenMessageY does
// not fit.
//
// It never answers negative: a screen with more rows than the frame can hold
// paints no block at all rather than a block above its own footer.
func chargenDerivedFit(rows int) int {
	n := (chargenMessageY-chargenTop)/chargenLine - (rows + chargenDerivedGap)
	if n < 0 {
		return 0
	}
	return n
}

// composeChargenScreen returns the CPU composite for the chargen screen's
// current stage, or an error naming why there is none. drawChargen and
// HeadlessFrame both reach the composed page only through this function.
//
// It refuses a nil model (a.flow.chargen unset — reachable only from a test
// that sets a.flow.screen without arming the model, same caveat drawChargen
// itself has always carried) and a setup with no PreCreate: chargen.go's own
// comment on that field says a nil value "keeps the old diagnostic model
// available to callers that construct a small, headless setup in tests" —
// every setup a real install builds carries one (pkg/game/chargen.go:179),
// so this refusal is a test-fixture case in practice, not a shipped one.
func (a *App) composeChargenScreen() (*image.RGBA, error) {
	c := a.flow.chargen
	if c == nil {
		return nil, fmt.Errorf("no active generation model")
	}
	if c.setup.PreCreate == nil {
		return nil, fmt.Errorf("the legacy diagnostic model has no CPU composite (draws through ebitenutil.DebugPrintAt)")
	}
	// A refused or failed Accept's line goes to the model, which draws it in
	// the description's message strip when it has one. Hover texts show as
	// tooltips only.
	c.SetDetailMessage(a.chargenDetailMessage())
	c.statHeld = a.chargenHeld
	p, inside := a.windowToNativeFrame(a.pointer.X, a.pointer.Y)
	tip := tipPanelWithPointer(c.TipPanel(), chargenTipRoom, p, inside, a.townTipPress, a.townTipPressOwner)
	return composeChargenPage(c, a.chargenHover, a.chargenPressed(), tip), nil
}

// drawChargen paints the generation screen: a header carrying the setup's
// title and how to drive it, one line per row with the focused row marked,
// the footer — what remains of the budget and the confirm label — and,
// when a confirm was refused or begin declined, why, on its own line. It
// paints through the SAME DebugPrintAt call drawPicker already uses, in the
// same shape: header, rows, then a message.
//
// a.hasMenu IS CLEARED HERE, exactly as drawPicker clears it: the canvas may
// still hold a composed menu frame from before OpenChargen armed this screen,
// and the flag is drawMenu's own cache key — leaving it set would let a later
// return to the menu skip its own redraw against a canvas this screen has
// since overwritten.
//
// a.flow.chargen IS CHECKED rather than assumed, for the same reason
// stepChargen checks it: reachable only from a test that sets f.screen
// without arming the model, never from the application's own transitions,
// but a screen that stops drawing is a bug report and a screen that panics
// is an incident.
func (a *App) drawChargen() {
	a.hasMenu = false // the canvas no longer holds a composed menu frame
	a.fillCanvas(pickerBackground)

	if pix, err := a.composeChargenScreen(); err == nil {
		a.writeCanvas(pix)
		return
	}
	c := a.flow.chargen
	if c == nil {
		return
	}

	a.printCanvas(c.HeaderText(), chargenLeft, chargenHeaderY)
	a.printCanvas("Name: "+c.NameText(), chargenLeft, chargenHeaderY+chargenLine)

	n := c.Rows()
	for i := 0; i < n; i++ {
		a.printCanvas(c.RowText(i), chargenLeft, chargenTop+i*chargenLine)
	}
	a.printCanvas(c.FooterText(), chargenLeft, chargenTop+n*chargenLine)

	// THE CONSEQUENCE BLOCK, one blank line under the footer. Its rows are
	// the model's own DerivedText, so this loop decides where the block sits
	// and nothing about what is in it; a setup naming no Derive answers no
	// lines and this loop paints nothing, which is the frame this screen drew
	// before the block existed.
	//
	// IT IS CLIPPED TO WHAT FITS, at both ends of a line: chargenDerivedFit
	// bounds how many lines there is room for and clipRunes bounds how wide
	// each one may be (0140). Neither is a judgement about the block's
	// contents — a block one line too long is still the wiring tier's answer,
	// and dropping its last line is strictly better than painting it over the
	// message line, where two strings would be legible as neither.
	block := c.DerivedText()
	if fit := chargenDerivedFit(n); len(block) > fit {
		block = block[:fit]
	}
	for k, line := range block {
		a.printCanvas(clipRunes(line, chargenCols),
			chargenLeft, chargenTop+(n+chargenDerivedGap+k)*chargenLine)
	}

	if a.flow.msg != "" {
		a.printCanvas(clipRunes(a.flow.msg, chargenCols), chargenLeft, chargenMessageY)
	}
}

// SetWindowIcon gives the game window its icon. Windows picks the taskbar,
// task-switcher and title-bar pictures from images by size. It is called before
// Run and takes effect with the window's first frame; no images leave the
// platform's default icon.
func SetWindowIcon(images []image.Image) { ebiten.SetWindowIcon(images) }

// Run opens the window and blocks until the front-end exits. Esc at the menu and
// the brooch's EXIT button are the two ways out, and both arrive here as the same
// termination value, which Ebitengine reports rather than as an error.
func (a *App) Run() error {
	defer a.StopAudio()
	defer a.FlushBackground()
	ebiten.SetWindowClosingHandled(true)
	defer ebiten.SetWindowClosingHandled(false)
	ebiten.SetWindowTitle(a.title)
	w, h := monitorSize()
	configureStartupWindow(w, h, runtime.GOOS)
	a.PlayStartupCutscenes()

	if err := runGame(a); err != nil && err != ebiten.Termination {
		return err
	}
	return nil
}

package ui

import (
	"errors"
	"reflect"
	"strings"
	"testing"

	"againrom/pkg/render/terrain"
)

func flowFixture() []PickerRow {
	return []PickerRow{
		{Text: "a.alm", Choosable: true},
		{Text: "b.alm", Choosable: true},
		{Text: "c.alm  [unreadable]", Choosable: false},
		{Text: "d.alm", Choosable: true},
		{Text: "e.alm", Choosable: true},
	}
}

// flowLoader returns a MapLoader that records every call and fails for the
// indices in fail. A success builds a real *Viewer the way the other pkg/ui
// tests do — synthetic grid, empty tileset, no game data.
//
// It hands back NO tick and NO order seam, which is a loader's ordinary
// right and is what the shipped loader does today. These subtests drive
// screen transitions, where nothing is advanced and nothing is ordered; the
// seam itself is driven by tickLoader and seamLoader below.
func flowLoader(t *testing.T, fail map[int]error, calls *[]int) MapLoader {
	t.Helper()
	return func(i int) (*Viewer, MapTick, MapOrder, MapCadence, MapAffect, MapAdvance, MapAttack, MapGrab, MapStance, MapMarch, error) {
		*calls = append(*calls, i)
		if err, ok := fail[i]; ok {
			return nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, err
		}
		v, err := NewViewer("t", grid(8, 8), &terrain.Tileset{})
		if err != nil {
			t.Fatalf("NewViewer: %v", err)
		}
		return v, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil
	}
}

// tickLoader returns a MapLoader that gives every successful load a FRESH
// MapTick counting its own calls, appending that load's counter to counts.
//
// Two loads therefore hand over two ticks that are told apart by which counter
// moves — which is what makes "the flow kept the first load's tick" observable
// at all. Comparing the funcs themselves is not an option: Go compares function
// values to nil and nothing else.
//
// The grid is large enough for the camera to move on it without meeting a
// clamp, so a subtest can watch the camera and the count on the same tick.
func tickLoader(t *testing.T, counts *[]*int) MapLoader {
	t.Helper()
	return func(int) (*Viewer, MapTick, MapOrder, MapCadence, MapAffect, MapAdvance, MapAttack, MapGrab, MapStance, MapMarch, error) {
		v, err := NewViewer("t", grid(60, 60), &terrain.Tileset{})
		if err != nil {
			t.Fatalf("NewViewer: %v", err)
		}
		n := new(int)
		*counts = append(*counts, n)
		return v, func() { *n++ }, nil, nil, nil, nil, nil, nil, nil, nil, nil
	}
}

// issued is one order as a test observes it arriving at the far side of the
// seam: the three scalars MapOrder carries and nothing else.
type issued struct {
	entity uint32
	x, y   int
}

// struck is one blow as a test observes it arriving at the far side of the
// seam: the two scalars MapAffect carries and nothing else.
type struck struct {
	entity uint32
	kill   bool
}

// aimed is one attack order as a test observes it arriving at the far side
// of the seam: the two ids MapAttack carries, and the spell id 0127 widened
// it with — zero for every case that predates that story and still builds
// this positionally.
type aimed struct {
	entity uint32
	victim uint32
	spell  uint32
}

// mapSeam is one load's whole share of the front-end's map seam, as the far side
// sees it: how many times that load's MapTick was called, every order its
// MapOrder received, every blow its MapAffect received and every attack order its
// MapAttack received, each in the order it received them.
//
// Both halves are recorded per LOAD rather than per loader, so two openings hand
// out two seams that are told apart by which one moved — the same reason
// tickLoader gives each load its own counter, and the only way to say that the
// flow is driving the load it currently holds.
type mapSeam struct {
	ticks   int
	orders  []issued
	blows   []struck
	attacks []aimed
	// stances and marches are 0146's two seams, recorded the same way: every
	// cell-free standing order and every aimed one this load's far side
	// received, in the order it received them.
	stances []stood
	marches []marched
	grabs   []grabbed
}

// grabbed is one pick-up as a test observes it arriving at the far side: the
// four values MapGrab carries and nothing else.
type grabbed struct {
	entity uint32
	x, y   int
	aimed  bool
}

// stood is one cell-free standing order as a test observes it arriving at
// the far side: the two values MapStance carries and nothing else.
type stood struct {
	entity uint32
	guard  bool
}

// marched is one aimed standing order as a test observes it arriving: the
// four values MapMarch carries.
type marched struct {
	entity uint32
	patrol bool
	x, y   int
}

// seamLoader returns a MapLoader giving every successful load a FRESH mapSeam,
// appended to seams, with a tick and an order that both write to it.
//
// The grid is the large one, so a camera on it moves without meeting a clamp and
// a cursor resolves to a cell far from the extent's edges.
func seamLoader(t *testing.T, seams *[]*mapSeam) MapLoader {
	t.Helper()
	return func(int) (*Viewer, MapTick, MapOrder, MapCadence, MapAffect, MapAdvance, MapAttack, MapGrab, MapStance, MapMarch, error) {
		v, err := NewViewer("t", grid(60, 60), &terrain.Tileset{})
		if err != nil {
			t.Fatalf("NewViewer: %v", err)
		}
		s := &mapSeam{}
		*seams = append(*seams, s)
		return v, func() { s.ticks++ },
			func(entity uint32, x, y int) {
				s.orders = append(s.orders, issued{entity: entity, x: x, y: y})
			}, nil,
			func(entity uint32, kill bool) {
				s.blows = append(s.blows, struck{entity: entity, kill: kill})
			}, nil,
			func(entity, victim, spell uint32, x, y int, cell bool) {
				s.attacks = append(s.attacks, aimed{entity: entity, victim: victim, spell: spell})
			},
			func(entity uint32, x, y int, aimedGrab bool) {
				s.grabs = append(s.grabs, grabbed{entity: entity, x: x, y: y, aimed: aimedGrab})
			},
			func(entity uint32, guard bool) {
				s.stances = append(s.stances, stood{entity: entity, guard: guard})
			},
			func(entity uint32, patrol bool, x, y int) {
				s.marches = append(s.marches, marched{entity: entity, patrol: patrol, x: x, y: y})
			}, nil
	}
}

// flowOn builds a flow parked on the given screen, by driving it there through
// the transitions only — never by assigning to f.screen — so the setup itself
// exercises the state machine rather than side-stepping it.
func flowOn(t *testing.T, s Screen) (*flow, *Picker, *[]int) {
	t.Helper()
	p := NewPicker(flowFixture())
	calls := &[]int{}
	f := newFlow(p, flowLoader(t, nil, calls))
	switch s {
	case ScreenMenu:
	case ScreenPicker:
		f.activateNewGame()
	case ScreenMap:
		f.activateNewGame()
		if !p.Select(0) {
			t.Fatalf("Select(0) = false")
		}
		f.choose()
	default:
		t.Fatalf("flowOn: unknown screen %v", s)
	}
	if f.screen != s {
		t.Fatalf("flowOn(%v): screen = %v, want %v", s, f.screen, s)
	}
	return f, p, calls
}

func TestFlow(t *testing.T) {
	t.Run("a fresh flow starts on the menu", func(t *testing.T) {
		p := NewPicker(flowFixture())
		calls := &[]int{}
		f := newFlow(p, flowLoader(t, nil, calls))
		if f.screen != ScreenMenu {
			t.Fatalf("newFlow: screen = %v, want ScreenMenu", f.screen)
		}
		if f.viewer != nil {
			t.Fatalf("newFlow: viewer = %v, want nil", f.viewer)
		}
		if f.msg != "" {
			t.Fatalf("newFlow: msg = %q, want empty", f.msg)
		}
		if len(*calls) != 0 {
			t.Fatalf("newFlow called the loader %d times, want 0", len(*calls))
		}
	})

	t.Run("the three screens are distinct", func(t *testing.T) {
		seen := map[string]Screen{}
		for _, s := range []Screen{ScreenMenu, ScreenPicker, ScreenMap} {
			name := s.String()
			if name == "" {
				t.Fatalf("Screen(%d).String() is empty", int(s))
			}
			if prev, ok := seen[name]; ok {
				t.Fatalf("Screen(%d) and Screen(%d) both render as %q", int(prev), int(s), name)
			}
			seen[name] = s
		}
		if ScreenMenu == ScreenPicker || ScreenPicker == ScreenMap || ScreenMenu == ScreenMap {
			t.Fatalf("the three Screen constants are not distinct")
		}
	})

	t.Run("activateNewGame moves menu to picker and nowhere else", func(t *testing.T) {
		f, _, _ := flowOn(t, ScreenMenu)
		f.activateNewGame()
		if f.screen != ScreenPicker {
			t.Fatalf("activateNewGame from the menu: screen = %v, want ScreenPicker", f.screen)
		}
		// Already on the picker: a second activation must change nothing.
		f.activateNewGame()
		if f.screen != ScreenPicker {
			t.Fatalf("activateNewGame from the picker: screen = %v, want ScreenPicker", f.screen)
		}

		mf, _, mcalls := flowOn(t, ScreenMap)
		before := mf.viewer
		mf.activateNewGame()
		if mf.screen != ScreenMap {
			t.Fatalf("activateNewGame from the map screen: screen = %v, want ScreenMap", mf.screen)
		}
		if mf.viewer != before {
			t.Fatalf("activateNewGame from the map screen replaced the viewer")
		}
		if len(*mcalls) != 1 {
			t.Fatalf("activateNewGame from the map screen triggered a load: calls = %v", *mcalls)
		}
	})

	// Owner reversal: WITH newGameChargen SET, activateNewGame arms generation
	// directly and never shows the picker — this is now the shipped route, and
	// cmd/againrom installs it for every launch that does not carry -picker.
	// Escape from a screen armed this way must return to the menu, not to a
	// picker the player never saw (pkg/ui/cursorlifecycle_test.go's own
	// "menu -> chargen" transition covers the cursor side of the same rule).
	t.Run("activateNewGame arms generation directly when newGameChargen is set", func(t *testing.T) {
		f, _, _ := flowOn(t, ScreenMenu)
		calls := 0
		f.newGameChargen = func() *ChargenEntry {
			calls++
			return &ChargenEntry{Model: NewChargen(chargenLegalSetup())}
		}
		f.activateNewGame()
		if f.screen != ScreenChargen {
			t.Fatalf("activateNewGame with newGameChargen set: screen = %v, want ScreenChargen", f.screen)
		}
		if f.chargenBack != ScreenMenu {
			t.Fatalf("chargenBack = %v, want ScreenMenu", f.chargenBack)
		}
		if calls != 1 {
			t.Fatalf("newGameChargen called %d times, want exactly 1", calls)
		}

		// A nil model falls back to the picker rather than doing nothing.
		f2, _, _ := flowOn(t, ScreenMenu)
		f2.newGameChargen = func() *ChargenEntry { return nil }
		f2.activateNewGame()
		if f2.screen != ScreenPicker {
			t.Fatalf("activateNewGame with a nil-returning newGameChargen: screen = %v, want ScreenPicker", f2.screen)
		}
	})

	t.Run("a loadable choice moves picker to map", func(t *testing.T) {
		p := NewPicker(flowFixture())
		calls := &[]int{}
		f := newFlow(p, flowLoader(t, nil, calls))
		f.activateNewGame()
		if !p.Select(3) {
			t.Fatalf("Select(3) = false")
		}
		f.choose()
		if f.screen != ScreenMap {
			t.Fatalf("choose() on a loadable row: screen = %v, want ScreenMap", f.screen)
		}
		if f.viewer == nil {
			t.Fatalf("choose() on a loadable row left viewer nil")
		}
		if f.msg != "" {
			t.Fatalf("choose() on a loadable row: msg = %q, want empty", f.msg)
		}
		if len(*calls) != 1 || (*calls)[0] != 3 {
			t.Fatalf("loader calls = %v, want exactly [3]", *calls)
		}
	})

	t.Run("a failing load reports and stays in the picker", func(t *testing.T) {
		p := NewPicker(flowFixture())
		calls := &[]int{}
		loadErr := errors.New("decode grid: payload 512, want 2048")
		f := newFlow(p, flowLoader(t, map[int]error{1: loadErr}, calls))
		f.activateNewGame()
		if !p.Select(1) {
			t.Fatalf("Select(1) = false")
		}
		f.choose()

		if f.screen != ScreenPicker {
			t.Fatalf("after a failing load: screen = %v, want ScreenPicker", f.screen)
		}
		if f.viewer != nil {
			t.Fatalf("after a failing load: viewer = %v, want nil", f.viewer)
		}
		if f.msg == "" {
			t.Fatalf("after a failing load: msg is empty — the failure must be reported")
		}
		if !strings.Contains(f.msg, loadErr.Error()) {
			t.Fatalf("after a failing load: msg = %q, want it to contain %q", f.msg, loadErr.Error())
		}
		rows := p.Rows()
		if len(rows) != len(flowFixture()) {
			t.Fatalf("a failing load changed the row count: %d, want %d",
				len(rows), len(flowFixture()))
		}
		if rows[1].Choosable {
			t.Fatalf("row 1 is still choosable after its load failed, want unusable")
		}
		if rows[1].Text != flowFixture()[1].Text {
			t.Fatalf("row 1 text = %q, want it left listed as %q",
				rows[1].Text, flowFixture()[1].Text)
		}
		for _, i := range []int{0, 3, 4} {
			if !rows[i].Choosable {
				t.Fatalf("a failing load on row 1 also demoted row %d", i)
			}
		}

		// Choosing it again does nothing: it is no longer choosable.
		wantMsg := f.msg
		f.choose()
		if f.screen != ScreenPicker {
			t.Fatalf("choosing the failed row again: screen = %v, want ScreenPicker", f.screen)
		}
		if len(*calls) != 1 {
			t.Fatalf("choosing the failed row again called the loader: calls = %v, want [1]", *calls)
		}
		if f.msg != wantMsg {
			t.Fatalf("choosing the failed row again: msg = %q, want %q unchanged", f.msg, wantMsg)
		}
		if f.viewer != nil {
			t.Fatalf("choosing the failed row again produced a viewer")
		}

		// The failure did not exit: Esc from here still just returns to the menu.
		if exit := f.escape(); exit {
			t.Fatalf("escape() after a failing load = true, want false — a failed load must not exit")
		}
		if f.screen != ScreenMenu {
			t.Fatalf("escape() after a failing load: screen = %v, want ScreenMenu", f.screen)
		}

		// A different choosable row still works afterwards.
		f.activateNewGame()
		if !p.Select(3) {
			t.Fatalf("Select(3) = false")
		}
		f.choose()
		if f.screen != ScreenMap {
			t.Fatalf("choosing a good row after a failure: screen = %v, want ScreenMap", f.screen)
		}
		if f.viewer == nil {
			t.Fatalf("choosing a good row after a failure left viewer nil")
		}
		// DD14: msg is cleared on any successful transition.
		if f.msg != "" {
			t.Fatalf("msg = %q after a successful transition, want it cleared", f.msg)
		}
		if len(*calls) != 2 || (*calls)[1] != 3 {
			t.Fatalf("loader calls = %v, want [1 3]", *calls)
		}
	})

	t.Run("an unchoosable row does nothing at all", func(t *testing.T) {
		p := NewPicker(flowFixture())
		calls := &[]int{}
		f := newFlow(p, flowLoader(t, nil, calls))
		f.activateNewGame()
		if !p.Select(2) {
			t.Fatalf("Select(2) = false")
		}
		f.choose()
		if len(*calls) != 0 {
			t.Fatalf("choose() on an unchoosable row called the loader: calls = %v", *calls)
		}
		if f.screen != ScreenPicker {
			t.Fatalf("choose() on an unchoosable row: screen = %v, want ScreenPicker", f.screen)
		}
		if f.msg != "" {
			t.Fatalf("choose() on an unchoosable row: msg = %q, want empty", f.msg)
		}
		if f.viewer != nil {
			t.Fatalf("choose() on an unchoosable row produced a viewer")
		}
		if p.Selection() != 2 {
			t.Fatalf("choose() on an unchoosable row moved the selection to %d", p.Selection())
		}
	})

	t.Run("escape unwinds and exits only at the menu", func(t *testing.T) {
		mf, _, _ := flowOn(t, ScreenMap)
		if mf.viewer == nil {
			t.Fatalf("precondition: the map screen has no viewer")
		}
		if exit := escapeOut(mf); exit {
			t.Fatalf("escape() from the map screen = true, want false")
		}
		if mf.screen != ScreenPicker {
			t.Fatalf("escape() from the map screen: screen = %v, want ScreenPicker", mf.screen)
		}
		if mf.viewer != nil {
			t.Fatalf("escape() from the map screen left viewer = %v, want nil", mf.viewer)
		}

		pf, _, _ := flowOn(t, ScreenPicker)
		if exit := pf.escape(); exit {
			t.Fatalf("escape() from the picker = true, want false")
		}
		if pf.screen != ScreenMenu {
			t.Fatalf("escape() from the picker: screen = %v, want ScreenMenu", pf.screen)
		}

		nf, _, _ := flowOn(t, ScreenMenu)
		if exit := nf.escape(); !exit {
			t.Fatalf("escape() at the menu = false, want true")
		}

		// Esc reports exit from no screen other than the menu.
		for _, s := range []Screen{ScreenPicker, ScreenMap} {
			f, _, _ := flowOn(t, s)
			if exit := f.escape(); exit {
				t.Fatalf("escape() from %v = true, want false — only the menu exits", s)
			}
		}
	})

	t.Run("full round trip", func(t *testing.T) {
		p := NewPicker(flowFixture())
		calls := &[]int{}
		f := newFlow(p, flowLoader(t, nil, calls))

		want := func(s Screen, what string) {
			t.Helper()
			if f.screen != s {
				t.Fatalf("%s: screen = %v, want %v", what, f.screen, s)
			}
		}

		want(ScreenMenu, "start")
		f.activateNewGame()
		want(ScreenPicker, "menu -> picker")
		if !p.Select(4) {
			t.Fatalf("Select(4) = false")
		}
		f.choose()
		want(ScreenMap, "picker -> map")
		if exit := escapeOut(f); exit {
			t.Fatalf("map -> picker reported exit")
		}
		want(ScreenPicker, "map -> picker")
		if exit := f.escape(); exit {
			t.Fatalf("picker -> menu reported exit")
		}
		want(ScreenMenu, "picker -> menu")
		if exit := f.escape(); !exit {
			t.Fatalf("menu escape = false, want true (exit)")
		}
		if len(*calls) != 1 || (*calls)[0] != 4 {
			t.Fatalf("loader calls over the round trip = %v, want [4]", *calls)
		}
	})

	// FR-9a: "The remaining six buttons ... MUST do nothing else; a click on one
	// of them is consumed and has no effect." DD14 discharges that by giving
	// `flow` no activation entry point but activateNewGame and no per-button
	// command table.
	//
	// EXIT (the seventh non-NEW-GAME button) is deliberately NOT here: DD31
	// puts it in the menu dispatch, because it changes no screen — it ends
	// the program — and `flow`'s vocabulary is screens. So this subtest still
	// says what it says: the flow drives exactly one activation.
	//
	// What this subtest actually establishes:
	//   * `flow` carries exactly the five DD14 fields — in particular no
	//     map/slice/array of per-button commands, which is the shape DD14
	//     rejects by name and the one that would invite an out-of-scope binding.
	//   * `*flow` exports no method at all, so nothing outside package ui can
	//     drive a transition by any route.
	//   * Invoking the only other entry points from the menu changes nothing.
	//
	// What it does NOT establish: reflect does not enumerate unexported methods,
	// so this cannot prove that no second unexported activation method exists.
	// That guarantee is the compiler's — a call to any activation entry point
	// other than activateNewGame does not compile — and this test is a
	// regression guard on the struct shape, not a proof of absence.
	t.Run("no activation for any button other than NEW GAME", func(t *testing.T) {
		ft := reflect.TypeOf(flow{})
		// `tick` joined the five in 0020: one map-screen advance held beside the
		// viewer. `order` joined them in 0028: the way out for a move order, held
		// beside the tick and dropped with it. `cadence` joined them in 0041, set
		// and dropped in those same two statements.
		//
		// `rate` and `stopped` joined them in 0041: the front-end's own cadence,
		// an int and a bool.
		//
		// `affect` joined them in 0033: the fourth member of the map seam, a way
		// to hit a world and nothing else. It is a FIELD and not a queue — what
		// a blow becomes and when it lands are the far side's business, the same
		// division MapOrder keeps.
		//
		// `advance` joined them in 0066: the fifth and last member of the map
		// seam, the way a notice is dismissed and the way the front-end learns
		// where a finished mission sends it. It is a FIELD and not a notice —
		// what is open, which part of it is showing and whether the mission is
		// over are the far side's business, the same division every seam above it
		// keeps.
		//
		// `farRung` and `farStopped` joined them in 0073: what the far side was
		// last TOLD, which is not what `rung` and `stopped` hold. Those two are
		// the player's request, and once a notice can stop the world there are two
		// causes of one stop — so the write rule needs the value that crossed,
		// not the value that was asked for. They are two scalars mirroring the
		// pair they record and not a snapshot of the whole cadence, for `rung` and
		// `stopped`'s own reason: there is no third state for them to hold either.
		// They are bools beside the existing cadence scalars, not a per-key
		// command table. Its persisted-speed extension adds
		// preferredRung/preferredRungSet and persistRung: the next map's normal
		// rung, rung-zero's presence bit and the optional local-settings sink.
		// None is simulation state.
		//
		// `grab` joined them in 0112: the seventh member of the map seam, a way to
		// empty a sack into the world and nothing else. It is a FIELD and not a
		// queue, the same division every seam above it keeps — and it is a func
		// of no arguments, which is what keeps this subtest passing rather than a
		// table of per-key commands appearing beside it.
		//
		// `chargen` and `chargenBegin` joined them in 0119: the generation
		// screen's own model pointer and its confirm callback. Neither is a
		// per-button table either — one is a pointer to the one screen's own
		// state, the other a single func from a confirmed result to what comes
		// next — so this subtest's own shape test is unmoved by their arrival.
		//
		// `chargenBack` and `chargenGate` joined them in 0140: where Escape sends
		// the generation screen, and whether a picker row opens it. The first is a
		// single Screen and the second a single func of a row index — neither is a
		// per-row table, which is the shape this subtest exists to refuse: what a
		// row means is the wiring tier's answer, asked one row at a time, never a
		// map this package holds.
		//
		// `newGameChargen` joined them at the owner's reversal of the picker-first
		// startup: what NEW GAME arms directly instead of the map picker. It is one
		// func of no arguments, on chargenGate's own reasoning — NEW GAME is one
		// button and not a row index, so there is exactly one destination to hold,
		// never a table.
		//
		// `saveGame`, `saveList`, `loadGame`, `menuList`, `menuBack`, `loadList`,
		// `loadBack` and `saves` joined them in 0143. Three are single funcs over
		// strings and bools — one seam each, not a table of per-row commands —
		// and the five beside them are the two screens' own list, the screen each
		// returns to, and what the list's rows were built from. The mini-menu's
		// FOUR ENTRIES are a function returning four rows and not a field here,
		// which is exactly the shape this subtest exists to keep: a per-button
		// command table would have been a map, and it is not.
		wantFields := map[string]bool{
			"screen": false, "picker": false, "load": false, "viewer": false,
			"cursor": false,
			"tick":   false, "order": false, "cadence": false, "affect": false,
			"advance": false, "attack": false, "grab": false,
			"rung": false, "stopped": false, "unpaced": false,
			"preferredRung": false, "preferredRungSet": false, "persistRung": false,
			"farRung": false, "farStopped": false, "farUnpaced": false,
			"msg":     false,
			"chargen": false, "chargenBegin": false,
			"chargenBack": false, "chargenGate": false, "newGameChargen": false, "newGameDirect": false,
			"town": false, "townList": false,
			"endingSeams": false, "ending": false,
			"endingPage": false, "endingTop": false, "endingFocus": false, "endingPress": false,
			"saveGame": false, "saveList": false, "loadGame": false,
			"saveDialogSeams": false, "saveDialog": false,
			"menuList": false, "menuBack": false,
			"loadList": false, "loadBack": false, "saves": false,
			"stance": false, "march": false,
			"menuSurface": false, "menuPage": false, "menuContext": false,
			"menuCanSave": false, "menuCanLoad": false, "menuCanSound": false,
			"menuTips": false, "setMenuTips": false,
			"menuSound": false, "setMenuSound": false,
			"gameOptions": false,
			// Sound settings seam and one captured gesture, not per-button commands.
			"soundOptions": false, "soundPointer": false,
			"tooltip": false, "persistTooltipDelay": false,
			"menuExit": false,
			"words":    false, "menuFont": false,
			"encodeMenuKey": false,
			"docSrc":        false, "docArt": false, "docFont": false,
			"docPanel": false, "docBack": false,
			"selfNotice": false,
			// One pending movie family survives the completed viewer's teardown.
			"completedCutscene": false,
			"hallFromMenu":      false,
			"modUI":             false, "menuArt": false, "helpScroll": false, "questPress": false, "questTop": false, "loadUI": false,
			// The menu buttons' one press latch, not a command table.
			"menuPress": false, "questBar": false,
		}
		if ft.NumField() != len(wantFields) {
			names := make([]string, ft.NumField())
			for i := range names {
				names[i] = ft.Field(i).Name
			}
			t.Fatalf("flow has %d fields %v, want exactly %d (DD14): no per-button command table",
				ft.NumField(), names, len(wantFields))
		}
		for i := 0; i < ft.NumField(); i++ {
			name := ft.Field(i).Name
			seen, ok := wantFields[name]
			if !ok {
				t.Fatalf("flow has an unexpected field %q (screen, picker, load, viewer, msg — DD14; "+
					"tick — 0020 DD-1; order — 0028 DD-5; cadence, rung, stopped — 0041 DD-5, DD-6; "+
					"affect — 0033 DD-9; advance — 0066 DD-9; farRung, farStopped — 0073 DD-2; "+
					"attack — 0075 DD-1; grab — 0112 DD-15; chargen, chargenBegin — 0119 plan DD-5; "+
					"chargenBack, chargenGate — 0140; town, townList — 0142 plan DD-3, DD-5; "+
					"saveGame, saveList, loadGame, menuList, menuBack, loadList, loadBack, saves — 0143 "+
					"FR-6, FR-7, FR-10; words, menuFont — 0168 plan DD-7; encodeMenuKey — 1014 "+
					"MENU-KEY-013 FR-9; cursor — story 1030 B2)", name)
			}
			if seen {
				t.Fatalf("flow declares field %q twice", name)
			}
			wantFields[name] = true
			// `saves` IS THE ONE ALLOWED SEQUENCE AND IT IS NOT A DISPATCH TABLE.
			// What it holds is what the far side's list seam last ANSWERED — a name
			// and a label per file on disk — so it is data this package was handed
			// and passes back unread, in exactly the way `picker`'s own rows are, and
			// it carries no behaviour for any row at all: choosing one calls the
			// single `loadGame` func with a string out of it. The rule this arm
			// exists for is that no field maps a BUTTON to a COMMAND, and a list of
			// file names maps nothing to anything.
			if name == "saves" {
				continue
			}
			switch k := ft.Field(i).Type.Kind(); k {
			case reflect.Map, reflect.Slice, reflect.Array:
				t.Fatalf("flow field %q is a %v — DD14 rejects a per-button dispatch table", name, k)
			}
		}

		if n := reflect.TypeOf(&flow{}).NumMethod(); n != 0 {
			t.Fatalf("*flow exposes %d exported methods, want 0 — the flow is package-internal", n)
		}
		if n := reflect.TypeOf(flow{}).NumMethod(); n != 0 {
			t.Fatalf("flow exposes %d exported methods, want 0", n)
		}

		// A release on one of the six unbound buttons reaches no entry point at
		// all; the closest observable analogue is invoking the picker's own
		// activation while the menu is up. Screen, rows and message must be
		// unchanged.
		f, p, calls := flowOn(t, ScreenMenu)
		beforeRows := append([]PickerRow(nil), p.Rows()...)
		f.choose()
		if f.screen != ScreenMenu {
			t.Fatalf("choose() on the menu screen: screen = %v, want ScreenMenu", f.screen)
		}
		if f.msg != "" {
			t.Fatalf("choose() on the menu screen: msg = %q, want empty", f.msg)
		}
		if f.viewer != nil {
			t.Fatalf("choose() on the menu screen produced a viewer")
		}
		if len(*calls) != 0 {
			t.Fatalf("choose() on the menu screen called the loader: calls = %v", *calls)
		}
		afterRows := p.Rows()
		if len(afterRows) != len(beforeRows) {
			t.Fatalf("the row set changed: %d rows, want %d", len(afterRows), len(beforeRows))
		}
		for i := range beforeRows {
			if afterRows[i] != beforeRows[i] {
				t.Fatalf("row %d changed: %+v, want %+v", i, afterRows[i], beforeRows[i])
			}
		}
	})

	t.Run("the same event sequence is reproducible", func(t *testing.T) {
		run := func() (Screen, string, int, bool) {
			p := NewPicker(flowFixture())
			calls := &[]int{}
			f := newFlow(p, flowLoader(t, map[int]error{1: errors.New("decode: short grid")}, calls))
			f.activateNewGame()
			p.Move(1)            // -> row 1
			f.choose()           // fails: msg set, row 1 demoted
			p.Move(1)            // -> row 2, unchoosable
			f.choose()           // nothing
			p.Move(1)            // -> row 3
			f.choose()           // loads: -> map
			exit := escapeOut(f) //
			p.Move(-2)
			return f.screen, f.msg, p.Selection(), exit
		}

		s1, m1, sel1, e1 := run()
		s2, m2, sel2, e2 := run()
		if s1 != s2 || m1 != m2 || sel1 != sel2 || e1 != e2 {
			t.Fatalf("the same sequence gave (%v, %q, %d, %v) then (%v, %q, %d, %v)",
				s1, m1, sel1, e1, s2, m2, sel2, e2)
		}
		// Non-vacuity: the sequence must actually reach a non-trivial state.
		if s1 != ScreenPicker {
			t.Fatalf("determinism sequence ended on %v, want ScreenPicker", s1)
		}
		if sel1 != 1 {
			t.Fatalf("determinism sequence ended with selection %d, want 1", sel1)
		}
	})
}

func TestFlowHoldsTheMapTickWithTheViewer(t *testing.T) {
	t.Run("a loadable choice stores the viewer and the tick together", func(t *testing.T) {
		counts := &[]*int{}
		p := NewPicker(flowFixture())
		f := newFlow(p, tickLoader(t, counts))
		f.activateNewGame()
		if !p.Select(0) {
			t.Fatalf("Select(0) = false")
		}
		f.choose()

		if f.screen != ScreenMap {
			t.Fatalf("choose(): screen = %v, want ScreenMap", f.screen)
		}
		if f.viewer == nil {
			t.Fatalf("choose() left the viewer nil")
		}
		if f.tick == nil {
			t.Fatalf("choose() left the tick nil while the loader handed one over")
		}
		if len(*counts) != 1 {
			t.Fatalf("the loader handed out %d ticks, want 1", len(*counts))
		}
		if got := *(*counts)[0]; got != 0 {
			t.Errorf("loading a map advanced it %d times, want 0", got)
		}
	})

	t.Run("a failing load stores neither", func(t *testing.T) {
		p := NewPicker(flowFixture())
		calls := &[]int{}
		f := newFlow(p, flowLoader(t, map[int]error{0: errors.New("decode grid: short")}, calls))
		f.activateNewGame()
		if !p.Select(0) {
			t.Fatalf("Select(0) = false")
		}
		f.choose()

		if f.screen != ScreenPicker {
			t.Fatalf("after a failing load: screen = %v, want ScreenPicker", f.screen)
		}
		if f.viewer != nil {
			t.Errorf("after a failing load the flow holds a viewer")
		}
		if f.tick != nil {
			t.Errorf("after a failing load the flow holds a tick")
		}
	})

	// SC-1: "After Esc the flow holds neither viewer nor tick."
	t.Run("escape drops the tick with the viewer", func(t *testing.T) {
		counts := &[]*int{}
		p := NewPicker(flowFixture())
		f := newFlow(p, tickLoader(t, counts))
		f.activateNewGame()
		if !p.Select(0) {
			t.Fatalf("Select(0) = false")
		}
		f.choose()
		if f.viewer == nil || f.tick == nil {
			t.Fatalf("precondition: the map screen holds viewer=%v tick!=nil=%v",
				f.viewer, f.tick != nil)
		}

		if exit := escapeOut(f); exit {
			t.Fatalf("escape() from the map screen = true, want false")
		}
		if f.viewer != nil {
			t.Errorf("escape() left viewer = %v, want nil", f.viewer)
		}
		if f.tick != nil {
			t.Errorf("escape() left the tick behind — whatever it closes over outlives the map screen")
		}
	})

	t.Run("reopening replaces the tick", func(t *testing.T) {
		counts := &[]*int{}
		p := NewPicker(flowFixture())
		f := newFlow(p, tickLoader(t, counts))

		open := func() {
			t.Helper()
			f.activateNewGame()
			if !p.Select(0) {
				t.Fatalf("Select(0) = false")
			}
			f.choose()
			if f.screen != ScreenMap {
				t.Fatalf("open: screen = %v, want ScreenMap", f.screen)
			}
		}

		open()
		f.tick() // one advance on the first world
		escapeOut(f)
		open()
		if len(*counts) != 2 {
			t.Fatalf("two openings handed out %d ticks, want 2", len(*counts))
		}
		f.tick() // one advance on the second

		if got := *(*counts)[0]; got != 1 {
			t.Errorf("the first world advanced %d times, want 1 — the flow kept the wrong tick", got)
		}
		if got := *(*counts)[1]; got != 1 {
			t.Errorf("the second world advanced %d times, want 1 — the flow kept the first load's tick", got)
		}
	})

	// The subtest above holds because TWO independent things are true: escape
	// drops the tick, and choose stores the new one unconditionally. Either alone
	// would carry it, so neither is checked by it — a choose that kept a tick it
	// already held is indistinguishable today, because escape guarantees it never
	// holds one on the picker.
	//
	// So the state is FORCED, exactly as it is unreachable: a flow on the
	// picker already holding a tick. That is not a state this front-end can be
	// in; it is the state in which "choose stores" and "escape drops" stop
	// propping each other up. If a later change stops escape dropping, this is
	// what still says a reopened map is not driven by the map before it.
	t.Run("choose replaces a tick the flow already holds", func(t *testing.T) {
		counts := &[]*int{}
		p := NewPicker(flowFixture())
		f := newFlow(p, tickLoader(t, counts))

		stale := 0
		f.activateNewGame()
		f.tick = func() { stale++ }

		if !p.Select(0) {
			t.Fatalf("Select(0) = false")
		}
		f.choose()
		if f.screen != ScreenMap {
			t.Fatalf("choose(): screen = %v, want ScreenMap", f.screen)
		}
		if len(*counts) != 1 {
			t.Fatalf("the loader handed out %d ticks, want 1", len(*counts))
		}

		f.tick()
		if stale != 0 {
			t.Errorf("choose kept the tick the flow was already holding: it ran %d times", stale)
		}
		if got := *(*counts)[0]; got != 1 {
			t.Errorf("the loaded tick ran %d times, want 1", got)
		}
	})
}

func TestFlowHoldsTheOrderSeamWithTheTick(t *testing.T) {
	// openOn drives a flow to the map screen through the transitions alone.
	openOn := func(t *testing.T, f *flow, p *Picker) {
		t.Helper()
		f.activateNewGame()
		if !p.Select(0) {
			t.Fatalf("Select(0) = false")
		}
		f.choose()
		if f.screen != ScreenMap {
			t.Fatalf("choose(): screen = %v, want ScreenMap", f.screen)
		}
	}

	t.Run("a loadable choice stores the order with the tick", func(t *testing.T) {
		seams := &[]*mapSeam{}
		p := NewPicker(flowFixture())
		f := newFlow(p, seamLoader(t, seams))
		openOn(t, f, p)

		if f.tick == nil || f.order == nil {
			t.Fatalf("choose() left tick non-nil %v and order non-nil %v, want both stored",
				f.tick != nil, f.order != nil)
		}
		if len(*seams) != 1 {
			t.Fatalf("the loader handed out %d seams, want 1", len(*seams))
		}
		s := (*seams)[0]
		if s.ticks != 0 || len(s.orders) != 0 {
			t.Fatalf("loading a map advanced it %d times and ordered %d times, want 0 and 0",
				s.ticks, len(s.orders))
		}

		f.tick()
		f.order(7, 3, 9)
		if s.ticks != 1 {
			t.Errorf("the held tick ran %d times, want 1", s.ticks)
		}
		if want := []issued{{entity: 7, x: 3, y: 9}}; len(s.orders) != 1 || s.orders[0] != want[0] {
			t.Errorf("the held order delivered %+v, want %+v", s.orders, want)
		}
	})

	t.Run("a failing load stores neither half of the seam", func(t *testing.T) {
		p := NewPicker(flowFixture())
		calls := &[]int{}
		f := newFlow(p, flowLoader(t, map[int]error{0: errors.New("decode grid: short")}, calls))
		f.activateNewGame()
		if !p.Select(0) {
			t.Fatalf("Select(0) = false")
		}
		f.choose()

		if f.screen != ScreenPicker {
			t.Fatalf("after a failing load: screen = %v, want ScreenPicker", f.screen)
		}
		if f.tick != nil || f.order != nil {
			t.Errorf("after a failing load the flow holds tick non-nil %v, order non-nil %v, want neither",
				f.tick != nil, f.order != nil)
		}
	})

	t.Run("escape drops the order on the statement it drops the tick", func(t *testing.T) {
		seams := &[]*mapSeam{}
		p := NewPicker(flowFixture())
		f := newFlow(p, seamLoader(t, seams))
		openOn(t, f, p)
		if f.tick == nil || f.order == nil {
			t.Fatalf("precondition: the map screen holds tick non-nil %v, order non-nil %v",
				f.tick != nil, f.order != nil)
		}

		if exit := escapeOut(f); exit {
			t.Fatalf("escape() from the map screen = true, want false")
		}
		if f.tick != nil {
			t.Errorf("escape() left the tick behind")
		}
		if f.order != nil {
			t.Errorf("escape() left the order seam behind — a way into the world the map screen released")
		}
	})

	t.Run("reopening replaces both halves", func(t *testing.T) {
		seams := &[]*mapSeam{}
		p := NewPicker(flowFixture())
		f := newFlow(p, seamLoader(t, seams))

		openOn(t, f, p)
		f.tick()
		f.order(1, 1, 1)
		escapeOut(f)
		openOn(t, f, p)
		if len(*seams) != 2 {
			t.Fatalf("two openings handed out %d seams, want 2", len(*seams))
		}
		f.tick()
		f.order(2, 5, 6)

		first, second := (*seams)[0], (*seams)[1]
		if first.ticks != 1 || len(first.orders) != 1 {
			t.Errorf("the discarded seam took %d ticks and %d orders, want the 1 and 1 it was left at",
				first.ticks, len(first.orders))
		}
		if second.ticks != 1 {
			t.Errorf("the reopened seam took %d ticks, want 1 — the flow kept the first load's tick", second.ticks)
		}
		if want := (issued{entity: 2, x: 5, y: 6}); len(second.orders) != 1 || second.orders[0] != want {
			t.Errorf("the reopened seam took orders %+v, want exactly [%+v] — the flow kept the first "+
				"load's order", second.orders, want)
		}
	})

	// Forced, exactly as the tick's own file forces it: a flow on the picker
	// already holding a seam is a state no transition reaches, and it is the
	// state in which a choose that KEPT what it held would be visible.
	t.Run("choose replaces an order seam the flow already holds", func(t *testing.T) {
		seams := &[]*mapSeam{}
		p := NewPicker(flowFixture())
		f := newFlow(p, seamLoader(t, seams))

		stale := 0
		f.activateNewGame()
		f.order = func(uint32, int, int) { stale++ }

		if !p.Select(0) {
			t.Fatalf("Select(0) = false")
		}
		f.choose()
		if f.screen != ScreenMap {
			t.Fatalf("choose(): screen = %v, want ScreenMap", f.screen)
		}
		if len(*seams) != 1 {
			t.Fatalf("the loader handed out %d seams, want 1", len(*seams))
		}

		f.order(4, 2, 2)
		if stale != 0 {
			t.Errorf("choose kept the order seam the flow was already holding: it took %d orders", stale)
		}
		if got := len((*seams)[0].orders); got != 1 {
			t.Errorf("the loaded order seam took %d orders, want 1", got)
		}
	})
}

// TestCommandModeIsSetWithTheSeamAndDroppedWithIt — 0030 SC-1: the
// viewer's command mode is written on the transition that stores the tick
// and the order, and cleared on the one that drops them, so a viewer under
// the map screen and a front-end holding a seam are one state and not two.
//
// It is asserted THROUGH THE GESTURE and not only on the field, because the
// field is not what the contract is about: a mode set on a viewer that then pans
// by a plain drag anyway would satisfy a field read and nothing else. The drag
// below is the shipped three-tick gesture, so what these subtests measure is the
// camera the front-end's map screen would actually have moved.
func TestCommandModeIsSetWithTheSeamAndDroppedWithIt(t *testing.T) {
	// panOf runs the shipped plain drag over a viewer AS IT STANDS and reports
	// how far the camera travelled.
	panOf := func(t *testing.T, v *Viewer) (dx, dy float64) {
		t.Helper()
		layoutViewport(v, 800, 600)
		v.Camera().X, v.Camera().Y = 500, 500
		v.Camera().Clamp()
		x0, y0 := v.Camera().X, v.Camera().Y
		for _, p := range [3][2]int{{400, 300}, {450, 320}, {420, 290}} {
			v.step(Input{PrimaryDown: true, CursorX: p[0], CursorY: p[1]}, dragFrozen)
		}
		return v.Camera().X - x0, v.Camera().Y - y0
	}

	t.Run("a loaded map screen's plain drag pans by zero", func(t *testing.T) {
		seams := &[]*mapSeam{}
		p := NewPicker(flowFixture())
		f := newFlow(p, seamLoader(t, seams))
		f.activateNewGame()
		if !p.Select(0) {
			t.Fatalf("Select(0) = false")
		}
		f.choose()
		if f.screen != ScreenMap {
			t.Fatalf("choose(): screen = %v, want ScreenMap", f.screen)
		}
		if !f.viewer.commandMode {
			t.Fatalf("choose() left the viewer out of command mode while holding tick %v and order %v",
				f.tick != nil, f.order != nil)
		}
		if dx, dy := panOf(t, f.viewer); dx != 0 || dy != 0 {
			t.Errorf("the map screen's plain drag panned by (%v,%v), want (0,0)", dx, dy)
		}
	})

	// The same viewer, after Esc: the flow drops the seam and the mode with it,
	// so a caller still holding that pointer holds a plain terrain viewer.
	t.Run("Esc drops the mode on the same statement it drops the seam", func(t *testing.T) {
		seams := &[]*mapSeam{}
		p := NewPicker(flowFixture())
		f := newFlow(p, seamLoader(t, seams))
		f.activateNewGame()
		if !p.Select(0) {
			t.Fatalf("Select(0) = false")
		}
		f.choose()
		v := f.viewer

		if exit := escapeOut(f); exit {
			t.Fatalf("escape() from the map screen reported exit, want false")
		}
		if f.tick != nil || f.order != nil {
			t.Fatalf("escape() kept the seam: tick %v, order %v", f.tick != nil, f.order != nil)
		}
		if v.commandMode {
			t.Errorf("escape() dropped the seam and left the viewer in command mode")
		}
		if dx, dy := panOf(t, v); dx == 0 && dy == 0 {
			t.Errorf("the dropped viewer's plain drag still panned by zero")
		}
	})

	t.Run("a viewer the flow never loaded is not in command mode", func(t *testing.T) {
		v, err := NewViewer("standalone", grid(60, 60), &terrain.Tileset{})
		if err != nil {
			t.Fatalf("NewViewer: %v", err)
		}
		if v.commandMode {
			t.Fatalf("a freshly built viewer is already in command mode")
		}
		if dx, dy := panOf(t, v); dx == 0 && dy == 0 {
			t.Errorf("the standalone viewer's plain drag panned by zero")
		}
	})
}

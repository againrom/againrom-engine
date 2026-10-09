package ui

import (
	"errors"
	"image"
	"reflect"
	"testing"
	"time"

	"github.com/hajimehoshi/ebiten/v2"

	"againrom/pkg/render/frame"
	"againrom/pkg/render/terrain"
)

// Separate-context acceptance test for the town screen. Every expectation is
// transcribed from spec.md and plan.md, not read back out of the code.

// fakeTown is a TownScreen with two rooms and a countable history, so what is
// asserted here is the FLOW's use of the seam and not any particular town.
//
// IT KNOWS NOTHING ABOUT MISSIONS OR CAMPAIGNS, which is the point of the seam
// being an interface: pkg/ui is written against five methods over strings, ints
// and one callback, and this fixture is the whole of what those five can say.
type fakeTown struct {
	inside  bool
	chosen  []int
	backs   int
	open    MapOpener
	msg     string
	headers []string
}

type fakeSquareTown struct{ fakeTown }

func (f *fakeSquareTown) AtTownSquare() bool { return !f.inside }

func (f *fakeTown) Header() string {
	if f.inside {
		return "inside"
	}
	return "square"
}

func (f *fakeTown) Rows() []TownRow {
	f.headers = append(f.headers, f.Header())
	if f.inside {
		return []TownRow{
			{Text: "a line about the room"},
			{Text: "something to press", Choosable: true},
		}
	}
	return []TownRow{
		{Text: "a door", Choosable: true},
		{Text: "another door", Choosable: true},
		{Text: "a shut door"},
	}
}

func (f *fakeTown) Footer() []string { return []string{"gold 0"} }

func (f *fakeTown) Choose(i int) TownAction {
	f.chosen = append(f.chosen, i)
	if !f.inside {
		f.inside = true
		return TownAction{}
	}
	return TownAction{Open: f.open, Msg: f.msg}
}

type fakeTownDialogue struct {
	fakeTown
	advances int
	pic      *image.RGBA
}

type fakeSurfaceDialogueTown struct {
	fakeTownDialogue
	surfaceClicks []TownSurfaceControl
	selected      bool
}

type fakeStatsSurfaceTown struct {
	fakeTown
	surfaceClicks []TownSurfaceControl
}

func (f *fakeStatsSurfaceTown) AtTownSurface() bool { return true }
func (f *fakeStatsSurfaceTown) TownSurface() TownSurfaceView {
	v := TownSurfaceView{Kind: TownSurfaceTavern,
		Hero: TownCharacterView{HasSubject: true, Statistics: true, MemberCount: 2}}
	v.Cells = []TownSurfaceCell{{Label: "NPC 22", Enabled: true}}
	v.Buttons = []TownSurfaceButton{{Label: "Talk", Enabled: true}}
	return v
}
func (f *fakeStatsSurfaceTown) TownSurfaceClick(c TownSurfaceControl, _ bool) TownAction {
	f.surfaceClicks = append(f.surfaceClicks, c)
	return TownAction{}
}

func (f *fakeSurfaceDialogueTown) AtTownSurface() bool { return true }
func (f *fakeSurfaceDialogueTown) TownSurface() TownSurfaceView {
	v := TownSurfaceView{Kind: TownSurfaceTavern, HoverCell: -1}
	v.Cells = []TownSurfaceCell{{Label: "NPC 22", Enabled: true}}
	v.Buttons = []TownSurfaceButton{{Label: "Hire", Enabled: true}, {Label: "Talk", Enabled: f.selected}}
	return v
}
func (f *fakeSurfaceDialogueTown) TownSurfaceClick(c TownSurfaceControl, _ bool) TownAction {
	f.surfaceClicks = append(f.surfaceClicks, c)
	if c.Kind == TownSurfaceControlCell {
		f.selected = true
	}
	if c.Kind == TownSurfaceControlButton && c.Index == 1 {
		f.pic = image.NewRGBA(image.Rect(0, 0, 240, 120))
	}
	return TownAction{}
}

func (f *fakeTownDialogue) TownDialogue() (*image.RGBA, bool) { return f.pic, f.pic != nil }
func (f *fakeTownDialogue) TownDialogueButton() (image.Rectangle, bool) {
	return image.Rect(100, 60, 180, 90), f.pic != nil
}
func (f *fakeTownDialogue) AdvanceTownDialogue() TownAction {
	f.advances++
	return TownAction{}
}

func TestTownDialogueOwnsInputAboveTheSharedSurface(t *testing.T) {
	town := &fakeSurfaceDialogueTown{}
	a := newTestApp(t, appRows(3), okLoader(t))
	a.SetTown(town)
	if !a.flow.showTown("") {
		t.Fatal("showTown refused the shared surface")
	}
	now := time.Unix(1_700_000_000, 0)
	r := townSurfaceButtonRect(TownSurfaceTavern, 0)
	x, y := (r.Min.X+r.Max.X)/2, (r.Min.Y+r.Max.Y)/2

	// The composed surface works when uncovered.
	a.stepTownAt(appInput{CursorX: x, CursorY: y, PrimaryPressed: true}, now)
	a.stepTownAt(appInput{CursorX: x, CursorY: y, PrimaryReleased: true}, now)
	if len(town.surfaceClicks) != 1 {
		t.Fatalf("uncovered surface clicks = %v", town.surfaceClicks)
	}

	// The same press/release cannot reach it while a dialogue is present;
	// Enter belongs to the dialogue too.
	town.pic = image.NewRGBA(image.Rect(0, 0, 240, 120))
	a.stepTownAt(appInput{CursorX: x, CursorY: y, PrimaryPressed: true}, now)
	a.stepTownAt(appInput{CursorX: x, CursorY: y, PrimaryReleased: true}, now)
	a.stepTownAt(appInput{Enter: true}, now)
	if len(town.surfaceClicks) != 1 || town.advances != 1 {
		t.Fatalf("covered input = clicks %v, dialogue advances %d", town.surfaceClicks, town.advances)
	}
}

// TestAppRoutesThroughTownStatisticsSurface proves 1022 spec B6: the
// DOLL/STATS toggle no longer replaces the room's own left content, so a
// room cell reaches App dispatch in Statistics mode exactly as it does in
// DOLL mode, alongside the persistent upper control.
func TestAppRoutesThroughTownStatisticsSurface(t *testing.T) {
	town := &fakeStatsSurfaceTown{}
	a := newTestApp(t, appRows(3), okLoader(t))
	a.SetTown(town)
	if !a.flow.showTown("") {
		t.Fatal("showTown refused the statistics surface")
	}
	now := time.Unix(1_700_000_000, 0)
	click := func(p image.Point) {
		a.stepTownAt(appInput{CursorX: p.X, CursorY: p.Y, PrimaryPressed: true}, now)
		a.stepTownAt(appInput{CursorX: p.X, CursorY: p.Y, PrimaryReleased: true}, now)
	}

	cell := townSurfaceCellRect(TownSurfaceTavern, 0, 1)
	click(image.Pt((cell.Min.X+cell.Max.X)/2, (cell.Min.Y+cell.Max.Y)/2))
	button := townSurfaceButtonRect(TownSurfaceTavern, 0)
	click(image.Pt((button.Min.X+button.Max.X)/2, (button.Min.Y+button.Max.Y)/2))
	want := []TownSurfaceControl{
		{Kind: TownSurfaceControlCell, Index: 0},
		{Kind: TownSurfaceControlButton, Index: 0},
	}
	if !reflect.DeepEqual(town.surfaceClicks, want) {
		t.Fatalf("statistics-mode clicks = %+v, want %+v", town.surfaceClicks, want)
	}
}

func TestHeadlessActivateUsesSharedTownSurfaceControls(t *testing.T) {
	town := &fakeSurfaceDialogueTown{}
	a := newTestApp(t, appRows(3), okLoader(t))
	a.SetTown(town)
	if !a.flow.showTown("") {
		t.Fatal("showTown refused the shared surface")
	}

	if err := a.HeadlessActivate("NPC 22"); err != nil {
		t.Fatalf("select NPC = %v", err)
	}
	if err := a.HeadlessActivate("dialogue"); err != nil {
		t.Fatalf("advance dialogue = %v", err)
	}
	want := []TownSurfaceControl{
		{Kind: TownSurfaceControlCell, Index: 0},
		{Kind: TownSurfaceControlButton, Index: 1},
	}
	if !reflect.DeepEqual(town.surfaceClicks, want) {
		t.Fatalf("surface clicks = %+v, want %+v", town.surfaceClicks, want)
	}
	if town.advances != 1 {
		t.Fatalf("dialogue advances = %d, want 1", town.advances)
	}
}

// TestHeadlessActivateRoutesThroughTownStatisticsSurface is the production
// semantic seam witness for the same painted-surface ownership as
// TestAppRoutesThroughTownStatisticsSurface (1022 spec B6). A retained room
// cell is activatable in Statistics mode exactly as in DOLL mode, alongside
// the persistent right-side command.
func TestHeadlessActivateRoutesThroughTownStatisticsSurface(t *testing.T) {
	town := &fakeStatsSurfaceTown{}
	a := newTestApp(t, appRows(3), okLoader(t))
	a.SetTown(town)
	if !a.flow.showTown("") {
		t.Fatal("showTown refused the statistics surface")
	}

	// "NPC 22" both selects the cell and presses its Talk button as one
	// semantic verb (headlessActivateTownSurface's own NPC-prefix rule); it
	// answers here only because the cell is now visible in Statistics mode
	// (B6). This also exercises the persistent right-side button.
	if err := a.HeadlessActivate("NPC 22"); err != nil {
		t.Fatalf("statistics-mode cell = %v", err)
	}
	want := []TownSurfaceControl{
		{Kind: TownSurfaceControlCell, Index: 0},
		{Kind: TownSurfaceControlButton, Index: 0},
	}
	if !reflect.DeepEqual(town.surfaceClicks, want) {
		t.Fatalf("statistics-mode clicks = %+v, want %+v", town.surfaceClicks, want)
	}
}

func (f *fakeTown) Back() bool {
	f.backs++
	if !f.inside {
		return false
	}
	f.inside = false
	return true
}

// townFlow is a flow standing on the town screen, with the seam installed.
func townFlow(t *testing.T, town TownScreen) *flow {
	t.Helper()
	f := newFlow(NewPicker(flowFixture()), nil)
	f.town = town
	if !f.showTown("") {
		t.Fatal("showTown refused a town that is not nil")
	}
	return f
}

// fakeShopTown is a TownScreen that also implements the trade seam, so what is
// asserted is the FLOW's use of it: a click on an occupied place crosses once
// and the row list is rebuilt afterwards.
type fakeShopTown struct {
	fakeTown
	clicked    []int
	controls   []ShopControl
	shifted    []bool // c.Shift as ShopClick received it, index for index with clicked
	scrolled   []scrolledBy
	dragged    [][2]ShopControl
	suppressed []int
	// mask is the ShopScreen's own SlotMask (1005 round 2), nil unless a test
	// sets it: the drag machine's App-level tests need a doll to press on,
	// and no test before this round asked ShopScreen for one. It is the
	// ORDINARY, unsuppressed mask: ShopScreen below substitutes a suppressed
	// COPY into SlotMask while suppressSlot is armed, mask itself is never
	// mutated.
	mask *SlotMask
	// dollIcon is slot 4's (one-based; index 3) own SlotIcon (round-2
	// adversarial review, item 2): the drag-icon capture tests need a doll
	// origin that actually resolves a picture, and no test before this round
	// asked ShopScreen for one.
	dollIcon     *image.RGBA
	suppressSlot int
	stats        bool
	character    TownCharacterView
}

// scrolledBy is one wheel event as the seam received it.
type scrolledBy struct {
	region ShopWheelRegion
	rows   int
}

func (f *fakeShopTown) AtTownShop() bool { return true }

func (f *fakeShopTown) ShopScreen() ShopScreenView {
	v := ShopScreenView{Purse: 100, Buy: 40, Sell: 9, Chosen: -1, Live: [4]bool{true, true, true, true}}
	v.Table[0] = ShopCell{Back: ShopBackItem, Price: 40}
	v.Table[1] = ShopCell{Back: ShopBackItem, Price: 9, Mine: true, Info: []string{"a sword"}}
	v.OrdinaryDollMask = f.mask
	v.SlotMask = f.mask
	if f.suppressSlot != 0 {
		v.SlotMask = suppressedCopy(f.mask, f.suppressSlot)
	}
	if f.dollIcon != nil {
		v.SlotIcon[3] = f.dollIcon
	}
	v.Character = f.character
	if !v.Character.HasSubject {
		v.Character = TownCharacterView{HasSubject: true, MemberCount: 1}
	}
	v.Character.Statistics = f.stats
	return v
}

// suppressedCopy is refreshShopDrag's own "the origin slot's own code
// cleared" (pkg/game/shopview.go), restated as a mask edit for this file's
// own fixture, which has no compositor of its own to recompose a figure
// with: every pixel the ordinary mask names `slot` no longer draws that
// slot's own layer, so a copy with those pixels zeroed is what the running
// game's own suppressed mask would show wherever nothing else occupies them.
func suppressedCopy(m *SlotMask, slot int) *SlotMask {
	if m == nil {
		return nil
	}
	out := &SlotMask{W: m.W, H: m.H, Slot: make([]uint8, len(m.Slot))}
	copy(out.Slot, m.Slot)
	for i, n := range out.Slot {
		if int(n) == slot {
			out.Slot[i] = 0
		}
	}
	return out
}

func (f *fakeShopTown) ShopClick(c ShopControl) TownAction {
	f.clicked = append(f.clicked, c.Index)
	f.controls = append(f.controls, c)
	f.shifted = append(f.shifted, c.Shift)
	return TownAction{Msg: "moved"}
}

func (f *fakeShopTown) ShopScroll(region ShopWheelRegion, rows int) {
	f.scrolled = append(f.scrolled, scrolledBy{region, rows})
}

// ShopDrag and ShopSuppressDoll complete the widened TownShopScreen seam
// (1005 round 2). Neither is exercised by this file's own tests — those
// live in shopdrag_test.go — so both are recorded rather than acted on,
// fakeTown's own no-op shape restated for two more calls.
func (f *fakeShopTown) ShopDrag(from, to ShopControl) TownAction {
	f.dragged = append(f.dragged, [2]ShopControl{from, to})
	return TownAction{Msg: "dragged"}
}

func (f *fakeShopTown) ShopSuppressDoll(slot int) {
	f.suppressed = append(f.suppressed, slot)
	f.suppressSlot = slot
}

// fakeShopDialogueTown is a shop room whose own conversation is open over it
// — pkg/game's AtTownShop answers true through exactly this state
// (townscreen.go: "t.room == roomShop || t.room == roomTalk &&
// t.dialogueBuilding == TownShop"), which is reached by entering the shop
// while it has an offer to state (for instance on returning from a mission).
// headerCalls counts Header(), which the room-list fallback drawTown falls
// back to reads and the composed shop screen never does; it is the hotfix's
// own witness.
type fakeShopDialogueTown struct {
	fakeShopTown
	pic         *image.RGBA
	headerCalls int
}

func (f *fakeShopDialogueTown) Header() string {
	f.headerCalls++
	return f.fakeShopTown.Header()
}

func (f *fakeShopDialogueTown) TownDialogue() (*image.RGBA, bool) { return f.pic, f.pic != nil }
func (f *fakeShopDialogueTown) TownDialogueButton() (image.Rectangle, bool) {
	return image.Rect(100, 60, 180, 90), f.pic != nil
}
func (f *fakeShopDialogueTown) AdvanceTownDialogue() TownAction { return TownAction{} }

// The owner reported that a dialogue opened over the shop — for instance
// on entering after mission 20 — hides the shop screen behind it (owner).
// That fallback is witnessed here by Header(): the composed shop screen
// never calls it, so a call means the fallback ran and the shop was not
// painted underneath.
func TestTheShopComposesBehindItsOwnDialogue(t *testing.T) {
	town := &fakeShopDialogueTown{pic: image.NewRGBA(image.Rect(0, 0, 240, 120))}
	a := newTestApp(t, appRows(3), okLoader(t))
	a.SetTown(town)
	if !a.flow.showTown("") {
		t.Fatal("showTown refused the dialogue-over-shop town")
	}
	a.canvas = ebiten.NewImage(frame.W, frame.H)

	a.drawTown()

	if town.headerCalls != 0 {
		t.Errorf("drawTown read the room list's Header() %d time(s) while the shop's own dialogue was open: "+
			"the shop screen was not composed and the room-list fallback ran instead, which is the owner's "+
			"report that the shop is not visible behind the dialogue", town.headerCalls)
	}
}

func TestAClickOnATablePlaceCrossesTheSeamAndRebuildsTheList(t *testing.T) {
	town := &fakeShopTown{}
	f := townFlow(t, town)
	rows := len(town.headers)

	if !f.clickShop(ShopControl{Kind: ShopControlTableCell, Index: 1}) {
		t.Fatal("clickShop refused a town standing in the shop")
	}
	if !reflect.DeepEqual(town.clicked, []int{1}) {
		t.Fatalf("clicked = %v, want exactly place 1", town.clicked)
	}
	if f.msg != "moved" {
		t.Errorf("msg = %q, want the action's own sentence", f.msg)
	}
	if len(town.headers) <= rows {
		t.Error("the row list was not rebuilt after the click")
	}

	// A town with no trade seam answers false and nothing is called.
	if townFlow(t, &fakeTown{}).clickShop(ShopControl{Kind: ShopControlTableCell}) {
		t.Error("a town with no shop seam accepted a table click")
	}
}

// The application's own dispatch fills in Shift from the held key, one
// statement after ShopControlAt's hit test (owner, DIV-046, DIV-047): a
// plain release reaches ShopClick with Shift false and a release with the
// key held reaches it with Shift true, over the SAME table cell.
func TestTheApplicationFillsInTheShiftModifierFromTheHeldKey(t *testing.T) {
	town := &fakeShopTown{}
	a := newTestApp(t, appRows(3), okLoader(t))
	a.SetTown(town)
	if !a.flow.showTown("") {
		t.Fatal("showTown refused the shop town")
	}

	r := ShopTableCellRect(1)
	x, y := (r.Min.X+r.Max.X)/2, (r.Min.Y+r.Max.Y)/2
	now := time.Unix(1_700_000_000, 0)

	a.step(appInput{CursorX: x, CursorY: y, PrimaryReleased: true}, now)
	a.step(appInput{CursorX: x, CursorY: y, PrimaryReleased: true, ShiftHeld: true}, now)

	want := []bool{false, true}
	if !reflect.DeepEqual(town.shifted, want) {
		t.Fatalf("ShopClick saw Shift=%v over two releases, want %v", town.shifted, want)
	}
	if !reflect.DeepEqual(town.clicked, []int{1, 1}) {
		t.Fatalf("clicked = %v, want the same cell both times", town.clicked)
	}
}

// The town is entered by a mission ending into it, through the same teardown
// every other destination uses, and it carries the seam's own sentence.
func TestANoticeCanSendTheFrontEndToTheTown(t *testing.T) {
	town := &fakeTown{}
	f := newFlow(NewPicker(flowFixture()), nil)
	f.town = town

	v, err := NewViewer("t", grid(8, 8), &terrain.Tileset{})
	if err != nil {
		t.Fatalf("NewViewer: %v", err)
	}
	v.SetFont(panelFont())
	v.SetNotice("over", NoticeOutcome)
	f.enter(v, func() {}, nil, nil, nil,
		func(...NoticeAction) (NoticeDest, string, MapOpener) { return NoticeToTown, "you are home", nil }, nil, nil, nil, nil)
	if !v.NoticeOpen() {
		t.Fatal("setup: the outcome notice is not open")
	}

	f.advanceNotice()

	if f.screen != ScreenTown {
		t.Fatalf("screen %v, want ScreenTown", f.screen)
	}
	if f.msg != "you are home" {
		t.Errorf("message %q, want the seam's own sentence", f.msg)
	}
	if f.viewer != nil || f.tick != nil || f.advance != nil {
		t.Error("the map screen was not torn down on the way to the town")
	}
	if f.townList == nil {
		t.Fatal("the town is showing over no list")
	}
	if v.commandMode {
		t.Error("the ended mission's viewer kept command mode")
	}
}

// The release that dismisses an outcome is consumed by the button even when
// it changes screens. Town doors activate on release, so letting it reach the
// newly opened town would enter whichever door lay under the pointer.
func TestMissionCompleteClickIsConsumedAcrossTheTownTransition(t *testing.T) {
	town := &fakeSquareTown{}
	a, seam := noticeApp(t)
	a.SetTown(town)
	seam.dest = NoticeToTown
	seam.msg = "mission complete"
	seam.v.SetNotice("MISSION COMPLETE", NoticeOutcome)

	l := AuthoredOutcomeLayout()
	button := l.Button.Add(l.Box.Min)
	now := time.Unix(1_700_000_000, 0)
	bx, by := (button.Min.X+button.Max.X)/2, (button.Min.Y+button.Max.Y)/2
	a.step(appInput{PrimaryPressed: true, CursorX: bx, CursorY: by}, now)
	if a.Screen() != ScreenMap {
		t.Fatalf("screen after outcome press = %v, want the map until the release", a.Screen())
	}
	a.step(appInput{PrimaryReleased: true, CursorX: bx, CursorY: by}, now)
	if a.Screen() != ScreenTown {
		t.Fatalf("screen after outcome release = %v, want ScreenTown", a.Screen())
	}
	if len(town.chosen) != 0 {
		t.Fatalf("outcome release chose town rows %v", town.chosen)
	}

	x, y, w, h := townSquareRect(0)
	doorX, doorY := x+w/2, y+h/2
	a.step(appInput{CursorX: doorX, CursorY: doorY}, now)
	if len(town.chosen) != 0 {
		t.Fatalf("outcome release fell through to town rows %v", town.chosen)
	}

	// Only a fresh press/release gesture may enter the door.
	a.step(appInput{PrimaryPressed: true, CursorX: doorX, CursorY: doorY}, now)
	a.step(appInput{PrimaryReleased: true, CursorX: doorX, CursorY: doorY}, now)
	if !reflect.DeepEqual(town.chosen, []int{0}) {
		t.Fatalf("fresh town click chose %v, want [0]", town.chosen)
	}
}

// A front end with NO town lands on the map list holding the same sentence,
// rather than on a torn-down map screen.
func TestNoticeToTownWithNoTownLandsOnTheMapList(t *testing.T) {
	f := newFlow(NewPicker(flowFixture()), nil)
	v, err := NewViewer("t", grid(8, 8), &terrain.Tileset{})
	if err != nil {
		t.Fatalf("NewViewer: %v", err)
	}
	v.SetFont(panelFont())
	v.SetNotice("over", NoticeOutcome)
	f.enter(v, func() {}, nil, nil, nil,
		func(...NoticeAction) (NoticeDest, string, MapOpener) { return NoticeToTown, "nowhere to go", nil }, nil, nil, nil, nil)

	f.advanceNotice()

	if f.screen != ScreenPicker {
		t.Fatalf("screen %v, want ScreenPicker", f.screen)
	}
	if f.msg != "nowhere to go" {
		t.Errorf("message %q, want the seam's own sentence", f.msg)
	}
}

// Escape unwinds a ROOM at a time and then the town. The far side gets first
// refusal and answers false only at the square.
func TestEscapeUnwindsARoomAndThenTheTown(t *testing.T) {
	town := &fakeTown{}
	f := townFlow(t, town)

	f.chooseTown() // walk into a room

	if exit := f.escape(); exit {
		t.Fatal("Escape in a town room asked to exit the program")
	}
	if f.screen != ScreenTown {
		t.Fatalf("screen %v after leaving a room, want ScreenTown", f.screen)
	}
	if town.backs != 1 {
		t.Errorf("Back called %d times, want 1", town.backs)
	}

	if exit := escapeOut(f); exit {
		t.Fatal("Escape at the square asked to exit the program")
	}
	if f.screen != ScreenMenu {
		t.Errorf("screen %v after leaving the town, want ScreenMenu", f.screen)
	}
	if f.town == nil {
		t.Error("the town seam was dropped on leaving — a mission may still end into it")
	}
}

// The row list is rebuilt on Choose and on Back and at no other time, so
// what is drawn is what the model holds one press ago and never two.
func TestTheTownListIsRebuiltOnEveryMutation(t *testing.T) {
	town := &fakeTown{}
	f := townFlow(t, town)

	if n := f.townList.Len(); n != 3 {
		t.Fatalf("the square shows %d rows, want 3", n)
	}
	f.townList.Select(1)
	f.chooseTown()
	if n := f.townList.Len(); n != 2 {
		t.Fatalf("the room shows %d rows, want 2 — the list was not rebuilt", n)
	}
	if f.townList.Selection() != 0 {
		t.Error("the selection did not reset on a room change")
	}
	f.escape()
	if n := f.townList.Len(); n != 3 {
		t.Errorf("the square shows %d rows after Back, want 3", n)
	}
}

func TestAnUnchoosableTownRowIsInert(t *testing.T) {
	town := &fakeTown{}
	f := townFlow(t, town)

	f.townList.Select(2) // "a shut door"
	f.chooseTown()

	if len(town.chosen) != 0 {
		t.Errorf("the far side was asked about %v, want nothing", town.chosen)
	}
	if f.screen != ScreenTown {
		t.Errorf("screen %v, want ScreenTown", f.screen)
	}
}

// Walking out of the gates enters a map screen through the SAME enter every
// other door uses — so the cadence ladder is established and the viewer is in
// command mode (FR-8a).
func TestWalkingOutOfTheGatesEntersAMapScreen(t *testing.T) {
	v, err := NewViewer("t", grid(8, 8), &terrain.Tileset{})
	if err != nil {
		t.Fatalf("NewViewer: %v", err)
	}
	town := &fakeTown{open: func() (*Viewer, MapTick, MapOrder, MapCadence, MapAffect, MapAdvance, MapAttack, MapGrab, MapStance, MapMarch, error) {
		return v, func() {}, nil, nil, nil, nil, nil, nil, nil, nil, nil
	}}
	f := townFlow(t, town)

	f.chooseTown()       // into the room
	f.townList.Select(1) // the row carrying the opener
	f.chooseTown()

	if f.screen != ScreenMap {
		t.Fatalf("screen %v, want ScreenMap", f.screen)
	}
	if f.viewer != v {
		t.Error("the flow adopted some other viewer")
	}
	if !v.commandMode {
		t.Error("the map screen was entered without command mode — enter() was bypassed")
	}
	if f.rung != terrain.CadenceRung(terrain.SpeedIndexPeriod(terrain.DefaultSpeedIndex)) {
		t.Error("the cadence ladder was not established — enter() was bypassed")
	}
}

// An opener that fails leaves the player IN THE TOWN holding the failure's own
// words, which is choose()'s own answer to the same event on the map list.
func TestAFailedOpenerLeavesThePlayerInTheTown(t *testing.T) {
	town := &fakeTown{open: func() (*Viewer, MapTick, MapOrder, MapCadence, MapAffect, MapAdvance, MapAttack, MapGrab, MapStance, MapMarch, error) {
		return nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, errors.New("that road is shut")
	}}
	f := townFlow(t, town)

	f.chooseTown()
	f.townList.Select(1)
	f.chooseTown()

	if f.screen != ScreenTown {
		t.Fatalf("screen %v, want ScreenTown", f.screen)
	}
	if f.msg != "that road is shut" {
		t.Errorf("message %q, want the opener's own words", f.msg)
	}
	if f.viewer != nil {
		t.Error("a failed opener left a viewer adopted")
	}
}

func TestTheTownAppendedToBothEnums(t *testing.T) {
	for _, c := range []struct {
		got  int
		want int
		name string
	}{
		{int(ScreenMenu), 0, "ScreenMenu"},
		{int(ScreenPicker), 1, "ScreenPicker"},
		{int(ScreenMap), 2, "ScreenMap"},
		{int(ScreenChargen), 3, "ScreenChargen"},
		{int(ScreenTown), 4, "ScreenTown"},
		{int(NoticeStay), 0, "NoticeStay"},
		{int(NoticeToMenu), 1, "NoticeToMenu"},
		{int(NoticeToMapList), 2, "NoticeToMapList"},
		{int(NoticeToMission), 3, "NoticeToMission"},
		{int(NoticeToTown), 4, "NoticeToTown"},
	} {
		if c.got != c.want {
			t.Errorf("%s = %d, want %d — a shifted constant retargets every switch on it",
				c.name, c.got, c.want)
		}
	}
	if ScreenTown.String() != "town" {
		t.Errorf("ScreenTown.String() = %q, want \"town\"", ScreenTown.String())
	}
}

// The town is drawn and hit-tested through the MAP LIST'S OWN model, so a
// row that is drawn is a row that can be clicked.
func TestTheTownIsHitTestedByTheMapListsOwnModel(t *testing.T) {
	town := &fakeTown{}
	f := townFlow(t, town)

	top, n := f.townList.Visible()
	if top != 0 || n != 3 {
		t.Fatalf("visible = (%d, %d), want (0, 3)", top, n)
	}
	for k := 0; k < n; k++ {
		row, ok := f.townList.RowAt(image.Pt(pickerLeft, pickerTop+k*pickerLine))
		if !ok || row != k {
			t.Errorf("row %d drawn at y=%d hit-tests to (%d, %v)",
				k, pickerTop+k*pickerLine, row, ok)
		}
	}
	if _, ok := f.townList.RowAt(image.Pt(pickerLeft, pickerTop-1)); ok {
		t.Error("a point above the list hit a row")
	}
}

func TestTownDialogueIsModalAndItsOKButtonAdvances(t *testing.T) {
	town := &fakeTownDialogue{pic: image.NewRGBA(image.Rect(0, 0, 240, 120))}
	a := newTestApp(t, appRows(3), okLoader(t))
	a.SetTown(town)
	if !a.flow.showTown("") {
		t.Fatal("showTown refused the dialogue town")
	}

	windowAt := func(p image.Point) appInput {
		ox, oy := a.place.Origin()
		s := a.place.Scale()
		return appInput{CursorX: int(float64(p.X)*s + ox), CursorY: int(float64(p.Y)*s + oy)}
	}
	// A click on a legacy list row behind the dialogue is consumed.
	covered := windowAt(image.Pt(pickerLeft, pickerTop))
	covered.PrimaryPressed = true
	a.stepTown(covered)
	covered.PrimaryPressed, covered.PrimaryReleased = false, true
	a.stepTown(covered)
	if len(town.chosen) != 0 || town.advances != 0 {
		t.Fatalf("click behind dialogue chose %v and advanced %d", town.chosen, town.advances)
	}

	button, ok := townDialogueButton(town)
	if !ok {
		t.Fatal("dialogue exposed no OK button")
	}
	click := windowAt(image.Pt((button.Min.X+button.Max.X)/2, (button.Min.Y+button.Max.Y)/2))
	click.PrimaryPressed = true
	a.stepTown(click)
	if town.advances != 0 {
		t.Fatalf("OK press advanced %d pages", town.advances)
	}
	click.PrimaryPressed, click.PrimaryReleased = false, true
	a.stepTown(click)
	if town.advances != 1 || len(town.chosen) != 0 {
		t.Fatalf("OK advances=%d chosen=%v, want 1 and no legacy choice", town.advances, town.chosen)
	}
}

func TestTownSquareUsesFourSeparateClickableRegions(t *testing.T) {
	for i := 0; i < 4; i++ {
		x, y, w, h := townSquareRect(i)
		row, ok := townSquareRowAt(x+w/2, y+h/2)
		if !ok || row != i {
			t.Errorf("region %d centre hit (%d, %v)", i, row, ok)
		}
	}
	if _, ok := townSquareRowAt(townSquareLeft+townSquareW+townSquareGap/2, townSquareTop); ok {
		t.Error("the gap between town regions is clickable")
	}
}

// The application's own arms: SetTown installs and does not enter, the keys
// that move the map list move this list, Escape unwinds a room, and Draw
// paints without a window.
//
// IT DRIVES App.step AND App.Draw and not the flow, because the dispatch arms
// are what a player's press actually reaches — a screen wired into the flow and
// not into the switch would pass every test above and show nothing.
func TestTheApplicationDrivesTheTown(t *testing.T) {
	town := &fakeTown{}
	a := newTestApp(t, appRows(3), okLoader(t))

	a.SetTown(town)
	if a.Screen() != ScreenMenu {
		t.Fatalf("SetTown moved the screen to %v — installing is not entering", a.Screen())
	}

	if !a.flow.showTown("") {
		t.Fatal("showTown refused an installed town")
	}
	now := time.Unix(1_700_000_000, 0)

	// Down moves the selection, exactly as it does on the map list.
	a.step(appInput{Down: true}, now)
	if got := a.flow.townList.Selection(); got != 1 {
		t.Errorf("Down left the selection at %d, want 1", got)
	}

	// Enter walks through the chosen door.
	a.step(appInput{Enter: true}, now)
	if len(town.chosen) != 1 || town.chosen[0] != 1 {
		t.Errorf("the far side was asked about %v, want [1]", town.chosen)
	}

	// Escape unwinds the room and then the town.
	if exit := a.step(appInput{Escape: true}, now); exit {
		t.Fatal("Escape in a room asked to exit")
	}
	if a.Screen() != ScreenTown {
		t.Errorf("screen %v after leaving a room, want ScreenTown", a.Screen())
	}
	a.step(appInput{Escape: true}, now)
	leaveViaMenu(a.flow) //
	if a.Screen() != ScreenMenu {
		t.Errorf("screen %v after leaving the town, want ScreenMenu", a.Screen())
	}

	// And the town paints. A canvas the size of the virtual frame is all it
	// needs; nothing here opens a window.
	a.flow.showTown("something went wrong")
	a.canvas = ebiten.NewImage(frame.W, frame.H)
	a.drawTown()
}

// A windowed picker draws and hit-tests the same shorter set, so a row the
// player cannot see is a row he cannot click.
func TestAWindowedPickerDrawsAndHitTestsTheSameShorterSet(t *testing.T) {
	rows := make([]PickerRow, 30)
	for i := range rows {
		rows[i] = PickerRow{Text: "row", Choosable: true}
	}
	p := NewPicker(rows).SetWindow(6)

	if _, n := p.Visible(); n != 6 {
		t.Fatalf("Visible reported %d rows, want the 6 the window allows", n)
	}
	if _, ok := p.RowAt(image.Pt(0, pickerTop+6*pickerLine)); ok {
		t.Error("a click below the window hit a row")
	}
	if _, ok := p.RowAt(image.Pt(0, pickerTop+5*pickerLine)); !ok {
		t.Error("a click on the last visible row hit nothing")
	}

	// Moving past the window scrolls it rather than drawing past the bottom.
	p.Move(10)
	top, n := p.Visible()
	if n != 6 || top != 5 {
		t.Errorf("after moving to row 10 the window is (%d,%d), want (5,6)", top, n)
	}

	// An out-of-range window restores the full one rather than blanking the list.
	if _, n := NewPicker(rows).SetWindow(0).Visible(); n != pickerVisible {
		t.Errorf("SetWindow(0) left %d rows visible, want the full window", n)
	}
	if _, n := NewPicker(rows).SetWindow(999).Visible(); n != pickerVisible {
		t.Errorf("SetWindow(999) left %d rows visible, want the full window", n)
	}
}

func TestTheWheelCrossesTheSeamWithTheRegionUnderThePointer(t *testing.T) {
	town := &fakeShopTown{}
	f := townFlow(t, town)

	if !f.scrollShop(image.Pt(80, 150), -1) {
		t.Fatal("the wheel over the shelf grid did not cross the seam")
	}
	if !f.scrollShop(image.Pt(200, 430), +1) {
		t.Fatal("the wheel over the backpack strip did not cross the seam")
	}
	if f.scrollShop(image.Pt(200, 340), +1) {
		t.Error("the wheel over the table crossed the seam")
	}
	if f.scrollShop(image.Pt(560, 300), +1) {
		t.Error("the wheel over the character panel crossed the seam")
	}
	want := []scrolledBy{{ShopWheelShelf, -1}, {ShopWheelPack, +1}}
	if !reflect.DeepEqual(town.scrolled, want) {
		t.Errorf("the seam received %v, want %v", town.scrolled, want)
	}
}

func TestTownStatusEchoIsNotShownAndErrorIs(t *testing.T) {
	var f flow
	f.applyTownAction(TownInfo("bought for 5 gold; you have 7 left"))
	if f.msg != "" {
		t.Errorf("a success echo reached the player's view as %q", f.msg)
	}
	f.applyTownAction(TownAction{Msg: "you cannot afford that"})
	if f.msg != "you cannot afford that" {
		t.Errorf("an error line was dropped: %q", f.msg)
	}
}

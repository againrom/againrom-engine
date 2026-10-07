package game

import (
	"fmt"
	"strings"

	"againrom/pkg/data"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

// This file is scenario version 4 (1005 round 2, seventh pass): the pointer
// step, and the two observables a pointer gesture over an inventory is judged
// by — what the DOLL is showing on the map screen, and what the SHOP room is
// showing in the town.
//
// WHY A SCENARIO NEEDED THIS AT ALL. Every step vocabulary before it reached a
// control by NAME — a row, a button, a member id — and the inventory has no
// names: its surfaces are pixels of a composed figure, and the gesture under
// test is a press, a move and a release over them. A story about dragging an
// item could therefore be unit-tested and never driven, which is what
// `closure.md`'s integration witness is for.
//
// THE POINT IS ALWAYS ASKED OF THE PRODUCTION HIT TEST (pkg/ui's own
// HeadlessDollSlotPoint and friends), never computed here. A scenario names a
// surface and the running build answers where it stands this frame, so a file
// keeps meaning the same thing when a layout constant moves.

// HeadlessPoint is where one pointer edge lands. EXACTLY ONE FORM per point:
// the forms address disjoint surfaces and a point naming two of them would have
// to pick one, silently.
type HeadlessPoint struct {
	// Spell names the real ID of a visible book cell, including unavailable
	// catalog cells. The pointer still reaches App; it does not select/bind.
	Spell *uint32 `json:"spell,omitempty"`
	// DollSlot is an equipment slot of the map screen's own doll figure,
	// 1..12, resolved to a pixel that figure's own slot mask answers for.
	DollSlot *int `json:"doll_slot,omitempty"`

	// DollBox is the doll box's own middle: inside the box, on no particular
	// slot. It is the destination a pack-origin drag is released on.
	DollBox bool `json:"doll_box,omitempty"`

	// PackCell is a cell of the map screen's own pack bar, the container
	// ELEMENT index it names (pkg/ui's own inventoryPackCellAt folds the
	// scroll in), which is the index enqueueEquip resolves against.
	PackCell *int `json:"pack_cell,omitempty"`

	// PackCode is that same cell named by what it HOLDS rather than by where
	// it sits. A container's order is a fact about the run — an unequip
	// appends, a sale removes — so a file naming a position asserts a layout
	// it did not choose, and the cell it meant moves under it.
	PackCode *uint16 `json:"pack_code,omitempty"`

	// Ground is a pixel of the running map that no inventory box, HUD toggle
	// bar or spellbook strip claims — where a release becomes a ground drop.
	Ground bool `json:"ground,omitempty"`

	// Shop names one of the shop room's own four drag surfaces.
	Shop *HeadlessShopPointRef `json:"shop,omitempty"`

	// ShopIdle is a pixel of the shop frame naming no control at all.
	ShopIdle bool `json:"shop_idle,omitempty"`

	// WorldMapMission is a scroll card of the world map screen, named by the
	// mission's own campaign number rather than by its display slot (which
	// shifts under CardBase's own paging) — the surface WorldMapClick reads
	// to select a mission and start travel to it (`TOWN-118`).
	WorldMapMission *int `json:"world_map_mission,omitempty"`

	// WorldMapTown is the world map's own separate return-to-town card
	// (display slot -1), WorldMapCardAt's other arm.
	WorldMapTown bool `json:"world_map_town,omitempty"`

	// WorldMapMiss is a pixel of the world map screen answering for neither a
	// card nor a mission region — the click WorldMapClick's own skip arm
	// reads while travel is in progress, fast-forwarding the route reveal to
	// its own end (`TOWN-121`).
	WorldMapMiss bool `json:"world_map_miss,omitempty"`

	// Document names one of the campaign documents panel's own three controls:
	// `left`, `right` or `ok`.
	Document string `json:"document,omitempty"`

	// OffsetX and OffsetY displace the resolved pixel (round-2 adversarial
	// review, tenth pass, counterexample 2): every form above answers ONE pixel
	// — a slot's own middle, a cell's own centre — and no prior version of
	// this file could express a press that lands anywhere else, so no scenario
	// could drive a hand's own tremor: a path that moves within one slot or
	// cell rather than standing still or crossing into another one. The tremor
	// is exactly the gesture counterexample 1 found dead in production (the
	// doll's own press-to-take-off, lost once a drag armed and released back on
	// its own origin), and it is not expressible without a way to move the
	// cursor a few pixels while the hit test still names the SAME surface.
	// Offsetting the already-resolved pixel, rather than inventing a second
	// geometry, keeps this file's own rule — the point comes from the
	// production hit test, never computed here — the offset is applied only
	// after that hit test has already answered.
	OffsetX int `json:"offset_x,omitempty"`
	OffsetY int `json:"offset_y,omitempty"`
}

// HeadlessShopPointRef is one cell of one shop surface: `doll` (slot 1..12),
// `shelf`, `pack` or `table` (each 0-based). `doll_box` names the doll's own
// drawn area with no particular slot, `HeadlessPoint`'s own `doll_box` (the
// mission's doll) restated for the shop — the destination a pack- or
// shelf-origin drag is released on when the shown member wears nothing at
// the release point (round-2 adversarial review, tenth pass).
type HeadlessShopPointRef struct {
	Surface string `json:"surface"`
	Index   int    `json:"index,omitempty"`

	// Code names a `pack`, `table` or `shelf` cell by what it holds.
	Code *uint16 `json:"code,omitempty"`
}

func (p *HeadlessPoint) validate() error {
	if p == nil {
		return fmt.Errorf("pointer requires at")
	}
	forms := 0
	if p.Spell != nil {
		forms++
		if *p.Spell == 0 || *p.Spell > 65535 {
			return fmt.Errorf("at.spell %d: expected a nonzero uint16 spell ID", *p.Spell)
		}
	}
	if p.DollSlot != nil {
		forms++
		if *p.DollSlot < 1 || *p.DollSlot > sim.EquipSlots {
			return fmt.Errorf("at.doll_slot %d: the doll has slots 1 to %d", *p.DollSlot, sim.EquipSlots)
		}
	}
	if p.PackCell != nil {
		forms++
		if *p.PackCell < 0 {
			return fmt.Errorf("at.pack_cell %d: cells are counted from zero", *p.PackCell)
		}
	}
	if p.PackCode != nil {
		forms++
	}
	if p.DollBox {
		forms++
	}
	if p.Ground {
		forms++
	}
	if p.ShopIdle {
		forms++
	}
	if p.WorldMapMission != nil {
		forms++
	}
	if p.WorldMapTown {
		forms++
	}
	if p.WorldMapMiss {
		forms++
	}
	if p.Document != "" {
		forms++
		switch strings.ToLower(strings.TrimSpace(p.Document)) {
		case "left", "right", "ok":
		default:
			return fmt.Errorf("at.document %q: left, right and ok are the three", p.Document)
		}
	}
	if p.Shop != nil {
		forms++
		surface := strings.ToLower(strings.TrimSpace(p.Shop.Surface))
		switch surface {
		case "doll", "doll_box", "shelf", "pack", "table", "picker_prev", "picker_next", "shelf_pick", "button", "merchant":
		default:
			return fmt.Errorf("at.shop.surface %q: doll, doll_box, shelf, pack, table, picker_prev "+
				"picker_next, shelf_pick, button and merchant are supported", p.Shop.Surface)
		}
		if surface == "doll_box" && p.Shop.Index != 0 {
			return fmt.Errorf("at.shop.surface doll_box names no particular slot; omit index")
		}
		if p.Shop.Code != nil && surface != "pack" && surface != "table" && surface != "shelf" {
			return fmt.Errorf("at.shop.code names a pack, table or shelf cell, not the %s", surface)
		}
	}
	if forms != 1 {
		return fmt.Errorf("at names %d forms; exactly one is required", forms)
	}
	return nil
}

// resolve asks the production hit tests where this point stands, in window
// pixels, then applies OffsetX/OffsetY. A form naming an item CODE is
// resolved against the live container or the live table first, into the
// index the hit test itself is asked for.
func (p *HeadlessPoint) resolve(front *FrontEnd, app *ui.App) (int, int, error) {
	x, y, err := p.resolveBase(front, app)
	if err != nil {
		return 0, 0, err
	}
	return x + p.OffsetX, y + p.OffsetY, nil
}

func (p *HeadlessPoint) resolveBase(front *FrontEnd, app *ui.App) (int, int, error) {
	switch {
	case p.Spell != nil:
		return app.HeadlessSpellPoint(*p.Spell)
	case p.DollSlot != nil:
		return app.HeadlessDollSlotPoint(*p.DollSlot)
	case p.DollBox:
		return app.HeadlessDollBoxPoint()
	case p.PackCell != nil:
		return app.HeadlessPackCellPoint(*p.PackCell)
	case p.PackCode != nil:
		idx, err := front.headlessPackElement(*p.PackCode)
		if err != nil {
			return 0, 0, err
		}
		return app.HeadlessPackCellPoint(idx)
	case p.Ground:
		return app.HeadlessGroundPoint()
	case p.ShopIdle:
		return app.HeadlessShopIdlePoint()
	case p.WorldMapMission != nil:
		return app.HeadlessWorldMapMissionPoint(*p.WorldMapMission)
	case p.WorldMapTown:
		return app.HeadlessWorldMapTownPoint()
	case p.WorldMapMiss:
		return app.HeadlessWorldMapMissPoint()
	case p.Document != "":
		return app.HeadlessDocumentPoint(p.Document)
	case p.Shop != nil:
		surface := strings.ToLower(strings.TrimSpace(p.Shop.Surface))
		index := p.Shop.Index
		if p.Shop.Code != nil {
			got, err := front.headlessShopCell(surface, *p.Shop.Code)
			if err != nil {
				return 0, 0, err
			}
			index = got
		}
		return app.HeadlessShopPoint(surface, index)
	}
	return 0, 0, fmt.Errorf("pointer: at names no surface")
}

// headlessPackElement is which element of the map subject's own container holds
// code — enqueueEquip's own CarriedStacks ordering, read off the live world.
func (f *FrontEnd) headlessPackElement(code uint16) (int, error) {
	if f == nil || f.live == nil || f.live.world == nil || !f.live.invSubjectSet {
		return 0, fmt.Errorf("at.pack_code %d: no map screen with an inventory subject is open", code)
	}
	stacks, _ := f.live.world.CarriedStacks(sim.EntityID(f.live.invSubject.ID))
	for i, stack := range stacks {
		if stack.Code == code && stack.Count > 0 {
			return i, nil
		}
	}
	return 0, fmt.Errorf("at.pack_code %d: the subject carries none of it", code)
}

// headlessShopCell is which cell of the shop's pack strip or table holds code.
// The pack strip is numbered as ShopClick numbers it, with the money cell at 0
// and the scroll folded in, so the answer is the index a press produces.
func (f *FrontEnd) headlessShopCell(surface string, code uint16) (int, error) {
	if f == nil || f.townUI == nil || !f.townUI.AtTownShop() {
		return 0, fmt.Errorf("at.shop.code %d: no shop room is open", code)
	}
	t := f.townUI
	switch surface {
	case "shelf":
		if t.shopChosen >= 0 && t.shopChosen < len(shopRoomShelves) {
			for i, item := range t.sess.Shop.Shelf(shopRoomShelves[t.shopChosen].shelf) {
				if uint16(item.Code) == code && item.Count > 0 {
					return i - t.shelfBase, nil
				}
			}
		}
		return 0, fmt.Errorf("at.shop.code %d: the selected shelf holds none of it", code)
	case "pack":
		for i, stack := range t.shopPackStacks() {
			if stack.Code == code && stack.Count > 0 {
				return i + 1 - t.packBase, nil
			}
		}
		return 0, fmt.Errorf("at.shop.code %d: the shown member's pack holds none of it", code)
	case "table":
		for i, place := range t.sess.Shop.Table() {
			if uint16(place.Code) == code {
				return i, nil
			}
		}
		return 0, fmt.Errorf("at.shop.code %d: the table holds none of it", code)
	}
	return 0, fmt.Errorf("at.shop.code %d: the %s is not addressed by code", code, surface)
}

// headlessPointer dispatches one pointer edge at a resolved surface.
func headlessPointer(front *FrontEnd, app *ui.App, step HeadlessStep) error {
	action := strings.ToLower(strings.TrimSpace(step.Action))
	switch action {
	case "hover", "press", "move", "release", "right-press", "right-move", "right-release":
	default:
		return fmt.Errorf("pointer action %q: use hover, press, move, release, right-press, right-move or right-release", step.Action)
	}
	x, y, err := step.At.resolve(front, app)
	if err != nil {
		return err
	}
	return app.HeadlessPointer(action, x, y)
}

// HeadlessInventoryState is what the map screen's own inventory is showing:
// the subject it is bound to, whether slot 1 is being painted from the starting
// weapon rather than from the entity's own equipment array, the twelve codes the
// DOLL draws, the twelve the entity actually wears, the subject's container and
// every sack on the ground. These are deliberately code projections for pointer
// and layout scenarios; the item-instance release witness reads the complete
// state from sim.World instead.
//
// THE TWO EQUIPMENT ARRAYS ARE BOTH REPORTED and that is the point: they differ
// exactly while the display fallback is live, which is the state this story's
// own defect class lives in.
type HeadlessInventoryState struct {
	Subject        uint32                 `json:"subject"`
	WeaponFallback bool                   `json:"weapon_fallback"`
	Figure         [sim.EquipSlots]uint16 `json:"figure"`
	Equipment      [sim.EquipSlots]uint16 `json:"equipment"`
	Carried        []uint16               `json:"carried"`
	Sacks          []HeadlessSack         `json:"sacks,omitempty"`
}

// HeadlessSack is one sack lying on the ground.
type HeadlessSack struct {
	X     int32    `json:"x"`
	Y     int32    `json:"y"`
	Gold  uint32   `json:"gold,omitempty"`
	Items []uint16 `json:"items,omitempty"`
}

// HeadlessShopState is what the open shop room is showing: which member the
// picker is on, his own twelve doll codes (the fallback folded in exactly as
// the drawn doll folds it), his container, the table, and the purse. The
// inventory fields are code projections; table rows retain price but omit the
// instance effect list because this surface witnesses shop gestures, not item
// identity.
type HeadlessShopState struct {
	Member  int                    `json:"member"`
	Gold    int                    `json:"gold"`
	Doll    [sim.EquipSlots]uint16 `json:"doll"`
	Worn    [sim.EquipSlots]uint16 `json:"worn"`
	Carried []uint16               `json:"carried"`
	Table   []HeadlessTablePlace   `json:"table,omitempty"`
}

// HeadlessTablePlace is one of the five table places.
type HeadlessTablePlace struct {
	Code  uint16 `json:"code"`
	Price int32  `json:"price"`
	Count int32  `json:"count"`
	Mine  bool   `json:"mine"`
}

// headlessInventory reads the live map screen's inventory. It reads and
// writes nothing.
//
// FIGURE IS mw.invComposedEquipment, NOT currentFigureEquipment() and NOT
// invFigureEquipment either (round-2 adversarial review, twelfth pass, C1b,
// corrected against the first attempt at this same fix): the field used to
// read invFigureEquipment on the theory that it is "what was drawn," which
// is itself false — invFigureEquipment is refreshEquipment's own guard
// SEED, re-derived from currentFigureEquipment (a LIVE read off the running
// sim.World) at every mission open and member switch, and updated again only
// when that guard's live-vs-live compare finds a difference. It therefore
// agrees with the live world by construction and can never disagree with it,
// which means it can never witness a composition (buildInventorySubject,
// inventory.go) that drew the figure from something OTHER than the live
// world — C1's own defect, where the doll was composed from member.Worn
// while the live world (and invFigureEquipment) already reflected
// member.Carry.Equipped. Both fields would report "correct" under that bug,
// forever, because neither one asks what was actually painted.
//
// invComposedEquipment IS what was actually painted: it is seeded, at
// mission open and at every member switch, from the SAME call
// buildInventorySubject itself makes (missionDollEquipment, inventory.go)
// rather than from a fresh live read, and moved forward only when
// refreshEquipment genuinely recomposes the figure from a live change (see
// that field's own doc, world.go). A caller wanting the raw live array
// instead — the entity's real worn set, fallback or no — still has
// Equipment, two lines below, which is unchanged.
// HeadlessInventory is what the open map screen's own inventory is showing, for
// a developer tool that has no window to read one off (cmd/screenshot).
//
// IT IS A READ AND NOTHING ELSE, on LiveWorld's own terms (resume.go): nothing
// in the game calls it, and the scenario driver reaches the same value through
// headlessInventory directly. It exists because a tool outside this package
// cannot otherwise ask which pack cell holds a given item, and `go test ./...`
// is green with no install, so a tool pointed at a real one is the only place
// that question can be asked of shipped content.
func (f *FrontEnd) HeadlessInventory() *HeadlessInventoryState { return f.headlessInventory() }

func (f *FrontEnd) headlessInventory() *HeadlessInventoryState {
	if f == nil || f.live == nil || f.live.world == nil || !f.live.invSubjectSet {
		return nil
	}
	mw := f.live
	id := sim.EntityID(mw.invSubject.ID)
	state := &HeadlessInventoryState{
		Subject:        mw.invSubject.ID,
		WeaponFallback: mw.invSubject.WeaponFallback,
		Figure:         equipmentSlots(mw.invComposedEquipment),
		Equipment:      equipmentSlots(mw.currentEquipment()),
	}
	if items, ok := mw.world.Carried(id); ok {
		state.Carried = items
	}
	for _, s := range mw.world.Sacks() {
		state.Sacks = append(state.Sacks, HeadlessSack{X: s.X, Y: s.Y, Gold: s.Gold, Items: s.Items})
	}
	return state
}

// headlessShop reads the open shop room. A town with no room open answers nil.
func (f *FrontEnd) headlessShop() *HeadlessShopState {
	if f == nil || f.townUI == nil || !f.townUI.AtTownShop() {
		return nil
	}
	t := f.townUI
	i := t.shopMemberIndex()
	state := &HeadlessShopState{Member: i, Gold: t.sess.Town.Gold(), Carried: append([]uint16(nil), t.shopPackItems()...)}
	if worn := t.shopWornSlots(i); worn != nil {
		state.Worn = *worn
	}
	for n := 1; n <= sim.EquipSlots; n++ {
		if code, ok := t.shopEquippedCode(i, n); ok {
			state.Doll[n-1] = uint16(code)
		}
	}
	for _, place := range t.sess.Shop.Table() {
		state.Table = append(state.Table, HeadlessTablePlace{
			Code: uint16(place.Code), Price: place.Price, Count: place.Count, Mine: place.Mine})
	}
	return state
}

// HeadlessSlotClause asserts one equipment slot: a code, or empty.
type HeadlessSlotClause struct {
	Slot  int     `json:"slot"`
	Code  *uint16 `json:"code,omitempty"`
	Empty bool    `json:"empty,omitempty"`
}

// HeadlessCarryClause asserts one item code in a container: present, present
// exactly count times, or absent.
type HeadlessCarryClause struct {
	Code   uint16 `json:"code"`
	Count  *int   `json:"count,omitempty"`
	Absent bool   `json:"absent,omitempty"`
}

// HeadlessInventoryAssertion is the map screen's own inventory, asserted.
type HeadlessInventoryAssertion struct {
	Subject        *uint32               `json:"subject,omitempty"`
	WeaponFallback *bool                 `json:"weapon_fallback,omitempty"`
	Figure         []HeadlessSlotClause  `json:"figure,omitempty"`
	Equipment      []HeadlessSlotClause  `json:"equipment,omitempty"`
	Carries        []HeadlessCarryClause `json:"carries,omitempty"`
	Sacks          []HeadlessCarryClause `json:"sacks,omitempty"`
}

// HeadlessShopAssertion is the open shop room, asserted.
type HeadlessShopAssertion struct {
	Member  *int                  `json:"member,omitempty"`
	Gold    *int                  `json:"gold,omitempty"`
	Doll    []HeadlessSlotClause  `json:"doll,omitempty"`
	Worn    []HeadlessSlotClause  `json:"worn,omitempty"`
	Carries []HeadlessCarryClause `json:"carries,omitempty"`
	Table   []HeadlessCarryClause `json:"table,omitempty"`
}

func (a *HeadlessInventoryAssertion) validate() error {
	if a == nil {
		return fmt.Errorf("assert_inventory requires inventory")
	}
	if a.Subject == nil && a.WeaponFallback == nil && len(a.Figure) == 0 &&
		len(a.Equipment) == 0 && len(a.Carries) == 0 && len(a.Sacks) == 0 {
		return fmt.Errorf("assert_inventory states nothing")
	}
	for _, c := range append(append([]HeadlessSlotClause(nil), a.Figure...), a.Equipment...) {
		if err := c.validate(); err != nil {
			return err
		}
	}
	return nil
}

func (a *HeadlessShopAssertion) validate() error {
	if a == nil {
		return fmt.Errorf("assert_shop requires shop")
	}
	if a.Member == nil && a.Gold == nil && len(a.Doll) == 0 && len(a.Worn) == 0 &&
		len(a.Carries) == 0 && len(a.Table) == 0 {
		return fmt.Errorf("assert_shop states nothing")
	}
	for _, c := range append(append([]HeadlessSlotClause(nil), a.Doll...), a.Worn...) {
		if err := c.validate(); err != nil {
			return err
		}
	}
	return nil
}

func (c HeadlessSlotClause) validate() error {
	if c.Slot < 1 || c.Slot > sim.EquipSlots {
		return fmt.Errorf("slot %d: equipment has slots 1 to %d", c.Slot, sim.EquipSlots)
	}
	if (c.Code == nil) == !c.Empty {
		return fmt.Errorf("slot %d states both a code and empty, or neither", c.Slot)
	}
	return nil
}

func (c HeadlessSlotClause) holds(slots [sim.EquipSlots]uint16, what string) error {
	got := slots[c.Slot-1]
	if c.Empty {
		if got != 0 {
			return fmt.Errorf("%s slot %d = %d, want empty", what, c.Slot, got)
		}
		return nil
	}
	if got != *c.Code {
		return fmt.Errorf("%s slot %d = %d, want %d", what, c.Slot, got, *c.Code)
	}
	return nil
}

func (c HeadlessCarryClause) holds(codes []uint16, what string) error {
	n := 0
	for _, code := range codes {
		if code == c.Code {
			n++
		}
	}
	switch {
	case c.Absent:
		if n != 0 {
			return fmt.Errorf("%s holds %d of code %d, want none", what, n, c.Code)
		}
	case c.Count != nil:
		if n != *c.Count {
			return fmt.Errorf("%s holds %d of code %d, want %d", what, n, c.Code, *c.Count)
		}
	default:
		if n == 0 {
			return fmt.Errorf("%s holds no code %d", what, c.Code)
		}
	}
	return nil
}

func assertHeadlessInventory(got *HeadlessInventoryState, want HeadlessInventoryAssertion) error {
	if got == nil {
		return fmt.Errorf("assert_inventory: no map screen with an inventory subject is open")
	}
	if want.Subject != nil && got.Subject != *want.Subject {
		return fmt.Errorf("inventory subject = %d, want %d", got.Subject, *want.Subject)
	}
	if want.WeaponFallback != nil && got.WeaponFallback != *want.WeaponFallback {
		return fmt.Errorf("inventory weapon_fallback = %v, want %v (figure %v, equipment %v)",
			got.WeaponFallback, *want.WeaponFallback, got.Figure, got.Equipment)
	}
	for _, c := range want.Figure {
		if err := c.holds(got.Figure, "figure"); err != nil {
			return err
		}
	}
	for _, c := range want.Equipment {
		if err := c.holds(got.Equipment, "equipment"); err != nil {
			return err
		}
	}
	for _, c := range want.Carries {
		if err := c.holds(got.Carried, "the subject's container"); err != nil {
			return err
		}
	}
	var loose []uint16
	for _, s := range got.Sacks {
		loose = append(loose, s.Items...)
	}
	for _, c := range want.Sacks {
		if err := c.holds(loose, "the ground"); err != nil {
			return err
		}
	}
	return nil
}

func assertHeadlessShop(got *HeadlessShopState, want HeadlessShopAssertion) error {
	if got == nil {
		return fmt.Errorf("assert_shop: no shop room is open")
	}
	if want.Member != nil && got.Member != *want.Member {
		return fmt.Errorf("shop member = %d, want %d", got.Member, *want.Member)
	}
	if want.Gold != nil && got.Gold != *want.Gold {
		return fmt.Errorf("shop gold = %d, want %d", got.Gold, *want.Gold)
	}
	for _, c := range want.Doll {
		if err := c.holds(got.Doll, "the shop doll"); err != nil {
			return err
		}
	}
	for _, c := range want.Worn {
		if err := c.holds(got.Worn, "the shown member's worn array"); err != nil {
			return err
		}
	}
	for _, c := range want.Carries {
		if err := c.holds(got.Carried, "the shown member's pack"); err != nil {
			return err
		}
	}
	var staged []uint16
	for _, place := range got.Table {
		for n := int32(0); n < place.Count; n++ {
			staged = append(staged, place.Code)
		}
	}
	for _, c := range want.Table {
		if err := c.holds(staged, "the table"); err != nil {
			return err
		}
	}
	return nil
}

// equipmentSlots is data.Equipment read back as the raw twelve codes, the
// inverse of hero.go's own equipmentFromSlots. It exists for the scenario
// event, which carries numbers rather than a package type.
func equipmentSlots(eq data.Equipment) [sim.EquipSlots]uint16 {
	var out [sim.EquipSlots]uint16
	for slot := 1; slot <= sim.EquipSlots; slot++ {
		code, _ := eq.Code(slot)
		out[slot-1] = uint16(code)
	}
	return out
}

package game

import (
	"bytes"
	"slices"
	"testing"

	"againrom/internal/firstgame"
	"againrom/pkg/data"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

// qolWitness is one run of the quality-of-life key witness: mission 10 with
// the default mage hero, who wears the installed Fire Arrow staff and carries
// a plain staff of the same item code.
type qolWitness struct {
	f       *FrontEnd
	app     *ui.App
	id      sim.EntityID
	spell   uint32
	charged sim.ItemInstance
	plain   sim.ItemInstance
	// start is the hero before any order; zoom and period are the view's
	// opening zoom and cadence period.
	start  sim.Entity
	zoom   float64
	period int
}

func qolWitnessOpen(t *testing.T) *qolWitness {
	t.Helper()
	f := releaseFront(t)
	f.SetDeterministicFrames(true)
	f.SetTipsOff(true)
	party := MissionPartyAs(true, f.StartWeapon.Value(), f.Bodies, f.Table)
	member := &party[0]
	worn := mapload.MemberItemEquipment(*member, f.Table)
	carried := mapload.MemberCarriedItems(*member, f.Table)
	charged := worn[0]
	plain := mapload.ItemInstanceFromCode(charged.Code, f.Table)
	if charged.Empty() || !charged.HasEnchantment() || plain.Empty() || sim.ItemEqual(plain, charged) {
		t.Fatalf("the default mage's staff %+v and its plain form %+v are not two weapons", charged, plain)
	}
	if slot, ok := EquipTarget(data.ItemCode(charged.Code), f.Table); !ok || slot != 1 {
		t.Fatalf("the installed rows put the staff %#x in slot %d (%v), want 1", charged.Code, slot, ok)
	}
	carried = append(carried, plain)
	syncMissionPartyStock(member, worn, carried)
	app := f.App("quality of life keys")
	app.SetCutscenes(nil)
	app.Layout(1024, 768)
	if err := app.OpenMission(f.MissionOpenerWith(10, party)); err != nil {
		t.Fatal(err)
	}
	openMissionPack(t, app)
	w := &qolWitness{f: f, app: app}
	w.id = shieldWitnessSelect(t, f, app)
	// The items as the World holds them: the loader fills in weight and
	// definition operands the constructor inputs lack.
	held, _ := f.live.world.EquippedItems(w.id)
	stacks, _ := f.live.world.CarriedStacks(w.id)
	w.charged = held[0]
	for _, st := range stacks {
		if st.Code == plain.Code && !st.Instance().HasEnchantment() {
			w.plain = st.Instance()
		}
	}
	if w.charged.Code != charged.Code || w.plain.Empty() {
		t.Fatalf("the mission holds staff %+v and plain staff %+v", w.charged, w.plain)
	}
	w.start, _ = f.live.entity(w.id)
	if w.start.KnownSpells&(1<<mageFireArrow) == 0 {
		t.Fatalf("the default mage does not know Fire Arrow: book %#x", w.start.KnownSpells)
	}
	w.spell = mageFireArrow
	if _, _, err := app.HeadlessSpellPoint(w.spell); err != nil {
		w.key(t, "book")
	}
	view := f.live.view.SaveApplication()
	w.zoom, w.period = view.Zoom, view.PeriodUS
	return w
}

func (w *qolWitness) key(t *testing.T, name string) {
	t.Helper()
	if err := w.app.HeadlessKey(name); err != nil {
		t.Fatal(err)
	}
}

func (w *qolWitness) pointer(t *testing.T, edges ...string) {
	t.Helper()
	x, y, err := w.app.HeadlessSpellPoint(w.spell)
	if err != nil {
		w.key(t, "book")
		if x, y, err = w.app.HeadlessSpellPoint(w.spell); err != nil {
			t.Fatal(err)
		}
	}
	for _, edge := range edges {
		if err := w.app.HeadlessPointer(edge, x, y); err != nil {
			t.Fatal(err)
		}
	}
}

func (w *qolWitness) steps(t *testing.T, n int) {
	t.Helper()
	for range n {
		if err := stepConsumableApp(w.app); err != nil {
			t.Fatal(err)
		}
	}
}

// pause sets the player pause on or off, so orders given between the two land
// on the same tick in every run whatever number of frames their input takes.
func (w *qolWitness) pause(t *testing.T, on bool) {
	t.Helper()
	if w.f.live.view.SaveApplication().PlayerPaused != on {
		w.key(t, "0")
	}
	if got := w.f.live.view.SaveApplication().PlayerPaused; got != on || w.f.live.stopped != on {
		t.Fatalf("pause %t: player paused %t, world stopped %t", on, got, w.f.live.stopped)
	}
}

// packCell double-clicks the pack element holding item.
func (w *qolWitness) packCell(t *testing.T, item sim.ItemInstance) {
	t.Helper()
	stacks, _ := w.f.live.world.CarriedStacks(w.id)
	cell := slices.IndexFunc(stacks, func(s sim.ItemStack) bool { return s.Count > 0 && sim.ItemEqual(s.Instance(), item) })
	if cell < 0 {
		t.Fatalf("the hero's pack holds no %+v: %+v", item, stacks)
	}
	usePackCell(t, w.app, cell)
}

func (w *qolWitness) holds(t *testing.T, label string, weapon, packed sim.ItemInstance) sim.Entity {
	t.Helper()
	worn, _ := w.f.live.world.EquippedItems(w.id)
	stacks, _ := w.f.live.world.CarriedStacks(w.id)
	inPack := slices.ContainsFunc(stacks, func(s sim.ItemStack) bool { return s.Count > 0 && sim.ItemEqual(s.Instance(), packed) })
	if !sim.ItemEqual(worn[0], weapon) || !inPack {
		t.Fatalf("%s: slot 1 holds %+v, pack %+v; want %+v worn and %+v packed", label, worn[0], stacks, weapon, packed)
	}
	e, _ := w.f.live.entity(w.id)
	return e
}

// chargedOff is the Fire Arrow staff taken off: in the first game it holds
// no Spell (ITEM-187), and worn again it builds one.
func (w *qolWitness) chargedOff() sim.ItemInstance {
	off := w.charged.Clone()
	if !tableFiles(w.f.Table).storedWeaponSpell {
		off.SourceEquipment.Spell = sim.SourceItemSpell{}
	}
	return off
}

// qolWitnessSave writes the mission through the game menu and returns to the
// map.
func qolWitnessSave(t *testing.T, w *qolWitness) (SaveStore, string, []byte) {
	t.Helper()
	store, name, raw := menuSAVE(t, w.f, w.app, OriginalStore{})
	if w.app.Screen() == ui.ScreenGameMenu {
		w.key(t, "escape")
	}
	if w.app.Screen() != ui.ScreenMap {
		t.Fatalf("after the SAVE the screen is %s, want the map", w.app.Screen())
	}
	return store, name, raw
}

// qolWitnessRun opens the mission, wears the plain staff from the pack and
// then takes the Fire Arrow staff back and toggles Fire Arrow's autocast:
// through W and Ctrl+A with the zoom and speed changed and reset by Home and
// End when keys is true, and through the pack and the spell strip otherwise.
func qolWitnessRun(t *testing.T, keys bool) *qolWitness {
	t.Helper()
	w := qolWitnessOpen(t)
	w.pause(t, true)
	w.packCell(t, w.plain)
	w.pause(t, false)
	w.steps(t, 4)
	off := w.chargedOff()
	if e := w.holds(t, "the plain staff worn", w.plain, off); e.WeaponSpell == w.start.WeaponSpell {
		t.Fatalf("the plain staff left the weapon spell %d of the Fire Arrow staff", e.WeaponSpell)
	}

	w.pause(t, true)
	if keys {
		w.key(t, "w")
		w.pointer(t, "press", "release")
		w.key(t, "autocast")
		w.pointer(t, "press", "release")
		for range 2 {
			if err := w.app.HeadlessPointer("wheel-up", 512, 300); err != nil {
				t.Fatal(err)
			}
			w.key(t, "numpad-plus")
		}
		view := w.f.live.view.SaveApplication()
		if view.Zoom == w.zoom || view.PeriodUS == w.period {
			t.Fatalf("control: the wheel and + left zoom %v and period %d at the opening %v and %d", view.Zoom, view.PeriodUS, w.zoom, w.period)
		}
		w.key(t, "zoom-reset")
		w.key(t, "speed-reset")
	} else {
		w.packCell(t, off)
		w.pointer(t, "right-press", "right-release")
	}
	w.pause(t, false)
	w.steps(t, 8)
	e := w.holds(t, "the Fire Arrow staff back", w.charged, w.plain)
	if e.WeaponSpell != w.start.WeaponSpell {
		t.Fatalf("the Fire Arrow staff back gives weapon spell %d, want %d", e.WeaponSpell, w.start.WeaponSpell)
	}
	wantAuto := wantAutocast(w)
	if uint32(e.AutoSpell) != wantAuto {
		t.Fatalf("autocast is %d after the toggle, want %d (it was %d)", e.AutoSpell, wantAuto, w.start.AutoSpell)
	}
	view := w.f.live.view.SaveApplication()
	if view.Zoom != w.zoom || view.PeriodUS != w.period || view.Unpaced || view.PressedSpell != 0 {
		t.Fatalf("zoom %v period %d unpaced %t spell %d, want the opening %v and %d, paced, none", view.Zoom, view.PeriodUS, view.Unpaced, view.PressedSpell, w.zoom, w.period)
	}
	return w
}

// wantAutocast is the spell the toggle leaves autocasting: Fire Arrow, or none
// when the hero autocast it already.
func wantAutocast(w *qolWitness) uint32 {
	if uint32(w.start.AutoSpell) == w.spell {
		return 0
	}
	return w.spell
}

// W, Ctrl+A, Home and End on an installed mission: W takes the Fire Arrow
// staff back from the pack, Ctrl+A toggles Fire Arrow's autocast, Home and End
// return the zoom and the game speed after the wheel and + moved them. The
// World and the SAV equal those of a run giving the same orders through the
// pack and the spell strip; a cold LOAD of that SAV equals the uninterrupted
// run for 64 ticks and has no previous weapon to take back. Loss controls: the
// SAV written before the second order differs, and W on the running mission,
// whose memory is kept, does take the plain staff back.
func TestReleaseQualityOfLifeKeysSaveOnlyOrdinaryOrders(t *testing.T) {
	firstgame.SkipOnSecondGame(t)
	keyed := qolWitnessRun(t, true)
	plain := qolWitnessRun(t, false)
	if keyed.f.live.world.Tick() != plain.f.live.world.Tick() {
		t.Fatalf("the runs stand at ticks %d and %d", keyed.f.live.world.Tick(), plain.f.live.world.Tick())
	}
	if !bytes.Equal(marshalWorld(t, keyed.f.live.world), marshalWorld(t, plain.f.live.world)) {
		t.Fatalf("the keys' World %#x differs from the ordinary orders' World %#x", keyed.f.live.world.Hash(), plain.f.live.world.Hash())
	}
	store, name, keyedSAV := qolWitnessSave(t, keyed)
	_, _, plainSAV := qolWitnessSave(t, plain)
	if !bytes.Equal(keyedSAV, plainSAV) {
		t.Fatalf("the keys' SAV (%d bytes) differs from the ordinary orders' SAV (%d bytes)", len(keyedSAV), len(plainSAV))
	}
	control := qolWitnessOpen(t)
	_, _, early := qolWitnessSave(t, control)
	if bytes.Equal(early, keyedSAV) {
		t.Fatal("control: the SAV written before the orders equals the one after them")
	}
	t.Logf("SAV %d bytes, World %#x at tick %d; autocast %d -> %d, weapon spell %d", len(keyedSAV), keyed.f.live.world.Hash(),
		keyed.f.live.world.Tick(), keyed.start.AutoSpell, wantAutocast(keyed), keyed.start.WeaponSpell)

	cold, coldApp := loadSAVWindow(t, store, name)
	if cold.live.world.Hash() != keyed.f.live.world.Hash() {
		t.Fatalf("LOAD gave World %#x, SAVE wrote %#x", cold.live.world.Hash(), keyed.f.live.world.Hash())
	}
	for n := range 64 {
		sim.Step(keyed.f.live.world, nil)
		sim.Step(cold.live.world, nil)
		if cold.live.world.Hash() != keyed.f.live.world.Hash() {
			t.Fatalf("tick %d after LOAD: World %#x, uninterrupted %#x", n+1, cold.live.world.Hash(), keyed.f.live.world.Hash())
		}
	}

	loaded := &qolWitness{f: cold, app: coldApp, charged: keyed.charged, plain: keyed.plain}
	loaded.id = shieldWitnessSelect(t, cold, coldApp)
	loaded.key(t, "w")
	loaded.steps(t, 4)
	loaded.holds(t, "W after LOAD", keyed.charged, keyed.plain)
	keyed.key(t, "w")
	keyed.steps(t, 4)
	keyed.holds(t, "control: W on the running mission", keyed.plain, keyed.chargedOff())
}

// A SAV written while W's order for a companion that is not the inventory
// subject is still queued reopens to the uninterrupted run: mission 10 with a
// generated companion that wears the short sword, then the two-handed sword
// from its pack; the primary hero is the subject, the whole selection takes
// W, and the SAVE comes before the order applies. After the cold LOAD both
// runs resume through the App for the same ticks and stay World for World
// equal, the companion's combat sheet included. The loss control steps the
// reopened World with the same order and no re-derivation.
func TestReleaseWeaponSwapQueuedBeforeASaveAppliesAfterTheLoad(t *testing.T) {
	firstgame.SkipOnSecondGame(t)
	f := releaseFront(t)
	f.SetDeterministicFrames(true)
	f.SetTipsOff(true)
	party := MissionParty(f.StartWeapon.Value(), f.Bodies, f.Table)
	companion := party[0]
	companion.ID, companion.StartingHero, companion.PlayerCharacter = "swap-companion", false, false
	worn := mapload.MemberItemEquipment(companion, f.Table)
	worn[0], worn[1] = mapload.ItemInstanceFromCode(shieldWitnessSword, f.Table), sim.ItemInstance{}
	syncMissionPartyStock(&companion, worn, []sim.ItemInstance{mapload.ItemInstanceFromCode(shieldWitnessGreatsword, f.Table)})
	companion.Weapon = nil
	party = append(party, companion)
	app := f.App("queued swap save")
	app.SetCutscenes(nil)
	app.Layout(1024, 768)
	if err := app.OpenMission(f.MissionOpenerWith(10, party)); err != nil {
		t.Fatal(err)
	}
	openMissionPack(t, app)
	primary := shieldWitnessSelect(t, f, app)
	second := f.live.mission.ids[1]
	q := &qolWitness{f: f, app: app, id: second}
	q.pause(t, true)
	if err := app.HeadlessSelectEntity(uint32(second)); err != nil {
		t.Fatal(err)
	}
	q.steps(t, 1)
	stacks, _ := f.live.world.CarriedStacks(second)
	cell := slices.IndexFunc(stacks, func(s sim.ItemStack) bool { return s.Code == shieldWitnessGreatsword })
	if cell < 0 {
		t.Fatal("the companion carries no two-handed sword")
	}
	usePackCell(t, app, cell)
	q.pause(t, false)
	q.steps(t, 4)
	q.pause(t, true)
	if err := app.HeadlessSelectEntity(uint32(primary)); err != nil {
		t.Fatal(err)
	}
	q.steps(t, 1)
	q.key(t, "select-all")
	q.key(t, "w")
	q.steps(t, 1)
	if f.live.invSubject.ID != uint32(primary) || len(f.live.pending) != 1 || f.live.pending[0].Entity != second {
		t.Fatalf("setup: subject %d, pending %+v; want the primary hero and one order of unit %d", f.live.invSubject.ID, f.live.pending, second)
	}
	store, name, raw := qolWitnessSave(t, q)
	cold, coldApp := loadSAVWindow(t, store, name)
	if !bytes.Equal(marshalWorld(t, f.live.world), marshalWorld(t, cold.live.world)) {
		t.Fatal("the cold LOAD World differs before the resumed tick")
	}
	if len(cold.live.pending) != 1 || cold.live.pending[0] != f.live.pending[0] {
		t.Fatalf("the cold LOAD holds orders %+v, want %+v", cold.live.pending, f.live.pending)
	}
	control := new(sim.World)
	if err := control.UnmarshalBinary(marshalWorld(t, cold.live.world)); err != nil {
		t.Fatal(err)
	}
	sim.Step(control, cold.live.pending)
	c := &qolWitness{f: cold, app: coldApp, id: second}
	q.pause(t, false)
	c.pause(t, false)
	for n := range 8 {
		q.steps(t, 1)
		c.steps(t, 1)
		if f.live.world.Tick() != cold.live.world.Tick() || f.live.world.Hash() != cold.live.world.Hash() {
			t.Fatalf("resumed step %d: cold World %#x at tick %d, uninterrupted %#x at tick %d", n+1,
				cold.live.world.Hash(), cold.live.world.Tick(), f.live.world.Hash(), f.live.world.Tick())
		}
		if n == 0 && control.Hash() == f.live.world.Hash() {
			t.Error("control: the order without the re-derivation gave the same World, so the comparison separates nothing")
		}
	}
	x, _ := f.live.world.Entity(second)
	y, _ := cold.live.world.Entity(second)
	z, _ := control.Entity(second)
	held := shieldWitnessRead(cold.live.world, second)
	if x.DamageBase != y.DamageBase || x.DamageSpread != y.DamageSpread || held.worn[0] != shieldWitnessSword {
		t.Errorf("the companion after the LOAD holds %#x with damage %d/%d; uninterrupted damage %d/%d",
			held.worn[0], y.DamageBase, y.DamageSpread, x.DamageBase, x.DamageSpread)
	}
	t.Logf("SAV %d bytes; World %#x at tick %d; companion damage %d/%d; control damage %d/%d", len(raw),
		cold.live.world.Hash(), cold.live.world.Tick(), y.DamageBase, y.DamageSpread, z.DamageBase, z.DamageSpread)
}

// W pressed after a queued drop of the remembered weapon equips nothing else:
// mission 10, the hero wears the short sword and carries the remembered
// two-handed sword ahead of a bow in the pack. While paused the inventory drops
// the two-handed sword and W is pressed. The swap waits for the drop, finds
// the weapon gone and queues nothing, so slot 1 keeps the short sword and the
// World equals the drop-only run (loss control). A third run presses W with
// no drop and takes the two-handed sword back.
func TestReleaseWeaponSwapAfterAQueuedDropEquipsNothingElse(t *testing.T) {
	type outcome struct {
		weapon uint16
		hash   uint64
		tick   uint64
	}
	run := func(t *testing.T, drop, press bool) outcome {
		t.Helper()
		f := releaseFront(t)
		f.SetDeterministicFrames(true)
		f.SetTipsOff(true)
		party := MissionParty(f.StartWeapon.Value(), f.Bodies, f.Table)
		worn := mapload.MemberItemEquipment(party[0], f.Table)
		worn[0], worn[1] = mapload.ItemInstanceFromCode(shieldWitnessGreatsword, f.Table), sim.ItemInstance{}
		syncMissionPartyStock(&party[0], worn, []sim.ItemInstance{
			mapload.ItemInstanceFromCode(shieldWitnessSword, f.Table), mapload.ItemInstanceFromCode(shieldWitnessBow, f.Table)})
		party[0].Weapon = nil
		app := f.App("drop then swap")
		app.SetCutscenes(nil)
		app.Layout(1024, 768)
		if err := app.OpenMission(f.MissionOpenerWith(10, party)); err != nil {
			t.Fatal(err)
		}
		openMissionPack(t, app)
		id := shieldWitnessSelect(t, f, app)
		q := &qolWitness{f: f, app: app, id: id}
		q.pause(t, true)
		// An ordinary drop and pickup put the bow after the two-handed sword.
		stacks, _ := f.live.world.CarriedStacks(id)
		bow := slices.IndexFunc(stacks, func(s sim.ItemStack) bool { return s.Code == shieldWitnessBow })
		if bow < 0 {
			t.Fatal("the hero carries no bow")
		}
		at, _ := f.live.world.Entity(id)
		f.live.enqueueDrop(false, bow, at.X, at.Y)
		q.pause(t, false)
		q.steps(t, 1)
		q.pause(t, true)
		stacks, _ = f.live.world.CarriedStacks(id)
		sword := slices.IndexFunc(stacks, func(s sim.ItemStack) bool { return s.Code == shieldWitnessSword })
		if sword < 0 {
			t.Fatal("the hero carries no short sword")
		}
		usePackCell(t, app, sword)
		q.pause(t, false)
		q.steps(t, 4)
		q.key(t, "grab")
		q.pause(t, true)
		prev := f.live.packIndex(id, f.live.weaponSwap.previous[id].weapon)
		packed, _ := f.live.world.CarriedStacks(id)
		if prev < 0 || prev >= len(packed)-1 || packed[prev+1].Code != shieldWitnessBow {
			t.Fatalf("setup: the remembered weapon is element %d of %+v, want one ahead of the bow", prev, packed)
		}
		if drop {
			e, _ := f.live.world.Entity(id)
			f.live.enqueueDrop(false, prev, e.X, e.Y)
		}
		if press {
			q.key(t, "w")
		}
		q.pause(t, false)
		q.steps(t, 3)
		h := shieldWitnessRead(f.live.world, id)
		return outcome{weapon: h.worn[0], hash: f.live.world.Hash(), tick: f.live.world.Tick()}
	}
	dropOnly := run(t, true, false)
	dropThenW := run(t, true, true)
	wOnly := run(t, false, true)
	t.Logf("drop only %#x World %#x; drop then W %#x World %#x; W only %#x; tick %d",
		dropOnly.weapon, dropOnly.hash, dropThenW.weapon, dropThenW.hash, wOnly.weapon, dropThenW.tick)
	if dropOnly.weapon != shieldWitnessSword {
		t.Fatalf("control: the drop alone left slot 1 holding %#x, want the short sword", dropOnly.weapon)
	}
	if dropThenW.weapon != shieldWitnessSword || dropThenW.hash != dropOnly.hash || dropThenW.tick != dropOnly.tick {
		t.Errorf("drop then W: slot 1 %#x World %#x, want the drop-only run's short sword and World %#x", dropThenW.weapon, dropThenW.hash, dropOnly.hash)
	}
	if wOnly.weapon != shieldWitnessGreatsword {
		t.Errorf("control: W with no drop left slot 1 holding %#x, want the two-handed sword", wOnly.weapon)
	}
}

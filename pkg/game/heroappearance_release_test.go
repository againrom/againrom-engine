package game

import (
	"slices"
	"testing"

	"againrom/internal/firstgame"
	"againrom/pkg/data"
	"againrom/pkg/formats/sav"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

// heroClassInSAV is the class the SAV a FrontEnd writes now carries for id.
func heroClassInSAV(t *testing.T, f *FrontEnd, id sim.EntityID) int32 {
	t.Helper()
	_, raw := writeOrdinarySAV(t, f, "hero-class.sav")
	doc, err := sav.DecodeDocumentData(raw)
	if err != nil {
		t.Fatal(err)
	}
	actions, err := readCurrentActions(&doc)
	if err != nil || actions == nil {
		t.Fatalf("SAV current actions: %v", err)
	}
	for _, a := range actions.Actions.Actors {
		if a.Entity == id && a.Current != nil && a.Current.Class != nil {
			return *a.Current.Class
		}
	}
	t.Fatalf("SAV carries no class for actor %d", id)
	return 0
}

// sourceArmClass is the class the retired flight class rule derived for a hero:
// the raw slots, and the mage arm read from the actor's source or native class.
func sourceArmClass(e sim.Entity, slots [sim.EquipSlots]uint16, list data.BodyList) (int32, bool) {
	source := e.SourceNow()
	fighter := source.Fighter
	if source.Class == 0 {
		if !e.NativeClass.Present {
			return e.Class, false
		}
		fighter = e.NativeClass.Fighter
	}
	_, _, class, _ := data.HeroAppearance(list, equipmentFromSlots(slots), !fighter, false)
	return class, !fighter
}

// The bow and the short sword the witnesses below hand a chargen fighter.
const witnessBow, witnessSword = 0x0114, 0x0103

// bowAndSwordHero opens mission 10 for a chargen fighter who wears no weapon
// and carries a bow and a short sword.
func bowAndSwordHero(t *testing.T) (*FrontEnd, *mapWorld, sim.EntityID) {
	t.Helper()
	f := releaseFront(t)
	f.SetDeterministicFrames(true)
	party := f.ChargenParty(ui.ChargenResult{Name: "Class witness", Choices: []int{0, 0, 0}, Stats: []int{31, 27, 24, 29}})
	hero := &party[0]
	hero.Worn[0], hero.WornItems[0], hero.Weapon = 0, sim.ItemInstance{}, nil
	for _, code := range []uint16{witnessBow, witnessSword} {
		hero.Carried = append(slices.Clone(hero.Carried), code)
		hero.CarriedItems = append(slices.Clone(hero.CarriedItems), mapload.ItemInstanceFromCode(code, f.Table))
	}
	a := f.App("hero class witness")
	t.Cleanup(a.StopAudio)
	a.SetCutscenes(nil)
	if err := a.OpenMission(f.MissionOpenerWith(10, party)); err != nil {
		t.Fatal(err)
	}
	id := equipmentReturnHero(t, f)
	f.live.switchInventorySubject(uint32(id))
	return f, f.live, id
}

// queueWeapon queues, through the inventory, the order that wears the carried
// code in slot 1; the next tickStep applies it.
func queueWeapon(t *testing.T, mw *mapWorld, id sim.EntityID, code uint16) {
	t.Helper()
	if !mw.invSubjectSet || mw.invSubject.ID != uint32(id) {
		t.Fatalf("the inventory subject is not hero %d", id)
	}
	carried, _ := mw.world.CarriedItems(id)
	index := slices.IndexFunc(carried, func(item sim.ItemInstance) bool { return item.Code == code })
	if index < 0 {
		t.Fatalf("hero carries no %#x", code)
	}
	queued := len(mw.pending)
	mw.enqueueEquip(index)
	if len(mw.pending) != queued+1 || mw.pending[queued].Kind != sim.KindEquip {
		t.Fatalf("the inventory queued no equip of %#x: %+v", code, mw.pending)
	}
}

// A hero's worn weapon decides his drawn body, the class the World stores, the
// class byte the SAV writes and the class whose projectile his shots use, all
// from one derivation: a bow makes him an archer (14) who shoots, a short
// sword a swordsman (3) who does not.
func TestReleaseAHeroesWornWeaponDrivesBodyClassSaveAndShot(t *testing.T) {
	firstgame.SkipOnSecondGame(t)
	f, mw, id := bowAndSwordHero(t)
	for _, step := range []struct {
		code  uint16
		body  data.HeroBody
		class int32
		shoot bool
	}{{witnessBow, data.BodyArcher, 14, true}, {witnessSword, data.BodySwordsman, 3, false}} {
		queueWeapon(t, mw, id, step.code)
		mw.tickStep(nil, false)
		if worn, _ := mw.world.Equipped(id); worn[0] != step.code {
			t.Fatalf("slot 1 holds %#x after equipping %#x", worn[0], step.code)
		}
		member := mw.missionPartyMember(id)
		e, _ := mw.entity(id)
		look := heroAppearance(mw.world, id, *member, mw.mission.list)
		if look.Body != step.body || look.Class != step.class || member.Body != string(step.body) || member.Class != step.class {
			t.Fatalf("%#x: builder %+v, record body %q class %d; want %q and %d", step.code, look, member.Body, member.Class, step.body, step.class)
		}
		if e.Class != step.class || flightClassRule(e) != step.class {
			t.Fatalf("%#x: stored class %d, flight class %d; want %d", step.code, e.Class, flightClassRule(e), step.class)
		}
		if got := heroClassInSAV(t, f, id); got != step.class {
			t.Fatalf("%#x: SAV class %d, want %d", step.code, got, step.class)
		}
		if shoots := f.Table.FlightInputs.Classes[flightClassRule(e)].Projectile > 0; shoots != step.shoot {
			t.Fatalf("%#x: class %d shoots %v, want %v", step.code, flightClassRule(e), shoots, step.shoot)
		}
	}
}

// An unarmed fighter on mission 10's hero placement keeps the placement's
// actor source, which names the mage arm the retired flight class rule read;
// his record and his type (HERO-APPEAR-041) name the fighter. His casts use
// the class the World stores, the unarmed fighter's 1.
func TestReleaseAFighterHeroCastsFromTheClassTheWorldStores(t *testing.T) {
	firstgame.SkipOnSecondGame(t)
	f := releaseFront(t)
	a := f.App("fighter flight class")
	t.Cleanup(a.StopAudio)
	a.SetCutscenes(nil)
	if err := a.OpenMission(f.MissionOpenerWith(10, helpWitnessParty(false))); err != nil {
		t.Fatal(err)
	}
	mw := f.live
	id := mw.mission.ids[0]
	member := mw.missionPartyMember(id)
	e, _ := mw.entity(id)
	slots, _ := mw.world.Equipped(id)
	old, sourceMage := sourceArmClass(e, slots, mw.mission.list)
	look := heroAppearance(mw.world, id, *member, mw.mission.list)
	t.Logf("hero %d type %d record mage %v source mage %v: builder %d, retired rule %d", id, e.TypeID, member.Mage, sourceMage, look.Class, old)
	if member.Mage || !sourceMage || old == look.Class {
		t.Fatalf("the witness state is gone: record mage %v, source mage %v, classes %d and %d", member.Mage, sourceMage, look.Class, old)
	}
	if e.Class != look.Class || flightClassRule(e) != look.Class || look.Class != 1 {
		t.Fatalf("stored class %d, flight class %d, builder %d; want 1", e.Class, flightClassRule(e), look.Class)
	}
}

// A living map-placed Hero-type person stores, at mission open, the class his
// equipment selects (ANIM-106, ANIM-107); the SAV writes it and the flight
// class rule reads it. Mission 70 places Naira with a bow beside a male
// primary, and Danath with a sword and shield beside a female one: his
// Humans row names class 3, his equipment class 4.
func TestReleasePlacedHeroTypeHumansStoreTheClassTheirEquipmentSelects(t *testing.T) {
	firstgame.SkipOnSecondGame(t)
	for _, tc := range []struct {
		name    string
		choices []int
		class   int32
	}{{"Naira", []int{0, 0, 0}, 14}, {"Danath", []int{1, 0, 0}, 4}} {
		f := releaseFront(t)
		party := f.ChargenParty(ui.ChargenResult{Name: "Placed witness", Choices: tc.choices, Stats: []int{20, 20, 20, 20}})
		a := f.App("placed hero class")
		a.SetCutscenes(nil)
		if err := a.OpenMission(f.MissionOpenerWith(70, party)); err != nil {
			a.StopAudio()
			t.Fatal(err)
		}
		mw := f.live
		var placed []sim.EntityID
		for id := range mw.mission.state.Start.Roster {
			e, held := mw.entity(id)
			if held && data.FigureIsHero(e.TypeID) && e.Alive() && !slices.Contains(mw.mission.state.Start.IDs, id) {
				placed = append(placed, id)
			}
		}
		if len(placed) != 1 {
			a.StopAudio()
			t.Fatalf("%s: mission 70 places %d living Hero-type persons, want 1", tc.name, len(placed))
		}
		id := placed[0]
		member := mw.mission.state.Start.Roster[id]
		e, _ := mw.entity(id)
		look := heroAppearance(mw.world, id, member, mw.mission.list)
		got := heroClassInSAV(t, f, id)
		t.Logf("%s: actor %d type %d: stored %d, builder %d, SAV %d", tc.name, id, e.TypeID, e.Class, look.Class, got)
		if look.Class != tc.class || e.Class != tc.class || flightClassRule(e) != tc.class || got != tc.class || member.Class != tc.class {
			t.Errorf("%s: builder %d, stored %d, flight %d, SAV %d, record %d; want %d",
				tc.name, look.Class, e.Class, flightClassRule(e), got, member.Class, tc.class)
		}
		a.StopAudio()
	}
}

// attackNearestHostile orders id at the nearest living hostile on the map.
func attackNearestHostile(t *testing.T, mw *mapWorld, id sim.EntityID) {
	t.Helper()
	h, _ := mw.entity(id)
	target, best := sim.EntityID(0), int32(-1)
	for _, v := range mw.world.Entities() {
		if v.ID == h.ID || v.Owner == h.Owner || !v.Alive() || v.OffMap || v.MaxHP <= 0 {
			continue
		}
		if d := max(abs32(v.X-h.X), abs32(v.Y-h.Y)); best < 0 || d < best {
			target, best = v.ID, d
		}
	}
	if best < 0 {
		t.Fatal("no hostile on the map")
	}
	mw.pending = append(mw.pending, sim.Attack(id, target))
}

// stepStarts runs one tickStep and answers its flight starts of picture.
func stepStarts(mw *mapWorld, picture int32) []sim.FlightStart {
	var out []sim.FlightStart
	mw.tickStep(nil, false, func(r sim.Report) {
		for _, s := range r.FlightStarts {
			if s.Picture == picture {
				out = append(out, s)
			}
		}
	})
	return out
}

// A hero's class follows an equipment order in the step that applies it
// (ANIM-107), before that step's shots are produced. On mission 10 an archer
// attacks the nearest hostile and his first arrow leaves in step R. If his
// bow becomes a short sword in step R, no arrow leaves then or later. A
// swordsman who takes up the bow in step R looses none in that step, as his
// sword's reach holds until the step ends, and shoots later.
func TestReleaseAWeaponSwapInTheReleaseStepDecidesThatStepsShot(t *testing.T) {
	firstgame.SkipOnSecondGame(t)
	const window = 240
	f, mw, id := bowAndSwordHero(t)
	arrow := f.Table.FlightInputs.Classes[14].Projectile
	queueWeapon(t, mw, id, witnessBow)
	mw.tickStep(nil, false)
	attackNearestHostile(t, mw, id)
	release := -1
	for s := 0; s < window && release < 0; s++ {
		if len(stepStarts(mw, arrow)) > 0 {
			release = s
		}
	}
	if arrow <= 0 || release < 0 {
		t.Fatalf("the archer loosed no arrow (picture %d) in %d steps", arrow, window)
	}
	for _, tc := range []struct {
		first, second uint16
		class         int32
		later         bool
	}{{witnessBow, witnessSword, 3, false}, {witnessSword, witnessBow, 14, true}} {
		_, mw, id := bowAndSwordHero(t)
		queueWeapon(t, mw, id, tc.first)
		mw.tickStep(nil, false)
		attackNearestHostile(t, mw, id)
		for s := 0; s < release; s++ {
			if len(stepStarts(mw, arrow)) > 0 {
				t.Fatalf("%#x: an arrow left in step %d, before the release step %d", tc.first, s, release)
			}
		}
		queueWeapon(t, mw, id, tc.second)
		if got := stepStarts(mw, arrow); len(got) > 0 {
			t.Fatalf("%#x to %#x in the release step %d: arrows %+v leave, want none", tc.first, tc.second, release, got)
		}
		if e, _ := mw.entity(id); e.Class != tc.class {
			t.Fatalf("%#x to %#x: stored class %d after the step, want %d", tc.first, tc.second, e.Class, tc.class)
		}
		shot := -1
		for s := release + 1; s < window && shot < 0; s++ {
			if len(stepStarts(mw, arrow)) > 0 {
				shot = s
			}
		}
		if (shot >= 0) != tc.later {
			t.Fatalf("%#x to %#x: first arrow after the release step at %d, want one: %v", tc.first, tc.second, shot, tc.later)
		}
		t.Logf("%#x to %#x in step %d: class %d; first later arrow %d", tc.first, tc.second, release, tc.class, shot)
	}
}

// castStart is class c's cast offset at facing, restated from the flight
// geometry the cast record reads (ANIM-147): the class's shot point for the
// facing, else the middle of its selection box.
func castStart(c sim.FlightClass, facing uint8) (int32, int32) {
	if i := (int(facing>>4) - 8) & 14; len(c.ShootOffset) >= i+2 {
		return 8 * (c.ShootOffset[i] - c.CenterX), 8 * (c.ShootOffset[i+1] - c.CenterY)
	}
	return c.SelectionWidth/2 - c.CenterX, c.SelectionHeight/2 - c.CenterY
}

// A mage who takes off his staff in the step his book Fire Arrow releases
// casts from the class his empty hands select (ANIM-107): the record starts
// where class 23's cast offset puts it, not where the staff mage's 24 does.
func TestReleaseAStaffTakenOffInTheCastStepMovesThatCastsStart(t *testing.T) {
	firstgame.SkipOnSecondGame(t)
	type cast struct {
		step  int
		event sim.CastEvent
		start sim.FlightStart
		class int32
	}
	picture := int32(data.CastPicture(1))
	run := func(unequip int) (cast, *FrontEnd) {
		f, _, _, caster, target := flightMageArena(t, true)
		mw := f.live
		if e, _ := mw.entity(caster); e.Class != 24 {
			t.Fatalf("the chargen mage opens as class %d, want the staff mage's 24", e.Class)
		}
		mw.switchInventorySubject(uint32(caster))
		flightLearn(t, mw, caster, 1)
		mw.castAt(uint32(caster), uint32(target.ID), 1)
		for s := 0; s < 600; s++ {
			if s == unequip {
				mw.enqueueUnequip(0)
			}
			var got cast
			found := false
			mw.tickStep(nil, false, func(r sim.Report) {
				for _, ev := range r.Casts {
					if ev.Caster == caster && !ev.Weapon && ev.Spell == 1 {
						got.event, found = ev, true
					}
				}
				for _, fs := range r.FlightStarts {
					if fs.Picture == picture {
						got.start = fs
					}
				}
			})
			if found {
				e, _ := mw.entity(caster)
				got.step, got.class = s, e.Class
				return got, f
			}
		}
		t.Fatal("the book cast never released")
		return cast{}, nil
	}
	staff, f := run(-1)
	bare, _ := run(staff.step)
	classes := f.Table.FlightInputs.Classes
	sx, sy := castStart(classes[24], staff.event.Facing)
	bx, by := castStart(classes[23], staff.event.Facing)
	t.Logf("release step %d facing %d: staff start %d,%d class %d; bare start %d,%d class %d; offsets 24 %d,%d 23 %d,%d",
		staff.step, staff.event.Facing, staff.start.X, staff.start.Y, staff.class, bare.start.X, bare.start.Y, bare.class, sx, sy, bx, by)
	if sx == bx && sy == by {
		t.Fatalf("classes 23 and 24 share the cast offset %d,%d at facing %d: the witness cannot tell them apart", sx, sy, staff.event.Facing)
	}
	if bare.step != staff.step || bare.event.Facing != staff.event.Facing || bare.class != 23 || staff.class != 24 {
		t.Fatalf("the casts differ beyond the staff: step %d/%d facing %d/%d class %d/%d",
			staff.step, bare.step, staff.event.Facing, bare.event.Facing, staff.class, bare.class)
	}
	if bare.start.X-staff.start.X != bx-sx || bare.start.Y-staff.start.Y != by-sy {
		t.Fatalf("the bare-handed cast starts at %d,%d, the staff's at %d,%d; want a shift of %d,%d (class 23 against 24)",
			bare.start.X, bare.start.Y, staff.start.X, staff.start.Y, bx-sx, by-sy)
	}
}

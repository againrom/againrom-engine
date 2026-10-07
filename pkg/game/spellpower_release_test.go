package game

import (
	"sort"
	"testing"

	"againrom/pkg/data"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

// powerParty is a mage trained to 100 in every school. Skill 100 and the
// capped Mind put the cast power above 100 with no item bonus.
func powerParty(f *FrontEnd) []mapload.PartyMember {
	party := hasteParty()
	hero := data.Hero{Body: 60, Reaction: 60, Mind: 100, Spirit: 100}
	for i := range hero.Skill {
		hero.Skill[i] = 100
	}
	party[0].Hero = hero
	party[0].KnownSpells = 1<<24 | 1<<15 | 1<<5 | 1<<18
	return party
}

func powerEffects(f *FrontEnd) []sim.ActiveEffect {
	out := f.live.world.ActiveEffects()
	sort.Slice(out, func(i, j int) bool { return out[i].Spell < out[j].Spell })
	for i := range out {
		out[i].Target, out[i].Caster = 0, 0
	}
	return out
}

// Effects cast at power 120 carry the lengthened and segmented durations, SAVE
// writes them through the ordinary producer and a cold LOAD holds the same
// effects.
func TestReleaseSpellPowerAbove100EffectsSaveAndColdLoad(t *testing.T) {
	f := releaseFront(t)
	f.SetDeterministicFrames(true)
	if err := f.App("spell power").OpenMission(f.MissionOpenerWith(101, powerParty(f))); err != nil {
		t.Fatal(err)
	}
	caster, target := f.live.mission.ids[0], f.live.mission.ids[1]
	rules := mapload.SpellRules(f.Table)
	rule := func(id uint16) sim.SpellRule {
		for _, r := range rules {
			if r.ID == id {
				return r
			}
		}
		t.Fatalf("no spell row %d", id)
		return sim.SpellRule{}
	}
	if got := sim.SpellCharacteristicsFor(sim.Rules{}, releaseEntity(t, f.live, caster), rule(24)).Power; got <= 100 {
		t.Fatalf("cast power %d, want above 100", got)
	}
	victim := releaseEntity(t, f.live, target)
	first := map[uint16]uint16{24: 4645, 5: 9291, 18: 4645, 15: 6439}
	for _, id := range []uint32{24, 5, 18, 15} {
		at, at2 := target, victim
		if id == 18 || id == 15 {
			at, at2 = caster, releaseEntity(t, f.live, caster)
		}
		f.live.attackOrCast(uint32(caster), uint32(at), id, int(at2.X), int(at2.Y), false)
		for tick := 0; ; tick++ {
			if tick > 600 {
				t.Fatal("spell never attached", id, f.live.world.BookSpellRefusal(caster, at, id))
			}
			f.live.tick()
			attached := false
			for _, e := range f.live.world.ActiveEffects() {
				if uint32(e.Spell) == id {
					attached = true
					at := int(e.Remaining) - int(first[e.Spell])
					if at < -3 || at > 3 {
						t.Errorf("spell %d attached with %d ticks, want %d", id, e.Remaining, first[e.Spell])
					}
				}
			}
			if attached {
				break
			}
		}
		for n := 0; n < 600 && releaseEntity(t, f.live, caster).CastWait > 0; n++ {
			f.live.tick()
		}
	}
	live := powerEffects(f)
	if len(live) != len(first) {
		t.Fatalf("live effects %+v, want spells 24, 5, 18, 15", live)
	}
	for _, e := range live {
		if e.Spell == 24 && e.Magnitude != 9 {
			t.Errorf("haste magnitude %d, want 9", e.Magnitude)
		}
	}
	store := SaveStore{Dir: t.TempDir()}
	name, _ := deadPatrolSave(t, f, store)
	cold := loadLocalLegacySave(t, store, name)
	if got := powerEffects(cold); !equalEffects(got, live) {
		t.Fatalf("cold LOAD effects %+v, want %+v", got, live)
	}
	for n := 0; n < 300; n++ {
		f.live.tick()
		cold.live.tick()
	}
	if got, again := powerEffects(cold), powerEffects(f); !equalEffects(got, again) {
		t.Fatalf("after the next ticks cold %+v, live %+v", got, again)
	}
	if got := releaseEntity(t, cold.live, cold.live.mission.ids[1]).Speed; got != releaseEntity(t, f.live, target).Speed {
		t.Fatalf("cold speed %d, live %d", got, releaseEntity(t, f.live, target).Speed)
	}
	_, resaved := deadPatrolSave(t, cold, SaveStore{Dir: t.TempDir()})
	if reread := hasteSaveRead(t, resaved, 15); reread.Magnitude != 9 {
		t.Fatalf("second SAVE haste magnitude %d, want 9", reread.Magnitude)
	}
}

func equalEffects(a, b []sim.ActiveEffect) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// Every installed damaging row stays inside the byte its effect record holds
// at the power bound, so no in-flight payload saturates on installed data.
func TestReleaseSpellDamageFitsTheEffectByteAtPower255(t *testing.T) {
	f := releaseFront(t)
	for _, rule := range mapload.SpellRules(f.Table) {
		if !rule.Damaging && rule.ID != 11 && !rule.Restorative {
			continue
		}
		c := sim.SpellCharacteristicsFor(sim.Rules{}, sim.Entity{Mind: 285}, rule)
		if c.Power != 255 {
			t.Fatalf("power %d, want 255", c.Power)
		}
		hi := (int64(rule.DamageMax) * (int64(c.Power) + 30)) / 30
		if hi > 255 {
			t.Errorf("spell %d damage at power 255 reaches %d, above the byte", rule.ID, hi)
		}
		t.Logf("spell %d: damage %d..%d at power 255", rule.ID, int64(rule.DamageMin)*(int64(c.Power)+30)/30, hi)
	}
}

// installedSpeedRange is the slowest and fastest positive Speed among the
// installed Units rows.
func installedSpeedRange(t *testing.T, f *FrontEnd) (slowest, fastest int32) {
	t.Helper()
	units := f.Table.Units
	for i := 1; i < units.Len(); i++ {
		d, err := data.NewUnitDef(units.EntryName(i), units.EntryParams(i))
		if err != nil || d.Speed <= 0 {
			continue
		}
		if slowest == 0 || d.Speed < slowest {
			slowest = d.Speed
		}
		fastest = max(fastest, d.Speed)
	}
	if slowest == 0 {
		t.Fatal("no installed unit has a positive speed")
	}
	return
}

// Slow and Freezing Cloud at power 255 take 18 each from a unit's speed and
// never below 1. A unit at the floor moves at the cadence of an unslowed
// speed-1 unit, and expiry returns the exact original speed.
func TestReleaseSlowAtPower255KeepsTheSpeedFloor(t *testing.T) {
	f := releaseFront(t)
	rules := mapload.SpellRules(f.Table)
	slowest, fastest := installedSpeedRange(t, f)
	t.Logf("installed speeds %d..%d", slowest, fastest)
	build := func(speed, mind int32, spells ...uint32) *sim.World {
		var known uint32
		for _, s := range spells {
			known |= 1 << s
		}
		caster := sim.Entity{ID: 1, X: 2, Y: 2, Owner: 1, HP: 100, MaxHP: 100, Mana: 1000, MaxMana: 1000, Mind: mind,
			Reach: 1, ScanRange: 8, KnownSpells: known, TokenSize: 1, TypeID: sim.HumanTypeID, DyingTime: 200}
		victim := sim.Entity{ID: 2, X: 6, Y: 2, Owner: 2, HP: 10000, MaxHP: 10000, DyingTime: 200, Speed: speed, TokenSize: 1}
		var relations sim.Relations
		relations.Set(1, 2, 1)
		w, err := sim.NewStockedSpelledWorld(1323, sim.Bounds{Width: 64, Height: 64}, sim.ModeCanonical,
			sim.Terrain{}, []sim.Entity{caster, victim}, nil, relations, nil, nil, rules)
		if err != nil {
			t.Fatal(err)
		}
		return w
	}
	entity := func(w *sim.World, id sim.EntityID) sim.Entity {
		for _, e := range w.Entities() {
			if e.ID == id {
				return e
			}
		}
		t.Fatalf("no entity %d", id)
		return sim.Entity{}
	}
	speedAfter := func(w *sim.World) int32 { return entity(w, 2).Speed }
	activeSpells := func(w *sim.World) map[uint16]sim.ActiveEffect {
		out := map[uint16]sim.ActiveEffect{}
		for _, e := range w.ActiveEffects() {
			out[e.Spell] = e
		}
		return out
	}
	transit := func(w *sim.World) uint16 {
		sim.Step(w, []sim.Command{sim.MoveTo(2, sim.CellPoint{X: 6, Y: 50})})
		return entity(w, 2).TransitTotal
	}
	for _, speed := range []int32{slowest, fastest} {
		w := build(speed, 285, 28, 7)
		sim.Step(w, []sim.Command{sim.Cast(1, 2, 28)})
		for i := 0; i < 400 && activeSpells(w)[28].Spell == 0; i++ {
			sim.Step(w, nil)
		}
		slow, ok := activeSpells(w)[28]
		// The stored magnitude is what landed: 18 less whatever the floor swallowed.
		if !ok || slow.Magnitude != -min(18, speed-1) {
			t.Fatalf("speed %d: Slow effect %+v, want magnitude %d", speed, slow, -min(18, speed-1))
		}
		want := max(1, speed-18)
		if got := speedAfter(w); got != want {
			t.Fatalf("speed %d under Slow = %d, want %d", speed, got, want)
		}
		sim.Step(w, []sim.Command{sim.CastAt(1, 7, sim.CellPoint{X: 6, Y: 2})})
		for i := 0; i < 600 && activeSpells(w)[7].Spell == 0; i++ {
			sim.Step(w, nil)
		}
		cloud, ok := activeSpells(w)[7]
		if !ok || cloud.Magnitude != -min(18, max(0, speed-1+slow.Magnitude)) {
			t.Fatalf("speed %d: Freezing Cloud effect %+v, want magnitude %d", speed, cloud, -min(18, max(0, speed-1+slow.Magnitude)))
		}
		want = max(1, speed-36)
		if got := speedAfter(w); got != want {
			t.Fatalf("speed %d under Slow and Freezing Cloud = %d, want %d", speed, got, want)
		}
		if speed <= 37 && want != 1 {
			t.Fatalf("speed %d: floor expected", speed)
		}
		if want == 1 {
			ref := build(1, 285)
			if got, floor := transit(w), transit(ref); got != floor || got == 0 {
				t.Fatalf("speed %d at the floor transits in %d, an unslowed speed-1 unit in %d", speed, got, floor)
			}
		}
		// Slow holds 65535 ticks, above the 9600 a counter does not pass, so it
		// stays; the cloud's own shorter effect may expire and give back its part.
		for i := 0; i < 3000; i++ {
			sim.Step(w, nil)
		}
		if got := speedAfter(w); got != max(1, speed-18) && got != want || activeSpells(w)[28].Remaining != 65535 {
			t.Fatalf("speed %d after 3000 ticks = %d with Slow remaining %d", speed, got, activeSpells(w)[28].Remaining)
		}

		// At power 120 the same effects expire and give the speed back.
		e := build(speed, 150, 28, 7)
		sim.Step(e, []sim.Command{sim.Cast(1, 2, 28)})
		for i := 0; i < 400 && activeSpells(e)[28].Spell == 0; i++ {
			sim.Step(e, nil)
		}
		sim.Step(e, []sim.Command{sim.CastAt(1, 7, sim.CellPoint{X: 6, Y: 2})})
		for i := 0; i < 600 && activeSpells(e)[7].Spell == 0; i++ {
			sim.Step(e, nil)
		}
		if got := speedAfter(e); got != max(1, speed-18) && got != 1 {
			t.Fatalf("speed %d at power 120 under both = %d", speed, got)
		}
		if got := speedAfter(e); speed <= 10 && got != 1 {
			t.Fatalf("speed %d at power 120 under both = %d, want the floor", speed, got)
		}
		for i := 0; i < 60000 && len(activeSpells(e)) > 0; i++ {
			sim.Step(e, nil)
		}
		if len(activeSpells(e)) != 0 {
			t.Fatalf("speed %d: power-120 effects never expired", speed)
		}
		if got := speedAfter(e); got != speed {
			t.Fatalf("speed %d after expiry = %d", speed, got)
		}
	}
}

// powerBonusMage is a chargen mage trained to 100 in every school wearing an
// armor piece whose five school bonuses lift each skill to 250, so skill and
// Mind put the cast power at 244.
func powerBonusMage(f *FrontEnd) []mapload.PartyMember {
	party := f.ChargenParty(ui.ChargenResult{Name: "power", Choices: []int{0, 1, 3}, Stats: []int{31, 27, 24, 29}})
	for i := range party[0].Hero.Skill {
		party[0].Hero.Skill[i] = 100
	}
	party[0].KnownSpells |= 1<<24 | 1<<15 | 1<<5 | 1<<18
	item := mapload.SourceConstructedItem(sim.PlainItem(0xec7e), f.Table)
	for kind := uint8(33); kind <= 37; kind++ {
		item.Effects = append(item.Effects, sim.ItemEffect{Kind: kind, Operand: 150})
	}
	party[0].WornItems[11], party[0].Worn[11] = item, item.Code
	party[0].Carry = &mapload.Carry{Equipped: party[0].Worn, EquippedItems: party[0].WornItems}
	return party
}

// A worn bonus lifts the power to 244. The self-cast effects hold the segmented
// Invisibility word and the saturated others; SAVE writes them, a cold LOAD holds
// the same skills, effects and spellbook, and both worlds stay equal afterwards.
func TestReleaseSpellPowerBoundEffectsSaveAndColdLoad(t *testing.T) {
	f := releaseFront(t)
	f.SetDeterministicFrames(true)
	app := f.App("spell power bound")
	app.Layout(1024, 768)
	if err := app.OpenMission(f.MissionOpenerWith(20, powerBonusMage(f))); err != nil {
		t.Fatal(err)
	}
	caster := f.live.mission.ids[0]
	for _, r := range mapload.SpellRules(f.Table) {
		if r.ID == 24 {
			if got := sim.SpellCharacteristicsFor(sim.Rules{}, releaseEntity(t, f.live, caster), r).Power; got != 244 {
				t.Fatalf("cast power %d, want 244", got)
			}
		}
	}
	at := releaseEntity(t, f.live, caster)
	want := map[uint16]uint16{24: 65535, 5: 65535, 18: 65535, 15: 13034}
	for _, id := range []uint32{24, 5, 18, 15} {
		f.live.attackOrCast(uint32(caster), uint32(caster), id, int(at.X), int(at.Y), false)
		for tick := 0; ; tick++ {
			if tick > 600 {
				t.Fatal("spell never attached", id, f.live.world.BookSpellRefusal(caster, caster, id))
			}
			f.live.tick()
			done := false
			for _, e := range f.live.world.ActiveEffects() {
				if uint32(e.Spell) == id {
					done = true
					if d := int(e.Remaining) - int(want[e.Spell]); d < -2 || d > 2 {
						t.Errorf("spell %d attached with %d ticks, want %d", id, e.Remaining, want[e.Spell])
					}
				}
			}
			if done {
				break
			}
		}
		for n := 0; n < 600 && releaseEntity(t, f.live, caster).CastWait > 0; n++ {
			f.live.tick()
		}
	}
	live := powerEffects(f)
	if len(live) != len(want) {
		t.Fatalf("live effects %+v, want spells 24, 5, 18, 15", live)
	}
	for _, e := range live {
		if e.Spell == 24 && e.Magnitude != 17 {
			t.Errorf("haste magnitude %d, want 17", e.Magnitude)
		}
	}
	store := SaveStore{Dir: t.TempDir()}
	name, _ := deadPatrolSave(t, f, store)
	cold := loadLocalLegacySave(t, store, name)
	if got := powerEffects(cold); !equalEffects(got, live) {
		t.Fatalf("cold LOAD effects %+v, want %+v", got, live)
	}
	ce, le := releaseEntity(t, cold.live, cold.live.mission.ids[0]), releaseEntity(t, f.live, caster)
	if ce.Skill != le.Skill || ce.Speed != le.Speed || ce.Book != le.Book {
		t.Fatalf("cold actor skill %v speed %d, live skill %v speed %d (book equal %t)", ce.Skill, ce.Speed, le.Skill, le.Speed, ce.Book == le.Book)
	}
	for n := 0; n < 300; n++ {
		f.live.tick()
		cold.live.tick()
	}
	if got, again := powerEffects(cold), powerEffects(f); !equalEffects(got, again) {
		t.Fatalf("after the next ticks cold %+v, live %+v", got, again)
	}
}

func equipInPlayMage(f *FrontEnd, base int32, mind int32) []mapload.PartyMember {
	party := f.ChargenParty(ui.ChargenResult{Name: "power", Choices: []int{0, 1, 3}, Stats: []int{31, 27, 24, 29}})
	for i := range party[0].Hero.Skill {
		party[0].Hero.Skill[i] = base
	}
	party[0].Hero.Mind = mind
	party[0].KnownSpells |= 1<<24 | 1<<15 | 1<<5 | 1<<18
	item := mapload.SourceConstructedItem(sim.PlainItem(0xec7e), f.Table)
	for kind := uint8(33); kind <= 37; kind++ {
		item.Effects = append(item.Effects, sim.ItemEffect{Kind: kind, Operand: 30})
	}
	party[0].Carry = &mapload.Carry{Items: []uint16{item.Code}, ItemInstances: []sim.ItemInstance{item}}
	return party
}

// A mage who equips a school bonus in play and saves loads with the same
// skills, spellbook ranges and spell power, and casts at that power.
func TestReleaseSpellPowerBonusEquippedInPlaySavesAndLoads(t *testing.T) {
	for _, tc := range []struct {
		base, mind int32
		above100   bool
	}{{100, 24, true}, {100, 100, true}, {100, 10, true}, {80, 60, false}, {60, 40, false}} {
		f := releaseFront(t)
		f.SetDeterministicFrames(true)
		app := f.App("equip in play")
		app.Layout(1024, 768)
		if err := app.OpenMission(f.MissionOpenerWith(20, equipInPlayMage(f, tc.base, tc.mind))); err != nil {
			t.Fatal(err)
		}
		caster := f.live.mission.ids[0]
		sim.Step(f.live.world, []sim.Command{sim.Equip(caster, 0, 12)})
		for i := 0; i < 5; i++ {
			f.live.tick()
		}
		live := releaseEntity(t, f.live, caster)
		if live.Skill[1] != tc.base+30 {
			t.Fatalf("base %d mind %d: skill %d after equip, want %d", tc.base, tc.mind, live.Skill[1], tc.base+30)
		}
		store := SaveStore{Dir: t.TempDir()}
		name, _ := deadPatrolSave(t, f, store)
		cold := loadLocalLegacySave(t, store, name)
		for i := 0; i < 5; i++ {
			f.live.tick()
			cold.live.tick()
		}
		got := releaseEntity(t, cold.live, cold.live.mission.ids[0])
		live = releaseEntity(t, f.live, caster)
		if got.Skill != live.Skill || got.Book != live.Book {
			t.Fatalf("base %d mind %d: cold skill %v, live %v (books equal %t)", tc.base, tc.mind, got.Skill, live.Skill, got.Book == live.Book)
		}
		for _, r := range mapload.SpellRules(f.Table) {
			if r.ID != 24 && r.ID != 1 && r.ID != 26 {
				continue
			}
			a, b := sim.SpellCharacteristicsFor(sim.Rules{}, live, r), sim.SpellCharacteristicsFor(sim.Rules{}, got, r)
			if a != b {
				t.Fatalf("base %d mind %d spell %d: live %+v, cold %+v", tc.base, tc.mind, r.ID, a, b)
			}
			if r.ID == 24 && tc.above100 && a.Power <= 100 && tc.base+30+live.Mind-30 > 100 {
				t.Fatalf("power %d, want above 100", a.Power)
			}
		}
		// The next ordinary save writes the same book.
		name2, _ := deadPatrolSave(t, cold, SaveStore{Dir: t.TempDir()})
		_ = name2
	}
}

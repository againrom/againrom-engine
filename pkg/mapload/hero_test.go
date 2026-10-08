package mapload_test

import (
	"testing"

	"againrom/pkg/data"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
)

// heroSword is the shipped starting weapon as the resolver builds it from an
// installed table. It is written out here rather than resolved, because these
// tests are about what a START does with a member's numbers.
var heroSword = data.Weapon{
	Name: "Iron Short Sword", DamageBase: 5, DamageSpread: 3,
	ToHit: 5, Defence: 0, AttackType: data.SkillBlade,
	ChargeTime: 9, RelaxTime: 5, Range: 1,
}

// lastEntity is the member a one-member party added: the party is APPENDED, so
// it is the world's last.
func lastEntity(t *testing.T, w *sim.World) sim.Entity {
	t.Helper()
	e := w.Entities()
	if len(e) == 0 {
		t.Fatal("the world holds no entity at all")
	}
	return e[len(e)-1]
}

// THE START REPORTS THE ID IT MINTED FOR EACH MEMBER, and the check is that
// those ids are the ids the world's own entities actually carry rather than
// a plausible arithmetic. A statistic reaches no entity field, so this is
// the only thread back from a placed unit to the character it was folded
// from, and a caller recomputing it as "the last n entities" would be
// holding a second copy of a rule this package owns.
func TestAStartReportsTheEntityIdOfEachMember(t *testing.T) {
	m := startMap(t, 40, 40, mapload.Cell{X: 17, Y: 20})
	party := []mapload.PartyMember{
		{Class: 100, Hero: data.NewCampaignHero(data.SkillBlade), Weapon: &heroSword},
		{Class: 101, Hero: data.NewCampaignHero(data.SkillAxe)},
		{Class: 102, Hero: data.NewCampaignHero(data.SkillPike), Weapon: &heroSword},
	}
	w, st := mustStart(t, m, party)

	if len(st.IDs) != len(party) || len(st.Cells) != len(party) {
		t.Fatalf("%d id(s) and %d cell(s) for %d member(s)", len(st.IDs), len(st.Cells), len(party))
	}
	byID := map[sim.EntityID]sim.Entity{}
	for _, e := range w.Entities() {
		byID[e.ID] = e
	}
	for i, id := range st.IDs {
		e, ok := byID[id]
		if !ok {
			t.Fatalf("member %d was reported as entity %d, which the world does not hold", i, id)
		}
		// The id, the cell and the class all have to name ONE unit, or the two
		// slices are parallel to each other and to nothing else.
		if e.Class != party[i].Class || e.X != st.Cells[i].X || e.Y != st.Cells[i].Y {
			t.Errorf("member %d is entity %d at (%d, %d) class %d; the start says (%d, %d) class %d",
				i, id, e.X, e.Y, e.Class, st.Cells[i].X, st.Cells[i].Y, party[i].Class)
		}
	}

	// A start with no party reports no ids, exactly as it reports no cells.
	if _, empty := mustStart(t, m, nil); len(empty.IDs) != 0 {
		t.Errorf("a partyless start reported %d id(s)", len(empty.IDs))
	}
}

// A STARTED PARTY MEMBER CARRIES THE FOLD'S OWN NUMBERS, and they are the eight
// the resolver reads. This is the whole of 0078 measured at the seam the player
// reaches: before it, six of these were zero.
func TestAStartedPartyMemberCarriesItsHerosNumbers(t *testing.T) {
	m := startMap(t, 40, 40, mapload.Cell{X: 17, Y: 20})
	w, _ := mustStart(t, m, []mapload.PartyMember{{
		Class:  100,
		Hero:   data.NewCampaignHero(data.SkillBlade),
		Weapon: &heroSword,
	}})

	got := lastEntity(t, w)
	want := data.NewCampaignHero(data.SkillBlade).Derive(&heroSword)
	if got.DamageBase != want.DamageBase || got.DamageSpread != want.DamageSpread {
		t.Errorf("damage (%d, %d), want (%d, %d)",
			got.DamageBase, got.DamageSpread, want.DamageBase, want.DamageSpread)
	}
	if got.ToHit != want.ToHit || got.Defence != want.Defence || got.Absorption != want.Absorption {
		t.Errorf("toHit %d defence %d absorption %d, want %d / %d / %d",
			got.ToHit, got.Defence, got.Absorption, want.ToHit, want.Defence, want.Absorption)
	}
	if got.AttackCharge != want.AttackChargeTime || got.AttackRelax != want.AttackRelaxTime {
		t.Errorf("cadence %d/%d, want the weapon's %d/%d",
			got.AttackCharge, got.AttackRelax, want.AttackChargeTime, want.AttackRelaxTime)
	}
	if !got.Humanoid {
		t.Error("a started party member is not classified Humanoid for canonical action recovery")
	}
	if got.AlwaysHits {
		t.Error("a hero carries the auto-hit mark; its only writer is the non-hero streamer")
	}
	// And the numbers are not zero, which is what the story is about.
	if got.DamageBase == 0 && got.DamageSpread == 0 {
		t.Error("the member still swings for nothing")
	}
}

func TestAStartedPartyMembersWeaponSpellReachesHisEntity(t *testing.T) {
	staff := data.Weapon{Name: "Wood Staff", DamageBase: 1, DamageSpread: 1,
		AttackType: data.SkillBlade, ChargeTime: 6, RelaxTime: 4, Range: 5,
		SpellName: "Fire_Arrow", SpellPower: 10}
	table := &mapload.Table{Spells: defCollection{
		{}, // 0: reserved
		{name: "Fire Arrow", params: spellRow(3, 1, 1, 7, 4, 8, 0)},
	}}
	m := startMap(t, 40, 40, mapload.Cell{X: 17, Y: 20})

	w, _, err := mapload.StartMission(m, table, mapload.DifficultyNormal, []mapload.PartyMember{{
		Class: 100, Hero: data.NewCampaignHero(data.SkillBlade), Weapon: &staff,
	}})
	if err != nil {
		t.Fatalf("StartMission: %v", err)
	}
	got := lastEntity(t, w)
	if got.WeaponSpell != 1 {
		t.Errorf("WeaponSpell = %d, want 1 (the resolved Fire Arrow id)", got.WeaponSpell)
	}
	if got.WeaponSpellLevel != 10 {
		t.Errorf("WeaponSpellLevel = %d, want 10 (the weapon's own level)", got.WeaponSpellLevel)
	}

	sim.Step(w, nil)
	afterTick := lastEntity(t, w)
	if afterTick.WeaponSpell != 1 || afterTick.WeaponSpellLevel != 10 {
		t.Errorf("after one tick, weapon spell = (%d, %d), want (1, 10) — the pair must survive Step",
			afterTick.WeaponSpell, afterTick.WeaponSpellLevel)
	}

	bare, _, err := mapload.StartMission(m, table, mapload.DifficultyNormal, []mapload.PartyMember{{
		Class: 100, Hero: data.NewCampaignHero(data.SkillBlade), Weapon: &heroSword,
	}})
	if err != nil {
		t.Fatalf("StartMission: %v", err)
	}
	if got := lastEntity(t, bare); got.WeaponSpell != 0 || got.WeaponSpellLevel != 0 {
		t.Errorf("weapon spell = (%d, %d) for a weapon carrying no spell, want (0, 0)",
			got.WeaponSpell, got.WeaponSpellLevel)
	}
}

// entityByID is a world's one entity carrying id, and whether the world
// holds it — the same pair cmd/missionrun's own hpOf answers, restated here
// because this file's tests read more than the one field that helper was
// built for.
func entityByID(w *sim.World, id sim.EntityID) (sim.Entity, bool) {
	for _, e := range w.Entities() {
		if e.ID == id {
			return e, true
		}
	}
	return sim.Entity{}, false
}

// TestAGeneratedCasterAttacksAtRangeWithNoToHitRoll is AC-13 and SC-1's own
// claim, driven through the full mint-and-cycle pipeline rather than
// asserted at either end alone: a party member built as a caster —
// Profile.ManaColumn true (FR-2a's own mana pool), a weapon carrying a
// spell — mints an entity that, ordered to attack a nearby victim, RELEASES
// A CAST rather than landing a blow.
//
// THE TWO DAMAGE SOURCES ARE DELIBERATELY IN DISJOINT RANGES: the staff's
// own melee pair is 1-2 (DamageBase 1, DamageSpread 1) and the spell's own
// columns, at power 10 (spellDamage's own `f = power+30`, pkg/sim/spell.go),
// scale to EXACTLY 15*40/30 = 20 with a spread of 0 — deterministic, and
// far outside the melee band. A victim landing at exactly 20 down could not
// have taken the weapon's own blow; one landing at 1 or 2 could not have
// taken the cast. therefore never misses" — is pkg/sim's own coverage
// (weaponspell_test.go) and is not re-measured here; what this test adds is
// that a party built through mapload.StartMission's ordinary door reaches
// that mechanism at all.
//
// TO CONFIRM THIS TEST WITNESSES THE WIRING AND NOT JUST pkg/sim's OWN
// MECHANISM, clear WeaponSpell in start.go's entity mint (as the earlier
// test in this file already isolates) and rerun: the victim's health never
// moves and this test times out its own 30-tick ceiling instead.
func TestAGeneratedCasterAttacksAtRangeWithNoToHitRoll(t *testing.T) {
	staff := data.Weapon{Name: "Wood Staff", DamageBase: 1, DamageSpread: 1,
		AttackType: data.SkillBlade, ChargeTime: 4, RelaxTime: 4, Range: 1,
		SpellName: "Fire_Arrow", SpellPower: 10}
	table := &mapload.Table{Spells: defCollection{
		{}, // 0: reserved
		{name: "Fire Arrow", params: spellRow(0, 1, 1, 5, 15, 15, 0)},
	}}
	m := startMap(t, 40, 40, mapload.Cell{X: 17, Y: 20})
	party := []mapload.PartyMember{
		// THE CASTER: Mage true is a loader/appearance fact alone (hero.go's
		// own doc on PartyMember.Mage) and plays no part in pkg/sim's own
		// FR-2a test, isMage — it is set here for the shape a real mage
		// carries, not because anything under test reads it.
		{Class: 100, Hero: data.NewCampaignHero(data.SkillBlade), Weapon: &staff,
			Profile: data.Profile{ManaColumn: true}, Mage: true},
		// THE VICTIM: the zero profile, so PartySpawn mints him at the
		// provisional SpawnHP (100) — comfortably more than either damage
		// source could remove in one release.
		{Class: 101, Hero: data.NewCampaignHero(data.SkillAxe)},
	}
	w, st, err := mapload.StartMission(m, table, mapload.DifficultyNormal, party)
	if err != nil {
		t.Fatalf("StartMission: %v", err)
	}
	if len(st.IDs) != 2 {
		t.Fatalf("StartMission minted %d id(s), want 2", len(st.IDs))
	}

	caster, ok := entityByID(w, st.IDs[0])
	if !ok {
		t.Fatal("setup: the caster's own entity is missing")
	}
	if caster.MaxMana <= 0 {
		t.Fatalf("setup: MaxMana = %d, want > 0 — the caster is not eligible under FR-2a", caster.MaxMana)
	}
	if caster.WeaponSpell == 0 {
		t.Fatal("setup: WeaponSpell = 0 — the staff's own spell did not reach the entity")
	}
	victim, ok := entityByID(w, st.IDs[1])
	if !ok {
		t.Fatal("setup: the victim's own entity is missing")
	}
	dx, dy := caster.X-victim.X, caster.Y-victim.Y
	if dx < 0 {
		dx = -dx
	}
	if dy < 0 {
		dy = -dy
	}
	dist := dx
	if dy > dist {
		dist = dy
	}
	if dist > 5 {
		t.Fatalf("setup: caster and victim are %d cells apart on an open map, want at most 5 "+
			"(the spell's own range) — freeCell's own outward walk should have placed the victim adjacent", dist)
	}
	startHP := victim.HP

	sim.Step(w, []sim.Command{{Kind: sim.KindAttack, Entity: caster.ID, X: int32(victim.ID)}})
	var released bool
	for i := 0; i < 30; i++ {
		v, ok := entityByID(w, victim.ID)
		if !ok {
			t.Fatal("the victim vanished from the world")
		}
		if v.HP != startHP {
			released = true
			if v.HP != startHP-18 {
				t.Errorf("victim health %d after %d tick(s), want exactly %d (startHP-18 after its 10%% protection, "+
					"the deterministic cast damage — a different number means the MELEE pair fired instead)",
					v.HP, i, startHP-18)
			}
			break
		}
		sim.Step(w, nil)
	}
	if !released {
		t.Fatal("30 ticks passed and the victim's health never moved — no cast was released")
	}
}

// THE HEALTH PAIR, THE RATE AND THE DOMAIN DO NOT MOVE. Each is derived from
// these same statistics by the original and each is a DISCLOSED DIVERGENCE of
// this story; a test states them so a later story that derives one has to come
// through here.
func TestAStartedPartyMemberKeepsTheHealthRateAndDomainItHad(t *testing.T) {
	m := startMap(t, 40, 40, mapload.Cell{X: 17, Y: 20})
	w, _ := mustStart(t, m, []mapload.PartyMember{{
		Class: 100, Hero: data.NewCampaignHero(data.SkillBlade), Weapon: &heroSword,
	}})

	got := lastEntity(t, w)
	if got.HP != mapload.SpawnHP || got.MaxHP != mapload.SpawnHP {
		t.Errorf("health %d/%d, want the provisional %d on both", got.HP, got.MaxHP, mapload.SpawnHP)
	}
	if got.Domain != sim.DomainGround {
		t.Errorf("domain %v, want ground", got.Domain)
	}
	// SPEED IS NO LONGER THE CONSTRUCTOR'S. This assertion is the pin that said
	// it was, deliberately moved by the story that derives it — which is the
	// pin doing its job rather than being edited around.
	want := data.NewCampaignHero(data.SkillBlade).Speed()
	if got.Speed != want {
		t.Errorf("speed %d, want his hero's own %d", got.Speed, want)
	}
	if got.Speed == data.UnitDefaults().Speed {
		t.Errorf("speed is still the unit table's %d", data.UnitDefaults().Speed)
	}
}

// THE ZERO-VALUE MEMBER CARRIES WHAT THE PARTY CARRIED BEFORE 0078, and it gets
// there through the same arithmetic rather than through a fallback — which is
// what leaves every world this tree already builds byte for byte unchanged.
func TestAZeroValueMemberCarriesTheNumbersThePartyHadBefore(t *testing.T) {
	m := startMap(t, 40, 40, mapload.Cell{X: 17, Y: 20})
	w, _ := mustStart(t, m, []mapload.PartyMember{{Class: 100}})

	got := lastEntity(t, w)
	def := data.UnitDefaults()
	if got.DamageBase != 0 || got.DamageSpread != 0 || got.ToHit != 0 ||
		got.Defence != 0 || got.Absorption != 0 || got.AlwaysHits {
		t.Errorf("a zero member carries %+v, want the six zeroes an unresolved placement takes", got)
	}
	if got.AttackCharge != def.AttackChargeTime || got.AttackRelax != def.AttackRelaxTime {
		t.Errorf("cadence %d/%d, want the constructor's %d/%d",
			got.AttackCharge, got.AttackRelax, def.AttackChargeTime, def.AttackRelaxTime)
	}
	// HIS SPEED REACHES ITS ANSWER BY THE SAME ARITHMETIC AND TAKES NO
	// FALLBACK. A hero with no Reaction gets what the law gives at Reaction 0,
	// which is 0 and not the unit table's 10 — and 0 is a rate the movement
	// law floors at 1, so he is slow rather than stuck.
	if got.Speed != 0 || got.Speed != (data.Hero{}).Speed() {
		t.Errorf("a zero member's speed is %d, want the %d the law gives at Reaction 0",
			got.Speed, (data.Hero{}).Speed())
	}
}

// A BARE HERO IS A REAL STATE and the start models it: at the chargen Body of
// 25 he swings for nothing, which is the original's own answer and not a hole.
func TestABarePartyMemberSwingsForNothingAtTheChargenStart(t *testing.T) {
	m := startMap(t, 40, 40, mapload.Cell{X: 17, Y: 20})
	w, _ := mustStart(t, m, []mapload.PartyMember{{
		Class: 100, Hero: data.NewCampaignHero(data.SkillBlade), // no weapon
	}})

	got := lastEntity(t, w)
	if got.DamageBase != 0 || got.DamageSpread != 0 {
		t.Errorf("a bare hero at Body 25 swings for (%d, %d), want (0, 0)",
			got.DamageBase, got.DamageSpread)
	}
	// He is not INERT, though: to-hit and defence come off the stats alone.
	if got.ToHit == 0 || got.Defence == 0 {
		t.Errorf("toHit %d defence %d — both come off the stats with no weapon",
			got.ToHit, got.Defence)
	}
}

func TestAStartedPartyMembersReachIsHisWeaponsRange(t *testing.T) {
	bow := data.Weapon{
		Name: "fixture bow", DamageBase: 2, DamageSpread: 2,
		ToHit: 4, AttackType: data.SkillShoot,
		ChargeTime: 12, RelaxTime: 6, Range: 4,
	}
	p := []mapload.PartyMember{
		{Class: 100, Hero: data.NewCampaignHero(data.SkillShoot), Weapon: &bow},
		{Class: 101, Hero: data.NewCampaignHero(data.SkillBlade), Weapon: &heroSword},
		{Class: 102, Hero: data.NewCampaignHero(data.SkillBlade)}, // bare
	}
	want := []uint8{4, 1, 1}
	w, st := mustStart(t, startMap(t, 40, 40, mapload.Cell{X: 17, Y: 20}), p)
	byID := map[sim.EntityID]sim.Entity{}
	for _, e := range w.Entities() {
		byID[e.ID] = e
	}
	for i := range p {
		e, ok := byID[st.IDs[i]]
		if !ok {
			t.Fatalf("party member %d is not in the world", i)
		}
		if e.Reach != want[i] {
			t.Errorf("party member %d reaches %d, want his weapon's %d", i, e.Reach, want[i])
		}
	}
	for _, e := range w.Entities() {
		if e.Reach == 0 {
			t.Errorf("entity %d leaves the start with no reach at all", e.ID)
		}
	}
}

func TestAWorldHoldingAnArmedPartyRoundTrips(t *testing.T) {
	m := startMap(t, 40, 40, mapload.Cell{X: 17, Y: 20})
	w, _ := mustStart(t, m, []mapload.PartyMember{
		{Class: 100, Hero: data.NewCampaignHero(data.SkillBlade), Weapon: &heroSword},
		{Class: 101, Hero: data.NewCampaignHero(data.SkillAxe), Weapon: &heroSword},
	})

	b, err := w.MarshalBinary()
	if err != nil {
		t.Fatalf("MarshalBinary: %v", err)
	}
	var back sim.World
	if err := back.UnmarshalBinary(b); err != nil {
		t.Fatalf("UnmarshalBinary: %v", err)
	}
	again, err := back.MarshalBinary()
	if err != nil {
		t.Fatalf("MarshalBinary (round two): %v", err)
	}
	if string(again) != string(b) {
		t.Fatal("the round trip did not reproduce the bytes")
	}
	if back.Hash() != w.Hash() {
		t.Fatalf("hash %x after the round trip, want %x", back.Hash(), w.Hash())
	}
}

// TestAPartyMembersOwnBookReachesHisEntity is 0127 FR-4a for the one unit a
// player actually commands. The placement path carried the mask from the first
// day of that story and this path did not, and the two are not interchangeable:
// mission ten places thirty-six units and not one of them is a caster, so a
// hero with no book meant the whole cast affordance drew an empty strip for the
// only unit the player can select. It was found by driving a real mission, not
// by a test, which is why there is one now.
//
// The value is the shipped one: PC_Fergard and PC_Reniesta both carry 266306,
// bits 1, 6, 12 and 18 — the cheapest spell of four of the five schools.
func TestAPartyMembersOwnBookReachesHisEntity(t *testing.T) {
	m := startMap(t, 40, 40, mapload.Cell{X: 17, Y: 20})
	party := []mapload.PartyMember{
		{Class: 100, KnownSpells: 266306, Hero: data.NewCampaignHero(data.SkillBlade)},
		// The second member states none, so this case also witnesses that the
		// field is read PER MEMBER rather than once for the party.
		{Class: 101, Hero: data.NewCampaignHero(data.SkillAxe)},
	}
	w, st := mustStart(t, m, party)

	byID := map[sim.EntityID]sim.Entity{}
	for _, e := range w.Entities() {
		byID[e.ID] = e
	}
	for i, want := range []uint32{266306, 0} {
		e, ok := byID[st.IDs[i]]
		if !ok {
			t.Fatalf("member %d was reported as entity %d, which the world does not hold", i, st.IDs[i])
		}
		if e.KnownSpells != want {
			t.Errorf("member %d's entity carries book %d, want %d", i, e.KnownSpells, want)
		}
	}
}

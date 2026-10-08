package mapload_test

import (
	"strings"
	"testing"

	"againrom/pkg/data"
	"againrom/pkg/formats/alm"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
)

// The Humans row slots this file writes. Transcribed from the row's own
// documented order rather than imported, for the reason every other slot number
// in this suite is: a test asserting that a column reached a field must not read
// that column's position out of the code it is testing.
const (
	slotHumanBody     = 0
	slotHumanReaction = 1
	slotHumanHealth   = 4
	slotHumanSpeed    = 6
	slotHumanSkill3   = 13
	slotHumanCharge   = 19
	slotHumanRelax    = 20
	humanRowWidth     = 26
)

// The values this file loads. NONE of them is a constructor default, and the
// cadence pair in particular is deliberately NOT 8 and 4 — those are also the
// pair a generated character with no weapon falls back to, so a fixture at 8/4
// could not tell "the template's own columns" from "the hero's bare pair".
const (
	srcHumanBody     = 40
	srcHumanReaction = 33
	srcHumanHealth   = 57
	srcHumanSpeed    = 19
	srcHumanSkill    = 25
	srcHumanCharge   = 21
	srcHumanRelax    = 6
)

// humanRow is a Humans row carrying those columns and leaving every other cell
// empty.
func humanRow(slots map[int]int32) []int32 {
	p := make([]int32, humanRowWidth)
	for i := range p {
		p[i] = -1
	}
	for s, v := range slots {
		p[s] = v
	}
	return p
}

// fullHumanRow is the row every test below places, keyed on type id 7.
func fullHumanRow() []int32 {
	return humanRow(map[int]int32{
		slotHumanType: 7,
		slotHumanBody: srcHumanBody, slotHumanReaction: srcHumanReaction,
		slotHumanHealth: srcHumanHealth, slotHumanSpeed: srcHumanSpeed,
		slotHumanSkill3: srcHumanSkill,
		slotHumanCharge: srcHumanCharge, slotHumanRelax: srcHumanRelax,
	})
}

// scaleEntry and scaleCollection are a shape or material table: a name and its
// nine doubles.
type scaleEntry struct {
	name    string
	doubles []float64
}

type scaleCollection []scaleEntry

func (c scaleCollection) Len() int                     { return len(c) }
func (c scaleCollection) EntryName(i int) string       { return c[i].name }
func (c scaleCollection) EntryDoubles(i int) []float64 { return c[i].doubles }

// identityScale is a one-entry scale table whose every factor is 1, so a weapon
// resolved through it carries its row's own columns and a test states one number
// rather than a product. Index 0 is the entry an unnamed prefix resolves through.
func identityScale() scaleCollection {
	return scaleCollection{{name: "Plain", doubles: []float64{1, 1, 1, 1, 1, 1, 1, 1, 1}}}
}

// The Weapons row slots this file writes, transcribed the same way.
const (
	slotWeaponType    = 5
	slotWeaponMin     = 6
	slotWeaponMax     = 7
	slotWeaponToHit   = 8
	slotWeaponDefence = 9
	slotWeaponCharge  = 0xc
	slotWeaponRelax   = 0xd
	weaponRowWidth    = 14
)

// weaponRow is a Weapons row of the given kind, damage pair, to-hit, defence
// and cadence.
func weaponRow(kind, min, max, toHit, defence, charge, relax int32) []int32 {
	p := make([]int32, weaponRowWidth)
	for i := range p {
		p[i] = -1
	}
	p[slotWeaponType] = kind
	p[slotWeaponMin], p[slotWeaponMax] = min, max
	p[slotWeaponToHit], p[slotWeaponDefence] = toHit, defence
	p[slotWeaponCharge], p[slotWeaponRelax] = charge, relax
	return p
}

// humanTable is a table resolving key 7 to one humans row carrying the given
// equipment names, with a weapon collection holding one melee sword and one
// ranged bow and a mail that is in neither.
func humanTable(equipment []string, weapons data.Collection,
	shapes, materials data.ScaleTable) *mapload.Table {

	return &mapload.Table{
		Units:  defCollection{{}},
		Humans: defCollection{{}, {name: "k7", params: fullHumanRow(), strings: equipment}},
		Shapes: shapes, Materials: materials, Weapons: weapons,
	}
}

// swordAndBow is the weapon collection every test below uses: a melee sword and
// a ranged bow, which is what makes "the search passes a ranged cell over"
// testable at all.
func swordAndBow() defCollection {
	return defCollection{
		{},
		{name: "Sword", params: weaponRow(data.SkillBludgen, 5, 8, 7, 2, 9, 4)},
		{name: "Bow", params: weaponRow(0xb, 30, 40, 100, 0, 3, 3)},
		{name: "Stub", params: weaponRow(data.SkillBlade, 1, 1, 0, 0, -1, -1)},
	}
}

// humanMap places one unit on key 7, below the class-key floor, so it takes the
// humans band by its type id.
func humanMap() *alm.Map {
	return &alm.Map{Width: 40, Height: 40,
		Units: []alm.Unit{{X: 0x0C80, Y: 0x0C80, ClassID: 7}}}
}

// loadHuman builds the one-placement world and returns its only entity.
func loadHuman(t *testing.T, tbl *mapload.Table, diff mapload.Difficulty) sim.Entity {
	t.Helper()
	w, err := mapload.FromALMWith(humanMap(), tbl, diff)
	if err != nil {
		t.Fatalf("FromALMWith: %v", err)
	}
	ents := w.Entities()
	if len(ents) != 1 {
		t.Fatalf("the fixture built %d entities, want 1", len(ents))
	}
	return ents[0]
}

// wantHumanCombat is what the definition tier says this fixture's row is worth,
// bare or armed. It asks the tier rather than restating the graph: this file's
// question is whether the LOADER delivers those numbers, and a second copy of
// the arithmetic here would be asserting one transcription against another.
func wantHumanCombat(t *testing.T, w *data.Weapon) data.Combat {
	t.Helper()
	d, err := data.NewHumanDef("k7", fullHumanRow())
	if err != nil {
		t.Fatalf("NewHumanDef: %v", err)
	}
	return d.Combat(w)
}

func wantHumanHealth(t *testing.T) int32 {
	t.Helper()
	d, err := data.NewHumanDef("k7", fullHumanRow())
	if err != nil {
		t.Fatalf("NewHumanDef: %v", err)
	}
	return d.DerivedMaximum()
}

// eightOfCombat renders a combat value in the shape eightOf renders an entity,
// so the two can be compared field for field.
func eightOfCombat(c data.Combat) eight {
	return eight{charge: c.AttackChargeTime, relax: c.AttackRelaxTime,
		toHit: c.ToHit, defence: c.Defence, absorb: c.Absorption,
		dmgBase: c.DamageBase, dmgSpread: c.DamageSpread, alwaysHits: c.AlwaysHits}
}

func TestAPlacedPersonCarriesHisDerivedHealthRateAndCadence(t *testing.T) {
	t.Parallel()

	e := loadHuman(t, humanTable(nil, nil, nil, nil), mapload.DifficultyNormal)
	wantHealth := wantHumanHealth(t)
	if e.HP != wantHealth || e.MaxHP != wantHealth {
		t.Errorf("health %d/%d, want the derived maximum %d", e.HP, e.MaxHP, wantHealth)
	}
	if wantHealth == srcHumanHealth {
		t.Fatal("the fixture's derived maximum collides with the row's own health column, so this test cannot tell a derived placement from a column-fed one")
	}
	if e.Speed != srcHumanSpeed {
		t.Errorf("rate %d, want the row's own %d", e.Speed, srcHumanSpeed)
	}
	if e.AttackCharge != srcHumanCharge || e.AttackRelax != srcHumanRelax {
		t.Errorf("cadence %d/%d, want the row's own %d/%d",
			e.AttackCharge, e.AttackRelax, srcHumanCharge, srcHumanRelax)
	}
	if e.AttackCharge == data.BareChargeTime && e.AttackRelax == data.BareRelaxTime {
		t.Error("the fixture's cadence collides with a bare character's, so this test proves nothing")
	}
	if got, want := eightOf(e), eightOfCombat(wantHumanCombat(t, nil)); got != want {
		t.Errorf("the eight are %+v, want the definition's %+v", got, want)
	}
	// And they are not the constructor's zeros, which is the defect this story
	// exists to remove.
	if e.DamageBase == 0 && e.DamageSpread == 0 && e.ToHit == 0 {
		t.Error("the placement carries no blow at all")
	}
}

func TestThePersonsWeaponIsCellZeroAndNoOtherCell(t *testing.T) {
	t.Parallel()

	sword, err := data.ResolveWeapon("Sword", identityScale(), identityScale(), swordAndBow())
	if err != nil {
		t.Fatalf("ResolveWeapon: %v", err)
	}
	bow, err := data.ResolveWeapon("Bow", identityScale(), identityScale(), swordAndBow())
	if err != nil {
		t.Fatalf("ResolveWeapon: %v", err)
	}

	for _, tc := range []struct {
		name      string
		equipment []string
		want      *data.Weapon
	}{
		{"the weapon alone", []string{"Sword"}, &sword},
		{"an armour name in cell 0 arms nobody, even with a real weapon behind it", []string{"Mail", "Boots", "Sword"}, nil},
		{"a ranged cell resolves before the melee one", []string{"Bow", "Sword"}, &bow},
		{"nothing that resolves", []string{"Mail", "Boots"}, nil},
		{"a ranged weapon alone now resolves too", []string{"Bow"}, &bow},
		{"empty cells only", []string{"", "", ""}, nil},
		{"no cells at all", nil, nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			tbl := humanTable(tc.equipment, swordAndBow(), identityScale(), identityScale())
			e := loadHuman(t, tbl, mapload.DifficultyNormal)
			if got, want := eightOf(e), eightOfCombat(wantHumanCombat(t, tc.want)); got != want {
				t.Errorf("the eight are %+v, want %+v", got, want)
			}
		})
	}

	// The sword, the bow and the bare answers must actually differ pairwise,
	// or some case above passes for the wrong reason.
	swordCombat := eightOfCombat(wantHumanCombat(t, &sword))
	bowCombat := eightOfCombat(wantHumanCombat(t, &bow))
	bareCombat := eightOfCombat(wantHumanCombat(t, nil))
	if swordCombat == bareCombat || bowCombat == bareCombat || swordCombat == bowCombat {
		t.Fatalf("the fixture's weapons do not tell apart: sword %+v, bow %+v, bare %+v",
			swordCombat, bowCombat, bareCombat)
	}
}

// TestAnEmptyWeaponCadenceCellLeavesTheTemplatesStanding is AC-9's armed half at
// the loader: the "Stub" row states no cadence, so the person keeps his own.
func TestAnEmptyWeaponCadenceCellLeavesTheTemplatesStanding(t *testing.T) {
	t.Parallel()

	e := loadHuman(t, humanTable([]string{"Stub"}, swordAndBow(), identityScale(), identityScale()),
		mapload.DifficultyNormal)
	if e.AttackCharge != srcHumanCharge || e.AttackRelax != srcHumanRelax {
		t.Errorf("cadence %d/%d, want the template's own %d/%d — the weapon states neither",
			e.AttackCharge, e.AttackRelax, srcHumanCharge, srcHumanRelax)
	}
	// It IS armed, so the empty cadence is the only thing that fell through.
	if e.DamageBase == 0 {
		t.Error("the placement is not armed at all, so the cadence proves nothing")
	}
}

func TestAPlacedPersonsWeaponSpellReachesHisEntity(t *testing.T) {
	t.Parallel()

	weapons := defCollection{
		{},
		{name: "Sword", params: weaponRow(data.SkillBludgen, 5, 8, 7, 2, 9, 4)},
		{name: "Staff", params: weaponRow(data.SkillBlade, 1, 1, 0, 0, -1, -1)},
	}
	spells := defCollection{
		{}, // 0: reserved
		{name: "Fire Arrow", params: spellRow(3, 1, 1, 7, 4, 8, 0)},
	}

	tbl := humanTable([]string{"Staff{castSpell=Fire_Arrow:10}"}, weapons, identityScale(), identityScale())
	tbl.Spells = spells
	e := loadHuman(t, tbl, mapload.DifficultyNormal)
	if e.WeaponSpell != 1 {
		t.Errorf("WeaponSpell = %d, want 1 (the resolved Fire Arrow id)", e.WeaponSpell)
	}
	if e.WeaponSpellLevel != 10 {
		t.Errorf("WeaponSpellLevel = %d, want 10 (the attachment's own level)", e.WeaponSpellLevel)
	}
	if e.DamageBase == 0 {
		t.Error("the placement is not armed at all, so the spell above proves nothing")
	}

	bare := humanTable([]string{"Staff"}, weapons, identityScale(), identityScale())
	bare.Spells = spells
	if e := loadHuman(t, bare, mapload.DifficultyNormal); e.WeaponSpell != 0 || e.WeaponSpellLevel != 0 {
		t.Errorf("weapon spell = (%d, %d) for the same weapon named bare, want (0, 0)",
			e.WeaponSpell, e.WeaponSpellLevel)
	}
}

// TestATableMissingAnyItemCollectionArmsNobody is AC-8.
//
// Each of the three is withheld ON ITS OWN, not only all three together, because
// the interesting cases are the two that a missing-scale-table identity factor
// would silently arm with an unscaled weapon.
func TestATableMissingAnyItemCollectionArmsNobody(t *testing.T) {
	t.Parallel()

	bare := eightOfCombat(wantHumanCombat(t, nil))
	for _, tc := range []struct {
		name              string
		shapes, materials data.ScaleTable
		weapons           data.Collection
	}{
		{"no shapes", nil, identityScale(), swordAndBow()},
		{"no materials", identityScale(), nil, swordAndBow()},
		{"no weapons", identityScale(), identityScale(), nil},
		{"none of the three", nil, nil, nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			tbl := humanTable([]string{"Sword"}, tc.weapons, tc.shapes, tc.materials)
			if got := eightOf(loadHuman(t, tbl, mapload.DifficultyNormal)); got != bare {
				t.Errorf("the eight are %+v, want the bare %+v", got, bare)
			}
		})
	}
}

func TestTheDifficultyDoesNotReachAPerson(t *testing.T) {
	t.Parallel()

	tbl := humanTable([]string{"Sword"}, swordAndBow(), identityScale(), identityScale())
	var first sim.Entity
	for i, diff := range []mapload.Difficulty{
		mapload.DifficultyEasy, mapload.DifficultyNormal, mapload.DifficultyHard,
	} {
		e := loadHuman(t, tbl, diff)
		if i == 0 {
			first = e
			continue
		}
		if e != first {
			t.Errorf("difficulty %d builds %+v, want the same entity every setting builds: %+v",
				int32(diff), e, first)
		}
	}
	if want := wantHumanHealth(t); first.HP != want {
		t.Errorf("health %d, want the unadjusted derived maximum %d", first.HP, want)
	}
}

func TestAWorldRefusesAPersonRowItCannotStream(t *testing.T) {
	t.Parallel()

	tbl := &mapload.Table{
		Units:  defCollection{{}},
		Humans: defCollection{{}, {name: "k7", params: fullHumanRow()[:humanRowWidth-1]}},
	}
	w, err := mapload.FromALMWith(humanMap(), tbl, mapload.DifficultyNormal)
	if err == nil {
		t.Fatal("a world was built over a person row too short to stream")
	}
	if w != nil {
		t.Error("a refused build returned a world beside its error")
	}
	if !strings.Contains(err.Error(), "k7") {
		t.Errorf("error %q does not name the offending row", err)
	}
}

// TestAWorldHoldingAnArmedPersonRoundTrips is AC-13's half that this package can
// witness: the numbers a person now carries survive the byte form.
func TestAWorldHoldingAnArmedPersonRoundTrips(t *testing.T) {
	t.Parallel()

	tbl := humanTable([]string{"Sword"}, swordAndBow(), identityScale(), identityScale())
	w, err := mapload.FromALMWith(humanMap(), tbl, mapload.DifficultyNormal)
	if err != nil {
		t.Fatalf("FromALMWith: %v", err)
	}
	form, err := w.MarshalBinary()
	if err != nil {
		t.Fatalf("MarshalBinary: %v", err)
	}
	var back sim.World
	if err := back.UnmarshalBinary(form); err != nil {
		t.Fatalf("UnmarshalBinary: %v", err)
	}
	if back.Hash() != w.Hash() {
		t.Errorf("the world hashes %#016x through its bytes, want %#016x", back.Hash(), w.Hash())
	}
	if got, want := eightOf(back.Entities()[0]), eightOf(w.Entities()[0]); got != want {
		t.Errorf("the eight came back as %+v, want %+v", got, want)
	}
}

// TestArmingAPersonMovesTheDigest is AC-12a: the digest this story is FOR moves,
// and it moves only because of the fields this story fills.
//
// It is asserted as a DIFFERENCE against the same world built with no item
// collections, and then as an equality once the filled fields are put back — so
// it is a second derivation rather than a number pasted out of a run, which
// could only ever say the loader agrees with itself.
func TestArmingAPersonMovesTheDigest(t *testing.T) {
	t.Parallel()

	armed, err := mapload.FromALMWith(humanMap(),
		humanTable([]string{"Sword"}, swordAndBow(), identityScale(), identityScale()),
		mapload.DifficultyNormal)
	if err != nil {
		t.Fatalf("armed: %v", err)
	}
	bare, err := mapload.FromALMWith(humanMap(), humanTable([]string{"Sword"}, nil, nil, nil),
		mapload.DifficultyNormal)
	if err != nil {
		t.Fatalf("bare: %v", err)
	}
	if armed.Hash() == bare.Hash() {
		t.Fatal("arming the person moved no digest at all")
	}

	// Put the four numbers a weapon moves back to the bare person's, and the two
	// worlds must become one. Anything else this story touched would survive
	// this and fail here.
	a, b := armed.Entities()[0], bare.Entities()[0]
	a.DamageBase, a.DamageSpread = b.DamageBase, b.DamageSpread
	a.ToHit, a.Defence = b.ToHit, b.Defence
	a.AttackCharge, a.AttackRelax = b.AttackCharge, b.AttackRelax
	// AND THE CREDITED SLOT: it rides on Combat.SkillSlot, a function of the
	// weapon alone exactly as Reach already is, so arming this person with a
	// melee weapon moves it off SkillGeneral the same way it moves the weapon's
	// own four. Put back to the bare person's for the same reason those four
	// are.
	a.XPSlot = b.XPSlot
	assertInitialModifier(t, a, [64]byte{18: 7, 32: 5, 33: 3, 42: 2})
	assertInitialModifier(t, b, [64]byte{})
	a.NativeBasis.Modifier = b.NativeBasis.Modifier
	if a != b {
		t.Errorf("the two worlds differ outside the weapon's own four:\n got %+v\nwant %+v", a, b)
	}
}

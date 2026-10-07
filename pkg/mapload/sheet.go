package mapload

import (
	"againrom/pkg/data"
	"againrom/pkg/formats/alm"
	"againrom/pkg/sim"
)

type Band uint8

const (
	// BandCreature is a placement Resolve took down ArmUnits: its Sheet is
	// stated off the units collection alone.
	BandCreature Band = iota
	// BandPerson is a placement Resolve took down any of the three humans rungs
	// — ArmNPC, ArmServerID, ArmHumansByType — and found an entry through:
	// its Sheet is stated off the humans collection. A party member is a person
	// for every purpose of this document (spec Vocabulary), which is why the
	// driver tier gives the party's own path this same value rather than a
	// value of its own (0137 T3).
	BandPerson
)

// Sheet is everything one placement's character sheet states (spec
// Vocabulary "Sheet", narrowed to the part this tier can state): the band,
// the four statistics, the six skill positions, the elemental family, the
// weapon-kind family, and the experience.
//
// IT IS THIS TIER'S OWN TYPE, not the window's. pkg/mapload sits below
// pkg/ui in the import DAG and must not name a type from it, so this is a
// plain value of builtins and fixed-size arrays — comparable, copied by
// assignment — and the driver that turns it into a window value (0137 T3)
// copies it field by field, the one place the two vocabularies meet.
//
// A POSITION THIS TYPE DOES NOT BACK WITH A COLUMN OR A DERIVATION IS LEFT
// AT ITS ZERO VALUE AND MUST NOT BE READ: a creature's General skill
// position (index 0) and a creature's Experience are both such positions. No
// sentinel is written for either — a sentinel is a number, and this
// contract forbids showing one where nothing is stated — so Band is the
// only thing that tells a reader which fields of an otherwise
// fully-populated Sheet to trust. A reader that does not check Band first
// and reads Skill[0] or Experience off a creature's Sheet reads a fact this
// tier never claimed.
type Sheet struct {
	Band Band
	// Mage selects the spell-school labels for the same six stored skill
	// slots. It is a presentation fact of a person; creatures leave it false.
	Mage bool

	Body, Reaction, Mind, Spirit int32

	// Skill is the six skill positions (spec Vocabulary "Skill positions"):
	// index 0 General, 1..5 Blade, Axe, Bludgeon, Pike, Shooting.
	//
	// FOR A PERSON, all six are the row's OWN levels — General included —
	// and NOT the derived-stat graph's restored, clamped ones: the graph's own
	// Skill field has already run each of 1..5 through the loadout's bonus and
	// the [SkillFloor, SkillCap] clamp (recompute.go step 6a) before Recompute
	// returns, and reading that field here would state a level nobody trained.
	// The party path already keeps this same rule for a started member
	// (pkg/game/world.go partyCharacters reads p.Hero.Skill, not the
	// recompute's Skill); this is that rule restated for a placed person.
	//
	// FOR A CREATURE (FR-4a), 1..5 hold the SAME five numbers WeaponKind holds,
	// Blade..Shooting order — copied at build time (creatureSheet, below)
	// rather than resolved a second way at read time, so a reader consulting
	// Skill and a reader consulting WeaponKind can never come to disagree.
	// Position 0 is UNSTATED: a creature has no skill level of any kind and the
	// units collection states none, so it is left at the zero value and Band is
	// what says not to read it.
	Skill [data.SkillSlots]int32

	// Elemental is the five-wide family keyed by Fire, Water, Air, Earth,
	// Astral: a creature's UnitDef.Protection, unchanged, or a person's derived
	// Protection, off the same recompute Skill above is silent about.
	Elemental [5]int32

	// WeaponKind is the five-wide family keyed by Blade, Axe, Bludgeon, Pike,
	// Shooting, in that order: a creature's UnitDef.Resistance, unchanged, or a
	// person's derived Resistance. It is the SAME array Skill's positions 1..5
	// copy from on the creature band (FR-4a) — this field's own source is
	// untouched by that copy.
	WeaponKind [5]int32

	// Experience is the graph's own total for a person: the sum of the six
	// per-slot experiences the row's OWN levels — the same ones Skill above
	// states, not the restored ones — account for. For a creature it is
	// UNSTATED: the units collection carries no per-slot experience column to
	// read one from, so this is left at the zero value and, like Skill[0]
	// above, Band alone says not to trust it.
	Experience int32

	// Sight is how far the unit sees, in cells (0140). The two bands fill it
	// from DIFFERENT SOURCES and the difference is the point: a person's is
	// the derived graph's own Sight, computed from Mind and Reaction like
	// every other number this sheet takes from that graph; a creature's is
	// the units collection's own scan-range column, which the graph never
	// touches for it.
	//
	// THAT THE SCAN-RANGE COLUMN IS WHAT A SHEET WOULD CALL "SIGHT" IS OURS,
	// not decoded. `UNIT-PANEL-010` establishes that the original's display
	// copies a sight value out of the actor, and `UNIT-PANEL-011` establishes
	// that which cached value appears where cannot be settled at all; neither
	// says this column is that value for a creature. It is the only scan
	// radius the collection carries and it is presented as such.
	Sight int32
}

// PlacedSheets is the Sheet every placement of m states, keyed by the entity
// id the world builder mints for it.
//
// THE KEY IS THE MINTED ID, on the same ground pkg/game/tiers.go's
// entityTiers already states it from: the i-th unit record takes id i,
// counting from zero, and this lookup and the world it describes are built
// from the same decoded map in the same call (R-3). A lookup keyed by loop
// index that happened to agree with the world builder would be the same
// numbers arrived at by an assumption instead of by the contract the world
// builder itself documents.
//
// EACH PLACEMENT IS RESOLVED WITH THIS PACKAGE'S OWN Resolve (spec
// Vocabulary "Band"; task T1): the units arm, found, yields a creature; any
// other arm, found, yields a person, because every one of the three humans
// rungs answers the same collection and a party member is a person for every
// purpose of this document; an arm that found nothing yields no entry at all
// — the placement is simply absent from the returned map, never a Sheet of
// zeroes.
//
// IT IS TOTAL AND REPORTS NOTHING. A nil map, a nil table, and a table
// carrying neither collection are all "no source": Resolve itself is
// nil-safe on a nil *Table (its own doc says so, and
// TestResolveWithNoTableStillNamesTheArm pins it), so every placement comes
// back not-found and the returned map holds no entry for it — no separate
// check is needed here to reach the same answer entityTiers reaches by
// checking t.Units up front. A resolved entry this tier's own definition
// contract cannot read (NewUnitDef or NewHumanDef refusing the row) is
// answered the same way: the world beside this call has already failed on
// that entry if it is fatal, and a caller asking only for a readout gets no
// entry rather than an error (the same choice entityTiers already makes).
//
// IT IS A PURE FUNCTION OF THE MAP AND THE TABLE. It reads no world, no
// entity and no clock, so two calls over the same map and table state the
// same sheets, and a readout built twice cannot state two sheets for one
// unit.
func PlacedSheets(m *alm.Map, t *Table) map[sim.EntityID]Sheet {
	if m == nil {
		return nil
	}
	out := make(map[sim.EntityID]Sheet, len(m.Units))
	for i, u := range m.Units {
		r := Resolve(u, t)
		if !r.Found() {
			continue
		}
		var (
			s  Sheet
			ok bool
		)
		if r.Arm == ArmUnits {
			s, ok = creatureSheet(t.units(), r.Index)
		} else {
			s, ok = personSheet(t.humans(), r.Index)
		}
		if ok {
			out[sim.EntityID(i)] = s
		}
	}
	return out
}

// creatureSheet is what a units-band entry states: its four statistics and
// both families, each read straight off the row's own columns and none of
// them derived — a non-hero has no derived-stat graph to run at all
// (data.UnitDef.Combat's own doc says so, and this sheet reads exactly the
// definition that doc describes).
func creatureSheet(c data.Collection, index int) (Sheet, bool) {
	d, err := data.NewUnitDef(c.EntryName(index), c.EntryParams(index))
	if err != nil {
		return Sheet{}, false
	}
	s := Sheet{
		Band:       BandCreature,
		Body:       d.Body,
		Reaction:   d.Reaction,
		Mind:       d.Mind,
		Spirit:     d.Spirit,
		Elemental:  d.Protection,
		WeaponKind: d.Resistance,
	}
	s.Skill[data.SkillBlade] = d.Resistance[0]
	s.Skill[data.SkillAxe] = d.Resistance[1]
	s.Skill[data.SkillBludgen] = d.Resistance[2]
	s.Skill[data.SkillPike] = d.Resistance[3]
	s.Skill[data.SkillShoot] = d.Resistance[4]
	s.Sight = d.ScanRange
	return s, true
}

// personSheet is what a humans-band entry states, off the same derived-stat
// graph the party path already runs for a started member (pkg/game/world.go
// partyCharacters).
func personSheet(c data.Collection, index int) (Sheet, bool) {
	d, err := data.NewHumanDef(c.EntryName(index), c.EntryParams(index))
	if err != nil {
		return Sheet{}, false
	}
	derived := d.Hero().Recompute(d.Profile(), data.Loadout{})
	return Sheet{
		Band:       BandPerson,
		Mage:       d.Profile().ManaColumn,
		Body:       derived.Body,
		Reaction:   derived.Reaction,
		Mind:       derived.Mind,
		Spirit:     derived.Spirit,
		Skill:      d.Skill,
		Elemental:  derived.Protection,
		WeaponKind: derived.Resistance,
		Experience: derived.Experience,
		Sight:      derived.Sight,
	}, true
}

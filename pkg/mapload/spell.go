package mapload

import (
	"strings"

	"againrom/pkg/data"
	"againrom/pkg/mod"
	"againrom/pkg/sim"
)

// unitSpellbook constructs the live book and the class spell slots from the
// three authored Spell and Probability cell pairs. The first cell controls
// presence; later cells cannot create a book alone. Each slot is kept only for
// a spell the book holds.
func unitSpellbook(params []int32, t *Table, typeID int32) (uint32, sim.Spellbook, [sim.CreatureSpellSlots]sim.CreatureSpell) {
	var slots [sim.CreatureSpellSlots]sim.CreatureSpell
	const firstSpell = 48
	book := sim.Spellbook{}
	if typeID == 0x47 || typeID == 0x48 { // UNIT-SPELL-007
		book.State = sim.BookPresent
	}
	if len(params) <= firstSpell || params[firstSpell] <= 0 {
		return 0, book, slots
	}
	book.State = sim.BookPresent
	rules := SpellRules(t)
	var known uint32
	for cell := firstSpell; cell < firstSpell+6 && cell < len(params); cell += 2 {
		id := params[cell]
		if id <= 0 {
			continue
		}
		if id > 28 || int(id) > len(rules) || rules[id-1].ID != uint16(id) {
			continue
		}
		book.Slots[id-1] = bookSpell(rules[id-1])
		known |= uint32(1) << uint(id)
		if cell+1 < len(params) {
			slots[(cell-firstSpell)/2] = sim.CreatureSpellFor(id, params[cell+1])
		}
	}
	return known, book, slots
}

// CreatureBookRow is one class's fixed book: a spawn of TypeID, and of Face
// when Face is nonzero, inserts Spells after its class cells and then writes
// Skill (general, fire, water, air, earth, astral); -1 leaves a word as it is.
type CreatureBookRow struct {
	TypeID, Face int32
	Spells       []int32
	Skill        [6]int32
}

// CreatureBooks is a game's creature book source: a spawn's class cells, then
// the first row its class and face match.
type CreatureBooks struct {
	Rows []CreatureBookRow
}

var (
	// FirstGameCreatureBooks: Dragon by face 1..4 and Daemon (UNIT-SPELL-007;
	// observed in every original save of these classes).
	FirstGameCreatureBooks = &CreatureBooks{Rows: []CreatureBookRow{
		{0x47, 1, []int32{2, 3, 13}, [6]int32{-1, 30, -1, 30, -1, -1}},
		{0x47, 2, []int32{2, 3, 13, 11, 27}, [6]int32{-1, 50, 60, 50, -1, 20}},
		{0x47, 3, []int32{2, 8, 11, 14, 27}, [6]int32{-1, 50, 50, 50, -1, 50}},
		{0x47, 4, []int32{2, 8, 11, 14, 17, 20}, [6]int32{-1, 90, 90, 90, 0, -1}},
		{0x48, 0, []int32{2, 8, 11, 14, 17, 20}, [6]int32{-1, 99, 99, 99, 20, -1}},
	}}
	// ClassCellBooks: the class cells alone; no second-game claim states a
	// fixed book.
	ClassCellBooks = &CreatureBooks{}
)

// creatureBooks is t's book source; nil is the first game's.
func (t *Table) creatureBooks() *CreatureBooks {
	if t == nil || t.CreatureBooks == nil {
		return FirstGameCreatureBooks
	}
	return t.CreatureBooks
}

func (b *CreatureBooks) rowFor(typeID, face int32) (CreatureBookRow, bool) {
	for _, r := range b.Rows {
		if r.TypeID == typeID && (r.Face == 0 || r.Face == face) {
			return r, true
		}
	}
	return CreatureBookRow{}, false
}

// add inserts the matched row's spells into book and writes its skill words.
// Each entry is the installed spell row, as a class cell spell is.
func (b *CreatureBooks) add(typeID, face int32, t *Table, known *uint32, book *sim.Spellbook, skill *[data.SkillSlots]int32) {
	fixed, ok := b.rowFor(typeID, face)
	if !ok {
		return
	}
	rules := SpellRules(t)
	for _, id := range fixed.Spells {
		if int(id) > len(rules) || rules[id-1].ID != uint16(id) {
			continue
		}
		book.Slots[id-1] = bookSpell(rules[id-1])
		*known |= uint32(1) << uint(id)
	}
	for i, v := range fixed.Skill {
		if v >= 0 && i < len(skill) {
			skill[i] = v
		}
	}
}

func bookSpell(rule sim.SpellRule) sim.BookSpell {
	entry := sim.BookSpell{Range: rule.MaxRange, ManaCost: uint16(rule.ManaCost)}
	if rule.Defensive {
		entry.Defensive = 1
	}
	return entry
}

// spellRules converts pkg/data's own loaded spell rows into pkg/sim's
// stdlib-only SpellRule record. It is sited HERE, in pkg/mapload, because
// pkg/sim may not import pkg/data (the determinism wall, AGENTS.md) and
// pkg/data may not import pkg/sim (the DAG's own other direction) —
// pkg/mapload is the one package this tree has that imports both, so the one
// place a []data.Spell becomes a []sim.SpellRule is here.
//
// ID IS THE SLICE'S OWN POSITION, one-based. data.LoadSpells walks a
// collection's entries 1..n-1 in order and returns one Spell per entry, with
// no id field of its own — the row's id is its own subscript, and that
// subscript is exactly the returned slice's index plus one, which is
// LoadSpells' own doc and not a fact restated here.
//
// EVERY OTHER FIELD IS A STRAIGHT CARRY, narrowed to SpellRule's own width
// where data.Spell carries a wider one — School and MaxRange are int32 in
// pkg/data and uint8 in pkg/sim, the byte form's own width (0127 T4) — on
// sightOf's and reachOf's own precedent (fromalm.go): the narrowing is
// applied here rather than refused, because no shipped spell's School or
// MaxRange approaches 256 and a refusal over that would turn every spell
// this build has ever loaded red for a reason a reader could learn nothing
// from.
func spellRules(spells []data.Spell, second bool) []sim.SpellRule {
	out := make([]sim.SpellRule, len(spells))
	for i, sp := range spells {
		out[i] = sim.SpellRule{
			ID:           uint16(i + 1),
			Delivery:     sp.Delivery,
			EffectSpeed:  sp.EffectSpeed,
			Complication: uint8(sp.Complication),
			ManaCost:     sp.ManaCost,
			School:       uint8(sp.School),
			MaxRange:     uint8(sp.MaxRange),
			DamageMin:    sp.DamageMin,
			DamageMax:    sp.DamageMax,
			TargetsUnit:  sp.TargetsUnit,
			Damaging:     sp.Damaging,
			Defensive:    sp.Defensive,
			Restorative:  sp.Restorative,
			// The two shape columns, carried through unaltered: pkg/data reads them
			// off the row and this function is the one place they cross into pkg/sim.
			Area:            sp.Area,
			Distribution:    sp.Distribution,
			Radius:          sp.Radius,
			AreaDuration:    sp.AreaDuration,
			SpellDuration:   sp.SpellDuration,
			EffectKind:      sim.EffectKind(sp.Effect.Kind),
			EffectMode:      sim.EffectMode(sp.Effect.Mode),
			EffectMagnitude: sp.Effect.Magnitude,
			EffectDuration:  sp.Effect.Duration,
		}
		// MAGIC-397: ROM1 numerical modes store low16(16*N).
		if !second && sp.Effect.Mode&(data.SpellEffectDuration|data.SpellEffectContinuous|data.SpellEffectCharges) != 0 {
			out[i].EffectDuration = uint16(uint32(sp.Effect.Duration) << 4)
		}
	}
	return out
}

// SecondGameSpellArms gives each second-game row its arm and classifies its
// damage pair by that arm: the second game's Heal and Drain Life are rows 24
// and 26, not the first game's 6 and 11 (R2-ENGINE-019, R2-ENGINE-024).
func SecondGameSpellArms(rules []sim.SpellRule) {
	sim.AssignSecondGameArms(rules)
	for i := range rules {
		arm := int(rules[i].Arm)
		if rules[i].Arm == sim.ArmNone {
			arm = 0
		}
		rules[i].Damaging, rules[i].Restorative = data.ClassifySpell(arm, rules[i].DamageMin, rules[i].DamageMax)
	}
}

// spellsFor is the world's own spell table, resolved off t's Spells
// collection: the one call FromALMWith makes to decide what
// sim.NewStockedSpelledWorld's own last argument is.
func spellsFor(t *Table) []sim.SpellRule {
	return SpellRules(t)
}

// SpellRules exposes the exact installed spell table handed to a World. The
// character-generation preview needs the same rows before a mission world
// exists; keeping the conversion here prevents that presentation path from
// growing a second interpretation of pkg/data's spell columns.
func SpellRules(t *Table) []sim.SpellRule {
	if t == nil {
		return nil
	}
	spells, err := data.LoadSpells(t.spells())
	if err != nil {
		return nil
	}
	maxDamage := int64(mod.SpellMaxDamage)
	if t.SpellArms != nil {
		maxDamage = 255
	}
	applyModSpells(spells, t.Mods.Spells, maxDamage)
	rules := spellRules(spells, t.SpellArms != nil)
	if t.SpellArms != nil {
		t.SpellArms(rules)
	}
	applyModSpellTargets(rules, spells, t.Mods.Spells)
	return rules
}

// SpellIDByToken is FR-1b's lookup half: TOKEN (a Weapon's or a Combat's own
// SpellName, underscores intact) resolved to a Spells row's own id — that
// row's subscript, spellRules' own ID above restated for a caller that has
// not built a []sim.SpellRule at all. The match is against the row's own
// NAME with every `_` replaced by a space, exact after that substitution,
// and where two rows carry one name the FIRST wins, ascending from index 1 —
// FindHumanByName's own tie-break (pkg/data/defsearch.go), findByName's own
// tie-break (pkg/data/weapon.go), restated for the one collection this
// package reads once (spellsFor, above) rather than searches: every name
// search this build makes agrees on what "first" means.
//
// IT TAKES A *Table, NOT A data.Collection, on t.spells()'s own nil-safety:
// every caller of this function already holds the *Table a placement or a
// re-arm resolved against, and a second nil check at each of those call
// sites would be this file's own rule restated three times instead of once.
//
// A nil Table, on Table's own rule for every other search here, is "no
// table": nothing resolves and every token answers (0, false) — the same
// answer an empty token, or one matching no row, already gives. FR-1a's own
// zero pair (a weapon carrying no spell) reaches this function as the empty
// string, which is refused before the walk for the same reason
// FindHumanByName refuses it: an entry whose own name is empty cannot match
// even an empty key.
func SpellIDByToken(t *Table, token string) (uint16, bool) {
	spells := t.spells()
	if spells == nil || token == "" {
		return 0, false
	}
	name := strings.ReplaceAll(token, "_", " ")
	for i := 1; i < spells.Len(); i++ {
		if n := spells.EntryName(i); n != "" && n == name {
			return uint16(i), true
		}
	}
	return 0, false
}

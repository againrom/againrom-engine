package mapload

import (
	"fmt"

	"againrom/pkg/data"
	"againrom/pkg/formats/alm"
	"againrom/pkg/sim"
)

// Seed is the seed every world FromALM builds is constructed with.
//
// It is a fixed named constant because a world's RNG state must follow from a
// deterministic input alone: this loader takes no seed argument and reads no
// clock, no environment and no file, so one map yields one world in every
// process and on every machine.
//
// The value is ours and arbitrary — the ASCII of "map_load". Nothing here
// recovers the original engine's generator, no research claim describes one, and
// a digest pinned anywhere for a world this loader built is a pin on our own
// choice of seed and generator, never on anything game-derived.
const Seed uint64 = 0x6D61705F6C6F6164

// SpawnHP is the health and the health maximum every unit a map places is born
// with. One constant serves both fields, so a spawned unit begins at full health
// by construction rather than by two numbers someone keeps equal.
//
// It lives at the SPAWN SITE and nowhere else. The world constructor is given no
// default and no parameter, so a world assembled any other way keeps exactly the
// pair it was handed — which is what leaves every hand-built world at 0/0, alive
// with no health system, and its state and its digest where they were.
//
// The value is ours and PROVISIONAL, in the way the seed above is ours. It is
// what a placement that resolves to no unit definition is born with — a
// placement given no table, one taking the npc or the humans arm, and one whose
// key names nothing — so it is still the health of every unit on a map built the
// single-argument way, and of roughly a fifth of the placements on one built
// with a table.
const SpawnHP int32 = 100

// DefaultSpeed is the rate every placement that resolves to no unit definition
// is born with — a placement given no table, one taking the npc or the humans
// arm, and one whose key names nothing.
//
// Unlike SpawnHP it is NOT ours: it is the base constructor's own value, the one
// the original leaves standing for every empty cell of the column, and it is
// taken from the definition tier's own defaults rather than written out a second
// time here, so the two cannot come to disagree.
//
// It is now ONE FIELD OF A VALUE THE LOADER TAKES WHOLE rather than the single
// default that arm reaches for. An unresolved placement is handed the whole
// constructor definition and every number it carries — the rate, the cadence
// pair, and the six a blow reads — comes off that one value, so this name is a
// view onto what the load already does and not a second statement of it. It
// stays exported because what an unresolved placement is rated at is a fact
// callers and tests assert against, and because a reader looking for that number
// looks for a name.
//
// It exists because the alternative is worse in a way a reader would not see. A
// placement left at a speed of zero is UNRATED, a cell a tick — sixteen times
// what a shipped class does — so on a map where most placements take the humans
// arm the unresolved units would be the fast ones. A hero's speed is derived
// from Reaction on an arm this tree does not model, which is why that number is
// not available to take instead.
var DefaultSpeed = data.UnitDefaults().Speed

// riderBonus is a placed person's rider speed term.
func riderBonus(typeID int32) int32 {
	if data.RiderTypeID(typeID) {
		return data.RiderSpeedBonus
	}
	return 0
}

// unitCell is the cell a placed unit stands on, and the ONLY conversion from
// a unit record to a cell in this package.
//
// A unit's position is fixed point in 1/256 of a cell, so the cell it names is
// the position shifted right by 8 and the fraction below that is dropped, never
// rounded: a unit sitting at the centre of its cell and a unit sitting at the
// corner load onto the same cell. This story has no sub-cell position to keep
// the fraction in.
//
// The shift is over the unsigned word the format stores. A signed division by
// 256 would be the same arithmetic only over the bottom half of the range: it
// truncates toward zero, so it would disagree on any position with the top bit
// set.
//
// FromALM and Schedule both come here rather than each shifting for itself.
func unitCell(u alm.Unit) (x, y int32) {
	return int32(u.X >> 8), int32(u.Y >> 8)
}

// FromALM builds a world from a decoded map: one entity per placed unit, in the
// map's own unit-slice order, and the map's extent as the world's bounds. A
// unit's cell is unitCell's, above.
//
// Every entity is born at SpawnHP on both health fields, and this is the ONE
// place in the tree that sets either. It takes no table, so nothing is consulted
// for a per-class maximum and a world stays a function of the decoded map alone.
//
// It is DEFINED IN TERMS of FromALMWith rather than standing beside it. There is
// one world-building implementation, so "this entry point keeps its behaviour
// exactly" cannot decay into two paths that agree today: a nil table resolves
// nothing, so every placement takes the provisional pair by the same code that
// would have given it a decoded one.
//
// An entity's class id is the class key its unit record stores, carried raw:
// the int16-to-int32 conversion IS the sign extension of the format tier's
// typed field. No registry is consulted — whether the key names a class is the
// render side's question — so a world stays a function of the decoded map
// alone.
//
// The i-th unit takes id i, counting from zero. Ids are therefore the slice
// indices — unique by construction. They are an artefact of this transform and
// not an identity the map records carry, so a world cannot be matched back
// against the map it came from.
//
// The constructor's error is discarded rather than propagated into a signature
// no caller could act on, and every way it can be raised is closed here by
// construction: the ids are indices and so never duplicate, the mode is a
// defined constant written on the line below, and all three planes are sized
// from the same extent this function narrows the bounds from — exactly that many
// bytes apiece, none of the block plane's setting a bit above bit 1 — so neither
// the length check nor the reserved-bit check can fire.
//
// The world routes in the canonical mode and is handed THE PLANES THE MAP ITSELF
// DESCRIBES: for blocking, water, mountain, the impassable flag, the placed
// scenery layer and the eight-cell ring; beside them what each cell costs a
// ground mover to enter and how high it stands. Canonical is named rather
// than left to the zero value, because a loader is exactly the caller a defaulted
// mode would slip past.
//
// A PLACED STRUCTURE REACHES THAT PLANE ONLY THROUGH THE TABLE. It is not a
// sixth arm: it is applied after the five, per footprint cell, and may SUBTRACT
// a block the arms put there, which is how a bridge crosses water and a doorway
// opens a wall. Its inputs — a rectangle and two 32-bit cell sets — live in the
// definition table's Buildings collection, so a world built the single-argument
// way carries no structure at all and its deck and doorway cells stay blocked.
// That is the same disclosure this comment has always made, now with a way out
// of it: pass a table.
//
// THE MAP ALSO AUTHORS WHO IS HOSTILE TO WHOM. Its type-5 roster carries one
// diplomacy row per record, and relationFrom below replays the engine's own
// map-load store of it into the world's relation; the world is built through the
// constructor that takes one. Until this existed a loaded map produced a world in
// which nobody was hostile to anybody, so no group ever found an enemy and a
// mission opened with its monsters standing still — the relation is what makes a
// shipped map fight without being told to. A map carrying no type-5 record still
// produces exactly that older world, by carrying no rows at all.
//
// A nil map is an extent of nothing: no bounds, no entities and no cells. It is
// answered rather than refused for the reason the derivation is total — this
// signature has no error a caller could act on, so every shape a map can take has
// to have an outcome.
//
// The world shares no memory with the map. Entities are built into a slice of
// this function's own, which the constructor copies again, an entity holds only
// integers, and each of the three planes is a fresh slice the constructor copies
// as well: nothing in the returned world reaches back into m, and stepping it
// cannot change what m says.
func FromALM(m *alm.Map) *sim.World {
	// The error is discarded rather than propagated into a signature no caller
	// could act on, and the one way it can be raised is closed here by
	// construction: it is raised for an undefined difficulty and for a table
	// entry this contract refuses, the difficulty is a named constant written on
	// the line itself, and no table means no entry is ever read.
	w, _ := FromALMWith(m, nil, DifficultyNormal)
	return w
}

// FromALMWith builds a world from a map together with a definition table and
// a difficulty. It is FromALM's contract in every respect the sentences
// above state, and differs in what a RESOLVED placement takes off its
// definition: the health maximum as both its health and its maximum, the
// rate, the movement domain, the EIGHT NUMBERS A FIGHT READS — the attack
// charge and relax, the to-hit, the defence, the absorption, the damage base
// and spread, and the mark that says its blows always land — and, since
// 0109, the MANA PAIR AND THE TWO REGENERATION PERIODS: a units row's own
// four, a humans row's own mana pair with the base constructor's periods,
// and the base constructor's own four for a placement that resolves to
// nothing — the same substitution every other number in this paragraph
// already takes. Where FromALM gives every placement the provisional pair,
// the ground domain and the constructor's own numbers.
//
// ALL OF THEM COME OFF ONE RESOLUTION, in one statement, so there is no path
// that gives an entity some of them and not the rest. That is the whole of the
// completeness claim and it is structural: the fields are written in a single
// composite literal off a single value, so a field left unfilled is a field a
// reader can see is absent, where a helper that filled "the combat numbers"
// would have a call site somebody could forget.
//
// THERE ARE TWO KINDS OF RESOLVED PLACEMENT and they are not two spellings of
// one thing. A Units match is a CREATURE: its row carries the eight and its
// class derives nothing, so the columns are the numbers, and the difficulty
// adjusts two of them. A humans-band match — any of the three rungs — is a
// PERSON, whose row carries no damage pair, no routing selector and no
// absorption at all, so his eight come out of the same derived-stat graph a
// party member's do, with the weapon his own row names in his hand, and no
// difficulty reaches him. One code path for both would have shipped every person
// swinging for nothing while looking finished.
//
// A PERSON'S HEALTH IS ALSO THE GRAPH'S, as of 0133-person-health: his row's
// own health column is streamed and carried on the definition but is not
// read here — HumanDef.DerivedMaximum runs his Hero through the same graph
// his eight already do, over his row's own Profile and no loadout, and that
// is both his live health and his maximum at placement, on every difficulty
// alike. A CREATURE'S HEALTH STAYS THE COLUMN'S, adjusted by the difficulty
// exactly as it always was — the two bands disagree about where their
// numbers come from as much as they disagree about the eight.
//
// A placement matching NOTHING is handed the definition tier's own constructor
// value whole and takes its rate and its eight off that, so it keeps exactly
// what a world built with no table gives it. Its health is the one number that
// does not come from there: it is the provisional pair, which no story has
// moved. Disclosed rather than papered over.
//
// It differs in a second respect: the block plane. A table carrying a Buildings
// collection lets this map's placed structures reach the plane, and a world
// built with none — every world FromALM builds — gets the five arms alone.
//
// AND IN A THIRD: the spell table.
//
// The table is an ARGUMENT and not something read from an archive here: this
// tier opens nothing, and a loader that reached for an installed file would give
// one map two worlds depending on what is installed. A nil table means "no
// table" and is the whole of what FromALM passes.
//
// The world records neither the raw table nor the difficulty. Both are consumed
// here into resolved Entity fields. Most predate this loader, but the weapon-kind
// Resistance family now contributes five explicit canonical bytes and XPSlot
// carries its selector (1039). Resistance widened the entity record in form 59;
// older readable forms upgrade those bytes to zero with a disclosure because a
// byte-form upgrader has neither this table nor the loadout needed to recompute
// them. Every resolved value enters the digest through that canonical form.
//
// It refuses a difficulty outside the three defined values, and refuses a
// resolved entry whose damage selector takes an arm this contract does not
// model — naming the entry, rather than shipping a monster with no damage. Both
// yield a nil world.
func FromALMWith(m *alm.Map, t *Table, diff Difficulty) (*sim.World, error) {
	w, _, err := FromALMRoster(m, t, diff)
	return w, err
}

// FromALMRoster is FromALMWith with the map's PERSON ROSTER beside the
// world: one PartyMember template per placement that resolved through a
// person row, keyed by the entity id that placement was minted with.
//
// IT EXISTS BECAUSE A MISSION BOUNDARY NEEDS WHAT AN ENTITY CANNOT SAY. An
// actor a script hands to the player crosses the boundary and becomes a roster
// member on the next map, and minting him there needs his four statistics, his
// profile and his spellbook — the inputs a definition supplied at load and that
// no field of sim.Entity carries. Body, Reaction and Spirit are the three the
// entity has no place for at all, so a companion rebuilt from his entity alone
// would fold his combat numbers from zeroes and arrive weaker than he left.
//
// IT IS BUILT IN THE SAME LOOP THE ENTITIES ARE, off the same resolution and
// the same definition, so a template and the entity it describes cannot come
// from two different rows. The key is the loop's own sim.EntityID(i) rather
// than a second statement of what an id is.
//
// ONLY THE PERSON ARMS FILL IT. A creature placement and an unresolved one have
// no person row to template from, and neither can cross the boundary anyway:
// the band test refuses them. So a missing key is the ordinary case, not a
// failure.
func FromALMRoster(m *alm.Map, t *Table, diff Difficulty) (*sim.World, map[sim.EntityID]PartyMember, error) {
	// Asked before the walk, so a map that resolves nothing still refuses an
	// undefined value rather than accepting it by never reaching the arithmetic.
	if !diff.defined() {
		return nil, nil, fmt.Errorf("mapload: difficulty %d is not 1, 2 or 3", int32(diff))
	}

	var units []alm.Unit
	var groups []alm.Group
	var b sim.Bounds
	if m != nil {
		units = m.Units
		groups = m.Groups
		b = sim.Bounds{Width: int32(m.Width), Height: int32(m.Height)}
	}

	ents := make([]sim.Entity, len(units))
	var loadout []sim.Stock
	// One template per placement that resolved through a person row — see this
	// function's own doc. It is nil until the first one, so a map with no
	// people hands the boundary nothing to look in rather than an empty map.
	var roster map[sim.EntityID]PartyMember
	for i, u := range units {
		x, y := unitCell(u)
		b, err := blockFor(u, t, diff)
		if err != nil {
			return nil, nil, fmt.Errorf("mapload: placement %d: %w", i, err)
		}
		if tmpl, ok := rosterTemplate(u, t, sim.EntityID(i)); ok {
			if roster == nil {
				roster = make(map[sim.EntityID]PartyMember)
			}
			roster[sim.EntityID(i)] = tmpl
		}
		// The group, the owner and the authored map id come off the record and
		// hold whether or not the placement resolved. UNIT-PLACE-034: an
		// authored current health is signed and applied verbatim; difficulty
		// scales only the resolved maximum.
		def := b.actorDefinition(t)
		if u.HasCurrentHP {
			def.HP = int32(u.CurrentHP)
		}
		if member, ok := roster[sim.EntityID(i)]; ok {
			def.NativeTraining = sim.NativeTraining{Present: true, Levels: member.Hero.Skill}
		}
		ents[i] = sim.NewActor(def, sim.ActorPlacement{ID: sim.EntityID(i), X: x, Y: y,
			Owner: u.Owner, Group: u.GroupID, MapUnitID: u.UnitID})
		if u.HasCurrentHP {
			// An authored body arrives already fallen; this applies the
			// death-time combat adjustment without running clearFelled, which
			// would wrongly pour its authored worn equipment onto the ground.
			sim.PrepareAuthoredBody(&ents[i])
		}
		// ONE sim.Stock PER PLACEMENT CARRYING BOTH the worn set and its
		// container overflow (0128 plan D-2): b.worn and b.carried are
		// already the whole of what wearRow or wornFromWeapon resolved,
		// gated by startingLoadout, so there is nothing left for this loop
		// to decide about WHERE an item goes — only whether there is
		// anything here worth an entry at all. A placement naming nothing
		// resolvable costs no entry, exactly as before this story.
		if !itemEquipmentEmpty(b.worn) || len(b.carried) > 0 {
			loadout = append(loadout, sim.Stock{ID: sim.EntityID(i),
				ItemInstances: cloneItemInstances(b.carried), EquippedItems: cloneItemEquipment(b.worn)})
		}
	}
	// The three planes the world stands on, from the one bundler every
	// world-building path here calls. The other two read the map alone and take
	// no table at all, because a structure restores a cell's cost baseline
	// rather than moving it.
	//
	// THE MAP ALSO AUTHORS ITS OWN GROUND LOOT AND ITS OWN STOCK: sacksFrom
	// decodes the type-8 section's ground arm into the sacks
	// sim.NewStockedWorld places on the map, and stockFrom decodes the same
	// section's other arm into the Stock entries that constructor puts into an
	// actor's container instead. The error the constructor can return is
	// discarded on the same ground every other arm of this function already
	// discards one on: sacksFrom has already dropped what would make it fire
	// over a sack (an out-of-bounds cell), and stockFrom never names an id ents
	// does not hold — every id it emits is an index into m.Units, the same
	// indices the loop above built ents from, one per unit — so the only way
	// left to raise the error is a difficulty or a domain this call has already
	// refused above.
	authoredSacks, err := sacksFrom(m, t)
	if err != nil {
		return nil, nil, err
	}
	authoredStock, err := stockFrom(m, t)
	if err != nil {
		return nil, nil, err
	}
	w, _ := sim.NewStructuredWorld(Seed, b, sim.ModeCanonical, Planes(m, t), ents, nil,
		relationFrom(groups), authoredSacks, append(authoredStock, loadout...), spellsFor(t),
		ghostTemplate(t), Structures(m, t))
	w.SetRules(t.rules())
	authored := units
	if m != nil && m.AuthoredUnits != nil {
		authored = m.AuthoredUnits
	}
	w.SetDiaryUnits(diaryRows(t), diaryUnits(authored, t))
	if err := bindCellTails(w, m); err != nil {
		return nil, nil, err
	}
	// AND THE WEIGHT OF EVERYTHING IT NAMES, off the same t. It is declared
	// AFTER the constructor rather than through it, because the pass reads the
	// finished world's own containers, slots and sacks -- the three the
	// constructor has just normalised -- rather than the argument lists it was
	// handed, which name the same codes in a shape that has not been folded
	// yet.
	declareItemWeights(w, t)
	if err := initializeCurrentPlayers(w, m, t); err != nil {
		return nil, nil, err
	}
	return w, roster, nil
}

// sacksFrom decodes a map's type-8 loot section into the sacks a world built
// from it should hold: one entry per GROUND record, in file order, at the
// record's cell, carrying the record's gold and its elements' item codes in
// element order.
//
// A STOCK RECORD PLACES NOTHING: its owner names an actor that already
// exists rather than putting anything on the ground, so Ground() alone
// decides whether a record survives this pass.
//
// A RECORD WHOSE CELL FALLS OUTSIDE THE MAP IS DROPPED, AND THE LOAD
// SUCCEEDS: the bound test reproduces the one sim.NewLootWorld would
// otherwise refuse the WHOLE world over, so a single bad record costs only
// itself. No shipped record reaches it (D-7). It is written out here rather
// than reached through pkg/sim, because the predicate the constructor
// refuses on is unexported — sim.Bounds describes a world's extent and
// this function has none yet to hand it.
//
// A DECODE ERROR YIELDS NO SACKS, NOT A LOAD FAILURE: loot is not what makes
// a map playable, and this loader's other passes already prefer a playable
// map to a refused one. Structures builds a world's structure list from a
// map's type-4 records (1033 B3): one Structure per record, in file order,
// the entity list's own precedent — every placed record becomes a
// Structure whether or not any script node ever names it.
//
// The id is the RECORD'S INDEX, not any word the record carries: StructureID
// is an opaque handle (structure.go), and ScriptStructures builds the same
// index over the same order to resolve a Target_Structure reference against
// it.
//
// Field42 starts at the definition value, then a Building placement with a zero
// Field0C overrides it to zero (ALM-128). Nonzero words keep the definition
// value. Shop selection tests the stored low16 kind for 34/35 (ALM-127), and
// its independent cap operation does not clear this field. The maximum value
// is a property of the INSTALLED TABLE the placement resolves against, on the
// same terms as a footprint: `ALM-CLS-053` names the type-4 spawn's own writes
// as `sizeX`/`sizeY` to the footprint, `scanRange` to `obj+0x48` and
// `healthMax` to `obj+0x44`/`+0x42`, and `+0x42` is the word both script arms
// read. `SAV-BLDG-037` measures `+0x42`/`+0x44` in the owner's own original
// saves as equal non-zero pairs — 1000/1000, 30000/30000, 100/100, 2000/2000 —
// which are the `healthMax` values `DAT-BLD-005` gives for those building
// kinds. The entry is selected by buildingParams (structures.go), the one
// reading of the selection rule this package has.
//
// Field42 remains offset-named because scripts expose that exact word, but its
// role is now established: it is current health paired with maximum health.
// `TERR-STRUCT-102` consumes its signed value for ruin selection and the spell
// paths consume and clamp the same word; structure.go owns that simulation
// meaning and its claim boundary.
//
// IT IS EXPORTED SO AN INSTRUMENT CAN READ THE SAME DERIVATION A WORLD IS
// BUILT FROM. cmd/classdump reports the seeded field per placement, and a
// reporter that recomputed the seed would be testing its own copy of this
// rule rather than this rule. It is pure and total: any map and any table
// yield a list, and the same pair always yields the same list.
//
// THE STORE TRUNCATES TO 16 BITS, reproducing the width of the destination the
// claim names rather than widening it here. No shipped entry reaches the
// truncation: `healthMax` runs 0..30000 over all 66 entries on both roots, with
// no negative row and no row above 65535.
//
// A PLACEMENT THAT RESOLVES NO ENTRY, AND AN ENTRY TOO NARROW TO CARRY THE
// POSITION, EACH LEAVE THE FIELD AT ZERO and still yield a Structure. The
// structure list is one entry per placed record whatever the table says, on the
// entity list's own precedent, and a table is a third input a map cannot vouch
// for. Neither case is reached on shipped content: `ALM-CLS-053` lands all
// 3141 shipped kinds on a named entry, and no shipped entry is short.
func Structures(m *alm.Map, t *Table) []sim.Structure {
	if m == nil || len(m.Objects) == 0 {
		return nil
	}
	c := t.buildings()
	out := make([]sim.Structure, len(m.Objects))
	for i, o := range m.Objects {
		out[i] = sim.Structure{ID: sim.StructureID(i), Col: int32(o.X >> 8), Row: int32(o.Y >> 8)}
		StructureUseMetadata(&out[i], uint16(byte(o.Kind)), t)
		p, ok := buildingParams(o, c)
		if ok && len(p) > healthMaxSlot {
			out[i].Field42 = uint16(p[healthMaxSlot])
			out[i].MaxHealth = uint16(p[healthMaxSlot])
		}
		if o.Field0C == 0 && uint16(o.Kind) != 34 && uint16(o.Kind) != 35 {
			out[i].Field42 = 0
		}
		if f, ok := resolve(o, c, &StructureCounts{}); ok {
			out[i].Col, out[i].Row = int32(f.Col), int32(f.Row)
			out[i].Width, out[i].Height = uint8(f.Width), uint8(f.Height)
			out[i].Attach = f.Attach
			out[i].Blocking = f.Blocking
		}
	}
	return out
}

func sacksFrom(m *alm.Map, t *Table) ([]sim.Sack, error) {
	if m == nil {
		return nil, nil
	}
	loot, err := m.Loot()
	if err != nil {
		return nil, nil
	}
	w, h := m.Width, m.Height
	var out []sim.Sack
	for _, r := range loot.Records {
		if !r.Ground() {
			continue
		}
		x, y := r.CellX(), r.CellY()
		if x < 0 || y < 0 || int(x) >= w || int(y) >= h {
			continue
		}
		items := make([]sim.ItemInstance, len(r.Elements))
		for i, e := range r.Elements {
			items[i], err = lootItemInstance(e, m, t)
			if err != nil {
				return nil, err
			}
		}
		out = append(out, sim.Sack{X: x, Y: y, Gold: r.Gold, ItemInstances: items})
	}
	return out, nil
}

// stockFrom decodes a map's type-8 loot section into the sim.Stock entries a
// world built from it should hold: one entry per STOCK record — the arm
// sacksFrom skips — naming the entity that record's elements arrive in,
// with no sack placed for it.
//
// THE JOIN IS A CORPUS JOIN, NOT A DECODED ONE (R-4). ITEM-OWNED-028, the
// research row that establishes the stock arm, grades the ID SPACE ITSELF
// Medium: nothing reads what writes the actor-side key, or what the shipped
// values correspond to in the .alm's own record sets. What this function
// relies on instead is a census over both lawful roots' campaign corpora, 28
// maps each: 43 stock records per root, every one matching exactly one
// unit's UnitID, none unmatched and none ambiguous. That is a BIJECTION OVER
// 43 CASES, not a proof, which is why a miss costs only itself rather than
// refusing the load or guessing an actor.
//
// A RECORD MATCHING NO UNIT, OR MORE THAN ONE, CONTRIBUTES NOTHING:
// sacksFrom's own rule for a record it cannot place, restated for the owner
// word instead of the cell. The unit-id-to-index map is built ONCE, over
// m.Units, counting how many units carry each id as it goes — the count is
// what "exactly one" reads, without a second pass over the units for every
// record.
//
// THE ENTITY ID IS THE UNIT'S OWN INDEX: FromALMWith's own rule is that the
// i-th unit takes id i, so the map built here is keyed by the widened
// UnitID and valued by that same index, and the Stock this function emits
// names it directly — there is no second translation between a unit and the
// entity it became.
//
// UNITID IS WIDENED TO THE OWNER WORD'S OWN 32 BITS for the comparison,
// never the other way: LootRecord.Owner is carried at the file's own u32
// width and narrowed nowhere by this reader (loot.go's own precedent), so
// narrowing it down to UnitID's 16 bits here would risk a collision no
// research row states. Widening the smaller field costs nothing a shipped
// record could observe.
//
// NEITHER THE CELL NOR THE GOLD IS READ ON THIS ARM: CellX, CellY and Gold
// are what sacksFrom reads for the GROUND arm of this same walk; the
// original never reads them for an owned record, so this function reads
// Owner and Elements alone.
//
// A NIL MAP OR A LOOT DECODE ERROR IS NO STOCK — sacksFrom's own two rules
// for the same two cases, read here rather than restated: loot is not what
// makes a map playable.
func stockFrom(m *alm.Map, t *Table) ([]sim.Stock, error) {
	if m == nil {
		return nil, nil
	}
	loot, err := m.Loot()
	if err != nil {
		return nil, nil
	}

	byUnitID := make(map[uint32]int, len(m.Units))
	count := make(map[uint32]int, len(m.Units))
	for i, u := range m.Units {
		id := uint32(u.UnitID)
		byUnitID[id] = i
		count[id]++
	}

	var out []sim.Stock
	for _, r := range loot.Records {
		if r.Ground() {
			continue
		}
		idx, ok := byUnitID[r.Owner]
		if !ok || count[r.Owner] != 1 {
			continue
		}
		items := make([]sim.ItemInstance, len(r.Elements))
		for i, e := range r.Elements {
			items[i], err = lootItemInstance(e, m, t)
			if err != nil {
				return nil, err
			}
		}
		out = append(out, sim.Stock{ID: sim.EntityID(idx), ItemInstances: items})
	}
	return out, nil
}

func lootItemInstance(element alm.LootElement, m *alm.Map, t *Table) (sim.ItemInstance, error) {
	item := ItemInstanceFromCode(element.ItemCode(), t)
	if element.TileMarkerIndex == 0 {
		return item, nil
	}
	index := element.TileMarkerIndex - 1
	if index >= uint32(len(m.Enchantments)) {
		return sim.ItemInstance{}, fmt.Errorf("alm: loot item %#04x links type9 record %d, but the map holds %d", item.Code, element.TileMarkerIndex, len(m.Enchantments))
	}
	recipe := m.Enchantments[index]
	if recipe.X != 0 || recipe.Y != 0 {
		return sim.ItemInstance{}, fmt.Errorf("alm: loot item %#04x links ineligible type9 record %d at (%d,%d)", item.Code, element.TileMarkerIndex, recipe.X, recipe.Y)
	}
	appendEffect := func(kind uint32, operand uint32) error {
		if kind > 255 {
			return fmt.Errorf("alm: type9 record %d effect kind %d exceeds u8", element.TileMarkerIndex, kind)
		}
		item.Effects = append(item.Effects, sim.ItemEffect{Kind: uint8(kind), Operand: operand})
		return nil
	}
	if recipe.A != 0 {
		// The head constructor stores B and C as the first two operand bytes.
		// They are already base and spread; C is not an upper endpoint.
		operand := uint32(uint8(recipe.B)) | uint32(uint8(recipe.C))<<8
		// ALM-T9CONTROL-175 narrows the head result before construction.
		// The separate authored tail keeps its own width validation below.
		if err := appendEffect(uint32(uint8(uint32(recipe.A)+43)), operand); err != nil {
			return sim.ItemInstance{}, err
		}
	}
	if uint16(recipe.SpellRaw) != 0 {
		kind := uint8(41)
		if item.Kind == 5 {
			kind = 42
		}
		item.Effects = append(item.Effects, sim.ItemEffect{Kind: kind, Operand: recipe.SpellRaw})
	}
	for _, effect := range recipe.Elements {
		kind := effect.Kind
		if kind == 41 {
			kind = 49
		}
		// A tail entry is copied as its two authored words. Scalar entries in
		// the shipped links carry zero in the high word; preserving that layout
		// is what turns raw 5:0 into +5 instead of an underflowed byte range.
		operand := uint32(effect.Low) | uint32(effect.High)<<16
		if err := appendEffect(uint32(kind), operand); err != nil {
			return sim.ItemInstance{}, err
		}
	}
	item.Price = RepriceItemInstance(item, t)
	return item, nil
}

// relationFrom is the map's own diplomacy, stored the way the engine's map-load
// path stores it (AI-DIPLO-005, ALM-GRP-041).
//
// THE FILE'S ROW IS 0-BASED OVER ROSTER RECORDS AND THE MATRIX IS 1-BASED, which
// is the whole of the index arithmetic and the only place it is written down.
// The i-th roster record is slot i+1 — the same 1-based slot a placed unit's
// owner word names (ALM-OWN-039), which is what lets an entity's Owner index this
// matrix at all — and element k of its sixteen words lands at column k+1. So
// COLUMN 0 IS NEVER WRITTEN by anything here, and it never can be: no slot names
// it, so no rule reads it either.
//
// EACH WORD IS NARROWED TO ITS LOW BYTE, and the narrowing is the engine's own
// (a one-byte store). It is not lossy on any shipped map —
// over 38 maps and 3056 cells the words are exactly {0, 1, 2} — but that is a
// corpus census at Medium, not a bound the format states, so a map carrying a
// word above 0xff loses its high byte here exactly as it would in the original.
// The format tier keeps the full width (alm.Group.Relation) so the loss happens
// once, here, and is visible as a conversion rather than as a decode.
//
// THE DIAGONAL IS THEN FORCED TO 2, unconditionally and after the row, and this
// is load-bearing rather than tidy: the shipped corpus spells its own diagonal
// as 2 in 186 records, as 1 in two and as 0 in three, so five shipped rosters
// disagree with the engine about whether a player is at war with itself. Bit 0
// of 2 is clear, so the forced value is "not hostile" — a slot never acquires
// its own units, whatever its file says. It is forced for EVERY roster record,
// including one past the sixteen columns a row covers, because the engine's
// store is unconditional too.
//
// A ROSTER SLOT OUTSIDE THE MATRIX CONTRIBUTES NOTHING. Relations.Set drops a
// write it cannot place rather than refusing it, so a map declaring more roster
// records than the matrix has slots loads, with the excess rows absent — which
// is the byte-write-into-a-fixed-block the original does. No shipped map comes
// near it: the editor caps a roster at sixteen.
//
// NO ROSTER YIELDS NO ROWS, and therefore the empty relation — under which
// nobody is hostile to anybody and nothing ever acquires. That is the world
// constructor's own materialisation of an unnamed relation and not a second rule
// stated here, so "a map with no type-5 record" and "a map whose rows are all
// zero" differ only where the file differs from them: the forced diagonals.
func relationFrom(groups []alm.Group) sim.Relations {
	var rel sim.Relations
	for i, g := range groups {
		slot := uint32(i) + 1
		for k, w := range g.Relation {
			rel.Set(slot, uint32(k)+1, byte(w))
		}
		rel.Set(slot, slot, 2)
	}
	return rel
}

// spawnBlock is everything one placement's resolution is worth: the health both
// fields take, the movement domain, the rate, the dwell of the body it will
// leave, and the eight numbers a blow reads.
//
// It is a VALUE and not a helper that writes into an entity, because the
// completeness claim above is structural: one function produces the whole of it
// down whichever arm ran, and one composite literal spends it. A helper writing
// fields would have a call site somebody could forget.
//
// mana, manaMax, healthPeriod and manaPeriod are 0109's own four: the mana
// pair and the two regeneration periods a placement is born with, named as
// sim.Entity names them so a value copied across this boundary keeps its
// name. Which source each arm reads them off is blockFor's own story — a
// units row, a humans row plus the base constructor, or the base constructor
// whole — and never a literal written out here.
//
// xpValue, mind and gainsXP are 0125's own three: a placement's own
// experience value, its own Mind, and whether its class gains experience at
// all. THE CREDITED SLOT IS NOT A FOURTH FIELD HERE — it already rides on
// combat.SkillSlot, which every arm's own combat already carries, and a
// second field for it would be a second channel the first could come to
// disagree with. Which source each of the three reads off is blockFor's own
// story, exactly as the mana pair's is, and never a literal written out
// here.
type spawnBlock struct {
	nativeBasis  sim.NativeActorBasis
	nativeClass  sim.NativeClass
	class        int32
	health       int32
	domain       sim.Domain
	speed        int32
	sight        uint8
	seeInvisible uint8
	dying        int32
	withdraw     int32
	wimpy        int32
	combat       data.Combat
	protection   [5]int32
	resistance   [5]uint8
	tokenSize    uint8

	mana, manaMax                                       int32
	healthPeriod, manaPeriod                            int32
	healthRegeneration, manaRegeneration, rotationSpeed int32
	secondaryDamage                                     data.SecondaryDamage

	xpValue                int32
	reaction, mind, spirit int32
	// capacity is the placement's carrying capacity: `Body x 10 + 1` on the
	// person arm (HERO-SIGHT-007), the constructor's data.UnitCapacity on the
	// two units arms (UNIT-CTOR-004). DIV-224.
	capacity           int32
	humanoid           bool
	gainsXP            bool
	suppressCorpseLoot bool

	typeID                               int32
	goldChance, treasureMin, treasureMax int32

	skill   [data.SkillSlots]int32
	skillXP [data.SkillSlots]int32

	// The resolved Human or Unit row supplies its spellbook. An unresolved
	// placement leaves both fields empty.
	knownSpells uint32
	book        sim.Spellbook
	// creatureSpells are the class's spell slots a creature draws against.
	creatureSpells [sim.CreatureSpellSlots]sim.CreatureSpell

	// worn is the WORN SET this placement's class row arms, slot 1 at index 0
	// exactly as sim.Stock.Equipped states it (spec Terms "Worn set") — the
	// units arm's own one weapon code, through wornFromWeapon, or the humans
	// arm's own wearRow reading all ten cells. It is the same resolution the
	// combat numbers above were folded from, handed on rather than searched for
	// again, so a unit's numbers and the items on his body cannot come off
	// different cells of the same row.
	//
	// It is an array of complete item instances because effects and stored
	// value cross the loader boundary with the piece. pkg/sim retains no item
	// table and therefore cannot reconstruct either from a code later.
	worn [sim.EquipSlots]sim.ItemInstance

	// carried is the CONTAINER OVERFLOW this placement's row costs: an armour
	// whose Slot column names no slot at all, or a shield whose row weapon
	// leaves no hand free. Nil costs nothing,
	// which is every placement this story does not touch and every row whose
	// cells cost no overflow.
	carried []sim.ItemInstance
}

// weaponCode is w's item code, and zero for no weapon at all — the one place
// a resolved weapon becomes the value a simulation can hold. Zero is what
// equip.go already means by an empty slot, so "this row names no weapon",
// "this build cannot resolve the one it names" and "this slot is free" are
// one value here rather than three.
func weaponCode(w *data.Weapon) uint16 {
	if w == nil {
		return 0
	}
	return uint16(w.Code)
}

// sightOf narrows a definition's scan-range column to the byte the actor
// field it fills actually is.
//
// THE NARROWING IS THE ENGINE'S OWN and not a guard of ours, exactly as the
// relation's low-byte conversion above is. Both spawn streamers put this column
// through a helper whose store is a one-byte store after the empty-cell
// compare, so a column value outside a byte loses its high bytes there — it is
// not clamped, not refused, and not the reason a row fails.
//
// IT IS NOT LOSSY ON ANY SHIPPED ROW: over both lawful roots the creature band's
// 56 rows carry 4 to 12 and the person band's 210 rows carry 4 to 7. That is a
// corpus census and not a bound the format states, so a row carrying 300 loses
// its high byte here exactly as it would in the original, and the definition tier
// keeps the full width so the loss happens once, here, and is visible as a
// conversion rather than as a decode.
func sightOf(scanRange int32) uint8 { return uint8(scanRange) }

// reachOf narrows a combat block's reach to the byte the actor field it
// fills actually is — sightOf's own move, over a value the resolution
// DERIVED from a weapon's range column rather than one read off a row's own
// cell. Whether a shipped row can drive this narrowing lossy is AC-1's own
// census and not asserted here; the byte's legal range is 1 to 255,
// sim.Entity.Reach's own, and this function performs no check against it —
// a decode refuses a reach outside that range, the way reachFault already
// does, and a loader is not that decoder.
func reachOf(reach int32) uint8 { return uint8(reach) }

// blockFor resolves one placement all the way to what it is worth, down exactly
// one of three arms.
//
// The arms are asked in the order the ladder itself splits: a creature first,
// because the class key divides the record set before anything else is looked
// at; then a person; then neither. The two resolved arms are asked through their
// own definition builders rather than through a shared one — the two collections
// agree about the empty-cell law and about nothing else.
//
// The DIFFICULTY is passed to the creature arm alone.
//
// A refusal from either builder takes the world with it. A placement shipped
// with half a definition is the failure the whole of this is against.
func blockFor(u alm.Unit, t *Table, diff Difficulty) (spawnBlock, error) {
	def, worn, ok, err := definitionFor(u, t, diff)
	if err != nil {
		return spawnBlock{}, err
	}
	if ok {
		return unitRowBlock(int32(u.ClassID), Resolve(u, t).Index, def, worn, t), nil
	}

	h, w, hworn, hcarried, ok, err := definitionForHuman(u, t)
	if err != nil {
		return spawnBlock{}, err
	}
	if ok {
		class := int32(u.ClassID)
		if _, composed := t.composedNPCIndex(int32(u.ClassSubID)); composed && u.Flags&npcFlagBit != 0 {
			class = h.TypeID
		}
		if Resolve(u, t).Arm == ArmServerID {
			class = h.TypeID
		}
		// Only the exact Hero NPC constructor replaces the Human row's type.
		typeID := h.TypeID
		if r := Resolve(u, t); r.Arm == ArmNPC && t.npc().Hero(int32(u.ClassSubID)) {
			dir, _ := data.FigureFor(h.TypeID, h.Face, h.Gender)
			typeID = sim.HeroTypeID(h.Profile().ManaColumn, dir.Female())
		}
		// THE HUMANS TABLE CARRIES NO PERIOD COLUMN AT ALL: a person's two periods
		// are the base constructor's own 100 and 50, taken from
		// data.UnitDefaults() rather than written out here a second time, so the
		// one number this row cannot state never gets a second source.
		reg := data.UnitDefaults()
		// THE OWNER RECOMPUTE, NOT THE BARE ONE (hotfix map-hero-stats,
		// docs/hotfix/LEDGER.md): der.Combat.Defence and der.Combat.Absorption
		// used to come off h.Derived(w) — the weapon alone, an EMPTY Profile,
		// no fold of the row's own worn armour — which left every worn piece
		// contributing nothing to a map-placed person's combat block and left
		// der.ManaMax un-derivable (Profile{}.ManaColumn is always false), so
		// this arm read the row's own raw mana cell below instead of a
		// derived pool. loadout below is resolved through
		// mapload.ResolveEquipmentLoadout over this row's OWN starting worn
		// set — the same function a party member's own spawn
		// (PartySpawnWithTable/partySpawn, this file's own start.go) and a
		// live skill-raise (pkg/game rearm.go recomputeRaisedSkills) already
		// resolve through — with everEquipped FALSE: a map placement has
		// never had anything taken off, so an empty weapon slot still
		// answers the row's own weapon w as ResolveEquipmentLoadout's own
		// doc states for that flag's false case, never a bare loadout.
		// lok false only where the table itself is unusable for a fold (nil
		// or missing item collections); the fallback below keeps exactly
		// this arm's pre-hotfix loadout (the weapon alone) for that case, so
		// a table too thin to fold armour still starts a person armed.
		nativeBasis := nativeInitialModifier(nativeInitialBase(&h.Skill), &hworn, t, h.Skill[0], true, h.Profile().Fighter)
		loadout, lok := ResolveEquipmentLoadout(equipmentFromSlots(itemEquipmentCodes(hworn)), w, false, t)
		if !lok {
			loadout = data.Loadout{Weapon: w, Rules: t.rules()}
		}
		ApplyItemEffects(&loadout, hworn, h.Profile().Fighter)
		der := h.DerivedWithLoadout(h.Profile(), loadout)
		var skillXP [data.SkillSlots]int32
		for i, level := range der.Skill {
			skillXP[i] = t.rules().SkillXPExtended(level)
		}
		mana := int32(0)
		if h.Profile().ManaColumn && der.ManaMax > 0 {
			mana = der.ManaMax
		}
		_, _, taught := EquippedPoolEffects(hworn)
		return spawnBlock{class: class, health: der.HealthMax, domain: domainForCode(h.MovementType),
			nativeBasis: nativeBasis.WithBody(uint16(der.Body)),
			nativeClass: sim.NativeClass{Present: true, Fighter: h.Profile().Fighter},
			humanoid:    true,
			speed:       h.Speed + riderBonus(typeID) + loadout.Mod.Speed, sight: sightOf(h.ScanRange + loadout.Mod.Sight), dying: h.DyingTime,
			combat: der.Combat, protection: der.Protection,
			resistance: data.DamageKindResistance(der.Resistance), tokenSize: uint8(h.TokenSize), skill: der.Skill, skillXP: skillXP,
			capacity: der.Capacity,
			mana:     mana, manaMax: mana,
			healthPeriod: reg.HealthRegenPeriod, manaPeriod: reg.ManaRegenPeriod,
			// THE HUMANS TABLE CARRIES NO EXPERIENCE-VALUE COLUMN EITHER: the period
			// pair's own reason, restated for a third number — reg is the same
			// base-constructor value already in hand, and its XPValue is the zero the
			// constructor leaves standing for every empty units cell, not a literal
			// chosen here. MIND IS NOT BORROWED: HumanDef carries its own column for
			// it, the same one Hero() reads, so a person's Mind is the row's own.
			// Training follows the constructor TypeID band, not the broad Humans
			// table arm: mission-only low-type people do not earn.
			xpValue: reg.XPValue, reaction: der.Reaction, mind: der.Mind, spirit: der.Spirit,
			gainsXP: sim.InPersistBand(typeID),
			// THE SERVER TYPE ID A PERSON'S ARM WRITES. It is the field the mission
			// boundary reads and nothing else here: the three treasure values beside
			// it stay at the creature arm's own zeroes, and the death-gold roll is
			// gated strictly above the band, so a person carrying this drops no gold.
			typeID: typeID,
			// THE ROW'S OWN SPELLBOOK (0127 FR-4a), off the same h every other field
			// on this arm comes off — a HumanDef is the one definition this tree
			// ever reads KnownSpells from.
			knownSpells:        h.KnownSpells | taught,
			healthRegeneration: der.HealthRegeneration, manaRegeneration: der.ManaRegeneration,
			rotationSpeed:      HumanTurnRate(der.Speed),
			secondaryDamage:    der.SecondaryDamage,
			suppressCorpseLoot: SuppressesCorpseLoot(personRowName(u, t)),
			worn:               hworn, carried: hcarried}, nil
	}

	// THE UNRESOLVED ARM SUBSTITUTES THE WHOLE CONSTRUCTOR DEFINITION, once,
	// before anything is read off it — so every number has exactly one source
	// whichever arm ran. The HEALTH stays outside the substitution: the
	// provisional pair is ours and is deliberately not the definition tier's 30,
	// which no story has moved, so it is the one number this arm still decides
	// for itself. The mana pair and the two periods are NOT outside it — they
	// are named individually below, off the same d, because the block this
	// function returns is a composite literal naming its fields one at a time
	// and the four new ones do not arrive for free.
	d := data.UnitDefaults()
	return spawnBlock{class: int32(u.ClassID), health: SpawnHP, domain: domainFor(d),
		speed: d.Speed, sight: sightOf(d.ScanRange), dying: d.DyingTime,
		combat: d.Combat(), protection: d.Protection,
		resistance: data.DamageKindResistance(d.Resistance),
		mana:       d.Mana, manaMax: d.ManaMax,
		healthPeriod: d.HealthRegenPeriod, manaPeriod: d.ManaRegenPeriod,
		rotationSpeed: d.RotationSpeed,
		// THE CONSTRUCTOR'S OWN EXPERIENCE VALUE AND MIND, off the same d every
		// other field here comes off — d.XPValue is its zero and d.Mind its 20,
		// neither invented for this arm. AN UNRESOLVED PLACEMENT DOES NOT GAIN, on
		// the creature arm's own ground: it substitutes the definition tier whole
		// and that tier derives no class, so there is nothing here that could
		// earn.
		xpValue: d.XPValue, reaction: d.Reaction, mind: d.Mind, spirit: d.Spirit, gainsXP: false, capacity: data.UnitCapacity()}, nil
}

// unitRowBlock is the creature arm's block for the Units row at index. class
// is a placement's ClassID or the raised Ghost's TypeID.
func unitRowBlock(class int32, index int, def data.UnitDef, worn [sim.EquipSlots]sim.ItemInstance, t *Table) spawnBlock {
	params := t.units().EntryParams(index)
	known, book, slots := unitSpellbook(params, t, def.TypeID)
	var skill [data.SkillSlots]int32
	if len(params) > 14 && params[14] >= 0 {
		skill[data.SkillGeneral] = params[14]
	}
	if def.Face == 4 {
		for i := 1; i < len(skill); i++ {
			skill[i] = 30
		}
	}
	nativeBasis := nativeInitialModifier(nativeInitialBase(nil), &worn, t, skill[0], false, true)
	return spawnBlock{class: class, health: def.HealthMax, domain: domainFor(def),
		nativeBasis: nativeBasis.WithBody(uint16(def.Body)),
		speed:       def.Speed, sight: sightOf(def.ScanRange), seeInvisible: sightOf(def.SeeInvisible), dying: def.DyingTime,
		withdraw: def.Withdraw, wimpy: def.Wimpy,
		combat: def.Combat(), protection: def.Protection,
		resistance: data.DamageKindResistance(def.Resistance), tokenSize: uint8(def.TokenSize),
		mana: def.Mana, manaMax: def.ManaMax,
		healthPeriod: def.HealthRegenPeriod, manaPeriod: def.ManaRegenPeriod,
		rotationSpeed:   def.RotationSpeed,
		secondaryDamage: def.SecondaryDamage,
		// THE ROW'S OWN EXPERIENCE VALUE AND MIND, off the same def every other
		// field above comes off. A CREATURE DOES NOT GAIN: a units row derives no
		// class from anywhere — UnitDef.Combat's own doc says as much of the
		// slot it leaves at zero — so there is no class here that could earn,
		// and the flag says so rather than leaving a gain to be paid at zero.
		xpValue: def.XPValue, reaction: def.Reaction, mind: def.Mind, spirit: def.Spirit, gainsXP: false,
		typeID: def.TypeID, goldChance: def.GoldChance,
		treasureMin: def.TreasureMin, treasureMax: def.TreasureMax, capacity: data.UnitCapacity(),
		skill: skill, knownSpells: known, book: book, creatureSpells: slots, worn: worn}
}

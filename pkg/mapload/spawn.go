package mapload

import (
	"fmt"
	"strings"

	"againrom/pkg/base"
	"againrom/pkg/data"
	"againrom/pkg/formats/alm"
	"againrom/pkg/mod"
	"againrom/pkg/sim"
)

// SuppressesCorpseLoot reports the template-name property used by the death
// routine. The comparison is case-sensitive and runs on the canonical
// definition row name, never on a localized or player-visible actor name.
func SuppressesCorpseLoot(templateName string) bool {
	return strings.Contains(templateName, "NPC")
}

// Difficulty is the three-valued scenario setting a spawner applies to what it
// builds. It is consumed ONCE, at the moment a unit is built, and lives
// thereafter only in that unit's own numbers, so nothing downstream can recover
// it and a world records it nowhere.
//
// The values are the setting's own. The three NAMES are ours, on the owner's
// testimony: the control is a three-button choice on the character pre-create
// screen and its three bitmaps carry no text at all, so what the setting is
// CALLED is not established by anything decoded. Which value weakens and which
// strengthens is not testimony but arithmetic, and is below.
type Difficulty int32

const (
	// DifficultyEasy weakens a placed non-hero: two thirds of its health.
	DifficultyEasy Difficulty = 1
	// DifficultyNormal is what a caller who says nothing gets, and is the
	// identity.
	DifficultyNormal Difficulty = 2
	// DifficultyHard strengthens it: half again the health, and fifty on each
	// of to-hit and defence.
	DifficultyHard Difficulty = 3
)

func (d Difficulty) defined() bool {
	return d == DifficultyEasy || d == DifficultyNormal || d == DifficultyHard
}

// Table is the definition table as a world builder consumes it: the three
// collections a placement can resolve against, and nothing else of the eleven.
//
// It is a THIRD INPUT to a world and never a fact of a map. A map says which
// class key a placement carries; what that key is worth is a property of the
// installed table, so one map with two tables is two worlds and neither is the
// map's fault. A nil Table, and a nil collection inside one, are both "no
// table": every search over them reports no match.
type Table struct {
	Units  data.Collection
	Humans data.Collection

	// Game selects how a placement resolves; the zero value is ROM1.
	Game base.Game

	// Rules are the game parameters the session runs under: the original
	// game's unless a mod changed them. The zero value is the original's.
	Rules sim.Rules

	// Mods describes the mods the session runs under; the zero value is none.
	// The save writer and reader read it to mark and check a save.
	Mods ModContext

	// Buildings is what a PLACED STRUCTURE resolves against, and it is read by
	// the footprint and health seed passes (structures.go, fromalm.go) — no unit
	// placement ever reaches it. It is one-based like the two above, but it is
	// subscripted rather than searched: a structure's key IS its entry's index.
	Buildings data.Collection

	// NPC is the scenario NPC lookup the first humans-band rung reads. It is a
	// CONCRETE type and not an interface like the three above, because it is a
	// lookup this tree builds rather than a view onto a collection somebody
	// else parsed — and because a nil one has to behave as "no registry" at
	// the call site, which a nil interface holding a nil pointer does not.
	//
	// It rides on the table rather than beside it for the reason the table
	// itself is an argument: what a placement resolves to is a property of the
	// installed files, and one map with two installs is two worlds.
	NPC *data.NPCDefs

	// Shapes, Materials and Weapons are what an EQUIPMENT NAME is worth. A
	// humans-band row names its own items and this is the only place a builder
	// can find out what they carry.
	//
	// THE THREE GO TOGETHER and are read as a set: a name's two leading words
	// select a row of the first two collections and the remainder a row of the
	// third, and each of the three supplies a factor the other two cannot. A
	// table holding some of them is answered as a table holding none — every
	// placement bare — rather than as one arming people off whichever factors
	// happen to be present, because a missing scale table's identity factor is
	// not "no scaling" but "scaled by one", and the shipped rows are between
	// three and five times what an item of that name actually carries.
	//
	// They ride here for the same reason the four above do: what a placement is
	// worth is a property of the installed files.
	Shapes, Materials data.ScaleTable
	Weapons           data.Collection

	// Armors and Shields are what an ARMOUR CELL and a SHIELD CELL of a
	// person's row resolve against: the same two classes data.ResolveArmor and
	// data.ResolveShield turn a name into, read off this table the same way
	// Weapons already is.
	//
	// EACH IS INDEPENDENT OF THE OTHER AND OF WEAPONS: a table missing Armors
	// answers armorItems() false and refuses only an armour cell; one missing
	// Shields answers shieldItems() false and refuses only a shield cell —
	// see those two accessors, below items(). The existing all-or-nothing rule
	// for Shapes, Materials and Weapons is UNCHANGED: every one of the three
	// classes reads both scale tables, so a table missing either still answers
	// every accessor false and every cell of every class refused, exactly as
	// items() already states for the weapon alone.
	Armors, Shields data.Collection

	// Spells is the installed Spells collection: what spellsFor (spell.go)
	// converts into the world's own []sim.SpellRule and hands to the world
	// FromALMWith and StartMission build. It rides here for the reason the four
	// above do — what a cast may target is a property of the installed files
	// — but it is READ ONCE rather than searched by a placement: a spell's id
	// is its own row's subscript (data.LoadSpells' own rule), not a class key a
	// unit or a human record names.
	Spells data.Collection

	// Magic names the ordered effect-key table. MagicItems is the class-14
	// item table whose row name supplies the item kind and whose first signed
	// parameter supplies stored value.
	Magic, MagicItems data.Collection

	// Names is the shipped item-name table (0151, ITEM-DISPNAME-036): a
	// code's raw sixteen bits looked up directly, not resolved through
	// Weapons, Armors or Shields. IT DOES NOT GATE ANYTHING (unlike the
	// all-or-nothing rule the three collections above share for a weapon
	// cell): a nil or empty Names answers every lookup false, and a caller
	// falls back to a different name for the code, never to refusing the
	// item itself — this is a display fact, not a placement one.
	Names data.ItemNames

	// HeroNames are the four hero pictures' installed names, raw install
	// bytes, in the character generator's picture order: male fighter, male
	// mage, female fighter, female mage. A hero started without the name field
	// takes the one of his picture. A table that read no names, or fewer than
	// four, holds four empty strings.
	HeroNames [4]string

	// composedNPC is the resolved Humans-row index for a composed scenario
	// NPC in this mission start. It is deliberately private: the registry
	// tokens need the entering player's archetype, so the answer belongs to a
	// start's shallow table copy and must not mutate the install-wide table.
	composedNPC map[int32]int
}

func (t *Table) units() data.Collection {
	if t == nil {
		return nil
	}
	return t.Units
}

func (t *Table) humans() data.Collection {
	if t == nil {
		return nil
	}
	return t.Humans
}

func (t *Table) npc() *data.NPCDefs {
	if t == nil {
		return nil
	}
	return t.NPC
}

func (t *Table) composedNPCIndex(id int32) (int, bool) {
	if t == nil {
		return 0, false
	}
	i, ok := t.composedNPC[id]
	return i, ok
}

// spells is Spells' own nil-safe read, on units' and humans' own ground:
// a nil Table names no Spells collection either.
func (t *Table) spells() data.Collection {
	if t == nil {
		return nil
	}
	return t.Spells
}

// items reports the three item collections, and whether the table carries
// ALL of them. A false is "this table cannot say what an item is worth",
// which is the only answer a partial set may give.
func (t *Table) items() (shapes, materials data.ScaleTable, weapons data.Collection, ok bool) {
	if t == nil || t.Shapes == nil || t.Materials == nil || t.Weapons == nil {
		return nil, nil, nil, false
	}
	return t.Shapes, t.Materials, t.Weapons, true
}

// shieldItems is items()'s own shape for a SHIELD cell: Shapes and
// Materials, the two scale tables a shield's resolve reads exactly as a
// weapon's does, and Shields in Weapons' own place. A table missing Shields
// answers false HERE ALONE — independently of Armors and of Weapons, a
// missing collection being a fact about one class and never about the walk
// — while a table missing Shapes or Materials answers false on every
// class's own accessor alike, items()'s own reason restated: without a scale
// table nothing of any class resolves.
func (t *Table) shieldItems() (shapes, materials data.ScaleTable, shields data.Collection, ok bool) {
	if t == nil || t.Shapes == nil || t.Materials == nil || t.Shields == nil {
		return nil, nil, nil, false
	}
	return t.Shapes, t.Materials, t.Shields, true
}

// armorItems is shieldItems' own shape for an ARMOUR cell: the same two
// scale tables, and Armors in Shields' own place, gated the same way and for
// the same reason.
func (t *Table) armorItems() (shapes, materials data.ScaleTable, armors data.Collection, ok bool) {
	if t == nil || t.Shapes == nil || t.Materials == nil || t.Armors == nil {
		return nil, nil, nil, false
	}
	return t.Shapes, t.Materials, t.Armors, true
}

// Arm names which of the four resolution paths a placement took. Every
// placement takes exactly one; three of the four may still reach no entry, and
// that is no match rather than an error.
type Arm uint8

const (
	// ArmNPC is the humans band's first rung: the flag word's bit 0 diverts the
	// placement through the scenario NPC lookup, whose definition id is then
	// searched DOWNWARD through the humans collection.
	ArmNPC Arm = iota
	// ArmServerID is the humans band's second rung: the record's own overriding
	// definition id, searched DOWNWARD through the humans collection. A record
	// that took the rung above never reaches it, however its own field reads.
	ArmServerID
	// ArmHumansByType is the humans band's last rung: the primary key as a type
	// id, over the humans collection.
	ArmHumansByType
	// ArmUnits is the whole band at or above the class-key floor: the units
	// collection on the key pair. It is the ONLY arm that can yield a stat
	// block, and it is reached whatever the flag word and the definition id
	// hold.
	ArmUnits
)

func (a Arm) String() string {
	switch a {
	case ArmNPC:
		return "npc"
	case ArmServerID:
		return "server-id"
	case ArmHumansByType:
		return "humans"
	case ArmUnits:
		return "units"
	}
	return fmt.Sprintf("arm(%d)", uint8(a))
}

// Resolution is what one placement resolved to: which arm ran, and the index the
// search reached. It is returned on its own and not only through a world,
// because counting arms over a map is exactly what says whether a table and a
// map agree, and building a world to find out would answer a different question.
type Resolution struct {
	Arm   Arm
	Index int
}

// Found reports whether the search reached an entry. The searched collections
// are one-based, so index 0 is the reserved entry and can never be a match.
func (r Resolution) Found() bool { return r.Index != data.NotFound }

const (
	// npcFlagBit is bit 0 of a placement's flag word.
	npcFlagBit = 1

	// defIDUnwritten is the fill a placement's definition-id field carries when
	// nothing wrote one. Zero means the same thing, and both must be excluded:
	// treated as an id, either would send every such placement down the
	// overriding arm and past the class key it does carry.
	defIDUnwritten uint32 = 0xcdcdcdcd

	// unitsKeyFloor is the class key at and above which a placement is a unit,
	// and below which it is a human. It is the spawner's own immediate — the
	// single compare that splits the record set before any flag is looked at —
	// and it decides which C++ class the original constructs, which is why it
	// stands outside the three tests below rather than among them.
	//
	// It replaced a wider band, `key < 0x40 && key ∉ {0x1a, 0x1b}`, read from
	// the SEARCH routine rather than from the spawner. The two disagree over
	// keys 0x1c..0x3f, no shipped record is known to sit there, and the
	// disagreement is reported to research rather than settled here (0065
	// provenance, Removed).
	unitsKeyFloor int32 = 0x1a
)

// Resolve says which arm a placement takes and what that arm reaches. It is the
// ONLY code that reads a placement record — the flag word, the definition id and
// the two class keys — and the searches it selects between never see any of them.
//
// THE ARMS ARE A LADDER AND THE CLASS KEY IS THE OUTERMOST RUNG. At or above
// unitsKeyFloor the placement is a unit and nothing else is consulted: its
// flag word and its definition id are never read, however they are set.
// Below it, the three humans rungs run in the order written, and a record
// that takes an earlier one can never reach its own identifier on a later
// one — which is the whole content of the ordering, and the thing a flat
// `if` chain over independent tests gets wrong. Two of `10.alm`'s three
// script-named units are decided by it, and the wrong order puts a generic
// peasant where the mission's dialogue names a witch.
//
// THE BAND TEST AND THE SEARCHES READ THE KEY DIFFERENTLY, and that is the
// contract rather than an inconsistency. The band compares the SIGNED word
// the record stores, because the original's compare is on the sign-extended
// value; each search takes its keys as their LOW BYTES, because the
// original's search does. So a key of 0x140 lands above the floor and then
// selects whatever 0x40 selects. No shipped map exercises either edge —
// the domain is 1..80 — so nothing measures the difference and the code
// has to say which it means.
//
// No match is no match. A placement whose key names nothing is answered with the
// arm it took and no entry, never with an error: a table and a map are two files
// that need not have been shipped together.
func Resolve(u alm.Unit, t *Table) Resolution {
	if t != nil && t.Game.Edition().SecondUnitKeys {
		return resolveROM2(u, t)
	}
	if int32(u.ClassID) >= unitsKeyFloor {
		return Resolution{Arm: ArmUnits,
			Index: data.FindUnit(t.units(), int32(uint8(u.ClassID)), int32(uint8(u.ClassSubID)))}
	}
	if u.Flags&npcFlagBit != 0 {
		// The npc subscript is the secondary key WHOLE, not truncated: the
		// original indexes the npc array with the record's own word and applies
		// the byte cast only inside the definition search, which this arm does
		// not reach. The lookup answers no entry for an absent section, a
		// section with no definition id, and the composition sentinel alike.
		if index, ok := t.composedNPCIndex(int32(u.ClassSubID)); ok {
			return Resolution{Arm: ArmNPC, Index: index}
		}
		sid, ok := t.npc().ServerID(int32(u.ClassSubID))
		if !ok {
			return Resolution{Arm: ArmNPC, Index: data.NotFound}
		}
		return Resolution{Arm: ArmNPC, Index: data.FindHumanByServerID(t.humans(), sid)}
	}
	if u.DefID != 0 && u.DefID != defIDUnwritten {
		// The id is compared as the dword the record stores, which is what
		// int32 of it is: the same 32 bits the column holds.
		return Resolution{Arm: ArmServerID, Index: data.FindHumanByServerID(t.humans(), int32(u.DefID))}
	}
	return Resolution{Arm: ArmHumansByType, Index: data.FindHumanByType(t.humans(), int32(uint8(u.ClassID)))}
}

const rom2PersonFlag = 0x10

func resolveROM2(u alm.Unit, t *Table) Resolution {
	if u.Flags&rom2PersonFlag != 0 {
		return Resolution{Arm: ArmServerID, Index: data.FindHumanByServerID(t.humans(), int32(u.ServerID))}
	}
	return Resolution{Arm: ArmUnits, Index: data.FindUnitByServerID(t.units(), int32(u.ServerID))}
}

// Adjust applies the difficulty setting to one definition and returns the
// result. It is a function of the definition and the value ALONE: it reads no
// world, no map and no placement, so two copies of one definition adjusted alike
// stay equal.
//
// The two multiplications are done in INTEGERS. The engine's own constants are
// two doubles followed by a truncation toward zero, and these forms agree with
// them exactly over the whole 0..65535 domain a stored maximum can occupy: 1.5
// is exact in binary, so the second is a truncating halving either way, and the
// first constant is representable only slightly ABOVE 0.66, so the floating
// product can only exceed the real one by around one part in 10^16 — far too
// small to carry a value over an integer boundary unless the real product is
// exactly an integer, which below 65536 happens only at multiples of 50, where
// both forms give the same number. No float enters this path, and the result
// crosses into hashed simulation state, which is why that argument is written
// down rather than assumed.
//
// The two additions are done in int32 where the engine stores 16 bits. The wrap
// is unreachable below a to-hit of 65486, which no column can hold beside its
// own 16-bit store.
func Adjust(d data.UnitDef, diff Difficulty) (data.UnitDef, error) {
	switch diff {
	case DifficultyEasy:
		d.HealthMax = d.HealthMax * 66 / 100
		d.Health = d.HealthMax
	case DifficultyNormal:
		// The identity, by having no arm rather than by an arm that computes
		// nothing.
	case DifficultyHard:
		d.ToHit += 50
		d.Defence += 50
		d.HealthMax = d.HealthMax * 3 / 2
		d.Health = d.HealthMax
	default:
		return data.UnitDef{}, fmt.Errorf("mapload: difficulty %d is not 1, 2 or 3", int32(diff))
	}
	return d, nil
}

// The definition-table codes of the three movement domains. They are the FILE's
// numbering and not the simulation's: this package maps between them, so a
// table whose codes moved could not silently renumber a shipped byte form, and
// the simulation's own zero value stays the ordinary ground mover.
const (
	movementGround = 1
	movementGhost  = 2
	movementAir    = 3
)

// domainFor is the movement domain a resolved definition names.
//
// It is TOTAL, and every value but the two non-ground codes answers ground —
// which is the original's own arm rather than a convenience: its selector has
// three tests and stores nothing when all three fail, leaving the constructor's
// ground mask standing. So a table cell nobody wrote, a cell holding 1, and a
// cell holding a code this build has never heard of all come out the same, and
// none of them is an error.
//
// That totality is also what carries the UNRESOLVED placement, without a branch
// of its own — and the reason has CHANGED while the answer has not. It used to
// be that such a placement arrived here as the ZERO definition, whose column is
// 0, and 0 is not one of the codes. It now arrives as the constructor's own
// definition, whose column is 1, and 1 is the ground code itself. Both roads
// reach ground, so no behaviour and no digest moved when the loader stopped
// taking the first; the sentence is rewritten rather than left standing because
// a rule whose conclusion is right and whose stated reason is false survives
// every review — everybody checks the conclusion.
//
// What the totality still buys is the thing worth having: one condition decides
// the domain, on the same value the rate and the eight combat numbers come off,
// so it cannot come to disagree with them.
func domainFor(d data.UnitDef) sim.Domain { return domainForCode(d.MovementType) }

// domainForCode is that mapping over the column itself, so the two definitions
// reach one implementation of it rather than each carrying a switch. The
// totality above is this function's; the sentence about what an unresolved
// placement reaches is the caller's.
func domainForCode(code int32) sim.Domain {
	switch code {
	case movementGhost:
		return sim.DomainGhost
	case movementAir:
		return sim.DomainAir
	}
	return sim.DomainGround
}

// definitionFor builds the ADJUSTED definition a placement resolved to, or
// reports that it resolved to none.
//
// Only a units match yields one. A placement that reached a humans entry, took
// the npc path or matched nothing gets no definition at all — the humans slot
// list is not published as a claim and a hero's health maximum is computed from
// body and experience rather than read from a column, so a stat block built off
// that arm would be an invention wearing a decode's clothes.
func definitionFor(u alm.Unit, t *Table, diff Difficulty) (def data.UnitDef, worn [sim.EquipSlots]sim.ItemInstance, ok bool, err error) {
	r := Resolve(u, t)
	if r.Arm != ArmUnits || !r.Found() {
		return data.UnitDef{}, [sim.EquipSlots]sim.ItemInstance{}, false, nil
	}
	def, worn, err = unitRowDefinition(t, r.Index, diff)
	if err != nil {
		return data.UnitDef{}, [sim.EquipSlots]sim.ItemInstance{}, false, err
	}
	return def, worn, true, nil
}

// unitRowDefinition is the adjusted definition of the Units row at index and
// the worn set its weapon string arms.
func unitRowDefinition(t *Table, index int, diff Difficulty) (data.UnitDef, [sim.EquipSlots]sim.ItemInstance, error) {
	c := t.units()
	d, err := data.NewUnitDef(c.EntryName(index), c.EntryParams(index))
	if err != nil {
		return data.UnitDef{}, [sim.EquipSlots]sim.ItemInstance{}, err
	}
	// UNIT-DIFF-002 copies the authored live ToHit into General during the
	// Units stream. UNIT-GATE-013 applies Hard's +50 later, at placement, to
	// live ToHit alone. Snapshot General before Adjust so the placement bonus
	// is not copied into the equip modifier and added twice.
	general := d.ToHit
	// Applied ONCE, before anything is built from the definition, so the value
	// cannot be applied twice by a second caller reading the same entry.
	d, err = Adjust(d, diff)
	if err != nil {
		return data.UnitDef{}, [sim.EquipSlots]sim.ItemInstance{}, err
	}
	// The class row's own equipment, off unitWeapon's own read of this row's
	// two trailing strings — a units row carries no reach column of its own
	// and no fold of its own either, so this is the only place a creature's row
	// meets the weapon it names.
	//
	// THE FOLD ADDS, IT DOES NOT ASSIGN. The melee arm carries DamageBase,
	// DamageSpread, weapon ToHit and Defence on top of the row. The ranged arm
	// instead adds the row-copied General accuracy and, for exact types 11/12,
	// its byte pair to SecondaryDamage.
	// Reach and both cadence halves are assigned from every resolved weapon;
	// SkillSlot is assigned on that same common arm, selecting a supported
	// melee kind or clearing a ranged row to General (1039). Writing these
	// eight folded fields back onto d therefore preserves both the additions
	// and the assignments; the spell pair follows from the same block below.
	strs := c.EntryStrings(index)
	w, weaponItem, weaponErr := unitWeaponItem(strs, t)
	if weaponErr != nil {
		return data.UnitDef{}, [sim.EquipSlots]sim.ItemInstance{}, weaponErr
	}
	if w != nil {
		cb := data.FoldWeapon(d.Combat(), w, general)
		d.DamageBase, d.DamageSpread = cb.DamageBase, cb.DamageSpread
		d.SecondaryDamage = cb.SecondaryDamage
		d.ToHit, d.Defence = cb.ToHit, cb.Defence
		d.AttackChargeTime, d.AttackRelaxTime = cb.AttackChargeTime, cb.AttackRelaxTime
		d.Reach = cb.Reach
		d.SkillSlot = cb.SkillSlot
		d.SpellName, d.SpellPower = cb.SpellName, cb.SpellPower
	} else {
		d.Reach = unitReach(strs, t)
	}
	if !weaponItem.Empty() {
		weaponItem = sourceWeaponItem(weaponItem, *w, t)
		weaponItem = SourceEquippedItem(weaponItem, t)
	}
	return d, wornFromWeaponItem(weaponItem), nil
}

func wornFromWeaponItem(item sim.ItemInstance) [sim.EquipSlots]sim.ItemInstance {
	var worn [sim.EquipSlots]sim.ItemInstance
	if slot, ok := data.EquipSlotFor(data.ItemCode(item.Code)); ok {
		worn[slot-1] = item.Clone()
	}
	return worn
}

// wornFromWeapon places w's own resolved code into its own slot — always
// slot 1, because weaponItemClass is a class constant and not a per-row
// value (weapon.go's own doc on that constant) — or leaves every slot
// empty for no weapon at all (0128 plan D-2).
//
// IT IS fromalm.go's OWN CONVERSION, MOVED HERE: spawnBlock now carries the
// WHOLE worn array rather than a bare code for a caller in that file to
// convert, so the conversion has to happen wherever the array is built —
// which for the units arm is here, because THE UNITS ARM RESOLVES ONE
// WEAPON AND NOTHING ELSE and this is the whole of what it contributes to a
// worn set, unlike wearRow's ten cells for a person.
//
// THE SLOT IS READ OFF THE CODE, NOT WRITTEN AS A LITERAL: data.EquipSlotFor
// is asked exactly as wearRow asks it of a shield's or an armour's own
// code, so this is not a second place that decides where a weapon goes.
func wornFromWeapon(w *data.Weapon) [sim.EquipSlots]uint16 {
	var worn [sim.EquipSlots]uint16
	code := weaponCode(w)
	if slot, ok := data.EquipSlotFor(data.ItemCode(code)); ok {
		worn[slot-1] = code
	}
	return worn
}

// startingLoadout retains the prior Human-row policy. Units use their held
// weapon directly; a zero Suitable value suppresses its death drop instead.
func startingLoadout(_ string, w *data.Weapon, t *Table, worn [sim.EquipSlots]uint16) [sim.EquipSlots]uint16 {
	if !carriable(w, t) {
		worn[cellWeapon] = 0
	}
	return worn
}

func startingLoadoutItems(_ string, w *data.Weapon, t *Table, worn [sim.EquipSlots]sim.ItemInstance) [sim.EquipSlots]sim.ItemInstance {
	if !carriable(w, t) {
		worn[cellWeapon] = sim.ItemInstance{}
	}
	return worn
}

// weaponCarrySlot is the Weapons row's own column ITEM-DEATH-012 gates the
// SLOT 1 unequip on: `L04625` pushes `0xf` and compares the answer against
// zero, and a weapon whose column is zero is never taken off the body. It is
// a runtime column number, i.e. the parameter array's own index, so it is the
// column the file's title array names one to its right — `sutableFor`, which
// is a name and not a decode: what the column MEANS is published Unknown and
// only the behaviour is claimed.
const weaponCarrySlot = 0xf

const weaponHandsColumn = 0xe

// weaponHands reads w's authored Hands cell. A nil weapon, an absent weapons
// collection and a short row report no value; each caller decides whether its
// own rule is positive or negative.
func weaponHands(w *data.Weapon, t *Table) (int32, bool) {
	if w == nil {
		return 0, false
	}
	_, _, weapons, ok := t.items()
	if !ok {
		return 0, false
	}
	row := int(w.Row)
	if row < 0 || row >= weapons.Len() {
		return 0, false
	}
	p := weapons.EntryParams(row)
	if len(p) <= weaponHandsColumn {
		return 0, false
	}
	return p[weaponHandsColumn], true
}

// WeaponBlocksShield is the inverse view of WeaponAllowsShield: a ranged
// weapon, or one in the decoded Hands=2 population, cannot share the held
// slots with a shield.
func WeaponBlocksShield(code data.ItemCode, t *Table) bool {
	return !WeaponAllowsShield(code, t)
}

// carriable applies only to Human-row starting loadouts. A missing Suitability
// column keeps the existing carried-item behaviour.
func carriable(w *data.Weapon, t *Table) bool {
	if w == nil {
		return false
	}
	_, _, weapons, ok := t.items()
	if !ok {
		return false
	}
	row := int(w.Row)
	if row < 0 || row >= weapons.Len() {
		return true
	}
	p := weapons.EntryParams(row)
	if len(p) <= weaponCarrySlot {
		return true
	}
	return p[weaponCarrySlot] != 0
}

// definitionForHuman builds the definition a humans-band placement resolved to,
// or reports that it resolved to none.
//
// Only a humans-band arm that REACHED AN ENTRY yields one. Which of the three
// rungs found it says nothing about what the entry is worth, so there is no test
// of the arm here beyond its not being the units one — the ladder's whole job
// was to pick an entry and it has finished.
//
// THE DIFFICULTY DOES NOT APPEAR, and that is the requirement rather than an
// omission. The thing being reconstructed steps over the entire three-way
// adjustment for this class, so the setting reaches this arm by there being
// no call rather than by an arm that computes nothing.
//
// A row this contract cannot read is refused by name and takes the world with
// it, exactly as a units row carrying an unmodelled damage arm does: a person
// shipped with half a definition is the failure this whole story exists to
// remove.
// IT ALSO HANDS BACK THE STARTING WORN SET AND ITS CONTAINER OVERFLOW
// (0128, widening the starting-equipment hotfix), definitionFor's own two
// results one band over — plus wep, the weapon pointer, for the same
// reason it always has: this band's caller still folds it into a combat
// block itself, the units band folds inside definitionFor and this one
// does not, so the results are the same resolution read twice, not two
// resolutions.
//
// wearRow READS ALL TEN CELLS, unlike unitWeapon's one — a person's row
// names a shield and eight armour cells beside his weapon, and this band is
// the only one that ever resolves them, on this contract's own ground: a
// units row carries none of the three.
func definitionForHuman(u alm.Unit, t *Table) (def data.HumanDef, w *data.Weapon, worn [sim.EquipSlots]sim.ItemInstance, carried []sim.ItemInstance, ok bool, err error) {
	r := Resolve(u, t)
	if r.Arm == ArmUnits || !r.Found() {
		return data.HumanDef{}, nil, [sim.EquipSlots]sim.ItemInstance{}, nil, false, nil
	}
	c := t.humans()
	d, err := data.NewHumanDef(c.EntryName(r.Index), c.EntryParams(r.Index))
	if err != nil {
		return data.HumanDef{}, nil, [sim.EquipSlots]sim.ItemInstance{}, nil, false, err
	}
	worn, rawCarried, wep, err := HumanRowEquipment(c.EntryStrings(r.Index), t)
	if err != nil {
		return data.HumanDef{}, nil, [sim.EquipSlots]sim.ItemInstance{}, nil, false, err
	}
	return d, wep, worn, rawCarried, true, nil
}

// HumanRowEquipment resolves one Humans row's own complete starting
// equipment. It preserves item effects and the row's own weapon, including a
// castSpell attachment. GeneratedWornSet is the separate character-generation
// path that replaces the row's weapon with a chosen weapon.
func HumanRowEquipment(cells []string, t *Table) (worn [sim.EquipSlots]sim.ItemInstance, carried []sim.ItemInstance, weapon *data.Weapon, err error) {
	rawWorn, carried, weapon, err := wearRowItems(cells, t)
	if err != nil {
		return worn, nil, nil, err
	}
	return startingLoadoutItems("", weapon, t, rawWorn), carried, weapon, nil
}

// GeneratedWornSet is a GENERATED character's own worn-set resolution (AC-1,
// AC-2, AC-15, plan D-4): his base archetype row's ten equipment cells, with
// the weapon character generation handed him substituted for the row's own
// weapon cell, resolved through the SAME two functions a PLACED person's row
// already resolves through, one band up.
//
// IT DOES NOT RE-IMPLEMENT wearRow OR startingLoadout, AND IT WRITES NO CODE
// INTO A SLOT ITSELF: cells is copied into a row of cellCount cells —
// never fewer, so a shorter slice still reaches every position wearRow reads
// by — the copy's weapon cell (cellWeapon, the SAME named cell wearRow
// already keys the weapon class on, not a slot number chosen here) is
// overwritten with weapon, and the whole ten then go through wearRow exactly
// as definitionForHuman's row does, and through startingLoadout exactly as
// definitionForHuman already applies it.
//
// STARTINGLOADOUT'S OWN TEMPLATE-NAME GATE IS PASSED THE EMPTY STRING,
// NEVER A ROW'S NAME: a generated character has no template name at all —
// that gate is ITEM-DEATH-012's own corpse rule, and this function builds
// no corpse — so handing it a name here would apply a placement's rule to
// a character who was never placed.
func GeneratedWornSet(cells []string, weapon string, t *Table) (worn [sim.EquipSlots]uint16, carried []uint16) {
	row := make([]string, cellCount)
	copy(row, cells)
	row[cellWeapon] = weapon

	rawWorn, carried, wep := wearRow(row, t)
	worn = startingLoadout("", wep, t, rawWorn)
	return worn, carried
}

// SpeakerOutfit is the worn set the row behind one dialogue speaker record
// states: the ten equipment cells of a `Humans` row, resolved through the
// SAME wearRow and startingLoadout a PLACED person of that row already goes
// through, so a speaker dressed by this function wears exactly what a
// placement of the same row wears — including the shield gate.
//
// THE ROW IS THE SECTION'S OWN WHERE IT NAMES ONE. sub is the `npc<n>`
// subscript; t.npc() is the same lookup a type-6 npc-arm placement resolves
// through (`MISSION-ARM-006`), and the row it names is found by server id
// exactly as that arm finds it. Mission 40's `npc25` therefore reaches
// `Humans` row 42, `PC_Paladin`, and its seven cells (`DLG-DRESS-024`).
//
// OTHERWISE THE ARCHETYPE ROW THE RECORD'S OWN TWO FLAGS CHOOSE, through
// data.ChargenBase — the one place this tree turns a (mage, female) pair
// into a shipped row, and the row a GENERATED character of that archetype
// starts from. 82 of the 105 shipped sections name no definition id at all,
// so this arm is the ordinary case rather than the exotic one.
//
// A ROW THAT DRESSES HIM IN NOTHING FALLS THROUGH TO THE ARCHETYPE ROW TOO,
// and that arm exists because the shipped collection uses it. Measured over
// the EN install, 53 records compose a figure and seven of them name a row
// of their own whose ten equipment cells are all empty — `PC_Treyrack`,
// `PC_Lakhlana`, `M10_Witch`, `M30_Healer`, `M40_Hima`, `M151_Lord` and
// `M71_PoorGuy`. Stopping at the named row would leave those seven naked,
// which is the state the owner ruled wrong. The archetype rows are dressed:
// `PC_Danath` fills slots 1, 7 and 12, which is a dressed figure with no
// helmet.
//
// IT IS ONLY EVER ASKED FOR A SPEAKER THE WORLD HAS NO LIVE ACTOR FOR. A
// live speaker wears what the world says he wears, which this function does
// not read and could not (`DLG-SPEAKER-022`).
//
// carried is dropped rather than returned: a synthesised speaker has no
// container, and the overflow a row's cells cost belongs to a person who
// exists.
//
// ok MEANS "THIS BUILD DRESSED HIM", not "a row was found": a table naming no
// row and a table whose every candidate row is empty answer alike, because the
// caller has the same thing to do about both — compose the bare figure, which
// is the original's own picture. row names the entry the worn set came off,
// for a report that has to say where an outfit came from; it is the empty
// string exactly when ok is false.
func SpeakerOutfit(sub int32, mage, female bool, t *Table) (worn [sim.EquipSlots]uint16, row string, ok bool) {
	c := t.humans()
	if c == nil {
		return worn, "", false
	}
	if sid, named := t.npc().ServerID(sub); named {
		if i := data.FindHumanByServerID(c, sid); i != data.NotFound {
			if w, name := rowOutfit(c, i, t); w != ([sim.EquipSlots]uint16{}) {
				return w, name, true
			}
		}
	}
	if _, i, found := data.ChargenBase(c, mage, female); found {
		if w, name := rowOutfit(c, i, t); w != ([sim.EquipSlots]uint16{}) {
			return w, name, true
		}
	}
	return worn, "", false
}

// rowOutfit is one Humans row's own worn set and its name, through the two
// functions definitionForHuman already puts a placed person's row through.
func rowOutfit(c data.Collection, index int, t *Table) ([sim.EquipSlots]uint16, string) {
	name := c.EntryName(index)
	rawWorn, _, wep := wearRow(c.EntryStrings(index), t)
	return startingLoadout(name, wep, t, rawWorn), name
}

const (
	cellWeapon    = 0
	cellShield    = 1
	cellArmorFrom = 2
	cellCount     = 10
)

// wearRow turns one row's ten equipment cells into the worn set they arm,
// the container overflow they cost, and the weapon pointer the combat fold
// still needs. It REPLACES firstWeapon rather than standing beside it:
// firstWeapon's own "first cell that resolves wins" predicate had nothing
// left to do once every cell is read by POSITION instead — the fact that
// an empty cell costs nothing and is never mistaken for a name that failed
// is the one firstWeapon's own doc recorded that is still true, and it still
// holds here, for all ten cells rather than one.
//
// IT IS THE ONLY PLACE THE CELL -> CLASS -> SLOT MAP IS WRITTEN. Cell 0
// resolves as a weapon into slot 1; cell 1 resolves as a shield into slot 2
// alone or beside a weapon that leaves a hand free, and into the container
// beside one that does not; cells 2 through 9 each resolve as armour, into
// the slot its OWN ROW names (data.EquipSlotFor on the resolved code, never
// the cell index) or into the container when that slot is not one of the
// twelve. AN EMPTY CELL IS SKIPPED and A NAME THAT RESOLVES TO NO ROW IS
// DROPPED — neither worn nor carried: t.items(), t.shieldItems() and
// t.armorItems() all answer a missing collection the same way a resolver
// answers a name with no row, so a table naming a class nowhere refuses every
// cell of that class exactly as a name this build cannot parse does.
//
// CELLS RUN IN ASCENDING ORDER AND THE LOOP CANNOT BE REORDERED: the shield
// step reads slot 1 exactly as cell 0 left it in worn, which is what lets
// the compatibility gate inspect the resolved weapon already worn rather
// than the row's raw name a second time.
//
// THE WEAPON POINTER RIDES BESIDE THE WORN SET rather than being searched
// for again: definitionForHuman's own caller folds it into a combat block,
// and it is the SAME resolution — cell 0's — that armed slot 1, not a
// second search over the row (plan D-2).
//
// NO DAMAGE, NO DEFENCE, NO ABSORPTION MOVES HERE: this function writes item
// codes into a slot array and a container slice, nothing else — what a
// worn piece contributes to any derived number is not decoded and this
// function does not touch one.
func wearRow(names []string, t *Table) (worn [sim.EquipSlots]uint16, carried []uint16, weapon *data.Weapon) {
	wornItems, carriedItems, weapon, _ := wearRowItems(names, t)
	for i := range wornItems {
		worn[i] = wornItems[i].Code
	}
	for _, item := range carriedItems {
		carried = append(carried, item.Code)
	}
	return worn, carried, weapon
}

func wearRowItems(names []string, t *Table) (worn [sim.EquipSlots]sim.ItemInstance, carried []sim.ItemInstance, weapon *data.Weapon, err error) {
	if len(names) > cellWeapon && names[cellWeapon] != "" {
		if _, _, _, ok := t.items(); ok {
			if w, item, resolveErr := resolveWeaponCell(names[cellWeapon], t); resolveErr == nil {
				weapon = &w
				if slot, ok := data.EquipSlotFor(w.Code); ok {
					worn[slot-1] = item
				}
			} else if strings.Contains(names[cellWeapon], "{") {
				return worn, nil, nil, resolveErr
			}
		}
	}

	if len(names) > cellShield && names[cellShield] != "" {
		if _, _, _, ok := t.shieldItems(); ok {
			if s, item, resolveErr := resolveShieldCell(names[cellShield], t); resolveErr == nil {
				if slot, ok := data.EquipSlotFor(s.Code); ok {
					if worn[cellWeapon].Empty() || WeaponAllowsShield(data.ItemCode(worn[cellWeapon].Code), t) {
						worn[slot-1] = item
					} else {
						carried = append(carried, item)
					}
				}
			} else if strings.Contains(names[cellShield], "{") {
				return worn, nil, weapon, resolveErr
			}
		}
	}

	_, _, _, ok := t.armorItems()
	for i := cellArmorFrom; i < cellCount && i < len(names); i++ {
		if names[i] == "" {
			continue
		}
		if !ok {
			continue
		}
		a, item, resolveErr := resolveArmorCell(names[i], t)
		if resolveErr != nil {
			if strings.Contains(names[i], "{") {
				return worn, nil, weapon, resolveErr
			}
			continue //
		}
		if slot, ok := data.EquipSlotFor(a.Code); ok {
			worn[slot-1] = item
		} else {
			carried = append(carried, item) //
		}
	}

	return worn, carried, weapon, nil
}

// unitReach is unitWeapon's own shape over a different question: not "what
// does this row carry" but "how far does it reach", so it takes WeaponRange
// in the place ResolveWeapon stands above — the same three-collection
// guard, the same skip of an empty cell, the same "first cell that resolves
// wins" (the UNITS band's own rule; a person's row is wearRow's, above, and
// reads every cell by position instead) — and answers 1, the constructor's
// own floor, when nothing does.
//
// A unit row carries two equipment strings and no shipped one names both, so
// "the first that resolves" and "the one there is" agree over the whole
// table.
func unitReach(names []string, t *Table) int32 {
	shapes, materials, weapons, ok := t.items()
	if !ok {
		return 1
	}
	for _, n := range names {
		if n == "" {
			continue
		}
		if r, ok := data.WeaponRange(n, shapes, materials, weapons); ok {
			return r
		}
	}
	return 1
}

// unitWeapon is a class row's own equipment resolved as a WEAPON, off the
// SAME trailing strings unitReach reads — the same three-collection guard,
// the same skip of an empty cell, the same "first cell that resolves under
// ResolveWeapon wins" the UNITS band has always searched by, because a
// creature's row names its equipment as two candidate cells tried in turn
// rather than wearRow's ten cells fixed by POSITION, and this is the same
// search over it, only for a caller that means to fold what it finds rather
// than read one column off it.
//
// A NIL RESULT IS NOT "THIS ROW IS BARE": it is "ResolveWeapon could not
// take any cell", which is true of a genuinely empty row and of one this
// build's name resolution still cannot parse alike. definitionFor does not
// have to tell the two apart — its own fallback to unitReach answers both
// the same way a definition already did before this story.
func unitWeapon(names []string, t *Table) *data.Weapon {
	shapes, materials, weapons, ok := t.items()
	if !ok {
		return nil
	}
	for _, n := range names {
		if n == "" {
			continue
		}
		w, err := data.ResolveWeapon(n, shapes, materials, weapons)
		if err != nil {
			continue
		}
		return &w
	}
	return nil
}

func unitWeaponItem(names []string, t *Table) (*data.Weapon, sim.ItemInstance, error) {
	if _, _, _, ok := t.items(); !ok {
		return nil, sim.ItemInstance{}, nil
	}
	for _, name := range names {
		if name == "" {
			continue
		}
		weapon, item, err := resolveWeaponCell(name, t)
		if err != nil {
			if strings.Contains(name, "{") {
				return nil, sim.ItemInstance{}, err
			}
			continue
		}
		return &weapon, item, nil
	}
	return nil, sim.ItemInstance{}, nil
}

// rosterTemplate is the PartyMember a placement would become if it crossed a
// mission boundary: the person row's four statistics, its profile, its
// spellbook, its weapon and its own name, with the placement's class key and
// a stable identity built from the entity id it was minted with.
//
// IT IS ONLY THE MINT'S INPUTS. Nothing here describes what has happened to the
// actor — no health, no experience, no pack and no worn set — because all of
// that is read off the live entity at the boundary and folded into a Carry.
// This half is what the entity cannot say and the boundary cannot recover.
//
// THE IDENTITY IS THE RUNTIME ID HE JOINED UNDER (0159 D-11), written out
// rather than left to OwnParty's inference, so it does not depend on where in
// the party he ends up. The original keeps a handed-over actor's runtime id
// across the boundary; this is that id in the one form this tree carries
// identity in.
//
// A PLACEMENT THAT TOOK THE SCENARIO NPC ARM ALSO CARRIES THAT RECORD'S OWN
// SUBSCRIPT, which is the field the town screen's figure lookup and the
// localized-name lookup already key on. A placement that took either of the
// other two person arms carries zero there, which is what every character not
// created from an npc record already carries.
//
// He is a PERSISTENT ROSTER MEMBER AND NOT A PRIMARY CHARACTER: PlayerCharacter
// is set, StartingHero is not, Temporary is not, and no mercenary type is
// written. The hand-over routine performs neither of the two writes that would
// make him primary — it writes no primary-character pointer and allocates no
// fresh runtime id — so nothing here may claim he is one.
//
// ok IS FALSE FOR EVERY PLACEMENT THAT RESOLVED THROUGH A CREATURE ROW OR
// THROUGH NOTHING. There is no person row to template from, and neither can
// cross the boundary anyway.
func rosterTemplate(u alm.Unit, t *Table, id sim.EntityID) (PartyMember, bool) {
	d, w, worn, carried, ok, err := definitionForHuman(u, t)
	if err != nil || !ok {
		return PartyMember{}, false
	}
	r := Resolve(u, t)
	dir, face := placedFigure(u, r, d, t)
	p := PartyMember{
		ID:                 fmt.Sprintf("join:%d", uint32(id)),
		Name:               personRowName(u, t),
		PlayerCharacter:    true,
		Class:              d.TypeID,
		Mage:               dir.Mage(),
		Profile:            d.Profile(),
		FigureDir:          string(dir),
		FigureFace:         face,
		Hero:               d.Hero(),
		KnownSpells:        d.KnownSpells,
		Worn:               itemEquipmentCodes(worn),
		Carried:            itemInstanceCodes(carried),
		WornItems:          cloneItemEquipment(worn),
		CarriedItems:       cloneItemInstances(carried),
		Weapon:             w,
		SuppressCorpseLoot: SuppressesCorpseLoot(personRowName(u, t)),
	}
	if r.Arm == ArmNPC {
		p.CompanionNPC = int(u.ClassSubID)
	}
	return p, true
}

// personRowName is the name of the row a person placement resolved to — the
// template name the collection itself stores, which is the only name this tier
// has for a placement. A localized display name, where an install ships one, is
// resolved a tier up off CompanionNPC.
func personRowName(u alm.Unit, t *Table) string {
	r := Resolve(u, t)
	if r.Arm == ArmUnits || !r.Found() {
		return ""
	}
	return t.humans().EntryName(r.Index)
}

func TableRules(t *Table) sim.Rules { return t.rules() }

// rules is the table's rules; a nil table runs under the original game's.
func (t *Table) rules() sim.Rules {
	if t == nil {
		return sim.Rules{}
	}
	return t.Rules
}

// ModContext is the mod set a session runs under.
type ModContext struct {
	// Set is the resolved mod set; empty when no mod is active.
	Set mod.Set
	// AcceptUnmarked lets a save without a mod mark start under the active
	// mods; without it such a save is refused.
	AcceptUnmarked bool
	// Items are the items the mods added to the item tables.
	Items []ModItem
	// Characters are the characters the mods edited, for the names the panels
	// show.
	Characters []ModCharacter
	// Companions are the join conditions the mods declare.
	Companions mod.CompanionData
	Spells     mod.SpellData
}

// ModItem is an item a mod added, in the form the save writer and the shop
// read. A save holds the stand-in code in the item's place; Code is the item's
// own.
type ModItem struct {
	Mod, Key string
	// Code and Row are the item's own code and its row in the item table.
	Code uint16
	Row  uint8
	// StandInCode and StandInRow are the original item a save holds instead.
	StandInCode uint16
	StandInRow  uint8
	// Price is the shop price of one unit; Stock puts one unit on every shop's
	// armour shelf.
	Price int32
	Stock bool
	// Layer is LayerUnder or LayerOver for a clothing layer, zero for an
	// ordinary item; Anchor is the equipment slot, 1 to 12, a layer is placed
	// against.
	Layer  int8
	Anchor uint8
}

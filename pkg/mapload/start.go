package mapload

import (
	"againrom/pkg/data"
	"againrom/pkg/formats/alm"
	"againrom/pkg/random"
	"againrom/pkg/rules"
	"againrom/pkg/sim"
)

// StartSeed is the seed every start's own draws begin at.
//
// It is a fixed named constant for the reason Seed is one: what a start chooses
// must follow from a deterministic input alone, so one map and one party yield
// one world in every process and on every machine. The value is ours and
// arbitrary — the ASCII of "misstart".
//
// It is DISTINCT FROM Seed on purpose. Nothing consumes the world's own
// generator today; the moment something does, two sequences begun at one seed
// would agree value for value, and a start's drop cell would predict the
// simulation's first draw. One constant is a cheap price for that not being a
// property of this tree.
const StartSeed uint64 = 0x6D69737374617274

// The fallback's range, per axis, when the map authorises no usable cell: 30 to
// 100 INCLUSIVE. The published form is `draw(0x46) + 0x1e` over a generator whose
// bound is inclusive, so the span is 71 values and not 70 — a clause the research
// corrected once, and the reason the span is written as a bound rather than as a
// width here.
const (
	fallbackLow  int32 = 30
	fallbackSpan int32 = 70
)

// PartyMember is one unit a start places for the player.
//
// IT CARRIES A HERO, AND ITS COMBAT NUMBERS ARE DERIVED FROM HIM. Until 0078
// this type held a class key and nothing else, on the ground that a party
// fighting at numbers this tree invented would be a fabrication. That was right
// and it is retired rather than reversed: the numbers are no longer invented,
// they are folded out of four statistics and one weapon by the original's own
// arithmetic, and both inputs come from the caller — the statistics from
// character generation's start, the weapon from the installed definition table.
//
// AND HIS MOVEMENT SPEED IS DERIVED FROM HIM TOO, off his Reaction by the
// original's own branch. It was the class table's default until then, which
// meant the one statistic that decides how fast he walks was consulted
// nowhere.
//
// AND SO ARE HIS TWO POOLS, AS OF 0119-chargen — the health maximum and
// the mana maximum come off the SAME recompute the eight combat numbers, the
// rate and the sight radius already came off, over the profile below. A
// member carrying no profile — the zero value, which is every party member
// this tree minted before this story — still gets SpawnHP on health and
// nothing on mana: the divergence 0078 disclosed is closed only for a member
// who carries one. HIS REACH NO LONGER BELONGS ON EITHER LIST — 0105 is
// the story for it, and the mint below writes the fold's own answer where
// nothing was written at all.
//
// THE ZERO VALUE IS THE PARTY THIS TREE PLACED BEFORE 0078, and it arrives
// there through the same arithmetic rather than through a fallback: a hero with
// no statistics holding no weapon derives five zeroes and the bare cadence,
// which is exactly what a placement resolving to no definition took.
type PartyMember struct {
	// ID is this member's stable identity across maps, selection changes and
	// save/load. Entity ids are deliberately not used for this: a map mints a
	// fresh entity id every time it starts, while the person does not become a
	// different person at that boundary.
	//
	// Older callers may leave ID empty. OwnParty assigns those members a
	// deterministic identity before a party crosses an ownership boundary.
	ID string

	// Temporary is membership metadata, not character state. A temporary
	// member still owns an independent Hero, Carry, equipment and appearance;
	// this bit only says whether campaign continuity keeps the membership.
	Temporary bool

	// Name is this player character's individual display name. It is loader and
	// presentation state and does not enter the canonical simulation form.
	Name string

	// PlayerCharacter distinguishes persistent full heroes from mercenaries.
	PlayerCharacter bool

	// StartingHero identifies the one primary character without making party
	// order carry that meaning.
	StartingHero bool

	// MercenaryType is zero for a full hero and the 1-based mercenary type for
	// a hired unit.
	MercenaryType uint8
	DefinitionRow uint8

	// SuppressCorpseLoot is the canonical template's death-container policy.
	// It stays with party identity because a joined or hired actor may cross a
	// mission boundary before the simulation next needs the policy.
	SuppressCorpseLoot bool

	// CompanionNPC is the scenario npc record that added this persistent
	// companion. Zero is a character not created by AddHero.
	CompanionNPC int

	// Class is the key the entity carries, the same field a placement's class
	// key lands in. It is what a renderer resolves art through.
	//
	// IT IS DERIVED FROM Body BY THE CALLER, and it is a field rather than a
	// method for the reason the field has always existed: a placement's key
	// arrives in it too, and one entity-minting loop reads one field. Until
	// 0085 it was the caller's outright, on the ground that nothing decoded said
	// what a hero's was. Something does: a player's character's drawn class is
	// not carried at all but produced from what he visibly wears, and this field
	// now holds that production's answer rather than a chosen number.
	Class int32

	// Body is the body name this member is DRAWN AS, and the value Class was
	// derived from.
	//
	// It is a LOADER INPUT and reaches no entity, no byte form and no digest —
	// the hero's statistics are here on the same terms. It is carried because a
	// hero's pixels do not come through the class record's own art address: the
	// sheet is composed from this name, while the record the name resolved to
	// supplies only the geometry. So the two are needed together and a consumer
	// holding the key alone cannot recover this.
	//
	// The empty name is a member drawn through its class record like any placed
	// unit, which is what every party assembled before 0085 was.
	Body string

	BodyDir string

	// Mage is whether this member is drawn and armed as a mage — a fact this
	// struct carries on its OWN field rather than reads off Profile.Fighter, on
	// D-5's own rejection: that flag's meaning is this project's own contested
	// reading of which archetype the pool graph names, and binding both the
	// weapon and the picture to it would make one authored choice decide two
	// unrelated things.
	//
	// It is a LOADER INPUT on Body's own terms above: it reaches no entity, no
	// byte form and no digest.
	Mage bool

	// Profile is the three inputs the derived-stat graph reads that are not a
	// statistic, a skill or an item (spec "The profile"): the class flag,
	// whether the health column is present, whether the mana column is
	// present. A generated character's come off the shipped row he starts
	// from (HumanDef.Profile, chargenbase.go) and nowhere else; nothing here
	// invents one.
	//
	// It is a LOADER INPUT, on Body's own terms above: it reaches no entity, no
	// byte form and no digest.
	//
	// The zero value is the profile every party member in this tree carried
	// before 0119-chargen: no flag, neither column present. THAT IS THE MINT'S
	// OWN DECISION, NOT THE GRAPH'S: as of 0133-person-health the graph derives
	// a real HealthMax for such a member whenever his Body is nonzero
	// (recompute.go step 3 reads no column), and PartySpawn below is what still
	// holds him at the provisional number instead of that real one.
	Profile data.Profile

	// FigureDir is the directory the inventory window's figure is composed
	// from, chosen from a generated character's class and sex.
	//
	// It is a LOADER INPUT, on Body's own terms above, and reaches no
	// entity, no byte form and no digest. UNLIKE Body, NOTHING IN THIS
	// STORY READS IT YET: the figure builder that consumes it is a later
	// task of this same story (0119-chargen T5), and until it lands the
	// field is carried and unread — the same state Body itself sat in
	// between 0085's two tasks.
	//
	// The empty string is what every party member in this tree carries
	// today, and it is the state the figure builder falls back from.
	FigureDir string

	// FigureFace is the inventory figure's face column: a generated character's
	// own base row's face, or 1 when there is no base row.
	//
	// It is a LOADER INPUT, on Body's own terms above, and reaches no
	// entity, no byte form and no digest. Like FigureDir beside it, nothing
	// in this story reads it yet — the figure builder of 0119-chargen T5
	// does.
	FigureFace int

	// Hero is the four statistics and six skill levels this member fights from.
	Hero data.Hero

	// OriginalHuman is optional source-backed current city state. Old native
	// saves omit it and keep their previous derivation behavior.
	OriginalHuman *OriginalHuman

	HiredRotationSpeed int32

	// KnownSpells is the spellbook this member starts with: the bitmask his own
	// base row states or the previous mission carried, bit i set meaning he
	// knows spell i (0127 FR-4a).
	//
	// UNLIKE Body, Profile, FigureDir AND FigureFace ABOVE IT REACHES THE
	// ENTITY, and so the byte form and the digest. The mission boundary writes
	// the live entity mask back here so a learned spell reaches the next mint.
	//
	// The zero value is an empty book, which is what every party assembled
	// before this story carried and what a member whose base row did not
	// resolve carries still.
	KnownSpells uint32
	Book        sim.Spellbook

	// SpellbookRestored makes KnownSpells authoritative at mint: equipment's
	// teachSpell effects already ran before the save and must not run again.
	// SpellbookPresent retains the original container's presence separately
	// from empty membership. Sim currently models membership only (1096).
	// Both are additive native metadata; zero preserves old-save behaviour.
	SpellbookRestored bool
	SpellbookPresent  bool

	// Worn is the twelve equipment slots this member starts wearing: his base
	// row's own cells, with the weapon character generation handed him
	// substituted into the row's weapon cell, resolved through GeneratedWornSet
	// (spawn.go) — the SAME wearRow and startingLoadout a placed person's row
	// already resolves through.
	//
	// UNLIKE Body, BodyDir, Mage, Profile, FigureDir AND FigureFace ABOVE,
	// IT REACHES THE ENTITY, ON KnownSpells' OWN TERMS: the mint below
	// folds it into the member's sim.Stock exactly as a carried member's
	// own Equipped already is, so it reaches the byte form and the digest
	// through that one Stock entry rather than through a field of
	// sim.Entity itself.
	//
	// The zero value — every slot empty — is what every party member in
	// this tree wore before this story, and NO CALLER FILLS IT YET (this
	// task's own boundary is pkg/mapload alone): the mint's own Stock
	// composition below therefore still produces exactly the entry it
	// produced before, for every party this tree builds today.
	Worn      [sim.EquipSlots]uint16
	WornItems [sim.EquipSlots]sim.ItemInstance

	// Carried is the container overflow his base row's cells cost: a piece
	// whose own row names no slot among the twelve (AC-2) —
	// GeneratedWornSet's other answer, and Worn's own companion.
	//
	// It reaches the entity on Worn's own terms, through the SAME
	// sim.Stock entry, and is nil for the same reason Worn is zero: no
	// caller fills it yet, and nil costs nothing, on Stock's own rule for
	// an empty slice.
	Carried      []uint16
	CarriedItems []sim.ItemInstance

	// Layers are the codes of the mod clothing layers this member wears, each
	// one unit of an item held in his pack. They reach the derived statistics
	// and the doll's picture and are written to the save's mod leaf; no hashed
	// state holds them, and nil is what every member wears without a mod.
	Layers []uint16

	// Weapon is what he holds, or nil for BARE HANDS — which is a state of the
	// original and not a hole in this one: an unarmed hero swings for
	// `ftol(1.1^Body/20)` either way, zero below Body 32.
	//
	// It is a POINTER because that is what tells "bare" apart from "a weapon
	// whose numbers are all zero", and the two differ: a weapon ASSIGNS the
	// cadence, so a zero-valued one would leave its wielder swinging at 0/0.
	Weapon *data.Weapon

	// WeaponMaterialized is true once Weapon's own slot-1 display fallback
	// (a member whose combat numbers already count him armed, but whose
	// worn array and pack have never actually held the item — 0157's
	// rosterTemplate leaves this member's array empty on purpose) has been
	// resolved into something real: a real item worn in slot 1, or Weapon's
	// own code observed sitting in a container the fallback would otherwise
	// duplicate.
	//
	// IT IS HISTORY, NOT A RE-DERIVED FACT, and that is the whole reason the
	// field exists (round-2 adversarial review, fifth pass, the root cause
	// behind counterexamples A and B). A live scan of the current equipment
	// array and the current pack — pkg/game's weaponFallbackSpent, and the
	// shop's own former pack-presence latch — answers correctly only while
	// the resolved item still sits in the one place the scan looks: sell it,
	// drop it, or carry it into a mission that never revisits that container,
	// and the scan finds nothing and reoffers a second copy of an item the
	// member already owns or already gave away. This bit is set once, the first
	// time either screen observes the fallback resolved, and never cleared, so
	// neither screen's display, drag or purse logic can ever re-open a question
	// this member has already answered.
	//
	// It rides across a mission boundary exactly as Weapon itself does —
	// through clonePartyMember's plain value copy, inside OwnParty and
	// CarryParty — and it rides inside a save exactly as Weapon does too:
	// Snapshot.Party (pkg/game/save.go) is encoding/gob over this whole
	// type, so an exported bool needs no envelope change to persist, and an
	// older save decodes it at its zero value, false.
	//
	// FALSE IS NOT THE CORRECT ANSWER FOR EVERY SUCH MEMBER (round-2
	// adversarial review, twelfth pass, C2, DIV-112): it is correct only for
	// one who never triggered the fallback before the save was written. A
	// member whose fallback HAD already resolved into a real item in an earlier
	// mission, and who then sold or dropped that item before the save was
	// captured, decodes with the latch unraised and reproduces exactly the
	// duplication counterexamples A and B (fifth pass) closed for every save
	// written after this field existed — the old save simply carries no
	// record that the question was ever answered. See DIV-112 for the disclosed
	// gap; this build does not migrate an old save's members to close it.
	WeaponMaterialized bool

	// Carry is what this member brought OUT of a previous mission — his
	// experience, his pack and his worn set — and NIL for a member who has
	// been in none, which is every party this tree built before the
	// continuity hotfix (carry.go).
	//
	// It is a LOADER INPUT on Body's own terms above and the ONLY field
	// here that is not a fact about who he is: every other one is an input
	// the mint folds, while this is what a world already wrote. The two
	// values it reaches are the entity's SkillXP and the sim.Stock that
	// entity is minted with; nothing else in this struct moves for it.
	Carry *Carry

	// Saved is what an ORIGINAL-GAME save file recorded for this member — his
	// cell and his two pools — and NIL for every member who did not come out
	// of one (saved.go).
	//
	// It is Carry's sibling and not one of its fields: a carry is what THIS
	// TREE's simulation wrote at a mission boundary, a Saved is what
	// ANOTHER PROGRAM wrote inside a mission, and the two arrive on
	// different paths and can both be nil. Where it is set it overrides two
	// things the mint would otherwise decide for itself: the cell (which
	// the drop walk chooses) and the two pool pairs (which PartySpawn folds
	// out of the profile).
	Saved *Saved
	// A non-spell effect carried over a town/mission boundary. Native live
	// worlds retain their own already-applied record instead of replaying it.
	PotionEffect *sim.ActiveEffect
}

// Hired reports whether this member was taken on at the tavern, of ANY type
// (owner). It is the one test three appearance sites and the inventory
// switch go through, so a rule about mercenaries cannot come to mean two
// different populations in two packages.
//
// IT IS NOT data.ComposesFigure(Class). That test answers a different question
// -- whether a class is drawn by composing a sheet -- and it answers it for the
// two siege types alone, Catapult and Ballista being the only shipped classes
// at or above 0x1a that can join a party (`MERC-TYPE-001`, `UNIT-PICT-035`).
// The eleven human types are all below the line and pass it. This field is what
// separates a hired man from a player character, and both tavern builders write
// it.
//
// A JOINER IS NOT HIRED. An actor a script hands to the player carries zero
// here and keeps the player character's own laws, which is what `PARTY-MERC-007`
// separates: a mercenary hangs off the Player's groups and does not cross a
// mission boundary, while a handed-over person is judged by the boundary's own
// three tests.
func (p PartyMember) Hired() bool { return p.MercenaryType != 0 }

// Start is what a mission start decided, beside the world it built.
//
// It is RETURNED and never stored on the world. A world is a hashed simulation
// state; where the loader happened to put things is a fact about the load, and a
// field for it would be a field every hand-assembled world would also carry.
type Start struct {
	// ConstructionTable is the exact mission-local NPC table used by this
	// start. It is transient constructor input, never a persisted SAVE basis.
	ConstructionTable *Table
	// Drop is the cell the party started from — the map's own, or the
	// fallback's.
	Drop Cell

	// Authorised is how many cells the map's script authorised. Zero and a
	// picked cell with a zero coordinate both send the start to the fallback,
	// and this is what tells them apart afterwards.
	Authorised int

	// Fallback reports that the drop cell was drawn rather than read.
	Fallback bool

	// Cells records party placement in list order. An unplaced actor keeps
	// Drop as its coordinate but has OffMap set in the returned World.
	Cells []Cell

	// IDs is the entity id each party member was minted with, in party order
	// and parallel to Cells.
	//
	// IT IS RECORDED HERE BECAUSE THIS IS WHERE IT IS KNOWN. The party is
	// appended after the map's placements, so a caller could recompute it as
	// "the last len(party) entities" — and that would be a second copy, one
	// package away, of a rule this function owns, kept true only by nobody
	// changing the append. A caller that wants to say something about a
	// member's entity has the id it was actually given.
	//
	// It is empty for a start with no party, exactly as Cells is.
	IDs []sim.EntityID

	// Crowded counts actors that could not be placed. They stay off the map.
	Crowded int

	// Roster is one PartyMember template per PLACEMENT that resolved through a
	// person row, keyed by the entity id that placement was minted with —
	// FromALMRoster's own answer, carried here because this is the struct a
	// caller already keeps beside the world for the life of a mission.
	//
	// IT DESCRIBES PLACEMENTS AND NEVER PARTY MEMBERS. The party's own ids are
	// IDs above; nothing this map placed is in that list, and nothing in that
	// list is a key here. What it is for is the boundary: an actor a script
	// hands to the player crosses into the next mission, and minting him there
	// needs the statistics, the profile and the spellbook his definition
	// supplied at load and no field of his entity carries.
	//
	// It is nil for a map with no people and for a start built with no table.
	Roster map[sim.EntityID]PartyMember
}

// savedMapUnitID is the authored map id a party member still answers to: the
// record's identifier word when he came out of an original save that named
// one, and 0 for a generated member the map never placed.
//
// It reads the same Saved.MapUnitID that ScriptUnits reads, so a script
// reference resolved at compile time and a map id read back at run time cannot
// come to disagree about one person.
func savedMapUnitID(p PartyMember) uint16 {
	if p.Saved == nil {
		return 0
	}
	return p.Saved.MapUnitID
}

// PartyEntity is the entity id StartMission assigns party member index. Map
// placements are minted first and party members follow in their roster order.
func PartyEntity(m *alm.Map, index int) sim.EntityID {
	placed := 0
	if m != nil {
		placed = len(m.Units)
	}
	return sim.EntityID(placed + index)
}

// PartySpawn is the ONE derivation of what a party member is minted with: the
// recompute every number he carries comes off, and the two pools that
// recompute is read through.
//
// ONE RECOMPUTE, AS OF 0119-chargen: p.Hero.Derive(p.Weapon), p.Hero.Speed()
// and p.Hero.Sight() — three calls, three different argument sets, neither
// of the last two reaching a profile or a weapon — are retired in favour
// of this single call, over the member's own profile and a loadout carrying
// his weapon. The combat block, the step rate, the sight radius and the two
// pools all come off this one value; no member is recomputed twice with
// different arguments.
//
// THE HEALTH PAIR: the recompute's own maximum when the member's OWN PROFILE
// STATES A HEALTH COLUMN AND that maximum is positive; SpawnHP otherwise.
// THE MANA PAIR: ManaMax under the same two conditions on the mana column,
// zero otherwise.
//
// THE COLUMN IS THE GATE, NOT THE POSITIVE MAXIMUM. The first draft of this
// line gated on positivity alone and was wrong, caught by 0119-chargen's own
// T4 against a landed test it must not edit around (hero_test.go's
// TestAStartedPartyMemberKeepsTheHealthRateAndDomainItHad) — and getting
// this right matters MORE as of 0133-person-health than it did then. The
// health arm's own gate (recompute.go step 3) no longer reads a column at
// either end; it runs on the capped Body times the class multiplier alone.
// So a TRAINED character carrying the zero profile — no base row, which is
// every party member this tree minted before 0119-chargen — derives a
// REAL, ORDINARY-LOOKING HealthMax from the recompute whenever his Body is
// nonzero. A positivity test on that number would mint such a member at a
// plausible health that is not his, silently — the same failure this
// line's history already caught once, now behind a far more convincing wrong
// answer. Gating on p.Profile.HealthColumn FIRST is what keeps a member at
// a provisional health when neither a base row nor a constructed fallback
// has said this member's health inputs are ready — a fact about the
// Profile HealthColumn actually carries, not about the ROW he came from:
// `chargenProfile`'s own "no base row resolves" fallback (pkg/game/hero.go)
// sets this bit too, so a member reaching this call through it derives a
// REAL HealthMax exactly like a member with a shipped row does, not the
// provisional value this comment described before that fallback set it.
// The mana arm carries no such asymmetry — ManaColumn already gates
// its WHOLE derivation rather than a product of two other terms, so ManaMax
// is exactly 0 whenever the column is absent — but the same two-condition
// test is written here anyway, so the rule reads identically on both pairs
// and a later change to the mana arm cannot silently widen this gate.
//
// NO FIELD IS ADDED TO sim.Entity AND pkg/sim IS NOT TOUCHED AT ALL: Mana,
// MaxMana, HealthRegenPeriod and ManaRegenPeriod are already there, minted
// at zero since 0109. StartMission's own loop is the first writer that puts
// a positive number in the first two; the serialized byte form does not move
// and its version constant is not spent.
//
// IT READS ONLY ITS ARGUMENT: no map, no table, no world, no clock. The same
// member yields the same three values in every process, so a preview computed
// on the generation screen and a mint performed a second later cannot differ.
func PartySpawn(p PartyMember) (d data.Derived, health, mana int32) {
	if d, health, mana, ok := originalHumanSpawn(p, nil); ok {
		return d, health, mana
	}
	equipped := memberItemEquipment(p)
	loadout := data.Loadout{Weapon: p.Weapon, RotationSpeed: RotationSpeedBase(p.Hired(), p.HiredRotationSpeed, p.Class, nil)}
	ApplyItemEffects(&loadout, equipped, p.Profile.Fighter)
	d, health, mana = partySpawn(p, loadout)
	directHealth, directMana, _ := EquippedPoolEffects(equipped)
	return d, adjustedSpawnPool(health, directHealth, p.Profile.HealthColumn, d.HealthMax),
		adjustedSpawnPool(mana, directMana, p.Profile.ManaColumn, d.ManaMax)
}

// PartySpawnWithTable is the production mission-start derivation. Unlike the
// table-free preview seam above, it folds the member's owned worn set into the
// same loadout Rearm uses after an equip or unequip.
func PartySpawnWithTable(p PartyMember, t *Table) (d data.Derived, health, mana int32) {
	if d, health, mana, ok := originalHumanSpawn(p, t); ok {
		return d, health, mana
	}
	equipped := MemberItemEquipment(p, t)
	carried := MemberCarriedItems(p, t)
	NormalizeShieldLoadout(&equipped, &carried, t)
	loadout := PartyLoadout(p, t)
	ApplyItemEffects(&loadout, equipped, p.Profile.Fighter)
	d, health, mana = partySpawn(p, loadout)
	d.RotationSpeed = HumanTurnRate(spawnSpeedWord(d, PartyLoad(p, t)))
	directHealth, directMana, _ := EquippedPoolEffects(equipped)
	return d, adjustedSpawnPool(health, directHealth, p.Profile.HealthColumn, d.HealthMax),
		adjustedSpawnPool(mana, directMana, p.Profile.ManaColumn, d.ManaMax)
}

func adjustedSpawnPool(current, delta int32, derived bool, maximum int32) int32 {
	if delta == 0 {
		return current
	}
	if !derived || maximum <= 0 {
		maximum = current
	}
	current += delta
	if current > maximum {
		return maximum
	}
	return current
}

func memberItemEquipment(p PartyMember) [sim.EquipSlots]sim.ItemInstance {
	if p.Carry != nil {
		return cloneItemEquipment(p.Carry.EquippedItems)
	}
	return cloneItemEquipment(p.WornItems)
}

// MemberItemEquipment returns a member's canonical worn instances. A fresh
// code-only member is materialised from the installed table; carried legacy
// state remains plain so its unknown provenance is not guessed.
func MemberItemEquipment(p PartyMember, t *Table) [sim.EquipSlots]sim.ItemInstance {
	if p.Carry != nil {
		if items := memberItemEquipment(p); !itemEquipmentEmpty(items) {
			return items
		}
		var out [sim.EquipSlots]sim.ItemInstance
		for i, code := range p.Carry.Equipped {
			out[i] = sim.PlainItem(code)
		}
		return out
	}
	if items := memberItemEquipment(p); !itemEquipmentEmpty(items) {
		return items
	}
	var out [sim.EquipSlots]sim.ItemInstance
	for i, code := range p.Worn {
		out[i] = ItemInstanceFromCode(code, t)
	}
	if p.Weapon != nil && out[0].Code != 0 && p.Weapon.SpellName != "" {
		if spell, ok := SpellIDByToken(t, p.Weapon.SpellName); ok {
			out[0].Effects = append(out[0].Effects, sim.ItemEffect{Kind: 41,
				Operand: uint32(spell) | uint32(uint16(int16(p.Weapon.SpellPower)))<<16})
			out[0].Price = RepriceItemInstance(out[0], t)
		}
	}
	return out
}

func MemberCarriedItems(p PartyMember, t *Table) []sim.ItemInstance {
	if p.Carry != nil {
		if len(p.Carry.ItemInstances) != 0 {
			return cloneItemInstances(p.Carry.ItemInstances)
		}
		out := make([]sim.ItemInstance, len(p.Carry.Items))
		for i, code := range p.Carry.Items {
			out[i] = sim.PlainItem(code)
		}
		return out
	}
	if len(p.CarriedItems) != 0 {
		return cloneItemInstances(p.CarriedItems)
	}
	out := make([]sim.ItemInstance, len(p.Carried))
	for i, code := range p.Carried {
		out[i] = ItemInstanceFromCode(code, t)
	}
	return out
}

// spawnSpeedWord is a native member's derived speed word at spawn: his
// unencumbered speed less the overload penalty, then his modifier
// (rules.NativeHumanSpeed). The turn rate takes its low byte.
func spawnSpeedWord(d data.Derived, load int32) int32 {
	word, _ := rules.NativeHumanSpeed(d.Speed, d.SpeedModifier, load, d.Capacity)
	return word
}

func partySpawn(p PartyMember, loadout data.Loadout) (d data.Derived, health, mana int32) {
	d = p.Hero.Recompute(p.Profile, loadout)
	d.RotationSpeed = HumanTurnRate(d.Speed)

	health = SpawnHP
	if p.Profile.HealthColumn && d.HealthMax > 0 {
		health = d.HealthMax
	}
	if p.Profile.ManaColumn && d.ManaMax > 0 {
		mana = d.ManaMax
	}
	return d, health, mana
}

// pools is the health pair, the mana pair and the two regeneration periods one
// party member is minted with.
//
// It exists because a resume from an original save has to override all six
// together and PartySpawn answers two of them — see restoredPools.
type pools struct {
	hp, maxHP                int32
	mana, maxMana            int32
	healthPeriod, manaPeriod int32
}

// restoredPools is the six pool values one party member is minted with: the
// fold's, or the FILE's for a member restored from an original save.
//
// IT IS ALL SIX OR NONE, and that is the whole reason it is one function. A
// restored character's health could be taken from his file while his health
// MAXIMUM came off the fold, and the result would be a character whose health
// bar is a ratio of two numbers written by two different programmes — a full bar
// reading as a third full, or a third-full bar reading as overflowing. The file
// states both members of each pair and the pair is what a reader sees, so the
// pair moves together.
//
// THE TWO PERIODS MOVE WITH THEM although they do not have to. Every Unit record
// measured carries the base constructor's own 100 and 50, which is exactly what
// data.UnitDefaults() answers, so this changes no world today. It is written
// this way because the file STATES them: a later save that carried a different
// pair would be honoured rather than silently normalised.
func restoredPools(p PartyMember, health, mana int32, regen data.UnitDef) pools {
	if s := p.Saved; s != nil {
		return pools{
			hp: s.HP, maxHP: s.MaxHP,
			mana: s.Mana, maxMana: s.MaxMana,
			healthPeriod: s.HealthRegenPeriod, manaPeriod: s.ManaRegenPeriod,
		}
	}
	return pools{
		hp: health, maxHP: health,
		mana: mana, maxMana: mana,
		healthPeriod: regen.HealthRegenPeriod, manaPeriod: regen.ManaRegenPeriod,
	}
}

// StartMission builds a mission's world: the map's own placements, then the
// party, standing where the map's script says the party starts.
//
// It is FromALMWith plus two decisions, and it is defined in terms of it rather
// than beside it, so there is one world-building implementation and "a start
// builds the same world a plain load does, with a party added" cannot decay into
// two paths that agree today.
//
// THE DROP CELL COMES FROM THE MAP AND THE DRAWS COME FROM StartSeed. One cell is
// drawn uniformly from what the map's script authorises; when the script
// authorises none, or when the drawn cell has a zero coordinate on either axis,
// each coordinate is instead drawn independently from the fallback range. The
// order of draws is part of the contract — pick, then column, then row — because
// reversing it gives a different world from the same seed.
//
// A CONSEQUENCE WORTH STATING: because the seed is a constant, the fallback is
// the SAME cell on every map that takes it. That is the price of a load whose
// output follows from its input, and it is a divergence — the original seeds from
// a clock, so its fallback differs every run. Nothing shipped takes the fallback:
// all 38 maps authorise a cell.
//
// The per-player override the original consults before the map's array is not
// reachable here and is not modelled: it lives on a player object this tree does
// not have.
//
// THE PARTY IS APPENDED, not inserted. The original puts the party at the head of
// its own tick order; this tree's entity id is the placement's slice index, so
// appending is what keeps every existing id and every existing digest for a map
// whose arms did not move. The divergence is recorded rather than absorbed.
//
// An empty party builds exactly the world FromALMWith builds, and still draws:
// where the party would have stood is decided whether or not anyone stands
// there, so a start's report says the same thing for a party of none as for a
// party of one.
func StartMission(m *alm.Map, t *Table, diff Difficulty, party []PartyMember) (*sim.World, Start, error) {
	return startMission(m, t, diff, party, 0, nil)
}

// StartCampaignMission is StartMission with the campaign mission number that
// selects the four composed NPC rows. StartMission remains the tier-zero entry
// point for standalone maps and synthetic callers that have no campaign.
func StartCampaignMission(m *alm.Map, t *Table, diff Difficulty, party []PartyMember,
	mission int) (*sim.World, Start, error) {
	return startMission(m, t, diff, party, mission, nil)
}

// startMission places the party with draws, or with a sequence at StartSeed
// when draws is nil. Draws on the original's generator continue into the
// World, which then starts from their state instead of Seed.
func startMission(m *alm.Map, t *Table, diff Difficulty, party []PartyMember,
	mission int, draws *sim.Draws) (*sim.World, Start, error) {
	// The loader owns the roster it is about to mint. In particular, no Carry
	// pointer or item slice held by a town/save caller remains writable through
	// this mission's member values.
	party = OwnParty(party)
	// The four composed scenario heroes do not name a fixed Humans row. Their
	// registry tokens select an archetype relative to the player's starting
	// hero, so resolve them on a mission-local table copy before any placement
	// consumer (world, roster, loadout or sheet) walks the map.
	t = tableForPartyNPCs(t, party, mission)
	party = NormalizePartyShieldLoadouts(party, t)
	// The map's own world is built FIRST, so an undefined difficulty is refused
	// by the one function that owns that refusal rather than by a second copy of
	// its test here.
	base, roster, err := FromALMRoster(m, t, diff)
	if err != nil {
		return nil, Start{}, err
	}

	if draws == nil {
		draws = sim.NewDraws(StartSeed)
	}
	cells := DropCells(m)
	// The map's person roster travels with the start (0159 D-9). It keys on
	// placement ids, which the party append below never reuses: the party takes
	// ids from len(ents) upward, above every placement's.
	st := Start{Authorised: len(cells), Roster: roster, ConstructionTable: t}

	// The pick is taken FIRST and unconditionally when there is a list, so the
	// sequence a fallback then reads does not depend on which branch was taken.
	// A one-element list is a degenerate draw that consumes nothing, which is
	// what makes a shipped map's start a function of the map alone.
	if len(cells) > 0 {
		st.Drop = cells[draws.Upto(int32(len(cells)-1))]
	}
	if st.Drop.X == 0 || st.Drop.Y == 0 {
		st.Fallback = true
		st.Drop = Cell{
			X: fallbackLow + draws.Upto(fallbackSpan),
			Y: fallbackLow + draws.Upto(fallbackSpan),
		}
	}

	planes := Planes(m, t)
	st.Cells = make([]Cell, len(party))
	preserved := make([]bool, len(party))
	for i := range party {
		// Mission resumes keep the saved cell. Between-mission imports clear
		// that half before reaching this loader and receive a new placement.
		st.Cells[i] = st.Drop
		if saved := party[i].Saved; saved != nil && (saved.Cell.X != 0 || saved.Cell.Y != 0) {
			st.Cells[i], preserved[i] = saved.Cell, true
		}
	}

	if len(party) == 0 {
		return base, st, nil
	}
	// A PARTY MEMBER'S EIGHT COMBAT NUMBERS ARE THE FOLD'S, and they are folded
	// HERE rather than handed over by the caller, so a member's numbers cannot
	// be set beside its hero and then disagree with him.
	//
	// What is still the unresolved placement's, UNCONDITIONALLY, is the domain.
	// The health pair is too, for a member carrying no profile — the zero
	// value, which is what every member here carried before 0119-chargen —
	// and for such a member it is still taken from THE SAME VALUE an unresolved
	// placement takes rather than from a copy of the number in it, for the
	// reason it always was: a second literal is exactly how the two populations
	// that resolve to nothing would come to differ, and one of them would be
	// the party. A member who DOES carry a profile takes the recompute's own
	// maximum instead, below.
	//
	// THE RATE IS NO LONGER AMONG THEM. It was the class table's 10 — the
	// constructor default for a unit whose row says nothing — while the
	// hero's own Reaction, which the original derives it from, was consulted
	// nowhere. It is now derived like every other number he has, and it is
	// derived UNIFORMLY: a member with no statistics gets what the law gives at
	// Reaction 0, which is 0, rather than falling back to the table. A fallback
	// here would be the one place a party number came from somewhere else, and
	// the rate law floors the resulting per-step rate at 1, so such a member
	// still moves.
	ents := base.Entities()
	st.IDs = make([]sim.EntityID, len(party))
	worn := make([]sim.Stock, 0, len(party))
	// THE TWO PERIODS EVERY MEMBER IS MINTED WITH: the base constructor's own
	// 100 and 50, taken from data.UnitDefaults() rather than written out here a
	// second time — the same source blockFor's unresolved arm reads, so a
	// member and an unresolved placement cannot come to disagree about what
	// they default to.
	//
	// THE TWO POOLS ARE NO LONGER MINTED BLIND, AS OF 0119-chargen: both come
	// off the ONE recompute below, over the member's own profile, rather than
	// off a constant written here. A member with no base row — the zero
	// profile, which is what every party member in this tree carried until this
	// story — still gets SpawnHP on health and nothing on mana, which is why
	// the no-flag mission path is unchanged; the divergence 0078 disclosed is
	// closed for a member who carries a profile, and only for one.
	regen := data.UnitDefaults()
	var hero PartyMember
	for _, p := range party {
		if p.StartingHero {
			hero = p
			break
		}
	}
	for i, p := range party {
		// The first two tavern types are siege CREATURES, not generated
		// humans (MERC-LEVEL-005). Resolve them through the placement block so
		// Catapult and Ballista keep their Units-row combat, domain, footprint
		// and equipment instead of being flattened into a zero-profile Hero.
		if p.MercenaryType == 1 || p.MercenaryType == 2 {
			def, b, err := siegeDefinition(p, t)
			if err != nil {
				return nil, Start{}, err
			}
			carriedNativeHistory(p, &def.NativeBasis, &def.NativeClass)
			id := sim.EntityID(len(ents))
			st.IDs[i] = id
			ents = append(ents, sim.NewActor(def, sim.ActorPlacement{ID: id, X: st.Cells[i].X, Y: st.Cells[i].Y,
				Owner: sim.SelfSlot, MapUnitID: savedMapUnitID(p)}))
			if !itemEquipmentEmpty(b.worn) || len(b.carried) > 0 {
				worn = append(worn, sim.Stock{ID: id, ItemInstances: cloneItemInstances(b.carried), EquippedItems: cloneItemEquipment(b.worn)})
			}
			continue
		}
		// THE ONE DERIVATION OF WHAT THIS MEMBER IS MINTED WITH — see
		// PartySpawn below, which is where the recompute and the two pool
		// gates now live so that a caller wanting to know what a member WOULD
		// be minted with (the generation screen's consequence block) reads the
		// same expression this loop mints from, rather than a second copy of
		// it that could drift.
		equippedItems := MemberItemEquipment(p, t)
		for slot := range equippedItems {
			if _, _, carriesSpell := equippedItems[slot].CastSpell(); carriesSpell {
				equippedItems[slot] = SourceEquippedItem(equippedItems[slot], t)
			}
		}
		carriedItems := MemberCarriedItems(p, t)
		NormalizeShieldLoadout(&equippedItems, &carriedItems, t)
		nativeBasis := sim.NativeActorBasis{}
		if p.Carry == nil || p.Carry.NativeHistory == nil {
			nativeBasis = nativeInitialModifier(nativeInitialBase(&p.Hero.Skill), &equippedItems, t, p.Hero.Skill[0], true, p.Profile.Fighter)
		}
		d, hp, mana := PartySpawnWithTable(p, t)
		// WHAT A RESTORED CHARACTER ARRIVES WITH IS THE FILE'S, NOT THE FOLD'S,
		// and all six values move together — see restoredPools below for why it
		// is all six or none.
		pool := restoredPools(p, hp, mana, regen)
		_, _, taught := EquippedPoolEffects(equippedItems)
		if p.SpellbookRestored || p.Book.State != sim.BookLegacy {
			taught = 0
		}
		// THE SPELL HIS WEAPON CARRIES, off the SAME d the fold's other seven
		// fields below come off: d.Combat.SpellName is still the token FoldWeapon
		// carried onto the recompute, and SpellIDByToken is FR-1b's lookup, over t
		// — the same table every other resolution in this function reads. A
		// token this table cannot resolve — no table, no Spells collection, no
		// matching row — answers 0, which is WeaponSpell's own "none"
		// (world.go), so a member whose weapon names an unloaded spell arms
		// exactly as one holding no spell at all (FR-2b's own rule, one tier
		// down).
		spellID, _ := SpellIDByToken(t, d.Combat.SpellName)
		weaponSource := sim.WeaponSpellNone
		if itemSpell, itemPower, ok := equippedItems[0].CastSpell(); ok {
			spellID, d.Combat.SpellPower = itemSpell, itemPower
			weaponSource = sim.WeaponSpellItem
		} else if spellID != 0 || d.Combat.SpellPower != 0 {
			weaponSource = sim.WeaponSpellInnate
		}

		// THE SIX SLOT EXPERIENCES AND MIND HE IS MINTED WITH: reward is
		// Hero.Reward(), the thin accessor over the SAME recompute c above already
		// ran, seeded with the sum his current levels already account for rather
		// than a reset to zero the first blow would then have to climb back out
		// of.
		reward := p.Hero.Reward()
		// A CARRIED MEMBER'S EXPERIENCE IS THE INTEGER HE ENDED THE LAST
		// MISSION WITH, not the one his levels account for (the continuity
		// hotfix; carry.go). The two are the same value in two spellings,
		// joined by S(n) and its FLOORING inverse — CarryParty has already
		// written the levels from these very integers, so the seed above and
		// this override agree to within the residue the floor drops, and it is
		// exactly that residue this line exists to keep. Without it a member
		// would give back every point of progress toward his next level at
		// every mission boundary.
		//
		// reward.Mind is NOT overridden: it is his capped statistic, folded
		// like every other number he has, and nothing in a mission moves it.
		if p.Carry != nil {
			reward.SkillXP = p.Carry.SkillXP
		}
		// HIS SERVER TYPE ID, AND WHETHER HE TRAINS (owner).
		//
		// A hired man carries his own Humans row's type id, which is what
		// p.Class holds for him -- buildMercenarySquad writes the row's TypeID
		// there, the same field and the same value the map path's composed-NPC
		// arm writes. Every one of the 52 shipped NPC%02d_%d rows states one in
		// 3..24, so none of them is inside the band; a player character keeps
		// the class and sex type of the player-character constructor.
		//
		// TRAINING FOLLOWS THE BAND and is not a second decision: fromalm.go's
		// person arm already writes gainsXP: sim.InPersistBand(typeID) under
		// the comment "mission-only low-type people do not earn", and a hired
		// man is exactly one. Before this he was minted in the hero band with
		// GainsXP true -- measured, NPC03_1 reached mission 10 carrying type id
		// 33, byte for byte the player character Danath's own.
		//
		// IT DOES NOT BY ITSELF STOP HIM CROSSING THE BOUNDARY, and nothing
		// here claims it does: CarryParty carries the whole party by index and
		// consults no band, so a member who walked in crosses whatever his type
		// id is. `PARTY-MERC-007` says a mercenary does not cross and that what
		// crosses instead is the count per type; this tree implements neither
		// the cull nor `MERC-DEATH-006`'s pool merge. See DIV-434.
		typeID := sim.HeroTypeID(p.Mage, data.FigureDir(p.FigureDir).Female())
		if p.Hired() {
			typeID = p.Class
		}
		dying, err := partyDyingTime(p, hero, typeID, t)
		if err != nil {
			return nil, Start{}, err
		}
		// A generated member has no row: his numbers are the fold's, his pools
		// the restored or derived ones, his reach and sight derived like his
		// rate. He stands on the player's roster slot and in no map group. A
		// member restored from an original save keeps the record's map unit id.
		// The treasure values stay zero; the death-gold roll is gated above
		// the person band. His dying time is his bound Humans row's.
		def := sim.ActorDefinition{
			Class: p.Class, TypeID: typeID, Humanoid: true, Domain: sim.DomainGround,
			HP: pool.hp, MaxHP: pool.maxHP, Mana: pool.mana, MaxMana: pool.maxMana,
			HealthRegenPeriod: pool.healthPeriod, ManaRegenPeriod: pool.manaPeriod,
			HealthRegeneration: d.HealthRegeneration, ManaRegeneration: d.ManaRegeneration,
			Speed: d.Speed, SpeedModifier: d.SpeedModifier, RotationSpeed: d.RotationSpeed, Capacity: d.Capacity,
			ScanRange: uint8(d.Sight), Reach: reachOf(d.Combat.Reach), TokenSize: 1, DyingTime: dying,
			ToHit: d.Combat.ToHit, Defence: d.Combat.Defence, Absorption: d.Combat.Absorption,
			DamageBase: d.Combat.DamageBase, DamageSpread: d.Combat.DamageSpread,
			SecondBase: d.Combat.SecondBase, SecondSpread: d.Combat.SecondSpread,
			SecondaryDamage: simSecondaryDamage(d.SecondaryDamage), AlwaysHits: d.Combat.AlwaysHits,
			AttackCharge: d.Combat.AttackChargeTime, AttackRelax: d.Combat.AttackRelaxTime,
			Protection: d.Protection, Resistance: data.DamageKindResistance(d.Resistance),
			WeaponSpell: spellID, WeaponSpellLevel: d.Combat.SpellPower, WeaponSpellSource: weaponSource,
			// MAGIC-BOOK-002: the book is the member's own, not gated on a stat.
			KnownSpells: p.KnownSpells | taught, Book: p.Book,
			Reaction: d.Reaction, Mind: reward.Mind, Spirit: d.Spirit,
			XPSlot: uint8(d.Combat.SkillSlot), GainsXP: sim.InPersistBand(typeID),
			Skill: d.Skill, SkillXP: reward.SkillXP, SuppressCorpseLoot: p.SuppressCorpseLoot,
			NativeBasis:    nativeBasis.WithBody(uint16(d.Body)),
			NativeClass:    sim.NativeClass{Present: true, Fighter: p.Profile.Fighter},
			NativeTraining: sim.NativeTraining{Present: true, Levels: p.Hero.Skill},
		}
		carriedNativeHistory(p, &def.NativeBasis, &def.NativeClass)
		st.IDs[i] = sim.EntityID(len(ents))
		ents = append(ents, sim.NewActor(def, sim.ActorPlacement{ID: st.IDs[i], X: st.Cells[i].X, Y: st.Cells[i].Y,
			Owner: sim.SelfSlot, MapUnitID: savedMapUnitID(p)}))
		// A CARRIED MEMBER ARRIVES HOLDING WHAT HE LEFT WITH, and the class
		// row's starting weapon is NOT minted for him a second time (the
		// continuity hotfix). That is the whole reason Carry is reached by
		// pointer: a member who ended a mission wearing nothing arrives
		// wearing nothing, and re-arming him here would undo a player's own
		// act every time a mission ended.
		//
		// A member holding neither an item nor a worn code produces NO Stock
		// entry, on Stock()'s own rule one package over — an entry naming two
		// empty records is not a claim the constructor could act on
		// differently from its absence. The canonical readers independently
		// materialise either legacy half when only the other half already has
		// instances; one canonical half must not erase the other code projection.
		if len(carriedItems) > 0 || !itemEquipmentEmpty(equippedItems) || p.Carry != nil && p.Carry.LiveLoad != nil {
			stock := sim.Stock{ID: st.IDs[i],
				ItemInstances: cloneItemInstances(carriedItems),
				EquippedItems: cloneItemEquipment(equippedItems)}
			if p.Carry != nil {
				stock.LoadState, stock.OrderedStacks = p.Carry.LiveLoad, p.Carry.OrderedStacks
			}
			worn = append(worn, stock)
		}
	}
	// The rebuild takes the same seed, mode, THREE PLANES and RELATION the load
	// above took, so the world a party is added to is the world that was built
	// and not a second derivation of it — the bundle is the one already in hand
	// rather than a fresh call, which is what makes that true of all three planes
	// rather than of the one this function happened to keep a name for.
	//
	// THE RELATION HAS TO BE NAMED HERE OR IT IS LOST, and losing it is silent in
	// the worst way: the world still builds, still hashes, still ticks, and its
	// only symptom is that nothing in it ever starts a fight — which is exactly
	// what a world looked like before the relation existed. It is taken off base
	// rather than rebuilt from the map, so the store runs once, in the one place
	// that owns it, and a start cannot come to disagree with a plain load about
	// what a roster says.
	//
	// THE SAME IS TRUE OF THE CONTAINERS, and this line is the THIRD time the
	// paragraph above has had to be written for a state added after the rebuild
	// was (0112, found at the orchestrator seat before landing). 0103 put the
	// map's ground sacks in a world and both rebuilds here dropped them for
	// eight stories, because nothing DREW a sack and "no sack anywhere" is
	// indistinguishable from "this map authors no loot". 0112 put the map's own
	// STOCK in a world — 43 records across the shipped campaign, identical on
	// both roots — and this rebuild dropped it the same way, for the same
	// reason: nothing yet shows a stocked actor, so "nobody carries anything"
	// reads exactly like "this map stocks nobody". Mission 10 authors three
	// items for one person at (36,51) and opened with the person empty.
	//
	// It is read back off base with Stock(), not rebuilt from the map, on the
	// relation's own ground one paragraph up: one place owns what a stock record
	// means, and a start cannot come to disagree with a plain load about it.
	//
	// THE PURSES ARE NOT NAMED HERE, and that is an argument rather than an
	// oversight: their only writer is sim.World.TakeSack, nothing calls it
	// between the load above and this line, so base's purses are all zero by
	// construction and naming them would carry nothing. The regression test for
	// this line compares base's purses against the rebuild's anyway, so the day
	// a story authors a purse at load, that test — and not a shipped mission —
	// is what says so.
	//
	// THE PARTY'S LOADOUT IS APPENDED TO base's OWN HOLDINGS rather than
	// written into a fourth argument, because a Stock states both halves of
	// what an actor begins with: base.Stock() carries the map's containers
	// AND the class rows' loadouts together, and the party's entries name
	// ids no record of base's can name. That is what keeps this rebuild from
	// being the place a fourth state is dropped — see the paragraph above
	// on the sack list and the containers, which is that lesson told twice.
	//
	// THE SPELL TABLE IS READ BACK OFF base TOO, on the relation's own ground
	// above: base was built through FromALMWith, which already resolved t's
	// Spells collection once, and reading base.Spells() here rather than
	// calling spellsFor(t) a second time is what keeps this rebuild from being
	// a second place that could answer a different table for the same t.
	st.Crowded = sim.PlaceMissionParty(base.Bounds(), planes.Block, ents, len(ents)-len(party), preserved,
		st.Drop.X, st.Drop.Y, draws)
	for i, id := range st.IDs {
		st.Cells[i] = Cell{X: ents[id].X, Y: ents[id].Y}
	}
	out, err := sim.NewStructuredWorld(Seed, base.Bounds(), sim.ModeCanonical, planes, ents, nil,
		base.Relations(), base.Sacks(), append(base.Stock(), worn...), base.Spells(), base.Ghost(),
		base.Structures())
	if err != nil {
		return nil, Start{}, err
	}
	out.SetRules(t.rules())
	out.CopyDiaryUnits(base)
	if err := out.DeclareCellTails(base.CellTails()); err != nil {
		return nil, Start{}, err
	}
	// THE WEIGHT TABLE IS REBUILT HERE TOO, on the containers' own reason
	// below: a rebuild carries only what it names, and the party's own worn and
	// carried codes are codes no world base ever held. It is not read back off
	// base for that reason -- base's table names the map's codes alone.
	declareItemWeights(out, t)
	if err := initializeCurrentPlayers(out, m, t); err != nil {
		return nil, Start{}, err
	}
	// Money is already participant-owned in sim. Quest documents follow the
	// same campaign surface: wherever a carried party member arrived holding
	// them, collect the stack onto the explicitly marked starting hero before
	// the mission is exposed. This is independent of selection and leaves every
	// other item with its own carrier. ITEM-DOC-053 names the code; the move
	// is DIV-2798.
	primary := QuestDocumentHolder(party)
	if primary >= 0 && primary < len(st.IDs) {
		for i, id := range st.IDs {
			if i == primary {
				continue
			}
			stacks, ok := out.CarriedStacks(id)
			if !ok {
				continue
			}
			for _, stack := range stacks {
				if stack.Code == uint16(data.QuestDocumentCode) {
					if err := out.MoveCarried(id, st.IDs[primary], stack.Code, stack.Count); err != nil {
						return nil, Start{}, err
					}
					break
				}
			}
		}
	}
	initializePotions(out, party, st, t)
	if err := initializeOriginalHumanMovement(out, party, st); err != nil {
		return nil, Start{}, err
	}
	DeclareSourceConstructors(out, t)
	if draws.Original() {
		out.SetRandom(random.Original, draws.State())
	}
	return out, st, nil
}

// QuestDocumentHolder is the index of the marked starting hero that mission
// start collects every member's quest document onto, or -1.
func QuestDocumentHolder(party []PartyMember) int {
	for i, p := range party {
		if p.PlayerCharacter && p.StartingHero && p.MercenaryType == 0 {
			return i
		}
	}
	return -1
}

func tableForPartyNPCs(t *Table, party []PartyMember, mission int) *Table {
	if t == nil || t.NPC == nil || t.Humans == nil || len(party) == 0 {
		return t
	}
	primary := party[0]
	for _, p := range party {
		if p.StartingHero {
			primary = p
			break
		}
	}
	playerMage := primary.Mage
	playerFemale := data.FigureDir(primary.FigureDir).Female()
	resolved := make(map[int32]int)
	for id := int32(21); id <= 24; id++ { // REG-SCN-098: the npc21..24 branch
		serverID, ok := t.NPC.CampaignServerID(id, mission, playerMage, playerFemale)
		if ok {
			index := data.FindHumanByServerID(t.Humans, serverID)
			// Standalone maps have no campaign tier. Preserve their archetype
			// composition even for a synthetic or modded table whose base rows do
			// not carry the shipped server ids; campaign starts require the exact
			// tiered id and never take this fallback.
			if index == data.NotFound && mission == 0 {
				mage, female, composed := t.NPC.ComposedArchetype(id, playerMage, playerFemale)
				if composed {
					_, index, _ = data.ChargenBase(t.Humans, mage, female)
				}
			}
			if index != data.NotFound {
				resolved[id] = index
			}
		}
	}
	if len(resolved) == 0 {
		return t
	}
	out := *t
	out.composedNPC = resolved
	return &out
}

// StartMissionScripted is StartMission over a world that RUNS the mission's
// compiled script.
//
// IT STANDS BESIDE StartMission AND DOES NOT REPLACE IT, because both shapes are
// real. A map opened to be looked at wants the plain world — ten of the shipped
// corpus's loose maps author no winning instant at all — and a MISSION wants the
// scripted one: with no script attached the world's clock advances and nothing
// else does, so no trigger is ever evaluated, no latch is ever set, and no
// outcome is ever reached. Attaching inside StartMission would have changed what
// every existing caller gets, which is the one thing StartMission's own no-party
// test exists to catch.
//
// THE SCRIPT IS THE CALLER'S, positional and last, exactly as it is on
// sim.NewScriptedWorld. The compile is a separate step carrying its own report —
// a raise list, unresolved references, and an error for a script that will not
// decode — and what each of those means is the caller's answer: pkg/game keeps a
// script error rather than failing the mission on it. Compiling here would take
// that decision away from the one place that holds it, and would compile a
// second time for every caller that already has.
//
// THE REBUILD TAKES THE SAME SEED, MODE, BOUNDS, PLANES AND RELATION the load
// above took, so a nil script yields the world StartMission returns, hash for
// hash. A script moves it only by presetting the registers its constant checks
// own, which is the script's own state arriving and not the world being built
// differently.
//
// The relation is the one of those five that no other rebuild would have missed
// quietly. This is the entry point EVERY MISSION comes through, so a rebuild that
// dropped it would leave the whole shipped campaign with the all-zero relation —
// monsters standing still on every map — while every test that builds a world
// through the plain load still passed.
func StartMissionScripted(m *alm.Map, t *Table, diff Difficulty, party []PartyMember,
	s *sim.Script) (*sim.World, Start, error) {
	return startMissionScripted(m, t, diff, party, 0, s, nil)
}

// StartCampaignMissionScripted is StartMissionScripted with campaign-tier NPC
// composition. It is the campaign door; the plain form remains available to
// map tools which deliberately have no mission number.
func StartCampaignMissionScripted(m *alm.Map, t *Table, diff Difficulty, party []PartyMember,
	mission int, s *sim.Script) (*sim.World, Start, error) {
	return startMissionScripted(m, t, diff, party, mission, s, nil)
}

// StartCampaignMissionScriptedWith is StartCampaignMissionScripted placing
// the party with draws; nil is the sequence at StartSeed.
func StartCampaignMissionScriptedWith(m *alm.Map, t *Table, diff Difficulty, party []PartyMember,
	mission int, s *sim.Script, draws *sim.Draws) (*sim.World, Start, error) {
	return startMissionScripted(m, t, diff, party, mission, s, draws)
}

func startMissionScripted(m *alm.Map, t *Table, diff Difficulty, party []PartyMember,
	mission int, s *sim.Script, draws *sim.Draws) (*sim.World, Start, error) {
	base, st, err := startMission(m, t, diff, party, mission, draws)
	if err != nil {
		return nil, Start{}, err
	}
	// THE CONTAINERS ARE NAMED HERE FOR StartMission's OWN REASON, written out
	// in full at that call site: a rebuild carries only what it names, and
	// 0112's containers are the third state in a row added after both rebuilds
	// already existed. This entry point is the one EVERY MISSION comes through,
	// so a rebuild that dropped them would leave the whole shipped campaign with
	// 43 stocked people carrying nothing while every test built through the
	// plain load still passed — which is exactly how the sack list shipped
	// broken for eight stories.
	//
	// THE SPELL TABLE IS READ BACK OFF base HERE TOO, on the containers' own
	// reason one paragraph up: StartMission already resolved it once, and this
	// entry point is the one every mission comes through, so a rebuild that
	// called spellsFor(t) again instead would be a second place the same table
	// could come to answer differently.
	out, err := sim.NewStructuredWorld(Seed, base.Bounds(), sim.ModeCanonical,
		Planes(m, t), base.Entities(), s, base.Relations(), base.Sacks(), base.Stock(), base.Spells(),
		base.Ghost(), base.Structures())
	if err != nil {
		return nil, Start{}, err
	}
	out.SetRules(t.rules())
	out.CopyDiaryUnits(base)
	if err := out.DeclareCellTails(base.CellTails()); err != nil {
		return nil, Start{}, err
	}
	// AND THE WEIGHT TABLE. This entry point is the one EVERY MISSION comes
	// through and it is the one that carries the compiled script, so it is also
	// the only place the script's own item literals reach the pass at all.
	declareItemWeights(out, t)
	out.CopyPotionEffects(base)
	if players, present := base.CurrentPlayers(); present {
		rows, participants := base.PlayerParticipants()
		if err := out.RestoreCurrentPlayers(players, rows, participants); err != nil {
			return nil, Start{}, err
		}
	}
	if err := initializeOriginalHumanMovement(out, party, st); err != nil {
		return nil, Start{}, err
	}
	DeclareSourceConstructors(out, t)
	if base.RandomMode() == random.Original {
		out.SetRandom(random.Original, base.RandomState())
	}
	return out, st, nil
}

// worldExtent is the map's own cell extent, and zero for a nil map — the same
// answer FromALMWith gives such a map, so the two cannot disagree about where the
// grid ends.
func worldExtent(m *alm.Map) (w, h int32) {
	if m == nil {
		return 0, 0
	}
	return int32(m.Width), int32(m.Height)
}

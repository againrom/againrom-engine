package sim

// This file is the occupancy rule's negative half: that rule adds NO state to the
// canonical world, and it moves no byte of the world's form and no bit of its
// digest. The form has since moved for OTHER reasons — a routing mode, a grid and
// a stall count are canonical state and are carried by it — so what is measured
// below is that the form is the contract's current one and that nothing occupancy
// derives crosses a decode; it is no longer a claim that the bytes never changed.
//
// Its two pins are shaped the way they are because the obvious criterion does not
// hold. The byte form and the digest are pinned already, and neither pin can see
// either shape this file exists to refuse. A field added to World or to Entity is
// invisible to both, because a field the encoder does not write changes no byte it
// does write — and the record width is a constant, so nothing about the form even
// wobbles. Occupancy kept in a package-level index is invisible twice over: it is
// in neither struct either. A criterion assembled out of the byte-form pins alone
// therefore passes a design that keeps exactly the state C-1 forbids, which is why
// there are two pins here and not one.
//
// So the field sets are pinned as a literal table of name, type and order and
// compared for EXACT EQUALITY — never searched for the absence of a name, which
// would only ever refuse the fields somebody thought to name. That is also what
// closes the chain the entity comparisons elsewhere leave open: they compare the
// fields an Entity has today, one at a time, and a field added tomorrow is outside
// every one of them. And a world is marshalled at every tick of a contended run,
// decoded into a second world, and stepped on from there to the end of the run:
// occupancy held anywhere the byte form cannot reach does not cross that boundary,
// so it shows up as a divergence instead of as a passing pin.
//
// The second pin's limit, stated because it is not obvious: it sees state
// that is HELD across the crossing.
//
// conBounds, conDistinct and conRefused come from contention_test.go, dist from
// step_test.go, pinWorld and pinBytes from binary_test.go and pinDigest from
// hash_test.go. Those last three are AC-12's pin: this file reads them and edits
// nothing, because a pin edited by the task that judges it witnesses nothing.

import (
	"bytes"
	"reflect"
	"testing"
)

// ---------------------------------------------------------------- the field set

// nstField is one struct field as a pin reads it: the name it is declared under
// and the type it is declared as, spelled the way reflect spells it.
type nstField struct {
	name string
	typ  string
}

// nstPinned is the pin — every type the canonical world is built out of, and the
// whole of what each one holds, in declaration order and written out by hand.
//
// World and Entity are the two the rule would have reached for. Bounds and rng are
// here because World holds them BY VALUE: a field added inside either is new state
// in the canonical world just as much, is just as invisible to the byte form, and
// would slip past a pin that stopped at the outer struct. What no table can do is
// notice a new type appearing beside these four; that one belongs to the source
// scan in internal/archtest, not here.
var nstPinned = []struct {
	what   string
	typ    reflect.Type
	fields []nstField
}{
	{"World", reflect.TypeOf(World{}), []nstField{
		{"tick", "uint64"},
		{"damageObservation", "*sim.damageObservation"},
		{"turnSteps", "map[sim.EntityID]struct {}"},
		{"turnStepScope", "bool"},
		{"hasSessionClock", "bool"},
		{"fullTick", "uint32"},
		{"rng", "sim.rng"},
		{"rules", "rules.Rules"},
		{"bounds", "sim.Bounds"},
		{"mode", "sim.Mode"},
		{"grid", "[]uint8"},
		{"cost", "[]uint8"},
		{"height", "[]uint8"},
		{"relations", "sim.Relations"},
		{"entities", "[]sim.Entity"},
		{"actorTraversal", "[]sim.EntityID"},
		{"entityIDFloor", "uint64"},
		{"routes", "[][]sim.cell"},
		{"groups", "[]sim.groupAI"},
		{"savedGroups", "*sim.savedGroupState"},
		{"sacks", "[]sim.Sack"},
		{"spells", "[]sim.SpellRule"},
		// 1001: the definition table's own `Ghost` row, what a Control Spirit
		// cast raises. IT IS THE ONE FIELD THE BYTE FORM DOES NOT CARRY and
		// the exception is deliberate: it is install-derived INPUT, resolved
		// by pkg/mapload from the same table the spell rows come from, and
		// UnmarshalBinary carries the receiver's own value across the decode
		// (binary.go). It holds nothing a tick derives, so the occupancy rule
		// this file exists for is untouched — what it costs is that two
		// worlds equal in bytes may differ in which template they raise from,
		// and that is stated at the decode rather than left to be discovered.
		{"ghost", "sim.GhostTemplate"},
		// Pure arithmetic code, rebound before any source producer. It holds
		// no derived or actor state; all operands live in ActorLoad.Source.
		{"sourceDerive", "sim.SourceDerive"},
		{"carried", "[][]sim.ItemStack"},
		{"equipment", "[][12]sim.ItemInstance"},
		{"purses", "[50]uint32"},
		// The burst picture's installed phase count, install-derived like ghost
		// and carried across a decode, and the blasts of one effect walk that
		// wait for their record; the queue is empty between ticks.
		{"burst", "sim.burstState"},
		{"itemWeights", "[]sim.ItemWeight"},
		{"script", "*sim.Script"},
		{"rom2", "*sim.rom2ScriptState"},
		{"registers", "[100]int32"},
		{"latches", "[1000]uint8"},
		{"won", "uint32"},
		{"lost", "uint32"},
		{"outcome", "sim.Outcome"},
		{"rawSessionHead", "[48]uint8"},
		{"rawSessionMid", "[400]uint8"},
		{"casts", "[]sim.scriptCast"},
		{"bookCasts", "[]sim.bookCast"},
		// engageDrew is pass scratch, cleared at the start of each decision pass
		// and read only inside it; it is in neither the byte form nor the digest.
		{"engageDrew", "[]sim.EntityID"},
		{"deliveries", "[]sim.spellDelivery"},
		{"scrollCasts", "[]sim.ScrollCast"},
		{"effects", "[]sim.cellEffect"},
		{"attached", "[]sim.attachedEffect"},
		{"formations", "[50]uint8"},
		{"autoHealing", "[50]sim.autoHealingPolicy"},
		{"cellTails", "[]sim.cellTail"},
		{"scorchedCells", "[]uint16"},
		{"structureUses", "[]sim.StructureUse"},
		{"areaCosts", "[]sim.areaCostEntry"},
		{"areaCostLive", "bool"},
		{"costWindow", "*sim.savedCostWindow"},
		// 1033 B3: one Structure per placed type-4 record, carried by the
		// byte form and hashed.
		{"structures", "[]sim.Structure"},
		// Legacy footprint or explicit saved-cell cache, rebuilt at load.
		{"structureSlots", "map[uint16]int"},
		{"nativeStructurePlanes", "bool"},
		{"hasSavedStructures", "bool"},
		{"savedStructures", "[]sim.SavedStructure"},
		{"savedStructureCells", "[]sim.SavedStructureCell"},
		{"originalDead", "[]sim.originalDeadRecord"},
		{"currentTerminalActors", "[]sim.CurrentTerminalActor"},
		{"removedNativeBases", "[]sim.NativeActorBasisRecord"},
		{"currentPlayers", "*sim.currentPlayerState"},
		{"savedMotion", "*sim.savedActorMotionState"},
		{"savedCellPlanes", "*sim.SavedCellPlanes"},
		{"savedObjects", "*sim.SavedObjects"},
		// 1131: carried-not-wire-form, the same class as rawSessionHead/
		// rawSessionMid above.
		{"savedCellRecords", "[]sim.SavedCellRecord"},
		// 1132: carried-not-wire-form, the same class immediately above.
		{"savedSpellEffects", "[]sim.SavedSpellEffect"},
		// 1133: same class again.
		{"savedProjectiles", "sim.SavedProjectiles"},
		{"savedWorldEffects", "*sim.SavedWorldEffects"},
		{"savedSpellGraph", "*sim.SavedSpellGraph"},
		{"effectOrder", "[]sim.WorldEffectRef"},
		{"effectWalking", "bool"},
		// 1135: same class again.
		{"savedDiaries", "[]sim.SavedDiary"},
		// Install-derived Diary input (the Units length and each placed
		// creature's row), carried across a decode like rules and ghost.
		{"diary", "sim.diaryRuntime"},
	}},
	{"Entity", reflect.TypeOf(Entity{}), []nstField{
		{"SourceBinding", "sim.SourceBinding"},
		{"NativeBasis", "sim.NativeActorBasis"},
		{"ID", "sim.EntityID"},
		{"X", "int32"},
		{"Y", "int32"},
		{"TargetX", "int32"},
		{"TargetY", "int32"},
		{"Class", "int32"},
		{"HasTarget", "bool"},
		{"Stall", "uint8"},
		{"HP", "int32"},
		{"MaxHP", "int32"},
		{"Withdraw", "int32"},
		{"Wimpy", "int32"},
		{"Domain", "sim.Domain"},
		{"Speed", "int32"},
		{"Transit", "uint16"},
		{"TransitTotal", "uint16"},
		{"Stride", "sim.NativeStride"},
		{"GroupSpeed", "uint8"},
		{"Facing", "uint8"},
		{"DesiredFacing", "uint8"},
		{"TurnRemaining", "uint8"},
		{"TurnTotal", "uint8"},
		{"TurnState", "sim.TurnState"},
		{"PotionStats", "[4]int32"},
		{"PotionHeadroom", "[4]int32"},
		{"Group", "uint32"},
		{"Owner", "uint32"},
		{"SuppressCorpseLoot", "bool"},
		{"KillCreditSource", "sim.EntityID"},
		{"HasKillCredit", "bool"},
		{"KillCreditSpell", "int8"},
		{"ScanRange", "uint8"},
		{"SeeInvisible", "uint8"},
		{"Reach", "uint8"},
		{"AttackTarget", "sim.EntityID"},
		{"AttackTargetKind", "sim.AttackTargetKind"},
		{"HasAttackTarget", "bool"},
		{"PendingAttackTarget", "sim.EntityID"},
		{"PendingAttackTargetKind", "sim.AttackTargetKind"},
		{"HasPendingAttackTarget", "bool"},
		{"PendingOrder", "sim.PendingOrder"},
		{"AdmittedBookSpell", "uint16"},
		{"AcquirePursuit", "bool"},
		{"PursuitIdle", "bool"},
		{"AttackPhase", "sim.AttackPhase"},
		{"AttackCountdown", "int32"},
		{"AttackCharge", "int32"},
		{"AttackRelax", "int32"},
		{"Humanoid", "bool"},
		{"ToHit", "int32"},
		{"Defence", "int32"},
		{"Absorption", "int32"},
		{"DamageBase", "int32"},
		{"DamageSpread", "int32"},
		{"AlwaysHits", "bool"},
		{"Load", "int32"},
		{"Capacity", "int32"},
		{"HumanMovement", "sim.HumanMovement"},
		{"ActorLoad", "sim.ActorLoad"},
		// The decay ladder: the stage a body is at, the dwell it still owes
		// before it is torn down, and the dying time a fresh body takes that
		// dwell from.
		{"Decay", "sim.DecayStage"}, {"Dwell", "uint16"}, {"DyingTime", "int32"},
		// The actor state and the patrol ring and leg (0099 T1): which state
		// machine an entity runs when its group is not deciding for it, and
		// the two cells and the current waypoint of the one arm this build has.
		{"ActorState", "uint8"},
		{"Retreat", "sim.RetreatContinuation"},
		{"PatrolHeadX", "int32"}, {"PatrolHeadY", "int32"},
		{"PatrolTailX", "int32"}, {"PatrolTailY", "int32"},
		{"PatrolLeg", "uint8"},
		// The post (0106 T1): the cell a stance anchors an actor at, written
		// unconditionally by the world constructor and, from T2, by the
		// script's stance commands.
		{"PostX", "int32"}, {"PostY", "int32"},
		// The regeneration record (0109 T1): the mana pool and its maximum,
		// the two regeneration periods, and the two hundredths remainders —
		// carried by the record and the form, read by the regeneration pass
		// a later task adds.
		{"Mana", "int32"}, {"MaxMana", "int32"},
		{"HealthRegenPeriod", "int32"}, {"ManaRegenPeriod", "int32"},
		{"HealthRegeneration", "int32"}, {"ManaRegeneration", "int32"},
		{"ActionClock", "sim.ActionClock"},
		{"attackNotice", "sim.attackNotice"},
		{"RotationSpeed", "int32"},
		{"SecondaryDamage", "sim.SecondaryDamage"},
		{"SecondBase", "uint8"},
		{"SecondSpread", "uint8"},
		{"CurrentProfileBasis", "sim.CurrentProfileBasis"},
		{"HealthHundredths", "uint8"}, {"ManaHundredths", "uint8"},
		// The command group (0117 T1): the group a player order built,
		// read by effectiveGroup ahead of the placed group above and
		// written by nothing this task adds.
		{"CommandGroup", "uint32"},
		// An entity's experience from use: the six per-slot experiences it has
		// earned, its own Mind, its own experience value, the slot a gain it earns
		// is credited to, and whether its class gains at all — read and written
		// by payExperience, a later task's own routine.
		{"SkillXP", "[6]int32"},
		{"Reaction", "int32"}, {"Mind", "int32"}, {"Spirit", "int32"}, {"XPValue", "int32"},
		{"TypeID", "int32"}, {"GoldChance", "int32"},
		{"TreasureMin", "int32"}, {"TreasureMax", "int32"},
		{"XPSlot", "uint8"}, {"GainsXP", "bool"},
		// The entity's own spellbook (0127 T3, FR-4b): a bitmask
		// subscripted by spell id, read by the cast and written by
		// nothing this build adds.
		{"KnownSpells", "uint32"},
		{"Book", "sim.Spellbook"},
		{"CreatureSpells", "[3]sim.CreatureSpell"},
		// The entity's own six skill levels, one per slot, read by a later task's
		// raise and spell power and written by nothing this task adds.
		{"Skill", "[6]int32"},
		{"NativeTraining", "sim.NativeTraining"},
		{"NativeClass", "sim.NativeClass"},
		// A weapon's own spell: the id and the level a caster's weapon-borne cast
		// releases in place of a strike, read by weaponSpell and closedOn and
		// written by nothing this package.
		{"WeaponSpell", "uint16"}, {"WeaponSpellLevel", "int32"},
		{"WeaponSpellSource", "sim.WeaponSpellSource"},
		{"AutoSpell", "uint16"}, {"CastWait", "uint8"},
		{"SpellFX", "uint16"}, {"SpellFXSpell", "uint8"},
		{"Protection", "[5]int32"}, {"Resistance", "[5]uint8"}, {"TokenSize", "uint8"},
		{"EscortTarget", "sim.EntityID"}, {"HasEscortTarget", "bool"},
		{"EscortRange", "uint8"},
		// Map presence: whether the mission script has taken this entity off the
		// map. It is state and not position — the coordinates above it are
		// untouched by the removal, which is why the return arm needs no authored
		// cell.
		{"OffMap", "bool"},
		// The authored map id: the type-6 record's own identifier word this entity
		// was placed under. It is canonical on the same criterion as every row
		// above it -- check opcode 9 writes it into a register, so two worlds
		// differing only in which map id a pursued unit carries would hash the
		// same and resume as each other were it left off this table.
		{"MapUnitID", "uint16"},
	}},
	{"RetreatContinuation", reflect.TypeOf(RetreatContinuation{}), []nstField{
		{"Known", "bool"}, {"Pending", "bool"}, {"Failure", "bool"}, {"Complete", "bool"},
		{"X", "int32"}, {"Y", "int32"}, {"Progress", "uint8"}, {"Counter", "uint8"},
	}},
	{"PendingOrder", reflect.TypeOf(PendingOrder{}), []nstField{
		{"Kind", "uint8"}, {"RowAdmitted", "bool"}, {"Target", "sim.EntityID"}, {"Spell", "uint16"},
		{"X", "int32"}, {"Y", "int32"},
	}},
	{"Bounds", reflect.TypeOf(Bounds{}), []nstField{
		{"Width", "int32"},
		{"Height", "int32"},
	}},
	{"rng", reflect.TypeOf(rng{}), []nstField{
		{"state", "uint64"},
	}},
	{"groupAI", reflect.TypeOf(groupAI{}), []nstField{
		{"owner", "uint32"},
		{"group", "uint32"},
		{"base", "uint8"},
		{"order", "uint8"},
		{"commandedX", "int32"},
		{"commandedY", "int32"},
		{"roamCounter", "uint8"},
	}},
	{"Sack", reflect.TypeOf(Sack{}), []nstField{
		{"ObjectID", "sim.SavedObjectID"},
		{"X", "int32"},
		{"Y", "int32"},
		{"Gold", "uint32"},
		{"Items", "[]uint16"},
		{"ItemInstances", "[]sim.ItemInstance"},
	}},
}

// TestTheCanonicalWorldsFieldSetsArePinned is C-1 as a shape rather than as a
// promise. The occupancy relation is derived while a tick is advanced and stored
// nowhere, and the only mechanical form that claim can take is this: the structs
// hold exactly the fields written out above, and one more on World or on Entity
// fails here — whether or not it changes a single byte, a single digest or a
// single outcome. An index rebuilt from the entities at the top of every tick
// changes none of the three, which is precisely why nothing else in this package
// can see it.
//
// The table moves when the canonical world genuinely gains state, and moving it
// is the declaration that the new field IS canonical: the mode and the grid are
// there because a world's routing must be a property of the world, and both are
// carried by the byte form and enter the digest. What the pin refuses is the
// field nobody declared.
//
// HP and MaxHP are the newest rows, and this pin carries the NEGATIVE half of
// the rule that put them there: the life state is read off those two integers
// and off nothing else, so a Dead bool — or any other flag recording a state the
// pair already answers — fails here by being a field nobody declared. That is
// the only mechanical form C-1 can take. Two sources for one fact are free to
// disagree after a write that moves one and not the other, and no digest, no
// byte form and no outcome can see the second source at all while the two happen
// to agree.
//
// Domain is the newest row, and its entry here is the declaration that a mover's
// movement domain is canonical state: it is carried by the byte form, it enters
// the digest, and it is READ by a step, at every offer a search makes. A field
// nothing read would have been refused by this pin exactly as an occupancy index
// is, and the reason it is admitted is that "where may this mover go" is a
// property of the world and not of whoever is asking.
//
// GroupSpeed is the newest row, and admitting it is the declaration that a
// group's rate term is canonical state rather than a value belonging to whoever
// issued the order. It has to be: it is READ at every cell transit, it survives
// the order that set it, the death of the member it was taken from and a round
// trip through the byte form, and a world resumed without it would move at
// speeds the world it was cut from does not. Left out of the form it would be
// exactly the second source this pin exists to refuse — a fact about how fast a
// unit walks that no digest, no byte form and no replay can see.
//
// THE SCRIPT AND ITS FOUR STATE FIELDS are the newest rows, and they are two
// declarations rather than one. The registers, the latches and the two counters
// are canonical for the plainest reason any field here is: a check writes a
// register every pass, a trigger reads one, a latch is the whole of why a
// one-shot trigger fires once, and the counters are the only thing that decides a
// mission — a world resumed without them replays a script it has already spent.
//
// THE COMPILED SCRIPT ITSELF is the row that had a rival, and the rival is what
// this pin exists to refuse. It could have been an input: the original re-derives
// it from the map at every load and stores none of it, so leaving it outside the
// world would have been the faithful-looking choice. It fails here on the
// routes' own criterion — a world here carries no map, so a script outside the
// form would be a fact about how the world behaves that no byte form, no digest
// and no replay can see, which is precisely the second source this table refuses.
//
// THE TWELVE ATTACK ROWS are the newest, and admitting them is the declaration
// that an attack order, the cycle it drives and the seven numbers a blow reads
// are canonical state. Each is READ by a step and each survives a round trip: a
// world resumed without the cycle would strike on different ticks from the world
// it was cut from, and one resumed without the seven would resolve different
// blows. Left out of the form they would be exactly the second source this pin
// exists to refuse.
//
// They and the script rows above arrived from two different stories at once, and
// the table is what makes that harmless: the two sets are disjoint, they live in
// different halves of the form, and a pin that lists every field by name cannot
// be satisfied by a merge that kept only one of them.
//
// OWNER is the newest row, and admitting it is the declaration that the roster
// slot a map placed a unit under is canonical state. It is the one row here
// admitted while NOTHING READS IT, and that is the declaration rather than an
// exception to it: the criterion this table applies is whether a fact about the
// world would be invisible to the byte form, the digest and a replay, not whether
// a step consults it. An owner left outside the form would be exactly that — a
// world resumed from its bytes would have forgotten who owns what, and the two
// script arms that hand ownership over would run to different answers in a
// resumed world than in the one it was cut from. The class key is admitted on
// the same footing and for the same reason.
//
// COST and HEIGHT are the newest rows, and admitting them is the declaration that
// what a cell costs a ground mover and how high it stands are properties of the
// WORLD rather than of whoever is asking. They clear the bar this table sets
// twice over: both are READ while a world is advanced — cost at every offer a
// ground search makes and at every transit start, height at every transit start
// — and both are carried by the byte form and hashed with everything else. Left
// outside, they would be exactly the second source this pin refuses: two worlds
// standing on different ground would route the same, cross at the same speed and
// hash the same.
//
// RELATIONS is the newest row, and admitting it is the declaration that who is
// hostile to whom is a property of the WORLD rather than of whoever is asking.
// It clears the bar this table sets on both counts: it is READ while a world is
// advanced — once per group per decision, at the filter that turns everything a
// group can see into the things it will fight — and it is carried by the byte
// form and hashed with everything else. Left outside, it would be exactly the
// second source this pin refuses: two worlds whose factions are at war and at
// peace would hash the same and resume as each other.
//
// groups AND groupAI ITSELF are the newest rows, and admitting the type as
// well as the field is 0095's own declaration alongside Bounds and rng's: a
// field added inside groupAI is new canonical state just as much as one added
// to World, and would be just as invisible to a pin that stopped at the outer
// struct. groups clears this table's bar on both counts a field here is ever
// admitted for — canonical because two worlds differing only in one base ARE
// two worlds, and carried by the byte form and hashed with everything else —
// even though NOTHING IN A TICK READS IT YET: the criterion this table applies
// is whether a fact about the world would be invisible to the byte form, the
// digest and a replay, not whether a step consults it, which is the ground
// Owner was admitted on before anything read it either.
//
// routes is an earlier story's row, and its entry here is that declaration and not a
// note of what happened. A stored route could have been a cache — a derivation
// kept for speed, outside the form and outside the digest — and measurement said
// otherwise: recomputing a stored route from its own later cells does not give
// the tail back, so a world resumed from its bytes would walk a different route
// from one that was never interrupted. Declared canonical, it is carried by the
// form and hashed with everything else; had it been left a cache, this pin is
// what would have refused it.
//
// sacks AND Sack ITSELF are the newest rows (0103), and admitting the type
// as well as the field is groupAI's own declaration repeated: a field added
// inside Sack is new canonical state just as much as one added to World, and
// would be just as invisible to a pin that stopped at the outer struct.
// sacks clears this table's bar on the ground the group list was admitted
// on: two worlds differing only in one sack's cell, purse or contents ARE
// two worlds, and the list is carried by the byte form and hashed with
// everything else, even though NOTHING ON A TICK PATH YET READS ONE — the
// criterion this table applies is whether a fact about the world would be
// invisible to the byte form, the digest and a replay, not whether a step
// consults it, which is the ground groups was admitted on before anything
// read it either.
//
// carried AND purses are the newest rows (0112 T1), and they are declared
// here a task AHEAD of the byte form and the digest, which no earlier row
// has done: T1's own boundary is that the container and the purse exist, are
// built and are read back, and do not reach the byte form yet. Left off this
// table, a field added to World would go unnoticed by every mechanical check
// there is, which is exactly the shape this pin exists to refuse regardless
// of which task wires the bytes. The crossing itself — carried and purses
// entering MarshalBinary and Hash — is the very next task's, measured in
// binary_test.go and hash_test.go, not here.
//
// spells AND KnownSpells are 0127 T3's own rows, and they were declared
// there a task ahead of the byte form and the digest on carried's own
// precedent: the table on World and the bitmask on Entity both exist, are
// built by the constructor and are read by the cast, before either crosses
// MarshalBinary or Hash. That crossing was measured in binary_test.go and
// hash_test.go, not here — what this table asks is only whether the
// canonical world HOLDS the field, which is true from the moment the
// constructor and the reader exist.
//
// Skill was the newest row, on KnownSpells' own precedent and not even a
// task ahead of the byte form and the digest: the crossing landed in the
// very same commit, so the six levels are canonical and carried by the byte
// form together.
//
// AutoSpell, CastWait, SpellFX AND SpellFXSpell are the newest rows now, and
// on WeaponSpell's own precedent they too arrive ALREADY CROSSING the byte
// form and the digest. The criterion is unchanged: two worlds differing only
// in which spell a unit has on repeat, or in which unit a spell has just
// touched, would hash the same and resume as each other were any of the four
// left off this table.
//
// WeaponSpell AND WeaponSpellLevel are the rows below them, and on Skill's
// own precedent they too arrive ALREADY CROSSING the byte form and the
// digest — binary_test.go and hash_test.go move in this same task. What
// this table asks is unchanged by that: the pair is canonical the moment
// weaponSpell (spell.go) and closedOn (combat.go) exist to read it, and the
// criterion is whether a fact about the world would be invisible to the byte
// form, the digest and a replay, which a weapon's own spell would be were
// either field left off this table — two casters differing only in what
// their weapons cast would hash the same and resume as each other.
func TestTheCanonicalWorldsFieldSetsArePinned(t *testing.T) {
	for _, tc := range nstPinned {
		got := make([]nstField, 0, tc.typ.NumField())
		for i := 0; i < tc.typ.NumField(); i++ {
			f := tc.typ.Field(i)
			got = append(got, nstField{name: f.Name, typ: f.Type.String()})
		}
		if !reflect.DeepEqual(got, tc.fields) {
			t.Errorf("%s holds %v, pinned as %v — a field added to the canonical world is state the "+
				"occupancy rule may not keep, and neither the byte form nor the digest can see one",
				tc.what, got, tc.fields)
		}
	}
}

// ---------------------------------------------------------------- the byte form

// nstForm is one world's canonical bytes, with the marshaller's never-nil error
// handled once.
func nstForm(t *testing.T, w *World) []byte {
	t.Helper()
	b, err := w.MarshalBinary()
	if err != nil {
		t.Fatalf("MarshalBinary: %v", err)
	}
	return b
}

func nstShape(t *testing.T, tick int, form []byte, cells, units, routeCells, groups int) {
	t.Helper()
	// sackCountLen is added unconditionally and nothing more: no world this
	// file builds ever holds a sack (0103 added the type and the constructor
	// parameter, not a caller here), so the section is always its bare empty
	// count. carryCountLen*units and purseLen are added the same way (0112): no
	// world this file builds ever names a stock or credits any gold, so the
	// carry section is one bare zero count per unit and the purse section is
	// always its bare empty width. equipRecordLen*units is added the same way
	// again (0124 T2): no world this file builds equips anything, so the
	// equipment section is one bare EquipSlots-wide zero record per unit, no
	// count at all. spellCountLen is added the same way once more: no world
	// this file builds ever names a spell table, so the section is always its
	// own bare two-byte zero count.
	if want := 61 + 34 + 3*cells + relationLen + 492*units + 4*units + 8*routeCells +
		groupCountLen + groupRecordLen*groups + sackCountLen + carryCountLen*units + equipRecordLen*units + treasureRecordLen*units + purseLen + spellCountLen + itemWeightCountLen + 2*castingCountLen + relationSlots + tailCountLen + structureCountLen + len(emptyItemStatePin(units)) + 4 + 4 + 4 + 4 + 4 + scriptStateLen + scriptCountsLen + entityIDFloorLen + spellDeliverySpanLen; len(form) != want {
		t.Fatalf("tick %d: the form of a %d-unit world over %d cell(s) carrying %d route cell(s) and "+
			"%d group record(s) is "+
			"%d bytes, want %d — the header, the THREE planes, the per-unit record width and the "+
			"route section are what the contract says",
			tick, units, cells, routeCells, groups, len(form), want)
	}
	if form[0] != formatVersion {
		t.Fatalf("tick %d: the form's version byte is %d, want %d", tick, form[0], formatVersion)
	}
}

// nstRouteCells is how many route cells a world is carrying in all.
func nstRouteCells(w *World) int {
	n := 0
	for _, r := range w.routes {
		n += len(r)
	}
	return n
}

func TestTheByteFormAndItsDigestAreTheOnesAlreadyPinned(t *testing.T) {
	w := pinWorld(t)
	form := nstForm(t, w)
	nstShape(t, 0, form, 20, 3, 0, len(w.groups))
	if !bytes.Equal(form, pinBytes) {
		t.Errorf("the pinned world marshals to\n % x\npinned as\n % x", form, pinBytes)
	}
	if got := w.Hash(); got != pinDigest {
		t.Errorf("the pinned world hashes %#016x, pinned as %#016x", got, pinDigest)
	}

	var back World
	if err := back.UnmarshalBinary(pinBytes); err != nil {
		t.Fatalf("UnmarshalBinary(pinBytes): %v — a form this version wrote must still decode", err)
	}
	if got := back.Hash(); got != pinDigest {
		t.Errorf("the decoded pin hashes %#016x, pinned as %#016x", got, pinDigest)
	}
}

// ---------------------------------------------------------------- the crossing

// nstCells is how many grid cell bytes a form over conBounds carries: one per
// in-bounds cell, written out here rather than multiplied out of the bounds
// value, since that is the arithmetic the encoder is being measured against.
const nstCells = 16 * 16

// nstWall is the cell the line below is ordered onto, held at the start by a unit
// that has nowhere to be. It is what makes the line jam rather than simply walk.
var nstWall = [2]int32{8, 2}

// nstStart is the run's world. Three units in a line along x with the LEADER
// carrying the highest id, so the line stretches and every follower spends ticks
// refused; a target-less unit standing on nstWall, so the line then jams against
// something that is not contending for anything; and one more unit sent in
// diagonally later, which ends the run deadlocked against the rear of the line.
// The ids are pairwise non-adjacent.
//
// The run has to contend on nearly every tick, and that is the whole reason for
// this shape rather than a handful of movers: two worlds compared over a run where
// the occupancy rule decided nothing would agree for reasons that have nothing to
// do with the rule.
var nstStart = []Entity{
	{ID: 1, X: 2, Y: 2},
	{ID: 4, X: 3, Y: 2},
	{ID: 7, X: 4, Y: 2},
	{ID: 10, X: nstWall[0], Y: nstWall[1]},
	{ID: 13, X: 2, Y: 5},
}

// nstSchedule is one command slice per tick. Orders arrive on four separate ticks,
// so a world decoded at a cut is not walking a path that was already settled before
// it: the wall is sent away at tick 5 and the rear of the line is re-ordered at
// tick 8, both after most of the cuts this run takes.
var nstSchedule = [][]Command{
	{{Entity: 1, X: nstWall[0], Y: nstWall[1]},
		{Entity: 4, X: nstWall[0], Y: nstWall[1]},
		{Entity: 7, X: nstWall[0], Y: nstWall[1]}},
	nil,
	{{Entity: 13, X: 6, Y: 2}},
	nil,
	{{Entity: 10, X: 12, Y: 6}},
	nil,
	nil,
	{{Entity: 1, X: 2, Y: 9}},
	nil,
}

// nstFixture checks the run stages what it claims to, on the test's own
// arithmetic: a well-formed start, ids that no neighbour comparison can answer by
// accident, every order given as a command rather than carried in, the wall
// actually held by a unit with nothing to do, and every command naming a unit the
// world holds.
func nstFixture(t *testing.T) {
	t.Helper()
	conDistinct(t, 0, nstStart)
	for i, e := range nstStart {
		if e.HasTarget {
			t.Fatalf("fixture: unit %d carries a target (%d,%d); every order here is a command",
				e.ID, e.TargetX, e.TargetY)
		}
		if i > 0 {
			prev := nstStart[i-1].ID
			if e.ID <= prev || e.ID == prev+1 {
				t.Fatalf("fixture: ids %d and %d must ascend and must not be adjacent", prev, e.ID)
			}
		}
	}

	var wall Entity
	standing := false
	for _, e := range nstStart {
		if e.X == nstWall[0] && e.Y == nstWall[1] {
			wall, standing = e, true
		}
	}
	if !standing {
		t.Fatalf("fixture: nothing stands on the wall cell (%d,%d) — the line would not jam",
			nstWall[0], nstWall[1])
	}
	for _, cmds := range nstSchedule {
		for _, c := range cmds {
			found := false
			for _, e := range nstStart {
				found = found || e.ID == c.Entity
			}
			if !found {
				t.Fatalf("fixture: a command names unit %d, which this world does not hold", c.Entity)
			}
			// The order that clears the wall has to send that unit somewhere: an
			// order naming the cell it is standing on would be a no-op, and the jam
			// this run is built on would never release.
			if c.Entity == wall.ID && dist(wall, c) == 0 {
				t.Fatalf("fixture: unit %d on the wall is ordered onto the cell it stands on", wall.ID)
			}
		}
	}
}

// nstRun is the contended run recorded from the first world: the canonical form and
// the digest after every tick INCLUDING tick 0, so forms[k] and digests[k] are the
// world at tick k, and the cells after every advanced tick, which is what the
// contention check reads.
type nstRun struct {
	forms   [][]byte
	digests []uint64
	cells   [][]Entity
}

func nstRecord(t *testing.T) nstRun {
	t.Helper()
	w := mustWorld(t, 1, conBounds, nstStart)
	out := nstRun{forms: [][]byte{nstForm(t, w)}, digests: []uint64{w.Hash()}}
	nstShape(t, 0, out.forms[0], nstCells, len(nstStart), nstRouteCells(w), len(w.groups))
	for tick, cmds := range nstSchedule {
		Step(w, cmds)
		form := nstForm(t, w)
		nstShape(t, tick+1, form, nstCells, len(nstStart), nstRouteCells(w), len(w.groups))
		out.forms = append(out.forms, form)
		out.digests = append(out.digests, w.Hash())
		out.cells = append(out.cells, w.Entities())
	}
	return out
}

// TestADecodedWorldStepsOnToTheSameDigestsAsTheFirst is the crossing, and it is
// the one criterion here that a design storing occupancy somewhere the encoding
// cannot reach fails. At every tick of the run the world is marshalled and decoded
// into a second world, and that second world is then stepped through the rest of
// the schedule beside the first, digest for digest to the end. Everything the rule
// needs has to come back out of those bytes: an index carried on the world and not
// rebuilt after a decode leaves the second world blocking nobody, and an index held
// at package level is worse — it belongs to whichever world stepped last, so with
// several worlds alive in one process it cannot be right for all of them. Either
// way the digests part company.
//
// The run's claim to have contended at all is the solo comparison, not the
// standstill: two worlds agree perfectly over a rule that moves nobody.
func TestADecodedWorldStepsOnToTheSameDigestsAsTheFirst(t *testing.T) {
	nstFixture(t)
	run := nstRecord(t)
	conRefused(t, "the crossing's run", nstStart, run.cells, nstSchedule)

	for cut := range run.forms {
		var shadow World
		if err := shadow.UnmarshalBinary(run.forms[cut]); err != nil {
			t.Fatalf("cut at tick %d: UnmarshalBinary: %v", cut, err)
		}
		if got := shadow.Hash(); got != run.digests[cut] {
			t.Fatalf("cut at tick %d: the decoded world hashes %#016x, the first world %#016x — "+
				"the crossing has to start from the same world it cut", cut, got, run.digests[cut])
		}
		if got := shadow.Tick(); got != uint64(cut) {
			t.Fatalf("cut at tick %d: the decoded world is at tick %d", cut, got)
		}

		for tick := cut; tick < len(nstSchedule); tick++ {
			Step(&shadow, nstSchedule[tick])
			if got, want := shadow.Hash(), run.digests[tick+1]; got != want {
				t.Fatalf("cut at tick %d, then tick %d: the decoded world hashes %#016x and the "+
					"first world %#016x — whatever the rule reads is not all in the byte form\n"+
					" decoded %+v\n first   %+v",
					cut, tick+1, got, want, shadow.Entities(), run.cells[tick])
			}
			if got := nstForm(t, &shadow); !bytes.Equal(got, run.forms[tick+1]) {
				t.Fatalf("cut at tick %d, then tick %d: the two worlds' forms differ\n % x\n % x",
					cut, tick+1, got, run.forms[tick+1])
			}
		}
	}
}

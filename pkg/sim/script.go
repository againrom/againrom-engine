package sim

// Mission scripts evaluate checks into registers, then execute passing triggers
// in stored order. The dialect selects opcode support and outcome semantics.
// Unsupported checks make dependent triggers inert so an unwritten register
// cannot satisfy an authored comparison against zero.

import (
	"fmt"
	"sort"
)

// The two fixed arrays and their caps, which are CUSTOMISATION LIMITS and are
// written here rather than smoothed away. The original allocates exactly these
// two and its own serializer states both lengths again; the shipped corpus's
// maxima are 64 checks and 37 triggers, so both caps have room in them.
//
// The original BOUNDS-CHECKS NEITHER, so a map with more than 100 checks
// overwrites the latch array and one with more than 1000 triggers runs off the
// end of it. That overrun is NOT reproduced — see registerAt and the two writers
// below, which refuse an out-of-range subscript instead. It is the one place
// this file knowingly parts company with what it reconstructs, and it parts
// company only where the original's own behaviour is undefined.
const (
	scriptRegisters = 100  // signed int result slots, one per compiled check
	scriptLatches   = 1000 // fire-once latch bytes, one per trigger position
)

// sessionRawHeadLen and sessionRawMidLen are the two opaque session-block
// spans SAV-SESS-031 locates but promotes no meaning for: 48 bytes at block
// offset 1400 and 400 at 1448 (pkg/formats/sav's own rawHeadLen/rawMidLen,
// restated here since pkg/sim does not import a format decoder). This
// package carries them byte for byte and never reads a bit of either.
const (
	sessionRawHeadLen = 48
	sessionRawMidLen  = 400
)

// ROM1 check opcodes. ROM2 additions are declared in rom2script.go.
// ScriptCheckConstant initializes a register during compilation.
const (
	ScriptCheckGroupCount    int32 = 1       // how many living units the named group still has
	ScriptCheckInBox         int32 = 2       // 1 iff the unit's cell is inside (p0,p1)..(p2,p3)
	ScriptCheckWithin        int32 = 3       // 1 iff distance(unit, (p0,p1)) <= p2
	ScriptCheckHealth        int32 = 4       // the unit's current health, gated on p0 == 6 (1033 B1, TRIG-CHECK-051)
	ScriptCheckAlive         int32 = 5       // 0 iff the unit is dead, else 1
	ScriptCheckUnitDistance  int32 = 6       // distance(unitA, unitB) & 0xff; 0xff if either is dead
	ScriptCheckDistance      int32 = 7       // distance(unit, (p0,p1)) & 0xff
	ScriptCheckPopulation    int32 = 8       // how many living units the named player has
	ScriptCheckTargetID      int32 = 9       // the authored map id of the unit this one is pursuing, else 0
	ScriptCheckRelation      int32 = 10      // relation[first player][second player] & 3; names no unit
	ScriptCheckDead11        int32 = 11      // a DEAD ARM: dispatched, writes no register
	ScriptCheckItemTestAlias int32 = 12      // byte-identical alias of check 17 (TRIG-ITEMTEST-040)
	ScriptCheckDead13        int32 = 13      // a DEAD ARM: dispatched, writes no register
	ScriptCheckSackAt        int32 = 14      // 1 iff a ground sack occupies (p0&0xff, p1&0xff); names no unit
	ScriptCheckNearest       int32 = 15      // least distance(unit, (p0,p1)) over the player's living units & 0xff; 0xff if none
	ScriptCheckItemDistance  int32 = 16      // distance(unit, (p0,p1)) & 0xff if the unit's container holds the named item, else 0xff (1033 B2, TRIG-CHECK-052)
	ScriptCheckItemTest      int32 = 17      // 1 iff the unit's container holds the named item code, else 0
	ScriptCheckVIP           int32 = 18      // writes NO register; a dead unit increments the lose counter
	ScriptCheckVariable      int32 = 19      // register = register[p0]
	ScriptCheckStructField   int32 = 21      // the referenced structure's Field42, sign-extended; names no unit (1033 B3, TRIG-CHECK-053)
	ScriptCheckConstant      int32 = 0x10002 // build-time preset; never dispatched
)

// The instant opcodes this build runs, out of a 34-arm table.
//
// The two HAND-OVER arms are one arm twice over: both take a player as their
// second parameter and differ only in what they name first. Neither reads
// anything — they are the only writers of an entity's owner in this package —
// and implementing them ARMS NO TRIGGER, because inertness is derived from
// unimplemented CHECKS and this adds none.
const (
	// ScriptInstantMessage names event text in Args[0]. ROM1 presentation reads
	// latch metadata; ROM2 reports each execution without changing world state.
	ScriptInstantMessage int32 = 2

	ScriptInstantSetVariable int32 = 3  // register[p0] = p1
	ScriptInstantWin         int32 = 4  // the win counter
	ScriptInstantLose        int32 = 5  // the lose counter
	ScriptInstantIncVariable int32 = 8  // register[p0]++
	ScriptInstantRelation    int32 = 10 // relation[p0][p1] = (relation[p0][p1] &^ 3) + p2; one direction
	ScriptInstantGiveUnit    int32 = 19 // the named unit's owner = the named player
	ScriptInstantGiveGroup   int32 = 22 // every member of the named group, likewise
	ScriptInstantGiveMoney   int32 = 23 // purses[player] += p0, wrapping at 32 bits

	// ScriptInstantGiveAll is opcode 28, Give All: it moves the WHOLE of Unit's
	// container onto Unit2's and leaves Unit's empty. UNLIKE THE TWO HAND-OVER
	// ARMS ABOVE it writes no owner — it is the one arm in the whole
	// vocabulary that needs a SECOND unit reference rather than a player, which
	// is the whole reason ScriptInstant grew one. It touches neither entity's
	// equipment, gold, position, group, state or tick, and implementing it ARMS
	// NO TRIGGER, on the two arms above's own ground: inertness is derived from
	// unimplemented CHECKS and this adds none.
	ScriptInstantGiveAll int32 = 28

	// The two ITEM arms. Each names one unit and one item code and writes that
	// unit's own CONTAINER — never its twelve worn places, which are
	// different state.
	//
	// 12 CREATES: it makes one unit of the code out of nothing and adds it.
	// 13 TAKES ONE UNIT AWAY and destroys it: an element at a count of 1
	// disappears, an element above 1 loses one from its count and keeps its
	// place. Neither is a transfer — opcode 11 is the transfer, and no shipped
	// map authors it.
	//
	// The pair is the map's own way of handing out a quest item and consuming
	// it at its destination. Implementing them ARMS NO TRIGGER, on the
	// hand-over arms' own ground: inertness is derived from unimplemented
	// CHECKS, and the check that reads an inventory — opcode 17 — is not
	// added here.
	ScriptInstantAddItem  int32 = 12
	ScriptInstantTakeItem int32 = 13

	// THE FIVE MAP-PRESENCE ARMS. They are the script's power to take a unit
	// off the map and put it back, and the state they move is one bit per
	// entity — see Entity.OffMap (world.go) for what the bit is and
	// presence.go for the two arms themselves.
	//
	// 16 REMOVES ONE UNIT and 17 RETURNS IT to the cell it never stopped
	// carrying. 32 and 33 are those two arms applied to every member of a
	// group, which is exactly what the original does: the two group arms are
	// one 24-instruction body twice over, differing only in which helper they
	// call (`TRIG-MAPGROUP-043`). 18 is a removal and a placement of a
	// SECOND unit at the first's cell — the shipped parameter names read
	// "Unit's true form" and "Unit's polymorphed form".
	//
	// IMPLEMENTING THEM ARMS NO TRIGGER, on the hand-over arms' own ground:
	// inertness is derived from unimplemented CHECKS and this adds none.
	ScriptInstantTakeOffMap  int32 = 16
	ScriptInstantReturnToMap int32 = 17
	ScriptInstantSwapOnMap   int32 = 18
	ScriptInstantGroupOffMap int32 = 32
	ScriptInstantGroupOnMap  int32 = 33

	// ScriptInstantGroupOrder is the group command: a SECOND DISPATCH on the
	// node's own first plain parameter (Args[0]), which names one of eleven
	// catalogue literals. Ten act here: the nine shipped values 1, 2, 3, 4, 5,
	// 10, 11, 14 and 15, plus command17 (1148). Catalogue literal18 is inert in
	// both. Every other value changes nothing. Patrol is unlike the five direct
	// group-order siblings: it does not set one of the five group orders at
	// all, it clears the group's own order to 0 and hands each member to the
	// per-actor layer instead (actor.go). The group named is carried on the
	// compiled record's own Group/HasGroup, never Args: a node naming no group
	// changes nothing either.
	ScriptInstantGroupOrder int32 = 6

	// THE THREE SPELL ARMS. 21 and 24 build a TEMPORARY CASTER — one
	// constructor with exactly two callers, and they are the two — and 29
	// re-times an area effect standing on a cell. 21 aims at a cell and 24 at a
	// unit; the two states the decoded machine calls 0xe and 0xd are this
	// build's HasTarget (scriptcast.go).
	//
	// 29's subject is what 21 leaves behind: every shipped instant-29 node
	// names a cell a shipped instant-21 node cast at, on the same map, and
	// matches on that cast's own spell id (`TRIG-CELLEFFECT-045`).
	//
	// IMPLEMENTING THEM ARMS NO TRIGGER, on the hand-over arms' own ground:
	// inertness is derived from unimplemented CHECKS and this adds none.
	ScriptInstantCastAtCell    int32 = 21
	ScriptInstantCastAtUnit    int32 = 24
	ScriptInstantCellEffectAge int32 = 29
)

// THE LAST FIVE INSTANTS. Together with the group sub-commands above, they
// complete the runtime operation vocabulary reachable from shipped triggers.
// Instant 20 and group sub-command 1 are authored but trigger-unreachable;
// group sub-command17 is implemented but absent from the measured shipped
// maps (AI-GROUPCMD-020).
const (
	// ScriptInstantFormation is opcode 7, which the shipped
	// `Description Instants.ini` names `Set formation`
	// (`AI-FORM-037`): the named player's formation mode becomes `(u8)p0`,
	// raw and unremapped. The campaign authors ONE node, on map 110, and
	// under `TRIG-PARAM-030`'s slot rule its `p0` is 0 — never in formation
	// — beside the author's own label, "brigands dont use formations".
	ScriptInstantFormation int32 = 7

	// ScriptInstantDropAll is opcode 20, Drop all (`TRIG-DROPALL-024`): the
	// unit's whole container goes to the ground AT THE UNIT'S OWN CELL,
	// merging into a sack already standing there, and the unit is re-seated
	// with a fresh empty one. The node carries no coordinate.
	ScriptInstantDropAll int32 = 20

	// ScriptInstantCellTail is opcode 25 (`TRIG-CELLTAIL-035`): six bytes
	// onto the addressed cell's own record. See celltail.go.
	ScriptInstantCellTail int32 = 25

	// ScriptInstantUnitEffectAge is opcode 30, the unit-side twin of opcode
	// 29 (`TRIG-EFFECTTIME-034`): the attached effect whose id matches
	// `(u8)p0` gets `(u16)p1` as its remaining duration. 29 re-times what
	// stands on a CELL and this re-times what is attached to a UNIT.
	ScriptInstantUnitEffectAge int32 = 30

	// ScriptInstantProperty is opcode 34 (`TRIG-PROPERTY-036`): a word-wide
	// property setter selected by `p0`. Three selectors store, every other
	// value stores nothing, and all paths then notify — which writes no
	// simulation state and so is nothing this build has to do.
	ScriptInstantProperty int32 = 34

	// ScriptInstantStructField is opcode 26 (1033 B3, `TRIG-CHECK-053`): the
	// referenced structure's Field42 becomes the low 16 bits of p0,
	// unconditionally — the setter half of check opcode 21's getter, over the
	// same field.
	ScriptInstantStructField int32 = 26
)

// The three PROPERTY SELECTORS instant 34 defines. They are the arm's own three
// immediates and not an enumeration this build invented; a selector outside them
// reaches no store at all, which is the arm's own fall-through.
const (
	propertyHealth     int32 = 6
	propertyDefence    int32 = 15
	propertyAbsorption int32 = 16
)

// scriptCheckSupported and scriptInstantSupported are the two tables that decide
// what this build can evaluate. They are the ONLY place either answer is given,
// so the compile-time report and the runtime dispatch cannot come to disagree
// about which arms exist — a switch with a silent default is exactly how an
// unimplemented arm becomes an evaluated-false one.
func scriptCheckSupported(op int32) bool {
	switch op {
	case ScriptCheckGroupCount,
		ScriptCheckInBox, ScriptCheckWithin, ScriptCheckHealth, ScriptCheckAlive,
		ScriptCheckUnitDistance, ScriptCheckDistance, ScriptCheckPopulation,
		ScriptCheckTargetID,
		ScriptCheckRelation, ScriptCheckDead11, ScriptCheckItemTestAlias, ScriptCheckDead13,
		ScriptCheckSackAt, ScriptCheckNearest, ScriptCheckItemDistance, ScriptCheckItemTest,
		ScriptCheckVIP,
		ScriptCheckVariable, ScriptCheckStructField, ScriptCheckConstant:
		return true
	}
	return false
}

// scriptInstantSupported takes the WHOLE instant, not the bare opcode:
// opcode 6 is a second dispatch, so whether this build runs a given node is
// a property of (Op, Args[0]) and not of Op alone. Every other opcode's
// answer is unchanged and reads only Op, exactly as before.
func scriptInstantSupported(in ScriptInstant) bool {
	switch in.Op {
	case ScriptInstantMessage,
		ScriptInstantSetVariable, ScriptInstantWin, ScriptInstantLose,
		ScriptInstantIncVariable, ScriptInstantRelation, ScriptInstantGiveUnit,
		ScriptInstantGiveGroup, ScriptInstantGiveMoney, ScriptInstantGiveAll,
		ScriptInstantAddItem, ScriptInstantTakeItem,
		ScriptInstantTakeOffMap, ScriptInstantReturnToMap, ScriptInstantSwapOnMap,
		ScriptInstantGroupOffMap, ScriptInstantGroupOnMap,
		ScriptInstantCastAtCell, ScriptInstantCastAtUnit, ScriptInstantCellEffectAge,
		ScriptInstantFormation, ScriptInstantDropAll, ScriptInstantCellTail,
		ScriptInstantUnitEffectAge, ScriptInstantProperty, ScriptInstantStructField:
		return true
	case ScriptInstantGroupOrder:
		return groupOrderSupported(in.Args[0])
	}
	return false
}

// groupOrderSupported is opcode 6's own sub-table (SC-2, 1148): true for the
// ten runtime handlers. Catalogue18 is inert. Values outside the catalogue
// are unsupported too.
//
// THIS TABLE AND cmdGroupOrder's OWN SWITCH CHANGE TOGETHER: one table and one
// dispatch, so a compile-time support report cannot claim an arm the runtime
// switch does not carry.
func groupOrderSupported(sub int32) bool {
	switch sub {
	case int32(orderGuard), int32(orderSwarm), int32(orderStandGround),
		int32(orderMove), int32(orderSwarm2), subCommandPatrol,
		subCommandAttack, subCommandDefend, subCommandFollow, int32(orderRoam):
		return true
	}
	return false
}

// subCommandPatrol is opcode 6's sub-command 14, Patrol. It is named
// separately from the five orderXxx constants beside it rather than reusing
// one of them, because it is not one of them: every one of the five writes
// its OWN number onto the group record as the order the group is now under,
// where Patrol writes 0 (orderNone) instead — the one sub-command whose
// own number is not the order it leaves behind.
const subCommandPatrol int32 = 14

// subCommandAttack, subCommandDefend and subCommandFollow are opcode 6's
// last three arms. They are named separately from the five orderXxx
// constants for subCommandPatrol's own reason and a second one: none of the
// three writes its own number onto the group record as the order the group
// is under — all three set the group's order to none — and none of them
// is a group order at all in the sense the other five are. What they write
// is PER MEMBER.
//
// `AI-SCRIPTATTACK-120` reads 10, `AI-FOLLOWSET-116` and `TRIG-GRPARM-047` read
// 11 and 15, and 11 and 15 are one body in the original emitted twice, differing
// in three bytes of which one is the state immediate.
const (
	subCommandAttack int32 = 10
	subCommandDefend int32 = 11
	subCommandFollow int32 = 15
)

// scriptCheckHealthGate is check opcode 4's own gate value (1033 B1,
// `TRIG-CHECK-051`): a call-site literal in the original, not a stored field,
// so raising it changes no shipped file (`TRIG-CHECK-054`).
const scriptCheckHealthGate int32 = 6

// scriptParams is how many plain integer parameters a node carries. It is the
// record's own width and not a convenience: parameters are stored BY SLOT, so
// the tenth may be used while the fourth is not.
const scriptParams = 10

// ScriptCheck is one compiled check: what to measure, which register to write it
// into, and the entities the measurement reads.
//
// Register is the check's OWN subscript, assigned once at compile time, and a
// check writes it once per pass and writes nothing else. Two checks never share
// one — that is refused when the script is built.
//
// Unit and Unit2 are ALREADY RESOLVED and are entity ids, not the ids the map
// wrote. Resolving them is the binder's job one tier up, and it has to be:
// a reference may name a placed record, a hero ordinal resolved against the live
// party, or an entry in a static name table, and only the tier that assembled
// the world knows which entity any of the three became. HasUnit and HasUnit2
// say whether the reference resolved at all, because entity id zero is a real
// entity and no id value is free to mean "none".
//
// A check whose reference did not resolve measures nothing and writes no
// register — the same treatment an unimplemented arm gets, and for the same
// reason.
//
// Group and HasGroup are a THIRD reference and are not one of the two above.
// A group parameter names one id space and not three, so it needs no resolving —
// it is the placed record's own word, carried across as it stands — but it does
// need a presence flag of its own, because group zero is a real group and no id
// value is free to mean "none". It is carried separately from Args for the
// reason the unit references are: the plain parameters are packed in encounter
// order, and admitting a group into that packing would move the parameters of
// every OTHER node that carries one, arms this build does not evaluate included.
//
// Player and Player2 are a FOURTH and FIFTH reference — check opcode 10 is
// the one arm of the vocabulary that names two players at once, so the
// record carries a pair on the same terms Unit and Unit2 already do. Like
// Group, and unlike Unit, a player is CARRIED rather than resolved: a
// Target_Player value names exactly one thing, the same 1-based roster slot
// space a placed record's own owner field is in, so there is nothing to look
// it up in and nothing that can fail. They join neither Args nor Group: the
// plain parameters are packed in encounter order, and admitting a reference
// into that packing would move the parameters of every OTHER node that
// carries one, arms this build does not evaluate included. Item and HasItem
// are the check record's item reference. They are ScriptInstant's own sixth
// reference, carried on that field's own terms and for the reasons its doc
// already gives: a packed uint16 item code, carried and not resolved,
// converted from the map file's Target_Item by the loader rather than by
// this package, with a presence flag of its own so that one reference slot
// on this record does not read differently from every other one.
//
// The check record needed one because check opcode 17 reads one. That arm takes
// the item code out of the compiled record and the container off the named unit
// (`TRIG-ITEMTEST-040`), and until this field existed pkg/mapload decoded the
// reference for a check node and then dropped it while building the record.
// bindParams already produced the value; the check builder did not carry it.
//
// The 23 shipped check-17 nodes all bind exactly [0:Unit t=4][1:Item t=8] AS
// THE MAP FILES AUTHOR THEM, so every one of them names both references when
// the script is compiled from the map.
//
// THAT IS A STATEMENT ABOUT THE MAP FILES AND NOT ABOUT EVERY WORLD (1032
// return 1, D). A world resumed from a save written before byte-form version 57
// carries checks whose reference was zero-filled by the upgrade, because the
// field did not exist in that form. pkg/game restores them from the mission it
// started (`sim.RestoreCheckItemRefs`, `DIV-276`), and a resume whose started
// mission compiles a different script keeps them unbound.
//
// Structure and HasStructure are the check record's SEVENTH reference (1033
// B3, `TRIG-CHECK-053`), on the item reference's own terms: check opcode 21
// is the one arm of the whole vocabulary that names a structure, and the
// field is what pkg/mapload's structure reference bindParams resolves goes
// into. It joins neither Args nor any of the other six references, for the
// packing reason every one of them already gives.
type ScriptCheck struct {
	Op           int32
	Register     int32
	Args         [scriptParams]int32
	Unit         EntityID
	Unit2        EntityID
	HasUnit      bool
	HasUnit2     bool
	Group        uint32
	HasGroup     bool
	Player       uint32
	Player2      uint32
	HasPlayer    bool
	HasPlayer2   bool
	Item         uint16
	HasItem      bool
	Structure    StructureID
	HasStructure bool
}

// ScriptInstant is one compiled instant: an opcode, its ten plain parameters, and
// the four references it may name.
//
// Unit, Group and Player are the same three shapes ScriptCheck carries and they
// are carried on the same terms. Unit is ALREADY RESOLVED — an entity id, through
// the same three identifier bands and the same caller-supplied table a check's
// unit reference goes through, because only the tier that assembled the world
// knows which entity a placed record, a hero ordinal or a name-table entry became.
// Group and Player are CARRIED, not resolved: each names exactly one id space, so
// there is nothing to look either up in and nothing that can fail.
//
// Unit2 and HasUnit2 are a FOURTH reference, on exactly the terms
// ScriptCheck's own Unit/Unit2 pair already stands on: opcode 28 is the one
// arm of the whole vocabulary that names two units, and until this field
// existed the record had nowhere to put the second one — it compiled with
// the giver bound and the receiver dropped. It is ALREADY RESOLVED the same
// way Unit is, through the same bands and the same table, because it is the
// binder's own second unit parameter and not a second kind of reference.
//
// EACH HAS ITS OWN PRESENCE FLAG, and none of the four can do without one: entity
// id zero is a real entity, group zero is a real group, and while roster slot zero
// names nobody the flag is what separates "the node named no player" from "the node
// named slot zero" — the arms answer those two differently.
//
// THEY DO NOT JOIN Args, and that is the point of carrying them separately
// rather than a matter of taste. The binder packs plain parameters in
// ENCOUNTER ORDER, so admitting a unit, group or player into that packing
// would shift the parameters of every other node carrying one — including
// the arms this build does not run, whose compiled records would change
// under it with nothing watching. Beside the plain parameters, they move
// nothing. Item and HasItem are a SIXTH reference. It is CARRIED, like the
// group and the player and unlike the two units, and it is already a PACKED
// ITEM CODE — the same kind of value a container element holds, not the
// map file's own Target_Item value. The map loader does that conversion,
// because how a map file spells a reference is its business and this package
// holds nothing about the file a world was built from.
//
// It is a uint16 because the original's field is a word and an authored item is
// a packed u16, so an arm compares a code against a container's code with no
// mask at the comparison. It has a presence flag for the reason the other five
// do: a code of zero does name nothing, and folding the flag away would make one
// reference slot on this record read differently from every other one.
//
// Structure and HasStructure are the instant record's SEVENTH reference (1033
// B3, `TRIG-CHECK-053`), the check record's own Structure field on the same
// terms: instant opcode 26 is the one action of the whole vocabulary that
// names a structure.
type ScriptInstant struct {
	Op           int32
	Args         [scriptParams]int32
	Unit         EntityID
	Unit2        EntityID
	HasUnit      bool
	HasUnit2     bool
	Group        uint32
	HasGroup     bool
	Player       uint32
	HasPlayer    bool
	Item         uint16
	HasItem      bool
	Structure    StructureID
	HasStructure bool
}

// ScriptPair is one of a trigger's three condition pairs: two REGISTER
// subscripts and the code that compares them.
//
// Used is what makes a pair a pair. A trigger carries three slots and the
// shipped corpus has 1263 of 1263 of them both-set or both-clear, never
// half-empty; an unused slot is not a comparison against register zero, it is no
// comparison at all, and the difference decides whether a trigger with one
// authored condition also has to satisfy two accidental ones.
type ScriptPair struct {
	Left  int32
	Right int32
	Cmp   int32
	Used  bool
}

// The six comparison codes, in the order the engine's own six-entry table puts
// them. A code outside this alphabet is PERMANENTLY FALSE rather than an error:
// one shipped pair carries 0xffffffff, and what the engine does with it is take
// the out-of-range arm every pass.
const (
	ScriptCmpEQ int32 = 0
	ScriptCmpNE int32 = 1
	ScriptCmpGT int32 = 2
	ScriptCmpLT int32 = 3
	ScriptCmpGE int32 = 4
	ScriptCmpLE int32 = 5
)

// ScriptNone is the value an unused instant slot carries. A trigger has four,
// and each is used only when it names a node.
const ScriptNone int32 = -1

// ScriptTrigger is one compiled trigger: three condition pairs ANDed together,
// up to four instants to run when they all hold, and the latch that decides
// whether it may run again.
//
// Once and Latch are the fire-once mechanism. Latch is the trigger's position in
// the MAP'S OWN trigger array, not its position in this compiled slice: the
// engine indexes the latch array by the former, and a map whose triggers were
// not all built would otherwise re-use another trigger's latch.
//
// Inert is DERIVED, not authored: it is set when the trigger reads a register no
// arm of this build writes. An inert trigger is skipped whole — it is not
// evaluated, its latch is not touched, and it can never fire. That is the
// loudness rule stated as behaviour rather than as a report.
type ScriptTrigger struct {
	Pairs    [3]ScriptPair
	Instants [4]int32
	Once     bool
	Latch    int32
	Inert    bool
}

// ScriptGapKind says which array a reported gap is in.
type ScriptGapKind uint8

const (
	// ScriptGapCheck is a check whose opcode this build does not evaluate.
	ScriptGapCheck ScriptGapKind = 0
	// ScriptGapInstant is an instant whose opcode this build does not run.
	ScriptGapInstant ScriptGapKind = 1
)

// ScriptGap is one arm the compiled script asks for and this build does not
// implement: which array it is in, which opcode, and where.
//
// It is the machine-readable half of the loudness rule. The behavioural half is
// ScriptTrigger.Inert and the skipped instant; this is what lets a consumer say
// so before a single tick has run.
//
// Sub is opcode 6's own second dispatch: the sub-command (Args[0]) an
// unsupported group-order node named. It is MEANINGFUL ONLY when Op ==
// ScriptInstantGroupOrder — zero and unread for every other gap, one
// opcode among 34 having a second dispatch being no reason to widen every
// other arm's report. A script authoring an implemented and an unimplemented
// sub-command reports exactly one gap, naming the unimplemented one (AC-10).
type ScriptGap struct {
	Kind  ScriptGapKind
	Op    int32
	Index int32
	Sub   int32
}

// Script is a compiled mission script: the checks, the instants and the triggers
// that bind them, with the arms this build cannot evaluate already worked out.
//
// Every field is unexported and every accessor hands back a copy, so a script
// cannot be edited after it has been validated — which is what lets the world
// that holds one rely on the invariants NewScript established.
type Script struct {
	dialect  ScriptDialect
	checks   []ScriptCheck
	instants []ScriptInstant
	triggers []ScriptTrigger

	gaps  []ScriptGap
	inert []int32
}

// NewScript compiles the three arrays into a script, computing which triggers
// are inert and which arms are unimplemented.
//
// It REFUSES rather than repairs, on every count that would make the arrays
// inconsistent with one another: a register subscript outside the file, two
// checks owning one register, a pair naming a register outside the file, a latch
// outside the latch array, and an instant slot naming no instant. Each is a
// state a binder can only reach by being wrong, and each would otherwise turn
// into a silent misread at evaluation time.
//
// An unimplemented opcode is NOT among the refusals. A map is entitled to author
// the whole vocabulary, and refusing the script would take the arms this build
// does implement down with the ones it does not; what the unimplemented arms get
// instead is the inert marking and the gap report.
func NewScript(checks []ScriptCheck, instants []ScriptInstant, triggers []ScriptTrigger) (*Script, error) {
	return newScript(ScriptROM1, checks, instants, triggers)
}

func newScript(dialect ScriptDialect, checks []ScriptCheck, instants []ScriptInstant, triggers []ScriptTrigger) (*Script, error) {
	// owner[r] is the check that writes register r, or -1. It is what decides
	// inertness, and building it is also how the "two checks, one register"
	// refusal is made.
	var owner [scriptRegisters]int32
	for i := range owner {
		owner[i] = ScriptNone
	}

	cp := append([]ScriptCheck(nil), checks...)
	for i := range cp {
		r := cp[i].Register
		if r < 0 || r >= scriptRegisters {
			return nil, fmt.Errorf("sim: script check %d owns register %d, the file has %d",
				i, r, scriptRegisters)
		}
		if owner[r] != ScriptNone {
			return nil, fmt.Errorf("sim: script checks %d and %d both own register %d",
				owner[r], i, r)
		}
		owner[r] = int32(i)
	}

	ip := append([]ScriptInstant(nil), instants...)

	// poisoned[r] is a register that no arm of this build ever writes BECAUSE
	// the check that owns it is unimplemented. A register owned by NO check is
	// not poisoned: that is an authored mission variable, legitimately zero
	// until an instant writes it, and a trigger reading one is doing exactly
	// what the map asked for.
	var poisoned [scriptRegisters]bool
	var gaps []ScriptGap
	for i := range cp {
		if scriptCheckSupported(cp[i].Op) || dialect == ScriptROM2 && rom2CheckSupported(cp[i].Op) {
			continue
		}
		gaps = append(gaps, ScriptGap{Kind: ScriptGapCheck, Op: cp[i].Op, Index: int32(i)})
		poisoned[cp[i].Register] = true
	}
	for i := range ip {
		if scriptInstantSupported(ip[i]) || dialect == ScriptROM2 && rom2InstantSupported(ip[i].Op) {
			continue
		}
		g := ScriptGap{Kind: ScriptGapInstant, Op: ip[i].Op, Index: int32(i)}
		if ip[i].Op == ScriptInstantGroupOrder {
			g.Sub = ip[i].Args[0]
		}
		gaps = append(gaps, g)
	}

	tp := append([]ScriptTrigger(nil), triggers...)
	var inert []int32
	for i := range tp {
		t := &tp[i]
		t.Inert = false
		if t.Latch < 0 || t.Latch >= scriptLatches {
			return nil, fmt.Errorf("sim: script trigger %d latches at %d, the array holds %d",
				i, t.Latch, scriptLatches)
		}
		for k := range t.Pairs {
			p := t.Pairs[k]
			if !p.Used {
				continue
			}
			if p.Left < 0 || p.Left >= scriptRegisters || p.Right < 0 || p.Right >= scriptRegisters {
				return nil, fmt.Errorf("sim: script trigger %d pair %d reads registers %d and %d, the file has %d",
					i, k, p.Left, p.Right, scriptRegisters)
			}
			if poisoned[p.Left] || poisoned[p.Right] {
				t.Inert = true
			}
		}
		for k, s := range t.Instants {
			if s == ScriptNone {
				continue
			}
			if s < 0 || int(s) >= len(ip) {
				return nil, fmt.Errorf("sim: script trigger %d instant slot %d names instant %d of %d",
					i, k, s, len(ip))
			}
		}
		// Inert is DERIVED here and any value the caller set is overwritten, so
		// "a trigger is inert exactly when it reads a register nothing writes"
		// is a property of this constructor rather than of whoever called it.
		if t.Inert {
			inert = append(inert, int32(i))
		}
	}

	return &Script{dialect: dialect, checks: cp, instants: ip, triggers: tp, gaps: gaps, inert: inert}, nil
}

// WithVIP returns this program with one additional protect-unit check.
//
// The extra objective is compiled into the same Script that owns every shipped
// condition. It therefore runs on the script cadence, increments the ordinary
// loss counter, survives the byte form, and participates in the world digest;
// it is not a presentation-side verdict. The caller supplies an already-bound
// entity id, on the same terms as CompileScript's Target_Unit references.
//
// A check is also required to own a register even though the VIP arm writes no
// value. The first unowned register is used so this addition cannot collide
// with an authored mission variable. A program that already owns all one
// hundred registers is refused rather than silently replacing one.
func (s *Script) WithVIP(unit EntityID) (*Script, error) {
	var checks []ScriptCheck
	var instants []ScriptInstant
	var triggers []ScriptTrigger
	if s != nil {
		checks = append(checks, s.checks...)
		instants = append(instants, s.instants...)
		triggers = append(triggers, s.triggers...)
	}
	var used [scriptRegisters]bool
	for _, c := range checks {
		if c.Register >= 0 && c.Register < scriptRegisters {
			used[c.Register] = true
		}
	}
	register := int32(ScriptNone)
	for i := range used {
		if !used[i] {
			register = int32(i)
			break
		}
	}
	if register == ScriptNone {
		return nil, fmt.Errorf("sim: cannot add a VIP check: all %d registers are owned", scriptRegisters)
	}
	checks = append(checks, ScriptCheck{
		Op: ScriptCheckVIP, Register: register, Unit: unit, HasUnit: true,
	})
	return newScript(s.Dialect(), checks, instants, triggers)
}

// Empty reports whether the script holds nothing at all. A world given an empty
// script and a world given none are the same world, which is why this predicate
// exists in one place rather than as three length tests.
func (s *Script) Empty() bool {
	return s == nil || (len(s.checks) == 0 && len(s.instants) == 0 && len(s.triggers) == 0)
}

// Checks, Instants and Triggers hand back copies of the compiled arrays.
func (s *Script) Checks() []ScriptCheck {
	if s == nil {
		return nil
	}
	return append([]ScriptCheck(nil), s.checks...)
}

func (s *Script) Instants() []ScriptInstant {
	if s == nil {
		return nil
	}
	return append([]ScriptInstant(nil), s.instants...)
}

func (s *Script) Triggers() []ScriptTrigger {
	if s == nil {
		return nil
	}
	return append([]ScriptTrigger(nil), s.triggers...)
}

// Unsupported returns every arm this script asks for that this build does not
// implement, in array order: the checks first, then the instants.
//
// IT IS THE ANSWER TO "not implemented or evaluated false". An empty result says
// every arm the map authored is one this build evaluates; a non-empty one names
// each arm exactly, before anything has been advanced a single tick.
func (s *Script) Unsupported() []ScriptGap {
	if s == nil {
		return nil
	}
	return append([]ScriptGap(nil), s.gaps...)
}

// InertTriggers returns the indices of the triggers that can never fire because
// they read a register no arm of this build writes.
//
// It is the second half of the same answer and it is the half a consumer cares
// about: an unimplemented CHECK is only interesting because of the triggers it
// takes down with it, and this says which.
func (s *Script) InertTriggers() []int32 {
	if s == nil {
		return nil
	}
	return append([]int32(nil), s.inert...)
}

// Outcome is what the mission has come to, and its three values are the
// original's own: in progress, complete, failed.
type Outcome uint8

const (
	// OutcomeUndecided is the mission still running, and it is the zero value.
	OutcomeUndecided Outcome = 0
	// OutcomeWon is the mission complete.
	OutcomeWon Outcome = 1
	// OutcomeLost is the mission failed.
	OutcomeLost Outcome = 2
)

func (o Outcome) defined() bool {
	return o == OutcomeUndecided || o == OutcomeWon || o == OutcomeLost
}

// The two phases of the sixteen-sub-tick cycle this file runs on.
//
// A tick here is the original's SUB-TICK — the unit its own clock paces against
// a millisecond period — and sixteen of them are one FULL tick. The script pass
// runs on phase 6 of that cycle and the outcome report on phase 15, and both
// numbers are the engine's own: the pass is dispatched from the arm its
// scheduler selects with `sub % 16 == 6`, and the reporter from the arm selected
// with `== 15`. So the whole authored script is evaluated once every sixteen
// ticks and the outcome is read nine ticks later, in that same cycle.
//
// scriptCycle is spelled once and both phases are taken modulo it, so the two
// cannot come to disagree about how long a full tick is.
const (
	scriptCycle       = 16
	scriptPassPhase   = 6
	scriptReportPhase = 15
)

// scriptPass advances the session script counter, then evaluates every check
// and every trigger.
//
// THE ORDER IS THE CONTRACT and it is not merely convenient. Every check writes
// its register BEFORE any trigger reads one, so a trigger never compares a
// register this tick's pass has written against one it has not; and the whole
// pass runs before any entity is advanced, so a condition measures the world as
// the tick found it and not as some of the tick left it.
//
// EVERY CHECK IS EVALUATED, including one no trigger names. That is not waste:
// a check may have a side effect and no value at all — the "protect this unit"
// objective is authored as a check nothing reads, whose entire purpose is that
// it counts a loss when its unit dies.
//
// tr is the OBSERVER and is nil on an ordinary step. It is threaded rather than
// stored so that an untraced pass is the traced pass with every recorder
// returning on its first line — see scripttrace.go.
func (w *World) scriptPass(tr *ScriptTrace) { w.scriptPassObserved(tr, nil) }

func (w *World) scriptPassObserved(tr *ScriptTrace, obs *castObs) {
	s := w.script
	// The native byte form represents absent and empty programs identically.
	// Original sessions own this clock even when no checks are compiled; a
	// legacy native world without a program keeps its inert register bank.
	hasProgram := s != nil && (len(s.checks) != 0 || len(s.instants) != 0 || len(s.triggers) != 0)
	if !w.hasSessionClock && !hasProgram {
		return
	}
	// SAV-650: slot93 increments before both evaluators. Authored time
	// conditions read this saved int32 counter, including its wrapping value.
	w.registers[93]++
	if s == nil {
		return
	}
	for i := range s.checks {
		w.runCheck(s.checks[i], int32(i), tr)
	}
	for i := range s.triggers {
		t := s.triggers[i]
		latchBefore := w.latches[t.Latch]
		run := tr.triggerStart(w, int32(i), t)
		// An inert trigger is skipped BEFORE the latch is touched. Its recorded
		// decision names that skip; canonical latch state stays exactly as it was.
		if t.Inert {
			tr.triggerDone(w, run, ScriptTriggerInert)
			continue
		}
		if w.savedTriggerBlocked(t) {
			tr.triggerDone(w, run, ScriptTriggerInert)
			continue
		}
		// The fire-once gate: a latched one-shot is skipped whole. A trigger
		// that is not one-shot never takes this arm, which is what makes it
		// re-fire on every pass its conditions hold.
		if t.Once && w.latches[t.Latch] != 0 {
			tr.triggerDone(w, run, ScriptTriggerSpent)
			continue
		}
		w.latches[t.Latch] = 0
		if !w.triggerHolds(t, tr, run) {
			tr.triggerDone(w, run, ScriptTriggerFailed)
			continue
		}
		w.latches[t.Latch] = 1
		decision := ScriptTriggerFired
		if latchBefore != 0 {
			decision = ScriptTriggerHeld
		}
		tr.triggerDone(w, run, decision)
		// The observation stands HERE, between the hold and the first instant,
		// because an instant may write a register a later trigger reads: the
		// values a pair was compared on exist only at this point in the pass.
		firing := tr.firing(w, int32(i), t)
		for slot, k := range t.Instants {
			if k == ScriptNone {
				continue
			}
			in := s.instants[k]
			before := tr.instantStart(w)
			w.runInstantObserved(in, obs)
			tr.instantDone(w, firing, int32(slot), k, in, before)
		}
	}
}

// triggerHolds is the trigger's condition: its used pairs compared and ANDed,
// with SHORT-CIRCUIT — the first failing pair ends the test.
//
// THE AND OF NO PAIR IS TRUE. A trigger carrying no used pair fires on the first
// pass, which is the editor's own "always" idiom; the accumulator starting at
// true is the whole of that rule and there is no separate arm for it.
//
// THERE IS NO OR ANYWHERE. Three pairs is the whole of a trigger's logic.
func (w *World) triggerHolds(t ScriptTrigger, tr *ScriptTrace, run int) bool {
	for i, p := range t.Pairs {
		if !p.Used {
			continue
		}
		left, right := w.registers[p.Left], w.registers[p.Right]
		holds := scriptCompare(p.Cmp, left, right)
		tr.triggerPair(run, i, p, left, right, holds)
		if !holds {
			return false
		}
	}
	return true
}

// scriptCompare applies one comparison code to two register values.
//
// A code outside the alphabet is FALSE, and that is the engine's own bound test
// rather than a defensive default: the dispatch is bounded at 5 and everything
// above it takes the failing arm. The corpus contains such a pair, so this is a
// path a shipped map reaches and not a hypothetical.
func scriptCompare(code, a, b int32) bool {
	switch code {
	case ScriptCmpEQ:
		return a == b
	case ScriptCmpNE:
		return a != b
	case ScriptCmpGT:
		return a > b
	case ScriptCmpLT:
		return a < b
	case ScriptCmpGE:
		return a >= b
	case ScriptCmpLE:
		return a <= b
	}
	return false
}

// runCheck evaluates one check and writes its register — or writes nothing,
// which is a real outcome and not a failure.
//
// SEVERAL ARMS WRITE NO REGISTER, for distinct reasons the complete trace names. The
// constant form was preset when the world was built and must not be written
// again. The VIP arm has no value at all: its whole effect is the loss it
// counts. The group count, the population count and the nearest-unit distance
// write none when their own reference — a group or a player — is absent,
// because there is nothing to measure; that is not the same as the zero or the
// 0xff each answers when the reference resolves to an owner with nothing
// living. A DEAD ARM writes none because the arm it reproduces writes none —
// and unlike the four around it, a dead arm leaves its readers LIVE, which is
// the whole difference between reproducing one and failing to implement it. And
// an arm this build does not implement writes nothing so that the trigger
// reading it — already marked inert — cannot be told a measurement was taken.
//
// i and tr are the OBSERVER's two arguments and neither is read by an arm: i is
// the check's own subscript, carried so that a recorded silence names the check
// rather than describing it, and tr is nil on an ordinary step.
func (w *World) runCheck(c ScriptCheck, i int32, tr *ScriptTrace) {
	var before int32
	var lostBefore uint32
	if tr != nil {
		before, lostBefore = w.registerAt(c.Register), w.lost
	}
	dispatched, wrote := true, false
	silence := ScriptSilenceNone
	write := func(v int32) {
		w.setRegister(c.Register, v)
		wrote = true
	}
	defer func() {
		tr.check(i, c, before, lostBefore, dispatched, wrote, silence, w)
	}()
	if w.rom2 != nil && rom2CheckSupported(c.Op) {
		v, ok := w.rom2Check(c)
		if ok {
			write(v)
		} else {
			silence = ScriptSilenceNoUnit
		}
		return
	}
	switch c.Op {
	case ScriptCheckConstant:
		// Preset at build time, never dispatched. Writing it here would
		// overwrite every instant that has since set the variable it is.
		dispatched, silence = false, ScriptSilenceBuildTimeConstant
		return

	case ScriptCheckVIP:
		// Writes NO register. A trigger comparing two of these compares two
		// registers nothing ever writes and is permanently false — which is what
		// makes a "protect this unit" objective a trigger with no action.
		e, ok := w.scriptEntity(c.Unit, c.HasUnit)
		if !ok {
			silence = ScriptSilenceNoUnit
			tr.silence(i, c, ScriptSilenceNoUnit)
			return
		}
		if scriptVIPDead(e) {
			w.lost++
			tr.vipLoss(i, c.Unit)
		}
		silence = ScriptSilenceVIPNoValue
		return

	case ScriptCheckGroupCount:
		// Group count reads the current SAV Group member list when present.
		// Map-only worlds use the placed Group word on each entity.
		//
		// A check naming NO group measures nothing and writes no register, which
		// is the treatment an unresolved unit reference gets and for the same
		// reason: there is nothing to count. It is NOT the same as a count of
		// zero, which is an answer — group zero is a real group, and a group id
		// no entity carries answers zero rather than failing.
		//
		// A fallen actor remains a member until teardown, even when its dwell
		// counter is zero. Saved Groups count their current member list. The
		// map-only path has no mutable member list, so it retains Stage 1
		// and excludes torn-down bodies (TRIG-REAP-017).
		//
		// The scan reads Group — the PLACED group the map gave each entity —
		// and not effectiveGroup. That choice did not exist before 0117: until
		// then nothing in this package moved an actor between groups at all, Group
		// was the only group word an entity carried, and this note said so. 0117
		// gives an actor a SECOND group word, a player's own — CommandGroup,
		// written by a move order — and Group is the one of the two "never
		// moves" is still true of. Reading Group here rather than effectiveGroup
		// is what keeps a player's click out of this count: a trigger comparing it
		// against zero is how a campaign says a place is cleared, and a commanded
		// actor is still a living member of the group the map placed it in for
		// exactly that purpose — whichever group it is being DECIDED under this
		// tick (engage.go's own effectiveGroup, which the engagement layer alone
		// reads).
		//
		// The scan is linear over the world's own ordered entities, and it
		// stays one: there is no membership index and none is wanted, because
		// Group itself never moves — an index over it would be a second
		// representation of a fact that never changes, and it would be an
		// iteration order reaching a value that enters the digest.
		if !c.HasGroup {
			silence = ScriptSilenceNoGroup
			tr.silence(i, c, ScriptSilenceNoGroup)
			return
		}
		n := int32(0)
		owner, _ := w.scriptGroupOwner(c.Group)
		for i := range w.entities {
			if e := w.entities[i]; e.Group == c.Group && e.Owner == owner && (!scriptDead(e) || e.Decay == DecayFallen) {
				n++
			}
		}
		if w.savedGroups != nil {
			groups, err := w.resolveSavedGroups(c.Group)
			if err != nil {
				silence = ScriptSilenceUnsupported
				tr.silence(i, c, silence)
				return
			}
			n = 0
			for _, g := range groups {
				n += int32(len(g.Members))
			}
		}
		write(n)
		return

	case ScriptCheckVariable:
		// The variable read, and the reason a mission variable and a check
		// result share one array: the subscript is the AUTHORED number, not a
		// compiled one, so a map may name a register a check overwrites every
		// pass. That collision is the map's to make and is reproduced.
		write(w.registerAt(c.Args[0]))
		return

	case ScriptCheckSackAt:
		// THE SACK QUERY. It belongs HERE, beside ScriptCheckVariable, and not in
		// the second switch below: a check-14 node carries two plain X/Y
		// parameters and NAMES NO UNIT, so the second switch — reached only
		// after w.scriptEntity resolves one — is unreachable for it. An arm
		// placed there would be correct code that never runs, leaving every node
		// at the unresolved-reference silence instead of a measurement.
		//
		// The named cell is the LOW BYTE of each of the first two plain parameters
		// — the check's own arithmetic, not a storage width (D-7) — so no
		// shipped node needs the truncation for it to be correct. A cell outside
		// the world's bounds answers 0 rather than stopping the pass: sackAt
		// itself can never find a sack there, since the constructor refuses one
		// outside the bounds, so no separate bounds test is needed here.
		//
		// NOTHING OF THE SACK REACHES THE REGISTER — not its gold, not its
		// contents, not how many records were merged into it: the only question
		// this arm can answer is whether one is there.
		v := int32(0)
		if w.sackAt(c.Args[0]&0xff, c.Args[1]&0xff) {
			v = 1
		}
		write(v)
		return

	case ScriptCheckPopulation:
		if !c.HasPlayer {
			silence = ScriptSilenceNoPlayer
			tr.silence(i, c, ScriptSilenceNoPlayer)
			return
		}
		n := int32(0)
		for i := range w.entities {
			if w.entities[i].Owner == c.Player && !scriptDead(w.entities[i]) {
				n++
			}
		}
		write(n)
		return

	case ScriptCheckNearest:
		if !c.HasPlayer {
			silence = ScriptSilenceNoPlayer
			tr.silence(i, c, ScriptSilenceNoPlayer)
			return
		}
		best := int32(0xff)
		for i := range w.entities {
			e := w.entities[i]
			if e.Owner != c.Player || scriptDead(e) {
				continue
			}
			if d := scriptByte(scriptDistance(e.X, e.Y, c.Args[0], c.Args[1])); d < best {
				best = d
			}
		}
		write(best)
		return

	case ScriptCheckRelation:
		// WHAT ONE PLAYER THINKS OF ANOTHER. It belongs HERE, beside
		// ScriptCheckPopulation and ScriptCheckNearest, and not in the second
		// switch: a check-10 node NAMES NO UNIT, so the second switch — reached
		// only after w.scriptEntity resolves one — is unreachable for it.
		//
		// It needs BOTH player references, unlike its two siblings above, which
		// need one: a check naming FEWER THAN TWO measures nothing and writes no
		// register, the same silence and the same reason — there is nothing to
		// measure. relations.Byte already answers zero for a slot the matrix does
		// not hold, so no separate out-of-range test is needed here.
		//
		// THE MASK TO TWO BITS IS THE POINT.
		if !c.HasPlayer || !c.HasPlayer2 {
			silence = ScriptSilenceNoPlayer
			tr.silence(i, c, ScriptSilenceNoPlayer)
			return
		}
		write(int32(w.relations.Byte(c.Player, c.Player2) & 3))
		return

	case ScriptCheckStructField:
		// THE STRUCTURE FIELD GETTER (1033 B3, `TRIG-CHECK-053`). It belongs
		// HERE, beside ScriptCheckSackAt and the group/population/nearest/
		// relation arms above, and not in the second switch below: a check-21
		// node names NO UNIT — its one reference is a structure — so the
		// second switch, reached only after w.scriptEntity resolves a unit,
		// is unreachable for it.
		//
		// UNCONDITIONAL AND SIGN-EXTENDED, on the arm's own instruction: no
		// p0/p1 test, and a MOVSX word load. Structure.Field42 is stored as a
		// raw uint16 and sign-extended here into the register, the same
		// treatment check 4's health field gets for free from Entity.HP
		// already being signed.
		//
		// A STRUCTURE REFERENCE THAT DID NOT RESOLVE MEASURES NOTHING, the
		// unresolved-unit treatment applied to this record's own reference kind.
		// `TRIG-BIND-010` states no shipped node reaches this branch — every
		// authored Target_Structure reference resolves against the map's own
		// structure table (`ALM-TRIG-046`) — so it is a safety net rather than a
		// reachable path, on the check-9 index guard's own ground: the alternative
		// is indexing without a bounds check, and a panic in this package on
		// shipped content is the one outcome that must not ship.
		st, ok := w.scriptStructure(c.Structure, c.HasStructure)
		if !ok {
			silence = ScriptSilenceNoStructure
			tr.silence(i, c, ScriptSilenceNoStructure)
			return
		}
		write(int32(int16(st.Field42)))
		return

	case ScriptCheckDead11, ScriptCheckDead13:
		silence = ScriptSilenceDeadArm
		return
	}

	// An arm this build does not evaluate leaves HERE rather than falling through
	// the switch below and reaching no case, which is where it used to stop. The
	// two are the same nothing — scriptEntity only looks an id up — and the early
	// return is what lets the silence be recorded as unsupported rather than as
	// an unresolved reference it never got as far as testing.
	if !scriptCheckSupported(c.Op) {
		dispatched, silence = false, ScriptSilenceUnsupported
		tr.silence(i, c, ScriptSilenceUnsupported)
		return
	}

	e, ok := w.scriptEntity(c.Unit, c.HasUnit)
	if !ok {
		// The reference did not resolve, so there is nothing to measure. No
		// register is written, exactly as for an unimplemented arm.
		silence = ScriptSilenceNoUnit
		tr.silence(i, c, ScriptSilenceNoUnit)
		return
	}

	switch c.Op {
	case ScriptCheckInBox:
		v := int32(0)
		if e.X >= c.Args[0] && e.X <= c.Args[2] && e.Y >= c.Args[1] && e.Y <= c.Args[3] {
			v = 1
		}
		write(v)

	case ScriptCheckWithin:
		v := int32(0)
		if scriptDistance(e.X, e.Y, c.Args[0], c.Args[1]) <= int64(c.Args[2]) {
			v = 1
		}
		write(v)

	case ScriptCheckAlive:
		v := int32(1)
		if scriptDead(e) {
			v = 0
		}
		write(v)

	case ScriptCheckDistance:
		write(scriptByte(scriptDistance(e.X, e.Y, c.Args[0], c.Args[1])))

	case ScriptCheckUnitDistance:
		other, ok2 := w.scriptEntity(c.Unit2, c.HasUnit2)
		// A dead unit at either end answers the saturated byte rather than a
		// distance, so a dead unit reads as unreachably far and not as adjacent.
		if !ok2 || scriptDead(e) || scriptDead(other) {
			write(0xff)
			return
		}
		write(scriptByte(scriptDistance(e.X, e.Y, other.X, other.Y)))

	case ScriptCheckHealth:
		// THE HEALTH GATE (1033 B1, `TRIG-CHECK-051`). The register takes the
		// unit's own current health, sign-extended, but ONLY when the node's
		// own first plain parameter equals scriptCheckHealthGate; off the
		// gate the arm writes NOTHING — not zero, a state a reader downstream
		// cannot tell apart from "not yet measured" any other way. Entity.HP
		// is already a signed int32 carrying the same value the original's
		// unit+0x94 word does (`HERO-HEALTH-032`), so no further sign
		// extension is needed here the way check 21's raw uint16 field needs
		// one.
		//
		// Every shipped check-4 node's own plain parameter is the gate value
		// (`TRIG-CHECK-054`), so the off-gate arm is not reachable from
		// shipped content; it is kept because the arm's own gate is a
		// property of the vocabulary and not of the corpus.
		if c.Args[0] == scriptCheckHealthGate {
			write(e.HP)
		} else {
			silence = ScriptSilenceHealthGate
		}

	case ScriptCheckItemDistance:
		// THE ITEM-GATED DISTANCE (1033 B2, `TRIG-CHECK-052`). THE NODE'S OWN X/Y
		// ARE READ 8 BITS WIDE, and the unit's position is not. `TRIG-CHECK-052`
		// names the two one-byte loads (node offsets 0x8 and 0xc)
		// against the unit's position taken whole from its own
		// position object (`[unit+0x10]`), and `TRIG-CHECK-054` classes that width
		// as an ENGINE READ WIDTH rather than a stored field: lifting it changes
		// no shipped file. So the mask goes on the PARAMETERS, the same place
		// check 14's own arm puts it, and the result is masked as well because the
		// arm masks before its store.
		//
		// NO SHIPPED NODE REACHES THE TRUNCATION: `TRIG-CHECK-054` gives the
		// widest authored X/Y in the whole corpus as 116/130, both inside 8
		// bits, on both roots. The mask is the read width reproduced, not a
		// behaviour any shipped map exercises.
		//
		// This does NOT settle scriptDistance's own open byte-width clause,
		// which is about whether the helper masks the coordinates it
		// subtracts. That question is unchanged and still Medium; what is
		// settled here is how wide THIS ARM loads its own two parameters.
		v := int32(0xff)
		if c.HasItem {
			if i := indexOfEntity(w.entities, c.Unit); i >= 0 && containerHolds(w.carried[i], c.Item) {
				v = scriptByte(scriptDistance(e.X, e.Y, c.Args[0]&0xff, c.Args[1]&0xff))
			}
		}
		write(v)

	case ScriptCheckItemTestAlias, ScriptCheckItemTest:
		// THE ITEM TEST (`TRIG-ITEMTEST-040`). Opcodes 12 and 17 run through one
		// shared case: the claim establishes that their two original bodies are
		// byte-identical apart from a call displacement resolving to the same
		// finder.
		//
		// The arm reads
		// the item code out of the compiled record and the container off the
		// named unit, and writes 1 when the container holds that code and 0
		// when it does not.
		//
		// IT WRITES NO SIMULATION STATE BEYOND ITS OWN REGISTER, which is the
		// row's own word. The routine the check calls is the finder instants 11
		// and 13 call first; those two act on what it returns and this one only
		// answers whether it returned anything.
		//
		// THE CONTAINER ONLY, NOT THE TWELVE WORN PLACES. The finder is called
		// with `ECX = unit+0x7c`, which is the carried container, and equipment is
		// different state at a different offset. A hero WEARING the named item and
		// carrying none of it answers 0.
		//
		// THE COMPARISON IS CODE AGAINST CODE WITH NO MASK. Both sides are packed
		// uint16 codes: pkg/mapload converts the map file's Target_Item to one at
		// bind time, and a container element already holds one.
		//
		// A NODE NAMING NO ITEM ANSWERS 0 RATHER THAN WRITING NOTHING, and the
		// PRESENCE FLAG is what decides it, not the code being zero. HasItem is
		// false when the condition authored no Target_Item parameter, and the arm
		// answers 0 without reading any container. The alternative -- searching
		// for code 0 and relying on no container ever holding one -- makes the
		// answer depend on a property of the carry store rather than on the
		// record, and it leaves the flag parsed and unread. scriptItemHolder,
		// which the item instants share, tests its own flag the same way. Every
		// check-17 node the shipped maps author binds an item -- 23 of them -- so
		// this case is not reachable from a script compiled off a map file. It IS
		// reachable from a save: a form below version 57 carries no item reference
		// at all, and one this build could not rebind from the mission it opened
		// resumes with the flag false (`DIV-276`).
		//
		// A DEAD OR DYING UNIT IS NOT A SPECIAL CASE. The arm has no health
		// test; a fallen actor still holds its container until the death pass
		// pours it out, and that pass is what changes the answer.
		v := int32(0)
		if c.HasItem {
			if i := indexOfEntity(w.entities, c.Unit); i >= 0 && containerHolds(w.carried[i], c.Item) {
				v = 1
			}
		}
		write(v)

	case ScriptCheckTargetID:
		// THE PURSUIT TARGET'S AUTHORED MAP ID (`TRIG-TARGETID-032`). The register
		// takes the map id of the unit the named subject is currently pursuing,
		// and 0 when it is pursuing nothing.
		//
		// THE PURSUIT THIS BUILD HAS IS AttackTarget. The arm being reconstructed
		// requires the subject's order object to be order 5, pursue-and-attack,
		// before it reads the order's target; a subject under any other order does
		// not have its target inspected and the register takes 0. HasAttackTarget
		// is this build's own statement of the same condition: combat.go clears
		// the pair together, so an actor holding one is an actor pursuing. Two
		// attack orders are not order 5 and answer 0: the acquisition turn, which
		// is order 6 (AI-PURSUE-040), and the idle order a refused route leaves
		// with the victim field untouched (AI-328).
		//
		// THE VALUE IS THE TARGET'S OWN, NOT THE SUBJECT'S. The original reads
		// `target+0x08`; this reads the target entity's MapUnitID, which is the
		// same field one tier up (`ALM-UNIT-018`).
		//
		// A STALE OR ABSENT TARGET ANSWERS 0 (DIV-240). The original has no null
		// branch there and faults. This build resolves the target through the
		// world by id, on the hand-over instants' own ground, and a target the
		// world no longer holds is not a state this package may crash on.
		//
		// THAT INDEX GUARD IS UNREACHABLE AND DELIBERATE. This package already
		// maintains the invariant it protects: the constructor normalises a
		// pursuit naming an entity the world does not hold (world.go's second
		// pass over the copied slice), and the removal sweep does the same
		// after a unit leaves (step.go's keep pass). No fixture can install a
		// dangling pursuit and no mutation of this line changes a test result,
		// which is recorded in 1029's closure rather than hidden: the
		// alternative to the guard is indexing with -1, and a panic in pkg/sim
		// on shipped content is the one outcome that must not ship.
		//
		// A TARGET WITH NO AUTHORED MAP ID ALSO ANSWERS 0 (DIV-241), which is
		// MapUnitID's own zero and not a second rule here.
		v := int32(0)
		if e.HasAttackTarget && e.AttackTargetKind == AttackTargetUnit && !e.AcquirePursuit && !e.PursuitIdle {
			if ti := indexOfEntity(w.entities, e.AttackTarget); ti >= 0 {
				v = int32(w.entities[ti].MapUnitID)
			}
		}
		write(v)
	}
	// An arm this build does not implement reaches no case and writes nothing.
	// It is already in Script.Unsupported and its readers are already inert;
	// there is deliberately no default that writes a value.
}

// runInstant runs one instant.
//
// An arm this build does not run reaches no case and does nothing. It does NOT
// stop the trigger: a trigger that fires runs the instants this build has and
// skips the ones it does not, so the half of a mission chain that can be
// reproduced is not held hostage by the half that cannot. Which arms were
// skipped is Script.Unsupported's answer, given before the first tick.
//
// THE TWO HAND-OVER ARMS ARE THE ONLY WRITERS OF AN ENTITY'S OWNER anywhere in
// this package, and neither reads one.
// handOver is what BOTH hand-over arms do to one actor, and the only writer of
// an entity's owner in this package.
//
// THREE WRITES, IN THIS ORDER, and the order is the original routine's own: the
// actor leaves the group it stood in, it is moved to the new owner, and it is
// placed alone in a group of its own. The routine the original dispatches both
// instants to removes the actor from its current group first, then rewrites its
// owner, then allocates a fresh group and inserts the actor into it. It writes
// no primary-character pointer and allocates no fresh runtime id, which is what
// separates it from a character being installed at a mission's start — so a
// handed-over actor is a member of the new owner's roster and never that
// roster's primary character, and it keeps the runtime id it already had.
//
// THE FRESH GROUP COMES FROM freeCommandGroup (0159 D-4), which already answers
// "the lowest id at or above every script-named group that no actor's EFFECTIVE
// group names". That is exactly a group nobody else stands in. The command
// group is cleared BEFORE the id is taken, on commandGroup's own ground: an id
// only this actor held reads free again the moment the scan runs.
//
// THE COMMAND GROUP IS CLEARED RATHER THAN CARRIED (0159 D-6). It is a second
// group word layered over the placed one, and leaving it standing would keep
// the actor inside the group it was just removed from for every reader that
// asks the effective group — including the next call to this function, which
// would then hand the next member the same id.
//
// The three writes are all it does. Nothing else moves: not the position, not
// the health, not the order's own destination, not the container and not the
// twelve worn places.
func (w *World) handOver(i int, player uint32) {
	if w.savedGroups != nil {
		w.handOverSaved(i, player)
		return
	}
	w.entities[i].CommandGroup = 0
	w.entities[i].Owner = player
	w.inheritAutoHealing(i)
	id := w.freeCommandGroup()
	w.entities[i].Group = id
	if id == 0 {
		return
	}
	// The group the hand-over builds is a new group at group order 0, the
	// state every freshly constructed group holds (AI-CMD-033, PARTY-JOIN-025).
	// Without a record the pair takes no decision at all, so the member stood
	// idle beside an enemy; with one, order 0 hands the member to its own actor
	// state, which for the constructor default is guard.
	members := []int{i}
	cx, cy := groupCentroid(w.entities, members)
	w.upsertGroup(groupAI{
		owner: player,
		group: id,
		base:  noticeBase(w.entities, members, cx, cy),
		order: orderNone,
	})
}

func (w *World) runInstant(in ScriptInstant) {
	w.runInstantObserved(in, nil)
}

func (w *World) runInstantObserved(in ScriptInstant, obs *castObs) {
	if w.rom2 != nil {
		if in.Op == ScriptInstantMessage {
			obs.recordScriptMessage(in.Args[0])
			return
		}
		if w.rom2Instant(in, obs) {
			return
		}
	}
	switch in.Op {
	case ScriptInstantMessage:

	case ScriptInstantSetVariable:
		w.setRegisterAt(in.Args[0], in.Args[1])
	case ScriptInstantIncVariable:
		w.setRegisterAt(in.Args[0], w.registerAt(in.Args[0])+1)
	case ScriptInstantWin:
		w.won++
	case ScriptInstantLose:
		w.lost++

	case ScriptInstantRelation:
		// THE DIPLOMACY WRITE. Converted to the roster-slot type the relation
		// uses; the third value, in.Args[2], is the addend and stays at its own
		// width, exactly as changeRelation wants it.
		w.relations.changeRelation(uint32(in.Args[0]), uint32(in.Args[1]), in.Args[2])

	case ScriptInstantGiveGroup:
		// The group hand-over. A LINEAR SCAN over the world's own ordered
		// entities, and no membership index is built: an index would be a second
		// representation of a fact this arm is the only thing in the tree that
		// changes, and it would have to be maintained by the arm that
		// invalidates it. Scanning the world's own slice is also what makes
		// "which entities this writes does not depend on storage order" a
		// property of the data structure rather than of this arm.
		//
		// A group identifier NO ENTITY CARRIES is not a failure — the loop
		// writes nothing, exactly as the count of such a group is zero rather
		// than an error.
		//
		// THE DEAD ARE INCLUDED, and this is where the group count's opposite
		// choice is worth pointing at: that arm excludes them because it answers
		// "is the camp cleared", and this one includes them because membership
		// and identity outlive death everywhere in this runtime. An arm that
		// skipped the fallen would make a hand-over depend on when it fired.
		//
		// THE MEMBERSHIP IS READ ONCE, BEFORE THE FIRST WRITE (0159 D-5).
		// handOver writes the very field this loop selects on, so a loop that
		// selected and wrote in one pass would hand over the first member,
		// move it out of the group, and then fail to find the rest — a group
		// of three would become a group of one. Collecting the indices first
		// is what makes the arm's answer the membership as the node found it.
		if !in.HasPlayer || !in.HasGroup {
			// A node that named no player, or no group, changes nothing. This is
			// NOT assigning zero: an entity's owner is left exactly as it stood,
			// which is the same separation between "named nothing" and "named a
			// zero" every other reference site in this file makes.
			return
		}
		members := w.groupMembers(in.Group)
		// EACH MEMBER ARRIVES IN A GROUP OF ITS OWN, not all of them together
		// in one: the original arms its per-member routine once per member and
		// that routine allocates a fresh group each time, so a group of three
		// handed over becomes three groups of one.
		for _, i := range members {
			w.handOver(i, in.Player)
		}

	case ScriptInstantGiveUnit:
		// The unit hand-over, resolved THROUGH THE WORLD rather than through the
		// compiled reference alone. That reference is an entity id fixed at
		// compile time and the world it runs against may no longer hold that
		// entity, so the id is looked up and a miss does nothing — the same
		// treatment every check's unit reference already gets, and the same
		// answer an absent reference gets.
		if !in.HasPlayer || !in.HasUnit {
			return
		}
		if i := indexOfEntity(w.entities, in.Unit); i >= 0 {
			w.handOver(i, in.Player)
		}

	case ScriptInstantGiveMoney:
		if in.HasPlayer && in.Player < relationSlots {
			w.purses[in.Player] += uint32(in.Args[0])
		}
	case ScriptInstantGiveAll:
		// THE POUR: the WHOLE of Unit's container becomes the tail of Unit2's —
		// appended, IN UNIT'S OWN ORDER, after whatever Unit2 already held — and
		// Unit's container becomes NIL rather than truncated, the same state every
		// entity that has never held anything is already in. There is no capacity
		// to reach, nothing that can refuse an item and no per-item failure, so
		// the original's own take-index-0-and-append loop reduces to one append
		// and one clear: one act with no partial outcome.
		//
		// AND THE RECEIVER'S CONTAINER IS FOLDED: the giver's ELEMENTS append
		// whole — a count moves as one value, which is D-1's point about the two
		// halves never travelling apart — and the merge then joins each of them
		// to the element the receiver already holds that code in, at THAT
		// element's place. Two actors each holding two of one code leave the
		// receiver with one element at count 4 (AC-6). The receiver's own order is
		// otherwise untouched, so AC-4's own witness of two pours in sequence
		// reads exactly as it did.
		//
		// NEITHER EQUIPMENT RECORD IS TOUCHED: the container and the twelve worn
		// places are different state, and a giver wearing armour is still wearing
		// it after the pour. Nothing else moves either — no gold, no position,
		// no owner, no group, no state byte, no health, no order and not the tick.
		//
		// FOUR REFUSALS, each leaving the world exactly as it was found: no first
		// reference, no second reference, a reference naming an entity this world
		// does not hold, and the two references resolving to ONE entity. The
		// self-transfer is the one refusal with no counterpart on the two
		// hand-over arms above: the original's own loop removes index 0 and
		// appends it back, so a self-transfer never terminates. That is undefined
		// behaviour rather than behaviour, and it is refused here on registerAt's
		// own precedent — the one other place this file already parts company
		// with what it reconstructs, and only where the original's own behaviour
		// is undefined.
		//
		// BOTH REFERENCES ARE RESOLVED THROUGH THE WORLD BY ID, not trusted from
		// the compiled record, on the unit hand-over's own ground: an id fixed at
		// compile time may no longer be in the world the trigger fires against,
		// and a miss does nothing. Finding both by id is also what makes which
		// entities this writes independent of the order the world happens to hold
		// them in.
		if !in.HasUnit || !in.HasUnit2 {
			return
		}
		gi := indexOfEntity(w.entities, in.Unit)
		ri := indexOfEntity(w.entities, in.Unit2)
		if gi < 0 || ri < 0 || gi == ri || !w.hasActorContainer(ri) {
			return
		}
		giverLoad, receiverLoad := w.beginLoadMutation(gi), w.beginLoadMutation(ri)
		if !w.sourceMutationReady(gi) || !w.sourceMutationReady(ri) {
			return
		}
		live := w
		if w.savedObjects != nil || w.entities[gi].ActorLoad.Source.Class != 0 || w.entities[ri].ActorLoad.Source.Class != 0 {
			n := w.sourceMutationCopy(gi)
			n.carried[ri] = cloneStacks(w.carried[ri])
			w = &n
		}
		for len(w.carried[gi]) > 0 {
			whole := w.carried[gi][0].ObjectID == 0
			item, ok := w.takeCarriedObject(gi, 0, whole)
			if !ok || !w.addCarried(ri, item) {
				return
			}
		}
		if !w.finishLoadMutation(ri, receiverLoad) || !w.finishLoadMutation(gi, giverLoad) || !w.savedMutationValid() {
			return
		}
		if w != live {
			*live = *w
		}

	case ScriptInstantAddItem:
		// THE CREATION: one unit of the node's own code appears in the named
		// unit's container. Nothing is taken from anywhere — this arm reads no
		// source at all, which is the whole difference between it and the
		// transfer.
		//
		// IT MERGES RATHER THAN APPENDS, through the same appendUnits +
		// foldContainer pair every other act that puts an item in a container
		// already takes: an authored loadout, a decode, a sack pickup and the
		// give-all pour. So a hero already holding the code holds ONE element at a
		// count one higher (AC-2), and a hero holding it for the first time gains
		// one element at the tail. A private merging add on this arm would be the
		// one path that could leave two elements naming one code behind.
		i, ok := w.scriptItemHolder(in)
		if !ok || !w.sourceMutationReady(i) {
			return
		}
		live := w
		if w.savedObjects != nil || w.entities[i].ActorLoad.Source.Class != 0 {
			n := w.sourceMutationCopy(i)
			w = &n
		}
		before := w.beginLoadMutation(i)
		if !w.addCarried(i, StackItem(PlainItem(in.Item), 1)) {
			return
		}
		if !w.finishLoadMutation(i, before) || !w.savedMutationValid() {
			return
		}
		if w != live {
			*live = *w
		}

	case ScriptInstantTakeItem:
		// THE REMOVAL: exactly ONE unit of the node's own code leaves the named
		// unit's container and is destroyed.
		//
		// THE SPLIT IS THE ORIGINAL'S OWN: an element holding one unit is
		// UNLINKED WHOLE, and an element holding more loses one from its count
		// and stays exactly where it is (AC-3). Removing the element in both
		// cases would empty a stack of five on one node; leaving an element at
		// a count of zero would break the container invariant that no element
		// holds nothing.
		//
		// IT WALKS TO THE FIRST ELEMENT HOLDING THE CODE. Enchanted instances can
		// now leave several elements with one code, so source order decides which
		// complete object this code-only script removes. A code the container does
		// not hold changes nothing — the original's own finder returns null and
		// the arm falls straight into its notification tail, which writes no
		// simulation state.
		//
		// NOTHING ELSE MOVES, on the give-all's own terms: not the twelve worn
		// places, which are different state from the container, not the gold, the
		// position, the owner, the group, the health, the order or the tick.
		i, ok := w.scriptItemHolder(in)
		if !ok {
			return
		}
		c := w.carried[i]
		if !w.sourceMutationReady(i) {
			return
		}
		for k := range c {
			if c[k].Code != in.Item {
				continue
			}
			w.consumeCarriedUnit(i, k)
			return
		}

	case ScriptInstantTakeOffMap:
		// THE REMOVAL. Resolved THROUGH THE WORLD by id on the unit hand-over's
		// own ground: the reference is fixed at compile time and the world it runs
		// against may no longer hold that entity, so a miss does nothing. The
		// arm's own idempotence is takeOffMap's (presence.go), not this call
		// site's.
		if !in.HasUnit {
			return
		}
		if i := indexOfEntity(w.entities, in.Unit); i >= 0 {
			w.takeOffMap(i)
		}

	case ScriptInstantReturnToMap:
		// THE RETURN. Same resolution, same refusals. The bounded search, the
		// failure that changes nothing and the absence of a retry are all
		// returnToMap's.
		if !in.HasUnit {
			return
		}
		if i := indexOfEntity(w.entities, in.Unit); i >= 0 {
			w.returnToMap(i)
		}

	case ScriptInstantSwapOnMap:
		if !in.HasUnit || !in.HasUnit2 {
			return
		}
		gi := indexOfEntity(w.entities, in.Unit)
		ri := indexOfEntity(w.entities, in.Unit2)
		if gi < 0 || ri < 0 || gi == ri {
			return
		}
		x, y := w.entities[gi].X, w.entities[gi].Y
		w.takeOffMap(gi)
		w.placeNear(ri, x, y)

	case ScriptInstantGroupOffMap:
		// THE GROUP REMOVAL: instant 16's arm once per member, in ascending entity
		// id. Neither this arm nor the one below tests the member first, neither
		// touches group membership, and a group id no entity carries changes
		// nothing — which is not a failure, on the group hand-over's own ground.
		if !in.HasGroup {
			return
		}
		for _, i := range w.groupMembers(in.Group) {
			w.takeOffMap(i)
		}

	case ScriptInstantGroupOnMap:
		// THE GROUP RETURN: instant 17's arm once per member. EACH MEMBER SEARCHES
		// FROM ITS OWN RETAINED CELL, because the arm it calls reads that cell off
		// the member — the original walks the group's list and hands each member
		// to the same helper the single return arm calls, with nothing carried
		// between members. A member whose search fails stays off the map and the
		// walk continues.
		if !in.HasGroup {
			return
		}
		for _, i := range w.groupMembers(in.Group) {
			w.returnToMap(i)
		}

	case ScriptInstantGroupOrder:
		w.cmdGroupOrder(in)

	// THE THREE SPELL ARMS (0165). The two cast arms only APPEND a pending
	// record: nothing is cast here, because the decoded machine appends a
	// temporary caster to a list and a later tick's walker resolves it
	// (scriptcast.go, stepScriptCasts). Instant 29 writes now, because its
	// own arm is a walk of six slots and a word store with nothing deferred
	// (celleffect.go, `TRIG-CELLEFFECT-045`).
	case ScriptInstantCastAtCell:
		w.castAtCell(in)
	case ScriptInstantCastAtUnit:
		w.castAtUnit(in)
	case ScriptInstantCellEffectAge:
		w.setCellEffectTime(in.Args[0], in.Args[1], in.Args[2], in.Args[3])

	// THE LAST FIVE (0166).
	case ScriptInstantFormation:
		// THE FORMATION MODE. The player is a REFERENCE and the mode is a plain
		// parameter, which is the whole content of the amendment this arm is built
		// on: a reference-typed slot takes no `p` index (`TRIG-PARAM-030`), so the
		// campaign's one node authors mode 0 and not the player index it was once
		// read as. A node naming no player writes nothing, on every other
		// reference site's own rule.
		if !in.HasPlayer {
			return
		}
		w.setFormationMode(in.Player, in.Args[0])

	case ScriptInstantDropAll:
		// DROP ALL, resolved through the world by id on the unit hand-over's own
		// ground.
		if !in.HasUnit {
			return
		}
		if i := indexOfEntity(w.entities, in.Unit); i >= 0 {
			w.dropAll(i)
		}

	case ScriptInstantCellTail:
		// THE CELL TAIL. It names no reference at all: its four plain parameters
		// are the spell, the power and the cell.
		w.setCellTail(in.Args[0], in.Args[1], in.Args[2], in.Args[3])

	case ScriptInstantUnitEffectAge:
		// THE ATTACHED EFFECT'S DURATION (TRIG-EFFECTTIME-034).
		if !in.HasUnit {
			return
		}
		if i := indexOfEntity(w.entities, in.Unit); i >= 0 {
			w.setUnitEffectTime(i, in.Args[0], in.Args[1])
		}

	case ScriptInstantProperty:
		// THE PROPERTY SETTER.
		if !in.HasUnit {
			return
		}
		if i := indexOfEntity(w.entities, in.Unit); i >= 0 {
			w.setUnitProperty(i, in.Args[0], in.Args[1])
		}

	case ScriptInstantStructField:
		// THE STRUCTURE FIELD SETTER (1033 B3, `TRIG-CHECK-053`): the
		// referenced structure's Field42 becomes the low 16 bits of the
		// node's own first plain parameter, unconditionally — no selector,
		// no notification call, unlike instant 34's three-way store above.
		//
		// A STRUCTURE REFERENCE THAT DID NOT RESOLVE DOES NOTHING, the
		// unresolved-unit treatment every other instant arm applies. Not
		// reachable from shipped content (`TRIG-BIND-010`); kept for the
		// check-21 arm's own reason.
		if in.HasStructure {
			if i := indexOfStructure(w.structures, in.Structure); i >= 0 {
				w.structures[i].Field42 = uint16(in.Args[0])
			}
		}
	}
}

// setUnitEffectTime is instant 30's dispatch leaf (TRIG-EFFECTTIME-034). The
// entity index proves that the compiled reference still resolves. The canonical
// attached list owns existence and remaining duration; SpellFX is presentation
// state and does not participate in matching.
func (w *World) setUnitEffectTime(i int, spell, duration int32) {
	w.setAttachedEffectDuration(w.entities[i].ID, uint8(spell), uint16(duration))
}

// setUnitProperty is instant 34's whole arm.
//
// THREE SELECTORS STORE AND EVERY OTHER VALUE STORES NOTHING, which is the arm's
// own fall-through and not a guard: the routine has three distinct immediates
// and no default store. The campaign authors selector 6 once and selector 16
// four times; 15 is implemented in the original and absent from the corpus, and
// it is implemented here for the same reason — it is one of the arm's three
// cases, and leaving it out would be a divergence with nothing behind it.
//
// THE VALUE IS A WORD and there is NO CLAMP, no maximum-health update and no
// derived-stat recomputation. A health above the unit's own maximum is
// written and kept.
//
// A HEALTH THAT LEAVES THE UNIT NOT ALIVE GOES THROUGH THE death transition.
// Conversely, a signed store crossing from non-positive to positive runs the
// dedicated revival transition: corpse presentation clears and the
// death-time defence shift is restored once (HERO-REVIVE-068). The word is
// sign-extended; treating it as uint16 would turn an authored negative body
// into a large living health value.
func (w *World) setUnitProperty(i int, selector, value int32) {
	e := &w.entities[i]
	switch selector {
	case propertyHealth:
		before := e.HP
		previous := *e
		e.setCurrentHealth(int32(int16(uint16(value))))
		w.reportHealthLoss(previous, i)
		if e.HP <= 0 {
			w.clearFelled(i)
		} else {
			w.restoreAfterHealthGain(i, before)
		}
	case propertyDefence:
		e.Defence = int32(uint16(value))
	case propertyAbsorption:
		e.Absorption = int32(uint16(value))
	}
}

// containerHolds reports whether a container holds at least one unit of code.
//
// It is the check side of the finder instants 11 and 13 already reach: the same
// walk over the same elements, answering whether one matches instead of acting
// on it. A folded container holds at most one element per code, so the walk
// stops at the first match and there is no count to accumulate.
//
// A code of 0 is not special-cased. No element can hold it, because 0 names no
// item, so the walk simply finds nothing.
func containerHolds(c []ItemStack, code uint16) bool {
	for _, st := range c {
		if st.Code == code {
			return true
		}
	}
	return false
}

// scriptItemHolder is the FOUR REFUSALS both item arms share, and the index
// of the container they write when none of them applies.
//
// It is one function and not two copies because the two arms refuse exactly the
// same four things and a copy is what would drift: a node binding no unit, a
// node binding no item, a node naming an entity this world does not hold, and a
// code of ZERO. Each leaves the world exactly as it was found.
//
// THE ZERO IS NOT A GUARD AGAINST A MALFORMED RECORD, it is the arm's own
// behaviour. Class is bits 8..11 of the code and a class of zero resolves to
// nothing, so the original's factory hands the add arm a null and its finder
// hands the take arm a miss; both then run only the notification, which writes
// no simulation state. carryFault already refuses a zero for the same reason on
// every other path into a container, so admitting one here would be the one way
// this package could build a container it cannot marshal and read again.
//
// THE ENTITY IS RESOLVED THROUGH THE WORLD BY ID rather than trusted from
// the compiled record, on the give-all's own ground: an id fixed at compile
// time may no longer be in the world the trigger fires against, and a miss
// does nothing. Finding it by id is also what makes which entity these arms
// write independent of the order the world happens to hold its entities in.
func (w *World) scriptItemHolder(in ScriptInstant) (int, bool) {
	if !in.HasUnit || !in.HasItem || in.Item == 0 {
		return 0, false
	}
	i := indexOfEntity(w.entities, in.Unit)
	return i, i >= 0
}

// cmdGroupOrder is opcode 6's own dispatch: which sub-command, over which
// group, at which cell.
//
// A node naming no Group changes nothing. Command17 shares one setter for
// native and imported Groups. Catalogue18 and unknown values reach no arm;
// the support table above reports the same boundary.
func (w *World) cmdGroupOrder(in ScriptInstant) {
	if in.HasGroup && in.Args[0] == int32(orderRoam) {
		w.cmdGroupRoam(in.Group)
		return
	}
	if w.savedGroups != nil {
		w.cmdSavedGroupOrder(in)
		return
	}
	if !in.HasGroup {
		return
	}
	switch in.Args[0] {
	case int32(orderGuard):
		w.cmdGroupGuard(in.Group)
	case int32(orderStandGround):
		w.cmdGroupStandGround(in.Group)
	case int32(orderSwarm):
		w.cmdGroupSwarm(in.Group, in.Args[1], in.Args[2])
	case int32(orderMove):
		w.cmdGroupCommandedMove(in.Group, orderMove, in.Args[1], in.Args[2])
	case int32(orderSwarm2):
		w.cmdGroupCommandedMove(in.Group, orderSwarm2, in.Args[1], in.Args[2])
	case subCommandPatrol:
		w.cmdGroupPatrol(in)
	case subCommandAttack:
		w.cmdGroupAttack(in)
	case subCommandDefend:
		w.cmdGroupEscort(in, actorStateDefend)
	case subCommandFollow:
		w.cmdGroupEscort(in, actorStateFollow)
	}
}

// stopGroupMembers is the STOP all three of the last sub-commands begin
// with: every living member's destination, stored route, stall count, victim
// and attack cycle cleared, and the group's own stored order set to none.
//
// It is ONE function with three callers because all three arms perform the same
// stop — the decoded dispatcher calls one helper per member and then writes 0
// over the group order — and three copies is what would drift.
//
// THE ORDER GOES TO NONE AND NOT TO THE SUB-COMMAND'S OWN NUMBER, which is the
// same thing subCommandPatrol already does and for the same reason: what these
// three write is PER MEMBER, and a group order beside it would be a second
// authority over members the arm has just given individual orders to.
//
// The group parameter names a raw id, which groupsNamed resolves to the one
// record of its single owner.
func (w *World) stopGroupMembers(group uint32) []int {
	if w.savedGroups != nil {
		return w.stopSavedGroupMembers(group)
	}
	var stopped []int
	for _, gi := range w.groupsNamed(group) {
		g := &w.groups[gi]
		g.order = orderNone
		for _, mi := range w.groupLivingMembers(g.owner, g.group) {
			w.clearOrder(mi)
			w.retainCycleForState(mi)
			w.entities[mi].clearGroupSpeed()
			stopped = append(stopped, mi)
		}
	}
	return stopped
}

// acquireInPlace is `AI-STATE-011`'s acquire-with-no-leash, which the NAMED
// unit of all three sub-commands takes.
//
// The decoded arm writes the state, the member's OWN cell as the order's anchor,
// and a zero leash. This build's anchor is the post pair, which is what
// cmdGroupGuard and cmdGroupStandGround already write for the same "hold where
// you stand" meaning, so the anchor is not a fourth representation of a cell.
//
// It is also what a VETOED member takes under sub-command 10, and that is the
// arm's own shape rather than a convenience here: the vetoed member's three
// writes are byte for byte the named unit's three.
func (w *World) acquireInPlace(i int) {
	e := &w.entities[i]
	e.clearEscort()
	e.ActorState = actorStateAcquire
	e.PostX, e.PostY = e.X, e.Y
}

// targetVetoed is sub-command 10's gate: whether the preference matrix
// refuses the pair (member, candidate) outright.
//
// The decoded arm calls the ORDINARY target-cost routine and compares the result
// with the sentinel `0xffffff`, which that routine returns from exactly one arm
// — the one taken when the preference matrix cell is 0. This build's scorer is
// candidateCost and its sentinel is scoreSeed, so the gate is that comparison and
// nothing else: none of the scorer's modifiers, distance rewrite or ordering
// affects it.
//
// orderNone IS PASSED because candidateCost's two variants are selected by
// `order == orderStandGround` alone and the decoded arm calls the ordinary one
// (spec SC-4). Any other non-stand-ground value would select the same variant;
// orderNone is the one that names no order at all.
func (w *World) targetVetoed(mi, ci int) bool {
	return w.candidateCost(mi, ci, orderNone) == scoreSeed
}

// cmdGroupAttack is sub-command 10, the script attack.
//
// THE GROUP IS WALKED TWICE, which is the dispatcher's own shape: the stop runs
// over every member before any member is given its new disposition, so no member
// is scored against a world half of which is already re-ordered.
//
// THE PER-MEMBER GATE IS THE POINT OF THIS ARM. `AI-SCRIPTATTACK-120` was
// published without it and the row's headline was retracted for exactly that:
// between the group-order store and the per-member call the dispatcher scores
// each member against the named unit, and on the veto sentinel it calls a
// DIFFERENT helper — the member acquires in place and never engages. The same
// gate sits on the player's own attack order.
//
// IT IS AUTHORED-UNREACHABLE IN THE SHIPPED CAMPAIGN, established positively
// (`TRIG-GRPARM-047`): the matrix's only zero cells need a candidate of movement
// domain 3, and all five shipped references are Humans, which ship no movement
// type on either root and are therefore domain 1. It is built anyway because the
// scoring inputs are already in this tree, and because it is exactly the seam an
// authored map would reach.
//
// THREE REFUSALS, each leaving the world exactly as it was found INCLUDING
// THE STOP: a node naming no group, a node naming no unit, and a node naming
// a unit this world does not hold. The last two are asked before the stop
// rather than after it, so a node whose target has left the world does not
// silently halt a group.
func (w *World) cmdGroupAttack(in ScriptInstant) {
	if !in.HasUnit {
		return
	}
	ti := indexOfEntity(w.entities, in.Unit)
	if ti < 0 || !w.entities[ti].OrdinaryTargetable() {
		return
	}
	for _, mi := range w.stopGroupMembers(in.Group) {
		if mi == ti || w.targetVetoed(mi, ti) {
			w.acquireInPlace(mi)
			continue
		}
		// TRIG-GRPARM-047 names the same explicit setter as the player command.
		w.entities[mi].clearEscort()
		accepted := !w.actorCastBusy(mi) && w.attachAttack(mi, in.Unit, AttackTargetUnit, true)
		// AND THE STATE THE ORDER BELONGS TO (1141). It was left at guard while no
		// arm read the byte; the per-actor guard arm reads it now, and a member
		// left at guard here would have its order broken off by that arm's own
		// leash on the next tick.
		if accepted {
			w.publishEngagementOrder(mi)
		}
	}
}

// cmdGroupEscort is sub-commands 11 and 15: ONE body and a state constant.
//
// That is the shape of the thing being reconstructed and not an economy here.
// `AI-FOLLOWSET-116` compares the two bodies byte for byte — 192 bytes at a
// fixed delta, differing in exactly three, of which one is the state immediate
// and the other two are the two changing halves of one call's displacement — so
// two functions in this file would be two copies of one piece of source.
//
// THE SPLIT IS ON `member == the named unit`: the named unit acquires in
// place, and every other member takes the escort state with the named unit
// as its target. The escort range is the node's own second plain value, or 3
// where that value's low byte is 0. Shipped ranges are 1 to 6 on both roots,
// so the coercion is authored-unreachable — but the arm loads a 32-bit
// authored field through a byte move, so an authored 256 does reach it
// (`TRIG-GRPLIMIT-048`), and the narrowing is written out rather than folded
// away for that reason.
func (w *World) cmdGroupEscort(in ScriptInstant, state uint8) {
	if !in.HasUnit {
		return
	}
	ti := indexOfEntity(w.entities, in.Unit)
	if ti < 0 {
		return
	}
	span := uint8(in.Args[1])
	if span == 0 {
		span = escortRangeDefault
	}
	for _, mi := range w.stopGroupMembers(in.Group) {
		if mi == ti {
			w.acquireInPlace(mi)
			continue
		}
		e := &w.entities[mi]
		e.ActorState = state
		e.EscortTarget, e.HasEscortTarget = in.Unit, true
		e.EscortRange = span
	}
}

// escortRangeDefault is what an escort range of 0 becomes: 3, the immediate both
// helpers store when the authored byte is zero (`AI-FOLLOWRANGE-115`). It is
// unreachable from shipped data — every shipped range is 1 to 6 on both roots —
// and it is reachable from an authored 256, which the byte path folds to 0.
const escortRangeDefault uint8 = 3

// cmdGroupGuard is sub-command 1: the order becomes Guard and the notice
// base is RE-FROZEN from the group's LIVING members' present geometry —
// noticeBase and freezeGroups' own construction rule, run again rather than
// a second one; there is no override parameter because the law's own is 0 on
// every path this build has. It is the only writer of a group's base after
// construction.
//
// A group id naming more than one owned record takes the write on the one
// record groupsNamed resolves it to.
//
// THE SAME WALK ALSO ANCHORS THE POST: every living member's post becomes
// the cell it presently occupies, unconditionally — no test on whether it
// is moving, engaged or idle — over the same members slice the notice base
// above is frozen from, rather than a second call to groupLivingMembers.
func (w *World) cmdGroupGuard(group uint32) {
	for _, gi := range w.groupsNamed(group) {
		g := &w.groups[gi]
		g.order = orderGuard
		members := w.groupLivingMembers(g.owner, g.group)
		var cx, cy int32
		if len(members) > 0 {
			cx, cy = groupCentroid(w.entities, members)
		}
		g.base = noticeBase(w.entities, members, cx, cy)
		for _, mi := range members {
			w.entities[mi].PostX, w.entities[mi].PostY = w.entities[mi].X, w.entities[mi].Y
		}
	}
}

// cmdGroupStandGround is sub-command 3: the order becomes Stand Ground,
// every living member's post is anchored at the cell it presently occupies,
// and each member is stood where it is (standScriptedMembers).
func (w *World) cmdGroupStandGround(group uint32) {
	for _, gi := range w.groupsNamed(group) {
		g := &w.groups[gi]
		g.order = orderStandGround
		members := w.groupLivingMembers(g.owner, g.group)
		for _, mi := range members {
			w.entities[mi].PostX, w.entities[mi].PostY = w.entities[mi].X, w.entities[mi].Y
		}
		w.standScriptedMembers(members)
	}
}

// cmdGroupSwarm is sub-command 2: the order becomes Swarm and the commanded
// cell becomes (x, y) — the node's second and third plain parameters. It
// gives no member a destination; armSwarm is the only reader of what this
// writes.
func (w *World) cmdGroupSwarm(group uint32, x, y int32) {
	for _, gi := range w.groupsNamed(group) {
		w.groups[gi].order = orderSwarm
		w.groups[gi].commandedX, w.groups[gi].commandedY = x, y
	}
}

// cmdGroupCommandedMove is sub-commands 4 and 5: the order and the commanded
// cell are written, and the group's LIVING members are handed
// issueGroupDestination — the same distribution the player's own group
// move performs, over the node's (x, y).
func (w *World) cmdGroupCommandedMove(group uint32, order uint8, x, y int32) {
	for _, gi := range w.groupsNamed(group) {
		g := &w.groups[gi]
		g.order = order
		g.commandedX, g.commandedY = x, y
		w.issueGroupDestination(w.groupLivingMembers(g.owner, g.group), cell{x: x, y: y})
	}
}

// cmdGroupPatrol is sub-command 14, Patrol: the named group's order becomes
// orderNone and nothing else on the group record moves — in particular not
// the commanded cell, which this sub-command does not read and an earlier
// command may have left behind. Every LIVING member is stopped, handed to
// the actor layer at patrol, and given a ring built from where it stands now
// to the node's own cell.
//
// IT DOES NOT REUSE issueGroupDestination, on purpose. That function is the
// distribution the player's own group move shares with the script's Move and
// Swarm 2 setters: it spreads a formation about a centroid, sets the group's
// rate term on the formation arm, and clamps — three behaviours this
// command does not have. The law's own Patrol command gives each member NO
// destination at all here; it is the actor pass, at most one tick later,
// that gives it one (actor.go). Sharing the body would have meant passing
// flags in to turn its own content off, for a function whose whole point is
// that content.
//
// The group parameter resolves through groupsNamed, as the siblings' does.
func (w *World) cmdGroupPatrol(in ScriptInstant) {
	for _, gi := range w.groupsNamed(in.Group) {
		g := &w.groups[gi]
		g.order = orderNone //
		for _, mi := range w.groupLivingMembers(g.owner, g.group) {
			e := &w.entities[mi]
			w.clearOrder(mi)          //
			w.retainCycleForState(mi) //
			e.clearGroupSpeed()       //
			e.ActorState = actorStatePatrol
			e.PatrolHeadX, e.PatrolHeadY = e.X, e.Y
			e.PatrolTailX, e.PatrolTailY = w.bounds.clamp(in.Args[1], in.Args[2])
			e.PatrolLeg = patrolLegTail
			e.PostX, e.PostY = e.X, e.Y
		}
	}
}

// scriptReport applies the campaign reporter's exact-one counter tests.
//
// LOSE IS TESTED FIRST, so a mission that wins and loses inside one pass is
// lost. And both tests are for EXACTLY ONE rather than for any nonzero count:
// that is the reporter's own comparison, and it has a consequence worth stating
// because a map can reach it — two losing arms firing in one pass take the
// counter from 0 to 2, and no report is ever made.
//
// Loss is terminal. A won mission can subsequently fail, but a lost mission
// cannot win without a separate reset (TRIG-END-009, MISSION-DEFEAT-045).
func (w *World) scriptReport() {
	if w.outcome == OutcomeLost {
		return
	}
	if w.rom2 != nil {
		if int32(w.lost) > 0 {
			w.outcome = OutcomeLost
		} else if w.won != 0 {
			w.outcome = OutcomeWon
		}
		return
	}
	if w.lost == 1 {
		w.outcome = OutcomeLost
		return
	}
	if w.won == 1 {
		w.outcome = OutcomeWon
	}
}

// scriptEntity resolves a compiled reference to a held entity or a departed one.
// A reference that never resolved measures nothing.
func (w *World) scriptEntity(id EntityID, has bool) (Entity, bool) {
	if !has {
		return Entity{}, false
	}
	i := indexOfEntity(w.entities, id)
	if i < 0 {
		return w.departedEntity(id)
	}
	return w.entities[i], true
}

// departedEntity views a removed actor's retained terminal row. The reap moves a
// torn-down actor to the dead list without freeing it (TRIG-REAP-017), and the
// check arms keep reading it: arm 5 answers 0, arm 6 0xff and arm 18 counts the
// loss (TRIG-COND-003, TRIG-COUNT-015, TRIG-DIST-014). The view is dead, holds
// the row's cell and health, and carries nothing.
func (w *World) departedEntity(id EntityID) (Entity, bool) {
	view := func(cell uint16, hp int32, stage uint8) (Entity, bool) {
		return Entity{ID: id, X: int32(cell & 255), Y: int32(cell >> 8), HP: hp, Decay: DecayStage(stage)}, true
	}
	for _, r := range w.currentTerminalActors {
		if r.ID == id {
			return view(r.Cell, r.HP, r.Stage)
		}
	}
	for _, r := range w.originalDead {
		if r.ID == id && r.terminal.Stage != 0 {
			return view(r.terminal.Cell, int32(r.terminal.HP), r.terminal.Stage)
		}
	}
	return Entity{}, false
}

// scriptDead is what "dead" means to a script, in ONE place.
//
// THE CARVE-OUT IS THE DYING WINDOW AND NOT THE DOWNED STATE. Alive, Downed and
// Dead partition the health, and the earlier reading put Downed on the dead
// side because a downed unit is out of the fight; that argument still holds
// after the window closes, and the window is Entity.Dying's own decay reading
// rather than a health one. A unit whose class names no dying time has no
// window at all and is unaffected — which is every entity in this package's own
// script tests, and why the shipped VIP test did not have to change.
//
// Heal and a positive script health write can now bring a body above zero back
// through the shared revival transition. Mission-loss deferral remains a
// separate owner rule because ROM1's immediate-loss re-placement arm is not
// implemented here (DIV-219).
func scriptDead(e Entity) bool { return !e.Alive() && !e.Dying() }

// scriptVIPDead is the narrower teardown predicate used by check 18. The
// original admits the mission-loss side effect only after the dying countdown
// has expired and health has reached -10; an authored zero-health body is a
// stable, healable objective rather than a completed death (HERO-ZERO-070,
// TRIG-REAP-017).
func scriptVIPDead(e Entity) bool {
	return scriptDead(e) && e.HP <= decayBonesHP
}

// scriptDistance is the distance a check measures.
//
// WHICH METRIC THAT IS used to be THE ONE THING IN THIS FILE THE EVIDENCE DID
// NOT FIX; TRIG-DIST-014 closes it, at High. Every arm that measures a
// distance calls into this same helper, and read at instruction level it is
// fifteen instructions returning max(|dx|, |dy|) — no multiply, no FSQRT, no
// FILD, and no addition of the two terms anywhere in it. Both rivals are
// excluded by the instructions actually present, not by analogy — though the
// analogy also holds: Chebyshev is the metric every other located distance
// helper in this engine computes, too — the acquisition radius, the guard
// spread, the wander threshold, the substitute-goal rings and the route
// budget are all whole-cell Chebyshev.
//
// IT IS A SEAM ON PURPOSE: one function, and every arm that measures a distance
// goes through it, so the day a further arm needs one this is the only edit.
//
// What TRIG-DIST-014 does NOT close is its byte-width clause, and that stays
// open at Medium: whether the original masks its coordinates to a byte before
// subtracting them, or subtracts at full width and masks only the answer.
// TRIG-CHECK-052 does not close it either. It settles how wide check 16 loads
// its OWN two parameters — 8 bits, at two named instructions — which is a
// property of that arm's parameter read and says nothing about the coordinates
// this helper subtracts; check 16 masks its parameters at the call site for
// that reason. This function takes the second reading — it hands back the unmasked int64
// difference, and a caller that needs the byte truncates the RESULT through
// scriptByte rather than truncating ax/ay/bx/by going in. That ordering is
// reproduced, not proven, and the two would disagree on a map built to make a
// coordinate wrap exactly at the byte boundary.
//
// The arithmetic is int64 so that two int32 coordinates at opposite ends of the
// range cannot wrap into a small, plausible distance.
func scriptDistance(ax, ay, bx, by int32) int64 {
	dx, dy := abs64(int64(ax)-int64(bx)), abs64(int64(ay)-int64(by))
	if dx > dy {
		return dx
	}
	return dy
}

// scriptByte is the truncation two distance arms apply to their answer: the low
// byte, so a distance of 300 measures 44 and a map cannot tell it from one of 44.
// It is the arm's own mask and is reproduced rather than widened.
func scriptByte(v int64) int32 { return int32(v & 0xff) }

// setRegister writes a COMPILED subscript, which NewScript has already bounded.
// It exists so that every register write in this file goes through a named
// function and the bound is stated once.
func (w *World) setRegister(r, v int32) { w.registers[r] = v }

// registerAt and setRegisterAt read and write an AUTHORED subscript — a number a
// map wrote, bounded by nothing.
//
// THIS IS WHERE THE OVERRUN IS REFUSED. The original bounds-checks neither array,
// so an authored subscript past the register file walks into the latch array and
// an authored one past that walks off the end. That is undefined behaviour and
// not a mechanism, so a read outside the file answers zero and a write outside
// it does nothing — a disclosed divergence, and the customisation limit is the
// hundred registers themselves.
func (w *World) registerAt(r int32) int32 {
	if r < 0 || r >= scriptRegisters {
		return 0
	}
	return w.registers[r]
}

func (w *World) setRegisterAt(r, v int32) {
	if r < 0 || r >= scriptRegisters {
		return
	}
	w.registers[r] = v
}

// presetRegisters applies the build-time constants: every constant check's
// register takes that node's first value.
//
// It runs once when a world is CONSTRUCTED from a map, which is what makes a
// constant a mission variable rather than a value restored every pass. A
// world decoded from an Againrom save does not run it again: registers is
// itself wire-form state, so the byte form already carries them as they last
// stood, and replaying a build-time preset over played state would discard
// it.
//
// ImportOriginalSession is the one caller that runs it a SECOND time, right
// after copying in an original save's own hundred registers. TRIG-SAVE-008
// (corrected) states the original restores the session block before it
// rebuilds the trigger programme, and MISSION-SLOT-008 states that rebuild's
// second pass assigns every check-node a slot, constant and ordinary check
// alike — but only the constant kind actually WRITES a value there, from its
// own node; an ordinary check's slot is not preset, it is recomputed the next
// time that check runs. So calling this again after the copy reproduces the
// preset half of the original's unconditional pass without an actual
// rebuild, for all three slot classes docs/1130/story.md names: a constant
// slot takes the compiled value, discarding whatever the file held; an
// ordinary check-owned slot keeps the file's value only until that check's
// own next full tick recomputes it, exactly as the original would; an
// authored variable index above the map's check-node count is neither, so
// nothing here or after ever touches it and the file's value survives.
func (w *World) presetRegisters() {
	if w.script == nil {
		return
	}
	for _, c := range w.script.checks {
		if c.Op == ScriptCheckConstant {
			w.registers[c.Register] = c.Args[0]
		}
	}
}

// Script returns the compiled script this world is running, or nil.
func (w *World) Script() *Script { return w.script }

// Outcome returns what the mission has come to.
func (w *World) Outcome() Outcome { return w.outcome }

// ScriptCounters returns the win and lose counters as they stand.
//
// They are handed out as a PAIR because the reporter reads them as a pair, and a
// consumer that saw one without the other could not tell a mission that has been
// decided from one whose counters have both moved.
func (w *World) ScriptCounters() (won, lost uint32) { return w.won, w.lost }

// ScriptRegister returns register r, or zero for a subscript outside the file —
// the same answer an authored subscript outside the file gets.
func (w *World) ScriptRegister(r int32) int32 { return w.registerAt(r) }

// ScriptRegisters returns the complete hundred-slot register file, in
// subscript order. It is a copy: the caller cannot reach live state through
// it. A native mission SAVE (docs/1130/story.md) is its intended reader.
func (w *World) ScriptRegisters() [scriptRegisters]int32 { return w.registers }

// RawSessionHead and RawSessionMid return the two opaque session-block spans
// ImportOriginalSession carried in, byte for byte, from an original save
// (SAV-SESS-031). Both are the zero array for a world that never imported
// one. A native mission SAVE is their intended reader; nothing in this
// package interprets either.
func (w *World) RawSessionHead() [sessionRawHeadLen]byte { return w.rawSessionHead }
func (w *World) RawSessionMid() [sessionRawMidLen]byte   { return w.rawSessionMid }

// SetRawSessionHead and SetRawSessionMid overwrite the two carried spans
// directly, with none of ImportOriginalSession's register/latch/diplomacy
// handling.
func (w *World) SetRawSessionHead(v [sessionRawHeadLen]byte) { w.rawSessionHead = v }
func (w *World) SetRawSessionMid(v [sessionRawMidLen]byte)   { w.rawSessionMid = v }

// ScriptPassJustRan reports whether the step just completed ran the script pass.
func (w *World) ScriptPassJustRan() bool {
	if w.hasSessionClock {
		return int32(uint32(w.tick)-1)%scriptCycle == scriptPassPhase
	}
	return w.tick > 0 && (w.tick-1)%scriptCycle == scriptPassPhase
}

// ScriptTriggerBlocked reports whether a saved group's unresolved count makes
// the pass skip the trigger at map position latch.
func (w *World) ScriptTriggerBlocked(latch int32) bool {
	if w.script == nil {
		return false
	}
	for _, t := range w.script.triggers {
		if t.Latch == latch {
			return w.savedTriggerBlocked(t)
		}
	}
	return false
}

// ScriptLatched reports whether the trigger at map position i has fired. A
// position outside the latch array has not.
func (w *World) ScriptLatched(i int32) bool {
	if i < 0 || i >= scriptLatches {
		return false
	}
	return w.latches[i] != 0
}

// sortedGaps is not used by the runtime and exists so the two consumers of a gap
// list — a report and a test — order it the same way. It sorts by kind, then
// opcode, then index.
func sortedGaps(gs []ScriptGap) []ScriptGap {
	out := append([]ScriptGap(nil), gs...)
	sort.Slice(out, func(i, j int) bool {
		if out[i].Kind != out[j].Kind {
			return out[i].Kind < out[j].Kind
		}
		if out[i].Op != out[j].Op {
			return out[i].Op < out[j].Op
		}
		return out[i].Index < out[j].Index
	})
	return out
}

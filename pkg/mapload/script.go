package mapload

// The binder converts decoded type-7 nodes into a sim program once at load time.
// References come from the caller. ROM1 uses its authored identifier bands;
// ROM2 resolves literal units and explicitly supplied party roles, reporting
// missing or dynamic references without inventing a subject.

import (
	"fmt"

	"againrom/pkg/formats/alm"
	"againrom/pkg/sim"
)

// The parameter type codes, and which of them are plain integers.
//
// A compiled node holds its plain parameters PACKED IN ENCOUNTER ORDER while the
// file stores every parameter by its own Par<i> slot — so a node whose Par0 is a
// unit reference and whose Par1 and Par2 are a coordinate pair compiles to plain
// parameters 0 and 1, which is what the arm that reads them expects. A binder
// that kept the file's slot numbering would hand every such arm the wrong
// numbers.
const (
	typeUnset     uint32 = 0 // the slot carries no parameter
	typeInt       uint32 = 1 // int, and Enum, which serialises as one
	typeGroup     uint32 = 2
	typePlayer    uint32 = 3
	typeUnit      uint32 = 4
	typeX         uint32 = 5
	typeY         uint32 = 6
	typeConst     uint32 = 7
	typeItem      uint32 = 8
	typeStructure uint32 = 9
)

func isPlainParam(t uint32) bool {
	return t == typeInt || t == typeX || t == typeY || t == typeConst
}

// Compile-time forms do not reach runtime dispatch. ROM1 consumes drop cells;
// ROM2 preserves packed coordinates without assigning an unknown consumer.
const (
	scriptConstOpcode  uint32 = 0x10002 // a CONDITION node: the mission variable's preset
	scriptDropOpcode   uint32 = 0x10002 // an ACTION node: the drop location
	scriptObjectOpcode uint32 = 0x10003
	scriptBuildFloor   uint32 = 0x10002

	// The Target_Unit bands.
	scriptHeroBandLow  = 10001
	scriptHeroBandHigh = 11000
)

// scriptMessageOpcode supplies ROM1 latch metadata; ROM2 emits execution reports.
const scriptMessageOpcode int32 = 2

// ScriptRefs is how a caller resolves the identifiers a script names.
//
// Units maps a placed unit's own id — the type-6 record's identifier word, which
// is what a below-band Target_Unit names — to the entity that record became. It
// is a lookup and is never iterated, so it puts no map iteration order on any
// path that reaches a world.
//
// Hero binds 10001 and Companion binds 10002. Roles binds other explicitly
// identified hero-band subjects; none is a raw party subscript. Presence
// flags and map membership distinguish entity zero from absence.
//
// Structures maps a type-4 record's own +0x12 word (`alm.Object.Field12`,
// `ALM-TRIG-046`) to the structure it became, on Units' own terms (1033 B3):
// a Target_Structure parameter names one of these words and nothing else,
// there being no hero band and no static table for this reference kind.
type ScriptRefs struct {
	Units        map[uint16]sim.EntityID
	Hero         sim.EntityID
	HasHero      bool
	Companion    sim.EntityID
	HasCompanion bool
	Roles        map[uint32]sim.EntityID
	Structures   map[uint16]sim.StructureID
}

// ScriptUnits is the unit-id table over the entity ids FromALM assigns: the type-6
// record at index i becomes entity i, so this pairs each record's own identifier
// word with that index.
//
// A duplicate identifier keeps the LAST record in the original's walk: the
// binder fills its unit map from each player's actor list in turn and a repeated
// key overwrites (UNIT-BIND-131). The walk visits players in roster order, taken
// here as ascending owner, and each player's actors in placement order, so the
// later record of the greater owner answers. Nothing in the shipped campaign
// needs the rule (the words are distinct per map); it is here so a customised
// map binds as the original does and not by iteration.
//
// THE PARTY IS THE SECOND SOURCE OF ENTRIES, and it exists because a resume
// takes records OUT of the map. A character restored from an original save
// can be a person the map placed and the player then took into his own
// group; the resume withdraws that record so one person does not become two
// entities, and the id the script names then belongs to nobody on the map. A
// member whose Saved names a unit contributes that id bound to the entity he
// becomes, so the reference follows the person rather than the record.
//
// WITHOUT IT THE REFERENCE DOES NOT GO QUIET, IT GOES ZERO. An unresolved check
// is still built and still takes its register, and it returns before writing
// anything; the register keeps its initial zero and a proximity comparison
// against a small constant is then satisfied. Mission 20 resumed from an
// original save declared victory at tick 16 on exactly that arm.
//
// A NIL PARTY IS THE MAP ALONE, which is what a caller holding no party — the
// almtool script dump — has always been given. A member's entry OVERWRITES a
// surviving record's: the two collide only where a claimed record was not
// withdrawn, and the commandable figure is the one a script arm should measure.
func ScriptUnits(m *alm.Map, party []PartyMember) map[uint16]sim.EntityID {
	if m == nil && len(party) == 0 {
		return nil
	}
	var units []alm.Unit
	if m != nil {
		units = m.Units
	}
	out := make(map[uint16]sim.EntityID, len(units)+len(party))
	last := make(map[uint16]int, len(units))
	for i, u := range units {
		if j, seen := last[u.UnitID]; seen && units[j].Owner > u.Owner {
			continue
		}
		last[u.UnitID] = i
		out[u.UnitID] = sim.EntityID(i)
	}
	for i := range party {
		s := party[i].Saved
		if s == nil || s.MapUnitID == 0 {
			continue
		}
		out[s.MapUnitID] = PartyEntity(m, i)
	}
	return out
}

// ScriptStructures is the structure-id table over the entity ids a world's
// structure list assigns: type-4 record i becomes StructureID(i)
// (Structures, fromalm.go), so this pairs each record's own +0x12 word
// with that index, on ScriptUnits' own terms.
//
// A duplicate word keeps the FIRST record carrying it, ScriptUnits' own rule:
// nothing in the shipped corpus needs it, and it is here so a customised map
// cannot make the answer depend on iteration.
func ScriptStructures(m *alm.Map) map[uint16]sim.StructureID {
	if m == nil || len(m.Objects) == 0 {
		return nil
	}
	out := make(map[uint16]sim.StructureID, len(m.Objects))
	for i, o := range m.Objects {
		if _, seen := out[o.Field12]; seen {
			continue
		}
		out[o.Field12] = sim.StructureID(i)
	}
	return out
}

// ScriptCell is a cell a build-time action named.
type ScriptCell struct{ X, Y int32 }

// ScriptRaise is one announcement a trigger raises when it fires: the trigger's
// LATCH — its position in the map's own trigger array, the same number the
// compiled trigger carries and the same subscript the world's latch array is
// indexed by — and the event number the action names.
//
// It is reported from the builder because this is where an authored action id
// and its compiled instant are joined. A build-time action has no runtime slot
// at all and therefore contributes neither a raise nor a compiled trigger slot.
type ScriptRaise struct {
	Latch int32
	Event int32
}

// ScriptRefKind names which of the lookup id spaces a ScriptUnresolved value
// came from. It exists because Value alone does not say: a Target_Unit id and
// a Target_Structure id overlap in range, and a renderer that reports the miss
// to a human must know which table it failed to resolve in, not only that it
// failed (1033 follow-up W-1).
type ScriptRefKind uint8

const (
	ScriptRefUnit ScriptRefKind = iota
	ScriptRefStructure
)

// String names the reference kind, so a renderer can use a ScriptRefKind
// directly in a format verb instead of holding its own copy of the two names.
func (k ScriptRefKind) String() string {
	if k == ScriptRefStructure {
		return "structure"
	}
	return "unit"
}

// ScriptUnresolved is one reference the binder could not resolve: which node,
// which id space, and the value it named.
type ScriptUnresolved struct {
	Condition bool // false for an action node
	Kind      ScriptRefKind
	NodeID    uint32
	Opcode    uint32
	Value     uint32
}

// ScriptReport is everything the compile did that the compiled program cannot
// say for itself.
//
// DropCells is the drop table the builder consumed: where the player's party is
// placed. It is carried rather than acted on, because putting a party on a map
// is not this package's business — it is reported so the tier that does that
// work has it without re-walking the script.
//
// DroppedTriggers names, by MAP POSITION, the triggers the builder did not build
// at all: a trigger whose first condition pair's left id is zero is dropped
// whole before anything is built. Twenty-three of the shipped corpus's 421 are.
//
// DiscardedActions names the build-time action opcodes that reached no arm: the
// object-construction form, and anything else at or above the build-time floor,
// which the engine drops with no message.
//
// Unresolved names every reference the binder could not resolve. The node is
// still built and still takes its register — see the note on the compile — but
// it measures nothing, and this is where a consumer finds out.
//
// IT IS NOT COVERED BY THE INERT-TRIGGER REPORT, and reading one for the other is
// how this goes unnoticed. Inertness is derived from UNIMPLEMENTED ARMS: a
// trigger is inert when it reads a register no arm of this build writes, and its
// whole evaluation is then skipped. An unresolved reference is the opposite
// shape — the arm IS implemented, so the trigger is live and is evaluated every
// pass, and the check simply returns before writing anything. The register keeps
// its initial ZERO, and a comparison against zero may well HOLD. So an
// unresolved reference does not disarm a trigger; it can arm one that should not
// have fired, and the inert count will say nothing about it.
// Raises is every announcement the built triggers can raise, in trigger order
// and then in action-slot order — see ScriptRaise for why it is published here
// rather than read back off the compiled program.
type ScriptReport struct {
	PackedCoordinates []uint16
	OmittedActions    []uint32
	OmittedChecks     []uint32
	OmittedTriggers   []int
	DropCells         []ScriptCell
	DroppedTriggers   []int
	DiscardedActions  []uint32
	Unresolved        []ScriptUnresolved
	Raises            []ScriptRaise
}

// Empty reports whether the compile had nothing to say.
func (r ScriptReport) Empty() bool {
	return len(r.OmittedTriggers) == 0 && len(r.OmittedActions) == 0 && len(r.OmittedChecks) == 0 && len(r.PackedCoordinates) == 0 && len(r.DropCells) == 0 && len(r.DroppedTriggers) == 0 &&
		len(r.DiscardedActions) == 0 && len(r.Unresolved) == 0 &&
		len(r.Raises) == 0
}

// CompileScript compiles a map's script into the program pkg/sim runs.
//
// A map carrying no script compiles to no script and an empty report, which is
// not an error: ten of the shipped corpus's loose maps author nothing that can
// win, and a map with no type-7 record at all is the loader's own skipped arm.
//
// THREE PASSES, in the builder's own order:
//
//  1. the ACTIONS, each becoming an instant, except the two forms the builder
//     consumes at load time and the ones it discards;
//  2. the CONDITIONS, each taking the NEXT REGISTER in list order — a runtime
//     check and a constant alike, which is why a mission variable and a check
//     result share one array and why a map may author a variable index that a
//     check overwrites every pass;
//  3. the TRIGGERS, each becoming a pattern, resolving its slots to the
//     subscripts the first two passes assigned.
//
// A NODE THAT CANNOT RESOLVE ITS REFERENCE IS STILL BUILT AND STILL TAKES ITS
// REGISTER, and that is a choice with a rival. The original refuses to build such
// a node and its register goes to the next one instead, which shifts every later
// subscript. Whether that shift ever happens is a question about THE ORIGINAL's
// builder, which resolves every reference all 38 maps make once the Target_Unit
// bands are honoured — it has a live player list, so the hero band costs it
// nothing (ALM-TRIG-046, whose count spans all four reference kinds).
//
// THAT IS NOT A STATEMENT ABOUT THIS BUILD, and this comment used to compress it
// into one — "nothing in the shipped corpus fails to resolve" — which read as a
// claim that the divergence is unreachable here. It is not. This build resolves
// the hero band only when its caller supplies a hero, so a caller that does not
// leaves 196 of the corpus's references unresolved. The sentence is named here
// rather than merely deleted, because for as long as it stood nobody looked.
//
// The divergence is reported rather than hidden: every such node is in
// ScriptReport.Unresolved.
func CompileScript(m *alm.Map, refs ScriptRefs) (*sim.Script, ScriptReport, error) {
	if m == nil {
		return nil, ScriptReport{}, nil
	}
	src, err := m.Script()
	if err != nil {
		return nil, ScriptReport{}, err
	}
	return CompileScriptFrom(src, refs)
}

// CompileScriptFrom is CompileScript over a script already decoded, and it is
// where the three passes are. The two are separate entry points for the reason
// FromALM and FromALMWith are: one is what a caller holding a map reaches for,
// and the other is what a caller holding the script itself — or building one —
// needs, without a map to route it through.
func CompileScriptFrom(src alm.Script, refs ScriptRefs) (*sim.Script, ScriptReport, error) {
	return compileScriptFrom(src, refs, sim.ScriptROM1)
}

func CompileROM2Script(m *alm.Map, refs ScriptRefs) (*sim.Script, ScriptReport, error) {
	if m == nil {
		return nil, ScriptReport{}, nil
	}
	src, err := m.Script()
	if err != nil {
		return nil, ScriptReport{}, err
	}
	return CompileROM2ScriptFrom(src, refs)
}

func CompileROM2ScriptFrom(src alm.Script, refs ScriptRefs) (*sim.Script, ScriptReport, error) {
	return compileScriptFrom(src, refs, sim.ScriptROM2)
}

func compileScriptFrom(src alm.Script, refs ScriptRefs, dialect sim.ScriptDialect) (*sim.Script, ScriptReport, error) {
	var rep ScriptReport
	if src.Empty() && dialect == sim.ScriptROM1 {
		return nil, rep, nil
	}

	// ---- pass 1: the actions.
	instants := make([]sim.ScriptInstant, 0, len(src.Actions))
	instantOf := make(map[uint32]int32, len(src.Actions))
	buildTime := make(map[uint32]bool)
	for _, n := range src.Actions {
		var args [10]int32
		var refd bound
		if dialect != sim.ScriptROM2 || n.Opcode != scriptDropOpcode {
			args, refd = bindScriptParams(n, refs, &rep, false, dialect)
		}
		if n.Opcode >= scriptBuildFloor {
			buildTime[n.ID] = true
			switch n.Opcode {
			case scriptDropOpcode:
				if dialect == sim.ScriptROM2 {
					rep.PackedCoordinates = append(rep.PackedCoordinates, uint16(uint8(n.Value[0]))|uint16(uint8(n.Value[1]))<<8)
				} else {
					rep.DropCells = append(rep.DropCells, ScriptCell{X: args[0], Y: args[1]})
				}
			default:
				// The object-construction form and anything else above the
				// floor: consumed or discarded by the builder, never dispatched.
				rep.DiscardedActions = append(rep.DiscardedActions, n.Opcode)
			}
			continue
		}
		if dialect == sim.ScriptROM2 && refd.failed {
			buildTime[n.ID] = true
			rep.OmittedActions = append(rep.OmittedActions, n.ID)
			continue
		}
		instantOf[n.ID] = int32(len(instants))
		instants = append(instants, sim.ScriptInstant{
			Op: int32(n.Opcode), Args: args,
			Unit: refd.unit, HasUnit: refd.hasUnit,
			Unit2: refd.unit2, HasUnit2: refd.hasUnit2,
			Group: refd.group, HasGroup: refd.hasGroup,
			Player: refd.player, HasPlayer: refd.hasPlayer,
			Item: refd.item, HasItem: refd.hasItem,
			Structure: refd.structure, HasStructure: refd.hasStructure,
		})
	}

	// ---- pass 2: the conditions.
	checks := make([]sim.ScriptCheck, 0, len(src.Conditions))
	registerOf := make(map[uint32]int32, len(src.Conditions))
	omittedChecks := make(map[uint32]bool)
	for _, n := range src.Conditions {
		var args [10]int32
		var refd bound
		if dialect == sim.ScriptROM2 && n.Opcode == scriptConstOpcode {
			args[0] = int32(n.Value[0])
		} else {
			args, refd = bindScriptParams(n, refs, &rep, true, dialect)
		}
		if dialect == sim.ScriptROM2 && n.Opcode != scriptConstOpcode && refd.failed {
			rep.OmittedChecks = append(rep.OmittedChecks, n.ID)
			omittedChecks[n.ID] = true
			continue
		}
		reg := int32(len(checks))
		registerOf[n.ID] = reg
		checks = append(checks, sim.ScriptCheck{
			Op: int32(n.Opcode), Register: reg, Args: args,
			Unit: refd.unit, HasUnit: refd.hasUnit,
			Unit2: refd.unit2, HasUnit2: refd.hasUnit2,
			Group: refd.group, HasGroup: refd.hasGroup,
			Player: refd.player, HasPlayer: refd.hasPlayer,
			Player2: refd.player2, HasPlayer2: refd.hasPlayer2,
			// THE ITEM REFERENCE. bindParams has decoded it for a condition node
			// since 0156, on the same pass and through the same arm the action nodes
			// above use; this record dropped it until check opcode 17 needed it.
			// Nothing about the decode changes -- the same Target_Item value reaches
			// the same scriptItemCodeBase transform whichever node kind carries it.
			Item: refd.item, HasItem: refd.hasItem,
			// THE STRUCTURE REFERENCE (1033 B3), on the item reference's own
			// terms above: resolved by bindParams on the same pass.
			Structure: refd.structure, HasStructure: refd.hasStructure,
		})
	}

	// ---- pass 3: the triggers.
	triggers := make([]sim.ScriptTrigger, 0, len(src.Triggers))
	for i, t := range src.Triggers {
		omitted := false
		if dialect == sim.ScriptROM2 {
			for k := range t.Left {
				if omittedChecks[t.Left[k]] || omittedChecks[t.Right[k]] {
					omitted = true
				}
			}
		}
		if omitted {
			rep.OmittedTriggers = append(rep.OmittedTriggers, i)
			continue
		}
		// A trigger whose FIRST pair's left id is zero is dropped whole, before
		// anything of it is built. It is not a trigger with no condition — that
		// is a trigger whose first pair IS set and whose others are not.
		if t.Left[0] == 0 {
			rep.DroppedTriggers = append(rep.DroppedTriggers, i)
			continue
		}
		var out sim.ScriptTrigger
		out.Once = t.Once != 0
		// The latch is the trigger's position in the MAP's array, not in this
		// slice — a dropped trigger leaves its latch unused rather than shifting
		// every later one onto its neighbour's.
		out.Latch = int32(i)
		for k := 0; k < 3; k++ {
			// Only a pair with BOTH ids set is a pair. The corpus has no
			// half-empty one, and an unset slot is no comparison rather than a
			// comparison against register zero.
			if t.Left[k] == 0 || t.Right[k] == 0 {
				continue
			}
			out.Pairs[k] = sim.ScriptPair{
				Left:  lookupOr0(registerOf, t.Left[k]),
				Right: lookupOr0(registerOf, t.Right[k]),
				Cmp:   int32(t.Cmp[k]),
				Used:  true,
			}
		}
		for k := 0; k < 4; k++ {
			out.Instants[k] = sim.ScriptNone
			if t.Acts[k] == 0 {
				continue
			}
			// A consumed or discarded build-time record owns no runtime
			// instant and therefore no runtime trigger slot. Do not pass it
			// through the general missing-id rule: aliasing it to instant 0
			// executes an unrelated action when the trigger fires.
			if buildTime[t.Acts[k]] {
				continue
			}
			// An id that names no built instant resolves to subscript 0, which
			// is the id-to-subscript table's own miss value — unless nothing was
			// built at all, where subscript 0 names nothing and the slot is left
			// empty.
			if len(instants) == 0 {
				continue
			}
			// THE HIT IS TAKEN SEPARATELY, and only a hit contributes a raise.
			// lookupOr0 below reproduces the engine's miss-to-zero behaviour for
			// what the simulation runs; a MISS must not become an announcement,
			// because subscript 0 is then another action entirely. This is the
			// only place in the program where the two can still be told apart.
			if idx, hit := instantOf[t.Acts[k]]; hit && instants[idx].Op == scriptMessageOpcode {
				rep.Raises = append(rep.Raises, ScriptRaise{
					Latch: out.Latch,
					Event: instants[idx].Args[0],
				})
			}
			out.Instants[k] = lookupOr0(instantOf, t.Acts[k])
		}
		triggers = append(triggers, out)
	}

	compile := sim.NewScript
	if dialect == sim.ScriptROM2 {
		compile = sim.NewROM2Script
	}
	s, err := compile(checks, instants, triggers)
	if err != nil {
		return nil, rep, fmt.Errorf("mapload: compiling the map's script: %w", err)
	}
	return s, rep, nil
}

func bindScriptParams(n alm.ScriptNode, refs ScriptRefs, rep *ScriptReport, cond bool, dialect sim.ScriptDialect) ([10]int32, bound) {
	if dialect == sim.ScriptROM2 {
		for i, typ := range n.Type {
			if typ == typeUnset {
				n.Type[i] = typeInt
			}
		}
	}
	return bindParamsDialect(n, refs, rep, cond, dialect)
}

// lookupOr0 is the id-to-subscript table's own behaviour: a hit, or subscript 0.
func lookupOr0(tbl map[uint32]int32, id uint32) int32 {
	if v, ok := tbl[id]; ok {
		return v
	}
	return 0
}

// bound is the reference half of one node's parameter block.
type bound struct {
	failed            bool
	unit, unit2       sim.EntityID
	hasUnit, hasUnit2 bool
	seenUnit          int

	group    uint32
	hasGroup bool

	player, player2       uint32
	hasPlayer, hasPlayer2 bool
	seenPlayer            int

	item    uint16
	hasItem bool

	structure    sim.StructureID
	hasStructure bool
}

// scriptItemCodeBase is what the builder ADDS to a Target_Item value to get
// the packed item code the compiled record carries.
//
// The original's own trigger builder stores `0xe18 + V` into the record's item
// field as a WORD, and the two item arms then hand that field straight to the
// item factory. So a Target_Item value is an offset into the class-14 block and
// not an item code: value 6 becomes 0x0e1e, whose class nibble is 14 and whose
// index — for class 14 alone, the whole low byte — is 30.
//
// THE ADDITION BELONGS HERE AND NOT IN THE ARM. It is a fact about how a map
// file spells a reference, which is this package's business; pkg/sim reads
// item codes and holds nothing about the file the world was built from.
// Every authored value in the shipped corpus is 2..36, so every compiled
// code is class 14 and none of them is zero.
const scriptItemCodeBase uint32 = 0x0e18

// bindParams splits one node's ten parameter slots into the compiled record's two
// halves: the plain integers, PACKED IN ENCOUNTER ORDER, and the resolved unit
// references.
//
// A slot carrying no parameter is skipped by both halves — the nine type codes
// start at 1, so a zero is an unset slot and not a tenth kind.
//
// A node's Target_Unit parameters are resolved into a PAIR, not a single
// value: the first one encountered becomes b.unit/hasUnit and the second
// becomes b.unit2/hasUnit2, for every node this function binds — a
// condition and an action alike. bindParams takes no side on which of them a
// caller keeps: that is CompileScriptFrom's own choice, made once per pass,
// and since 0129 both passes keep the whole pair — the check literal has
// carried it from the start, and the instant literal now carries it too.
//
// Only the UNIT references are RESOLVED, and the group and player references are
// CARRIED — a different thing rather than a lesser one. A Target_Unit value names
// one of three id spaces and only the caller knows which entity any of them
// became; a Target_Group value names exactly one thing, the placed record's own
// group word, and a Target_Player value names exactly one thing too, the SAME
// 1-based roster slot space a placed record's owner field is in. Neither has
// anything to be looked up in and neither can fail.
//
// That the two are one id space is measured rather than assumed, and the rival
// reading was live: over every Target_Player parameter of both installed roots,
// 170/170 and 168/168 fall inside the type-5 roster's slot range, while reading
// them as a roster record's own id word fails 26 times on each. So an arm that
// hands a unit to a player ASSIGNS the word rather than translating it.
//
// An ITEM reference is carried too, and it is the one reference that is
// TRANSFORMED rather than copied: scriptItemCodeBase's own note.
//
// A STRUCTURE reference is resolved since 1033 B3, on the unit reference's
// own terms (only the first slot; no arm of the vocabulary names two): a
// Target_Structure value is looked up in refs.Structures, and a miss is
// reported through rep.Unresolved exactly as a miss on a Target_Unit value
// is.
func bindParams(n alm.ScriptNode, refs ScriptRefs, rep *ScriptReport, cond bool) ([10]int32, bound) {
	return bindParamsDialect(n, refs, rep, cond, sim.ScriptROM1)
}

func bindParamsDialect(n alm.ScriptNode, refs ScriptRefs, rep *ScriptReport, cond bool, dialect sim.ScriptDialect) ([10]int32, bound) {
	var args [10]int32
	var b bound
	next := 0
	for i := 0; i < 10; i++ {
		t := n.Type[i]
		v := n.Value[i]
		switch {
		case t == typeUnset:
			continue
		case isPlainParam(t):
			if next < len(args) {
				args[next] = int32(v)
				next++
			}
		case t == typeUnit:
			id, ok := resolveUnit(v, refs)
			if dialect == sim.ScriptROM2 {
				id, ok = resolveROM2Unit(v, refs)
			}
			if !ok {
				b.failed = true
				rep.Unresolved = append(rep.Unresolved, ScriptUnresolved{
					Condition: cond, Kind: ScriptRefUnit, NodeID: n.ID, Opcode: n.Opcode, Value: v})
			}
			switch b.seenUnit {
			case 0:
				b.unit, b.hasUnit = id, ok
			case 1:
				b.unit2, b.hasUnit2 = id, ok
			}
			b.seenUnit++
		case t == typeGroup:
			// CARRIED, not resolved, and only the first slot. A Target_Group
			// value names exactly one thing — the placed record's own group
			// word — so there is nothing to look it up in and nothing that can
			// fail; the presence flag is what says a group was named at all,
			// because group zero is a real group. No arm of the vocabulary
			// names two, and taking the first is what a node carrying a second
			// would get from a forward-reading builder.
			if !b.hasGroup {
				b.group, b.hasGroup = v, true
			}
		case t == typePlayer:
			// CARRIED like the group above and on the same terms — the roster
			// slot as the file wrote it, no table, nothing that can fail. Check
			// opcode 10 — "what one player thinks of another" — DOES name two,
			// so a third player parameter is what gets dropped rather than a
			// second: the same shape seenUnit already gives the two unit
			// references above, taken here for the two player references.
			//
			// The presence flags are what say a player was named at all. A slot
			// is 1-based so zero already names nobody, but the arms that read
			// these treat "no player parameter" as WRITE NOTHING rather than as
			// write zero, and only a flag keeps those apart — for the first
			// player and, since check 10, for the second as well.
			switch b.seenPlayer {
			case 0:
				b.player, b.hasPlayer = v, true
			case 1:
				b.player2, b.hasPlayer2 = v, true
			}
			b.seenPlayer++
		case t == typeItem:
			// CARRIED like the group and the players above, and on the same terms —
			// there is no table to look a Target_Item value up in and nothing that
			// can fail — except that it is TRANSFORMED on the way: the builder's
			// own store is `0xe18 + V` as a word, so what the compiled record holds
			// is a packed item code and not the file's own value. Only the first slot
			// is taken, which is what a node carrying a second would get from a
			// forward-reading builder; no arm of the vocabulary names two.
			//
			// The presence flag is what says an item was named at all. A code of zero
			// already names nothing — class is bits 8..11 and class zero resolves
			// to nothing — so the flag could have been folded away, and it is a
			// byte like its neighbours for the reason the player flag is.
			if !b.hasItem {
				b.item, b.hasItem = uint16(scriptItemCodeBase+v), true
			}
		case t == typeStructure:
			// RESOLVED since 1033 B3, on the unit reference's own terms: only
			// the first slot is taken, and a miss is reported. The presence
			// flag is what says a structure was named at all — StructureID
			// zero is a real structure, the entity id's own reason.
			if !b.hasStructure {
				id, ok := refs.Structures[uint16(v)]
				if !ok {
					b.failed = true
					rep.Unresolved = append(rep.Unresolved, ScriptUnresolved{
						Condition: cond, Kind: ScriptRefStructure, NodeID: n.ID, Opcode: n.Opcode, Value: v})
				}
				b.structure, b.hasStructure = id, ok
			}
		}
	}
	return args, b
}

func resolveROM2Unit(v uint32, refs ScriptRefs) (sim.EntityID, bool) {
	if v >= 8000 && v <= 9999 {
		return 0, false
	}
	if v <= 65535 {
		if id, ok := refs.Units[uint16(v)]; ok {
			return id, true
		}
	}
	if v == 10001 && refs.HasHero {
		return refs.Hero, true
	}
	if v == 10002 && refs.HasCompanion {
		return refs.Companion, true
	}
	id, ok := refs.Roles[v]
	return id, ok
}

// resolveUnit resolves one Target_Unit value through its three id bands.
func resolveUnit(v uint32, refs ScriptRefs) (sim.EntityID, bool) {
	switch {
	case v < scriptHeroBandLow:
		id, ok := refs.Units[uint16(v)]
		return id, ok
	case v <= scriptHeroBandHigh:
		if v == scriptHeroBandLow && refs.HasHero {
			return refs.Hero, true
		}
		if v == scriptHeroBandLow+1 && refs.HasCompanion {
			return refs.Companion, true
		}
		id, ok := refs.Roles[v]
		return id, ok
	default:
		// A static name table this tree does not carry. Not one shipped map's
		// reference falls here — 196 of the corpus's are hero-band and 0 are
		// name-band — so it is reported rather than guessed at.
		return 0, false
	}
}

package main

// The `script` verb: what a map's authored mission script decodes to, what it
// compiles to, and — the part a consumer most needs — every arm of it this build
// cannot evaluate.
//
// It is a developer verb like the rest of almtool: it reads one map from disk,
// prints measurements, writes nothing and embeds no install path.

import (
	"fmt"
	"strings"

	"againrom/pkg/mapload"
)

// groupOrderOpcode is pkg/sim's ScriptInstantGroupOrder, 6, spelled as a
// local literal rather than an import: cmd/almtool's own census stops at
// the tier pkg/mapload sits on (internal/archtest's DAG refuses
// cmd/almtool -> pkg/sim), so the one number this file needs to recognise
// the group command's own gaps is named here instead — the same trade
// pkg/mapload's own scriptMessageOpcode already makes for a different
// opcode, for the same reason.
const groupOrderOpcode int32 = 6

// The opcodes the trigger listing names, spelled here for the same reason and on
// the same terms as groupOrderOpcode above: this file may not import pkg/sim.
//
// FOUR OF THEM ARE THE ONES THAT DECIDE A MISSION or take a unit off the map:
// pkg/sim's ScriptInstantWin, ScriptInstantLose, ScriptInstantTakeOffMap and
// ScriptInstantGroupOffMap. They are named and the other 30 instant opcodes are
// not, deliberately. A half-filled name table is worse than none — a reader
// takes an unnamed arm for an unimportant one — and the question this listing
// exists to answer is whether a chain reaches an outcome, not what every arm of
// it is called. The rest print as bare opcode numbers, which is what the gap
// tallies above already do.
//
// checkConstant is ScriptCheckConstant, which is not an arm at all: a check
// carrying it is a build-time preset whose register takes the node's first value
// once and is never written again. A pair comparing against one is comparing
// against a constant, and a reader who cannot see that reads a permanently false
// pair as a live one.
const (
	instantWin         int32 = 4
	instantLose        int32 = 5
	instantTakeOffMap  int32 = 16
	instantGroupOffMap int32 = 32

	checkConstant int32 = 0x10002

	// scriptNone is pkg/sim's ScriptNone: the value an unused instant slot
	// carries.
	scriptNone int32 = -1

	// The two Target_Unit band edges, read from pkg/mapload's own
	// scriptHeroBandLow/High and spelled here on the same terms as the opcodes
	// above. A value below the low edge is a placed record's own unit id; the
	// band itself is an ordinal into the live player list; above the high edge
	// is a static name table this tree does not carry. The listing needs them to
	// say why a reference did not resolve, which is four facts and not one.
	heroBandLow  = 10001
	heroBandHigh = 11000
)

func cmdScript(path string) error {
	m, _, err := loadMap(path)
	if err != nil {
		return err
	}

	src, err := m.Script()
	if err != nil {
		return err
	}
	fmt.Printf("map:        %q  %dx%d  version=%d\n", m.Name, m.Width, m.Height, m.FormatVersion)
	fmt.Printf("type7:      present=%v  entryCount=%d  body=%d bytes\n",
		m.Present(7), m.Triggers.EntryCount, len(m.Triggers.Body))
	if src.Empty() {
		fmt.Println("script:     none — this map authors no mission script")
		return nil
	}
	fmt.Printf("decoded:    %d action(s), %d condition(s), %d trigger(s)\n",
		len(src.Actions), len(src.Conditions), len(src.Triggers))

	// The unit table is the map's own; the hero is not on the map at all — a
	// campaign map places nobody for the player — so a hero-band reference is
	// reported unresolved here rather than pointed at an invented entity. That
	// is what this verb is for: the tool says what it cannot resolve instead of
	// resolving it wrongly.
	refs := mapload.ScriptRefs{Units: mapload.ScriptUnits(m, nil), Structures: mapload.ScriptStructures(m)}
	s, rep, err := mapload.CompileScript(m, refs)
	if err != nil {
		return err
	}
	fmt.Printf("compiled:   %d check(s), %d instant(s), %d trigger(s)\n",
		len(s.Checks()), len(s.Instants()), len(s.Triggers()))

	fmt.Println()
	fmt.Println("-- the binder --")
	for _, c := range rep.DropCells {
		fmt.Printf("  drop location   (%d,%d)\n", c.X, c.Y)
	}
	for _, i := range rep.DroppedTriggers {
		fmt.Printf("  trigger dropped at map position %d (its first pair names no left condition)\n", i)
	}
	for _, op := range rep.DiscardedActions {
		fmt.Printf("  build-time action opcode %#x discarded\n", op)
	}
	fmt.Printf("  unresolved references: %d\n", len(rep.Unresolved))
	for _, u := range rep.Unresolved {
		kind := "action"
		if u.Condition {
			kind = "condition"
		}
		fmt.Printf("    %s node id=%d opcode=%d names %s %d (%s)\n",
			kind, u.NodeID, u.Opcode, u.Kind, u.Value, refBand(u.Kind, u.Value))
	}

	fmt.Println()
	fmt.Println("-- what this build cannot evaluate --")
	gaps := s.Unsupported()
	if len(gaps) == 0 {
		fmt.Println("  nothing: every arm this map authors is one this build evaluates")
	}
	// Counted per opcode so a map raising one arm two hundred times reads as
	// one line and not as two hundred. Opcode 6's own gaps are counted PER
	// SUB-COMMAND rather than folded into one instant-6 tally: the report is
	// meant to separate the sub-commands this build runs from those it does not
	// (AC-10), and a single count for the opcode would erase exactly that
	// distinction.
	checkCount := map[int32]int{}
	instantCount := map[int32]int{}
	groupCmdCount := map[int32]int{}
	for _, g := range gaps {
		switch {
		case g.Kind == 0:
			checkCount[g.Op]++
		case g.Op == groupOrderOpcode:
			groupCmdCount[g.Sub]++
		default:
			instantCount[g.Op]++
		}
	}
	printArms("check", checkCount)
	printArms("instant", instantCount)
	printGroupCmdArms(groupCmdCount)

	inert := s.InertTriggers()
	fmt.Printf("  inert triggers: %d of %d — these read a register no arm of this build writes and "+
		"can never fire\n", len(inert), len(s.Triggers()))
	for _, i := range inert {
		fmt.Printf("    compiled trigger %d (map position %d)\n", i, s.Triggers()[i].Latch)
	}

	// EVERY TRIGGER'S CONDITION PAIRS AND ITS INSTANT CHAIN, and the check that
	// writes each register a pair reads.
	//
	// WHY A LISTING AND NOT A COUNT. The tallies above answer "can this trigger
	// fire". They do not answer "and then what", and the two are different
	// questions with different consequences: a trigger that becomes able to
	// fire and then wins the mission changes a shipped map's outcome, and one
	// that becomes able to fire and then prints a message does not. This is
	// that program, committed.
	//
	// A REGISTER WITH NO WRITER IS THE POINT, not an edge case. A pair reading a
	// register no check writes compares a value that is zero for the whole
	// mission, so a pair against a non-zero constant is permanently false and its
	// trigger is dead however live the tallies make it look. The listing says so
	// rather than leaving the register unexplained, because that sentence is the
	// difference between a trigger a story armed and one that merely stopped
	// being reported as inert.
	//
	// It prints every trigger, live and inert alike. An inert trigger's chain is
	// what would run if the missing arm were built, which is what a story
	// planning that arm needs to read.
	//
	// IT IS WRITTEN INLINE rather than as functions taking the compiled types.
	// internal/archtest's DAG refuses cmd/almtool -> pkg/sim, so no signature in
	// this file may name ScriptTrigger, ScriptCheck or ScriptInstant; the values
	// are reachable through mapload.CompileScript's return without naming their
	// types, and closures keep it that way. The three helpers below take plain
	// int32 for the same reason.
	triggers, allChecks, allInstants := s.Triggers(), s.Checks(), s.Instants()

	fmt.Println()
	fmt.Println("-- the triggers --")
	fmt.Println("  Compiled with no party, so the map's own unit table resolves and nothing else.")
	fmt.Println("  A hero-band reference is therefore unresolved in this run by construction.")
	if len(triggers) == 0 {
		fmt.Println("  none")
		return nil
	}

	// Which check writes each register. A register may be written by more than
	// one check and every writer is printed, because two checks sharing a
	// subscript is a fact about the map rather than a defect in the report.
	writers := map[int32][]int{}
	for i, c := range allChecks {
		writers[c.Register] = append(writers[c.Register], i)
	}
	badCheck, badInstant := map[int32]bool{}, map[int32]bool{}
	for _, g := range gaps {
		if g.Kind == 0 {
			badCheck[g.Index] = true
		} else {
			badInstant[g.Index] = true
		}
	}

	// THE TWO NUMBER SPACES. A compiled check's subscript is its REGISTER
	// subscript: pass 2 takes reg := len(checks) before appending. It is not the
	// authored node id the binder report above prints, and joining the two by
	// value reads one numbering's fact off another's. Pass 2 walks
	// src.Conditions in order with no skips, so subscript i is condition i and
	// that condition's own ID is the authored number. The listing prints both,
	// because the binder report is the only place the raw reference value is.
	nodeID := func(ci int) uint32 {
		if ci >= 0 && ci < len(src.Conditions) {
			return src.Conditions[ci].ID
		}
		return 0
	}

	// The RAW reference value and which id space it was read from, which the
	// compiled record does not keep and the binder report does. It separates a
	// dangling map reference from a hero-band ordinal, and a unit miss from a
	// structure miss — three different findings and not one.
	type unresolvedRef struct {
		kind  mapload.ScriptRefKind
		value uint32
	}
	unresolvedAt := map[uint32][]unresolvedRef{}
	for _, u := range rep.Unresolved {
		if u.Condition {
			unresolvedAt[u.NodeID] = append(unresolvedAt[u.NodeID], unresolvedRef{u.Kind, u.Value})
		}
	}

	// refNote says WHY a check's unit and structure references are absent or
	// present, and it asks the BINDER rather than an opcode table. An earlier
	// draft of this listing kept its own list of which check opcodes take a
	// unit and reported every other HasUnit == false as an unresolved
	// reference. That list was wrong for check opcodes 15 and 19 on the first
	// map it was run against: both left HasUnit false while the binder
	// reported no failure, because neither node carried a unit parameter at
	// all. The binder already knows -- it appends to ScriptReport.Unresolved
	// for exactly the references that failed, tagged with which id space each
	// one is -- so the cases separate with no opcode knowledge.
	//
	// IT MUST ALSO READ HasStructure, not only HasUnit. A first version of this
	// fix read HasUnit alone, which is what 1033's follow-up W-1 found: every
	// check-21 node's structure reference resolves and none of it reads here,
	// so every such node printed "names no unit" — true and beside the point.
	//
	// A FAILURE IS NOT ONE FACT. A unit value below the hero band is a
	// dangling map reference and is the map's own defect. A unit value in the
	// band is an ordinal into the live player list: ordinal 1 resolves
	// whenever the caller supplies a party, and THIS VERB SUPPLIES NONE, so it
	// fails here by construction and says nothing about the game. A higher
	// ordinal names nobody in a one-player build even with a party. Story
	// 1029's adversarial pass, which compiled with a party, and this tool,
	// which does not, disagreed about exactly one check for exactly this
	// reason. A structure value has one shape only: the map does not place a
	// structure with that word (refFailureNote's own comment).
	refNote := func(ci int) string {
		c := allChecks[ci]
		fails := unresolvedAt[nodeID(ci)]
		if len(fails) == 0 {
			resolved := make([]string, 0, 2)
			if c.HasUnit {
				if c.HasUnit2 {
					resolved = append(resolved, "both unit references resolved")
				} else {
					resolved = append(resolved, "unit resolved")
				}
			}
			if c.HasStructure {
				resolved = append(resolved, "structure resolved")
			}
			if len(resolved) == 0 {
				return "names no unit or structure"
			}
			return strings.Join(resolved, "; ")
		}
		parts := make([]string, 0, len(fails)+2)
		if c.HasUnit {
			parts = append(parts, "one unit reference resolved")
		}
		if c.HasStructure {
			parts = append(parts, "one structure reference resolved")
		}
		for _, f := range fails {
			parts = append(parts, refFailureNote(f.kind, f.value))
		}
		return "unresolved: " + strings.Join(parts, "; ")
	}

	printRegister := func(r int32) {
		w := writers[r]
		if len(w) == 0 {
			// NO CHECK WRITES IT, which is not the same as "it stays zero". A
			// variable-writing instant carries its register in Args and which arg
			// that is depends on the opcode, so this listing cannot scan for one
			// without keeping an opcode table -- the thing that was just wrong about
			// unit parameters. The line says what was measured and names what was
			// not, and a reader who needs the stronger claim reads the instants.
			fmt.Printf("      register %d: written by no check; the instants are not scanned for a writer\n", r)
			return
		}
		for _, ci := range w {
			c := allChecks[ci]
			if c.Op == checkConstant {
				fmt.Printf("      register %d <- check %d (authored node %d), a build-time constant of %d\n",
					r, ci, nodeID(ci), c.Args[0])
				continue
			}
			gap := ""
			if badCheck[int32(ci)] {
				gap = " [not implemented]"
			}
			fmt.Printf("      register %d <- check %d (authored node %d), opcode %d, %s%s\n",
				r, ci, nodeID(ci), c.Op, refNote(ci), gap)
		}
	}

	for i, t := range triggers {
		state := "live"
		if t.Inert {
			state = "INERT"
		}
		once := "repeats"
		if t.Once {
			once = "fires once"
		}
		fmt.Printf("  trigger %d (map position %d, %s, %s)\n", i, t.Latch, once, state)

		used := 0
		for p, pr := range t.Pairs {
			if !pr.Used {
				continue
			}
			used++
			fmt.Printf("    pair %d: register %d %s register %d\n", p, pr.Left, cmpName(pr.Cmp), pr.Right)
			printRegister(pr.Left)
			printRegister(pr.Right)
		}
		if used == 0 {
			fmt.Println("    no condition pair: nothing gates this trigger")
		}

		ran := 0
		for _, n := range t.Instants {
			if n == scriptNone {
				continue
			}
			ran++
			if int(n) >= len(allInstants) {
				fmt.Printf("    instant node %d: outside the instant array\n", n)
				continue
			}
			note := instantNote(allInstants[n].Op)
			if badInstant[n] {
				note += " [not implemented]"
			}
			fmt.Printf("    instant node %d: opcode %d%s\n", n, allInstants[n].Op, note)
		}
		if ran == 0 {
			fmt.Println("    no instant: this trigger runs nothing")
		}
	}
	return nil
}

// cmpName spells one of the six comparison codes. A code outside the alphabet is
// printed as itself: the engine takes the out-of-range arm every pass, so such a
// pair is permanently false and the number is what a reader needs to see.
func cmpName(code int32) string {
	switch code {
	case 0:
		return "=="
	case 1:
		return "!="
	case 2:
		return ">"
	case 3:
		return "<"
	case 4:
		return ">="
	case 5:
		return "<="
	}
	return fmt.Sprintf("(comparison code %d, outside the six-entry table — permanently false)", code)
}

// instantNote names the four opcodes that end a mission or take a unit off the
// map, and nothing else. The const block above carries why.
func instantNote(op int32) string {
	switch op {
	case instantWin:
		return " (WIN)"
	case instantLose:
		return " (LOSE)"
	case instantTakeOffMap:
		return " (takes a unit off the map)"
	case instantGroupOffMap:
		return " (takes a group off the map)"
	}
	return ""
}

// printArms prints one unsupported-arm tally, in ascending opcode order so two
// runs over one map read alike.
func printArms(what string, count map[int32]int) {
	if len(count) == 0 {
		return
	}
	ops := make([]int32, 0, len(count))
	for op := range count {
		ops = append(ops, op)
	}
	// A small insertion sort: the vocabularies are 22 and 34 arms.
	for i := 1; i < len(ops); i++ {
		for j := i; j > 0 && ops[j] < ops[j-1]; j-- {
			ops[j], ops[j-1] = ops[j-1], ops[j]
		}
	}
	for _, op := range ops {
		fmt.Printf("  %s arm %d: %d node(s) not implemented\n", what, op, count[op])
	}
}

// printGroupCmdArms is printArms' own shape for opcode 6's own sub-dispatch:
// one line per unimplemented sub-command rather than one line for the whole
// opcode, which is what would hide which of the ten a map actually authors
// and this build does not run.
func printGroupCmdArms(count map[int32]int) {
	if len(count) == 0 {
		return
	}
	subs := make([]int32, 0, len(count))
	for sub := range count {
		subs = append(subs, sub)
	}
	for i := 1; i < len(subs); i++ {
		for j := i; j > 0 && subs[j] < subs[j-1]; j-- {
			subs[j], subs[j-1] = subs[j-1], subs[j]
		}
	}
	for _, sub := range subs {
		fmt.Printf("  instant arm 6 sub-command %d (group command): %d node(s) not implemented\n",
			sub, count[sub])
	}
}

// unitBand names which of the three Target_Unit id spaces a value falls in, so
// an unresolved reference says WHY it did not resolve.
func unitBand(v uint32) string {
	switch {
	case v < 10001:
		return "a placed unit this map does not carry"
	case v <= 11000:
		return "a hero ordinal — the party is not on the map"
	default:
		return "the static name table, which this tree does not carry"
	}
}

// unitFailureNote spells one failed Target_Unit reference by the BAND its raw
// value falls in. The three bands are pkg/mapload resolveUnit's own, and they
// are three different findings:
//
// Below the low edge the value is a placed record's own unit id, and a failure
// means the map names a unit it does not place. That is the map's defect and
// the only one of the three that is.
//
// Inside the band the value is an ordinal into the live player list. Ordinal 1
// resolves whenever the caller supplies a party. THE script VERB SUPPLIES NONE,
// so ordinal 1 fails here by construction and the note says so rather than
// reporting the tool's own configuration as a property of the map. A higher
// ordinal names nobody in a one-player build even with a party, which is a
// standing limitation of this tree and not of the run.
//
// Above the high edge is a static name table this tree does not carry. No
// shipped map's reference falls there, per CompileScript's own comment, so the
// arm exists to keep the note honest if one ever does.
func unitFailureNote(v uint32) string {
	switch {
	case v < heroBandLow:
		return fmt.Sprintf("map unit %d is not in this map's unit table", v)
	case v == heroBandLow:
		return "hero ordinal 1, which resolves when the caller supplies a party and did not here"
	case v <= heroBandHigh:
		return fmt.Sprintf("hero ordinal %d, which names nobody in a one-player build even with a party", v-heroBandLow+1)
	}
	return fmt.Sprintf("name-table id %d, a table this tree does not carry", v)
}

// structureBand and structureFailureNote are unitBand and unitFailureNote's
// own shape for a Target_Structure reference. There is exactly one id space
// (ScriptRefs' own comment: "no hero band and no static table for this
// reference kind"), so unlike a unit reference a structure failure has one
// shape and not three.
func structureBand(uint32) string {
	return "a structure this map does not place"
}

func structureFailureNote(v uint32) string {
	return fmt.Sprintf("structure %d is not in this map's structure table", v)
}

// refBand and refFailureNote dispatch on a ScriptUnresolved's own Kind rather
// than assuming unit vocabulary. Both listings that render a ScriptUnresolved
// value call these, and neither holds its own copy of which kind means what —
// the same drift 1033's follow-up W-1 found where one of the two read the
// value in unit vocabulary unconditionally.
func refBand(kind mapload.ScriptRefKind, v uint32) string {
	if kind == mapload.ScriptRefStructure {
		return structureBand(v)
	}
	return unitBand(v)
}

func refFailureNote(kind mapload.ScriptRefKind, v uint32) string {
	if kind == mapload.ScriptRefStructure {
		return structureFailureNote(v)
	}
	return unitFailureNote(v)
}

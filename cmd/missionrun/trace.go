package main

// THE SCRIPT TRACE READOUT — the mission script's own account of itself.
//
// pkg/sim records what the script did on a tick; this file is the only thing
// that turns that record into the map's own terms. A pair names two register
// subscripts, a register is written by a check, and a check names a unit by an
// entity id the binder assigned — so a raw trace reads as three levels of
// indirection away from anything a mission author wrote. Every line below
// collapses those three: the register's owning check, that check's arm, and the
// number the map's own script calls the unit.
//
// IT PRINTS ONLY WHAT CHANGED. A pass's firings and its VIP losses are events
// and are printed every time; a silent check is a STANDING condition — the same
// check writes nothing on every pass for the whole mission — so it is printed
// when it first appears and again only if its reason changes. Printing it per
// pass would bury the events under it at a hundred and sixty lines a mission.

import (
	"fmt"
	"io"

	"againrom/pkg/sim"
)

// tracer is the stepping seam: every advance in this tool goes through it, and
// it either steps plainly or steps observed and prints what it saw.
//
// It holds the naming tables the readout needs — the map's own unit numbers, and
// the compiled script — because those are fixed for a mission and looking them
// up per line is how a formatter ends up re-deriving the world.
type tracer struct {
	on     bool
	out    io.Writer
	names  map[sim.EntityID]uint16
	script *sim.Script
	silent map[int32]sim.ScriptSilence

	// The counters as the last printed report read them. A report that reads
	// what the last one read says nothing new; a report that reads MORE with the
	// mission still undecided says the most surprising thing this runtime does —
	// the reporter tests for exactly one, so two losses in a pass are swallowed.
	repWon, repLost uint32

	// cen samples the world after every advance, when a census was asked for.
	// It hangs here and not on the caller because EVERY advance in this tool
	// goes through step, and a census that misses one under-counts silently —
	// which is the one failure a movement count must not have. Nil until
	// watch is called, and nil-safe.
	cen *census
}

// watch attaches a census to every advance this tracer makes.
func (t *tracer) watch(c *census) { t.cen = c }

// newTracer builds the readout for one mission. names is inverted from the map's
// own script-unit table, so a compiled entity id can be printed as the number
// the mission's triggers are written against.
func newTracer(on bool, out io.Writer, script map[uint16]sim.EntityID, s *sim.Script) *tracer {
	t := &tracer{on: on, out: out, script: s,
		names: make(map[sim.EntityID]uint16, len(script)), silent: map[int32]sim.ScriptSilence{}}
	for n, id := range script {
		t.names[id] = n
	}
	return t
}

// step advances the world once. WITH TRACING OFF IT CALLS Step, not StepTraced
// with the result dropped: the tool must be able to state that a plain run took
// the plain path.
func (t *tracer) step(w *sim.World, cmds []sim.Command) {
	if !t.on {
		sim.Step(w, cmds)
		t.cen.sample(w)
		return
	}
	tr := sim.StepTraced(w, cmds)
	t.report(w, &tr)
	t.cen.sample(w)
}

// preamble is what the compiled script says before a tick has run: the arms this
// build does not have, and the triggers they take down with them.
//
// It is printed once because it cannot change — inertness is derived when the
// script is compiled — and it is printed FIRST because it is the only thing that
// separates "this trigger never fired" from "this trigger could not".
func (t *tracer) preamble() {
	if !t.on || t.script == nil {
		return
	}
	fmt.Fprintf(t.out, "script  %d checks, %d instants, %d triggers\n",
		len(t.script.Checks()), len(t.script.Instants()), len(t.script.Triggers()))
	for _, g := range t.script.Unsupported() {
		kind, name := "check", checkName(g.Op)
		if g.Kind == sim.ScriptGapInstant {
			kind, name = "instant", instantName(g.Op)
			if g.Op == sim.ScriptInstantGroupOrder {
				name = fmt.Sprintf("%s sub-command %d", name, g.Sub)
			}
		}
		fmt.Fprintf(t.out, "script  UNSUPPORTED %s %d: %s\n", kind, g.Index, name)
	}
	if inert := t.script.InertTriggers(); len(inert) > 0 {
		fmt.Fprintf(t.out, "script  INERT triggers %v — they read a register nothing writes\n", inert)
	}
}

// report prints one traced step. An empty trace came off one of the fifteen
// ticks in sixteen on which no script phase runs.
func (t *tracer) report(w *sim.World, tr *sim.ScriptTrace) {
	if tr.Empty() {
		return
	}
	for _, s := range tr.Silent {
		if was, seen := t.silent[s.Check]; seen && was == s.Why {
			continue
		}
		t.silent[s.Check] = s.Why
		fmt.Fprintf(t.out, "tick %d  check %d %s WROTE NOTHING into r%d: %s\n",
			tr.Tick, s.Check, t.checkText(s.Check), s.Register, silenceText(s.Why))
	}
	for _, f := range tr.Firings {
		once := ""
		if f.Once {
			once = ", once"
		}
		fmt.Fprintf(t.out, "tick %d  trigger %d FIRED (map latch %d%s)\n", tr.Tick, f.Trigger, f.Latch, once)
		for k, p := range f.Pairs {
			if !p.Pair.Used {
				continue
			}
			fmt.Fprintf(t.out, "          pair %d  %s %s %s  ->  %d %s %d  = %v\n",
				k, t.regText(p.Pair.Left), cmpName(p.Pair.Cmp), t.regText(p.Pair.Right),
				p.Left, cmpName(p.Pair.Cmp), p.Right, cmpHolds(p.Pair.Cmp, p.Left, p.Right))
		}
		for _, in := range f.Instants {
			mark := ""
			if !in.Supported {
				mark = "  [THIS BUILD DOES NOT RUN IT]"
			}
			fmt.Fprintf(t.out, "          slot %d  instant %d %s%s\n",
				in.Slot, in.Instant, instantName(in.Op), mark)
		}
	}
	for _, v := range tr.VIP {
		// The health is printed because "not alive" covers two states here and
		// they are two different next questions: a unit at exactly zero is DOWNED
		// and a further blow would finish it, one below zero was killed outright.
		// A script counts the loss at the first of the two.
		fmt.Fprintf(t.out, "tick %d  check %d %s COUNTED A LOSS: its unit is not alive (%s)\n",
			tr.Tick, v.Check, t.checkText(v.Check), hpText(w, v.Unit))
	}
	if tr.Pass && (len(tr.Firings) > 0 || len(tr.VIP) > 0) {
		fmt.Fprintf(t.out, "tick %d  counters won=%d lost=%d\n", tr.Tick, tr.Won, tr.Lost)
	}
	if tr.Report && (tr.Outcome != sim.OutcomeUndecided || tr.Won != t.repWon || tr.Lost != t.repLost) {
		t.repWon, t.repLost = tr.Won, tr.Lost
		fmt.Fprintf(t.out, "tick %d  REPORT won=%d lost=%d -> %s\n",
			tr.Tick, tr.Won, tr.Lost, outcomeName(tr.Outcome))
	}
}

// regText is a register as a reader wants it: its subscript and the check that
// writes it. A register no check owns is an authored mission variable and says
// so rather than being left as a bare number.
func (t *tracer) regText(r int32) string {
	if t.script == nil {
		return fmt.Sprintf("r%d", r)
	}
	if c, i, ok := t.script.RegisterOwner(r); ok {
		return fmt.Sprintf("r%d[check %d %s]", r, i, t.checkDesc(c))
	}
	return fmt.Sprintf("r%d[variable]", r)
}

// checkText names a check by its subscript in the compiled slice.
func (t *tracer) checkText(i int32) string {
	if t.script == nil {
		return ""
	}
	cs := t.script.Checks()
	if i < 0 || int(i) >= len(cs) {
		return ""
	}
	return t.checkDesc(cs[i])
}

// checkDesc is one check written out in the terms the map authored it in: the
// arm, the unit as the map's own script numbers it, and the arm's own
// parameters. Only the parameters an arm READS are printed — a node carries ten
// slots and eight of them are noise for any given arm.
func (t *tracer) checkDesc(c sim.ScriptCheck) string {
	u := t.unitText(c.Unit, c.HasUnit)
	switch c.Op {
	case sim.ScriptCheckGroupCount:
		if !c.HasGroup {
			return "groupcount(NO GROUP)"
		}
		return fmt.Sprintf("groupcount(group %d)", c.Group)
	case sim.ScriptCheckInBox:
		return fmt.Sprintf("inbox(%s in %d,%d..%d,%d)", u, c.Args[0], c.Args[1], c.Args[2], c.Args[3])
	case sim.ScriptCheckWithin:
		return fmt.Sprintf("within(%s of %d,%d by %d)", u, c.Args[0], c.Args[1], c.Args[2])
	case sim.ScriptCheckAlive:
		return fmt.Sprintf("alive(%s)", u)
	case sim.ScriptCheckUnitDistance:
		return fmt.Sprintf("unitdist(%s,%s)", u, t.unitText(c.Unit2, c.HasUnit2))
	case sim.ScriptCheckDistance:
		return fmt.Sprintf("dist(%s to %d,%d)", u, c.Args[0], c.Args[1])
	case sim.ScriptCheckVIP:
		return fmt.Sprintf("vip(%s)", u)
	case sim.ScriptCheckVariable:
		return fmt.Sprintf("var(r%d)", c.Args[0])
	case sim.ScriptCheckConstant:
		return fmt.Sprintf("const(%d)", c.Args[0])
	case sim.ScriptCheckStructField:
		return fmt.Sprintf("structfield(%s)", structText(c.Structure, c.HasStructure))
	}
	return fmt.Sprintf("%s(%s)", checkName(c.Op), u)
}

// structText is a compiled STRUCTURE reference. It exists because the fallback
// arm of checkDesc prints a unit reference, and a check-21 node names no unit:
// its one reference is a structure. Printed through unitText the node read
// `structfield(NO UNIT)`, which states a true fact about units and reads as a
// dangling reference.
//
// A structure id is the record index the map placed, which is the number
// cmd/classdump `-databin` counts and orders its own report by, so the two
// instruments name the same thing.
func structText(id sim.StructureID, has bool) string {
	if !has {
		return "NO STRUCTURE"
	}
	return fmt.Sprintf("structure %d", uint32(id))
}

// unitText is a compiled entity reference in the MAP'S OWN NUMBERING where the
// map has one. A reference that never resolved, and an entity the map's script
// never named — a party member is the ordinary case — say so instead of being
// printed as an entity id that matches nothing a reader can look up.
func (t *tracer) unitText(id sim.EntityID, has bool) string {
	if !has {
		return "NO UNIT"
	}
	if n, ok := t.names[id]; ok {
		return fmt.Sprintf("u%d", n)
	}
	return fmt.Sprintf("entity %d", uint32(id))
}

// checkName and instantName are the arms this build knows by name. An opcode
// outside either table prints as its number: the two supported tables in pkg/sim
// are the only place "does this build have this arm" is answered, and a name
// table that guessed would be a second answer to it.
func checkName(op int32) string {
	switch op {
	case sim.ScriptCheckGroupCount:
		return "groupcount"
	case sim.ScriptCheckInBox:
		return "inbox"
	case sim.ScriptCheckWithin:
		return "within"
	case sim.ScriptCheckHealth:
		return "health"
	case sim.ScriptCheckAlive:
		return "alive"
	case sim.ScriptCheckUnitDistance:
		return "unitdist"
	case sim.ScriptCheckDistance:
		return "dist"
	case sim.ScriptCheckTargetID:
		return "targetid"
	case sim.ScriptCheckItemDistance:
		return "itemdist"
	case sim.ScriptCheckItemTestAlias, sim.ScriptCheckItemTest:
		return "itemtest"
	case sim.ScriptCheckVIP:
		return "vip"
	case sim.ScriptCheckVariable:
		return "var"
	case sim.ScriptCheckStructField:
		return "structfield"
	case sim.ScriptCheckConstant:
		return "const"
	}
	return fmt.Sprintf("check op %d", op)
}

func instantName(op int32) string {
	switch op {
	case sim.ScriptInstantMessage:
		return "message"
	case sim.ScriptInstantSetVariable:
		return "setvar"
	case sim.ScriptInstantWin:
		return "WIN"
	case sim.ScriptInstantLose:
		return "LOSE"
	case sim.ScriptInstantIncVariable:
		return "incvar"
	case sim.ScriptInstantGiveUnit:
		return "giveunit"
	case sim.ScriptInstantGiveGroup:
		return "givegroup"
	case sim.ScriptInstantGroupOrder:
		return "groupcmd"
	case sim.ScriptInstantAddItem:
		return "additem"
	case sim.ScriptInstantTakeItem:
		return "takeitem"
	case sim.ScriptInstantStructField:
		return "structfield"
	}
	return fmt.Sprintf("instant op %d", op)
}

// cmpName is the comparison as it reads. A code outside the six-entry alphabet
// is named as the permanently-false arm it is rather than as an unknown, because
// the shipped corpus contains one.
func cmpName(code int32) string {
	switch code {
	case sim.ScriptCmpEQ:
		return "=="
	case sim.ScriptCmpNE:
		return "!="
	case sim.ScriptCmpGT:
		return ">"
	case sim.ScriptCmpLT:
		return "<"
	case sim.ScriptCmpGE:
		return ">="
	case sim.ScriptCmpLE:
		return "<="
	}
	return fmt.Sprintf("cmp %d (NEVER TRUE)", code)
}

// cmpHolds re-states the comparison the simulation made, for the line that
// prints it. It is a readout of two values a trace already carries and decides
// nothing.
func cmpHolds(code, a, b int32) bool {
	switch code {
	case sim.ScriptCmpEQ:
		return a == b
	case sim.ScriptCmpNE:
		return a != b
	case sim.ScriptCmpGT:
		return a > b
	case sim.ScriptCmpLT:
		return a < b
	case sim.ScriptCmpGE:
		return a >= b
	case sim.ScriptCmpLE:
		return a <= b
	}
	return false
}

func silenceText(s sim.ScriptSilence) string {
	switch s {
	case sim.ScriptSilenceUnsupported:
		return "this build does not evaluate that arm (its readers are inert)"
	case sim.ScriptSilenceNoUnit:
		return "its unit reference resolves to no entity this world holds"
	case sim.ScriptSilenceNoGroup:
		return "it names no group"
	case sim.ScriptSilenceNoPlayer:
		return "its player reference resolves to no player this world holds"
	case sim.ScriptSilenceNoStructure:
		return "its structure reference resolves to no structure this world holds"
	}
	// A REASON THIS TABLE DOES NOT NAME PRINTS ITS OWN NUMBER rather than the
	// word "unknown". The table is written by hand and pkg/sim's reason list
	// grows, so the failure mode is a reason silently reading as the same
	// unhelpful string every other unnamed reason reads as; two reasons went
	// unnamed here that way. A number tells the reader which arm to add.
	return fmt.Sprintf("unnamed silence reason %d", uint8(s))
}

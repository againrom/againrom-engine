package sim

// THE SCRIPT TRACE — what one step observed the mission script do.
//
// The runtime in script.go decides a mission and says nothing about how. Two
// counters move, a reporter reads them, and an outcome comes out; nothing
// anywhere records WHICH authored arm moved a counter, so a mission that ends
// the wrong way ends it invisibly. This file is that record, and it is
// OBSERVATION ONLY: nothing here is read by the runtime, nothing here is stored
// on a world, and nothing here enters the byte form or the digest.
//
// It is a RETURN VALUE, not a sink. A trace is built by StepTraced, handed back
// to the caller, and dropped; there is no field to set, no callback to register
// and nothing left behind between two steps. That is what makes "tracing changes
// nothing" a property of the shape rather than a promise: an untraced Step
// passes a nil trace down the same path, every recorder returns at once on a nil
// receiver, and the two runs are the same run. A hook stored on the world would
// have been state beside the digest, and a package-level sink would have been
// state beside every world at once.
//
// WHAT IT RECORDS is every check disposition, every trigger decision and every
// instant slot. The older Firings, VIP and Silent projections remain so their
// existing consumers need not reconstruct those three subsets from the complete
// records.

// ScriptSilence is why a check wrote no register.
//
// A check that writes nothing is not an error and is not rare. The older Silent
// projection carries the four measurement failures it always carried; the
// complete ScriptCheckRun also names build-time constants, VIP no-value arms,
// dead arms and the health selector's deliberate silence.
type ScriptSilence uint8

const (
	// ScriptSilenceNone marks a check that wrote its result. It is used only in
	// ScriptCheckRun, beside HasSilence; the older Silent projection never
	// contains it.
	ScriptSilenceNone ScriptSilence = 0xff
	// ScriptSilenceUnsupported is an arm this build does not evaluate. Its
	// readers are already marked inert, so it cannot mislead a trigger — it is
	// recorded because the pass is where a consumer is looking.
	ScriptSilenceUnsupported ScriptSilence = 0
	// ScriptSilenceNoUnit is a unit reference that resolves to no held entity
	// and no retained dead row. IT IS THE DANGEROUS ONE:
	// inertness is derived from unimplemented CHECKS alone, so a trigger reading
	// this register is live and compares whatever the register last held.
	ScriptSilenceNoUnit ScriptSilence = 1
	// ScriptSilenceNoGroup is a group count naming no group. There is nothing to
	// count, which is not the zero a group with no living member answers.
	ScriptSilenceNoGroup ScriptSilence = 2
	// ScriptSilenceNoPlayer is the population count or the nearest-unit distance
	// naming no player. There is nothing to measure, which is not the zero the
	// population count answers for a player with no living entity, nor the 0xff
	// the nearest-unit distance answers for the same player. It is not serialised
	// anywhere: a trace is a return value, dropped at the end of the step that
	// built it, and never crosses the byte form.
	ScriptSilenceNoPlayer ScriptSilence = 3
	// ScriptSilenceNoStructure is a structure reference that did not resolve
	// (1033 B3). `TRIG-BIND-010` states no shipped node reaches it — every
	// authored Target_Structure reference resolves against the map's own
	// structure table — so unlike ScriptSilenceNoUnit it is a safety net
	// rather than a documented shipped-content hazard.
	ScriptSilenceNoStructure ScriptSilence = 4
	// ScriptSilenceBuildTimeConstant is a constant node. NewScript accepts it
	// and the world constructor presets its owned register; a script pass does
	// not dispatch it again.
	ScriptSilenceBuildTimeConstant ScriptSilence = 16
	// ScriptSilenceVIPNoValue is a dispatched VIP arm. It writes no register by
	// design; LostBefore and LostAfter say whether its side effect ran.
	ScriptSilenceVIPNoValue ScriptSilence = 17
	// ScriptSilenceDeadArm is one of the two dispatched dead arms. Its complete
	// effect is deliberately no write.
	ScriptSilenceDeadArm ScriptSilence = 18
	// ScriptSilenceHealthGate is a dispatched health check whose authored gate
	// parameter did not select the write.
	ScriptSilenceHealthGate ScriptSilence = 19
)

// ScriptCheckRun is one check's complete runtime disposition in one pass.
//
// Dispatched distinguishes an arm entered by runCheck from a build-time
// constant and an unsupported opcode. Wrote is recorded at the write site, not
// inferred by comparing Before and Value: writing the value a register already
// held is still a write. LostBefore and LostAfter expose the VIP arm's separate
// counter effect without making that counter part of the observer.
type ScriptCheckRun struct {
	Check      int32
	Op         int32
	Register   int32
	Dispatched bool
	Wrote      bool
	Before     int32
	Value      int32
	HasSilence bool
	Silence    ScriptSilence
	LostBefore uint32
	LostAfter  uint32
}

// ScriptSilentCheck is one check that took no measurement this pass: which check
// it is, what it asks for, the register it did not write, and why.
type ScriptSilentCheck struct {
	Check    int32
	Op       int32
	Register int32
	Why      ScriptSilence
}

// ScriptVIPLoss is a VIP check that counted a loss this pass, naming the check
// and the entity it found dead.
//
// It is its own record because a loss it counts belongs to NO TRIGGER. A
// consumer looking only at what fired would see a mission lost with nothing
// having fired, which is exactly the blind spot this file exists to close.
type ScriptVIPLoss struct {
	Check int32
	Unit  EntityID
}

// ScriptPairValue is one condition pair with the two register values it was
// compared on.
//
// THE VALUES ARE THE POINT. The pair alone is authored data a consumer can read
// off the compiled script at any time; what nothing else can recover is what the
// two registers held at the moment the comparison was made, which is the
// difference between "the trigger compares distance against 3" and "the trigger
// compared 255 against 3".
type ScriptPairValue struct {
	Index  int32
	Pair   ScriptPair
	Left   int32
	Right  int32
	Result bool
}

// ScriptTriggerDecision is the one branch a trigger took in a pass.
type ScriptTriggerDecision uint8

const (
	ScriptTriggerInert ScriptTriggerDecision = iota
	ScriptTriggerSpent
	ScriptTriggerFailed
	ScriptTriggerFired
	// ScriptTriggerHeld is a repeating trigger whose condition remained true
	// across passes. It runs its instants again but has no new latch edge.
	ScriptTriggerHeld
)

// ScriptTriggerRun is one trigger's complete disposition in one pass. Pairs
// contains only comparisons that ran, in authored order; a failed result is the
// last element because trigger conditions short-circuit. FirstFailed is
// ScriptNone when no pair failed.
type ScriptTriggerRun struct {
	Trigger     int32
	Latch       int32
	Once        bool
	Decision    ScriptTriggerDecision
	LatchBefore byte
	LatchAfter  byte
	FirstFailed int32
	Pairs       []ScriptPairValue
}

// ScriptInstantOutcome says what the runtime dispatch itself established. It
// deliberately does not claim semantic correctness: the coverage witness joins
// this visit to a claim-backed effect oracle.
type ScriptInstantOutcome uint8

const (
	ScriptInstantUnsupported ScriptInstantOutcome = iota
	ScriptInstantNoStateChange
	ScriptInstantStateChanged
)

// ScriptInstantRun is one instant slot of a firing trigger: where it sat, which
// instant it named, its opcode, and whether this build has that arm.
//
// Supported is recorded rather than inferred because runInstant reaches no case
// for an arm it does not have and does nothing, silently. A firing whose only
// instant is unsupported did happen and changed nothing, and those are two
// different sentences.
type ScriptInstantRun struct {
	Slot          int32
	Instant       int32
	Op            int32
	SubCommand    int32
	HasSubCommand bool
	Supported     bool
	Outcome       ScriptInstantOutcome
	BeforeHash    uint64
	AfterHash     uint64
}

// ScriptFiring is one trigger that held: which trigger, the latch it holds down,
// the pairs it compared and the instants it ran.
//
// Trigger is the subscript in the compiled slice and Latch is the trigger's
// position in the MAP'S OWN array — the two differ whenever a map's triggers
// were not all built, and the map's number is the one a reader can look up.
type ScriptFiring struct {
	Trigger  int32
	Latch    int32
	Once     bool
	Pairs    [3]ScriptPairValue
	Instants []ScriptInstantRun
}

// ScriptTrace is everything one step observed of the script. It is the value
// StepTraced returns and it is empty for the fifteen steps of every sixteen on
// which neither script phase runs.
//
// Tick is the tick the step ran ON, which is one less than the tick the world
// reports afterwards: a step ends by advancing the clock, so a pass recorded
// here at 262 is read back off a world that says 263. The phase arithmetic
// (`tick % 16`) is the recorded number's, not the reported one's.
//
// Won, Lost and Outcome are the three the reporter reads, AS THE SCRIPT PHASE
// LEFT THEM. Nothing later in a step touches any of the three, so they are also
// what the step ends with; they are taken at the phase because that is where
// they mean "what the script just did".
type ScriptTrace struct {
	Tick     uint64
	Pass     bool
	Report   bool
	Checks   []ScriptCheckRun
	Triggers []ScriptTriggerRun
	Firings  []ScriptFiring
	VIP      []ScriptVIPLoss
	Silent   []ScriptSilentCheck
	Won      uint32
	Lost     uint32
	Outcome  Outcome
}

// Empty reports whether this trace observed nothing at all: no phase ran on the
// step it came from.
func (tr *ScriptTrace) Empty() bool {
	return tr == nil || (!tr.Pass && !tr.Report)
}

// The three recorders. EACH IS SAFE ON A NIL TRACE, and that is the whole of how
// an untraced step pays nothing: Step passes nil, every call below returns on its
// first line, and no branch anywhere in the runtime asks whether tracing is on.
func (tr *ScriptTrace) silence(i int32, c ScriptCheck, why ScriptSilence) {
	if tr == nil {
		return
	}
	tr.Silent = append(tr.Silent, ScriptSilentCheck{
		Check: i, Op: c.Op, Register: c.Register, Why: why,
	})
}

func (tr *ScriptTrace) vipLoss(i int32, unit EntityID) {
	if tr == nil {
		return
	}
	tr.VIP = append(tr.VIP, ScriptVIPLoss{Check: i, Unit: unit})
}

// check appends the disposition collected at the arm's actual write and return
// sites. It is called by one defer in runCheck so every exit is represented.
func (tr *ScriptTrace) check(i int32, c ScriptCheck, before int32, lostBefore uint32,
	dispatched, wrote bool, silence ScriptSilence, w *World) {
	if tr == nil {
		return
	}
	r := ScriptCheckRun{
		Check: i, Op: c.Op, Register: c.Register,
		Dispatched: dispatched, Wrote: wrote, Before: before,
		Value: w.registerAt(c.Register), Silence: ScriptSilenceNone,
		LostBefore: lostBefore, LostAfter: w.lost,
	}
	if !wrote {
		r.HasSilence, r.Silence = true, silence
	}
	tr.Checks = append(tr.Checks, r)
}

func (tr *ScriptTrace) triggerStart(w *World, i int32, t ScriptTrigger) int {
	if tr == nil {
		return -1
	}
	tr.Triggers = append(tr.Triggers, ScriptTriggerRun{
		Trigger: i, Latch: t.Latch, Once: t.Once,
		LatchBefore: w.latches[t.Latch], FirstFailed: ScriptNone,
	})
	return len(tr.Triggers) - 1
}

func (tr *ScriptTrace) triggerPair(run, index int, p ScriptPair, left, right int32, result bool) {
	if tr == nil || run < 0 {
		return
	}
	r := &tr.Triggers[run]
	r.Pairs = append(r.Pairs, ScriptPairValue{
		Index: int32(index), Pair: p, Left: left, Right: right, Result: result,
	})
	if !result {
		r.FirstFailed = int32(index)
	}
}

func (tr *ScriptTrace) triggerDone(w *World, run int, decision ScriptTriggerDecision) {
	if tr == nil || run < 0 {
		return
	}
	tr.Triggers[run].Decision = decision
	tr.Triggers[run].LatchAfter = w.latches[tr.Triggers[run].Latch]
}

// firing records a trigger that held, WITH THE REGISTERS AS THEY STAND. It is
// called between triggerHolds returning true and the first instant running, and
// that placement is the contract: the pass writes no register between those two
// points, so the values read here are the ones the comparison was made on — and
// an instant of an earlier trigger may already have moved a register this one
// reads, which is exactly why they cannot be re-read after the pass.
func (tr *ScriptTrace) firing(w *World, i int32, t ScriptTrigger) int {
	if tr == nil {
		return -1
	}
	f := ScriptFiring{Trigger: i, Latch: t.Latch, Once: t.Once}
	for k, p := range t.Pairs {
		f.Pairs[k].Index = int32(k)
		f.Pairs[k].Pair = p
		if !p.Used {
			continue
		}
		f.Pairs[k].Left = w.registers[p.Left]
		f.Pairs[k].Right = w.registers[p.Right]
		f.Pairs[k].Result = true
	}
	tr.Firings = append(tr.Firings, f)
	return len(tr.Firings) - 1
}

// instantStart takes the state digest immediately before the dispatch. It is a
// method so an untraced Step returns before asking Hash to traverse the world.
func (tr *ScriptTrace) instantStart(w *World) uint64 {
	if tr == nil {
		return 0
	}
	return w.Hash()
}

func (tr *ScriptTrace) instantDone(w *World, firing int, slot int32, instant int32,
	in ScriptInstant, before uint64) {
	if tr == nil || firing < 0 {
		return
	}
	after := w.Hash()
	r := ScriptInstantRun{
		Slot: slot, Instant: instant, Op: in.Op,
		Supported: scriptInstantSupported(in), BeforeHash: before, AfterHash: after,
	}
	if in.Op == ScriptInstantGroupOrder {
		r.HasSubCommand, r.SubCommand = true, in.Args[0]
	}
	switch {
	case !r.Supported:
		r.Outcome = ScriptInstantUnsupported
	case before == after:
		r.Outcome = ScriptInstantNoStateChange
	default:
		r.Outcome = ScriptInstantStateChanged
	}
	tr.Firings[firing].Instants = append(tr.Firings[firing].Instants, r)
}

// RegisterOwner is the check that writes register r, and whether any check owns
// it at all.
//
// It is what turns a trace back into the map's own terms: a pair names two
// register subscripts and nothing else, and the condition a reader wants is the
// CHECK behind each — its arm, its unit, its box, its radius. NewScript has
// already refused two checks owning one register, so the answer is unique.
//
// A register OWNED BY NO CHECK is a real answer and not a miss: that is an
// authored mission variable, zero until an instant writes it.
//
// The scan is linear over the compiled checks. The shipped corpus's largest
// script has 64 of them and this is called by a consumer formatting a line, not
// by anything on a tick's path, so an index would be a second representation of
// a fact the constructor already fixed.
func (s *Script) RegisterOwner(r int32) (ScriptCheck, int32, bool) {
	if s == nil {
		return ScriptCheck{}, ScriptNone, false
	}
	for i := range s.checks {
		if s.checks[i].Register == r {
			return s.checks[i], int32(i), true
		}
	}
	return ScriptCheck{}, ScriptNone, false
}

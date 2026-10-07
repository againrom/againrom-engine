package main

import (
	"fmt"
	"sort"
	"strings"

	"againrom/pkg/game"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
)

type controlledResult struct {
	Dispatched  bool
	Effect      bool
	Disposition string
	Divergence  string
	Seed        string
	Oracle      string
}

func runControlled(result *coverageResult, maps []*loadedMap) error {
	byMission := make(map[int]*loadedMap, len(maps))
	for _, m := range maps {
		byMission[m.Mission] = m
	}
	for _, key := range operationKeys() {
		op := operationCoverage{Root: result.Root, Family: key.Family, Opcode: key.Opcode, Claims: claimOracle[key]}
		var candidates []int
		for i, row := range result.Nodes {
			if row.Reachable && row.Family == key.Family && row.Opcode == key.Opcode {
				op.Nodes++
				if row.Natural {
					op.Natural++
				}
				candidates = append(candidates, i)
			}
		}
		if len(candidates) == 0 {
			return fmt.Errorf("operation %s:%d has no reachable authored node", key.Family, key.Opcode)
		}

		if key.Family == "check-build" || key.Family == "instant-build" {
			op.Dispatched = false
			op.Effect = true
			op.Witness = "builder"
			op.Disposition = "PASS"
			op.Oracle = result.Nodes[candidates[0]].Oracle
			for _, i := range candidates {
				if !result.Nodes[i].Effect {
					return fmt.Errorf("builder row %s:%d node %s/%d failed its postcondition",
						key.Family, key.Opcode, result.Nodes[i].Map, result.Nodes[i].Node)
				}
			}
			result.Operations = append(result.Operations, op)
			continue
		}

		var failures []string
		witnessed := 0
		allDispatched, allEffects := true, true
		divergences := map[string]int{}
		// A row is an operation family, but its evidence population is every
		// reachable authored node carrying that key. One representative would
		// let a different parameter/reference shape under the same opcode escape
		// the effect oracle. The complete node rows are already the stable join;
		// run every one and require every one to close.
		for _, i := range candidates {
			row := &result.Nodes[i]
			m := byMission[row.Mission]
			got, err := controlNode(m, *row)
			if err != nil {
				if len(failures) < 3 {
					failures = append(failures, fmt.Sprintf("%s/%d: %v", row.Map, row.Node, err))
				}
				continue
			}
			if got.Disposition == "" {
				got.Disposition = "PASS"
			}
			row.Controlled, row.Dispatched, row.Effect = true, got.Dispatched, got.Effect
			row.Disposition, row.Divergence = got.Disposition, got.Divergence
			row.Seed, row.Oracle = got.Seed, got.Oracle
			if row.Source == "" {
				row.Source = "controlled"
			}
			op.Controlled = true
			allDispatched = allDispatched && got.Dispatched
			allEffects = allEffects && got.Effect
			if got.Disposition == "DIVERGES" {
				if got.Divergence == "" {
					return fmt.Errorf("operation %s:%d node %s/%d diverges without a ledger id",
						key.Family, key.Opcode, row.Map, row.Node)
				}
				divergences[got.Divergence]++
			} else if got.Disposition != "PASS" || !got.Effect {
				return fmt.Errorf("operation %s:%d node %s/%d has disposition=%q effect=%v",
					key.Family, key.Opcode, row.Map, row.Node, got.Disposition, got.Effect)
			}
			if op.Witness != "" {
				op.Witness += ";"
			}
			op.Witness += fmt.Sprintf("%s#%d", row.Map, row.Node)
			op.Seed, op.Oracle = got.Seed, got.Oracle
			witnessed++
		}
		if witnessed != len(candidates) {
			return fmt.Errorf("operation %s:%d witnessed %d of %d shipped nodes: %s",
				key.Family, key.Opcode, witnessed, len(candidates), strings.Join(failures, "; "))
		}
		op.Dispatched = allDispatched
		switch len(divergences) {
		case 0:
			op.Effect, op.Disposition = allEffects, "PASS"
		case 1:
			for id := range divergences {
				op.Divergence = id
			}
			op.Effect, op.Disposition = false, "DIVERGES"
		default:
			return fmt.Errorf("operation %s:%d population names multiple divergences: %v",
				key.Family, key.Opcode, divergences)
		}
		closedDivergence := op.Disposition == "DIVERGES" && op.Divergence != "" && op.Oracle != ""
		if !op.Controlled || !op.Dispatched || (!op.Effect && !closedDivergence) {
			return fmt.Errorf("operation %s:%d has no controlled dispatch/effect witness: %s",
				key.Family, key.Opcode, strings.Join(failures, "; "))
		}
		result.Operations = append(result.Operations, op)
	}
	if len(result.Operations) != 52 {
		return fmt.Errorf("controlled matrix has %d operation rows, want 52", len(result.Operations))
	}
	return nil
}

func controlNode(m *loadedMap, row nodeCoverage) (controlledResult, error) {
	switch row.Family {
	case "check":
		return controlCheck(m, row.Node)
	case "instant", "group":
		return controlInstant(m, row.Node, row.Family)
	default:
		return controlledResult{}, fmt.Errorf("no runtime controller for %s", row.Family)
	}
}

func controlCheck(m *loadedMap, raw int) (controlledResult, error) {
	checks := m.Started.World.Script().Checks()
	if raw < 0 || raw >= len(checks) {
		return controlledResult{}, fmt.Errorf("compiled check index outside program")
	}
	return controlExactCheck(m, checks[raw], "fresh production mission state; exact compiled check unchanged")
}

// controlExactCheck drives one supplied check through the production script
// builder, controlled-world constructor and StepTraced dispatch. Synthetic
// coverage may replace an opcode while retaining a real mission record's
// references; every ordinary controlled row passes its compiled record
// unchanged.
func controlExactCheck(m *loadedMap, check sim.ScriptCheck, seed string) (controlledResult, error) {
	program, err := sim.NewScript([]sim.ScriptCheck{check}, nil, nil)
	if err != nil {
		return controlledResult{}, fmt.Errorf("NewScript exact check: %w", err)
	}
	w, err := sim.NewControlledScriptWorld(m.Started.World, program)
	if err != nil {
		return controlledResult{}, err
	}
	advanceToScriptPass(w)
	expected, wrote, lostDelta, oracle, err := expectedCheck(w, check)
	if err != nil {
		return controlledResult{}, err
	}
	trace := sim.StepTraced(w, nil)
	if !trace.Pass || len(trace.Checks) != 1 {
		return controlledResult{}, fmt.Errorf("StepTraced returned %d check records on pass=%v", len(trace.Checks), trace.Pass)
	}
	run := trace.Checks[0]
	if !run.Dispatched {
		return controlledResult{}, fmt.Errorf("check trace says dispatch did not enter")
	}
	if run.Wrote != wrote {
		return controlledResult{}, fmt.Errorf("write=%v silence=(%v,%d), claim oracle wants write=%v (%s); check=%+v",
			run.Wrote, run.HasSilence, run.Silence, wrote, oracle, check)
	}
	if wrote && run.Value != expected {
		return controlledResult{}, fmt.Errorf("value=%d, claim oracle wants %d (%s)", run.Value, expected, oracle)
	}
	if run.LostAfter-run.LostBefore != lostDelta {
		return controlledResult{}, fmt.Errorf("loss delta=%d, claim oracle wants %d (%s)", run.LostAfter-run.LostBefore, lostDelta, oracle)
	}
	return controlledResult{Dispatched: true, Effect: true, Seed: seed, Oracle: oracle}, nil
}

func expectedCheck(w *sim.World, c sim.ScriptCheck) (value int32, wrote bool, lost uint32, oracle string, err error) {
	entities := w.Entities()
	entity, hasEntity := entityByID(entities, c.Unit)
	hasEntity = c.HasUnit && hasEntity
	if c.Op != sim.ScriptCheckGroupCount && c.Op != sim.ScriptCheckPopulation && c.Op != sim.ScriptCheckNearest &&
		c.Op != sim.ScriptCheckRelation && c.Op != sim.ScriptCheckSackAt && c.Op != sim.ScriptCheckVariable &&
		c.Op != sim.ScriptCheckStructField && c.Op != sim.ScriptCheckVIP && !hasEntity {
		return 0, false, 0, "unresolved unit reference preserves the owned register", nil
	}
	switch c.Op {
	case sim.ScriptCheckGroupCount:
		if !c.HasGroup {
			return 0, false, 0, "absent group reference preserves the owned register", nil
		}
		// The check reads the one resolved group's cached count; an id carried
		// by two owners resolves to the later owner's group (AI-366).
		var owner uint32
		var carried bool
		for _, e := range entities {
			if e.Group == c.Group && (!carried || e.Owner > owner) {
				owner, carried = e.Owner, true
			}
		}
		for _, e := range entities {
			if e.Group == c.Group && e.Owner == owner && scriptLiving(e) {
				value++
			}
		}
		return value, true, 0, "TRIG-COUNT-015: count living members of the resolved group carrying the placed group id", nil
	case sim.ScriptCheckInBox:
		if entity.X >= c.Args[0] && entity.X <= c.Args[2] && entity.Y >= c.Args[1] && entity.Y <= c.Args[3] {
			value = 1
		}
		return value, true, 0, "TRIG-COND-003: inclusive authored rectangle", nil
	case sim.ScriptCheckWithin:
		if chebyshev(entity.X, entity.Y, c.Args[0], c.Args[1]) <= int64(c.Args[2]) {
			value = 1
		}
		return value, true, 0, "TRIG-DIST-014: Chebyshev distance compared with the authored radius", nil
	case sim.ScriptCheckHealth:
		if c.Args[0] != 6 {
			return 0, false, 0, "HERO-HEALTH-032: selector other than 6 preserves the register", nil
		}
		return entity.HP, true, 0, "HERO-HEALTH-032: selector 6 reads signed current health", nil
	case sim.ScriptCheckAlive:
		if scriptLiving(entity) {
			value = 1
		}
		return value, true, 0, "TRIG-COND-003: alive-or-dying actor is 1; fully dead actor is 0", nil
	case sim.ScriptCheckUnitDistance:
		other, ok := entityByID(entities, c.Unit2)
		if !c.HasUnit2 || !ok || !scriptLiving(entity) || !scriptLiving(other) {
			return 0xff, true, 0, "TRIG-DIST-014: absent or dead endpoint saturates to 255", nil
		}
		return byteDistance(entity.X, entity.Y, other.X, other.Y), true, 0, "TRIG-DIST-014: byte-masked Chebyshev unit distance", nil
	case sim.ScriptCheckDistance:
		return byteDistance(entity.X, entity.Y, c.Args[0], c.Args[1]), true, 0, "TRIG-DIST-014: byte-masked Chebyshev cell distance", nil
	case sim.ScriptCheckPopulation:
		if !c.HasPlayer {
			return 0, false, 0, "absent player reference preserves the owned register", nil
		}
		for _, e := range entities {
			if e.Owner == c.Player && scriptLiving(e) {
				value++
			}
		}
		return value, true, 0, "TRIG-REAP-017: count the player's living actors", nil
	case sim.ScriptCheckTargetID:
		if entity.HasAttackTarget {
			if target, ok := entityByID(entities, entity.AttackTarget); ok {
				value = int32(target.MapUnitID)
			}
		}
		return value, true, 0, "TRIG-TARGETID-032: current pursuit target's authored map id, else zero", nil
	case sim.ScriptCheckRelation:
		if !c.HasPlayer || !c.HasPlayer2 {
			return 0, false, 0, "absent player reference preserves the owned register", nil
		}
		return int32(w.Relations().Byte(c.Player, c.Player2) & 3), true, 0, "TRIG-DIPLO-020: low two bits of the directed relation cell", nil
	case sim.ScriptCheckSackAt:
		x, y := c.Args[0]&0xff, c.Args[1]&0xff
		for _, sack := range w.Sacks() {
			if sack.X == x && sack.Y == y {
				value = 1
				break
			}
		}
		return value, true, 0, "TRIG-SACK-022: sack presence at byte-truncated cell", nil
	case sim.ScriptCheckNearest:
		if !c.HasPlayer {
			return 0, false, 0, "absent player reference preserves the owned register", nil
		}
		value = 0xff
		for _, e := range entities {
			if e.Owner != c.Player || !scriptLiving(e) {
				continue
			}
			if d := byteDistance(e.X, e.Y, c.Args[0], c.Args[1]); d < value {
				value = d
			}
		}
		return value, true, 0, "TRIG-NEAREST-029: minimum byte-masked living-unit distance, seeded 255", nil
	case sim.ScriptCheckItemDistance:
		value = 0xff
		if c.HasItem && carriedCount(w, c.Unit, c.Item) > 0 {
			value = byteDistance(entity.X, entity.Y, c.Args[0]&0xff, c.Args[1]&0xff)
		}
		return value, true, 0, "TRIG-DIST-014: carried-item gate then byte-coordinate distance, else 255", nil
	case sim.ScriptCheckItemTestAlias, sim.ScriptCheckItemTest:
		if c.HasItem && carriedCount(w, c.Unit, c.Item) > 0 {
			value = 1
		}
		return value, true, 0, "TRIG-ITEMTEST-040: named code in carried container only", nil
	case sim.ScriptCheckVIP:
		if !c.HasUnit || !hasEntity {
			return 0, false, 0, "unresolved VIP reference preserves both register and loss counter", nil
		}
		if !scriptLiving(entity) {
			lost = 1
		}
		return 0, false, lost, "TRIG-END-009: VIP writes no value and increments loss only when fully dead", nil
	case sim.ScriptCheckVariable:
		value = w.ScriptRegister(c.Args[0])
		if c.Args[0] == 93 {
			// This oracle reads the pre-pass world. SAV-650 advances slot93
			// before the check copies it, including signed32 wraparound.
			value++
		}
		return value, true, 0, "TRIG-PARAM-030;SAV-650: copy the authored slot after the session counter advances", nil
	case sim.ScriptCheckStructField:
		if !c.HasStructure {
			return 0, false, 0, "unresolved structure reference preserves the owned register", nil
		}
		for _, st := range w.Structures() {
			if st.ID == c.Structure {
				return int32(int16(st.Field42)), true, 0, "SAV-BLDG-037: sign-extend the structure field word", nil
			}
		}
		return 0, false, 0, "missing structure preserves the owned register", nil
	}
	return 0, false, 0, "", fmt.Errorf("no check oracle for opcode %d", c.Op)
}

func controlInstant(m *loadedMap, raw int, family string) (controlledResult, error) {
	ci := -1
	for i, r := range m.instantRawOf {
		if r == raw {
			ci = i
			break
		}
	}
	if ci < 0 {
		return controlledResult{}, fmt.Errorf("authored action has no compiled instant")
	}
	in := m.Started.World.Script().Instants()[ci]
	base, preseed, err := prepareInstantBase(m.Started.World, in)
	if err != nil {
		return controlledResult{}, err
	}
	w, before, after, run, err := executeExactInstant(base, in)
	if err != nil {
		return controlledResult{}, err
	}
	ok, oracle := instantOracle(in, before, after, run, family)
	if !ok {
		return controlledResult{}, fmt.Errorf("effect postcondition failed: %s; instant=%+v outcome=%d hash=%016x->%016x",
			oracle, in, run.Outcome, before.Hash, after.Hash)
	}
	// The positive-path seed must not erase the ordinary path that the exact
	// shipped node sees in a freshly started mission. Run that path separately
	// whenever preparation produced a different world and require its effect
	// oracle too. This closes both hit/miss arms without changing the compiled
	// instant under observation.
	if base != m.Started.World {
		_, plainBefore, plainAfter, plainRun, err := executeExactInstant(m.Started.World, in)
		if err != nil {
			return controlledResult{}, fmt.Errorf("unseeded exact path: %w", err)
		}
		plainOK, plainOracle := instantOracle(in, plainBefore, plainAfter, plainRun, family)
		if !plainOK {
			return controlledResult{}, fmt.Errorf("unseeded exact path failed: %s; instant=%+v outcome=%d hash=%016x->%016x carried=%v->%v sacks=%d->%d entity=%+v->%+v",
				plainOracle, in, plainRun.Outcome, plainBefore.Hash, plainAfter.Hash,
				plainBefore.Carried[in.Unit], plainAfter.Carried[in.Unit], len(plainBefore.Sacks), len(plainAfter.Sacks),
				plainBefore.Entities[in.Unit], plainAfter.Entities[in.Unit])
		}
		oracle += "; unseeded exact path also passed: " + plainOracle
	}
	_ = w
	seed := "fresh production mission state; exact compiled instant unchanged; always-true one-shot trigger"
	if preseed != "" {
		seed += "; " + preseed
	}
	if in.Op == sim.ScriptInstantMessage {
		messageOracle, err := controlMessageBroadcast(m, raw, in)
		if err != nil {
			return controlledResult{}, err
		}
		oracle += "; " + messageOracle
	}

	// The two lifetime setters have distinct match and no-match effects in the
	// active claim. The fresh mission supplies the no-match arm. Seed a matching
	// effect through a second production script and require the exact shipped
	// setter to reach the store as well.
	if in.Op == sim.ScriptInstantCellEffectAge || in.Op == sim.ScriptInstantUnitEffectAge {
		seeded, seedText, err := seedEffectForSetter(m.Started.World, in)
		if err != nil {
			return controlledResult{}, err
		}
		_, matchedBefore, matchedAfter, matchedRun, err := executeExactInstant(seeded, in)
		if err != nil {
			return controlledResult{}, err
		}
		matched, matchedOracle := instantOracle(in, matchedBefore, matchedAfter, matchedRun, family)
		if !matched || matchedRun.Outcome != sim.ScriptInstantStateChanged {
			return controlledResult{}, fmt.Errorf("matching-effect postcondition failed: %s; outcome=%d before=%+v after=%+v",
				matchedOracle, matchedRun.Outcome, matchedBefore.Attached, matchedAfter.Attached)
		}
		seed += "; " + seedText
		oracle += "; matching and absent-effect paths both witnessed"
	}
	return controlledResult{Dispatched: true, Effect: true, Seed: seed, Oracle: oracle}, nil
}

func controlMessageBroadcast(m *loadedMap, raw int, in sim.ScriptInstant) (string, error) {
	if raw < 0 || raw >= len(m.Source.Actions) {
		return "", fmt.Errorf("message action index outside authored program")
	}
	actionID := m.Source.Actions[raw].ID
	latch := int32(-1)
	for i, trigger := range m.Source.Triggers {
		if trigger.Left[0] == 0 {
			continue
		}
		for _, id := range trigger.Acts {
			if id == actionID {
				latch = int32(i)
				break
			}
		}
		if latch >= 0 {
			break
		}
	}
	if latch < 0 {
		return "", fmt.Errorf("reachable message action has no accepted trigger slot")
	}
	program, err := sim.NewScript(nil, []sim.ScriptInstant{in}, []sim.ScriptTrigger{{
		Instants: [4]int32{0, sim.ScriptNone, sim.ScriptNone, sim.ScriptNone}, Once: true, Latch: latch,
	}})
	if err != nil {
		return "", fmt.Errorf("NewScript message broadcast: %w", err)
	}
	w, err := sim.NewControlledScriptWorld(m.Started.World, program)
	if err != nil {
		return "", err
	}
	announcer := game.NewAnnouncer(w, append([]mapload.ScriptRaise(nil), m.Started.Raises...))
	advanceToScriptPass(w)
	trace := sim.StepTraced(w, nil)
	if !trace.Pass || len(trace.Firings) != 1 || len(trace.Firings[0].Instants) != 1 {
		return "", fmt.Errorf("message broadcast probe did not fire exactly once")
	}
	events := announcer.Sample(w)
	for _, event := range events {
		if event == in.Args[0] {
			return "TRIG-MSG-023: the production announcer emitted the authored event from the builder's exact raise join", nil
		}
	}
	return "", fmt.Errorf("production announcer emitted %v, missing authored event %d", events, in.Args[0])
}

func prepareInstantBase(base *sim.World, in sim.ScriptInstant) (*sim.World, string, error) {
	var seed sim.ScriptInstant
	var description string
	switch in.Op {
	case sim.ScriptInstantTakeItem:
		if !in.HasUnit || !in.HasItem {
			return base, "exact node has an unresolved unit or item reference; the claimed path is a no-op", nil
		}
		if _, ok := entityByID(base.Entities(), in.Unit); !ok {
			return base, "exact node's resolved unit is absent from the started world; the claimed path is a no-op", nil
		}
		seed = sim.ScriptInstant{Op: sim.ScriptInstantAddItem, Unit: in.Unit, HasUnit: in.HasUnit,
			Item: in.Item, HasItem: in.HasItem}
		description = "one matching carried item seeded through instant 12"
	case sim.ScriptInstantReturnToMap:
		if !in.HasUnit {
			return base, "exact node has no resolved unit; the claimed path is a no-op", nil
		}
		if _, ok := entityByID(base.Entities(), in.Unit); !ok {
			return base, "exact node's resolved unit is absent from the started world; the claimed path is a no-op", nil
		}
		seed = sim.ScriptInstant{Op: sim.ScriptInstantTakeOffMap, Unit: in.Unit, HasUnit: in.HasUnit}
		description = "the referenced unit removed through instant 16 before its exact return"
	case sim.ScriptInstantSwapOnMap:
		if !in.HasUnit || !in.HasUnit2 || in.Unit == in.Unit2 {
			return base, "exact node lacks two distinct resolved units; the claimed path is a no-op", nil
		}
		if _, ok := entityByID(base.Entities(), in.Unit); !ok {
			return base, "exact node's first unit is absent from the started world; the claimed path is a no-op", nil
		}
		if _, ok := entityByID(base.Entities(), in.Unit2); !ok {
			return base, "exact node's second unit is absent from the started world; the claimed path is a no-op", nil
		}
		seed = sim.ScriptInstant{Op: sim.ScriptInstantTakeOffMap, Unit: in.Unit2, HasUnit: in.HasUnit2}
		description = "the replacement unit removed through instant 16 before the exact swap"
	case sim.ScriptInstantGiveAll:
		if !in.HasUnit || !in.HasUnit2 || in.Unit == in.Unit2 {
			return base, "exact node lacks two distinct resolved units; the claimed path is a no-op", nil
		}
		if _, ok := entityByID(base.Entities(), in.Unit); !ok {
			return base, "exact node's giver is absent from the started world; the claimed path is a no-op", nil
		}
		if _, ok := entityByID(base.Entities(), in.Unit2); !ok {
			return base, "exact node's receiver is absent from the started world; the claimed path is a no-op", nil
		}
		code, ok := scriptSeedItem(base)
		if !ok {
			return nil, "", fmt.Errorf("give-all seed found no production item code")
		}
		seed = sim.ScriptInstant{Op: sim.ScriptInstantAddItem, Unit: in.Unit, HasUnit: in.HasUnit,
			Item: code, HasItem: true}
		description = fmt.Sprintf("item %d seeded on the giver through instant 12", code)
	case sim.ScriptInstantDropAll:
		if !in.HasUnit {
			return base, "exact node has no resolved unit; the claimed path is a no-op", nil
		}
		if _, ok := entityByID(base.Entities(), in.Unit); !ok {
			return base, "exact node's unit is absent from the started world; the claimed path is a no-op", nil
		}
		code, ok := scriptSeedItem(base)
		if !ok {
			return nil, "", fmt.Errorf("drop-all seed found no production item code")
		}
		seed = sim.ScriptInstant{Op: sim.ScriptInstantAddItem, Unit: in.Unit, HasUnit: in.HasUnit,
			Item: code, HasItem: true}
		description = fmt.Sprintf("item %d seeded on the dropper through instant 12", code)
	case sim.ScriptInstantGroupOnMap:
		if !in.HasGroup {
			return base, "exact node has no group reference; the claimed path is a no-op", nil
		}
		members := false
		for _, e := range base.Entities() {
			if e.Group == in.Group {
				members = true
				break
			}
		}
		if !members {
			return base, "exact raw group id names no started-world member; the claimed path is a no-op", nil
		}
		seed = sim.ScriptInstant{Op: sim.ScriptInstantGroupOffMap, Group: in.Group, HasGroup: in.HasGroup}
		description = "every named group member removed through instant 32 before its exact return"
	default:
		return base, "", nil
	}
	w, _, _, run, err := executeExactInstant(base, seed)
	if err != nil {
		return nil, "", fmt.Errorf("effect preseed: %w", err)
	}
	if run.Outcome != sim.ScriptInstantStateChanged {
		return nil, "", fmt.Errorf("effect preseed changed no state: %s", description)
	}
	return w, description, nil
}

func scriptSeedItem(w *sim.World) (uint16, bool) {
	for _, in := range w.Script().Instants() {
		if in.HasItem && in.Item != 0 {
			return in.Item, true
		}
	}
	for _, e := range w.Entities() {
		stacks, _ := w.CarriedStacks(e.ID)
		for _, stack := range stacks {
			if stack.Code != 0 {
				return stack.Code, true
			}
		}
	}
	return 0, false
}

func executeExactInstant(base *sim.World, in sim.ScriptInstant) (*sim.World, instantObservation, instantObservation, sim.ScriptInstantRun, error) {
	w, before, after, run, err := executeProbeInstant(base, in)
	if err != nil {
		return nil, instantObservation{}, instantObservation{}, sim.ScriptInstantRun{}, err
	}
	if !run.Supported || run.Outcome == sim.ScriptInstantUnsupported {
		return nil, instantObservation{}, instantObservation{}, sim.ScriptInstantRun{}, fmt.Errorf("trace marks exact opcode %d unsupported", in.Op)
	}
	return w, before, after, run, nil
}

func executeProbeInstant(base *sim.World, in sim.ScriptInstant) (*sim.World, instantObservation, instantObservation, sim.ScriptInstantRun, error) {
	program, err := sim.NewScript(nil, []sim.ScriptInstant{in}, []sim.ScriptTrigger{{
		Instants: [4]int32{0, sim.ScriptNone, sim.ScriptNone, sim.ScriptNone}, Once: true, Latch: 0,
	}})
	if err != nil {
		return nil, instantObservation{}, instantObservation{}, sim.ScriptInstantRun{}, fmt.Errorf("NewScript exact instant: %w", err)
	}
	w, err := sim.NewControlledScriptWorld(base, program)
	if err != nil {
		return nil, instantObservation{}, instantObservation{}, sim.ScriptInstantRun{}, err
	}
	advanceToScriptPass(w)
	before := observeInstant(w, in)
	trace := sim.StepTraced(w, nil)
	if !trace.Pass || len(trace.Firings) != 1 || len(trace.Firings[0].Instants) != 1 {
		return nil, instantObservation{}, instantObservation{}, sim.ScriptInstantRun{}, fmt.Errorf("StepTraced did not return the one probe dispatch")
	}
	run := trace.Firings[0].Instants[0]
	if run.Slot != 0 || run.Instant != 0 || run.Op != in.Op {
		return nil, instantObservation{}, instantObservation{}, sim.ScriptInstantRun{},
			fmt.Errorf("StepTraced identified probe as slot=%d instant=%d opcode=%d, want 0/0/%d",
				run.Slot, run.Instant, run.Op, in.Op)
	}
	if in.Op == sim.ScriptInstantGroupOrder {
		if !run.HasSubCommand || run.SubCommand != in.Args[0] {
			return nil, instantObservation{}, instantObservation{}, sim.ScriptInstantRun{},
				fmt.Errorf("StepTraced identified group subcommand as present=%v value=%d, want true/%d",
					run.HasSubCommand, run.SubCommand, in.Args[0])
		}
	} else if run.HasSubCommand {
		return nil, instantObservation{}, instantObservation{}, sim.ScriptInstantRun{},
			fmt.Errorf("StepTraced invented group subcommand %d for opcode %d", run.SubCommand, in.Op)
	}
	after := observeInstant(w, in)
	return w, before, after, run, nil
}

func seedEffectForSetter(base *sim.World, setter sim.ScriptInstant) (*sim.World, string, error) {
	var seed sim.ScriptInstant
	var matches func(*sim.World) bool
	switch setter.Op {
	case sim.ScriptInstantCellEffectAge:
		seed = sim.ScriptInstant{Op: sim.ScriptInstantCastAtCell, Args: [10]int32{
			setter.Args[0], setter.Args[1], setter.Args[0], setter.Args[1], setter.Args[2], 99,
		}}
		matches = func(w *sim.World) bool {
			for _, e := range w.CellEffects() {
				if uint16(e.X)+uint16(e.Y)<<8 == uint16(setter.Args[0])+uint16(setter.Args[1])<<8 && byte(e.Spell) == byte(setter.Args[2]) {
					return true
				}
			}
			return false
		}
	case sim.ScriptInstantUnitEffectAge:
		target, ok := entityByID(base.Entities(), setter.Unit)
		if !setter.HasUnit || !ok {
			return nil, "", fmt.Errorf("cannot seed attached effect: exact setter target is absent")
		}
		seed = sim.ScriptInstant{Op: sim.ScriptInstantCastAtUnit, Unit: setter.Unit, HasUnit: true,
			Args: [10]int32{target.X, target.Y, setter.Args[0], 99}}
		matches = func(w *sim.World) bool {
			for _, e := range w.ActiveEffects() {
				if e.Target == setter.Unit && byte(e.Spell) == byte(setter.Args[0]) {
					return true
				}
			}
			return false
		}
	default:
		return nil, "", fmt.Errorf("opcode %d is not an effect setter", setter.Op)
	}
	w, _, _, run, err := executeExactInstant(base, seed)
	if err != nil {
		return nil, "", fmt.Errorf("effect seed dispatch: %w", err)
	}
	if run.Outcome != sim.ScriptInstantStateChanged {
		return nil, "", fmt.Errorf("effect seed dispatch changed no state")
	}
	for tick := 0; tick < 128; tick++ {
		if matches(w) {
			return w, "matching effect seeded by a production script cast before the exact setter", nil
		}
		sim.Step(w, nil)
	}
	return nil, "", fmt.Errorf("production script cast did not create the setter's matching effect within 128 ticks")
}

type instantObservation struct {
	Hash      uint64
	Register  int32
	Won, Lost uint32
	Formation uint8
	Relation  byte
	Purse     uint32
	Entities  map[sim.EntityID]sim.Entity
	// Carried is deliberately a code/count projection: script item opcodes
	// name codes and their oracle asks only whether that named population
	// changed. Sacks below are the complete canonical instances, so this
	// witness never reconstructs simulation state from the projection.
	Carried    map[sim.EntityID]map[uint16]uint32
	Sacks      []sim.Sack
	Casts      []sim.ScriptCast
	Tails      []sim.CellTail
	Cell       []sim.CellEffect
	Attached   []sim.ActiveEffect
	Structures []sim.Structure
	Groups     []sim.ScriptGroupState
}

func observeInstant(w *sim.World, in sim.ScriptInstant) instantObservation {
	o := instantObservation{
		Hash: w.Hash(), Entities: map[sim.EntityID]sim.Entity{}, Carried: map[sim.EntityID]map[uint16]uint32{},
		Sacks: w.Sacks(), Casts: w.ScriptCasts(), Tails: w.CellTails(), Cell: w.CellEffects(),
		Attached: w.ActiveEffects(), Structures: w.Structures(), Groups: sim.ObserveScriptGroups(w, in.Group),
	}
	o.Won, o.Lost = w.ScriptCounters()
	o.Register = w.ScriptRegister(in.Args[0])
	o.Formation = w.FormationMode(in.Player)
	o.Relation = w.Relations().Byte(uint32(in.Args[0]), uint32(in.Args[1]))
	o.Purse = w.Purse(in.Player)
	for _, e := range w.Entities() {
		o.Entities[e.ID] = e
		stacks, _ := w.CarriedStacks(e.ID)
		counts := map[uint16]uint32{}
		for _, stack := range stacks {
			counts[stack.Code] += stack.Count
		}
		o.Carried[e.ID] = counts
	}
	return o
}

func instantOracle(in sim.ScriptInstant, before, after instantObservation, run sim.ScriptInstantRun, family string) (bool, string) {
	if family == "group" || in.Op == sim.ScriptInstantGroupOrder {
		return groupOracle(in, before, after, run)
	}
	switch in.Op {
	case sim.ScriptInstantMessage:
		return run.BeforeHash == run.AfterHash && run.Outcome == sim.ScriptInstantNoStateChange,
			"TRIG-MSG-023: deliberate empty arm leaves canonical state unchanged"
	case sim.ScriptInstantSetVariable:
		return after.Register == in.Args[1], "TRIG-PARAM-030: authored variable slot equals the assigned value"
	case sim.ScriptInstantWin:
		return after.Won == before.Won+1 && after.Lost == before.Lost, "TRIG-END-009: win counter increments alone"
	case sim.ScriptInstantLose:
		return after.Lost == before.Lost+1 && after.Won == before.Won, "TRIG-END-009: loss counter increments alone"
	case sim.ScriptInstantIncVariable:
		return after.Register == before.Register+1, "TRIG-PARAM-030: authored variable slot increments by one"
	case sim.ScriptInstantFormation:
		if !in.HasPlayer {
			return instantUnchanged(run), "AI-FORM-037: absent player reference is a deliberate no-op"
		}
		return after.Formation == uint8(in.Args[0]), "AI-FORM-037: referenced player's formation byte equals the authored mode"
	case sim.ScriptInstantDropAll:
		return dropAll(in, before, after, run), "TRIG-DROPALL-024: the whole carried container joins or creates the sack at the unit's own cell, then becomes empty"
	case sim.ScriptInstantRelation:
		want := byte(int32(before.Relation&^3) + in.Args[2])
		return after.Relation == want, "TRIG-DIPLO-019: directed relation becomes (old &^ 3) + addend as a byte"
	case sim.ScriptInstantAddItem:
		return itemDelta(in, before, after, run, 1), "TRIG-ADDITEM-027: one named item joins the referenced unit's carried container"
	case sim.ScriptInstantTakeItem:
		return itemDelta(in, before, after, run, -1), "TRIG-TAKEITEM-038: one named carried item is detached and destroyed, or miss is a no-op"
	case sim.ScriptInstantTakeOffMap:
		return offMapSingle(in, before, after, run, true), "TRIG-OFFMAP-041: referenced unit loses map presence and retains identity/state"
	case sim.ScriptInstantReturnToMap:
		return returnSingle(in, before, after, run), "TRIG-RETURN-042: off-map unit returns near its retained cell; on-map unit is unchanged"
	case sim.ScriptInstantSwapOnMap:
		return swapPresence(in, before, after, run), "TRIG-OFFMAP-041/TRIG-RETURN-042: first unit leaves and second is placed near its retained cell"
	case sim.ScriptInstantGiveUnit:
		return giveUnit(in, before, after, run), "TRIG-ACT-004: referenced unit takes the referenced owner and a fresh group"
	case sim.ScriptInstantCastAtCell:
		return castAtCell(in, before, after), "TRIG-CASTACTOR-044: exact node appends one pending cell cast with byte coordinates and word power"
	case sim.ScriptInstantGiveGroup:
		return giveGroup(in, before, after, run), "TRIG-ACT-004: every member carrying the raw group id takes the referenced owner"
	case sim.ScriptInstantGiveMoney:
		if !in.HasPlayer {
			return instantUnchanged(run), "TRIG-MONEY-028: absent player reference is a deliberate no-op"
		}
		return after.Purse == before.Purse+uint32(in.Args[0]), "TRIG-MONEY-028: referenced purse receives the authored unsigned amount"
	case sim.ScriptInstantCastAtUnit:
		return castAtUnit(in, before, after, run), "TRIG-CASTACTOR-044: exact node appends one pending unit cast with resolved target"
	case sim.ScriptInstantCellTail:
		return cellTail(in, after), "TRIG-CELLTAIL-035: addressed cell carries {spell,power,0,y,0,y} bytes"
	case sim.ScriptInstantGiveAll:
		return giveAll(in, before, after, run), "TRIG-GIVEALL-025: giver's whole carried container pours into receiver and giver becomes empty"
	case sim.ScriptInstantCellEffectAge:
		return cellEffectAge(in, before, after, run), "TRIG-EFFECTTIME-034/TRIG-CELLEFFECT-045: all matches at the 16-bit ADD cell key whose effect-id byte equals the full 32-bit spell parameter take the authored word duration; no match is a no-op"
	case sim.ScriptInstantUnitEffectAge:
		return unitEffectAge(in, before, after, run), "TRIG-EFFECTTIME-034: matching attached effect takes the authored word duration; no match is a no-op"
	case sim.ScriptInstantGroupOffMap:
		return groupPresence(in, before, after, run, true), "TRIG-MAPGROUP-043: every member carrying the raw group id loses map presence"
	case sim.ScriptInstantGroupOnMap:
		return groupReturn(in, before, after, run), "TRIG-MAPGROUP-043: each off-map group member attempts its own retained-cell return"
	case sim.ScriptInstantProperty:
		return propertySet(in, before, after, run), "TRIG-PROPERTY-036: selector 6/15/16 writes health/defence/absorption word; other selector stores nothing"
	}
	return false, fmt.Sprintf("no instant effect oracle for opcode %d", in.Op)
}

func instantUnchanged(run sim.ScriptInstantRun) bool {
	return run.Outcome == sim.ScriptInstantNoStateChange && run.BeforeHash == run.AfterHash
}

func advanceToScriptPass(w *sim.World) {
	for w.Tick()%16 != 6 {
		sim.Step(w, nil)
	}
}

func entityByID(entities []sim.Entity, id sim.EntityID) (sim.Entity, bool) {
	for _, e := range entities {
		if e.ID == id {
			return e, true
		}
	}
	return sim.Entity{}, false
}

func scriptLiving(e sim.Entity) bool { return e.Alive() || e.Dying() }

func chebyshev(ax, ay, bx, by int32) int64 {
	dx := int64(ax) - int64(bx)
	dy := int64(ay) - int64(by)
	if dx < 0 {
		dx = -dx
	}
	if dy < 0 {
		dy = -dy
	}
	if dx > dy {
		return dx
	}
	return dy
}

func byteDistance(ax, ay, bx, by int32) int32 { return int32(chebyshev(ax, ay, bx, by) & 0xff) }

func carriedCount(w *sim.World, id sim.EntityID, code uint16) uint32 {
	stacks, _ := w.CarriedStacks(id)
	var n uint32
	for _, stack := range stacks {
		if stack.Code == code {
			n += stack.Count
		}
	}
	return n
}

func itemDelta(in sim.ScriptInstant, before, after instantObservation, run sim.ScriptInstantRun, delta int) bool {
	b, a := before.Carried[in.Unit][in.Item], after.Carried[in.Unit][in.Item]
	if !in.HasUnit || !in.HasItem || in.Item == 0 {
		return instantUnchanged(run)
	}
	if delta > 0 {
		return a == b+1
	}
	if b == 0 {
		return a == 0
	}
	return a+1 == b
}

func offMapSingle(in sim.ScriptInstant, before, after instantObservation, run sim.ScriptInstantRun, want bool) bool {
	b, bok := before.Entities[in.Unit]
	a, aok := after.Entities[in.Unit]
	if !in.HasUnit || !bok || !aok {
		return instantUnchanged(run)
	}
	return a.OffMap == want && a.ID == b.ID && a.X == b.X && a.Y == b.Y && a.Owner == b.Owner && a.Group == b.Group
}

func returnSingle(in sim.ScriptInstant, before, after instantObservation, run sim.ScriptInstantRun) bool {
	b, bok := before.Entities[in.Unit]
	a, aok := after.Entities[in.Unit]
	if !in.HasUnit || !bok || !aok {
		return instantUnchanged(run)
	}
	if !b.OffMap {
		return !a.OffMap
	}
	return !a.OffMap && chebyshev(a.X, a.Y, b.X, b.Y) <= 3
}

func dropAll(in sim.ScriptInstant, before, after instantObservation, run sim.ScriptInstantRun) bool {
	b, ok := before.Entities[in.Unit]
	if !in.HasUnit || !ok {
		return instantUnchanged(run)
	}
	holding := before.Carried[in.Unit]
	if len(after.Carried[in.Unit]) != 0 {
		return false
	}
	// TRIG-DROPALL-024's original arm hands the container to the sack
	// routine even when it contains no item. An empty source therefore still
	// plants an empty sack when none exists; pouring it into an existing sack
	// has no canonical effect. This is separate from worn equipment, which the
	// arm neither reads nor moves.
	if len(holding) == 0 {
		beforeSack, hadSack := observedSackAt(before.Sacks, b.X, b.Y)
		afterSack, hasSack := observedSackAt(after.Sacks, b.X, b.Y)
		if hadSack {
			return hasSack && afterSack.Gold == beforeSack.Gold &&
				equalItemCounts(beforeSack.Items, afterSack.Items) && instantUnchanged(run)
		}
		return hasSack && afterSack.Gold == 0 && len(afterSack.Items) == 0 &&
			len(after.Sacks) == len(before.Sacks)+1
	}
	beforeAt, afterAt := map[uint16]uint32{}, map[uint16]uint32{}
	for _, sack := range before.Sacks {
		if sack.X == b.X && sack.Y == b.Y {
			for _, code := range sack.Items {
				beforeAt[code]++
			}
		}
	}
	for _, sack := range after.Sacks {
		if sack.X == b.X && sack.Y == b.Y {
			for _, code := range sack.Items {
				afterAt[code]++
			}
		}
	}
	for code, count := range holding {
		if afterAt[code] != beforeAt[code]+count {
			return false
		}
	}
	return true
}

func observedSackAt(sacks []sim.Sack, x, y int32) (sim.Sack, bool) {
	for _, sack := range sacks {
		if sack.X == x && sack.Y == y {
			return sack, true
		}
	}
	return sim.Sack{}, false
}

func equalItemCounts(a, b []uint16) bool {
	if len(a) != len(b) {
		return false
	}
	counts := make(map[uint16]int, len(a))
	for _, code := range a {
		counts[code]++
	}
	for _, code := range b {
		counts[code]--
	}
	for _, count := range counts {
		if count != 0 {
			return false
		}
	}
	return true
}

func swapPresence(in sim.ScriptInstant, before, after instantObservation, run sim.ScriptInstantRun) bool {
	b1, ok1 := before.Entities[in.Unit]
	b2, ok2 := before.Entities[in.Unit2]
	a1, aok1 := after.Entities[in.Unit]
	a2, aok2 := after.Entities[in.Unit2]
	if !in.HasUnit || !in.HasUnit2 || !ok1 || !ok2 || !aok1 || !aok2 || in.Unit == in.Unit2 {
		return instantUnchanged(run)
	}
	return a1.OffMap && !a2.OffMap && chebyshev(a2.X, a2.Y, b1.X, b1.Y) <= 3 && b2.ID == a2.ID
}

func giveUnit(in sim.ScriptInstant, before, after instantObservation, run sim.ScriptInstantRun) bool {
	b, bok := before.Entities[in.Unit]
	a, aok := after.Entities[in.Unit]
	if !in.HasPlayer || !in.HasUnit || !bok || !aok {
		return instantUnchanged(run)
	}
	return a.Owner == in.Player && a.CommandGroup == 0 && (b.Owner == in.Player || a.Group != b.Group)
}

func castAtCell(in sim.ScriptInstant, before, after instantObservation) bool {
	if len(after.Casts) != len(before.Casts)+1 {
		return false
	}
	c := after.Casts[len(after.Casts)-1]
	power := uint16(in.Args[5])
	if power == 0 {
		power = 99
	}
	return !c.AtUnit && c.FromX == int32(uint8(in.Args[0])) && c.FromY == int32(uint8(in.Args[1])) &&
		c.ToX == int32(uint8(in.Args[2])) && c.ToY == int32(uint8(in.Args[3])) && c.Spell == uint16(uint8(in.Args[4])) && c.Power == power
}

// resolvedGroupOwner is the owner a script group id answers to: the later
// owner among those carrying it (AI-366).
func resolvedGroupOwner(o instantObservation, group uint32) (owner uint32, ok bool) {
	for _, e := range o.Entities {
		if e.Group == group && (!ok || e.Owner > owner) {
			owner, ok = e.Owner, true
		}
	}
	return owner, ok
}

func giveGroup(in sim.ScriptInstant, before, after instantObservation, run sim.ScriptInstantRun) bool {
	if !in.HasPlayer || !in.HasGroup {
		return instantUnchanged(run)
	}
	found := false
	owner, _ := resolvedGroupOwner(before, in.Group)
	for id, b := range before.Entities {
		if b.Group != in.Group || b.Owner != owner {
			continue
		}
		found = true
		a := after.Entities[id]
		if a.Owner != in.Player || a.CommandGroup != 0 {
			return false
		}
	}
	return found || instantUnchanged(run)
}

func castAtUnit(in sim.ScriptInstant, before, after instantObservation, run sim.ScriptInstantRun) bool {
	if !in.HasUnit {
		return instantUnchanged(run)
	}
	if len(after.Casts) != len(before.Casts)+1 {
		return false
	}
	c := after.Casts[len(after.Casts)-1]
	power := uint16(in.Args[3])
	if power == 0 {
		power = 99
	}
	return c.AtUnit && c.Target == in.Unit && c.FromX == int32(uint8(in.Args[0])) && c.FromY == int32(uint8(in.Args[1])) &&
		c.Spell == uint16(uint8(in.Args[2])) && c.Power == power
}

func cellTail(in sim.ScriptInstant, after instantObservation) bool {
	x, y := int32(uint8(in.Args[2])), int32(uint8(in.Args[3]))
	want := [6]byte{byte(in.Args[0]), byte(in.Args[1]), 0, byte(in.Args[3]), 0, byte(in.Args[3])}
	for _, tail := range after.Tails {
		if tail.X == x && tail.Y == y {
			return tail.Bytes == want
		}
	}
	return false
}

func giveAll(in sim.ScriptInstant, before, after instantObservation, run sim.ScriptInstantRun) bool {
	if !in.HasUnit || !in.HasUnit2 || in.Unit == in.Unit2 {
		return instantUnchanged(run)
	}
	if _, ok := before.Entities[in.Unit]; !ok {
		return instantUnchanged(run)
	}
	if _, ok := before.Entities[in.Unit2]; !ok {
		return instantUnchanged(run)
	}
	for code, count := range before.Carried[in.Unit] {
		if after.Carried[in.Unit2][code] != before.Carried[in.Unit2][code]+count {
			return false
		}
	}
	return len(after.Carried[in.Unit]) == 0
}

func cellEffectAge(in sim.ScriptInstant, before, after instantObservation, run sim.ScriptInstantRun) bool {
	matched := 0
	for _, b := range before.Cell {
		if uint16(b.X)+uint16(b.Y)<<8 != uint16(in.Args[0])+uint16(in.Args[1])<<8 || byte(b.Spell) != byte(in.Args[2]) {
			continue
		}
		matched++
		found := false
		for _, a := range after.Cell {
			if a.X == b.X && a.Y == b.Y && a.Spell == b.Spell && a.Remaining == uint16(in.Args[3]) {
				found = true
			}
		}
		if !found {
			return false
		}
	}
	return matched > 0 || run.Outcome == sim.ScriptInstantNoStateChange
}

func unitEffectAge(in sim.ScriptInstant, before, after instantObservation, run sim.ScriptInstantRun) bool {
	matched := 0
	for _, b := range before.Attached {
		if b.Target != in.Unit || byte(b.Spell) != byte(in.Args[0]) {
			continue
		}
		matched++
		found := false
		for _, a := range after.Attached {
			if a.Target == b.Target && a.Spell == b.Spell && a.Remaining == uint16(in.Args[1]) {
				found = true
			}
		}
		if !found {
			return false
		}
	}
	return matched > 0 || run.Outcome == sim.ScriptInstantNoStateChange
}

func groupPresence(in sim.ScriptInstant, before, after instantObservation, run sim.ScriptInstantRun, off bool) bool {
	if !in.HasGroup {
		return instantUnchanged(run)
	}
	found := false
	owner, _ := resolvedGroupOwner(before, in.Group)
	for id, b := range before.Entities {
		if b.Group == in.Group && b.Owner == owner {
			found = true
			if after.Entities[id].OffMap != off {
				return false
			}
		}
	}
	return found || instantUnchanged(run)
}

func groupReturn(in sim.ScriptInstant, before, after instantObservation, run sim.ScriptInstantRun) bool {
	if !in.HasGroup {
		return instantUnchanged(run)
	}
	found := false
	owner, _ := resolvedGroupOwner(before, in.Group)
	for id, b := range before.Entities {
		if b.Group != in.Group || b.Owner != owner || !b.OffMap {
			continue
		}
		found = true
		a, ok := after.Entities[id]
		if !ok || a.OffMap || chebyshev(a.X, a.Y, b.X, b.Y) > 3 {
			return false
		}
	}
	return found || instantUnchanged(run)
}

func propertySet(in sim.ScriptInstant, before, after instantObservation, run sim.ScriptInstantRun) bool {
	b, bok := before.Entities[in.Unit]
	a, aok := after.Entities[in.Unit]
	if !in.HasUnit || !bok || !aok {
		return instantUnchanged(run)
	}
	want := int32(int16(uint16(in.Args[1])))
	switch in.Args[0] {
	case 6:
		return a.HP == want
	case 15:
		return a.Defence == want
	case 16:
		return a.Absorption == want
	default:
		return a.HP == b.HP && a.Defence == b.Defence && a.Absorption == b.Absorption
	}
}

func groupOracle(in sim.ScriptInstant, before, after instantObservation, run sim.ScriptInstantRun) (bool, string) {
	if !in.HasGroup {
		return instantUnchanged(run), "TRIG-GROUP-005: absent group reference is a deliberate no-op"
	}
	if len(before.Groups) == 0 {
		return instantUnchanged(run), "TRIG-GROUP-005: raw group id naming no runtime group is a deliberate no-op"
	}
	states := append([]sim.ScriptGroupState(nil), after.Groups...)
	sort.Slice(states, func(i, j int) bool { return states[i].Owner < states[j].Owner })
	wantOrder := uint8(in.Args[0])
	if in.Args[0] == 10 || in.Args[0] == 11 || in.Args[0] == 14 || in.Args[0] == 15 {
		wantOrder = 0
	}
	for _, g := range states {
		if g.Order != wantOrder {
			return false, fmt.Sprintf("TRIG-GROUP-005: group order=%d, want %d for sub-command %d", g.Order, wantOrder, in.Args[0])
		}
		if in.Args[0] == 2 || in.Args[0] == 4 || in.Args[0] == 5 {
			if g.CommandedX != in.Args[1] || g.CommandedY != in.Args[2] {
				return false, fmt.Sprintf("AI-GROUPCMD-020: commanded cell=(%d,%d), want (%d,%d)",
					g.CommandedX, g.CommandedY, in.Args[1], in.Args[2])
			}
		}
	}
	if in.Args[0] == 10 || in.Args[0] == 11 || in.Args[0] == 15 {
		if !in.HasUnit {
			return instantUnchanged(run), "TRIG-GRPARM-047: absent target reference leaves group untouched"
		}
		span := uint8(in.Args[1])
		if span == 0 {
			span = 3
		}
		owner, _ := resolvedGroupOwner(before, in.Group)
		for id, b := range before.Entities {
			if b.Group != in.Group || b.Owner != owner || !scriptLiving(b) {
				continue
			}
			a := after.Entities[id]
			switch in.Args[0] {
			case 10:
				if id != in.Unit && (!a.HasAttackTarget || a.AttackTarget != in.Unit) {
					return false, "AI-SCRIPTATTACK-120: non-target member did not acquire the named victim"
				}
			case 11, 15:
				if id != in.Unit && (!a.HasEscortTarget || a.EscortTarget != in.Unit || a.EscortRange != span) {
					return false, "TRIG-GRPLIMIT-048: escort target/range postcondition failed"
				}
			}
		}
	}
	if in.Args[0] == 14 {
		owner, _ := resolvedGroupOwner(before, in.Group)
		for id, b := range before.Entities {
			if b.Group == in.Group && b.Owner == owner && scriptLiving(b) && after.Entities[id].ActorState != 0x0a {
				return false, "AI-GROUPCMD-020: patrol member did not enter patrol state"
			}
		}
	}
	return true, fmt.Sprintf("claim-backed group sub-command %d order/member postconditions", in.Args[0])
}

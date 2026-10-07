package main

import (
	"fmt"

	"againrom/pkg/sim"
)

const (
	syntheticItemTest12  = "campaign-absent check opcode 12 alias"
	syntheticDropAll     = "authored-unreachable instant 20"
	syntheticGuard       = "authored-unreachable group sub-command 1"
	syntheticRoam        = "campaign-absent group sub-command 17"
	syntheticDwell       = "campaign-absent inert group sub-command 18"
	syntheticAttackVeto  = "authored-unreachable group-10 target veto"
	syntheticDefendRange = "authored-unreachable group-11 zero-range coercion"
	syntheticFollowRange = "authored-unreachable group-15 zero-range coercion"
)

func runSynthetic(result *coverageResult, maps []*loadedMap) error {
	if err := runSyntheticItemTest12(result, maps); err != nil {
		return err
	}
	if err := runExactSyntheticPopulation(result, maps, operationKey{"instant", sim.ScriptInstantDropAll}, syntheticDropAll, 3); err != nil {
		return err
	}
	if err := runExactSyntheticPopulation(result, maps, operationKey{"group", 1}, syntheticGuard, 1); err != nil {
		return err
	}
	if err := runAbsentGroupCommands(result, maps); err != nil {
		return err
	}
	if err := runSyntheticAttackVeto(result, maps); err != nil {
		return err
	}
	if err := runSyntheticZeroRange(result, maps, 11, syntheticDefendRange); err != nil {
		return err
	}
	if err := runSyntheticZeroRange(result, maps, 15, syntheticFollowRange); err != nil {
		return err
	}
	return nil
}

// runSyntheticItemTest12 proves the campaign-absent duplicate at the same
// production seam as every shipped check. It copies one real check-17 record,
// changes only its opcode to 12 and sends it through NewScript,
// NewControlledScriptWorld and StepTraced. A real authored 12 makes this
// synthetic premise stale and fails the witness instead of being hidden.
func runSyntheticItemTest12(result *coverageResult, maps []*loadedMap) error {
	var source *loadedMap
	var check sim.ScriptCheck
	raw := -1
	for _, m := range maps {
		for i, candidate := range m.Started.World.Script().Checks() {
			switch candidate.Op {
			case sim.ScriptCheckItemTestAlias:
				return fmt.Errorf("synthetic %s found authored opcode 12 at %s#%d; it now belongs in the shipped denominator",
					syntheticItemTest12, m.Name, m.checkRawOf[i])
			case sim.ScriptCheckItemTest:
				if source == nil {
					source, check, raw = m, candidate, m.checkRawOf[i]
				}
			}
		}
	}
	if source == nil {
		return fmt.Errorf("synthetic %s found no real check-17 record to duplicate", syntheticItemTest12)
	}
	check.Op = sim.ScriptCheckItemTestAlias
	seed := "copied a real mission check-17 record; replaced only the byte-identical opcode with 12"
	got, err := controlExactCheck(source, check, seed)
	if err != nil {
		return fmt.Errorf("synthetic %s at %s#%d: %w", syntheticItemTest12, source.Name, raw, err)
	}
	if !got.Dispatched || !got.Effect {
		return fmt.Errorf("synthetic %s reached no dispatch/effect boundary", syntheticItemTest12)
	}
	result.Synthetic = append(result.Synthetic, syntheticCoverage{
		Root: result.Root, Case: syntheticItemTest12, Map: source.Name, Mission: source.Mission, Node: raw,
		Family: "check", Opcode: sim.ScriptCheckItemTestAlias, SubCommand: -1,
		Present: false, Accepted: true, Controlled: true, Dispatched: true, Effect: true,
		Disposition: "PASS", Seed: got.Seed, Oracle: got.Oracle,
		Claims: syntheticOracle[operationKey{"check", sim.ScriptCheckItemTestAlias}],
	})
	return nil
}

func runExactSyntheticPopulation(result *coverageResult, maps []*loadedMap, key operationKey, name string, want int) error {
	byMission := make(map[int]*loadedMap, len(maps))
	for _, m := range maps {
		byMission[m.Mission] = m
	}
	seen := 0
	for i := range result.Nodes {
		row := &result.Nodes[i]
		if row.Family != key.Family || row.Opcode != key.Opcode || row.Reachable {
			continue
		}
		got, err := controlNode(byMission[row.Mission], *row)
		if err != nil {
			return fmt.Errorf("synthetic %s at %s#%d: %w", name, row.Map, row.Node, err)
		}
		if !got.Dispatched || !got.Effect {
			return fmt.Errorf("synthetic %s at %s#%d reached no dispatch/effect boundary", name, row.Map, row.Node)
		}
		row.Controlled, row.Dispatched, row.Effect = true, true, true
		row.Source, row.Disposition = "synthetic", "PASS"
		row.Seed, row.Oracle = got.Seed, got.Oracle
		result.Synthetic = append(result.Synthetic, syntheticCoverage{
			Root: result.Root, Case: name, Map: row.Map, Mission: row.Mission, Node: row.Node,
			Family: row.Family, Opcode: row.Opcode, SubCommand: row.SubCommand,
			Present: true, Accepted: row.Accepted, Controlled: true, Dispatched: true, Effect: true,
			Disposition: "PASS", Seed: got.Seed, Oracle: got.Oracle, Claims: row.Claims,
		})
		seen++
	}
	if seen != want {
		return fmt.Errorf("synthetic %s has %d authored-unreachable nodes, want the claim-backed population %d", name, seen, want)
	}
	return nil
}

type groupSource struct {
	Map     *loadedMap
	Raw     int
	Instant sim.ScriptInstant
}

func groupSources(maps []*loadedMap, sub int32) []groupSource {
	var out []groupSource
	for _, m := range maps {
		for ci, in := range m.Started.World.Script().Instants() {
			if in.Op != sim.ScriptInstantGroupOrder || !in.HasGroup || (sub >= 0 && in.Args[0] != sub) {
				continue
			}
			if len(sim.ObserveScriptGroups(m.Started.World, in.Group)) == 0 {
				continue
			}
			out = append(out, groupSource{Map: m, Raw: m.instantRawOf[ci], Instant: in})
		}
	}
	return out
}

func runAbsentGroupCommands(result *coverageResult, maps []*loadedMap) error {
	for _, sub := range []int32{17, 18} {
		if present := groupSources(maps, sub); len(present) != 0 {
			return fmt.Errorf("synthetic group sub-command %d is present in %d shipped nodes; its absence row is stale", sub, len(present))
		}
	}
	sources := groupSources(maps, -1)
	if len(sources) == 0 {
		return fmt.Errorf("synthetic absent group commands found no real mission group reference")
	}
	source := sources[0]
	for _, tc := range []struct {
		name        string
		sub         int32
		disposition string
		divergence  string
		effect      bool
		oracle      string
	}{
		{syntheticRoam, 17, "PASS", "", true,
			"TRIG-GROUP-005/AI-GROUPCMD-020: a synthetic command17 dispatch changes the referenced native Groups to order17; the local setter and subsequent Roam arithmetic have separate simulation tests"},
		{syntheticDwell, 18, "PASS", "", true,
			"TRIG-GROUP-005: catalogued literal 18 has no runtime case and is a deliberate canonical no-op"},
	} {
		in := source.Instant
		in.Args[0] = tc.sub
		_, before, after, run, err := executeProbeInstant(source.Map.Started.World, in)
		if err != nil {
			return fmt.Errorf("synthetic %s: %w", tc.name, err)
		}
		if tc.sub == 17 {
			effect, reason := groupOracle(in, before, after, run)
			if !run.Supported || run.Outcome != sim.ScriptInstantStateChanged || !effect {
				return fmt.Errorf("synthetic %s: supported=%v outcome=%d effect=%v: %s", tc.name, run.Supported, run.Outcome, effect, reason)
			}
		} else if run.Supported || run.Outcome != sim.ScriptInstantUnsupported || run.BeforeHash != run.AfterHash {
			return fmt.Errorf("synthetic %s: supported=%v outcome=%d hash %016x->%016x, want unsupported no-op",
				tc.name, run.Supported, run.Outcome, run.BeforeHash, run.AfterHash)
		}
		result.Synthetic = append(result.Synthetic, syntheticCoverage{
			Root: result.Root, Case: tc.name, Map: source.Map.Name, Mission: source.Map.Mission, Node: source.Raw,
			Family: "group", Opcode: tc.sub, SubCommand: tc.sub,
			Present: false, Accepted: true, Controlled: true, Dispatched: run.Supported, Effect: tc.effect,
			Disposition: tc.disposition, Divergence: tc.divergence,
			Seed:   "copied a real mission group reference; replaced only the absent sub-command literal; always-true one-shot trigger",
			Oracle: tc.oracle, Claims: syntheticOracle[operationKey{"group", tc.sub}],
		})
	}
	return nil
}

func runSyntheticAttackVeto(result *coverageResult, maps []*loadedMap) error {
	sources := groupSources(maps, -1)
	airTargets, vetoGroups := 0, 0
	for _, source := range sources {
		entities := source.Map.Started.World.Entities()
		var target sim.Entity
		foundTarget := false
		for _, e := range entities {
			if !scriptLiving(e) || e.OffMap || e.Domain != sim.DomainAir {
				continue
			}
			airTargets++
			if e.Group == source.Instant.Group {
				continue
			}
			target, foundTarget = e, true
			break
		}
		if hasVetoedMember(entities, source.Instant.Group) {
			vetoGroups++
		}
		if !foundTarget || !hasVetoedMember(entities, source.Instant.Group) {
			continue
		}
		in := source.Instant
		in.Args[0], in.Unit, in.HasUnit = 10, target.ID, true
		_, before, after, run, err := executeExactInstant(source.Map.Started.World, in)
		if err != nil {
			return fmt.Errorf("synthetic %s: %w", syntheticAttackVeto, err)
		}
		if run.Outcome != sim.ScriptInstantStateChanged || !attackVetoOracle(in, before, after) {
			return fmt.Errorf("synthetic %s failed its acquire-in-place postcondition", syntheticAttackVeto)
		}
		result.Synthetic = append(result.Synthetic, syntheticCoverage{
			Root: result.Root, Case: syntheticAttackVeto, Map: source.Map.Name, Mission: source.Map.Mission, Node: source.Raw,
			Family: "group", Opcode: 10, SubCommand: 10,
			Present: false, Accepted: true, Controlled: true, Dispatched: true, Effect: true, Disposition: "PASS",
			Seed:   "copied a real mission group reference and replaced its target with a real mission air-domain actor; the active claim proves shipped group-10 targets cannot reach that column",
			Oracle: "TRIG-GRPARM-047: every vetoed ground/ghost member acquires in place and does not receive the named attack target",
			Claims: "TRIG-GRPARM-047;AI-SCRIPTATTACK-120",
		})
		return nil
	}
	return fmt.Errorf("synthetic %s found no real mission group plus air-domain target fixture (sources=%d air_targets=%d veto_groups=%d)",
		syntheticAttackVeto, len(sources), airTargets, vetoGroups)
}

func hasVetoedMember(entities []sim.Entity, group uint32) bool {
	for _, e := range entities {
		if e.Group == group && scriptLiving(e) && !e.OffMap && e.Reach <= 1 &&
			(e.Domain == sim.DomainGround || e.Domain == sim.DomainGhost) {
			return true
		}
	}
	return false
}

func attackVetoOracle(in sim.ScriptInstant, before, after instantObservation) bool {
	matched := 0
	for id, b := range before.Entities {
		if b.Group != in.Group || !scriptLiving(b) || b.OffMap || b.Reach > 1 ||
			(b.Domain != sim.DomainGround && b.Domain != sim.DomainGhost) {
			continue
		}
		matched++
		a, ok := after.Entities[id]
		if !ok || a.ActorState != 0x0c || a.HasAttackTarget || a.PostX != b.X || a.PostY != b.Y {
			return false
		}
	}
	for _, group := range after.Groups {
		if group.Order != 0 {
			return false
		}
	}
	return matched > 0
}

func runSyntheticZeroRange(result *coverageResult, maps []*loadedMap, sub int32, name string) error {
	sources := groupSources(maps, sub)
	if len(sources) == 0 {
		return fmt.Errorf("synthetic %s found no exact group-%d source", name, sub)
	}
	source := sources[0]
	in := source.Instant
	in.Args[1] = 256
	_, before, after, run, err := executeExactInstant(source.Map.Started.World, in)
	if err != nil {
		return fmt.Errorf("synthetic %s: %w", name, err)
	}
	ok, oracle := groupOracle(in, before, after, run)
	if !ok || run.Outcome != sim.ScriptInstantStateChanged {
		return fmt.Errorf("synthetic %s failed: %s (outcome=%d)", name, oracle, run.Outcome)
	}
	result.Synthetic = append(result.Synthetic, syntheticCoverage{
		Root: result.Root, Case: name, Map: source.Map.Name, Mission: source.Map.Mission, Node: source.Raw,
		Family: "group", Opcode: sub, SubCommand: sub,
		Present: false, Accepted: true, Controlled: true, Dispatched: true, Effect: true, Disposition: "PASS",
		Seed:   "copied the exact compiled group command and replaced only its authored range with 256; low byte is zero",
		Oracle: oracle + "; TRIG-GRPLIMIT-048: byte zero coerces to escort range 3",
		Claims: "TRIG-GRPARM-047;TRIG-GRPLIMIT-048",
	})
	return nil
}

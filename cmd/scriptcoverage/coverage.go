package main

import (
	"fmt"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"againrom/pkg/formats/alm"
	"againrom/pkg/game"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
)

const buildTimeOpcode uint32 = 0x10002

type operationKey struct {
	Family string
	Opcode int32
}

type nodeKey struct {
	Mission int
	Family  string
	Node    int
}

// nodeCoverage is one stable authored node identity and every stage it crossed.
// Natural and Controlled are separate facts; neither is inferred from the
// other, and Source says which observation supplied the effect witness.
type nodeCoverage struct {
	Root, Map, Family string
	Mission           int
	Node              int
	Opcode            int32
	SubCommand        int32
	HasSubCommand     bool
	Present           bool
	Accepted          bool
	Reachable         bool
	Natural           bool
	NaturalChanged    bool
	NaturalNoChange   bool
	Controlled        bool
	Dispatched        bool
	Effect            bool
	Source            string
	Disposition       string
	Divergence        string
	Seed              string
	Oracle            string
	Claims            string
}

type operationCoverage struct {
	Root, Family    string
	Opcode          int32
	Claims          string
	Nodes           int
	Natural         int
	NaturalChanged  int
	NaturalNoChange int
	Controlled      bool
	Dispatched      bool
	Effect          bool
	Disposition     string
	Divergence      string
	Witness         string
	Seed            string
	Oracle          string
}

// syntheticCoverage is deliberately outside the 52-row shipped denominator.
// It names an absent operation, an authored-but-unreachable node or a semantic
// branch whose active claim proves no shipped record can reach it.
type syntheticCoverage struct {
	Root, Case  string
	Map         string
	Mission     int
	Node        int
	Family      string
	Opcode      int32
	SubCommand  int32
	Present     bool
	Accepted    bool
	Controlled  bool
	Dispatched  bool
	Effect      bool
	Disposition string
	Divergence  string
	Seed        string
	Oracle      string
	Claims      string
}

type coverageResult struct {
	Root       string
	Maps       int
	Checks     int
	Instants   int
	Triggers   int
	Reachable  int
	Natural    int
	Operations []operationCoverage
	Nodes      []nodeCoverage
	Synthetic  []syntheticCoverage
}

type loadedMap struct {
	Mission int
	Name    string
	Started *game.Mission
	Source  alm.Script

	checkRawOf   []int
	instantRawOf []int
	triggerRawOf []int
	rows         map[nodeKey]int
}

var claimOracle = map[operationKey]string{
	{"check", 1}:  "TRIG-COUNT-015;TRIG-GRPLIST-016;TRIG-REAP-017",
	{"check", 2}:  "TRIG-COND-003",
	{"check", 3}:  "TRIG-DIST-014",
	{"check", 4}:  "TRIG-COND-003;HERO-HEALTH-032",
	{"check", 5}:  "TRIG-COND-003",
	{"check", 6}:  "TRIG-DIST-014",
	{"check", 7}:  "TRIG-DIST-014",
	{"check", 8}:  "TRIG-COND-003;TRIG-REAP-017",
	{"check", 9}:  "TRIG-TARGETID-032",
	{"check", 10}: "TRIG-DIPLO-020",
	{"check", 14}: "TRIG-SACK-022",
	{"check", 15}: "TRIG-NEAREST-029",
	{"check", 16}: "TRIG-COND-003;TRIG-DIST-014",
	{"check", 17}: "TRIG-ITEMTEST-040",
	{"check", 18}: "TRIG-END-009;TRIG-ESCORT-031",
	{"check", 19}: "TRIG-COND-003;TRIG-PARAM-030",
	{"check", 21}: "ALM-CLS-053;SAV-BLDG-037",

	{"check-build", int32(buildTimeOpcode)}: "TRIG-COND-003;TRIG-PARAM-030",

	{"instant", 2}:  "TRIG-MSG-023",
	{"instant", 3}:  "TRIG-ACT-004;TRIG-PARAM-030",
	{"instant", 4}:  "TRIG-END-009",
	{"instant", 5}:  "TRIG-END-009",
	{"instant", 6}:  "TRIG-GROUP-005;AI-GROUPCMD-020",
	{"instant", 7}:  "AI-FORM-037",
	{"instant", 8}:  "TRIG-ACT-004;TRIG-PARAM-030",
	{"instant", 10}: "TRIG-DIPLO-019",
	{"instant", 12}: "TRIG-ADDITEM-027",
	{"instant", 13}: "TRIG-TAKEITEM-038",
	{"instant", 16}: "TRIG-OFFMAP-041",
	{"instant", 17}: "TRIG-RETURN-042",
	{"instant", 18}: "TRIG-OFFMAP-041;TRIG-RETURN-042",
	{"instant", 19}: "TRIG-ACT-004",
	{"instant", 21}: "TRIG-CAST-033;TRIG-CASTACTOR-044",
	{"instant", 22}: "TRIG-ACT-004",
	{"instant", 23}: "TRIG-MONEY-028",
	{"instant", 24}: "TRIG-CAST-033;TRIG-CASTACTOR-044",
	{"instant", 25}: "TRIG-CELLTAIL-035",
	{"instant", 28}: "TRIG-GIVEALL-025",
	{"instant", 29}: "TRIG-EFFECTTIME-034;TRIG-CELLEFFECT-045",
	{"instant", 30}: "TRIG-EFFECTTIME-034;MAGIC-EFFMODE-009;MAGIC-ATTACH-016",
	{"instant", 32}: "TRIG-MAPGROUP-043",
	{"instant", 33}: "TRIG-MAPGROUP-043",
	{"instant", 34}: "TRIG-PROPERTY-036;HERO-HEALTH-032;HERO-FOLD-033;UNIT-PLACE-034",

	{"instant-build", int32(buildTimeOpcode)}: "TRIG-DROP-013",

	{"group", 2}:  "TRIG-GROUP-005;AI-GROUPCMD-020",
	{"group", 3}:  "TRIG-GROUP-005;AI-GROUPCMD-020",
	{"group", 4}:  "TRIG-GROUP-005;AI-GROUPCMD-020",
	{"group", 5}:  "TRIG-GROUP-005;AI-GROUPCMD-020",
	{"group", 10}: "TRIG-GROUP-005;AI-SCRIPTATTACK-120;TRIG-GRPARM-047",
	{"group", 11}: "TRIG-GROUP-005;AI-DEFEND-111;AI-FOLLOWSET-116;TRIG-GRPARM-047;TRIG-GRPLIMIT-048",
	{"group", 14}: "TRIG-GROUP-005;AI-GROUPCMD-020",
	{"group", 15}: "TRIG-GROUP-005;AI-FOLLOWSET-116;TRIG-GRPARM-047;TRIG-GRPLIMIT-048",
}

// syntheticOracle names authored families outside the shipped reachable
// denominator. They remain in the node matrix and are never promoted into the
// 52 rows merely because a controlled programme can exercise them.
var syntheticOracle = map[operationKey]string{
	{"check", 12}:   "TRIG-ITEMTEST-040",
	{"instant", 20}: "TRIG-DROPALL-024",
	{"group", 1}:    "TRIG-GROUP-005;AI-GROUPCMD-020",
	{"group", 17}:   "TRIG-GROUP-005;AI-GROUPCMD-020;AI-ROAM-025",
	{"group", 18}:   "TRIG-GROUP-005;AI-GROUPCMD-020",
}

func operationKeys() []operationKey {
	out := make([]operationKey, 0, len(claimOracle))
	for k := range claimOracle {
		out = append(out, k)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Family != out[j].Family {
			return out[i].Family < out[j].Family
		}
		return out[i].Opcode < out[j].Opcode
	})
	return out
}

func rootLabel(root string) string {
	label := strings.ToLower(filepath.Base(filepath.Clean(root)))
	if label == "en" || label == "ru" {
		return label
	}
	return label
}

func missionNames(archives *game.Archives) ([]struct {
	Number int
	Name   string
}, error) {
	reader := game.ArchiveMaps(archives.Containers)
	var out []struct {
		Number int
		Name   string
	}
	for _, name := range reader.Names() {
		base := strings.TrimSuffix(strings.ToLower(name), filepath.Ext(name))
		n, err := strconv.Atoi(base)
		if err != nil {
			return nil, fmt.Errorf("campaign map %q has no numeric mission identity", name)
		}
		out = append(out, struct {
			Number int
			Name   string
		}{n, name})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Number < out[j].Number })
	return out, nil
}

func loadCoverage(root string) (*coverageResult, []*loadedMap, error) {
	archives, err := game.OpenArchives(root)
	if err != nil {
		return nil, nil, err
	}
	defs, err := game.LoadDefinitions(archives.Containers)
	if err != nil {
		return nil, nil, err
	}
	names, err := missionNames(archives)
	if err != nil {
		return nil, nil, err
	}
	result := &coverageResult{Root: rootLabel(root), Maps: len(names)}
	var maps []*loadedMap
	for _, name := range names {
		party := game.MissionPartyAs(false, defs.StartWeapon, defs.Bodies, defs.Table)
		started, err := game.StartMission(archives.Containers, name.Number, defs.Table,
			mapload.DifficultyNormal, party)
		if err != nil {
			return nil, nil, fmt.Errorf("mission %d: %w", name.Number, err)
		}
		src, err := started.Map.Script()
		if err != nil {
			return nil, nil, fmt.Errorf("mission %d script: %w", name.Number, err)
		}
		m := &loadedMap{Mission: name.Number, Name: name.Name, Started: started, Source: src, rows: map[nodeKey]int{}}
		if err := indexMap(result, m); err != nil {
			return nil, nil, fmt.Errorf("mission %d: %w", name.Number, err)
		}
		maps = append(maps, m)
	}
	return result, maps, nil
}

func addNode(result *coverageResult, m *loadedMap, row nodeCoverage) {
	row.Root, row.Map, row.Mission, row.Present = result.Root, m.Name, m.Mission, true
	if !row.HasSubCommand {
		row.SubCommand = -1
	}
	if row.Claims == "" {
		row.Claims = claimOracle[operationKey{row.Family, row.Opcode}]
		if row.Claims == "" {
			row.Claims = syntheticOracle[operationKey{row.Family, row.Opcode}]
		}
	}
	m.rows[nodeKey{m.Mission, row.Family, row.Node}] = len(result.Nodes)
	result.Nodes = append(result.Nodes, row)
}

func indexMap(result *coverageResult, m *loadedMap) error {
	compiled := m.Started.World.Script()
	if compiled == nil {
		return fmt.Errorf("production mission has no compiled script")
	}
	checks, instants, triggers := compiled.Checks(), compiled.Instants(), compiled.Triggers()
	result.Checks += len(checks)
	result.Instants += len(instants)
	result.Triggers += len(triggers)

	if len(checks) != len(m.Source.Conditions) {
		return fmt.Errorf("builder produced %d checks from %d authored nodes", len(checks), len(m.Source.Conditions))
	}
	for i, n := range m.Source.Conditions {
		family := "check"
		if n.Opcode == buildTimeOpcode {
			family = "check-build"
		}
		if checks[i].Op != int32(n.Opcode) {
			return fmt.Errorf("check %d compiled as opcode %d, authored %d", i, checks[i].Op, n.Opcode)
		}
		m.checkRawOf = append(m.checkRawOf, i)
		row := nodeCoverage{Family: family, Node: i, Opcode: int32(n.Opcode), Accepted: true, Reachable: true}
		if family == "check-build" {
			row.Source, row.Dispatched = "builder", false
			row.Effect = m.Started.World.ScriptRegister(checks[i].Register) == checks[i].Args[0]
			row.Disposition = "PASS"
			row.Oracle = "owned register equals the authored preset after mission construction"
			if !row.Effect {
				return fmt.Errorf("constant check %d register %d is %d, want preset %d", i, checks[i].Register,
					m.Started.World.ScriptRegister(checks[i].Register), checks[i].Args[0])
			}
		}
		addNode(result, m, row)
	}

	actionByID := make(map[uint32]int, len(m.Source.Actions))
	compiledOfAction := make(map[int]int)
	for i, n := range m.Source.Actions {
		actionByID[n.ID] = i
		if n.Opcode >= buildTimeOpcode {
			row := nodeCoverage{Family: "instant-build", Node: i, Opcode: int32(n.Opcode), Accepted: true,
				Reachable: true, Effect: n.Opcode == buildTimeOpcode, Source: "builder",
				Oracle: "record absent from runtime instant list and every compiled trigger slot"}
			if row.Effect {
				row.Disposition = "PASS"
			}
			addNode(result, m, row)
			continue
		}
		ci := len(m.instantRawOf)
		if ci >= len(instants) || instants[ci].Op != int32(n.Opcode) {
			return fmt.Errorf("action %d opcode %d has no matching compiled instant", i, n.Opcode)
		}
		m.instantRawOf = append(m.instantRawOf, i)
		compiledOfAction[i] = ci
		addNode(result, m, nodeCoverage{Family: "instant", Node: i, Opcode: int32(n.Opcode), Accepted: true})
		if n.Opcode == uint32(sim.ScriptInstantGroupOrder) {
			in := instants[ci]
			addNode(result, m, nodeCoverage{Family: "group", Node: i, Opcode: in.Args[0],
				SubCommand: in.Args[0], HasSubCommand: true, Accepted: true})
		}
	}
	if len(m.instantRawOf) != len(instants) {
		return fmt.Errorf("builder produced %d instants, mapped %d", len(instants), len(m.instantRawOf))
	}

	compiledTrigger := 0
	for raw, t := range m.Source.Triggers {
		accepted := t.Left[0] != 0
		addNode(result, m, nodeCoverage{Family: "trigger", Node: raw, Accepted: accepted, Reachable: accepted})
		if !accepted {
			continue
		}
		if compiledTrigger >= len(triggers) || triggers[compiledTrigger].Latch != int32(raw) {
			return fmt.Errorf("trigger %d has no matching compiled latch", raw)
		}
		m.triggerRawOf = append(m.triggerRawOf, raw)
		ct := triggers[compiledTrigger]
		for slot, id := range t.Acts {
			if id == 0 {
				continue
			}
			rawAction, ok := actionByID[id]
			if !ok {
				continue
			}
			n := m.Source.Actions[rawAction]
			if n.Opcode >= buildTimeOpcode {
				if ct.Instants[slot] != sim.ScriptNone {
					return fmt.Errorf("build-time action %d survives in trigger %d slot %d as runtime instant %d",
						rawAction, raw, slot, ct.Instants[slot])
				}
				continue
			}
			ci := compiledOfAction[rawAction]
			if ct.Instants[slot] != int32(ci) {
				return fmt.Errorf("trigger %d slot %d compiled instant %d, want %d", raw, slot, ct.Instants[slot], ci)
			}
			markReachable(result, m, "instant", rawAction)
			if n.Opcode == uint32(sim.ScriptInstantGroupOrder) {
				markReachable(result, m, "group", rawAction)
			}
		}
		compiledTrigger++
	}
	if compiledTrigger != len(triggers) {
		return fmt.Errorf("builder produced %d triggers, mapped %d", len(triggers), compiledTrigger)
	}
	return nil
}

func markReachable(result *coverageResult, m *loadedMap, family string, raw int) {
	if i, ok := m.rows[nodeKey{m.Mission, family, raw}]; ok {
		result.Nodes[i].Reachable = true
	}
}

func expectedTotals(label string) (checks, instants, triggers int) {
	if label == "ru" {
		return 678, 759, 397
	}
	return 680, 759, 398
}

func validateDenominator(result *coverageResult) error {
	if result.Maps != 28 {
		return fmt.Errorf("corpus drift: loaded %d campaign maps, want 28", result.Maps)
	}
	wc, wi, wt := expectedTotals(result.Root)
	if result.Checks != wc || result.Instants != wi || result.Triggers != wt {
		return fmt.Errorf("corpus drift: %s compiled checks/instants/triggers=%d/%d/%d, want %d/%d/%d",
			result.Root, result.Checks, result.Instants, result.Triggers, wc, wi, wt)
	}
	seen := map[operationKey]bool{}
	for _, row := range result.Nodes {
		if row.Reachable && claimOracle[operationKey{row.Family, row.Opcode}] != "" {
			seen[operationKey{row.Family, row.Opcode}] = true
		}
	}
	if len(seen) != 52 {
		var missing []string
		for _, k := range operationKeys() {
			if !seen[k] {
				missing = append(missing, fmt.Sprintf("%s:%d", k.Family, k.Opcode))
			}
		}
		return fmt.Errorf("corpus drift: %d reachable claim-backed operation rows, want 52; missing=%s",
			len(seen), strings.Join(missing, ","))
	}
	result.Reachable = len(seen)
	return nil
}

func runNatural(result *coverageResult, maps []*loadedMap, ticks int) error {
	if ticks <= 0 {
		return fmt.Errorf("natural drive needs a positive tick count")
	}
	seenInstant := map[nodeKey]bool{}
	for _, m := range maps {
		play := game.NewPlayWorld(m.Started)
		w := play.World
		for tick := 0; tick < ticks; tick++ {
			tr := play.StepTraced(naturalCommands(w, tick))
			if !tr.Pass {
				continue
			}
			for _, check := range tr.Checks {
				if check.Check < 0 || int(check.Check) >= len(m.checkRawOf) {
					return fmt.Errorf("mission %d trace names check %d outside %d", m.Mission, check.Check, len(m.checkRawOf))
				}
				raw := m.checkRawOf[check.Check]
				family := "check"
				if m.Source.Conditions[raw].Opcode == buildTimeOpcode {
					family = "check-build"
				}
				if family == "check" {
					markNatural(result, m, family, raw, check.Dispatched)
				}
			}
			for _, trigger := range tr.Triggers {
				if trigger.Trigger < 0 || int(trigger.Trigger) >= len(m.triggerRawOf) {
					return fmt.Errorf("mission %d trace names trigger %d outside %d", m.Mission, trigger.Trigger, len(m.triggerRawOf))
				}
				raw := m.triggerRawOf[trigger.Trigger]
				markNaturalEffect(result, m, "trigger", raw, true,
					"ordinary pass recorded the decision, compared pairs and latch transition")
			}
			for _, firing := range tr.Firings {
				for _, instant := range firing.Instants {
					if instant.Instant < 0 || int(instant.Instant) >= len(m.instantRawOf) {
						return fmt.Errorf("mission %d trace names instant %d outside %d", m.Mission, instant.Instant, len(m.instantRawOf))
					}
					raw := m.instantRawOf[instant.Instant]
					markNaturalInstant(result, m, "instant", raw, instant)
					seenInstant[nodeKey{m.Mission, "instant", raw}] = true
					if instant.HasSubCommand {
						markNaturalInstant(result, m, "group", raw, instant)
					}
				}
			}
		}
	}
	result.Natural = len(seenInstant)
	refreshOperationCounts(result)
	return nil
}

func refreshOperationCounts(result *coverageResult) {
	for i := range result.Operations {
		op := &result.Operations[i]
		op.Natural, op.NaturalChanged, op.NaturalNoChange = 0, 0, 0
		for _, row := range result.Nodes {
			if row.Reachable && row.Family == op.Family && row.Opcode == op.Opcode && row.Natural {
				op.Natural++
				if row.NaturalChanged {
					op.NaturalChanged++
				}
				if row.NaturalNoChange {
					op.NaturalNoChange++
				}
			}
		}
	}
}

func markNaturalInstant(result *coverageResult, m *loadedMap, family string, raw int, run sim.ScriptInstantRun) {
	markNatural(result, m, family, raw, run.Supported)
	if i, ok := m.rows[nodeKey{m.Mission, family, raw}]; ok {
		switch run.Outcome {
		case sim.ScriptInstantStateChanged:
			result.Nodes[i].NaturalChanged = true
		case sim.ScriptInstantNoStateChange:
			result.Nodes[i].NaturalNoChange = true
		}
	}
}

func markNatural(result *coverageResult, m *loadedMap, family string, raw int, dispatched bool) {
	if i, ok := m.rows[nodeKey{m.Mission, family, raw}]; ok {
		result.Nodes[i].Natural = true
		result.Nodes[i].Dispatched = result.Nodes[i].Dispatched || dispatched
		if result.Nodes[i].Source == "" {
			result.Nodes[i].Source = "natural"
		}
	}
}

func markNaturalEffect(result *coverageResult, m *loadedMap, family string, raw int, dispatched bool, oracle string) {
	if i, ok := m.rows[nodeKey{m.Mission, family, raw}]; ok {
		result.Nodes[i].Natural = true
		result.Nodes[i].Dispatched = result.Nodes[i].Dispatched || dispatched
		result.Nodes[i].Effect = true
		result.Nodes[i].Source = "natural"
		result.Nodes[i].Oracle = oracle
	}
}

// naturalCommands is a deterministic ordinary-input strategy. The standing
// drive remains part of the run, but every 64 ticks the player's first living
// on-map actor is told either to attack a currently hostile actor or to move to
// a bounded neighbouring cell. No register, latch or script predicate is
// written by the driver.
func naturalCommands(w *sim.World, tick int) []sim.Command {
	if tick%64 != 0 {
		return nil
	}
	entities := w.Entities()
	var actor sim.Entity
	found := false
	for _, e := range entities {
		if e.Owner == sim.SelfSlot && e.Alive() && !e.OffMap {
			actor, found = e, true
			break
		}
	}
	if !found {
		return nil
	}
	if tick%128 == 0 {
		relations := w.Relations()
		for _, e := range entities {
			if e.ID == actor.ID || !e.Alive() || e.OffMap || !relations.Hostile(actor.Owner, e.Owner) {
				continue
			}
			return []sim.Command{sim.Attack(actor.ID, e.ID)}
		}
	}
	b := w.Bounds()
	dx := int32(1)
	if (tick/64)%2 != 0 {
		dx = -1
	}
	x, y := actor.X+dx, actor.Y
	if x < 0 || x >= b.Width {
		x, y = actor.X, actor.Y+dx
	}
	if y < 0 || y >= b.Height {
		x, y = actor.X, actor.Y
	}
	return []sim.Command{sim.MoveTo(actor.ID, sim.CellPoint{X: x, Y: y})}
}

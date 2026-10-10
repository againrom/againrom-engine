package game

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strconv"
	"strings"
	"testing"

	"againrom/pkg/formats/alm"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

// unbuiltCensusRoster is one roster a census row binds a map against.
type unbuiltCensusRoster struct {
	name  string
	party []mapload.PartyMember
	full  bool // every hero-band value the script names resolves
}

// unbuiltCensusNode is one node the binder leaves unbuilt under a roster.
type unbuiltCensusNode struct {
	check    bool
	id       uint32
	opcode   uint32
	values   []string
	triggers []int
}

// unbuiltCensusMap is one map's census under one roster.
type unbuiltCensusMap struct {
	address, roster string
	nodes           []unbuiltCensusNode
	subscript0      *alm.ScriptNode
	register0       *alm.ScriptNode
	groupsAbsent    []uint32
	visible         bool
	triggers        map[int]alm.ScriptTrigger
	src             alm.Script
}

// unbuiltCensusInstantName names the instant opcodes a census row must not
// reach through subscript 0 without stopping the story: the two mission
// outcomes and the two arms that take a unit off the map.
var unbuiltCensusInstantName = map[uint32]string{4: "Mission Win", 5: "Mission Fail", 16: "take unit off map", 32: "take group off map"}

// censusUnbuiltNodes lists, for one map and one roster, every node whose
// reference the binder cannot resolve at mission load, every trigger naming
// it, and what subscript 0 and register 0 are once unbuilt nodes take none
// (TRIG-BIND-010, TRIG-HEROFAIL-078, TRIG-M100-096). It reads the compile's
// Unresolved report, so it measures the same lookups the mission start makes,
// and fails when the compile's omitted nodes are not exactly those.
func censusUnbuiltNodes(m *alm.Map, refs mapload.ScriptRefs) (unbuiltCensusMap, error) {
	var out unbuiltCensusMap
	src, err := m.Script()
	if err != nil {
		return out, err
	}
	_, rep, err := mapload.CompileScript(m, refs)
	if err != nil {
		return out, err
	}
	failedAction, failedCheck := map[uint32][]string{}, map[uint32][]string{}
	for _, u := range rep.Unresolved {
		v := fmt.Sprintf("%s %d", u.Kind, u.Value)
		if u.Condition {
			failedCheck[u.NodeID] = append(failedCheck[u.NodeID], v)
		} else {
			failedAction[u.NodeID] = append(failedAction[u.NodeID], v)
		}
	}
	groups := map[uint32]bool{}
	for _, u := range m.Units {
		groups[u.GroupID] = true
	}
	absent := map[uint32]bool{}
	for _, nodes := range [][]alm.ScriptNode{src.Actions, src.Conditions} {
		for _, n := range nodes {
			for i, typ := range n.Type {
				if typ == 2 && !groups[n.Value[i]] && !absent[n.Value[i]] {
					absent[n.Value[i]] = true
					out.groupsAbsent = append(out.groupsAbsent, n.Value[i])
				}
			}
		}
	}
	sort.Slice(out.groupsAbsent, func(i, j int) bool { return out.groupsAbsent[i] < out.groupsAbsent[j] })
	for id := range failedAction {
		if !slices.Contains(rep.OmittedActions, id) {
			return out, fmt.Errorf("action %d does not resolve and the compile built it", id)
		}
	}
	for id := range failedCheck {
		if !slices.Contains(rep.OmittedChecks, id) {
			return out, fmt.Errorf("check %d does not resolve and the compile built it", id)
		}
	}
	if len(rep.OmittedActions) != len(failedAction) || len(rep.OmittedChecks) != len(failedCheck) {
		return out, fmt.Errorf("the compile omitted %v %v beyond the unresolved nodes", rep.OmittedActions, rep.OmittedChecks)
	}
	live := func(i int, t alm.ScriptTrigger) bool { return t.Left[0] != 0 }
	for _, n := range src.Actions {
		if n.Opcode >= 0x10002 {
			continue
		}
		if vals, ok := failedAction[n.ID]; ok {
			node := unbuiltCensusNode{id: n.ID, opcode: n.Opcode, values: vals}
			for i, t := range src.Triggers {
				if live(i, t) && slices.Contains(t.Acts[:], n.ID) {
					node.triggers = append(node.triggers, i)
				}
			}
			out.nodes = append(out.nodes, node)
			continue
		}
		if out.subscript0 == nil {
			n := n
			out.subscript0 = &n
		}
	}
	for _, n := range src.Conditions {
		if vals, ok := failedCheck[n.ID]; ok {
			node := unbuiltCensusNode{check: true, id: n.ID, opcode: n.Opcode, values: vals}
			for i, t := range src.Triggers {
				if !live(i, t) {
					continue
				}
				for k := range t.Left {
					if t.Left[k] != 0 && t.Right[k] != 0 && (t.Left[k] == n.ID || t.Right[k] == n.ID) {
						node.triggers = append(node.triggers, i)
						break
					}
				}
			}
			out.nodes = append(out.nodes, node)
			continue
		}
		if out.register0 == nil {
			n := n
			out.register0 = &n
		}
	}
	out.src = src
	out.triggers = map[int]alm.ScriptTrigger{}
	for _, n := range out.nodes {
		if len(n.triggers) != 0 {
			out.visible = true
		}
		for _, i := range n.triggers {
			out.triggers[i] = src.Triggers[i]
		}
	}
	return out, nil
}

func (c unbuiltCensusMap) lines() []string {
	head := fmt.Sprintf("%s roster=%s unbuilt=%d", c.address, c.roster, len(c.nodes))
	if len(c.groupsAbsent) != 0 {
		head += fmt.Sprintf(" groups-without-placement=%v", c.groupsAbsent)
	}
	out := []string{head}
	describe := func(n *alm.ScriptNode) string {
		if n == nil {
			return "none"
		}
		var params []string
		for i, typ := range n.Type {
			if typ != 0 {
				params = append(params, fmt.Sprintf("t%d=%d", typ, n.Value[i]))
			}
		}
		return fmt.Sprintf("id=%d opcode=%d %q [%s]", n.ID, n.Opcode, n.Label, strings.Join(params, " "))
	}
	for _, n := range c.nodes {
		kind := "action"
		if n.check {
			kind = "check"
		}
		out = append(out, fmt.Sprintf("  unbuilt %s id=%d opcode=%d names %s; named by triggers %v", kind, n.id, n.opcode, strings.Join(n.values, ", "), n.triggers))
	}
	if len(c.nodes) != 0 {
		out = append(out, "  subscript 0: "+describe(c.subscript0), "  register 0: "+describe(c.register0))
	}
	var named []int
	for i := range c.triggers {
		named = append(named, i)
	}
	sort.Ints(named)
	for _, i := range named {
		t := c.triggers[i]
		out = append(out, fmt.Sprintf("  trigger %d %q once=%d pairs=%v/%v cmp=%v acts=%v", i, t.Name, t.Once, t.Left, t.Right, t.Cmp, t.Acts))
		for _, n := range c.src.Conditions {
			if slices.Contains(t.Left[:], n.ID) || slices.Contains(t.Right[:], n.ID) {
				out = append(out, "    check "+describe(&n))
			}
		}
		for _, n := range c.src.Actions {
			if slices.Contains(t.Acts[:], n.ID) {
				out = append(out, "    action "+describe(&n))
			}
		}
	}
	return out
}

// censusMissionNumber is the mission a campaign map name such as 100.alm
// starts, or 0 for a name that is not a number.
func censusMissionNumber(name string) int {
	n, err := strconv.Atoi(strings.TrimSuffix(strings.ToLower(name), ".alm"))
	if err != nil {
		return 0
	}
	return n
}

// unbuiltCensusRosters is the primary alone in each of the four archetypes,
// and a full roster in which every hero-band value the script names resolves.
func unbuiltCensusRosters(f *FrontEnd) []unbuiltCensusRoster {
	var out []unbuiltCensusRoster
	for _, sex := range []int{0, 1} {
		for _, class := range []int{0, 1} {
			party := f.ChargenParty(ui.ChargenResult{Name: "Census", Choices: []int{sex, class, 0}, Stats: []int{30, 30, 30, 30}})
			out = append(out, unbuiltCensusRoster{name: fmt.Sprintf("primary-%s-%s", []string{"male", "female"}[sex], []string{"fighter", "mage"}[class]), party: party})
		}
	}
	return append(out, unbuiltCensusRoster{name: "full", party: out[0].party, full: true})
}

// TestReleaseScriptUnbuiltNodeCensus is the census of every campaign map in
// the installed scenario archive: which nodes the ROM1 binder leaves unbuilt
// under each roster, which triggers name them, and what subscript 0 and
// register 0 are. It pins the maps whose triggers name an unbuilt node and
// the maps where such a slot reaches a Mission Win, a Mission Fail or an
// off-map arm through subscript 0. AGAINROM_UNBUILT_CENSUS names a directory
// the census text is written to.
func TestReleaseScriptUnbuiltNodeCensus(t *testing.T) {
	f := releaseFront(t)
	maps := ArchiveMaps(f.Archives.Containers)
	names := slices.Clone(maps.Names())
	sort.Strings(names)
	var text []string
	visible, reached := map[string]bool{}, map[string]bool{}
	for _, name := range names {
		b, err := maps.Read(name)
		if err != nil {
			t.Fatal(err)
		}
		m, err := alm.Open(b)
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		mapload.WithdrawBorderPlacements(m)
		for _, roster := range unbuiltCensusRosters(f) {
			refs := campaignScriptRefs(m, f.Table, roster.party, censusMissionNumber(name))
			if roster.full {
				src, err := m.Script()
				if err != nil {
					t.Fatal(err)
				}
				if refs.Roles == nil {
					refs.Roles = map[uint32]sim.EntityID{}
				}
				refs.Hero, refs.HasHero = mapload.PartyEntity(m, 0), true
				refs.Companion, refs.HasCompanion = mapload.PartyEntity(m, 0)+1, true
				for _, nodes := range [][]alm.ScriptNode{src.Actions, src.Conditions} {
					for _, n := range nodes {
						for i, typ := range n.Type {
							if v := n.Value[i]; typ == 4 && v > 10002 && v <= 11000 {
								refs.Roles[v] = mapload.PartyEntity(m, 0) + sim.EntityID(v-10001)
							}
						}
					}
				}
			}
			c, err := censusUnbuiltNodes(m, refs)
			if err != nil {
				t.Fatalf("%s: %v", name, err)
			}
			c.address, c.roster = name, roster.name
			if len(c.nodes) != 0 || len(c.groupsAbsent) != 0 {
				text = append(text, c.lines()...)
			}
			if c.visible {
				visible[name] = true
			}
			if s0 := c.subscript0; s0 != nil {
				for _, n := range c.nodes {
					if what, outcome := unbuiltCensusInstantName[s0.Opcode]; outcome && !n.check && len(n.triggers) != 0 {
						reached[name+" "+what] = true
					}
				}
			}
		}
	}
	var changed []string
	for name := range visible {
		changed = append(changed, name)
	}
	sort.Strings(changed)
	var outcomes []string
	for r := range reached {
		outcomes = append(outcomes, r)
	}
	sort.Strings(outcomes)
	text = append(text, fmt.Sprintf("maps whose triggers name an unbuilt node: %v", changed),
		fmt.Sprintf("maps whose subscript 0 is an outcome or off-map arm reached by an unbuilt slot: %v", outcomes))
	for _, line := range text {
		t.Log(line)
	}
	if dir := os.Getenv("AGAINROM_UNBUILT_CENSUS"); dir != "" {
		path := filepath.Join(dir, "census-"+filepath.Base(os.Getenv("AGAINROM_ASSETS"))+".txt")
		if err := os.WriteFile(path, []byte(strings.Join(text, "\n")+"\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

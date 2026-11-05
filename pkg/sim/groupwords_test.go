package sim

import (
	"bytes"
	"crypto/sha256"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

var (
	updateGroupWords = flag.Bool("update-group-words", false,
		"rewrite testdata/groupwords.txt from the current code")
	dumpGroupWords = flag.String("group-words-dump", "",
		"write each Group words record's whole transcript into this directory")
)

// gwParty and gwHostile are the record fixtures' selectors: the party's two
// SelfSlot units in Group 2, two owner-9 guards in Group 3 and a lone owner-9
// guard in Group 4, far from both.
const (
	gwParty   uint32 = 2
	gwHostile uint32 = 3
	gwFar     uint32 = 4
)

// gwWorld is a World of the first or the second game holding the party at
// party and party+1, the owner-9 pair near the middle and the lone guard far
// off, one usable structure beside the party, slot 9 and the party hostile
// to each other. A named World's script names the owner-9 pair's Group.
func gwWorld(t *testing.T, second bool, party cell, named bool) *World {
	t.Helper()
	unit := func(id EntityID, owner, group uint32, x, y int32, sight uint8) Entity {
		e := engFighter(id, owner, x, y)
		e.Group, e.ScanRange, e.HP, e.MaxHP, e.Speed = group, sight, 60, 60, 50
		return e
	}
	ents := []Entity{
		unit(1, SelfSlot, gwParty, party.x, party.y, 6),
		unit(2, SelfSlot, gwParty, party.x+1, party.y, 7),
		unit(3, 9, gwHostile, 24, 24, 8),
		unit(4, 9, gwHostile, 26, 25, 9),
		unit(5, 9, gwFar, 40, 8, 6),
	}
	rel := engRel(t, [3]uint32{9, SelfSlot, 1}, [3]uint32{SelfSlot, 9, 1})
	var instants []ScriptInstant
	if named {
		instants = []ScriptInstant{{Op: ScriptInstantMessage, Group: gwHostile, HasGroup: true}}
	}
	var s *Script
	if second {
		s = secondGameScript(t, nil, instants, nil)
	} else if named {
		var err error
		if s, err = NewScript(nil, instants, nil); err != nil {
			t.Fatal(err)
		}
	}
	use := Structure{ID: 0, Kind: 15, Col: party.x + 4, Row: party.y, Width: 1, Height: 1, Attach: 1, Blocking: 1}
	// The map is wide enough for a roam's 20-cell step from the pair.
	w, err := NewStructuredWorld(7, Bounds{Width: 64, Height: 64}, ModeCanonical, Terrain{}, ents, s, rel, nil, nil, nil, GhostTemplate{}, []Structure{use})
	if err != nil {
		t.Fatal(err)
	}
	if playsSecondGame(w) != second {
		t.Fatalf("fixture: second game %v, want %v", playsSecondGame(w), second)
	}
	return w
}

// groupWordsCase is one order a Group words writer sets: the party's first
// cell, the order given before the first recorded step, a second order
// given before step at, and the steps recorded.
type groupWordsCase struct {
	name  string
	party cell
	ticks int
	start func(t *testing.T, w *World) []Command
	named bool
	at    int
	then  func(t *testing.T, w *World) []Command
}

// gwAll runs each start in turn and gives every command they return.
func gwAll(starts ...func(*testing.T, *World) []Command) func(*testing.T, *World) []Command {
	return func(t *testing.T, w *World) []Command {
		var cmds []Command
		for _, s := range starts {
			cmds = append(cmds, s(t, w)...)
		}
		return cmds
	}
}

func gwWake(_ *testing.T, w *World) []Command { wakeGroups(w); return nil }

// gwEmptyGuard runs one Guard decision of the owner-9 pair's Group with an
// empty member list and its latch set, as only a direct decision reaches it.
func gwEmptyGuard(t *testing.T, w *World) []Command {
	g := regGroup(t, w, 9, gwHostile)
	members := g.Members
	g.Members = nil
	g.AI[0x30] = 1
	w.savedDecision(g, orderGuard, nil)
	g.Members = members
	return nil
}

func gwTag(kind func(EntityID, CellPoint, uint32) Command, at CellPoint) func(*testing.T, *World) []Command {
	return func(*testing.T, *World) []Command { return []Command{kind(1, at, 1), kind(2, at, 1)} }
}

func gwStance(order uint8) func(*testing.T, *World) []Command {
	return func(*testing.T, *World) []Command {
		return []Command{GroupStance(1, int32(order), 1), GroupStance(2, int32(order), 1)}
	}
}

func gwInstant(in ScriptInstant) func(*testing.T, *World) []Command {
	return func(_ *testing.T, w *World) []Command { w.runInstant(in); return nil }
}

func gw(name string, party cell, ticks int, start func(*testing.T, *World) []Command) groupWordsCase {
	return groupWordsCase{name: name, party: party, ticks: ticks, start: start}
}

func gwNative(_ *testing.T, w *World) []Command {
	w.savedGroups = w.constructedGroups(true)
	return nil
}

var groupWordsCases = []groupWordsCase{
	gw("constructed", cell{6, 6}, 120, nil),
	gw("constructed-woken", cell{6, 6}, 120, gwWake),
	gw("native-records", cell{6, 6}, 120, gwNative),
	gw("move", cell{6, 6}, 120, gwTag(GroupMoveTo, CellPoint{X: 20, Y: 20})),
	gw("move-apart", cell{6, 6}, 60, func(*testing.T, *World) []Command {
		return []Command{GroupMoveTo(1, CellPoint{X: 12, Y: 30}, 1), MoveTo(2, CellPoint{X: 30, Y: 4})}
	}),
	gw("swarm", cell{6, 6}, 120, gwTag(GroupSwarmTo, CellPoint{X: 22, Y: 22})),
	gw("patrol", cell{6, 6}, 120, gwTag(GroupPatrolTo, CellPoint{X: 16, Y: 6})),
	gw("guard", cell{17, 18}, 160, gwStance(orderGuard)),
	gw("stand", cell{17, 18}, 120, gwStance(orderStandGround)),
	gw("engaged", cell{19, 20}, 160, nil),
	gw("structure-use", cell{6, 6}, 120, func(*testing.T, *World) []Command { return []Command{UseStructure(1, 0)} }),
	gw("hand-over", cell{6, 6}, 120, func(_ *testing.T, w *World) []Command {
		w.handOver(indexOfEntity(w.entities, 5), SelfSlot)
		return nil
	}),
	{name: "named", party: cell{6, 6}, ticks: 120, named: true},
	{name: "named-native", party: cell{17, 18}, ticks: 120, named: true, start: gwNative},
	{name: "move-then-guard", party: cell{6, 6}, ticks: 160, start: gwTag(GroupMoveTo, CellPoint{X: 16, Y: 16}), at: 12, then: gwStance(orderGuard)},
	{name: "swarm-then-stand", party: cell{6, 6}, ticks: 120, start: gwTag(GroupSwarmTo, CellPoint{X: 18, Y: 18}), at: 20, then: gwStance(orderStandGround)},
	{name: "latch-empty", party: cell{17, 18}, ticks: 80, start: gwAll(gwWake, gwEmptyGuard)},
	gw("roam", cell{17, 18}, 400, gwAll(gwWake, gwInstant(lcGroupOrder(gwHostile, 0, int32(orderRoam))))),
	gw("roam-far", cell{6, 6}, 240, gwInstant(lcGroupOrder(gwHostile, 0, int32(orderRoam)))),
	gw("script-guard", cell{6, 6}, 160, gwInstant(lcGroupOrder(gwHostile, 0, int32(orderGuard)))),
	gw("script-guard-woken", cell{17, 18}, 160, gwAll(gwWake, gwInstant(lcGroupOrder(gwHostile, 0, int32(orderGuard))))),
	gw("script-swarm", cell{6, 6}, 120, gwInstant(lcGroupOrder(gwHostile, 0, int32(orderSwarm), 10, 12))),
	gw("script-stand", cell{6, 6}, 80, gwInstant(lcGroupOrder(gwHostile, 0, int32(orderStandGround)))),
	gw("script-move", cell{6, 6}, 120, gwInstant(lcGroupOrder(gwHostile, 0, int32(orderMove), 12, 8))),
	gw("script-swarm2", cell{6, 6}, 120, gwInstant(lcGroupOrder(gwHostile, 0, int32(orderSwarm2), 9, 7))),
	gw("script-patrol", cell{6, 6}, 120, gwInstant(lcGroupOrder(gwHostile, 0, subCommandPatrol, 30, 34))),
	gw("script-attack", cell{6, 6}, 120, gwInstant(lcGroupOrder(gwHostile, 1, subCommandAttack))),
	gw("stop", cell{17, 18}, 120, func(_ *testing.T, w *World) []Command { w.stopGroupMembers(gwHostile); return nil }),
	gw("activity-clear", cell{17, 18}, 120, gwInstant(ScriptInstant{Op: 39, Group: gwHostile, HasGroup: true})),
}

// groupWordsRecord steps w and writes, for the World before the first step
// and after each step, its hash and random state, and every registry Group
// whose AI words changed since the line before.
func groupWordsRecord(t *testing.T, w *World, c groupWordsCase, cmds []Command) []byte {
	var b bytes.Buffer
	held := map[uint32][76]byte{}
	groups := func() {
		if w.savedGroups == nil {
			return
		}
		seen := map[uint32]bool{}
		for _, g := range w.savedGroups.Groups {
			seen[g.ID] = true
			if prev, ok := held[g.ID]; ok && prev == g.AI {
				continue
			}
			held[g.ID] = g.AI
			fmt.Fprintf(&b, "  group %d selector %d owner %d members %d ai %x\n", g.ID, g.Selector, g.Owner.Owner, len(g.Members), g.AI)
		}
		for id := range held {
			if !seen[id] {
				delete(held, id)
				fmt.Fprintf(&b, "  gone %d\n", id)
			}
		}
	}
	fmt.Fprintf(&b, "start hash %016x rng %016x\n", w.Hash(), w.rng.state)
	groups()
	for tick := range c.ticks {
		switch {
		case tick == 0:
			Step(w, cmds)
		case c.then != nil && tick == c.at:
			Step(w, c.then(t, w))
		default:
			Step(w, nil)
		}
		fmt.Fprintf(&b, "tick %d hash %016x rng %016x\n", tick, w.Hash(), w.rng.state)
		groups()
	}
	return b.Bytes()
}

// Every Group words writer, for each game's profile: each tick's Group AI
// blocks, World hash and random state over a move, a swarm, a patrol, a
// guard and a stand order, the constructed guard and Stand Ground Groups
// with and without their native records, an engaged guard's notice, a
// structure use, a hand-over, a roam and the script's Group orders, a stop
// and the second game's activity change are the recorded ones.
func TestGroupWordsRecord(t *testing.T) {
	var got strings.Builder
	for _, c := range groupWordsCases {
		for _, second := range []bool{false, true} {
			game := "first"
			if second {
				game = "second"
			}
			w := gwWorld(t, second, c.party, c.named)
			var cmds []Command
			if c.start != nil {
				cmds = c.start(t, w)
			}
			record := groupWordsRecord(t, w, c, cmds)
			if *dumpGroupWords != "" {
				if err := os.WriteFile(filepath.Join(*dumpGroupWords, c.name+"-"+game+".txt"), record, 0o644); err != nil {
					t.Fatal(err)
				}
			}
			fmt.Fprintf(&got, "%s %s ticks %d record %x final %016x\n", c.name, game, c.ticks, sha256.Sum256(record), w.Hash())
		}
	}
	path := filepath.Join("testdata", "groupwords.txt")
	if *updateGroupWords {
		if err := os.MkdirAll("testdata", 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(got.String()), 0o644); err != nil {
			t.Fatal(err)
		}
		return
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if got.String() == string(want) {
		return
	}
	gl, wl := strings.Split(got.String(), "\n"), strings.Split(string(want), "\n")
	for i := range max(len(gl), len(wl)) {
		var g, w string
		if i < len(gl) {
			g = gl[i]
		}
		if i < len(wl) {
			w = wl[i]
		}
		if g != w {
			t.Errorf("record line %d is\n  %s\nwant\n  %s", i+1, g, w)
		}
	}
}

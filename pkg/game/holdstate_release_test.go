package game

import (
	"encoding/binary"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"testing"

	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
)

// holdStateFrames is what a fresh process must reproduce from the SAV in Dir:
// the World digest after each of the ticks that follow the LOAD, and the two
// warriors' actor states straight after it.
type holdStateFrames struct {
	Dir     string
	Heroes  []sim.EntityID
	States  []uint8
	Digests []uint64
}

const holdStateTicks = 240

// Hold Position pressed on two warriors stores actor state 0xc for each, on the
// entity and in its SAV order (AI-351). The mission is saved through F2 and the
// file holds U50 = 0xc for both; a main-menu LOAD in this process and another
// in a fresh process restore 0xc, and the next action, the first warrior's
// blows on the creature beside him, follows tick for tick.
func TestReleaseHoldStoresStateCSAVColdProcess(t *testing.T) {
	if path := os.Getenv("AGAINROM_HOLDSTATE_CHILD"); path != "" {
		holdStateChild(t, path)
		return
	}
	arena := openLoadedCycleArena(t, 2)
	arena.front.ConfigureSaveSeams(arena.app, SaveStore{Dir: t.TempDir()}, OriginalStore{}, nil)
	if err := arena.app.HeadlessKey("e"); err != nil {
		t.Fatal(err)
	}
	if got := arena.app.HeadlessSelection(); len(got) != 2 {
		t.Fatalf("the E key selected %v, want both warriors", got)
	}
	holdPress(t, arena, arena.heroes[0])
	_, orders, _ := arena.live.world.SavedGroups()
	for _, id := range arena.heroes {
		if e := arena.get(id); e.ActorState != 0xc {
			t.Fatalf("after Hold Position warrior %d holds state %#x, want 0xc", id, e.ActorState)
		}
		i := slices.IndexFunc(orders, func(o sim.SavedActorOrder) bool { return o.Entity == id })
		if i < 0 || orders[i].State != 0xc {
			t.Fatalf("after Hold Position warrior %d's order is %v, want state 0xc", id, i >= 0)
		}
	}
	dir, raw := castOrderF2Save(t, arena.front, arena.app, "hold state")
	holdStateWire(t, raw, arena.heroes)
	cold, want := holdStateLoad(t, dir, arena.heroes)
	full := arena.get(arena.east).HP
	for range holdStateTicks {
		arena.live.tick()
		cold.live.tick()
		if !midStrikeArenaSame(arena.live.world, cold.live.world) {
			t.Fatal("the cold LOAD's next action differs from the uninterrupted session")
		}
		want.Digests = append(want.Digests, cold.live.world.Hash())
	}
	if hp := arena.get(arena.east).HP; hp >= full {
		t.Fatalf("%d ticks on Hold Position beside the creature: its health stayed %d", holdStateTicks, hp)
	}
	holdStateFreshProcess(t, want)
	t.Logf("Hold, F2, cold LOAD here and in a fresh process: U50 = 0xc for warriors %v; the creature's health fell from %d to %d", arena.heroes, full, arena.get(arena.east).HP)
}

// holdStateWire requires U50 = 0xc in the SAV for every member.
func holdStateWire(t *testing.T, raw []byte, members []sim.EntityID) {
	t.Helper()
	doc, err := sav.DecodeDocumentData(raw)
	if err != nil {
		t.Fatal(err)
	}
	actions, err := readCurrentActions(&doc)
	if err != nil || actions == nil {
		t.Fatal("action supplement", err)
	}
	for _, id := range members {
		if got := binary.LittleEndian.Uint32(savedRecordRawForTest(t, castOrderRecord(t, doc, actions, id), "U50")); got != 0xc {
			t.Fatalf("the SAV holds U50 = %#x for member %d, want 0xc", got, id)
		}
	}
}

// holdStateLoad loads the one SAV in dir through the main menu and requires
// state 0xc for every member.
func holdStateLoad(t *testing.T, dir string, members []sim.EntityID) (*FrontEnd, holdStateFrames) {
	t.Helper()
	cold, _ := castOrderSession(t, dir)
	want := holdStateFrames{Dir: dir, Heroes: members}
	for _, id := range members {
		e, _ := cold.live.entity(id)
		want.States = append(want.States, e.ActorState)
		if e.ActorState != 0xc {
			t.Fatalf("cold LOAD restored member %d at state %#x, want 0xc", id, e.ActorState)
		}
	}
	return cold, want
}

// standStateColdProcess writes raw, loads it here and in a fresh process, and
// requires both to keep state 0xc and one digest per tick.
func standStateColdProcess(t *testing.T, raw []byte, members []sim.EntityID) {
	t.Helper()
	holdStateWire(t, raw, members)
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "stand state.sav"), raw, 0600); err != nil {
		t.Fatal(err)
	}
	cold, want := holdStateLoad(t, dir, members)
	for range holdStateTicks {
		cold.live.tick()
		want.Digests = append(want.Digests, cold.live.world.Hash())
	}
	holdStateFreshProcess(t, want)
}

func holdStateFreshProcess(t *testing.T, want holdStateFrames) {
	t.Helper()
	encoded, err := json.Marshal(want)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "hold-state.json")
	if err := os.WriteFile(path, encoded, 0600); err != nil {
		t.Fatal(err)
	}
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	child := exec.Command(executable, "-test.run=^TestReleaseHoldStoresStateCSAVColdProcess$", "-test.v")
	child.Env = append(os.Environ(), "AGAINROM_HOLDSTATE_CHILD="+path)
	if output, err := child.CombinedOutput(); err != nil {
		t.Fatalf("fresh-process LOAD: %v\n%s", err, output)
	}
}

func holdStateChild(t *testing.T, path string) {
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var want holdStateFrames
	if err := json.Unmarshal(raw, &want); err != nil {
		t.Fatal(err)
	}
	f, _ := castOrderSession(t, want.Dir)
	for k, id := range want.Heroes {
		if e, _ := f.live.entity(id); e.ActorState != want.States[k] {
			t.Fatalf("fresh process: warrior %d at state %#x, want %#x", id, e.ActorState, want.States[k])
		}
	}
	for k, digest := range want.Digests {
		f.live.tick()
		if got := f.live.world.Hash(); got != digest {
			t.Fatalf("fresh process: tick %d digest %#x, want %#x", k+1, got, digest)
		}
	}
}

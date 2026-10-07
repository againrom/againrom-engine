package game

import (
	"os"
	"path/filepath"
	"testing"

	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
)

// formationSaveWords maps each live Player-1 Human's archive index to its
// saved speed word, and names the overloaded ones.
func formationSaveWords(t *testing.T, raw []byte) (words map[uint16]int16, overloaded []uint16) {
	t.Helper()
	words = map[uint16]int16{}
	for _, a := range formationHumans(t, raw) {
		ls := a.Character.LoadState
		words[a.ArchiveIndex] = ls.Speed
		if ls.Capacity > 0 && ls.Load >= ls.Capacity {
			overloaded = append(overloaded, a.ArchiveIndex)
		}
	}
	return words, overloaded
}

func formationHumans(t *testing.T, raw []byte) []sav.ActorRecord {
	t.Helper()
	f, err := sav.Open(raw)
	if err != nil {
		t.Fatal(err)
	}
	g, err := f.ActorGraph()
	if err != nil {
		t.Fatal(err)
	}
	var out []sav.ActorRecord
	for _, a := range g.Actors {
		if a.Class == "Human" && !a.Dead() && a.OwnerSlot == 1 && a.Character.LoadState.Present {
			out = append(out, a)
		}
	}
	return out
}

// formationOverload loads the slowest Player-1 Human of an original save to
// twice its capacity and writes its speed word and mover byte by SAV-1116.
func formationOverload(t *testing.T, raw []byte) []byte {
	t.Helper()
	var slow sav.ActorRecord
	for _, a := range formationHumans(t, raw) {
		if slow.Class == "" || a.Character.LoadState.Speed < slow.Character.LoadState.Speed {
			slow = a
		}
	}
	doc, err := sav.DecodeDocumentData(raw)
	if err != nil {
		t.Fatal(err)
	}
	// The record is the one Human with the chosen actor's runtime id.
	var r *sav.DocumentRecordData
	ls := slow.Character.LoadState
	for i := range doc.Objects {
		o := &doc.Objects[i]
		if o.Class != "Human" || humanSpeedValue(t, o, "RuntimeID") != slow.RuntimeID || int16(humanSpeedValue(t, o, "Speed")) != ls.Speed {
			continue
		}
		if r != nil {
			t.Fatalf("two Human records match archive %d", slow.ArchiveIndex)
		}
		r = o
	}
	if r == nil {
		t.Fatalf("no Human record matches archive %d", slow.ArchiveIndex)
	}
	v := func(name string) uint32 { return humanSpeedValue(t, r, name) }
	acc := int32(v("Inventory20"))
	capacity := int32(int16(v("Capacity")))
	own := int16(2*capacity - acc/2)
	speed := humanSpeedLaw(uint16(v("Body")), uint16(v("Reaction")), humanSpeedRaw(r, "UD4"), uint16(v("T0E")), own, acc)
	if speed >= ls.Speed {
		t.Fatalf("overload left speed %d from %d", speed, ls.Speed)
	}
	savedObjectSetValue(r, "U8E", uint32(uint16(own)))
	savedObjectSetValue(r, "U90", uint32(uint16(int32(own)+acc/2)))
	savedObjectSetValue(r, "Speed", uint32(uint16(speed)))
	humanSpeedRaw(r, "U154")[10] = byte(speed)
	out, err := sav.EncodeDocumentData(doc)
	if err != nil {
		t.Fatal(err)
	}
	if w, over := formationSaveWords(t, out); w[slow.ArchiveIndex] != speed || len(over) != 1 {
		t.Fatalf("edited archive %d speed %d overloaded %v", slow.ArchiveIndex, w[slow.ArchiveIndex], over)
	}
	return out
}

// An original-written mission whose party holds an overloaded Human: after
// LOAD and one formation move, every member moves at the minimum of the
// members' saved speed words, and SAVE writes that term to the group record.
func TestReleaseFormationMovesAtTheSlowestSavedSpeed(t *testing.T) {
	root := os.Getenv("AGAINROM_ASSETS")
	if root == "" {
		t.Skip("no AGAINROM_ASSETS: the formation speed witness needs a lawful install")
	}
	// EN game0019 holds Brian at load 457, capacity 411. A root without
	// that save overloads the slowest Human of game0002 in a TempDir copy.
	name := "game0019.sav"
	raw, err := os.ReadFile(filepath.Join(root, name))
	if err == nil {
		if _, over := formationSaveWords(t, raw); len(over) == 0 {
			err = os.ErrNotExist
		}
	}
	if err != nil {
		name = "game0002.sav"
		if raw, err = os.ReadFile(filepath.Join(root, name)); err != nil {
			t.Fatal(err)
		}
		raw = formationOverload(t, raw)
	}
	words, overloaded := formationSaveWords(t, raw)
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, name), raw, 0o644); err != nil {
		t.Fatal(err)
	}
	f := releaseFront(t)
	f.SetDeterministicFrames(true)
	app := f.App("formation-speed")
	app.Layout(1024, 768)
	store := SaveStore{Dir: t.TempDir()}
	save, list, load := f.SaveSeams(store, OriginalStore{Dir: dir}, nil)
	app.SetSaveSeams(save, list, load)
	groundAppLoad(t, app, list, name)

	// After LOAD the readout states the group record's term for every entity.
	termed := 0
	for _, d := range f.live.entityDraws() {
		term := int(f.live.world.GroupRateTerm(sim.EntityID(d.ID)))
		if d.GroupSpeed != term {
			t.Errorf("%s entity %d readout group speed %d, rate term %d", name, d.ID, d.GroupSpeed, term)
		}
		if term != 0 {
			termed++
		}
	}
	t.Logf("%s: %d entities read a nonzero saved group term after LOAD", name, termed)

	// Formation On: the order moves in formation however far apart the
	// members stand (MOVE-GATE-035).
	for press := 0; press < 3; press++ {
		if mode, _ := f.live.world.CommandFormationMode(sim.SelfSlot); mode == 1 {
			break
		}
		f.live.cycleFormation()
		if err := app.HeadlessStep(); err != nil {
			t.Fatal(err)
		}
	}
	if mode, _ := f.live.world.CommandFormationMode(sim.SelfSlot); mode != 1 {
		t.Fatalf("formation mode %d, want On", mode)
	}
	if err := app.HeadlessKey("e"); err != nil {
		t.Fatal(err)
	}
	var members []sim.EntityID
	want := int16(0xfa)
	for _, id := range app.HeadlessSelection() {
		e, ok := f.live.entity(sim.EntityID(id))
		w, human := words[e.SourceBinding.ArchiveIndex]
		if !ok || !human {
			t.Fatalf("selection holds entity %d with no saved Player-1 Human", id)
		}
		members = append(members, e.ID)
		want = min(want, w)
	}
	if len(members) < 2 || len(overloaded) == 0 {
		t.Fatalf("%s selects %d Humans, %d overloaded", name, len(members), len(overloaded))
	}
	// One formation order through the ground click, retried at another
	// ground cell while a member's formation cell refuses the destination.
	ordered := false
	for _, pt := range [][2]int{{0, 0}, {400, 200}, {600, 250}, {300, 300}, {500, 150}, {700, 350}} {
		x, y, err := app.HeadlessGroundPoint()
		if err != nil {
			t.Fatal(err)
		}
		if pt != [2]int{} {
			x, y = pt[0], pt[1]
		}
		for _, action := range []string{"press", "release"} {
			if err := app.HeadlessPointer(action, x, y); err != nil {
				t.Fatal(err)
			}
		}
		if err := app.HeadlessStep(); err != nil {
			t.Fatal(err)
		}
		ordered = true
		for _, id := range members {
			e, _ := f.live.entity(id)
			ordered = ordered && e.HasTarget
		}
		if ordered {
			t.Logf("order at window (%d,%d)", x, y)
			break
		}
	}
	if !ordered {
		t.Fatal("no ground cell gave every member a destination")
	}
	started := map[sim.EntityID]bool{}
	for tick := 0; tick < 200 && len(started) < len(members); tick++ {
		if err := app.HeadlessStep(); err != nil {
			t.Fatal(err)
		}
		for _, id := range members {
			if e, _ := f.live.entity(id); e.TransitTotal != 0 && !started[id] {
				started[id] = true
				if got, _ := f.live.world.RateSpeed(id); got != int32(want) {
					t.Errorf("%s member %d rate term %d, want the saved minimum %d", name, id, got, want)
				}
			}
		}
	}
	drawn := map[sim.EntityID]int{}
	for _, d := range f.live.entityDraws() {
		drawn[sim.EntityID(d.ID)] = d.GroupSpeed
	}
	for _, id := range members {
		if got, ok := drawn[id]; !ok || got != int(want) {
			t.Errorf("%s member %d readout group speed %d (drawn %v), want the saved minimum %d", name, id, got, ok, want)
		}
	}
	if len(started) < len(members) {
		for _, id := range members {
			e, _ := f.live.entity(id)
			t.Logf("member %d at (%d,%d) target %v (%d,%d) state %d group %d", id, e.X, e.Y, e.HasTarget, e.TargetX, e.TargetY, e.ActorState, e.GroupSpeed)
		}
		t.Fatalf("%d of %d members started a transit", len(started), len(members))
	}

	// The next SAVE writes the group record's term.
	written, err := save(true)
	if err != nil {
		t.Fatal(err)
	}
	out, err := store.Read(written)
	if err != nil {
		t.Fatal(err)
	}
	file, err := sav.Open(out)
	if err != nil {
		t.Fatal(err)
	}
	graph, err := file.ActorGraph()
	if err != nil {
		t.Fatal(err)
	}
	outWords, _ := formationSaveWords(t, out)
	terms := 0
	for _, g := range graph.Groups {
		held := 0
		for _, ai := range g.Members {
			if _, ok := outWords[ai]; ok {
				held++
			}
		}
		if held != len(members) {
			continue
		}
		terms++
		if g.AI[0x44] != uint8(want) {
			t.Errorf("SAVE group term %d, want %d", g.AI[0x44], want)
		}
	}
	if terms != 1 {
		t.Fatalf("SAVE holds %d groups of all %d members", terms, len(members))
	}
	t.Logf("%s: %d members, %d overloaded, term %d", name, len(members), len(overloaded), want)
}

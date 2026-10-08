package game

import (
	"os"
	"path/filepath"
	"testing"

	"againrom/pkg/data"
	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

// raisedGhostRecord returns the SAV record of the one placement-free Unit of
// TypeID typ.
func raisedGhostRecord(t *testing.T, raw []byte, typ int32) sav.ActorRecord {
	t.Helper()
	file, err := sav.Open(raw)
	if err != nil {
		t.Fatal(err)
	}
	graph, err := file.ActorGraph()
	if err != nil {
		t.Fatal(err)
	}
	var out []sav.ActorRecord
	for _, a := range graph.Actors {
		if a.Class == "Unit" && int32(a.TypeID) == typ && a.MapUnitID == 0 {
			out = append(out, a)
		}
	}
	if len(out) != 1 {
		t.Fatalf("SAV holds %d placement-free Units of type %d, want 1", len(out), typ)
	}
	return out[0]
}

// requireLiveRaisedGhost requires exactly one living, placement-free actor of
// TypeID typ owned by the player.
func requireLiveRaisedGhost(t *testing.T, f *FrontEnd, typ int32) {
	t.Helper()
	n := 0
	for _, e := range f.live.world.Entities() {
		if e.TypeID == typ && e.MapUnitID == 0 && e.Owner == sim.SelfSlot && e.Alive() {
			n++
		}
	}
	if n != 1 {
		t.Fatalf("live world holds %d raised Ghosts, want 1", n)
	}
}

// TestReleaseRaisedGhostSurvivesSAV raises a Ghost with Control Spirit in
// mission 10 and writes SAV through the ordinary producer. The Ghost record
// names the Units row called Ghost and its face; cold LOAD keeps the Ghost and
// a second SAVE keeps the row. The same SAV with the Ghost's row and face
// zeroed, as the producer wrote them before it named the row, loads too.
func TestReleaseRaisedGhostSurvivesSAV(t *testing.T) {
	f := releaseFront(t)
	f.Options = OptionsStore{}
	f.SetDeterministicFrames(true)
	party := f.ChargenParty(ui.ChargenResult{Name: "Ghost witness", Choices: []int{0, 1, 0}, Stats: []int{20, 25, 40, 40}})
	party[0].KnownSpells |= 1 << 25
	f.Carried = party
	if err := f.App("raised Ghost SAV").OpenMission(f.MissionOpenerWith(10, f.Carried)); err != nil {
		t.Fatal(err)
	}
	w := f.live.world
	typ := w.Ghost().TypeID
	var mage, victim sim.Entity
	for _, e := range w.Entities() {
		switch {
		case mage.ID == 0 && e.Owner == sim.SelfSlot && e.Humanoid:
			mage = e
		case victim.ID == 0 && e.Owner != sim.SelfSlot && e.Alive() && !e.Humanoid && e.TypeID != typ:
			victim = e
		}
	}
	if mage.ID == 0 || victim.ID == 0 {
		t.Fatalf("mission 10 has no mage %d or creature %d", mage.ID, victim.ID)
	}
	placed := false
	for _, d := range [][2]int32{{1, 0}, {-1, 0}, {0, 1}, {0, -1}} {
		if w.HeadlessPlace(victim.ID, mage.X+d[0], mage.Y+d[1]) == nil {
			placed = true
			break
		}
	}
	if !placed {
		t.Fatal("no free cell beside the mage")
	}
	if err := w.HeadlessKill(victim.ID); err != nil {
		t.Fatal(err)
	}
	for range 256 {
		f.live.tick()
		if e, ok := liveEntity(f, victim.ID); ok && e.Decay == sim.DecayBones {
			break
		}
	}
	beforePath, _ := writeOrdinarySAV(t, f, "ghost-before-raise.sav")
	beforeCold := loadAreaContinuation(t, beforePath)
	if beforeCold.live.world.Ghost().NativeBasis != w.Ghost().NativeBasis {
		t.Fatal("cold SAV lost the resolved Ghost constructor basis")
	}
	f.live.pending = append(f.live.pending, sim.Cast(mage.ID, victim.ID, 25))
	beforeCold.live.pending = append(beforeCold.live.pending, sim.Cast(mage.ID, victim.ID, 25))
	for range 256 {
		f.live.tick()
		beforeCold.live.tick()
		if _, ok := liveEntity(f, victim.ID); !ok {
			break
		}
	}
	requireLiveRaisedGhost(t, f, typ)
	requireRaisedGhostNativeHistory(t, f, typ, w.Ghost().NativeBasis)
	requireRaisedGhostNativeHistory(t, beforeCold, typ, w.Ghost().NativeBasis)

	row := 0
	for i := 1; i < f.Table.Units.Len(); i++ {
		if f.Table.Units.EntryName(i) == "Ghost" {
			row = i
			break
		}
	}
	def, err := data.NewUnitDef("Ghost", f.Table.Units.EntryParams(row))
	if err != nil {
		t.Fatal(err)
	}
	path, raw := writeOrdinarySAV(t, f, "ghost.sav")
	saved := raisedGhostRecord(t, raw, typ)
	if int(saved.DefRow) != row || int32(saved.Face) != def.Face {
		t.Fatalf("SAV Ghost row %d face %d, want %d and %d", saved.DefRow, saved.Face, row, def.Face)
	}
	cold := loadAreaContinuation(t, path)
	requireLiveRaisedGhost(t, cold, typ)
	requireRaisedGhostNativeHistory(t, cold, typ, w.Ghost().NativeBasis)
	for range 3 {
		cold.live.tick()
	}
	requireRaisedGhostNativeHistory(t, cold, typ, w.Ghost().NativeBasis)
	_, again := writeOrdinarySAV(t, cold, "ghost-again.sav")
	if got := raisedGhostRecord(t, again, typ); got.DefRow != saved.DefRow || got.Face != saved.Face {
		t.Fatalf("second SAV Ghost row %d face %d, want %d and %d", got.DefRow, got.Face, saved.DefRow, saved.Face)
	}

	doc, origins, err := sav.DecodeDocumentDataWithOrigins(raw)
	if err != nil {
		t.Fatal(err)
	}
	object := 0
	for _, o := range origins {
		if o.ArchiveIndex == saved.ArchiveIndex {
			object = int(o.ObjectIndex)
		}
	}
	if object < 1 || object > len(doc.Objects) {
		t.Fatalf("Ghost archive %d has no document object", saved.ArchiveIndex)
	}
	r := &doc.Objects[object-1]
	zeroed := 0
	for i := range r.Values {
		switch r.Values[i].Name {
		case "T0C", "U4B":
			r.Values[i].Value = 0
			zeroed++
		}
	}
	if r.Class != "Unit" || zeroed != 2 {
		t.Fatalf("Ghost archive %d is %s with %d zeroed fields", saved.ArchiveIndex, r.Class, zeroed)
	}
	legacy, err := sav.EncodeDocumentData(doc)
	if err != nil {
		t.Fatal(err)
	}
	if got := raisedGhostRecord(t, legacy, typ); got.DefRow != 0 || got.Face != 0 {
		t.Fatalf("zeroed Ghost row %d face %d", got.DefRow, got.Face)
	}
	legacyPath := filepath.Join(t.TempDir(), "ghost-row0.sav")
	if err := os.WriteFile(legacyPath, legacy, 0o600); err != nil {
		t.Fatal(err)
	}
	requireLiveRaisedGhost(t, loadAreaContinuation(t, legacyPath), typ)
}

func requireRaisedGhostNativeHistory(t *testing.T, f *FrontEnd, typ int32, want sim.NativeActorBasis) {
	t.Helper()
	if !want.BasePresent || want.BaseKnown != 0x003fffff || !want.ModifierPresent || want.ModifierKnown != ^uint64(0) || !want.BodyPresent || !want.BodyKnown {
		t.Fatal("resolved Ghost template lacks its constructor availability", want)
	}
	for _, e := range f.live.world.Entities() {
		if e.TypeID == typ && e.Owner == sim.SelfSlot && e.MapUnitID == 0 && e.Alive() {
			if e.ActorLoad.Source.Class != 0 || e.NativeBasis != want {
				t.Fatal("raised native Ghost lost exact constructor history", e.ID, e.NativeBasis, want)
			}
			before := f.live.world.Hash()
			changed := want.WithBody(want.Body + 1)
			if err := f.live.world.RestoreNativeActorBases([]sim.NativeActorBasisRecord{{ID: e.ID, Basis: changed}}); err != nil {
				t.Fatal(err)
			}
			if f.live.world.Hash() == before {
				t.Fatal("actual raised Ghost history is excluded from World hash")
			}
			if err := f.live.world.RestoreNativeActorBases([]sim.NativeActorBasisRecord{{ID: e.ID, Basis: want}}); err != nil || f.live.world.Hash() != before {
				t.Fatal("restoring exact Ghost history changed unrelated state", err)
			}
			return
		}
	}
	t.Fatal("native history witness has no actual raised Ghost")
}

// liveEntity returns the live entity with the given ID.
func liveEntity(f *FrontEnd, id sim.EntityID) (sim.Entity, bool) {
	for _, e := range f.live.world.Entities() {
		if e.ID == id {
			return e, true
		}
	}
	return sim.Entity{}, false
}

package game

import (
	"reflect"
	"testing"

	"againrom/pkg/formats/sav"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
)

func TestCurrentConsumedCorpseZeroIDSAVAndMissingBinding(t *testing.T) {
	for _, bound := range []bool{false, true} {
		f, snapshot, base := partialCurrentGraph(t)
		spells := make(dbCollection, 26)
		for i := 1; i < len(spells); i++ {
			spells[i] = dbEntry{name: "synthetic spell", params: make([]int32, 19)}
		}
		p := spells[25].params
		p[1], p[4], p[6], p[8] = 2, 1, 4, 1
		f.Table.Spells = spells
		body := sim.Entity{ID: 0, X: 15, Y: 15, HP: -12, MaxHP: 29, Decay: sim.DecayBones, Owner: sim.SelfSlot, TypeID: 1, TokenSize: 1, DyingTime: 200}
		mage := sim.Entity{ID: 41, X: 14, Y: 15, HP: 40, MaxHP: 40, Owner: sim.SelfSlot, TypeID: 1, TokenSize: 1, ScanRange: 10, Mana: 100, MaxMana: 100, Mind: 40, KnownSpells: 1 << 25}
		ghost := sim.GhostTemplate{Class: 1, TypeID: 1, Domain: sim.DomainGhost, Speed: 1, TokenSize: 1}
		w, err := sim.NewSummoningWorld(9, base.Bounds(), sim.ModeCanonical, sim.Terrain{}, []sim.Entity{body, mage}, consumedCorpseScript(t, 0),
			sim.Relations{}, nil, nil, mapload.SpellRules(f.Table), ghost)
		if err != nil {
			t.Fatal(err)
		}
		snapshot.SavedDocument = nil
		snapshot.CurrentPartyIDs, snapshot.CurrentRoster = nil, nil
		snapshot.Residue.VisualIdentities, snapshot.Residue.VisualNext = nil, 0
		snapshot.ghost = &ghost
		if bound {
			snapshot.SavedDocument, err = f.materializeCurrentWorld(snapshot, w)
			if err != nil {
				t.Fatal(err)
			}
		}
		sim.Step(w, []sim.Command{sim.Cast(41, 0, 25)})
		for range 256 {
			if len(w.CurrentTerminalActors()) != 0 {
				break
			}
			sim.Step(w, nil)
		}
		want := sim.CurrentTerminalActor{ID: 0, Cell: 0x0f0f, HP: -10001, Stage: 5}
		if got := w.CurrentTerminalActors(); !reflect.DeepEqual(got, []sim.CurrentTerminalActor{want}) {
			t.Fatal("actual cast did not consume ID0", got)
		}
		snapshot.World, err = w.MarshalBinary()
		if err != nil {
			t.Fatal(err)
		}
		raw, err := f.ExportCurrentSave(snapshot, "consumed ID0")
		if err != nil {
			t.Fatal(err)
		}
		consumedCorpseTuple(t, raw, want, bound)
		cold := coldCurrentScript(t, f, raw)
		if got := cold.live.world.CurrentTerminalActors(); !reflect.DeepEqual(got, []sim.CurrentTerminalActor{want}) {
			t.Fatal("raw SAV lost consumed zero-ID tuple", got)
		}
		for range 256 {
			cold.live.tick()
		}
		if cold.live.world.ScriptRegister(0) != -10001 || cold.live.world.ScriptRegister(1) != 0 || cold.live.world.Script().Checks()[0].Unit != 0 || !cold.live.world.Script().Checks()[0].HasUnit {
			t.Fatal("zero-ID script binding was lost")
		}
		next := bindingOrderSave(t, cold, "zero consumed resave")
		consumedCorpseTuple(t, next, want, bound)
		doc, err := sav.DecodeDocumentData(raw)
		if err != nil {
			t.Fatal(err)
		}
		a, err := readCurrentActions(&doc)
		if err != nil {
			t.Fatal(err)
		}
		if !pureCurrentTerminalActorValue(a.Values[0], 0) {
			t.Fatal("zero terminal acquired live policy")
		}
	}
}

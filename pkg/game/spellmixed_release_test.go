package game

import (
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"testing"

	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
)

type spellTickWitness struct {
	HP          int32
	Decay       uint8
	Credit      bool
	CreditSpell int8
	Roots       []string
}

type mixedSpellWitness struct {
	Target      uint32
	Ticks       []spellTickWitness
	AfterSecond spellTickWitness
}

func spellWorldTick(t *testing.T, f *FrontEnd, target uint32) spellTickWitness {
	t.Helper()
	e := world1170Entity(t, f, target)
	w := f.live.world
	out := spellTickWitness{HP: e.HP, Decay: uint8(e.Decay), Credit: e.HasKillCredit, CreditSpell: e.KillCreditSpell}
	graph := w.SavedSpellGraph()
	areas, err := w.NativeAreaSaveStates()
	if err != nil {
		t.Fatal(err)
	}
	deliveries := w.NativeSpellDeliverySaveStates()
	area := func(spell, remaining uint16, mode, stage byte) string {
		return fmt.Sprintf("Area:%d:%d:%d:%d", spell, remaining, mode, stage)
	}
	var node func(uint32) string
	node = func(id uint32) string {
		n := graph.Nodes[id-1]
		switch n.Value.Class {
		case "SpellTransport":
			child := n.Primary
			if child == 0 {
				child = n.Fallback
			}
			if child == 0 {
				return fmt.Sprintf("Transport:%d:null", n.Value.ST4C)
			}
			return fmt.Sprintf("Transport:%d:%s", n.Value.ST4C, node(child))
		case "AreaEffect":
			for _, d := range w.SavedWorldEffectDrivers().Areas {
				if d.ID == id {
					return area(d.Spell, n.Value.AE4C, d.Mode, n.Value.AE48[3])
				}
			}
		case "PointEffect":
			return fmt.Sprintf("Point:%d", n.Spell)
		}
		return n.Value.Class
	}
	for _, ref := range w.CurrentWorldEffectOrder() {
		switch ref.Kind {
		case sim.EffectSavedGraph:
			out.Roots = append(out.Roots, node(ref.Index))
		case sim.EffectSavedArea:
			for _, d := range w.SavedWorldEffectDrivers().Areas {
				if d.ID == ref.Index {
					v := w.SavedSpellEffects()[d.Root]
					out.Roots = append(out.Roots, area(d.Spell, v.AE4C, d.Mode, v.AE48[3]))
				}
			}
		case sim.EffectNativeArea:
			a := areas[ref.Index]
			out.Roots = append(out.Roots, area(a.Spell, a.Remaining, a.Mode, a.Stage))
		case sim.EffectNativeDelivery:
			d := deliveries[ref.Index]
			child := fmt.Sprintf("Point:%d", d.Area.Spell)
			if d.AtCell {
				child = area(d.Area.Spell, d.Area.Remaining, d.Area.Mode, d.Area.Stage)
			}
			if !d.Released {
				child = fmt.Sprintf("Transport:%d:%s", d.Remaining, child)
			}
			out.Roots = append(out.Roots, child)
		}
	}
	return out
}

func castSpellWitness(t *testing.T, f *FrontEnd, in sim.ScriptInstant, done func() bool) {
	t.Helper()
	script, err := sim.NewScript(nil, []sim.ScriptInstant{in}, []sim.ScriptTrigger{{Instants: [4]int32{0, sim.ScriptNone, sim.ScriptNone, sim.ScriptNone}, Once: true, Latch: 999}})
	if err != nil {
		t.Fatal(err)
	}
	installTestScript(t, f, script)
	for tick := 0; tick < 256; tick++ {
		f.live.tick()
		b, s, c := f.live.world.NativeCastContinuations()
		if f.live.world.ScriptLatched(999) && done() && b+s+c == 0 {
			return
		}
	}
	t.Fatal("witness cast did not finish")
}

func rewriteSpellTransportCounter(t *testing.T, path string, counter uint32) {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	doc, err := sav.DecodeDocumentData(raw)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, id := range doc.World.Effects {
		if id == 0 || int(id) > len(doc.Objects) || doc.Objects[id-1].Class != "SpellTransport" {
			continue
		}
		savedObjectSetValue(&doc.Objects[id-1], "ST4C", counter)
		found = true
		break
	}
	if !found {
		t.Fatal("ordinary SAV has no active SpellTransport")
	}
	updated, err := sav.EncodeDocumentData(doc)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, updated, 0600); err != nil {
		t.Fatal(err)
	}
}

func TestReleaseMixedSpellSAVContinuation(t *testing.T) {
	for _, tail := range []bool{false, true} {
		t.Run(fmt.Sprintf("tail-%v", tail), func(t *testing.T) {
			if path := os.Getenv("AGAINROM_MIXED_SAV_INPUT"); path != "" {
				var want mixedSpellWitness
				raw, err := os.ReadFile(path + ".json")
				if err != nil {
					t.Fatal(err)
				}
				if err := json.Unmarshal(raw, &want); err != nil {
					t.Fatal(err)
				}
				f := loadAreaContinuation(t, path)
				for tick, expected := range want.Ticks {
					if tick > 0 {
						f.live.tick()
					}
					if got := spellWorldTick(t, f, want.Target); !reflect.DeepEqual(got, expected) {
						t.Fatalf("mixed tick%d got %+v want %+v", tick, got, expected)
					}

				}
				for _, e := range f.live.world.SavedSpellEffects() {
					if e.Class == "PointEffect" || e.Class == "SpellTransport" {
						t.Fatal("mixed delivered roots remain")
					}
				}
				second := saveCorpseMission(t, f, t.TempDir())
				f = loadAreaContinuation(t, second)
				for range 3 {
					f.live.tick()
				}
				if got := spellWorldTick(t, f, want.Target); !reflect.DeepEqual(got, want.AfterSecond) {
					t.Fatal("mixed second SAV replay", got)
				}
				t.Logf("mixed ordered HP/counter/credit sequence: %+v", want)
				return
			}
			f := spellWitnessSource(t)
			var target sim.Entity
			for _, e := range f.live.world.Entities() {
				if e.Owner == sim.SelfSlot && e.Alive() && e.HP > 100 && e.SourceBinding.RuntimeID != 0 {
					target = e
					break
				}
			}
			if target.SourceBinding.RuntimeID == 0 {
				t.Fatal("mixed positive target absent")
			}
			x, y := f.live.world.Bounds().Width-8, f.live.world.Bounds().Height-8
			if err := f.live.world.HeadlessPlace(target.ID, x, y); err != nil {
				t.Fatal(err)
			}
			castSpellWitness(t, f, sim.ScriptInstant{Op: sim.ScriptInstantCastAtUnit, Args: [10]int32{x - 6, y, 13, 30}, HasUnit: true, Unit: target.ID}, func() bool { return f.live.world.PendingSpellDeliveries() > 0 })
			first := saveCorpseMission(t, f, t.TempDir())
			rewriteSpellTransportCounter(t, first, 100)
			f = loadAreaContinuation(t, first)
			castSpellWitness(t, f, sim.ScriptInstant{Op: sim.ScriptInstantCastAtCell, Args: [10]int32{x, y + 1, x, y, 7, 30}}, func() bool { return f.live.world.HasNativeAreaEffects() })
			castSpellWitness(t, f, sim.ScriptInstant{Op: sim.ScriptInstantCellEffectAge, Args: [10]int32{x, y, 7, 32}}, func() bool { return true })
			target = world1170Entity(t, f, target.SourceBinding.RuntimeID)
			castSpellWitness(t, f, sim.ScriptInstant{Op: sim.ScriptInstantCastAtUnit, Args: [10]int32{x - 1, y, 1, 30}, HasUnit: true, Unit: target.ID}, func() bool { return f.live.world.PendingSpellDeliveries() > 0 })
			counter := f.live.world.NativeSpellDeliverySaveStates()[0].Remaining
			if tail {
				counter++
			}
			want := mixedSpellWitness{Target: target.SourceBinding.RuntimeID}
			path := saveCorpseMission(t, f, t.TempDir())
			if tail {
				rewriteSpellTransportCounter(t, path, uint32(counter))
			}
			f = loadAreaContinuation(t, path)
			for tick := 0; tick <= 32; tick++ {
				if tick > 0 {
					f.live.tick()
				}
				want.Ticks = append(want.Ticks, spellWorldTick(t, f, want.Target))
			}
			if want.Ticks[3].HP >= want.Ticks[0].HP {
				t.Fatal("mixed positive result absent")
			}
			for range 3 {
				f.live.tick()
			}
			want.AfterSecond = spellWorldTick(t, f, want.Target)
			proof, _ := json.MarshalIndent(want, "", "  ")
			if err := os.WriteFile(path+".json", proof, 0600); err != nil {
				t.Fatal(err)
			}
			emitSpellWitness(t, path, fmt.Sprintf("mixed-tail-%v", tail), proof)
			runSpellWitnessChild(t, path, "AGAINROM_MIXED_SAV_INPUT")
		})
	}
}

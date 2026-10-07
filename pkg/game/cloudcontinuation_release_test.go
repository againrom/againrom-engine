package game

import (
	"encoding/json"
	"os"
	"reflect"
	"slices"
	"strings"
	"testing"

	"againrom/pkg/data"
	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
)

type cloudDeliveryCollection struct{ data.Collection }

func (c cloudDeliveryCollection) EntryParams(i int) []int32 {
	p := slices.Clone(c.Collection.EntryParams(i))
	if i == 7 {
		p[5], p[7] = 2, 256
	}
	return p
}

type cloudTickWitness struct {
	Tick, TransportCounter int
	WorldTick              uint64
	Present                bool
	Cells                  []uint16
	Counter                uint16
	FirstPaint             byte
}

type cloudContinuationWitness struct {
	Spell          uint16
	Layer          byte
	SecondSaveTick int
	Ticks          []cloudTickWitness
}

func cloudWorldTick(t *testing.T, f *FrontEnd, tick int) cloudTickWitness {
	t.Helper()
	w := f.live.world
	out := cloudTickWitness{Tick: tick, WorldTick: w.Tick(), TransportCounter: -1, Cells: []uint16{}}
	set := func(counter uint16, phase byte, cells []uint16) {
		if out.Present {
			t.Fatal("cloud witness has multiple spell7 areas")
		}
		out.Present, out.Counter, out.FirstPaint = true, counter, phase
		out.Cells = append(out.Cells, cells...)
	}
	for _, d := range w.NativeSpellDeliverySaveStates() {
		if d.AtCell && d.Area.Spell == 7 {
			set(d.Area.Remaining, 0, nil)
			if !d.Released {
				out.TransportCounter = int(d.Remaining)
			}
		}
	}
	areas, err := w.NativeAreaSaveStates()
	if err != nil {
		t.Fatal(err)
	}
	for _, a := range areas {
		if a.Spell == 7 {
			set(a.Remaining, 1, a.Cells)
		}
	}
	graph := w.SavedSpellGraph()
	if drivers := w.SavedWorldEffectDrivers(); drivers != nil {
		for _, d := range drivers.Areas {
			if d.Spell != 7 {
				continue
			}
			var v sim.SavedSpellEffect
			if graph != nil {
				n := graph.Nodes[d.ID-1]
				if n.Retired {
					continue
				}
				v = n.Value
			} else if d.Root >= 0 {
				v = w.SavedSpellEffects()[d.Root]
			} else {
				continue
			}
			set(v.AE4C, v.AE48[0], d.Cells)
			var owned []uint16
			for _, c := range w.SavedCellRecords() {
				if c.SpellEffects[1] == d.Identity {
					owned = append(owned, c.Cell)
				}
			}
			if !slices.Equal(owned, d.Cells) {
				t.Fatal("driver cells differ from current layer1 ownership", owned, d.Cells)
			}
		}
	}
	for _, e := range w.SavedSpellEffects() {
		if e.Class == "SpellTransport" {
			out.TransportCounter = int(e.ST4C)
		}
	}
	return out
}

func requireCloudSAV(t *testing.T, path string, want cloudTickWitness) {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	doc, err := sav.DecodeDocumentData(raw)
	if err != nil {
		t.Fatal(err)
	}
	got := cloudTickWitness{Tick: want.Tick, WorldTick: uint64(doc.Head.CounterA), TransportCounter: -1, Cells: []uint16{}}
	want.WorldTick = uint64(uint32(want.WorldTick))
	for i := range doc.Objects {
		r := &doc.Objects[i]
		if r.Class == "SpellTransport" {
			counter, _ := savedStructureValue(r, "ST4C")
			got.TransportCounter = int(counter)
		}
		spell, _ := savedStructureValue(r, "T0C")
		if r.Class != "AreaEffect" || spell != 7 {
			continue
		}
		if got.Present {
			t.Fatal("SAV duplicated cloud")
		}
		counter, _ := savedStructureValue(r, "AE4C")
		phase, err := savedMotionRaw(r, "AE48", 4)
		if err != nil {
			t.Fatal(err)
		}
		got.Present, got.Counter, got.FirstPaint = true, uint16(counter), phase[0]
		identity, _ := savedStructureValue(r, "Identity")
		for _, cell := range doc.World.Cells {
			for layer, pointer := range cell.Layers {
				if pointer == identity {
					if layer != 1 {
						t.Fatal("cloud painted another SAV layer", layer)
					}
					got.Cells = append(got.Cells, cell.Cell)
				}
			}
		}
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("raw ordinary SAV cloud %+v want %+v", got, want)
	}
}

func cloudProofFile(t *testing.T, path string, proof cloudContinuationWitness) []byte {
	t.Helper()
	raw, err := json.MarshalIndent(proof, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path+".json", raw, 0600); err != nil {
		t.Fatal(err)
	}
	return raw
}

func coldCloudContinuation(t *testing.T, path string) {
	t.Helper()
	raw, err := os.ReadFile(path + ".json")
	if err != nil {
		t.Fatal(err)
	}
	var proof cloudContinuationWitness
	if err := json.Unmarshal(raw, &proof); err != nil {
		t.Fatal(err)
	}
	f := loadAreaContinuation(t, path)
	for i, want := range proof.Ticks {
		if i > 0 {
			f.live.tick()
		}
		if got := cloudWorldTick(t, f, want.Tick); !reflect.DeepEqual(got, want) {
			t.Fatalf("cold cloud %+v want %+v", got, want)
		}
		if want.Tick == proof.SecondSaveTick {
			second := saveCorpseMission(t, f, t.TempDir())
			requireCloudSAV(t, second, want)
			proof.SecondSaveTick, proof.Ticks = -1, proof.Ticks[i:]
			cloudProofFile(t, second, proof)
			runSpellWitnessChild(t, second, "AGAINROM_CLOUD_SAV_INPUT")
			return
		}
		if i == len(proof.Ticks)-1 {
			last := saveCorpseMission(t, f, t.TempDir())
			requireCloudSAV(t, last, want)
			cold := loadAreaContinuation(t, last)
			if got := cloudWorldTick(t, cold, want.Tick); !reflect.DeepEqual(got, want) {
				t.Fatal("expired cloud changed after final SAVE", got, want)
			}
		}
	}
	t.Logf("vanilla-table cold LOAD retains cloud cells/phase through pulse and expiry at tick%d", proof.Ticks[len(proof.Ticks)-1].Tick)
}

func TestReleaseCloudDeliverySAVContinuation(t *testing.T) {
	for _, cut := range []string{"before", "handoff", "high-before", "high-handoff"} {
		t.Run(cut, func(t *testing.T) {
			if path := os.Getenv("AGAINROM_CLOUD_SAV_INPUT"); path != "" {
				coldCloudContinuation(t, path)
				return
			}
			_, raw := groundCorpusFile(t, "2027-09-07/game0125.sav", "3a055c8dcef6f053721e1c1478c82552c199b92fb178e9f034fe7500077b4efd")
			f := releaseFront(t)
			f.Table.Spells = cloudDeliveryCollection{f.Table.Spells}
			f.Options = OptionsStore{}
			f.SetDeterministicFrames(true)
			open, _, err := f.RestoreOriginal(raw)
			if err != nil {
				t.Fatal(err)
			}
			if err := f.App("cloud delivery SAV").OpenMission(open); err != nil {
				t.Fatal(err)
			}
			quiet, _ := sim.NewScript(nil, nil, nil)
			installTestScript(t, f, quiet)
			for range 40 {
				f.live.tick()
			}
			if strings.HasPrefix(cut, "high-") {
				p := f.live.world.CurrentPolicy()
				p.TickHigh, p.ClockKnown = 1, false
				if err := f.live.world.RestoreCurrentContinuation(&p, nil, f.live.world.Actions(), nil); err != nil {
					t.Fatal(err)
				}
			}
			x, y := f.live.world.Bounds().Width-8, f.live.world.Bounds().Height-8
			castSpellWitness(t, f, sim.ScriptInstant{Op: sim.ScriptInstantCastAtCell, Args: [10]int32{x - 1, y, x, y, 7, 0}}, func() bool { return f.live.world.PendingSpellDeliveries() > 0 })
			d := f.live.world.NativeSpellDeliverySaveStates()
			if len(d) != 1 || d[0].Remaining != 1 || d[0].Released {
				t.Fatal("expected sole native transport with counter1", d)
			}
			if strings.HasSuffix(cut, "handoff") {
				f.live.tick()
			}
			path := saveCorpseMission(t, f, t.TempDir())
			proof := cloudContinuationWitness{Spell: 7, Layer: 1, SecondSaveTick: 2}
			end := int(d[0].Area.Remaining) + 4
			if end > 1200 {
				t.Fatal("unbounded cloud witness", end)
			}
			for tick := 0; tick <= end; tick++ {
				if tick > 0 {
					f.live.tick()
				}
				proof.Ticks = append(proof.Ticks, cloudWorldTick(t, f, tick))
			}
			if proof.Ticks[end].Present || proof.Ticks[0].FirstPaint != 0 || len(proof.Ticks[0].Cells) != 0 || len(proof.Ticks[2].Cells) != 9 {
				t.Fatal("native cloud phase or retirement witness is incomplete", proof.Ticks[0], proof.Ticks[2], proof.Ticks[end])
			}
			requireCloudSAV(t, path, proof.Ticks[0])
			encoded := cloudProofFile(t, path, proof)
			emitSpellWitness(t, path, "cloud-"+cut, encoded)
			runSpellWitnessChild(t, path, "AGAINROM_CLOUD_SAV_INPUT")
			t.Logf("%s ordinary SAVE: first paint/counter/cells and second SAV match uninterrupted ticks0..%d", cut, end)
		})
	}
}

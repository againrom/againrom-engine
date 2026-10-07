package sim

import (
	"bytes"
	"fmt"
	"testing"
)

func transportGraphWorld(t *testing.T, counter uint16) *World {
	t.Helper()
	w := deliveryTestWorld(t, 1)
	p := SavedEffect{Class: "Effect_DirectDamage", E0C: 1}
	p.DirectDamage[19], p.DirectDamage[21] = 9, 1
	g := &SavedSpellGraph{Nodes: []SavedSpellNode{
		{Value: SavedSpellEffect{Class: "SpellTransport", ST4C: counter}, Primary: 2},
		{Value: SavedSpellEffect{Class: "PointEffect", PE44: 555, PE48: &p}, Spell: 1, Target: 2, HasTarget: true},
	}, Roots: []uint32{1}}
	w.savedWorldEffects = &SavedWorldEffects{}
	if err := w.ImportSavedSpellGraph(g); err != nil {
		t.Fatal(err)
	}
	return w
}

func requireSpellGraphBinary(t *testing.T, w *World) *World {
	t.Helper()
	b, err := w.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	var back World
	if err := back.UnmarshalBinary(b); err != nil {
		t.Fatal(err)
	}
	again, err := back.MarshalBinary()
	if err != nil || !bytes.Equal(b, again) {
		t.Fatal("graph native continuity", err)
	}
	return &back
}

func TestSavedTransportSignedBoundary(t *testing.T) {
	for _, counter := range []uint16{0, 1, 2, 4, 7, 10, 0x7fff, 0x8000, 0x8001, 0xffff} {
		t.Run(fmt.Sprint(counter), func(t *testing.T) {
			w := transportGraphWorld(t, counter)
			back := requireSpellGraphBinary(t, w)
			remaining := counter
			for tick := 0; tick < 32769; tick++ {
				remaining--
				w.stepSavedWorldEffects(nil)
				back.stepSavedWorldEffects(nil)
				if w.entities[1].HP != back.entities[1].HP {
					t.Fatal("LOAD differs", tick)
				}
				if int16(remaining) > 0 {
					if w.entities[1].HP != 1000 {
						t.Fatal("early delivery")
					}
					continue
				}
				if w.entities[1].HP != 1000 || len(w.savedSpellEffects) != 1 || w.savedSpellEffects[0].Class != "PointEffect" {
					t.Fatal("tail handoff applied before the next list pass")
				}
				back = requireSpellGraphBinary(t, w)
				w.stepSavedWorldEffects(nil)
				back.stepSavedWorldEffects(nil)
				if w.entities[1].HP != 991 || w.Hash() != back.Hash() || len(w.savedSpellEffects) != 0 {
					t.Fatal("child next-pass payload")
				}
				back = requireSpellGraphBinary(t, w)
				back.stepSavedWorldEffects(nil)
				if back.entities[1].HP != 991 {
					t.Fatal("payload replay")
				}
				return
			}
			t.Fatal("counter did not finish")
		})
	}
}

func TestSavedTransportAliasesAndConcurrentDelivery(t *testing.T) {
	for _, shared := range []bool{true, false} {
		w := transportGraphWorld(t, 2)
		g := w.SavedSpellGraph()
		g.Nodes = append(g.Nodes, g.Nodes[0])
		if !shared {
			g.Nodes = append(g.Nodes, g.Nodes[1])
			g.Nodes[2].Primary = 4
		}
		g.Roots = []uint32{1, 3, 1}
		if err := w.ImportSavedSpellGraph(g); err != nil {
			t.Fatal(err)
		}
		back := requireSpellGraphBinary(t, w)
		for range 3 {
			Step(w, nil)
			Step(back, nil)
			if w.Hash() != back.Hash() {
				t.Fatal("shared native hash differs")
			}
		}
		want := int32(982)
		if shared {
			want = 991
		}
		if w.entities[1].HP != want || len(w.savedSpellEffects) != 0 {
			t.Fatal("alias application", shared, w.entities[1].HP)
		}
	}
}

func TestSavedTransportTargetLifetimeAndOrdinaryPayload(t *testing.T) {
	for _, removed := range []bool{false, true} {
		w := transportGraphWorld(t, 1)
		if removed {
			w.remove([]EntityID{2})
		} else {
			w.entities[1].HP = -57
			w.entities[1].Decay = DecayBones
		}
		w.stepSavedWorldEffects(nil)
		w.stepSavedWorldEffects(nil)
		if len(w.savedSpellEffects) != 0 {
			t.Fatal("dead target retained delivery")
		}
		if !removed && w.entities[1].HP != -57 {
			t.Fatal("dead target damaged")
		}
		requireSpellGraphBinary(t, w)
	}
	w := transportGraphWorld(t, 1)
	g := w.SavedSpellGraph()
	g.Nodes[1].Value.PE48 = &SavedEffect{Class: "Effect", E0C: 1, E3C: 17, E3D: 1, E40: 3 | uint32(12)<<16}
	if err := w.ImportSavedSpellGraph(g); err != nil {
		t.Fatal(err)
	}
	w.stepSavedWorldEffects(nil)
	w.stepSavedWorldEffects(nil)
	if len(w.ActiveEffects()) != 1 || w.ActiveEffects()[0].Magnitude != 3 || w.ActiveEffects()[0].Remaining != 12 {
		t.Fatal("ordinary frozen payload", w.ActiveEffects())
	}
	back := requireSpellGraphBinary(t, w)
	back.stepSavedWorldEffects(nil)
	if len(back.ActiveEffects()) != 1 {
		t.Fatal("ordinary payload replay")
	}
}

func TestSavedTransportAreaHandoffAndIndexShift(t *testing.T) {
	w := transportGraphWorld(t, 2)
	g := w.SavedSpellGraph()
	g.Nodes[1].Value = SavedSpellEffect{Class: "AreaEffect", AE48: [4]byte{1, 1, 0, 0}, AE44: g.Nodes[1].Value.PE48}
	g.Nodes[1].HasTarget, g.Nodes[1].Target = false, 0
	w.savedWorldEffects.Areas = []SavedAreaDriver{{ID: 2, Root: -1, Identity: 55, Key: cellKey(5, 1), Layer: 255, Mode: AreaModeBlast, Spell: 1}}
	if err := w.ImportSavedSpellGraph(g); err != nil {
		t.Fatal(err)
	}
	back := requireSpellGraphBinary(t, w)
	for range 3 {
		Step(w, nil)
		Step(back, nil)
		if w.Hash() != back.Hash() {
			t.Fatal("area handoff hash")
		}
	}
	if w.entities[1].HP != 991 || len(w.savedSpellEffects) != 0 || w.savedWorldEffects != nil || w.savedSpellGraph != nil {
		t.Fatal("area handoff", w.entities[1].HP, w.savedWorldEffects)
	}
	requireSpellGraphBinary(t, w)
}

func TestSavedPointFatalBoundaryWithoutInventedCaster(t *testing.T) {
	w := transportGraphWorld(t, 1)
	w.entities[1].HP = 4
	back := requireSpellGraphBinary(t, w)
	for range 2 {
		Step(w, nil)
		Step(back, nil)
		if w.Hash() != back.Hash() {
			t.Fatal("fatal delivery native LOAD diverged")
		}
	}
	e := w.entities[1]
	if e.HP != -5 || e.Alive() || e.HasKillCredit || len(w.savedSpellEffects) != 0 {
		t.Fatal("fatal point result or invented credit", e.HP, e.Decay, e.HasKillCredit)
	}
	back = requireSpellGraphBinary(t, w)
	Step(back, nil)
	if back.entities[1].HP != -5 {
		t.Fatal("fatal point replayed")
	}
}

func TestSavedSpellGraphRejectsCorruptionAndKeepsLegacy(t *testing.T) {
	w := transportGraphWorld(t, 1)
	before := w.Hash()
	g := w.SavedSpellGraph()
	g.Nodes[0].Primary = 1
	if err := w.ImportSavedSpellGraph(g); err == nil || w.Hash() != before {
		t.Fatal("cycle changed world")
	}
	w.savedSpellGraph = nil
	legacy, err := w.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	if legacy[0] != formatVersion {
		t.Fatal("legacy version changed")
	}
	var old World
	if err := old.UnmarshalBinary(legacy); err != nil {
		t.Fatal(err)
	}
	old.stepSavedWorldEffects(nil)
	if old.entities[1].HP != 1000 || old.savedSpellGraph != nil {
		t.Fatal("legacy implicitly armed")
	}
}

func TestSpellGraphDecodeKeepsBoundSourceArithmetic(t *testing.T) {
	w := transportGraphWorld(t, 1)
	b, err := w.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	var back World
	back.BindSourceDerive(func(s SourceActor, _ int32, _ Rules) (SourceActor, error) { return s, nil })
	if err := back.UnmarshalBinary(b); err != nil {
		t.Fatal(err)
	}
	if back.sourceDerive == nil {
		t.Fatal("LOAD dropped the loader's source arithmetic binding")
	}
}

func transportCloudWorld(t *testing.T) *World {
	t.Helper()
	w := transportGraphWorld(t, 1)
	w.spells[0].ID, w.spells[0].Area, w.spells[0].Distribution = 7, true, distributionDiamond
	p := &SavedCellPlanes{}
	p.Costs[0], p.Costs[5] = 255, 8
	for y := int32(0); y < w.bounds.Height; y++ {
		for x := int32(0); x < w.bounds.Width; x++ {
			key := cellKey(x, y)
			p.Cost[key], p.CostKnown[key] = 8, 1
		}
	}
	if err := w.ImportOriginalCellPlanes(p); err != nil {
		t.Fatal(err)
	}
	g := w.SavedSpellGraph()
	g.Nodes[1].Spell, g.Nodes[1].Target, g.Nodes[1].HasTarget = 7, 0, false
	g.Nodes[1].Value = SavedSpellEffect{Class: "AreaEffect", AE48: [4]byte{0, 1, 0, 0}, AE4C: 1, AE44: g.Nodes[1].Value.PE48}
	w.savedWorldEffects.Areas = []SavedAreaDriver{{ID: 2, Root: -1, Identity: 55, Key: cellKey(5, 1), Layer: 1, Mode: AreaModeCloud, Spell: 7}}
	if err := w.ImportSavedSpellGraph(g); err != nil {
		t.Fatal(err)
	}
	return w
}

func TestSavedTransportCloudHandoffOwnsCellsAndFinalPulse(t *testing.T) {
	w := transportCloudWorld(t)
	w.stepSavedWorldEffects(nil)
	if len(w.savedWorldEffects.Areas[0].Cells) != 0 || w.entities[1].HP != 1000 || w.savedSpellEffects[0].AE48[0] != 0 {
		t.Fatal("tail handoff executed cloud")
	}
	w = requireSpellGraphBinary(t, w)
	w.stepSavedWorldEffects(nil)
	if len(w.savedWorldEffects.Areas[0].Cells) == 0 || w.entities[1].HP != 1000 || w.savedSpellEffects[0].AE4C != 1 || w.savedSpellEffects[0].AE48[0] != 1 {
		t.Fatal("first paint changed countdown or applied payload")
	}
	w = requireSpellGraphBinary(t, w)
	w.stepSavedWorldEffects(nil)
	if w.entities[1].HP != 991 || w.savedSpellEffects[0].AE4C != 0 {
		t.Fatal("cloud zero pulse", w.entities[1].HP)
	}
	w.stepSavedWorldEffects(nil)
	if len(w.savedSpellEffects) != 0 || w.savedWorldEffects != nil || w.savedSpellGraph != nil {
		t.Fatal("cloud owner not retired")
	}
	for _, c := range w.savedCellRecords {
		if c.SpellEffects[1] != 0 {
			t.Fatal("stale cloud layer")
		}
	}
	requireSpellGraphBinary(t, w)
}

func TestSavedTransportPrepaintedChildAndDiscardedFallback(t *testing.T) {
	for _, discarded := range []bool{false, true} {
		w := transportCloudWorld(t)
		w.paintSavedArea(0)
		if discarded {
			g := w.SavedSpellGraph()
			g.Nodes = append(g.Nodes, SavedSpellNode{Value: SavedSpellEffect{Class: "PointEffect"}})
			g.Nodes[0].Primary, g.Nodes[0].Fallback = 3, 2
			if err := w.ImportSavedSpellGraph(g); err != nil {
				t.Fatal(err)
			}
		}
		w = requireSpellGraphBinary(t, w)
		w.stepSavedWorldEffects(nil)
		w = requireSpellGraphBinary(t, w)
		w.stepSavedWorldEffects(nil)
		w.stepSavedWorldEffects(nil)
		want := int32(991)
		if discarded {
			want = 1000
		}
		if w.entities[1].HP != want || len(w.savedSpellEffects) != 0 || w.savedWorldEffects != nil || w.savedSpellGraph != nil {
			t.Fatal("prepainted child continuation", discarded, w.entities[1].HP)
		}
		for _, c := range w.savedCellRecords {
			if c.SpellEffects[1] != 0 {
				t.Fatal("discarded layer survived")
			}
		}
		requireSpellGraphBinary(t, w)
	}
}

func TestSavedCloudFirstPaintAndEmptyInitializedPhase(t *testing.T) {
	for _, tail := range []bool{false, true} {
		for _, counter := range []uint16{0, 1, 17} {
			w := transportCloudWorld(t)
			g := w.SavedSpellGraph()
			g.Nodes[1].Value.AE4C = counter
			if !tail {
				g.Nodes = append(g.Nodes, SavedSpellNode{Value: SavedSpellEffect{Class: "SpellEffect"}})
				g.Roots = append(g.Roots, 3)
			}
			if err := w.ImportSavedSpellGraph(g); err != nil {
				t.Fatal(err)
			}
			w.stepSavedWorldEffects(nil)
			if tail {
				w = requireSpellGraphBinary(t, w)
				w.stepSavedWorldEffects(nil)
			}
			if w.savedSpellGraph.Nodes[1].Value.AE4C != counter || w.entities[1].HP != 1000 || len(w.savedWorldEffects.Areas[0].Cells) == 0 {
				t.Fatal("first paint must return without pulse/decrement", tail, counter)
			}
			w = requireSpellGraphBinary(t, w)
			pulses := int32(0)
			for remaining := int(counter) - 1; remaining >= 0; remaining-- {
				w.stepSavedWorldEffects(nil)
				if remaining%16 == 0 {
					pulses++
				}
				if w.entities[1].HP != 1000-9*pulses || w.savedSpellGraph.Nodes[1].Value.AE4C != uint16(remaining) {
					t.Fatal("cloud pulse boundary", tail, counter, remaining, w.entities[1].HP)
				}
			}
			w.stepSavedWorldEffects(nil)
			if tail {
				if w.savedSpellGraph != nil || w.savedWorldEffects != nil {
					t.Fatal("terminal cloud carriers remain", counter)
				}
			} else if w.savedSpellGraph == nil || !w.savedSpellGraph.Nodes[1].Retired || len(w.savedWorldEffects.Areas[0].Cells) != 0 {
				t.Fatal("cloud cleanup boundary", tail, counter)
			}
			requireSpellGraphBinary(t, w)
		}
	}
	w := transportCloudWorld(t)
	g := w.SavedSpellGraph()
	g.Nodes[1].Value.AE48[0] = 199
	if err := w.ImportSavedSpellGraph(g); err != nil {
		t.Fatal(err)
	}
	w.stepSavedWorldEffects(nil)
	w = requireSpellGraphBinary(t, w)
	w.stepSavedWorldEffects(nil)
	if len(w.savedWorldEffects.Areas[0].Cells) != 0 || w.savedSpellEffects[0].AE48[0] != 199 || w.savedSpellEffects[0].AE4C != 0 || w.entities[1].HP != 1000 {
		t.Fatal("initialized empty cloud repainted or changed its phase byte")
	}
}

func TestSavedTransportNonTailAppendAndOrphanFallback(t *testing.T) {
	w := transportGraphWorld(t, 1)
	g := w.SavedSpellGraph()
	g.Nodes = append(g.Nodes, SavedSpellNode{Value: SavedSpellEffect{Class: "SpellEffect"}}, SavedSpellNode{Value: SavedSpellEffect{Class: "AreaEffect"}})
	g.Nodes[0].Fallback = 4
	g.Roots = append(g.Roots, 3)
	if err := w.ImportSavedSpellGraph(g); err != nil {
		t.Fatal(err)
	}
	w.stepSavedWorldEffects(nil)
	if w.entities[1].HP != 991 || !w.savedSpellGraph.Nodes[3].Retired || len(w.savedSpellGraph.Roots) != 1 || w.savedSpellGraph.Roots[0] != 3 {
		t.Fatal("non-tail append or fallback lifetime")
	}
	requireSpellGraphBinary(t, w)
}

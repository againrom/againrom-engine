package game

import (
	"reflect"
	"slices"
	"testing"

	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
)

type terminalContentRefs struct {
	Name string
	Keys []uint32
}

type terminalContentNode struct {
	Record sav.DocumentRecordData
	Refs   []terminalContentRefs
}

type terminalContentGraph struct {
	Roots []uint32
	Refs  []terminalContentRefs
	Nodes []terminalContentNode
}

func terminalHumanContentsGraph(t *testing.T, doc sav.DocumentData, identity uint32) terminalContentGraph {
	t.Helper()
	var graph terminalContentGraph
	key := func(object uint16) uint32 {
		if object == 0 {
			return 0
		}
		if int(object) > len(doc.Objects) {
			t.Fatal("contents reference outside document", object)
		}
		value, err := savedStructureValue(&doc.Objects[object-1], "Identity")
		if err != nil || value == 0 {
			t.Fatal("contents reference lacks identity", object, err)
		}
		return value
	}
	var root uint16
	for _, object := range doc.DeadActors {
		if key(object) == identity {
			graph.Roots = append(graph.Roots, identity)
			if root != 0 && root != object {
				t.Fatal("multiple terminal records share identity")
			}
			root = object
		}
	}
	if root == 0 || doc.Objects[root-1].Class != "Human" {
		t.Fatal("terminal Human root missing", identity)
	}
	seen := map[uint16]bool{}
	var visit func(uint16)
	refs := func(slot sav.DocumentRefsData) terminalContentRefs {
		out := terminalContentRefs{Name: slot.Name}
		for _, child := range slot.Objects {
			out.Keys = append(out.Keys, key(child))
		}
		return out
	}
	visit = func(object uint16) {
		if object == 0 || seen[object] {
			return
		}
		seen[object] = true
		record := doc.Objects[object-1]
		node := terminalContentNode{Record: record}
		node.Record.RefSlots = nil
		for _, slot := range record.RefSlots {
			node.Refs = append(node.Refs, refs(slot))
			for _, child := range slot.Objects {
				visit(child)
			}
		}
		graph.Nodes = append(graph.Nodes, node)
	}
	for _, slot := range doc.Objects[root-1].RefSlots {
		if !slices.Contains([]string{"Effects", "Inventory", "HeldWeapon", "HeldShield", "Worn"}, slot.Name) {
			continue
		}
		graph.Refs = append(graph.Refs, refs(slot))
		for _, child := range slot.Objects {
			visit(child)
		}
	}
	return graph
}

func terminalHumanContentsFixture(t *testing.T) ([]byte, uint32) {
	t.Helper()
	zero := uint32(0)
	weapon := &holdingFixtureItem{class: "Weapon", code: 0x0101, row: 1, count: 1, kind: 2, price: 83}
	armor := &holdingFixtureItem{class: "Armor", code: 0x0c01, row: 1, count: 1, kind: 1, price: 71}
	item := &holdingFixtureItem{class: "Item", code: 0x0e06, count: 3, kind: 3, price: 29}
	body := &poolFixtureActor{mapID: 92, cell: 0x0807, hp: uint16(65536 - 10001), maxHP: 30, stage: 5, human: true, runtime: &zero,
		holdings: &holdingFixture{weapon: weapon, items: []*holdingFixtureItem{item, weapon, item}}}
	body.holdings.worn[0], body.holdings.worn[7], body.holdings.worn[11] = armor, armor, armor
	living := &poolFixtureActor{mapID: 91, cell: 0x1211, hp: 23, maxHP: 30, name: "Living binding"}
	f := currentRetainedRuntimeFront(t)
	raw := completeCurrentDeadDocument(t, f, savedContainer(poolFixtureBody([]*poolFixturePlayer{{groups: [][]*poolFixtureActor{{living}}}, {}}, []*poolFixtureActor{body, body})))
	doc, err := sav.DecodeDocumentData(raw)
	if err != nil {
		t.Fatal(err)
	}
	root := doc.DeadActors[0]
	identity, _ := savedStructureValue(&doc.Objects[root-1], "Identity")
	position, err := savedMotionRaw(&doc.Objects[root-1], "Block12", 12)
	if err != nil {
		t.Fatal(err)
	}
	position[4], position[5] = 72, 184
	doc.Objects = append(doc.Objects, literalSavedEffectRecord(0x610001), literalSavedEffectRecord(0x610002))
	effect, second := uint16(len(doc.Objects)-1), uint16(len(doc.Objects))
	literalSavedObjectRefs(t, &doc.Objects[root-1], "Effects", []uint16{effect, second, effect}, true)
	held, _ := savedObjectRefs(&doc.Objects[root-1], "HeldWeapon")
	literalSavedObjectRefs(t, &doc.Objects[held[0]-1], "Effects", []uint16{second, effect, second}, true)
	doc, _, err = sav.ReindexDocumentData(doc)
	if err != nil {
		t.Fatal(err)
	}
	raw, err = sav.EncodeDocumentData(doc)
	if err != nil {
		t.Fatal(err)
	}
	return raw, identity
}

func requireTerminalHumanInactive(t *testing.T, world *sim.World, identity uint32) sim.OriginalDeadRecord {
	t.Helper()
	for _, dead := range world.OriginalDeadActors() {
		if dead.Source.Identity != identity {
			continue
		}
		if dead.Source.Class != 2 || dead.Current.Stage != 5 || dead.Current.HP >= -10000 || dead.Current.RuntimeID != 0 || dead.Source.HeldWeapon != (sim.OriginalDeadWeapon{}) {
			t.Fatal("terminal Human changed its passive provenance", dead)
		}
		for _, e := range world.Entities() {
			if e.ID == dead.ID || e.SourceBinding.Identity == identity || dead.Source.MapUnitID != 0 && e.MapUnitID == dead.Source.MapUnitID {
				t.Fatal("terminal Human became an Entity", e)
			}
		}
		for _, e := range world.ActiveEffects() {
			if e.Target == dead.ID {
				t.Fatal("terminal Human acquired an active effect")
			}
		}
		for _, cell := range world.SavedCellRecords() {
			if cell.Ground.Key == identity || cell.Air.Key == identity {
				t.Fatal("terminal Human retains occupancy", cell)
			}
		}
		return dead
	}
	t.Fatal("terminal Human provenance missing", identity)
	return sim.OriginalDeadRecord{}
}

func TestTerminalHumanContentsSurviveSAVColdLoadAndTicks(t *testing.T) {
	raw, identity := terminalHumanContentsFixture(t)
	doc, err := sav.DecodeDocumentData(raw)
	if err != nil {
		t.Fatal(err)
	}
	want := terminalHumanContentsGraph(t, doc, identity)
	if len(want.Roots) != 2 || len(want.Nodes) != 5 {
		t.Fatal("fixture lost shared roots/items/effects", want)
	}
	f := openCurrentRetainedRuntime(t, currentRetainedRuntimeFront(t), raw)
	body := requireTerminalHumanInactive(t, f.live.world, identity)
	if body.Current.FineX != 72 || body.Current.FineY != 184 {
		t.Fatal("terminal source fine position changed", body)
	}
	purse, sacks := f.live.world.Purse(1), f.live.world.Sacks()
	for cycle := range 2 {
		raw = currentRuntimeSave(t, f)
		written, err := sav.DecodeDocumentData(raw)
		if err != nil {
			t.Fatal(err)
		}
		if got := terminalHumanContentsGraph(t, written, identity); !reflect.DeepEqual(got, want) {
			t.Fatalf("cycle %d changed complete contents, identity, order, nulls or aliases:\nwant %+v\ngot %+v", cycle, want, got)
		}
		cold := openCurrentRetainedRuntime(t, currentRetainedRuntimeFront(t), raw)
		for tick := range 65 {
			assertCurrentWorldEqual(t, f.live.world, cold.live.world, "terminal Human cold continuation")
			if got := requireTerminalHumanInactive(t, cold.live.world, identity); got != body {
				t.Fatal("terminal Human tuple changed", cycle, tick, got)
			}
			if cold.live.world.Purse(1) != purse || !reflect.DeepEqual(cold.live.world.Sacks(), sacks) || len(cold.live.world.ActiveEffects()) != 0 {
				t.Fatal("terminal contents replayed loot, rewards or effects", cycle, tick)
			}
			sim.Step(f.live.world, nil)
			sim.Step(cold.live.world, nil)
		}
		f = cold
	}
}

func TestTerminalHumanContentsRequireExactRetainedSource(t *testing.T) {
	raw, _ := terminalHumanContentsFixture(t)
	file, err := sav.Open(raw)
	if err != nil {
		t.Fatal(err)
	}
	dead, err := file.DeadActors()
	if err != nil || len(dead) != 1 {
		t.Fatal(dead, err)
	}
	for name, alter := range map[string]func(*SnapshotSAVDocument, *[]sav.DocumentObjectOrigin, *sav.DeadActor){
		"missing document": func(s *SnapshotSAVDocument, _ *[]sav.DocumentObjectOrigin, _ *sav.DeadActor) { s.Document = nil },
		"missing root": func(s *SnapshotSAVDocument, _ *[]sav.DocumentObjectOrigin, _ *sav.DeadActor) {
			s.Document.DeadActors = nil
		},
		"wrong identity":  func(_ *SnapshotSAVDocument, _ *[]sav.DocumentObjectOrigin, d *sav.DeadActor) { d.Identity++ },
		"wrong archive":   func(_ *SnapshotSAVDocument, _ *[]sav.DocumentObjectOrigin, d *sav.DeadActor) { d.ArchiveIndex++ },
		"wrong class":     func(_ *SnapshotSAVDocument, _ *[]sav.DocumentObjectOrigin, d *sav.DeadActor) { d.Class = "Unit" },
		"nonterminal":     func(_ *SnapshotSAVDocument, _ *[]sav.DocumentObjectOrigin, d *sav.DeadActor) { d.Stage = 4 },
		"missing origins": func(_ *SnapshotSAVDocument, o *[]sav.DocumentObjectOrigin, _ *sav.DeadActor) { *o = nil },
		"wrong object": func(s *SnapshotSAVDocument, o *[]sav.DocumentObjectOrigin, d *sav.DeadActor) {
			for i := range *o {
				if (*o)[i].ArchiveIndex == d.ArchiveIndex {
					(*o)[i].ObjectIndex = s.Document.Players[0]
				}
			}
		},
		"duplicate archive": func(_ *SnapshotSAVDocument, o *[]sav.DocumentObjectOrigin, d *sav.DeadActor) {
			for _, origin := range *o {
				if origin.ArchiveIndex == d.ArchiveIndex {
					*o = append(*o, origin)
					return
				}
			}
		},
		"duplicate object": func(s *SnapshotSAVDocument, o *[]sav.DocumentObjectOrigin, d *sav.DeadActor) {
			for i := range *o {
				if (*o)[i].ArchiveIndex != d.ArchiveIndex {
					(*o)[i].ObjectIndex = s.Document.DeadActors[0]
					return
				}
			}
		},
		"nonterminal root": func(s *SnapshotSAVDocument, _ *[]sav.DocumentObjectOrigin, _ *sav.DeadActor) {
			mustSetValue(&s.Document.Objects[s.Document.DeadActors[0]-1], "Stage", 4)
		},
		"duplicate identity": func(s *SnapshotSAVDocument, _ *[]sav.DocumentObjectOrigin, _ *sav.DeadActor) {
			s.Document.Objects = append(s.Document.Objects, s.Document.Objects[s.Document.DeadActors[0]-1])
			s.Document.DeadActors = append(s.Document.DeadActors, uint16(len(s.Document.Objects)))
		},
		"null root": func(s *SnapshotSAVDocument, _ *[]sav.DocumentObjectOrigin, _ *sav.DeadActor) {
			s.Document.DeadActors = append(s.Document.DeadActors, 0)
		},
		"out of bounds root": func(s *SnapshotSAVDocument, _ *[]sav.DocumentObjectOrigin, _ *sav.DeadActor) {
			s.Document.DeadActors = append(s.Document.DeadActors, uint16(len(s.Document.Objects)+1))
		},
	} {
		t.Run(name, func(t *testing.T) {
			state, origins := decodeSavedDocument(raw)
			d := dead[0]
			if !retainedTerminalDeadContents(state, origins, d) {
				t.Fatal("exact terminal source refused")
			}
			alter(state, &origins, &d)
			if retainedTerminalDeadContents(state, origins, d) {
				t.Fatal("inexact retained source accepted")
			}
		})
	}
}

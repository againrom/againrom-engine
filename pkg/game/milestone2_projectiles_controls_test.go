package game

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"slices"
	"testing"

	"againrom/internal/synth"
	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
)

func projectile1157Front(t *testing.T) *FrontEnd {
	f := poolFixtureFront(t, 91)
	f.Campaign = resolved(saveCampaign(), nil)
	return f
}

// The fixture writer specifies its own names/values and uses the independent
// synthetic YA1 writer. No production Projectile encoder supplies this tree.
func projectile1157Fixture(t *testing.T, mode string) []byte {
	t.Helper()
	f := projectile1157Front(t)
	source, err := sav.Open(completeDocumentFixture1115(t, f))
	if err != nil {
		t.Fatal(err)
	}
	dir := func(name string, children ...synth.RegNode) synth.RegNode {
		return synth.RegNode{Name: name, Kind: 17, Children: children}
	}
	i32 := func(name string, value uint32) synth.RegNode {
		return synth.RegNode{Name: name, Kind: 2, Int: int32(value)}
	}
	nodes := []synth.RegNode{
		dir("Character", synth.RegNode{Name: "Name", Kind: 0, Str: ""}),
		dir("CurrentState", i32("InBattle", 1)),
		dir("GameOptions", i32("FlyingHP", 0), i32("Formation", 0), i32("ShowHP", 0), i32("ShowTimeFlow", 0), i32("Speed", 0), i32("Wimpy", 0)),
		dir("Inventory", i32("IsOpen", 0)),
		dir("Objects", synth.RegNode{Name: "Selection", Kind: 6}),
		dir("SpellBook", i32("IsOpen", 0), i32("Pressed", 0), synth.RegNode{Name: "Shortcuts", Kind: 6, Ints: []int32{-1, -1, -1, -1}}),
		dir("View", i32("X", 0), i32("Y", 0)),
		// The independent source map is40x40. Keep a complete application
		// plane while varying only the Projectile store below.
		dir("Fog", i32("FirstState", 0), synth.RegNode{Name: "Data", Kind: 6, Ints: []int32{1600}}),
	}
	ids := []int32{0x1234010a, 7, 0x4321010a}
	if mode == "empty" || mode == "missing-IDs" || mode == "missing-section" {
		ids = nil
	}
	if mode == "singleton" {
		ids = ids[:1]
	}
	if mode != "missing-section" {
		var leaves []synth.RegNode
		if mode != "missing-FreeIndex" {
			leaves = append(leaves, i32("FreeIndex", 0x7654fedc))
		}
		if mode != "missing-IDs" {
			if mode == "singleton" {
				leaves = append(leaves, i32("IDs", uint32(ids[0])))
			} else {
				leaves = append(leaves, synth.RegNode{Name: "IDs", Kind: 6, Ints: ids})
			}
		}
		nodes = append(nodes, dir("Projectiles", leaves...))
	}
	seen := map[uint16]bool{}
	for _, value := range ids {
		id := uint16(value)
		if seen[id] {
			continue
		}
		seen[id] = true
		var leaves []synth.RegNode
		for i, name := range []string{"x", "y", "z", "picture", "dir", "phase", "lastaction", "action", "actiondir", "actiontarget", "actionx", "actiony", "actionz", "actionphase", "actionsegments", "actionspell"} {
			if mode == "partial" && name == "actionspell" {
				continue
			}
			leaves = append(leaves, i32(name, 0x81234500+uint32(id)*13+uint32(i)*0x10203))
		}
		nodes = append(nodes, dir(fmt.Sprintf("Prj%d", id), leaves...))
	}
	source.Store = synth.Reg(17, nodes)
	return source.Marshal()
}

func projectile1157Resume(t *testing.T, raw []byte) (projectile1157Source, *Mission) {
	t.Helper()
	source, err := sav.Open(raw)
	if err != nil {
		t.Fatal(err)
	}
	want, err := projectile1157Read(source.Store)
	if err != nil {
		t.Fatal(err)
	}
	f := projectile1157Front(t)
	ms, _, err := loadOriginalMission(f, raw)
	if err != nil {
		t.Fatal(err)
	}
	return want, ms
}

func TestProjectiles1157IndependentStateAndNativeContinuation(t *testing.T) {
	for _, mode := range []string{"array", "singleton", "empty", "missing-FreeIndex", "missing-IDs", "missing-section"} {
		t.Run(mode, func(t *testing.T) {
			raw := projectile1157Fixture(t, mode)
			want, _ := projectile1157Resume(t, raw)
			if mode == "array" && (want.free != 0xfedc || !slices.Equal(want.ids, []uint16{266, 7, 266}) || len(want.items) != 2 || uint32(want.items[0].fields[0]) != 0x81235282 || len(want.values) != 34) {
				t.Fatalf("literal allocator/IDs/fields differ before App import: %+v", want)
			}
			projectile1157App(t, raw, func() *FrontEnd { return projectile1157Front(t) })
		})
	}
}

func projectile1157FieldPointers(p *sim.SavedProjectile) []*int32 {
	return []*int32{&p.X, &p.Y, &p.Z, &p.Picture, &p.Dir, &p.Phase, &p.LastAction, &p.Action,
		&p.ActionDir, &p.ActionTarget, &p.ActionX, &p.ActionY, &p.ActionZ, &p.ActionPhase, &p.ActionSegments, &p.ActionSpell}
}

func TestProjectiles1157LossControls(t *testing.T) {
	want, ms := projectile1157Resume(t, projectile1157Fixture(t, "array"))
	if diff := want.worldDifferences(ms.World.SavedProjectiles()); len(diff) != 0 {
		t.Fatal("positive World", diff)
	}
	if diff := want.documentDifferences(ms.savedDocument); len(diff) != 0 {
		t.Fatal("positive Document", diff)
	}
	for i, name := range projectile1157Names {
		t.Run("World/"+name, func(t *testing.T) {
			got := ms.World.SavedProjectiles()
			*projectile1157FieldPointers(&got.Items[0])[i] = 0
			if len(want.worldDifferences(got)) == 0 {
				t.Fatal("lost field escaped raw comparison")
			}
		})
	}
	for _, name := range []string{"allocator", "ID-high-level-order", "ID-multiplicity", "distinct-population", "item-order", "item-identity"} {
		t.Run("World/"+name, func(t *testing.T) {
			got := ms.World.SavedProjectiles()
			switch name {
			case "allocator":
				got.FreeIndex = 0
			case "ID-high-level-order":
				got.IDs[0], got.IDs[1] = got.IDs[1], got.IDs[0]
			case "ID-multiplicity":
				got.IDs = got.IDs[:2]
			case "distinct-population":
				got.Items = got.Items[:1]
			case "item-order":
				got.Items[0], got.Items[1] = got.Items[1], got.Items[0]
			case "item-identity":
				got.Items[1].ID = got.Items[0].ID
			}
			if len(want.worldDifferences(got)) == 0 {
				t.Fatal("lost population/identity escaped raw comparison")
			}
		})
	}
	for _, record := range ms.savedDocument.Document.State.ValueRecords {
		if !projectile1157Path(record.Path) {
			continue
		}
		t.Run("Document/"+record.Path, func(t *testing.T) {
			for _, remove := range []bool{false, true} {
				got, err := cloneSavedDocument(ms.savedDocument)
				if err != nil {
					t.Fatal(err)
				}
				for i := range got.Document.State.ValueRecords {
					r := &got.Document.State.ValueRecords[i]
					if r.Path != record.Path {
						continue
					}
					if remove {
						got.Document.State.ValueRecords = slices.Delete(got.Document.State.ValueRecords, i, i+1)
					} else if len(r.Value.Bytes) != 0 {
						r.Value.Bytes[3] ^= 1 // Changes only high bits of the first ID.
					} else {
						r.Value.Int32 ^= 0x10000
					}
					break
				}
				if len(want.documentDifferences(got)) == 0 {
					t.Fatal("lost/changed record escaped raw Document comparison")
				}
			}
		})
	}
	for _, dir := range []string{"/Projectiles", "/Prj266", "/Prj7"} {
		t.Run("Document directory/"+dir, func(t *testing.T) {
			got, err := cloneSavedDocument(ms.savedDocument)
			if err != nil {
				t.Fatal(err)
			}
			got.Document.State.DirectoryRecords = slices.DeleteFunc(got.Document.State.DirectoryRecords, func(r sav.CityStateDirectoryData) bool { return r.Path == dir })
			if len(want.documentDifferences(got)) == 0 {
				t.Fatal("missing directory escaped comparison")
			}
		})
	}
	if len(want.documentDifferences(nil)) == 0 || len(want.documentDifferences(&SnapshotSAVDocument{Unavailable: "control"})) == 0 {
		t.Fatal("missing Document escaped comparison")
	}
}

func TestProjectiles1157PartialLeafBoundary(t *testing.T) {
	want, ms := projectile1157Resume(t, projectile1157Fixture(t, "partial"))
	if diff := want.worldDifferences(ms.World.SavedProjectiles()); len(diff) != 0 {
		t.Fatal("engine's declared zero defaults", diff)
	}
	if ms.savedDocument == nil || ms.savedDocument.Unavailable == "" || len(want.documentDifferences(ms.savedDocument)) == 0 {
		t.Fatal("partial Prj input must disclose the existing complete-Document boundary")
	}
	t.Log("partial Prj fields: World uses engine zero defaults; complete retained Document remains unavailable, outside full-retention acceptance: " + ms.savedDocument.Unavailable)
}

func TestProjectiles1157RawBounds(t *testing.T) {
	file, err := sav.Open(projectile1157Fixture(t, "array"))
	if err != nil {
		t.Fatal(err)
	}
	for end := 0; end < len(file.Store); end++ {
		if _, err := projectile1157Read(file.Store[:end]); err == nil {
			t.Fatalf("truncated prefix %d accepted", end)
		}
	}
	for _, change := range []string{"magic", "record-count", "root-range", "cycle", "pool-span", "IDs-type", "IDs-pool"} {
		t.Run(change, func(t *testing.T) {
			b := bytes.Clone(file.Store)
			set := func(off int, value uint32) { binary.LittleEndian.PutUint32(b[off:], value) }
			idsName := bytes.Index(b, append([]byte("IDs"), make([]byte, 13)...))
			if idsName < 16 {
				t.Fatal("literal IDs record absent")
			}
			switch change {
			case "magic":
				b[0] ^= 1
			case "record-count":
				set(16, 0xffffffff)
			case "root-range":
				set(4, 0xffffffff)
			case "cycle":
				set(24+4, 0)
				set(24+8, 1)
			case "pool-span":
				b = append(b, 0)
			case "IDs-type":
				set(idsName-4, 4)
			case "IDs-pool":
				set(idsName-12, 0xffffffff)
			}
			if _, err := projectile1157Read(b); err == nil {
				t.Fatal("malformed raw structure escaped reader")
			}
		})
	}
}

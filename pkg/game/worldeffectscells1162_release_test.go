package game

import (
	"bytes"
	"encoding/binary"
	"slices"
	"testing"

	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
)

// Called by the registered release entry, on both installed roots. The first
// case never edits the source or relocates an actor. The second changes only a
// private source to a spell19/layer3 wall with an explicit open baseline cell.
func worldEffects1162ExpiryCuts(t *testing.T, raw []byte, source spell1152Graph) {
	t.Helper()
	for _, wall := range []bool{false, true} {
		name := "untouched-light-expiry"
		if wall {
			name = "controlled-earthwall-expiry"
		}
		t.Run(name, func(t *testing.T) {
			f := releaseFront(t)
			app, store, path := openWorldEffectsTestSave(t, f, raw, "unchanged-light.sav")
			file, err := sav.Open(raw)
			if err != nil {
				t.Fatal(err)
			}
			// Read this mission30 save's own raw table. The older1115 ALM
			// constructor oracle deliberately selects mission10, so it cannot
			// supply mission30 terrain costs. This checks saved plane overlays.
			cells := map[uint16][52]byte{}
			for i := 0; i < file.World.CellRecCount; i++ {
				off := file.World.CellRecDataOff + 54*i
				var payload [52]byte
				copy(payload[:], file.Body[off+2:off+54])
				cells[binary.LittleEndian.Uint16(file.Body[off:])] = payload
			}
			originalCellNodesCheck1115(t, "untouched Light LOAD", f.live.world, cells)
			planes, present := f.live.world.SavedCellPlanes()
			if !present {
				t.Fatal("untouched Light lost plane authority")
			}
			blocks := map[uint16][2]byte{}
			for i := range file.World.Blocks {
				b := file.Body[file.World.BlocksDataOff+4*i:]
				blocks[binary.LittleEndian.Uint16(b[2:])] = [2]byte{b[0], b[1]}
			}
			for key, want := range blocks {
				if planes.Static[key] != want[0] || planes.Dynamic[key] != want[1] {
					t.Fatalf("untouched Light raw Block%04x differs", key)
				}
			}
			if d := light1162Differences(f.live.world, f.live.mission.state.savedDocument, source, 0); len(d) != 0 {
				t.Fatal("untouched source graph", d)
			}
			// Original Document preserves archive order; current SAVE sorts
			// the nodes. Compare initial raw payloads by their actual cell key.
			for _, c := range f.live.mission.state.savedDocument.Document.World.Cells {
				if want, ok := cells[c.Cell]; !ok || c.Payload() != want {
					t.Fatalf("untouched Document Cell%04x differs from raw source", c.Cell)
				}
			}
			ticks, layer := 19, 4
			var subject uint16
			if wall {
				controlled, key := worldEffects1162WallSource(t, raw)
				subject, ticks, layer = key, 2, 3
				f = releaseFront(t)
				app, store, path = openWorldEffectsTestSave(t, f, controlled, "controlled-wall.sav")
				p, _ := f.live.world.SavedCellPlanes()
				if p.Static[subject] != 0x25 || p.Dynamic[subject] != 0x25 {
					t.Fatal("controlled wall must block the explicit empty baseline", p.Static[subject], p.Dynamic[subject])
				}
			}
			drivers := f.live.world.SavedWorldEffectDrivers()
			if drivers == nil || len(drivers.Areas) != 1 || len(drivers.Areas[0].Cells) != 57 || drivers.Areas[0].Layer != uint8(layer) {
				t.Fatal("current cloud binding", drivers)
			}
			keys := slices.Clone(drivers.Areas[0].Cells)
			removed := map[uint16]bool{}
			for tick := 0; tick < ticks; tick++ {
				if tick == ticks-1 {
					_, current, _, _ := f.live.world.SavedActorMotions()
					for _, c := range current {
						if !slices.Contains(keys, c.Cell) {
							continue
						}
						payload := c.Payload
						clear(payload[20+4*layer : 24+4*layer])
						payload[2]--
						removed[c.Cell] = payload[2] == 0 && payload[44] == 0 && bytes.Equal(payload[4:20], make([]byte, 16))
					}
				}
				f.live.tick()
			}
			check := func(front *FrontEnd, state *SnapshotSAVDocument) {
				t.Helper()
				if len(front.live.world.SavedSpellEffects()) != 0 || len(state.Document.World.Effects) != 0 || front.live.world.SavedWorldEffectDrivers() != nil || len(state.WorldEffects.Areas) != 0 {
					t.Fatal("root/child retirement did not finish")
				}
				if d := cellProducer1162Differences(front.live.world, state); len(d) != 0 {
					t.Fatal("retired cut current World/Document", d)
				}
				_, current, _, _ := front.live.world.SavedActorMotions()
				residual := front.live.world.SavedCellRecords()
				for _, key := range keys {
					found := false
					for _, c := range current {
						if c.Cell != key {
							continue
						}
						found = true
						if binary.LittleEndian.Uint32(c.Payload[20+4*layer:]) != 0 || c.Payload[2] != 0 {
							t.Fatalf("retired current payload retained layer/count at%04x", key)
						}
					}
					if found == removed[key] {
						t.Fatalf("empty-node cleanup/current owner%04x: found%t removed%t", key, found, removed[key])
					}
					found = false
					for _, c := range residual {
						if c.Cell == key {
							found = true
							if c.SpellEffects[layer] != 0 || c.LayerCount != 0 {
								t.Fatalf("retired residual layer/count at%04x", key)
							}
						}
					}
					if found == removed[key] {
						t.Fatalf("empty-node cleanup/residual%04x: found%t removed%t", key, found, removed[key])
					}
				}
				if !wall {
					if d := light1162Differences(front.live.world, state, source, 19); len(d) != 0 {
						t.Fatal("unchanged Light retirement", d)
					}
				} else {
					p, _ := front.live.world.SavedCellPlanes()
					if p.Cost[subject] != 8 || p.Static[subject] != 0 || p.Dynamic[subject] != 0x20 {
						t.Fatalf("wall removal must recompute cost8/static0/dynamic20, got%02x/%02x/%02x", p.Cost[subject], p.Static[subject], p.Dynamic[subject])
					}
				}
			}
			snapshot := producer1162Snapshot(t, f)
			check(f, snapshot.SavedDocument)
			// Suppressing the current-cell or plane projector must fail this
			// direct oracle. Native validation separately rejects stale payloads.
			for _, control := range []string{"layer", "count", "planes", "lost-cell"} {
				bad, err := cloneSavedDocument(snapshot.SavedDocument)
				if err != nil {
					t.Fatal(err)
				}
				key := keys[0]
				if control != "planes" {
					if len(bad.Document.World.Cells) == 0 {
						t.Fatal("no unrelated current cell for loss control")
					}
					key = bad.Document.World.Cells[0].Cell
				}
				for i := range bad.Document.World.Cells {
					c := &bad.Document.World.Cells[i]
					if c.Cell != key {
						continue
					}
					switch control {
					case "layer":
						c.Layers[layer] = drivers.Areas[0].Identity
					case "count":
						c.LayerCount = 1
					case "lost-cell":
						bad.Document.World.Cells = slices.Delete(bad.Document.World.Cells, i, i+1)
					}
					break
				}
				if control == "planes" {
					for i := range bad.Document.World.Blocks {
						if bad.Document.World.Blocks[i].Cell == key {
							bad.Document.World.Blocks[i].Static ^= 1
							break
						}
					}
				}
				if d := cellProducer1162Differences(f.live.world, bad); len(d) == 0 {
					t.Fatal("stale/lost cell projector escaped", control)
				}
				before := f.live.world.Hash()
				if err := restoreSavedDocument(&Mission{World: f.live.world}, bad); err == nil || f.live.world.Hash() != before {
					t.Fatal("stale cell native admission was not atomic", control, err)
				}
			}
			fresh := producer1162Fresh(t, f, app, store, path)
			// Inspect adopted Document before any new projection could repair it.
			check(fresh, fresh.live.mission.state.savedDocument)
			// A real next command after fresh LOAD must be accepted identically;
			// controlled local damage avoids assuming an original future AI order.
			var actor sim.EntityID
			var hp int32
			found := false
			for _, e := range f.live.world.Entities() {
				if e.Alive() && e.HP > 10 {
					actor, hp, found = e.ID, e.HP, true
					break
				}
			}
			if !found {
				t.Fatal("no actor for explicit next damage command")
			}
			for _, front := range []*FrontEnd{f, fresh} {
				headlessDamage(t, front.live.world, actor, 1)
				sim.Step(front.live.world, nil)
				e, exists := entityIn(front.live.world.Entities(), actor)
				if !exists || e.HP != hp-1 {
					t.Fatal("next damage command after expiry LOAD", e.HP, hp)
				}
				check(front, producer1162Snapshot(t, front).SavedDocument)
			}
			if f.live.world.Hash() != fresh.live.world.Hash() {
				t.Fatal("next action diverged after retired-cut LOAD")
			}
			t.Logf("%s: initial raw cells/planes agree; %d simulation steps retire57 cells; direct World/Document, ordinary menu SAVE, private source removal, fresh LOAD before re-projection, next damage1; four stale/loss controls rejected atomically", name, ticks)
		})
	}
}

func worldEffects1162WallSource(t *testing.T, raw []byte) ([]byte, uint16) {
	t.Helper()
	doc, err := sav.DecodeDocumentData(raw)
	if err != nil {
		t.Fatal(err)
	}
	root := &doc.Objects[doc.World.Effects[0]-1]
	identity, err := savedStructureValue(root, "Identity")
	if err != nil {
		t.Fatal(err)
	}
	savedObjectSetValue(root, "T0C", 19)
	savedObjectSetValue(root, "AE4C", 1)
	refs, _ := savedObjectRefs(root, "AE44")
	for _, name := range []string{"T0C", "E0C", "E3C", "E3D", "E40"} {
		savedObjectSetValue(&doc.Objects[refs[0]-1], name, 0)
	}
	keys := map[uint16]bool{}
	var subject uint16
	for i := range doc.World.Cells {
		c := &doc.World.Cells[i]
		if c.Layers[4] != identity {
			continue
		}
		c.Layers[3], c.Layers[4] = identity, 0
		keys[c.Cell] = true
		if subject == 0 && c.GroundActor == 0 && c.AirActor == 0 && c.Building == 0 && c.Sack == 0 {
			subject, c.Cost, c.Static = c.Cell, 8, 0
		}
	}
	if subject == 0 {
		t.Fatal("no empty cell for explicit wall baseline")
	}
	for i := range doc.World.Blocks {
		b := &doc.World.Blocks[i]
		if keys[b.Cell] {
			b.Static |= 5
			b.Dyn |= 5
		}
		if b.Cell == subject {
			b.Static, b.Dyn = 0x25, 0x25
		}
	}
	controlled, err := sav.EncodeDocumentData(doc)
	if err != nil {
		t.Fatal(err)
	}
	return controlled, subject
}

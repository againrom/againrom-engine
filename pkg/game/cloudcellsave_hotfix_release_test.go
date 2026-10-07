package game

import (
	"bytes"
	"math/bits"
	"reflect"
	"testing"

	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
)

type cloudCellTick struct {
	Tick                    uint64
	Present                 bool
	Spell, Remaining, Power uint16
	Phase                   uint8
	Caster                  sim.EntityID
	HasCaster               bool
	Cells                   []uint16
	CasterHP, CasterMana    int32
}

func cloudCellWitness(t *testing.T, f *FrontEnd, caster sim.EntityID) cloudCellTick {
	t.Helper()
	w := f.live.world
	out := cloudCellTick{Tick: w.Tick()}
	areas, err := w.NativeAreaSaveStates()
	if err != nil {
		t.Fatal(err)
	}
	for _, a := range areas {
		if a.Mode != sim.AreaModeCloud {
			continue
		}
		if out.Present {
			t.Fatal("more than one current cloud")
		}
		out.Present, out.Spell, out.Remaining, out.Phase = true, a.Spell, a.Remaining, a.Policy.CloudPhase
		out.Power, out.Caster, out.HasCaster, out.Cells = a.Policy.Power, a.Policy.Caster, a.Policy.HasCaster, a.Cells
	}
	e := releaseEntity(t, f.live, caster)
	out.CasterHP, out.CasterMana = e.HP, e.Mana
	return out
}

// cloudCellLayers maps every cell whose area layer holds the document's one
// cloud AreaEffect identity to that record.
func cloudCellLayers(t *testing.T, doc sav.DocumentData, layer uint8) map[uint16]sav.DocumentCellData {
	t.Helper()
	var identity uint32
	for i := range doc.Objects {
		if doc.Objects[i].Class != "AreaEffect" {
			continue
		}
		if identity != 0 {
			t.Fatal("SAV holds more than one AreaEffect")
		}
		identity, _ = savedStructureValue(&doc.Objects[i], "Identity")
	}
	if identity == 0 {
		t.Fatal("SAV holds no AreaEffect")
	}
	out := map[uint16]sav.DocumentCellData{}
	for _, c := range doc.World.Cells {
		if c.Layers[layer] == identity {
			if _, dup := out[c.Cell]; dup {
				t.Fatalf("cell %04x has two records", c.Cell)
			}
			out[c.Cell] = c
		}
	}
	return out
}

// cloudCellWorld is the document's World with the cloud's layer pointer
// written as 1. Every SAVE reserves a fresh archive key for a current area.
func cloudCellWorld(t *testing.T, doc sav.DocumentData, layer uint8) sav.DocumentWorldData {
	t.Helper()
	w := *doc.World
	w.Cells = append([]sav.DocumentCellData(nil), w.Cells...)
	painted := cloudCellLayers(t, doc, layer)
	for i := range w.Cells {
		if _, ok := painted[w.Cells[i].Cell]; ok {
			w.Cells[i].Layers[layer] = 1
		}
	}
	return w
}

// A mission whose world runs on a loaded engine SAV: the party mage casts a
// cloud through the map's own cast command over cells the loaded document has
// no record for. SAVE writes the file with a record and the cloud's layer
// pointer on every covered cell, a cold LOAD from the main menu restores the
// same cloud, a second SAVE writes the same World, and the loaded world keeps
// the uninterrupted world's hash each tick until the cloud ends.
func TestReleaseCloudOverUnrecordedCellsSavesAndReloads(t *testing.T) {
	f := releaseFront(t)
	f.SetDeterministicFrames(true)
	const mission = 20
	if err := f.App("cloud cell save").OpenMission(f.MissionOpenerWith(mission, MissionPartyAs(true, f.StartWeapon.Value(), f.Bodies, f.Table))); err != nil {
		t.Fatal(err)
	}
	store := SaveStore{Dir: t.TempDir()}
	opened, _ := deadPatrolSave(t, f, store)
	f = loadLocalLegacySave(t, store, opened)
	loadedCells := map[uint16]sav.DocumentCellData{}
	for _, c := range f.live.mission.state.savedDocument.Document.World.Cells {
		loadedCells[c.Cell] = c
	}

	var mage sim.Entity
	for _, e := range f.live.world.Entities() {
		if e.Owner == sim.SelfSlot && e.Alive() && e.KnownSpells != 0 {
			mage = e
		}
	}
	spell, layer := uint16(0), uint8(0)
	for known := mage.KnownSpells; known != 0 && spell == 0; known &= known - 1 {
		id := uint16(bits.TrailingZeros32(known))
		rule, ok := f.live.world.Spell(uint32(id))
		if l, cloud := areaLayerIndex(id); ok && cloud && rule.Area && rule.AreaMode() == sim.AreaModeCloud {
			spell, layer = id, l
		}
	}
	if spell == 0 {
		t.Fatalf("mission %d mage %d knows no cloud spell: %032b", mission, mage.ID, mage.KnownSpells)
	}
	f.live.attackOrCast(uint32(mage.ID), 0, uint32(spell), int(mage.X)+3, int(mage.Y), true)
	for i := 0; !cloudCellWitness(t, f, mage.ID).Present; i++ {
		if i > 256 {
			t.Fatalf("mage %d never cast cloud spell %d", mage.ID, spell)
		}
		f.live.tick()
	}
	f.live.tick()
	at := cloudCellWitness(t, f, mage.ID)
	var fresh, recorded []uint16
	for _, key := range at.Cells {
		if _, ok := loadedCells[key]; ok {
			recorded = append(recorded, key)
		} else {
			fresh = append(fresh, key)
		}
	}
	if len(fresh) == 0 || len(recorded) == 0 {
		t.Fatalf("cloud cells %04x: %d without and %d with a loaded record, want both", at.Cells, len(fresh), len(recorded))
	}

	saved, _, err := f.Snapshot(true)
	if err != nil {
		t.Fatal(err)
	}
	name, doc := deadPatrolSave(t, f, store)
	layers := cloudCellLayers(t, doc, layer)
	if len(layers) != len(at.Cells) {
		t.Fatalf("SAV paints %d cells with the cloud, want %d", len(layers), len(at.Cells))
	}
	for _, key := range at.Cells {
		c, ok := layers[key]
		if !ok {
			t.Fatalf("SAV cell %04x lacks the cloud's layer pointer", key)
		}
		var count uint8
		for _, p := range c.Layers {
			if p != 0 {
				count++
			}
		}
		if c.LayerCount != count {
			t.Fatalf("SAV cell %04x layer count %d, want %d", key, c.LayerCount, count)
		}
		if old, ok := loadedCells[key]; ok {
			old.Layers[layer], old.LayerCount = c.Layers[layer], c.LayerCount
			if c != old {
				t.Fatalf("recorded cell %04x changed beyond its layer: %+v want %+v", key, c, old)
			}
		} else if c.GroundActor|c.AirActor|c.Building|c.Sack != 0 || c.Operation != 0 {
			t.Fatalf("constructed cell %04x carries an occupant or tail: %+v", key, c)
		}
	}

	g := loadLocalLegacySave(t, store, name)
	loaded, _, err := g.Snapshot(true)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(loaded.World, saved.World) {
		var left, right sim.World
		if left.UnmarshalBinary(saved.World) == nil && right.UnmarshalBinary(loaded.World) == nil {
			currentMenuWorldDiagnostics(t, &left, &right)
		}
		t.Fatal("the loaded World differs from the saved one")
	}
	if got := cloudCellWitness(t, g, mage.ID); !reflect.DeepEqual(got, at) {
		t.Fatalf("LOAD cloud %+v, want %+v", got, at)
	}
	_, again := deadPatrolSave(t, g, SaveStore{Dir: t.TempDir()})
	if !reflect.DeepEqual(cloudCellWorld(t, again, layer), cloudCellWorld(t, doc, layer)) {
		currentItemFieldDiagnostics(t, "World", reflect.ValueOf(cloudCellWorld(t, doc, layer)), reflect.ValueOf(cloudCellWorld(t, again, layer)))
		t.Fatal("SAVE after LOAD writes a different World")
	}
	end := int(at.Remaining) + 2
	for tick := 1; tick <= end; tick++ {
		f.live.tick()
		g.live.tick()
		want, got := cloudCellWitness(t, f, mage.ID), cloudCellWitness(t, g, mage.ID)
		if !reflect.DeepEqual(got, want) || g.live.world.Hash() != f.live.world.Hash() {
			t.Fatalf("tick %d after LOAD: %+v hash %016x, want %+v hash %016x", tick, got, g.live.world.Hash(), want, f.live.world.Hash())
		}
		if tick == end && want.Present {
			t.Fatalf("cloud still present %d ticks after SAVE", end)
		}
	}
	t.Logf("mission %d mage %d spell %d layer %d: %d cells, %d constructed; LOAD, second SAVE and %d ticks match, World hash included", mission, mage.ID, spell, layer, len(at.Cells), len(fresh), end)
}

package game

import (
	"encoding/binary"
	"reflect"
	"testing"

	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

// openMissionGameMenu dismisses pending notices with Escape until the menu opens.
func openMissionGameMenu(t *testing.T, app *ui.App) {
	t.Helper()
	for tries := 0; app.Screen() != ui.ScreenGameMenu && tries < 6; tries++ {
		if err := app.HeadlessKey("escape"); err != nil {
			t.Fatal(err)
		}
	}
	if app.Screen() != ui.ScreenGameMenu {
		t.Fatalf("ordinary generated map menu did not open: screen %s", app.Screen())
	}
}

func currentActorCells(w *sim.World, id sim.EntityID) map[uint16]uint8 {
	_, cells, _, _ := w.SavedActorMotions()
	owned := map[uint16]uint8{}
	for _, cell := range cells {
		if cell.Ground.Bound && cell.Ground.Entity == id {
			owned[cell.Cell] |= 1
		}
		if cell.Air.Bound && cell.Air.Entity == id {
			owned[cell.Cell] |= 2
		}
	}
	return owned
}

// Position and occupancy are independent current values during a crossing.
// The caller reads both before encoding; this oracle decodes only the wire.
func generatedCurrentWireDifference(raw []byte, runtime uint32, hp int32, pos [2]int32, owned map[uint16]uint8, structures, sacks int) string {
	d, err := sav.DecodeDocumentData(raw)
	if err != nil {
		return err.Error()
	}
	if d.World == nil || len(d.World.Sacks) != sacks || len(d.World.Buildings) != structures {
		return "current graph roots lost"
	}
	var actor *sav.DocumentRecordData
	for i := range d.Objects {
		r := &d.Objects[i]
		id, err := savedStructureValue(r, "RuntimeID")
		if err == nil && id == runtime && (r.Class == "Human" || r.Class == "Unit" || r.Class == "Humanoid") {
			if actor != nil {
				return "ambiguous runtime actor"
			}
			actor = r
		}
	}
	if actor == nil {
		return "runtime actor lost"
	}
	health, err := savedStructureValue(actor, "Health")
	if err != nil || int32(int16(health)) != hp {
		return "current health lost"
	}
	p, err := savedMotionRaw(actor, "Block12", 12)
	if err != nil {
		return err.Error()
	}
	cell := binary.LittleEndian.Uint16(p)
	if [2]int32{int32(cell&255)*256 + int32(p[4]), int32(cell>>8)*256 + int32(p[5])} != pos {
		return "current near-cell/fraction lost"
	}
	key, _ := savedStructureValue(actor, "Identity")
	linked := map[uint16]uint8{}
	for _, c := range d.World.Cells {
		if c.GroundActor == key {
			linked[c.Cell] |= 1
		}
		if c.AirActor == key {
			linked[c.Cell] |= 2
		}
	}
	if !reflect.DeepEqual(linked, owned) {
		return "current cell ownership lost"
	}
	return ""
}

// The ordinary writer supplies the transport binding. Expected health and
// occupancy come directly from the independent current World.
func generatedCurrentDocument(t *testing.T, f *FrontEnd, s Snapshot) sav.DocumentData {
	t.Helper()
	before := f.live.world.Hash()
	raw, err := f.ExportCurrentSave(s, "current combat control")
	if err != nil || f.live.world.Hash() != before {
		t.Fatal("current SAVE failed or mutated World", err)
	}
	doc, err := sav.DecodeDocumentData(raw)
	if err != nil {
		t.Fatal(err)
	}
	return doc
}

func generatedActorRecord(t *testing.T, doc *sav.DocumentData, id sim.EntityID) *sav.DocumentRecordData {
	t.Helper()
	actions, err := readCurrentActions(doc)
	if err != nil || actions == nil {
		t.Fatal("current binding absent", err)
	}
	for _, b := range actions.Bindings {
		if !b.Structure && !b.Missing && b.ID == id && b.Object != 0 && int(b.Object) <= len(doc.Objects) {
			return &doc.Objects[b.Object-1]
		}
	}
	t.Fatalf("current actor %d has no ordinary record", id)
	return nil
}

func generatedCurrentActorCells(w *sim.World, e sim.Entity) map[uint16]uint8 {
	motions, _, _, _ := w.SavedActorMotions()
	for _, m := range motions {
		if m.Entity == e.ID && m.Current {
			return currentActorCells(w, e.ID)
		}
	}
	out := map[uint16]uint8{}
	if !e.Alive() || e.OffMap {
		return out
	}
	layer := uint8(1)
	if e.Domain == sim.DomainAir {
		layer = 2
	}
	for y := int32(0); y < int32(max(uint8(1), e.TokenSize)); y++ {
		for x := int32(0); x < int32(max(uint8(1), e.TokenSize)); x++ {
			out[uint16(e.X+x)|uint16(e.Y+y)<<8] = layer
		}
	}
	return out
}

func assertNativeWorldWire(t *testing.T, f *FrontEnd, s Snapshot) {
	t.Helper()
	if s.SavedDocument != nil {
		t.Fatal("native mission entry fabricated an imported document")
	}
	var hero sim.Entity
	found := false
	for _, e := range f.live.world.Entities() {
		if e.Owner == sim.SelfSlot {
			hero, found = e, true
			break
		}
	}
	if !found {
		t.Fatal("native hero absent")
	}
	doc := generatedCurrentDocument(t, f, s)
	target := generatedActorRecord(t, &doc, hero.ID)
	runtime, err := savedStructureValue(target, "RuntimeID")
	if err != nil || runtime == 0 {
		t.Fatal("ordinary runtime missing", err)
	}
	pos, owned := world1170Position(f.live.world, hero), generatedCurrentActorCells(f.live.world, hero)
	oracle := func(b []byte) string {
		return generatedCurrentWireDifference(b, runtime, hero.HP, pos, owned, len(f.live.world.Structures()), len(f.live.world.Sacks()))
	}
	wire, err := sav.EncodeDocumentData(doc)
	if err != nil {
		t.Fatal(err)
	}
	if why := oracle(wire); why != "" {
		t.Fatal("current initial state failed wire oracle", why)
	}
	for _, loss := range []string{"position", "identity permutation", "occupancy"} {
		t.Run(loss, func(t *testing.T) {
			d, err := sav.CloneDocumentData(doc)
			if err != nil {
				t.Fatal(err)
			}
			actor := generatedActorRecord(t, &d, hero.ID)
			key, _ := savedStructureValue(actor, "Identity")
			switch loss {
			case "position":
				p, _ := savedMotionRaw(actor, "Block12", 12)
				p[0]++
			case "identity permutation":
				for i := range d.Objects {
					r := &d.Objects[i]
					if r != actor && (r.Class == "Human" || r.Class == "Unit") {
						other, _ := savedStructureValue(r, "Identity")
						savedObjectSetValue(r, "Identity", key)
						savedObjectSetValue(actor, "Identity", other)
						break
					}
				}
			case "occupancy":
				for i := range d.World.Cells {
					c := &d.World.Cells[i]
					if c.GroundActor == key {
						c.GroundActor = 0
					}
					if c.AirActor == key {
						c.AirActor = 0
					}
				}
			}
			raw, err := sav.EncodeDocumentData(d)
			if err != nil {
				t.Fatal(err)
			}
			if why := oracle(raw); why == "" {
				t.Fatal(loss, "escaped independent wire oracle")
			}
		})
	}
}

func assertCurrentCombatWire(t *testing.T, f *FrontEnd, s Snapshot, initial Snapshot) {
	t.Helper()
	var baseline sim.World
	if err := baseline.UnmarshalBinary(initial.World); err != nil {
		t.Fatal(err)
	}
	hp := map[sim.EntityID]int32{}
	for _, e := range baseline.Entities() {
		hp[e.ID] = e.HP
	}
	var target sim.Entity
	found := false
	for _, e := range f.live.world.Entities() {
		if before, ok := hp[e.ID]; ok && e.HP < before {
			target, found = e, true
			break
		}
	}
	if !found {
		t.Fatal("no real changed health relative to current initial World")
	}
	doc := generatedCurrentDocument(t, f, s)
	runtime, err := savedStructureValue(generatedActorRecord(t, &doc, target.ID), "RuntimeID")
	if err != nil {
		t.Fatal(err)
	}
	oracle := func(raw []byte) string {
		return generatedCurrentWireDifference(raw, runtime, target.HP, world1170Position(f.live.world, target), generatedCurrentActorCells(f.live.world, target), len(f.live.world.Structures()), len(f.live.world.Sacks()))
	}
	raw, err := sav.EncodeDocumentData(doc)
	if err != nil {
		t.Fatal(err)
	}
	if why := oracle(raw); why != "" {
		t.Fatal("current combat wire lost current state", why)
	}
	stale, err := sav.CloneDocumentData(doc)
	if err != nil {
		t.Fatal(err)
	}
	savedObjectSetValue(generatedActorRecord(t, &stale, target.ID), "Health", uint32(uint16(hp[target.ID])))
	staleRaw, err := sav.EncodeDocumentData(stale)
	if err != nil {
		t.Fatal(err)
	}
	if why := oracle(staleRaw); why == "" {
		t.Fatal("stale pre-combat health passed current-state oracle")
	}
	before := f.live.world.Hash()
	if _, _, err := f.RestoreOriginal(raw[:8]); err == nil || f.live.world.Hash() != before {
		t.Fatal("truncated ordinary LOAD was accepted or changed live state", err)
	}
}

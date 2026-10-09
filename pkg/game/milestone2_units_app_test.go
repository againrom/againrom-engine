package game

import (
	"encoding/binary"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"againrom/pkg/formats/sav"
)

func unit1156InitialCheck(t *testing.T, want unitScalarSet, ms *Mission) {
	t.Helper()
	document := want.documentDifferences(ms.savedDocument)
	differences, _, n := want.worldDifferences(ms.World, ms.ActorManifest)
	if len(document) != 0 || len(differences) != 0 || n == 0 {
		t.Fatalf("raw -> imported Document before Snapshot: %v; raw -> World: compared=%d differences=%v", document, n, differences)
	}
}

// Current native persistence is checked separately from initial raw restore.
// Snapshot may legitimately project current pools, positions and order fields.
// This extraction reads only actual selected state; it never becomes the raw
// oracle and supplies no expected original field value to initial acceptance.
func unit1156CurrentDocument(t *testing.T, state *SnapshotSAVDocument) map[int]unit1156Record {
	t.Helper()
	if state == nil || state.Document == nil || state.Unavailable != "" {
		t.Fatal("native Unit document unavailable")
	}
	out := map[int]unit1156Record{}
	for i, object := range state.Document.Objects {
		if !unit1156Class(object.Class) {
			continue
		}
		r := unit1156Record{class: object.Class, values: map[string]uint32{}, raw: map[string][]byte{}}
		for _, field := range object.Values {
			switch field.Name {
			case "RuntimeID", "T0C", "T0E", "T08", "T18", "T1C", "Identity", "Reference", "U49", "U4A", "U4B", "U4C", "U60", "U61", "U6C", "Body", "Reaction", "Mind", "Spirit", "Speed", "U8E", "U90", "Capacity", "Health", "HealthMax", "HealthRegen", "Mana", "ManaMax", "ManaRegen", "UA2", "UA3", "UA0", "UA4", "U12C", "U130", "U134", "U135", "U136", "U138", "Stage", "U148", "U144":
				if _, exists := r.values[field.Name]; exists {
					t.Fatal("duplicate current scalar", field.Name)
				}
				r.values[field.Name] = field.Value
			}
		}
		for _, field := range object.Raw {
			switch field.Name {
			case "Block12", "U50", "U54", "U58":
				r.raw[field.Name] = append([]byte(nil), field.Bytes...)
			}
		}
		for _, field := range object.Texts {
			if field.Name == "Name" {
				r.name = field.Value
			}
		}
		if len(r.values) != 42 || len(r.raw) != 4 {
			t.Fatal("incomplete current Unit scalar population", i, len(r.values), len(r.raw))
		}
		out[i] = r
	}
	return out
}

func unit1156App(t *testing.T, raw []byte, front func(*testing.T) *FrontEnd) {
	t.Helper()
	f := front(t)
	f.SetDeterministicFrames(true)
	source, err := sav.Open(raw)
	if err != nil {
		t.Fatal(err)
	}
	want, err := unitScalarExpected(source, raw)
	if err != nil {
		t.Fatal(err)
	}
	ms, _, err := loadOriginalMission(f, raw)
	if err != nil {
		t.Fatal(err)
	}
	unit1156InitialCheck(t, want, ms)
	app := f.App("Unit scalar acceptance")
	app.Layout(1024, 768)
	privateSource := filepath.Join(t.TempDir(), "game1156.sav")
	if err := os.WriteFile(privateSource, raw, 0600); err != nil {
		t.Fatal(err)
	}
	store := SaveStore{Dir: t.TempDir()}
	save, list, load := nativeContinuationSeams1170(t, f, store, OriginalStore{Dir: filepath.Dir(privateSource)}, nil)
	app.SetSaveSeams(save, list, load)
	groundAppLoad(t, app, list, "game1156.sav")
	unit1156InitialCheck(t, want, f.live.mission.state)
	first := f.live
	groundAppLoad(t, app, list, "game1156.sav")
	if f.live == first {
		t.Fatal("map-menu original LOAD did not replace the driver")
	}
	unit1156InitialCheck(t, want, f.live.mission.state)
	driver := f.live
	before, _, err := f.Snapshot(true)
	if err != nil {
		t.Fatal(err)
	}
	current := unit1156CurrentDocument(t, before.SavedDocument)
	hash := driver.world.Hash()
	if err := app.HeadlessKey("escape"); err != nil {
		t.Fatal(err)
	}
	if err := app.HeadlessGameMenuAction("save"); err != nil {
		t.Fatal(err)
	}
	entries, err := listAGS(store)
	if err != nil || len(entries) != 1 || filepath.Ext(entries[0].Name) != ".ags" {
		t.Fatalf("menu SAVE using the explicit AGS codec: %v %v", entries, err)
	}
	if err := os.Remove(privateSource); err != nil {
		t.Fatal(err)
	}
	fresh := front(t)
	fresh.SetDeterministicFrames(true)
	app2 := fresh.App("Unit scalar fresh LOAD")
	app2.Layout(1024, 768)
	save, list, load = nativeContinuationSeams1170(t, fresh, store, OriginalStore{}, nil)
	app2.SetSaveSeams(save, list, load)
	groundAppLoad(t, app2, list, entries[0].Name)
	if fresh.live.world.Hash() != hash {
		t.Fatal("native LOAD changed World hash")
	}
	if !reflect.DeepEqual(current, unit1156CurrentDocument(t, fresh.live.mission.state.savedDocument)) {
		t.Fatal("native LOAD lost selected retained Unit state despite unchanged World hash")
	}
	start := driver.world.Tick()
	for i := 1; i <= 20; i++ {
		previous := driver.world.Tick()
		driver.tick()
		fresh.live.tick()
		if driver.world.Tick() <= previous || driver.world.Hash() != fresh.live.world.Hash() {
			t.Fatalf("native continuation did not advance equally at step %d (start %d now %d)", i, start, driver.world.Tick())
		}
		left, _, leftErr := f.Snapshot(true)
		right, _, rightErr := fresh.Snapshot(true)
		if leftErr != nil || rightErr != nil {
			t.Fatal("continuation Snapshot", leftErr, rightErr)
		}
		if !reflect.DeepEqual(unit1156CurrentDocument(t, left.SavedDocument), unit1156CurrentDocument(t, right.SavedDocument)) {
			t.Fatalf("selected retained Unit state differs at advancing step %d", i)
		}
	}
	t.Logf("unit scalars: %d raw records through both original App LOAD doors, menu SAVE using the explicit AGS codec, source removal, fresh FrontEnd LOAD and 20 advancing hash+retained-state pairs", len(want.records))
}

func unit1156FixtureFront(t *testing.T) *FrontEnd {
	f := poolFixtureFront(t, 91)
	f.Campaign = resolved(saveCampaign(), nil)
	return f
}

func unit1156AppFixture(t *testing.T) []byte {
	t.Helper()
	f := unit1156FixtureFront(t)
	a := &poolFixtureActor{mapID: 91, cell: 0x0605, hp: 31, maxHP: 101, mana: 23, maxMana: 103, profile: literalProfile1107(), name: "Unit scalar fixture"}
	body := poolFixtureBody([]*poolFixturePlayer{{}, {groups: [][]*poolFixtureActor{{a}}}}, nil)
	// The selected input has nonzero retained-only fields, differing words,
	// and full fractional sight. The writer's positions do not use a decoder.
	binary.LittleEndian.PutUint16(body[a.off+23:], 0x5a63)
	control := a.off + 509
	binary.LittleEndian.PutUint32(body[control+12:], 0x1234abcd) // U58
	body[control+16], body[control+17] = 0x91, 0xa2
	state := a.off + 533 + len(a.name)
	binary.LittleEndian.PutUint16(body[state+30:], 11)
	binary.LittleEndian.PutUint16(body[state+32:], 0x0837)
	binary.LittleEndian.PutUint32(body[state+35:], 0x112233)
	body[state+41] = 0x83
	binary.LittleEndian.PutUint32(body[state+42:], 0x53647586)
	binary.LittleEndian.PutUint32(body[state+47:], 0x12345678)
	binary.LittleEndian.PutUint32(body[state+51:], 0x87654321)
	return completeDocumentTail1115(t, f, savedContainer(body))
}

func TestUnit1156NonzeroAppContinuation(t *testing.T) {
	unit1156App(t, unit1156AppFixture(t), unit1156FixtureFront)
}

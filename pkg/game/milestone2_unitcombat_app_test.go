package game

import (
	"encoding/binary"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"againrom/pkg/formats/sav"
)

func unit1158InitialCheck(t *testing.T, want unit1158Set, ms *Mission) {
	t.Helper()
	document := want.documentDifferences(ms.savedDocument)
	differences, _, n := want.entityDifferences(ms.World.Entities(), ms.savedDocument)
	if len(document) != 0 || len(differences) != 0 || n == 0 {
		t.Fatalf("raw -> imported Document BEFORE Snapshot: %v; raw -> World: compared=%d differences=%v", document, n, differences)
	}
}

// Current-state extraction is only the native-persistence assertion. It never
// supplies expected values or counts to the independent original-byte reader.
func unit1158CurrentDocument(t *testing.T, state *SnapshotSAVDocument) map[int]unitCombatRecord {
	t.Helper()
	if state == nil || state.Document == nil || state.Unavailable != "" {
		t.Fatal("native Unit combat Document unavailable")
	}
	out := map[int]unitCombatRecord{}
	for i, object := range state.Document.Objects {
		if !unit1156Class(object.Class) {
			continue
		}
		r := unitCombatRecord{class: object.Class, raw: map[string][]byte{}}
		for _, block := range unit1158Blocks {
			for _, field := range object.Raw {
				if field.Name != block.name {
					continue
				}
				if _, exists := r.raw[field.Name]; exists || len(field.Bytes) != block.width || block.name == "H1CC" && object.Class == "Unit" {
					t.Fatalf("current DTO %d %s duplicate/wrong-width/class block %s", i+1, object.Class, field.Name)
				}
				r.raw[field.Name] = append([]byte(nil), field.Bytes...)
			}
		}
		n := 5
		if object.Class == "Unit" {
			n = 4
		}
		if len(r.raw) != n {
			t.Fatalf("current DTO %d %s missing combat blocks", i+1, object.Class)
		}
		out[i] = r
	}
	return out
}

func unit1158App(t *testing.T, raw []byte, front func(*testing.T) *FrontEnd) {
	t.Helper()
	source, err := sav.Open(raw)
	if err != nil {
		t.Fatal(err)
	}
	want, err := readUnitCombatExpected(source, raw)
	if err != nil {
		t.Fatal(err)
	}
	for _, door := range []string{"title", "map-menu"} {
		t.Run(door, func(t *testing.T) {
			f := front(t)
			f.SetDeterministicFrames(true)
			app := f.App("Unit combat acceptance")
			app.Layout(1024, 768)
			privateSource := filepath.Join(t.TempDir(), "game1158.sav")
			if err := os.WriteFile(privateSource, raw, 0600); err != nil {
				t.Fatal(err)
			}
			store := SaveStore{Dir: t.TempDir()}
			save, list, load := nativeContinuationSeams1170(t, f, store, OriginalStore{Dir: filepath.Dir(privateSource)}, nil)
			app.SetSaveSeams(save, list, load)
			if door == "map-menu" {
				groundAppLoad(t, app, list, "game1158.sav")
				unit1158InitialCheck(t, want, f.live.mission.state)
			}
			previousDriver := f.live
			groundAppLoad(t, app, list, "game1158.sav")
			if f.live == nil || f.live == previousDriver {
				t.Fatal("original LOAD did not replace the driver")
			}
			unit1158InitialCheck(t, want, f.live.mission.state)
			driver := f.live
			before, _, err := f.Snapshot(true)
			if err != nil {
				t.Fatal(err)
			}
			current := unit1158CurrentDocument(t, before.SavedDocument)
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
			if _, err := os.Stat(privateSource); !os.IsNotExist(err) {
				t.Fatalf("private original source remains: %v", err)
			}
			fresh := front(t)
			fresh.SetDeterministicFrames(true)
			app2 := fresh.App("Unit combat fresh native LOAD")
			app2.Layout(1024, 768)
			save, list, load = nativeContinuationSeams1170(t, fresh, store, OriginalStore{}, nil)
			app2.SetSaveSeams(save, list, load)
			groundAppLoad(t, app2, list, entries[0].Name)
			if fresh.live.world.Hash() != hash {
				t.Fatal("native LOAD changed World hash")
			}
			t.Log("native LOAD World hash matches; comparing retained Document separately")
			if !reflect.DeepEqual(current, unit1158CurrentDocument(t, fresh.live.mission.state.savedDocument)) {
				t.Fatal("native LOAD lost retained Unit combat state despite unchanged World hash")
			}
			for i := 1; i <= 20; i++ {
				leftTick, rightTick := driver.world.Tick(), fresh.live.world.Tick()
				driver.tick()
				fresh.live.tick()
				if driver.world.Tick() <= leftTick || fresh.live.world.Tick() <= rightTick || driver.world.Hash() != fresh.live.world.Hash() {
					t.Fatalf("native continuation failed advancement/hash at step %d", i)
				}
				left, _, leftErr := f.Snapshot(true)
				right, _, rightErr := fresh.Snapshot(true)
				if leftErr != nil || rightErr != nil {
					t.Fatal("continuation Snapshot", leftErr, rightErr)
				}
				if !reflect.DeepEqual(unit1158CurrentDocument(t, left.SavedDocument), unit1158CurrentDocument(t, right.SavedDocument)) {
					t.Fatalf("retained Unit combat state differs at advancing step %d", i)
				}
			}
			t.Logf("unit combat: %d raw actors, %s original App LOAD, menu SAVE, private-source removal, fresh FrontEnd native LOAD, 20 EACH advancing hash+retained-state pairs", len(want.records), door)
		})
	}
}

func unit1158FixtureFront(t *testing.T) *FrontEnd {
	f := poolFixtureFront(t, 91, 92)
	f.Campaign = resolved(saveCampaign(), nil)
	return f
}

func unit1158AppFixture(t *testing.T) []byte {
	t.Helper()
	f := unit1158FixtureFront(t)
	a := &poolFixtureActor{mapID: 91, cell: 0x0605, hp: 31, maxHP: 101, mana: 23, maxMana: 103, name: "Combat Unit", profile: literalProfile1107()}
	b := &poolFixtureActor{mapID: 92, cell: 0x0807, hp: 37, maxHP: 107, mana: 29, maxMana: 109, name: "Combat Human", human: true, profile: literalProfile1107()}
	body := poolFixtureBody([]*poolFixturePlayer{{}, {groups: [][]*poolFixtureActor{{a, nil, b, a}}}}, nil)
	for _, actor := range []*poolFixtureActor{a, b} {
		// Literal fixture writer: Token37 + effects/counts8 precede blocks.
		at := actor.off + 45
		for i, value := range []uint16{0x8001, 0xfffe, 0x1234, 0x5678, 0xabcd, 0x7ffe} {
			binary.LittleEndian.PutUint16(body[at+2+2*i:], value)
		}
		for i := range 24 {
			body[at+46+i] = byte(0x51 + i) // complete independent U114
		}
		for i := range 64 {
			body[at+70+i] = byte(0x81 + i) // modifier, including unused tails
		}
		// Keep signed regeneration arithmetic in a small supported domain.
		binary.LittleEndian.PutUint16(body[at+80:], 50)
		binary.LittleEndian.PutUint16(body[at+84:], 50)
	}
	// No holdings/books: Humanoid XP follows CString+55 scalar bytes,
	// U68(2), two presence bytes and Unit's final17 bytes.
	xp := b.off + 609 + len(b.name)
	for i, value := range []uint32{0x80000001, 0xffffffff, 0x12345678, 0x7fffffff, 0x01020304, 0x89abcdef} {
		binary.LittleEndian.PutUint32(body[xp+4*i:], value)
	}
	return completeDocumentTail1115(t, f, savedContainer(body))
}

func TestUnit1158NonzeroAppContinuation(t *testing.T) {
	unit1158App(t, unit1158AppFixture(t), unit1158FixtureFront)
}

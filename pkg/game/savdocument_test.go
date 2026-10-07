package game

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/gob"
	"encoding/json"
	"fmt"
	"hash/crc32"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"testing"

	"againrom/internal/synth"
	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
)

// The existing literal actor writer deliberately omits the application tail.
// Supply its independently spelled full grammar for native-document tests.
func completeDocumentFixture1115(t *testing.T, f *FrontEnd) []byte {
	t.Helper()
	actor := &poolFixtureActor{mapID: 91, cell: 0x0605, hp: 10, maxHP: 100, mana: 10, maxMana: 100, profile: literalProfile1107()}
	b := clockFixture1112(9343, 584, actor)
	return completeDocumentTail1115(t, f, b)
}

func completeDocumentTail1115(t *testing.T, f *FrontEnd, b []byte) []byte {
	t.Helper()
	b = b[:int(binary.LittleEndian.Uint32(b[4:]))+256]
	dir := func(name string, children ...synth.RegNode) synth.RegNode {
		return synth.RegNode{Name: name, Kind: 1, Children: children}
	}
	i32 := func(name string, value int32) synth.RegNode { return synth.RegNode{Name: name, Kind: 2, Int: value} }
	state := synth.Reg(17, []synth.RegNode{
		dir("Character", synth.RegNode{Name: "Name", Kind: 0, Str: ""}),
		dir("CurrentState", i32("InBattle", 1)),
		dir("GameOptions", i32("FlyingHP", 0), i32("Formation", 0), i32("ShowHP", 0), i32("ShowTimeFlow", 0), i32("Speed", 0), i32("Wimpy", 0)),
		dir("Inventory", i32("IsOpen", 0)),
		dir("Objects", synth.RegNode{Name: "Selection", Kind: 6}),
		// -1 is the documented "no spell bound" sentinel (quickSpellsFromOriginalIndices);
		// four literal 0 indices used to collide on the same translated spell ID and trip
		// validateQuickSpells once master's QuickSpells feature landed.
		dir("SpellBook", i32("IsOpen", 0), i32("Pressed", 0), synth.RegNode{Name: "Shortcuts", Kind: 6, Ints: []int32{-1, -1, -1, -1}}),
		dir("View", i32("X", 0), i32("Y", 0)),
		dir("Fog", i32("FirstState", 0), synth.RegNode{Name: "Data", Kind: 6, Ints: []int32{1600}}),
		dir("Projectiles", i32("FreeIndex", 0), synth.RegNode{Name: "IDs", Kind: 6}),
	})
	b = append(b, state...)
	b = append(b, encodeOriginalCampaignProjection(campaignProjectionAt(f.Campaign.Value(), 10))...)
	if _, err := sav.DecodeDocumentData(b); err != nil {
		t.Fatal("literal complete fixture", err)
	}
	return b
}

func documentSnapshot1115(t *testing.T) (*FrontEnd, Snapshot) {
	t.Helper()
	f := poolFixtureFront(t, 91)
	f.Campaign = resolved(saveCampaign(), nil)
	source := completeDocumentFixture1115(t, f)
	open, town, err := f.RestoreOriginal(source)
	if err != nil || town {
		t.Fatal("prepare source", town, err)
	}
	if err := f.App("complete document").OpenMission(open); err != nil {
		t.Fatal(err)
	}
	clear(source)
	s, _, err := f.Snapshot(true)
	if err != nil {
		t.Fatal(err)
	}
	if s.SavedDocument == nil || s.SavedDocument.Document == nil || len(s.SavedDocument.Actors) != 1 {
		t.Fatal("full document not admitted")
	}
	return f, s
}

func TestDocument1115NativeOwnsWholeModelAndCurrentClock(t *testing.T) {
	f, first := documentSnapshot1115(t)
	baseline, err := sav.EncodeDocumentData(*first.SavedDocument.Document)
	if err != nil {
		t.Fatal(err)
	}
	for range 17 {
		sim.Step(f.live.world, nil)
	}
	current, _, err := f.Snapshot(true)
	if err != nil {
		t.Fatal(err)
	}
	clock, _ := f.live.world.SessionClock()
	if current.SavedDocument.Document.Head.CounterA != clock.SubTick || current.SavedDocument.Document.Head.CounterB != clock.FullTick || clock.SubTick != 9360 {
		t.Fatal("document replays imported clock", clock)
	}
	encoded, err := EncodeSave(current, "complete native document")
	if err != nil {
		t.Fatal(err)
	}
	loaded, _, err := DecodeSave(encoded)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(current.SavedDocument, loaded.SavedDocument) {
		t.Fatal("native graph changed")
	}
	for range 4 {
		again, err := EncodeSave(loaded, "complete native document")
		if err != nil || !bytes.Equal(encoded, again) {
			t.Fatal("native document encoding is not deterministic", err)
		}
	}
	loaded.SavedDocument.Document.Label = []byte("changed copy")
	loaded.SavedDocument.Document.World.Session.Raw08[0] ^= 0x55
	if reflect.DeepEqual(current.SavedDocument, loaded.SavedDocument) {
		t.Fatal("decoded document aliases original")
	}
	after, err := sav.EncodeDocumentData(*first.SavedDocument.Document)
	if err != nil || !bytes.Equal(baseline, after) {
		t.Fatal("Snapshot mutated earlier owned document", err)
	}
	fresh := poolFixtureFront(t, 91)
	fresh.Campaign = resolved(saveCampaign(), nil)
	open, town, err := fresh.Restore(current)
	if err != nil || town {
		t.Fatal("native prepare", town, err)
	}
	if err := fresh.App("fresh document").OpenMission(open); err != nil {
		t.Fatal(err)
	}
	resaved, _, err := fresh.Snapshot(true)
	if err != nil || !reflect.DeepEqual(current.SavedDocument, resaved.SavedDocument) {
		t.Fatal("fresh native LOAD lost document", err)
	}
	sim.Step(f.live.world, nil)
	sim.Step(fresh.live.world, nil)
	if f.live.world.Hash() != fresh.live.world.Hash() {
		t.Fatal("next native continuation differs")
	}
}

// Bypass EncodeSave's semantic checks, as an untrusted sender can do.
func uncheckedDocumentEnvelope1115(t *testing.T, s Snapshot) []byte {
	t.Helper()
	var payload bytes.Buffer
	if err := gob.NewEncoder(&payload).Encode(s); err != nil {
		t.Fatal(err)
	}
	b := append([]byte(saveMagic), saveVersion, 0, 0)
	b = binary.LittleEndian.AppendUint32(b, crc32.ChecksumIEEE(payload.Bytes()))
	b = binary.LittleEndian.AppendUint32(b, uint32(payload.Len()))
	return append(b, payload.Bytes()...)
}

func TestDocument1115MalformedNativeIsAtomic(t *testing.T) {
	f, source := documentSnapshot1115(t)
	for name, mutate := range map[string]func(*Snapshot){
		"version":          func(s *Snapshot) { s.SavedDocument.Version++ },
		"document version": func(s *Snapshot) { s.SavedDocument.Document.Version++ },
		"mission":          func(s *Snapshot) { s.SavedDocument.Document.Head.Mission++ },
		"difficulty":       func(s *Snapshot) { s.SavedDocument.Document.Head.Difficulty++ },
		"nonactor":         func(s *Snapshot) { s.SavedDocument.Actors[0].ObjectIndex = 1 },
		"out of range":     func(s *Snapshot) { s.SavedDocument.Actors[0].ObjectIndex = 65535 },
		"duplicate":        func(s *Snapshot) { s.SavedDocument.Actors = append(s.SavedDocument.Actors, s.SavedDocument.Actors[0]) },
		"inactive":         func(s *Snapshot) { s.SavedDocument.Unavailable = "forged" },
		"city":             func(s *Snapshot) { s.Mission = 0 },
	} {
		t.Run(name, func(t *testing.T) {
			s := source
			s.SavedDocument, _ = cloneSavedDocument(source.SavedDocument)
			mutate(&s)
			if _, err := EncodeSave(s, "bad"); err == nil {
				t.Fatal("encoded malformed document")
			}
			if _, _, err := DecodeSave(uncheckedDocumentEnvelope1115(t, s)); err == nil {
				t.Fatal("decoded malformed document")
			}
			before := f.live.world.Hash()
			if _, _, err := f.Restore(s); err == nil {
				t.Fatal("prepared malformed document")
			}
			if f.live.world.Hash() != before {
				t.Fatal("failed LOAD changed live world")
			}
		})
	}
	for _, mutate := range []func(*Snapshot){
		func(s *Snapshot) { s.SavedDocument.Document.Head.CounterA++ },
		func(s *Snapshot) { s.SavedDocument.Actors[0].EntityID += 1000 },
		func(s *Snapshot) { s.SavedDocument.Actors[0].Retired = true },
	} {
		s := source
		s.SavedDocument, _ = cloneSavedDocument(source.SavedDocument)
		mutate(&s)
		before, _, _ := f.Snapshot(true)
		if _, _, err := f.Restore(s); err == nil {
			t.Fatal("accepted inconsistent native/document pair")
		}
		after, _, err := f.Snapshot(true)
		if err != nil || !reflect.DeepEqual(before, after) {
			t.Fatal("late refusal changed active session", err)
		}
	}
}

func TestDocument1115PartialImportStaysExplicitAndNewMissionClearsIt(t *testing.T) {
	f := poolFixtureFront(t, 91)
	open, _, err := f.RestoreOriginal(clockFixture1112(9343, 584, &poolFixtureActor{mapID: 91, cell: 0x0605, hp: 10, maxHP: 100, profile: literalProfile1107()}))
	if err != nil {
		t.Fatal(err)
	}
	app := f.App("partial document")
	if err := app.OpenMission(open); err != nil {
		t.Fatal(err)
	}
	s, _, err := f.Snapshot(true)
	if err != nil || s.SavedDocument == nil || s.SavedDocument.Document != nil || s.SavedDocument.Unavailable == "" {
		t.Fatal("partial import gained full-document claim", err)
	}
	b, err := EncodeSave(s, "partial")
	if err != nil {
		t.Fatal(err)
	}
	got, _, err := DecodeSave(b)
	if err != nil || !reflect.DeepEqual(s.SavedDocument, got.SavedDocument) {
		t.Fatal("partial boundary vanished", err)
	}
	if err := app.OpenMission(f.MissionOpener(10)); err != nil {
		t.Fatal(err)
	}
	fresh, _, err := f.Snapshot(true)
	if err != nil {
		t.Fatal(err)
	}
	control := poolFixtureFront(t, 91)
	control.Town = NewTown(control.Campaign.Value())
	control.Town.gold = f.Town.Gold()
	if err := control.App("independent new mission").OpenMission(control.MissionOpener(10)); err != nil {
		t.Fatal(err)
	}
	want, _, err := control.Snapshot(true)
	if err != nil || reflect.DeepEqual(s.SavedDocument, fresh.SavedDocument) || !reflect.DeepEqual(want.SavedDocument, fresh.SavedDocument) || !bytes.Equal(want.World, fresh.World) {
		t.Fatal("new mission inherited old document", err)
	}
}

func TestDocument1115BothDoorsMenuSaveFreshProcess(t *testing.T) {
	if path := os.Getenv("AGAINROM_DOCUMENT_1115_NATIVE"); path != "" {
		f := currentPoolFixtureFront(t, 91)
		f.Campaign = resolved(saveCampaign(), nil)
		app := f.App("fresh complete document")
		store := SaveStore{Dir: filepath.Dir(path)}
		save, list, load := f.SaveSeams(store, OriginalStore{}, nil)
		app.SetSaveSeams(save, list, load)
		groundAppLoad(t, app, list, localOriginalSaveToken(filepath.Base(path)))
		if got := fmt.Sprintf("%x", f.live.world.Hash()); got != os.Getenv("AGAINROM_DOCUMENT_1115_WORLD") {
			t.Fatal("fresh world differs", got)
		}
		s, _, err := f.Snapshot(true)
		if err != nil || s.SavedDocument == nil || s.SavedDocument.Document == nil {
			t.Fatal("fresh complete document absent", err)
		}
		// The retained complete graph is the cold input, including unowned residue.
		// Snapshot may rebuild absent native Item identities for transport; its
		// projected keys are not native state and need not equal the input keys.
		retained := f.live.mission.state.savedDocument
		if retained == nil || retained.Document == nil {
			t.Fatal("cold input graph absent")
		}
		b, err := canonicalDocument1115(t, *retained.Document)
		if err != nil || fmt.Sprintf("%x", sha256.Sum256(b)) != os.Getenv("AGAINROM_DOCUMENT_1115_MODEL") {
			t.Fatal("cold input graph changed", err)
		}
		before, _ := f.live.world.SessionClock()
		f.live.tick()
		next, _, err := f.Snapshot(true)
		if err != nil || next.SavedDocument.Document.Head.CounterA != before.SubTick+1 {
			t.Fatal("next real driver tick did not update document", err)
		}
		if _, err := EncodeSave(next, "next native document"); err != nil {
			t.Fatal(err)
		}
		t.Log("complete document: menu SAVE, source removed, fresh App LOAD and next driver tick PASS")
		return
	}
	for _, onMap := range []bool{false, true} {
		t.Run(fmt.Sprintf("from-map-%t", onMap), func(t *testing.T) {
			f := currentPoolFixtureFront(t, 91)
			f.Campaign = resolved(saveCampaign(), nil)
			app := f.App("complete document source")
			if onMap {
				if err := app.OpenMission(f.MissionOpener(10)); err != nil {
					t.Fatal(err)
				}
			}
			originals, store := t.TempDir(), SaveStore{Dir: t.TempDir()}
			path := filepath.Join(originals, "game1115.sav")
			raw := completeDocumentFixture1115(t, f)
			if err := os.WriteFile(path, raw, 0600); err != nil {
				t.Fatal(err)
			}
			save, list, load := f.SaveSeams(store, OriginalStore{Dir: originals}, nil)
			app.SetSaveSeams(save, list, load)
			groundAppLoad(t, app, list, "game1115.sav")
			clear(raw)
			// Only this test-created synthetic input is removed. The child gets
			// a SAV path and cannot reopen an original source or its directory.
			if err := os.Remove(path); err != nil {
				t.Fatal(err)
			}
			for range 17 {
				f.live.tick()
			}
			s, _, err := f.Snapshot(true)
			if err != nil {
				t.Fatal(err)
			}
			doc, err := sav.EncodeDocumentData(*s.SavedDocument.Document)
			if err != nil {
				t.Fatal(err)
			}
			if err := app.HeadlessKey("escape"); err != nil {
				t.Fatal(err)
			}
			if err := app.HeadlessGameMenuAction("save"); err != nil {
				t.Fatal(err)
			}
			entries, err := store.List()
			if err != nil || len(entries) != 1 {
				t.Fatal("menu SAVE", entries, err, app.HeadlessMessage())
			}
			written, err := store.Read(entries[0].Name)
			if err != nil {
				t.Fatal(err)
			}
			ordinary, err := sav.DecodeDocumentData(written)
			if err != nil {
				t.Fatal(err)
			}
			doc, err = canonicalDocument1115(t, ordinary)
			if err != nil {
				t.Fatal(err)
			}
			exe, err := os.Executable()
			if err != nil {
				t.Fatal(err)
			}
			cmd := exec.Command(exe, "-test.run=^TestDocument1115BothDoorsMenuSaveFreshProcess$", "-test.v")
			cmd.Env = append(os.Environ(), "AGAINROM_DOCUMENT_1115_NATIVE="+filepath.Join(store.Dir, entries[0].Name), "AGAINROM_DOCUMENT_1115_WORLD="+fmt.Sprintf("%x", f.live.world.Hash()), "AGAINROM_DOCUMENT_1115_MODEL="+fmt.Sprintf("%x", sha256.Sum256(doc)))
			out, err := cmd.CombinedOutput()
			if err != nil || !bytes.Contains(out, []byte("next driver tick PASS")) {
				t.Fatalf("fresh child %v\n%s", err, out)
			}
			t.Log(string(out))
		})
	}
}

// LOAD may re-encode the typed continuation JSON. Compare its decoded fields
// canonically while retaining every ordinary field and unknown residue byte.
func canonicalDocument1115(t *testing.T, d sav.DocumentData) ([]byte, error) {
	t.Helper()
	a, err := readCurrentActions(&d)
	if err != nil {
		return nil, err
	}
	if a != nil {
		raw, err := json.Marshal(a)
		if err != nil {
			return nil, err
		}
		d.State.ValueRecords = append([]sav.CityStateRecordData(nil), d.State.ValueRecords...)
		d.State.DirectoryRecords = append([]sav.CityStateDirectoryData(nil), d.State.DirectoryRecords...)
		if err := sav.SetNativeActions(&d.State, raw); err != nil {
			return nil, err
		}
	}
	return sav.EncodeDocumentData(d)
}

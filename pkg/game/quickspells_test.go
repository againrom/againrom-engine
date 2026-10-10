package game

import (
	"bytes"
	"encoding/binary"
	"encoding/gob"
	"errors"
	"hash/crc32"
	"reflect"
	"testing"

	"againrom/pkg/data"
	"againrom/pkg/mapload"
	"againrom/pkg/ui"
)

// An independent envelope producer. Its schema deliberately does not use
// Snapshot, so old/malformed tests cannot pass by sharing the production writer.
func quickEnvelope1119(t *testing.T, value any) []byte {
	t.Helper()
	var payload bytes.Buffer
	if err := gob.NewEncoder(&payload).Encode(value); err != nil {
		t.Fatal(err)
	}
	b := []byte("AGRMSAVE\x01\x00\x00")
	b = binary.LittleEndian.AppendUint32(b, crc32.ChecksumIEEE(payload.Bytes()))
	b = binary.LittleEndian.AppendUint32(b, uint32(payload.Len()))
	return append(b, payload.Bytes()...)
}

func TestQuickSpellsNativeIndependentOldNewAndMalformed(t *testing.T) {
	want := [4]uint32{89, 17, 3, 65535}
	f := &FrontEnd{InstallResources: InstallResources{Campaign: resolved(saveCampaign(), nil)}, CampaignSession: CampaignSession{Town: saveTown(t), quickSpells: want}}
	snapshot, label, err := f.Snapshot(false)
	if err != nil {
		t.Fatal(err)
	}
	b, err := EncodeSave(snapshot, label)
	if err != nil {
		t.Fatal(err)
	}
	// Read the public envelope and the named field without DecodeSave.
	labelLen := int(binary.LittleEndian.Uint16(b[9:11]))
	start := 11 + labelLen
	if string(b[:8]) != "AGRMSAVE" || b[8] != 1 || binary.LittleEndian.Uint32(b[start:]) != crc32.ChecksumIEEE(b[start+8:]) {
		t.Fatal("native header/checksum changed")
	}
	var independent struct{ QuickSpells [4]uint32 }
	if err := gob.NewDecoder(bytes.NewReader(b[start+8:])).Decode(&independent); err != nil || independent.QuickSpells != want {
		t.Fatalf("independent field=%v err=%v", independent.QuickSpells, err)
	}
	back, _, err := DecodeSave(b)
	if err != nil {
		t.Fatal(err)
	}
	g := &FrontEnd{InstallResources: InstallResources{Campaign: resolved(saveCampaign(), nil)}}
	if _, town, err := g.Restore(back); err != nil || !town || g.quickSpells != want {
		t.Fatalf("fresh native restore=%v town%v err%v", g.quickSpells, town, err)
	}
	old := quickEnvelope1119(t, struct {
		Open bool
		Gold int
	}{true, 12})
	back, _, err = DecodeSave(old)
	if err != nil || back.QuickSpells != [4]uint32{} {
		t.Fatalf("old schema=%v err=%v", back.QuickSpells, err)
	}
	if _, _, err := g.Restore(back); err != nil || g.quickSpells != [4]uint32{} {
		t.Fatal("old save retained previous bindings")
	}
	for _, bad := range [][4]uint32{{65536}, {17, 17}} {
		t.Run("malformed", func(t *testing.T) {
			g.quickSpells = want
			before := *g
			if _, err := EncodeSave(Snapshot{QuickSpells: bad}, ""); err == nil {
				t.Fatal("writer accepted malformed slots")
			}
			raw := quickEnvelope1119(t, struct {
				Open        bool
				QuickSpells [4]uint32
			}{true, bad})
			if _, _, err := DecodeSave(raw); err == nil {
				t.Fatal("decoder accepted malformed slots")
			}
			if _, err := g.prepareRestore(Snapshot{QuickSpells: bad}); err == nil {
				t.Fatal("direct restore accepted malformed slots")
			}
			if !reflect.DeepEqual(before, *g) {
				t.Fatal("failed restore changed active session")
			}
		})
	}
	for _, value := range []any{struct{ QuickSpells [3]uint32 }{}, struct{ QuickSpells [5]uint32 }{}} {
		if _, _, err := DecodeSave(quickEnvelope1119(t, value)); err == nil {
			t.Fatal("non-four native slot shape accepted")
		}
	}
}

func TestQuickSpellsCandidateAndMissionLifetime(t *testing.T) {
	f := missionFrontEnd(t)
	f.Town = NewTown(Campaign{})
	want := [4]uint32{17, 3, 6, 89}
	f.quickSpells = want
	open := f.MissionOpener(10)
	_, _, _, _, _, _, _, _, _, _, err := open()
	if err != nil {
		t.Fatal(err)
	}
	if f.quickSpells != want {
		t.Fatal("mission entry lost bindings")
	}
	f.quickSpells[0] = 91
	f.arriveInTown()
	if f.quickSpells[0] != 91 {
		t.Fatal("town arrival lost bindings")
	}
	before := *f
	if _, err := f.prepareRestore(Snapshot{Mission: 20, QuickSpells: want}); err == nil {
		t.Fatal("missing world accepted")
	}
	if !reflect.DeepEqual(before, *f) {
		t.Fatal("failed mission restore changed bindings/session")
	}
	f.installCandidate(&restoreCandidate{townOnly: true, town: NewTown(Campaign{}), quickSpells: want})
	if f.quickSpells != want {
		t.Fatal("candidate did not own bindings")
	}
	f.resetSessionForNewGame()
	if f.quickSpells != [4]uint32{} {
		t.Fatal("new campaign did not reset bindings")
	}
}

func TestQuickSpellsNewGameCommitsResetOnlyAfterPreparation(t *testing.T) {
	f := missionFrontEnd(t)
	f.Table, f.Humans = &mapload.Table{}, fourBaseHumans()
	f.Bodies, f.Town = data.NewBodyList("unarmed", "mage"), NewTown(Campaign{})
	want := [4]uint32{16, 1, 6, 19}
	f.quickSpells = want
	res := ui.ChargenResult{Choices: []int{1, 1, 2}, Stats: []int{30, 20, 18, 16}}
	if _, err := f.prepareNewGame(20, res); err == nil {
		t.Fatal("missing map should fail preparation")
	}
	if f.quickSpells != want {
		t.Fatal("failed NEW GAME cleared the running campaign")
	}
	open, err := f.prepareNewGame(10, res)
	if err != nil {
		t.Fatal(err)
	}
	if f.quickSpells != want {
		t.Fatal("successful draft changed unadopted session")
	}
	v, _, _, _, _, _, _, _, _, _, err := open()
	if err != nil {
		t.Fatal(err)
	}
	if slots, current, armed := v.QuickSpellState(); f.quickSpells != [4]uint32{} || slots != [4]uint32{} || current != 0 || armed {
		t.Fatal("NEW GAME commit did not reset only the adopted session")
	}
}

func TestQuickSpellsCustomTownUsesCurrentSAV(t *testing.T) {
	f, document := originalCityRouteFixture(t)
	document.err = errors.New("retained writer must not select SAVE")
	f.quickSpells = [4]uint32{17, 89}
	want := f.quickSpells
	store := SaveStore{Dir: t.TempDir()}
	save, list, load := f.SaveSeams(store, OriginalStore{}, nil)
	name, err := save(false)
	if err != nil || !IsOriginal(name) {
		t.Fatal("custom quick-spell SAVE", name, err)
	}
	rows := list()
	if len(rows) != 1 || rows[0].Name != localOriginalSaveToken(name) {
		t.Fatal("custom quick-spell list", rows)
	}
	f.quickSpells = [4]uint32{}
	if _, town, err := load(rows[0].Name); err != nil || !town || f.quickSpells != want {
		t.Fatal("custom quick-spell LOAD", f.quickSpells, town, err)
	}
	if len(document.updates) != 0 {
		t.Fatal("custom quick spells selected retained writer")
	}
}

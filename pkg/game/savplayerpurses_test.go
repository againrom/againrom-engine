package game

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"againrom/internal/synth"
	"againrom/pkg/formats/sav"
	"againrom/pkg/render/terrain"
	"againrom/pkg/ui"
)

// Two literal ALM GiveMoney actions execute on every true trigger pass. Player
// 7 deliberately differs from its second distinct root/native identity 2.
func purseMoneyALM1115() []byte {
	b := make([]byte, 12+3*796+184)
	put := func(off int, value uint32) { binary.LittleEndian.PutUint32(b[off:], value) }
	put(0, 2)
	for i, row := range [][2]uint32{{1, 5}, {7, 17}} {
		a := 4 + i*796
		put(a+0x40, 23)
		put(a+0x44, uint32(10+i*10))
		put(a+0x4c, row[0])
		put(a+0x50, row[1])
		put(a+0x74, 3) // Player
		put(a+0x78, 1) // signed amount
	}
	c := 4 + 2*796
	put(c, 1)
	c += 4
	put(c+0x40, 0x10002)
	put(c+0x44, 1)
	put(c+0x74, 7) // constant zero
	tr := 8 + 3*796
	put(tr, 1)
	tr += 4
	put(tr+0x80, 1)
	put(tr+0x84, 2)
	put(tr+0x98, 10)
	put(tr+0x9c, 20)
	// No once bit: the next post-LOAD tick must execute both producers again.
	return b
}

func purseFront1115(t *testing.T) *FrontEnd {
	t.Helper()
	b := synth.ALM(synth.ALMOptions{Width: 40, Height: 40,
		Units: []synth.ALMUnit{{X: 5<<8 | 128, Y: 6<<8 | 128}}, Type7Payload: purseMoneyALM1115()})
	binary.LittleEndian.PutUint16(rawSections1092(t, b)[6][0x40:], 91)
	f := poolFixtureFrontMap(t, b)
	f.Table, f.Campaign = actorRegistryTable(), resolved(saveCampaign(), nil)
	f.Units = &terrain.UnitSet{Classes: map[int32]*terrain.UnitClass{}}
	for _, class := range []int32{3, 33, 35} {
		f.Units.Classes[class] = worldFixtureArt(16, 16, 8, 16, 3, 3, 1)
	}
	return f
}

func purseLiteral1115(t *testing.T, f *FrontEnd, slots, money [2]uint32) []byte {
	t.Helper()
	doc, err := sav.DecodeDocumentData(newGroupLiteral(t, f, false))
	if err != nil {
		t.Fatal(err)
	}
	for i, object := range []uint16{1, 4} {
		p := &doc.Objects[object-1]
		newGroupSetValue1115(t, p, "Slot", slots[i])
		newGroupSetValue1115(t, p, "SlotAgain", slots[i])
		newGroupSetValue1115(t, p, "Participant", uint32(i))
		newGroupSetValue1115(t, p, "Money", money[i])
	}
	raw, err := sav.EncodeDocumentData(doc)
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

func purseDocumentMoney1115(t *testing.T, state *SnapshotSAVDocument) [2]uint32 {
	t.Helper()
	var result [2]uint32
	var seen [2]bool
	for _, record := range state.Document.Objects {
		if record.Class != "Player" {
			continue
		}
		key := actorProjectionValue(t, record, "This")
		for i, want := range []uint32{newGroupLeft1115, newGroupRight1115} {
			if key == want {
				if seen[i] {
					t.Fatal("duplicate Player identity in independent expectation")
				}
				seen[i] = true
				result[i] = actorProjectionValue(t, record, "Money")
			}
		}
	}
	if seen != [2]bool{true, true} {
		t.Fatal("missing exact Player identities")
	}
	return result
}

func purseCheck1115(t *testing.T, f *FrontEnd, want [2]uint32) Snapshot {
	t.Helper()
	if got := [2]uint32{f.live.world.Purse(1), f.live.world.Purse(7)}; got != want {
		t.Fatalf("actual live purses=%#x want %#x", got, want)
	}
	s, _, err := f.Snapshot(true)
	if err != nil {
		t.Fatal(err)
	}
	wantBindings := []SnapshotSAVPlayerPurse{{PlayerID: 1, Slot: 1}, {PlayerID: 2, Slot: 7}}
	if s.SavedDocument.PlayerPurses == nil || s.SavedDocument.PlayerPurses.Version != 1 || !reflect.DeepEqual(s.SavedDocument.PlayerPurses.Players, wantBindings) {
		t.Fatal("current exact Player purse owner absent", s.SavedDocument.PlayerPurses)
	}
	if got := purseDocumentMoney1115(t, s.SavedDocument); got != want {
		t.Fatalf("current document Money=%#x want %#x", got, want)
	}
	return s
}

func TestSavedPlayerPurses1115OriginalDoorsNativeSaveAndNextGoldAction(t *testing.T) {
	for _, fromMap := range []bool{false, true} {
		for _, initial := range [][2]uint32{{0, 0}, {0x80000001, 0xffffffff}} {
			t.Run(fmt.Sprintf("fromMap=%t/money=%x", fromMap, initial), func(t *testing.T) {
				f := purseFront1115(t)
				raw := purseLiteral1115(t, f, [2]uint32{1, 7}, initial)
				ms, _, err := ResumeOriginalSave(f.Archives.Containers, raw, f.Table, f.Difficulty, nil, nil)
				if err != nil || ms.World.Purse(1) != initial[0] || ms.World.Purse(7) != initial[1] {
					t.Fatal("low-level original LOAD lost full unsigned Player purses", err)
				}
				app := f.App("exact Player purse continuation")
				if fromMap {
					if err := app.OpenMission(f.MissionOpener(10)); err != nil {
						t.Fatal(err)
					}
				}
				inputs, store := t.TempDir(), SaveStore{Dir: t.TempDir()}
				path := filepath.Join(inputs, "game1115.sav")
				if err := os.WriteFile(path, raw, 0600); err != nil {
					t.Fatal(err)
				}
				save, list, load := agsSaveSeams(f, store, OriginalStore{Dir: inputs}, nil)
				app.SetSaveSeams(save, list, load)
				groundAppLoad(t, app, list, "game1115.sav")
				clear(raw)
				if err := os.Remove(path); err != nil { // only this synthetic input
					t.Fatal(err)
				}
				purseCheck1115(t, f, initial)
				for range 16 { // one real scheduler cycle, no SetPurse/test grant
					f.live.tick()
				}
				want := [2]uint32{initial[0] + 5, initial[1] + 17}
				before := purseCheck1115(t, f, want)
				name := crossingMenuSave(t, app, store)
				fresh := purseFront1115(t)
				freshApp := fresh.App("fresh native exact Player purses")
				save, list, load = agsSaveSeams(fresh, store, OriginalStore{}, nil)
				freshApp.SetSaveSeams(save, list, load)
				groundAppLoad(t, freshApp, list, name)
				after := purseCheck1115(t, fresh, want)
				if !bytes.Equal(before.World, after.World) || f.live.world.Hash() != fresh.live.world.Hash() {
					t.Fatal("ordinary native SAVE/fresh LOAD changed current Player purses")
				}
				for range 16 {
					f.live.tick()
					fresh.live.tick()
				}
				want[0], want[1] = want[0]+5, want[1]+17
				purseCheck1115(t, f, want)
				purseCheck1115(t, fresh, want)
				if f.live.world.Hash() != fresh.live.world.Hash() {
					t.Fatal("next gold-producing tick differs after native LOAD")
				}
			})
		}
	}
}

func TestSavedPlayerPurses1115AbsentOwnerDoesNotReconstructOrProject(t *testing.T) {
	f := purseFront1115(t)
	open, _, err := f.RestoreOriginal(purseLiteral1115(t, f, [2]uint32{1, 7}, [2]uint32{19, 23}))
	if err != nil {
		t.Fatal(err)
	}
	if err := f.App("legacy purse owner absence").OpenMission(open); err != nil {
		t.Fatal(err)
	}
	s := purseCheck1115(t, f, [2]uint32{19, 23})
	s.SavedDocument.PlayerPurses = nil // compatibility construction, not old fixture bytes
	newGroupSetValue1115(t, &s.SavedDocument.Document.Objects[0], "Money", 456)
	raw, err := EncodeSave(s, "explicit old purse absence")
	if err != nil {
		t.Fatal(err)
	}
	decoded, _, err := DecodeSave(raw)
	if err != nil || decoded.SavedDocument.PlayerPurses != nil {
		t.Fatal("native decode invented purse owner", err)
	}
	fresh := purseFront1115(t)
	open, _, err = fresh.Restore(decoded)
	if err != nil {
		t.Fatal(err)
	}
	if err := fresh.App("legacy native purse LOAD").OpenMission(open); err != nil {
		t.Fatal(err)
	}
	for range 16 {
		fresh.live.tick()
	}
	after, _, err := fresh.Snapshot(true)
	if err != nil || after.SavedDocument.PlayerPurses != nil || purseDocumentMoney1115(t, after.SavedDocument) != ([2]uint32{456, 23}) || fresh.live.world.Purse(1) != 24 || fresh.live.world.Purse(7) != 40 {
		t.Fatal("old native purse absence repaired from document or newly projected", err)
	}
}

func TestSavedPlayerPurses1115AmbiguousSlotsUncoveredAndUnsupportedOriginalRefused(t *testing.T) {
	for _, slots := range [][2]uint32{{1, 1}, {1, 0}, {1, 17}, {1, 49}, {1, 50}, {1, 32767}, {1, 32768}, {1, 65535}} {
		t.Run(fmt.Sprint(slots), func(t *testing.T) {
			f := purseFront1115(t)
			open, _, err := f.RestoreOriginal(purseLiteral1115(t, f, slots, [2]uint32{19, 0xfffffffe}))
			if slots[1] != 1 {
				// The existing original Player reader already refuses these
				// slots. Do not weaken that admission to exercise our owner.
				if err == nil || open != nil || f.live != nil {
					t.Fatal("unsupported source Player slot was adopted", err)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if err := f.App("uncovered Player purse").OpenMission(open); err != nil {
				t.Fatal(err)
			}
			s, _, err := f.Snapshot(true)
			if err != nil || s.SavedDocument.PlayerPurses == nil {
				t.Fatal(err)
			}
			rows := s.SavedDocument.PlayerPurses.Players
			if slots[1] == 1 {
				if rows[0].Unavailable != savedPurseSlotAmbiguous || rows[1].Unavailable != savedPurseSlotAmbiguous {
					t.Fatal("same-slot Players were aliased", rows)
				}
			} else if rows[0].Unavailable != "" || rows[1].Unavailable != savedPurseSlotUnavailable {
				t.Fatal("out-of-domain Player purse was narrowed/admitted", rows)
			}
			for range 16 {
				f.live.tick()
			}
			after, _, err := f.Snapshot(true)
			want := [2]uint32{19, 0xfffffffe}
			if slots[1] != 1 {
				want[0] += 5
			}
			if err != nil || purseDocumentMoney1115(t, after.SavedDocument) != want {
				t.Fatal("uncovered Player's Money was overwritten by a different purse", err)
			}
		})
	}
}

func TestSavedPlayerPurses1115SourceAndNativeDomainBoundaries(t *testing.T) {
	for _, slot := range []uint32{1, 16} {
		f := purseFront1115(t)
		raw := purseLiteral1115(t, f, [2]uint32{slot, 7}, [2]uint32{0xffffffff, 0})
		open, _, err := f.RestoreOriginal(raw)
		if err != nil {
			t.Fatal(err)
		}
		if err := f.App("source Player slot boundary").OpenMission(open); err != nil {
			t.Fatal(err)
		}
		if f.live.world.Purse(slot) != 0xffffffff || f.live.world.Purse(7) != 0 {
			t.Fatal("source boundary purse lost unsigned value", slot)
		}
	}
	f := purseFront1115(t)
	doc, err := sav.DecodeDocumentData(purseLiteral1115(t, f, [2]uint32{1, 7}, [2]uint32{19, 23}))
	if err != nil {
		t.Fatal(err)
	}
	groups := &SnapshotSAVGroupBindings{Version: 1, PlayersPresent: true,
		Players: []SnapshotSAVGroupPlayerBinding{{ID: 71, ObjectIndex: 1}, {ID: 99, ObjectIndex: 4}}}
	for _, slot := range []uint32{0, 1, 16, 17, 49, 50, 32767, 32768, 65535, 65536, 0xffffffff} {
		newGroupSetValue1115(t, &doc.Objects[0], "Slot", slot)
		rows, err := savedPlayerPurseRows(&doc, groups)
		if slot > 65535 {
			if err == nil || rows != nil {
				t.Fatal("non-word Slot was narrowed", slot)
			}
			continue
		}
		if err != nil || len(rows) != 2 || rows[0].PlayerID != 71 || rows[0].Slot != slot {
			t.Fatal("native coverage lost exact identity/word", slot, err)
		}
		want := ""
		if slot >= 50 {
			want = savedPurseSlotUnavailable
		}
		if rows[0].Unavailable != want {
			t.Fatal("native purse boundary incorrectly covered", slot, rows)
		}
	}
}

func TestSavedPlayerPurses1115MoneyMismatchRefusedBeforeNativeAdoption(t *testing.T) {
	f := purseFront1115(t)
	open, _, err := f.RestoreOriginal(purseLiteral1115(t, f, [2]uint32{1, 7}, [2]uint32{19, 23}))
	if err != nil {
		t.Fatal(err)
	}
	app := f.App("purse mismatch active session")
	if err := app.OpenMission(open); err != nil {
		t.Fatal(err)
	}
	before := purseCheck1115(t, f, [2]uint32{19, 23})
	bad := groupPlayersValidationClone1115(t, before)
	newGroupSetValue1115(t, &bad.SavedDocument.Document.Objects[3], "Money", 0x80000000)
	if raw, err := EncodeSave(bad, "mismatched Money"); err == nil || raw != nil {
		t.Fatal("EncodeSave accepted contradictory Player Money", err)
	}
	raw := uncheckedDocumentEnvelope1115(t, bad)
	label := []byte("contradictory Player Money")
	binary.LittleEndian.PutUint16(raw[9:11], uint16(len(label)))
	raw = append(append(bytes.Clone(raw[:11]), label...), raw[11:]...)
	if out, label, err := DecodeSave(raw); err == nil || label != "" || !reflect.DeepEqual(out, Snapshot{}) {
		t.Fatal("checksum-valid Money disagreement survived native decode", err)
	}
	groupPlayersValidationRestoreReject1115(t, f, bad)
	store := SaveStore{Dir: t.TempDir()}
	if err := os.WriteFile(filepath.Join(store.Dir, "hostile.ags"), raw, 0600); err != nil {
		t.Fatal(err)
	}
	save, list, load := agsSaveSeams(f, store, OriginalStore{}, nil)
	app.SetSaveSeams(save, list, load)
	if err := headlessOpenLoad(app); err != nil {
		t.Fatal(err)
	}
	if err := app.HeadlessActivate(list()[0].Label); err != nil {
		t.Fatal(err)
	}
	if app.Screen() != ui.ScreenLoad {
		t.Fatal("ordinary LOAD adopted contradictory Player Money")
	}
	after, _, err := f.Snapshot(true)
	if err != nil || !reflect.DeepEqual(before, after) {
		t.Fatal("refused ordinary native LOAD changed live session", err)
	}
}

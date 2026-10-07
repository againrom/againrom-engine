package game

import (
	"bytes"
	"encoding/binary"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

// These controls use the independent complete archive fixture, not the new
// purse importer's row builder, to reach the real native persistence doors.
func TestPlayerPurses1115HostileEnvelopeKeepsActiveSession(t *testing.T) {
	for _, tc := range []struct {
		name string
		edit func(*testing.T, *Snapshot)
	}{
		{"zero version", func(_ *testing.T, s *Snapshot) { s.SavedDocument.PlayerPurses.Version = 0 }},
		{"future version", func(_ *testing.T, s *Snapshot) { s.SavedDocument.PlayerPurses.Version = 2 }},
		{"missing row", func(_ *testing.T, s *Snapshot) { s.SavedDocument.PlayerPurses.Players = nil }},
		{"duplicate row", func(_ *testing.T, s *Snapshot) {
			p := s.SavedDocument.PlayerPurses
			p.Players = append(p.Players, p.Players[0])
		}},
		{"zero identity", func(_ *testing.T, s *Snapshot) { s.SavedDocument.PlayerPurses.Players[0].PlayerID = 0 }},
		{"other identity", func(_ *testing.T, s *Snapshot) { s.SavedDocument.PlayerPurses.Players[0].PlayerID++ }},
		{"other slot", func(_ *testing.T, s *Snapshot) { s.SavedDocument.PlayerPurses.Players[0].Slot++ }},
		{"invented unavailable", func(_ *testing.T, s *Snapshot) {
			s.SavedDocument.PlayerPurses.Players[0].Unavailable = "unsupported producer"
		}},
		{"forged domain exemption", func(_ *testing.T, s *Snapshot) {
			s.SavedDocument.PlayerPurses.Players[0].Unavailable = savedPurseSlotUnavailable
		}},
		{"forged alias exemption", func(_ *testing.T, s *Snapshot) {
			s.SavedDocument.PlayerPurses.Players[0].Unavailable = savedPurseSlotAmbiguous
		}},
		{"nul exemption", func(_ *testing.T, s *Snapshot) {
			s.SavedDocument.PlayerPurses.Players[0].Unavailable = "\x00"
		}},
		{"document low money bit", func(t *testing.T, s *Snapshot) { flipPurseMoney1115(t, s, 1) }},
		{"document signed money bit", func(t *testing.T, s *Snapshot) { flipPurseMoney1115(t, s, 0x80000000) }},
		{"native-only money change", func(t *testing.T, s *Snapshot) {
			var w sim.World
			if err := w.UnmarshalBinary(s.World); err != nil {
				t.Fatal(err)
			}
			slot := s.SavedDocument.PlayerPurses.Players[0].Slot
			if !w.SetPurse(slot, w.Purse(slot)^0x80000001) {
				t.Fatal("independent native purse mutation failed")
			}
			var err error
			s.World, err = w.MarshalBinary()
			if err != nil {
				t.Fatal(err)
			}
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			front, before := documentSnapshot1115(t)
			if before.SavedDocument.PlayerPurses == nil || len(before.SavedDocument.PlayerPurses.Players) != 2 || before.SavedDocument.PlayerPurses.Players[0].Unavailable != "" {
				t.Fatal("literal fixture lacks its two distinct Player purses")
			}
			bad := before
			var err error
			bad.SavedDocument, err = cloneSavedDocument(before.SavedDocument)
			if err != nil {
				t.Fatal(err)
			}
			tc.edit(t, &bad)
			envelope := uncheckedDocumentEnvelope1115(t, bad)
			label := []byte("Hostile purse state")
			binary.LittleEndian.PutUint16(envelope[9:11], uint16(len(label)))
			header := append(bytes.Clone(envelope[:11]), label...)
			envelope = append(header, envelope[11:]...)
			if output, _, err := DecodeSave(envelope); err == nil || !reflect.DeepEqual(output, Snapshot{}) {
				t.Fatal("checksum-valid hostile purse envelope was accepted", err)
			}
			oldLive, oldTown, oldShop, oldHash := front.live, front.Town, front.Shop, front.live.world.Hash()
			unchanged := func() {
				t.Helper()
				after, _, err := front.Snapshot(true)
				if err != nil || front.live != oldLive || front.Town != oldTown || front.Shop != oldShop || front.live.world.Hash() != oldHash || !reflect.DeepEqual(before, after) {
					t.Fatal("purse refusal changed active session, hash or Snapshot", err)
				}
			}
			if open, town, err := front.Restore(bad); err == nil || open != nil || town {
				t.Fatal("direct Restore adopted conflicting purse state", open != nil, town, err)
			}
			unchanged()
			store := SaveStore{Dir: t.TempDir()}
			if err := os.WriteFile(filepath.Join(store.Dir, "hostile-purse.ags"), envelope, 0600); err != nil {
				t.Fatal(err)
			}
			app := front.App("purse rejection")
			save, list, load := agsSaveSeams(front, store, OriginalStore{}, nil)
			app.SetSaveSeams(save, list, load)
			rows := list()
			if len(rows) != 1 {
				t.Fatal("hostile envelope has no LOAD row", rows)
			}
			if err := headlessOpenLoad(app); err != nil {
				t.Fatal(err)
			}
			if err := app.HeadlessActivate(rows[0].Label); err != nil {
				t.Fatal(err)
			}
			if app.Screen() != ui.ScreenLoad || app.HeadlessMessage() == "" {
				t.Fatal("ordinary LOAD did not expose the purse refusal")
			}
			unchanged()
		})
	}
}

func flipPurseMoney1115(t *testing.T, snapshot *Snapshot, mask uint32) {
	t.Helper()
	object := snapshot.SavedDocument.GroupBindings.Players[0].ObjectIndex
	for i := range snapshot.SavedDocument.Document.Objects[object-1].Values {
		v := &snapshot.SavedDocument.Document.Objects[object-1].Values[i]
		if v.Name == "Money" {
			v.Value ^= mask
			return
		}
	}
	t.Fatal("independent fixture has no Player Money")
}

// Failure after staged SetPurse must not leak a partially installed purse.
// The unrelated motion contradiction deliberately fails final validation.
func TestPlayerPurses1115LateImportFailureIsAtomic(t *testing.T) {
	front, before := documentSnapshot1115(t)
	bad := before
	var err error
	bad.SavedDocument, err = cloneSavedDocument(before.SavedDocument)
	if err != nil {
		t.Fatal(err)
	}
	flipPurseMoney1115(t, &bad, 0x80000001)
	actor := &bad.SavedDocument.Document.Objects[bad.SavedDocument.Actors[0].ObjectIndex-1]
	found := false
	for i := range actor.Raw {
		if actor.Raw[i].Name == "Block12" {
			actor.Raw[i].Bytes[4] ^= 1
			found = true
		}
	}
	if !found {
		t.Fatal("fixture has no actor Position")
	}
	mission := Mission{World: front.live.world, savedDocument: before.SavedDocument}
	worldPointer, documentPointer := mission.World, mission.savedDocument
	oldHash := mission.World.Hash()
	if err := importSavedPlayerPurses(&mission, bad.SavedDocument); err == nil || !strings.Contains(err.Error(), "Position") {
		t.Fatal("fixture did not reach final motion validation after staging money", err)
	}
	if mission.World != worldPointer || mission.savedDocument != documentPointer || mission.World.Hash() != oldHash {
		t.Fatal("late failed import published a purse or replaced an owner")
	}
	after, _, err := front.Snapshot(true)
	if err != nil || !reflect.DeepEqual(before, after) {
		t.Fatal("late failed import changed the active full Snapshot", err)
	}
}

func TestPlayerPurses1115ExactOwnersSurviveGameplayGraphReindex(t *testing.T) {
	f := newGroupFront(t, 2)
	want := [2]uint32{19, 0xfffffffe}
	open, town, err := f.RestoreOriginal(purseLiteral1115(t, f, [2]uint32{1, 2}, want))
	if err != nil || town {
		t.Fatal("literal two-Player LOAD", town, err)
	}
	app := f.App("Player purses across GiveUnit")
	if err := app.OpenMission(open); err != nil {
		t.Fatal(err)
	}
	before := groupDocumentSnapshot1115(t, f)
	for range 20 {
		f.live.tick()
	}
	after := groupDocumentSnapshot1115(t, f)
	if !f.live.world.ScriptLatched(0) || newGroupActors1115(t, f.live.world)[newGroupA].Owner != 2 {
		t.Fatal("ordinary GiveUnit did not execute")
	}
	if reflect.DeepEqual(before.SavedDocument.GroupBindings.Players, after.SavedDocument.GroupBindings.Players) {
		t.Fatal("fixture did not move a Player document index")
	}
	check := func(f *FrontEnd, s Snapshot) {
		t.Helper()
		if s.SavedDocument.PlayerPurses == nil || !reflect.DeepEqual(before.SavedDocument.PlayerPurses, s.SavedDocument.PlayerPurses) ||
			purseDocumentMoney1115(t, s.SavedDocument) != want || [2]uint32{f.live.world.Purse(1), f.live.world.Purse(2)} != want {
			t.Fatal("graph reindex changed a Player's exact purse ownership")
		}
	}
	check(f, after)
	store := SaveStore{Dir: t.TempDir()}
	save, list, load := agsSaveSeams(f, store, OriginalStore{}, nil)
	app.SetSaveSeams(save, list, load)
	name := crossingMenuSave(t, app, store)
	fresh := newGroupFront(t, 2)
	freshApp := fresh.App("fresh purses after GiveUnit")
	save, list, load = agsSaveSeams(fresh, store, OriginalStore{}, nil)
	freshApp.SetSaveSeams(save, list, load)
	groundAppLoad(t, freshApp, list, name)
	check(fresh, groupDocumentSnapshot1115(t, fresh))
	for range 20 {
		f.live.tick()
		fresh.live.tick()
		if f.live.world.Hash() != fresh.live.world.Hash() {
			t.Fatal("next action differs after reindexed Player purse LOAD")
		}
	}
}

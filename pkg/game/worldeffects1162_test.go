package game

import (
	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
	"bytes"
	"encoding/binary"
	"fmt"
	"os"
	"reflect"
	"strings"
	"testing"
)

// Synthetic complete SAV body: independent literal writer, no installed bytes.
func ring1162Source(t *testing.T, spell, mode, direction byte) []byte {
	f := spell1152FixtureFront(t)
	source, err := sav.Open(completeDocumentFixture1115(t, f))
	if err != nil {
		t.Fatal(err)
	}
	loc, _, err := source.SpellEffectArchiveLocation()
	if err != nil {
		t.Fatal(err)
	}
	s := &spell1152Literal{next: uint16(loc.NextIndex), classes: map[string]uint16{}}
	for id, class := range loc.Classes {
		s.classes[class] = id
	}
	token := func(id uint32) {
		s.b = append(s.b, 10, 10, 10, 10, 128, 128, 0, 0, 0, 0, 0, 0)
		s.u32(0)
		s.b = append(s.b, spell)
		s.u16(0)
		s.u32(uint32(mode))
		s.u16(0)
		s.u32(0)
		s.u32(id)
		s.u32(0)
	}
	s.u32(1)
	s.object("AreaEffect")
	token(0xa1234567)
	s.b = append(s.b, 0, 1, 211, 2, direction, 0)
	s.u16(0)
	s.object("Effect_DirectDamage")
	token(0xb1234567)
	s.b = append(s.b, 6, 0)
	s.u32(17)
	s.b = append(s.b, spell)
	s.b = append(s.b, make([]byte, 24)...)
	if source.World.BlocksOff != loc.Off+4 {
		t.Fatal("fixture no longer empty")
	}
	end := source.World.SessionOff + 4374 + 4 + 8 + 400
	body := append([]byte(nil), source.Body[:loc.Off]...)
	body = append(body, s.b...)
	body = append(body, source.Body[source.World.BlocksOff:end]...)
	source.Body = body
	return source.Marshal()
}

func TestWorldEffects1162StagedAppChangedDocumentAndNative(t *testing.T) {
	for _, tc := range []struct {
		spell, mode, direction byte
		limit                  int
	}{{4, 2, 0, 2}, {9, 2, 64, 6}, {9, 2, 96, 6}, {21, 2, 0, 32}, {7, 0, 0, 1}} {
		t.Run(fmt.Sprintf("%d/%d/%d", tc.spell, tc.mode, tc.direction), func(t *testing.T) {
			raw := ring1162Source(t, tc.spell, tc.mode, tc.direction)
			f := spell1152FixtureFront(t)
			app, store, path := openWorldEffectsTestSave(t, f, raw, "staged.sav")
			check := func(front *FrontEnd, tick int) {
				t.Helper()
				snapshot, _, err := front.Snapshot(true)
				if err != nil {
					t.Fatal("Snapshot", tick, err)
				}
				terminal := 1 + 3*(tc.limit-1)
				if tc.mode == 0 {
					terminal = 1
				}
				if tick >= terminal {
					if len(front.live.world.SavedSpellEffects()) != 0 || front.live.world.SavedWorldEffectDrivers() != nil || len(snapshot.SavedDocument.Document.World.Effects) != 0 || len(snapshot.SavedDocument.WorldEffects.Areas) != 0 {
						t.Fatal("terminal world/Document roots or exclusive child retained", tick)
					}
					return
				}
				stage := byte(0)
				timer := uint16(0)
				if tick > 0 {
					stage = byte((tick-1)/3 + 1)
					timer = uint16(2 - (tick-1)%3)
				}
				effect := front.live.world.SavedSpellEffects()[0]
				if effect.AE48 != [4]byte{211, 2, tc.direction, stage} || effect.AE4C != timer {
					t.Fatal("independent stage/timer", tick, effect.AE48, effect.AE4C)
				}
				binding := snapshot.SavedDocument.WorldEffects.Areas[0]
				root := &snapshot.SavedDocument.Document.Objects[binding.ObjectIndex-1]
				block, err := savedMotionRaw(root, "AE48", 4)
				counter, _ := savedStructureValue(root, "AE4C")
				if err != nil || block[0] != 211 || block[3] != stage || counter != uint32(timer) {
					t.Fatal("changed Document stage/counter", tick)
				}
				if snapshot.SavedDocument.WorldEffects.Unavailable != "" {
					t.Fatal("bounded direct-damage payload refused", snapshot.SavedDocument.WorldEffects.Unavailable)
				}
			}
			check(f, 0)
			f.live.tick()
			check(f, 1)
			if err := os.Remove(path); err != nil {
				t.Fatal(err)
			}
			fresh := spell1152MenuFresh(t, f, app, store, spell1152FixtureFront)
			check(fresh, 1)
			end := 1 + 3*(tc.limit-1)
			for tick := 2; tick <= end+1; tick++ {
				f.live.tick()
				fresh.live.tick()
				check(f, tick)
				check(fresh, tick)
				if f.live.world.Hash() != fresh.live.world.Hash() {
					t.Fatal("native next stage changed", tick)
				}
			}
		})
	}
}

func TestWorldEffects1162BindingSuppressionAndCounterControls(t *testing.T) {
	raw := ring1162Source(t, 4, 2, 0)
	f := spell1152FixtureFront(t)
	ms, _, err := loadOriginalMission(f, raw)
	if err != nil {
		t.Fatal(err)
	}
	b, err := ms.World.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	// Constant-time footer removal simulates the historical frozen carrier.
	carrier := beforeAttackNotices(t, b)
	span := int(binary.LittleEndian.Uint32(carrier[len(carrier)-4:]))
	if span == 0 {
		t.Fatal("driver never imported")
	}
	for _, edit := range []func(*SnapshotSAVDocument){
		func(s *SnapshotSAVDocument) { s.WorldEffects = nil },
		func(s *SnapshotSAVDocument) { s.WorldEffects.Areas[0].ID++ },
		func(s *SnapshotSAVDocument) {
			savedObjectSetValue(&s.Document.Objects[s.WorldEffects.Areas[0].ObjectIndex-1], "Reference", 7)
		},
		func(s *SnapshotSAVDocument) {
			savedObjectSetValue(&s.Document.Objects[s.WorldEffects.Areas[0].ChildIndex-1], "Reference", 7)
		},
		func(s *SnapshotSAVDocument) {
			r := &s.Document.Objects[s.WorldEffects.Areas[0].ObjectIndex-1]
			savedObjectSetValue(r, "T08", 1)
		},
	} {
		bad, err := cloneSavedDocument(ms.savedDocument)
		if err != nil {
			t.Fatal(err)
		}
		edit(bad)
		if restoreSavedDocument(&Mission{World: ms.World}, bad) == nil {
			t.Fatal("lost/wrong driver binding admitted")
		}
	}
}

// The preceding bytes are literal fixtures. Remove the independent outermost
// length-delimited form91 block; never regenerate those historical envelopes.
func beforeWorldEffectForm1162(t *testing.T, b []byte) []byte {
	t.Helper()
	b = beforeAttackNotices(t, b)
	if len(b) == 0 || b[0] != 91 && b[0] != 92 {
		return b
	}
	if len(b) < 38 {
		t.Fatal("truncated world-effect fixture")
	}
	span := uint64(binary.LittleEndian.Uint32(b[len(b)-4:]))
	if span > uint64(len(b)-38) {
		t.Fatal("invalid world-effect fixture span")
	}
	out := bytes.Clone(b[:len(b)-4-int(span)])
	out[0] = 90
	return out
}

func TestWorldEffects1162ProjectionPreservesOpaqueStoreRows(t *testing.T) {
	doc := sav.DocumentData{State: sav.DocumentStateData{
		DirectoryRecords: []sav.CityStateDirectoryData{{Path: "/PRJ7", Kind: 17}, {Path: "/Prj007", Kind: 17}, {Path: "/Prj9", Kind: 17}, {Path: "/Prj999", Kind: 17}},
		ValueRecords: []sav.CityStateRecordData{
			{Path: "/PROJECTILES/FreeIndex", Value: sav.CityStateValueData{Kind: 2, Int32: 42}},
			{Path: "/PROJECTILES/IDS", Value: sav.CityStateValueData{Kind: 6, Bytes: []byte{7, 0, 0x34, 0x12, 9, 0, 0, 0, 7, 0, 0x21, 0x43}}},
			{Path: "/Prj007/unknown", Value: sav.CityStateValueData{Kind: 6, Bytes: []byte{91, 92}}},
			{Path: "/Prj9/constructor-default", Value: sav.CityStateValueData{Kind: 2, Int32: 17}},
			{Path: "/Prj999/unused", Value: sav.CityStateValueData{Kind: 2, Int32: 99}},
		},
	}}
	for _, name := range projectile1157Names {
		doc.State.ValueRecords = append(doc.State.ValueRecords, sav.CityStateRecordData{Path: "/PRJ7/" + strings.ToUpper(name), Value: sav.CityStateValueData{Kind: 2}})
	}
	if !completeSavedProjectileRecord(&doc, 7) || completeSavedProjectileRecord(&doc, 9) {
		t.Fatal("source leaf admission/case fold")
	}
	opaque := append([]sav.CityStateRecordData(nil), doc.State.ValueRecords[2:5]...)
	current := sim.SavedProjectiles{FreeIndex: 42, IDs: []uint16{7, 9, 7}, Items: []sim.SavedProjectile{{ID: 7, X: 99}, {ID: 9, X: 123}}}
	if err := validateSavedProjectileDocument(&doc, current, []uint16{7}); err == nil {
		t.Fatal("stale current leaf admitted")
	}
	if err := projectSavedProjectiles(&doc, current, []uint16{7}); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(doc.State.ValueRecords[2:5], opaque) || doc.State.ValueRecords[5].Value.Int32 != 99 {
		t.Fatal("current write touched opaque rows or ignored ASCII fold")
	}
	current.IDs = []uint16{9}
	current.Items = current.Items[1:]
	if err := projectSavedProjectiles(&doc, current, []uint16{7}); err != nil {
		t.Fatal(err)
	}
	if len(doc.State.DirectoryRecords) != 3 || doc.State.DirectoryRecords[0].Path != "/Prj007" || !reflect.DeepEqual(doc.State.ValueRecords[2:], opaque) || !bytes.Equal(doc.State.ValueRecords[1].Value.Bytes, []byte{9, 0, 0, 0}) {
		t.Fatal("retirement touched an unbound/unused section or lost ID multiplicity transport", doc.State)
	}
	if err := validateSavedProjectileDocument(&doc, current, []uint16{7}); err != nil {
		t.Fatal("current retirement Document", err)
	}
}

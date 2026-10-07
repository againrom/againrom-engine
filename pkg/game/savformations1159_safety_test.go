package game

import (
	"bytes"
	"encoding/binary"
	"reflect"
	"testing"

	"againrom/pkg/sim"
)

func formation1159Snapshot(t *testing.T) (*FrontEnd, Snapshot) {
	t.Helper()
	f := formation1159Front(t)
	open, town, err := f.RestoreOriginal(formation1159Literal(t, f))
	if err != nil || town {
		t.Fatal("original fixture", err)
	}
	if err := f.App("formation safety").OpenMission(open); err != nil {
		t.Fatal(err)
	}
	s, _, err := f.Snapshot(true)
	if err != nil {
		t.Fatal(err)
	}
	return f, s
}

func formation1159Footer(t *testing.T, b []byte) (int, int) {
	t.Helper()
	b = beforeWorldEffectForm1162(t, b)
	if len(b) < 4 || b[0] != 90 {
		t.Fatal("expected native form90")
	}
	start := len(b) - 4 - int(binary.LittleEndian.Uint32(b[len(b)-4:]))
	if start < 0 || binary.LittleEndian.Uint32(b[start:]) != 2 {
		t.Fatal("expected exact two-Player footer")
	}
	groups := start + 4 + 2*11
	if binary.LittleEndian.Uint32(b[groups:]) != 1 {
		t.Fatal("expected one Group owner")
	}
	return start, groups + 4
}

// Peel only the additive form90 suffix; predecessor fixtures stay frozen.
func beforeSavedFormationForm1159(t *testing.T, b []byte) []byte {
	t.Helper()
	b = beforeWorldEffectForm1162(t, b)
	if len(b) == 0 || b[0] != 90 {
		return b
	}
	if len(b) < 38 {
		t.Fatal("truncated formation fixture")
	}
	span := uint64(binary.LittleEndian.Uint32(b[len(b)-4:]))
	if span > uint64(len(b)-38) {
		t.Fatal("invalid formation fixture span")
	}
	out := bytes.Clone(b[:len(b)-4-int(span)])
	out[0] = 89
	return out
}

func TestSavedFormation1159NativePairRefusalsAreAtomic(t *testing.T) {
	f, before := formation1159Snapshot(t)
	start, group := formation1159Footer(t, before.World)
	retarget := func(s *Snapshot, reference bool) {
		binary.LittleEndian.PutUint32(s.World[group+4:], 2)
		b := s.SavedDocument.GroupBindings
		b.Groups[0].Owner = SnapshotSAVGroupReferenceBinding{ObjectIndex: b.Players[1].ObjectIndex, Key: 0x2222, Class: 1, Owner: 1}
		if reference {
			var world sim.World
			if err := world.UnmarshalBinary(s.World); err != nil {
				t.Fatal(err)
			}
			groups, orders, _ := world.SavedGroups()
			groups[0].Owner.Key, groups[0].Owner.Owner = 0x2222, 1
			if err := world.ImportSavedGroups(groups, orders); err != nil {
				t.Fatal(err)
			}
			s.World, _ = world.MarshalBinary()
		}
	}
	for _, tc := range []struct {
		name string
		edit func(*Snapshot)
	}{
		{"lost native carrier with complete document", func(s *Snapshot) { s.World = append(s.World[:start], 0, 0, 0, 0) }},
		{"current native byte", func(s *Snapshot) { s.World[start+14] ^= 1 }},
		{"native Player identity", func(s *Snapshot) { binary.LittleEndian.PutUint32(s.World[start+4:], 2) }},
		{"native command identifier", func(s *Snapshot) { s.World[start+8] ^= 1 }},
		{"native trigger identifier", func(s *Snapshot) { s.World[start+10] ^= 1 }},
		{"other valid native owner", func(s *Snapshot) { binary.LittleEndian.PutUint32(s.World[group+4:], 2) }},
		{"null native owner", func(s *Snapshot) { binary.LittleEndian.PutUint32(s.World[group+4:], 0) }},
		{"owner mismatch behind unrelated coverage gap", func(s *Snapshot) {
			binary.LittleEndian.PutUint32(s.World[group+4:], 2)
			s.SavedDocument.GroupBindings.Unavailable = "unrelated native actor producer unavailable"
		}},
		{"coupled exact owner and binding", func(s *Snapshot) { retarget(s, false) }},
		{"coupled exact owner and binding behind coverage gap", func(s *Snapshot) {
			retarget(s, false)
			s.SavedDocument.GroupBindings.Unavailable = "unrelated native actor producer unavailable"
		}},
		{"coupled owner binding and native reference retain conflicting G44", func(s *Snapshot) {
			retarget(s, true)
			s.SavedDocument.GroupBindings.Unavailable = "unrelated native actor producer unavailable"
		}},
		{"retained G44 behind coverage gap", func(s *Snapshot) {
			b := s.SavedDocument.GroupBindings
			g := b.Groups[0]
			newGroupSetValue1115(t, &s.SavedDocument.Document.Objects[g.PlayerObject-1].Groups[g.InlineIndex], "G44", 0x2222)
			b.Unavailable = "unrelated native actor producer unavailable"
		}},
		{"lost document", func(s *Snapshot) { s.SavedDocument = nil }},
		{"lost Group bindings with retained document", func(s *Snapshot) { s.SavedDocument.GroupBindings = nil }},
		{"lost all auxiliary bindings", func(s *Snapshot) {
			s.SavedDocument.GroupBindings, s.SavedDocument.PlayerPurses, s.SavedDocument.Objects = nil, nil, nil
		}},
		{"false presence marker", func(s *Snapshot) { s.SavedDocument.GroupBindings.FormationsPresent = false }},
		{"duplicate exact Player binding", func(s *Snapshot) { s.SavedDocument.GroupBindings.Players[1] = s.SavedDocument.GroupBindings.Players[0] }},
		{"bound original formation byte", func(s *Snapshot) {
			index := s.SavedDocument.GroupBindings.Players[0].ObjectIndex
			_, _, raw, err := savedPlayerFormationFields(&s.SavedDocument.Document.Objects[index-1])
			if err != nil {
				t.Fatal(err)
			}
			raw[31] ^= 1
		}},
		{"bound original identity", func(s *Snapshot) {
			index := s.SavedDocument.GroupBindings.Players[0].ObjectIndex
			newGroupSetValue1115(t, &s.SavedDocument.Document.Objects[index-1], "This", 0x9999)
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			bad := before
			bad.World = bytes.Clone(before.World)
			var err error
			bad.SavedDocument, err = cloneSavedDocument(before.SavedDocument)
			if err != nil {
				t.Fatal(err)
			}
			tc.edit(&bad)
			if out, err := EncodeSave(bad, "hostile formation"); err == nil || out != nil {
				t.Fatal("EncodeSave published a hostile pair", err)
			}
			if out, label, err := DecodeSave(uncheckedDocumentEnvelope1115(t, bad)); err == nil || label != "" || !reflect.DeepEqual(out, Snapshot{}) {
				t.Fatal("DecodeSave published a hostile pair", err)
			}
			oldLive, oldTown, oldShop := f.live, f.Town, f.Shop
			if open, town, err := f.Restore(bad); err == nil || open != nil || town {
				t.Fatal("Restore prepared a hostile pair", err)
			}
			after, _, err := f.Snapshot(true)
			if err != nil || !reflect.DeepEqual(before, after) || f.live != oldLive || f.Town != oldTown || f.Shop != oldShop {
				t.Fatal("refusal changed the running session", err)
			}
		})
	}
}

func TestSavedFormation1159ValidOwnerCoverageGaps(t *testing.T) {
	_, source := formation1159Snapshot(t)
	for _, tc := range []struct {
		name     string
		resolved bool
		key, raw uint32
	}{
		{"resolved owner", true, 0x1111, 0x1111},
		{"null owner", false, 0, 0},
		{"unresolved source owner", false, 0xdeadbeef, 0xdeadbeef},
		{"projected null owner", false, 0xdeadbeef, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			good := source
			var err error
			good.SavedDocument, err = cloneSavedDocument(source.SavedDocument)
			if err != nil {
				t.Fatal(err)
			}
			var world sim.World
			if err := world.UnmarshalBinary(source.World); err != nil {
				t.Fatal(err)
			}
			groups, orders, _ := world.SavedGroups()
			b := good.SavedDocument.GroupBindings
			if !tc.resolved {
				groups[0].OwnerID = 0
				groups[0].Owner = sim.SavedGroupReference{Key: tc.key}
				b.Groups[0].Owner = SnapshotSAVGroupReferenceBinding{Key: tc.key}
			}
			g := b.Groups[0]
			newGroupSetValue1115(t, &good.SavedDocument.Document.Objects[g.PlayerObject-1].Groups[g.InlineIndex], "G44", tc.raw)
			b.Unavailable = "unrelated native actor producer unavailable"
			if err := world.ImportSavedGroups(groups, orders); err != nil {
				t.Fatal(err)
			}
			good.World, _ = world.MarshalBinary()
			native, err := EncodeSave(good, "valid owner gap")
			if err != nil {
				t.Fatal("valid gap refused EncodeSave", err)
			}
			back, label, err := DecodeSave(native)
			if err != nil || label != "valid owner gap" || !bytes.Equal(back.World, good.World) || !reflect.DeepEqual(back.SavedDocument, good.SavedDocument) {
				t.Fatal("valid gap changed at native readback", err)
			}
			fresh := formation1159Front(t)
			open, town, err := fresh.Restore(back)
			if err != nil || open == nil || town {
				t.Fatal("valid gap refused Restore", err)
			}
			if err := fresh.App("valid owner gap").OpenMission(open); err != nil {
				t.Fatal(err)
			}
			if fresh.live.world.Hash() != world.Hash() {
				t.Fatal("valid owner gap changed current World")
			}
			fresh.live.cycleFormation()
			for range 16 {
				fresh.live.tick()
				players, _ := fresh.live.world.SavedPlayerFormations()
				if players[0].Mode == 255 {
					break
				}
			}
			players, _ := fresh.live.world.SavedPlayerFormations()
			if players[0].Mode != 255 || players[1].Mode != 0 {
				t.Fatal("valid gap changed formation dispatch", players)
			}
			actors := 0
			for _, e := range fresh.live.world.Entities() {
				if e.SourceBinding.Class == 0 {
					continue
				}
				if tc.resolved && (!e.HasTarget || e.TargetX != int32(29+2*actors) || e.TargetY != 30) || !tc.resolved && e.HasTarget {
					t.Fatal("valid gap changed next movement policy", e.ID, e.TargetX, e.TargetY)
				}
				actors++
			}
			if actors != 2 {
				t.Fatal("valid gap lost source actors", actors)
			}
		})
	}
}

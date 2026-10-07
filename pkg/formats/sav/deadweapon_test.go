package sav

import "testing"

func deadWeaponSentinelRecord(t *testing.T) *Record {
	t.Helper()
	// Independent CArchive Weapon: schema/class tag, Token's 37 bytes,
	// empty Effects, 12 Item bytes, W52[24], W6A[22], W50 and null WeaponSpell.
	b := []byte{255, 255, 1, 0, 6, 0, 'W', 'e', 'a', 'p', 'o', 'n'}
	for v := byte(1); v <= 37; v++ {
		b = append(b, v)
	}
	b = append(b, 0, 0, 0, 0)
	for v := byte(38); v <= 96; v++ {
		b = append(b, v)
	}
	b = append(b, 0, 0)
	w := &walker{b: b, next: 1, classes: map[uint16]string{}, objects: map[uint16]*Record{}}
	r, err := w.object(0)
	if err != nil || w.p != len(b) {
		t.Fatalf("sentinel weapon: %v end=%d/%d", err, w.p, len(b))
	}
	return r
}

func TestDeadWeaponEveryWireMemberIsRetained(t *testing.T) {
	r := deadWeaponSentinelRecord(t)
	want := DeadWeapon{
		Present: true, ArchiveIndex: 2, Cell: 0x0201, PackedCell: 0x0403, FineX: 5, FineY: 6, PositionU06: 0x0807,
		TerrainKey: 0x0c0b0a09, RuntimeID: 0x100f0e0d, T0C: 17, T0E: 0x1312, T08: 0x17161514, T18: 0x1918,
		T1C: 0x1d1c1b1a, Identity: 0x21201f1e, Reference: 0x25242322,
		F40: 0x2726, F42: 0x2928, F44: 42, F45: 43, F46: 44, F48: 0x2e2d, F4A: 0x302f, F47: 49,
		W52: [24]byte{50, 51, 52, 53, 54, 55, 56, 57, 58, 59, 60, 61, 62, 63, 64, 65, 66, 67, 68, 69, 70, 71, 72, 73},
		W6A: [22]byte{74, 75, 76, 77, 78, 79, 80, 81, 82, 83, 84, 85, 86, 87, 88, 89, 90, 91, 92, 93, 94, 95}, W50: 96,
	}
	got, ok := projectDeadWeapon(r)
	if !ok || got != want {
		t.Fatalf("projected=%+v want=%+v supported=%t", got, want, ok)
	}
	r.Raw["W52"][0], r.Raw["W6A"][0], r.Raw["Block12"][0] = 0, 0, 0
	if got != want {
		t.Fatal("source buffers alias retained weapon")
	}
}

func TestDeadWeaponRefusesContentOrUnrepresentedFields(t *testing.T) {
	for name, mutate := range map[string]func(*Record){
		"effect":           func(r *Record) { r.Counts["Effects"] = 1 },
		"spell reference":  func(r *Record) { r.Refs["WeaponSpell"] = []*Record{{Class: "Spell"}} },
		"unknown scalar":   func(r *Record) { r.Value["Future"] = 1 },
		"missing scalar":   func(r *Record) { delete(r.Value, "F47") },
		"wrong class":      func(r *Record) { r.Class = "Shield" },
		"short member":     func(r *Record) { r.Raw["W52"] = r.Raw["W52"][:23] },
		"missing identity": func(r *Record) { r.Value["Identity"] = 0 },
	} {
		t.Run(name, func(t *testing.T) {
			r := deadWeaponSentinelRecord(t)
			mutate(r)
			if _, ok := projectDeadWeapon(r); ok {
				t.Fatal("accepted unsupported Weapon")
			}
		})
	}
}

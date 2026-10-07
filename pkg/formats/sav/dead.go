package sav

import "fmt"

// DeadActor is an exact top-level dead-manager element, not an actor-head
// scan. Keys are archive-local provenance, not live pointers. Stage, health
// and timer are independent wire scalars (SAV-DEADLOAD-124..130).
type DeadActor struct {
	Off                                       int
	ArchiveIndex                              uint16
	Class                                     string
	Identity, RuntimeID, TerrainKey, OwnerKey uint32
	References                                [5]uint32 // +5c, +64, +44, +68, +40; source keys only
	MapUnitID, Cell                           uint16
	FineX, FineY, Stage                       uint8
	HP                                        int16
	Timer                                     int8
	Effects, Carried, Worn                    int
	ContainerPresent                          bool
	ContainerTail                             [2]uint32
	HeldWeapon                                DeadWeapon
	// TypeID is the head's +0x0e word, the class-defining field ActorGraph's
	// own ActorRecord already reads off this identical record shape for both
	// Players and the dead list (actorgraph.go: "T0E"). This project's own
	// present dead-actor corpus (game0032/game0033, a hired mercenary's own
	// MapUnitID-0 record) carries a nonzero, stable T0E across two saves,
	// confirming the field is populated for a dead-list object and not only a
	// live one.
	TypeID uint16
}

// DeadActors advances one shared class/object index through all Players and
// the counted dead list. A repeated archive reference names one actor; a
// different record with the same identity is ambiguous and rejected.
func (f *File) DeadActors() ([]DeadActor, error) {
	if f == nil {
		return nil, fmt.Errorf("sav: dead actors require a save")
	}
	doc, _, err := f.exactDocument()
	if err != nil {
		return nil, err
	}
	seen := make(map[*Record]bool)
	keys := make(map[uint32]bool)
	var out []DeadActor
	for _, r := range doc.dead {
		if seen[r] {
			continue
		}
		seen[r] = true
		key := r.value("Identity")
		if key == 0 || keys[key] {
			return nil, fmt.Errorf("sav: dead actor at %d has zero or repeated identity %#x", r.Off, key)
		}
		keys[key] = true
		p := r.Raw["Block12"]
		if len(p) != 12 {
			return nil, fmt.Errorf("sav: dead actor at %d has incomplete Position", r.Off)
		}
		d := DeadActor{Off: r.Off, ArchiveIndex: r.Index, Class: r.Class, Identity: key,
			RuntimeID: r.value("RuntimeID"), TerrainKey: u32(p, 8), OwnerKey: r.value("Reference"),
			MapUnitID: uint16(r.value("T08")), Cell: u16(p, 0), FineX: p[4], FineY: p[5],
			Stage: uint8(r.value("Stage")), HP: int16(r.value("Health")), Timer: int8(r.value("U6C")),
			TypeID:  uint16(r.value("T0E")),
			Effects: r.Counts["Effects"], Carried: r.Counts["Inventory"],
			Worn:             len(r.Refs["HeldWeapon"]) + len(r.Refs["HeldShield"]) + len(r.Refs["Worn"]),
			ContainerPresent: r.value("HasInventory") != 0,
			References:       [5]uint32{r.value("U5C"), r.value("U64"), r.value("U44"), 0, r.value("U40")},
			ContainerTail:    [2]uint32{r.value("Inventory1C"), r.value("Inventory20")},
		}
		if refs := r.Refs["U68"]; len(refs) != 0 {
			d.References[3] = refs[0].value("Identity")
		}
		if refs := r.Refs["HeldWeapon"]; len(refs) == 1 {
			d.HeldWeapon, _ = projectDeadWeapon(refs[0])
		}
		out = append(out, d)
	}
	return out, nil
}

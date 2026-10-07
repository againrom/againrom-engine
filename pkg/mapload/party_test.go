package mapload_test

import (
	"bytes"
	"testing"

	"againrom/pkg/data"
	"againrom/pkg/formats/alm"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
)

// OwnParty is the ownership boundary, not a slice copy. Even a malformed
// caller that aliases all reference-backed character state between two people
// cannot make selection or mission construction turn a write to one into a
// write to the other.
func TestOwnPartyBreaksEveryMutableCrossMemberAlias(t *testing.T) {
	weapon := &data.Weapon{Name: "shared source", DamageBase: 7}
	carry := &mapload.Carry{Items: []uint16{0x0101, 0x0202}}
	carry.SkillXP[1] = 111
	carry.Equipped[0] = 0x0303
	saved := &mapload.Saved{Cell: mapload.Cell{X: 4, Y: 5}, HP: 6}
	carried := []uint16{0x0404, 0x0505}
	in := []mapload.PartyMember{
		{ID: "person", StartingHero: true, Weapon: weapon, Carry: carry, Saved: saved, Carried: carried},
		{ID: "person", CompanionNPC: 22, Weapon: weapon, Carry: carry, Saved: saved, Carried: carried},
	}

	out := mapload.OwnParty(in)
	if out[0].ID != "person" || out[1].ID != "person:2" {
		t.Fatalf("stable unique ids = %q, %q", out[0].ID, out[1].ID)
	}
	if out[0].Weapon == out[1].Weapon || out[0].Carry == out[1].Carry || out[0].Saved == out[1].Saved {
		t.Fatal("owned members still share pointer-backed character state")
	}

	out[0].Weapon.DamageBase = 99
	out[0].Carry.SkillXP[1] = 999
	out[0].Carry.Items[0] = 0xffff
	out[0].Carry.Equipped[0] = 0xffff
	out[0].Saved.HP = 999
	out[0].Carried[0] = 0xffff
	if out[1].Weapon.DamageBase != 7 || out[1].Carry.SkillXP[1] != 111 ||
		out[1].Carry.Items[0] != 0x0101 || out[1].Carry.Equipped[0] != 0x0303 ||
		out[1].Saved.HP != 6 || out[1].Carried[0] != 0x0404 {
		t.Fatalf("member 0 mutation leaked into member 1: %#v", out[1])
	}
	if in[0].Weapon.DamageBase != 7 || in[0].Carry.SkillXP[1] != 111 ||
		in[0].Carry.Items[0] != 0x0101 || in[0].Carry.Equipped[0] != 0x0303 ||
		in[0].Saved.HP != 6 || in[0].Carried[0] != 0x0404 {
		t.Fatal("owned mutation leaked back into the caller's party")
	}
}

// TestOwnPartyCarriesWeaponMaterializedAsAPlainValue is counterexample A/B's
// own witness for the cross-mission boundary (round-2 adversarial review,
// fifth pass): WeaponMaterialized is a persisted history bit, not re-derived
// from Weapon or Carry, so OwnParty's plain value-copy must carry it through
// even when the member's Carry shows no present-state evidence that the
// fallback ever fired — an empty pack, the starting weapon's own code
// nowhere in it.
func TestOwnPartyCarriesWeaponMaterializedAsAPlainValue(t *testing.T) {
	weapon := &data.Weapon{Name: "resolved starting weapon", DamageBase: 3}
	in := []mapload.PartyMember{
		{ID: "person", StartingHero: true, Weapon: weapon, WeaponMaterialized: true,
			Carry: &mapload.Carry{}},
		{ID: "bystander", CompanionNPC: 22, Weapon: weapon},
	}

	out := mapload.OwnParty(in)
	if !out[0].WeaponMaterialized {
		t.Error("OwnParty dropped WeaponMaterialized for a member whose Carry holds no trace of the resolved weapon")
	}
	if out[1].WeaponMaterialized {
		t.Error("OwnParty set WeaponMaterialized true for a member who never had it set")
	}
}

// placed adds n placements to a map, spread along a row well clear of the drop
// cell so nothing here depends on where the party's outward walk goes.
func placed(m *alm.Map, n int) *alm.Map {
	m.Units = make([]alm.Unit, n)
	for i := range m.Units {
		m.Units[i] = alm.Unit{
			X: uint32(2+i%30) << 8, Y: uint32(2+i/30) << 8,
			ClassID: int16(200 + i), UnitID: uint16(500 + i),
		}
	}
	return m
}

// THE PIN. What PartyEntity answers is what StartMission actually did — asserted
// against the started world rather than against a recomputed count, because a
// count recomputed the same wrong way agrees with itself.
func TestPartyEntityIsTheIdTheStartAssigns(t *testing.T) {
	for _, placements := range []int{0, 1, 5, 35} {
		for _, size := range []int{0, 1, 3, 6} {
			m := placed(startMap(t, 40, 40, mapload.Cell{X: 17, Y: 20}), placements)
			p := party(size)
			w, st := mustStart(t, m, p)

			ents := w.Entities()
			if got, want := len(ents), placements+size; got != want {
				t.Fatalf("placements=%d size=%d: world holds %d entities, want %d",
					placements, size, got, want)
			}
			for i := range p {
				want := mapload.PartyEntity(m, i)
				got := ents[placements+i]
				if got.ID != want {
					t.Errorf("placements=%d size=%d: member %d is entity %d, PartyEntity says %d",
						placements, size, i, got.ID, want)
				}
				// The id alone would survive an off-by-one that swapped two
				// members; the cell is what ties the id to the right one.
				if got.X != st.Cells[i].X || got.Y != st.Cells[i].Y {
					t.Errorf("placements=%d size=%d: member %d entity at (%d,%d), start put it at %v",
						placements, size, i, got.X, got.Y, st.Cells[i])
				}
			}
		}
	}
}

// A map with no placements and no map at all are the same world, so they give
// the same answer — and member 0 of an empty map is entity 0, which is a real
// entity id and not a sentinel.
func TestPartyEntityWithNothingPlaced(t *testing.T) {
	empty := placed(startMap(t, 40, 40, mapload.Cell{X: 17, Y: 20}), 0)
	for i := range 3 {
		if got, want := mapload.PartyEntity(empty, i), sim.EntityID(i); got != want {
			t.Errorf("no placements, member %d: got %d, want %d", i, got, want)
		}
		if got, want := mapload.PartyEntity(nil, i), sim.EntityID(i); got != want {
			t.Errorf("nil map, member %d: got %d, want %d", i, got, want)
		}
	}
}

// TestAPartyMemberStandsOnThePlayersRosterSlot is 0094 AC-1 and AC-2, and it is
// asserted against sim's own constant rather than against a 1 written here: a
// literal would agree with the placement even on the day the placement was wrong.
//
// THE GROUP WORD IS ASSERTED TOO, and it is the half a reader is likelier to
// think is missing rather than chosen. Zero is what the field already held, and
// what it buys is that every member of one start shares a group with every other
// member and with no placement — so a party is one group, whatever the map holds.
func TestAPartyMemberStandsOnThePlayersRosterSlot(t *testing.T) {
	for _, placements := range []int{0, 1, 35} {
		for _, size := range []int{1, 3, 6} {
			m := placed(startMap(t, 40, 40, mapload.Cell{X: 17, Y: 20}), placements)
			w, _ := mustStart(t, m, party(size))
			ents := w.Entities()
			for i := 0; i < size; i++ {
				e := ents[placements+i]
				if e.Owner != sim.SelfSlot {
					t.Errorf("placements=%d size=%d: member %d owns slot %d, want %d — "+
						"slot 0 is outside the relation matrix in both directions",
						placements, size, i, e.Owner, uint32(sim.SelfSlot))
				}
				if e.Group != 0 {
					t.Errorf("placements=%d size=%d: member %d carries group word %d, want 0",
						placements, size, i, e.Group)
				}
			}
			// The placements are the discriminator: a change that stamped the slot
			// on everything rather than on the party would pass the loop above.
			for i := 0; i < placements; i++ {
				if ents[i].Owner != 0 {
					t.Errorf("placements=%d: placement %d owns slot %d, want 0 — its record names none",
						placements, i, ents[i].Owner)
				}
			}
		}
	}
}

// TestAStartedWorldCarriesThePartysSlotThroughTheByteForm is 0094 AC-10.
//
// The owner slot has been a field of the encoded entity since long before this
// story, so no version and no field is taken here. What was never exercised is
// the population: every world that round-tripped with an owner in it got that
// owner off a map record, and a party member's came from nowhere. This is the
// path the game actually saves and hashes, driven from a start rather than from
// hand-built entities.
//
// THE ASSERTION IS ON THE DECODED ENTITY AND NOT ON THE VERSION BYTE. A version
// literal is a statement about what somebody believed the form to be; the entity
// that comes back is the form.
func TestAStartedWorldCarriesThePartysSlotThroughTheByteForm(t *testing.T) {
	m := placed(startMap(t, 40, 40, mapload.Cell{X: 17, Y: 20}), 4)
	w, st := mustStart(t, m, party(3))

	form, err := w.MarshalBinary()
	if err != nil {
		t.Fatalf("MarshalBinary: %v", err)
	}
	var back sim.World
	if err := back.UnmarshalBinary(form); err != nil {
		t.Fatalf("UnmarshalBinary: %v", err)
	}
	again, err := back.MarshalBinary()
	if err != nil {
		t.Fatalf("MarshalBinary after the decode: %v", err)
	}
	if !bytes.Equal(form, again) {
		t.Errorf("a started world does not round-trip: %d byte(s) out, %d back",
			len(form), len(again))
	}
	if back.Hash() != w.Hash() {
		t.Errorf("the decoded world hashes %#016x, the started one %#016x",
			back.Hash(), w.Hash())
	}
	held := map[sim.EntityID]uint32{}
	for _, e := range back.Entities() {
		held[e.ID] = e.Owner
	}
	for i, id := range st.IDs {
		if held[id] != sim.SelfSlot {
			t.Errorf("member %d comes back on slot %d, want %d",
				i, held[id], uint32(sim.SelfSlot))
		}
	}
}

// TestAStartedPartyMembersLoadoutSurvivesTheByteForm is 0134's own widening
// of the test above, at the same shape: a member's Worn array and Carried
// slice now reach hashed state through his own sim.Stock entry, and this is
// the path the game actually saves and hashes, driven from a start rather
// than from a hand-built Stock.
//
// A SECOND MEMBER CARRYING NEITHER IS ASSERTED BESIDE THE LOADED ONE, on
// this file's own reason above: a build that spread one member's loadout
// over the wrong entity would satisfy neither half.
func TestAStartedPartyMembersLoadoutSurvivesTheByteForm(t *testing.T) {
	m := placed(startMap(t, 40, 40, mapload.Cell{X: 17, Y: 20}), 2)
	var worn [sim.EquipSlots]uint16
	worn[0], worn[7] = 0x0107, 0x0208
	p := []mapload.PartyMember{
		{Class: 100},
		{Class: 101, Worn: worn, Carried: []uint16{0x0301, 0x0302}},
	}
	w, st := mustStart(t, m, p)

	form, err := w.MarshalBinary()
	if err != nil {
		t.Fatalf("MarshalBinary: %v", err)
	}
	var back sim.World
	if err := back.UnmarshalBinary(form); err != nil {
		t.Fatalf("UnmarshalBinary: %v", err)
	}
	if back.Hash() != w.Hash() {
		t.Errorf("the decoded world hashes %#016x, the started one %#016x", back.Hash(), w.Hash())
	}

	if eq, ok := back.Equipped(st.IDs[0]); !ok || eq != ([sim.EquipSlots]uint16{}) {
		t.Errorf("member 0 comes back equipped %v (present=%v), want every slot empty — "+
			"he carries neither a Worn code nor a Carried one", eq, ok)
	}
	eq, ok := back.Equipped(st.IDs[1])
	if !ok || eq != worn {
		t.Errorf("member 1 comes back equipped %v (present=%v), want %v", eq, ok, worn)
	}
	carried, ok := back.Carried(st.IDs[1])
	if !ok || len(carried) != 2 || carried[0] != 0x0301 || carried[1] != 0x0302 {
		t.Errorf("member 1 comes back carrying %v (present=%v), want [0x0301 0x0302]", carried, ok)
	}
}

// Package mapload_test exercises PartyLoadout's own everEquipped argument
// directly: the mission-construction path a member's combat stats are
// re-derived from at every mission boundary (PartySpawnWithTable).
//
// Everything here is synthetic, on this package's own convention: no map, no
// table read from disk, no game install.
package mapload_test

import (
	"testing"

	"againrom/pkg/data"
	"againrom/pkg/mapload"
)

// startingSword is a melee weapon fixture whose damage pair is far from
// zero, so a member armed from it and a member armed bare-handed cannot
// derive the same combat block by coincidence.
func startingSword() *data.Weapon {
	return &data.Weapon{Name: "Sword", Code: 4242, DamageBase: 30, DamageSpread: 6, ToHit: 5}
}

// TestPartyLoadoutRespectsTheMaterializedLatch is the mutation witness for
// round-2's C3 fix (adversarial review, seventh pass): PartyLoadout's own
// everEquipped argument to ResolveEquipmentLoadout must be
// p.WeaponMaterialized, not an unconditional false. A member who has never
// had the latch raised — WeaponMaterialized false, slot 1 never seen
// occupied — still arms from the fallback weapon at mission construction,
// exactly as before this fix. A member whose latch IS raised — the
// starting weapon was taken off, in this mission or an earlier one, and slot
// 1 now reads empty — must arm bare-handed: a raised latch means the
// array, not the fallback picture, speaks for slot 1 from then on.
func TestPartyLoadoutRespectsTheMaterializedLatch(t *testing.T) {
	sword := startingSword()

	bare := mapload.PartyMember{Weapon: sword}
	tookOff := mapload.PartyMember{Weapon: sword, WeaponMaterialized: true}

	bareLoadout := mapload.PartyLoadout(bare, nil)
	if bareLoadout.Weapon == nil || bareLoadout.Weapon.Code != sword.Code {
		t.Fatalf("bare member's loadout = %+v, want the fallback weapon (code %d)", bareLoadout.Weapon, sword.Code)
	}

	tookOffLoadout := mapload.PartyLoadout(tookOff, nil)
	if tookOffLoadout.Weapon != nil {
		t.Fatalf("member with the latch raised and an empty slot 1 armed from %+v, want Weapon nil (bare-handed)",
			tookOffLoadout.Weapon)
	}

	// The two must derive a DIFFERENT combat block through the SAME mint
	// mission construction uses — the player-visible consequence the C3
	// defect produced: a hero who sold his starting weapon kept its damage
	// at every following mission boundary.
	bareDerived, _, _ := mapload.PartySpawnWithTable(bare, nil)
	tookOffDerived, _, _ := mapload.PartySpawnWithTable(tookOff, nil)
	if bareDerived.Combat.DamageBase == tookOffDerived.Combat.DamageBase &&
		bareDerived.Combat.DamageSpread == tookOffDerived.Combat.DamageSpread {
		t.Fatalf("a bare member and one who took his starting weapon off both mint DamageBase %d, DamageSpread %d — the fallback was re-applied",
			tookOffDerived.Combat.DamageBase, tookOffDerived.Combat.DamageSpread)
	}
}

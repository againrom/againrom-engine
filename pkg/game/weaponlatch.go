package game

import "againrom/pkg/mapload"

// THE STARTING WEAPON'S MATERIALIZATION LATCH — one writer, and a test that
// fails when a second appears.
//
// PartyMember.WeaponMaterialized (pkg/mapload/start.go) records that a member's
// starting weapon has stopped being a picture and become a real item: it sits in
// an equipment slot, in a container, on a shop table, on the ground, or it has
// been sold. Until then the weapon exists only as PartyMember.Weapon, which the
// doll and the shop's own faces widen slot 1 with (currentFigureEquipment,
// shopSlot1Code) so a member the loader never gave a slot-1 array cell is drawn
// holding what he is armed with.
//
// THE LATCH IS ONE-WAY AND IT IS PERSISTED. Once true it never returns to false:
// there is exactly one bit and it means "this has already happened at least
// once", so no reader has to re-derive it from a present state that the item may
// since have left. Re-derivation is what this file exists to prevent — four
// separate adversarial passes returned this story on it, each time because some
// site inferred the latch from what a container or an array held THIS INSTANT:
//
//   - a pack scan for the starting weapon's own code read a SECOND unit of that
//     code as proof the first had materialized, and latched the fallback off for
//     a member who had never taken anything off;
//   - a mid-mission write reached a party slice the save did not read, so the
//     latch was correct in memory and false on disk;
//   - the shop set it on one of its two arms, so taking a REAL weapon off left
//     it false and the fallback was offered a second time, minting a duplicate.
//
// THE MECHANISM AGAINST THE NEXT ONE IS materializeStartingWeapon PLUS
// TestWeaponMaterializedHasOneWriter (weaponlatch_scan_test.go), which parses
// this package's own source and fails when PartyMember.WeaponMaterialized is
// assigned anywhere but here, or mentioned in a file the closure table does not
// name. The table it keeps honest is the write/read enumeration in
// docs/1005-interactive-doll/closure.md. A comment asking the next reader to
// remember would not have failed anything.

// materializeStartingWeapon raises the latch on member. It is the ONLY
// assignment to PartyMember.WeaponMaterialized in this package, and the scan
// test above is what keeps that true.
//
// A nil member is the ordinary case rather than an error: a subject the mission
// party could not resolve (a map actor that is not a party member at all) has no
// record to remember anything in, and the caller's own present-state effect —
// the item it just moved — still stands.
func materializeStartingWeapon(member *mapload.PartyMember) {
	if member == nil {
		return
	}
	member.WeaponMaterialized = true
}

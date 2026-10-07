package game

import (
	"testing"

	"againrom/pkg/data"
	"againrom/pkg/mapload"
	"againrom/pkg/ui"
)

// A town-chapter companion grant (addChapterCompanions, npc:22) appends
// straight to f.Carried after the current mission's document is already
// frozen. She reaches retainMissionCity's per-binding loop with a nil Carry,
// the same shape missionCityProvenance's own graft path already tolerates.
// checkReturn refuses a nil Carry unconditionally by design (every genuinely
// carried-through member has one), so retainMissionCity must not call it for
// her at all. This is an engine-constructed fixture built from a genuinely
// opened mission's own live roster and a lawful install's NPC registry, not a
// saved game.
func TestReleaseMissionCityReturnSkipsCheckReturnForAGraftedMemberWithNoLiveCarry(t *testing.T) {
	f := releaseFront(t)
	// A fresh campaign's own mission 20 has no preceding town
	// (f.originalCity == nil at open), so the hero carries none of an
	// imported return's own provenance flags (OriginalHuman, Saved, Hired)
	// that would independently refuse checkReturn and obscure this fix.
	party := f.ChargenParty(ui.ChargenResult{Name: "Graft return", Choices: []int{1, 0, 0}, Stats: []int{31, 27, 24, 29}})
	app := f.App("graft return")
	if err := app.OpenMission(f.MissionOpenerWith(20, party)); err != nil {
		t.Fatalf("open mission 20: %v", err)
	}
	// Use the finish boundary's current roster projection, including Book state.
	f.Carried = mapload.CarryRoster(f.live.mission.party, f.live.world, f.live.mission.ids, nil)
	// Weapon.Name is "the literal this weapon was resolved from" (data.Weapon's
	// own doc comment), not a canonical label: chargen's own starting-weapon
	// literal ("Iron Short Sword") and the baseline's table-driven reverse
	// composition from the identical Code ("Common Iron Short Sword") are free
	// to differ in spelling for the same weapon. A live session that saved and
	// reloaded even once before this point would already carry the composed
	// spelling; this fixture never does, so it is aligned here by hand -- a
	// pre-existing, out-of-scope naming quirk this fix neither causes nor
	// claims to repair, kept from obscuring what this test actually isolates.
	for i := range f.Carried {
		if !f.Carried[i].StartingHero || f.Carried[i].Weapon == nil {
			continue
		}
		shapes, materials, weapons := f.tableWeapons()
		if canonical, err := data.WeaponFromCode(f.Carried[i].Weapon.Code, shapes, materials, weapons); err == nil {
			f.Carried[i].Weapon.Name = canonical.Name
		}
	}
	// npc 22 (the same companion
	// TestReleaseMissionCityBindsAChapterGrantedCompanionWithNoLiveEntity
	// grafts at the provenance layer) is appended directly to f.Carried, the
	// same shape addChapterCompanions leaves behind on a town arrival: her
	// own Carry stays nil because she was never loaded into this mission's
	// live party.
	companion, ok := mapload.CampaignNPCMember(f.Table, 22, 30, f.Carried)
	if !ok {
		t.Fatal("CampaignNPCMember(22, 30): the installed registry has no composed npc22 row")
	}
	if companion.Carry != nil {
		t.Fatal("a freshly granted companion must have no live Carry to reproduce this gap")
	}
	body, dir, class, matched := data.HeroAppearance(f.Bodies, mapload.EquipmentFromParty(companion), companion.Mage, false)
	if !matched {
		t.Fatal("chapter companion appearance did not compose")
	}
	companion.Body, companion.BodyDir, companion.Class = string(body), dir, class
	f.Carried = append(f.Carried, companion)

	if err := f.retainMissionCity(f.townInstall(), 20, f.live.world); err != nil {
		t.Fatalf("retainMissionCity: %v, want the grafted companion's nil Carry to skip checkReturn rather than refuse it", err)
	}
	if f.originalCity == nil || len(f.originalCity.bindings) != len(f.Carried) {
		t.Fatalf("originalCity bindings = %v, want one per current party member (%d)", f.originalCity, len(f.Carried))
	}
	var heroChecked, companionSkipped bool
	for _, b := range f.originalCity.bindings {
		if b.partyID == companion.ID {
			if b.returned != nil {
				t.Fatal("grafted companion binding.returned must stay nil: checkReturn cannot verify a member with no live Carry")
			}
			companionSkipped = true
			continue
		}
		if b.returned == nil {
			t.Fatalf("party member %s lost its verified return", b.partyID)
		}
		heroChecked = true
	}
	if !heroChecked || !companionSkipped {
		t.Fatalf("expected both a checked hero binding and a skipped companion binding, heroChecked=%v companionSkipped=%v", heroChecked, companionSkipped)
	}
	for _, member := range f.Carried {
		if member.ID == companion.ID && (member.Carry == nil || member.Weapon == nil ||
			member.Weapon.SpellName != companion.Weapon.SpellName || member.Weapon.SpellPower != companion.Weapon.SpellPower) {
			t.Fatal("binding the grant lost its current staff or source basis")
		}
	}
}

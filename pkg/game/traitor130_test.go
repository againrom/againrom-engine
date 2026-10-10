package game

import (
	"testing"

	"againrom/pkg/formats/alm"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
)

func TestCampaignCompanionBindingUsesRoleInsteadOfPartyPosition(t *testing.T) {
	m := &alm.Map{}
	table := shippedHeroTable(t)
	party := []mapload.PartyMember{
		heroMember("hero", false, true, 3, starting),
		{ID: "hire", MercenaryType: 1},
		heroMember("another", false, false, 5),
		heroMember("npc:22", true, true, 1),
	}
	refs := campaignScriptRefs(m, table, party, 130)
	if !refs.HasCompanion || refs.Companion != mapload.PartyEntity(m, 3) {
		t.Fatalf("companion = %+v", refs)
	}
	if absent := campaignScriptRefs(m, table, party[:3], 130); absent.HasCompanion {
		t.Fatal("bound an unrelated second member")
	}
}

func TestTransferredCharacterLossGuardSurvivesRemovalAndNativeEnvelope(t *testing.T) {
	world := func(owner uint32, present bool) *mapWorld {
		t.Helper()
		entities := []sim.Entity{{ID: 1, Owner: sim.SelfSlot, HP: 10, MaxHP: 10}}
		if present {
			entities = append(entities, sim.Entity{ID: 2, Owner: owner, HP: -10001, MaxHP: 10, TypeID: sim.HumanTypeID, Humanoid: true})
		}
		w := heroWorld(t, nil, nil, entities)
		return partyWorld(t, w, townCompanionParty(), []sim.EntityID{1, 2})
	}
	mw := world(8, true)
	if mw.guardedCharacterLost() {
		t.Fatal("enemy companion's death counted as an owned character's death")
	}
	b, err := EncodeSave(Snapshot{Residue: mw.residue()}, "transferred")
	if err != nil {
		t.Fatal(err)
	}
	s, _, err := DecodeSave(b)
	if err != nil {
		t.Fatal(err)
	}
	afterRemoval := world(0, false)
	afterRemoval.applyResidue(s.Residue)
	if afterRemoval.guardedCharacterLost() || afterRemoval.mission.outcome == sim.OutcomeLost {
		t.Fatal("cold load forgot a transferred character after final removal")
	}
	returned := world(sim.SelfSlot, true)
	returned.applyResidue(s.Residue)
	if !returned.guardedCharacterLost() {
		t.Fatal("a currently owned dead character inherited an enemy exemption")
	}
	missing := world(0, false)
	if !missing.guardedCharacterLost() {
		t.Fatal("an absent owned character without a transfer record did not lose")
	}
	s.Residue.MissionLost = true
	oldLoss := world(8, true)
	oldLoss.applyResidue(s.Residue)
	if oldLoss.mission.outcome != sim.OutcomeLost {
		t.Fatal("transfer repair cleared an already recorded loss")
	}
}

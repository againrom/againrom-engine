package game

import (
	"reflect"
	"testing"

	"againrom/pkg/mapload"
)

// A town's gate leave asks the addition producer for nobody: only a won
// mission's leave reads slot 773 (R2-ENGINE-634, R2-ENGINE-1592). The slot
// keeps its value until the next ordinary departure zeroes it. The first
// game's gate leave keeps its party as before.
func TestSecondGateLeaveAddsNoDepartureCompanion(t *testing.T) {
	f, _, screen, _ := secondBagFront(t, nil)
	f.Table = identityTable()
	f.Carried = []mapload.PartyMember{identityHero()}
	c := f.Town.second
	c.current, c.bank[773] = secondTown(2), 23
	destinationTownCampaign{screen}.leave()
	if got := carriedIdentities(f.Carried, f.Table); len(got) != 1 || got[0] != 21 {
		t.Fatalf("a town's gate leave carries identities %v, want the hero alone", got)
	}
	if c.current != (partyPlace{}) || c.bank[773] != 23 {
		t.Fatalf("the gate leave stands at %+v with slot 773 %d, want no place and 23", c.current, c.bank[773])
	}

	party := []mapload.PartyMember{{ID: "hero", StartingHero: true}, {ID: "npc:23", CompanionNPC: 23, PlayerCharacter: true}}
	first := &townScreen{shopController: shopController{sess: &CampaignSession{Town: NewTown(Campaign{}),
		Carried: mapload.CloneParty(party)}, in: &InstallResources{Table: identityTable()}}}
	first.campaignTown().leave()
	if !reflect.DeepEqual(first.sess.Carried, party) {
		t.Fatalf("the first game's gate leave changed the party to %+v", first.sess.Carried)
	}
}

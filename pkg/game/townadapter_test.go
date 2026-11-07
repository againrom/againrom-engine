package game

import "testing"

func TestTownBankAdapterKeepsEveryIDAndTheCampaignBank(t *testing.T) {
	c := newSecondCampaign()
	tw := NewTown(Campaign{Towns: secondBook()})
	tw.resetTownStates(secondBook().catalog.Towns, c)
	if len(tw.others)+1 != len(secondBook().catalog.Towns) {
		t.Fatal("the bank campaign did not build every town row")
	}
	c.bank[770] = 29
	c.shop.records[2][0].Mask = 7
	for _, id := range []int{1, 2, 3, 2} {
		tw.selectTownState(id)
		state := tw.townState
		if state.id != id || state.bankOffers != c || state.bankOffers.bank[770] != 29 || state.bankOffers.shop.records[2][0].Mask != 7 {
			t.Fatalf("town %d lost its ID, bank or shop adapter", id)
		}
	}
	loaded := captureSecondCampaign(c).restore()
	tw.resetTownStates(secondBook().catalog.Towns, loaded)
	for id, state := range tw.others {
		if state.id != id || state.bankOffers != loaded || state.bankOffers == c {
			t.Fatalf("loaded town %d refers to another campaign", id)
		}
	}
}

package game

import (
	"bytes"
	"crypto/sha256"
	"reflect"
	"slices"
	"testing"

	"againrom/internal/firstgame"
	"againrom/pkg/mapload"
	"againrom/pkg/ui"
)

func TestReleaseCityGroupsRefundedReturnsUndoTheHire(t *testing.T) {
	firstgame.SkipOnSecondGame(t)
	seed, _ := hireForOrderTest(t, "Refund Groups")
	live, app, _ := cityGroupsColdApp(t, currentTownSave(t, seed))
	save := func(f *FrontEnd, a *ui.App) []byte {
		store := SaveStore{Dir: t.TempDir()}
		f.ConfigureSaveSeams(a, store, OriginalStore{}, nil)
		raw := cityRosterF2Save(t, a, store, "refund-groups")
		if err := a.HeadlessGameMenuAction("return"); err != nil || a.Screen() != ui.ScreenTown {
			t.Fatal("return from SAVE to town", err, a.Screen())
		}
		return raw
	}
	press := func(f *FrontEnd, hired bool) {
		s := f.TownScreen().(*townScreen)
		s.room = roomTavern
		if !slices.ContainsFunc(s.tavernMercenaries(), func(o TavernOffer) bool { return o.Type == 14 }) {
			t.Fatal("type-14 tavern control is absent")
		}
		s.tavernSelection = tavernCandidateKey{kind: tavernCandidateMercenary, id: 14}
		action := s.TownSurfaceClick(ui.TownSurfaceControl{Kind: ui.TownSurfaceControlButton, Index: tavernButtonHire}, false)
		if action.Msg != "" || f.Town.MercenaryHired(14) != hired {
			t.Fatalf("visible type-14 press: message %q, hired %v, want %v", action.Msg, f.Town.MercenaryHired(14), hired)
		}
	}
	before := save(live, app)
	groups := slices.Clone(live.Town.cityGroups)
	party := mapload.CloneParty(live.Carried)
	gold := live.Town.Gold()
	for pair := 0; pair < 64; pair++ {
		press(live, true)
		press(live, false)
	}
	after := save(live, app)
	t.Logf("64 visible refunded pairs: groups %d -> %d, party %d -> %d, gold %d -> %d, SAV %d -> %d bytes", len(groups), len(live.Town.cityGroups), len(party), len(live.Carried), gold, live.Town.Gold(), len(before), len(after))
	if !cityGroupsEqual(groups, live.Town.cityGroups) || !reflect.DeepEqual(party, live.Carried) || gold != live.Town.Gold() {
		t.Fatal("refunded pairs changed group payload/order, party or gold")
	}
	if !bytes.Equal(before, after) {
		t.Fatalf("refunded pairs changed F2 SAV: %x -> %x", sha256.Sum256(before), sha256.Sum256(after))
	}
	cold, coldApp, _ := cityGroupsColdApp(t, after)
	cityRosterSame(t, live, cold)
	if !cityGroupsEqual(groups, cold.Town.cityGroups) || !bytes.Equal(before, save(cold, coldApp)) {
		t.Fatal("cold LOAD changed groups or the F2 SAV after refunded pairs")
	}
	press(cold, true)
	hired, hiredApp, _ := cityGroupsColdApp(t, save(cold, coldApp))
	cityRosterSame(t, cold, hired)
	press(hired, false)
	returned := save(hired, hiredApp)
	if !cityGroupsEqual(groups, hired.Town.cityGroups) {
		t.Fatalf("SAVE/cold LOAD between hire and refunded return changed group payload/order: %v -> %v", liveGroupIDs(groups), liveGroupIDs(hired.Town.cityGroups))
	}
	cityRosterSame(t, live, hired)
	_, beforeActions, _ := cityProjectionWire(t, before)
	_, returnedActions, _ := cityProjectionWire(t, returned)
	t.Logf("cross-LOAD refunded return: groups %d -> %d, party %d, gold %d, object allocator %d -> %d", len(groups), len(hired.Town.cityGroups), len(hired.Carried), hired.Town.Gold(), beforeActions.Inventory.NextID, returnedActions.Inventory.NextID)
	final, finalApp, _ := cityGroupsColdApp(t, returned)
	cityRosterSame(t, live, final)
	if !cityGroupsEqual(groups, final.Town.cityGroups) || !bytes.Equal(returned, save(final, finalApp)) {
		t.Fatal("cold LOAD of the cross-LOAD refunded return changed groups or F2 SAV")
	}
	t.Logf("cold LOAD and cross-LOAD refund: %d groups, %d party members, gold %d, 64-pair SAV SHA-256 %x; cross-LOAD SAV SHA-256 %x", len(final.Town.cityGroups), len(final.Carried), final.Town.Gold(), sha256.Sum256(before), sha256.Sum256(returned))
}

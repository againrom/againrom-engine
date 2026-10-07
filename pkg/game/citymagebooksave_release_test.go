package game

import (
	"fmt"
	"testing"

	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

func cityMageIndex(t *testing.T, f *FrontEnd) int {
	t.Helper()
	first := -1
	for i, p := range f.Carried {
		if !p.Mage {
			continue
		}
		if first < 0 {
			first = i
		}
		if p.Book.State == sim.BookPresent {
			return i
		}
	}
	if first < 0 {
		t.Fatal("city has no mage")
	}
	return first
}

func cityMageSaved(t *testing.T, raw []byte, name string) sav.CityCharacter {
	t.Helper()
	file, err := sav.Open(raw)
	if err != nil {
		t.Fatal(err)
	}
	city, err := file.CityProvenance()
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range city.Roster() {
		if c.Name == name {
			return c
		}
	}
	t.Fatalf("saved city has no %q", name)
	return sav.CityCharacter{}
}

func cityMageMissionID(t *testing.T, f *FrontEnd, memberID string) sim.EntityID {
	t.Helper()
	for i, p := range f.live.mission.party {
		if p.ID == memberID {
			return f.live.mission.ids[i]
		}
	}
	t.Fatal("next mission omitted the mage", memberID)
	return 0
}

func cityMageBuyBook(t *testing.T, f *FrontEnd, app *ui.App, memberID string, item ShopItem) {
	t.Helper()
	shop := f.TownScreen().(*townScreen)
	if shop.room != roomShop {
		if err := app.HeadlessActivate("SHOP"); err != nil {
			t.Fatal(err)
		}
		for n := 0; shop.room == roomTalk && n < 32; n++ {
			if err := app.HeadlessActivate("dialogue"); err != nil {
				t.Fatal(err)
			}
		}
	}
	if shop.room != roomShop {
		t.Fatal("shop did not open", shop.room)
	}
	for n := 0; n < len(f.Carried); n++ {
		if m := shop.shopPartyMember(shop.shopMemberIndex()); m != nil && m.ID == memberID {
			break
		}
		cityPotionPointer(t, app, "picker_next", 0, "press", "release")
	}
	if m := shop.shopPartyMember(shop.shopMemberIndex()); m == nil || m.ID != memberID {
		t.Fatal("member picker did not reach the mage")
	}
	f.Shop.shelves[ShelfBooks] = []ShopItem{item}
	f.Town.gold = 100000
	cityPotionPointer(t, app, "shelf_pick", 3, "press", "release")
	gold := f.Town.Gold()
	packAt := releaseShopBuyToPack(t, app, f, item.Instance().Code)
	if err := app.HeadlessPointer("press", packAt.X, packAt.Y); err != nil {
		t.Fatal(err)
	}
	cityPotionPointer(t, app, "doll_box", 0, "move", "release")
	spell, _ := item.Instance().BookSpell()
	m := trainingPartyMember(t, f, memberID)
	if m.KnownSpells&(1<<spell) == 0 || m.Book.State == sim.BookPresent && m.Book.Slots[spell-1] == (sim.BookSpell{}) || f.Town.Gold() != gold-int(item.Price) {
		t.Fatalf("production book purchase did not teach spell %d: known %#x gold %d->%d chosen %d msg %q member %s", spell, m.KnownSpells, gold, f.Town.Gold(), shop.shopChosen, app.HeadlessMessage(), shop.shopPartyMember(shop.shopMemberIndex()).ID)
	}
}

func cityMageCastCells(t *testing.T, f *FrontEnd, app *ui.App, id sim.EntityID, spell uint32) map[sim.CellPoint]int {
	t.Helper()
	castOrderSelect(t, app, f.live, id)
	me, _ := f.live.entity(id)
	occupied := map[sim.CellPoint]bool{}
	for _, e := range f.live.world.Entities() {
		occupied[sim.CellPoint{X: e.X, Y: e.Y}] = true
	}
	cells := map[sim.CellPoint]int{}
	for py := 100; py < 560; py += 4 {
		for px := 160; px < 750; px += 4 {
			x, y, err := app.HeadlessDropCell(px, py)
			if err != nil {
				continue
			}
			at := sim.CellPoint{X: int32(x), Y: int32(y)}
			dx, dy := int(at.X-me.X), int(at.Y-me.Y)
			distance := max(dx, -dx, dy, -dy)
			if distance >= 2 && !occupied[at] && f.live.world.BookSpellCellRefusal(id, at.X, at.Y, spell) == "" {
				cells[at] = distance
			}
		}
	}
	return cells
}

func cityMageNearest(maps ...map[sim.CellPoint]int) (sim.CellPoint, bool) {
	var best sim.CellPoint
	bestDistance, found := 0, false
	for at, distance := range maps[0] {
		shared := true
		for _, other := range maps[1:] {
			if _, ok := other[at]; !ok {
				shared = false
			}
		}
		if shared && (!found || distance < bestDistance || distance == bestDistance && (at.Y < best.Y || at.Y == best.Y && at.X < best.X)) {
			best, bestDistance, found = at, distance, true
		}
	}
	return best, found
}

func cityMageCastInput(t *testing.T, f *FrontEnd, app *ui.App, id sim.EntityID, spell uint32, at sim.CellPoint) {
	t.Helper()
	castOrderSelect(t, app, f.live, id)
	if _, _, err := app.HeadlessSpellPoint(spell); err != nil {
		if err := app.HeadlessKey("book"); err != nil {
			t.Fatal(err)
		}
	}
	sx, sy, err := app.HeadlessSpellPoint(spell)
	if err != nil {
		t.Fatal(err)
	}
	for _, edge := range []string{"press", "release"} {
		if err := app.HeadlessPointer(edge, sx, sy); err != nil {
			t.Fatal(err)
		}
	}
	castOrderGroundClick(t, app, at)
	if len(f.live.pending) != 1 || f.live.pending[0].Kind != sim.KindCastAt || uint32(f.live.pending[0].Spell) != spell ||
		f.live.pending[0].X != at.X || f.live.pending[0].Y != at.Y {
		t.Fatalf("pointer input queued %+v, want spell %d at %v", f.live.pending, spell, at)
	}
}

func TestReleaseCityMageBookChangeF2SAVAndNextMissionCast(t *testing.T) {
	t.Run("original city", func(t *testing.T) {
		_, raw := groundCorpusFile(t, "2026-08-15/game0010.sav", "89cfca4c14e2b0bafd1fe28911badf246e213d83398874699947008739e8c5d4")
		f, app, store := cityPotionLoadApp(t, raw)
		cityMageBookWitness(t, f, app, store)
	})
	t.Run("current arrival", func(t *testing.T) {
		store := SaveStore{Dir: t.TempDir()}
		f, app := saveDialogArrivedTown(t, store.Dir)
		t.Cleanup(app.StopAudio)
		if f.originalCity != nil {
			t.Fatal("native arrival unexpectedly depends on a loaded city")
		}
		cityMageBookWitness(t, f, app, store)
	})
}

func cityMageBookWitness(t *testing.T, f *FrontEnd, app *ui.App, store SaveStore) {
	t.Helper()
	live, liveApp := f, app
	idx := cityMageIndex(t, f)
	member := f.Carried[idx]
	memberID, name := member.ID, member.Name
	before := cityRosterF2Save(t, app, store, "before-change")
	for n := 0; app.Screen() != ui.ScreenTown && n < 4; n++ {
		if err := app.HeadlessKey("escape"); err != nil {
			t.Fatal(err)
		}
	}
	if app.Screen() != ui.ScreenTown {
		t.Fatal("F2 SAVE did not return to the city", app.Screen())
	}

	var candidates []ShopItem
	for _, item := range shopBookPool(f.Table, 50000) {
		if spell, ok := item.Instance().BookSpell(); ok && spell != 26 && member.KnownSpells&(1<<spell) == 0 {
			candidates = append(candidates, item)
		}
	}
	if len(candidates) == 0 {
		t.Fatal("the mage already knows every merchant spell")
	}
	probe, probeApp, _ := cityPotionLoadApp(t, before)
	for _, item := range candidates {
		cityMageBuyBook(t, probe, probeApp, memberID, item)
	}
	if err := probeApp.OpenMission(probe.MissionOpener(probe.Town.Chapter())); err != nil {
		t.Fatal(err)
	}
	var chosen ShopItem
	var spell uint32
	probeID := cityMageMissionID(t, probe, memberID)
	for _, item := range candidates {
		s, _ := item.Instance().BookSpell()
		if _, ok := cityMageNearest(cityMageCastCells(t, probe, probeApp, probeID, uint32(s))); ok {
			chosen, spell = item, uint32(s)
			break
		}
	}
	if spell == 0 {
		t.Fatal("no merchant spell admits a ground cast near the mage in the next mission")
	}

	cityMageBuyBook(t, f, app, memberID, chosen)
	s := f.TownScreen().(*townScreen)
	gold, slot, price := f.Town.Gold(), 0, 0
	for i := 1; i <= 5; i++ {
		if p := memberSchoolPrice(trainingPartyMember(t, f, memberID), i); p > 0 && p <= gold && (slot == 0 || p < price) {
			slot, price = i, p
		}
	}
	if slot == 0 {
		t.Fatal("no affordable mage school slot")
	}
	s.room, s.schoolCell, s.shopMember = roomSchool, slot+4, idx
	if err := app.HeadlessActivate(f.Words.SchoolTrain); err != nil || f.Town.Gold() != gold-price {
		t.Fatal("App Train", err, app.HeadlessMessage())
	}
	changed := trainingPartyMember(t, f, memberID)
	if changed.KnownSpells&(1<<spell) == 0 {
		t.Fatal("training dropped the learned spell")
	}

	for cycle := 0; cycle < 2; cycle++ {
		saved := cityRosterF2Save(t, app, store, fmt.Sprintf("book-%d", cycle))
		mage := cityMageSaved(t, saved, name)
		if changed.Book.State == sim.BookPresent {
			requireSavedBook(t, mage.KnownSpells, mage.Spells, changed.KnownSpells, changed.Book)
		} else if mage.KnownSpells != changed.KnownSpells {
			t.Fatalf("saved membership %#x, live %#x", mage.KnownSpells, changed.KnownSpells)
		}
		if mage.KnownSpells&(1<<spell) == 0 {
			t.Fatal("F2 SAVE omitted the learned spell", cycle)
		}
		if got := trainingPartyMember(t, live, memberID); got.Book != changed.Book || got.KnownSpells != changed.KnownSpells {
			t.Fatal("city SAVE changed the uninterrupted book")
		}
		cold, coldApp, coldStore := cityPotionLoadApp(t, saved)
		cityRosterSame(t, live, cold)
		if got := trainingPartyMember(t, cold, memberID); got.Book != changed.Book || got.KnownSpells != changed.KnownSpells {
			t.Fatalf("cold LOAD %d book %+v known %#x, live %+v %#x", cycle, got.Book, got.KnownSpells, changed.Book, changed.KnownSpells)
		}
		f, app, store = cold, coldApp, coldStore
	}

	for i, target := range []*FrontEnd{live, f} {
		if err := []*ui.App{liveApp, app}[i].OpenMission(target.MissionOpener(target.Town.Chapter())); err != nil {
			t.Fatal(err)
		}
	}
	liveID, coldID := cityMageMissionID(t, live, memberID), cityMageMissionID(t, f, memberID)
	cityPotionMissionEqual(t, live.live.world, f.live.world, 0)
	at, ok := cityMageNearest(cityMageCastCells(t, live, liveApp, liveID, spell), cityMageCastCells(t, f, app, coldID, spell))
	if !ok {
		t.Fatal("next mission admits no visible ground cast of the learned spell in both sessions")
	}
	mana, _ := f.live.entity(coldID)
	cityMageCastInput(t, live, liveApp, liveID, spell, at)
	cityMageCastInput(t, f, app, coldID, spell, at)
	landed := castOrderLockstep(t, live, f, 256, func(h *FrontEnd, events []sim.CastEvent) bool {
		for _, ev := range events {
			if uint32(ev.Spell) == spell && ev.AtCell && ev.ToX == at.X && ev.ToY == at.Y {
				return true
			}
		}
		return false
	})
	if after, _ := f.live.entity(coldID); after.Mana >= mana.Mana {
		t.Fatalf("cold mission cast spent no mana: %d -> %d", mana.Mana, after.Mana)
	}

	old, oldApp, _ := cityPotionLoadApp(t, before)
	if trainingPartyMember(t, old, memberID).KnownSpells&(1<<spell) != 0 {
		t.Fatal("pre-change SAV already knows the spell: the witness cannot discriminate")
	}
	if err := oldApp.OpenMission(old.MissionOpener(old.Town.Chapter())); err != nil {
		t.Fatal(err)
	}
	if old.live.world.BookSpellCellRefusal(cityMageMissionID(t, old, memberID), at.X, at.Y, spell) == "" {
		t.Fatal("loss control: the unchanged book admits the learned spell")
	}
	t.Logf("%s learned spell %d and trained school slot %d (gold %d->%d); two F2 city SAV/cold LOAD cycles keep the book; next mission cast through book and ground click landed %d ticks later with equal World hashes; pre-change SAV refuses the cast", name, spell, slot, gold, f.Town.Gold(), landed)
}

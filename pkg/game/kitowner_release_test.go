package game

import (
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"againrom/pkg/formats/sav"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

func kitOwnerEdition(f *FrontEnd) string {
	if f.textSelector() == 1 {
		return "RU"
	}
	return "EN"
}

func kitOwnerOutput(t *testing.T, edition string) string {
	t.Helper()
	dir := os.Getenv("AGAINROM_OWNER_KIT_OUT")
	if dir == "" {
		dir = t.TempDir()
	}
	out := filepath.Join(dir, edition)
	if err := os.MkdirAll(out, 0o755); err != nil {
		t.Fatal(err)
	}
	return out
}

func kitOwnerWrite(t *testing.T, dir, slot string, raw []byte) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, slot), raw, 0o644); err != nil {
		t.Fatal(err)
	}
}

func kitOwnerCityLabel(edition string) string {
	if edition == "RU" {
		return "9608 Город v3 RU"
	}
	return "9607 город v3 EN"
}

func kitOwnerCityFile(edition string) string {
	if edition == "RU" {
		return "game9608.sav"
	}
	return "game9607.sav"
}

func kitOwnerShop(t *testing.T, f *FrontEnd, app *ui.App) *townScreen {
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
	return shop
}

func kitOwnerDrinkRegeneration(t *testing.T, f *FrontEnd, app *ui.App) *sim.ActiveEffect {
	t.Helper()
	shop := kitOwnerShop(t, f, app)
	shop.shopMember = 0
	member := shop.shopPartyMember(shop.shopMemberIndex())
	if member == nil || member.PotionEffect != nil {
		t.Fatal("fixture requires a member without a potion")
	}
	var item ShopItem
	found := false
	for _, s := range shopPotionStock(f.Table, rand.New(rand.NewSource(1))) {
		if f.Table.MagicItems.EntryName(int(s.Code&0xff)) == "Potion Health Regeneration" {
			item, found = s, true
		}
	}
	if !found {
		t.Fatal("installed shop does not sell the regeneration potion")
	}
	f.Shop.shelves[ShelfBooks] = []ShopItem{item}
	f.Town.gold = 100000
	cityPotionPointer(t, app, "shelf_pick", 3, "press", "release")
	gold := f.Town.Gold()
	packAt := releaseShopBuyToPack(t, app, f, uint16(item.Code))
	if err := app.HeadlessPointer("press", packAt.X, packAt.Y); err != nil {
		t.Fatal(err)
	}
	cityPotionPointer(t, app, "doll_box", 0, "move", "release")
	member = shop.shopPartyMember(shop.shopMemberIndex())
	e := member.PotionEffect
	if e == nil || e.Kind != sim.EffectHealthRegeneration || e.Magnitude != 100 || e.Remaining != 960 || f.Town.Gold() != gold-int(item.Price) {
		t.Fatalf("potion use left %+v, gold %d -> %d", e, gold, f.Town.Gold())
	}
	return e
}

func kitOwnerLearnSpell(t *testing.T, f *FrontEnd, app *ui.App) (mapload.PartyMember, uint32) {
	t.Helper()
	idx := cityMageIndex(t, f)
	member := f.Carried[idx]
	var chosen ShopItem
	var spell uint32
	for _, item := range shopBookPool(f.Table, 50000) {
		if s, ok := item.Instance().BookSpell(); ok && s != 26 && member.KnownSpells&(1<<s) == 0 {
			chosen, spell = item, uint32(s)
			break
		}
	}
	if spell == 0 {
		t.Fatal("the mage already knows every merchant spell")
	}
	s := f.TownScreen().(*townScreen)
	gold, slot, price := f.Town.Gold(), 0, 0
	for i := 1; i <= 5; i++ {
		if p := memberSchoolPrice(trainingPartyMember(t, f, member.ID), i); p > 0 && p <= gold && (slot == 0 || p < price) {
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
	if err := app.HeadlessActivate(f.Words.SchoolExit); err != nil {
		t.Fatal(err)
	}
	cityMageBuyBook(t, f, app, member.ID, chosen)
	changed := trainingPartyMember(t, f, member.ID)
	if changed.KnownSpells&(1<<spell) == 0 {
		t.Fatal("training dropped the learned spell")
	}
	return changed, spell
}

func kitOwnerWalkToChapter(t *testing.T, chapter int) *FrontEnd {
	t.Helper()
	f := releaseFront(t)
	_, screen := reachabilityWalkArrive(t, f, "Kit Hero")
	f.Town.gold = 5_000_000
	for _, m := range reachabilityWalkStates() {
		if m >= chapter {
			break
		}
		screen.markWorldSelected(m)
		if _, ok := f.Town.Won(m); !ok {
			t.Fatalf("Won(%d) refused", m)
		}
		f.addChapterCompanions(f.Town.Chapter())
	}
	if f.Town.Chapter() != chapter {
		t.Fatalf("walk reached chapter %d, want %d", f.Town.Chapter(), chapter)
	}
	return f
}

func kitOwnerBuildCity(t *testing.T) ([]byte, string) {
	lineage := kitOwnerWalkToChapter(t, 120)
	edition := kitOwnerEdition(lineage)
	seed := currentTownSave(t, lineage)
	f, app, store := cityGroupsColdApp(t, seed)
	t.Logf("%s chapter %d party %d gold %d", edition, f.Town.Chapter(), len(f.Carried), f.Town.Gold())

	potion := kitOwnerDrinkRegeneration(t, f, app)
	mage, spell := kitOwnerLearnSpell(t, f, app)

	f.Town.gold = 3_000_000
	s := f.TownScreen().(*townScreen)
	s.room = roomTavern
	offered := map[int]int{}
	for _, o := range s.tavernMercenaries() {
		offered[o.Type] = o.Count
	}
	var plan []int
	for _, typ := range []int{6, 14, 13, 2, 6, 2, 6, 2} {
		if offered[typ] == 0 {
			t.Fatalf("chapter %d tavern does not offer type %d: %v", f.Town.Chapter(), typ, offered)
		}
		cityGroupsToggle(t, f, typ)
		plan = append(plan, typ)
	}
	if len(f.Carried) <= 12 {
		t.Fatalf("party holds %d members", len(f.Carried))
	}
	hired := map[int]int{}
	for _, m := range f.Carried {
		if m.Hired() {
			hired[int(m.MercenaryType)]++
		}
	}
	for _, typ := range []int{6, 14, 13, 2} {
		if hired[typ] == 0 || hired[typ] != offered[typ] {
			t.Fatalf("type %d: %d hired of %d offered (%v)", typ, hired[typ], offered[typ], hired)
		}
	}
	label := kitOwnerCityLabel(edition)
	raw := cityRosterF2Save(t, app, store, label)
	t.Logf("BUILD %s label %q party %d hired %v plan %v potion %+v mage %s learned spell %d bytes %d", edition, label, len(f.Carried), hired, plan, *potion, mage.Name, spell, len(raw))
	return raw, edition
}

func TestKitOwnerCityM10(t *testing.T) {
	raw, edition := kitOwnerBuildCity(t)
	kitOwnerWrite(t, kitOwnerOutput(t, edition), kitOwnerCityFile(edition), raw)
	loaded := strings.Join(kitOwnerReceive(t, raw), "; ")
	for _, want := range []string{"party 14 members, hired: type 2 x1, type 6 x4, type 13 x4, type 14 x3", "potion Kit Hero kind 12 magnitude 100 remaining 960", "knows spells [1 2 6 12 18]"} {
		if !strings.Contains(loaded, want) {
			t.Fatalf("cold LOAD of the city file lacks %q: %s", want, loaded)
		}
	}
}

func kitOwnerObserve(f *FrontEnd) []string {
	var out []string
	add := func(format string, args ...any) { out = append(out, fmt.Sprintf(format, args...)) }
	party := f.Carried
	if f.live != nil && f.live.mission != nil {
		party = f.live.mission.party
		add("place mission %d", f.liveMission)
	} else {
		add("place city chapter %d", f.Town.Chapter())
		add("gold %d", f.Town.Gold())
	}
	hired := map[int]int{}
	for _, m := range party {
		if m.Hired() {
			hired[int(m.MercenaryType)]++
		}
	}
	types := make([]int, 0, len(hired))
	for typ := range hired {
		types = append(types, typ)
	}
	slices.Sort(types)
	var hireText []string
	for _, typ := range types {
		hireText = append(hireText, fmt.Sprintf("type %d x%d", typ, hired[typ]))
	}
	add("party %d members, hired: %s", len(party), strings.Join(hireText, ", "))
	for _, m := range party {
		if m.PotionEffect != nil {
			add("potion %s kind %d magnitude %d remaining %d", m.Name, m.PotionEffect.Kind, m.PotionEffect.Magnitude, m.PotionEffect.Remaining)
		}
		if m.Mage {
			var spells []int
			for id := 1; id < 32; id++ {
				if m.KnownSpells&(1<<id) != 0 {
					spells = append(spells, id)
				}
			}
			add("mage %s knows spells %v", m.Name, spells)
		}
	}
	if f.live != nil && f.live.world != nil {
		w := f.live.world
		areas, _ := w.NativeAreaSaveStates()
		drivers := w.SavedWorldEffectDrivers()
		driverProjectiles := 0
		if drivers != nil {
			driverProjectiles = len(drivers.Projectiles)
		}
		bodies := currentBodies(w)
		add("sacks %d", len(w.Sacks()))
		add("bodies %d", len(bodies))
		add("area effects %d", len(areas))
		add("spell deliveries in flight %d", len(w.NativeSpellDeliverySaveStates()))
		add("projectile records %d, projectile drivers %d", len(w.SavedProjectiles().Items), driverProjectiles)
		for _, p := range w.SavedProjectiles().Items {
			add("projectile %d picture %d at %d,%d dir %d phase %d action %d target %d aim %d,%d actionphase %d actionsegments %d", p.ID, p.Picture, p.X, p.Y, p.Dir, p.Phase, p.Action, p.ActionTarget, p.ActionX, p.ActionY, p.ActionPhase, p.ActionSegments)
		}
	}
	return out
}

func kitOwnerCombatLabel(edition string) string {
	if edition == "RU" {
		return "9604 Бой RU"
	}
	return "9603 Ёлка EN"
}

func kitOwnerCombatFile(edition string) string {
	if edition == "RU" {
		return "game9604.sav"
	}
	return "game9603.sav"
}

func kitOwnerCastCell(t *testing.T, f *FrontEnd, caster sim.EntityID, spell uint32, minDistance int) sim.CellPoint {
	t.Helper()
	me, _ := f.live.entity(caster)
	for d := minDistance; d <= 8; d++ {
		for dy := -d; dy <= d; dy++ {
			for dx := -d; dx <= d; dx++ {
				if max(dx, -dx, dy, -dy) != d {
					continue
				}
				at := sim.CellPoint{X: me.X + int32(dx), Y: me.Y + int32(dy)}
				if f.live.world.BookSpellCellRefusal(caster, at.X, at.Y, spell) == "" {
					return at
				}
			}
		}
	}
	t.Fatalf("no cell admits spell %d near %d,%d", spell, me.X, me.Y)
	return sim.CellPoint{}
}

func kitOwnerBuildCombat(t *testing.T) ([]byte, string) {
	dir := t.TempDir()
	f, app := saveDialogArrivedTown(t, dir)
	t.Cleanup(app.StopAudio)
	edition := kitOwnerEdition(f)
	store := SaveStore{Dir: dir}
	mageIdx := cityMageIndex(t, f)
	mage := f.Carried[mageIdx]
	learned := map[uint32]bool{}
	for _, item := range shopBookPool(f.Table, 50000) {
		if s, ok := item.Instance().BookSpell(); ok && (s == 2 || s == 3) && mage.KnownSpells&(1<<s) == 0 {
			cityMageBuyBook(t, f, app, mage.ID, item)
			learned[uint32(s)] = true
		}
	}
	if !learned[2] || !learned[3] {
		t.Fatalf("merchant books for spells 2 and 3 not bought: %v", learned)
	}
	if err := app.OpenMission(f.MissionOpener(f.Town.Chapter())); err != nil {
		t.Fatal(err)
	}
	mw := f.live
	w := mw.world
	hero, mageID := mw.mission.ids[0], mw.mission.ids[mageIdx]
	const foe sim.EntityID = 0
	mw.pending = append(mw.pending, sim.Attack(hero, foe))
	for tick := 0; tick < 400; tick++ {
		mw.tick()
		if e, _ := mw.entity(foe); !e.Alive() && tick > 30 {
			break
		}
	}
	if e, _ := mw.entity(foe); e.Alive() {
		t.Fatal("the hero did not kill the creature")
	}
	if len(currentBodies(w)) == 0 {
		t.Fatal("no body after the kill")
	}

	h, _ := mw.entity(hero)
	sacksBefore := len(w.Sacks())
	stacks, _ := w.CarriedStacks(hero)
	if len(stacks) == 0 {
		t.Fatal("the hero carries nothing to drop")
	}
	mw.pending = append(mw.pending, sim.DropCarried(hero, 0, sim.CellPoint{X: h.X, Y: h.Y}))
	mw.tick()
	if groundAt(w.Sacks(), h.X, h.Y) == nil || len(w.Sacks()) != sacksBefore+1 {
		t.Fatal("the dropped item made no Sack")
	}

	standing := kitOwnerCastCell(t, f, mageID, 3, 2)
	mw.pending = append(mw.pending, sim.CastAt(mageID, 3, standing))
	for tick := 0; tick < 24; tick++ {
		mw.tick()
	}
	if areas, _ := w.NativeAreaSaveStates(); len(areas) == 0 {
		t.Fatal("no standing area after the first cast")
	}
	flying := kitOwnerCastCell(t, f, mageID, 2, 5)
	mw.pending = append(mw.pending, sim.CastAt(mageID, 2, flying))
	for tick := 0; len(w.NativeSpellDeliverySaveStates()) == 0; tick++ {
		if tick > 60 {
			t.Fatal("the second cast never entered flight")
		}
		mw.tick()
	}
	areas, _ := w.NativeAreaSaveStates()
	if len(areas) == 0 {
		t.Fatal("the standing area ended before the cast was in flight")
	}
	for _, line := range kitOwnerObserve(f) {
		t.Log("LIVE", line)
	}
	label := kitOwnerCombatLabel(edition)
	raw := cityRosterF2Save(t, app, store, label)
	t.Logf("BUILD %s label %q bytes %d", edition, label, len(raw))
	return raw, edition
}

func TestKitOwnerCombatM10(t *testing.T) {
	raw, edition := kitOwnerBuildCombat(t)
	kitOwnerWrite(t, kitOwnerOutput(t, edition), kitOwnerCombatFile(edition), raw)
	loaded := kitOwnerReceive(t, raw)
	if len(loaded) == 0 {
		t.Fatal("cold LOAD of the combat file observed nothing")
	}
	text := strings.Join(loaded, "; ")
	for _, want := range []string{"sacks 5", "area effects 1", "spell deliveries in flight 1", "after: spell deliveries in flight 0", "after: area effects 1"} {
		if !strings.Contains(text, want) {
			t.Fatalf("cold LOAD of the combat file lacks %q: %s", want, text)
		}
	}
}

func TestKitOwnerReceiveM10(t *testing.T) {
	path := os.Getenv("AGAINROM_OWNER_KIT_LOAD")
	if path == "" {
		raw, _ := kitOwnerBuildCity(t)
		kitOwnerReceive(t, raw)
		return
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	kitOwnerReceive(t, raw)
}

func kitOwnerReceive(t *testing.T, raw []byte) []string {
	t.Helper()
	doc, err := sav.DecodeDocumentData(raw)
	if err != nil {
		t.Fatalf("RECEIVE cannot decode the file: %v", err)
	}
	if file, err := sav.Open(raw); err == nil {
		t.Logf("RECEIVE label bytes % x", file.Label)
	}
	say := func(prefix string, lines []string) {
		for _, l := range lines {
			t.Logf("RECEIVE %s %s", prefix, l)
		}
	}
	if doc.World == nil {
		f, app, store := cityGroupsColdApp(t, raw)
		_, list, _ := f.SaveSeams(store, OriginalStore{}, nil)
		for _, row := range list() {
			t.Logf("RECEIVE chooser label %q", row.Label)
		}
		loaded := kitOwnerObserve(f)
		say("loaded", loaded)
		shop := kitOwnerShop(t, f, app)
		t.Logf("RECEIVE next action: shop opened, room %v", shop.room)
		s := f.TownScreen().(*townScreen)
		s.room = roomTavern
		for _, o := range s.tavernMercenaries() {
			t.Logf("RECEIVE tavern offer type %d count %d hired %v", o.Type, o.Count, o.Hired)
		}
		if err := app.OpenMission(f.MissionOpener(f.Town.Chapter())); err != nil {
			t.Fatalf("RECEIVE next mission: %v", err)
		}
		before := f.live.world.Hash()
		for range 8 {
			f.live.tick()
		}
		t.Logf("RECEIVE next action: mission %d entered with %d members, 8 ticks run, hash changed %v", f.Town.Chapter(), len(f.live.mission.ids), f.live.world.Hash() != before)
		return loaded
	}
	f := releaseFront(t)
	app, _ := openOriginalSAVApp(t, f, raw, "kit.sav")
	t.Cleanup(app.StopAudio)
	if f.live == nil || f.live.world == nil {
		t.Fatal("RECEIVE mission LOAD opened no world")
	}
	loaded := kitOwnerObserve(f)
	say("loaded", loaded)
	var target sim.EntityID
	hasTarget := false
	if d := f.live.world.SavedWorldEffectDrivers(); d != nil && len(d.Projectiles) > 0 {
		target, hasTarget = d.Projectiles[0].Target, d.Projectiles[0].HasTarget
	}
	inFlight := len(f.live.world.SavedProjectiles().Items) > 0
	for tick := 1; tick <= 96; tick++ {
		f.live.tick()
		items := f.live.world.SavedProjectiles().Items
		if len(items) > 0 {
			p := items[0]
			t.Logf("RECEIVE projectile trace tick %d: %d at %d,%d phase %d actionphase %d actionsegments %d", tick, p.ID, p.X, p.Y, p.Phase, p.ActionPhase, p.ActionSegments)
		} else if inFlight {
			inFlight = false
			if e, ok := f.live.entity(target); hasTarget && ok {
				t.Logf("RECEIVE projectile landed after %d ticks; target entity %d at %d,%d hp %d alive %v", tick, target, e.X, e.Y, e.HP, e.Alive())
			} else {
				t.Logf("RECEIVE projectile landed after %d ticks; target %d has no live entity", tick, target)
			}
		}
	}
	after := kitOwnerObserve(f)
	say("after 96 ticks", after)
	for _, l := range after {
		loaded = append(loaded, "after: "+l)
	}
	return loaded
}

// TestKitOwnerResaveMission120Release loads the original game's mission resave
// of the city kit, in which the kit's siege hire is a Human actor whose
// definition row is 0, and runs the mission on.
func TestKitOwnerResaveMission120Release(t *testing.T) {
	path := os.Getenv("AGAINROM_OWNER_KIT_RESAVE_MISSION")
	if path == "" {
		t.Skip("no AGAINROM_OWNER_KIT_RESAVE_MISSION: the owner's mission resave is not committed")
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	f := releaseFront(t)
	app, _ := openOriginalSAVApp(t, f, raw, "kit.sav")
	t.Cleanup(app.StopAudio)
	if f.live == nil || f.live.world == nil || f.liveMission != 120 {
		t.Fatalf("LOAD opened mission %d", f.liveMission)
	}
	rowZero := 0
	for _, e := range f.live.world.Entities() {
		if b := e.SourceBinding; b.ActorClass() == 2 && b.DefinitionRow() == 0 && e.Owner == 1 {
			rowZero++
		}
	}
	if rowZero != 1 {
		t.Fatalf("the world holds %d Human actors with definition row 0 owned by the player, want 1", rowZero)
	}
	before := f.live.world.Hash()
	for range 600 {
		f.live.tick()
	}
	if f.live.world.Hash() == before {
		t.Fatal("600 ticks left the world unchanged")
	}
	// 14: LOAD also reads the original's own Ballista Unit
	// as a hire, beside the legacy row-0 Human the kit wrote (DIV-1709).
	if got := len(f.live.mission.party); got != 14 {
		t.Fatalf("party holds %d members after 600 ticks, want 14", got)
	}
	t.Logf("RECEIVE mission 120 loaded, row-0 Human actors %d, 600 ticks run, party %d", rowZero, len(f.live.mission.party))
}

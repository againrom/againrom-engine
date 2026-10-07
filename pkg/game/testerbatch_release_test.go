package game

import (
	"image"
	"testing"

	"againrom/pkg/mapload"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

// missionRel reads one row of the live relation matrix as a digit string.
func missionRel(w *sim.World, from uint32, slots uint32) string {
	rel := w.Relations()
	row := make([]byte, 0, slots)
	for to := uint32(1); to <= slots; to++ {
		row = append(row, '0'+rel.Byte(from, to))
	}
	return string(row)
}

func missionTicks(mw *mapWorld, n int) {
	for i := 0; i < n; i++ {
		mw.tick()
		mw.mission.open = false
	}
}

// A marker cached by selecting a mission keeps painting after the mission is
// won and has lost its scroll (TOWN-123, MISSION-MAP-073).
func TestReleaseWorldMapMarkerOutlivesItsWonMission(t *testing.T) {
	f := releaseFront(t)
	_, screen := reachabilityWalkArrive(t, f, "Marker traveler")
	for _, m := range []int{10, 20, 30, 40, 50, 60} {
		f.Town.CollectDocuments(m)
		if _, ok := f.Town.Won(m); !ok {
			t.Fatalf("mission %d refused its win", m)
		}
		takeCityOffers(t, f)
	}
	f.arriveInTown()
	screen.Choose(3)
	view := screen.WorldMapView()
	index := worldMarkerMissionIndex(t, view, 70)
	if view.Missions[index].Marker != nil {
		t.Fatal("mission 70 painted its marker before selection")
	}
	card := ui.WorldMapCardRect(index)
	screen.WorldMapClick(image.Pt((card.Min.X+card.Max.X)/2, (card.Min.Y+card.Max.Y)/2))
	view = screen.WorldMapView()
	if view.Missions[index].Marker == nil {
		t.Fatal("selecting mission 70 cached no marker")
	}
	if b := view.Missions[index].Marker.Bounds(); b.Dx() != 640 || b.Dy() != 480 {
		t.Fatalf("mission 70 marker is %v, want a 640x480 overlay", b)
	}

	if _, ok := f.Town.Won(70); !ok {
		t.Fatal("mission 70 refused its win")
	}
	takeCityOffers(t, f)
	screen.beginWorldMapReturn(70)
	view = screen.WorldMapView()
	for _, m := range view.Missions {
		if m.Number == 70 {
			t.Fatal("the completed mission still has a scroll; the control no longer separates the marker from the scroll")
		}
	}
	if len(view.Markers) != 1 {
		t.Fatalf("after the win the map holds %d cached markers, want the monastery's one", len(view.Markers))
	}
	with := ui.ComposeWorldMap(view)
	view.Markers = nil
	without := ui.ComposeWorldMap(view)
	data := f.worldMapAssets().data
	object := data.Objects[data.Missions[70]]
	area := object.Region.Inset(-30)
	changed := 0
	for y := 0; y < 480; y++ {
		for x := 0; x < 640; x++ {
			if with.RGBAAt(x, y) == without.RGBAAt(x, y) {
				continue
			}
			changed++
			if !image.Pt(x, y).In(area) {
				t.Fatalf("marker pixel (%d,%d) lies outside mission 70's map rectangle %v", x, y, object.Region)
			}
		}
	}
	if changed == 0 {
		t.Fatal("the cached marker of the won mission painted nothing")
	}
}

type kargallasFix struct {
	mw               *mapWorld
	hero, renie, drg sim.EntityID
}

// openKargallas opens mission 90 with a male starting hero and Reniesta, so
// hero ordinal 2 resolves to her.
func openKargallas(t *testing.T) kargallasFix {
	t.Helper()
	f := releaseFront(t)
	f.SetDeterministicFrames(true)
	f.Carried = f.ChargenParty(ui.ChargenResult{Name: "Campaign witness", Choices: []int{0, 0, 3}, Stats: []int{31, 27, 24, 29}})
	f.arriveInTown()
	for _, n := range f.Campaign.Value().Main {
		if n < 90 {
			f.Town.Won(n)
		}
	}
	f.addChapterCompanions(30)
	takeCampaignOffer(t, f, 90)
	if err := f.App("kargallas").OpenMission(f.MissionOpenerWith(90, f.NextParty())); err != nil {
		t.Fatal(err)
	}
	k := kargallasFix{mw: f.live}
	m := k.mw.mission.state.Map
	for i, p := range k.mw.mission.party {
		if p.CompanionNPC == 22 {
			k.renie = mapload.PartyEntity(m, i)
		}
		if p.StartingHero {
			k.hero = mapload.PartyEntity(m, i)
		}
	}
	if k.renie == 0 || k.hero == 0 {
		t.Fatal("the party lacks Reniesta or the starting hero")
	}
	k.drg = mapload.ScriptUnits(m, k.mw.mission.party)[187]
	return k
}

func (k kargallasFix) holdsItem(id sim.EntityID) bool {
	stacks, _ := k.mw.world.CarriedStacks(id)
	for _, s := range stacks {
		if s.Code == 0xe22 {
			return true
		}
	}
	return false
}

// pickUpItem puts the character beside the sack at (22,109) and orders the
// ordinary pick-up.
func (k kargallasFix) pickUpItem(t *testing.T, id sim.EntityID) {
	t.Helper()
	if err := k.mw.world.HeadlessPlace(id, 22, 108); err != nil {
		t.Fatal(err)
	}
	k.mw.grab(uint32(id), 22, 109, true)
	for i := 0; i < 300 && !k.holdsItem(id); i++ {
		k.mw.tick()
	}
	if !k.holdsItem(id) {
		t.Fatal("the character did not take the sack's item")
	}
}

// ALM-CATAPULT-213: owner 8 holds 0 or 2 toward every player, and nothing
// orders it to act.
func TestReleaseKargallasCatapultIsNonHostileToEveryPlayer(t *testing.T) {
	k := openKargallas(t)
	w := k.mw.world
	if got, want := missionRel(w, 8, 9), "022222222"; got != want {
		t.Fatalf("Catapult row = %s, want %s", got, want)
	}
	if got := missionRel(w, 1, 9); got[7] != '0' {
		t.Fatalf("Self row %s is hostile toward the Catapult", got)
	}
	var catapult sim.EntityID
	for _, e := range w.Entities() {
		if e.Owner == 8 && e.X == 102 && e.Y == 14 {
			catapult = e.ID
		}
	}
	if catapult == 0 {
		t.Fatal("no catapult at (102,14) owned by player 8")
	}
	missionTicks(k.mw, 1500)
	if e, _ := w.Entity(catapult); e.HasTarget {
		t.Fatalf("the Catapult acquired a target at (%d,%d) with no hostile relation and no order", e.TargetX, e.TargetY)
	}
}

// TRIG-DRAGON-079: the dragon's curse ends only when ordinal 2 holds the item
// within 6; the win removes the item from ordinals 1, 2, 3 and 5 first.
func TestReleaseKargallasDragonCurseFollowsItsTrigger(t *testing.T) {
	for _, tc := range []struct {
		name          string
		pickBeforeWin bool
		win           bool
		pickAfterWin  bool
		wantCursed    bool
		wantItemKept  bool
	}{
		{name: "holder approaches before any win", pickBeforeWin: true, wantCursed: false, wantItemKept: true},
		{name: "siege won while holding, then approach", pickBeforeWin: true, win: true, wantCursed: true, wantItemKept: false},
		{name: "siege won, item taken afterwards, then approach", win: true, pickAfterWin: true, wantCursed: false, wantItemKept: true},
		{name: "approach without the item", wantCursed: true, wantItemKept: false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			k := openKargallas(t)
			w := k.mw.world
			missionTicks(k.mw, 40)
			if !w.HasEffectSpell(k.drg, 20) {
				t.Fatal("the dragon is not cursed at the start")
			}
			if tc.pickBeforeWin {
				k.pickUpItem(t, k.renie)
			}
			if tc.win {
				if _, err := w.HeadlessKillPlayer(7); err != nil {
					t.Fatal(err)
				}
				missionTicks(k.mw, 200)
				if w.Outcome() != sim.OutcomeWon {
					t.Fatalf("killing every owner-7 unit left the outcome %v", w.Outcome())
				}
				if tc.pickBeforeWin && k.holdsItem(k.renie) {
					t.Fatal("the win left the item on ordinal 2")
				}
			}
			if tc.pickAfterWin {
				k.pickUpItem(t, k.renie)
			}
			if err := w.HeadlessPlace(k.renie, 19, 17); err != nil {
				t.Fatal(err)
			}
			missionTicks(k.mw, 100)
			if got := w.HasEffectSpell(k.drg, 20); got != tc.wantCursed {
				t.Fatalf("dragon cursed = %v, want %v", got, tc.wantCursed)
			}
			if got := k.holdsItem(k.renie); got != tc.wantItemKept {
				t.Fatalf("Reniesta holds the item = %v, want %v", got, tc.wantItemKept)
			}
		})
	}
}

// MISSION-DROP-074: unit 119 draws its tier-3 Troll row.
func TestReleaseMission111TrollCarriesTheTierThreeDropRow(t *testing.T) {
	f := openCampaignMission(t, 111)
	mw := f.live
	troll := mapload.ScriptUnits(mw.mission.state.Map, mw.mission.party)[119]
	e, ok := mw.world.Entity(troll)
	if !ok {
		t.Fatal("unit 119 is absent")
	}
	if e.TypeID <= 0x40 || e.GoldChance != 90 || e.TreasureMin != 24200 || e.TreasureMax != 36300 {
		t.Fatalf("unit 119 type %#x chance %d purse %d+U[0,%d], want a type above 0x40, 90, 24200+U[0,36300]",
			e.TypeID, e.GoldChance, e.TreasureMin, e.TreasureMax)
	}
	total := func() (sum uint64) {
		for _, s := range mw.world.Sacks() {
			sum += uint64(s.Gold)
		}
		return
	}
	before := total()
	if err := mw.world.HeadlessKill(troll); err != nil {
		t.Fatal(err)
	}
	missionTicks(mw, 64)
	if got := total() - before; got != 0 && (got < 24200 || got > 60500) {
		t.Fatalf("the troll's death added %d gold, want 0 or 24200..60500", got)
	}
}

// TRIG-BRIGAND-080: the bridge brigands are hostile at load, no script orders
// one, and the bridge talk raises as a walker comes within 3 of the bridge
// cell, within one script pass of the brigands' own acquisition.
func TestReleaseBridgeBrigandsAreHostileAtLoadAndTheTalkRaisesAtTheRadius(t *testing.T) {
	f := openCampaignMission(t, 120)
	mw := f.live
	w := mw.world
	if got, want := missionRel(w, 1, 5), "21110"; got != want {
		t.Fatalf("Self row = %s, want %s", got, want)
	}
	if got := missionRel(w, 2, 5); got[0] != '1' {
		t.Fatalf("Brigands row %s is not hostile toward Self", got)
	}
	hero := mapload.PartyEntity(mw.mission.state.Map, 0)
	talk, ok := ReadEventText(mw.mission.src, 120, 3)
	if !ok {
		t.Fatal("event 3 text is unreadable")
	}
	missionTicks(mw, 20)
	for _, e := range w.Entities() {
		if e.Owner == 2 && e.HasTarget {
			t.Fatalf("brigand %d holds an order before any Self unit is near", e.ID)
		}
	}
	if err := w.HeadlessPlace(hero, 49, 38); err != nil {
		t.Fatal(err)
	}
	mw.enqueue(uint32(hero), 66, 38)
	talked, attacked := -1, -1
	for i := 0; i < 600 && (talked < 0 || attacked < 0); i++ {
		mw.tick()
		if mw.mission.open {
			if talked < 0 && string(mw.mission.payload) == string(talk) {
				talked = i
			}
			mw.mission.open = false
		}
		for _, e := range w.Entities() {
			if attacked < 0 && e.Owner == 2 && e.Group == 3 && e.HasTarget {
				attacked = i
			}
		}
	}
	// The talk waits for the next script pass (16 ticks), the brigand's
	// acquisition runs every tick, so a walker on this road meets both within
	// one pass of each other and either may come first.
	if talked < 0 || attacked < 0 || talked-attacked > 16 || attacked-talked > 16 {
		t.Fatalf("bridge talk at tick %d, first brigand target at tick %d; want both within one script pass", talked, attacked)
	}
}

// TRIG-VIP-081 and MISSION-DEFEAT-045: the Talker's death is the loss, a
// Self unit within 6 completes the mission, a wound or the goblin's death
// decides nothing.
func TestReleaseTalkerDeathLosesAndApproachCompletes(t *testing.T) {
	type actors struct {
		w                    *sim.World
		hero, talker, goblin sim.EntityID
		talkerHP             int32
	}
	for _, tc := range []struct {
		name string
		do   func(a actors) error
		want sim.Outcome
	}{
		{"hero within 6", func(a actors) error { return a.w.HeadlessPlace(a.hero, 69, 44) }, sim.OutcomeWon},
		{"Talker killed from afar", func(a actors) error { return a.w.HeadlessKill(a.talker) }, sim.OutcomeLost},
		{"Talker wounded only", func(a actors) error { return a.w.HeadlessDamage(a.talker, a.talkerHP/2) }, sim.OutcomeUndecided},
		{"goblin killed", func(a actors) error { return a.w.HeadlessKill(a.goblin) }, sim.OutcomeUndecided},
		{"Talker killed with a hero within 6", func(a actors) error {
			if err := a.w.HeadlessKill(a.talker); err != nil {
				return err
			}
			return a.w.HeadlessPlace(a.hero, 69, 44)
		}, sim.OutcomeLost},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := openCampaignMission(t, 110)
			mw := f.live
			w := mw.world
			m := mw.mission.state.Map
			units := mapload.ScriptUnits(m, mw.mission.party)
			a := actors{w: w, hero: mapload.PartyEntity(m, 0), talker: units[57], goblin: units[25]}
			e, ok := w.Entity(a.talker)
			if !ok || e.Owner != 5 || e.X != 69 || e.Y != 39 {
				t.Fatalf("unit 57 = %+v, want player 5 at (69,39)", e)
			}
			a.talkerHP = e.HP
			for from := uint32(1); from <= 6; from++ {
				if from != 5 && w.Relations().Byte(from, 5) != 0 {
					t.Fatalf("player %d is not at 0 toward the Talker", from)
				}
			}
			if got := missionRel(w, 5, 6); got != "000020" {
				t.Fatalf("Talker row = %s, want 0 toward every player and the map's own 2 on the diagonal", got)
			}
			if err := tc.do(a); err != nil {
				t.Fatal(err)
			}
			missionTicks(mw, 200)
			if got := w.Outcome(); got != tc.want {
				t.Fatalf("outcome = %v, want %v", got, tc.want)
			}
		})
	}
}

// ALM-VIEW-215: nothing in mission 140's script or terrain reveals Rood Glaen
// or the tower to the participant.
func TestReleaseMission140ShowsNeitherTheBodyNorTheTowerUntilSeen(t *testing.T) {
	f := openCampaignMission(t, 140)
	mw := f.live
	w := mw.world
	hero := mapload.PartyEntity(mw.mission.state.Map, 0)
	rood := mapload.ScriptUnits(mw.mission.state.Map, mw.mission.party)[443]
	body, ok := w.Entity(rood)
	if !ok || body.X != 242 || body.Y != 115 {
		t.Fatalf("unit 443 = %+v, want Rood Glaen at (242,115)", body)
	}
	cell := func(plane []byte, x, y int) bool { return plane[y*mw.fog.cols+x] != 0 }
	hidden := func() bool {
		for _, p := range [][2]int{{242, 115}, {243, 112}} {
			if cell(mw.fog.visible, p[0], p[1]) || cell(mw.fog.explored, p[0], p[1]) {
				return false
			}
		}
		return true
	}
	if !hidden() {
		t.Fatal("the body or the tower is visible at the start")
	}
	missionTicks(mw, 1500)
	if !hidden() {
		t.Fatal("running the mission's script revealed the body or the tower")
	}
	// Control: the same planes show both to a hero standing beside them.
	if err := w.HeadlessPlace(hero, 238, 115); err != nil {
		t.Fatal(err)
	}
	missionTicks(mw, 20)
	if !cell(mw.fog.visible, 242, 115) || !cell(mw.fog.visible, 243, 112) {
		t.Fatal("control: a hero beside the body does not see it, so the probe cannot discriminate")
	}
}

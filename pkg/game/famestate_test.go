package game

import (
	"encoding/base64"
	"reflect"
	"testing"

	"againrom/pkg/mapload"
	"againrom/pkg/render/terrain"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

func fameRelations(t *testing.T, cells ...[3]uint32) sim.Relations {
	t.Helper()
	raw := make([]byte, 50*50)
	for _, c := range cells {
		raw[c[0]*50+c[1]] = byte(c[2])
	}
	rel, err := sim.NewRelations(raw)
	if err != nil {
		t.Fatal(err)
	}
	return rel
}

func fameWorld(t *testing.T, relations sim.Relations, entities ...sim.Entity) *sim.World {
	t.Helper()
	w, err := sim.NewRelatedWorld(1181, sim.Bounds{Width: 12, Height: 8},
		sim.ModeCanonical, sim.Terrain{}, entities, nil, relations)
	if err != nil {
		t.Fatal(err)
	}
	return w
}

func fameDriver(t *testing.T, w *sim.World) *mapWorld {
	t.Helper()
	v, err := ui.NewViewer("fame", terrain.Grid{Width: 12, Height: 8, Tiles: make([]uint16, 12*8)}, &terrain.Tileset{})
	if err != nil {
		t.Fatal(err)
	}
	return newMapWorld(w, nil, nil, v)
}

func fameSetClock(t *testing.T, w *sim.World, subtick uint32) {
	t.Helper()
	if err := w.ImportOriginalSession(sim.OriginalSession{
		HasClock: true, Clock: sim.SessionClock{SubTick: subtick},
		TriggerLatches: make([]byte, 1000), Diplomacy: make([]byte, 50*50),
	}); err != nil {
		t.Fatal(err)
	}
}

func TestFameScoreUsesSeparateBinary64AndSignedDwords(t *testing.T) {
	for _, tc := range []struct {
		name         string
		time, events uint32
		xp, want     int32
	}{
		{"zero time binary64 boundary", 0, 3, 1000000, 6},
		{"division truncates", 3, 5, 100, 16},
		{"negative XP truncates toward zero", 3, 5, -100, -16},
		{"signed time", 0xffffffff, 5, 100, -50},
		{"signed events", 2, 0xffffffff, 100, -5},
		{"signed64 result retains low32", 1, 100, 2147483647, -10},
		{"no events", 1, 0, 2147483647, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := earnedFameScore(tc.time, tc.events, tc.xp); got != tc.want {
				t.Fatalf("score(%08x,%08x,%d)=%d, want %d", tc.time, tc.events, tc.xp, got, tc.want)
			}
		})
	}
}

func TestFameMissionTimeUsesSignedSubticksAndAcceptedCompletionOnce(t *testing.T) {
	for _, tc := range []struct {
		subtick, start, want uint32
	}{
		{0, 0, 0}, {15, 0, 0}, {16, 0, 1}, {31, 0, 1},
		{0x7fffffff, 0, 0x07ffffff}, {0x80000000, 0, 0xf8000000},
		{0xfffffff1, 17, 17}, {0xfffffff0, 17, 16}, {16, 0xffffffff, 0},
	} {
		c := Campaign{Main: []int{10}}
		f := &FrontEnd{InstallResources: InstallResources{Campaign: resolved(c, nil)}, CampaignSession: CampaignSession{Town: NewTown(c), fame: SnapshotFame{Known: true, Time: tc.start}}}
		w := fameWorld(t, sim.Relations{}, sim.Entity{ID: 7, X: 1, Y: 1, Owner: sim.SelfSlot, HP: 100, MaxHP: 100})
		fameSetClock(t, w, tc.subtick)
		f.FinishMission(0, nil, w, nil)
		if f.fame.Time != tc.start {
			t.Fatal("unaccepted mission added time")
		}
		f.FinishMission(10, nil, w, nil)
		f.FinishMission(10, nil, w, nil)
		if f.fame.Time != tc.want || !f.Town.Done(10) {
			t.Fatalf("subtick=%08x start=%08x time=%08x, want %08x", tc.subtick, tc.start, f.fame.Time, tc.want)
		}
	}
}

func TestFameTerminalResultUsesPrimaryHeroLiveXPAndIsImmutable(t *testing.T) {
	c := Campaign{Main: []int{10, 20}, Last: map[int]bool{10: true}}
	f := &FrontEnd{InstallResources: InstallResources{Campaign: resolved(c, nil)}, CampaignSession: CampaignSession{Town: NewTown(c), fame: SnapshotFame{Known: true, Events: 3}}}
	party := []mapload.PartyMember{
		{ID: "companion", Name: "Selected companion", PlayerCharacter: true},
		{ID: "hero", Name: "Earned hero", PlayerCharacter: true, StartingHero: true},
	}
	w := fameWorld(t, sim.Relations{},
		sim.Entity{ID: 7, X: 1, Y: 1, Owner: 1, HP: 100, MaxHP: 100, SkillXP: [6]int32{9000000}},
		sim.Entity{ID: 9, X: 3, Y: 1, Owner: 1, HP: 100, MaxHP: 100, SkillXP: [6]int32{100000, 200000, 300000, 100000, 200000, 100000}},
	)
	// The sixth XP slot participates; the selected/first companion does not.
	f.FinishMission(10, party, w, []sim.EntityID{7, 9})
	if got := f.fame.Result; got == nil || *got != (FameResult{Name: "Earned hero", Score: 6}) {
		t.Fatalf("earned result=%+v", got)
	}
	result := f.fame.Result
	f.fame.Events = 999
	party[1].Name = "Changed later"
	fameSetClock(t, w, 160)
	f.FinishMission(10, party, w, []sim.EntityID{7, 9})
	if f.fame.Result != result || *result != (FameResult{Name: "Earned hero", Score: 6}) || f.fame.Time != 0 {
		t.Fatalf("acknowledging the win again changed result/time: %+v", f.fame)
	}
	// A wrapped signed32 XP aggregate remains a negative score, not an int64
	// sum or a clamped display value. The roster supplies the primary identity.
	f = &FrontEnd{InstallResources: InstallResources{Campaign: resolved(c, nil)}, CampaignSession: CampaignSession{Town: NewTown(c), fame: SnapshotFame{Known: true, Events: 1}}}
	w = fameWorld(t, sim.Relations{}, sim.Entity{ID: 9, X: 3, Y: 1, Owner: 1, HP: 100, MaxHP: 100,
		SkillXP: [6]int32{2147483647, 1}})
	f.FinishMissionWithRoster(10, nil, w, nil, map[sim.EntityID]mapload.PartyMember{9: party[1]})
	if got := f.fame.Result; got == nil || got.Score != -4294 || got.Name != "Changed later" {
		t.Fatalf("roster primary signed32 XP result=%+v", got)
	}
}

func TestFameProductionTickCountsDecayTransitionsAndKeepsWorldDigest(t *testing.T) {
	rel := fameRelations(t, [3]uint32{1, 2, 1}, [3]uint32{3, 1, 1}, [3]uint32{1, 4, 2})
	entities := []sim.Entity{
		{ID: 1, X: 1, Y: 1, Owner: 2, HP: 100, MaxHP: 100, DyingTime: 4},
		{ID: 2, X: 2, Y: 1, Owner: 2, HP: -15, MaxHP: 100},
		{ID: 3, X: 3, Y: 1, Owner: 2, HP: -25, MaxHP: 100},
		{ID: 4, X: 4, Y: 1, Owner: 2, HP: -45, MaxHP: 100},
		{ID: 5, X: 5, Y: 1, Owner: 2, HP: -601, MaxHP: 100},
		{ID: 6, X: 6, Y: 1, Owner: 3, HP: -15, MaxHP: 100},
		{ID: 7, X: 7, Y: 1, Owner: 2, HP: -15, MaxHP: 100, Decay: 2},
		{ID: 8, X: 8, Y: 1, Owner: 4, HP: -15, MaxHP: 100},
	}
	w := fameWorld(t, rel, entities...)
	control := fameWorld(t, rel, entities...)
	mw := fameDriver(t, w)
	f := &FrontEnd{CampaignSession: CampaignSession{fame: SnapshotFame{Known: true}}}
	f.liveDriver(mw, 10, nil)
	if f.fame.Events != 0 {
		t.Fatal("binding already dead actors earned events", f.fame.Events)
	}
	command := sim.Command{Kind: sim.KindKill, Entity: 1}
	mw.pending = append(mw.pending, command)
	mw.tick()
	sim.Step(control, []sim.Command{command})
	if f.fame.Events != 3 {
		t.Fatalf("events=%d, want stages1->2/3/4 only", f.fame.Events)
	}
	if w.Hash() != control.Hash() {
		t.Fatal("enabling campaign observation changed simulation digest")
	}
	for i := 0; i < 3; i++ {
		mw.push()
		mw.observeFame()
	}
	if f.fame.Events != 3 || w.Hash() != control.Hash() {
		t.Fatal("repeated projection/observation added an event or changed world")
	}
	for _, state := range mw.fame.previous {
		if state.ID == 5 {
			t.Fatal("removed actor remains in observation baseline")
		}
	}
	// A new cold world gets its own baseline and detaches the old driver.
	form, err := w.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	var restored sim.World
	if err := restored.UnmarshalBinary(form); err != nil {
		t.Fatal(err)
	}
	back := fameDriver(t, &restored)
	f.liveDriver(back, 10, nil)
	back.observeFame()
	if mw.fame.state != nil || f.fame.Events != 3 {
		t.Fatal("cold restore counted existing bodies or retained old observer", f.fame.Events)
	}
}

func TestFameObservationUsesCurrentHostilityResurrectionAndIdentity(t *testing.T) {
	state := SnapshotFame{Known: true, Events: 0xffffffff}
	observer := fameObserver{state: &state, previous: []sim.CorpseState{{ID: 7, Owner: 2, Stage: 2}}}
	hostile := fameRelations(t, [3]uint32{1, 2, 1})
	neutral := fameRelations(t)
	steps := []struct {
		states []sim.CorpseState
		rel    sim.Relations
		want   uint32
	}{
		{[]sim.CorpseState{{ID: 7, Owner: 2, Stage: 0}}, hostile, 0xffffffff},
		{[]sim.CorpseState{{ID: 7, Owner: 2, Stage: 1}}, neutral, 0xffffffff},
		{[]sim.CorpseState{{ID: 7, Owner: 2, Stage: 2}}, hostile, 0},
		{[]sim.CorpseState{{ID: 7, Owner: 2, Stage: 3}}, hostile, 0},
		{nil, hostile, 0},
		{[]sim.CorpseState{{ID: 9, Owner: 2, Stage: 1}}, hostile, 0},
		{[]sim.CorpseState{{ID: 9, Owner: 2, Stage: 2}}, neutral, 0},
		{[]sim.CorpseState{{ID: 10, Owner: 2, Stage: 3}}, hostile, 1},
	}
	for i, step := range steps {
		observer.observe(step.states, step.rel)
		if state.Events != step.want {
			t.Fatalf("step %d events=%d, want %d", i, state.Events, step.want)
		}
	}
}

func TestFameNativeRoundTripOwnsResultAndRejectsUnknownResult(t *testing.T) {
	f := &FrontEnd{CampaignSession: CampaignSession{Town: NewTown(Campaign{}), fame: SnapshotFame{Known: true, Time: 31, Events: 7, Result: &FameResult{Name: "Saved hero", Score: 81}}}}
	f.Town.Arrive()
	s, _, err := f.Snapshot(false)
	if err != nil {
		t.Fatal(err)
	}
	f.fame.Result.Recorded = true
	if s.Fame == nil || s.Fame.Result.Recorded {
		t.Fatal("live publication changed a prior snapshot")
	}
	b, err := EncodeSave(s, "fame")
	if err != nil {
		t.Fatal(err)
	}
	decoded, _, err := DecodeSave(b)
	if err != nil {
		t.Fatal(err)
	}
	g := &FrontEnd{}
	if _, town, err := g.Restore(decoded); err != nil || !town {
		t.Fatalf("restore town=%v err=%v", town, err)
	}
	if !reflect.DeepEqual(g.fame, *s.Fame) {
		t.Fatalf("native roundtrip fame=%+v, want %+v", g.fame, *s.Fame)
	}
	g.fame.Result.Recorded = true
	if decoded.Fame.Result.Recorded || s.Fame.Result.Recorded {
		t.Fatal("restored publication changed its source snapshot")
	}
	before := cloneFame(g.fame)
	bad := Snapshot{Fame: &SnapshotFame{Result: &FameResult{Name: "invented", Score: 99}}}
	if _, err := EncodeSave(bad, "bad"); err == nil {
		t.Fatal("encoded an earned result with unknown history")
	}
	if _, _, err := g.Restore(bad); err == nil || !reflect.DeepEqual(g.fame, before) {
		t.Fatal("invalid restore accepted or replaced prior fame")
	}
	bad = Snapshot{Mission: 10, Fame: &SnapshotFame{Known: true, Time: 99, Events: 101}}
	if _, _, err := g.Restore(bad); err == nil || !reflect.DeepEqual(g.fame, before) {
		t.Fatal("late world validation refusal replaced prior fame")
	}
}

func TestFameRestoredCampaignFallbackAndFreshProjection(t *testing.T) {
	c, town := restoredTownFixture(t)
	f := &FrontEnd{InstallResources: InstallResources{Campaign: resolved(c, nil)}, CampaignSession: CampaignSession{Town: town, fame: SnapshotFame{Known: true, Time: 35, Events: 12}}}
	s, _, err := f.Snapshot(false)
	if err != nil {
		t.Fatal(err)
	}
	if !s.CampaignState || s.Campaign.MissionTime != 35 || s.Campaign.ScoreEvents != 12 || !s.Campaign.ScoreEventsKnown {
		t.Fatalf("campaign projection retained old score totals: %+v", s.Campaign)
	}
	// Only a projection whose writer carried both original counter dwords may
	// supply the compatibility baseline for a snapshot without native Fame.
	s.Fame = nil
	g := &FrontEnd{InstallResources: InstallResources{Campaign: resolved(c, nil)}}
	if _, _, err := g.Restore(s); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(g.fame, SnapshotFame{Known: true, Time: 35, Events: 12}) {
		t.Fatalf("known campaign fallback=%+v", g.fame)
	}
	s.Campaign.ScoreEventsKnown = false
	if _, _, err := g.Restore(s); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(g.fame, SnapshotFame{Time: 35, Events: 12}) {
		t.Fatalf("incomplete campaign observations or unknown policy lost: %+v", g.fame)
	}
	s.Campaign.ScoreEventsKnown = true
	s.Fame = &SnapshotFame{Known: true, Time: 77, Events: 19}
	if _, _, err := g.Restore(s); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(g.fame, *s.Fame) {
		t.Fatalf("older campaign projection overrode native score history: %+v", g.fame)
	}
}

func TestFameLegacyUnknownAndNewGameAdoption(t *testing.T) {
	old, err := base64.StdEncoding.DecodeString(preMercenaryStateSaveFixtureBase64)
	if err != nil {
		t.Fatal(err)
	}
	s, _, err := DecodeSave(old)
	if err != nil {
		t.Fatal(err)
	}
	if s.Fame != nil {
		t.Fatal("frozen old schema somehow supplied fame history")
	}
	s.Mission, s.World, s.Open = 0, nil, true
	f := missionFrontEnd(t)
	f.fame = SnapshotFame{Known: true, Time: 9, Events: 11, Result: &FameResult{Name: "old", Score: 50}}
	if _, _, err := f.Restore(s); err != nil {
		t.Fatal(err)
	}
	if f.fame.Known || f.fame.Result != nil {
		t.Fatal("old AGS acquired invented history or a result")
	}
	lastMission := f.Campaign.Value()
	lastMission.Last = map[int]bool{10: true}
	f.Campaign = resolved(lastMission, nil)
	w := fameWorld(t, sim.Relations{}, sim.Entity{ID: 1, X: 1, Y: 1, Owner: 1, HP: 100, MaxHP: 100})
	fameSetClock(t, w, 32)
	f.FinishMission(10, nil, w, nil)
	if f.fame.Known || f.fame.Result != nil {
		t.Fatal("later play converted unknown prior history into an earned result")
	}
	f.fame = SnapshotFame{Time: 18, Events: 4}
	before := f.fame
	res := ui.ChargenResult{Name: "Fresh", Difficulty: 1, Stats: []int{25, 25, 25, 25}}
	if _, _, _, _, _, _, _, _, _, _, err := f.NewGameOpener(20, res)(); err == nil {
		t.Fatal("missing mission opened")
	}
	if !reflect.DeepEqual(f.fame, before) {
		t.Fatal("failed new game reset campaign score history")
	}
	open, err := f.prepareNewGame(10, res)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(f.fame, before) {
		t.Fatal("prepared new game committed score history early")
	}
	if _, _, _, _, _, _, _, _, _, _, err := open(); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(f.fame, SnapshotFame{Known: true}) {
		t.Fatalf("new game did not initialize complete empty history: %+v", f.fame)
	}
}

// A bind keeps the counter the source carried: loaded bodies at any stage earn
// nothing (DIV-1268).
func TestFameBindKeepsSavedCounterWithLoadedBodies(t *testing.T) {
	rel := fameRelations(t, [3]uint32{1, 2, 1})
	w := fameWorld(t, rel,
		sim.Entity{ID: 2, X: 2, Y: 1, Owner: 2, HP: -15, MaxHP: 100, Decay: 2},
		sim.Entity{ID: 3, X: 3, Y: 1, Owner: 2, HP: -15, MaxHP: 100, Decay: 4},
	)
	f := &FrontEnd{CampaignSession: CampaignSession{fame: SnapshotFame{Known: true, Events: 10}}}
	mw := fameDriver(t, w)
	f.liveDriver(mw, 10, nil)
	mw.observeFame()
	if f.fame.Events != 10 {
		t.Fatal("bind changed the saved counter", f.fame.Events)
	}
}

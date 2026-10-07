package game

import (
	"bytes"
	"image"
	"reflect"
	"testing"
	"time"

	"againrom/pkg/audio"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

func schoolTrainingFixture(t *testing.T) (*FrontEnd, *townScreen, *time.Time, *schoolRotateRecorder) {
	t.Helper()
	f := shellFrontEnd()
	fighter := f.Carried[0]
	fighter.Name, fighter.Mage = "Training fighter", false
	mage := fighter
	mage.Name, mage.Mage = "Training mage", true
	f.Carried = []mapload.PartyMember{fighter, mage}
	schoolArt, err := LoadTownSchoolArt(townSchoolTrainingSource())
	if err != nil {
		t.Fatal(err)
	}
	f.TownSchoolArt = resolved(schoolArt, nil)
	now := time.Unix(300, 0)
	f.TownAnimationNow = func() time.Time { return now }
	f.SchoolRandom = func(n int) int {
		if n != schoolTrainingRandomRange {
			t.Fatalf("school random range = %d", n)
		}
		return 0
	}
	recorder := &schoolRotateRecorder{}
	f.SoundPlayer = recorder
	f.SoundBank = &SoundBank{named: map[string]soundCacheEntry{
		schoolRotateSound: {sample: audio.Sample{Rate: audio.DeviceRate, PCM: []int16{31, -7}}, ok: true},
	}}
	s := f.townUI
	s.Back()
	s.Choose(2)
	if s.room != roomSchool || !s.schoolTraining.ready || !s.schoolTraining.active {
		t.Fatalf("school entry = room%d training%+v", s.room, s.schoolTraining)
	}
	return f, s, &now, recorder
}

func paintSchoolTraining(t *testing.T, s *townScreen, now *time.Time, elapsed time.Duration) *image.RGBA {
	t.Helper()
	*now = now.Add(elapsed)
	return paintDiamondSchool(t, s)
}

func finishSchoolEntryTransition(t *testing.T, s *townScreen, now *time.Time) {
	t.Helper()
	for i := 0; i < 19; i++ {
		paintSchoolTraining(t, s, now, 84*time.Millisecond)
	}
	if s.schoolTrainingBusy() || s.schoolColumn.frame != 0 || s.schoolPanelClass() != schoolFighterClass {
		t.Fatalf("fighter entry did not settle: training%+v column%+v", s.schoolTraining, s.schoolColumn)
	}
}

func TestSchoolTrainingStrictGateTransitionColumnThresholdAndInputBlock(t *testing.T) {
	_, s, now, recorder := schoolTrainingFixture(t)
	// This test isolates both tr families; idle vectors have their own test.
	s.in.TownSchoolArt.Value().Training[schoolFighterClass].Idle = nil
	s.in.TownSchoolArt.Value().Training[schoolMageClass].Idle = nil
	if s.schoolTrainingBusy() || !s.schoolTraining.sides[schoolFighterClass].transitionActive ||
		s.schoolTraining.sides[schoolFighterClass].transitionIndex != 0 {
		t.Fatalf("entry transition = %+v", s.schoolTraining)
	}
	if s.schoolPanelClass() != schoolFighterClass {
		t.Fatal("entry transition hid the skill panel although the column already rests at its endpoint")
	}
	beforeStats := s.townStats
	s.TownSurfaceClick(ui.TownSurfaceControl{Kind: ui.TownSurfaceControlMode}, false)
	if s.townStats == beforeStats {
		t.Fatal("entry transition swallowed a live mode click")
	}
	s.TownSurfaceClick(ui.TownSurfaceControl{Kind: ui.TownSurfaceControlMode}, false)
	s.TownSurfaceClick(ui.TownSurfaceControl{Kind: ui.TownSurfaceControlCell, Index: 0}, false)
	if s.schoolCell != 0 {
		t.Fatal("entry transition swallowed a live skill-cell click")
	}
	s.clearSchoolSelection()
	if s.townStats != beforeStats || s.schoolCell != schoolNoSelection || s.room != roomSchool ||
		!s.schoolTraining.sides[schoolFighterClass].transitionActive || s.schoolTraining.sides[schoolFighterClass].transitionIndex != 0 {
		t.Fatal("live-input probe left stray state or perturbed the untouched entry transition")
	}

	paintSchoolTraining(t, s, now, 83*time.Millisecond)
	if got := s.schoolTraining.sides[schoolFighterClass].transitionIndex; got != 0 {
		t.Fatalf("83ms equality advanced tr to %d", got)
	}
	paintSchoolTraining(t, s, now, time.Millisecond)
	if got := s.schoolTraining.sides[schoolFighterClass].transitionIndex; got != 1 {
		t.Fatalf("84ms advanced tr to %d, want 1", got)
	}
	for i := 1; i < 19; i++ {
		paintSchoolTraining(t, s, now, 84*time.Millisecond)
	}
	if s.schoolTrainingBusy() || s.schoolPanelClass() != schoolFighterClass || len(recorder.samples) != 0 {
		t.Fatalf("entry completion = busy%v class%d sounds%d", s.schoolTrainingBusy(), s.schoolPanelClass(), len(recorder.samples))
	}

	s.TownSurfaceClick(ui.TownSurfaceControl{Kind: ui.TownSurfaceControlNext}, false)
	if s.shopMemberIndex() != 1 || s.TownSurface().Hero.Member != 1 || s.schoolTraining.waitClass != schoolMageClass ||
		s.schoolColumn.frame != 0 || s.schoolColumn.target != 15 || len(recorder.samples) != 0 {
		t.Fatalf("mage selection = member%d training%+v column%+v sounds%d", s.shopMemberIndex(), s.schoolTraining, s.schoolColumn, len(recorder.samples))
	}
	if s.schoolPanelClass() != -1 {
		t.Fatal("old endpoint remained interactive during mage transition")
	}
	paintSchoolTraining(t, s, now, 83*time.Millisecond)
	if s.schoolTraining.sides[schoolMageClass].transitionIndex != 0 || s.schoolColumn.frame != 0 {
		t.Fatal("class transition admitted equality boundary")
	}
	for want := 1; want <= 4; want++ {
		elapsed := 84 * time.Millisecond
		if want == 1 {
			elapsed = time.Millisecond
		}
		paintSchoolTraining(t, s, now, elapsed)
		if got := s.schoolTraining.sides[schoolMageClass].transitionIndex; got != want || s.schoolColumn.frame != 0 || len(recorder.samples) != 0 {
			t.Fatalf("mage tr%d = index%d column%d sounds%d", want, got, s.schoolColumn.frame, len(recorder.samples))
		}
	}
	paintSchoolTraining(t, s, now, 84*time.Millisecond)
	if s.schoolTraining.sides[schoolMageClass].transitionIndex != 5 || s.schoolColumn.frame != 1 || len(recorder.samples) != 1 {
		t.Fatalf("mage threshold = tr%d column%d sounds%d", s.schoolTraining.sides[schoolMageClass].transitionIndex, s.schoolColumn.frame, len(recorder.samples))
	}
	paintSchoolTraining(t, s, now, time.Hour)
	if s.schoolTraining.sides[schoolMageClass].transitionIndex != 6 || s.schoolColumn.frame != 2 {
		t.Fatal("long gap caught up more or less than one shared step")
	}
	for i := 0; s.schoolTrainingBusy() && i < 64; i++ {
		paintSchoolTraining(t, s, now, 84*time.Millisecond)
	}
	if s.schoolTrainingBusy() || s.schoolColumn.frame != 15 || s.schoolPanelClass() != schoolMageClass || len(recorder.samples) != 1 {
		t.Fatalf("mage completion = busy%v column%+v class%d sounds%d", s.schoolTrainingBusy(), s.schoolColumn, s.schoolPanelClass(), len(recorder.samples))
	}
	wantPlace := audio.Placement{Left: audio.GainUnit, Right: audio.GainUnit}
	if recorder.places[0] != wantPlace {
		t.Fatalf("Rotate placement = %+v", recorder.places[0])
	}

	// The opposite class uses its own measured index6 threshold and the same
	// one-request-at-column-start rule.
	s.TownSurfaceClick(ui.TownSurfaceControl{Kind: ui.TownSurfaceControlPrevious}, false)
	if s.shopMemberIndex() != 0 || s.schoolTraining.waitClass != schoolFighterClass || s.schoolColumn.target != 0 {
		t.Fatalf("fighter selection = member%d training%+v column%+v", s.shopMemberIndex(), s.schoolTraining, s.schoolColumn)
	}
	paintSchoolTraining(t, s, now, 83*time.Millisecond)
	if s.schoolTraining.sides[schoolFighterClass].transitionIndex != 0 || s.schoolColumn.frame != 15 {
		t.Fatal("fighter class transition admitted equality boundary")
	}
	for want := 1; want <= 5; want++ {
		elapsed := 84 * time.Millisecond
		if want == 1 {
			elapsed = time.Millisecond
		}
		paintSchoolTraining(t, s, now, elapsed)
		if got := s.schoolTraining.sides[schoolFighterClass].transitionIndex; got != want || s.schoolColumn.frame != 15 || len(recorder.samples) != 1 {
			t.Fatalf("fighter tr%d = index%d column%d sounds%d", want, got, s.schoolColumn.frame, len(recorder.samples))
		}
	}
	paintSchoolTraining(t, s, now, 84*time.Millisecond)
	if s.schoolTraining.sides[schoolFighterClass].transitionIndex != 6 || s.schoolColumn.frame != 14 || len(recorder.samples) != 2 {
		t.Fatalf("fighter threshold = tr%d column%d sounds%d", s.schoolTraining.sides[schoolFighterClass].transitionIndex, s.schoolColumn.frame, len(recorder.samples))
	}
	for i := 0; s.schoolTrainingBusy() && i < 64; i++ {
		paintSchoolTraining(t, s, now, 84*time.Millisecond)
	}
	if s.schoolTrainingBusy() || s.schoolColumn.frame != 0 || s.schoolPanelClass() != schoolFighterClass || len(recorder.samples) != 2 {
		t.Fatalf("fighter completion = busy%v column%+v class%d sounds%d", s.schoolTrainingBusy(), s.schoolColumn, s.schoolPanelClass(), len(recorder.samples))
	}
}

func TestSchoolTrainingIdleBoundariesMageSkipFighterReverseAndPriority(t *testing.T) {
	f, s, now, recorder := schoolTrainingFixture(t)
	finishSchoolEntryTransition(t, s, now)
	// Isolate the mage episode; an incomplete sibling disables only itself.
	f.TownSchoolArt.Value().Training[schoolFighterClass].Idle = nil
	p := &s.schoolTrainingStatic
	p.initialized = [schoolClassCount]bool{true, true}
	p.idleLast = [schoolClassCount]time.Time{*now, *now}
	p.idleExtra = [schoolClassCount]time.Duration{}

	paintSchoolTraining(t, s, now, schoolTrainingIdleBase)
	if s.schoolTraining.sides[schoolMageClass].idleActive {
		t.Fatal("idle armed at elapsed equality")
	}
	paintSchoolTraining(t, s, now, time.Millisecond)
	mage := &s.schoolTraining.sides[schoolMageClass]
	if !mage.idleActive || mage.idleIndex != -1 || mage.idleCached != 0 || mage.idleDirection != 1 {
		t.Fatalf("idle arm = %+v", *mage)
	}
	if got := s.schoolTrainingFrame().Mage; got != f.TownSchoolArt.Value().Training[schoolMageClass].Idle[0] {
		t.Fatal("armed mage did not publish m0001")
	}
	paintSchoolTraining(t, s, now, 84*time.Millisecond)
	if mage.idleIndex != 0 || mage.idleCached != 0 {
		t.Fatalf("first gated mage step = %+v", *mage)
	}
	for want := 1; want <= 10; want++ {
		paintSchoolTraining(t, s, now, 84*time.Millisecond)
		if mage.idleIndex != want || mage.idleCached != want {
			t.Fatalf("mage ascent %d = %+v", want, *mage)
		}
	}
	paintSchoolTraining(t, s, now, 84*time.Millisecond)
	if mage.idleDirection != -1 || mage.idleIndex != 8 || mage.idleCached != 10 || p.holdLimit[schoolMageClass] != 20 {
		t.Fatalf("mage terminal = side%+v hold%d/%d", *mage, p.hold[schoolMageClass], p.holdLimit[schoolMageClass])
	}
	for i := 0; i < 19; i++ {
		paintSchoolTraining(t, s, now, 84*time.Millisecond)
	}
	if mage.idleCached != 10 || p.hold[schoolMageClass] != 19 {
		t.Fatalf("mage hold = cached%d count%d", mage.idleCached, p.hold[schoolMageClass])
	}
	paintSchoolTraining(t, s, now, 84*time.Millisecond)
	if mage.idleIndex != 7 || mage.idleCached != 7 {
		t.Fatalf("mage return did not skip indices 9/8: %+v", *mage)
	}
	for want := 6; want >= 0; want-- {
		paintSchoolTraining(t, s, now, 84*time.Millisecond)
		if mage.idleIndex != want || mage.idleCached != want || !mage.idleActive {
			t.Fatalf("mage reverse %d = %+v", want, *mage)
		}
	}
	paintSchoolTraining(t, s, now, 84*time.Millisecond)
	if mage.idleActive || s.schoolTrainingFrame().Mage != nil {
		t.Fatal("mage reverse terminal remained drawn")
	}

	// Fighter holds m0009 and returns normally to m0008 (index7).
	fighter := &s.schoolTraining.sides[schoolFighterClass]
	*fighter = schoolTrainingSideAnimation{idleActive: true, idleIndex: 8, idleCached: 8, idleDirection: 1}
	p.hold[schoolFighterClass], p.holdLimit[schoolFighterClass] = 0, 0
	paintSchoolTraining(t, s, now, 84*time.Millisecond)
	if fighter.idleIndex != 8 || fighter.idleCached != 8 || fighter.idleDirection != -1 || p.holdLimit[schoolFighterClass] != 20 {
		t.Fatalf("fighter terminal = %+v hold%d", *fighter, p.holdLimit[schoolFighterClass])
	}
	for i := 0; i < 20; i++ {
		paintSchoolTraining(t, s, now, 84*time.Millisecond)
	}
	if fighter.idleIndex != 7 || fighter.idleCached != 7 {
		t.Fatalf("fighter reverse did not begin 8->7: %+v", *fighter)
	}

	// Same-side m owns update and draw while tr waits. Arming tr forces an
	// ascending m into bounded reverse without clearing it on that paint.
	*fighter = schoolTrainingSideAnimation{}
	*mage = schoolTrainingSideAnimation{transitionActive: true, transitionIndex: 2, idleActive: true, idleIndex: 5, idleCached: 5, idleDirection: 1}
	p.hold[schoolMageClass], p.holdLimit[schoolMageClass] = 0, 0
	paintSchoolTraining(t, s, now, 84*time.Millisecond)
	if mage.idleIndex != 4 || mage.idleCached != 4 || mage.idleDirection != -1 || mage.transitionIndex != 2 ||
		s.schoolTrainingFrame().Mage != f.TownSchoolArt.Value().Training[schoolMageClass].Idle[4] {
		t.Fatalf("m-over-tr priority = %+v", *mage)
	}
	for mage.idleActive {
		paintSchoolTraining(t, s, now, 84*time.Millisecond)
	}
	paintSchoolTraining(t, s, now, 84*time.Millisecond)
	if mage.transitionIndex != 3 {
		t.Fatalf("tr did not resume after m: %+v", *mage)
	}
	if len(recorder.samples) != 0 {
		t.Fatalf("idle episode requested %d animation sounds", len(recorder.samples))
	}
}

func TestSchoolTrainingReadOnlyLifecycleAndNativeBoundary(t *testing.T) {
	f, s, now, _ := schoolTrainingFixture(t)
	beforeRead := s.schoolTraining
	view := s.TownSurface()
	ui.TownSurfaceControlAt(view, image.Pt(240, 212))
	ui.ComposeTownSurface(view)
	s.Rows()
	if !reflect.DeepEqual(s.schoolTraining, beforeRead) {
		t.Fatal("view, hit-test or raw compositor advanced school training")
	}
	paintSchoolTraining(t, s, now, 84*time.Millisecond)
	if s.schoolTraining.sides[schoolFighterClass].transitionIndex != 1 {
		t.Fatal("own-surface paint did not advance once")
	}

	s.SchoolTrainingActive(false)
	frozen := s.schoolTraining.sides
	paintSchoolTraining(t, s, now, time.Hour)
	if s.schoolTraining.sides != frozen {
		t.Fatal("inactive school caught up hidden time")
	}
	s.SchoolTrainingActive(true)
	paintSchoolTraining(t, s, now, 83*time.Millisecond)
	if s.schoolTraining.sides != frozen {
		t.Fatal("resume admitted strict equality")
	}
	paintSchoolTraining(t, s, now, time.Millisecond)
	if s.schoolTraining.sides[schoolFighterClass].transitionIndex != 2 {
		t.Fatal("resume did not advance after rebased 84ms")
	}
	s.room, s.dialogueBuilding = roomTalk, TownSchool
	paintSchoolTraining(t, s, now, 84*time.Millisecond)
	if s.schoolTraining.sides[schoolFighterClass].transitionIndex != 3 {
		t.Fatal("school dialogue stopped the room animation")
	}

	mw, _ := readoutWorld(t, sim.Entity{ID: 1, X: 4, Y: 5})
	f.live = mw
	worldBefore, hashBefore := marshalWorld(t, mw.world), mw.world.Hash()
	snapshotBefore, _, err := f.Snapshot(false)
	if err != nil {
		t.Fatal(err)
	}
	nativeBefore, err := EncodeSave(snapshotBefore, "same")
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 80; i++ {
		paintSchoolTraining(t, s, now, 84*time.Millisecond)
	}
	snapshotAfter, _, err := f.Snapshot(false)
	if err != nil {
		t.Fatal(err)
	}
	nativeAfter, err := EncodeSave(snapshotAfter, "same")
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(nativeBefore, nativeAfter) || hashBefore != mw.world.Hash() || !bytes.Equal(worldBefore, marshalWorld(t, mw.world)) {
		t.Fatal("school presentation changed native bytes, simulation hash or binary state")
	}

	s.room = roomSchool
	staticBefore := s.schoolTrainingStatic
	s.Back()
	if s.room != roomSquare || s.schoolTraining != (schoolTrainingAnimation{}) || s.schoolTrainingStatic != staticBefore {
		t.Fatal("school exit did not drop only object-local state")
	}
	s.Choose(2)
	// Reentry arms the fighter's own tr bit (still animates) but, like first
	// entry, never sets waitClass: only a changed class blocks school input.
	if !s.schoolTraining.ready || s.schoolTraining.waitClass != schoolTrainingNoWait ||
		!s.schoolTraining.sides[schoolFighterClass].transitionActive || s.schoolTrainingStatic != staticBefore {
		t.Fatal("school reentry did not rebuild object state over retained process statics")
	}
	s.resetForNewGame()
	if s.schoolTraining != (schoolTrainingAnimation{}) || s.schoolTrainingStatic != staticBefore {
		t.Fatal("new game retained object-local school state or erased process statics")
	}
}

// TestSchoolTrainingEntryAdmitsInputButClassChangeStillBlocks is the R-1
// regression. Before the fix, enterSchoolTraining armed the entering class's
// own tr bit by calling armSchoolTrainingTransition with changedClass=false,
// and that call unconditionally set waitClass too, so schoolTrainingBusy()
// -- the shared gate behind Choose, Back, TownSurfaceClick,
// townSurfaceButton and schoolPanelClass -- stayed true for the whole entry
// cycle (19 fighter or 23 mage states at the shared >83ms gate) with no
// dialogue in front of it and no message. This test fails on 33e6fc07 and
// passes once armSchoolTrainingTransition sets waitClass exclusively on the
// changedClass path (schooltraining.go).
func TestSchoolTrainingEntryAdmitsInputButClassChangeStillBlocks(t *testing.T) {
	_, s, now, recorder := schoolTrainingFixture(t)

	// Property 4: entry still animates the arming transition.
	entrySide := s.schoolTraining.sides[schoolFighterClass]
	if !entrySide.transitionActive || entrySide.transitionIndex != 0 {
		t.Fatalf("entry did not arm its own transition: %+v", entrySide)
	}

	// Property 1: the whole surface, including the skill panel, is live on
	// the first painted frame after entry.
	if s.schoolTrainingBusy() {
		t.Fatal("entry blocked school input")
	}
	if s.schoolPanelClass() != schoolFighterClass || s.schoolColumn.frame != schoolColumnEndpoint(false) {
		t.Fatal("skill panel is not showing the entered class on the first frame")
	}
	view := s.TownSurface()
	if view.SchoolClass != schoolFighterClass || (view.SchoolColumnSet && view.SchoolColumnFrame != view.SchoolClass*15) {
		t.Fatal("schoolPanelVisible's own condition would be false on the first painted frame")
	}
	live := 0
	for i := 0; i < 5; i++ {
		if view.Cells[i].Enabled {
			live++
		}
	}
	if live != 5 {
		t.Fatalf("live fighter skill cells = %d, want 5", live)
	}
	for i := 5; i < 10; i++ {
		if view.Cells[i].Enabled {
			t.Fatalf("mage skill cell %d is enabled for a fighter member", i)
		}
	}
	if msg := s.townSurfaceButton(0).Msg; msg != "" {
		t.Fatalf("Train button = %q, want no line", msg)
	}
	beforeStats := s.townStats
	s.TownSurfaceClick(ui.TownSurfaceControl{Kind: ui.TownSurfaceControlMode}, false)
	if s.townStats == beforeStats {
		t.Fatal("the mode click was swallowed")
	}
	s.TownSurfaceClick(ui.TownSurfaceControl{Kind: ui.TownSurfaceControlMode}, false)
	s.TownSurfaceClick(ui.TownSurfaceControl{Kind: ui.TownSurfaceControlCell, Index: 0}, false)
	if s.schoolCell != 0 {
		t.Fatal("the skill-cell click was swallowed")
	}
	s.clearSchoolSelection()
	// Escape/Back leaves the room outright instead of reporting a lie, and
	// reentry stays live and keeps animating too.
	if !s.Back() || s.room != roomSquare {
		t.Fatal("Back was gated instead of leaving the school")
	}
	s.Choose(2)
	if s.room != roomSchool || s.schoolTrainingBusy() || !s.schoolTraining.sides[schoolFighterClass].transitionActive {
		t.Fatal("reentry blocked input or did not animate its own transition")
	}
	if len(recorder.samples) != 0 {
		t.Fatal("entry at an already-matching endpoint was not silent")
	}

	// Properties 2 and 3: a CHANGED class still blocks the whole surface --
	// Back still reports "handled" without actually leaving, TOWN-429's own
	// documented shape -- until the tr counter returns to 0 and the column
	// reaches the matching endpoint, and Rotate is still requested exactly
	// once when that class change first starts column motion.
	s.TownSurfaceClick(ui.TownSurfaceControl{Kind: ui.TownSurfaceControlNext}, false)
	if !s.schoolTrainingBusy() || s.schoolTraining.waitClass != schoolMageClass {
		t.Fatal("a changed class no longer blocks school input")
	}
	if !s.Back() || s.room != roomSchool {
		t.Fatal("a changed class no longer blocks Back/Escape")
	}
	for i := 0; s.schoolTrainingBusy() && i < 64; i++ {
		paintSchoolTraining(t, s, now, 84*time.Millisecond)
	}
	if s.schoolTrainingBusy() || s.schoolColumn.frame != schoolColumnEndpoint(true) || len(recorder.samples) != 1 {
		t.Fatalf("class change did not settle correctly: busy%v column%+v sounds%d", s.schoolTrainingBusy(), s.schoolColumn, len(recorder.samples))
	}
}

// TestSchoolTrainingSuspendedFreezesTheLegacyColumnFallbackToo is the O-1
// fix. AdvanceTownSurfaceAnimation's degraded-install column fallback used to
// run whenever advanceSchoolTraining returned false, and that includes
// SchoolTrainingActive(false) (focus loss, a menu or a cutscene, DIV-872)
// while a class change is still moving the column: the fallback ignores
// waitClass and the threshold entirely, so it kept walking the column toward
// its new target every paint even though both training sides were correctly
// frozen. schoolTrainingSuspended now gates the fallback the same way.
func TestSchoolTrainingSuspendedFreezesTheLegacyColumnFallbackToo(t *testing.T) {
	_, s, now, _ := schoolTrainingFixture(t)
	finishSchoolEntryTransition(t, s, now)
	s.TownSurfaceClick(ui.TownSurfaceControl{Kind: ui.TownSurfaceControlNext}, false)
	if !s.schoolTrainingBusy() || s.schoolColumn.frame != 0 || s.schoolColumn.target != 15 {
		t.Fatalf("class change did not arm a pending column walk: %+v", s.schoolColumn)
	}
	s.SchoolTrainingActive(false)
	for i := 0; i < 40; i++ {
		paintSchoolTraining(t, s, now, 84*time.Millisecond)
	}
	if s.schoolColumn.frame != 0 {
		t.Fatalf("suspension did not freeze the legacy column fallback, frame=%d", s.schoolColumn.frame)
	}
	s.SchoolTrainingActive(true)
	for i := 0; i < 40 && s.schoolColumn.frame != 15; i++ {
		paintSchoolTraining(t, s, now, 84*time.Millisecond)
	}
	if s.schoolColumn.frame != 15 {
		t.Fatal("resume never let the suspended class change finish")
	}
}

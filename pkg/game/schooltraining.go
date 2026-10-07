package game

import (
	"image"
	"math/rand"
	"time"

	"againrom/pkg/ui"
)

const (
	schoolTrainingInterval    = 83 * time.Millisecond
	schoolTrainingIdleBase    = 3000 * time.Millisecond
	schoolTrainingRandomRange = 32768
	schoolTrainingNoWait      = -1
)

// schoolTrainingSideAnimation is one school side's object-local presentation
// state. Transition and idle may both be armed, but idle has update and draw
// priority until its bounded reverse completes (TOWN-430..432).
type schoolTrainingSideAnimation struct {
	transitionActive bool
	transitionIndex  int

	idleActive    bool
	idleIndex     int
	idleCached    int
	idleDirection int
}

// schoolTrainingAnimation is presentation only. It is rebuilt on room entry
// and never enters a save, simulation world or hash. The private generator is
// likewise isolated from every gameplay and other-town random stream.
type schoolTrainingAnimation struct {
	ready, active bool
	last          time.Time
	random        *rand.Rand

	waitClass     int
	columnStarted bool
	sides         [schoolClassCount]schoolTrainingSideAnimation
}

// schoolTrainingStatic projects TOWN-433's process-static armer timestamps,
// random extras and hold counters without package globals. townScreen itself
// is process-retained, so this state survives room-object resets and new-game
// model replacement while remaining presentation-only.
type schoolTrainingStatic struct {
	initialized [schoolClassCount]bool
	idleLast    [schoolClassCount]time.Time
	idleExtra   [schoolClassCount]time.Duration
	hold        [schoolClassCount]int
	holdLimit   [schoolClassCount]int

	// shineCycle and shineLast are TOWN-500's idle shine: the slot index the
	// idle icon stands on and the time it last advanced.
	shineCycle int
	shineLast  time.Time
	shineReady bool
}

// schoolShineInterval is the least time between two steps of the idle shine's
// slot index (TOWN-500).
const schoolShineInterval = 500 * time.Millisecond

// advanceSchoolShine steps the idle shine's slot index once, modulo five, when
// at least schoolShineInterval has passed since the last step; a missed step
// is never caught up.
func (t *townScreen) advanceSchoolShine(now time.Time) {
	p := &t.schoolTrainingStatic
	if !p.shineReady || now.Before(p.shineLast) {
		p.shineReady, p.shineLast = true, now
		return
	}
	if now.Sub(p.shineLast) >= schoolShineInterval {
		p.shineCycle = (p.shineCycle + 1) % 5
		p.shineLast = now
	}
}

// schoolStoredToDisplaySlot maps the stored slot order of the original's icon
// arrays (sword, axe, pike, club, bow; fire, water, earth, air, astral) to this
// build's shared slot order, which swaps the third and fourth (DIV-121).
var schoolStoredToDisplaySlot = [5]int{0, 1, 3, 2, 4}

func schoolClassForMage(mage bool) int {
	if mage {
		return schoolMageClass
	}
	return schoolFighterClass
}

func schoolTrainingFamilyComplete(frames []image.Image, count int) bool {
	if len(frames) != count {
		return false
	}
	for _, frame := range frames {
		if frame == nil {
			return false
		}
	}
	return true
}

func (t *townScreen) schoolTrainingArt() *[schoolClassCount]ui.SchoolTrainingArt {
	if t == nil || t.sess == nil || t.in.TownSchoolArt.Value() == nil {
		return nil
	}
	return &t.in.TownSchoolArt.Value().Training
}

func (t *townScreen) inSchoolTraining() bool {
	return t != nil && (t.room == roomSchool || t.room == roomTalk && t.dialogueBuilding == TownSchool)
}

func (t *townScreen) resetSchoolTraining() {
	if t != nil {
		t.schoolTraining = schoolTrainingAnimation{}
	}
}

func (t *townScreen) leaveSchoolTraining() {
	t.resetSchoolTraining()
	t.schoolSounds.Stop()
	if t.soundRoom() != roomSchool && t.audioRoom == roomSchool {
		t.destroyRoomAudio()
	}
}

// enterSchoolTraining rebuilds object-local state and arms the selected
// class's tr cycle. Process-static timers deliberately remain separate.
func (t *townScreen) enterSchoolTraining() {
	if t == nil || t.sess == nil {
		return
	}
	t.resetSchoolColumn()
	now := t.townAnimationNow()
	t.schoolTraining = schoolTrainingAnimation{
		ready:     true,
		active:    true,
		last:      now,
		random:    rand.New(rand.NewSource(now.UnixNano())),
		waitClass: schoolTrainingNoWait,
	}
	party, member := t.shopParty(), t.shopMemberIndex()
	if member < 0 || member >= len(party) {
		return
	}
	t.armSchoolTrainingTransition(schoolClassForMage(party[member].Mage), now, false)
}

// SchoolTrainingActive is Againrom's explicit delivery policy for focus,
// menu and cutscene pauses. A resume rebases the shared gate and both idle
// clocks, so hidden wall time can never catch up animation frames.
func (t *townScreen) SchoolTrainingActive(active bool) {
	if t == nil || t.sess == nil {
		return
	}
	active = active && t.inSchoolTraining()
	if active && !t.schoolTraining.ready {
		t.enterSchoolTraining()
	}
	a := &t.schoolTraining
	if !active {
		a.active = false
		return
	}
	if a.active {
		return
	}
	now := t.townAnimationNow()
	a.last = now
	t.schoolTrainingStatic.shineLast = now
	for class := range t.schoolTrainingStatic.initialized {
		if t.schoolTrainingStatic.initialized[class] {
			t.schoolTrainingStatic.idleLast[class] = now
		}
	}
	a.active = true
}

func (t *townScreen) schoolTrainingRoll() int {
	if draw := t.draws.schoolDraw(); draw != nil {
		v := draw(schoolTrainingRandomRange) % schoolTrainingRandomRange
		if v < 0 {
			v += schoolTrainingRandomRange
		}
		return v
	}
	if t.schoolTraining.random == nil {
		t.schoolTraining.random = rand.New(rand.NewSource(t.townAnimationNow().UnixNano()))
	}
	return t.schoolTraining.random.Intn(schoolTrainingRandomRange)
}

func (t *townScreen) nextSchoolIdleExtra() time.Duration {
	return time.Duration(t.schoolTrainingRoll()/10) * time.Millisecond
}

// nextSchoolHoldLimit preserves the instruction arithmetic exactly. In
// particular a maximum 15-bit result maps back to 20, not to 39.
func (t *townScreen) nextSchoolHoldLimit() int {
	return ((t.schoolTrainingRoll()*20)/0x7fff)%20 + 20
}

func (t *townScreen) schoolTrainingTransitionComplete(class int) bool {
	art := t.schoolTrainingArt()
	return art != nil && class >= 0 && class < schoolClassCount &&
		schoolTrainingFamilyComplete(art[class].Transition, schoolTrainingFamilyShape[class].transitionLast+1)
}

func (t *townScreen) schoolTrainingIdleComplete(class int) bool {
	art := t.schoolTrainingArt()
	return art != nil && class >= 0 && class < schoolClassCount &&
		schoolTrainingFamilyComplete(art[class].Idle, schoolTrainingFamilyShape[class].idleLast)
}

func (t *townScreen) schoolColumnComplete() bool {
	if t == nil || t.sess == nil || t.in.TownSchoolArt.Value() == nil {
		return false
	}
	for _, frame := range t.in.TownSchoolArt.Value().Column {
		if frame == nil {
			return false
		}
	}
	return true
}

// armSchoolTrainingTransition arms a complete tr family. When the optional
// family is absent, callers retain the landed legacy column path instead of
// blocking a usable school behind invisible cosmetic state.
//
// DIV-873
func (t *townScreen) armSchoolTrainingTransition(class int, now time.Time, changedClass bool) bool {
	if class < 0 || class >= schoolClassCount || !t.schoolTrainingTransitionComplete(class) {
		return false
	}
	a := &t.schoolTraining
	side := &a.sides[class]
	side.transitionActive, side.transitionIndex = true, 0
	if changedClass {
		a.waitClass = class
	}
	mage := class == schoolMageClass
	if !t.schoolColumn.ready {
		t.schoolColumn.reset(mage)
	}
	if changedClass {
		t.schoolColumn.retarget(mage, now)
	}
	if !t.schoolColumnComplete() {
		t.schoolColumn.reset(mage)
	}
	a.columnStarted = t.schoolColumn.frame == schoolColumnEndpoint(mage)
	return true
}

func (t *townScreen) schoolTrainingBusy() bool {
	return t != nil && t.schoolTraining.ready && t.schoolTraining.waitClass != schoolTrainingNoWait
}

// schoolTrainingSuspended reports whether SchoolTrainingActive(false) has
// deliberately paused an already-entered room: focus loss, a menu or a
// cutscene (DIV-872). advanceSchoolTraining's own leading guard already
// freezes both tr/idle sides for exactly this state; AdvanceTownSurfaceAnimation
// uses this to freeze the legacy column fallback with them, so a class-change
// column walk cannot keep moving while the rest of the room is paused.
func (t *townScreen) schoolTrainingSuspended() bool {
	return t != nil && t.inSchoolTraining() && t.schoolTraining.ready && !t.schoolTraining.active
}

func (t *townScreen) armSchoolIdle(class int, now time.Time) {
	a, p := &t.schoolTraining, &t.schoolTrainingStatic
	side := &a.sides[class]
	busy := side.transitionActive || side.idleActive
	if !p.initialized[class] {
		p.initialized[class] = true
		p.idleLast[class] = now
		if t.schoolTrainingIdleComplete(class) {
			p.idleExtra[class] = t.nextSchoolIdleExtra()
		}
		return
	}
	if now.Before(p.idleLast[class]) {
		p.idleLast[class] = now
		return
	}
	if busy {
		p.idleLast[class] = now
		return
	}
	if !t.schoolTrainingIdleComplete(class) || now.Sub(p.idleLast[class]) <= schoolTrainingIdleBase+p.idleExtra[class] {
		return
	}
	*side = schoolTrainingSideAnimation{
		transitionActive: side.transitionActive,
		transitionIndex:  side.transitionIndex,
		idleActive:       true,
		idleIndex:        -1,
		idleCached:       0,
		idleDirection:    1,
	}
	p.idleLast[class] = now
}

func (t *townScreen) finishSchoolIdle(class int, now time.Time) {
	side, p := &t.schoolTraining.sides[class], &t.schoolTrainingStatic
	side.idleActive, side.idleIndex, side.idleCached, side.idleDirection = false, 0, 0, 0
	p.idleLast[class] = now
	p.idleExtra[class] = t.nextSchoolIdleExtra()
}

func (t *townScreen) advanceSchoolIdle(class int, now time.Time) {
	side, p := &t.schoolTraining.sides[class], &t.schoolTrainingStatic
	last := schoolTrainingFamilyShape[class].idleLast - 1
	if side.transitionActive && side.idleDirection > 0 {
		side.idleDirection = -1
		p.hold[class] = p.holdLimit[class]
	}
	if side.idleDirection > 0 {
		if side.idleIndex < last {
			side.idleIndex++
			side.idleCached = side.idleIndex
			return
		}
		side.idleDirection = -1
		p.hold[class] = 0
		p.holdLimit[class] = t.nextSchoolHoldLimit()
		if class == schoolMageClass {
			// TOWN-431 rewrites the logical index but holds cached m0011.
			side.idleIndex = 8
		}
		return
	}
	if p.hold[class] < p.holdLimit[class] {
		p.hold[class]++
		if p.hold[class] < p.holdLimit[class] {
			return
		}
	}
	if side.idleIndex <= 0 {
		t.finishSchoolIdle(class, now)
		return
	}
	side.idleIndex--
	side.idleCached = side.idleIndex
}

func (t *townScreen) advanceSchoolTransition(class int) {
	side := &t.schoolTraining.sides[class]
	count := schoolTrainingFamilyShape[class].transitionLast + 1
	side.transitionIndex = (side.transitionIndex + 1) % count
	if side.transitionIndex == 0 {
		side.transitionActive = false
	}
}

func schoolColumnThreshold(class int) int {
	if class == schoolMageClass {
		return 5
	}
	return 6
}

// advanceSchoolTraining owns the complete-install shared strict >83ms gate.
// It returns whether that gate owned column delivery for this paint; a false
// result lets the degraded-art legacy column path keep working.
func (t *townScreen) advanceSchoolTraining() bool {
	if t == nil || !t.inSchoolTraining() || !t.schoolTraining.ready || !t.schoolTraining.active {
		return false
	}
	a := &t.schoolTraining
	now := t.townAnimationNow()
	ownedColumn := a.waitClass != schoolTrainingNoWait
	if now.Before(a.last) {
		a.last = now
		for class := range t.schoolTrainingStatic.initialized {
			if t.schoolTrainingStatic.initialized[class] && now.Before(t.schoolTrainingStatic.idleLast[class]) {
				t.schoolTrainingStatic.idleLast[class] = now
			}
		}
		return ownedColumn
	}
	for class := 0; class < schoolClassCount; class++ {
		t.armSchoolIdle(class, now)
	}
	if now.Sub(a.last) <= schoolTrainingInterval {
		return ownedColumn
	}
	a.last = now
	for class := 0; class < schoolClassCount; class++ {
		side := &a.sides[class]
		if side.idleActive {
			t.advanceSchoolIdle(class, now)
		} else if side.transitionActive {
			t.advanceSchoolTransition(class)
		}
	}

	if a.waitClass != schoolTrainingNoWait {
		class := a.waitClass
		side := &a.sides[class]
		mage := class == schoolMageClass
		if !a.columnStarted && side.transitionIndex >= schoolColumnThreshold(class) {
			a.columnStarted = true
			t.requestSchoolRotateSound()
		}
		if a.columnStarted && t.schoolColumn.frame != schoolColumnEndpoint(mage) {
			t.schoolColumn.advanceStep()
		}
		if !side.transitionActive && t.schoolColumn.frame == schoolColumnEndpoint(mage) {
			a.waitClass = schoolTrainingNoWait
		}
	}
	return ownedColumn
}

func schoolTrainingPicture(frames []image.Image, index int) image.Image {
	if index < 0 || index >= len(frames) {
		return nil
	}
	return frames[index]
}

// schoolTrainingFrame resolves m-over-tr priority before the UI seam. If a
// winning family disappears from a synthetic/degraded art object after it was
// armed, it yields nil rather than exposing the lower-priority family.
func (t *townScreen) schoolTrainingFrame() ui.SchoolTrainingFrame {
	var frame ui.SchoolTrainingFrame
	art := t.schoolTrainingArt()
	if art == nil || !t.schoolTraining.ready {
		return frame
	}
	choose := func(class int) image.Image {
		side := &t.schoolTraining.sides[class]
		if side.idleActive {
			return schoolTrainingPicture(art[class].Idle, side.idleCached)
		}
		if side.transitionActive {
			return schoolTrainingPicture(art[class].Transition, side.transitionIndex)
		}
		return nil
	}
	frame.Fighter = choose(schoolFighterClass)
	frame.Mage = choose(schoolMageClass)
	return frame
}

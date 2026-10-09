package game

import (
	"image"
	"math/rand"
	"time"

	"againrom/pkg/audio"
	"againrom/pkg/ui"
)

const townExteriorInterval = 67 * time.Millisecond

type townExteriorAnimation struct {
	frame                                         ui.TownExteriorFrame
	ready, present, active                        bool
	selector                                      int
	last                                          time.Time
	random                                        *rand.Rand
	ambientRandom                                 *rand.Rand
	shop, tavern, school, sign, fluger            bool
	fighterStep, mageStep, guardStep              int
	gateLatch, guardLatch, shopLatch, schoolLatch bool
	// Bird progress has three physical words even when the selected prefix is
	// shorter. terminalPaint holds the one final overlay-only snapshot until
	// the next live paint, because App advances before it reacquires the view.
	birdActive, birdTerminalPaint bool
	birdGroup, birdCount          int
	birdProgress                  [3]int
	birdClock, birdTerminalAt     time.Time
	starActive, starVisible       bool
	starCurrent                   int
	crowdActive                   bool
	crowdVoice                    audio.Voice
	voices                        [12]audio.Voice
	fam                           townFamilies
}

const (
	exteriorShop = iota
	exteriorSchool
	exteriorTavern
	exteriorGate
	exteriorGuard
	exteriorSign
	exteriorFluger
	exteriorBird
	exteriorStar
	exteriorHorse1
	exteriorHorse2
	exteriorHorse3
)

const (
	townBirdFrameCount = 57
	townStarFrameCount = 9
)

func (a *townExteriorAnimation) stop(slot int) {
	if a.voices[slot] != nil {
		audio.StopReset(a.voices[slot])
		a.voices[slot] = nil
	}
}

func (a *townExteriorAnimation) silence() {
	for slot := range a.voices {
		a.stop(slot)
	}
}

func (t *townScreen) resetTownExterior() {
	t.stopTownCrowd()
	t.exterior.silence()
	t.exterior = townExteriorAnimation{}
}

func (t *townScreen) ensureTownExterior() {
	a := &t.exterior
	if a.ready {
		return
	}
	a.ready = true
	a.selector = -1
	a.frame.Door = 8
	a.frame.Guard = t.townGuardRestFrame()
	now := t.townAnimationNow()
	a.random = rand.New(rand.NewSource(now.UnixNano()))
	a.ambientRandom = rand.New(rand.NewSource(now.UnixNano() ^ 0x5deece66d))
	a.birdClock = now
	a.starCurrent = 0
	a.starVisible = t.townStarArtReady()
}

func (t *townScreen) townExteriorFrame() *ui.TownExteriorFrame {
	f := t.exterior.frame
	if !t.exterior.ready {
		f.Door = 8
		f.Guard = t.townGuardRestFrame()
		f.Star = ui.TownExteriorSpriteFrame{Frame: 0, Visible: t.townStarArtReady()}
		return &f
	}
	a := &t.exterior
	if a.birdActive {
		f.BirdOverlayVisible = true
		art := t.townExteriorArt()
		for i := 0; i < a.birdCount && i < len(f.Birds); i++ {
			family := a.birdGroup*3 + i
			frame := a.birdProgress[i]
			visible := frame >= 0 && frame < townBirdFrameCount && art != nil &&
				family >= 0 && family < len(art.Birds) && frame < len(art.Birds[family])
			f.Birds[i] = ui.TownExteriorSpriteFrame{Family: family, Frame: frame, Visible: visible}
		}
	}
	f.Star = ui.TownExteriorSpriteFrame{Frame: a.starCurrent, Visible: a.starVisible && t.townStarArtReady()}
	t.townFamilyFrames(&f)
	return &f
}

// townGuardRestFrame is the frame the guards show on every entry to the
// square: the last frame of their sheet, whatever frame, step or latch they
// held when the square was left (TOWN-477). A sheet that did not load has no
// frame to show, so the answer is 0.
func (t *townScreen) townGuardRestFrame() int {
	if art := t.townExteriorArt(); art != nil && len(art.Guard) > 0 {
		return len(art.Guard) - 1
	}
	return 0
}

func (t *townScreen) townExteriorArt() *ui.TownExteriorArt {
	if t == nil || t.sess == nil || t.in.TownSquareArt.Value() == nil {
		return nil
	}
	return t.in.TownSquareArt.Value().Exterior
}

func (t *townScreen) townStarArtReady() bool {
	art := t.townExteriorArt()
	return art != nil && len(art.Stars) == townStarFrameCount
}

func (t *townScreen) townBirdArtPresent() bool {
	art := t.townExteriorArt()
	if art == nil {
		return false
	}
	for _, family := range art.Birds {
		if len(family) == townBirdFrameCount {
			return true
		}
	}
	return false
}

// TownSquareActive is an explicit client lifecycle policy (DIV-834/836).
// Pausing is not a blank-mask event: retained flags and sound latches remain.
func (t *townScreen) TownSquareActive(active bool) {
	if t == nil || t.sess == nil {
		return
	}
	onSquare := t.AtTownSquare()
	active = active && onSquare
	if !active && !t.exterior.ready {
		return
	}
	t.ensureTownExterior()
	a := &t.exterior
	if !onSquare {
		if t.audioRoom == roomSquare {
			t.destroyRoomAudio()
		}
		a.present = false
		a.birdActive, a.birdTerminalPaint = false, false
		a.starActive, a.starVisible = false, false
	}
	if onSquare && !a.present {
		now := t.townAnimationNow()
		a.present = true
		a.birdActive, a.birdTerminalPaint = false, false
		a.birdGroup, a.birdCount, a.birdProgress = 0, 0, [3]int{}
		a.birdClock = now
		a.starActive, a.starCurrent = false, 0
		a.starVisible = t.townStarArtReady()
		t.enterTownFamilies(now)
	}
	if a.active != active {
		a.active = active
		if !active {
			a.selector = -1
			a.silence()
			t.stopTownCrowd()
		} else {
			t.ensureTownCrowd()
		}
	}
}

func boundedPresentationRoll(draw func(int) int, fallback *rand.Rand, n int) int {
	if n <= 0 {
		return 0
	}
	value := 0
	if draw != nil {
		value = draw(n)
	} else if fallback != nil {
		value = fallback.Intn(n)
	}
	value %= n
	if value < 0 {
		value += n
	}
	return value
}

func (t *townScreen) exteriorRoll(n int) int {
	if draw := t.draws.animationDraw(); draw != nil {
		return boundedPresentationRoll(draw, nil, n)
	}
	return boundedPresentationRoll(nil, t.exterior.random, n)
}

func (t *townScreen) townAmbientRoll(n int) int {
	return boundedPresentationRoll(t.draws.ambientDraw(), t.exterior.ambientRandom, n)
}

func (t *townScreen) exteriorSound(slot int, path string) {
	a := &t.exterior
	if v := a.voices[slot]; v != nil {
		if v.Playing() {
			return
		}
		a.stop(slot)
	}
	if sample, ok := t.in.SoundBank.namedSample(path); ok {
		source := "town-exterior"
		if slot >= exteriorHorse1 {
			source = "town-horse"
		}
		a.voices[slot] = audio.Dispatch(t.roomSoundPlayer(audio.EffectsChannel), sample,
			audio.FixedRequest(source, path, audio.EffectsChannel, 128, false,
				audio.Placement{Left: audio.GainUnit, Right: audio.GainUnit}))
	}
}

func (t *townScreen) ensureTownCrowd() {
	a := &t.exterior
	if !a.active || a.crowdActive || t.sound.ambientDevice() == nil {
		return
	}
	sample, ok := t.in.SoundBank.namedSample("town/crowd.wav")
	if !ok {
		return
	}
	request := audio.FixedRequest("town-crowd", "town/crowd.wav", audio.EffectsChannel, 128, true,
		audio.Placement{Left: audio.GainUnit, Right: audio.GainUnit})
	if ui.DeliveryOwner(t.sound.soundDevice()) != nil {
		a.crowdVoice = audio.Dispatch(t.roomSoundPlayer(audio.EffectsChannel), sample, request)
	} else {
		a.crowdVoice = ui.RequestAmbient(t.sound.ambientDevice(), ui.AmbientTownCrowd, sample, request)
	}
	a.crowdActive = true
}

func (t *townScreen) stopTownCrowd() {
	if t == nil || t.sess == nil || !t.exterior.crowdActive {
		return
	}
	if t.sound.ambientDevice() != nil {
		if ui.DeliveryOwner(t.sound.soundDevice()) != nil {
			audio.StopReset(t.exterior.crowdVoice)
		} else {
			t.sound.ambientDevice().StopLoop(ui.AmbientTownCrowd)
		}
	}
	t.exterior.crowdVoice = nil
	t.exterior.crowdActive = false
}

func (t *townScreen) exteriorGateAvailable() bool {
	if t.gateRule != nil {
		return t.gateRule()
	}
	return t.sess.Town.gateMission() != -1
}

// TownSquarePointer receives each admitted update, not just mask transitions.
// TOWN-399..402 arm without resetting frames, even on pointer residence.
func (t *townScreen) TownSquarePointer(p image.Point) {
	if t == nil || t.sess == nil || !t.AtTownSquare() {
		return
	}
	t.TownSquareActive(true)
	a := &t.exterior
	a.selector = -1
	if art := t.in.TownSquareArt; art.Value() != nil && art.Value().Mask != nil && p.In(art.Value().Mask.Bounds()) {
		switch art.Value().Mask.ColorIndexAt(p.X, p.Y) {
		case 0x80:
			a.selector = 2
		case 0x90:
			a.selector = 1
		case 0xa0:
			a.selector = 8
		case 0xb0:
			a.selector = 16
		case 0xc0:
			a.selector = 4
		}
	}
	a.guardStep = 1
	switch a.selector {
	case -1:
		a.shopLatch, a.schoolLatch = false, false
	case 1:
		if t.exteriorRoll(100) > 95 {
			a.shop = true
		}
		if !a.shopLatch {
			a.stop(exteriorSchool)
			t.exteriorSound(exteriorShop, "town/shop/enter.wav")
			a.shopLatch, a.schoolLatch = true, false
		}
	case 2:
		a.tavern = true
	case 4:
		a.school = true
		if !a.schoolLatch {
			a.stop(exteriorShop)
			t.exteriorSound(exteriorSchool, "town/school/point.wav")
			a.schoolLatch, a.shopLatch = true, false
		}
	case 8:
		if !t.exteriorGateAvailable() {
			a.guardStep = -1
		}
	case 16:
		a.starActive = true
	}
}

func exteriorCycle(frame *int, enabled *bool, count int) {
	if !*enabled || count == 0 {
		return
	}
	*frame++
	if *frame >= count {
		*frame, *enabled = 0, false
	}
}

// The caller tests shared enable once. Fighter clearing it must not skip the
// mage's same-hub update (TOWN-402).
func (t *townScreen) exteriorSchoolStep(frame, step *int, count int) {
	if count == 0 {
		return
	}
	if *frame == 0 && *step == -1 {
		*step = 0
		t.exterior.school = false
		return
	}
	if *frame == 0 && *step == 0 {
		if t.exteriorRoll(100) > 95 {
			*step = 1
		}
	} else if *frame == count-1 {
		*step = 0
		if t.exteriorRoll(100) > 95 {
			*step = -1
		}
	}
	*frame += *step
}

// AdvanceTownSquareAnimation is called by production composition. Update and
// hit-test/view readers cannot advance this paint-owned hub.
func (t *townScreen) AdvanceTownSquareAnimation() {
	if t == nil || t.sess == nil || !t.AtTownSquare() || !t.exterior.active {
		return
	}
	square := t.in.TownSquareArt
	if square.Value() == nil || square.Value().Exterior == nil {
		return
	}
	a, art := &t.exterior, square.Value().Exterior
	now := t.townAnimationNow()
	if t.townPaintLast.IsZero() {
		t.townPaintLast, a.last = now, now
		return
	}
	if a.birdTerminalPaint && now != a.birdTerminalAt {
		a.birdTerminalPaint, a.birdActive = false, false
	}
	t.armTownBirds(now)
	if now.Sub(t.townPaintLast) <= townExteriorInterval {
		t.paintTownFamilies(now)
		return
	}
	t.townPaintLast, a.last = now, now
	if t.exteriorRoll(100) > 94 {
		a.sign = true
	}
	if t.exteriorRoll(100) > 97 {
		a.fluger = true
	}
	if a.birdActive {
		for i := range a.birdProgress {
			a.birdProgress[i]++
		}
		terminal := a.birdCount > 0
		for i := 0; i < a.birdCount && i < len(a.birdProgress); i++ {
			terminal = terminal && a.birdProgress[i] >= townBirdFrameCount
		}
		if terminal {
			a.birdTerminalPaint = true
			a.birdTerminalAt = now
			// Start the next retained delay after this episode's terminal
			// snapshot, so the following paint is genuinely inactive.
			a.birdClock = now
		}
	}
	t.advanceTownStar()
	exteriorCycle(&a.frame.Shop, &a.shop, len(art.Shop))
	if a.tavern && len(art.Tavern) > 0 && a.frame.Tavern == 0 {
		a.stop(exteriorShop)
		a.stop(exteriorSchool)
		t.exteriorSound(exteriorTavern, "town/point.wav")
	}
	exteriorCycle(&a.frame.Tavern, &a.tavern, len(art.Tavern))
	if a.school {
		t.exteriorSchoolStep(&a.frame.Fighter, &a.fighterStep, len(art.Fighter))
		t.exteriorSchoolStep(&a.frame.Mage, &a.mageStep, len(art.Mage))
	}
	if len(art.Door) > 0 {
		if !t.exteriorGateAvailable() {
			a.frame.Door = len(art.Door) - 1 // do not change the latch
		} else {
			inside := a.selector == 8
			if inside != a.gateLatch {
				a.stop(exteriorGate)
				path := "town/gatedn.wav"
				if inside {
					path = "town/gateup.wav"
				}
				t.exteriorSound(exteriorGate, path)
				a.gateLatch = inside
			}
			if inside && a.frame.Door > 0 {
				a.frame.Door--
			}
			if !inside && a.frame.Door < len(art.Door)-1 {
				a.frame.Door++
			}
		}
	}
	if a.sign && len(art.Sign) > 0 && a.frame.Sign == 0 {
		t.exteriorSound(exteriorSign, "town/flag.wav")
	}
	exteriorCycle(&a.frame.Sign, &a.sign, len(art.Sign))
	if a.fluger && len(art.Fluger) > 0 && a.frame.Fluger == 0 {
		t.exteriorSound(exteriorFluger, "town/flugel.wav")
	}
	exteriorCycle(&a.frame.Fluger, &a.fluger, len(art.Fluger))
	if a.guardStep != 0 && len(art.Guard) > 0 {
		a.frame.Guard += a.guardStep
		forward := a.guardStep > 0
		if forward != a.guardLatch {
			a.stop(exteriorGuard)
			path := "town/guard1.wav"
			if forward {
				path = "town/guard2.wav"
			}
			t.exteriorSound(exteriorGuard, path)
			a.guardLatch = forward
		}
		if a.frame.Guard < 0 || a.frame.Guard >= len(art.Guard) {
			if a.frame.Guard < 0 {
				a.frame.Guard = 0
			} else {
				a.frame.Guard = len(art.Guard) - 1
			}
			a.guardStep = 0
			a.stop(exteriorGuard)
		}
	}
	t.stepTownFamilies(now)
	t.paintTownFamilies(now)
}

func (t *townScreen) nextTownBirdDelay() time.Duration {
	return time.Duration(1000+t.townAmbientRoll(2000)) * time.Millisecond
}

func (t *townScreen) armTownBirds(now time.Time) {
	a := &t.exterior
	if !t.townBirdArtPresent() || a.birdActive {
		return
	}
	if !t.latches.birdDelayReady {
		t.latches.birdDelay = t.nextTownBirdDelay()
		t.latches.birdDelayReady = true
	}
	if now.Sub(a.birdClock) <= t.latches.birdDelay {
		return
	}
	a.birdGroup = t.townAmbientRoll(3)
	a.birdCount = 1 + t.townAmbientRoll(3)
	a.birdProgress = [3]int{}
	a.birdClock = now
	a.birdActive = true
	t.latches.birdDelay = t.nextTownBirdDelay()
	path := "town/birds2.wav"
	if a.birdCount == 1 {
		path = "town/birds1.wav"
	}
	t.exteriorSound(exteriorBird, path)
}

func (t *townScreen) advanceTownStar() {
	a := &t.exterior
	if !a.starActive {
		return
	}
	if a.starCurrent == 0 {
		t.exteriorSound(exteriorStar, "town/stars.wav")
	}
	a.starCurrent++
	if a.starCurrent < townStarFrameCount {
		a.starVisible = t.townStarArtReady()
		return
	}
	t.latches.starTerminalCnt++
	if t.latches.starTerminalCnt >= 10 {
		a.starCurrent = 0
		t.latches.starTerminalCnt = 0
	}
	a.starVisible = false
	a.starActive = false
}

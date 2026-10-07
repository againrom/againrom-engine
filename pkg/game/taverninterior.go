package game

import (
	"image"
	"math/rand"
	"time"

	"againrom/pkg/audio"
	"againrom/pkg/ui"
)

const (
	tavernInteriorCycleInterval  = 100 * time.Millisecond
	tavernInteriorTenderInterval = 83 * time.Millisecond
	tavernInteriorSteamInterval  = 10 * time.Second
	tavernInteriorDelayBase      = 3000
	tavernInteriorRandomRange    = 32768
)

const (
	tavernSoundDrink = iota
	tavernSoundGlotok
	tavernSoundSteam
	tavernSoundWater
	tavernSoundChair
	tavernSoundBreath
	tavernSoundEnter
	tavernSoundCount
)

var tavernInteriorSoundPaths = [tavernSoundCount]string{
	"town/inn/drink.wav",
	"town/inn/glotok.wav",
	"town/inn/steam.wav",
	"town/inn/water.wav",
	"town/inn/chair.wav",
	"town/shop/breath.wav",
	"town/inn/enter.wav",
}

type tavernInteriorPaint struct {
	valid                          bool
	candle, cauldron, tender, mode int
}

// tavernInteriorAnimation is presentation state only. The four family
// indices stay distinct, and breath/drink keep cached-picture indices beside
// their logical indices because TOWN-410 proves they can disagree.
type tavernInteriorAnimation struct {
	ready, active bool

	candleIndex, cauldronIndex int
	breathIndex, breathCached  int
	drinkIndex, drinkCached    int
	mode, direction            int

	cycleLast, tenderLast, steamLast time.Time
	delay                            time.Duration
	random                           *rand.Rand
	paint                            tavernInteriorPaint
	voices                           [tavernSoundCount]audio.Voice
}

func (a *tavernInteriorAnimation) stop(slot int) {
	if a == nil || slot < 0 || slot >= len(a.voices) || a.voices[slot] == nil {
		return
	}
	audio.StopReset(a.voices[slot])
	a.voices[slot] = nil
}

func (a *tavernInteriorAnimation) silence() {
	if a == nil {
		return
	}
	for slot := range a.voices {
		a.stop(slot)
	}
}

func (t *townScreen) inTavernInterior() bool {
	return t != nil && (t.room == roomTavern || t.room == roomTalk && t.dialogueBuilding == TownTavern)
}

func (t *townScreen) resetTavernInterior() {
	if t == nil {
		return
	}
	t.tavernInterior.silence()
	t.tavernInterior = tavernInteriorAnimation{}
}

func (t *townScreen) leaveTavernInterior() {
	if t == nil {
		return
	}
	t.tavernInterior.active = false
	t.tavernInterior.silence()
	if !t.inTavernInterior() && t.audioRoom == roomTavern {
		t.destroyRoomAudio()
	}
}

func (t *townScreen) enterTavernInterior() {
	if t == nil || t.sess == nil {
		return
	}
	t.resetTavernInterior()
	now := t.townAnimationNow()
	a := &t.tavernInterior
	*a = tavernInteriorAnimation{
		ready:      true,
		cycleLast:  now,
		tenderLast: now,
		steamLast:  now,
		random:     rand.New(rand.NewSource(now.UnixNano())),
		paint:      tavernInteriorPaint{valid: true},
	}
	a.delay = t.nextTavernInteriorDelay()
	t.requestTavernInteriorSound(tavernSoundEnter)
}

// TavernInteriorActive is an explicit Againrom pause policy for lifecycle
// arms the accepted research leaves open. A resume rebases all private clocks,
// so time spent unfocused, in a menu or outside the room cannot catch up.
func (t *townScreen) TavernInteriorActive(active bool) {
	if t == nil || t.sess == nil {
		return
	}
	active = active && t.inTavernInterior()
	if active && !t.tavernInterior.ready {
		t.enterTavernInterior()
	}
	a := &t.tavernInterior
	if !active {
		a.active = false
		a.silence()
		return
	}
	if a.active {
		return
	}
	now := t.townAnimationNow()
	a.cycleLast, a.tenderLast, a.steamLast = now, now, now
	a.active = true
}

func (t *townScreen) tavernInteriorRoll() int {
	if draw := t.draws.tavernDraw(); draw != nil {
		v := draw(tavernInteriorRandomRange)
		v %= tavernInteriorRandomRange
		if v < 0 {
			v += tavernInteriorRandomRange
		}
		return v
	}
	return t.tavernInterior.random.Intn(tavernInteriorRandomRange)
}

func (t *townScreen) nextTavernInteriorDelay() time.Duration {
	return time.Duration(tavernInteriorDelayBase+t.tavernInteriorRoll()/16) * time.Millisecond
}

func (t *townScreen) requestTavernInteriorSound(slot int) {
	if t == nil || t.sess == nil || slot < 0 || slot >= len(tavernInteriorSoundPaths) {
		return
	}
	a := &t.tavernInterior
	if voice := a.voices[slot]; voice != nil {
		if voice.Playing() {
			return
		}
		a.stop(slot)
	}
	sample, ok := t.in.SoundBank.namedSample(tavernInteriorSoundPaths[slot])
	if !ok {
		return
	}
	a.voices[slot] = audio.Dispatch(t.roomSoundPlayer(audio.EffectsChannel), sample,
		audio.FixedRequest("tavern-interior", tavernInteriorSoundPaths[slot], audio.EffectsChannel, 128,
			slot == tavernSoundWater, audio.Placement{Left: audio.GainUnit, Right: audio.GainUnit}))
}

func (t *townScreen) tavernInteriorArt() *ui.TavernInteriorArt {
	if t == nil || t.sess == nil || t.in.TownTavernArt.Value() == nil {
		return nil
	}
	return &t.in.TownTavernArt.Value().Interior
}

func completeTavernFamily(frames []image.Image, loaded int) bool {
	if len(frames) != loaded {
		return false
	}
	for _, frame := range frames {
		if frame == nil {
			return false
		}
	}
	return true
}

func tavernInteriorPicture(frames []image.Image, index int) image.Image {
	if index < 0 || index >= len(frames) {
		return nil
	}
	return frames[index]
}

func (t *townScreen) tavernInteriorFrame() ui.TavernInteriorFrame {
	a, art := &t.tavernInterior, t.tavernInteriorArt()
	if art == nil || !a.paint.valid {
		return ui.TavernInteriorFrame{}
	}
	frame := ui.TavernInteriorFrame{
		Candle:   tavernInteriorPicture(art.Candle, a.paint.candle),
		Cauldron: tavernInteriorPicture(art.Cauldron, a.paint.cauldron),
	}
	switch a.paint.mode {
	case 1:
		frame.Tender = tavernInteriorPicture(art.Drink, a.paint.tender)
	case 2:
		frame.Tender = tavernInteriorPicture(art.Breath, a.paint.tender)
	}
	return frame
}

func (t *townScreen) armTavernInterior(now time.Time, art *ui.TavernInteriorArt) {
	a := &t.tavernInterior
	if now.Sub(a.tenderLast) <= a.delay {
		return
	}
	if a.delay/time.Millisecond%2 != 0 {
		if completeTavernFamily(art.Drink, tavernDrinkLoaded) {
			a.mode, a.direction = 1, 1
			return
		}
	} else if completeTavernFamily(art.Breath, tavernBreathLoaded) {
		a.mode = 2
		t.requestTavernInteriorSound(tavernSoundChair)
		t.requestTavernInteriorSound(tavernSoundBreath)
		return
	}
	// Againrom fallback: a selected family missing as a whole does not trap
	// the mode machine. Retain any running other family and retry after a new
	// bounded delay; idle remains idle.
	a.tenderLast = now
	a.delay = t.nextTavernInteriorDelay()
}

func tavernForward(index, cached *int, frames []image.Image) bool {
	if *index < 0 || *index >= len(frames)-1 {
		return false
	}
	next := *index + 1
	if frames[next] == nil {
		return false
	}
	*index, *cached = next, next
	return true
}

func tavernReverse(index, cached *int, frames []image.Image) bool {
	if *index <= 0 || *index >= len(frames) {
		return false
	}
	next := *index - 1
	if frames[next] == nil {
		return false
	}
	*index, *cached = next, next
	return true
}

func (t *townScreen) finishTavernEpisode(now time.Time, clearDirection bool) {
	a := &t.tavernInterior
	a.mode = 0
	if clearDirection {
		a.direction = 0
	}
	a.delay = t.nextTavernInteriorDelay()
	a.tenderLast = now
}

// advanceTavernInterior performs one reached central-child paint. It publishes
// the pictures drawn on this paint before applying either strict-gated step;
// later view reads never advance the controller themselves.
func (t *townScreen) advanceTavernInterior() {
	if t == nil || !t.inTavernInterior() || !t.tavernInterior.ready || !t.tavernInterior.active {
		return
	}
	art := t.tavernInteriorArt()
	if art == nil {
		return
	}
	a := &t.tavernInterior
	now := t.townAnimationNow()
	t.armTavernInterior(now, art)

	a.paint = tavernInteriorPaint{
		valid:    true,
		candle:   a.candleIndex,
		cauldron: a.cauldronIndex,
		mode:     a.mode,
	}
	if a.mode == 1 {
		a.paint.tender = a.drinkCached
	} else if a.mode == 2 {
		a.paint.tender = a.breathCached
	}

	t.requestTavernInteriorSound(tavernSoundWater)
	if now.Sub(a.steamLast) > tavernInteriorSteamInterval {
		a.steamLast = now
		t.requestTavernInteriorSound(tavernSoundSteam)
	}
	if a.mode == 1 && a.drinkIndex == 30 {
		t.requestTavernInteriorSound(tavernSoundDrink)
	}

	if now.Sub(a.cycleLast) > tavernInteriorCycleInterval {
		if completeTavernFamily(art.Candle, tavernCandleLoaded) {
			a.candleIndex = (a.candleIndex + 1) % tavernCandleLoop
		}
		if completeTavernFamily(art.Cauldron, tavernCauldronLoaded) {
			a.cauldronIndex = (a.cauldronIndex + 1) % tavernCauldronLoop
		}
		a.cycleLast = now
	}

	if a.mode == 0 || now.Sub(a.tenderLast) <= tavernInteriorTenderInterval {
		return
	}
	a.tenderLast = now
	switch a.mode {
	case 2:
		if !tavernForward(&a.breathIndex, &a.breathCached, art.Breath) {
			// TOWN-410 resets the index only. The cached last picture and the
			// independent drink direction remain untouched.
			a.breathIndex = 0
			t.finishTavernEpisode(now, false)
		}
	case 1:
		if a.direction > 0 {
			if tavernForward(&a.drinkIndex, &a.drinkCached, art.Drink) {
				return
			}
			a.direction = -1
			if tavernReverse(&a.drinkIndex, &a.drinkCached, art.Drink) {
				return
			}
		} else if tavernReverse(&a.drinkIndex, &a.drinkCached, art.Drink) {
			return
		}
		t.requestTavernInteriorSound(tavernSoundGlotok)
		t.finishTavernEpisode(now, true)
	}
}

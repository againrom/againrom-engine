package game

import (
	"slices"
	"testing"
	"time"

	"againrom/pkg/audio"
	"againrom/pkg/render/terrain"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

// The watched units; each has its own attacker. The healed one regenerates
// and is also struck without damage.
const (
	pacedHurtOnce  sim.EntityID = 0
	pacedHurtTwice sim.EntityID = 1
	pacedFalls     sim.EntityID = 2
	pacedHealed    sim.EntityID = 3
)

// One class per watched unit: strike, easy, hard, fall at 1 to 4.
var pacedCueClasses = map[int32]UnitSound{
	21: {Slots: []int32{0, 211, 221, 231, 241}},
	22: {Slots: []int32{0, 212, 222, 232, 242}},
	23: {Slots: []int32{0, 213, 223, 233, 243}},
	24: {Slots: []int32{0, 214, 224, 234, 244}},
}

// HeadlessStep's clock step.
const pacedCueFrameLength = 20 * time.Millisecond

func pacedCueWorld(t *testing.T) (mw *mapWorld, effects, speech *hurtVoiceRecorder) {
	t.Helper()
	const side = 20
	ents := []sim.Entity{
		{ID: pacedHurtOnce, X: 3, Y: 3, Owner: sim.SelfSlot, Class: 21, HP: 1000, MaxHP: 1000},
		{ID: pacedHurtTwice, X: 3, Y: 7, Owner: sim.SelfSlot, Class: 22, HP: 1000, MaxHP: 1000},
		{ID: pacedFalls, X: 3, Y: 11, Owner: sim.SelfSlot, Class: 23, HP: 40, MaxHP: 100},
		{ID: pacedHealed, X: 3, Y: 15, Owner: sim.SelfSlot, Class: 24, HP: 1000, MaxHP: 1000, HealthRegenPeriod: 1},
		{ID: 4, X: 4, Y: 3, Owner: 2, Class: 1, HP: 1000, MaxHP: 1000, ScanRange: 2, DamageBase: 7, AlwaysHits: true, AttackCharge: 30, AttackRelax: 30},
		{ID: 5, X: 4, Y: 7, Owner: 2, Class: 1, HP: 1000, MaxHP: 1000, ScanRange: 2, DamageBase: 5, AlwaysHits: true, AttackCharge: 2, AttackRelax: 2},
		{ID: 6, X: 4, Y: 11, Owner: 2, Class: 1, HP: 1000, MaxHP: 1000, ScanRange: 2, DamageBase: 9, AlwaysHits: true, AttackCharge: 3, AttackRelax: 3},
		{ID: 7, X: 4, Y: 15, Owner: 2, Class: 1, HP: 1000, MaxHP: 1000, ScanRange: 2, DamageBase: 6, AlwaysHits: true, AttackCharge: 4, AttackRelax: 4},
		{ID: 8, X: 2, Y: 15, Owner: 2, Class: 1, HP: 1000, MaxHP: 1000, ScanRange: 2, DamageBase: 0, AlwaysHits: true, AttackCharge: 3, AttackRelax: 3},
	}
	var rel sim.Relations
	rel.Set(2, sim.SelfSlot, 1)
	w, err := sim.NewRelatedWorld(1, sim.Bounds{Width: side, Height: side}, sim.ModeCanonical, sim.Terrain{}, ents, nil, rel)
	if err != nil {
		t.Fatal(err)
	}
	art := worldFixtureArt(16, 16, 8, 14, 4, 4, 64)
	art.Anim = swingAnimDesc()
	art.Corpse = art
	v, err := ui.NewViewer("paced cues", terrain.Grid{Width: side, Height: side, Tiles: make([]uint16, side*side)}, &terrain.Tileset{})
	if err != nil {
		t.Fatal(err)
	}
	classes := map[int32]*terrain.UnitClass{1: art}
	bank := &SoundBank{cache: map[int]soundCacheEntry{}, named: map[string]soundCacheEntry{}}
	for class, snd := range pacedCueClasses {
		classes[class] = art
		for _, slot := range snd.Slots[1:] {
			bank.cache[int(slot)] = soundCacheEntry{sample: audio.Sample{Rate: audio.DeviceRate, PCM: []int16{int16(slot)}}, ok: true}
		}
	}
	mw = newMapWorld(w, nil, &terrain.UnitSet{Classes: classes}, v)
	mw.sounds = pacedCueClasses
	effects, speech = &hurtVoiceRecorder{}, &hurtVoiceRecorder{}
	v.SetAudio(effects, bank)
	v.SetSpeechAudio(speech)
	mw.push()
	return mw, effects, speech
}

type pacedCueVictim struct {
	blows uint32
	life  uint8
	hp    int32
}

type pacedCueRun struct {
	effects, speech []hurtVoicePlay
	numerals        []int
	victims         [][4]pacedCueVictim
	tick, hash      uint64
}

// oneByOne splits each frame into paced calls of at most one tick.
func runPacedCues(t *testing.T, periodUS, frames int, oneByOne bool) pacedCueRun {
	t.Helper()
	mw, effects, speech := pacedCueWorld(t)
	period := time.Duration(periodUS) * time.Microsecond
	now, prev := time.Unix(1000, 0), time.Time{}
	tick := func() {
		mw.frameAdvance(func() {
			if oneByOne && !prev.IsZero() {
				for at := prev.Add(period); at.Before(now); at = at.Add(period) {
					mw.paceTo(at)
				}
			}
			mw.paceTo(now)
		})
		prev, now = now, now.Add(pacedCueFrameLength)
	}
	a := ui.NewApp("paced cues", nil, nil, nil)
	if err := a.OpenMission(func() (*ui.Viewer, ui.MapTick, ui.MapOrder, ui.MapCadence, ui.MapAffect, ui.MapAdvance, ui.MapAttack, ui.MapGrab, ui.MapStance, ui.MapMarch, error) {
		return mw.view, tick, mw.enqueue, mw.setCadenceMode, mw.affect, nil, mw.attackOrCast, mw.grab, mw.stance, mw.march, nil
	}); err != nil {
		t.Fatal(err)
	}
	mw.setCadence(periodUS, false)
	var run pacedCueRun
	for frame := 0; frame < frames; frame++ {
		effects.frame, speech.frame = frame, frame
		if err := a.HeadlessStep(); err != nil {
			t.Fatal(err)
		}
		_, live := mw.view.DamageNumerals()
		run.numerals = append(run.numerals, live)
		var victims [4]pacedCueVictim
		for id := range victims {
			e, _ := mw.world.Entity(sim.EntityID(id))
			victims[id] = pacedCueVictim{blows: mw.blows[sim.EntityID(id)], life: lifeOf(&e), hp: e.HP}
		}
		run.victims = append(run.victims, victims)
	}
	run.effects, run.speech = effects.plays, speech.plays
	run.tick, run.hash = mw.world.Tick(), mw.world.Hash()
	return run
}

func (r pacedCueRun) pacedCueSeen(id sim.EntityID, want func(before, after pacedCueVictim) bool) bool {
	for f := 1; f < len(r.victims); f++ {
		if want(r.victims[f-1][id], r.victims[f][id]) {
			return true
		}
	}
	return false
}

func pacedCuePlayed(plays []hurtVoicePlay, srcs ...string) bool {
	for _, p := range plays {
		if slices.Contains(srcs, p.src) {
			return true
		}
	}
	return false
}

// A paced call of many ticks plays the cues and numerals of the same ticks
// projected one by one.
func TestAPacedCallOfManyTicksDeliversEveryCue(t *testing.T) {
	for _, tc := range []struct{ rate, frames int }{{1024, 80}, {512, 80}, {32, 120}} {
		periodUS := 1_000_000 / tc.rate
		paced := runPacedCues(t, periodUS, tc.frames, false)
		single := runPacedCues(t, periodUS, tc.frames, true)
		if paced.tick != single.tick || paced.hash != single.hash {
			t.Fatalf("%d/s: tick %d hash %x, one by one tick %d hash %x", tc.rate, paced.tick, paced.hash, single.tick, single.hash)
		}
		if !slices.Equal(paced.speech, single.speech) {
			t.Errorf("%d/s: speech %v, one by one %v", tc.rate, paced.speech, single.speech)
		}
		if !slices.Equal(paced.effects, single.effects) {
			t.Errorf("%d/s: effects %v, one by one %v", tc.rate, paced.effects, single.effects)
		}
		if !slices.Equal(paced.numerals, single.numerals) {
			t.Errorf("%d/s: live numerals %v, one by one %v", tc.rate, paced.numerals, single.numerals)
		}
		if !slices.Equal(paced.victims, single.victims) {
			t.Errorf("%d/s: the watched units differ between the drivers", tc.rate)
		}
		for _, cue := range []struct {
			name  string
			plays []hurtVoicePlay
			srcs  []string
		}{
			{"hurt once", paced.speech, []string{"class slot 221", "class slot 231"}},
			{"hurt several times", paced.speech, []string{"class slot 222", "class slot 232"}},
			{"fall", paced.speech, []string{"class slot 243"}},
			{"healed unit hurt", paced.speech, []string{"class slot 224", "class slot 234"}},
			{"strike without damage", paced.effects, []string{"class slot 214"}},
		} {
			if !pacedCuePlayed(cue.plays, cue.srcs...) {
				t.Errorf("%d/s: no %s cue", tc.rate, cue.name)
			}
		}
		if slices.Max(paced.numerals) == 0 {
			t.Errorf("%d/s: no damage numeral", tc.rate)
		}
		if tc.rate < 512 {
			continue
		}
		for _, frame := range []struct {
			name string
			id   sim.EntityID
			want func(before, after pacedCueVictim) bool
		}{
			{"one blow", pacedHurtOnce, func(b, a pacedCueVictim) bool { return a.blows-b.blows == 1 }},
			{"several blows", pacedHurtTwice, func(b, a pacedCueVictim) bool { return a.blows-b.blows >= 2 }},
			{"a blow and the fall", pacedFalls, func(b, a pacedCueVictim) bool {
				return a.blows > b.blows && b.life == ui.LifeAlive && a.life != ui.LifeAlive
			}},
			{"a blow and a regeneration past it", pacedHealed, func(b, a pacedCueVictim) bool { return a.blows > b.blows && a.hp > b.hp }},
		} {
			if !paced.pacedCueSeen(frame.id, frame.want) {
				t.Errorf("%d/s: no frame holds %s", tc.rate, frame.name)
			}
		}
	}
}

// Paced calls reach the digest of the same ticks projected one by one.
func TestPacedTicksReachTheDigestOfTicksProjectedOneByOne(t *testing.T) {
	for _, rate := range []int{1024, 512} {
		paced, _, _ := pacedCueWorld(t)
		single, _, _ := pacedCueWorld(t)
		paced.setCadence(1_000_000/rate, false)
		at := time.Unix(1000, 0)
		paced.paceTo(at)
		for call := 0; call < 6; call++ {
			at = at.Add(37 * time.Millisecond)
			n, _ := paced.paceTo(at)
			if n < 2 {
				t.Fatalf("%d/s: call %d fired %d ticks", rate, call, n)
			}
			for i := 0; i < n; i++ {
				single.tick()
			}
			if paced.world.Tick() != single.world.Tick() || paced.world.Hash() != single.world.Hash() {
				t.Fatalf("%d/s: call %d: tick %d hash %x, one by one tick %d hash %x", rate, call,
					paced.world.Tick(), paced.world.Hash(), single.world.Tick(), single.world.Hash())
			}
		}
		if e, _ := paced.world.Entity(pacedFalls); e.Alive() || paced.blows[pacedHurtTwice] < 2 {
			t.Fatalf("%d/s: the fight landed too little: falls alive %v, %d blows", rate, e.Alive(), paced.blows[pacedHurtTwice])
		}
	}
}

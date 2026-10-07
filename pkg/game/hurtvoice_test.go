package game

import (
	"fmt"
	"slices"
	"testing"

	"againrom/pkg/audio"
	"againrom/pkg/data"
	"againrom/pkg/render/terrain"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

// hurtVoiceBanks are the eight human voice banks in the order their markers
// are numbered.
var hurtVoiceBanks = []string{"mf_hero", "ff_hero", "m_mage", "f_mage", "mf_merc", "ff_merc", "m_peasant", "f_peasant"}

// hurtVoiceClasses are two drawn classes' Sound arrays: the archer's shipped
// array on both roots, whose wound and death samples a female figure also
// plays, and a stand-in fighter's.
var hurtVoiceClasses = map[int32]UnitSound{
	14: {Slots: []int32{130, 211, 225, 235, 245}},
	2:  {Slots: []int32{131, 212, 222, 223, 241}},
}

// hurtVoiceBank is a sound bank whose every sample carries one PCM value
// naming its source: a numbered slot carries its own number, and a bank leaf
// carries 1000 times the bank's place in hurtVoiceBanks, plus one, plus 1 for
// easy, 2 for hard or 3 for die.
func hurtVoiceBank() *SoundBank {
	b := &SoundBank{cache: map[int]soundCacheEntry{}, named: map[string]soundCacheEntry{}}
	for _, class := range hurtVoiceClasses {
		for _, slot := range class.Slots {
			b.cache[int(slot)] = soundCacheEntry{sample: audio.Sample{Rate: audio.DeviceRate, PCM: []int16{int16(slot)}}, ok: true}
		}
	}
	for i, bank := range hurtVoiceBanks {
		for leaf, name := range []string{"easy", "hard", "die"} {
			marker := int16(1000*(i+1) + leaf + 1)
			b.named[bank+"/"+name+".wav"] = soundCacheEntry{sample: audio.Sample{Rate: audio.DeviceRate, PCM: []int16{marker}}, ok: true}
		}
	}
	return b
}

// hurtVoiceSource names the source of one recorded sample.
func hurtVoiceSource(s audio.Sample) string {
	if len(s.PCM) != 1 {
		return fmt.Sprintf("an unmarked sample of %d frames", len(s.PCM))
	}
	m := int(s.PCM[0])
	if m < 1000 {
		return fmt.Sprintf("class slot %d", m)
	}
	bank, leaf := m/1000-1, m%1000-1
	if bank >= len(hurtVoiceBanks) || leaf < 0 || leaf > 2 {
		return fmt.Sprintf("marker %d", m)
	}
	return hurtVoiceBanks[bank] + "/" + []string{"easy", "hard", "die"}[leaf] + ".wav"
}

// hurtVoicePlay is one sample the viewer played: the frame it played on and
// its source.
type hurtVoicePlay struct {
	frame int
	src   string
}

// hurtVoiceRecorder is the viewer's effects device; frame is set by the
// driving loop before each frame.
type hurtVoiceRecorder struct {
	frame int
	plays []hurtVoicePlay
}

func (r *hurtVoiceRecorder) Play(s audio.Sample, _ audio.Placement) {
	r.plays = append(r.plays, hurtVoicePlay{r.frame, hurtVoiceSource(s)})
}

func (r *hurtVoiceRecorder) RequestSample(s audio.Sample, request audio.Request) audio.Voice {
	r.Play(s, request.Placement)
	return nil
}

// hurtVoiceFight opens a mission screen over a hand-built world holding hero
// at (4, 4) and one hostile fighter beside him, whose ordinary automatic
// attack is the only source of damage: damage per blow, with charge and relax
// ticks. Every running frame is one tick, and the viewer's sounds go to the
// returned recorder.
func hurtVoiceFight(t *testing.T, hero sim.Entity, fig figureID, damage, charge int32) (*mapWorld, *ui.App, *hurtVoiceRecorder) {
	t.Helper()
	enemy := sim.Entity{ID: 1, X: 5, Y: 4, Owner: 2, Class: 1, HP: 1000, MaxHP: 1000,
		ScanRange: 5, DamageBase: damage, AlwaysHits: true, AttackCharge: charge, AttackRelax: charge}
	var rel sim.Relations
	rel.Set(2, sim.SelfSlot, 1)
	w, err := sim.NewRelatedWorld(1, sim.Bounds{Width: swingW, Height: swingH}, sim.ModeCanonical,
		sim.Terrain{}, []sim.Entity{hero, enemy}, nil, rel)
	if err != nil {
		t.Fatal(err)
	}
	art := worldFixtureArt(16, 16, 8, 14, 4, 4, 64)
	art.Anim = swingAnimDesc()
	art.Corpse = art
	v, err := ui.NewViewer("hurt voice", terrain.Grid{
		Width: swingW, Height: swingH, Tiles: make([]uint16, swingW*swingH),
	}, &terrain.Tileset{})
	if err != nil {
		t.Fatal(err)
	}
	mw := newMapWorld(w, nil, &terrain.UnitSet{Classes: map[int32]*terrain.UnitClass{1: art, hero.Class: art}}, v)
	mw.sounds = hurtVoiceClasses
	if fig != (figureID{}) {
		mw.figures = map[sim.EntityID]figureID{hero.ID: fig}
	}
	rec := &hurtVoiceRecorder{}
	v.SetAudio(rec, hurtVoiceBank())
	v.SetSpeechAudio(rec)
	mw.push()
	a := ui.NewApp("hurt voice", nil, nil, nil)
	if err := a.OpenMission(func() (*ui.Viewer, ui.MapTick, ui.MapOrder, ui.MapCadence, ui.MapAffect, ui.MapAdvance, ui.MapAttack, ui.MapGrab, ui.MapStance, ui.MapMarch, error) {
		return v, mw.deterministicFrame, mw.enqueue, mw.setCadenceMode, mw.affect, nil, mw.attackOrCast, mw.grab, mw.stance, mw.march, nil
	}); err != nil {
		t.Fatal(err)
	}
	return mw, a, rec
}

// TestDanasWithABowHurtsInTheMaleHeroVoice is the owner's report: Danas, the
// male fighter hero, draws as the archer class while he holds a bow, and
// that class names the female wound samples. Struck through ordinary
// combat, every wound he voices must still come from mf_hero.
func TestDanasWithABowHurtsInTheMaleHeroVoice(t *testing.T) {
	danas := sim.Entity{ID: 0, X: 4, Y: 4, Owner: sim.SelfSlot, TypeID: sim.HeroTypeID(false, false),
		Class: 14, Humanoid: true, HP: 100, MaxHP: 100}
	mw, a, rec := hurtVoiceFight(t, danas, figureID{Dir: data.FigureDirManFighter, Hero: true}, 7, 8)
	for rec.frame = 0; rec.frame < 150; rec.frame++ {
		if err := a.HeadlessStep(); err != nil {
			t.Fatal(err)
		}
	}
	hero, _ := mw.world.Entity(0)
	if hero.HP >= 100 {
		t.Fatalf("the fixture landed no blow: Danas is at %d", hero.HP)
	}
	if len(rec.plays) == 0 {
		t.Fatalf("Danas fell to %d and voiced no wound", hero.HP)
	}
	for _, p := range rec.plays {
		if p.src != "mf_hero/easy.wav" && p.src != "mf_hero/hard.wav" {
			t.Errorf("Danas at %d/100 voiced %s, want an mf_hero wound", hero.HP, p.src)
		}
	}
}

// TestFastBlowsWaitOutTheWoundGate lands a blow every few ticks on a hero
// who stays above half health. A wound sounds only once 75 frames, 1500 ms,
// have passed since the last wound that sounded, and then the first blow to
// land sounds (ANIM-094).
func TestFastBlowsWaitOutTheWoundGate(t *testing.T) {
	danas := sim.Entity{ID: 0, X: 4, Y: 4, Owner: sim.SelfSlot, TypeID: sim.HeroTypeID(false, false),
		Class: 14, Humanoid: true, HP: 400, MaxHP: 400}
	mw, a, rec := hurtVoiceFight(t, danas, figureID{Dir: data.FigureDirManFighter, Hero: true}, 3, 8)
	var blows []int
	health := danas.HP
	for rec.frame = 0; rec.frame < 400; rec.frame++ {
		if err := a.HeadlessStep(); err != nil {
			t.Fatal(err)
		}
		if e, _ := mw.world.Entity(0); e.HP < health {
			blows, health = append(blows, rec.frame), e.HP
		}
	}
	var want []hurtVoicePlay
	last := -1
	for _, f := range blows {
		if last < 0 || f-last >= 75 {
			want, last = append(want, hurtVoicePlay{f, "mf_hero/easy.wav"}), f
		}
	}
	t.Logf("blows on frames %v; heard %v", blows, rec.plays)
	if len(want) < 3 || len(blows) < 2*len(want) {
		t.Fatalf("the fixture landed %d blows, %d past the gate: want several blows inside each gap", len(blows), len(want))
	}
	if !slices.Equal(rec.plays, want) {
		t.Fatalf("heard %v\nwant  %v", rec.plays, want)
	}
}

// TestEachHumanTakesTheBankOfItsOwnSexAndClass is the selector (ANIM-096,
// HERO-APPEAR-055) as the push states it: a hero keeps his bank whatever he
// wields or is drawn as, the healer takes the female mage bank, a weapon
// moves a person from the peasant pair to the mercenary pair, a creature has
// none, and every Sound array stays the drawn class's own.
func TestEachHumanTakesTheBankOfItsOwnSexAndClass(t *testing.T) {
	type person struct {
		name   string
		typeID int32
		class  int32
		dir    data.FigureDir
		weapon bool
		want   string
	}
	people := []person{
		{"Danas with a bow", sim.HeroTypeID(false, false), 14, data.FigureDirManFighter, true, "mf_hero"},
		{"Danas unarmed", sim.HeroTypeID(false, false), 2, data.FigureDirManFighter, false, "mf_hero"},
		{"Naira with a bow", sim.HeroTypeID(false, true), 14, data.FigureDirWomanFighter, true, "ff_hero"},
		{"Fergard", sim.HeroTypeID(true, false), 2, data.FigureDirManMage, true, "m_mage"},
		{"Reniesta", sim.HeroTypeID(true, true), 2, data.FigureDirWomanMage, false, "f_mage"},
		{"the healer", 0x18, 14, data.FigureDirWomanMage, true, "f_mage"},
		{"a male mage placed without a staff", 0x17, 2, data.FigureDirManMage, false, "m_mage"},
		{"an armed man", 3, 2, data.FigureDirManFighter, true, "mf_merc"},
		{"an armed woman", 14, 14, data.FigureDirWomanFighter, true, "ff_merc"},
		{"an unarmed man", 1, 2, data.FigureDirManFighter, false, "m_peasant"},
		{"an unarmed woman", 1, 14, data.FigureDirWomanFighter, false, "f_peasant"},
		{"a creature", 0x40, 2, "", false, ""},
	}
	var ents []sim.Entity
	var stocks []sim.Stock
	figures := map[sim.EntityID]figureID{}
	for i, p := range people {
		id := sim.EntityID(i)
		ents = append(ents, sim.Entity{ID: id, X: int32(i % swingW), Y: int32(i / swingW), TypeID: p.typeID,
			Class: p.class, Humanoid: p.want != "", HP: 10, MaxHP: 10})
		if p.weapon {
			stocks = append(stocks, sim.Stock{ID: id, Equipped: [sim.EquipSlots]uint16{0: eqSwordCode}})
		}
		if p.want != "" {
			figures[id] = figureID{Dir: p.dir, Hero: data.FigureIsHero(p.typeID)}
		}
	}
	w, err := sim.NewStockedWorld(1, sim.Bounds{Width: swingW, Height: swingH}, sim.ModeCanonical, sim.Terrain{},
		ents, nil, sim.Relations{}, nil, stocks)
	if err != nil {
		t.Fatal(err)
	}
	v, err := ui.NewViewer("hurt voice banks", terrain.Grid{
		Width: swingW, Height: swingH, Tiles: make([]uint16, swingW*swingH),
	}, &terrain.Tileset{})
	if err != nil {
		t.Fatal(err)
	}
	mw := newMapWorld(w, nil, &terrain.UnitSet{}, v)
	mw.sounds, mw.figures = hurtVoiceClasses, figures
	for _, d := range mw.entityDraws() {
		p := people[d.ID]
		if d.Voice != p.want {
			t.Errorf("%s voices from %q, want %q", p.name, d.Voice, p.want)
		}
		if want := soundSlots(hurtVoiceClasses, p.class); !slices.Equal(d.Sound, want) {
			t.Errorf("%s carries Sound %v, want the drawn class's own %v", p.name, d.Sound, want)
		}
	}
}

// hurtVoiceFightPlays runs hero through the fight hurtVoiceFight builds, the
// attacker landing 20 points every 82 or more ticks, past the 1500 ms gate,
// so every blow is voiced: the easy source while the health held before it is
// at least half, the hard source below half and on the fallen body, and the
// die source on the frame of the fall. Nothing sounds once the body passes
// -10, where the attack stops, nor while it decays. It returns what was heard,
// what must be heard, with each event's source named by source, and the
// hero's health after each frame.
func hurtVoiceFightPlays(t *testing.T, hero sim.Entity, fig figureID, source func(leaf string) string) (heard, want []hurtVoicePlay, levels []int32) {
	t.Helper()
	mw, a, rec := hurtVoiceFight(t, hero, fig, 20, 40)
	for rec.frame = 0; rec.frame < 900; rec.frame++ {
		if err := a.HeadlessStep(); err != nil {
			t.Fatal(err)
		}
		e, _ := mw.world.Entity(0)
		levels = append(levels, e.HP)
	}
	// Above -10 only a blow lowers this body's health: the decay walk
	// starts below zero, and the first blow on the fallen body takes
	// it from 0 to -20.
	for f := 1; f < len(levels); f++ {
		stored, now := levels[f-1], levels[f]
		if now >= stored || stored <= -10 {
			continue
		}
		leaf := "easy"
		if stored < 50 {
			leaf = "hard"
		}
		want = append(want, hurtVoicePlay{f, source(leaf)})
		if stored > 0 && now <= 0 {
			want = append(want, hurtVoicePlay{f, source("die")})
		}
	}
	if levels[len(levels)-1] > -10 {
		t.Fatalf("the fight stopped at %d, want the body past -10", levels[len(levels)-1])
	}
	return rec.plays, want, levels
}

// TestAFightVoicesEveryBlowAndTheFallFromTheOwnBank runs one ordinary fight
// per hero and for the healer, each voiced from its own bank.
func TestAFightVoicesEveryBlowAndTheFallFromTheOwnBank(t *testing.T) {
	for _, tc := range []struct {
		name   string
		typeID int32
		class  int32
		dir    data.FigureDir
		bank   string
	}{
		{"Danas with a bow", sim.HeroTypeID(false, false), 14, data.FigureDirManFighter, "mf_hero"},
		{"Danas unarmed", sim.HeroTypeID(false, false), 2, data.FigureDirManFighter, "mf_hero"},
		{"Naira", sim.HeroTypeID(false, true), 14, data.FigureDirWomanFighter, "ff_hero"},
		{"Fergard", sim.HeroTypeID(true, false), 2, data.FigureDirManMage, "m_mage"},
		{"Reniesta", sim.HeroTypeID(true, true), 2, data.FigureDirWomanMage, "f_mage"},
		{"the healer", 0x18, 14, data.FigureDirWomanMage, "f_mage"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			hero := sim.Entity{ID: 0, X: 4, Y: 4, Owner: sim.SelfSlot, TypeID: tc.typeID,
				Class: tc.class, Humanoid: true, HP: 100, MaxHP: 100}
			heard, want, levels := hurtVoiceFightPlays(t, hero, figureID{Dir: tc.dir, Hero: data.FigureIsHero(tc.typeID)},
				func(leaf string) string { return tc.bank + "/" + leaf + ".wav" })
			t.Logf("health %d..%d over %d frames; heard %v", levels[0], levels[len(levels)-1], len(levels), heard)
			if !slices.Equal(heard, want) {
				t.Fatalf("heard %v\nwant  %v", heard, want)
			}
		})
	}
}

// TestAClassVoicedFightVoicesTheFallFromItsDrawnClass runs the same fight for
// creatures with no figure. Its wounds play indexes 2 and 3 of the drawn
// class's Sound array and its fall index 4, on the frame it falls (ANIM-094,
// ANIM-095). Each class's array is the drawn class's own, whatever the
// creature is.
func TestAClassVoicedFightVoicesTheFallFromItsDrawnClass(t *testing.T) {
	for _, tc := range []struct {
		name  string
		class int32
		slots [3]int32
	}{
		{"a creature drawn as class 14", 14, [3]int32{225, 235, 245}},
		{"a creature drawn as class 2", 2, [3]int32{222, 223, 241}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			creature := sim.Entity{ID: 0, X: 4, Y: 4, Owner: sim.SelfSlot, TypeID: 0x40,
				Class: tc.class, HP: 100, MaxHP: 100}
			heard, want, levels := hurtVoiceFightPlays(t, creature, figureID{}, func(leaf string) string {
				return fmt.Sprintf("class slot %d", tc.slots[slices.Index([]string{"easy", "hard", "die"}, leaf)])
			})
			t.Logf("health %d..%d over %d frames; heard %v", levels[0], levels[len(levels)-1], len(levels), heard)
			if !slices.Equal(heard, want) {
				t.Fatalf("heard %v\nwant  %v", heard, want)
			}
			fall := -1
			for f := 1; f < len(levels) && fall < 0; f++ {
				if levels[f-1] > 0 && levels[f] <= 0 {
					fall = f
				}
			}
			if fall < 0 {
				t.Fatalf("the creature never fell: health %d..%d", levels[0], levels[len(levels)-1])
			}
			at := hurtVoicePlay{fall, fmt.Sprintf("class slot %d", tc.slots[2])}
			if !slices.Contains(heard, at) {
				t.Fatalf("nothing requested %v on the frame of the fall; heard %v", at, heard)
			}
		})
	}
}

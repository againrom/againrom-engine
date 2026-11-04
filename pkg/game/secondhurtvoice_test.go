package game

import (
	"fmt"
	"slices"
	"testing"

	"againrom/pkg/audio"
	"againrom/pkg/base"
	"againrom/pkg/data"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
)

// secondHurtVoices are the second-game voices the wound fixtures name, in
// the order their markers are numbered.
var secondHurtVoices = []string{"aldor", "galinel", "igles", "troll", "f_peasant", "m_peasant"}

func TestSecondGameExplicitZeroWordUsesGenericHurtVoice(t *testing.T) {
	mw, _, _ := hurtVoiceFight(t, sim.Entity{ID: 0, X: 4, Y: 4, Owner: sim.SelfSlot, TypeID: 0x40, Class: 14, HP: 100, MaxHP: 100}, figureID{}, 20, 40)
	e, _ := mw.world.Entity(0)
	second := base.GameROM2.Edition()
	heard, want, _ := secondHurtFightPlays(t, e, figureID{}, func(mw *mapWorld) {
		mw.mission = &missionNotices{table: &mapload.Table{Edition: &second}, npcKeys: map[sim.EntityID]uint16{0: 0}}
	}, secondHurtSource("m_peasant"))
	if !slices.Equal(heard, want) {
		t.Fatalf("explicit word 0 heard %v, want %v", heard, want)
	}
}

// secondHurtBank holds the drawn classes' slots of hurtVoiceBank and, for
// each voice, easy, hard and dead under characters/, voice v's leaf l
// (1 easy, 2 hard, 3 dead) carrying marker 9000 + 10v + l.
func secondHurtBank() *SoundBank {
	b := hurtVoiceBank()
	b.named = map[string]soundCacheEntry{}
	for v, voice := range secondHurtVoices {
		for l, leaf := range []string{"easy", "hard", "dead"} {
			b.named["characters/"+voice+"/"+leaf+".wav"] = soundCacheEntry{sample: audio.Sample{Rate: audio.DeviceRate, PCM: []int16{int16(9000 + 10*v + l + 1)}}, ok: true}
		}
	}
	return b
}

// secondHurtSource names voice's recording of a fight leaf as the recorder
// reports it.
func secondHurtSource(voice string) func(leaf string) string {
	v := slices.Index(secondHurtVoices, voice)
	return func(leaf string) string {
		l := slices.Index([]string{"easy", "hard", "die"}, leaf)
		return fmt.Sprintf("marker %d", 9000+10*v+l+1)
	}
}

// A second-game unit's wounds and fall play easy, hard and dead of its
// voice (R2-ENGINE-721, R2-ENGINE-718) on the transitions of R2-ENGINE-804:
// easy for a blow struck at half health or more, hard below half and on the
// fallen body above -10, dead on the fall; nothing at -10 or below. The test
// admits a person and any unit whose identity word is at most 23, so a
// creature hire and a creature placed with word 21 voice their wounds; a
// creature with no word plays its drawn class's Sound array.
func TestSecondGameWoundsPlayTheUnitsOwnVoice(t *testing.T) {
	second := base.GameROM2.Edition()
	for _, tc := range []struct {
		name   string
		typeID int32
		dir    data.FigureDir
		member *mapload.PartyMember
		word   uint16
		voice  string
	}{
		{"a male fighter hero", sim.HeroTypeID(false, false), data.FigureDirManFighter, &voiceHeroMember, 0, "aldor"},
		{"a female mage hero", sim.HeroTypeID(true, true), data.FigureDirWomanMage, &voiceHeroMember, 0, "galinel"},
		{"companion 22", sim.HeroTypeID(false, false), data.FigureDirManFighter, ptr(voiceCompanionMember(22)), 0, "igles"},
		{"hire 4, a creature", 0x40, "", ptr(voiceHireMember(4, true)), 0, "troll"},
		{"a creature placed with word 21", 0x40, "", nil, 21, "aldor"},
		{"a creature hire 9", 0x40, "", ptr(voiceHireMember(9, true)), 0, "m_peasant"},
		{"a placed unarmed woman", 1, data.FigureDirWomanFighter, nil, 100, "f_peasant"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			hero := sim.Entity{ID: 0, X: 4, Y: 4, Owner: sim.SelfSlot, TypeID: tc.typeID,
				Class: 14, Humanoid: tc.dir != "", HP: 100, MaxHP: 100}
			var fig figureID
			if tc.dir != "" {
				fig = figureID{Dir: tc.dir, Hero: data.FigureIsHero(tc.typeID)}
			}
			heard, want, levels := secondHurtFightPlays(t, hero, fig, func(mw *mapWorld) {
				m := &missionNotices{table: &mapload.Table{Edition: &second}}
				if tc.member != nil {
					m.party, m.ids = []mapload.PartyMember{*tc.member}, []sim.EntityID{0}
				} else {
					m.npcKeys = map[sim.EntityID]uint16{0: tc.word}
				}
				mw.mission = m
			}, secondHurtSource(tc.voice))
			t.Logf("health %d..%d over %d frames; heard %v", levels[0], levels[len(levels)-1], len(levels), heard)
			if !slices.Equal(heard, want) {
				t.Fatalf("heard %v\nwant  %v", heard, want)
			}
		})
	}
	t.Run("a creature with no word", func(t *testing.T) {
		creature := sim.Entity{ID: 0, X: 4, Y: 4, Owner: sim.SelfSlot, TypeID: 0x40, Class: 14, HP: 100, MaxHP: 100}
		slots := map[string]int32{"easy": 225, "hard": 235, "die": 245}
		heard, want, _ := secondHurtFightPlays(t, creature, figureID{}, func(mw *mapWorld) {
			mw.mission = &missionNotices{table: &mapload.Table{Edition: &second}}
		}, func(leaf string) string { return fmt.Sprintf("class slot %d", slots[leaf]) })
		if !slices.Equal(heard, want) {
			t.Fatalf("heard %v\nwant  %v", heard, want)
		}
	})
}

// secondHurtFightPlays is hurtVoiceFightPlays over the second-game bank,
// with setup run on the world before the first frame.
func secondHurtFightPlays(t *testing.T, hero sim.Entity, fig figureID, setup func(*mapWorld), source func(leaf string) string) (heard, want []hurtVoicePlay, levels []int32) {
	t.Helper()
	mw, a, rec := hurtVoiceFight(t, hero, fig, 20, 40)
	setup(mw)
	mw.view.SetAudio(rec, secondHurtBank())
	mw.view.SetSpeechAudio(rec)
	for rec.frame = 0; rec.frame < 900; rec.frame++ {
		if err := a.HeadlessStep(); err != nil {
			t.Fatal(err)
		}
		e, _ := mw.world.Entity(0)
		levels = append(levels, e.HP)
	}
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

package game

import (
	"fmt"
	"slices"
	"strings"
	"testing"

	"againrom/pkg/data"
	"againrom/pkg/herokind"
	"againrom/pkg/render/terrain"
	"againrom/pkg/sim"
	"againrom/pkg/town"
	"againrom/pkg/ui"
)

func TestSpeechHeroAndMessageAudienceKeepTheirBaseEncoding(t *testing.T) {
	rows := []struct {
		kind         herokind.Kind
		female, mage bool
		index, cell  int
		bit          ui.HeroAudience
		letter, bank string
	}{
		{herokind.MaleFighter, false, false, 0, 3, 8, "f", "mf_hero"},
		{herokind.FemaleFighter, true, false, 1, 1, 2, "a", "ff_hero"},
		{herokind.MaleMage, false, true, 2, 2, 4, "m", "m_mage"},
		{herokind.FemaleMage, true, true, 3, 0, 1, "s", "f_mage"},
	}
	original := townTalkRules[town.TalkSection]
	t.Cleanup(func() { townTalkRules[town.TalkSection] = original })
	for _, r := range rows {
		t.Run(r.kind.String(), func(t *testing.T) {
			k := herokind.Of(r.mage, r.female)
			decoded := herokind.Kind(r.index)
			if k.Index() != r.index || decoded.Female() != r.female || decoded.Mage() != r.mage {
				t.Fatalf("selector %d, decoded %v; want %+v", k.Index(), decoded, r)
			}
			if bit := ui.HeroOf(r.mage, r.female); bit != r.bit || ui.HeroCells(true, true)[r.cell] != r.bit {
				t.Fatalf("bit %d, cell %d; want bit %d", bit, r.cell, r.bit)
			}
			audience := EventAudience{HeroFemale: r.female, HeroMage: r.mage, tags: secondEventTags{}}
			name, ok := townSpeechTagPath(original.speech, "npc5talk7", "npc=21, part=1", 1, audience)
			if want := "speech/talk/0021" + r.letter + "007p01.wav"; !ok || name != want {
				t.Fatalf("speech %q, %v; want %q", name, ok, want)
			}
			profile := original
			letters := []byte("fams")
			letters[r.index], letters[3] = letters[3], letters[r.index]
			profile.speech.heroLetters = string(letters)
			townTalkRules[town.TalkSection] = profile
			s, _ := secondSpeechTown(t, secondSpeechHero(false, false))
			s.in.Archives.Base.Profile.Language = "english"
			s.sess.Carried = nil
			s.openTownText("npc5talk7")
			s.dialogue.tag, s.dialogue.displayPart = "npc=21, part=1", 1
			if name, ok := s.townSpeechPath(); !ok || name != "speech/talk/0021s007p01.wav" {
				t.Fatalf("fallback at index %d = %q, %v", r.index, name, ok)
			}
			townTalkRules[town.TalkSection] = original
			mw := &mapWorld{figures: map[sim.EntityID]figureID{1: {}}}
			if bank := mw.voiceBank(sim.Entity{ID: 1, TypeID: int32(0x21 + r.index)}); bank != r.bank {
				t.Fatalf("voice bank %q, want %q", bank, r.bank)
			}
		})
	}
	masks := []struct {
		mask       ui.HeroAudience
		sex, class bool
		word, lits string
	}{
		{0, false, false, "", ""},
		{1, true, true, "womanmage", "iamfemale,iammage"},
		{2, true, true, "womanfighter", "iamfemale,iamfighter"},
		{3, true, false, "woman", "iamfemale"},
		{4, true, true, "manmage", "iammale,iammage"},
		{5, false, true, "mage", "iammage"},
		{6, false, false, "", ""},
		{7, false, false, "", ""},
		{8, true, true, "manfighter", "iammale,iamfighter"},
		{9, false, false, "", ""},
		{10, false, true, "fighter", "iamfighter"},
		{11, false, false, "", ""},
		{12, true, false, "man", "iammale"},
		{13, false, false, "", ""},
		{14, false, false, "", ""},
		{15, false, false, "", ""},
	}
	for _, m := range masks {
		t.Run(fmt.Sprintf("mask_%02d", m.mask), func(t *testing.T) {
			sex, class := ui.HeroSplit([]ui.HeroAudience{m.mask})
			if sex != m.sex || class != m.class {
				t.Fatalf("split = %v/%v, want %v/%v", sex, class, m.sex, m.class)
			}
			wantWord := m.word
			if wantWord != "" {
				wantWord = "editor.message.hero." + wantWord
			}
			if word := ui.HeroWord(m.mask); word != wantWord {
				t.Fatalf("word %q, want %q", word, wantWord)
			}
			for _, axes := range [][2]bool{{false, false}, {true, false}, {false, true}, {true, true}} {
				var want []string
				for _, lit := range strings.FieldsFunc(m.lits, func(r rune) bool { return r == ',' }) {
					if (lit == "iamfemale" || lit == "iammale") && axes[0] || (lit == "iammage" || lit == "iamfighter") && axes[1] {
						want = append(want, lit)
					}
				}
				if got := heroLiterals(m.mask, axes[0], axes[1]); !slices.Equal(got, want) {
					t.Fatalf("literals %v for axes %v, want %v", got, axes, want)
				}
			}
			for _, r := range rows {
				if got, want := m.mask&r.bit != 0, slices.Contains(map[ui.HeroAudience][]int{
					0: {}, 1: {3}, 2: {1}, 3: {1, 3}, 4: {2}, 5: {2, 3}, 6: {1, 2}, 7: {1, 2, 3},
					8: {0}, 9: {0, 3}, 10: {0, 1}, 11: {0, 1, 3}, 12: {0, 2}, 13: {0, 2, 3}, 14: {0, 1, 2}, 15: {0, 1, 2, 3},
				}[m.mask], r.index); got != want {
					t.Fatalf("kind %v membership %v, want %v", r.kind, got, want)
				}
				if got, want := m.mask.Kinds().Holds(r.kind), m.mask&r.bit != 0; got != want {
					t.Fatalf("decoded kind %v membership %v, want %v", r.kind, got, want)
				}
			}
		})
	}
	for _, c := range []struct {
		sex, class bool
		cells      []ui.HeroAudience
	}{{false, false, []ui.HeroAudience{15}}, {true, false, []ui.HeroAudience{3, 12}},
		{false, true, []ui.HeroAudience{5, 10}}, {true, true, []ui.HeroAudience{1, 2, 4, 8}}} {
		if got := ui.HeroCells(c.sex, c.class); !slices.Equal(got, c.cells) {
			t.Fatalf("cells for %v/%v = %v, want %v", c.sex, c.class, got, c.cells)
		}
	}
}

func TestSpeechHeroWireRangeKeepsItsVoiceAndDeadArt(t *testing.T) {
	voices := []string{"mf_hero", "ff_hero", "m_mage", "f_mage"}
	mw := &mapWorld{figures: map[sim.EntityID]figureID{1: {}}}
	fighter, mage := &terrain.UnitClass{}, &terrain.UnitClass{}
	units := &terrain.UnitSet{Classes: map[int32]*terrain.UnitClass{1: fighter, 24: mage}}
	for typ := int32(0x20); typ < 0x40; typ++ {
		index := (int(typ) + 3) % 4
		if bank := mw.voiceBank(sim.Entity{ID: 1, TypeID: typ}); bank != voices[index] {
			t.Fatalf("wire type %x voice %q, want %q", typ, bank, voices[index])
		}
		ms := &Mission{World: &sim.World{}, DeadArt: map[sim.EntityID]uint16{1: uint16(typ)}}
		want := fighter
		if index >= 2 {
			want = mage
		}
		if got := missionAppearanceArt(ms, units, data.NewBodyList(data.BodyUnarmed), nil, nil)[1]; got != want {
			t.Fatalf("wire type %x dead art %p, want %p", typ, got, want)
		}
	}
}

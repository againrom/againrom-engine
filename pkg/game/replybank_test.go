package game

import (
	"slices"
	"testing"
	"time"

	"againrom/pkg/audio"
	"againrom/pkg/data"
	"againrom/pkg/render/terrain"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

// replyBankPeople are own units of every kind the replies reach, each with the
// bank the selector gives it (ANIM-096, HERO-APPEAR-055): the mage and hero
// pairs by type, then the mercenary pair when visible slot 0 is filled and the
// peasant pair when it is empty, with the sex of the drawn figure. A creature
// has no figure and no bank.
var replyBankPeople = []struct {
	name    string
	typeID  int32
	dir     data.FigureDir
	weapon  bool
	guarded bool
	bank    string
}{
	{"an unarmed man", 1, data.FigureDirManFighter, false, false, "m_peasant"},
	{"an unarmed woman", 1, data.FigureDirWomanFighter, false, false, "f_peasant"},
	{"an armed man", 14, data.FigureDirManFighter, true, false, "mf_merc"},
	{"an armed woman", 14, data.FigureDirWomanFighter, true, false, "ff_merc"},
	{"a hero the party guards", sim.HeroTypeID(false, false), data.FigureDirManFighter, true, true, "mf_hero"},
	{"a hero placed beside the party", sim.HeroTypeID(false, true), data.FigureDirWomanFighter, false, false, "ff_hero"},
	{"a male mage", 0x17, data.FigureDirManMage, false, false, "m_mage"},
	{"a female mage", 0x18, data.FigureDirWomanMage, true, false, "f_mage"},
	{"an own creature", 0x40, "", false, false, ""},
}

// replyBankFixture is a bank in which every select recording of the eight
// human banks carries its own number.
func replyBankFixture() *SoundBank {
	named := map[string]soundCacheEntry{}
	for i, bank := range hurtVoiceBanks {
		for n := 1; n <= 2; n++ {
			named[bank+"/select"+string(rune('0'+n))+".wav"] = soundCacheEntry{sample: audio.Sample{PCM: []int16{int16(100*(i+1) + n)}}, ok: true}
		}
	}
	return &SoundBank{named: named}
}

// TestARepliesBankIsTheBankItsWoundsUse asks each kind of own unit for its
// selection reply through the viewer's reply, and requires the recording to
// be select1 of the bank the same unit's wounds take, and none for a creature.
// One selector serves every voice request of a drawable (ANIM-094), so a
// person no weapon shows replies from the peasant bank and never from the
// mercenary bank.
func TestARepliesBankIsTheBankItsWoundsUse(t *testing.T) {
	var ents []sim.Entity
	var stocks []sim.Stock
	figures := map[sim.EntityID]figureID{}
	var guarded []sim.EntityID
	for i, p := range replyBankPeople {
		id := sim.EntityID(i)
		ents = append(ents, sim.Entity{ID: id, X: int32(i % swingW), Y: int32(i / swingW), Owner: sim.SelfSlot,
			TypeID: p.typeID, Class: 1, Humanoid: p.dir != "", HP: 10, MaxHP: 10})
		if p.weapon {
			stocks = append(stocks, sim.Stock{ID: id, Equipped: [sim.EquipSlots]uint16{0: eqSwordCode}})
		}
		if p.dir != "" {
			figures[id] = figureID{Dir: p.dir, Hero: data.FigureIsHero(p.typeID)}
		}
		if p.guarded {
			guarded = append(guarded, id)
		}
	}
	w, err := sim.NewStockedWorld(1, sim.Bounds{Width: swingW, Height: swingH}, sim.ModeCanonical, sim.Terrain{},
		ents, nil, sim.Relations{}, nil, stocks)
	if err != nil {
		t.Fatal(err)
	}
	v, err := ui.NewViewer("reply banks", terrain.Grid{
		Width: swingW, Height: swingH, Tiles: make([]uint16, swingW*swingH),
	}, &terrain.Tileset{})
	if err != nil {
		t.Fatal(err)
	}
	mw := newMapWorld(w, nil, &terrain.UnitSet{}, v)
	mw.figures = figures
	mw.mission = &missionNotices{guarded: guarded}
	draws := map[uint32]ui.MapEntity{}
	for _, d := range mw.entityDraws() {
		draws[d.ID] = d
	}
	voices := &acknowledgmentRecorder{}
	f := &FrontEnd{InstallResources: InstallResources{SoundBank: replyBankFixture()}, RuntimeServices: RuntimeServices{SpeechPlayer: voices}}
	_, selection := f.runtimeAudio().unitReplies(mw, func() int { return 0 })
	now := time.Unix(10, 0)
	for i, p := range replyBankPeople {
		if got := draws[uint32(i)].Voice; got != p.bank {
			t.Errorf("%s: wounds sound from %q, want %q", p.name, got, p.bank)
		}
		mark := len(voices.samples)
		selection([]uint32{uint32(i)}, now)
		now = now.Add(time.Minute)
		var want []int16
		if p.bank != "" {
			want = []int16{int16(100*(slices.Index(hurtVoiceBanks, p.bank)+1) + 1)}
		}
		if got := heard(voices, mark); !slices.Equal(got, want) {
			t.Errorf("%s: the selection reply requested %v, want %v (%s/select1.wav)", p.name, got, want, p.bank)
		}
	}
}

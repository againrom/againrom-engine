package game

import (
	"reflect"
	"testing"

	"againrom/pkg/audio"
	"againrom/pkg/base"
	"againrom/pkg/sim"
)

// namedSoundRecord answers two named interface members and lists the ones the
// viewer requests, in order.
type namedSoundRecord struct{ played []string }

func (p *namedSoundRecord) Play(audio.Sample, audio.Placement) {}

func (p *namedSoundRecord) RequestSample(_ audio.Sample, request audio.Request) audio.Voice {
	p.played = append(p.played, request.Selector)
	return nil
}

type namedSoundBank struct{}

func (namedSoundBank) Sample(int) (audio.Sample, bool) { return audio.Sample{}, false }

func (namedSoundBank) NamedSample(path string) (audio.Sample, bool) {
	switch path {
	case "put_on.wav", "put_off.wav":
		return audio.Sample{Rate: audio.DeviceRate, PCM: []int16{1}}, true
	}
	return audio.Sample{}, false
}

// dollSoundMission opens a mission whose one member carries a sword and a
// mace in the pack, on the edition's table, with the sound recorder wired.
func dollSoundMission(t *testing.T, game base.Game) (*mapWorld, *namedSoundRecord) {
	t.Helper()
	table := eqDefsTable(t)
	edition := game.Edition()
	table.Edition = &edition
	w, err := sim.NewStockedWorld(1, sim.Bounds{Width: worldFixtureW, Height: worldFixtureH}, sim.ModeCanonical, sim.Terrain{},
		[]sim.Entity{{ID: 7, X: 3, Y: 3, HP: 10, MaxHP: 10}}, nil, sim.Relations{}, nil,
		[]sim.Stock{{ID: 7, Items: []uint16{eqNoSlotCode, eqSwordCode, eqMaceCode}}})
	if err != nil {
		t.Fatalf("NewStockedWorld: %v", err)
	}
	mw := equipMission(t, w, 7, eqHero(), nil, table)
	rec := &namedSoundRecord{}
	mw.view.SetAudio(rec, namedSoundBank{})
	return mw, rec
}

// In the second game's mission inventory every gesture that puts an item on
// the figure plays Put_On once and every gesture that takes one off plays
// Put_Off once, when the change lands; a swap plays Put_On, and a refused
// gesture plays nothing. The first game's mission inventory plays nothing.
func TestMissionInventoryDollPlaysPutOnAndPutOff(t *testing.T) {
	steps := []struct {
		name string
		do   func(mw *mapWorld)
		want []string
	}{
		{"wear from the pack", func(mw *mapWorld) { mw.enqueueEquip(1) }, []string{"put_on.wav"}},
		{"swap one worn weapon for another", func(mw *mapWorld) { mw.enqueueEquip(1) }, []string{"put_on.wav"}},
		{"take off by the worn box", func(mw *mapWorld) { mw.enqueueUnequip(0) }, []string{"put_off.wav"}},
		{"take off an empty slot", func(mw *mapWorld) { mw.enqueueUnequip(5) }, nil},
		{"wear an item with no slot", func(mw *mapWorld) { mw.enqueueEquip(0) }, nil},
		{"wear again", func(mw *mapWorld) { mw.enqueueEquip(1) }, []string{"put_on.wav"}},
		{"drop the worn item to the ground", func(mw *mapWorld) { mw.enqueueDrop(true, 0, 3, 3) }, []string{"put_off.wav"}},
		{"a pack drop plays nothing", func(mw *mapWorld) { mw.enqueueDrop(false, 0, 3, 3) }, nil},
	}
	for _, game := range []struct {
		name  string
		game  base.Game
		sound bool
	}{{"second game", base.GameROM2, true}, {"first game", base.GameROM1, false}} {
		t.Run(game.name, func(t *testing.T) {
			mw, rec := dollSoundMission(t, game.game)
			for _, st := range steps {
				rec.played = nil
				st.do(mw)
				mw.tick()
				mw.refreshEquipment()
				want := st.want
				if !game.sound {
					want = nil
				}
				if !reflect.DeepEqual(rec.played, want) {
					t.Errorf("%s: played %v, want %v", st.name, rec.played, want)
				}
			}
		})
	}
}

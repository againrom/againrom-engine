package game

import (
	"reflect"
	"testing"

	"againrom/pkg/audio"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

type fixedUISoundPlayer struct{ slots []int }

func (p *fixedUISoundPlayer) Play(s audio.Sample, _ audio.Placement) {
	p.slots = append(p.slots, int(s.PCM[0]))
}

func (p *fixedUISoundPlayer) RequestSample(s audio.Sample, request audio.Request) audio.Voice {
	p.Play(s, request.Placement)
	return nil
}

type fixedUISoundBank map[int]audio.Sample

func (b fixedUISoundBank) Sample(slot int) (audio.Sample, bool) {
	s, ok := b[slot]
	return s, ok
}

func TestOutcomeChildCreationPlaysCompleteAndFailedSelectorsOnce(t *testing.T) {
	for _, tc := range []struct {
		name    string
		outcome sim.Outcome
		slot    int
		kind    ui.NoticeKind
	}{
		{"completed", sim.OutcomeWon, int(ui.UISoundMissionComplete), ui.NoticeSuccess},
		{"failed", sim.OutcomeLost, int(ui.UISoundMissionFailed), ui.NoticeFailure},
	} {
		t.Run(tc.name, func(t *testing.T) {
			p := &fixedUISoundPlayer{}
			bank := fixedUISoundBank{
				tc.slot: {Rate: audio.DeviceRate, PCM: []int16{int16(tc.slot)}},
			}
			v := &ui.Viewer{}
			v.SetAudio(p, bank)
			mw := &mapWorld{view: v, mission: &missionNotices{outcome: tc.outcome}}

			mw.showOutcome()

			if got, want := p.slots, []int{tc.slot}; !reflect.DeepEqual(got, want) {
				t.Fatalf("played slots = %v, want %v", got, want)
			}
			if !mw.mission.outcomeShown || !mw.mission.open || mw.mission.kind != tc.kind {
				t.Fatalf("outcome child state = open:%v shown:%v kind:%v", mw.mission.open, mw.mission.outcomeShown, mw.mission.kind)
			}
		})
	}
}

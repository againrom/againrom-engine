package sim

import (
	"bytes"
	"fmt"
	"testing"
)

func TestOriginalOutcome1097IndependentCountersNativeAndReporter(t *testing.T) {
	for _, outcome := range []Outcome{OutcomeUndecided, OutcomeWon, OutcomeLost} {
		for _, won := range []uint32{0, 1, 2, ^uint32(0)} {
			for _, lost := range []uint32{0, 1, 2, ^uint32(0)} {
				t.Run(fmt.Sprintf("outcome%d_win%d_lose%d", outcome, won, lost), func(t *testing.T) {
					w := originalSessionWorld(t)
					state := originalSessionFixture()
					state.TriggerLatches[7] = 1 // do not replay the unconditional WIN
					state.Outcome, state.Won, state.Lost = outcome, won, lost
					if err := w.ImportOriginalSession(state); err != nil {
						t.Fatal(err)
					}
					before, err := w.MarshalBinary()
					if err != nil {
						t.Fatal(err)
					}
					var back World
					if err := back.UnmarshalBinary(before); err != nil {
						t.Fatal(err)
					}
					after, _ := back.MarshalBinary()
					if !bytes.Equal(before, after) || back.Hash() != w.Hash() || back.Outcome() != outcome {
						t.Fatal("native round trip changed independent outcome/counters")
					}
					want := outcome
					if want != OutcomeLost {
						if lost == 1 {
							want = OutcomeLost
						} else if won == 1 {
							want = OutcomeWon
						}
					}
					for i := 0; i < 32; i++ {
						Step(&back, nil)
					}
					if actualWon, actualLost := back.ScriptCounters(); actualWon != won || actualLost != lost || back.Outcome() != want {
						t.Fatalf("reporter = %v/%d/%d, want %v/%d/%d", back.Outcome(), actualWon, actualLost, want, won, lost)
					}
				})
			}
		}
	}
}

func TestOriginalOutcome1097InvalidOutcomeDoesNotMutateWorld(t *testing.T) {
	w := originalSessionWorld(t)
	before, _ := w.MarshalBinary()
	for _, value := range []Outcome{3, 255} {
		state := originalSessionFixture()
		state.Outcome, state.Won, state.Lost = value, 99, 77
		state.TriggerLatches[7] = 1
		if err := w.ImportOriginalSession(state); err == nil {
			t.Fatalf("accepted outcome %d", value)
		}
		after, _ := w.MarshalBinary()
		if !bytes.Equal(before, after) {
			t.Fatal("invalid outcome partially mutated world")
		}
	}
}

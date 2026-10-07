package game

import (
	"bytes"
	"path/filepath"
	"testing"
	"time"
)

func TestReleaseCommandAcknowledgments1188InstalledGestureAndCheckbox(t *testing.T) {
	f := releaseFront(t)
	for _, deterministic := range []bool{true, false} {
		t.Run(map[bool]string{true: "deterministic-parent", false: "physical-parent"}[deterministic], func(t *testing.T) {
			probe := releaseFront(t)
			probe.SetDeterministicFrames(deterministic)
			probe.Options = OptionsStore{Path: filepath.Join(t.TempDir(), "options.txt")}
			probe.SoundBank = OpenSounds(f.Archives.Root)
			var report bytes.Buffer
			if err := probe.witnessCommandAcknowledgmentsWithClockControl(&report, func(opened *FrontEnd, enabled bool) {
				elapsed := -time.Second
				if !enabled {
					elapsed = time.Hour
				}
				opened.live.last = time.Now().Add(elapsed)
			}); err != nil {
				t.Fatal(err, report.String())
			}
			if enabled, err := probe.Options.Acknowledgments(); err != nil || enabled {
				t.Fatal("checkbox did not persist", enabled, err)
			}
			t.Log(report.String())
		})
	}
}

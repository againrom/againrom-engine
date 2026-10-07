package game

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"againrom/pkg/audio"
)

func TestAcknowledgments1188PreferencesColdStartAndFailedWrite(t *testing.T) {
	path := filepath.Join(t.TempDir(), "options.txt")
	f := &FrontEnd{PersistenceContext: PersistenceContext{Options: OptionsStore{Path: path}}}
	f.LoadOptions()
	if f.acknowledgmentsOff {
		t.Fatal("missing preference disabled replies")
	}
	for value, enabled := range map[string]bool{"0": false, "1": true, "7": true, "-1": true, "broken": true} {
		if err := os.WriteFile(path, []byte("Other=kept\nAcknowledgement="+value+"\n"), 0600); err != nil {
			t.Fatal(err)
		}
		f.LoadOptions()
		if f.acknowledgmentsOff == enabled {
			t.Fatal("wrong preference", value)
		}
	}
	if err := f.setAcknowledgments(false); err != nil {
		t.Fatal(err)
	}
	cold := &FrontEnd{PersistenceContext: PersistenceContext{Options: OptionsStore{Path: path}}}
	cold.LoadOptions()
	if !cold.acknowledgmentsOff {
		t.Fatal("cold process lost disabled preference")
	}
	contents, _ := os.ReadFile(path)
	if !strings.Contains(string(contents), "Other=kept\n") {
		t.Fatal("unrelated preference lost")
	}
	cold.Options.Path = t.TempDir()
	if err := cold.setAcknowledgments(true); err == nil || !cold.acknowledgmentsOff {
		t.Fatal("failed write changed live option")
	}
}

type acknowledgmentRecorder struct{ samples []audio.Sample }

func (r *acknowledgmentRecorder) Play(s audio.Sample, _ audio.Placement) {
	r.samples = append(r.samples, s)
}

func (r *acknowledgmentRecorder) RequestSample(s audio.Sample, request audio.Request) audio.Voice {
	r.Play(s, request.Placement)
	return nil
}

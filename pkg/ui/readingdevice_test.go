package ui

import (
	"io"
	"testing"
	"time"

	"againrom/pkg/audio"
)

func TestReadingDeviceOnlyInSilentTestProcess(t *testing.T) {
	for _, c := range []struct {
		test  bool
		level float64
		want  bool
	}{
		{false, 0, false}, {false, 1, false}, {true, 0.05, false}, {true, 1, false}, {true, 0, true},
	} {
		if got := readingDeviceSelected(c.test, c.level); got != c.want {
			t.Errorf("readingDeviceSelected(%v, %v) = %v, want %v", c.test, c.level, got, c.want)
		}
	}
	out, err := openAudioContext()
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := out.(*readingDevice); !ok {
		t.Fatalf("this silent test process opened %T, want the reading device", out)
	}
}

// waitRead polls cond for at most bound, as the release witnesses do.
func waitRead(bound time.Duration, cond func() bool) bool {
	deadline := time.Now().Add(bound)
	for !cond() && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	return cond()
}

func TestReadingDeviceReadsAtTheDeviceRate(t *testing.T) {
	d := newReadingDevice(audio.DeviceRate)
	second := make([]byte, 4*audio.DeviceRate)
	s := d.NewPlayerFromBytes(append(second, second...)).(*readingStream)
	if s.IsPlaying() || s.Position() != 0 {
		t.Fatal("an unplayed stream reports playing or a position")
	}
	s.Play()
	if !s.IsPlaying() || s.read == 0 {
		t.Fatalf("Play did not read the source: playing=%v read=%d", s.IsPlaying(), s.read)
	}
	if !waitRead(time.Second, func() bool { return s.Position() > 0 }) {
		t.Fatal("position did not advance while playing")
	}
	start := time.Now()
	time.Sleep(200 * time.Millisecond)
	s.Pause()
	elapsed, at := time.Since(start), s.Position()
	if at <= 0 || at > elapsed+300*time.Millisecond {
		t.Fatalf("position %v after %v of play", at, elapsed)
	}
	time.Sleep(50 * time.Millisecond)
	if s.IsPlaying() || s.Position() != at {
		t.Fatal("a paused stream kept playing")
	}
	if err := s.Seek(1900 * time.Millisecond); err != nil || s.Position() != 1900*time.Millisecond {
		t.Fatalf("seek: %v, position %v", err, s.Position())
	}
	s.Play()
	if !waitRead(2*time.Second, func() bool { return !s.IsPlaying() }) {
		t.Fatal("a stream at its end kept playing")
	}
	if s.Position() != 2*time.Second {
		t.Fatalf("ended at %v, want 2s", s.Position())
	}
	if _, err := d.NewPlayer(io.MultiReader()); err != nil {
		t.Fatal(err)
	}
}

// A player the device does not read fails each device-state check the
// release witnesses make: the voice is not playing, its phase stays zero and
// AudioDeviceState reports no active gain. The same checks pass on a read one.
func TestUnreadPlayerFailsTheDeviceStateChecks(t *testing.T) {
	for _, unread := range []bool{false, true} {
		d := newReadingDevice(audio.DeviceRate)
		if unread {
			d.unread = func(io.Reader) bool { return true }
		}
		restore := SetDeliveryPlayerFactory(func(reader io.ReadSeeker) (DeliveryDevicePlayer, error) {
			p, err := d.NewPlayer(reader)
			if err != nil {
				return nil, err
			}
			return newDevicePlayer(p), nil
		})
		owner, err := OpenSharedAudio(audio.DefaultSettings, audio.DefaultSettings)
		restore()
		if err != nil {
			t.Fatal(err)
		}
		sample := audio.Sample{Rate: audio.DeviceRate, PCM: make([]int16, audio.DeviceRate)}
		for i := range sample.PCM {
			sample.PCM[i] = int16(i%200 - 100)
		}
		request := audio.FixedRequest("speech", "speech", audio.SpeechChannel, 128, false,
			audio.Placement{Left: audio.GainUnit, Right: audio.GainUnit})
		scope := owner.NewScope()
		voice := audio.Dispatch(scope.Player(audio.SpeechChannel), sample, request).(*deliveryVoice)
		playing := voice.Playing()
		phased := waitRead(750*time.Millisecond, func() bool { return voice.Phase() != 0 })

		music := &musicDevice{ctx: d, st: audio.DefaultSettings}
		music.Start(audio.Track{Rate: audio.DeviceRate, StereoPCM: make([]byte, 4*audio.DeviceRate)})
		_, gains, ok := AudioDeviceState(music)
		active := ok && len(gains) == 1

		if playing == unread || phased == unread || active == unread {
			t.Errorf("unread=%v: voice playing=%v phase=%v music active=%v", unread, playing, phased, active)
		}
		music.Stop()
		scope.Destroy()
		owner.Service.Close()
	}
}

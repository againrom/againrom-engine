package video

import (
	"io"
	"sync/atomic"
	"testing"
	"time"

	"againrom/pkg/video/smacker"
)

type fakeClock struct {
	pos atomic.Int64
	ok  atomic.Bool
}

func (f *fakeClock) PlayedBytes() (int64, bool) { return f.pos.Load(), f.ok.Load() }

func stereoInfo() Info {
	return Info{Width: 4, Height: 4, Frames: 10, AudioRate: 22050, AudioChannels: 2, AudioBitDepth: 16}
}

func TestSoundTargetIsOneIntervalOfAudioPerFrame(t *testing.T) {
	sp := newSoundPacing(stereoInfo(), 6666)
	if got := sp.target(0); got != 0 {
		t.Fatalf("target(0) = %d, want 0", got)
	}
	// 22050 Hz, 4 bytes per sample frame, 66.66 ms.
	if got, want := sp.target(3), int64(3*88200*6666/100000); got != want {
		t.Fatalf("target(3) = %d, want %d", got, want)
	}
	if newSoundPacing(Info{Width: 4, Height: 4, Frames: 1}, 6666) != nil {
		t.Fatal("a movie with no track must not get sound pacing")
	}
}

func testClock(stop <-chan struct{}) pacerClock {
	return pacerClock{start: time.Now(), stop: stop}
}

func TestSoundWaitHoldsUntilThePositionReachesTheTarget(t *testing.T) {
	info := stereoInfo()
	sp := newSoundPacing(info, 6666)
	snd := &soundSource{}
	fc := &fakeClock{}
	fc.ok.Store(true)
	snd.set(fc)
	stop := make(chan struct{})
	defer close(stop)
	// The timer interval is far longer than the test: only the position can
	// release the frame.
	p := newTimerPacer(1 << 30)
	done := make(chan bool, 1)
	go func() { done <- waitFrameSynced(p, testClock(stop), snd, sp, 2) }()
	target := sp.target(2)
	select {
	case <-done:
		t.Fatal("frame released at position 0")
	case <-time.After(30 * time.Millisecond):
	}
	fc.pos.Store(target - soundSlack - 1)
	select {
	case <-done:
		t.Fatal("frame released just short of the target")
	case <-time.After(30 * time.Millisecond):
	}
	fc.pos.Store(target - soundSlack)
	select {
	case ok := <-done:
		if !ok {
			t.Fatal("wait reported stop")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("frame not released at the target")
	}
}

func TestSoundWaitFirstFrameNeedsNoAudio(t *testing.T) {
	sp := newSoundPacing(stereoInfo(), 6666)
	snd := &soundSource{}
	fc := &fakeClock{}
	fc.ok.Store(true)
	snd.set(fc)
	stop := make(chan struct{})
	defer close(stop)
	if !waitFrameSynced(newTimerPacer(1<<30), testClock(stop), snd, sp, 0) {
		t.Fatal("stopped")
	}
}

func TestNoSessionOrNoClockUsesTheTimer(t *testing.T) {
	sp := newSoundPacing(stereoInfo(), 2000)
	stop := make(chan struct{})
	defer close(stop)
	for name, snd := range map[string]*soundSource{"no clock": {}, "no session": func() *soundSource {
		s := &soundSource{}
		s.set(&fakeClock{})
		return s
	}()} {
		start := time.Now()
		if !waitFrameSynced(newTimerPacer(2000), testClock(stop), snd, sp, 5) {
			t.Fatalf("%s: stopped", name)
		}
		if el := time.Since(start); el < 15*time.Millisecond || el > time.Second {
			t.Fatalf("%s: waited %v, want about 20 ms", name, el)
		}
	}
}

func TestSoundWaitGoesToTimerWhenTheSessionEnds(t *testing.T) {
	sp := newSoundPacing(stereoInfo(), 2000)
	snd := &soundSource{}
	fc := &fakeClock{}
	fc.ok.Store(true)
	snd.set(fc)
	stop := make(chan struct{})
	defer close(stop)
	done := make(chan bool, 1)
	go func() { done <- waitFrameSynced(newTimerPacer(2000), testClock(stop), snd, sp, 5) }()
	time.Sleep(10 * time.Millisecond)
	fc.ok.Store(false)
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("timer did not take over after the session ended")
	}
}

func TestStalledPositionEndsSoundPacing(t *testing.T) {
	old := soundStallLimit
	soundStallLimit = 40 * time.Millisecond
	defer func() { soundStallLimit = old }()
	sp := newSoundPacing(stereoInfo(), 2000)
	snd := &soundSource{}
	fc := &fakeClock{}
	fc.ok.Store(true)
	snd.set(fc)
	stop := make(chan struct{})
	defer close(stop)
	if !waitFrameSynced(newTimerPacer(2000), testClock(stop), snd, sp, 5) {
		t.Fatal("stopped")
	}
	if !sp.lost {
		t.Fatal("a position that never moved did not end sound pacing")
	}
}

func TestSoundWaitInterruptedByStop(t *testing.T) {
	sp := newSoundPacing(stereoInfo(), 6666)
	snd := &soundSource{}
	fc := &fakeClock{}
	fc.ok.Store(true)
	snd.set(fc)
	stop := make(chan struct{})
	done := make(chan bool, 1)
	go func() { done <- waitFrameSynced(newTimerPacer(1<<30), testClock(stop), snd, sp, 4) }()
	time.Sleep(10 * time.Millisecond)
	close(stop)
	select {
	case ok := <-done:
		if ok {
			t.Fatal("stop reported ready")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("stop did not interrupt the sound wait")
	}
}

type scriptedSource struct {
	n        int
	interval uint32
}

func (s *scriptedSource) IntervalUnits() uint32 { return s.interval }
func (s *scriptedSource) Next() (*smacker.Frame, error) {
	if s.n >= 3 {
		return nil, io.EOF
	}
	s.n++
	f := &smacker.Frame{Indices: make([]byte, 320*180)}
	return f, nil
}

// The stream ends at the final frame with no further wait (VIDEO-082).
func TestStreamEndsAtTheFinalFrameWithoutAHold(t *testing.T) {
	info := Info{Width: 640, Height: 360, Frames: 3}
	pr, pw := io.Pipe()
	stop := make(chan struct{})
	defer close(stop)
	// Interval 200 ms: a hold after the final frame would add 200 ms.
	go writeSmackerStream(pw, &scriptedSource{interval: 20000}, info, newPresenter(320, 180, 3, nil), &soundSource{}, stop)
	start := time.Now()
	if _, err := io.Copy(io.Discard, pr); err != nil {
		t.Fatal(err)
	}
	// Three frames, each waited one interval: 600 ms; a hold would make 800.
	if el := time.Since(start); el < 500*time.Millisecond || el > 750*time.Millisecond {
		t.Fatalf("stream took %v, want about 600 ms with no hold after the final frame", el)
	}
}

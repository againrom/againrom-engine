package ui

import (
	"bytes"
	"fmt"
	"io"
	"runtime"
	"sync/atomic"
	"testing"
	"time"

	"againrom/pkg/audio"
)

type refillCountingSource struct{ calls atomic.Int64 }

func (s *refillCountingSource) Read(p []byte) (int, error) {
	if len(p) == 0 {
		return 0, nil
	}
	s.calls.Add(1)
	p[0] = 1
	return len(p), nil
}

func TestCutsceneEmptyQueueDoesNotBlockOtherPlayerBufferRefills(t *testing.T) {
	const refills = 10_000
	halfSecond := 4 * audio.DeviceRate / 2
	d, cutscene := heldCutsceneDevice(audio.Settings{Master: audio.MasterUnit}, halfSecond)
	d.Start(audio.DeviceRate, 2)
	defer d.Stop()
	<-cutscene.playing
	d.mu.Lock()
	session := d.s
	d.mu.Unlock()
	if session == nil {
		t.Fatal("cutscene session not opened")
	}
	d.Push(make([]byte, halfSecond))
	select {
	case <-session.played:
	case <-time.After(3 * time.Second):
		t.Fatal("cutscene player did not preload its half-second buffer")
	}

	source := &refillCountingSource{}
	other := &heldPlayer{src: source, fill: halfSecond, playing: make(chan struct{})}
	other.Play()
	if !other.IsPlaying() || source.calls.Load() != 1 {
		t.Fatal("second player did not start with one half-second buffer")
	}

	result := make(chan error, 1)
	go func() {
		cutsceneBuffer, otherBuffer := make([]byte, halfSecond), make([]byte, halfSecond)
		for i := 0; i < refills; i++ {
			n, err := session.q.Read(cutsceneBuffer)
			if err != nil || n != halfSecond {
				result <- fmt.Errorf("cutscene refill %d: n=%d err=%v", i, n, err)
				return
			}
			n, err = other.src.Read(otherBuffer)
			if err != nil || n != halfSecond {
				result <- fmt.Errorf("second-player refill %d: n=%d err=%v", i, n, err)
				return
			}
		}
		result <- nil
	}()
	select {
	case err := <-result:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(10 * time.Second):
		stacks := make([]byte, 1<<20)
		n := runtime.Stack(stacks, true)
		d.Stop()
		<-result
		t.Fatalf("cutscene's empty queue held the other player's refill; goroutines:\n%s", stacks[:n])
	}
	if got := source.calls.Load(); got != refills+1 {
		t.Fatalf("second player received %d half-second buffers, want %d", got, refills+1)
	}
}

func TestCutsceneQueuePreservesPCMBeforeUnderrunSilence(t *testing.T) {
	q := newCutsceneAudioQueue(2)
	q.push([]byte{1, 2, 3, 4})
	buf := bytes.Repeat([]byte{0xff}, 8)
	n, err := q.Read(buf)
	if err != nil || n != len(buf) || !bytes.Equal(buf, []byte{1, 2, 3, 4, 0, 0, 0, 0}) {
		t.Fatalf("queued PCM then silence: n=%d err=%v buf=%v", n, err, buf)
	}
	q.closeQueue()
	if n, err = q.Read(buf); n != 0 || err != io.EOF {
		t.Fatalf("closed queue: n=%d err=%v, want EOF", n, err)
	}
}

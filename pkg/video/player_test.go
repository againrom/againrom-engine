package video

import (
	"bytes"
	"context"
	"image"
	"io"
	"testing"
	"time"
)

func eventPlayer(events ...event) *Player {
	_, cancel := context.WithCancel(context.Background())
	p := &Player{cancel: cancel, events: make(chan event, len(events))}
	for _, e := range events {
		p.events <- e
	}
	return p
}

func TestAdvanceNeverDropsQueuedFramesAndKeepsLastUntilEOF(t *testing.T) {
	a, b := image.NewRGBA(image.Rect(0, 0, 1, 1)), image.NewRGBA(image.Rect(0, 0, 1, 1))
	a.Pix, b.Pix = []byte{255, 0, 0, 255}, []byte{0, 255, 0, 255}
	p := eventPlayer(event{frame: a}, event{frame: b}, event{})
	now := time.Unix(1, 0)
	if !p.Advance(now) || p.Frame() != a {
		t.Fatal("first frame was dropped")
	}
	if !p.Advance(now.Add(time.Second)) || p.Frame() != b {
		t.Fatal("second frame was dropped")
	}
	if p.Advance(now.Add(2*time.Second)) || p.Err() != nil {
		t.Fatalf("EOF did not finish cleanly: %v", p.Err())
	}
	if p.Advance(now.Add(3 * time.Second)) {
		t.Fatal("completed player restarted")
	}
}

func TestStalledDecoderAndSkipInterruptBlockedReader(t *testing.T) {
	for _, skip := range []bool{false, true} {
		r, w := io.Pipe()
		p := NewPlayer(r)
		now := time.Unix(1, 0)
		if !p.Advance(now) {
			t.Fatal("startup is initially pending")
		}
		if skip {
			p.Close()
		} else if p.Advance(now.Add(StallLimit)) || p.Err() == nil {
			t.Fatal("stalled header did not time out")
		}
		select {
		case <-p.done:
		case <-time.After(time.Second):
			t.Fatal("reader survives cancellation")
		}
		_ = w.Close()
		p.Close()
	}
}

func TestMalformedFrameAndExtraDataFail(t *testing.T) {
	header := []byte{'A', 'R', 'V', '2', 1, 0, 0, 0, 1, 0, 0, 0, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}
	for _, body := range [][]byte{{1, 2}, {1, 2, 3, 255, 99}} {
		p := NewPlayer(io.NopCloser(bytes.NewReader(append(append([]byte(nil), header...), body...))))
		deadline := time.Now().Add(time.Second)
		for p.Advance(time.Now()) && time.Now().Before(deadline) {
			time.Sleep(time.Millisecond)
		}
		if !p.closed || p.Err() == nil {
			t.Fatal("malformed stream did not fail")
		}
		p.Close()
	}
}

package devicefake

import (
	"bytes"
	"io"
	"testing"
	"time"

	"againrom/pkg/audio"
)

func TestHandDrivenPlayer(t *testing.T) {
	data := make([]byte, 4*audio.DeviceRate)
	p := &Player{Reader: bytes.NewReader(data)}
	if p.IsPlaying() || p.Volume() != 0 {
		t.Fatal("fresh player not idle")
	}
	p.Play()
	p.SetVolume(0.5)
	if !p.IsPlaying() || p.Volume() != 0.5 {
		t.Fatal("play or volume lost")
	}
	if err := p.Seek(250 * time.Millisecond); err != nil || p.Position() != 250*time.Millisecond || p.Seeks != 1 {
		t.Fatalf("seek: %v %v %d", err, p.Position(), p.Seeks)
	}
	buf := make([]byte, 4)
	if _, err := p.Reader.Read(buf); err != nil {
		t.Fatal(err)
	}
	p.Pause()
	if p.IsPlaying() {
		t.Fatal("pause ignored")
	}
	p.Play()
	if err := p.Close(); err != nil || p.Playing || p.Closes != 1 {
		t.Fatal("close did not stop and count")
	}
	p.Playing = true
	p.Pump()
	if !p.Playing {
		t.Fatal("hand-driven player must not consume by itself")
	}
}

func TestClockedPlayerConsumesAtRate(t *testing.T) {
	now := time.Unix(100, 0)
	rate := 1000
	p := &Player{Reader: bytes.NewReader(make([]byte, 4*rate/2)), Clock: func() time.Time { return now }, Rate: rate}
	p.Play()
	now = now.Add(250 * time.Millisecond)
	if !p.IsPlaying() || p.Position() != 250*time.Millisecond {
		t.Fatalf("quarter second: playing=%v at %v", p.IsPlaying(), p.Position())
	}
	now = now.Add(time.Second)
	if p.IsPlaying() {
		t.Fatal("reader end did not finish the voice")
	}
	if err := p.Seek(100 * time.Millisecond); err != nil || p.Position() != 100*time.Millisecond {
		t.Fatalf("seek: %v %v", err, p.Position())
	}
	p.Play()
	now = now.Add(time.Millisecond)
	p.Close()
	p.Play()
	if p.IsPlaying() || p.Closes != 1 {
		t.Fatal("closed player played")
	}
	now = now.Add(-time.Hour)
	p.Pump()
}

func TestDefaultRateIsDeviceRate(t *testing.T) {
	now := time.Unix(0, 0)
	r := bytes.NewReader(make([]byte, 4*audio.DeviceRate))
	p := &Player{Reader: r, Clock: func() time.Time { return now }}
	p.Play()
	now = now.Add(time.Second / 2)
	p.Pump()
	if got := p.Position(); got != time.Second/2 {
		t.Fatalf("position %v", got)
	}
	rest, _ := io.ReadAll(r)
	if len(rest) != 2*audio.DeviceRate {
		t.Fatalf("left %d bytes", len(rest))
	}
}

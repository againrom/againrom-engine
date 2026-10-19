package ui

import (
	"bytes"
	"errors"
	"io"
	"sync"
	"time"
)

// readingDevice stands in for the Ebitengine context in a silent test
// process. Ebitengine opens its device only once a game loop runs, so a
// windowless test would otherwise never read a player. This device reads
// every playing stream at the device rate, as a real device does, and plays
// nothing. openAudioContext selects it only in a test process.
type readingDevice struct {
	bytesPerSecond int64
	mu             sync.Mutex
	playing        map[*readingStream]struct{}
	clock          sync.Once
	// unread names a source the device never reads. Only a loss control sets it.
	unread func(io.Reader) bool
}

const readingTick = 10 * time.Millisecond

func newReadingDevice(rate int) *readingDevice {
	return &readingDevice{bytesPerSecond: int64(rate) * 4, playing: make(map[*readingStream]struct{})}
}

func (d *readingDevice) NewPlayer(src io.Reader) (deviceStream, error) {
	s := &readingStream{device: d, src: src}
	s.bufferSize = s.defaultBufferSize()
	if d.unread != nil && d.unread(src) {
		s.unread = true
	}
	return s, nil
}

func (d *readingDevice) NewPlayerFromBytes(src []byte) deviceStream {
	s, _ := d.NewPlayer(bytes.NewReader(src))
	return s
}

func (d *readingDevice) start(s *readingStream) {
	d.mu.Lock()
	d.playing[s] = struct{}{}
	d.mu.Unlock()
	d.clock.Do(func() { go d.run() })
}

func (d *readingDevice) stop(s *readingStream) {
	d.mu.Lock()
	delete(d.playing, s)
	d.mu.Unlock()
}

func (d *readingDevice) run() {
	for {
		time.Sleep(readingTick)
		d.mu.Lock()
		streams := make([]*readingStream, 0, len(d.playing))
		for s := range d.playing {
			streams = append(streams, s)
		}
		d.mu.Unlock()
		now := time.Now()
		for _, s := range streams {
			s.mu.Lock()
			s.advanceLocked(now)
			s.mu.Unlock()
		}
	}
}

// readingStream buffers what it reads from its source and consumes the
// buffer at the device rate. Playing means the device has read samples.
type readingStream struct {
	device     *readingDevice
	mu         sync.Mutex
	src        io.Reader
	unread     bool
	playing    bool
	eof        bool
	read       int64
	buffered   int64
	offset     int64
	credit     int64
	last       time.Time
	bufferSize int64
	volume     float64
	scratch    []byte
}

func (s *readingStream) defaultBufferSize() int64 {
	return s.device.bytesPerSecond / 2 / 4 * 4
}

func (s *readingStream) fillLocked() {
	if s.unread {
		return
	}
	for !s.eof && s.buffered < s.bufferSize {
		want := min(s.bufferSize-s.buffered, int64(4096))
		if int64(len(s.scratch)) < want {
			s.scratch = make([]byte, want)
		}
		n, err := s.src.Read(s.scratch[:want])
		s.read += int64(n)
		s.buffered += int64(n)
		if err != nil {
			s.eof = true
		}
		if n == 0 {
			return
		}
	}
}

func (s *readingStream) advanceLocked(now time.Time) {
	if !s.playing {
		return
	}
	s.credit += int64(now.Sub(s.last)) * s.device.bytesPerSecond / int64(time.Second)
	s.last = now
	consume := min(s.credit/4*4, s.buffered)
	s.credit -= consume
	s.buffered -= consume
	s.offset += consume
	s.fillLocked()
	if s.eof && s.buffered == 0 {
		s.playing = false
		s.device.stop(s)
	}
}

func (s *readingStream) Play() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.playing {
		return
	}
	s.fillLocked()
	if s.eof && s.buffered == 0 {
		return
	}
	s.playing, s.last, s.credit = true, time.Now(), 0
	s.device.start(s)
}

func (s *readingStream) Pause() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.playing {
		s.advanceLocked(time.Now())
		s.playing = false
		s.device.stop(s)
	}
}

func (s *readingStream) PauseAndStopReading() { s.Pause() }

// IsPlaying is true only once the device has read samples from the source.
func (s *readingStream) IsPlaying() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.playing && s.read > 0
}

func (s *readingStream) Position() time.Duration {
	s.mu.Lock()
	defer s.mu.Unlock()
	return time.Duration(s.offset * int64(time.Second) / s.device.bytesPerSecond)
}

func (s *readingStream) Seek(offset time.Duration) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	seeker, ok := s.src.(io.Seeker)
	if !ok {
		return errors.New("audio: the source is not seekable")
	}
	o := int64(offset) * s.device.bytesPerSecond / int64(time.Second)
	o -= o % 4
	if _, err := seeker.Seek(o, io.SeekStart); err != nil {
		return err
	}
	s.offset, s.buffered, s.credit, s.eof = o, 0, 0, false
	s.last = time.Now()
	if s.playing {
		s.fillLocked()
	}
	return nil
}

func (s *readingStream) Volume() float64 {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.volume
}

func (s *readingStream) SetVolume(volume float64) {
	s.mu.Lock()
	s.volume = volume
	s.mu.Unlock()
}

func (s *readingStream) SetBufferSize(size time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()
	n := int64(size) * s.device.bytesPerSecond / int64(time.Second) / 4 * 4
	if size <= 0 || n <= 0 {
		n = s.defaultBufferSize()
	}
	s.bufferSize = n
}

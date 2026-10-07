package video

import (
	"fmt"
	"io"
	"sync"
	"time"

	"againrom/pkg/video/smacker"
)

// smackerAudioTrack is the sole track this project's playback path pulls:
// the first of the container's seven possible audio tracks, exactly the
// track the installed decoder's own selector named (native_windows_386.go's
// audioTrackFlag, the SDK's first-track bit) before this story's swap. No
// payload in either shipped video archive carries a second track this
// project would need to choose between (EN+RU corpus sweep).
const smackerAudioTrack = 0

// StartSmackerDecoder decodes compressed Smacker media entirely in this
// process, with pkg/video/smacker, and feeds the same ARV2 stream Player
// already reads from an external decoder (stream.go/player.go). It replaces
// StartDecoder as the production path: no helper subprocess, no installed
// smackw32.dll, and no runtime.GOOS restriction, since the whole decode is
// this project's own Go code. The movie plays without a sidecar.
func StartSmackerDecoder(media []byte) (*Player, error) {
	return StartSmackerDecoderWith(media, nil)
}

// StartSmackerDecoderWith is StartSmackerDecoder with the movie's sidecar
// (VIDEO-071). The stream's frames are the presented frames: windowed to the
// output region, doubled for a small movie, with the sidecar's fades and pans
// applied. A nil sidecar means no effects.
func StartSmackerDecoderWith(media []byte, sc *Sidecar) (*Player, error) {
	if len(media) == 0 || len(media) > MaxMediaBytes {
		return nil, fmt.Errorf("video: media size %d outside permitted range", len(media))
	}
	d, err := smacker.Open(media)
	if err != nil {
		return nil, err
	}
	outW, outH, _ := presentedSize(d.Width(), d.Height())
	info := Info{Width: uint32(outW), Height: uint32(outH), Frames: uint32(d.Frames())}
	if err := info.Validate(); err != nil {
		return nil, err
	}
	// Accept a track only at this project's one fixed playback rate, exactly
	// InspectNativeInput's own gate (native.go: "no resampling -- DIV-1255"),
	// and only in a coding this decoder implements. A track at any other rate,
	// bit depth, channel count or coding is left unattached rather than played
	// back at the wrong pitch or speed: video without audio, never a refusal
	// to play the movie at all.
	if channels, bitDepth, rate, exists := d.AudioTrack(smackerAudioTrack); exists && d.AudioDecodable(smackerAudioTrack) && rate == nativeAudioRate && bitDepth == 16 && (channels == 1 || channels == 2) {
		info.AudioRate, info.AudioChannels, info.AudioBitDepth = uint32(rate), uint8(channels), 16
		if err := info.Validate(); err != nil {
			return nil, err
		}
		d.EnableAudio(smackerAudioTrack, true)
	}
	d.EnableVideo(true)
	pr, pw := io.Pipe()
	stop := make(chan struct{})
	snd := &soundSource{}
	go writeSmackerStream(pw, d, info, newPresenter(d.Width(), d.Height(), d.Frames(), sc), snd, stop)
	pl := NewPlayer(&stopOnClose{PipeReader: pr, stop: stop})
	pl.sound = snd
	return pl, nil
}

// frameSource is the part of a decoder the stream writer uses.
type frameSource interface {
	IntervalUnits() uint32
	Next() (*smacker.Frame, error)
}

// stopOnClose lets a reader Close interrupt a pacing wait as well as the next
// pipe write.
type stopOnClose struct {
	*io.PipeReader
	stop chan struct{}
	once sync.Once
}

func (s *stopOnClose) Close() error {
	s.once.Do(func() { close(s.stop) })
	return s.PipeReader.Close()
}

// writeSmackerStream runs on its own goroutine for the life of one movie. A
// reader Close (StartSmackerDecoder's Player, on skip or teardown) closes stop
// and fails the next pipe write, which ends this loop promptly -- the same
// responsiveness process.go's subprocess Kill gave the now-retired helper
// path.
//
// Pacing follows the decoder wait (VIDEO-073, VIDEO-081): while the output
// device plays this movie's track, frame k waits until the played bytes reach
// k frame intervals of audio; with no track, no device session or a stalled
// position it waits on the timer, the first frame included, with no floor and
// no ceiling on the interval. The stream ends at the final frame with no wait
// after it (VIDEO-082), so the close cuts the audio still queued.
func writeSmackerStream(w *io.PipeWriter, d frameSource, info Info, pres *presenter, snd *soundSource, stop <-chan struct{}) {
	err := func() error {
		if err := WriteHeader(w, info); err != nil {
			return err
		}
		interval := d.IntervalUnits()
		pacer := newTimerPacer(interval)
		clock := pacerClock{start: time.Now(), stop: stop}
		sp := newSoundPacing(info, interval)
		for delivered := 0; delivered < int(info.Frames); delivered++ {
			if !waitFrameSynced(pacer, clock, snd, sp, delivered) {
				return io.ErrClosedPipe
			}
			f, err := d.Next()
			if err == io.EOF {
				return nil
			}
			if err != nil {
				return err
			}
			if err := WriteFrame(w, info, pres.present(f.Indices, f.Palette, f.NewPalette), f.Audio[smackerAudioTrack]); err != nil {
				return err
			}
		}
		return nil
	}()
	_ = w.CloseWithError(err)
}

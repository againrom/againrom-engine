package video

import (
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"time"
)

// nativeAudioRate is the sole sample rate decodeNative's audio pull accepts.
// It equals audio.DeviceRate; the constant is repeated here rather than
// imported so pkg/video keeps no dependency on pkg/audio.
const nativeAudioRate = 22050

// NativeInput is a preflight bound, not a validation of compressed Smacker
// data. The installed decoder still runs in a disposable child process.
//
// Info's own AudioRate/AudioChannels/AudioBitDepth (stream.go) are set here
// from the SMK2 container header, not from decodeNative's decode: the low 24
// bits of the header's AudioRate[0] word (offset 72) are a rate in Hz and
// bit 28 is set for two channels, clear for one; AudioBitDepth is fixed at
// 16, matched by every sample checked. Both are left zero, alongside
// AudioTrackBytes, when the header names no track or a rate other than
// nativeAudioRate.
type NativeInput struct {
	Info
	Interval time.Duration
	// AudioTrackBytes bounds one SmackGetTrackData call's destination for
	// this movie's first audio track: the same header's AudioSize[0] word
	// (offset 24). Zero means this movie carries no track decodeNative
	// pulls, and Info.AudioRate is left zero to match.
	AudioTrackBytes uint32
}

func InspectNativeInput(f *os.File) (NativeInput, error) {
	st, err := f.Stat()
	if err != nil {
		return NativeInput{}, err
	}
	if !st.Mode().IsRegular() || st.Size() < 104 || st.Size() > MaxMediaBytes {
		return NativeInput{}, fmt.Errorf("video: invalid compressed input size")
	}
	var h [104]byte
	if _, err := f.ReadAt(h[:], 0); err != nil {
		return NativeInput{}, err
	}
	if string(h[:4]) != "SMK2" {
		return NativeInput{}, fmt.Errorf("video: unsupported compressed signature")
	}
	v := NativeInput{Info: Info{Width: binary.LittleEndian.Uint32(h[4:]), Height: binary.LittleEndian.Uint32(h[8:]), Frames: binary.LittleEndian.Uint32(h[12:])}}
	if err := v.Validate(); err != nil {
		return NativeInput{}, err
	}
	// VIDEO-047 proves only block-aligned basic extents. Refuse other sizes.
	if v.Width%4 != 0 || v.Height%4 != 0 {
		return NativeInput{}, fmt.Errorf("video: unsupported non-block-aligned extent")
	}
	// AudioSize[0] (offset 24) and AudioRate[0] (offset 72) of the SMK2
	// header, read only to size a destination buffer and label the ARV2
	// wire header decodeNative emits -- see NativeInput's own comment for
	// where this layout comes from and how it was cross-checked. A track
	// this decode does not recognise (an unsupported rate, or a size beyond
	// MaxAudioChunk) leaves the movie's audio fields at zero: video without
	// audio, never a refusal to inspect the movie at all.
	if size := binary.LittleEndian.Uint32(h[24:28]); size > 0 && size <= MaxAudioChunk {
		rateWord := binary.LittleEndian.Uint32(h[72:76])
		if rate := rateWord & 0x00ffffff; rate == nativeAudioRate {
			channels := uint32(1)
			if rateWord&0x10000000 != 0 {
				channels = 2
			}
			v.AudioTrackBytes = size
			v.Info.AudioRate = rate
			v.Info.AudioChannels = uint8(channels)
			v.Info.AudioBitDepth = 16
			if err := v.Validate(); err != nil {
				return NativeInput{}, err
			}
		}
	}
	interval := int64(int32(binary.LittleEndian.Uint32(h[16:])))
	if interval < 0 {
		interval = -interval
	} else {
		interval *= 100
	}
	// VIDEO-050's clock units are 1/100 millisecond. This is an authored
	// silent presentation timer, not a call to the unbounded native wait arm.
	v.Interval = time.Duration(interval) * 10 * time.Microsecond
	if v.Interval < time.Millisecond || v.Interval > time.Second || v.Interval*time.Duration(v.Frames) > 20*time.Minute {
		return NativeInput{}, fmt.Errorf("video: timing outside bounded playback range")
	}
	return v, nil
}

// DecodeNative emits an ARV2 stream from the selected lawful DLL: video
// frames and, for a movie whose header names a track this decode recognises,
// that track's own PCM attached to each frame that carries any. Packed565
// selects the independent packed-word path used by the release pixel oracle;
// normal gameplay always uses palette indices. Paced holds each frame for its
// own interval, as playback needs; an oracle comparison reads frames unpaced.
func DecodeNative(dllPath, inputPath string, out io.Writer, packed565, paced bool) error {
	return decodeNative(dllPath, inputPath, out, packed565, paced)
}

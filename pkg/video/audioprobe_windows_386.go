package video

import (
	"encoding/binary"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"syscall"
	"unsafe"
)

// AudioProbeVariant names one candidate way to ask the installed decoder for
// track data with no sound device open. openFlags replaces the SmackOpen
// flags decodeNative always passes (0x1000, the file-handle selector VIDEO-045
// proves): every variant here keeps that bit and ORs in one candidate
// "track-request" bit this story's brief asks us to settle empirically,
// nothing is assumed from outside documentation. soundOnOff optionally calls
// SmackSoundOnOff(state, soundArg) right after a successful open, matching
// VIDEO-034's "enables decoder sound after successful setup".
type AudioProbeVariant struct {
	Name       string
	openFlags  uintptr
	soundOnOff bool
	soundArg   uintptr
	track      uintptr
	// filename opens by an ANSI path instead of the borrowed file-handle route
	// decodeNative always uses. The borrowed-handle route may be a video-only
	// fast path; the original game's own cutscene entry is not proved to use
	// it (VIDEO-045 only proves the handle route is CONSUMABLE, not that it is
	// what the original always picks), so this is an independent hypothesis,
	// not a documented alternative.
	filename bool
	// outParam tries the alternative shape where SmackSoundInTrack's second
	// argument is an OUT pointer to a byte count rather than a track index,
	// and SmackGetTrackData's middle argument is the destination buffer
	// rather than a track index -- plausible because every probed movie
	// carries exactly one audio track, so an explicit selector may not exist.
	outParam bool
	// driver selects a library-wide sound driver BEFORE SmackOpen: "win" calls
	// the zero-argument SmackSoundUseWin, "ds0" calls the one-argument
	// SmackSoundUseDirectSound(0). Both are genuinely arity-safe global calls
	// (unlike the crashed per-state SmackSoundOnOff-before-open attempt this
	// file's history records), on the hypothesis that the pull API might need
	// a driver armed before any movie is opened, not after.
	driver string
}

// AudioProbeResult is one variant's empirical outcome: nothing here is
// trusted before it is observed on the real DLL and the real media.
type AudioProbeResult struct {
	Variant       string
	Frames        uint32
	FramesWithPCM uint32
	TotalBytes    int64
	MinFrameBytes int
	MaxFrameBytes int
	// WidthOK/HeightOK sanity-check that SmackOpen actually parsed the header
	// for this variant, so a silent open failure cannot be mistaken for "no
	// audio track".
	DimensionsOK bool
	Err          error
}

// ProbeAudioTrack is the settling instrument this story's brief calls for.
// It sweeps candidate SmackOpen flag bits (with and without a following
// SmackSoundOnOff call) against one real movie, through the same
// export-binding and process-boundary rules decodeNative already enforces
// (absolute paths, bounded export lookup, no shell), and never writes PCM
// anywhere but the caller-supplied path. pcmOut receives ONLY the first
// variant that returns any bytes at all; callers pass "" to skip the file
// and just read counts. Every probed movie in the EN VIDEO4/VIDEO8 corpus
// carries exactly one audio track, and this is the flag it answers to; a
// movie with a second or third track, if one exists anywhere in the corpus,
// is Unknown and untested (no such movie was found).
const audioTrackFlag = uintptr(0x2000)

func ProbeAudioTrack(dllPath, inputPath, pcmOut string) ([]AudioProbeResult, error) {
	if !filepath.IsAbs(dllPath) || !filepath.IsAbs(inputPath) {
		return nil, fmt.Errorf("video: absolute paths required")
	}
	var variants []AudioProbeVariant
	variants = append(variants, AudioProbeVariant{Name: "baseline-no-change"})
	// One candidate bit at a time, low 24 bits (0x1000 already occupies bit
	// 12). That crash, not the AudioRate analogy, is this sweep's actual reason
	// to stop at bit 23.
	for bit := 0; bit < 24; bit++ {
		if bit == 12 {
			continue
		}
		v := uintptr(1) << uint(bit)
		variants = append(variants, AudioProbeVariant{Name: fmt.Sprintf("open-bit-%d", bit), openFlags: v})
	}
	variants = append(variants,
		AudioProbeVariant{Name: "soundonoff-arg1", soundOnOff: true, soundArg: 1},
		AudioProbeVariant{Name: "soundonoff-allbits", soundOnOff: true, soundArg: ^uintptr(0)},
		AudioProbeVariant{Name: "soundonoff-arg1-track1", soundOnOff: true, soundArg: 1, track: 1},
		// track-index-1, with no SmackSoundOnOff call at all, isolates
		// Finding 2 from Finding 1: this is the corrected argument order and
		// gate, asked for track INDEX 1 rather than the working FLAG
		// 0x2000, and is expected to return nothing on a single-track movie.
		AudioProbeVariant{Name: "track-index-1", track: 1},
		// The borrowed-handle route may simply never carry audio. An ANSI
		// filename open is the independent alternative VIDEO-045 leaves open;
		// every one of these keeps the same DoFrame/InTrack/GetTrackData loop.
		AudioProbeVariant{Name: "filename-open", filename: true},
		AudioProbeVariant{Name: "filename-open-soundonoff", filename: true, soundOnOff: true, soundArg: 1},
		AudioProbeVariant{Name: "filename-open-soundonoff-allbits", filename: true, soundOnOff: true, soundArg: ^uintptr(0)},
	)
	for _, bit := range []int{0, 1, 2, 3, 4, 5, 6, 7, 12} {
		variants = append(variants, AudioProbeVariant{Name: fmt.Sprintf("filename-open-bit-%d", bit), filename: true, openFlags: uintptr(1) << uint(bit)})
	}
	variants = append(variants,
		AudioProbeVariant{Name: "outparam-no-soundonoff", outParam: true},
		AudioProbeVariant{Name: "outparam-soundonoff-arg1", outParam: true, soundOnOff: true, soundArg: 1},
		AudioProbeVariant{Name: "outparam-filename-soundonoff-arg1", outParam: true, filename: true, soundOnOff: true, soundArg: 1},
	)
	variants = append(variants,
		AudioProbeVariant{Name: "driver-win-baseline", driver: "win"},
		AudioProbeVariant{Name: "driver-win-soundonoff", driver: "win", soundOnOff: true, soundArg: 1},
		AudioProbeVariant{Name: "driver-ds0-baseline", driver: "ds0"},
		AudioProbeVariant{Name: "driver-ds0-soundonoff", driver: "ds0", soundOnOff: true, soundArg: 1},
	)
	// Every variant above that leaves track at its zero value is asking for
	// track INDEX 0, which Finding 2 shows the DLL never answers -- only the
	// FLAG 0x2000 does. Defaulting the untouched majority to that flag here,
	// once, is what makes the sweep's own baseline truthful; a variant that
	// deliberately probes a different value (track-index-1,
	// soundonoff-arg1-track1, and the outParam alternates, which pass no
	// selector at all) is left exactly as it asked to be tested.
	for i := range variants {
		if variants[i].track == 0 && !variants[i].outParam {
			variants[i].track = audioTrackFlag
		}
	}
	// A now-removed variant called SmackSoundOnOff with a non-state first
	// argument BEFORE SmackOpen and crashed the probe process with an access
	// violation inside the DLL call (0xc0000005 at syscall.(*Proc).Call).
	// That crash is itself a finding, reported alongside these results:
	// SmackSoundOnOff reads its first argument as a live per-instance state
	// pointer, not a library-wide track mask, so no variant here calls it
	// before SmackOpen or with anything but a real state value.
	var results []AudioProbeResult
	wrote := false
	for _, v := range variants {
		r, pcm, err := probeOneVariant(dllPath, inputPath, v)
		r.Variant = v.Name
		if err != nil {
			r.Err = err
		}
		results = append(results, r)
		if !wrote && pcmOut != "" && r.TotalBytes > 0 {
			if werr := os.WriteFile(pcmOut, pcm, 0600); werr != nil {
				return results, werr
			}
			wrote = true
		}
	}
	return results, nil
}

func probeOneVariant(dllPath, inputPath string, v AudioProbeVariant) (AudioProbeResult, []byte, error) {
	res := AudioProbeResult{}
	f, err := os.Open(inputPath)
	if err != nil {
		return res, nil, err
	}
	defer f.Close()
	meta, err := InspectNativeInput(f)
	if err != nil {
		return res, nil, err
	}
	res.Frames = meta.Frames

	arities := map[string]int{"SmackOpen": 3, "SmackClose": 1, "SmackDoFrame": 1, "SmackNextFrame": 1,
		"SmackSoundOnOff": 2, "SmackSoundInTrack": 2, "SmackGetTrackData": 3,
		"SmackSoundUseWin": 0, "SmackSoundUseDirectSound": 1}
	bindings, err := exportBindings(dllPath, arities)
	if err != nil {
		return res, nil, err
	}
	path, err := syscall.UTF16PtrFromString(dllPath)
	if err != nil {
		return res, nil, err
	}
	loader := syscall.NewLazyDLL("kernel32.dll").NewProc("LoadLibraryExW")
	handle, _, loadErr := loader.Call(uintptr(unsafe.Pointer(path)), 0, 0x1100)
	if handle == 0 {
		return res, nil, fmt.Errorf("video: load decoder: %w", loadErr)
	}
	dll := &syscall.DLL{Name: dllPath, Handle: syscall.Handle(handle)}
	defer dll.Release()
	procs := make(map[string]*syscall.Proc)
	for base, name := range bindings {
		p, err := dll.FindProc(name)
		if err != nil {
			return res, nil, err
		}
		procs[base] = p
	}

	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	switch v.driver {
	case "win":
		procs["SmackSoundUseWin"].Call()
	case "ds0":
		procs["SmackSoundUseDirectSound"].Call(0)
	}

	var state uintptr
	if v.filename {
		name, err := syscall.BytePtrFromString(inputPath)
		if err != nil {
			return res, nil, err
		}
		state, _, _ = procs["SmackOpen"].Call(uintptr(unsafe.Pointer(name)), v.openFlags, ^uintptr(0))
	} else {
		state, _, _ = procs["SmackOpen"].Call(f.Fd(), 0x1000|v.openFlags, ^uintptr(0))
	}
	if state == 0 {
		return res, nil, fmt.Errorf("video: decoder refused input")
	}
	defer procs["SmackClose"].Call(state)

	// A borrowed state read, exactly like decodeNative's own readState: this
	// confirms SmackOpen actually parsed THIS input for this variant, so a
	// silently failed open cannot be mistaken for a real "no audio" result.
	var fields [16]byte
	readMemory := syscall.NewLazyDLL("kernel32.dll").NewProc("ReadProcessMemory")
	var read uintptr
	readMemory.Call(^uintptr(0), state, uintptr(unsafe.Pointer(&fields[0])), uintptr(len(fields)), uintptr(unsafe.Pointer(&read)))
	if read == uintptr(len(fields)) {
		gotW := binary.LittleEndian.Uint32(fields[4:])
		gotH := binary.LittleEndian.Uint32(fields[8:])
		gotN := binary.LittleEndian.Uint32(fields[12:])
		res.DimensionsOK = gotW == meta.Width && gotH == meta.Height && gotN == meta.Frames
	}

	if v.soundOnOff {
		procs["SmackSoundOnOff"].Call(state, v.soundArg)
	}

	// A generous per-frame ceiling: 22050 Hz * 2 bytes * 2 channels over one
	// second is 88200 bytes, and no probed interval exceeds one second
	// (InspectNativeInput already refuses that). 1<<20 stays far above any
	// single frame's audio chunk while remaining a bounded, authored size.
	chunk := make([]byte, 1<<20)
	var pin runtime.Pinner
	pin.Pin(&chunk[0])
	defer pin.Unpin()

	var count uint32
	var countPin runtime.Pinner
	countPin.Pin(&count)
	defer countPin.Unpin()

	var pcm []byte
	res.MinFrameBytes = -1
	for n := uint32(0); n < meta.Frames; n++ {
		procs["SmackDoFrame"].Call(state)
		var got int
		if v.outParam {
			count = 0
			procs["SmackSoundInTrack"].Call(state, uintptr(unsafe.Pointer(&count)))
			got = int(count)
			if got > 0 {
				if got > len(chunk) {
					got = len(chunk)
				}
				procs["SmackGetTrackData"].Call(state, uintptr(unsafe.Pointer(&chunk[0])), uintptr(unsafe.Pointer(&count)))
			}
		} else {
			has, _, _ := procs["SmackSoundInTrack"].Call(state, v.track)
			if has != 0 {
				wrote, _, _ := procs["SmackGetTrackData"].Call(state, uintptr(unsafe.Pointer(&chunk[0])), v.track)
				got = int(wrote)
				if got > len(chunk) {
					got = len(chunk)
				}
			}
		}
		if got > 0 {
			pcm = append(pcm, chunk[:got]...)
			res.TotalBytes += int64(got)
			res.FramesWithPCM++
			if res.MinFrameBytes < 0 || got < res.MinFrameBytes {
				res.MinFrameBytes = got
			}
			if got > res.MaxFrameBytes {
				res.MaxFrameBytes = got
			}
		}
		if n+1 < meta.Frames {
			procs["SmackNextFrame"].Call(state)
		}
	}
	if res.MinFrameBytes < 0 {
		res.MinFrameBytes = 0
	}
	runtime.KeepAlive(chunk)
	return res, pcm, nil
}

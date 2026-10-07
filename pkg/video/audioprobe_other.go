//go:build !windows || !386

package video

import "fmt"

// AudioProbeResult mirrors the windows/386 type so cmd/audioprobe links on
// every platform; the diagnostic itself only runs where the installed DLL
// does.
type AudioProbeResult struct {
	Variant       string
	Frames        uint32
	FramesWithPCM uint32
	TotalBytes    int64
	MinFrameBytes int
	MaxFrameBytes int
	DimensionsOK  bool
	Err           error
}

func ProbeAudioTrack(_, _, _ string) ([]AudioProbeResult, error) {
	return nil, fmt.Errorf("video: audio probe must be built for windows/386")
}

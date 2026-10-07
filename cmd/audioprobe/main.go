// Command audioprobe is a throwaway diagnostic: it settles whether the
// installed Smacker DLL will hand over decompressed track data with no sound
// device open, and with what SmackSoundOnOff argument, before any playback
// code is written against the answer. Build it exactly like cutscenehelper,
// windows/386, and point it at one already-extracted .smk file. It writes PCM
// only to the path -pcm names, never inside an install, and prints
// per-variant counts to stdout.
package main

import (
	"flag"
	"fmt"
	"os"

	"againrom/pkg/video"
)

func main() {
	dll := flag.String("dll", "", "absolute lawful installed DLL path")
	input := flag.String("input", "", "absolute private compressed .smk path")
	pcm := flag.String("pcm", "", "absolute path to write the first successful variant's raw PCM (optional)")
	flag.Parse()
	results, err := video.ProbeAudioTrack(*dll, *input, *pcm)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	for _, r := range results {
		fmt.Printf("variant=%-32s dims-ok=%-5v frames=%d frames-with-pcm=%d total-bytes=%d min=%d max=%d err=%v\n",
			r.Variant, r.DimensionsOK, r.Frames, r.FramesWithPCM, r.TotalBytes, r.MinFrameBytes, r.MaxFrameBytes, r.Err)
	}
}

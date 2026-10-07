// Command cutscenehelper is an optional Windows/386 adapter for the lawful
// installed Smacker DLL. Build it separately from the normal amd64 game.
package main

import (
	"flag"
	"fmt"
	"os"

	"againrom/pkg/video"
)

func main() {
	dll := flag.String("dll", "", "absolute lawful installed DLL path")
	input := flag.String("input", "", "absolute private compressed input path")
	packed := flag.Bool("oracle565", false, "release witness: independent packed RGB565 path")
	unpaced := flag.Bool("unpaced", false, "release witness: emit frames without their playback interval")
	flag.Parse()
	if err := video.DecodeNative(*dll, *input, os.Stdout, *packed, !*unpaced); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

package game

import (
	"image"
	"os"
	"strings"
	"testing"
	"time"
)

// Every installed movie of both archives presents at the output region: 640x360,
// or its own 640x480 for the three logos (VIDEO-072, VIDEO-074). Every sidecar
// parses with its declared counts, starts at the origin, and the four pan
// registries are M10/01 and M50/01 in both archives (REG-CUT-053).
func TestReleaseCutscenePresentationAndSidecars(t *testing.T) {
	root := os.Getenv("AGAINROM_ASSETS")
	if root == "" {
		t.Skip("AGAINROM_ASSETS not set")
	}
	tall, wide, movies, pans := 0, 0, 0, map[string]bool{}
	for _, archive := range []string{"video4", "video8"} {
		bank := OpenCutscenes(root, archive)
		if err := bank.open(); err != nil {
			t.Fatal(err)
		}
		for _, e := range bank.index.Entries() {
			prefix := archive + "/"
			if !strings.HasPrefix(e.Address, prefix) || !strings.HasSuffix(e.Address, ".smk") {
				continue
			}
			name := strings.TrimPrefix(e.Address, prefix)
			movies++
			sc, err := bank.sidecar(name)
			if err != nil || sc == nil {
				t.Fatalf("%s %s sidecar: %v %v", archive, name, sc, err)
			}
			if sc.StartX != 0 || sc.StartY != 0 {
				t.Errorf("%s %s start = %d,%d", archive, name, sc.StartX, sc.StartY)
			}
			if len(sc.Pans) > 0 {
				pans[archive+"/"+name] = true
			}
			p, err := bank.Open(name)
			if err != nil {
				t.Fatalf("%s %s: %v", archive, name, err)
			}
			deadline := time.Now().Add(5 * time.Second)
			for p.Frame() == nil && time.Now().Before(deadline) {
				if !p.Advance(time.Now()) {
					break
				}
				time.Sleep(time.Millisecond)
			}
			f := p.Frame()
			p.Close()
			if f == nil {
				t.Fatalf("%s %s: no frame", archive, name)
			}
			switch {
			case f.Rect.Dx() == 640 && f.Rect.Dy() == 360:
				wide++
			case f.Rect.Dx() == 640 && f.Rect.Dy() == 480:
				tall++
			default:
				t.Errorf("%s %s presented at %v", archive, name, f.Rect)
			}
		}
	}
	if movies != 33 || tall != 3 || wide != 30 {
		t.Errorf("movies=%d 640x360=%d 640x480=%d, want 33/30/3", movies, wide, tall)
	}
	want := []string{"video4/m10/01.smk", "video4/m50/01.smk", "video8/m10/01.smk", "video8/m50/01.smk"}
	if len(pans) != len(want) {
		t.Errorf("pan registries = %v, want %v", pans, want)
	}
	for _, w := range want {
		if !pans[w] {
			t.Errorf("pan registry %s missing", w)
		}
	}
}

// The final frames of the two pan movies follow their fade to black: no frame
// of the last fade carries a new palette, so the display stays black to the
// end (VIDEO-071 step 4).
func TestReleaseCutsceneFinalFramesStayBlackAfterFadeOut(t *testing.T) {
	root := os.Getenv("AGAINROM_ASSETS")
	if root == "" {
		t.Skip("AGAINROM_ASSETS not set")
	}
	for _, name := range []string{"m10/01.smk", "m50/01.smk"} {
		p, err := OpenCutscenes(root, "video4").Open(name)
		if err != nil {
			t.Fatal(err)
		}
		var last *image.RGBA
		var n uint32
		var peak byte
		deadline := time.Now().Add(30 * time.Second)
		for time.Now().Before(deadline) && p.Advance(time.Now()) {
			if p.FrameNumber() != n {
				n, last = p.FrameNumber(), p.Frame()
				if n == 40 {
					for i := 0; i < len(last.Pix); i += 4 {
						peak = max(peak, last.Pix[i], last.Pix[i+1], last.Pix[i+2])
					}
				}
			}
			time.Sleep(time.Millisecond)
		}
		p.Close()
		if p.Err() != nil || n < 90 || last == nil {
			t.Fatalf("%s: frames=%d err=%v", name, n, p.Err())
		}
		if peak == 0 {
			t.Errorf("%s: mid-movie frame is black", name)
		}
		for i := 0; i < len(last.Pix); i += 4 {
			if last.Pix[i] != 0 || last.Pix[i+1] != 0 || last.Pix[i+2] != 0 {
				t.Fatalf("%s: final frame %d is not black at byte %d", name, n, i)
			}
		}
	}
}

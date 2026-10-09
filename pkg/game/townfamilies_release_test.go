package game

import (
	"bytes"
	"fmt"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"againrom/pkg/audio"
	"againrom/pkg/town"
)

// writeTownFamiliesShot writes one frame to AGAINROM_SHOT_DIR, or to the
// test's temporary directory when it is unset, and returns the path.
func writeTownFamiliesShot(t *testing.T, dir, root, name string, pix *image.RGBA) string {
	t.Helper()
	dir = filepath.Join(dir, filepath.Base(root))
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, name+".png")
	file, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	encodeErr := png.Encode(file, pix)
	if closeErr := file.Close(); encodeErr == nil {
		encodeErr = closeErr
	}
	if encodeErr != nil {
		t.Fatal(encodeErr)
	}
	return path
}

// TestReleaseTownFamiliesInstalledHorseBabaDervish drives the production
// App over the installed town square and compares every frame with an
// independent decode of the literal sheets (TOWN-439, 489..495).
func TestReleaseTownFamiliesInstalledHorseBabaDervish(t *testing.T) {
	f := releaseFront(t)
	root := f.Archives.Root
	shotDir := os.Getenv("AGAINROM_SHOT_DIR")
	if shotDir == "" {
		shotDir = t.TempDir()
	}
	if v := f.TownSquareArt.Value(); v == nil || len(v.Problems) != 0 {
		t.Fatalf("town art: %v", f.TownSquareArt.Err())
	}
	oracle := loadTownSquareOracle(t, f)
	wantCount := func(name string, got, want int) {
		t.Helper()
		if got != want {
			t.Fatalf("%s: %d frames, want %d", name, got, want)
		}
	}
	art := f.TownSquareArt.Value()
	same := func(name string, got, want []image.Image) {
		t.Helper()
		wantCount(name, len(got), len(want))
		for i := range got {
			if !bytes.Equal(got[i].(*image.RGBA).Pix, want[i].(*image.RGBA).Pix) {
				t.Fatalf("%s frame %d differs from the literal decoder", name, i)
			}
		}
	}
	for p := range oracle.horse {
		for v := range oracle.horse[p] {
			wantCount(fmt.Sprintf("horse%d/a%d", p+1, v+1), len(oracle.horse[p][v]), 15)
			same(fmt.Sprintf("horse%d/a%d", p+1, v+1), art.Pictures(fmt.Sprintf("horse/%d/%d", p, v)), oracle.horse[p][v])
		}
	}
	for p := range oracle.baba {
		for v := range oracle.baba[p] {
			wantCount(fmt.Sprintf("baba%d/a%d", p+1, v+1), len(oracle.baba[p][v]), 31+v)
			same(fmt.Sprintf("baba%d/a%d", p+1, v+1), art.Pictures(fmt.Sprintf("baba/%d/%d", p, v)), oracle.baba[p][v])
		}
	}
	for p := range oracle.dervish {
		wantCount(fmt.Sprintf("dervish%d", p+1), len(oracle.dervish[p]), 30)
		same(fmt.Sprintf("dervish%d", p+1), art.Pictures(fmt.Sprintf("dervish/%d", p)), oracle.dervish[p])
	}

	f.Carried = f.NextParty()
	f.arriveInTown()
	f.Town.announceMission(f.Town.currentMain())
	now := time.Unix(900, 0)
	f.TownAnimationNow = func() time.Time { return now }
	f.TownAnimationRandom = func(int) int { return 0 }
	birdWait := f.townProcess.WaitLatch("birds")
	birdWait.Ready, birdWait.Wait = true, 24*time.Hour
	voices := &exteriorRecorder{}
	f.SoundPlayer = voices
	f.SoundBank = OpenSounds(f.Archives.Root)
	before, label, err := f.Snapshot(false)
	if err != nil {
		t.Fatal(err)
	}
	nativeBefore, err := EncodeSave(before, label)
	if err != nil {
		t.Fatal(err)
	}
	app, screen := exteriorApp(t, f)

	draws := &familyDraws{}
	screen.squareView().SetRawDraw("wildlife", draws.next)
	// horse position 4, baba position 3, dervish position 3 (the baba's,
	// re-rolled) then 1, baba delay 2000 ms, horse delay 2000 ms.
	draws.script = []int{rawH(3), rawB(2), rawB(2), 0, rawZero, rawZero}
	screen.resetTownExterior()
	screen.squareView().SetClockLast(time.Time{})
	screen.TownSquareActive(true)
	if len(draws.script) != 0 {
		t.Fatalf("entry left %v", draws.script)
	}

	shots, hubs := 0, 0
	check := func(name string, dt time.Duration, inspect func(exteriorFrame)) {
		t.Helper()
		pix := exteriorPaint(t, app, &now, dt)
		if dt > time.Duration(rom1Town.Clock.PeriodMS)*time.Millisecond {
			hubs++
		}
		frame := screen.sqExteriorFrame()
		if frame.Dervish.Frame != hubs%30 {
			t.Fatalf("%s: dervish frame %d after %d hubs", name, frame.Dervish.Frame, hubs)
		}
		if inspect != nil {
			inspect(frame)
		}
		want := oracle.compose(frame, -1)
		if !bytes.Equal(pix.Pix, want.Pix) {
			for y := 0; y < 480; y++ {
				for x := 0; x < 640; x++ {
					if pix.RGBAAt(x, y) != want.RGBAAt(x, y) {
						t.Fatalf("%s: pixel %d,%d = %v, oracle %v", name, x, y, pix.RGBAAt(x, y), want.RGBAAt(x, y))
					}
				}
			}
		}
		if name != "" {
			path := writeTownFamiliesShot(t, shotDir, root, "families-"+name, pix)
			shots++
			t.Log("frame", path)
		}
	}

	check("entry", 0, func(frame exteriorFrame) {
		if frame.Horse != (familyFrame{Visible: true, Position: 3}) ||
			frame.Baba != (familyFrame{Visible: true, Position: 2}) ||
			frame.Dervish != (familyFrame{Visible: true, Position: 0}) {
			t.Fatalf("entry families %+v %+v %+v", frame.Horse, frame.Baba, frame.Dervish)
		}
	})
	check("", 2000*time.Millisecond, func(exteriorFrame) {
		if screen.sqWildlife().Member("horse").Active || screen.sqWildlife().Member("baba").Active {
			t.Fatal("elapsed equal to the entry delay armed a family")
		}
	})
	// Horse A3 (raw 22000) and baba A2 (raw 16384) arm on the same paint,
	// baba first. Delays 4499 ms and 4499 ms follow.
	draws.script = []int{rawArmDelayMid, rawHalf, rawArmDelayMid, rawSheet3}
	check("arm", time.Millisecond, func(frame exteriorFrame) {
		if len(draws.script) != 0 || frame.Horse.Sheet != 2 || frame.Horse.Frame != 0 || frame.Baba.Sheet != 1 || frame.Baba.Frame != 0 {
			t.Fatalf("arm frames horse %+v baba %+v, %d draws left", frame.Horse, frame.Baba, len(draws.script))
		}
	})
	wantSound := func(path string) audio.Sample {
		t.Helper()
		sfx, e := OpenContainers(filepath.Join(f.Archives.Root, "sfx.res"))
		if e != nil {
			t.Fatal(e)
		}
		raw, e := sfx.ReadFile("sfx/" + path)
		if e != nil {
			t.Fatal(path, e)
		}
		sample, e := audio.DecodeWAV(raw, audio.DeviceRate)
		if e != nil {
			t.Fatal(path, e)
		}
		return sample
	}
	horse1, horse2 := wantSound("town/horse1.wav"), wantSound("town/horse2.wav")
	count := func(want audio.Sample) int {
		n := 0
		for _, got := range voices.samples {
			if reflect.DeepEqual(got, want) {
				n++
			}
		}
		return n
	}
	if count(horse1)+count(horse2) != 0 {
		t.Fatal("horse sound before its gate")
	}
	check("horse-frame1", 68*time.Millisecond, func(frame exteriorFrame) {
		if frame.Horse.Frame != 1 || frame.Baba.Frame != 1 || count(horse1) != 1 {
			t.Fatalf("first step: horse frame %d baba frame %d Horse1 requests %d", frame.Horse.Frame, frame.Baba.Frame, count(horse1))
		}
	})
	// Inside the dwell the playing Horse1 is not requested again.
	check("", 10*time.Millisecond, func(exteriorFrame) {
		if count(horse1) != 1 {
			t.Fatalf("Horse1 requested again while playing: %d", count(horse1))
		}
	})
	for f := 2; f <= 13; f++ {
		check("", 68*time.Millisecond, func(frame exteriorFrame) {
			if frame.Horse.Frame != f {
				t.Fatalf("hub %d: horse frame %d", f, frame.Horse.Frame)
			}
		})
	}
	check("horse-frame14", 68*time.Millisecond, func(frame exteriorFrame) {
		if frame.Horse.Frame != 14 || count(horse2) != 1 {
			t.Fatalf("frame %d Horse2 requests %d", frame.Horse.Frame, count(horse2))
		}
	})
	check("", 10*time.Millisecond, func(exteriorFrame) {
		if count(horse2) != 1 {
			t.Fatalf("Horse2 requested again while playing: %d", count(horse2))
		}
	})
	check("horse-terminal", 68*time.Millisecond, func(frame exteriorFrame) {
		if frame.Horse.Frame != 0 || screen.sqWildlife().Member("horse").Active {
			t.Fatalf("terminal horse %+v active=%v", frame.Horse, screen.sqWildlife().Member("horse").Active)
		}
		for _, v := range voices.voices {
			if v.stops != 0 {
				t.Fatal("a horse voice was stopped by the episode ending")
			}
		}
	})
	for screen.sqWildlife().Member("baba").Active {
		check("", 68*time.Millisecond, nil)
	}
	check("baba-terminal", 10*time.Millisecond, func(frame exteriorFrame) {
		if frame.Baba.Frame != 0 || frame.Baba.Sheet != 1 {
			t.Fatalf("terminal baba %+v", frame.Baba)
		}
	})
	// More hubs bring the dervish through its wrap from frame 29 to 0.
	for (hubs+1)%30 != 0 {
		check("", 68*time.Millisecond, nil)
	}
	check("dervish-wrap", 68*time.Millisecond, func(frame exteriorFrame) {
		if !frame.Dervish.Visible || frame.Dervish.Frame != 0 {
			t.Fatalf("dervish wrap %+v", frame.Dervish)
		}
	})

	// Leaving the square runs the town sound cleanup: playing horse voices stop.
	var playing []*exteriorVoice
	for _, v := range voices.voices {
		if v.playing {
			playing = append(playing, v)
		}
	}
	if len(playing) == 0 {
		t.Fatal("no horse voice was still playing at leave")
	}
	screen.TownSquareActive(false)
	for _, v := range playing {
		if v.playing || v.stops == 0 {
			t.Fatal("leaving the square did not stop a playing horse voice")
		}
	}

	after, label, err := f.Snapshot(false)
	if err != nil {
		t.Fatal(err)
	}
	nativeAfter, err := EncodeSave(after, label)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(nativeBefore, nativeAfter) {
		t.Fatal("the town families changed native save bytes")
	}
	t.Logf("1282 installed oracle: 27 sheets match the literal decoder; %d frames compared pixel for pixel; Horse1 %d and Horse2 %d requests; native bytes unchanged", shots, count(horse1), count(horse2))
}

// A seeded run on the installed art: every arm comes on the first paint that
// strictly exceeds the delay measured from the last step, with delays inside
// 2000..6999 ms.
func TestReleaseTownFamiliesSeededDelaysFromTheLastStep(t *testing.T) {
	f := releaseFront(t)
	f.Carried = f.NextParty()
	f.arriveInTown()
	f.Town.announceMission(f.Town.currentMain())
	now := time.Unix(1200, 0)
	f.TownAnimationNow = func() time.Time { return now }
	f.TownAnimationRandom = func(int) int { return 0 }
	birdWait := f.townProcess.WaitLatch("birds")
	birdWait.Ready, birdWait.Wait = true, 24*time.Hour
	f.SoundPlayer = &exteriorRecorder{}
	f.SoundBank = OpenSounds(f.Archives.Root)
	app, screen := exteriorApp(t, f)
	screen.resetTownExterior()
	screen.squareView().ResetDraw("wildlife")
	screen.squareView().SetClockLast(time.Time{})
	screen.TownSquareActive(true)
	exteriorPaint(t, app, &now, 0)
	fam := screen.sqWildlife()
	if d := fam.Member("horse").Wait; d < 2000*time.Millisecond || d > 3999*time.Millisecond {
		t.Fatalf("horse entry delay %v", d)
	}
	if d := fam.Member("baba").Wait; d < 2000*time.Millisecond || d > 3999*time.Millisecond {
		t.Fatalf("baba entry delay %v", d)
	}
	const dt = 68 * time.Millisecond
	arms := map[string]int{}
	for i := 0; i < 6000; i++ {
		prev := map[string]town.FamilyMember{"horse": *fam.Member("horse"), "baba": *fam.Member("baba")}
		exteriorAdvanceWithoutBlit(t, app, &now, dt)
		cur := map[string]town.FamilyMember{"horse": *fam.Member("horse"), "baba": *fam.Member("baba")}
		for name, p := range prev {
			c := cur[name]
			if !c.Active || c.Current != 0 {
				continue
			}
			arms[name]++
			// The clock before this paint holds the last step (or entry).
			elapsed := now.Sub(p.Clock)
			if p.Active || elapsed <= p.Wait || elapsed > p.Wait+dt {
				t.Fatalf("%s armed after %v with delay %v (active before: %v)", name, elapsed, p.Wait, p.Active)
			}
			if c.Wait < 2000*time.Millisecond || c.Wait > 6999*time.Millisecond {
				t.Fatalf("%s arm delay %v", name, c.Wait)
			}
		}
	}
	if arms["horse"] < 20 || arms["baba"] < 20 {
		t.Fatalf("seeded run made %v arms", arms)
	}
	t.Logf("installed seeded run: %v arms in 6000 hubs", arms)
}

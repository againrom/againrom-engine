package game

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"againrom/pkg/audio"
	"againrom/pkg/formats/bmp"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
	"github.com/hajimehoshi/ebiten/v2"
)

func TestReleaseSchoolTraining1124InstalledAppFramesSoundAndNative(t *testing.T) {
	f := releaseFront(t)
	if f.TownSchoolArt.Value() == nil {
		t.Fatalf("school art: %v", f.TownSchoolArt.Err())
	}
	decode := func(path string, width int) image.Image {
		t.Helper()
		raw, err := f.Archives.Containers.ReadFile(path)
		if err != nil {
			t.Fatalf("%s: %v", path, err)
		}
		decoded, err := bmp.Decode(raw)
		if err != nil || decoded.Width != width || decoded.Height != 224 {
			t.Fatalf("%s = %dx%d: %v", path, decoded.Width, decoded.Height, err)
		}
		pic := image.NewRGBA(image.Rect(0, 0, decoded.Width, decoded.Height))
		for i, c := range decoded.Pix {
			pic.SetRGBA(i%decoded.Width, i/decoded.Width, color.RGBA{R: c.R, G: c.G, B: c.B, A: 0xff})
		}
		return pic
	}

	// Literal paths, bounds and counts are independent of the production
	// constants and loader. Together these four loops decode all 62 members.
	var fighterTR, fighterM, mageTR, mageM []image.Image
	for i := 0; i <= 18; i++ {
		fighterTR = append(fighterTR, decode(fmt.Sprintf("movies/training/fighter/tr%04d.bmp", i), 160))
	}
	for i := 1; i <= 9; i++ {
		fighterM = append(fighterM, decode(fmt.Sprintf("movies/training/fighter/m%04d.bmp", i), 160))
	}
	for i := 0; i <= 22; i++ {
		mageTR = append(mageTR, decode(fmt.Sprintf("movies/training/mage/tr%04d.bmp", i), 172))
	}
	for i := 1; i <= 11; i++ {
		mageM = append(mageM, decode(fmt.Sprintf("movies/training/mage/m%04d.bmp", i), 172))
	}
	if len(fighterTR)+len(fighterM)+len(mageTR)+len(mageM) != 62 {
		t.Fatal("literal training population is not 62")
	}
	for name, pair := range map[string]struct {
		got  []image.Image
		want []image.Image
	}{
		"fighter/tr": {f.TownSchoolArt.Value().Training[0].Transition, fighterTR},
		"fighter/m":  {f.TownSchoolArt.Value().Training[0].Idle, fighterM},
		"mage/tr":    {f.TownSchoolArt.Value().Training[1].Transition, mageTR},
		"mage/m":     {f.TownSchoolArt.Value().Training[1].Idle, mageM},
	} {
		if len(pair.got) != len(pair.want) {
			t.Fatalf("installed %s count = %d, want %d", name, len(pair.got), len(pair.want))
		}
		for i := range pair.want {
			if !imagesEqual(pair.got[i], pair.want[i]) {
				t.Fatalf("installed %s frame %d differs from literal decode", name, i)
			}
		}
	}

	fighter := f.ChargenParty(ui.ChargenResult{Name: "Training fighter", Choices: []int{0, 0, 0}, Stats: []int{30, 30, 20, 20}})
	mage := f.ChargenParty(ui.ChargenResult{Name: "Training mage", Choices: []int{1, 1, 0}, Stats: []int{20, 20, 30, 30}})
	if len(fighter) == 0 || len(mage) == 0 || fighter[0].Mage || !mage[0].Mage {
		t.Fatal("installed chargen did not provide both school classes")
	}
	f.Carried = mapload.OwnParty([]mapload.PartyMember{fighter[0], mage[0]})
	f.Town.gold = 100000
	f.arriveInTown()
	snapshot, label, err := f.Snapshot(false)
	if err != nil {
		t.Fatal(err)
	}
	payload, err := EncodeSave(snapshot, label)
	if err != nil {
		t.Fatal(err)
	}
	store := SaveStore{Dir: t.TempDir()}
	if _, err := store.Write(time.Unix(1124, 0), payload); err != nil {
		t.Fatal(err)
	}

	now := time.Unix(1124, 0)
	f.TownAnimationNow = func() time.Time { return now }
	f.SchoolRandom = func(n int) int {
		if n != 32768 {
			t.Fatalf("school random range = %d", n)
		}
		return 0
	}
	recorder := &schoolRotateRecorder{}
	soundBank := OpenSounds(f.Archives.Root)
	if soundBank == nil {
		t.Fatal("installed sound archive did not open")
	}
	sounds, err := OpenContainers(filepath.Join(f.Archives.Root, "sfx.res"))
	if err != nil {
		t.Fatal(err)
	}
	rawSound, err := sounds.ReadFile("sfx/Town/School/Rotate.wav")
	if err != nil {
		t.Fatal(err)
	}
	wantSound, err := audio.DecodeWAV(rawSound, audio.DeviceRate)
	if err != nil || len(wantSound.PCM) == 0 {
		t.Fatalf("installed Rotate: %v", err)
	}

	app := f.App("1124-installed-school-training")
	app.Layout(640, 480)
	app.SetSaveSeams(agsSaveSeams(f, store, OriginalStore{}, nil))
	if err := app.HeadlessActivate("load game"); err != nil {
		t.Fatal(err)
	}
	if err := app.HeadlessActivate("@first"); err != nil {
		t.Fatal(err)
	}
	if err := app.HeadlessActivate("SCHOOL"); err != nil {
		t.Fatal(err)
	}
	if err := app.HeadlessStep(); err != nil {
		t.Fatal(err)
	}
	s := f.TownScreen().(*townScreen)
	for n := 0; s.room == roomTalk && n < 64; n++ {
		if err := app.HeadlessActivate("dialogue"); err != nil {
			t.Fatal(err)
		}
	}
	if s.room != roomSchool || !s.schoolTraining.active {
		t.Fatalf("App entered room%d active%v", s.room, s.schoolTraining.active)
	}
	s.CloseTip()
	// Headless load and dialogue routing may request unrelated UI sounds. The
	// recorder begins at the school animation seam so its single request can be
	// compared byte-for-byte with the literal Rotate waveform.
	f.SoundPlayer, f.SoundBank = recorder, soundBank

	mw, _ := readoutWorld(t, sim.Entity{ID: 1124, X: 4, Y: 5})
	f.live = mw
	worldBytes, worldHash := marshalWorld(t, mw.world), mw.world.Hash()
	before, _, err := f.Snapshot(false)
	if err != nil {
		t.Fatal(err)
	}
	nativeBefore, err := EncodeSave(before, "same")
	if err != nil {
		t.Fatal(err)
	}

	drawApp := func(elapsed time.Duration) *image.RGBA {
		t.Helper()
		now = now.Add(elapsed)
		app.Draw(ebiten.NewImage(640, 480))
		// Recompose the read-only view published by that App paint; calling
		// ComposeTownScreen here would deliver a second animation step.
		return ui.ComposeTownSurface(s.TownSurface())
	}
	check := func(name string, actual *image.RGBA, wantMage, wantFighter image.Image) {
		t.Helper()
		view := s.TownSurface()
		view.SchoolTraining = ui.SchoolTrainingFrame{Mage: wantMage, Fighter: wantFighter}
		want := ui.ComposeTownSurface(view)
		if !imagesEqual(actual, want) {
			for y := 0; y < 480; y++ {
				for x := 0; x < 640; x++ {
					if actual.RGBAAt(x, y) != want.RGBAAt(x, y) {
						t.Fatalf("%s first pixel %d,%d = %v, want %v", name, x, y, actual.RGBAAt(x, y), want.RGBAAt(x, y))
					}
				}
			}
			t.Fatalf("%s differs from literal-frame composition", name)
		}
		writeSchoolTrainingWitness(t, f, name, actual)
	}

	frame := drawApp(0)
	check("fighter-tr0000", frame, nil, fighterTR[0])
	if s.schoolTraining.sides[schoolFighterClass].transitionIndex != 0 {
		t.Fatal("zero-time App paint advanced entry tr")
	}
	drawApp(83 * time.Millisecond)
	if s.schoolTraining.sides[schoolFighterClass].transitionIndex != 0 {
		t.Fatal("installed entry tr admitted 83ms equality")
	}
	for i := 0; i < 19; i++ {
		elapsed := 84 * time.Millisecond
		if i == 0 {
			elapsed = time.Millisecond
		}
		frame = drawApp(elapsed)
		if i == 5 {
			check("fighter-tr0006", frame, nil, fighterTR[6])
		}
	}
	if s.schoolTrainingBusy() || s.schoolPanelClass() != schoolFighterClass {
		t.Fatal("installed fighter entry transition did not finish")
	}

	// Prevent the other idle side from arming during this bounded transition.
	s.schoolTrainingStatic.initialized = [schoolClassCount]bool{true, true}
	s.schoolTrainingStatic.idleLast = [schoolClassCount]time.Time{now, now}
	s.schoolTrainingStatic.idleExtra = [schoolClassCount]time.Duration{3276 * time.Millisecond, 0}
	s.TownSurfaceClick(ui.TownSurfaceControl{Kind: ui.TownSurfaceControlNext}, false)
	if s.shopMemberIndex() != 1 || s.TownSurface().Hero.Member != 1 || s.TownSurface().Hero.Figure == nil || len(recorder.samples) != 0 {
		t.Fatalf("installed class picker = member%d hero%d figure%v sounds%d busy%v class%d",
			s.shopMemberIndex(), s.TownSurface().Hero.Member, s.TownSurface().Hero.Figure != nil,
			len(recorder.samples), s.schoolTrainingBusy(), s.schoolPanelClass())
	}
	frame = drawApp(0)
	check("mage-tr0000", frame, mageTR[0], nil)
	for want := 1; want <= 5; want++ {
		frame = drawApp(84 * time.Millisecond)
		if s.schoolTraining.sides[schoolMageClass].transitionIndex != want {
			t.Fatalf("installed mage tr = %d, want %d", s.schoolTraining.sides[schoolMageClass].transitionIndex, want)
		}
	}
	check("mage-tr0005-column01", frame, mageTR[5], nil)
	if s.schoolColumn.frame != 1 || len(recorder.samples) != 1 || !reflect.DeepEqual(recorder.samples[0], wantSound) ||
		recorder.places[0] != (audio.Placement{Left: audio.GainUnit, Right: audio.GainUnit}) {
		t.Fatalf("threshold column/sound = frame%d samples%d", s.schoolColumn.frame, len(recorder.samples))
	}
	columnRaw, err := f.Archives.Containers.ReadFile("graphics/interface/training/column/rt0001.bmp")
	if err != nil {
		t.Fatal(err)
	}
	columnDecoded, err := bmp.Decode(columnRaw)
	if err != nil || columnDecoded.Width != 148 || columnDecoded.Height != 208 {
		t.Fatalf("column literal: %v", err)
	}
	for y := 0; y < 208; y++ {
		for x := 0; x < 148; x++ {
			c := columnDecoded.At(x, y)
			if got := frame.RGBAAt(168+x, 176+y); got != (color.RGBA{R: c.R, G: c.G, B: c.B, A: 0xff}) {
				t.Fatalf("training overpainted column at %d,%d: %v", x, y, got)
			}
		}
	}
	s.schoolDiamond.arm()
	frame = drawApp(0)
	diamondRaw, err := f.Archives.Containers.ReadFile("graphics/interface/training/diamond/on0001.bmp")
	if err != nil {
		t.Fatal(err)
	}
	diamondDecoded, err := bmp.Decode(diamondRaw)
	if err != nil || diamondDecoded.Width != 80 || diamondDecoded.Height != 76 {
		t.Fatalf("diamond literal: %v", err)
	}
	for y := 0; y < 76; y++ {
		for x := 0; x < 80; x++ {
			c := diamondDecoded.At(x, y)
			if got := frame.RGBAAt(200+x, 60+y); got != (color.RGBA{R: c.R, G: c.G, B: c.B, A: 0xff}) {
				t.Fatalf("training changed diamond at %d,%d: %v", x, y, got)
			}
		}
	}
	for i := 0; s.schoolTrainingBusy() && i < 64; i++ {
		drawApp(84 * time.Millisecond)
	}
	if s.schoolTrainingBusy() || s.schoolColumn.frame != 15 || s.schoolPanelClass() != schoolMageClass {
		t.Fatal("installed mage transition did not settle at endpoint")
	}

	// Exact idle boundary and the mage terminal skip, on live App paints.
	s.schoolTrainingStatic.initialized = [schoolClassCount]bool{true, true}
	s.schoolTrainingStatic.idleLast = [schoolClassCount]time.Time{now, now}
	s.schoolTrainingStatic.idleExtra = [schoolClassCount]time.Duration{3276 * time.Millisecond, 0}
	drawApp(3000 * time.Millisecond)
	if s.schoolTraining.sides[schoolMageClass].idleActive {
		t.Fatal("installed mage idle armed at equality")
	}
	frame = drawApp(time.Millisecond)
	check("mage-m0001", frame, mageM[0], nil)
	drawApp(84 * time.Millisecond) // logical -1 -> 0, same cached m0001
	for i := 1; i <= 10; i++ {
		frame = drawApp(84 * time.Millisecond)
	}
	if s.schoolTraining.sides[schoolMageClass].idleIndex != 10 {
		t.Fatal("installed mage idle did not reach m0011")
	}
	check("mage-m0011", frame, mageM[10], nil)
	drawApp(84 * time.Millisecond) // terminal: index8, cached10, hold20
	for i := 0; i < 19; i++ {
		frame = drawApp(84 * time.Millisecond)
	}
	check("mage-m0011-hold", frame, mageM[10], nil)
	frame = drawApp(84 * time.Millisecond)
	if s.schoolTraining.sides[schoolMageClass].idleIndex != 7 || s.schoolTraining.sides[schoolMageClass].idleCached != 7 {
		t.Fatal("installed mage return did not skip m0010/m0009")
	}
	check("mage-m0008-return", frame, mageM[7], nil)
	for s.schoolTraining.sides[schoolMageClass].idleActive {
		drawApp(84 * time.Millisecond)
	}

	// The fighter shares the hold arithmetic but returns normally 8 -> 7.
	s.schoolTraining.sides[schoolFighterClass] = schoolTrainingSideAnimation{}
	s.schoolTrainingStatic.idleLast[schoolFighterClass] = now
	s.schoolTrainingStatic.idleExtra[schoolFighterClass] = 0
	s.schoolTrainingStatic.idleLast[schoolMageClass] = now
	s.schoolTrainingStatic.idleExtra[schoolMageClass] = 3276 * time.Millisecond
	frame = drawApp(3001 * time.Millisecond)
	check("fighter-m0001", frame, nil, fighterM[0])
	for i := 1; i <= 8; i++ {
		frame = drawApp(84 * time.Millisecond)
	}
	drawApp(84 * time.Millisecond) // terminal: cached8, hold20
	for i := 0; i < 19; i++ {
		frame = drawApp(84 * time.Millisecond)
	}
	frame = drawApp(84 * time.Millisecond)
	if s.schoolTraining.sides[schoolFighterClass].idleIndex != 7 || s.schoolTraining.sides[schoolFighterClass].idleCached != 7 {
		t.Fatal("installed fighter did not reverse m0009 -> m0008")
	}
	check("fighter-m0008-return", frame, nil, fighterM[7])
	if len(recorder.samples) != 1 {
		t.Fatalf("idle episodes added %d sound requests", len(recorder.samples)-1)
	}

	after, _, err := f.Snapshot(false)
	if err != nil {
		t.Fatal(err)
	}
	nativeAfter, err := EncodeSave(after, "same")
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(nativeBefore, nativeAfter) || mw.world.Hash() != worldHash || !bytes.Equal(worldBytes, marshalWorld(t, mw.world)) {
		t.Fatal("installed school presentation changed native bytes, simulation hash or binary state")
	}
	t.Logf("installed school training: 62 literal opaque frames; fighter/mage tr, mage skip and fighter reverse reached by App paints; column+diamond retained; one exact Rotate waveform; native/hash unchanged")
}

func writeSchoolTrainingWitness(t *testing.T, f *FrontEnd, name string, pix *image.RGBA) {
	t.Helper()
	dir := os.Getenv("AGAINROM_STORY1124_FRAMES")
	if dir == "" {
		return
	}
	dir, err := filepath.Abs(dir)
	if err != nil {
		t.Fatal(err)
	}
	rel, err := filepath.Rel(f.Archives.Root, dir)
	if err != nil || rel == "." || rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		t.Fatalf("school training capture directory must be outside the install: %q", dir)
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	out, err := os.Create(filepath.Join(dir, name+".png"))
	if err != nil {
		t.Fatal(err)
	}
	encodeErr := png.Encode(out, pix)
	closeErr := out.Close()
	if encodeErr != nil {
		t.Fatal(encodeErr)
	}
	if closeErr != nil {
		t.Fatal(closeErr)
	}
}

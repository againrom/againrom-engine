package game

import (
	"fmt"
	"image/color"
	"image/draw"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"testing"
	"time"
	"unsafe"

	"againrom/pkg/audio"
	"againrom/pkg/render/terrain"
	"againrom/pkg/ui"
)

type censusMusicDevice struct {
	fakePlayer
	polled  bool
	invalid bool
}

func (p *censusMusicDevice) Start(track audio.Track) {
	p.invalid = p.invalid || track.Rate != audio.DeviceRate || len(track.StereoPCM) < 4 || len(track.StereoPCM)%4 != 0
}

func (p *censusMusicDevice) Ended() bool {
	p.polled = true
	return false
}

func headlessCensusMusic(t *testing.T, f *FrontEnd) {
	t.Helper()
	device := &censusMusicDevice{}
	f.MusicPlayer = device
	t.Cleanup(func() {
		if device.invalid {
			t.Error("headless music received invalid decoded PCM")
		}
		if device.polled {
			t.Error("headless census now depends on music completion; use a device with timed playback")
		}
	})
}

func cleanupFrontAudio(t *testing.T, f *FrontEnd) {
	t.Helper()
	devices := []interface{ Stop() }{f.MusicPlayer, f.AmbientPlayer, f.CutsceneAudioPlayer}
	owner := ui.DeliveryOwner(f.SoundPlayer)
	t.Cleanup(func() {
		for _, device := range devices {
			if device != nil {
				device.Stop()
			}
		}
		if owner != nil && owner.Service != nil {
			owner.Service.Close()
		}
	})
}

type retainedTestMusic struct {
	ui.MusicDevice
	stops int
}

func (p *retainedTestMusic) Stop() { p.stops++ }

func TestFrontEndAudioCleanupOwnsOnlyItsOriginalDevices(t *testing.T) {
	owned, other := &retainedTestMusic{}, &retainedTestMusic{}
	ownedSFX := &ui.SharedAudio{Service: audio.NewDelivery(audio.Limits{}, nil)}
	otherSFX := &ui.SharedAudio{Service: audio.NewDelivery(audio.Limits{}, nil)}
	defer ownedSFX.Service.Close()
	defer otherSFX.Service.Close()
	t.Run("case", func(t *testing.T) {
		f := &FrontEnd{RuntimeServices: RuntimeServices{MusicPlayer: owned, SoundPlayer: ownedSFX.NewScope().Player(audio.EffectsChannel)}}
		cleanupFrontAudio(t, f)
		f.MusicPlayer = other
		f.SoundPlayer = otherSFX.NewScope().Player(audio.EffectsChannel)
		if owned.stops != 0 || other.stops != 0 {
			t.Fatal("cleanup interrupted a running case")
		}
		if ownedSFX.Service.Snapshot().Closed || otherSFX.Service.Snapshot().Closed {
			t.Fatal("cleanup closed a running SFX service")
		}
	})
	if owned.stops != 1 || other.stops != 0 {
		t.Fatalf("stops owned/other = %d/%d, want 1/0", owned.stops, other.stops)
	}
	if !ownedSFX.Service.Snapshot().Closed || otherSFX.Service.Snapshot().Closed {
		t.Fatal("cleanup did not close only its captured SFX service")
	}
}

func TestInstallClassEditsStayInTheirFrontEnd(t *testing.T) {
	frame := &terrain.StaticFrame{}
	statics := &terrain.StaticSet{}
	statics.Classes[1] = &terrain.StaticClass{Width: 10, Frames: []*terrain.StaticFrame{frame}, Timeline: []int{3},
		Dead: &terrain.StaticClass{Width: 7}}
	structures := &terrain.StructureSet{}
	structures.Classes[1] = &terrain.StructureClass{Name: "Tower", Frames: []*terrain.StaticFrame{frame}, Timeline: []int{4}, Rank: []int{2}}
	a, b := cloneCandidateStatics(statics), cloneCandidateStatics(statics)
	x, y := cloneCandidateStructures(structures), cloneCandidateStructures(structures)
	a.Classes[1].Width, a.Classes[1].Dead.Width = 90, 91
	a.Classes[1].Frames[0], a.Classes[1].Timeline[0] = nil, 92
	x.Classes[1].Name = "Edited"
	x.Classes[1].Frames[0], x.Classes[1].Timeline[0], x.Classes[1].Rank[0] = nil, 93, 94
	for _, candidate := range []*terrain.StaticSet{statics, b} {
		if c := candidate.Classes[1]; c.Width != 10 || c.Dead.Width != 7 || c.Frames[0] != frame || c.Timeline[0] != 3 {
			t.Fatal("editing one front end changed another static descriptor")
		}
	}
	for _, candidate := range []*terrain.StructureSet{structures, y} {
		if c := candidate.Classes[1]; c.Name != "Tower" || c.Frames[0] != frame || c.Timeline[0] != 4 || c.Rank[0] != 2 {
			t.Fatal("editing one front end changed another structure descriptor")
		}
	}
}

// TestMain runs the package's tests and then checks that no front end wrote
// to a shared install: each share this process built must still equal a
// fresh, unshared read of the same root, value for value, so every front end
// the tests built, played, saved and loaded is a witness.
func TestMain(m *testing.M) {
	if dir := os.Getenv("AGAINROM_PAUSED_RENDER_CHILD"); dir != "" {
		if err := runPausedPresentationProbe(dir); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		os.Exit(0)
	}
	trace := os.Getenv("AGAINROM_TRACE_TESTMAIN") != ""
	began := time.Now()
	code := m.Run()
	if trace {
		fmt.Fprintf(os.Stderr, "TESTMAIN pid=%d tests done code=%d after %s; installSharesWritten begins\n", os.Getpid(), code, time.Since(began).Round(time.Millisecond))
	}
	if code == 0 {
		checked := time.Now()
		for _, problem := range installSharesWritten() {
			fmt.Fprintln(os.Stderr, "FAIL: install share written:", problem)
			code = 1
		}
		if trace {
			fmt.Fprintf(os.Stderr, "TESTMAIN pid=%d installSharesWritten ended after %s\n", os.Getpid(), time.Since(checked).Round(time.Millisecond))
		}
	}
	os.Exit(code)
}

// installSharesWritten compares every share built so far with a fresh read.
func installSharesWritten() []string {
	installShares.mu.Lock()
	keys := make([]string, 0, len(installShares.slots))
	for k := range installShares.slots {
		keys = append(keys, k)
	}
	installShares.mu.Unlock()
	sort.Strings(keys)
	var problems []string
	for _, key := range keys {
		installShares.mu.Lock()
		slot := installShares.slots[key]
		installShares.mu.Unlock()
		if slot.share == nil {
			continue
		}
		fresh, err := loadInstallShare(slot.share.archives.Root)
		if err != nil {
			problems = append(problems, fmt.Sprintf("%s: fresh read: %v", key, err))
			continue
		}
		for _, p := range installShareDiff(slot.share, fresh) {
			problems = append(problems, key+": "+p)
		}
	}
	return problems
}

// installShareDiff walks a shared and a fresh share side by side, unexported
// fields included, and names each path whose value differs.
func installShareDiff(shared, fresh *installShare) []string {
	d := &installShareDiffer{seen: map[[2]uintptr]bool{}}
	d.walk("share", reflect.ValueOf(shared).Elem(), reflect.ValueOf(fresh).Elem())
	return d.out
}

type installShareDiffer struct {
	seen map[[2]uintptr]bool
	out  []string
}

func (d *installShareDiffer) walk(path string, a, b reflect.Value) {
	if len(d.out) >= 20 {
		return
	}
	if a.IsValid() != b.IsValid() || a.IsValid() && a.Type() != b.Type() {
		d.out = append(d.out, path+": shape differs")
		return
	}
	if !a.IsValid() {
		return
	}
	switch a.Kind() {
	case reflect.Pointer, reflect.Interface:
		if a.IsNil() || b.IsNil() {
			if a.IsNil() != b.IsNil() {
				d.out = append(d.out, path+": nil on one side")
			}
			return
		}
		if a.Kind() == reflect.Pointer {
			k := [2]uintptr{a.Pointer(), b.Pointer()}
			if d.seen[k] {
				return
			}
			d.seen[k] = true
		}
		d.walk(path, a.Elem(), b.Elem())
	case reflect.Struct:
		for i := 0; i < a.NumField(); i++ {
			d.walk(path+"."+a.Type().Field(i).Name, a.Field(i), b.Field(i))
		}
	case reflect.Slice, reflect.Array:
		if a.Len() != b.Len() {
			d.out = append(d.out, fmt.Sprintf("%s: length %d shared, %d fresh", path, a.Len(), b.Len()))
			return
		}
		if a.Len() > 0 && installSharePlain(a.Type().Elem()) && (a.Kind() == reflect.Slice || a.CanAddr()) {
			if string(installShareBytes(a)) != string(installShareBytes(b)) {
				d.out = append(d.out, path+": bytes differ")
			}
			return
		}
		for i := 0; i < a.Len(); i++ {
			d.walk(path, a.Index(i), b.Index(i))
		}
	case reflect.Map:
		if a.Len() != b.Len() {
			d.out = append(d.out, fmt.Sprintf("%s: %d keys shared, %d fresh", path, a.Len(), b.Len()))
			return
		}
		for _, k := range a.MapKeys() {
			bv := b.MapIndex(k)
			if !bv.IsValid() {
				d.out = append(d.out, fmt.Sprintf("%s: key %v only shared", path, k))
				continue
			}
			d.walk(path, a.MapIndex(k), bv)
		}
	case reflect.Func, reflect.Chan, reflect.UnsafePointer:
	default:
		if fmt.Sprint(installShareRead(a)) != fmt.Sprint(installShareRead(b)) {
			d.out = append(d.out, path+": value differs")
		}
	}
}

// installSharePlain reports whether values of t hold no pointer, so their
// memory is their whole value.
func installSharePlain(t reflect.Type) bool {
	switch t.Kind() {
	case reflect.Bool, reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr,
		reflect.Float32, reflect.Float64, reflect.Complex64, reflect.Complex128:
		return true
	case reflect.Array:
		return installSharePlain(t.Elem())
	case reflect.Struct:
		for i := 0; i < t.NumField(); i++ {
			if !installSharePlain(t.Field(i).Type) {
				return false
			}
		}
		return true
	}
	return false
}

// installShareBytes is the memory of a slice or addressable array of plain
// values. Padding is zero in both sides: every value was built by the same
// loaders from zeroed memory.
func installShareBytes(v reflect.Value) []byte {
	n := v.Len() * int(v.Type().Elem().Size())
	if v.Kind() == reflect.Slice {
		return unsafe.Slice((*byte)(v.UnsafePointer()), n)
	}
	return unsafe.Slice((*byte)(unsafe.Pointer(v.UnsafeAddr())), n)
}

// installShareRead reads a scalar, exported or not.
func installShareRead(v reflect.Value) any {
	if v.CanInterface() {
		return v.Interface()
	}
	if v.CanAddr() {
		return reflect.NewAt(v.Type(), unsafe.Pointer(v.UnsafeAddr())).Elem().Interface()
	}
	switch v.Kind() {
	case reflect.Bool:
		return v.Bool()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return v.Int()
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return v.Uint()
	case reflect.Float32, reflect.Float64:
		return v.Float()
	case reflect.Complex64, reflect.Complex128:
		return v.Complex()
	case reflect.String:
		return v.String()
	}
	return v.Kind()
}

// TestInstallShareKeyFollowsTheArchives checks that a share is keyed by the
// root and each required archive's size and modification time, and that a
// root missing an archive has no key.
func TestInstallShareKeyFollowsTheArchives(t *testing.T) {
	root := t.TempDir()
	if _, ok := installShareKey(root); ok {
		t.Fatal("a root with no archives has a key")
	}
	for _, name := range RequiredArchives() {
		if err := os.WriteFile(filepath.Join(root, name), []byte(name), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	first, ok := installShareKey(root)
	if !ok {
		t.Fatal("a root with every archive has no key")
	}
	if again, _ := installShareKey(root); again != first {
		t.Fatalf("the key moved with nothing rewritten: %q then %q", first, again)
	}
	graphics := filepath.Join(root, GraphicsArchive)
	later := time.Now().Add(time.Hour)
	if err := os.Chtimes(graphics, later, later); err != nil {
		t.Fatal(err)
	}
	touched, _ := installShareKey(root)
	if touched == first {
		t.Fatal("a rewritten archive kept its key")
	}
	if err := os.WriteFile(graphics, []byte("longer graphics"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(graphics, later, later); err != nil {
		t.Fatal(err)
	}
	if resized, _ := installShareKey(root); resized == touched {
		t.Fatal("a resized archive kept its key")
	}
}

// TestInstallShareDiffSeesAWrite checks the no-write witness itself: a share
// compared with itself has no difference, and one pixel or one unit-class
// field written through a shared pointer is named.
func TestInstallShareDiffSeesAWrite(t *testing.T) {
	root := os.Getenv("AGAINROM_ASSETS")
	if root == "" {
		t.Skip("AGAINROM_ASSETS is not set")
	}
	share, err := sharedInstall(root)
	if err != nil {
		t.Fatal(err)
	}
	fresh, err := loadInstallShare(root)
	if err != nil {
		t.Fatal(err)
	}
	if diff := installShareDiff(share, fresh); len(diff) != 0 {
		t.Fatalf("a fresh read differs from the share: %v", diff)
	}
	f, err := NewFrontEnd(root)
	if err != nil {
		t.Fatal(err)
	}
	if f.Archives != share.archives || f.TownSquareArt.Value() != share.townSquare || f.Units == share.units {
		t.Fatal("the front end does not take the archives and pictures from the share, or holds the shared unit set itself")
	}
	var class int32 = -1
	for id := range fresh.units.Classes {
		if class < 0 || id < class {
			class = id
		}
	}
	fresh.units.Classes[class].Z++
	if diff := installShareDiff(share, fresh); len(diff) == 0 {
		t.Fatal("a unit class field written after the read is not named")
	}
	fresh.units.Classes[class].Z--
	if diff := installShareDiff(share, fresh); len(diff) != 0 {
		t.Fatalf("the restored unit class still differs: %v", diff)
	}
	img, ok := fresh.townSquare.Pictures("base")[0].(draw.Image)
	if !ok {
		t.Fatalf("the town square background is a %T", fresh.townSquare.Pictures("base")[0])
	}
	r, g, b, _ := img.At(0, 0).RGBA()
	img.Set(0, 0, color.RGBA{R: ^uint8(r >> 8), G: ^uint8(g >> 8), B: ^uint8(b >> 8), A: 0xff})
	if diff := installShareDiff(share, fresh); len(diff) == 0 {
		t.Fatal("a town pixel written after the read is not named")
	}
}

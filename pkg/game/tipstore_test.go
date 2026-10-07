package game

import (
	"path/filepath"
	"testing"

	"againrom/pkg/render/terrain"
)

// The local persisted preference store (1018 spec behaviour 4, DIV-160).

// A store with no Path answers the shipped default (tips shown) and never
// touches a file — the "runs, cannot persist" shape DefaultSaveDir's own doc
// describes.
func TestOptionsStoreWithNoPathDefaultsToTipsShown(t *testing.T) {
	var s OptionsStore
	on, err := s.TipsMode()
	if err != nil {
		t.Fatalf("TipsMode with no Path: %v", err)
	}
	if !on {
		t.Fatal("TipsMode with no Path = false, want the shipped default (true)")
	}
	if err := s.SetTipsMode(false); err == nil {
		t.Fatal("SetTipsMode with no Path reported success; want an error, since there is nowhere to write")
	}
}

// A store whose file does not exist yet answers the same default, and does
// not create the file merely by being read.
func TestOptionsStoreMissingFileDefaultsToTipsShown(t *testing.T) {
	s := OptionsStore{Path: filepath.Join(t.TempDir(), "options.txt")}
	on, err := s.TipsMode()
	if err != nil {
		t.Fatalf("TipsMode with a missing file: %v", err)
	}
	if !on {
		t.Fatal("TipsMode with a missing file = false, want true")
	}
}

// SetTipsMode(false) then TipsMode() round-trips across two OptionsStore
// values sharing one Path, the same way a fresh process re-reads what a
// previous one wrote.
func TestOptionsStoreSetTipsModeRoundTrips(t *testing.T) {
	path := filepath.Join(t.TempDir(), "options.txt")
	if err := (OptionsStore{Path: path}).SetTipsMode(false); err != nil {
		t.Fatalf("SetTipsMode(false): %v", err)
	}
	on, err := (OptionsStore{Path: path}).TipsMode()
	if err != nil {
		t.Fatalf("TipsMode after SetTipsMode(false): %v", err)
	}
	if on {
		t.Fatal("TipsMode after SetTipsMode(false) = true, want false")
	}
	if err := (OptionsStore{Path: path}).SetTipsMode(true); err != nil {
		t.Fatalf("SetTipsMode(true): %v", err)
	}
	on, err = (OptionsStore{Path: path}).TipsMode()
	if err != nil {
		t.Fatalf("TipsMode after SetTipsMode(true): %v", err)
	}
	if !on {
		t.Fatal("TipsMode after SetTipsMode(true) = false, want true")
	}
}

// GameSpeed and an unknown key survive a TipsMode write untouched: each
// option writer owns one key and must preserve the rest of the local store.
func TestOptionsStoreSetTipsModePreservesOtherKeys(t *testing.T) {
	path := filepath.Join(t.TempDir(), "options.txt")
	s := OptionsStore{Path: path}
	if err := s.writeAll(map[string]string{"GameSpeed": "2", "Shadows": "1"}); err != nil {
		t.Fatalf("seeding writeAll: %v", err)
	}
	if err := s.SetTipsMode(false); err != nil {
		t.Fatalf("SetTipsMode(false): %v", err)
	}
	m, err := s.readAll()
	if err != nil {
		t.Fatalf("readAll: %v", err)
	}
	if m["GameSpeed"] != "2" || m["Shadows"] != "1" {
		t.Fatalf("readAll after SetTipsMode = %v, want GameSpeed=2 and Shadows=1 preserved", m)
	}
	if m[tipsModeKey] != "0" {
		t.Fatalf("readAll after SetTipsMode(false)[%s] = %q, want %q", tipsModeKey, m[tipsModeKey], "0")
	}
}

func TestOptionsStoreGameSpeedDefaultsSafely(t *testing.T) {
	want := terrain.DefaultCadenceRung

	for _, tc := range []struct {
		name string
		raw  *string
	}{
		{name: "missing file"},
		{name: "malformed", raw: stringPointer("fast")},
		{name: "negative", raw: stringPointer("-1")},
		{name: "past ladder", raw: stringPointer("999")},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := OptionsStore{Path: filepath.Join(t.TempDir(), "options.txt")}
			if tc.raw != nil {
				if err := s.writeAll(map[string]string{gameSpeedKey: *tc.raw}); err != nil {
					t.Fatalf("seed GameSpeed: %v", err)
				}
			}
			got, err := s.GameSpeed()
			if err != nil {
				t.Fatalf("GameSpeed: %v", err)
			}
			if got != want {
				t.Fatalf("GameSpeed = rung %d, want shipped default rung %d", got, want)
			}
		})
	}

	var noPath OptionsStore
	if got, err := noPath.GameSpeed(); err != nil || got != want {
		t.Fatalf("GameSpeed with no Path = %d, %v; want %d, nil", got, err, want)
	}
}

func stringPointer(s string) *string { return &s }

func TestOptionsStoreGameSpeedRoundTripsEveryNormalRungAndPreservesOtherKeys(t *testing.T) {
	path := filepath.Join(t.TempDir(), "options.txt")
	s := OptionsStore{Path: path}
	if err := s.writeAll(map[string]string{tipsModeKey: "0", "Shadows": "1"}); err != nil {
		t.Fatalf("seed options: %v", err)
	}
	for want := terrain.CadenceRungMin; want <= terrain.CadenceRungMax; want++ {
		if err := s.SetGameSpeed(want); err != nil {
			t.Fatalf("SetGameSpeed(%d): %v", want, err)
		}
		got, err := (OptionsStore{Path: path}).GameSpeed()
		if err != nil || got != want {
			t.Fatalf("fresh store GameSpeed after rung %d = %d, %v", want, got, err)
		}
	}
	m, err := s.readAll()
	if err != nil {
		t.Fatalf("readAll: %v", err)
	}
	if m[tipsModeKey] != "0" || m["Shadows"] != "1" {
		t.Fatalf("SetGameSpeed dropped other keys: %v", m)
	}
	if err := s.SetGameSpeed(terrain.CadenceRungMax + 1); err == nil {
		t.Fatal("SetGameSpeed accepted a rung past the ladder")
	}
}

func TestOptionsStoreWimpyModeDefaultsSafelyWithoutRewritingTheStore(t *testing.T) {
	for _, tc := range []struct {
		name string
		raw  *string
	}{
		{name: "missing file"},
		{name: "malformed", raw: stringPointer("high")},
		{name: "negative", raw: stringPointer("-1")},
		{name: "past High", raw: stringPointer("3")},
	} {
		t.Run(tc.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "options.txt")
			s := OptionsStore{Path: path}
			if tc.raw != nil {
				if err := s.writeAll(map[string]string{wimpyModeKey: *tc.raw, "Shadows": "1"}); err != nil {
					t.Fatalf("seed WimpyMode: %v", err)
				}
			}
			got, err := s.WimpyMode()
			if err != nil || got != wimpyModeOff {
				t.Fatalf("WimpyMode = %d, %v; want Off/%d, nil", got, err, wimpyModeOff)
			}
			if tc.raw != nil {
				m, err := s.readAll()
				if err != nil {
					t.Fatalf("readAll after fallback: %v", err)
				}
				if m[wimpyModeKey] != *tc.raw || m["Shadows"] != "1" {
					t.Fatalf("fallback rewrote the store: %v", m)
				}
			}
		})
	}

	var noPath OptionsStore
	if got, err := noPath.WimpyMode(); err != nil || got != wimpyModeOff {
		t.Fatalf("WimpyMode with no Path = %d, %v; want %d, nil", got, err, wimpyModeOff)
	}
}

func TestOptionsStoreWimpyModeRoundTripsAllThreeStatesAndPreservesOtherKeys(t *testing.T) {
	path := filepath.Join(t.TempDir(), "options.txt")
	s := OptionsStore{Path: path}
	if err := s.writeAll(map[string]string{tipsModeKey: "0", gameSpeedKey: "4", "Shadows": "1"}); err != nil {
		t.Fatalf("seed options: %v", err)
	}
	for want := wimpyModeOff; want <= wimpyModeHigh; want++ {
		if err := s.SetWimpyMode(want); err != nil {
			t.Fatalf("SetWimpyMode(%d): %v", want, err)
		}
		got, err := (OptionsStore{Path: path}).WimpyMode()
		if err != nil || got != want {
			t.Fatalf("fresh store WimpyMode after %d = %d, %v", want, got, err)
		}
	}
	m, err := s.readAll()
	if err != nil {
		t.Fatalf("readAll: %v", err)
	}
	if m[tipsModeKey] != "0" || m[gameSpeedKey] != "4" || m["Shadows"] != "1" {
		t.Fatalf("SetWimpyMode dropped other keys: %v", m)
	}
	if err := s.SetWimpyMode(wimpyModeHigh + 1); err == nil {
		t.Fatal("SetWimpyMode accepted a value past High")
	}
}

func TestFrontEndLoadOptionsCachesWimpyModeWithoutTouchingAWorld(t *testing.T) {
	path := filepath.Join(t.TempDir(), "options.txt")
	s := OptionsStore{Path: path}
	if err := s.SetWimpyMode(wimpyModeHigh); err != nil {
		t.Fatalf("SetWimpyMode: %v", err)
	}
	f := &FrontEnd{PersistenceContext: PersistenceContext{Options: s}}
	f.LoadOptions()
	if f.wimpyMode != wimpyModeHigh {
		t.Fatalf("cached WimpyMode = %d, want High/%d", f.wimpyMode, wimpyModeHigh)
	}
}

func TestFrontEndAppRestoresTheStoredNormalGameSpeed(t *testing.T) {
	path := filepath.Join(t.TempDir(), "options.txt")
	s := OptionsStore{Path: path}
	want := terrain.CadenceRungMax
	if err := s.SetGameSpeed(want); err != nil {
		t.Fatalf("SetGameSpeed: %v", err)
	}
	f := missionFrontEnd(t)
	f.Options = OptionsStore{Path: path}
	if got := f.App("stored-speed").MapCadencePreference(); got != want {
		t.Fatalf("App cadence preference = rung %d, want stored rung %d", got, want)
	}
}

// DefaultOptionsPath is one level up from DefaultSaveDir's own answer, for
// both an explicit override and the auto-detected default: the store sits
// BESIDE saves/, not inside it.
func TestDefaultOptionsPathIsBesideDefaultSaveDir(t *testing.T) {
	override := filepath.Join(t.TempDir(), "custom-saves")
	saveDir, err := DefaultSaveDir(override)
	if err != nil {
		t.Fatalf("DefaultSaveDir(override): %v", err)
	}
	optPath, err := DefaultOptionsPath(override)
	if err != nil {
		t.Fatalf("DefaultOptionsPath(override): %v", err)
	}
	want := filepath.Join(filepath.Dir(saveDir), optionsFileName)
	if optPath != want {
		t.Fatalf("DefaultOptionsPath(override) = %q, want %q (beside %q)", optPath, want, saveDir)
	}
}

// FrontEnd.LoadOptions/TipsOff/SetTipsOff (1018 spec behaviour 4): the
// front end's own cached flag mirrors the store, and a toggle press
// persists immediately rather than only in memory.
func TestFrontEndLoadOptionsAndSetTipsOff(t *testing.T) {
	path := filepath.Join(t.TempDir(), "options.txt")
	f := &FrontEnd{PersistenceContext: PersistenceContext{Options: OptionsStore{Path: path}}}
	f.LoadOptions()
	if f.TipsOff() {
		t.Fatal("a fresh store's TipsOff() = true, want false (shipped default is tips shown)")
	}

	f.SetTipsOff(true)
	if !f.TipsOff() {
		t.Fatal("TipsOff() after SetTipsOff(true) = false, want true")
	}
	on, err := (OptionsStore{Path: path}).TipsMode()
	if err != nil {
		t.Fatalf("TipsMode after SetTipsOff(true): %v", err)
	}
	if on {
		t.Fatal("the store still reads TipsMode = true after SetTipsOff(true); the toggle did not persist")
	}

	// A second FrontEnd sharing the same store's Path sees the persisted
	// value on its own LoadOptions, the same way a restarted process would
	// (spec behaviour 4, "survives restart").
	f2 := &FrontEnd{PersistenceContext: PersistenceContext{Options: OptionsStore{Path: path}}}
	f2.LoadOptions()
	if !f2.TipsOff() {
		t.Fatal("a second FrontEnd reading the same store's LoadOptions() = tips shown, want suppressed")
	}
}

// A nil FrontEnd's TipsOff and a FrontEnd with no Options.Path's SetTipsOff
// do not panic — every other install-backed field's "absent source, no-op"
// shape (tipstore.go's own doc).
func TestFrontEndTipsOffAndSetTipsOffDoNotPanicWithNoStore(t *testing.T) {
	var nilFront *FrontEnd
	if nilFront.TipsOff() {
		t.Fatal("a nil FrontEnd's TipsOff() = true, want false")
	}
	f := &FrontEnd{}
	f.SetTipsOff(true)
	if !f.TipsOff() {
		t.Fatal("SetTipsOff(true) on a FrontEnd with no Options.Path did not update the in-memory flag")
	}
}

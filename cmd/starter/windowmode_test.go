package main

import (
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	basepkg "againrom/pkg/base"
)

func TestWindowSettingDefaultsOffAndRoundTrips(t *testing.T) {
	for _, value := range []string{"", "false", "true", "unrecognised"} {
		t.Run(value, func(t *testing.T) {
			file := mustParse(t, "[options]\nwindow = "+value+"\nfuture = keep\n")
			got := loadSettings(file)
			want := value == "true"
			if got.Windowed != want {
				t.Fatalf("window=%t, want %t", got.Windowed, want)
			}
			control := got
			control.Windowed = false
			if !reflect.DeepEqual(control, loadSettings(mustParse(t, "[options]\nfuture = keep\n"))) {
				t.Fatal("the window key changed another setting")
			}
			got.store(file)
			if after := loadSettings(file); after.Windowed != want {
				t.Fatalf("stored window=%t, want %t", after.Windowed, want)
			}
			if !strings.Contains(string(file.Bytes()), "future = keep") {
				t.Fatal("unknown setting was lost")
			}
		})
	}
}

func TestWindowCheckboxIsOffAndPlayForwardsOnlyItsFlag(t *testing.T) {
	for _, profile := range []string{basepkg.ROM1EN, basepkg.ROM1RU, basepkg.ROM2EN, basepkg.ROM2RU} {
		t.Run(profile, func(t *testing.T) {
			r := newRig(t, "[starter]\nlast-base = en\n[bases]\nen = {root}\n")
			r.fakes.validRoots[r.root] = profileInfo(profile, true)
			a := r.app()
			openSettings(t, a)
			found := 0
			for _, item := range a.layout() {
				if item.act.kind == aWindowed {
					found++
					if item.checked || item.text != a.t(cOff) || item.kind != iToggle {
						t.Fatalf("default checkbox: %+v", item)
					}
				}
			}
			if found != 1 {
				t.Fatalf("found %d window checkboxes, want 1", found)
			}
			do(t, a, action{kind: aPlay})
			want := []string{filepath.Join(r.dir, "againrom.exe"), "-assets", r.root, "-base", profile}
			if len(r.fakes.started) != 1 || !reflect.DeepEqual(r.fakes.started[0], want) {
				t.Fatalf("default launch: %q, want %q", r.fakes.started, want)
			}
			before := a.s
			do(t, a, action{kind: aWindowed})
			if !a.s.Windowed {
				t.Fatal("checkbox did not enable the window")
			}
			control := a.s
			control.Windowed = false
			if !reflect.DeepEqual(control, before) {
				t.Fatal("checkbox changed another setting")
			}
			for _, item := range a.layout() {
				if item.act.kind == aWindowed && (!item.checked || item.text != a.t(cOn)) {
					t.Fatalf("checked checkbox: %+v", item)
				}
			}
			do(t, a, action{kind: aPlay})
			want = append(want, "-window")
			if len(r.fakes.started) != 2 || !reflect.DeepEqual(r.fakes.started[1], want) {
				t.Fatalf("enabled launch: %q, want %q", r.fakes.started, want)
			}
			do(t, a, action{kind: aWindowed})
			do(t, a, action{kind: aPlay})
			if len(r.fakes.started) != 3 || !reflect.DeepEqual(r.fakes.started[2], want[:len(want)-1]) {
				t.Fatalf("disabled launch: %q", r.fakes.started)
			}
		})
	}
}

// TestWindowCheckboxSavesAndReloads saves the ticked box through the
// starter's own Save button and reads it back from the written file.
func TestWindowCheckboxSavesAndReloads(t *testing.T) {
	r := newRig(t, "[starter]\nlast-base = en\n[bases]\nen = {root}\n")
	r.fakes.validRoots[r.root] = profileInfo(basepkg.ROM1EN, true)
	a := r.app()
	openSettings(t, a)
	do(t, a, action{kind: aWindowed})
	do(t, a, action{kind: aSave})
	again := r.app()
	openSettings(t, again)
	if !again.s.Windowed {
		t.Fatal("a fresh starter did not reload the saved window setting")
	}
	do(t, again, action{kind: aWindowed})
	do(t, again, action{kind: aSave})
	if r.app().s.Windowed {
		t.Fatal("a fresh starter did not reload the cleared window setting")
	}
}

package main

import (
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	basepkg "againrom/pkg/base"
)

func TestSmoothRegenSettingDefaultsOffAndRoundTrips(t *testing.T) {
	for _, value := range []string{"", "false", "true", "unrecognised"} {
		t.Run(value, func(t *testing.T) {
			file := mustParse(t, "[options]\nsmooth-regen = "+value+"\nfuture = keep\n")
			got := loadSettings(file)
			want := value == "true"
			if got.SmoothRegen != want {
				t.Fatalf("smooth-regen=%t, want %t", got.SmoothRegen, want)
			}
			got.store(file)
			if after := loadSettings(file); after.SmoothRegen != want {
				t.Fatalf("stored smooth-regen=%t, want %t", after.SmoothRegen, want)
			}
			if !strings.Contains(string(file.Bytes()), "future = keep") {
				t.Fatal("unknown setting was lost")
			}
		})
	}
}

func TestSmoothRegenOldFileWithoutTheKeyReadsOff(t *testing.T) {
	got := loadSettings(mustParse(t, "[options]\nchicken = true\nmarkers = true\n"))
	if got.SmoothRegen || !got.Chicken || !got.Markers {
		t.Fatalf("old file read as %+v", got)
	}
}

func TestSmoothRegenCheckboxIsOffAndPlayForwardsOnlyItsFlag(t *testing.T) {
	for _, profile := range []string{basepkg.ROM1EN, basepkg.ROM1RU} {
		t.Run(profile, func(t *testing.T) {
			r := newRig(t, "[starter]\nlast-base = en\n[bases]\nen = {root}\n")
			r.fakes.validRoots[r.root] = profileInfo(profile, true)
			a := r.app()
			openSettings(t, a)
			found := false
			for _, item := range a.layout() {
				if item.act.kind == aSmoothRegen {
					found = true
					if item.checked || item.text != a.t(cOff) || item.kind != iToggle {
						t.Fatalf("default checkbox: %+v", item)
					}
				}
			}
			if !found {
				t.Fatal("smooth regeneration checkbox is absent")
			}
			do(t, a, action{kind: aPlay})
			want := []string{filepath.Join(r.dir, "againrom.exe"), "-assets", r.root, "-base", profile}
			if len(r.fakes.started) != 1 || !reflect.DeepEqual(r.fakes.started[0], want) {
				t.Fatalf("default launch: %q, want %q", r.fakes.started, want)
			}
			before := a.s
			do(t, a, action{kind: aSmoothRegen})
			if !a.s.SmoothRegen {
				t.Fatal("checkbox did not enable smooth regeneration")
			}
			control := a.s
			control.SmoothRegen = false
			if !reflect.DeepEqual(control, before) {
				t.Fatal("checkbox changed another setting")
			}
			do(t, a, action{kind: aPlay})
			want = append(want, "-smooth-regen")
			if len(r.fakes.started) != 2 || !reflect.DeepEqual(r.fakes.started[1], want) {
				t.Fatalf("enabled launch: %q, want %q", r.fakes.started, want)
			}
			do(t, a, action{kind: aSmoothRegen})
			do(t, a, action{kind: aPlay})
			if len(r.fakes.started) != 3 || !reflect.DeepEqual(r.fakes.started[2], want[:len(want)-1]) {
				t.Fatalf("disabled launch: %q", r.fakes.started)
			}
		})
	}
}

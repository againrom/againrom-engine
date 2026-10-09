package main

import (
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	basepkg "againrom/pkg/base"
)

func TestChickenSettingDefaultsOffAndRoundTrips(t *testing.T) {
	for _, value := range []string{"", "false", "true", "unrecognised"} {
		t.Run(value, func(t *testing.T) {
			file := mustParse(t, "[options]\nchicken = "+value+"\nfuture = keep\n")
			got := loadSettings(file)
			want := value == "true"
			if got.Chicken != want {
				t.Fatalf("chicken=%t, want %t", got.Chicken, want)
			}
			got.store(file)
			if after := loadSettings(file); after.Chicken != want {
				t.Fatalf("stored chicken=%t, want %t", after.Chicken, want)
			}
			if !strings.Contains(string(file.Bytes()), "future = keep") {
				t.Fatal("unknown setting was lost")
			}
		})
	}
}

func TestChickenCheckboxIsOffAndPlayForwardsOnlyItsFlag(t *testing.T) {
	for _, profile := range []string{basepkg.ROM1EN, basepkg.ROM1RU} {
		t.Run(profile, func(t *testing.T) {
			r := newRig(t, "[starter]\nlast-base = en\n[bases]\nen = {root}\n")
			r.fakes.validRoots[r.root] = profileInfo(profile, true)
			a := r.app()
			found := false
			for _, item := range a.layout() {
				if item.act.kind == aChicken {
					found = true
					if item.checked || item.text != "off" || item.kind != iToggle {
						t.Fatalf("default checkbox: %+v", item)
					}
				}
			}
			if !found {
				t.Fatal("Chicken checkbox is absent")
			}
			do(t, a, action{kind: aPlay})
			want := []string{filepath.Join(r.dir, "againrom.exe"), "-assets", r.root, "-base", profile}
			if len(r.fakes.started) != 1 || !reflect.DeepEqual(r.fakes.started[0], want) {
				t.Fatalf("default launch: %q, want %q", r.fakes.started, want)
			}
			before := a.s
			do(t, a, action{kind: aChicken})
			if !a.s.Chicken {
				t.Fatal("checkbox did not enable Chicken")
			}
			control := a.s
			control.Chicken = false
			if !reflect.DeepEqual(control, before) {
				t.Fatal("checkbox changed another setting")
			}
			do(t, a, action{kind: aPlay})
			want = append(want, "-chicken")
			if len(r.fakes.started) != 2 || !reflect.DeepEqual(r.fakes.started[1], want) {
				t.Fatalf("enabled launch: %q, want %q", r.fakes.started, want)
			}
			do(t, a, action{kind: aChicken})
			do(t, a, action{kind: aPlay})
			if len(r.fakes.started) != 3 || !reflect.DeepEqual(r.fakes.started[2], want[:len(want)-1]) {
				t.Fatalf("disabled launch: %q", r.fakes.started)
			}
		})
	}
}

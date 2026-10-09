package main

import (
	"reflect"
	"testing"

	"againrom/pkg/game"
)

func TestChickenFlagDefaultsOffAndAcceptsBooleanForms(t *testing.T) {
	for _, tc := range []struct {
		name string
		args []string
		want bool
	}{
		{"default", nil, false},
		{"on", []string{"-chicken"}, true},
		{"explicit on", []string{"-chicken=true"}, true},
		{"explicit off", []string{"-chicken=false"}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := parse(tc.args)
			if err != nil || got.chicken != tc.want {
				t.Fatalf("parse(%q): chicken=%t err=%v, want %t", tc.args, got.chicken, err, tc.want)
			}
			control, err := parse(nil)
			if err != nil {
				t.Fatal(err)
			}
			got.chicken = false
			if !reflect.DeepEqual(got, control) {
				t.Fatal("Chicken changed another launch option")
			}
		})
	}
}

func TestChickenFlagReachesTheFrontEnd(t *testing.T) {
	root := defaultInstall(t)
	for _, enabled := range []bool{false, true} {
		front, err := frontEnd(root, options{chicken: enabled}, game.OptionsStore{})
		if err != nil {
			t.Fatal(err)
		}
		if got := front.ChickenAtMissionStart(); got != enabled {
			t.Fatalf("front-end launch option=%t, want %t", got, enabled)
		}
	}
}

package main

import (
	"reflect"
	"testing"
)

func TestSmoothRegenFlagDefaultsOffAndAcceptsBooleanForms(t *testing.T) {
	for _, tc := range []struct {
		name string
		args []string
		want bool
	}{
		{"default", nil, false},
		{"on", []string{"-smooth-regen"}, true},
		{"explicit on", []string{"-smooth-regen=true"}, true},
		{"explicit off", []string{"-smooth-regen=false"}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := parse(tc.args)
			if err != nil || got.smoothRegen != tc.want {
				t.Fatalf("parse(%q): smoothRegen=%t err=%v, want %t", tc.args, got.smoothRegen, err, tc.want)
			}
			control, err := parse(nil)
			if err != nil {
				t.Fatal(err)
			}
			got.smoothRegen = false
			if !reflect.DeepEqual(got, control) {
				t.Fatal("-smooth-regen changed another launch option")
			}
		})
	}
}

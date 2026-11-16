package main

import (
	"reflect"
	"testing"
)

func TestMitchellFlagDefaultsOffAndChangesNothingElse(t *testing.T) {
	control, err := parse(nil)
	if err != nil || control.mitchell {
		t.Fatalf("default parse: mitchell=%t err=%v", control.mitchell, err)
	}
	got, err := parse([]string{"-mitchell"})
	if err != nil || !got.mitchell {
		t.Fatalf("-mitchell: mitchell=%t err=%v", got.mitchell, err)
	}
	got.mitchell = false
	if !reflect.DeepEqual(got, control) {
		t.Fatal("-mitchell changed another launch option")
	}
}

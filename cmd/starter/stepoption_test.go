package main

import (
	"reflect"
	"testing"
)

func TestStepOptionIgnoresNoAction(t *testing.T) {
	var a app
	before := a.s
	if a.stepOption(aNone) {
		t.Fatal("stepOption claimed aNone")
	}
	if !reflect.DeepEqual(a.s, before) {
		t.Fatalf("aNone changed the settings: %+v", a.s)
	}
}

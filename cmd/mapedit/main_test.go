package main

import (
	"io"
	"strings"
	"testing"
)

func TestMapEditor1092ArgumentFailuresDoNotOpenWindow(t *testing.T) {
	for _, c := range []struct {
		args []string
		want string
	}{
		{[]string{"extra"}, "unexpected arguments"},
		{[]string{"-size", "800x600garbage"}, "invalid size"},
		{[]string{"-size", "639x480"}, "invalid size"},
		{[]string{"-check"}, "requires -map"},
		{[]string{"-scroll-details", "1"}, "requires -snapshot"},
		{[]string{"-scroll-details", "-1"}, "range"},
		{[]string{"-focus-reference", "1"}, "requires -select"},
		{[]string{"-focus-reference", "-1"}, "positive reference"},
		{[]string{"-new", "32x32", "-map", "scenario/10.alm"}, "exclude each other"},
		{[]string{"-out", "x.alm"}, "-out requires -new"},
	} {
		if err := run(c.args, io.Discard); err == nil || !strings.Contains(err.Error(), c.want) {
			t.Fatalf("%v: %v", c.args, err)
		}
	}
}

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
	} {
		if err := run(c.args, io.Discard); err == nil || !strings.Contains(err.Error(), c.want) {
			t.Fatalf("%v: %v", c.args, err)
		}
	}
}

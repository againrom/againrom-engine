package main

import (
	"io"
	"strings"
	"testing"
)

func TestModMissionNeedsEveryArgument(t *testing.T) {
	for _, args := range [][]string{nil, {"-install", "x", "-map", "10"}, {"-install", "x", "-out", "y"}, {"-map", "10", "-out", "y"}} {
		if err := run(args, io.Discard); err == nil || !strings.Contains(err.Error(), "usage") {
			t.Errorf("%v: %v", args, err)
		}
	}
}

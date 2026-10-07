package main

import "testing"

// The address filter is the census's DENOMINATOR: everything it rejects is a
// file the count never mentions, so a filter that is too narrow reports a
// smaller corpus rather than a smaller answer, and nothing about the output
// says so. It is tested from addresses written here — no install is read.
func TestIsEventText(t *testing.T) {
	for _, tc := range []struct {
		addr string
		want bool
	}{
		{"main/text/battle/m10/event01.txt", true},
		{"main/text/battle/m151/event13.txt", true},
		{"MAIN/TEXT/BATTLE/M10/EVENT01.TXT", true},
		{"main/text/battle/m10/event1.txt", true},

		{"main/text/inn/npc/npc22m30.txt", false},
		{"main/text/battle/event01.txt", false},
		{"main/text/battle/m10/sub/event01.txt", false},
		{"main/text/battle/m10/event01.bin", false},
		{"graphics/interface/ar1.bmp", false},
		{"", false},
	} {
		if got := isEventText(tc.addr); got != tc.want {
			t.Errorf("isEventText(%q) = %v, want %v", tc.addr, got, tc.want)
		}
	}
}

// The verb is read before the flags are parsed. Handing the flag package a
// non-flag first argument makes it stop there, so every flag after the verb
// would be left unparsed and the asset root would come out empty — a failure
// that reads as "no install" rather than as "the arguments were not understood".
func TestTheVerbDoesNotSwallowTheFlags(t *testing.T) {
	if err := run([]string{"census", "-assets", ""}, discard{}); err == nil {
		t.Fatal("an empty asset root must be refused")
	} else if got := err.Error(); got == "usage: dlgtool census [-assets DIR]" {
		t.Fatalf("the flags were not parsed: %v", got)
	}
	if err := run(nil, discard{}); err == nil {
		t.Fatal("no verb must be refused")
	}
	if err := run([]string{"list"}, discard{}); err == nil {
		t.Fatal("an unknown verb must be refused")
	}
}

type discard struct{}

func (discard) Write(p []byte) (int, error) { return len(p), nil }

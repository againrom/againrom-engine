package main

// Characterization pin for the standalone viewer's command-line surface, written
// BEFORE its load path is moved into a library package and modified by nothing
// afterwards.
//
// The nine tests in main_test.go pin what the tool PRINTS. Nothing pinned which
// flags it accepts, so a flag renamed or dropped during the move would have
// changed the tool's observable interface while every existing assertion stayed
// green. This closes that gap.

import (
	"io"
	"strings"
	"testing"
)

// shippedFlags is the standalone viewer's complete flag set, as shipped.
//
// The inventory is HAND-WRITTEN and that is the whole instrument: it is not read
// back off the flag set, so a flag added to run() without being added here and
// to the usage line below leaves this file green while the tool's documented
// surface and its real one part company. 0017 T12 adds two, in the position the
// usage line gives them; 0051 T4 adds two more, -blocked and the -databin path
// it reads its definition table from.
var shippedFlags = []struct {
	name string
	arg  string // a value to pass, or "" for a boolean flag
}{
	{"assets", "somedir"},
	{"graphics", "some.res"},
	{"map", "some.alm"},
	{"check", ""},
	{"noanimation", ""},
	{"speed", "4"},
	{"objects", ""},
	{"units", ""},
	{"statics", ""},
	{"staticmarkers", ""},
	{"structures", ""},
	{"ruins", ""},
	{"objectanim", ""},
	{"unshaded", ""},
	{"flat", ""},
}

// shippedUsage is the usage line as shipped, character for character.
const shippedUsage = "usage: mapview -assets <dir> -map <file.alm> [-graphics <file.res>] " +
	"[-databin <file.res>] [-speed 0..8] [-noanimation] [-objects] [-units] [-statics] " +
	"[-staticmarkers] [-structures] [-ruins] [-objectanim] [-unshaded] [-flat] [-blocked] [-check]"

func TestFlagSet(t *testing.T) {
	t.Run("the usage line is unchanged", func(t *testing.T) {
		if usage != shippedUsage {
			t.Errorf("usage line changed:\n got %q\nwant %q", usage, shippedUsage)
		}
	})

	t.Run("every shipped flag is still defined", func(t *testing.T) {
		// A flag the parser does not know produces "flag provided but not
		// defined". Any other outcome means the flag parsed; the run then fails
		// later for want of a map, which is not what this pin is about.
		for _, f := range shippedFlags {
			args := []string{"-" + f.name}
			if f.arg != "" {
				args = append(args, f.arg)
			}
			err := run(args, io.Discard)
			if err != nil && strings.Contains(err.Error(), "not defined") {
				t.Errorf("-%s is no longer a defined flag: %v", f.name, err)
			}
		}
	})

	t.Run("an unknown flag is still rejected", func(t *testing.T) {
		// Guards the check above from passing vacuously: if the parser accepted
		// everything, "every shipped flag is defined" would prove nothing.
		err := run([]string{"-nosuchflag"}, io.Discard)
		if err == nil {
			t.Fatalf("an undefined flag was accepted")
		}
		if !strings.Contains(err.Error(), "not defined") {
			t.Errorf("an undefined flag gave %q, want a \"not defined\" error", err)
		}
	})

	t.Run("the usage line names every flag the parser defines", func(t *testing.T) {
		// Ties the two halves together: a flag added without being documented, or
		// documented without existing, breaks one of these.
		for _, f := range shippedFlags {
			if !strings.Contains(shippedUsage, "-"+f.name) {
				t.Errorf("usage line does not mention -%s", f.name)
			}
		}
	})

	t.Run("-map is required and the failure quotes the usage line", func(t *testing.T) {
		err := run(nil, io.Discard)
		if err == nil {
			t.Fatalf("run with no arguments succeeded")
		}
		if !strings.Contains(err.Error(), shippedUsage) {
			t.Errorf("the missing-map error does not quote the usage line: %q", err)
		}
	})
}

// Command touchedtests lists the tests a change can reach: the ordinary and
// release tests that call, directly or through other functions, a symbol, test
// helper or testdata file that `git diff <base>...HEAD` touches. It
// type-checks the checkout with go/types (internal/archtest's loader), so a
// method call reaches the method of its own type, and a call through an
// interface every implementation. A test may be selected that cannot fail.
//
// Usage: touchedtests [-root dir] [-explain] <base-ref>. The checkout at dir
// must be at HEAD with no change to a tracked file. Each output line is an
// import path, a test name and "ordinary" or "release", separated by tabs.
// scripts/check-touched.sh runs them.
package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"againrom/internal/archtest"
)

func main() {
	root := flag.String("root", ".", "the checkout, at HEAD")
	explain := flag.Bool("explain", false, "print each changed declaration with the tests it reaches alone")
	flag.Parse()
	if flag.NArg() != 1 {
		fmt.Fprintln(os.Stderr, "usage: touchedtests [-root dir] [-explain] <base-ref>")
		os.Exit(2)
	}
	if *explain {
		Explain = os.Stderr
	}
	if err := run(*root, flag.Arg(0), os.Stdout, os.Stderr); err != nil {
		fmt.Fprintln(os.Stderr, "touchedtests:", err)
		os.Exit(1)
	}
}

func run(root, base string, out, log io.Writer) error {
	start := time.Now()
	root, err := filepath.Abs(root)
	if err != nil {
		return err
	}
	if dirty, err := git(root, "status", "--porcelain", "--untracked-files=no"); err != nil {
		return err
	} else if len(strings.TrimSpace(string(dirty))) > 0 {
		return fmt.Errorf("a tracked file of %s differs from HEAD; commit it first", root)
	}
	diff, err := git(root, "diff", "--no-renames", "--unified=0", "--no-color", base+"...HEAD", "--")
	if err != nil {
		return err
	}
	changes := parseDiff(string(diff))
	m, err := archtest.LoadTypeCheckedModule(root)
	if err != nil {
		return err
	}
	loaded := time.Since(start)
	tests, ignored, err := Select(m, root, changes)
	if err != nil {
		return err
	}
	for _, t := range tests {
		kind := "ordinary"
		if t.Release {
			kind = "release"
		}
		fmt.Fprintf(out, "%s\t%s\t%s\n", t.Package, t.Name, kind)
	}
	fmt.Fprintf(log, "touchedtests: %d changed files, %d tests selected, %d changed paths not Go or testdata; type-checked in %.1f s, selected in %.1f s\n",
		len(changes), len(tests), len(ignored), loaded.Seconds(), (time.Since(start) - loaded).Seconds())
	return nil
}

func git(root string, args ...string) ([]byte, error) {
	cmd := exec.Command("git", append([]string{"-C", root}, args...)...)
	b, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("git %s: %v", strings.Join(args, " "), err)
	}
	return b, nil
}

var (
	diffFile = regexp.MustCompile(`^\+\+\+ b/(.*)$`)
	diffHunk = regexp.MustCompile(`^@@ -\d+(?:,\d+)? \+(\d+)(?:,(\d+))? @@`)
)

// parseDiff reads the changed files and the touched lines of their new
// versions from a zero-context diff. A pure deletion touches the line before
// the cut and the line after it. A deleted file is not listed.
func parseDiff(diff string) []Change {
	var out []Change
	for _, l := range strings.Split(diff, "\n") {
		if m := diffFile.FindStringSubmatch(l); m != nil {
			out = append(out, Change{Path: m[1]})
			continue
		}
		if strings.HasPrefix(l, "+++ ") { // /dev/null: a deleted file
			out = append(out, Change{})
			continue
		}
		if m := diffHunk.FindStringSubmatch(l); m != nil && len(out) > 0 {
			from, _ := strconv.Atoi(m[1])
			n := 1
			if m[2] != "" {
				n, _ = strconv.Atoi(m[2])
			}
			c := &out[len(out)-1]
			if n == 0 {
				c.Spans = append(c.Spans, Span{from, from + 1})
			} else {
				c.Spans = append(c.Spans, Span{from, from + n - 1})
			}
		}
	}
	kept := out[:0]
	for _, c := range out {
		if c.Path != "" {
			kept = append(kept, c)
		}
	}
	return kept
}

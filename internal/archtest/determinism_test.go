package archtest

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestCheckSimDeterminismNamesEachViolation drives the pure evaluator with
// synthetic sources, one banned thing per case. The live pkg/sim is clean, so a
// scan of it can only ever show the check passing; these cases are where it is
// shown to FAIL, and each banned thing is caught on its own rather than only in
// company.
func TestCheckSimDeterminismNamesEachViolation(t *testing.T) {
	cases := []struct {
		name     string
		src      string
		want     int    // violations expected
		wantSub  string // substring expected in the first violation's message
		wantFrom string // exact From expected, where the line matters
	}{
		{
			name:    "import os",
			src:     "package sim\n\nimport \"os\"\n\nvar assets = os.Getenv(\"AGAINROM_ASSETS\")\n",
			want:    1,
			wantSub: "imports os",
		},
		{
			name:    "import time",
			src:     "package sim\n\nimport \"time\"\n\nfunc stamp() int64 { return time.Now().UnixNano() }\n",
			want:    1,
			wantSub: "imports time",
		},
		{
			name:    "import math/rand",
			src:     "package sim\n\nimport \"math/rand\"\n\nfunc roll() int { return rand.Intn(6) }\n",
			want:    1,
			wantSub: "imports math/rand",
		},
		{
			// The same generator in a newer package is the same generator.
			name:    "import math/rand/v2 (nested under a banned root)",
			src:     "package sim\n\nimport \"math/rand/v2\"\n\nfunc roll() int { return rand.IntN(6) }\n",
			want:    1,
			wantSub: "math/rand/v2",
		},
		{
			name:    "import os/exec (nested under a banned root)",
			src:     "package sim\n\nimport \"os/exec\"\n\nvar run = exec.Command\n",
			want:    1,
			wantSub: "os/exec",
		},
		{
			// A DECLARED float, with no literal anywhere: a check that looked for
			// literals alone would let the field through.
			name:    "float64 declared as a field type",
			src:     "package sim\n\ntype entity struct {\n\tx int32\n\ty float64\n}\n",
			want:    1,
			wantSub: "float64",
		},
		{
			name:    "float32 used only as a conversion",
			src:     "package sim\n\nfunc scale(n int32) int32 {\n\treturn int32(float32(n) / 2)\n}\n",
			want:    1,
			wantSub: "float32",
		},
		{
			name:    "complex128 declared",
			src:     "package sim\n\nvar z complex128\n",
			want:    1,
			wantSub: "complex128",
		},
		{
			// A float LITERAL inside a function body, naming no float type: a
			// check that read import declarations alone would miss it.
			name:     "float literal in a function body",
			src:      "package sim\n\nfunc drift() int32 {\n\tstep := 1.5\n\treturn int32(step)\n}\n",
			want:     1,
			wantSub:  "floating-point literal 1.5",
			wantFrom: "pkg/sim/x.go:4", // the line, so a reader is sent to the finding
		},
		{
			name:    "imaginary literal in a function body",
			src:     "package sim\n\nfunc wobble() int32 {\n\tz := 2i\n\t_ = z\n\treturn 0\n}\n",
			want:    1,
			wantSub: "imaginary literal 2i",
		},
		{
			name: "a clean source",
			src: "package sim\n\nimport (\n\t\"encoding/binary\"\n\t\"hash/fnv\"\n\t\"sort\"\n)\n\n" +
				"func encode(tick uint64, ids []uint32) uint64 {\n" +
				"\tsort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })\n" +
				"\tb := make([]byte, 8)\n" +
				"\tbinary.LittleEndian.PutUint64(b, tick<<8|uint64(len(ids)))\n" +
				"\th := fnv.New64a()\n\th.Write(b)\n\treturn h.Sum64()\n}\n",
			want: 0,
		},
		{
			// The pkg/sim/rng.go case, reproduced: the package explains why
			// math/rand is not its generator, and the explanation is worth more
			// than the convenience of a text grep. Parsed syntax has no comments
			// in it, so this is clean.
			name: "banned names in a comment or a string are not code",
			src: "package sim\n\n" +
				"// math/rand is not an option here: it is banned in this package, and it\n" +
				"// is process-global besides. No float64 or complex128 value appears below,\n" +
				"// and nothing here reads os or time.\n" +
				"const gamma = 0x9E3779B97F4A7C15\n\n" +
				"var note = \"no float64 here, and no os or time either\"\n",
			want: 0,
		},
		{
			// The match is on the path root at a segment boundary. These three
			// are not the banned packages, and a check that matched by substring
			// or by bare prefix would say they were.
			name: "look-alike import paths are not the banned ones",
			src: "package sim\n\nimport (\n\t\"math/bits\"\n\t\"math/random\"\n\t\"oshelper\"\n\t\"timeline\"\n)\n\n" +
				"var _ = bits.Len64\nvar _ = random.Roll\nvar _ = oshelper.Path\nvar _ = timeline.Index\n",
			want: 0,
		},
		{
			// A file the scan cannot read is a file it can say nothing about, so
			// it is a violation rather than a silent skip — the same reasoning as
			// the empty file set below.
			name:    "a source that does not parse is a violation, not a skip",
			src:     "package sim\n\nfunc broken( {\n",
			want:    1,
			wantSub: "does not parse",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			vs := CheckSimDeterminism(map[string]string{"pkg/sim/x.go": tc.src})
			if len(vs) != tc.want {
				t.Fatalf("expected %d violation(s), got %d: %v", tc.want, len(vs), vs)
			}
			if tc.want == 0 {
				return
			}
			if !strings.Contains(vs[0].String(), tc.wantSub) {
				t.Errorf("violation %q does not mention %q", vs[0].String(), tc.wantSub)
			}
			if !strings.HasPrefix(vs[0].From, "pkg/sim/x.go") {
				t.Errorf("violation should name the file it was found in, got From=%q", vs[0].From)
			}
			if tc.wantFrom != "" && vs[0].From != tc.wantFrom {
				t.Errorf("violation should be located at %q, got %q", tc.wantFrom, vs[0].From)
			}
		})
	}
}

// TestCheckSimDeterminismReportsInFileOrder pins that one input yields one
// report: findings come back in path order and then in source order, never in Go's
// map-iteration order, so a failing scan reads the same on every run.
func TestCheckSimDeterminismReportsInFileOrder(t *testing.T) {
	files := map[string]string{
		"pkg/sim/c.go": "package sim\n\nimport \"time\"\n\nvar _ = time.Now\n",
		"pkg/sim/a.go": "package sim\n\nimport \"os\"\n\nvar _ = os.Getpid\n",
		"pkg/sim/b.go": "package sim\n\nvar x float64\n",
	}
	want := []string{"pkg/sim/a.go:3", "pkg/sim/b.go:3", "pkg/sim/c.go:3"}
	for range 8 { // map order is randomised per iteration, not once per process
		vs := CheckSimDeterminism(files)
		if len(vs) != len(want) {
			t.Fatalf("expected %d violations, got %v", len(want), vs)
		}
		for i, w := range want {
			if vs[i].From != w {
				t.Fatalf("violation %d: expected %s, got %s (full report %v)", i, w, vs[i].From, vs)
			}
		}
	}
}

// TestCheckSimDeterminismEmptyFileSet pins that finding nothing to read is a
// violation and not a pass: a scan whose loader has gone hollow otherwise
// reports exactly what a clean package reports.
func TestCheckSimDeterminismEmptyFileSet(t *testing.T) {
	for _, files := range []map[string]string{nil, {}} {
		vs := CheckSimDeterminism(files)
		if len(vs) != 1 {
			t.Fatalf("expected 1 violation for an empty file set, got %v", vs)
		}
		if !strings.Contains(vs[0].String(), "no sources scanned") {
			t.Errorf("violation %q does not say the scan read nothing", vs[0].String())
		}
	}
}

// TestSimSourcesAreDeterministic is AC-7 on the live tree: pkg/sim's production
// sources import no os, time or math/rand and name no floating-point type or
// literal.
func TestSimSourcesAreDeterministic(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	root, err := FindModuleRoot(wd)
	if err != nil {
		t.Fatalf("module root not found from %s: %v", wd, err)
	}
	files, err := LoadSimSources(root)
	if err != nil {
		t.Fatalf("load sim sources: %v", err)
	}
	if len(files) == 0 {
		t.Fatal("loader found no pkg/sim sources; the scan would pass by reading nothing")
	}
	for _, v := range CheckSimDeterminism(files) {
		t.Errorf("unexpected determinism violation: %s", v)
	}

	// The tree carries the evidence that separates a syntax scan from a text
	// grep: pkg/sim names math/rand in prose - rng.go to say why it is not the
	// generator, doc.go to state the ban - while importing no such thing. A grep
	// fails here; this check reads parsed syntax, so it passes. If no source
	// says it any more, that distinction is no longer witnessed on the live tree
	// and only the synthetic comment case above still makes it.
	prose := 0
	for _, src := range files {
		if strings.Contains(src, "math/rand") {
			prose++
		}
	}
	if prose == 0 {
		t.Error("no pkg/sim production source still names math/rand in prose: this test no longer tells a syntax scan from a text grep")
	}
}

// TestLoadSimSourcesSkipsTestFiles pins the loader's one exclusion.
func TestLoadSimSourcesSkipsTestFiles(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "pkg", "sim")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	write := func(name, content string) {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("world.go", "package sim\n\ntype world struct{ tick uint64 }\n")
	write("step.go", "package sim\n\nfunc step(w *world) { w.tick++ }\n")
	write("world_test.go", "package sim\n\nimport (\n\t\"os\"\n\t\"testing\"\n)\n\n"+
		"func TestX(t *testing.T) {\n\t_ = os.Getpid()\n\tvar f float64 = 1.5\n\t_ = f\n}\n")
	write("notes.txt", "float64 os time math/rand\n")

	files, err := LoadSimSources(root)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	// Two production sources, so a loader that stopped after one would be short
	// here rather than only where a later assertion happened to notice.
	if len(files) != 2 {
		t.Fatalf("expected both production sources and nothing else, got %v", keysOf(files))
	}
	for _, want := range []string{"pkg/sim/world.go", "pkg/sim/step.go"} {
		if _, ok := files[want]; !ok {
			t.Fatalf("expected %s to be loaded, got %v", want, keysOf(files))
		}
	}
	if vs := CheckSimDeterminism(files); len(vs) != 0 {
		t.Errorf("test-file and non-Go content must not reach the scan, got %v", vs)
	}
}

// TestLoadSimSourcesMissingPackage pins that an absent pkg/sim is an error rather
// than an empty set quietly handed on.
func TestLoadSimSourcesMissingPackage(t *testing.T) {
	if _, err := LoadSimSources(t.TempDir()); err == nil {
		t.Fatal("expected an error when pkg/sim does not exist")
	}
}

func keysOf(m map[string]string) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}

package archtest

import (
	"os"
	"strings"
	"testing"
)

func TestCheckRandomnessNamesEachViolation(t *testing.T) {
	cases := []struct {
		name, src, want string
	}{
		{"math/rand", "package p\n\nimport \"math/rand\"\n\nvar _ = rand.Intn\n", "imports a generator"},
		{"math/rand/v2", "package p\n\nimport \"math/rand/v2\"\n\nvar _ = rand.IntN\n", "imports a generator"},
		{"multiplier", "package p\n\nconst m = 214013\n", "constant 214013"},
		{"hex multiplier", "package p\n\nconst m = 0x343fd\n", "constant 0x343fd"},
		{"increment", "package p\n\nconst c = 0x269EC3\n", "constant 0x269EC3"},
		{"clock seeded stream", "package p\n\nimport \"time\"\n\nvar _ = random.NewGo(time.Now().UnixNano())\n", "seeds NewGo from the clock"},
		{"clock seeded session", "package p\n\nfunc f(now Clock) { s.Begin(random.Session{Seed: uint64(now.UnixMilli())}) }\n", "seeds Begin from the clock"},
		{"seed function reads clock", "package p\n\nimport \"time\"\n\nfunc mySeed() int64 { return time.Now().Unix() }\n", "seed function mySeed"},
		{"crypto/rand", "package p\n\nimport \"crypto/rand\"\n\nvar _ = rand.Read\n", "imports a generator"},
		{"hash/maphash", "package p\n\nimport \"hash/maphash\"\n\nvar _ = maphash.MakeSeed\n", "imports a generator"},
		// The two forms the first ratchet passed: a clock value through a
		// variable, and one in a pkg/random composite literal.
		{"clock through a variable", "package p\n\nfunc f() { s := time.Now().UnixNano(); random.NewStream(s) }\n", "seeds NewStream from the clock"},
		{"clock in an MSVC literal", "package p\n\nvar g = random.MSVC{State: uint32(time.Now().UnixNano())}\n", "seeds random.MSVC from the clock"},
		{"clock through two variables", "package p\n\nfunc f() { now := time.Now(); s := uint64(now.Unix()) + 1; _ = &random.Session{Seed: s} }\n", "seeds random.Session from the clock"},
		{"clock through a package variable", "package p\n\nvar start = time.Now()\n\nfunc f(svc *random.Service) { svc.Begin(random.Session{Seed: uint64(start.Unix())}) }\n", "from the clock"},
		{"clock into a State field", "package p\n\nfunc f(m *random.MSVC) { t := time.Now(); m.State = uint32(t.Unix()) }\n", "into a generator's State"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			vs := CheckRandomness(map[string]string{"pkg/game/x.go": tc.src})
			if len(vs) != 1 || !strings.Contains(vs[0].String(), tc.want) {
				t.Fatalf("violations %v, want one naming %q", vs, tc.want)
			}
		})
	}
}

func TestCheckRandomnessPassesProseAndOtherNumbers(t *testing.T) {
	src := "package p\n\n// math/rand and 214013 in prose.\nvar s = \"214013\"\nvar n = 214014\n\nfunc tick() int64 { return time.Now().UnixNano() }\n"
	if vs := CheckRandomness(map[string]string{"pkg/game/x.go": src}); len(vs) != 0 {
		t.Fatalf("violations %v, want none", vs)
	}
	if vs := CheckRandomness(nil); len(vs) != 1 {
		t.Fatal("an empty scan passed")
	}
	// A clock value that reaches no seed, and a seed from a value that is
	// not the clock's, pass.
	src = "package p\n\nfunc f(n uint64) { start := time.Now(); _ = start; s := n + 1; random.NewStream(int64(s)); _ = random.MSVC{State: uint32(s)} }\n"
	if vs := CheckRandomness(map[string]string{"pkg/game/x.go": src}); len(vs) != 0 {
		t.Fatalf("violations %v, want none", vs)
	}
}

// TestRandomnessAllowListIsPinned holds the allow list to its entries and
// proves an allowance admits only its own rule in its own file.
func TestRandomnessAllowListIsPinned(t *testing.T) {
	want := []string{"pkg/random/gosource.go math/rand", "pkg/random/msvc.go constant", "pkg/ui/panel.go hash/maphash"}
	if len(RandomnessAllowed) != len(want) {
		t.Fatalf("allow list %v changed", RandomnessAllowed)
	}
	for i, a := range RandomnessAllowed {
		if a.File+" "+a.Rule != want[i] || a.Reason == "" {
			t.Fatalf("allowance %d is %+v, want %q with a reason", i, a, want[i])
		}
	}
	src := "package ui\n\nimport \"hash/maphash\"\n\nvar _ = maphash.MakeSeed\n"
	if vs := CheckRandomness(map[string]string{"pkg/ui/panel.go": src}); len(vs) != 0 {
		t.Fatalf("the allowed memo key failed: %v", vs)
	}
	if vs := CheckRandomness(map[string]string{"pkg/ui/other.go": src}); len(vs) != 1 {
		t.Fatalf("the allowance reached another file: %v", vs)
	}
	src = "package ui\n\nimport \"math/rand\"\n\nvar _ = rand.Intn\n"
	if vs := CheckRandomness(map[string]string{"pkg/ui/panel.go": src}); len(vs) != 1 {
		t.Fatalf("the allowance admitted another rule: %v", vs)
	}
}

func TestSanctionedClockSeedIsPinned(t *testing.T) {
	if len(SanctionedClockSeeds) != 1 || SanctionedClockSeeds["pkg/game/randomsession.go"] != "clockSessionSeed" {
		t.Fatalf("sanctioned clock seeds %v changed", SanctionedClockSeeds)
	}
	src := "package game\n\nimport \"time\"\n\nfunc clockSessionSeed() uint64 { return uint64(time.Now().UnixNano()) }\n"
	if vs := CheckRandomness(map[string]string{"pkg/game/other.go": src}); len(vs) != 1 {
		t.Fatalf("the sanctioned name passed in another file: %v", vs)
	}
}

// TestLiveRandomnessMatchesItsDebt holds the covered packages to the debt
// list: no file above its count, and a fall lowers the count.
func TestLiveRandomnessMatchesItsDebt(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	root, err := FindModuleRoot(wd)
	if err != nil {
		t.Fatal(err)
	}
	files, err := LoadRandomnessSources(root)
	if err != nil {
		t.Fatal(err)
	}
	// Every package the first ratchet held, and the ones it missed, is read.
	for _, pkg := range []string{"pkg/sim", "pkg/mapload", "pkg/game", "pkg/ui", "pkg/town", "pkg/random",
		"pkg/render", "pkg/mod", "pkg/modrt", "pkg/audio", "pkg/video", "cmd/againrom"} {
		seen := false
		for name := range files {
			seen = seen || strings.HasPrefix(name, pkg+"/")
		}
		if !seen {
			t.Errorf("no production source read from %s", pkg)
		}
	}
	for name := range files {
		if strings.HasSuffix(name, "_test.go") || strings.Contains(name, "/testdata/") {
			t.Errorf("a non-production file was read: %s", name)
		}
	}
	got := map[string]int{}
	for _, v := range CheckRandomness(files) {
		file, _, _ := strings.Cut(v.From, ":")
		got[file]++
		if got[file] > RandomnessDebt[file] {
			t.Errorf("randomness outside pkg/random: %s", v)
		}
	}
	for file, n := range RandomnessDebt {
		if got[file] < n {
			t.Errorf("%s fell from %d to %d: lower RandomnessDebt", file, n, got[file])
		}
	}
	if _, ok := files["pkg/game/randomsession.go"]; !ok {
		t.Error("the sanctioned clock seed's file is gone; update SanctionedClockSeeds")
	}
}

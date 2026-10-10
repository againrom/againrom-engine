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
	for _, pkg := range RandomnessPackages {
		seen := false
		for name := range files {
			seen = seen || strings.HasPrefix(name, pkg+"/")
		}
		if !seen {
			t.Errorf("no production source read from %s", pkg)
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

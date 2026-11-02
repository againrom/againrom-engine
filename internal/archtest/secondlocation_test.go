package archtest

import (
	"os"
	"strings"
	"testing"
)

// TestCheckSecondLocationNamesEachCopy plants each shape a second location
// entry or a second departure declaration takes.
func TestCheckSecondLocationNamesEachCopy(t *testing.T) {
	cases := []struct {
		name, file, src, want string
	}{
		{"a second clear", "pkg/game/a.go",
			"package game\nfunc f(b *[1024]int32) { clear(b[752:768]) }\n", "enterLocation"},
		{"a second clear on a field", "pkg/game/a.go",
			"package game\nfunc (t T) f() { clear(t.c.bank[752:768]) }\n", "enterLocation"},
		{"the entry itself", "pkg/game/seconddeparture.go",
			"package game\nfunc enterLocation(b *[1024]int32) { clear(b[752:768]) }\n", ""},
		{"a case switch", "pkg/game/secondcampaign.go",
			"package game\nfunc (c *C) f(n int) { switch n {\n case 20:\n g()\n } }\n", "secondDepartures"},
		{"a write of a departure", "pkg/game/secondcampaign.go",
			"package game\nfunc (c *C) f() { c.bank[771] = 1 }\n", "writes"},
		{"a head write", "pkg/game/secondcampaign.go",
			"package game\nfunc (c *C) f(i int) { c.bank[768] += 10\n c.bank[532+i] = 1 }\n", ""},
		{"a case switch elsewhere", "pkg/game/autohealing.go",
			"package game\nfunc f(n int) { switch n {\n case 100:\n g()\n } }\n", ""},
		{"a census table", "pkg/game/secondcensus.go",
			"package game\nvar secondDepartureCases = map[int][]SecondExit{10: {{Target: \"x\"}}}\n", "secondDepartures"},
		{"a row table elsewhere", "pkg/game/b.go",
			"package game\nvar rows = []secondDepartureRow{{from: 10}}\n", "secondDepartures"},
		{"the rows", "pkg/game/seconddeparture.go",
			"package game\nvar secondDepartures = []secondDepartureRow{{from: 10}}\n", ""},
		{"another package", "pkg/ui/a.go",
			"package ui\nfunc f(b *[1024]int32) { clear(b[752:768]) }\n", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			vs := CheckSecondLocation(map[string]string{tc.file: tc.src})
			if tc.want == "" {
				if len(vs) != 0 {
					t.Fatalf("CheckSecondLocation = %v, want none", vs)
				}
				return
			}
			if len(vs) < 1 || !strings.Contains(vs[0].Reason, tc.want) || !strings.HasPrefix(vs[0].From, tc.file+":") {
				t.Fatalf("CheckSecondLocation = %v, want one naming %q", vs, tc.want)
			}
		})
	}
	if vs := CheckSecondLocation(nil); len(vs) != 1 {
		t.Fatalf("CheckSecondLocation(nil) = %v, want one violation", vs)
	}
}

// TestSecondLocationHasOneBuilder is the live-tree half.
func TestSecondLocationHasOneBuilder(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	root, err := FindModuleRoot(wd)
	if err != nil {
		t.Fatal(err)
	}
	files, err := LoadDrawnTextSources(root)
	if err != nil {
		t.Fatal(err)
	}
	for _, v := range CheckSecondLocation(files) {
		t.Errorf("%s", v)
	}
}

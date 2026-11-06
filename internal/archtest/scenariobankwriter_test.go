package archtest

import (
	"os"
	"strings"
	"testing"
)

// TestCheckScenarioBankWriterNamesEachStore drives the scan with planted
// copies of each store shape, and with the shapes it admits.
func TestCheckScenarioBankWriterNamesEachStore(t *testing.T) {
	cases := []struct {
		name, file, src string
		want            int
	}{
		{"a plain store in pkg/game", "pkg/game/a.go", "package game\nfunc f(c *camp) { c.bank[1] = 2 }\n", 1},
		{"a compound store", "pkg/game/a.go", "package game\nfunc f(c *camp) { c.bank[768] += 10 }\n", 1},
		{"a tuple store", "pkg/game/a.go", "package game\nfunc f(bank *[1024]int32) { bank[1], bank[2] = 1, 2 }\n", 2},
		{"an increment", "pkg/game/a.go", "package game\nfunc f(bank [4]int32) { bank[1]++ }\n", 1},
		{"a slot address", "pkg/game/a.go", "package game\nfunc f(c *camp) { s := &c.bank[3]; *s = 1 }\n", 1},
		{"a read", "pkg/game/a.go", "package game\nfunc f(c *camp) int32 { x := c.bank[3]; return x + c.bank[4] }\n", 0},
		{"the writer file", scenarioBankWriter, "package sim\nfunc f(bank *[4]int32) { bank[1] = 2 }\n", 0},
		{"a test file", "pkg/game/a_test.go", "package game\nfunc f(bank *[4]int32) { bank[1] = 2 }\n", 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := CheckScenarioBankWriter(map[string]string{tc.file: tc.src, "pkg/sim/other.go": "package sim\n"})
			if len(got) != tc.want {
				t.Fatalf("CheckScenarioBankWriter = %v, want %d", got, tc.want)
			}
			for _, v := range got {
				if !strings.Contains(v.Reason, scenarioBankWriter) {
					t.Fatalf("violation %v does not name the writer", v)
				}
			}
		})
	}
	if vs := CheckScenarioBankWriter(nil); len(vs) != 1 {
		t.Fatalf("CheckScenarioBankWriter(nil) = %v, want one violation", vs)
	}
}

// TestScenarioBankHasOneWriter is the live-tree half.
func TestScenarioBankHasOneWriter(t *testing.T) {
	root := moduleRootForTest(t)
	files, err := LoadDrawnTextSources(root)
	if err != nil {
		t.Fatal(err)
	}
	for _, v := range CheckScenarioBankWriter(files) {
		t.Errorf("%s", v)
	}
}

func moduleRootForTest(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	root, err := FindModuleRoot(wd)
	if err != nil {
		t.Fatal(err)
	}
	return root
}

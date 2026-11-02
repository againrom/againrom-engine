package archtest

import (
	"os"
	"strings"
	"testing"
)

// TestCheckRuntimeIDAllocatorNamesEachSecondAllocator plants each shape a
// second runtime-id allocator takes.
func TestCheckRuntimeIDAllocatorNamesEachSecondAllocator(t *testing.T) {
	cases := []struct {
		name, file, src, want string
	}{
		{"a next function", "pkg/game/a.go",
			"package game\nfunc nextSavedRuntimeID(used map[uint32]bool) uint32 { return 1 }\n", "second runtime-id allocator"},
		{"a method named lowestClear", "pkg/sim/b.go",
			"package sim\nfunc (m *M) lowestClear() uint32 { return 1 }\n", "second runtime-id allocator"},
		{"a map walk", "pkg/game/a.go",
			"package game\nfunc mintRuntimeID(used map[uint32]bool) uint32 { id := uint32(1)\n for used[id] { id++ }\n return id }\n", "first free runtime id"},
		{"a negated map walk", "pkg/game/a.go",
			"package game\nfunc f(token T, used []bool) uint32 { id := token.RuntimeID\n for !used[id] && id != token.RuntimeID { id-- }\n return id }\n", "first free runtime id"},
		{"a counting loop with an if", "pkg/ui/a.go",
			"package ui\nfunc f(used map[uint32]bool) uint32 { var runtimeID uint32\n for { if used[runtimeID] { runtimeID++\n continue }\n return runtimeID } }\n", "first free runtime id"},
		{"the builder file", "pkg/sim/runtimeids.go",
			"package sim\nfunc (m *M) lowestClear() uint32 { return 1 }\n", ""},
		{"a test file", "pkg/game/a_test.go",
			"package game\nfunc nextSavedRuntimeID() uint32 { return 1 }\n", ""},
		{"a loop in a function with no runtime id", "pkg/game/a.go",
			"package game\nfunc freeGroup(used map[int64]bool) int64 { n := int64(1)\n for used[n] { n++ }\n return n }\n", ""},
		{"a function that mints from the set", "pkg/game/a.go",
			"package game\nfunc mintCityRuntimeID(set *sim.RuntimeIDSet) (uint32, error) { return set.LowestClear(), nil }\n", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			vs := CheckRuntimeIDAllocator(map[string]string{tc.file: tc.src})
			if tc.want == "" {
				if len(vs) != 0 {
					t.Fatalf("CheckRuntimeIDAllocator = %v, want none", vs)
				}
				return
			}
			if len(vs) == 0 || !strings.Contains(vs[0].Reason, tc.want) || !strings.HasPrefix(vs[0].From, tc.file+":") {
				t.Fatalf("CheckRuntimeIDAllocator = %v, want one naming %q", vs, tc.want)
			}
		})
	}
	if vs := CheckRuntimeIDAllocator(nil); len(vs) != 1 {
		t.Fatalf("CheckRuntimeIDAllocator(nil) = %v, want one violation", vs)
	}
}

// TestRuntimeIDsHaveOneAllocator is the live-tree half.
func TestRuntimeIDsHaveOneAllocator(t *testing.T) {
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
	for _, v := range CheckRuntimeIDAllocator(files) {
		t.Errorf("%s", v)
	}
}

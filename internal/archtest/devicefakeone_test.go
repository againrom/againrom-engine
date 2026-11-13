package archtest

import (
	"os"
	"strings"
	"testing"
)

func TestCheckDeviceFakeNamesEachViolation(t *testing.T) {
	cases := []struct {
		name, file, src, want string
	}{
		{"a second fake in a test", "pkg/game/a_test.go",
			"package game\ntype fake struct{}\nfunc (p *fake) IsPlaying() bool { return false }\nfunc (p *fake) Seek(int) error { return nil }\n",
			"fake is a second delivery device fake"},
		{"a wrapper with one of the methods", "pkg/game/a_test.go",
			"package game\ntype w struct{}\nfunc (p *w) Pause() {}\nfunc (p *w) Seek(int) error { return nil }\n", ""},
		{"the production device player", "pkg/ui/a.go",
			"package ui\ntype d struct{}\nfunc (p *d) IsPlaying() bool { return false }\nfunc (p *d) Seek(int) error { return nil }\n", ""},
		{"production importing the fake", "pkg/ui/a.go",
			"package ui\nimport \"againrom/internal/devicefake\"\nvar _ = devicefake.Player{}\n", "imports internal/devicefake"},
		{"a test importing the fake", "pkg/ui/a_test.go",
			"package ui\nimport \"againrom/internal/devicefake\"\nvar _ = devicefake.Player{}\n", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := CheckDeviceFake(map[string]string{tc.file: tc.src})
			if tc.want == "" {
				if len(got) != 0 {
					t.Fatalf("CheckDeviceFake = %v, want none", got)
				}
				return
			}
			if len(got) != 1 || !strings.Contains(got[0].Reason, tc.want) {
				t.Fatalf("CheckDeviceFake = %v, want one naming %q", got, tc.want)
			}
		})
	}
	if vs := CheckDeviceFake(nil); len(vs) != 1 {
		t.Fatalf("CheckDeviceFake(nil) = %v, want one violation", vs)
	}
}

// TestDeliveryDeviceFakeIsOne is the live-tree half.
func TestDeliveryDeviceFakeIsOne(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	root, err := FindModuleRoot(wd)
	if err != nil {
		t.Fatal(err)
	}
	files, err := LoadDeviceFakeSources(root)
	if err != nil {
		t.Fatal(err)
	}
	for _, v := range CheckDeviceFake(files) {
		t.Errorf("%s", v)
	}
}

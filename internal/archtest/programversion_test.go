package archtest

import (
	"os"
	"testing"
)

func TestCheckProgramVersionNamesTheFunc(t *testing.T) {
	for _, name := range []string{"cmd/tool/version.go", "pkg/editorapp/version.go"} {
		bad := map[string]string{name: "package main\nfunc programVersion() string { return \"\" }\n"}
		if vs := CheckProgramVersion(bad); len(vs) != 1 {
			t.Fatalf("CheckProgramVersion(%s) = %v, want one violation", name, vs)
		}
	}
	ok := map[string]string{
		"cmd/tool/version.go": "package main\nvar program = 1\nfunc versionLine() string { return \"\" }\n",
		"pkg/x/x.go":          "package x\nfunc programVersion() string { return \"\" }\n",
		"cmd/tool/m.go":       "package main\ntype T struct{}\nfunc (T) programVersion() string { return \"\" }\n",
	}
	if vs := CheckProgramVersion(ok); len(vs) != 0 {
		t.Fatalf("CheckProgramVersion = %v, want none", vs)
	}
	if vs := CheckProgramVersion(nil); len(vs) != 1 {
		t.Fatalf("CheckProgramVersion(nil) = %v, want one violation", vs)
	}
}

func TestEachProgramAsksBuildinfoForItsVersion(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	root, err := FindModuleRoot(wd)
	if err != nil {
		t.Fatal(err)
	}
	files, err := LoadProgramSources(root)
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"cmd/againrom/version.go", "pkg/editorapp/version.go"} {
		if _, ok := files[name]; !ok {
			t.Fatalf("%s is missing; the scan guards nothing there", name)
		}
	}
	for _, v := range CheckProgramVersion(files) {
		t.Errorf("%s", v)
	}
}

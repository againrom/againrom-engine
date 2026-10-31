package archtest

import (
	"os"
	"testing"
)

func TestCheckProgramVersionNamesTheFunc(t *testing.T) {
	bad := map[string]string{"cmd/tool/version.go": "package main\nfunc programVersion() string { return \"\" }\n"}
	if vs := CheckProgramVersion(bad); len(vs) != 1 {
		t.Fatalf("CheckProgramVersion = %v, want one violation", vs)
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
	files, err := LoadCmdSources(root)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := files["cmd/againrom/version.go"]; !ok {
		t.Fatal("cmd/againrom/version.go is missing; the scan guards nothing")
	}
	for _, v := range CheckProgramVersion(files) {
		t.Errorf("%s", v)
	}
}

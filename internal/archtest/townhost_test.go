package archtest

import (
	"os"
	"strings"
	"testing"
)

func TestOneTownHostOwnsItsSeams(t *testing.T) {
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
	for _, v := range CheckTownHost(files) {
		t.Error(v)
	}
}

func TestTownHostGuardFindsAnotherReceiver(t *testing.T) {
	source := "package game\ntype townScreen struct{}\ntype roomPageView struct{}\n"
	for _, seam := range append(append([]string{}, townHostSeams...), townOwnedSeams...) {
		source += "func (*townScreen) " + seam + "() {}\n"
	}
	for _, seam := range []string{"Size", "Paint", "ControlAt", "TipAt"} {
		source += "func (roomPageView) " + seam + "() {}\n"
	}
	files := map[string]string{"pkg/game/town.go": source}
	if vs := CheckTownHost(files); len(vs) != 0 {
		t.Fatalf("one host: %v", vs)
	}
	extra := "package game\ntype otherTown struct{}\n"
	for _, seam := range townHostSeams {
		extra += "func (*otherTown) " + seam + "() {}\n"
	}
	files["pkg/game/other.go"] = extra
	found := false
	for _, v := range CheckTownHost(files) {
		found = found || strings.Contains(v.From, "otherTown")
	}
	if !found {
		t.Fatal("second host was not found")
	}
	extra = "package game\ntype otherView struct{}\n"
	for _, seam := range []string{"Size", "Paint", "ControlAt", "TipAt"} {
		extra += "func (otherView) " + seam + "() {}\n"
	}
	files["pkg/game/other.go"] = extra
	vs := CheckTownHost(files)
	if len(vs) != 1 || vs[0].From != "pkg/game:otherView" || vs[0].Reason != "town square has a second scene view" {
		t.Fatalf("second scene view: %v", vs)
	}
	files["pkg/game/other.go"] = "package game\ntype embeddedTown struct { *townScreen }\n"
	found = false
	for _, v := range CheckTownHost(files) {
		found = found || strings.Contains(v.From, "embeddedTown")
	}
	if !found {
		t.Fatal("promoted town host methods were not found")
	}
}

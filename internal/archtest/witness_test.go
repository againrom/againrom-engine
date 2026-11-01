package archtest

import (
	"os"
	"strings"
	"testing"
)

// A test writes an image through witness.PNG and reads one through
// witness.Decode, and through nothing else: no test file outside the witness
// package imports an image codec or opens an image file itself.
func TestTestImagesAreWrittenByTheWitnessHelper(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	root, err := FindModuleRoot(wd)
	if err != nil {
		t.Fatalf("module root not found from %s: %v", wd, err)
	}
	holds, err := WitnessHomeHolds(root)
	if err != nil {
		t.Fatal(err)
	}
	if !holds {
		t.Fatalf("%s no longer holds PNG", witnessHome)
	}
	sites, err := WitnessSites(root)
	if err != nil {
		t.Fatalf("measure: %v", err)
	}
	for _, s := range sites {
		t.Errorf("%s; hand the image to witness.PNG, witness.Bytes or witness.Decode", s)
	}
}

func witnessHits(src string) string {
	return strings.Join(witnessInSource("x_test.go", []byte(src)), ", ")
}

// One planted instance per shape: each is named, and a look-alike is not.
func TestWitnessScanSeesEachShape(t *testing.T) {
	for _, tc := range []struct{ name, src, want string }{
		{"a png import", `package x
import (
	"testing"
	"image/png"
)
var _ = png.Encode`, `x_test.go:4 imports image/png`},
		{"a png import under an alias", `package x
import p "image/png"
var _ = p.Decode`, `x_test.go:2 imports image/png`},
		{"a jpeg import", `package x
import _ "image/jpeg"`, `x_test.go:2 imports image/jpeg`},
		{"a gif import", `package x
import "image/gif"
var _ = gif.Encode`, `x_test.go:2 imports image/gif`},
		{"a bmp import", `package x
import "golang.org/x/image/bmp"
var _ = bmp.Encode`, `x_test.go:2 imports golang.org/x/image/bmp`},
		{"a tiff import", `package x
import "golang.org/x/image/tiff"
var _ = tiff.Encode`, `x_test.go:2 imports golang.org/x/image/tiff`},
		{"a file created for a picture", `package x
import "os"
func f(dir string) { os.Create(dir + "/shot.png") }`, `x_test.go:3 os.Create opens "/shot.png"`},
		{"bytes written to a joined picture name", `package x
import ("os"; "path/filepath")
func f(dir string, b []byte) { os.WriteFile(filepath.Join(dir, "frame.PNG"), b, 0o644) }`, `x_test.go:3 os.WriteFile opens "frame.PNG"`},
		{"a picture name built by Sprintf", `package x
import ("fmt"; "os")
func f(n int) { os.OpenFile(fmt.Sprintf("s%d.jpg", n), os.O_WRONLY, 0) }`, `x_test.go:3 os.OpenFile opens "s%d.jpg"`},
		{"a gif name", `package x
import "os"
func f() { os.Create("a.gif") }`, `x_test.go:3 os.Create opens "a.gif"`},
		{"the helper is not a shape", `package x
import "againrom/internal/witness"
func f(t T, dir string, img I) { witness.PNG(t, dir+"/a.png", img) }`, ``},
		{"a name in a composite literal is not a file", `package x
import "os"
func f(p string) { os.WriteFile(p, mk([]F{{Path: "a.png"}}), 0o644) }`, ``},
		{"a codec name in a string is not an import", `package x
var mime = "image/png"`, ``},
		{"a text file is not a picture", `package x
import "os"
func f() { os.Create("notes.txt") }`, ``},
	} {
		if got := witnessHits(tc.src); got != tc.want {
			t.Errorf("%s: got %q, want %q", tc.name, got, tc.want)
		}
	}
}

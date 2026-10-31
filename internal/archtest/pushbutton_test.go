package archtest

import (
	"os"
	"strings"
	"testing"
)

// fillAndEdges is a fill and four two-pixel edge fills, each written by
// call from its rectangle.
func fillAndEdges(call func(rect string) string) string {
	var b strings.Builder
	for _, r := range []string{"x, y, x+w, y+h", "x, y, x+w, y+2", "x, y+h-2, x+w, y+h", "x, y, x+2, y+h", "x+w-2, y, x+w, y+h"} {
		b.WriteString("\t" + call(r) + "\n")
	}
	return b.String()
}

func boxCall(rect string) string { return "box(image.Rect(" + rect + "), c)" }

func drawCall(rect string) string {
	return "draw.Draw(pix, image.Rect(" + rect + "), &image.Uniform{C: c}, image.Point{}, draw.Src)"
}

// TestCheckPushButtonNamesEachShape plants each shape of a second button
// painter and shows the scan names it.
func TestCheckPushButtonNamesEachShape(t *testing.T) {
	canvas := "package ui\nimport \"github.com/hajimehoshi/ebiten/v2/ebitenutil\"\nfunc drawRow(dst *ebiten.Image, x, y, w, h float64) {\n" +
		"\tebitenutil.DrawRect(dst, x, y, w, h, fill)\n}\n"
	cases := []struct {
		name, file, src string
		want            string // a substring of the one violation; "" for none
	}{
		{"a canvas rectangle", "pkg/ui/a.go", canvas, "ebitenutil.DrawRect paints a rectangle"},
		{"a canvas rectangle in the kit file", "pkg/ui/widgetbutton.go", canvas, "ebitenutil.DrawRect paints a rectangle"},
		{"a canvas rectangle under an import alias", "pkg/ui/a.go",
			"package ui\nimport eu \"github.com/hajimehoshi/ebiten/v2/ebitenutil\"\nfunc f() { eu.DrawRect(dst, 0, 0, 1, 1, c) }\n",
			"ebitenutil.DrawRect paints a rectangle"},
		{"a fill and four edges through box in a closure", "pkg/ui/a.go",
			"package ui\nfunc compose() {\n\tbox := func(r image.Rectangle, c color.Color) {}\n\tpaint(func(x, y, w, h int) {\n" +
				fillAndEdges(boxCall) + "\t})\n}\n",
			"compose fills a rectangle and draws a two-pixel border"},
		{"a fill and four edges through draw.Draw with a uniform source", "pkg/ui/a.go",
			"package ui\nfunc paintRow(pix *image.RGBA, x, y, w, h int) {\n" +
				fillAndEdges(drawCall) + "}\n",
			"paintRow fills a rectangle and draws a two-pixel border"},
		{"a fill and four edges through vector.DrawFilledRect", "pkg/ui/a.go",
			"package ui\nimport \"github.com/hajimehoshi/ebiten/v2/vector\"\nfunc paintRow(dst *ebiten.Image, x, y, w, h float32) {\n" +
				"\tvector.DrawFilledRect(dst, x, y, w, h, fill, false)\n\tvector.DrawFilledRect(dst, x, y, w, 2, edge, false)\n" +
				"\tvector.DrawFilledRect(dst, x, y+h-2, w, 2, edge, false)\n\tvector.DrawFilledRect(dst, x, y, 2, h, edge, false)\n" +
				"\tvector.DrawFilledRect(dst, x+w-2, y, 2, h, edge, false)\n}\n",
			"paintRow fills a rectangle and draws a two-pixel border"},
		{"the same body inside the kit file", "pkg/ui/widgetbutton.go",
			"package ui\nfunc drawPushButton() {\n" + fillAndEdges(boxCall) + "}\n", ""},
		{"a fill with one outline", "pkg/ui/a.go",
			"package ui\nfunc paintBox() {\n\tdraw.Draw(dst, r, &image.Uniform{C: c}, image.Point{}, draw.Src)\n\toutline(dst, r, edge)\n}\n", ""},
		{"several fills without a two-pixel edge", "pkg/ui/a.go",
			"package ui\nfunc paintGrid() {\n\tbox(image.Rect(0, 0, 9, 9), a)\n\tbox(image.Rect(0, 0, 9, 3), b)\n\tbox(image.Rect(0, 3, 9, 6), b)\n\tbox(image.Rect(0, 6, 9, 9), b)\n\tbox(image.Rect(0, 0, 1, 9), b)\n}\n", ""},
		{"a copy of a picture is no fill", "pkg/ui/a.go",
			"package ui\nfunc paintPic() {\n\tfor i := 0; i < 5; i++ {\n\t\tdraw.Draw(dst, image.Rect(0, 0, 2, 2), pic, image.Point{}, draw.Src)\n\t}\n}\n", ""},
		{"another package", "pkg/render/a.go",
			"package render\nfunc f() { ebitenutil.DrawRect(dst, 0, 0, 1, 1, c) }\n", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			vs := CheckPushButton(map[string]string{tc.file: tc.src})
			if tc.want == "" {
				if len(vs) != 0 {
					t.Fatalf("CheckPushButton = %v, want none", vs)
				}
				return
			}
			if len(vs) != 1 || !strings.Contains(vs[0].Reason, tc.want) {
				t.Fatalf("CheckPushButton = %v, want one naming %q", vs, tc.want)
			}
		})
	}
	if vs := CheckPushButton(nil); len(vs) != 1 {
		t.Fatalf("CheckPushButton(nil) = %v, want one violation", vs)
	}
}

// TestPushButtonsHaveOneBuilder is the live-tree half.
func TestPushButtonsHaveOneBuilder(t *testing.T) {
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
	for _, v := range CheckPushButton(files) {
		t.Errorf("%s", v)
	}
}

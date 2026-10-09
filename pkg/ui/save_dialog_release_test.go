package ui

import (
	"fmt"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"againrom/pkg/formats/res"
	"againrom/pkg/formats/spr16"
	"againrom/pkg/formats/textinput"
	"againrom/pkg/render/text"
)

func saveDialogInstallEntry(t *testing.T, archive *res.FileArchive, suffix string) []byte {
	t.Helper()
	var names []string
	for _, entry := range archive.Entries() {
		name := strings.ToLower(strings.ReplaceAll(entry.Path, "\\", "/"))
		if name == suffix || strings.HasSuffix(name, "/"+suffix) {
			names = append(names, entry.Path)
		}
	}
	if len(names) != 1 {
		t.Fatalf("install entry %s: %d matches", suffix, len(names))
	}
	b, err := archive.ReadFile(names[0])
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func saveDialogInstallFont(t *testing.T, root string) *text.Font {
	t.Helper()
	graphics, err := res.OpenFileIndex(filepath.Join(root, "graphics.res"))
	if err != nil {
		t.Fatal(err)
	}
	frames, err := spr16.DecodeG(saveDialogInstallEntry(t, graphics, "font1/font1.16"))
	if err != nil {
		t.Fatal(err)
	}
	advances, err := spr16.Advances(saveDialogInstallEntry(t, graphics, "font1/font1.dat"))
	if err != nil {
		t.Fatal(err)
	}
	if len(frames) != len(advances) || len(frames) == 0 {
		t.Fatal("font atlas and advance table disagree")
	}
	font := &text.Font{Spacing: 2, Glyphs: make([]text.Glyph, len(frames))}
	for i, f := range frames {
		pixels := make([]text.Pixel, len(f.Pixels))
		for j, p := range f.Pixels {
			pixels[j] = text.Pixel{Level: p.Value, Painted: p.Painted}
		}
		font.Glyphs[i] = text.Glyph{Width: f.Width, Height: f.Height, Pixels: pixels, Advance: advances[i]}
	}
	main, err := res.OpenFileIndex(filepath.Join(root, "main.res"))
	if err != nil {
		t.Fatal(err)
	}
	id := saveDialogInstallEntry(t, main, "id")
	if len(id) == 0 {
		t.Fatal("empty install language")
	}
	font.Selector = int(id[len(id)-1] - '0')
	return font
}

// This witness reads the installed font1 and its language selector, then draws
// the production save composer. Optional captures have an explicit output path.
func TestReleaseSaveDialogUsesInstalledFontWithoutClipping(t *testing.T) {
	root := os.Getenv("AGAINROM_ASSETS")
	if root == "" {
		t.Skip("AGAINROM_ASSETS not set")
	}
	font := saveDialogInstallFont(t, root)
	t.Run("load-unicode", func(t *testing.T) {
		loaded := ""
		a := unicodeLoadApp(t, font, &loaded)
		pix, note, err := a.HeadlessFrame()
		if err != nil || note != "" {
			t.Fatalf("Unicode LOAD frame: %q / %v", note, err)
		}
		assertUnicodeLoadRow(t, a, font, pix)
		writeSaveDialogCapture(t, "load-unicode", pix)
	})
	for _, tc := range []struct {
		name    string
		back    Screen
		format  SaveFormat
		confirm bool
		remove  bool
	}{
		{"town-sav", ScreenTown, SaveSAV, false, false},
		{"mission-sav", ScreenMap, SaveSAV, false, false},
		{"replace-town-sav", ScreenTown, SaveSAV, true, false},
		{"replace-mission-sav", ScreenMap, SaveSAV, true, false},
		{"delete-selected", ScreenTown, SaveSAV, false, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			directory := `C:\Games\Rage of Mages\Saves`
			s := &saveDialogSpy{directories: map[string]SaveDirectory{directory: {
				Path: directory, Directories: []string{"Campaign", "Earlier saves"},
				Entries: []SaveEntry{{Name: "Before the assault.ags", Label: "Hagen - town"}, {Name: "Camp.sav"}, {Name: "Journey.ags"}, {Name: "Second journey.ags"}, {Name: "The old tower.sav"}, {Name: "Unfinished mission.ags"}},
			}}, paths: []string{filepath.Join(directory, "Journey.sav")}}
			if tc.confirm {
				s.existing = append([]string(nil), s.paths...)
			}
			a := newSaveDialogApp(t, s, tc.back)
			a.SetWords(AuthoredWords(), font, func(r rune) (byte, bool) { return textinput.EncodeRune(r, font.Selector) })
			if err := a.HeadlessSaveEdit(directory, "Journey", tc.format); err != nil {
				t.Fatal(err)
			}
			if tc.confirm {
				mustSaveAction(t, a, "save")
			}
			if tc.remove {
				a.flow.saveDialogSeams.CanDelete = func(string, string) bool { return true }
				a.flow.saveDialogSeams.PrepareDelete = func(string, string) (func() error, error) {
					return func() error { return nil }, nil
				}
				if err := a.HeadlessSaveSelect(3); err != nil {
					t.Fatal(err)
				}
				mustSaveAction(t, a, "delete")
			}
			paint, err := a.saveDialogPaint()
			if err != nil {
				t.Fatal(err)
			}
			for _, label := range paint.texts {
				width, height := font.Measure(a.flow.menuDisplayText(label.text))
				bounds := image.Rect(label.at.X, label.at.Y, label.at.X+width, label.at.Y+height)
				if !bounds.In(savePanelRect) {
					t.Errorf("text %q leaves panel: %v", label.text, bounds)
				}
			}
			w := a.flow.saveWords()
			if !tc.confirm && !tc.remove {
				for c, label := range map[saveControl]string{saveOpenControl: w.Open, saveUpControl: w.Up, saveWriteControl: w.Save, saveCancelControl: w.Cancel, saveDeleteControl: w.Delete} {
					if a.saveTextWidth(label) > saveControlRect(c).Dx()-10 {
						t.Errorf("button %s clips %q", saveControlName(c), label)
					}
				}
				if tc.back == ScreenMap && tc.format == SaveSAV && len(a.saveWrapped(w.MapSAVDetail, 560)) > 3 {
					t.Error("mission SAV description loses a line")
				}
				if tc.format == SaveSAV {
					if w.LatinNameHint == "" || !strings.Contains(w.KeyHint, w.LatinNameHint) || len(a.saveWrapped(w.KeyHint, 560)) > 2 {
						t.Error("font character hint is absent or clipped")
					}
				}
			}
			pix, note, err := a.HeadlessFrame()
			if err != nil || note != "" {
				t.Fatalf("save frame: %q / %v", note, err)
			}
			writeSaveDialogCapture(t, tc.name, pix)
		})
	}
}

func writeSaveDialogCapture(t *testing.T, name string, pix *image.RGBA) {
	t.Helper()
	if output := os.Getenv("AGAINROM_SAVE_DIALOG_RENDER_OUTPUT"); output != "" {
		if err := os.MkdirAll(output, 0755); err != nil {
			t.Fatal(err)
		}
		path := filepath.Join(output, fmt.Sprintf("%s.png", name))
		f, err := os.Create(path)
		if err != nil {
			t.Fatal(err)
		}
		if err := png.Encode(f, pix); err != nil {
			_ = f.Close()
			t.Fatal(err)
		}
		if err := f.Close(); err != nil {
			t.Fatal(err)
		}
		t.Logf("capture %s", path)
	}
}

// The Save name caret stays two white columns at Home with the installed font.
func TestReleaseSaveCaretStaysWhiteAtHome(t *testing.T) {
	root := os.Getenv("AGAINROM_ASSETS")
	if root == "" {
		t.Skip("AGAINROM_ASSETS not set")
	}
	font := saveDialogInstallFont(t, root)
	a := newSaveDialogApp(t, &saveDialogSpy{}, ScreenTown)
	a.SetWords(AuthoredWords(), font, nil)
	if err := a.HeadlessKey("home"); err != nil {
		t.Fatal(err)
	}
	a.blink.off = false
	var e editField
	var pix *image.RGBA
	var err error
	calls := recordWidgets(t, func() { pix, err = a.composeSaveDialogScreen() })
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, c := range calls {
		if c.kind == widgetEdit && c.rect == saveControlRect(saveNameControl) {
			e = c.state.(editField)
			found = true
		}
	}
	if !found || !e.Focus || !e.Phase || e.Caret != 0 {
		t.Fatalf("caret setup: found %v state %+v", found, e)
	}
	x := e.Rect.Min.X + 4
	mismatches := 0
	for y := e.Rect.Min.Y + 2; y < e.Rect.Max.Y-3; y++ {
		for dx := 0; dx < 2; dx++ {
			if pix.RGBAAt(x+dx, y) != editCaret {
				mismatches++
			}
		}
	}
	if mismatches != 0 {
		t.Fatalf("installed Save Home caret: %d pixels overwritten, want two white columns", mismatches)
	}
}

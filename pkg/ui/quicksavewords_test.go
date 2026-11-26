package ui

import (
	"fmt"
	"reflect"
	"testing"

	"againrom/pkg/words"
)

func quickTownApp(t *testing.T, lang string) *App {
	t.Helper()
	a := mismatchApp(t, nil)
	a.flow.words.Engine = words.For(lang)
	a.flow.town = &stubTown{rows: []TownRow{{Text: "gates", Choosable: true}}}
	if err := a.flow.loadPrepared(nil, true); err != nil || a.Screen() != ScreenTown {
		t.Fatal("town", err, a.Screen())
	}
	return a
}

// TestQuickLoadAndSaveLinesComeFromTheWordsTable: the success and failure
// lines of a quick save and a quick load are the words of the install's
// language, not English literals.
func TestQuickLoadAndSaveLinesComeFromTheWordsTable(t *testing.T) {
	for _, lang := range []string{"english", "russian"} {
		book := words.For(lang)
		a := quickTownApp(t, lang)
		a.flow.loadUI.quick.Load = func() (MapOpener, bool, error) { return nil, true, nil }
		a.stepQuickSave(false)
		if want := book.Text("quick.load_done"); a.flow.msg != want || want == "" {
			t.Errorf("%s load done %q want %q", lang, a.flow.msg, want)
		}
		a.flow.loadUI.quick.Load = func() (MapOpener, bool, error) { return nil, false, fmt.Errorf("boom") }
		a.stepQuickSave(false)
		if want := book.Text("quick.load_failed") + ": boom"; a.flow.msg != want {
			t.Errorf("%s load failed %q want %q", lang, a.flow.msg, want)
		}
		a.flow.loadUI.quick.Save = func(bool) error { return fmt.Errorf("boom") }
		a.stepQuickSave(true)
		if want := book.Text("quick.save_failed") + ": boom"; a.flow.msg != want {
			t.Errorf("%s save failed %q want %q", lang, a.flow.msg, want)
		}
	}
	if ru, en := words.For("russian"), words.For("english"); ru.Text("quick.load_done") == en.Text("quick.load_done") {
		t.Error("the Russian quickload line is the English one")
	}
}

// TestAQuickLoadRefusedOverTheModsOpensTheLoadPageWindow: a quick load the mods
// refuse opens the window page load opens, with the same rows, not a message
// line.
func TestAQuickLoadRefusedOverTheModsOpensTheLoadPageWindow(t *testing.T) {
	a := quickTownApp(t, "english")
	a.flow.loadUI.quick.Load = func() (MapOpener, bool, error) {
		return nil, false, fmt.Errorf("load: %w", fixtureMismatch{everyMismatchKind})
	}
	a.stepQuickSave(false)
	got, open := a.HeadlessModMismatch()
	if !open || a.Screen() != ScreenLoad {
		t.Fatalf("window %v on %v, message %q", open, a.Screen(), a.flow.msg)
	}
	if !reflect.DeepEqual(got.Rows, everyMismatchRow) || a.flow.msg != "" {
		t.Fatalf("rows %q message %q", got.Rows, a.flow.msg)
	}
}

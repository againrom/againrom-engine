// Command menuaccelcheck witnesses MENU-KEY-013's closing behaviour (1014)
// against a lawful install.
//
//	menuaccelcheck labels [-assets DIR]
//	menuaccelcheck key [-assets DIR] -save NAME -key RUNE
//
// labels prints the thirteen MenuXxx words this build resolves off the install's
// own dialogs.txt, each as bytes (hex, never rendered text — golden rule 2),
// whether it carries a `~` mark, at what byte offset, and the marked byte
// itself with its CP866 reading (via pkg/formats/res.DecodeCP866, the same
// decoder the archive tier already exports for this exact purpose — no
// second table). It is FIGURES AND HEX ONLY, safe to record in a work item's
// evidence.
//
// key opens an original save under -save (a name headlessOpenLoad's own
// ACTIVATE would match — omit the extension), raises the in-game menu the
// same way Escape does, and types -key (one rune) through the SAME production
// dispatch a keyboard session uses: ui.App.HeadlessType, which app.go now
// ranges as runes rather than bytes (1014). It prints the screen and message
// before and after, so a reader can see whether the row was reached and what
// its action did.
//
// key never writes into the asset root: the writable save store is a
// temporary directory this process creates and removes, never the install.
//
// The asset root comes from -assets or AGAINROM_ASSETS and is never compiled
// in.
package main

import (
	"flag"
	"fmt"
	"os"

	"againrom/pkg/formats/res"
	"againrom/pkg/game"
	"againrom/pkg/ui"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "menuaccelcheck:", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: menuaccelcheck <labels|key> [flags]")
	}
	switch args[0] {
	case "labels":
		return runLabels(args[1:])
	case "key":
		return runKey(args[1:])
	default:
		return fmt.Errorf("unknown verb %q (want labels or key)", args[0])
	}
}

func openFrontEnd(fs *flag.FlagSet) (*game.FrontEnd, error) {
	assets := fs.Lookup("assets").Value.String()
	root := game.ResolveAssetRoot(assets, os.Getenv("AGAINROM_ASSETS"))
	if root == "" {
		return nil, fmt.Errorf("no asset root: pass -assets or set AGAINROM_ASSETS")
	}
	return game.NewFrontEnd(root)
}

func runLabels(args []string) error {
	fs := flag.NewFlagSet("menuaccelcheck labels", flag.ContinueOnError)
	fs.String("assets", "", "game install root (or AGAINROM_ASSETS)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	f, err := openFrontEnd(fs)
	if err != nil {
		return err
	}
	selector := -1 // -1 marks "no font" rather than colliding with a real 0
	if f.Font.Value() != nil {
		selector = f.Font.Value().Selector
	}
	fmt.Printf("font selector: %d (font err: %v)\n", selector, f.Font.Err())

	rows := []struct{ name, s string }{
		{"MenuSave", f.Words.MenuSave}, {"MenuLoad", f.Words.MenuLoad},
		{"MenuDiplomacy", f.Words.MenuDiplomacy},
		{"MenuGameOptions", f.Words.MenuGameOptions}, {"MenuSoundOptions", f.Words.MenuSoundOptions},
		{"MenuQuestObjectives", f.Words.MenuQuestObjectives}, {"MenuEndQuest", f.Words.MenuEndQuest},
		{"MenuReturn", f.Words.MenuReturn}, {"MenuAbort", f.Words.MenuAbort},
		{"MenuChangeMap", f.Words.MenuChangeMap}, {"MenuVictory", f.Words.MenuVictory},
		{"MenuExitMain", f.Words.MenuExitMain}, {"MenuExitWindows", f.Words.MenuExitWindows},
	}
	for _, r := range rows {
		b := []byte(r.s)
		mark, at := markedByte(b)
		fmt.Printf("%-20s % x", r.name, b)
		if at >= 0 {
			fmt.Printf("  marked byte 0x%02x at offset %d (CP866 %q)", mark, at, res.DecodeCP866([]byte{mark}))
		} else {
			fmt.Printf("  no mark")
		}
		fmt.Println()
	}
	return nil
}

// markedByte returns the byte a single `~` marks and its offset in b, or
// (0, -1) when b marks none. It is gamemenu.go's own walk (gameMenuAccelerator,
// pkg/ui, unexported), reproduced read-only for this diagnostic: it decides
// nothing this build acts on, only what this tool prints.
func markedByte(b []byte) (byte, int) {
	for i := 1; i < len(b); i++ {
		if b[i-1] != '~' {
			continue
		}
		if b[i] == '~' {
			i++
			continue
		}
		return b[i], i
	}
	return 0, -1
}

func runKey(args []string) error {
	fs := flag.NewFlagSet("menuaccelcheck key", flag.ContinueOnError)
	fs.String("assets", "", "game install root (or AGAINROM_ASSETS)")
	save := fs.String("save", "", "the original save's own display label, or a prefix (matched by ui.App.HeadlessActivate)")
	key := fs.String("key", "", "one rune to type over the raised in-game menu")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *save == "" || *key == "" {
		return fmt.Errorf("key requires -save and -key")
	}
	runes := []rune(*key)
	if len(runes) != 1 {
		return fmt.Errorf("-key must be exactly one rune, got %q", *key)
	}

	f, err := openFrontEnd(fs)
	if err != nil {
		return err
	}
	assetRoot := fs.Lookup("assets").Value.String()
	root := game.ResolveAssetRoot(assetRoot, os.Getenv("AGAINROM_ASSETS"))

	// THE WRITABLE STORE IS A SCRATCH DIRECTORY, NEVER THE INSTALL
	// (pipeline/check-preserved-installs.sh). The original save this verb
	// reads comes from OriginalStore, opened read-only over the same root
	// SaveStore.Write must never reach.
	scratch, err := os.MkdirTemp("", "menuaccelcheck-saves")
	if err != nil {
		return err
	}
	defer os.RemoveAll(scratch)

	a := f.App("menuaccelcheck")
	a.SetSaveSeams(f.SaveSeams(game.SaveStore{Dir: scratch}, game.OriginalStore{Dir: root}, nil))

	if err := a.HeadlessKey("load"); err != nil {
		return fmt.Errorf("open load window: %w", err)
	}
	if err := a.HeadlessActivate(*save); err != nil {
		return fmt.Errorf("load %q: %w", *save, err)
	}
	fmt.Printf("loaded %q: screen = %s\n", *save, a.Screen())

	if err := a.HeadlessKey("escape"); err != nil {
		return fmt.Errorf("open game menu: %w", err)
	}
	if a.Screen() != ui.ScreenGameMenu {
		return fmt.Errorf("escape did not raise the game menu; screen is %s", a.Screen())
	}
	before := fmt.Sprintf("%v", a.HeadlessRows())
	fmt.Printf("game menu open: %d rows %s\n", len(a.HeadlessRows()), before)

	if err := a.HeadlessType(string(runes[0]), false); err != nil {
		return fmt.Errorf("type %q: %w", *key, err)
	}
	fmt.Printf("typed %q (U+%04X): screen = %s, message = %q\n", *key, runes[0], a.Screen(), a.HeadlessMessage())
	return nil
}

package game

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf8"

	"againrom/pkg/formats/sav"
	"againrom/pkg/formats/textinput"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

// Test-only conversion helpers. The shipped game converts nothing; these let
// tests turn a frozen AGS fixture or a snapshot into SAV bytes and write one
// explicitly named SAV output.

// LegacyConversion is what one legacy snapshot became: the SAV bytes, the label
// bytes they carry, every label compromise, and the state the production LOAD
// installed before the producer captured it.
type LegacyConversion struct {
	SAV       []byte
	Label     string
	LabelDebt []string
	Loaded    Snapshot
}

// maxConvertedLabel is the longest label the SAV label region holds, leaving
// room for its terminating zero.
const maxConvertedLabel = 0xff

// ConvertLegacySnapshot commits the production LOAD of a snapshot decoded from
// the retired AGS format and emits SAV bytes through the same current-state
// producer as ordinary SAVE. sourceLabel is the AGS header label; MigrateLabel
// turns it into install-encoded SAV label bytes. A state the producer cannot
// express is an error the caller reports as debt; this function never
// substitutes a city for a mission or reseeds a map.
func (f *FrontEnd) ConvertLegacySnapshot(snapshot Snapshot, sourceLabel string) (LegacyConversion, error) {
	var out LegacyConversion
	open, town, err := f.Restore(snapshot)
	if err != nil {
		return out, err
	}
	if !town {
		if _, _, _, _, _, _, _, _, _, _, err = open(); err != nil {
			return out, err
		}
	}
	// Use the state the production LOAD actually installed, including any
	// legacy repair, so a repair cannot silently bypass the equality gate.
	if out.Loaded, err = f.ObserveLoadedState(!town); err != nil {
		return out, err
	}
	captured, _, err := f.Snapshot(!town)
	if err != nil {
		return out, err
	}
	out.Label, out.LabelDebt = f.MigrateLabel(sourceLabel)
	out.SAV, err = f.ExportCurrentSave(captured, out.Label)
	return out, err
}

// MigrateLabel turns an AGS header label into SAV label bytes through the
// install's own code page (DIV-1339). An AGS label is UTF-8 text; a SAV label
// is install bytes. A rune the install page cannot hold becomes '?', a label
// that is not UTF-8 keeps its bytes, and a label longer than the SAV region is
// cut; each compromise is returned as debt, never as a refusal.
func (f *FrontEnd) MigrateLabel(source string) (string, []string) {
	var debt []string
	label := source
	ascii := true
	for i := 0; i < len(source); i++ {
		if source[i] >= utf8.RuneSelf || source[i] < 0x20 || source[i] == 0x7f {
			ascii = false
			break
		}
	}
	switch {
	case ascii:
	case !utf8.ValidString(source):
		debt = append(debt, "label is not UTF-8; source bytes kept")
	default:
		selector := f.textSelector()
		encoded := make([]byte, 0, len(source))
		for _, r := range source {
			b, ok := textinput.EncodeRune(r, selector)
			if !ok {
				b = '?'
				debt = append(debt, fmt.Sprintf("label rune %U is not representable in the install code page; written as '?'", r))
			}
			encoded = append(encoded, b)
		}
		label = string(encoded)
	}
	if len(label) > maxConvertedLabel {
		label = label[:maxConvertedLabel]
		debt = append(debt, fmt.Sprintf("label cut to %d bytes", maxConvertedLabel))
	}
	return label, debt
}

// LoadObservedSAV runs a cold LOAD of SAV bytes on a fresh FrontEnd and returns
// the state it installed.
func (f *FrontEnd) LoadObservedSAV(saved []byte) (Snapshot, error) {
	open, town, err := f.RestoreOriginal(saved)
	if err != nil {
		return Snapshot{}, err
	}
	if !town {
		if _, _, _, _, _, _, _, _, _, _, err = open(); err != nil {
			return Snapshot{}, err
		}
	}
	return f.ObserveLoadedState(!town)
}

// CheckConvertedSaveTarget is a read-only preflight, also used by the command
// before loading inputs/assets. The output obeys the configured install fence
// and the archive-ancestor census, including aliases of not-yet-created children.
func CheckConvertedSaveTarget(path, assetsRoot string) error {
	if path == "" {
		return fmt.Errorf("conversion output path is required")
	}
	if assetsRoot == "" {
		return fmt.Errorf("conversion asset root is required for the read-only install fence")
	}
	return refuseOriginalWriteTarget(filepath.Dir(path), assetsRoot)
}

// WriteConvertedSave publishes one explicitly named SAV output through
// SaveStore's synced, no-replace atomic protocol. There is no directory or
// filename default. ConvertSave's only remaining direction is AGS-to-SAV
// (AGS writing is retired), so this always validates and writes SAV bytes.
func WriteConvertedSave(path string, data []byte, assetsRoot string) error {
	if err := CheckConvertedSaveTarget(path, assetsRoot); err != nil {
		return err
	}
	ext := strings.ToLower(filepath.Ext(path))
	if ext != ".sav" {
		return fmt.Errorf("conversion output must have .sav extension")
	}
	if _, err := sav.DecodeDocumentData(data); err != nil {
		return err
	}
	if _, err := os.Lstat(path); err == nil {
		return fmt.Errorf("conversion output already exists: %w", os.ErrExist)
	} else if !os.IsNotExist(err) {
		return err
	}
	store := SaveStore{Dir: filepath.Dir(path)}
	validate := func(dir saveDirectoryIdentity) error {
		return refuseOriginalWriteTarget(dir.path, assetsRoot)
	}
	_, err := store.write(data, "save-conversion", ext, validate, func(at int) (string, bool) {
		return filepath.Base(path), at == 0
	})
	if err != nil {
		return fmt.Errorf("publish conversion output without replacement: %w", err)
	}
	return nil
}

// Restore prepares a snapshot and reports the one commit that replaces this
// front end's game: a map opener for a mission save, or the town.
//
// IT IS THE ONE RESTORE AND IT NAMES NO FILE FORMAT. This story ships one
// producer of a Snapshot; a reader of some other format would produce one of
// these and reach exactly this method.
//
// NO FRONT-END FIELD IS WRITTEN UNTIL THE CANDIDATE IS COMPLETE. A mission
// opener returned from here performs only the commit and returns already-built
// seams; all fallible map, asset, world and mission work happened before this
// method returned. A town has no opener, so its fully prepared state commits
// here after preparation succeeds.
func (f *FrontEnd) Restore(s Snapshot) (ui.MapOpener, bool, error) {
	candidate, err := f.prepareRestore(s)
	if err != nil {
		return nil, false, err
	}
	if candidate.townOnly {
		// SHOP-TOWN-022's second sender: loading a save that is NOT in a
		// battle sends the same SetCap-and-Generate the homecoming sends.
		f.installCandidate(candidate)
		return nil, true, nil
	}
	return openPrepared(candidate.prepared, func() { f.installCandidate(candidate) }), false, nil
}

// openLoadNotice opens a dialogue notice naming what an older byte-form
// version could not restore (1032 B3), reusing the same notice window
// showOutcome and openDialogue already draw through rather than a second
// display path.
//
// IT CARRIES NO EVENT PAYLOAD. m.payload stays nil, so advanceNotice never
// looks for an event part; what it pages through is m.pages, the list handed
// in here.
func (mw *mapWorld) openLoadNotice(pages []string) {
	if len(pages) == 0 {
		return
	}
	m := mw.mission
	m.open, m.kind, m.payload, m.part, m.portrait = true, ui.NoticeDialogue, nil, 1, false
	m.pages = pages
	if m.outcome == sim.OutcomeWon || m.outcome == sim.OutcomeLost {
		m.outcomeShown = false // restore the terminal panel after the disclosure
	}
	mw.view.SetDialogue(ui.Dialogue{Text: pages[0]})
}

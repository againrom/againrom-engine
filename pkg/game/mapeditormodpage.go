package game

import (
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"

	"againrom/pkg/mod"
	"againrom/pkg/ui"
)

// modPageGate is the gate of every edit a mod window page writes: the mod is
// writable and no read-only refusal of its records holds.
func (s *mapInspectionEdits) modPageGate() error {
	x := s.inspector
	if err := x.modWritable(); err != nil {
		return err
	}
	if msg := x.modContextRefusal(); msg != "" {
		return errors.New(msg)
	}
	return nil
}

// modPageEdit is the opening of one edit of a mod window page. The workspace
// is prepared; unless key is one of free, which write nothing, the gate holds
// and an asset key imports its file; then edit gets the trimmed value.
func (s *mapInspectionEdits) modPageEdit(key, value string, edit func(key, value string) (*ui.InspectionDocument, error), free ...string) (*ui.InspectionDocument, error) {
	if err := s.prepareWorkspace(); err != nil {
		return nil, err
	}
	if !slices.Contains(free, key) {
		if err := s.modPageGate(); err != nil {
			return nil, err
		}
		if isModAssetKey(key) {
			d, _, err := s.setModAsset(key, value)
			return d, err
		}
	}
	return edit(key, strings.TrimSpace(value))
}

// modGone is a page's missing-row error in its word; args fill the word.
func (x *MapInspector) modGone(word string, args ...any) error {
	if len(args) == 0 {
		return errors.New(x.word(word))
	}
	return fmt.Errorf(x.word(word), args...)
}

// modRowKey splits a row key "<kind>.<index>.<field>". ok is whether the
// index is a row number from 0; the caller checks the kind.
func modRowKey(key string) (kind string, i int, field string, ok bool) {
	kind, rest, _ := strings.Cut(key, ".")
	index, field, _ := strings.Cut(rest, ".")
	i, err := strconv.Atoi(index)
	return kind, i, field, err == nil && i >= 0
}

// modRowsPage is the save gate of the held rows of a mod window page.
type modRowsPage struct {
	save  string                              // the page's save word: names the file, then the reason
	rel   string                              // the held file
	text  func(s *mapInspectionEdits) rowText // nil: no text lookup
	check func(s *mapInspectionEdits, held mod.Data) error
}

// modRowsGate refuses a save while the page's held file is one the game
// cannot read, or check refuses its records.
func (s *mapInspectionEdits) modRowsGate(p modRowsPage) error {
	d := s.workspace.Project
	if d == nil {
		return nil
	}
	if _, ok := d.Files[p.rel]; !ok {
		return nil
	}
	var text rowText
	if p.text != nil {
		text = p.text(s)
	}
	held, err := s.heldRows(d, text, p.rel)
	if err != nil {
		return fmt.Errorf(s.inspector.word(p.save), p.rel, err)
	}
	if p.check == nil {
		return nil
	}
	return p.check(s, held)
}

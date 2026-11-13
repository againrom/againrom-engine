package game

import (
	"fmt"
	"os"

	"againrom/pkg/formats/alm"
	"againrom/pkg/mod"
	"againrom/pkg/ui"
	"againrom/pkg/words"
)

// A map's texts inside the map. The switch of the File menu and of the save
// window, off by default,
// makes the save write the page tree the text folder would hold into the
// Againrom texts record of an extended map, and no text folder; a folder
// already beside the map is left as it is. The record is not document
// state: the open takes it out of the model and turns the switch on when
// the map carried it, and each writer of the map's bytes puts it back from
// the pages of that moment. mod.Mission.ReadTexts reads it before the
// folders.

// TextsInside reports that the save writes the pages inside the map: the
// switch is on and the map is extended.
func (s *mapInspectionEdits) TextsInside() bool {
	return s.textsInside && s.model != nil && s.model.Extended()
}

// SetTextsInside turns the switch on or off. On an original map it answers
// the script form question: only an extended map holds the record.
func (s *mapInspectionEdits) SetTextsInside(on bool) error {
	if on && s.model != nil && !s.model.Extended() {
		return &ui.ScriptFormNeeded{Why: words.For(s.inspector.pageLanguage()).Text("editor.save.textsinside.almx")}
	}
	s.textsInside = on
	return nil
}

// withTexts is b with the map's pages in its texts record when the switch is
// on, else b: the page tree pagesForSave gives the folder writer. A map that
// with the record is over the size mod.Mission.ReadTexts reads is refused:
// the game would read no page from it.
func (s *mapInspectionEdits) withTexts(b []byte) ([]byte, error) {
	if !s.TextsInside() {
		return b, nil
	}
	pages, err := s.pagesForSave()
	if err != nil {
		return nil, err
	}
	out, err := alm.AppendTextsRecord(b, pages)
	if err == nil && len(out) > mod.MaxFileBytes {
		return nil, fmt.Errorf(words.For(s.inspector.pageLanguage()).Text("editor.save.textsinside.size"), len(out), mod.MaxFileBytes)
	}
	return out, err
}

// sourceTexts is the page tree the texts record of the map's own file holds,
// and whether the file holds the record; none when the map has no host file,
// it cannot be read, or holds no record.
func (s *mapInspectionEdits) sourceTexts() (map[string][]byte, bool, error) {
	path, ok := mapHostPath(ui.ParseMapSource(s.source), s.inspector.archives)
	if !ok {
		return nil, false, nil
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, false, nil
	}
	return alm.TextsOf(b)
}

// textsKept is the refusal of a save that would leave the pages the map's
// own file holds inside in neither the map nor its folder: nothing is
// written, and the answer names the cause.
func (s *mapInspectionEdits) textsKept(cause error) error {
	return fmt.Errorf(words.For(s.inspector.pageLanguage()).Text("editor.save.textsinside.kept"), cause)
}

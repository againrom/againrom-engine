package game

import (
	"bytes"
	"errors"
	"fmt"
	"hash/fnv"
	"image"
	"image/draw"
	"image/png"
	"path"
	"path/filepath"

	"againrom/pkg/data"
	"againrom/pkg/mod"
	"againrom/pkg/ui"
)

// speakerPictureSize is the size of a mod speaker record's picture: the
// message window's face pane, so the picture is drawn whole and the pane
// cuts nothing.
func speakerPictureSize() image.Point { return ui.NoticeFaceWindow(0, 0).Size() }

// decodeSpeakerPicture is the face a mod's PNG gives, refused unless it is a
// PNG of the pane's size.
func decodeSpeakerPicture(b []byte) (*image.RGBA, error) {
	decoded, err := png.Decode(bytes.NewReader(b))
	if err != nil {
		return nil, fmt.Errorf("the picture is not a PNG the game can read: %v", err)
	}
	want := speakerPictureSize()
	if got := decoded.Bounds().Size(); got != want {
		return nil, fmt.Errorf("the picture is %dx%d; a speaker's picture is %dx%d, the message window's face", got.X, got.Y, want.X, want.Y)
	}
	out := image.NewRGBA(image.Rect(0, 0, want.X, want.Y))
	draw.Draw(out, out.Bounds(), decoded, decoded.Bounds().Min, draw.Src)
	return out, nil
}

// speakerLayer turns merged mod records into the catalog's layer, each name in
// the install's own byte alphabet. A refusal names the mod, the file and the
// line.
func speakerLayer(records []mod.Speaker, selector int) ([]data.NPCLayerRecord, error) {
	var layer []data.NPCLayerRecord
	for _, r := range records {
		pic, err := decodeSpeakerPicture(r.PictureBytes)
		if err != nil {
			return nil, r.Errorf("picture", "%v", err)
		}
		name, err := encodeSaveLabel(r.Name, selector)
		if err != nil {
			return nil, r.Errorf("name", "the name %q cannot be drawn by this install: %v", r.Name, err)
		}
		layer = append(layer, data.NPCLayerRecord{
			NPCEntry: data.NPCEntry{ID: r.ID, Section: fmt.Sprintf("npc%d", r.ID), Name: name, Mod: r.Mod, Map: r.Map, Unit: r.Unit, Bound: r.Bound},
			Face:     data.NPCFace{Kind: data.NPCModPicture, Picture: pic, HasWindow: true},
		})
	}
	return layer, nil
}

// setModSpeakers lays the mods' speaker records over the table's one NPC
// catalog and rebuilds the speaker table from it, so every reader of a
// speaker sees them.
func (f *FrontEnd) setModSpeakers(d mod.SpeakerData) ([]mod.Override, error) {
	if d.Empty() {
		return nil, nil
	}
	records, overrides, err := mod.MergeSpeakers(d)
	if err != nil {
		return nil, err
	}
	if f.Table == nil {
		return nil, errors.New("the install has no speaker records for the mods to add to")
	}
	layer, err := speakerLayer(records, f.textSelector())
	if err != nil {
		return nil, err
	}
	f.Table.NPCCatalog = f.Table.NPCCatalog.WithLayer(layer)
	f.NPCFaces = speakerFaces(f.Table)
	return overrides, nil
}

// speakerSelector is the install's text selector, 0 with no install open.
func (x *MapInspector) speakerSelector() int {
	if x.archives == nil || x.archives.Containers == nil {
		return 0
	}
	return LanguageSelector(x.archives.Containers)
}

// speakerMapKey is the map's path in the mod the editor saves into, when the
// open map lives in that mod.
func (s *mapInspectionEdits) speakerMapKey() (string, bool) {
	m := s.inspector.mod
	if m == nil {
		return "", false
	}
	e, ok := m.entryFor(s.source)
	return e.Map, ok
}

// speakerUnitID is the script number a speaker record binds placement unit
// of the open map by. A unit whose number is 0 or held by another unit of the
// map has no number that names it alone, and is refused.
func (s *mapInspectionEdits) speakerUnitID(unit int) (uint16, error) {
	m, err := s.model.Map()
	if err != nil {
		return 0, err
	}
	if unit < 0 || unit >= len(m.Units) {
		return 0, fmt.Errorf("no unit %d on this map", unit)
	}
	x, id := s.inspector, m.Units[unit].UnitID
	if id == 0 {
		return 0, errors.New(x.word("editor.speaker.bind.noid"))
	}
	for i, u := range m.Units {
		if i != unit && u.UnitID == id {
			return 0, fmt.Errorf(x.word("editor.speaker.bind.shared"), id)
		}
	}
	return id, nil
}

// SpeakerBindRefusal says why a speaker record cannot be bound to placement
// unit, nil when it can.
func (s *mapInspectionEdits) SpeakerBindRefusal(unit int) error {
	_, err := s.speakerUnitID(unit)
	return err
}

// SpeakerPictureFolder is the mod folder a new speaker's picture window
// starts in.
func (s *mapInspectionEdits) SpeakerPictureFolder() string {
	if m := s.inspector.mod; m != nil {
		return m.dir
	}
	return ""
}

// boundSpeaker is the mod record bound to placement unit of the open map, by
// the unit's script number. A unit with no number of its own answers none.
func (s *mapInspectionEdits) boundSpeaker(unit int) (int64, bool) {
	key, ok := s.speakerMapKey()
	if !ok {
		return 0, false
	}
	id, err := s.speakerUnitID(unit)
	if err != nil {
		return 0, false
	}
	for _, e := range s.inspector.table.NPCCatalog.Records() {
		if e.Bound && e.Map == key && e.Unit == int32(id) {
			return int64(e.ID), true
		}
	}
	return 0, false
}

// speakerMissOffersRecord reports whether the map lives in a mod folder the
// editor may write and a unit missed for the reason miss gives stands at
// mission start.
func (s *mapInspectionEdits) speakerMissOffersRecord(miss ui.SpeakerMiss) bool {
	if m := s.inspector.mod; m == nil || m.copyOnly {
		return false
	}
	switch miss.Cause {
	case ui.SpeakerNoFit, ui.SpeakerNotFirst, ui.SpeakerNoPicture, ui.SpeakerByHero, ui.SpeakerDown:
		_, ok := s.speakerMapKey()
		return ok
	}
	return false
}

// CanAddSpeaker reports whether a speaker record for placement unit, which
// no record shows for the reason miss gives, can be written. It cannot while
// the mod's records did not load: their numbers are unknown.
func (s *mapInspectionEdits) CanAddSpeaker(unit int, miss ui.SpeakerMiss) bool {
	return s.inspector.modContextRefusal() == "" && s.speakerMissOffersRecord(miss)
}

// SpeakerAddRefusal says why a miss that would offer a record does not: the
// mod's records did not load. It is empty otherwise.
func (s *mapInspectionEdits) SpeakerAddRefusal(miss ui.SpeakerMiss) string {
	if !s.speakerMissOffersRecord(miss) {
		return ""
	}
	return s.inspector.modContextRefusal()
}

// AddSpeaker adds a bound record and its face to the project draft in one undo.
func (s *mapInspectionEdits) AddSpeaker(unit int, name, picture string) (int64, error) {
	x := s.inspector
	key, ok := s.speakerMapKey()
	if !ok || x.mod.copyOnly {
		return 0, errors.New("this map does not live in a mod the editor can write")
	}
	if refusal := x.modContextRefusal(); refusal != "" {
		return 0, errors.New(refusal)
	}
	m := x.mod
	if name == "" {
		return 0, errors.New("the speaker needs a name")
	}
	if _, err := encodeSaveLabel(name, x.speakerSelector()); err != nil {
		return 0, fmt.Errorf("the name %q cannot be drawn by this install: %v", name, err)
	}
	unitID, err := s.speakerUnitID(unit)
	if err != nil {
		return 0, err
	}
	if err := s.prepareWorkspace(); err != nil {
		return 0, err
	}
	before, err := s.draftSpeakersFile()
	if err != nil {
		return 0, err
	}
	id, err := x.freeSpeakerNumber(before)
	if err != nil {
		return 0, err
	}
	face, pic, err := s.modAssetDraft(speakerAssets, picture, path.Base(speakerPictureFile(id)))
	if err != nil {
		return 0, err
	}
	rec := mod.Speaker{ID: id, Name: name, Picture: face, Map: key, Unit: int32(unitID), Bound: true}
	src := mod.AppendSpeaker(before, rec)
	if _, err := mod.ReadSpeakers(m.id, m.dir, mod.SpeakersFile, src, speakerDraftFiles(s.workspace, rec.Picture, pic)); err != nil {
		return 0, err
	}
	_, err = s.changeWorkspace(func(w *editorWorkspaceState) error {
		if w.Project == nil {
			w.Project = m.document()
		}
		w.Project.Files[mod.SpeakersFile] = src
		w.Project.Files[rec.Picture] = bytes.Clone(pic)
		return nil
	})
	if err != nil {
		return 0, err
	}
	return int64(id), nil
}

func speakerDraftFiles(w *editorWorkspaceState, picture string, data []byte) map[string][]byte {
	files := map[string][]byte{}
	if w != nil && w.Project != nil {
		for name, b := range w.Project.Files {
			files[name] = b
		}
	}
	files[picture] = data
	return files
}

func (s *mapInspectionEdits) draftSpeakersFile() ([]byte, error) {
	if w := s.workspace; w != nil && w.Project != nil {
		if b, ok := w.Project.Files[mod.SpeakersFile]; ok {
			return b, nil
		}
	}
	return s.inspector.mod.speakersFile()
}

// speakerAssets is the folder under assets that holds the speakers' faces.
const speakerAssets = "speakers"

// speakerPictureFile is the path in the mod of the picture record id's add
// imports.
func speakerPictureFile(id int32) string {
	return fmt.Sprintf("%s/%s/%d.png", mod.AssetsFolder, speakerAssets, id)
}

// speakersFile is the text of the mod's speakers file, nil when it holds none.
func (m *editorMod) speakersFile() ([]byte, error) { return modSpeakersFile(m.dir) }

// modSpeakersFile is the text of the speakers file of the mod at dir, a
// folder or a packed file, nil when it holds none.
func modSpeakersFile(dir string) ([]byte, error) {
	t, err := mod.OpenTree(dir)
	if err != nil {
		return nil, err
	}
	if st, err := t.Stat(mod.SpeakersFile); err != nil || st.IsDir() {
		return nil, nil
	}
	b, _, err := t.ReadInside(mod.SpeakersFile)
	return b, err
}

// speakerNumberStart is where the number of a new record of the mod id is
// looked for from: a stable hash of the id into the mods' band, so two mods
// made apart start at different numbers.
func speakerNumberStart(id string) int32 {
	h := fnv.New32a()
	_, _ = h.Write([]byte(id))
	return int32(mod.FirstModSpeaker + h.Sum32()%uint32(mod.LastModSpeaker-mod.FirstModSpeaker+1))
}

// freeSpeakerNumber is the number a new record of the current mod takes: the
// first number from the mod's start, going up through the band and round to
// its bottom, that no record holds. It reads the mod's own speakers file src,
// the layered catalog and the speakers file of every mod beside this one. A
// number whose picture file a record of this mod names counts as held.
func (x *MapInspector) freeSpeakerNumber(src []byte) (int32, error) {
	own, err := mod.ParseSpeakers(x.mod.id, mod.SpeakersFile, src)
	if err != nil {
		return 0, err
	}
	held := map[int32]bool{}
	pictures := map[string]bool{}
	for _, r := range own.Speakers {
		held[r.ID] = true
		pictures[r.Picture] = true
	}
	if x.table != nil {
		for _, e := range x.table.NPCCatalog.Records() {
			held[e.ID] = true
		}
	}
	// A mods folder that cannot be listed, or a neighbour whose file cannot
	// be read, holds no number the add can see; it refuses nothing here.
	entries, _ := mod.Scan(filepath.Dir(filepath.Clean(x.mod.dir)))
	for _, e := range entries {
		b, err := modSpeakersFile(e.Dir)
		if err != nil {
			continue
		}
		other, err := mod.ParseSpeakers(e.Folder, mod.SpeakersFile, b)
		if err != nil {
			continue
		}
		for _, r := range other.Speakers {
			held[r.ID] = true
		}
	}
	span := int32(mod.LastModSpeaker - mod.FirstModSpeaker + 1)
	offset := speakerNumberStart(x.mod.id) - mod.FirstModSpeaker
	for i := int32(0); i < span; i++ {
		id := mod.FirstModSpeaker + (offset+i)%span
		if !held[id] && !pictures[speakerPictureFile(id)] {
			return id, nil
		}
	}
	return 0, fmt.Errorf("every record number from %d to %d is taken", mod.FirstModSpeaker, mod.LastModSpeaker)
}

// carryModSpeakers puts the speaker records of the mod the open map lives in
// on the test mission, so a test play shows them.
func (s *mapInspectionEdits) carryModSpeakers(m *mod.Mission) error {
	x := s.inspector
	if x.mod == nil {
		return nil
	}
	src, err := s.draftSpeakersFile()
	if err != nil || src == nil {
		return err
	}
	d, err := mod.ReadSpeakers(x.mod.id, x.mod.dir, mod.SpeakersFile, src, speakerDraftFiles(s.workspace, "", nil))
	if err != nil {
		return err
	}
	if m.Carried == nil {
		m.Carried = map[string][]byte{}
	}
	m.Carried[mod.SpeakersFile] = src
	for _, r := range d.Speakers {
		if r.Picture != "" {
			m.Carried[r.Picture] = r.PictureBytes
		}
	}
	return nil
}

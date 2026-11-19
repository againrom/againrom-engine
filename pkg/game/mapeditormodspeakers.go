package game

import (
	"errors"
	"fmt"
	"path"
	"slices"
	"strconv"
	"strings"

	"againrom/pkg/mod"
	"againrom/pkg/namefold"
	"againrom/pkg/ui"
)

// speakersAddKey is the Speakers page's adder.
const speakersAddKey = "newspeaker"

// speakerName names a record where the page shows it.
func speakerName(r mod.Speaker) string { return fmt.Sprintf("%s (%d)", r.Name, r.ID) }

// speakersRows is the save gate of the speakers file: a speaker record the
// game cannot show is refused, naming the record and what to do.
var speakersRows = modRowsPage{save: "editor.speakers.err.save", rel: mod.SpeakersFile,
	check: func(s *mapInspectionEdits, held mod.Data) error {
		x := s.inspector
		for _, r := range held.Speakers.Speakers {
			if r.Off {
				continue
			}
			var problem string
			switch {
			case r.Name == "":
				problem = x.word("editor.speakers.err.noname")
			case len(r.PictureBytes) == 0:
				problem = x.word("editor.speakers.err.nopicture")
			default:
				if _, err := decodeSpeakerPicture(r.PictureBytes); err != nil {
					problem = x.inLanguage(err).Error()
				} else if _, err := encodeSaveLabel(r.Name, x.speakerSelector()); err != nil {
					problem = fmt.Sprintf(x.word("editor.speaker.err.undrawable"), r.Name)
				}
			}
			if problem != "" {
				return fmt.Errorf(x.word("editor.speakers.err.save"), speakerName(r), problem)
			}
		}
		return nil
	}}

// speakersFields is W's Speakers page: the Add button, then each record's
// name, picture and removal.
func (s *mapInspectionEdits) speakersFields() []ui.InspectionField {
	x := s.inspector
	if !s.modPageHolds() {
		return nil
	}
	project := s.workspace.Project
	if project == nil {
		project = x.mod.document()
	}
	held, err := s.heldRows(project, nil, mod.SpeakersFile)
	if err != nil {
		return []ui.InspectionField{x.pageLine("error", "", err.Error())}
	}
	add := workspaceText(speakersAddKey, x.word("editor.speakers.add"), "")
	add.Meaning = x.word("editor.speakers.add.help")
	out := []ui.InspectionField{add}
	for i, r := range held.Speakers.Speakers {
		key := "speaker." + strconv.Itoa(i) + "."
		start := len(out)
		if r.Off {
			out = append(out, x.pageLine(key+"state", "", x.word("editor.speakers.isoff")))
		} else {
			name := workspaceText(key+"name", x.word("editor.speakers.name"), r.Name)
			name.Meaning = x.word("editor.speakers.name.help")
			pic := x.word("editor.speakers.nopicture")
			if r.Picture != "" {
				pic = fmt.Sprintf(x.word("editor.speakers.picture.is"), r.Picture)
			}
			file := workspaceText(key+"picture", x.word("editor.speakers.picture"), "")
			file.Meaning = x.word("editor.speakers.picture.help")
			out = append(out, name, x.pageLine(key+"file", "", pic), file)
			if r.Bound {
				out = append(out, x.pageLine(key+"bound", "", fmt.Sprintf(x.word("editor.speakers.bound"), r.Unit, r.Map)))
			}
		}
		out = append(out, ui.InspectionField{Key: key + "remove", Label: x.word("editor.speakers.remove"), Meaning: x.word("editor.speakers.remove.help"), Action: true, Column: 1})
		for k := start; k < len(out); k++ {
			out[k].Group = speakerName(r)
		}
	}
	return out
}

// editSpeakers is one draft step of the speakers file in the project w.
func (s *mapInspectionEdits) editSpeakers(w *editorWorkspaceState, edit func(*mod.SpeakerData) error) error {
	return kindFiles{s: s, w: w}.editRows(nil, []string{mod.SpeakersFile}, func(d *mod.Data) error { return edit(&d.Speakers) })
}

// setSpeakersField makes one edit of the Speakers page a draft step.
func (s *mapInspectionEdits) setSpeakersField(key, value string) (*ui.InspectionDocument, error) {
	return s.modPageEdit(key, value, s.speakersEdit)
}

func (s *mapInspectionEdits) speakersEdit(key, value string) (*ui.InspectionDocument, error) {
	x := s.inspector
	gone := x.modGone("editor.speakers.err.gone", key)
	if key == speakersAddKey {
		return s.addSpeakerFrom(value)
	}
	kind, i, field, ok := modRowKey(key)
	if !ok || kind != "speaker" {
		return nil, gone
	}
	if field == "picture" {
		return s.changeSpeakerPicture(i, value, gone)
	}
	return s.editKind(func(k kindFiles) error {
		return s.editSpeakers(k.w, func(d *mod.SpeakerData) error {
			if i >= len(d.Speakers) {
				return gone
			}
			return s.editSpeaker(k, d, i, field, value, gone)
		})
	})
}

// addSpeakerFrom adds a record of a free number, named New speaker, whose
// picture is the host file value, imported as the mod's asset in the same
// draft step: a record with no picture is no draft the run can read.
func (s *mapInspectionEdits) addSpeakerFrom(value string) (*ui.InspectionDocument, error) {
	x := s.inspector
	if value == "" {
		return nil, errors.New(x.word("editor.speakers.err.nopath"))
	}
	own, err := s.draftSpeakersFile()
	if err != nil {
		return nil, err
	}
	id, err := x.freeSpeakerNumber(own)
	if err != nil {
		return nil, err
	}
	rel, data, err := s.modAssetDraft(speakerAssets, value, path.Base(speakerPictureFile(id)))
	if err != nil {
		return nil, err
	}
	_, doc, err := s.importModFile(rel, data, true, nil, func(w *editorWorkspaceState, rel string) error {
		return s.editSpeakers(w, func(d *mod.SpeakerData) error {
			d.Speakers = append(d.Speakers, mod.NewSpeaker(id, x.word("editor.speaker.add.name"), rel))
			return nil
		})
	})
	return doc, err
}

// editSpeaker makes the rename or removal of record i. A record a message
// names is not removed: the refusal names where it is used.
func (s *mapInspectionEdits) editSpeaker(k kindFiles, d *mod.SpeakerData, i int, field, value string, gone error) error {
	x := s.inspector
	r := &d.Speakers[i]
	switch field {
	case "remove":
		if uses := s.speakerUses(k.w, r.ID); len(uses) > 0 {
			return fmt.Errorf(x.word("editor.speakers.err.used"), speakerName(*r), strings.Join(uses, ", "))
		}
		pic := r.Picture
		d.Speakers = slices.Delete(d.Speakers, i, i+1)
		if pic != "" && !slices.ContainsFunc(d.Speakers, func(o mod.Speaker) bool { return o.Picture == pic }) && mod.HeldDraft(pic) {
			return k.drop(pic)
		}
	case "name":
		if r.Off {
			return gone
		}
		if value == "" {
			return errors.New(x.word("editor.speaker.add.noname"))
		}
		if _, err := encodeSaveLabel(value, x.speakerSelector()); err != nil {
			return x.inLanguage(wordErr("editor.speaker.err.undrawable", value))
		}
		r.Name = value
		r.SetField("name")
	default:
		return gone
	}
	return nil
}

// changeSpeakerPicture imports the host file value as record i's picture, the
// import of the mod's assets, in one draft step with the record's new
// picture path.
func (s *mapInspectionEdits) changeSpeakerPicture(i int, value string, gone error) (*ui.InspectionDocument, error) {
	x := s.inspector
	if value == "" {
		return nil, errors.New(x.word("editor.speakers.err.nopath"))
	}
	held, err := s.heldRows(s.workspace.Project, nil, mod.SpeakersFile)
	if err != nil {
		return nil, err
	}
	if i >= len(held.Speakers.Speakers) || held.Speakers.Speakers[i].Off {
		return nil, gone
	}
	id := held.Speakers.Speakers[i].ID
	rel, data, err := s.modAssetDraft(speakerAssets, value, path.Base(speakerPictureFile(id)))
	if err != nil {
		return nil, err
	}
	_, doc, err := s.importModFile(rel, data, true, nil, func(w *editorWorkspaceState, rel string) error {
		return s.editSpeakers(w, func(d *mod.SpeakerData) error {
			r := &d.Speakers[i]
			old := r.Picture
			r.Picture = rel
			r.SetField("picture")
			if old != "" && old != rel && !slices.ContainsFunc(d.Speakers, func(o mod.Speaker) bool { return o.Picture == old }) &&
				strings.HasPrefix(namefold.ModTree(old), mod.AssetsFolder+"/"+speakerAssets+"/") {
				return kindFiles{s: s, w: w}.drop(old)
			}
			return nil
		})
	})
	return doc, err
}

// speakerUses is where the open project names record id in a message: the
// project's page and strings files, the open map's pages and the mod's page
// files the project does not hold.
func (s *mapInspectionEdits) speakerUses(w *editorWorkspaceState, id int32) []string {
	var uses []string
	scan := func(place string, b []byte) {
		if namesSpeaker(string(b), int(id)) && !slices.Contains(uses, place) {
			uses = append(uses, place)
		}
	}
	held := map[string]bool{}
	for name, b := range w.Project.Files {
		held[namefold.ModTree(name)] = true
		if strings.HasSuffix(namefold.ModTree(name), ".txt") || slices.Contains(textFiles(), name) {
			scan(name, b)
		}
	}
	for name, b := range w.Pages {
		scan(name, b)
	}
	if m := s.inspector.mod; m != nil {
		for _, rel := range m.files {
			if !strings.HasSuffix(namefold.ModTree(rel), ".txt") || held[namefold.ModTree(rel)] {
				continue
			}
			if b, err := modDataFile(m.dir, rel); err == nil {
				scan(rel, b)
			}
		}
	}
	slices.Sort(uses)
	return uses
}

// namesSpeaker reports whether a tag of text names speaker id with npc=.
func namesSpeaker(text string, id int) bool {
	for {
		i := strings.IndexByte(text, '<')
		if i < 0 {
			return false
		}
		text = text[i+1:]
		j := strings.IndexByte(text, '>')
		if j < 0 {
			return false
		}
		if v, named := tagSpeaker(text[:j]); named && v == id {
			return true
		}
		text = text[j+1:]
	}
}

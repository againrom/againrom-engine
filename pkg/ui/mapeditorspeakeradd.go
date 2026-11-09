package ui

import (
	"errors"
	"fmt"
	"strings"

	"againrom/pkg/pathdialog"
)

// InspectionSpeakerRecords is a map host that can write a speaker record into
// the mod that holds the map. A map outside a mod answers CanAddSpeaker false.
type InspectionSpeakerRecords interface {
	// CanAddSpeaker reports whether a record can be added for placement
	// unit, which no record shows for the reason miss gives.
	CanAddSpeaker(unit int, miss SpeakerMiss) bool
	// SpeakerAddRefusal says why a miss that would offer a record does not:
	// the line the click shows, empty when no such reason holds.
	SpeakerAddRefusal(miss SpeakerMiss) string
	// SpeakerBindRefusal says why a record cannot be bound to placement unit,
	// nil when it can.
	SpeakerBindRefusal(unit int) error
	// AddSpeaker writes the record bound to placement unit, with the name and
	// the PNG at the host path picture, and answers its number.
	AddSpeaker(unit int, name, picture string) (int64, error)
	// SpeakerPictureFolder is the folder the picture's window starts in.
	SpeakerPictureFolder() string
}

// speakerPrompt is what the add-a-speaker prompt holds: the clicked unit and
// the way back to the speaker pick that opened it.
type speakerPrompt struct {
	unit int
	back func()
}

// speakerAddHost is the document's host when it can write speaker records.
func (e *MapEditor) speakerAddHost() (InspectionSpeakerRecords, bool) {
	if e.doc == nil {
		return nil, false
	}
	host, ok := e.doc.Edits.(InspectionSpeakerRecords)
	return host, ok
}

// askSpeaker opens the add-a-speaker prompt for placement unit, its text
// already the name when a refused picture brought it back. back returns to
// the armed speaker pick; the popup's own cancel point stays as it was.
func (e *MapEditor) askSpeaker(unit int, name string, back func()) {
	saved := e.cancelRestore
	e.openPrompt(editorPrompt{kind: promptSpeaker, path: name, speaker: &speakerPrompt{unit: unit, back: back}})
	e.cancelRestore = saved
}

// speakerPictureRequest is what the picture's window lists: PNG files, from
// the host's folder, else the editor's base folder.
func (e *MapEditor) speakerPictureRequest(host InspectionSpeakerRecords) pathdialog.Request {
	dir := host.SpeakerPictureFolder()
	if dir == "" {
		dir = e.pathDialog.baseDir
	}
	return pathdialog.Request{
		Kind: pathdialog.OpenFile, Title: e.word("editor.speaker.add.picture"), StartDir: dir,
		Filters: []pathdialog.Filter{{Name: e.word("editor.speaker.add.pictures"), Extensions: []string{"png"}}},
	}
}

// speakerPromptStep takes the record's name and asks for its picture. An
// empty name leaves the prompt open.
func (e *MapEditor) speakerPromptStep(value string) error {
	p := e.prompt.speaker
	host, ok := e.speakerAddHost()
	if p == nil || !ok {
		e.prompt = editorPrompt{}
		return errors.New("this map cannot hold a speaker record")
	}
	name := strings.TrimSpace(value)
	if name == "" {
		return errors.New("the speaker needs a name")
	}
	unit, back := p.unit, p.back
	e.prompt = editorPrompt{}
	if !e.hasPathDialog() {
		e.speakerPictureFailed(back, nil)
		return nil
	}
	e.askChoice(pathChoice{
		request:   e.speakerPictureRequest(host),
		chosen:    func(path string) { e.speakerPictureChosen(host, unit, name, path, back) },
		cancelled: back,
		failed:    func(err error) { e.speakerPictureFailed(back, err) },
	})
	return nil
}

// speakerPictureChosen writes the record from the picture the window
// answered. A refused picture brings the name prompt back with its text and
// the refusal, so another picture can be chosen.
func (e *MapEditor) speakerPictureChosen(host InspectionSpeakerRecords, unit int, name, path string, back func()) {
	id, err := host.AddSpeaker(unit, name, path)
	if err != nil {
		e.askSpeaker(unit, name, back)
		e.message = err.Error()
		return
	}
	e.pick, e.pickNote, e.pickWhy = pickNone, "", ""
	e.refreshSpeakerChoices()
	e.paramPicked(targetSpeaker, pickedValue{number: id})
}

// speakerPictureFailed goes back to the speaker pick when the host has no
// window to show; a nil err is no dialog at all. The picture has no other
// route: the record is not written.
func (e *MapEditor) speakerPictureFailed(back func(), err error) {
	back()
	if err == nil {
		err = pathdialog.ErrUnavailable
	}
	e.message = fmt.Sprintf(e.word("editor.speaker.add.nodialog"), err)
}

// refreshSpeakerChoices asks the host again for the speaker lists of the open
// message's cards, so a record just added is listed by name as an installed
// record is. Each card keeps the value it holds.
func (e *MapEditor) refreshSpeakerChoices() {
	host, ok := e.doc.Edits.(InspectionMessages)
	if !ok {
		return
	}
	o := &e.obj
	for i, part := range o.messageParts {
		f := 2*i + 1
		if f >= len(o.fields) {
			break
		}
		fresh := host.MessageFields(e.messagePage, part.Source, int(part.Heroes))
		if len(fresh) != 2 {
			continue
		}
		fresh[1].Value = o.fields[f].Value
		o.fields[f] = fresh[1]
	}
}

package ui

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"againrom/pkg/namefold"
	"againrom/pkg/pathdialog"
)

type editorPathReply struct {
	result pathdialog.Result
	err    error
}

// pathChoice is one choice of a path through the host's dialog: what the
// window lists and what the editor does with its answer. failed takes the
// reason the host could not answer.
type pathChoice struct {
	request   pathdialog.Request
	chosen    func(path string)
	cancelled func()
	failed    func(err error)
}

type editorPathDialog struct {
	ask     pathdialog.Func
	baseDir string
	// mapsFolder answers the profile folder the editor saves maps in, where
	// the Open window starts; nil or an empty answer keeps the usual start.
	mapsFolder func() string
	choice     pathChoice
	pending    <-chan editorPathReply
	rearm      bool
}

func (e *MapEditor) SetPathDialog(ask pathdialog.Func, baseDir string) {
	if baseDir != "" {
		if absolute, err := filepath.Abs(baseDir); err == nil {
			baseDir = absolute
		}
	}
	e.pathDialog.ask, e.pathDialog.baseDir = ask, baseDir
}

// SetMapsFolder names the host's maps folder, which the Open window starts in.
func (e *MapEditor) SetMapsFolder(folder func() string) { e.pathDialog.mapsFolder = folder }

// pathQuestion is the path question a row asks the host's window, as data:
// the window, its start-folder rule ("" keeps the base folder), what the
// answer and a cancel do, and the typed fallback (err nil: no window).
type pathQuestion struct {
	pathdialog.Question
	start     func() string
	chosen    func(path string)
	cancelled func()
	fallback  func(err error)
}

// questionRequest is the window q asks for, its start rule applied.
func (e *MapEditor) questionRequest(q pathQuestion) pathdialog.Request {
	start := ""
	if q.start != nil {
		start = q.start()
	}
	if start == "" {
		start = e.pathDialog.baseDir
	}
	return q.Question.Request(start)
}

// askPathQuestion is the one opener of the host's window.
func (e *MapEditor) askPathQuestion(q pathQuestion) {
	if !e.hasPathDialog() {
		q.fallback(nil)
		return
	}
	cancelled := q.cancelled
	if cancelled == nil {
		cancelled = func() {}
	}
	e.askChoice(pathChoice{request: e.questionRequest(q), chosen: q.chosen, cancelled: cancelled, failed: q.fallback})
}

// mapsStart is the host's maps folder; "" when the host names none.
func (e *MapEditor) mapsStart() string {
	if e.pathDialog.mapsFolder == nil {
		return ""
	}
	return e.pathDialog.mapsFolder()
}

// mapQuestion is File > Open's and Save As's question; the fallback is the
// typed Open or Save line.
func (e *MapEditor) mapQuestion(kind pathdialog.Kind) pathQuestion {
	q := pathQuestion{
		Question: pathdialog.Question{
			Kind: kind, Title: "Open map",
			Filters: []pathdialog.Filter{{Name: "ALM maps", Extensions: []string{"alm", "almx"}}, {Name: "All files", Extensions: []string{"*"}}},
		},
		start:     func() string { return e.mapStart(kind) },
		chosen:    func(path string) { e.mapChosen(kind, path) },
		cancelled: func() { e.cancelPopup() },
		fallback:  func(err error) { e.pathFallback(kind, err) },
	}
	// An extended map is saved as an .almx (DIV-4603).
	ext := ".alm"
	if e.doc != nil && e.doc.Script != nil && e.doc.Script.Extended {
		ext = ".almx"
	}
	if kind == pathdialog.SaveFile {
		q.Title = "Save map as"
		q.DefaultName = "new-map" + ext
	}
	if source := e.mapSource(); source != "" && kind == pathdialog.SaveFile {
		q.DefaultName = filepath.Base(filepath.FromSlash(source))
		if x := filepath.Ext(q.DefaultName); ext == ".almx" && !namefold.HostEqual(x, ext) {
			q.DefaultName = strings.TrimSuffix(q.DefaultName, x) + ext
		}
	}
	return q
}

// mapSource is the open map's source; "" for none or a new map.
func (e *MapEditor) mapSource() string {
	if e.doc == nil || e.doc.Source == "" || ParseMapSource(e.doc.Source).Kind == SourceNew {
		return ""
	}
	return e.doc.Source
}

// mapStart is the open map's folder; Open starts in the host's maps folder
// and Save As in its save folder where they name one.
func (e *MapEditor) mapStart(kind pathdialog.Kind) string {
	start := ""
	if source := e.mapSource(); source != "" {
		if src := ParseMapSource(source); src.Kind != SourceScenario {
			if src.Kind == SourceLoose {
				source = filepath.Join(e.pathDialog.baseDir, filepath.FromSlash(src.Path))
			}
			if absolute, err := filepath.Abs(source); err == nil {
				start = filepath.Dir(absolute)
			}
		}
	}
	if kind == pathdialog.OpenFile {
		if dir := e.mapsStart(); dir != "" {
			start = dir
		}
	}
	if kind == pathdialog.SaveFile && e.editable() {
		if f, ok := Seam[InspectionSaveFolder](e.doc.Edits); ok {
			if dir := f.SaveFolder(); dir != "" {
				start = dir
			}
		}
	}
	return start
}

// hasPathDialog reports that the host can show a dialog now.
func (e *MapEditor) hasPathDialog() bool {
	return e.pathDialog.ask != nil && e.mcp.call == nil
}

// mapChosen opens or saves the map at the path the window answered.
func (e *MapEditor) mapChosen(kind pathdialog.Kind, path string) {
	e.leave(leaveDrag | leaveReport | leavePick)
	var err error
	if kind == pathdialog.SaveFile {
		err = e.saveAs(path)
	} else {
		err = e.Open(path)
	}
	// A save that asks to fold the script keeps its question.
	if err != nil && e.prompt.kind != promptScriptForm {
		message := e.message
		e.cancelPopup(popPrompt)
		e.message = message
	}
}

func (e *MapEditor) askPath(kind pathdialog.Kind) {
	e.rememberCancel()
	e.askPathQuestion(e.mapQuestion(kind))
}

// askChoice shows choice's window on a worker; PollPathDialog applies the
// answer. askPathQuestion is its one caller.
func (e *MapEditor) askChoice(c pathChoice) {
	e.cancelRecordDrag()
	e.brush.endStroke()
	e.down, e.moved, e.cancelledDrag = false, false, false
	done := make(chan editorPathReply, 1)
	e.pathDialog.choice, e.pathDialog.pending = c, done
	ask, r := e.pathDialog.ask, c.request
	go func() {
		result, err := ask(r)
		done <- editorPathReply{result, err}
	}()
}

func (e *MapEditor) pathFallback(kind pathdialog.Kind, err error) {
	prompt := promptOpen
	if kind == pathdialog.SaveFile {
		prompt = promptSave
	}
	e.leave(leaveDrag | leaveReport | leavePick)
	e.openPrompt(editorPrompt{kind: prompt})
	// The typed path is the fallback; the note says why the dialog did not
	// open.
	switch {
	case errors.Is(err, pathdialog.ErrUnavailable):
		e.message = e.plainWord("editor.path.nodialog")
	case err != nil:
		e.message = fmt.Sprintf(e.plainWord("editor.path.dialogfailed"), err)
	}
}

// Only Update applies a worker's result; the completion frame consumes no input.
func (e *MapEditor) PollPathDialog() bool {
	if e.pathDialog.pending == nil {
		return false
	}
	select {
	case reply := <-e.pathDialog.pending:
		choice := e.pathDialog.choice
		e.pathDialog.pending, e.pathDialog.rearm = nil, true
		e.pathDialog.choice = pathChoice{}
		switch {
		case reply.err != nil:
			choice.failed(reply.err)
		case reply.result.Cancelled:
			choice.cancelled()
		case reply.result.Path == "":
			choice.failed(fmt.Errorf("no path returned"))
		default:
			choice.chosen(reply.result.Path)
		}
	default:
	}
	return true
}

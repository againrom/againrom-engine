package ui

import (
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

func (e *MapEditor) pathRequest(kind pathdialog.Kind) pathdialog.Request {
	r := pathdialog.Request{
		Kind: kind, Title: "Open map", StartDir: e.pathDialog.baseDir,
		Filters: []pathdialog.Filter{{Name: "ALM maps", Extensions: []string{"alm", "almx"}}, {Name: "All files", Extensions: []string{"*"}}},
	}
	// An extended map is saved as an .almx (DIV-4603).
	ext := ".alm"
	if e.doc != nil && e.doc.Script != nil && e.doc.Script.Extended {
		ext = ".almx"
	}
	if kind == pathdialog.SaveFile {
		r.Title = "Save map as"
		r.DefaultName = "new-map" + ext
	}
	if e.doc != nil {
		source := e.doc.Source
		src := ParseMapSource(source)
		if source != "" && src.Kind != SourceNew {
			if kind == pathdialog.SaveFile {
				r.DefaultName = filepath.Base(filepath.FromSlash(source))
				if x := filepath.Ext(r.DefaultName); ext == ".almx" && !namefold.HostEqual(x, ext) {
					r.DefaultName = strings.TrimSuffix(r.DefaultName, x) + ext
				}
			}
			if src.Kind != SourceScenario {
				if src.Kind == SourceLoose {
					source = filepath.Join(e.pathDialog.baseDir, filepath.FromSlash(src.Path))
				}
				if absolute, err := filepath.Abs(source); err == nil {
					r.StartDir = filepath.Dir(absolute)
				}
			}
		}
	}
	if kind == pathdialog.OpenFile && e.pathDialog.mapsFolder != nil {
		if dir := e.pathDialog.mapsFolder(); dir != "" {
			r.StartDir = dir
		}
	}
	if kind == pathdialog.SaveFile && e.editable() {
		if f, ok := e.doc.Edits.(InspectionSaveFolder); ok {
			if dir := f.SaveFolder(); dir != "" {
				r.StartDir = dir
			}
		}
	}
	return r
}

// hasPathDialog reports that the host can show a dialog now.
func (e *MapEditor) hasPathDialog() bool {
	return e.pathDialog.ask != nil && e.mcp.call == nil
}

// mapChoice is the choice of a map to open or save: Open and Save As are
// built by pathRequest and end in Open or saveAs.
func (e *MapEditor) mapChoice(kind pathdialog.Kind) pathChoice {
	return pathChoice{
		request: e.pathRequest(kind),
		chosen: func(path string) {
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
		},
		cancelled: func() { e.cancelPopup() },
		failed:    func(err error) { e.pathFallback(kind, err) },
	}
}

func (e *MapEditor) askPath(kind pathdialog.Kind) {
	e.rememberCancel()
	if !e.hasPathDialog() {
		e.pathFallback(kind, nil)
		return
	}
	e.askChoice(e.mapChoice(kind))
}

// askChoice shows choice's window on a worker; PollPathDialog applies the
// answer. The caller has checked hasPathDialog.
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
	if err != nil {
		e.message = fmt.Sprintf("System dialog unavailable: %v. Type a path below.", err)
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

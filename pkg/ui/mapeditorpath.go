package ui

import (
	"fmt"
	"path/filepath"
	"strings"

	"againrom/pkg/pathdialog"
)

type editorPathReply struct {
	result pathdialog.Result
	err    error
}

type editorPathDialog struct {
	ask     pathdialog.Func
	baseDir string
	kind    pathdialog.Kind
	pending <-chan editorPathReply
	rearm   bool
}

func (e *MapEditor) SetPathDialog(ask pathdialog.Func, baseDir string) {
	if baseDir != "" {
		if absolute, err := filepath.Abs(baseDir); err == nil {
			baseDir = absolute
		}
	}
	e.pathDialog.ask, e.pathDialog.baseDir = ask, baseDir
}

func (e *MapEditor) pathRequest(kind pathdialog.Kind) pathdialog.Request {
	r := pathdialog.Request{
		Kind: kind, Title: "Open map", StartDir: e.pathDialog.baseDir,
		Filters: []pathdialog.Filter{{Name: "ALM maps", Extensions: []string{"alm"}}, {Name: "All files", Extensions: []string{"*"}}},
	}
	if kind == pathdialog.SaveFile {
		r.Title = "Save map as"
		r.DefaultName = "new-map.alm"
	}
	if e.doc != nil {
		source := e.doc.Source
		if source != "" && !strings.HasPrefix(source, "new/") {
			if kind == pathdialog.SaveFile {
				r.DefaultName = filepath.Base(filepath.FromSlash(source))
			}
			if !strings.HasPrefix(source, "scenario/") {
				if strings.HasPrefix(source, "loose/") {
					source = filepath.Join(e.pathDialog.baseDir, filepath.FromSlash(strings.TrimPrefix(source, "loose/")))
				}
				if absolute, err := filepath.Abs(source); err == nil {
					r.StartDir = filepath.Dir(absolute)
				}
			}
		}
	}
	return r
}

func (e *MapEditor) askPath(kind pathdialog.Kind) {
	if e.pathDialog.ask == nil {
		e.pathFallback(kind, nil)
		return
	}
	r := e.pathRequest(kind)
	e.cancelRecordDrag()
	e.brush.endStroke()
	e.down, e.moved, e.cancelledDrag = false, false, false
	done := make(chan editorPathReply, 1)
	e.pathDialog.kind, e.pathDialog.pending = kind, done
	ask := e.pathDialog.ask
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
		kind := e.pathDialog.kind
		e.pathDialog.pending, e.pathDialog.rearm = nil, true
		switch {
		case reply.err != nil:
			e.pathFallback(kind, reply.err)
		case reply.result.Cancelled:
		case reply.result.Path == "":
			e.pathFallback(kind, fmt.Errorf("no path returned"))
		default:
			e.leave(leaveDrag | leaveReport | leavePick)
			if kind == pathdialog.SaveFile {
				_ = e.saveAs(reply.result.Path)
			} else {
				_ = e.Open(reply.result.Path)
			}
		}
	default:
	}
	return true
}

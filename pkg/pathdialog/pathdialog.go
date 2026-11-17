// Package pathdialog selects file or folder paths.
package pathdialog

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"unicode/utf8"
)

type Kind uint8

const (
	OpenFile Kind = iota
	SaveFile
	ChooseFolder
)

type Filter struct {
	Name       string
	Extensions []string
}

type Request struct {
	Kind        Kind
	Title       string
	Filters     []Filter
	StartDir    string
	DefaultName string
}

// NewRequest builds a Request from its parts; hosts ask through Question.Request.
func NewRequest(kind Kind, title, startDir string, filters ...Filter) Request {
	return Request{Kind: kind, Title: title, StartDir: startDir, Filters: filters}
}

// Question is a window's question as data: its kind, title, filters and
// default name. A host builds one per question and asks its Request.
type Question struct {
	Kind        Kind
	Title       string
	Filters     []Filter
	DefaultName string
}

// Request is the window q asks, started in start ("" lets Show choose).
func (q Question) Request(start string) Request {
	r := NewRequest(q.Kind, q.Title, start, q.Filters...)
	r.DefaultName = q.DefaultName
	return r
}

type Result struct {
	Path      string
	Cancelled bool
}

type Func func(Request) (Result, error)

var ErrUnavailable = errors.New("system path dialog unavailable")

var dialogMu sync.Mutex

// Show waits for a modal dialog. Call it from a worker, keeping the window's
// event loop running until the result is received.
func Show(request Request) (Result, error) {
	if err := validate(request); err != nil {
		return Result{}, err
	}
	request.StartDir = startFolder(request.StartDir)
	dialogMu.Lock()
	defer dialogMu.Unlock()
	return show(request)
}

// startFolder is the nearest folder at or above dir that exists, or "" when
// none does: a start folder not yet made opens the window one level up, not
// a failure.
func startFolder(dir string) string {
	if dir == "" {
		return ""
	}
	for {
		if info, err := os.Stat(dir); err == nil && info.IsDir() {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir || parent == "." {
			return ""
		}
		dir = parent
	}
}

func unavailable(Request) (Result, error) {
	return Result{}, ErrUnavailable
}

func validate(request Request) error {
	if request.Kind > ChooseFolder {
		return fmt.Errorf("unknown path dialog kind %d", request.Kind)
	}
	for _, s := range []string{request.Title, request.StartDir, request.DefaultName} {
		if strings.ContainsRune(s, 0) || !utf8.ValidString(s) {
			return errors.New("path dialog text is not valid UTF-8 or contains NUL")
		}
	}
	for _, filter := range request.Filters {
		if filter.Name == "" || strings.ContainsRune(filter.Name, 0) || !utf8.ValidString(filter.Name) || len(filter.Extensions) == 0 {
			return errors.New("path dialog filter requires a name and extensions")
		}
		for _, extension := range filter.Extensions {
			if extension == "*" && len(filter.Extensions) == 1 {
				continue
			}
			if extension == "" || strings.ContainsAny(extension, `.\/:*?;`) || strings.ContainsRune(extension, 0) || !utf8.ValidString(extension) {
				return fmt.Errorf("invalid path dialog extension %q", extension)
			}
		}
	}
	return nil
}

func selected(path string, cancelled bool) (Result, error) {
	if cancelled {
		return Result{Cancelled: true}, nil
	}
	if path == "" {
		return Result{}, errors.New("system path dialog returned an empty path")
	}
	return Result{Path: path}, nil
}

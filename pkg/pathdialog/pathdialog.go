// Package pathdialog asks the host system for one file or folder path.
package pathdialog

import (
	"errors"
	"fmt"
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
	dialogMu.Lock()
	defer dialogMu.Unlock()
	return show(request)
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

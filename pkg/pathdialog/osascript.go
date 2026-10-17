package pathdialog

import (
	"fmt"
	"os"
	"strings"
)

type osascriptRunner func(args []string) (stdout, stderr string, err error)

// A fixed first argument keeps request text from being read as an option.
const (
	osascriptLead      = "againrom"
	osascriptTitleItem = "(item 2 of argv)"
	osascriptStartItem = "((item 3 of argv) as POSIX file)"
	osascriptNameItem  = "(item 4 of argv)"
	osascriptTypeItems = "(items 5 thru -1 of argv)"
)

func osascriptCommand(request Request, startDir bool) []string {
	command := "choose folder"
	switch request.Kind {
	case OpenFile:
		command = "choose file"
	case SaveFile:
		command = "choose file name"
	}
	if request.Title != "" {
		command += " with prompt " + osascriptTitleItem
	}
	if startDir {
		command += " default location " + osascriptStartItem
	}
	if request.Kind == SaveFile && request.DefaultName != "" {
		command += " default name " + osascriptNameItem
	}
	types := osascriptTypes(request)
	if len(types) > 0 {
		command += " of type " + osascriptTypeItems
	}
	script := strings.Join([]string{
		"on run argv",
		"activate",
		"set chosen to " + command,
		"return POSIX path of chosen",
		"end run",
	}, "\n")
	start := ""
	if startDir {
		start = request.StartDir
	}
	args := []string{"-e", script, osascriptLead, request.Title, start, request.DefaultName}
	return append(args, types...)
}

func osascriptTypes(request Request) []string {
	if request.Kind != OpenFile {
		return nil
	}
	var types []string
	for _, filter := range request.Filters {
		for _, extension := range filter.Extensions {
			if extension == "*" {
				return nil
			}
			types = append(types, extension)
		}
	}
	return types
}

func osascriptResult(kind Kind, stdout, stderr string, err error) (Result, error) {
	if err != nil {
		if strings.Contains(stderr, "(-128)") {
			return selected("", true)
		}
		message := strings.TrimSpace(stderr)
		if message == "" {
			return Result{}, fmt.Errorf("system path dialog: %w", err)
		}
		return Result{}, fmt.Errorf("system path dialog: %w: %s", err, message)
	}
	path := strings.TrimSuffix(stdout, "\n")
	if kind == ChooseFolder && len(path) > 1 {
		path = strings.TrimSuffix(path, "/")
	}
	return selected(path, false)
}

func showOsascript(request Request, run osascriptRunner) (Result, error) {
	startDir := false
	if request.StartDir != "" {
		info, err := os.Stat(request.StartDir)
		startDir = err == nil && info.IsDir()
	}
	stdout, stderr, err := run(osascriptCommand(request, startDir))
	return osascriptResult(request.Kind, stdout, stderr, err)
}

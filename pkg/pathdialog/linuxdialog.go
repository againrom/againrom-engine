package pathdialog

import (
	"path"
	"strings"
)

type linuxDialogRunner func(string, []string) (string, int, error)

func linuxDialogArgs(tool string, request Request) []string {
	start := request.StartDir
	if start != "" {
		start = strings.TrimRight(start, "/") + "/"
	}
	if request.Kind == SaveFile {
		start += request.DefaultName
	}
	if tool == "zenity" {
		args := []string{"--file-selection", "--title=" + request.Title}
		if start != "" {
			args = append(args, "--filename="+start)
		}
		switch request.Kind {
		case ChooseFolder:
			args = append(args, "--directory")
		case SaveFile:
			args = append(args, "--save", "--confirm-overwrite")
		}
		if request.Kind != ChooseFolder {
			for _, filter := range request.Filters {
				args = append(args, "--file-filter="+filter.Name+" | "+linuxDialogPatterns(filter))
			}
		}
		return args
	}
	if start == "" {
		start = "."
	}
	if !path.IsAbs(start) {
		start = "./" + start
	}
	kind := "--getopenfilename"
	switch request.Kind {
	case ChooseFolder:
		kind = "--getexistingdirectory"
	case SaveFile:
		kind = "--getsavefilename"
	}
	args := []string{"--title", request.Title, kind, start}
	if request.Kind != ChooseFolder && len(request.Filters) != 0 {
		var filters []string
		for _, filter := range request.Filters {
			filters = append(filters, linuxDialogPatterns(filter)+"|"+filter.Name)
		}
		args = append(args, strings.Join(filters, "\n"))
	}
	return args
}

func linuxDialogPatterns(filter Filter) string {
	patterns := make([]string, len(filter.Extensions))
	for i, extension := range filter.Extensions {
		patterns[i] = "*." + extension
		if extension == "*" {
			patterns[i] = "*"
		}
	}
	return strings.Join(patterns, " ")
}

func showLinuxDialog(request Request, lookup func(string) (string, error), run linuxDialogRunner) (Result, error) {
	for _, tool := range []string{"zenity", "kdialog"} {
		executable, err := lookup(tool)
		if err != nil {
			continue
		}
		stdout, code, err := run(executable, linuxDialogArgs(tool, request))
		if code == 1 {
			return selected("", true)
		}
		if err == nil && code == 0 {
			return selected(strings.TrimSuffix(stdout, "\n"), false)
		}
	}
	return unavailable(request)
}

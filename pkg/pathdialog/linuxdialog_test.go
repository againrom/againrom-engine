package pathdialog

import (
	"errors"
	"reflect"
	"testing"
)

func TestLinuxDialogChoice(t *testing.T) {
	failed := errors.New("cannot run")
	for _, tc := range []struct {
		name        string
		missing     map[string]bool
		failFirst   bool
		failAll     bool
		cancel      bool
		wantCalls   []string
		unavailable bool
	}{
		{name: "zenity first", wantCalls: []string{"zenity"}},
		{name: "kdialog second", missing: map[string]bool{"zenity": true}, wantCalls: []string{"kdialog"}},
		{name: "neither installed", missing: map[string]bool{"zenity": true, "kdialog": true}, unavailable: true},
		{name: "zenity cannot run", failFirst: true, wantCalls: []string{"zenity", "kdialog"}},
		{name: "neither runs", failAll: true, wantCalls: []string{"zenity", "kdialog"}, unavailable: true},
		{name: "cancel zenity", cancel: true, wantCalls: []string{"zenity"}},
		{name: "cancel kdialog", missing: map[string]bool{"zenity": true}, cancel: true, wantCalls: []string{"kdialog"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var calls []string
			result, err := showLinuxDialog(Request{Kind: ChooseFolder}, func(tool string) (string, error) {
				if tc.missing[tool] {
					return "", failed
				}
				return tool, nil
			}, func(tool string, args []string) (string, int, error) {
				calls = append(calls, tool)
				if tc.cancel {
					return "", 1, failed
				}
				if tc.failAll || tc.failFirst && tool == "zenity" {
					return "", -1, failed
				}
				return "/games/Рage ü with spaces \n", 0, nil
			})
			if !reflect.DeepEqual(calls, tc.wantCalls) {
				t.Fatalf("calls %v, want %v", calls, tc.wantCalls)
			}
			if tc.unavailable {
				if !errors.Is(err, ErrUnavailable) || result != (Result{}) {
					t.Fatalf("unavailable: %+v, %v", result, err)
				}
			} else if err != nil || tc.cancel && result != (Result{Cancelled: true}) || !tc.cancel && result.Path != "/games/Рage ü with spaces " {
				t.Fatalf("result %+v, %v", result, err)
			}
		})
	}
}

func TestLinuxDialogArguments(t *testing.T) {
	for _, tc := range []struct {
		tool string
		kind Kind
		want []string
	}{
		{"zenity", ChooseFolder, []string{"--file-selection", "--title=Choose 'путь'; $(false)", "--filename=/games/Игра ü/", "--directory"}},
		{"zenity", OpenFile, []string{"--file-selection", "--title=Choose 'путь'; $(false)", "--filename=/games/Игра ü/", "--file-filter=Saves | *.sav *.alm"}},
		{"zenity", SaveFile, []string{"--file-selection", "--title=Choose 'путь'; $(false)", "--filename=/games/Игра ü/new save.sav", "--save", "--confirm-overwrite", "--file-filter=Saves | *.sav *.alm"}},
		{"kdialog", ChooseFolder, []string{"--title", "Choose 'путь'; $(false)", "--getexistingdirectory", "/games/Игра ü/"}},
		{"kdialog", OpenFile, []string{"--title", "Choose 'путь'; $(false)", "--getopenfilename", "/games/Игра ü/", "*.sav *.alm|Saves"}},
		{"kdialog", SaveFile, []string{"--title", "Choose 'путь'; $(false)", "--getsavefilename", "/games/Игра ü/new save.sav", "*.sav *.alm|Saves"}},
	} {
		request := Request{Kind: tc.kind, Title: "Choose 'путь'; $(false)", StartDir: "/games/Игра ü", DefaultName: "new save.sav", Filters: []Filter{{Name: "Saves", Extensions: []string{"sav", "alm"}}}}
		if got := linuxDialogArgs(tc.tool, request); !reflect.DeepEqual(got, tc.want) {
			t.Errorf("%s %d: %q, want %q", tc.tool, tc.kind, got, tc.want)
		}
	}
	if got := linuxDialogPatterns(Filter{Extensions: []string{"*"}}); got != "*" {
		t.Fatalf("all files: %q", got)
	}
}

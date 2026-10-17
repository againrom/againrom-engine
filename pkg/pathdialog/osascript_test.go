package pathdialog

import (
	"errors"
	"fmt"
	"os/exec"
	"reflect"
	"strings"
	"testing"
)

const oddText = `-e "Карта" 'x' \ & quit end run`

func splicedText(script string, request Request) error {
	texts := []string{request.Title, request.StartDir, request.DefaultName}
	for _, filter := range request.Filters {
		texts = append(texts, filter.Extensions...)
	}
	for _, text := range texts {
		if text != "" && strings.Contains(script, text) {
			return fmt.Errorf("script contains request text %q", text)
		}
	}
	return nil
}

func TestOsascriptCommand(t *testing.T) {
	filters := []Filter{{Name: "ALM maps", Extensions: []string{"alm", "almx"}}}
	for _, test := range []struct {
		name     string
		request  Request
		startDir bool
		line     string
		argv     []string
	}{
		{"folder", Request{Kind: ChooseFolder, Title: oddText, StartDir: oddText + "dir"}, true,
			"set chosen to choose folder with prompt (item 2 of argv) default location ((item 3 of argv) as POSIX file)",
			[]string{"againrom", oddText, oddText + "dir", ""}},
		{"folder without start", Request{Kind: ChooseFolder, Title: "Заголовок", StartDir: "missing"}, false,
			"set chosen to choose folder with prompt (item 2 of argv)",
			[]string{"againrom", "Заголовок", "", ""}},
		{"bare folder", Request{Kind: ChooseFolder}, false,
			"set chosen to choose folder",
			[]string{"againrom", "", "", ""}},
		{"open", Request{Kind: OpenFile, Title: oddText, StartDir: "/maps", Filters: filters}, true,
			"set chosen to choose file with prompt (item 2 of argv) default location ((item 3 of argv) as POSIX file) of type (items 5 thru -1 of argv)",
			[]string{"againrom", oddText, "/maps", "", "alm", "almx"}},
		{"open all files", Request{Kind: OpenFile, Title: "Заголовок", Filters: append(filters, Filter{Name: "All", Extensions: []string{"*"}})}, false,
			"set chosen to choose file with prompt (item 2 of argv)",
			[]string{"againrom", "Заголовок", "", ""}},
		{"save", Request{Kind: SaveFile, Title: "Заголовок", StartDir: "/maps", DefaultName: oddText + ".alm", Filters: filters}, true,
			"set chosen to choose file name with prompt (item 2 of argv) default location ((item 3 of argv) as POSIX file) default name (item 4 of argv)",
			[]string{"againrom", "Заголовок", "/maps", oddText + ".alm"}},
	} {
		t.Run(test.name, func(t *testing.T) {
			args := osascriptCommand(test.request, test.startDir)
			if len(args) < 2 || args[0] != "-e" {
				t.Fatalf("args = %q", args)
			}
			want := "on run argv\nactivate\n" + test.line + "\nreturn POSIX path of chosen\nend run"
			if args[1] != want {
				t.Errorf("script =\n%s\nwant\n%s", args[1], want)
			}
			if !reflect.DeepEqual(args[2:], test.argv) {
				t.Errorf("argv = %q, want %q", args[2:], test.argv)
			}
			if err := splicedText(args[1], test.request); err != nil {
				t.Error(err)
			}
		})
	}
}

func TestSplicedScriptFails(t *testing.T) {
	request := Request{Kind: ChooseFolder, Title: oddText}
	script := osascriptCommand(request, false)[1]
	spliced := strings.Replace(script, osascriptTitleItem, `"`+request.Title+`"`, 1)
	if spliced == script || splicedText(spliced, request) == nil {
		t.Fatal("a spliced title passed the argv check")
	}
}

func TestOsascriptResult(t *testing.T) {
	exit := errors.New("exit status 1")
	odd := "/Users/тест/Мои \"карты\" 'x'/new map.alm"
	for _, test := range []struct {
		name   string
		kind   Kind
		stdout string
		stderr string
		err    error
		want   Result
		fail   string
	}{
		{"file", OpenFile, odd + "\n", "", nil, Result{Path: odd}, ""},
		{"save", SaveFile, odd + "\n", "", nil, Result{Path: odd}, ""},
		{"folder", ChooseFolder, "/Users/тест/Rage of Mages/\n", "", nil, Result{Path: "/Users/тест/Rage of Mages"}, ""},
		{"root folder", ChooseFolder, "/\n", "", nil, Result{Path: "/"}, ""},
		{"inner newline kept", OpenFile, "/a\nb\n", "", nil, Result{Path: "/a\nb"}, ""},
		{"cancel", ChooseFolder, "", "0:85: execution error: User canceled. (-128)\n", exit, Result{Cancelled: true}, ""},
		{"failure", OpenFile, "", "0:12: execution error: File not found. (-43)\n", exit, Result{}, "File not found. (-43)"},
		{"failure without stderr", OpenFile, "", "", exit, Result{}, "exit status 1"},
		{"empty path", OpenFile, "\n", "", nil, Result{}, "empty path"},
	} {
		t.Run(test.name, func(t *testing.T) {
			got, err := osascriptResult(test.kind, test.stdout, test.stderr, test.err)
			if got != test.want {
				t.Errorf("result = %+v, want %+v", got, test.want)
			}
			switch {
			case test.fail == "" && err != nil:
				t.Errorf("error = %v", err)
			case test.fail != "" && (err == nil || !strings.Contains(err.Error(), test.fail)):
				t.Errorf("error = %v, want %q", err, test.fail)
			case test.err != nil && test.fail != "" && !errors.Is(err, test.err):
				t.Errorf("error %v does not wrap %v", err, test.err)
			}
		})
	}
}

func TestShowOsascript(t *testing.T) {
	dir := t.TempDir()
	var got []string
	run := func(args []string) (string, string, error) {
		got = args
		return "/picked/\n", "", nil
	}
	result, err := showOsascript(Request{Kind: ChooseFolder, Title: "Заголовок", StartDir: dir}, run)
	if err != nil || result != (Result{Path: "/picked"}) {
		t.Fatalf("result = %+v, %v", result, err)
	}
	if len(got) != 6 || got[4] != dir || !strings.Contains(got[1], osascriptStartItem) {
		t.Fatalf("existing start folder not passed: %q", got)
	}
	_, _ = showOsascript(Request{Kind: ChooseFolder, StartDir: dir + "/missing"}, run)
	if got[4] != "" || strings.Contains(got[1], osascriptStartItem) {
		t.Fatalf("missing start folder passed: %q", got)
	}
	missing := func([]string) (string, string, error) {
		return "", "", fmt.Errorf("%w: %v", ErrUnavailable, exec.ErrNotFound)
	}
	if _, err := showOsascript(Request{Kind: OpenFile}, missing); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("missing osascript = %v", err)
	}
}

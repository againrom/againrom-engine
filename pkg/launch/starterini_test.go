package launch

import (
	"os"
	"path/filepath"
	"testing"
)

// The selected base is the one named last, else the first, and last stays as
// it was when there is no base.
func TestSelectedKeyAndBase(t *testing.T) {
	bases := []NamedBase{{Key: "en", Path: "/g/en"}, {Key: "ru", Path: "/g/ru"}}
	for _, c := range []struct {
		name      string
		bases     []NamedBase
		last      string
		wantKey   string
		wantPath  string
		wantFound bool
	}{
		{"named base", bases, "ru", "ru", "/g/ru", true},
		{"named base in another case", bases, "RU", "RU", "/g/ru", true},
		{"unknown name falls to the first", bases, "de", "en", "/g/en", true},
		{"empty name falls to the first", bases, "", "en", "/g/en", true},
		{"no base keeps last", nil, "ru", "ru", "", false},
		{"no base and no name", nil, "", "", "", false},
	} {
		if got := SelectedKey(c.bases, c.last); got != c.wantKey {
			t.Errorf("%s: SelectedKey = %q, want %q", c.name, got, c.wantKey)
		}
		path, ok := SelectedBase(c.bases, c.last)
		if path != c.wantPath || ok != c.wantFound {
			t.Errorf("%s: SelectedBase = %q, %v, want %q, %v", c.name, path, ok, c.wantPath, c.wantFound)
		}
	}
}

// The mods folder has one resolver: the folder named, else mods beside the
// programs; a program the starter did not start reads the name from the
// starter.ini beside it.
func TestModsFolderHasOneResolver(t *testing.T) {
	dir := t.TempDir()
	beside := filepath.Join(dir, "mods")
	if got := ModsFolder("", dir); got != beside {
		t.Errorf("ModsFolder with no name = %q, want %q", got, beside)
	}
	if got := ModsFolder("  /shared/mods  ", dir); got != "/shared/mods" {
		t.Errorf("ModsFolder with a name = %q", got)
	}
	if got := StarterModsDir(dir); got != "" {
		t.Errorf("StarterModsDir with no starter.ini = %q", got)
	}
	if got := ProgramModsFolder("", dir); got != beside {
		t.Errorf("ProgramModsFolder with no starter.ini = %q, want %q", got, beside)
	}
	named := filepath.Join(t.TempDir(), "named")
	text := "[" + ModsSection + "]\n" + ModsDirKey + " = " + named + "\nenabled = a,b\n"
	if err := os.WriteFile(StarterINI(dir), []byte(text), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := StarterModsDir(dir); got != named {
		t.Errorf("StarterModsDir = %q, want %q", got, named)
	}
	if got := ProgramModsFolder("", dir); got != named {
		t.Errorf("ProgramModsFolder = %q, want the starter's %q", got, named)
	}
	flag := filepath.Join(t.TempDir(), "flag")
	if got := ProgramModsFolder(flag, dir); got != flag {
		t.Errorf("ProgramModsFolder with a flag = %q, want %q", got, flag)
	}
	// A blank mods folder adds no argument to the editor's start.
	args, err := Editor.Build(One(func(st Setting) string { return st.Load("") }))
	if err != nil {
		t.Fatal(err)
	}
	for _, a := range args {
		if a == "-"+EditorModsDir.Flag {
			t.Errorf("a blank mods folder built %q", args)
		}
	}
}

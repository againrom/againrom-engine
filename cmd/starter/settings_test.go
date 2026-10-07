package main

import (
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"againrom/pkg/ini"
)

const sampleINI = "; my starter\r\n[starter]\r\nagainrom = D:\\game\\againrom.exe\r\nclose-on-play = yes\r\nlast-base = ru\r\nfuture-key = keep\r\n\r\n[bases]\r\nen = C:\\games\\en\r\nru = C:\\games\\ru\r\nthis line is malformed\r\n\r\n[mods]\r\ndir = D:\\mods\r\nenabled = b, a ,\r\n\r\n[options]\r\nsound = OFF\r\nvolume = 40\r\nmovies = no\r\nvideo = 8x\r\nmarkers = 1\r\nsaves = D:\\saves\r\nmission = 12\r\nskill = axe\r\nextra = -foo \"a b\"\r\n"

func TestLoadSettingsReadsEveryKey(t *testing.T) {
	f, problems := ini.Parse([]byte(sampleINI))
	if len(problems) != 1 || problems[0].Line != 11 {
		t.Fatalf("problems %v", problems)
	}
	got := loadSettings(f)
	want := settings{
		Againrom: `D:\game\againrom.exe`, CloseOnPlay: true, LastBase: "ru",
		Bases:   []base{{"en", `C:\games\en`}, {"ru", `C:\games\ru`}},
		ModsDir: `D:\mods`, Enabled: []string{"b", "a"},
		Sound: soundOff, Volume: "40", Movies: false, Video: video8x, Markers: true,
		Saves: `D:\saves`, Mission: "12", Skill: "axe", Extra: `-foo "a b"`,
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %+v\nwant %+v", got, want)
	}
}

func TestMissingIniGivesDefaults(t *testing.T) {
	f, problems := ini.Parse(nil)
	if len(problems) != 0 {
		t.Fatal(problems)
	}
	if got := loadSettings(f); !reflect.DeepEqual(got, defaultSettings()) {
		t.Fatalf("%+v", got)
	}
	if d := defaultSettings(); d.Sound != soundDefault || !d.Movies || d.Video != videoNormal {
		t.Fatalf("%+v", d)
	}
}

func TestStoreKeepsUnknownKeysCommentsAndMalformedLines(t *testing.T) {
	f, _ := ini.Parse([]byte(sampleINI))
	s := loadSettings(f)
	s.Volume = "75"
	s.removeBase("en")
	s.toggleMod("c")
	s.store(f)
	out := string(f.Bytes())
	for _, keep := range []string{"; my starter\r\n", "future-key = keep\r\n", "this line is malformed\r\n", "ru = C:\\games\\ru\r\n"} {
		if !strings.Contains(out, keep) {
			t.Errorf("%q lost:\n%s", keep, out)
		}
	}
	if strings.Contains(out, "en = C:") {
		t.Errorf("removed base survived:\n%s", out)
	}
	again := loadSettings(mustParse(t, out))
	if again.Volume != "75" || !reflect.DeepEqual(again.Enabled, []string{"b", "a", "c"}) || again.Sound != soundOff {
		t.Fatalf("%+v", again)
	}
	// storing the loaded settings of a stored file changes nothing
	second := mustParse(t, out)
	loadSettings(second).store(second)
	if string(second.Bytes()) != out {
		t.Fatalf("a second store changed the file:\n%s\n---\n%s", second.Bytes(), out)
	}
}

func mustParse(t *testing.T, text string) *ini.File {
	t.Helper()
	f, _ := ini.Parse([]byte(text))
	return f
}

func TestStoreWritesAFreshFileInDocumentedOrder(t *testing.T) {
	f, _ := ini.Parse(nil)
	s := defaultSettings()
	s.addBase(`C:\Games\EN`)
	s.LastBase = "en"
	s.store(f)
	want := "[starter]\nagainrom =\nclose-on-play = false\nlast-base = en\n[bases]\nen = C:\\Games\\EN\n[mods]\ndir =\nenabled =\naccept-unmarked = false\n[options]\nsound = default\nvolume =\nmovies = true\nvideo = normal\nmarkers = false\nsaves =\nmission =\npicker = false\nskill =\nextra =\n"
	if got := string(f.Bytes()); got != want {
		t.Fatalf("got:\n%s\nwant:\n%s", got, want)
	}
}

func TestLaunchArgsExactArgv(t *testing.T) {
	root, mods := `C:\g\en`, `C:\m`
	cases := []struct {
		name string
		mod  func(*settings)
		want []string
	}{
		{"defaults add nothing but the base", func(*settings) {}, []string{"-assets", root}},
		{"mods", func(s *settings) { s.Enabled = []string{"b", "a"} }, []string{"-assets", root, "-mods", "b,a", "-mods-dir", mods}},
		{"sound off", func(s *settings) { s.Sound = soundOff }, []string{"-assets", root, "-sound=false"}},
		{"sound on", func(s *settings) { s.Sound = soundOn }, []string{"-assets", root, "-sound=true"}},
		{"volume", func(s *settings) { s.Volume = " 0 " }, []string{"-assets", root, "-volume", "0"}},
		{"no movies", func(s *settings) { s.Movies = false }, []string{"-assets", root, "-movies=false"}},
		{"4x", func(s *settings) { s.Video = video4x }, []string{"-assets", root, "-4x"}},
		{"8x", func(s *settings) { s.Video = video8x }, []string{"-assets", root, "-8x"}},
		{"markers", func(s *settings) { s.Markers = true }, []string{"-assets", root, "-markers"}},
		{"saves", func(s *settings) { s.Saves = `C:\my saves` }, []string{"-assets", root, "-saves", `C:\my saves`}},
		{"mission", func(s *settings) { s.Mission = "7" }, []string{"-assets", root, "-mission", "7"}},
		{"picker", func(s *settings) { s.Picker = true }, []string{"-assets", root, "-picker"}},
		{"skill", func(s *settings) { s.Skill = "pike" }, []string{"-assets", root, "-skill", "pike"}},
		{"extra", func(s *settings) { s.Extra = `-headless "C:\a b\s.json"` }, []string{"-assets", root, "-headless", `C:\a b\s.json`}},
		{"accept unmarked with mods", func(s *settings) { s.Enabled, s.AcceptUnmarked = []string{"a"}, true }, []string{"-assets", root, "-mods", "a", "-mods-dir", mods, "-mods-accept-unmarked"}},
		{"accept unmarked without mods", func(s *settings) { s.AcceptUnmarked = true }, []string{"-assets", root}},
		{"everything in order", func(s *settings) {
			s.Enabled = []string{"a"}
			s.Sound, s.Volume, s.Movies, s.Video, s.Markers = soundOn, "100", false, video8x, true
			s.Saves, s.Mission, s.Skill, s.Extra = "sv", "3", "axe", "-x"
		}, []string{"-assets", root, "-mods", "a", "-mods-dir", mods, "-sound=true", "-volume", "100", "-movies=false", "-8x", "-markers", "-saves", "sv", "-mission", "3", "-skill", "axe", "-x"}},
	}
	for _, c := range cases {
		s := defaultSettings()
		c.mod(&s)
		got, err := s.launchArgs(root, mods)
		if err != nil || !reflect.DeepEqual(got, c.want) {
			t.Errorf("%s: got %q (%v)\nwant %q", c.name, got, err, c.want)
		}
	}
}

func TestLaunchArgsRefusals(t *testing.T) {
	for name, mod := range map[string]func(*settings){
		"volume text":     func(s *settings) { s.Volume = "loud" },
		"volume high":     func(s *settings) { s.Volume = "101" },
		"volume negative": func(s *settings) { s.Volume = "-1" },
		"mission zero":    func(s *settings) { s.Mission = "0" },
		"mission text":    func(s *settings) { s.Mission = "x" },
		"picker+mission":  func(s *settings) { s.Picker, s.Mission = true, "3" },
		"open quote":      func(s *settings) { s.Extra = `-x "a` },
	} {
		s := defaultSettings()
		mod(&s)
		if got, err := s.launchArgs("r", "m"); err == nil {
			t.Errorf("%s accepted: %q", name, got)
		}
	}
	if _, err := defaultSettings().launchArgs("  ", "m"); err == nil {
		t.Error("no base accepted")
	}
}

func TestSplitArgs(t *testing.T) {
	for in, want := range map[string][]string{
		``:                  nil,
		`  `:                nil,
		`a  b	c`:            {"a", "b", "c"},
		`-x "a b" c`:        {"-x", "a b", "c"},
		`"" x`:              {"", "x"},
		`pre"fix ed"post y`: {"prefix edpost", "y"},
	} {
		got, err := splitArgs(in)
		if err != nil || !reflect.DeepEqual(got, want) {
			t.Errorf("%q: %q %v, want %q", in, got, err, want)
		}
	}
}

func TestBaseKeysAreUniqueAndSafe(t *testing.T) {
	var s settings
	if k := s.addBase(`C:\games\EN`); k != "en" {
		t.Fatal(k)
	}
	if k := s.addBase(`D:\other\en`); k != "en-2" {
		t.Fatal(k)
	}
	if k := s.addBase(`C:\games\en`); k != "en" {
		t.Fatalf("same path added twice under %s", k)
	}
	if k := s.addBase(`C:\Игры\ROM1 demo`); k != "rom1-demo" {
		t.Fatal(k)
	}
	if k := baseKey(`\\`); k == "" {
		t.Fatal("empty key")
	}
}

func TestToggleModKeepsLoadOrder(t *testing.T) {
	var s settings
	s.toggleMod("a")
	s.toggleMod("b")
	s.toggleMod("c")
	s.toggleMod("b")
	s.toggleMod("b")
	if !reflect.DeepEqual(s.Enabled, []string{"a", "c", "b"}) {
		t.Fatal(s.Enabled)
	}
}

func TestPathsBesideTheStarter(t *testing.T) {
	dir := filepath.Join("c", "builds", "current")
	if got := defaultINIPath(dir); got != filepath.Join(dir, "starter.ini") {
		t.Fatal(got)
	}
	var s settings
	if got := s.modsDir(dir); got != filepath.Join(dir, "mods") {
		t.Fatal(got)
	}
	s.ModsDir = "elsewhere"
	if got := s.modsDir(dir); got != "elsewhere" {
		t.Fatal(got)
	}
	self := filepath.Join(dir, "starter.exe")
	if got := s.gameExecutable(self, "windows"); got != filepath.Join(dir, "againrom.exe") {
		t.Fatal(got)
	}
	if got := s.gameExecutable(self, "linux"); got != filepath.Join(dir, "againrom") {
		t.Fatal(got)
	}
	s.Againrom = "x/y.exe"
	if got := s.gameExecutable(self, "windows"); got != "x/y.exe" {
		t.Fatal(got)
	}
}

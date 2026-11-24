package mod

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseDialogueReadsTheFixtureToItsNodesAndReplies(t *testing.T) {
	data, err := os.ReadFile("testdata/dialogue/elder.toml")
	if err != nil {
		t.Fatal(err)
	}
	d, err := ParseDialogue("elder.toml", data)
	if err != nil {
		t.Fatal(err)
	}
	if len(d.Nodes) != 3 || d.Nodes[0].ID != 1 || len(d.Nodes[0].Replies) != 4 || len(d.Nodes[1].Replies) != 0 {
		t.Fatalf("nodes %+v", d.Nodes)
	}
	n := d.Nodes[0]
	if n.Text.Words("en") != "Strangers! Will you drive the raiders out?" || !strings.HasPrefix(n.Text.Words("ru"), "Чужаки") {
		t.Errorf("texts %v", n.Text)
	}
	a, b, c, e := n.Replies[0], n.Replies[1], n.Replies[2], n.Replies[3]
	if a.Next != 2 || len(a.Do) != 1 || a.Do[0] != 12 || a.Once || a.Disabled {
		t.Errorf("reply 1 %+v", a)
	}
	if !b.Once || b.Mark != 140 || b.Next != 3 {
		t.Errorf("reply 2 %+v", b)
	}
	if !c.Disabled || len(c.When) != 1 || c.When[0] != (DialogueCondition{Left: 3, Right: 5}) || len(c.Do) != 1 || c.Next != 0 {
		t.Errorf("reply 3 %+v", c)
	}
	if e.Disabled || len(e.When) != 1 || len(e.Do) != 1 || e.Do[0] != 14 {
		t.Errorf("reply 4 %+v", e)
	}
	if got, ok := d.Node(3); !ok || got.Replies[0].Next != 1 {
		t.Errorf("node 3 %+v", got)
	}
	if (DialogueText{"ru": "x"}).Words("en") != "x" {
		t.Error("a text missing in the game's language did not fall to the other one")
	}
}

func TestParseDialogueRefusalNamesFileAndLine(t *testing.T) {
	head := "[[node]]\nid = 1\nen = \"a\"\n"
	for name, c := range map[string]struct {
		src  string
		line int
		msg  string
	}{
		"bad toml":           {head + "[[reply]]\nen = \n", 5, "en"},
		"unknown key":        {head + "[[reply]]\nen = \"b\"\nlook = 1\n", 6, "unknown key"},
		"unknown table":      {head + "[thing]\n", 4, "unknown table"},
		"reply before node":  {"[[reply]]\nen = \"b\"\n", 1, "before any [[node]]"},
		"key outside":        {"id = 1\n", 1, "outside"},
		"no text":            {"[[node]]\nid = 1\n", 1, "no text"},
		"no id":              {"[[node]]\nen = \"a\"\n", 1, "no id"},
		"duplicate id":       {head + "[[node]]\nid = 1\nen = \"b\"\n", 4, "twice"},
		"next names nothing": {head + "[[reply]]\nen = \"b\"\nnext = 9\n", 4, "names no node"},
		"once without mark":  {head + "[[reply]]\nen = \"b\"\nonce = true\n", 4, "needs mark"},
		"bad unavailable":    {head + "[[reply]]\nen = \"b\"\nunavailable = \"grey\"\n", 6, "hidden"},
		"bad when":           {head + "[[reply]]\nen = \"b\"\nwhen = [{ left = 1 }]\n", 6, "left and right"},
		"bad cmp":            {head + "[[reply]]\nen = \"b\"\nwhen = [{ left = 1, right = 2, cmp = 9 }]\n", 6, "cmp"},
		"bad do":             {head + "[[reply]]\nen = \"b\"\ndo = [0]\n", 6, "do holds"},
		"five replies":       {head + strings.Repeat("[[reply]]\nen = \"b\"\n", 5), 12, "more than 4"},
		"empty":              {"# nothing\n", 0, "no [[node]]"},
	} {
		_, err := ParseDialogue("elder.toml", []byte(c.src))
		var fe *ItemFileError
		if !errors.As(err, &fe) {
			t.Errorf("%s: %v", name, err)
			continue
		}
		if fe.File != "elder.toml" || fe.Line != c.line || !strings.Contains(fe.Msg, c.msg) {
			t.Errorf("%s: %q, want line %d and %q", name, err, c.line, c.msg)
		}
	}
}

func dialogueMission(t *testing.T, files map[string]string) (Mission, string) {
	t.Helper()
	dir := t.TempDir()
	for name, data := range files {
		p := filepath.Join(dir, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(p), 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(data), 0600); err != nil {
			t.Fatal(err)
		}
	}
	src := "[[mission]]\nkey = \"m\"\nkind = \"replace\"\nnumber = 20\nmap = \"maps/town.almx\"\n"
	d, err := ParseMissions("m", MissionsFile, []byte(src), nil, "en")
	if err != nil {
		t.Fatal(err)
	}
	return d.Missions[0], dir
}

func TestMissionReadsTheDialogueFilesBesideItsMap(t *testing.T) {
	fixture, err := os.ReadFile("testdata/dialogue/elder.toml")
	if err != nil {
		t.Fatal(err)
	}
	m, dir := dialogueMission(t, map[string]string{
		"maps/town.almx":                  "map",
		"maps/town.dialogs/elder.toml":    string(fixture),
		"maps/town.dialogs/notes.txt":     "ignored",
		"maps/town.dialogs/Bad Name.toml": "ignored",
	})
	if err := m.ReadFiles(dir); err != nil || len(m.DialogueFiles) != 1 || string(m.DialogueFiles["elder"]) != string(fixture) {
		t.Fatalf("dialogue files %v: %v", m.DialogueFiles, err)
	}
	if !(MissionPlace{Map: "maps/town.almx"}).Companion("maps/town.dialogs/elder.toml") || !IsMissionFile("maps/town.dialogs/elder.toml") {
		t.Error("the dialogue file is not one of the map's files")
	}
	m, dir = dialogueMission(t, map[string]string{"maps/town.almx": "map"})
	if err := m.ReadFiles(dir); err != nil || m.DialogueFiles != nil {
		t.Errorf("a map with no dialogues: %v %v", m.DialogueFiles, err)
	}
}

func TestMissionRefusesAMalformedDialogueFileByFileAndLine(t *testing.T) {
	m, dir := dialogueMission(t, map[string]string{
		"maps/town.almx":               "map",
		"maps/town.dialogs/elder.toml": "[[node]]\nid = 1\nen = \"a\"\n[[reply]]\nen = \"b\"\nnext = 9\n",
	})
	err := m.ReadFiles(dir)
	if err == nil || !strings.Contains(err.Error(), "maps/town.dialogs/elder.toml:4: next = 9 names no node") {
		t.Fatalf("got %v", err)
	}
}

func TestDialogueFilesLimitIsTheCallersAndZeroKeepsABigDraft(t *testing.T) {
	big := make([]byte, MaxFileBytes+1)
	read := func(string) ([]byte, bool) { return big, true }
	entries := []string{"a.toml", "sub/b.toml", "Bad.toml", "c.txt"}
	if got := DialogueFiles(entries, MaxFileBytes, read); len(got) != 0 {
		t.Errorf("over the limit kept: %d files", len(got))
	}
	got := DialogueFiles(entries, 0, read)
	if len(got) != 1 || len(got["a"]) != len(big) {
		t.Errorf("no limit: %d files, want only a", len(got))
	}
}

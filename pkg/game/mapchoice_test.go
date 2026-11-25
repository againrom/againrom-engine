package game

import (
	"bytes"
	"encoding/binary"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"againrom/internal/synth"
	"againrom/pkg/base"
	"againrom/pkg/formats/alm"
	"againrom/pkg/render/menu"
	"againrom/pkg/ui"
	"againrom/pkg/vfs"
	"againrom/pkg/words"
)

func synthFirstMap(t *testing.T, name string, word70, word74 uint32) []byte {
	t.Helper()
	b := synth.ALM(synth.ALMOptions{Width: 16, Height: 16, Name: name})
	// Record 0 payload begins immediately after the 20-byte file header and
	// 20-byte type-0 record header.
	payload := alm.FileHeaderSize + alm.RecordHeaderSize
	binary.LittleEndian.PutUint32(b[payload+alm.MetaWord70:], word70)
	binary.LittleEndian.PutUint32(b[payload+alm.MetaWord74:], word74)
	return b
}

func synthSecondMap(t *testing.T, name string, word84, level88 uint32) []byte {
	t.Helper()
	meta := make([]byte, 660)
	binary.LittleEndian.PutUint32(meta[0x00:], 16)
	binary.LittleEndian.PutUint32(meta[0x04:], 16)
	copy(meta[0x44:], []byte(name))
	binary.LittleEndian.PutUint32(meta[0x84:], word84)
	binary.LittleEndian.PutUint32(meta[alm.SecondMetaLevel:], level88)

	tiles := make([]byte, 2*16*16)
	alts := make([]byte, 16*16)

	var buf bytes.Buffer
	var fh [alm.FileHeaderSize]byte
	binary.LittleEndian.PutUint32(fh[alm.FileMagicAt:], 0x0052374D)
	binary.LittleEndian.PutUint32(fh[alm.FileHeaderLength:], alm.FileHeaderSize)
	binary.LittleEndian.PutUint32(fh[alm.FileRecordCount:], 3)
	binary.LittleEndian.PutUint32(fh[alm.FileFormatVersion:], 1600)
	buf.Write(fh[:])

	writeRec := func(typeID, declared uint32, payload []byte) {
		var rh [alm.RecordHeaderSize]byte
		binary.LittleEndian.PutUint32(rh[alm.RecordTag:], 5)
		binary.LittleEndian.PutUint32(rh[alm.RecordHeaderLength:], alm.RecordHeaderSize)
		binary.LittleEndian.PutUint32(rh[alm.RecordPayloadSize:], declared)
		binary.LittleEndian.PutUint32(rh[alm.RecordTypeID:], typeID)
		buf.Write(rh[:])
		buf.Write(payload)
	}
	writeRec(alm.TypeMeta, 644, meta)
	writeRec(alm.TypeTiles, uint32(len(tiles)), tiles)
	writeRec(alm.TypeAltitude, uint32(len(alts)), alts)
	return buf.Bytes()
}

func snapshotDir(t *testing.T, dir string) map[string][]byte {
	t.Helper()
	out := map[string][]byte{}
	ents, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range ents {
		if e.IsDir() {
			continue
		}
		b, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			t.Fatal(err)
		}
		out[e.Name()] = b
	}
	return out
}

func syntheticFrontForMapChoice(t *testing.T, profileID, lang, dir string, extra []MapEntry) (*FrontEnd, *ui.App) {
	t.Helper()
	profile, ok := base.Find(profileID)
	if !ok {
		t.Fatalf("missing profile %s", profileID)
	}
	loose, err := vfs.Open(nil, []string{dir})
	if err != nil {
		t.Fatal(err)
	}
	dm, err := DirMaps(dir, loose)
	if err != nil {
		t.Fatal(err)
	}
	maps := append(BuildMapList(dm, nil), extra...)
	f := &FrontEnd{
		InstallResources: InstallResources{
			Archives: &Archives{Base: base.Match{Profile: profile}},
			Maps:     maps,
		},
	}
	f.Words.Engine = words.For(lang)
	app := ui.NewApp("map choice", &menu.Assets{}, MapPickerRows(maps, 0, ""), nil)
	app.Layout(640, 480)
	return f, app
}

func TestMultiplayerMapChoiceFiltersFirstGameAndCountsBadFiles(t *testing.T) {
	dir := t.TempDir()
	write := func(rel string, data []byte) {
		p := filepath.Join(dir, rel)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, data, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	write("single0.alm", synthFirstMap(t, "Solo Zero", 0, 1))
	write("single1.alm", synthFirstMap(t, "Solo One", 1, 2))
	write("multi2.alm", synthFirstMap(t, "Duel", 2, 3))
	write("multi4.alm", synthFirstMap(t, "Arena", 4, 5))
	write("corrupt.alm", []byte("not an alm map"))
	trunc := synthFirstMap(t, "Cut", 4, 2)
	write("truncated.alm", trunc[:len(trunc)-40])
	write("sub/nested.alm", synthFirstMap(t, "Nested", 4, 1))
	write("readme.txt", []byte("ignored"))

	before := snapshotDir(t, dir)

	kept, failed, err := MultiplayerMapsFromDir(dir, MultiplayerMapRuleOf(base.GameROM1))
	if err != nil {
		t.Fatal(err)
	}
	if failed != 2 {
		t.Fatalf("failed = %d, want 2 bad maps counted", failed)
	}
	var sources []string
	for _, e := range kept {
		sources = append(sources, e.Source)
	}
	if want := []string{"multi2.alm", "multi4.alm"}; !reflect.DeepEqual(sources, want) {
		t.Fatalf("first-game multiplayer maps = %v, want %v", sources, want)
	}

	// Campaign archive and mission rows mixed into f.Maps are never listed.
	extra := []MapEntry{
		{Source: "10.alm", FromArchive: true, Word70: 4},
		{Source: "10.alm", Mission: 10, Word70: 4},
	}
	f, app := syntheticFrontForMapChoice(t, base.ROM1EN, "english", dir, extra)
	var chosen ui.MapChoiceResult
	if bad := f.OpenMultiplayerMapChoice(app, ui.ScreenMenu, func(res ui.MapChoiceResult) { chosen = res }); bad != 2 {
		t.Fatalf("OpenMultiplayerMapChoice failed = %d, want 2", bad)
	}
	rows := app.HeadlessRows()
	if len(rows) != 2 || rows[0].Text != "multi2.alm - Duel" || rows[1].Text != "multi4.alm - Arena" {
		t.Fatalf("first-game rows = %+v, want Duel and Arena without level text", rows)
	}
	if err := app.HeadlessKey("down"); err != nil {
		t.Fatal(err)
	}
	if err := app.HeadlessKey("enter"); err != nil {
		t.Fatal(err)
	}
	if app.Screen() != ui.ScreenMenu || !chosen.OK || chosen.Chosen != "multi4.alm" {
		t.Fatalf("Ok returned screen=%v res=%+v, want ScreenMenu multi4.alm", app.Screen(), chosen)
	}

	chosen = ui.MapChoiceResult{Chosen: "stale", OK: true}
	f.OpenMultiplayerMapChoice(app, ui.ScreenMenu, func(res ui.MapChoiceResult) { chosen = res })
	if err := app.HeadlessKey("escape"); err != nil {
		t.Fatal(err)
	}
	if app.Screen() != ui.ScreenMenu || chosen.OK || chosen.Chosen != "" {
		t.Fatalf("Cancel returned screen=%v res=%+v, want ScreenMenu none", app.Screen(), chosen)
	}
	if after := snapshotDir(t, dir); !reflect.DeepEqual(before, after) {
		t.Fatal("map choice modified files in the map directory")
	}
}

func TestMultiplayerMapChoiceListsSecondGameWithLevelAndCountsBadFiles(t *testing.T) {
	dir := t.TempDir()
	write := func(name string, data []byte) {
		if err := os.WriteFile(filepath.Join(dir, name), data, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	write("low0.alm", synthSecondMap(t, "Valley", 0, 1))
	write("low1.alm", synthSecondMap(t, "Fort", 1, 3))
	write("high4.alm", synthSecondMap(t, "Citadel", 4, 5))
	write("broken.alm", []byte("bad bytes"))

	before := snapshotDir(t, dir)

	kept, failed, err := MultiplayerMapsFromDir(dir, MultiplayerMapRuleOf(base.GameROM2))
	if err != nil {
		t.Fatal(err)
	}
	if failed != 1 {
		t.Fatalf("failed = %d, want 1", failed)
	}
	var sources []string
	for _, e := range kept {
		sources = append(sources, e.Source)
	}
	if want := []string{"high4.alm", "low0.alm", "low1.alm"}; !reflect.DeepEqual(sources, want) {
		t.Fatalf("second-game multiplayer maps = %v, want %v", sources, want)
	}

	for _, tc := range []struct {
		id, lang string
		wantRows []string
	}{
		{
			base.ROM2EN, "english",
			[]string{"high4.alm - Citadel  Level 5", "low0.alm - Valley  Level 1", "low1.alm - Fort  Level 3"},
		},
		{
			base.ROM2RU, "russian",
			[]string{"high4.alm - Citadel  Уровень 5", "low0.alm - Valley  Уровень 1", "low1.alm - Fort  Уровень 3"},
		},
	} {
		t.Run(tc.id, func(t *testing.T) {
			f, app := syntheticFrontForMapChoice(t, tc.id, tc.lang, dir, nil)
			var chosen ui.MapChoiceResult
			if bad := f.OpenMultiplayerMapChoice(app, ui.ScreenMenu, func(res ui.MapChoiceResult) { chosen = res }); bad != 1 {
				t.Fatalf("failed = %d, want 1", bad)
			}
			rows := app.HeadlessRows()
			if len(rows) != len(tc.wantRows) {
				t.Fatalf("rows = %+v, want %v", rows, tc.wantRows)
			}
			for i, want := range tc.wantRows {
				if rows[i].Text != want {
					t.Errorf("row %d = %q, want %q", i, rows[i].Text, want)
				}
			}
			if err := app.HeadlessKey("down"); err != nil {
				t.Fatal(err)
			}
			if err := app.HeadlessKey("enter"); err != nil {
				t.Fatal(err)
			}
			if app.Screen() != ui.ScreenMenu || !chosen.OK || chosen.Chosen != "low0.alm" {
				t.Fatalf("Ok returned screen=%v res=%+v, want low0.alm", app.Screen(), chosen)
			}

			chosen = ui.MapChoiceResult{Chosen: "stale", OK: true}
			f.OpenMultiplayerMapChoice(app, ui.ScreenMenu, func(res ui.MapChoiceResult) { chosen = res })
			if err := app.HeadlessKey("escape"); err != nil {
				t.Fatal(err)
			}
			if app.Screen() != ui.ScreenMenu || chosen.OK || chosen.Chosen != "" {
				t.Fatalf("Cancel returned screen=%v res=%+v, want none", app.Screen(), chosen)
			}
		})
	}
	if after := snapshotDir(t, dir); !reflect.DeepEqual(before, after) {
		t.Fatal("map choice modified files in the map directory")
	}
}

func TestMultiplayerMapChoiceWordsFollowInstallLanguage(t *testing.T) {
	for _, tc := range []struct {
		id, lang                string
		title, ok, cancel, none string
	}{
		{base.ROM1EN, "english", "Select Map", "OK", "Cancel", "No maps"},
		{base.ROM1RU, "russian", "Выбор карты", "ОК", "Отмена", "Нет карт"},
		{base.ROM2EN, "english", "Select Map", "OK", "Cancel", "No maps"},
		{base.ROM2RU, "russian", "Выбор карты", "ОК", "Отмена", "Нет карт"},
	} {
		profile, ok := base.Find(tc.id)
		if !ok {
			t.Fatalf("missing profile %s", tc.id)
		}
		f := &FrontEnd{InstallResources: InstallResources{Archives: &Archives{Base: base.Match{Profile: profile}}}}
		f.Words.Engine = words.For(tc.lang)
		want := ui.MapChoiceWords{
			Title:  encodeEngineWord(tc.title, 0),
			OK:     encodeEngineWord(tc.ok, 0),
			Cancel: encodeEngineWord(tc.cancel, 0),
			Empty:  encodeEngineWord(tc.none, 0),
		}
		if got := f.mapChoiceWords(); got != want {
			t.Errorf("%s: words = %+v, want %+v", tc.id, got, want)
		}
		if level := f.Words.Engine.Text("mapchoice.level"); !strings.Contains(level, "%d") {
			t.Errorf("%s: mapchoice.level = %q, want %%d placeholder", tc.id, level)
		}
	}
}

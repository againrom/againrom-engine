package mod

import (
	"fmt"
	"testing"

	"againrom/pkg/town"
)

func TestTOMLStringRoundTrip(t *testing.T) {
	cases := []struct {
		name   string
		val    string
		refuse bool
	}{
		{"quote", `a"b`, false},
		{"backslash", `a\b`, false},
		{"newline", "a\nb", false},
		{"cr", "a\rb", false},
		{"tab", "a\tb", false},
		{"ctrl01", "a\x01b", false},
		{"del7f", "a\x7fb", false},
		{"bad_utf8", "a\x87\xa0b", true},
		{"non_ascii", "aЖb", false},
	}

	look := func(string) (string, bool) { return "-", true }

	writers := []struct {
		name string
		run  func(v string) (string, error)
	}{
		{
			name: "writeString",
			run: func(v string) (string, error) {
				q, err := writeString(v)
				if err != nil {
					return "", err
				}
				got, rest, err := readString(q)
				if err != nil {
					return "", err
				}
				if rest != "" {
					return "", fmt.Errorf("trailing %q", rest)
				}
				return got, nil
			},
		},
		{
			name: "WriteManifest",
			run: func(v string) (string, error) {
				out, err := WriteManifest(nil, map[string]any{"id": "sample", "title": v, "version": "1"})
				if err != nil {
					return "", err
				}
				m, err := ParseManifest(out)
				if err != nil {
					return "", err
				}
				return m.Title, nil
			},
		},
		{
			name: "writeStrings",
			run: func(v string) (string, error) {
				rel := StringsFile("en")
				out, err := writeStrings(rel, nil, Data{Strings: map[string]Strings{"en": {"k": v}}})
				if err != nil {
					return "", err
				}
				got, err := ParseStrings(rel, out)
				if err != nil {
					return "", err
				}
				return got["k"], nil
			},
		},
		{
			name: "WriteDialogue",
			run: func(v string) (string, error) {
				d := Dialogue{File: "d.toml", Nodes: []DialogueNode{{ID: 1, Text: DialogueText{"en": v}}}}
				out, err := WriteDialogue(d)
				if err != nil {
					return "", err
				}
				got, err := ParseDialogue("d.toml", out)
				if err != nil {
					return "", err
				}
				return got.Nodes[0].Text["en"], nil
			},
		},
		{
			name: "writeGlobals",
			run: func(v string) (string, error) {
				d := Data{Globals: GlobalData{Globals: []Global{{ID: FirstGlobal, Name: "flag", Notes: map[string]string{"en": v}}}}}
				out, err := writeGlobals(GlobalsFile, nil, d)
				if err != nil {
					return "", err
				}
				got, err := ParseGlobals("", GlobalsFile, out)
				if err != nil {
					return "", err
				}
				return got.Globals[0].Notes["en"], nil
			},
		},
		{
			name: "writeScreens",
			run: func(v string) (string, error) {
				d := Data{Screens: ScreenData{Screens: []Screen{{Key: "info", Kind: ScreenInfo, TitleKey: v, MenuKey: "m", Main: true, TextKeys: []string{"t"}}}}}
				out, err := writeScreens(ScreensFile, nil, d)
				if err != nil {
					return "", err
				}
				got, err := ParseScreens("", ScreensFile, out, look, "en")
				if err != nil {
					return "", err
				}
				return got.Screens[0].TitleKey, nil
			},
		},
		{
			name: "writeSettings",
			run: func(v string) (string, error) {
				d := Data{Settings: SettingData{Settings: []Setting{{Key: "opt", Kind: KindBool, Labels: map[string]string{"en": v}}}}}
				out, err := writeSettings(SettingsName, nil, d)
				if err != nil {
					return "", err
				}
				got, err := ParseSettings(out)
				if err != nil {
					return "", err
				}
				return got[0].Labels["en"], nil
			},
		},
		{
			name: "writeSpeakers",
			run: func(v string) (string, error) {
				d := Data{Speakers: SpeakerData{Speakers: []Speaker{NewSpeaker(FirstModSpeaker, v, "p.png")}}}
				out, err := writeSpeakers(SpeakersFile, nil, d)
				if err != nil {
					return "", err
				}
				got, err := ParseSpeakers("", SpeakersFile, out)
				if err != nil {
					return "", err
				}
				return got.Speakers[0].Name, nil
			},
		},
		{
			name: "MoveSpeakerMaps",
			run: func(v string) (string, error) {
				src := []byte("[[speaker]]\nid = 20001\nname = \"s\"\npicture = \"p.png\"\nmap = \"old.alm\"\nunit = 1\n")
				out, _, err := MoveSpeakerMaps(src, []FileMove{{From: "old.alm", To: v}})
				if err != nil {
					return "", err
				}
				got, err := ParseSpeakers("", SpeakersFile, out)
				if err != nil {
					return "", err
				}
				return got.Speakers[0].Map, nil
			},
		},
		{
			name: "RewriteValues",
			run: func(v string) (string, error) {
				out, err := RewriteValues([]byte("k = \"old\"\n"), []ValueEdit{{Line: 1, Value: v}})
				if err != nil {
					return "", err
				}
				tables, err := parseTOMLWith(out, tomlOptions{})
				if err != nil {
					return "", err
				}
				return tables[0].pairs[0].val.str, nil
			},
		},
		{
			name: "WriteTowns",
			run: func(v string) (string, error) {
				out, err := WriteTowns(nil, []town.Entry{{ID: 1, Name: v}})
				if err != nil {
					return "", err
				}
				recs, _, err := ParseTowns("", TownsFile, out)
				if err != nil {
					return "", err
				}
				return recs[0].Entry.Name, nil
			},
		},
		{
			name: "WriteTownData",
			run: func(v string) (string, error) {
				after := TownData{Records: []TownRecord{{Entry: town.Entry{ID: 1, Name: v}, Fields: []PatchField{{Name: "name"}}}}}
				out, err := WriteTownData(nil, TownData{}, after)
				if err != nil {
					return "", err
				}
				recs, _, err := ParseTowns("", TownsFile, out)
				if err != nil {
					return "", err
				}
				return recs[0].Entry.Name, nil
			},
		},
		{
			name: "writeCharacters",
			run: func(v string) (string, error) {
				d := Data{Characters: CharacterData{Rows: []CharacterRow{{Target: v, Face: 1}}}}
				out, err := writeCharacters(CharactersFile, nil, d)
				if err != nil {
					return "", err
				}
				got, err := ParseCharacters("", CharactersFile, out, look, "en")
				if err != nil {
					return "", err
				}
				return got.Rows[0].Target, nil
			},
		},
		{
			name: "writeWeaponBodies",
			run: func(v string) (string, error) {
				d := Data{Bodies: BodyData{Weapons: []WeaponBody{{Weapon: v, Body: "hero"}}}}
				out, err := writeWeaponBodies(WeaponBodiesFiles[0], nil, d)
				if err != nil {
					return "", err
				}
				got, err := ParseWeaponBodies("", WeaponBodiesFiles[0], out)
				if err != nil {
					return "", err
				}
				return got[0].Weapon, nil
			},
		},
		{
			name: "writeItems",
			run: func(v string) (string, error) {
				d := Data{Items: ItemData{Rows: []ItemRow{{Key: "cap", NameKey: v, Slot: "head", StandIn: "head"}}}}
				out, err := writeItems(ItemsFile, nil, d)
				if err != nil {
					return "", err
				}
				got, err := ParseItems("", ItemsFile, out, look, "en")
				if err != nil {
					return "", err
				}
				return got.Rows[0].NameKey, nil
			},
		},
		{
			name: "WriteMissions",
			run: func(v string) (string, error) {
				m := Mission{Key: "m1", Number: 50, Map: "maps/m1.alm", After: 1, Party: []GuestMember{{Character: v}}}
				out, err := WriteMissions(nil, nil, []Mission{m})
				if err != nil {
					return "", err
				}
				tables, err := parseTOMLWith(out, tomlOptions{arrays: true})
				if err != nil {
					return "", err
				}
				for _, p := range tables[1].pairs {
					if p.key == "party" {
						return p.val.list[0].tbl[0].val.str, nil
					}
				}
				return "", fmt.Errorf("missing party pair")
			},
		},
		{
			name: "writeSpells",
			run: func(v string) (string, error) {
				d := Data{Spells: SpellData{Rows: []SpellRow{{Target: v, Mana: SpellInt{Set: true, Val: 1}}}}}
				out, err := writeSpells(SpellsFile, nil, d)
				if err != nil {
					return "", err
				}
				got, err := ParseSpells("", SpellsFile, out, nil)
				if err != nil {
					return "", err
				}
				return got.Rows[0].Target, nil
			},
		},
		{
			name: "writeTerrain",
			run: func(v string) (string, error) {
				d := Data{Terrain: TerrainData{Groups: []TerrainGroup{{Group: FirstModTerrainGroup, Class: 1, Name: v}}}}
				out, err := writeTerrain(TerrainFile, nil, d)
				if err != nil {
					return "", err
				}
				got, err := ParseTerrain("", TerrainFile, out)
				if err != nil {
					return "", err
				}
				return got.Groups[0].Name, nil
			},
		},
		{
			name: "bodySourceRows",
			run: func(v string) (string, error) {
				rows, err := bodySourceRows([]BodySheet{{Name: "hero", Heroes: v, HeroesLight: "l.png", Width: 1, Height: 1, Directions: 1, Move: BodyAction{Frames: 1}, Attack: BodyAction{Frames: 1}}})
				if err != nil {
					return "", err
				}
				out, err := writeRows(nil, nil, rows, func(k string) string { return k })
				if err != nil {
					return "", err
				}
				tables, err := parseTOMLWith(out, tomlOptions{arrays: true})
				if err != nil {
					return "", err
				}
				for _, p := range tables[1].pairs {
					if p.key == "heroes" {
						return p.val.str, nil
					}
				}
				return "", fmt.Errorf("missing heroes pair")
			},
		},
		{
			name: "companionSourceRows",
			run: func(v string) (string, error) {
				j := CompanionJoin{Companion: 1, Chapter: 1, Key: v, Fields: []PatchField{{Name: "key"}}}
				rows, err := companionSourceRows([]CompanionJoin{j})
				if err != nil {
					return "", err
				}
				out, err := writeRows(nil, nil, rows, func(k string) string { return k })
				if err != nil {
					return "", err
				}
				tables, err := parseTOMLWith(out, tomlOptions{arrays: true})
				if err != nil {
					return "", err
				}
				for _, p := range tables[1].pairs {
					if p.key == "key" {
						return p.val.str, nil
					}
				}
				return "", fmt.Errorf("missing key pair")
			},
		},
	}

	for _, w := range writers {
		t.Run(w.name, func(t *testing.T) {
			for _, c := range cases {
				t.Run(c.name, func(t *testing.T) {
					got, err := w.run(c.val)
					if c.refuse {
						if err == nil {
							t.Fatalf("expected refusal, got %q", got)
						}
						return
					}
					if err != nil {
						t.Fatalf("error: %v", err)
					}
					if got != c.val {
						t.Fatalf("got %q, want %q", got, c.val)
					}
				})
			}
		})
	}

	t.Run("readStringRefusesBadUTF8", func(t *testing.T) {
		for _, raw := range []string{"\"a\x87\xa0b\"", "'a\x87\xa0b'"} {
			if got, _, err := readString(raw); err == nil {
				t.Errorf("readString(%q) accepted %q", raw, got)
			}
		}
	})
}

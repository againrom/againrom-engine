package game

import (
	"bytes"
	"image/color"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"

	"againrom/pkg/data"
	"againrom/pkg/mod"
	"againrom/pkg/ui"
)

// bodiesPage is the editor with W open on a new mod, over an install that
// ships a body list, two bodies under both directories and a weapon table
// whose rows 1 to 3 are Hands, Sword and War Hammer.
func bodiesPage(t *testing.T, lang string) (*MapInspector, *mapInspectionEdits, string) {
	t.Helper()
	x, s, folder := charactersPage(t, lang)
	src, units, weapons, _ := bodyInstall(t)
	x.units = units
	x.table.Weapons = weapons
	x.bodyArt = nil
	list := "unarmed\nclubman\nclubman\n\n"
	x.archives.Containers.Overlay(HeroPictureAddress, func() ([]byte, error) { return []byte(list), nil })
	for _, dir := range []string{data.HeroDirHeroes, data.HeroDirHeroesLight} {
		for _, b := range []data.HeroBody{data.BodyUnarmed, data.BodyClubman} {
			sheet := src[graphicsPrefix+data.HeroSheetPath(dir, b)]
			x.archives.Containers.Overlay(graphicsPrefix+data.HeroSheetPath(dir, b), func() ([]byte, error) { return sheet, nil })
		}
	}
	return x, s, folder
}

func bodyField(t *testing.T, s *mapInspectionEdits, key string) (ui.InspectionField, bool) {
	t.Helper()
	for _, f := range s.WorkspaceFields(ui.ModBodiesScope) {
		if f.Key == key {
			return f, true
		}
	}
	return ui.InspectionField{}, false
}

// bodySteps is the workspace each page's edits left, in order.
var bodySteps = map[*mapInspectionEdits][][]byte{}

func setBody(t *testing.T, s *mapInspectionEdits, key, value string) {
	t.Helper()
	steps := len(s.undoWorkspace)
	if _, ok := bodySteps[s]; !ok {
		bodySteps[s] = [][]byte{workspaceBytes(s.workspace)}
	}
	if _, err := s.SetWorkspaceField(ui.ModBodiesScope, key, value); err != nil {
		t.Fatalf("%s = %q: %v", key, value, err)
	}
	if len(s.undoWorkspace) != steps+1 {
		t.Fatalf("%s made %d undo steps", key, len(s.undoWorkspace)-steps)
	}
	bodySteps[s] = append(bodySteps[s], workspaceBytes(s.workspace))
}

// sheetFile writes a body sheet picture of the starter geometry to a host file.
func sheetFile(t *testing.T, name string, tint color.NRGBA) string {
	t.Helper()
	b := mod.BodySheet{Width: 40, Height: 40, Directions: 8,
		Move: mod.BodyAction{Frames: 6}, Attack: mod.BodyAction{Frames: 5}}
	path := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(path, encodePNG(t, drawBodySheet(b, tint)), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func heldBodies(t *testing.T, s *mapInspectionEdits) mod.BodyData {
	t.Helper()
	d, err := s.heldRows(s.workspace.Project, nil, bodyFiles(s.workspace.Project)...)
	if err != nil {
		t.Fatal(err)
	}
	return d.Bodies
}

// The bodies page adds a body from a picture, sets its frames and timing,
// gives a weapon that body, refuses to remove a body a weapon uses, and
// every edit is one undo step that restores the files.
func TestModBodiesPageAddsBodiesAndGivesWeaponsTheirBody(t *testing.T) {
	for _, lang := range []string{"en", "ru"} {
		t.Run(lang, func(t *testing.T) {
			_, s, folder := bodiesPage(t, lang)
			before := tree(t, folder)
			fields := s.WorkspaceFields(ui.ModBodiesScope)
			if len(fields) != 1 || fields[0].Key != bodiesAddKey || !fields[0].Action || len(fields[0].Pick) == 0 {
				t.Fatalf("the empty page holds %+v", fields)
			}
			setBody(t, s, bodiesAddKey, sheetFile(t, "Hammer Two.png", color.NRGBA{R: 0x30, G: 0x50, B: 0xc0, A: 0xff}))
			got := heldBodies(t, s)
			if len(got.Bodies) != 1 || got.Bodies[0].Name != "hammer_two" || got.Bodies[0].Heroes != "assets/bodies/hammer two.png" ||
				got.Bodies[0].HeroesLight != got.Bodies[0].Heroes || got.Bodies[0].Width != 40 || got.Bodies[0].OriginY != 36 {
				t.Fatalf("the added body: %+v", got.Bodies)
			}
			if _, ok := bodyField(t, s, "body.0.state"); ok {
				t.Fatal("a picture that fits shows a problem")
			}
			setBody(t, s, "body.0.moveframes", "4")
			setBody(t, s, "body.0.movetime", "3")
			setBody(t, s, "body.0.idleframes", "2")
			setBody(t, s, "body.0.weaponlast", "2")
			setBody(t, s, "body.0.picturelight", sheetFile(t, "light.png", color.NRGBA{R: 0x40, G: 0xa0, B: 0x50, A: 0xff}))
			b := heldBodies(t, s).Bodies[0]
			if b.Move.Frames != 4 || !slices.Equal(b.Move.Ticks, []int{3, 3, 3, 3}) || b.Idle.Frames != 2 || !b.WeaponLast || b.HeroesLight != "assets/bodies/light.png" {
				t.Fatalf("the edited body: %+v", b)
			}
			if _, held := s.workspace.Project.Files["assets/bodies/light.png"]; !held {
				t.Fatal("the light armour picture is not in the project")
			}
			weapon, ok := bodyField(t, s, bodiesWeaponKey)
			if !ok || len(weapon.Choices) != 4 {
				t.Fatalf("the weapons offered: %+v", weapon.Choices)
			}
			setBody(t, s, bodiesWeaponKey, strconv.Itoa(3))
			pick, _ := bodyField(t, s, "weapon.0.body")
			if len(pick.Choices) != 3 || pick.Value != 2 {
				t.Fatalf("the bodies offered: %+v value %d", pick.Choices, pick.Value)
			}
			setBody(t, s, "weapon.0.body", "3")
			if w := heldBodies(t, s).Weapons; len(w) != 1 || w[0].Row != 3 || w[0].Body != "hammer_two" {
				t.Fatalf("the weapon body: %+v", w)
			}
			steps := len(s.undoWorkspace)
			if _, err := s.SetWorkspaceField(ui.ModBodiesScope, "body.0.remove", ""); err == nil || !strings.Contains(err.Error(), "War Hammer") || len(s.undoWorkspace) != steps {
				t.Fatalf("a used body is removed: %v", err)
			}
			if _, err := s.SetWorkspaceField(ui.ModBodiesScope, "body.0.moveframes", "99"); err == nil || len(s.undoWorkspace) != steps {
				t.Fatalf("frames out of range: %v", err)
			}
			setBody(t, s, "weapon.0.remove", "")
			setBody(t, s, "body.0.remove", "")
			if got := heldBodies(t, s); len(got.Bodies) != 0 || len(got.Weapons) != 0 {
				t.Fatalf("after the removals: %+v", got)
			}
			if _, held := s.workspace.Project.Files["assets/bodies/hammer two.png"]; held {
				t.Fatal("the removed body keeps its picture")
			}
			steps = len(bodySteps[s])
			for n := steps - 2; n >= 0; n-- {
				if _, err := s.Undo(); err != nil {
					t.Fatal(err)
				}
				if !bytes.Equal(workspaceBytes(s.workspace), bodySteps[s][n]) {
					t.Fatalf("the undo to step %d leaves another workspace", n)
				}
			}
			if got := s.workspace.Project.Files; got[mod.BodiesFile] != nil || got[mod.WeaponBodiesFile] != nil || !maps.Equal(tree(t, folder), before) {
				t.Fatalf("the undo left files: %v", slices.Sorted(maps.Keys(got)))
			}
		})
	}
}

package game

import (
	"bytes"
	"errors"
	"fmt"
	"image/png"
	"path"
	"strings"

	"againrom/pkg/data"
	"againrom/pkg/installpath"
	"againrom/pkg/mapload"
	"againrom/pkg/mod"
	"againrom/pkg/namefold"
	"againrom/pkg/rules"
	"againrom/pkg/ui"
)

// spellPictureKeys are the fields of a spell row that name a picture, in
// the page's order.
var spellPictureKeys = []string{"icon", "cast", "burst"}

// setSpellPicture sets the picture field of r to p. A cast picture or burst
// set anew draws by its own sheet's frames, so the registry row the row
// stated for the picture before goes; the row's imported mark goes with its
// last picture.
func setSpellPicture(r *mod.SpellRow, field string, p mod.SpellPicture) {
	if p.Set {
		p.Line = map[string]int{"icon": r.Icon.Line, "cast": r.Cast.Line, "burst": r.Burst.Line}[field]
	}
	switch field {
	case "icon":
		r.Icon = p
	case "cast":
		r.Cast, r.CastRow = p, nil
	case "burst":
		r.Burst, r.BurstRow = p, nil
	}
	if len(r.Pictures()) == 0 {
		r.Imported = mod.SpellBool{}
	}
}

// importSpellPicture copies the host file source into the open mod under
// assets/spells, held to the check the game reads the picture by, and makes
// it field of catalog spell name's row in the same draft step, through the
// one file import.
func (s *mapInspectionEdits) importSpellPicture(base *mapload.Table, name, field, source string, replace bool) (*ui.InspectionDocument, error) {
	if err := s.prepareWorkspace(); err != nil {
		return nil, err
	}
	rel, b, err := s.modAssetDraft(mod.SpellIcon.Kind, source, modAssetName(source))
	if err != nil {
		return nil, err
	}
	if want := mod.SpellIcon.Names(rel); want != (field == "icon") {
		return nil, s.inspector.notSpellPicture(field, rel)
	}
	_, d, err := s.importModFile(rel, b, replace, func() (*ui.InspectionDocument, error) {
		return s.importSpellPicture(base, name, field, source, true)
	}, func(w *editorWorkspaceState, rel string) error {
		return spellsDraft(base, func(d *mod.SpellData) error {
			setSpellPicture(spellRow(d, name), field, mod.SpellPicture{Set: true, Path: rel})
			return nil
		})(kindFiles{s: s, w: w})
	})
	return d, err
}

// notSpellPicture refuses a file of the wrong sort for field: a PNG for the
// book icon, a sheet for the cast picture and the burst.
func (x *MapInspector) notSpellPicture(field, rel string) error {
	return x.inLanguage(spellPictureRefusal(field, rel))
}

// spellPictureRefusal is the refusal of rel for the picture field: the
// file sorts the field takes.
func spellPictureRefusal(field, rel string) error {
	takes := data.ProjectileSheetExt(true) + ", " + data.ProjectileSheetExt(false)
	if field == "icon" {
		takes = ".png"
	}
	return wordErr("editor.spells.err.kind", path.Base(rel), wordRef("editor.spells."+field), takes)
}

// otherGameInstall is the root of the first install the starter lists whose
// spells are not the game's of the editor's install, looked for once.
func (s *mapInspectionEdits) otherGameInstall() (string, bool) {
	x := s.inspector
	look := x.spellsLook()
	if !look.otherTried {
		look.otherTried = true
		game := mapload.SpellGameOf(x.contextBase().table)
		if x.testPlay != nil && x.testPlay.Installs != nil {
			for _, root := range x.testPlay.Installs() {
				info := InspectInstall(root)
				if root == "" || !info.Valid() {
					continue
				}
				if data.SpellPresentationNamed(info.Base.Profile.GameOf().Edition().Spells).Game != game {
					look.other = root
					break
				}
			}
		}
	}
	return look.other, look.other != ""
}

// copyOtherGameSpell copies catalog spell name's book icon, flying object
// and burst from the install of the other game into the open mod's
// assets/spells, each sheet with its registry row, and writes them into the
// spell's row in the same draft step, marked imported. The mod keeps no
// reference to the install. A picture the other game lacks is left out; a
// spell with no book cell there lacks only its icon.
func (s *mapInspectionEdits) copyOtherGameSpell(base *mapload.Table, name string, replace bool) (*ui.InspectionDocument, error) {
	x := s.inspector
	if err := s.prepareWorkspace(); err != nil {
		return nil, err
	}
	root, ok := s.otherGameInstall()
	if !ok {
		return nil, errors.New(x.word("editor.spells.othergame.absent"))
	}
	host, err := installpath.New(root).Path(GraphicsArchive)
	if err != nil {
		return nil, err
	}
	src, err := OpenContainers(host)
	if err != nil {
		return nil, err
	}
	other := data.SpellPresentationNamed(InspectInstall(root).Base.Profile.GameOf().Edition().Spells)
	spaced := strings.ReplaceAll(rules.SpellKey(name), "_", " ")
	n, ok := rules.SpellNumber(other.Game, name)
	if !ok {
		return nil, fmt.Errorf(x.word("editor.spells.othergame.lacking"), spaced)
	}
	stem := namefold.ModTree(rules.SpellKey(name))
	var files []modDraftFile
	var fields []string
	var sheets []*mod.SpellSheet
	add := func(field, file string, b []byte, sheet *mod.SpellSheet) error {
		rel, err := s.modAssetBytes(mod.SpellIcon.Kind, file, b)
		if err != nil {
			return fmt.Errorf("%s: %w", file, err)
		}
		files, fields, sheets = append(files, modDraftFile{rel, b}), append(fields, field), append(sheets, sheet)
		return nil
	}
	if icon := cutBookIcon(loadSpellAtlas(src), other.BookRows, nil, uint16(n)); icon != nil {
		var buf bytes.Buffer
		if err := png.Encode(&buf, icon); err != nil {
			return nil, err
		}
		if err := add("icon", stem+".png", buf.Bytes(), nil); err != nil {
			return nil, err
		}
	}
	records, _ := loadProjectileRecords(src)
	for _, c := range []struct {
		field   string
		picture int
	}{{"cast", data.CastPicture(n)}, {"burst", data.BurstPicture(n)}} {
		rec, ok := records.ByID(int32(c.picture))
		if !ok || rec.SpritePath() == "" {
			continue
		}
		b, err := src.ReadFile(graphicsPrefix + rec.SpritePath())
		if err != nil {
			continue
		}
		sheet := mod.NewSpellSheet(rec.Phases, rec.RotationPhases, rec.Width, rec.Height, rec.Palette, rec.Flip)
		if err := add(c.field, stem+"-"+c.field+path.Ext(rec.SpritePath()), b, sheet); err != nil {
			return nil, err
		}
	}
	if len(files) == 0 {
		return nil, fmt.Errorf(x.word("editor.spells.othergame.nothing"), spaced)
	}
	_, d, err := s.importModFiles(files, replace, func() (*ui.InspectionDocument, error) {
		return s.copyOtherGameSpell(base, name, true)
	}, func(w *editorWorkspaceState, rels []string) error {
		return spellsDraft(base, func(d *mod.SpellData) error {
			r := spellRow(d, name)
			for i, field := range fields {
				setSpellPicture(r, field, mod.SpellPicture{Set: true, Path: rels[i]})
				switch field {
				case "cast":
					r.CastRow = sheets[i]
				case "burst":
					r.BurstRow = sheets[i]
				}
			}
			r.Imported = mod.SpellBool{Set: true, Val: true, Line: r.Imported.Line}
			return nil
		})(kindFiles{s: s, w: w})
	})
	return d, err
}

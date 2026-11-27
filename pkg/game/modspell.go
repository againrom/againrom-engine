package game

import (
	"errors"
	"fmt"
	"image"
	"image/color"
	"path"
	"strings"

	"againrom/pkg/data"
	"againrom/pkg/mapload"
	"againrom/pkg/mod"
	"againrom/pkg/namefold"
	"againrom/pkg/render/terrain"
)

func (f *FrontEnd) SetModSpells(d mod.SpellData) error {
	_, err := f.setModSpells(d)
	return err
}

// setModSpells refuses a row the table cannot take, then keeps the rows
// merged one per spell over the table's names.
func (f *FrontEnd) setModSpells(d mod.SpellData) ([]mod.Override, error) {
	if d.Empty() {
		return nil, nil
	}
	if f == nil || f.Table == nil {
		return nil, fmt.Errorf("the front end has no definition table to carry the mods")
	}
	for _, r := range d.Rows {
		if line, msg := mapload.ModSpellRefusal(f.Table, r); msg != "" {
			return nil, itemFileError(r.Mod, r.File, line, "%s", msg)
		}
	}
	if _, at, msg := mapload.ModSpellProfile(f.Table, d); msg != "" {
		return nil, itemFileError(at.Mod, at.File, at.Line, "%s", msg)
	}
	var names []string
	if spells, err := data.LoadSpells(f.Table.Spells); err == nil {
		for _, sp := range spells {
			names = append(names, sp.Name)
		}
	}
	d, overrides, err := mod.MergeSpells(d, names)
	if err != nil {
		return nil, err
	}
	f.Table.Mods.Spells = d
	if err := f.setModSpellPictures(d); err != nil {
		return nil, err
	}
	return overrides, f.freezeSpellFormulas()
}

// setModSpellPictures reads and checks the pictures of the merged rows d:
// each book icon is kept on the table for the book, and each cast picture
// and burst is served under the projectile folder and put into the
// projectile records in place of the install's row of its picture, so the
// casts of the game run with the mods draw and fly by it. A refusal names
// the mod, the file, the line and the picture.
func (f *FrontEnd) setModSpellPictures(d mod.SpellData) error {
	f.Table.Mods.SpellIcons = nil
	var sheets []modSpellSheet
	for _, r := range d.Rows {
		n, ok := mapload.ModSpellNumber(f.Table, r.Target)
		if !ok {
			continue
		}
		for _, p := range r.Pictures() {
			ref, picture := mod.SpellEffect, data.CastPicture(int(n))
			switch p.Field {
			case "icon":
				ref = mod.SpellIcon
			case "burst":
				picture = data.BurstPicture(int(n))
			}
			b, err := modSpellPicture(ref, p.Picture, d.Pictures)
			if err == nil && p.Field == "icon" {
				var icon *image.RGBA
				if icon, err = decodeSpellIcon(b); err == nil {
					if f.Table.Mods.SpellIcons == nil {
						f.Table.Mods.SpellIcons = map[uint16]*image.RGBA{}
					}
					f.Table.Mods.SpellIcons[n] = icon
					continue
				}
			}
			if err != nil {
				return itemFileError(p.Picture.Mod, p.Picture.File, p.Picture.Line, "%s %q: %v", p.Field, p.Picture.Path, err)
			}
			sheets = append(sheets, modSpellSheet{p, int32(picture), b})
		}
	}
	if len(sheets) == 0 || f.Archives == nil || f.Archives.Containers == nil {
		return nil
	}
	src := f.Archives.Containers
	records, err := loadProjectileRecordsOrAbsent(src)
	if err != nil {
		return err
	}
	shared := sharedProjectileTable(src)
	files := f.modFileLayer(mod.Precedence{}, nil)
	var rows []data.Projectile
	for _, s := range sheets {
		p := s.field.Picture
		base, _ := records.ByID(s.picture)
		row, err := spellSheetRow(s.picture, base, p.Mod, p.Path, s.bytes, s.field.Row, shared)
		if err != nil {
			return itemFileError(p.Mod, p.File, p.Line, "%s %q: %v", s.field.Field, p.Path, err)
		}
		if err := files.serveBytes(p.Mod, p.File, graphicsPrefix+row.SpritePath(), s.bytes); err != nil {
			return err
		}
		rows = append(rows, row)
	}
	records = records.With(rows...)
	f.Table.Projectiles = records
	g := f.Archives.Game()
	presentation := data.SpellPresentationNamed(g.Edition().Spells)
	if f.Projectiles != nil {
		f.Projectiles = projectileSet(files.fs, records, presentation.Game)
	}
	if f.Table.FlightInputs != nil {
		if f.Table.FlightInputs, err = flightInputsWith(files.fs, presentation, filesOf(g).flight, records); err != nil {
			return err
		}
	}
	return nil
}

// modSpellSheet is one cast picture or burst of a merged row: its field,
// the picture id it draws and its bytes.
type modSpellSheet struct {
	field   mod.SpellPictureField
	picture int32
	bytes   []byte
}

// modSpellPicture is the bytes of a row's picture: the editor's draft where
// pictures holds one, else the file of the mod that names it.
func modSpellPicture(ref mod.Reference, p mod.SpellPicture, pictures map[string][]byte) ([]byte, error) {
	clean, err := ref.Path(p.Path)
	if err != nil {
		return nil, err
	}
	if b, held := pictures[clean]; held {
		return b, nil
	}
	b, _, err := ref.ReadIn(p.Dir, clean)
	return b, err
}

// decodeSpellIcon is the book icon a mod's PNG gives: exactly one book cell.
func decodeSpellIcon(b []byte) (*image.RGBA, error) {
	side := data.SpellIconSide
	return decodeFixedPicture(b, image.Pt(side, side), wordRef("editor.spells.err.iconsize"))
}

// spellSheetRow is the projectile row of picture id a mod's sheet b at rel
// in mod modID draws by: the install's row base, when there is one, for the
// keys that are not the picture's, then the keys sheet states, and the rest
// from the frames as the projectile loader reads them. A sheet that loader
// reads no frame from is refused.
func spellSheetRow(id int32, base *data.Projectile, modID, rel string, b []byte, sheet *mod.SpellSheet, shared *[256]color.RGBA) (data.Projectile, error) {
	row := data.Projectile{ID: id}
	if base != nil {
		row = *base
	}
	a16 := namefold.HostEqual(path.Ext(rel), data.ProjectileSheetExt(true))
	row.File = "mod/" + modID + "/" + strings.TrimSuffix(rel, path.Ext(rel))
	row.A16, row.Palette = 0, 1
	if a16 {
		row.A16 = 1
	}
	if sheet != nil && sheet.Palette.Set {
		row.Palette = sheet.Palette.Val
	}
	var frames []*terrain.EffectFrame
	if row.A16 != 0 {
		frames = effectSheetFrames(b)
	} else {
		frames = indexedSheetFrames(b, row.Palette == 0, shared)
	}
	if len(frames) == 0 {
		return row, errors.New(spellSheetUnread(a16, row.Palette))
	}
	row.Phases, row.RotationPhases, row.Width, row.Height, row.Flip = int32(len(frames)), 1, int32(frames[0].Width), int32(frames[0].Height), 0
	if sheet != nil {
		stated := sheet.Ints()
		for i, dst := range []*int32{&row.Phases, &row.RotationPhases, &row.Width, &row.Height, nil, &row.Flip} {
			if v := stated[i]; dst != nil && v.Set {
				*dst = v.Val
			}
		}
	}
	return row, nil
}

// spellSheetUnread says why the projectile loader reads no frame from a
// sheet read as a .16a one (a16) or a .256 one drawn through palette.
func spellSheetUnread(a16 bool, palette int32) string {
	const reads = "the projectile loader reads no frame from it: not a %s sheet "
	switch {
	case a16:
		return fmt.Sprintf(reads+"with a colour table and frames", data.ProjectileSheetExt(true))
	case palette == 0:
		return fmt.Sprintf(reads+"with frames, or the install has no shared projectile colour table for palette 0", data.ProjectileSheetExt(false))
	}
	return fmt.Sprintf(reads+"with its own colour table and frames", data.ProjectileSheetExt(false))
}

func (f *FrontEnd) freezeSpellFormulas() error {
	set, err := mapload.SpellFormulas(f.Table, f.Table.Mods.Spells)
	if err != nil {
		return err
	}
	r, err := f.Table.Rules.WithSpellFormulas(set)
	if err != nil {
		return err
	}
	f.Table.Rules = r
	return nil
}

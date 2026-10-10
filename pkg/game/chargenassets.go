package game

import (
	"fmt"
	"image"

	"againrom/pkg/base"
	"againrom/pkg/formats/bmp"
	"againrom/pkg/render/terrain"
	"againrom/pkg/render/text"
	"againrom/pkg/ui"
)

// ChargenAssets is the startup-resolved source payload for character
// generation. Strings retain their original bytes; render/text applies the
// install selector when they are drawn.
type ChargenAssets struct {
	Presentation *ui.ChargenPresentation
	Prompt       string
	Back         string
	Play         string
	Reset        string
	EmptyName    string
	ReservedName string
	SkillHover   [2][5]string
	Selector     int
	// HeroNames are the pictures' names-table lines, in picture order, and
	// EnterName is the line the page's enter writes.
	HeroNames [4]string
	EnterName string
}

func chargenRGBA(data []byte, addr string) (*image.RGBA, error) {
	pic, err := bmp.DecodeRGBA(data)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", addr, err)
	}
	return pic, nil
}

func readChargenBMP(src terrain.EntrySource, addr string) (*image.RGBA, error) {
	if src == nil {
		return nil, fmt.Errorf("%s: no archive", addr)
	}
	b, err := src.ReadFile(addr)
	if err != nil {
		return nil, err
	}
	return chargenRGBA(b, addr)
}

func readChargenMask(src terrain.EntrySource, addr string) (*image.Paletted, error) {
	if src == nil {
		return nil, fmt.Errorf("%s: no archive", addr)
	}
	b, err := src.ReadFile(addr)
	if err != nil {
		return nil, err
	}
	mask, err := bmp.DecodePaletted(b)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", addr, err)
	}
	return mask, nil
}

func chargenSize(pic image.Image, wantW, wantH int, addr string) error {
	if pic == nil || pic.Bounds().Dx() != wantW || pic.Bounds().Dy() != wantH {
		if pic == nil {
			return fmt.Errorf("%s: missing image", addr)
		}
		return fmt.Errorf("%s: size %dx%d, want %dx%d", addr, pic.Bounds().Dx(), pic.Bounds().Dy(), wantW, wantH)
	}
	return nil
}

func chargenPatchFits(pic image.Image, at image.Point, bounds image.Rectangle, addr string) error {
	if pic == nil || !pic.Bounds().Add(at).In(bounds) {
		if pic == nil {
			return fmt.Errorf("%s: missing patch", addr)
		}
		return fmt.Errorf("%s: patch %v at %v lies outside %v", addr, pic.Bounds(), at, bounds)
	}
	return nil
}

// chargenMaskCodes makes the source mask part of the mandatory presentation
// contract. A correctly sized but blank mask would otherwise load and leave
// every pictorial choice inert at runtime.
func chargenMaskCodes(mask *image.Paletted, want []uint8, addr string) error {
	if mask == nil {
		return fmt.Errorf("%s: missing mask", addr)
	}
	seen := [256]bool{}
	for _, index := range mask.Pix {
		seen[index] = true
	}
	for _, index := range want {
		if !seen[index] {
			return fmt.Errorf("%s: missing required mask index %d", addr, index)
		}
	}
	return nil
}

// chargenTextAt is one required generator slot, or the address-bearing error a
// missing one is.
//
// IT READS THE SHARED TABLE. Before that story this file carried its own
// splitter, which guessed a NUL separator when the payload held more than
// one NUL and split on LF otherwise. Neither rule is decoded.
//
// THE GENERATOR STILL FAILS ON AN ABSENT SLOT while every word 0168 added falls
// back to English. The difference is deliberate and is LoadChargenAssets' own
// stated rule: presenting a partial generator would look like an authored
// screen, where a menu row in the wrong language plainly is not.
func chargenTextAt(t *TextTable, path string, slot int) (string, error) {
	s, ok := t.At(slot)
	if !ok {
		return "", fmt.Errorf("%s slot %d is absent", path, slot)
	}
	return s, nil
}

// heroPictureNames is the four pictures' names-table lines in picture order,
// or the address-bearing error a missing one is. The name field's hero press
// and the party a hero started without the generator carries both read their
// names here, so the two cannot name one picture differently. The names are
// in the font's code page under code.
func heroPictureNames(src entrySource, l *ui.GeneratorDescription, code TextCode) ([4]string, error) {
	var names [4]string
	if l == nil {
		return names, fmt.Errorf("no generator description")
	}
	b, err := src.ReadFile(l.Words.Names)
	if err != nil {
		return names, err
	}
	rows := SplitTextTable(code.Bytes(b))
	for i, hero := range l.PreCreate.Heroes {
		if names[i], err = chargenTextAt(rows, l.Words.Names, hero.NameLine); err != nil {
			return [4]string{}, err
		}
	}
	return names, nil
}

// chargenArt reads one described picture: its size when the description
// states one, and that it lies inside bounds when drawn at its origin.
func chargenArt(src terrain.EntrySource, key string, size *ui.GeneratorPoint, at ui.GeneratorPoint, bounds image.Rectangle) (*image.RGBA, error) {
	pic, err := readChargenBMP(src, key)
	if err != nil {
		return nil, err
	}
	if size != nil {
		if err = chargenSize(pic, size[0], size[1], key); err != nil {
			return nil, err
		}
	}
	if err = chargenPatchFits(pic, at.Pt(), bounds, key); err != nil {
		return nil, err
	}
	return pic, nil
}

// chargenPane reads one detail-page pane picture at its stated size; a keyed
// pane's pure black becomes a hole. A nil pane is none.
func chargenPane(src terrain.EntrySource, pane *ui.GeneratorPane) (*image.RGBA, error) {
	if pane == nil || pane.Key == "" {
		return nil, nil
	}
	pic, err := readChargenBMP(src, pane.Key)
	if err != nil {
		return nil, err
	}
	if err = chargenSize(pic, pane.Size[0], pane.Size[1], pane.Key); err != nil {
		return nil, err
	}
	if pane.Keyed {
		pic = keyBlack(pic)
	}
	return pic, nil
}

// chargenPaneImage is a pane as an image: nil, not a nil picture
// inside an image, when the description names none.
func chargenPaneImage(src terrain.EntrySource, pane *ui.GeneratorPane) (image.Image, error) {
	pic, err := chargenPane(src, pane)
	if pic == nil || err != nil {
		return nil, err
	}
	return pic, nil
}

func chargenMaskOf(src terrain.EntrySource, key string, size ui.GeneratorPoint, want []int) (*image.Paletted, error) {
	mask, err := readChargenMask(src, key)
	if err != nil {
		return nil, err
	}
	if err = chargenSize(mask, size[0], size[1], key); err != nil {
		return nil, err
	}
	codes := make([]uint8, len(want))
	for i, w := range want {
		codes[i] = uint8(w)
	}
	if err = chargenMaskCodes(mask, codes, key); err != nil {
		return nil, err
	}
	return mask, nil
}

func chargenFont(src terrain.EntrySource, f ui.GeneratorFont) (*text.Font, error) {
	if f.Atlas == "16a" {
		return LoadFont(src, f.Name, FontCoverage)
	}
	return LoadFont(src, f.Name, FontShades)
}

// LoadChargenAssets reads the generator presentation the description names,
// once. Required controls and words fail as address-bearing construction
// errors: presenting a partial source UI would look like an authored
// fallback. The words are in the font's code page under game's edition.
func LoadChargenAssets(src terrain.EntrySource, l *ui.GeneratorDescription, game base.Game) (*ChargenAssets, error) {
	if l == nil {
		return nil, fmt.Errorf("character generator: no description")
	}
	p := &ui.ChargenPresentation{Layout: l}
	page := image.Rectangle{Max: l.Page.Size.Pt()}
	pre, d := &l.PreCreate, &l.Detail
	var err error
	// TOWN-223 names this independent animated decoration. It is optional
	// for incomplete diagnostic archives; a full install supplies its frames.
	if s := pre.Sparkle; s != nil && src != nil {
		if raw, readErr := src.ReadFile(s.Key); readErr == nil {
			p.Sparkles, _ = decodeCursor16AFrames(s.Key, raw)
		}
	}
	for i, level := range pre.Levels {
		for j, art := range level.Art {
			if p.Levels[i][j], err = chargenArt(src, art.Key, art.Size, art.At, page); err != nil {
				return nil, err
			}
		}
	}
	bg := pre.Background
	if p.Background, err = chargenArt(src, bg.Key, bg.Size, bg.At, page); err != nil {
		return nil, err
	}
	if p.PreMask, err = chargenMaskOf(src, pre.Mask.Key, pre.Mask.Size, pre.Mask.Required); err != nil {
		return nil, err
	}
	for _, b := range []struct {
		art *ui.GeneratorArt
		dst *image.Image
	}{{&pre.Back.Art, &p.Amulet}, {&pre.Forward.Art, &p.Forward}} {
		if b.art.Key == "" {
			continue
		}
		pic, err := chargenArt(src, b.art.Key, b.art.Size, b.art.At, page)
		if err != nil {
			return nil, err
		}
		*b.dst = pic
	}
	for i, hero := range pre.Heroes {
		for j, art := range hero.Art {
			if p.Choices[i][j], err = chargenArt(src, art.Key, art.Size, art.At, page); err != nil {
				return nil, err
			}
		}
	}
	for _, loop := range pre.Loops {
		var members []image.Image
		for k := 0; k < loop.Count; k++ {
			key := fmt.Sprintf(loop.Key, loop.First+k)
			pic, err := chargenArt(src, key, &loop.Size, loop.At, page)
			if err != nil {
				return nil, err
			}
			members = append(members, pic)
		}
		p.Loops = append(p.Loops, members)
	}
	if p.Plate, err = chargenPaneImage(src, &d.Plate); err != nil {
		return nil, err
	}
	if p.PlateSeam, err = chargenPaneImage(src, d.PlateSeam); err != nil {
		return nil, err
	}
	if p.NavArt, err = chargenPaneImage(src, &d.Nav); err != nil {
		return nil, err
	}
	if p.NavSeam, err = chargenPaneImage(src, d.NavSeam); err != nil {
		return nil, err
	}
	for i, c := range d.Commands {
		for j, key := range []string{c.Off, c.On} {
			if key == "" {
				continue
			}
			pic, err := readChargenBMP(src, key)
			if err != nil {
				return nil, err
			}
			if err = chargenSize(pic, c.Size[0], c.Size[1], key); err != nil {
				return nil, err
			}
			if c.Keyed {
				pic = keyBlack(pic)
			}
			p.NavButtons[i][j] = pic
		}
	}
	if p.DollPane.Body, err = chargenPaneImage(src, &d.Doll); err != nil {
		return nil, err
	}
	if p.DollPane.Seam, err = chargenPaneImage(src, d.DollSeam); err != nil {
		return nil, err
	}
	if p.CardBackground, err = chargenPane(src, &d.Card); err != nil {
		return nil, err
	}
	if p.CardSeam, err = chargenPaneImage(src, d.CardSeam); err != nil {
		return nil, err
	}
	column := image.Rectangle{Max: d.ColumnRect.Rectangle().Size()}
	for class, c := range d.Classes {
		if p.Columns[class], err = chargenArt(src, c.Column.Key, c.Column.Size, c.Column.At, column); err != nil {
			return nil, err
		}
		var codes []int
		for _, skill := range c.Skills[:c.Selectable] {
			codes = append(codes, skill.Mask)
		}
		size := ui.GeneratorPoint{column.Dx(), column.Dy()}
		if c.Mask.Size != nil {
			size = *c.Mask.Size
		}
		if p.ColumnMask[class], err = chargenMaskOf(src, c.Mask.Key, size, codes); err != nil {
			return nil, err
		}
		// Each cell's three pictures load in the order the page reads them:
		// selected at rest, hovered, selected and hovered. The fourth state
		// is the column's own art.
		states := []string{d.SkillStates.SelectedAtRest, d.SkillStates.Hover, d.SkillStates.Selected}
		for skill, s := range c.Skills[:c.Selectable] {
			for state, name := range states {
				if p.Skills[class][skill][state], err = chargenArt(src, s.Dir+name, nil, s.At, column); err != nil {
					return nil, err
				}
			}
		}
	}
	st := &d.Stats
	for direction, names := range []ui.GeneratorStatButtons{st.MinusArt, st.PlusArt} {
		for state, name := range []string{names.Rest, names.Hover, names.Down, names.Unused, names.Disabled} {
			key := st.ButtonDir + name
			pic, err := readChargenBMP(src, key)
			if err != nil {
				return nil, err
			}
			if err = chargenSize(pic, st.ButtonSize[0], st.ButtonSize[1], key); err != nil {
				return nil, err
			}
			p.StatButtons[direction][state] = pic
		}
	}
	if p.Font, err = chargenFont(src, l.Words.TextFont); err != nil {
		return nil, err
	}
	if p.NameFont, err = chargenFont(src, l.Words.NameFont); err != nil {
		return nil, err
	}
	if l.Words.TipFont == "text" {
		p.TipFont = p.Font
	}
	if vf := l.Detail.Stats.ValueFont; vf != nil {
		if p.ValueFont, err = chargenFont(src, *vf); err != nil {
			return nil, err
		}
	}
	b, err := src.ReadFile(l.Words.Table)
	if err != nil {
		return nil, err
	}
	code := InstallTextCode(src, game.Edition())
	rows := SplitTextTable(code.Bytes(b))
	a := &ChargenAssets{Presentation: p, Selector: LanguageSelector(src)}
	get := func(slot int) (string, error) {
		return chargenTextAt(rows, l.Words.Table, slot)
	}
	if a.HeroNames, err = heroPictureNames(src, l, code); err != nil {
		return nil, err
	}
	nameRows, err := src.ReadFile(l.Words.Names)
	if err != nil {
		return nil, err
	}
	if a.EnterName, err = chargenTextAt(SplitTextTable(code.Bytes(nameRows)), l.Words.Names, pre.Name.EnterLine); err != nil {
		return nil, err
	}
	if a.Prompt, err = get(pre.Name.Prompt.Slot); err != nil {
		return nil, err
	}
	if s := d.Refusals.EmptySlot; s != nil {
		if a.EmptyName, err = get(*s); err != nil {
			return nil, err
		}
	}
	if s := d.Refusals.ReservedSlot; s != nil {
		if a.ReservedName, err = get(*s); err != nil {
			return nil, err
		}
	}
	for class, c := range d.Classes {
		for skill := 0; skill < c.Selectable; skill++ {
			if a.SkillHover[class][skill], err = get(c.Tooltip + skill); err != nil {
				return nil, err
			}
		}
	}
	for _, c := range []struct {
		role string
		dst  *string
	}{{"play", &a.Play}, {"reset", &a.Reset}, {"back", &a.Back}} {
		for _, command := range d.Commands {
			if command.Role == c.role {
				if *c.dst, err = get(command.Slot); err != nil {
					return nil, err
				}
			}
		}
	}
	p.Font.Selector = a.Selector
	p.NameFont.Selector = a.Selector
	if p.ValueFont != nil {
		p.ValueFont.Selector = a.Selector
	}
	return a, nil
}

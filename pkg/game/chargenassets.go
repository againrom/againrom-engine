package game

import (
	"fmt"

	"againrom/pkg/base"
	"againrom/pkg/render/terrain"
	"againrom/pkg/render/text"
	"againrom/pkg/screenart"
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

// generatorArt is the generator description's art as the one screen art
// loader's manifest, in the order the generator reads it. Every entry is
// required but the sparkle, which incomplete diagnostic archives may lack
// (TOWN-223). A picture with an origin must lie inside its page or column.
func generatorArt(l *ui.GeneratorDescription) []screenart.Entry {
	var out []screenart.Entry
	pt := func(p ui.GeneratorPoint) *screenart.Point { q := screenart.Point(p); return &q }
	opt := func(p *ui.GeneratorPoint) *screenart.Point {
		if p == nil {
			return nil
		}
		return pt(*p)
	}
	page := &screenart.Rect{0, 0, l.Page.Size[0], l.Page.Size[1]}
	art := func(name, key string, size *screenart.Point, at ui.GeneratorPoint, inside *screenart.Rect) {
		out = append(out, screenart.Entry{Name: name, Format: "picture", Key: key, Size: size, At: pt(at), Inside: inside, Required: true})
	}
	pane := func(name string, p *ui.GeneratorPane) {
		if p == nil || p.Key == "" {
			return
		}
		e := screenart.Entry{Name: name, Format: "picture", Key: p.Key, Size: pt(p.Size), Required: true}
		if p.Keyed {
			e.Transparent = "black"
		}
		out = append(out, e)
	}
	mask := func(name, key string, size ui.GeneratorPoint, want []int) {
		out = append(out, screenart.Entry{Name: name, Format: "mask", Key: key, Size: pt(size), RequiredBytes: want, Required: true})
	}
	pre, d := &l.PreCreate, &l.Detail
	if s := pre.Sparkle; s != nil {
		out = append(out, screenart.Entry{Name: "sparkle", Format: "sprites", Key: s.Key})
	}
	for i, level := range pre.Levels {
		for j, a := range level.Art {
			art(fmt.Sprintf("level/%d/%d", i, j), a.Key, opt(a.Size), a.At, page)
		}
	}
	art("background", pre.Background.Key, opt(pre.Background.Size), pre.Background.At, page)
	mask("pre-mask", pre.Mask.Key, pre.Mask.Size, pre.Mask.Required)
	for _, b := range []struct {
		name string
		art  *ui.GeneratorArt
	}{{"back", &pre.Back.Art}, {"forward", &pre.Forward.Art}} {
		if b.art.Key != "" {
			art(b.name, b.art.Key, opt(b.art.Size), b.art.At, page)
		}
	}
	for i, hero := range pre.Heroes {
		for j, a := range hero.Art {
			art(fmt.Sprintf("hero/%d/%d", i, j), a.Key, opt(a.Size), a.At, page)
		}
	}
	for i, loop := range pre.Loops {
		out = append(out, screenart.Entry{Name: fmt.Sprintf("loop/%d", i), Format: "series", Key: loop.Key, First: loop.First,
			Count: loop.Count, Size: pt(loop.Size), At: pt(loop.At), Inside: page, Required: true})
	}
	pane("plate", &d.Plate)
	pane("plate-seam", d.PlateSeam)
	pane("nav", &d.Nav)
	pane("nav-seam", d.NavSeam)
	for i, c := range d.Commands {
		for j, key := range []string{c.Off, c.On} {
			pane(fmt.Sprintf("command/%d/%d", i, j), &ui.GeneratorPane{Key: key, Size: c.Size, Keyed: c.Keyed})
		}
	}
	pane("doll", &d.Doll)
	pane("doll-seam", d.DollSeam)
	pane("card", &d.Card)
	pane("card-seam", d.CardSeam)
	column := d.ColumnRect.Rectangle().Size()
	inColumn := &screenart.Rect{0, 0, column.X, column.Y}
	// Each cell's three pictures load in the order the page reads them:
	// selected at rest, hovered, selected and hovered. The fourth state is
	// the column's own art.
	states := []string{d.SkillStates.SelectedAtRest, d.SkillStates.Hover, d.SkillStates.Selected}
	for class, c := range d.Classes {
		art(fmt.Sprintf("column/%d", class), c.Column.Key, opt(c.Column.Size), c.Column.At, inColumn)
		var codes []int
		for _, skill := range c.Skills[:c.Selectable] {
			codes = append(codes, skill.Mask)
		}
		size := ui.GeneratorPoint{column.X, column.Y}
		if c.Mask.Size != nil {
			size = *c.Mask.Size
		}
		mask(fmt.Sprintf("column-mask/%d", class), c.Mask.Key, size, codes)
		for skill, sk := range c.Skills[:c.Selectable] {
			for state, name := range states {
				art(fmt.Sprintf("skill/%d/%d/%d", class, skill, state), sk.Dir+name, nil, sk.At, inColumn)
			}
		}
	}
	st := &d.Stats
	for direction, names := range []ui.GeneratorStatButtons{st.MinusArt, st.PlusArt} {
		for state, name := range []string{names.Rest, names.Hover, names.Down, names.Unused, names.Disabled} {
			out = append(out, screenart.Entry{Name: fmt.Sprintf("stat/%d/%d", direction, state), Format: "picture",
				Key: st.ButtonDir + name, Size: pt(st.ButtonSize), Required: true})
		}
	}
	return out
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
	set, err := screenArtOf(src).Load("generator", generatorArt(l))
	if err != nil {
		return nil, err
	}
	pre, d := &l.PreCreate, &l.Detail
	for _, f := range set.Pictures("sparkle") {
		p.Sparkles = append(p.Sparkles, rgbaOf(f))
	}
	for i, level := range pre.Levels {
		for j := range level.Art {
			p.Levels[i][j] = set.Picture(fmt.Sprintf("level/%d/%d", i, j))
		}
	}
	p.Background, p.PreMask = set.Picture("background"), set.Mask("pre-mask")
	p.Amulet, p.Forward = set.Picture("back"), set.Picture("forward")
	for i, hero := range pre.Heroes {
		for j := range hero.Art {
			p.Choices[i][j] = set.Picture(fmt.Sprintf("hero/%d/%d", i, j))
		}
	}
	for i := range pre.Loops {
		p.Loops = append(p.Loops, set.Pictures(fmt.Sprintf("loop/%d", i)))
	}
	p.Plate, p.PlateSeam = set.Picture("plate"), set.Picture("plate-seam")
	p.NavArt, p.NavSeam = set.Picture("nav"), set.Picture("nav-seam")
	for i := range d.Commands {
		for j := range p.NavButtons[i] {
			p.NavButtons[i][j] = set.Picture(fmt.Sprintf("command/%d/%d", i, j))
		}
	}
	p.DollPane.Body, p.DollPane.Seam = set.Picture("doll"), set.Picture("doll-seam")
	p.CardBackground, p.CardSeam = rgbaOf(set.Picture("card")), set.Picture("card-seam")
	for class, c := range d.Classes {
		p.Columns[class], p.ColumnMask[class] = set.Picture(fmt.Sprintf("column/%d", class)), set.Mask(fmt.Sprintf("column-mask/%d", class))
		for skill := range c.Skills[:c.Selectable] {
			for state := range p.Skills[class][skill] {
				p.Skills[class][skill][state] = set.Picture(fmt.Sprintf("skill/%d/%d/%d", class, skill, state))
			}
		}
	}
	for direction := range p.StatButtons {
		for state := range p.StatButtons[direction] {
			p.StatButtons[direction][state] = set.Picture(fmt.Sprintf("stat/%d/%d", direction, state))
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
		for skill := 0; c.Tooltip != nil && skill < c.Selectable; skill++ {
			if a.SkillHover[class][skill], err = get(*c.Tooltip + skill); err != nil {
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

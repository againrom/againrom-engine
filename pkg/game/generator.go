package game

import (
	_ "embed"
	"fmt"

	"againrom/pkg/base"
	"againrom/pkg/ui"
)

// rom1GeneratorJSON is the first game's character generator as data for the
// generator builder. Every value in it carries the claim, divergence row or
// owner ruling behind it.
//
//go:embed generators/rom1.json
var rom1GeneratorJSON []byte

// rom2GeneratorJSON is the second game's character generator as data, cited
// the same way.
//
//go:embed generators/rom2.json
var rom2GeneratorJSON []byte

// generatorJSON are the encoded generator descriptions an edition can name,
// by that name.
var generatorJSON = map[string][]byte{"rom1": rom1GeneratorJSON, "rom2": rom2GeneratorJSON}

// generatorDescriptions are the decoded descriptions. A description that does
// not decode is a build defect, so it stops the process at start.
var generatorDescriptions = func() map[string]*ui.GeneratorDescription {
	out := map[string]*ui.GeneratorDescription{}
	for name, data := range generatorJSON {
		out[name] = mustDecodeGenerator(data)
	}
	return out
}()

func mustDecodeGenerator(data []byte) *ui.GeneratorDescription {
	d, err := ui.DecodeGenerator(data)
	if err != nil {
		panic(err)
	}
	return d
}

// GeneratorDescription is the generator description the profile's edition
// names, or nil when its game has none. Callers must not change it.
func GeneratorDescription(p base.Profile) *ui.GeneratorDescription {
	return generatorDescriptions[p.Edition().Generator]
}

// heroNames are the hero name lines of the edition's generator description,
// as defaultHeroName indexes them: class plus twice sex, so male fighter,
// male mage, female fighter, female mage, whatever order the description
// gives its pictures.
func heroNames(src entrySource, e base.Edition, code TextCode) ([4]string, error) {
	l := generatorDescriptions[e.Generator]
	pictures, err := heroPictureNames(src, l, code)
	if err != nil {
		return pictures, err
	}
	var names [4]string
	var seen [4]bool
	for i, h := range l.PreCreate.Heroes {
		at := h.Class + 2*h.Sex
		if at < 0 || at >= len(names) || i >= len(pictures) || seen[at] {
			return [4]string{}, fmt.Errorf("generator %s: hero %d is not one of four sex and class pictures", e.Generator, i)
		}
		names[at], seen[at] = pictures[i], true
	}
	return names, nil
}

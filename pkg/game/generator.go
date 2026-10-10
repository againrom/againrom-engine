package game

import (
	_ "embed"

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

// heroNameGenerator is the description whose hero name lines a hero started
// without the generator is named from.
func heroNameGenerator() *ui.GeneratorDescription { return generatorDescriptions["rom1"] }

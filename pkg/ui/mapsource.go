package ui

import (
	"strconv"
	"strings"
)

// MapSourceKind is what an editor map source addresses.
type MapSourceKind uint8

const (
	// SourceHost is a host file path, the whole source.
	SourceHost MapSourceKind = iota
	// SourceNew is "new/<W>x<H>": a map built from nothing.
	SourceNew
	// SourceScenario is "scenario/<name>": a map in the install's scenario
	// archive, read by the whole source.
	SourceScenario
	// SourceLoose is "loose/<path>": a loose map of the install.
	SourceLoose
)

// mapSourcePrefixes is each addressed kind's prefix. The scenario prefix is
// the scenario archive's address prefix, which pkg/game derives from the
// archive's name and its test holds equal to this one.
var mapSourcePrefixes = [...]struct {
	kind   MapSourceKind
	prefix string
}{{SourceNew, "new/"}, {SourceScenario, "scenario/"}, {SourceLoose, "loose/"}}

// MapSource is an editor map source parsed once: its kind, the source as
// written, the path after the kind's prefix (the whole source for a host
// file), and for a scenario map the mission number its file name gives.
type MapSource struct {
	Kind      MapSourceKind
	Text      string
	Path      string
	Number    int
	HasNumber bool
}

// ParseMapSource is the one reading of an editor map source.
func ParseMapSource(source string) MapSource {
	for _, p := range mapSourcePrefixes {
		if rest, ok := strings.CutPrefix(source, p.prefix); ok {
			s := MapSource{Kind: p.kind, Text: source, Path: rest}
			if p.kind == SourceScenario {
				n, err := strconv.Atoi(strings.TrimSuffix(rest, ".alm"))
				s.Number, s.HasNumber = n, err == nil
			}
			return s
		}
	}
	return MapSource{Kind: SourceHost, Text: source, Path: source}
}

// MapSourceKey is the source of a map of kind at path.
func MapSourceKey(kind MapSourceKind, path string) string {
	for _, p := range mapSourcePrefixes {
		if p.kind == kind {
			return p.prefix + path
		}
	}
	return path
}

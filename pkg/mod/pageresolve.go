package mod

import (
	"path"

	"againrom/pkg/locale"
)

// PageSource is where a resolved mission page comes from.
type PageSource uint8

// The sources, lowest priority first: the install's own page, a page of a
// mod's mission entry, and a page of the map's own text folder.
const (
	PageInstall PageSource = iota
	PageMod
	PageMap
)

func (s PageSource) String() string {
	switch s {
	case PageInstall:
		return "install"
	case PageMod:
		return "mod"
	case PageMap:
		return "map"
	}
	return "unknown"
}

// PageLayer is the pages one source holds for a mission: by file name, the
// names read from the other language's folders, and the language the source
// is read in.
type PageLayer struct {
	Source PageSource
	Lang   string
	Files  map[string][]byte
	Other  map[string]bool
}

// ResolvedPage is one page of a mission: its bytes, the layer it was taken
// from and the language of the folder it was read in.
type ResolvedPage struct {
	Bytes  []byte
	Source PageSource
	Locale string
}

// MissionPages is a mission's resolved pages with the entry and the mod they
// belong to, as the mission start, the map check and the editor read them. Err
// is the read that failed, and Pages then holds what was read before it.
type MissionPages struct {
	ID    string
	Entry Mission
	Pages map[string]ResolvedPage
	Err   error
}

// ResolvePages is the one page resolver. Layers come lowest priority first: a
// page of the language the layer is read in is taken from the last layer that
// holds it, and a name no layer holds in that language is taken from the last
// layer that holds it as an other-language page. keep, when set, drops the
// names it refuses.
func ResolvePages(keep func(name string) bool, layers ...PageLayer) map[string]ResolvedPage {
	own := map[string]ResolvedPage{}
	other := map[string]ResolvedPage{}
	for _, l := range layers {
		for name, b := range l.Files {
			if keep != nil && !keep(name) {
				continue
			}
			if l.Other[name] {
				other[name] = ResolvedPage{Bytes: b, Source: l.Source, Locale: locale.Other(l.Lang)}
				continue
			}
			own[name] = ResolvedPage{Bytes: b, Source: l.Source, Locale: l.Lang}
		}
	}
	for name, p := range other {
		if _, held := own[name]; !held {
			own[name] = p
		}
	}
	return own
}

// ReadPageLayer reads the pages of m's texts folders under dir in lang, as
// source's layer. It is the only reader of a mission's text files.
func (m Mission) ReadPageLayer(source PageSource, dir, lang string) (PageLayer, error) {
	files, other, err := m.ReadTexts(dir, lang)
	return PageLayer{Source: source, Lang: m.pageLang(lang), Files: files, Other: other}, err
}

// HeldPageLayer is the pages the mod loader holds for m.
func (m Mission) HeldPageLayer() PageLayer {
	return PageLayer{Source: PageMod, Lang: m.pageLang(m.language), Files: m.TextFiles, Other: m.OtherLanguageTexts}
}

// HeldPages is m's resolved pages as the mod loader holds them.
func (m Mission) HeldPages() *MissionPages {
	return &MissionPages{ID: m.Mod, Entry: m, Pages: ResolvePages(nil, m.HeldPageLayer())}
}

// pageLang is the language m's pages are read in: lang, or the one its texts
// folder names, or the fallback.
func (m Mission) pageLang(lang string) string {
	if lang != "" {
		return lang
	}
	if tail := path.Base(m.Texts); locale.Known(tail) {
		return tail
	}
	return locale.Fallback
}

// Files is the page bytes by name, a map of the caller's own.
func (p *MissionPages) Files() map[string][]byte {
	out := make(map[string][]byte, len(p.Pages))
	for name, page := range p.Pages {
		out[name] = page.Bytes
	}
	return out
}

// Put sets a page the editor holds in the map's own folder.
func (p *MissionPages) Put(name string, b []byte, lang string) {
	if p.Pages == nil {
		p.Pages = map[string]ResolvedPage{}
	}
	p.Pages[name] = ResolvedPage{Bytes: b, Source: PageMap, Locale: lang}
}

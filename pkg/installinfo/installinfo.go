// Package installinfo is the one builder of the cheap look at a directory as
// an install: the profile it was detected as, the files it lacks and its
// language. The game calls it and so does the starter; it imports no game
// code, so a program that needs only this look does not link the game.
package installinfo

import (
	"errors"
	"fmt"

	"againrom/pkg/base"
	"againrom/pkg/locale"
	"againrom/pkg/vfs"
)

// MainPrefix is the address prefix of everything read out of the main archive.
// It is spelt once, here, for the language entry below and for pkg/game's
// addresses.
const MainPrefix = "main/"

// LanguagePath is the entry an install names its own language in.
const LanguagePath = MainPrefix + "id"

// Info is what a cheap look at a directory says about it as an install.
type Info struct {
	// Missing lists the files the directory lacks: the required archives in
	// RequiredArchives order, or, for a root whose main archive is a known
	// build, the files that build's profile ships. Empty means none is missing.
	Missing []string
	// Language is the install's own language name when main.res could be read:
	// "english", "russian", or "language selector N" for another value. Empty
	// when it could not be read.
	Language string
	// Base is the profile the root was detected as; the zero Match when the
	// root is not an install.
	Base base.Match
	// BaseErr is why no profile fits, nil when Base is set.
	BaseErr error
}

// Valid reports whether the directory is an install of a detected profile.
func (i Info) Valid() bool { return len(i.Missing) == 0 && i.BaseErr == nil }

// EntrySource is what the language selector reads: an entry by its address.
type EntrySource interface {
	ReadFile(name string) ([]byte, error)
}

// archiveLanguage reads the language entry of the main archive at path alone:
// "english", "russian", "language selector N", or "" when it cannot be read.
func archiveLanguage(path string) string {
	fs, err := vfs.Open([]string{path}, nil)
	if err != nil {
		return ""
	}
	return languageEntry(LanguageSelector(fs))
}

// languageEntry is the language entry of an install language selector: its
// locale's entry, else "language selector N".
func languageEntry(selector int) string {
	if l, ok := locale.BySelector(selector); ok {
		return l.Entry
	}
	return fmt.Sprintf("language selector %d", selector)
}

// LanguageSelector is the install's own language selector: the TRAILING ASCII
// DIGIT of LanguagePath's payload, taken as a number. The shipped roots hold
// "english 0" and "russian 1", so the two releases select 0 and 1 and the
// executable behind them is the same bytes — which is the whole reason a
// consumer that ships one program and two data sets is doing what the engine
// does.
//
// EVERY FAILURE IS SELECTOR 0, and none of them is an error. No source, no
// entry, an empty payload and a payload ending in something that is not a digit
// all answer 0, which is the identity conversion. That is deliberate and it is
// the only safe direction: a selector this function got wrong by guessing would
// silently redraw every string in the game, whereas 0 can only ever leave
// drawing exactly as it was. An install that fails to say which language it is
// is therefore treated as the one whose rule changes nothing.
//
// It reads the LAST byte and not the last RUNE. The payload is the game's own
// byte string and the value it carries is a single decimal digit; a multi-digit
// selector is not something either shipped root has, and inventing a parse for
// one would be inventing a fact.
func LanguageSelector(src EntrySource) int {
	if src == nil {
		return 0
	}
	b, err := src.ReadFile(LanguagePath)
	if err != nil || len(b) == 0 {
		return 0
	}
	c := b[len(b)-1]
	if c < '0' || c > '9' {
		return 0
	}
	return int(c - '0')
}

// Detect matches root against the known base profiles. It never writes. An
// error says why no profile fits: a missing archive, or a known build lacking a
// file its profile ships.
func Detect(root string) (base.Match, error) {
	return base.Detect(root, archiveLanguage)
}

// Inspect looks at root without opening the whole install: it detects the
// base profile (the directory listing, plus the main archive's digest when its
// size is a known build's) and, when all archives are present, reads the
// language entry from main.res alone. It never writes.
func Inspect(root string) Info {
	var info Info
	m, err := Detect(root)
	info.Base, info.BaseErr = m, err
	var notInstall *base.NotInstallError
	var partial *base.PartialError
	switch {
	case errors.As(err, &notInstall):
		info.Missing = notInstall.Missing
		return info
	case errors.As(err, &partial):
		info.Missing = partial.Missing
		return info
	case err != nil:
		return info
	}
	info.Language = archiveLanguage(m.Main)
	return info
}

package game

import (
	"errors"
	"fmt"

	"againrom/pkg/base"
)

// InstallInfo is what a cheap look at a directory says about it as an install.
type InstallInfo struct {
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
func (i InstallInfo) Valid() bool { return len(i.Missing) == 0 && i.BaseErr == nil }

// archiveLanguage reads the language entry of the main archive at path alone:
// "english", "russian", "language selector N", or "" when it cannot be read.
func archiveLanguage(path string) string {
	fs, err := OpenContainers(path)
	if err != nil {
		return ""
	}
	switch sel := LanguageSelector(fs); sel {
	case 0:
		return "english"
	case 1:
		return "russian"
	default:
		return fmt.Sprintf("language selector %d", sel)
	}
}

// DetectBase matches root against the known base profiles. It never writes. An
// error says why no profile fits: a missing archive, or a known build lacking a
// file its profile ships.
func DetectBase(root string) (base.Match, error) {
	return base.Detect(root, archiveLanguage)
}

// InspectInstall looks at root without opening the whole install: it detects the
// base profile (the directory listing, plus the main archive's digest when its
// size is a known build's) and, when all archives are present, reads the
// language entry from main.res alone. It never writes.
func InspectInstall(root string) InstallInfo {
	var info InstallInfo
	m, err := DetectBase(root)
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

package game

import (
	"strings"

	"againrom/pkg/installpath"
)

// ResolveAssetRoot returns the game asset root directory from configuration,
// preferring an explicit flag value over the environment. Precedence is
// flagValue > envValue > "" (empty means "not configured"). Surrounding
// whitespace is trimmed. The asset root is always supplied by configuration
// — a CLI flag or the AGAINROM_ASSETS environment variable — and no game
// install path is ever compiled into the source.
func ResolveAssetRoot(flagValue, envValue string) string {
	if v := strings.TrimSpace(flagValue); v != "" {
		return v
	}
	return strings.TrimSpace(envValue)
}

// DiscoverAssetRoot returns the first candidate directory that carries every
// archive OpenArchives requires, or "" when none does.
//
// It is the last step of the precedence above and runs only when neither the
// flag nor the environment named a root. The caller supplies the candidates and
// owns their order; no install path is compiled in, so boundary 3 still holds:
// the root remains a runtime input, and this reads it off the filesystem instead
// of off a flag.
//
// A directory qualifies only when all five required archives are present.
// Requiring the whole set is what separates an install from a directory that
// happens to hold one `.res` file, and it is the same set the front end opens a
// moment later, so a directory this function accepts is one OpenArchives can
// open.
//
// The file-name comparison folds case through the install name resolver. The
// two lawful installs disagree: the EN root ships `graphics.res` and the RU
// root ships `GRAPHICS.RES`.
func DiscoverAssetRoot(candidates ...string) string {
	for _, candidate := range candidates {
		dir := strings.TrimSpace(candidate)
		if dir == "" {
			continue
		}
		if hasRequiredArchives(dir) {
			return dir
		}
	}
	return ""
}

// hasRequiredArchives reports whether dir holds every name RequiredArchives
// lists as a file, through the install name resolver: one listing of dir,
// names compared with case folded, so a case-sensitive filesystem gives the
// answer Windows' own does.
//
// An unreadable directory is not an install, and neither is one holding two
// names that differ only by case: this function's caller is deciding between
// candidates, and a candidate it cannot read unambiguously is one it cannot use.
func hasRequiredArchives(dir string) bool {
	return requiredArchivesIn(installpath.New(dir))
}

// requiredArchivesIn is hasRequiredArchives over the resolver of a directory.
func requiredArchivesIn(files *installpath.Resolver) bool {
	for _, name := range RequiredArchives() {
		e, err := files.Resolve(name)
		if err != nil || e.Dir {
			return false
		}
	}
	return true
}

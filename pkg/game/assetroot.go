package game

import (
	"os"
	"strings"
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
// The file-name comparison folds case. The two lawful installs disagree: the EN
// root ships `graphics.res` and the RU root ships `GRAPHICS.RES`. Folding case
// here gives the same answer on a case-sensitive filesystem as on Windows' own,
// rather than leaving the result to depend on the host being forgiving.
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
// lists, comparing file names with case folded.
//
// It reads the directory once and then answers from that listing. Calling
// os.Stat per archive would ask the host filesystem to do the case folding,
// which Windows does and a case-sensitive filesystem does not, so the two would
// disagree about the same install.
//
// An unreadable directory is not an install. A missing or permission-denied
// candidate returns false rather than an error: this function's caller is
// deciding between candidates, and a candidate it cannot read is one it cannot
// use.
func hasRequiredArchives(dir string) bool {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return false
	}
	present := make(map[string]bool, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		present[strings.ToLower(entry.Name())] = true
	}
	for _, name := range RequiredArchives() {
		if !present[strings.ToLower(name)] {
			return false
		}
	}
	return true
}

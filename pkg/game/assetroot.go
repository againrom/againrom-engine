package game

import (
	"os"
	"path/filepath"
	"strings"

	"againrom/pkg/installpath"
)

// ProcessDirs names the directories to search for an install when nobody
// configured an asset root: the working directory, then the directory holding
// the executable. DiscoverAssetRoot takes them in that order.
//
// THE WORKING DIRECTORY IS FIRST BECAUSE IT IS WHAT THE ORIGINAL USES.
// RES-DIR-037 is High for the code: rom.exe's resource manager is constructed
// before main, calls GetCurrentDirectoryA and AddDirectory(cwd), and AddArchive
// then indexes dir[0] alone with no loop, so an archive not sitting beside the
// working directory never opens whatever else is registered. The working
// directory IS the original's install lookup, and ordering the executable's own
// directory ahead of it would put an authored choice in front of a decoded fact.
//
// The executable's directory is second, and it is ours rather than the
// original's. It answers the case the working directory cannot — a shortcut that
// starts elsewhere, or a binary invoked by absolute path — and it is an addition
// layered after the original's behaviour, never in front of it. `go run` reaches
// neither: its executable lives in a build cache, and its working directory is
// the source tree.
//
// Neither candidate is compiled in; both are read from the running process, so
// boundary 3 holds.
//
// THIS IS NOT ALL THE ORIGINAL DOES. The same claim records that startup also
// registers INSTALLDIR from HKLM\SOFTWARE\1C\Allods, the stripped GetTempPathA
// result, and the CD registry value plus "Allods". None of those is implemented
// here, and the gap is DIV-430 rather than a silence. The claim's own second
// half is Medium — which directories a given launch actually holds was not
// observed at runtime — so what is decoded is the mechanism, not the population.
//
// An error from either call drops that candidate rather than failing the run.
// Nothing here is required: the whole function only ever supplies a fallback, and
// a fallback that refuses to be computed would turn a missing root into a
// different error than the one the caller is about to print.
func ProcessDirs(executable string) []string {
	dirs := make([]string, 0, 2)
	if wd, err := os.Getwd(); err == nil {
		dirs = append(dirs, wd)
	}
	if executable != "" {
		dirs = append(dirs, filepath.Dir(executable))
	}
	return dirs
}

// InstallHolding is the install that holds the file at path: the nearest
// folder above it that carries every required archive. It answers false for a
// path that is no file, and for one outside every install.
func InstallHolding(path string) (string, bool) {
	if info, err := os.Stat(path); err != nil || info.IsDir() {
		return "", false
	}
	abs, err := filepath.Abs(filepath.Dir(path))
	if err != nil {
		return "", false
	}
	resolved, err := resolvePathForSaveSafety(abs)
	if err != nil {
		return "", false
	}
	root, found, err := requiredArchiveAncestor(resolved)
	if err != nil || !found || DiscoverAssetRoot(root) == "" {
		return "", false
	}
	return root, true
}

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

// Package savename names the save file extensions and picks the one a save is written under.
package savename

import (
	"path/filepath"

	"againrom/pkg/namefold"
)

const (
	Original = ".sav"
	Extended = ".savx"
)

func Extensions() []string { return []string{Original, Extended} }

func For(extended bool) string {
	if extended {
		return Extended
	}
	return Original
}

func Of(name string) (string, bool) {
	ext := filepath.Ext(name)
	for _, known := range Extensions() {
		if namefold.HostEqual(ext, known) {
			return known, true
		}
	}
	return "", false
}

func Is(name string) bool {
	_, ok := Of(name)
	return ok
}

func IsOriginal(name string) bool {
	ext, ok := Of(name)
	return ok && ext == Original
}

func Hide(name string) string {
	if IsOriginal(name) {
		return name[:len(name)-len(Original)]
	}
	return name
}

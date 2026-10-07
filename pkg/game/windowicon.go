package game

import (
	"fmt"
	"image"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"againrom/pkg/formats/winicon"
)

// WindowIconExecutable is the file of an install that holds the game's icon:
// the original executable's first icon group. The install's .ico files are not
// the game's icon, they carry the installer's own pictures.
const WindowIconExecutable = "rom.exe"

// WindowIcon reads the installed game's icon for its window: one image per size
// the executable's icon group stores, smallest first. Where a size is stored at
// several colour depths the deepest is taken, the one Windows itself shows on a
// display of more than sixteen colours. Windows picks the taskbar, task-switcher
// and title-bar pictures from these by size.
//
// The install is only read. An install without the executable, or with one whose
// icon cannot be decoded, returns an error, and the caller starts without an icon.
func WindowIcon(root string) ([]image.Image, error) {
	path, err := installFilePath(root, WindowIconExecutable)
	if err != nil {
		return nil, err
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	group, err := winicon.FromExecutable(f)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", filepath.Base(path), err)
	}
	return deepestPerSize(group.Images), nil
}

// installFilePath finds the file of an install by name, comparing names with
// case folded: the two lawful installs disagree, EN ships rom.exe and RU ROM.EXE.
func installFilePath(root, name string) (string, error) {
	entries, err := os.ReadDir(root)
	if err != nil {
		return "", err
	}
	for _, e := range entries {
		if !e.IsDir() && strings.EqualFold(e.Name(), name) {
			return filepath.Join(root, e.Name()), nil
		}
	}
	return "", fmt.Errorf("%s is not in %s", name, root)
}

// deepestPerSize keeps, for each size, the first image of the greatest depth,
// and returns the kept ones by width then height.
func deepestPerSize(images []winicon.Image) []image.Image {
	type size struct{ w, h int }
	best := map[size]int{}
	var sizes []size
	for i, im := range images {
		b := im.Pix.Bounds()
		s := size{b.Dx(), b.Dy()}
		j, seen := best[s]
		if !seen {
			sizes = append(sizes, s)
		}
		if !seen || im.Depth > images[j].Depth {
			best[s] = i
		}
	}
	sort.Slice(sizes, func(a, b int) bool {
		if sizes[a].w != sizes[b].w {
			return sizes[a].w < sizes[b].w
		}
		return sizes[a].h < sizes[b].h
	})
	out := make([]image.Image, len(sizes))
	for i, s := range sizes {
		out[i] = images[best[s]].Pix
	}
	return out
}

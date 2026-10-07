package game

import (
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"againrom/pkg/base"
	"againrom/pkg/vfs"
	"againrom/pkg/video"
)

// CutsceneBank selects one optional, disk-indexed video archive. The archive
// choice is explicit: it never searches the process CWD or an unrelated install.
type CutsceneBank struct {
	root, archive string
	once          sync.Once
	index         *vfs.FS
	err           error
	logos         *CutsceneBank
}

func OpenCutscenes(root, archive string) *CutsceneBank {
	if archive == "video4" || archive == "video8" {
		if match, err := DetectBase(root); err == nil && match.Profile.GameOf() == base.GameROM2 {
			archive = "video"
		}
	}
	b := &CutsceneBank{root: root, archive: archive}
	if archive == "video8" {
		b.logos = &CutsceneBank{root: root, archive: "video4"}
	}
	return b
}

func (b *CutsceneBank) open() error {
	if b == nil {
		return fmt.Errorf("video: no asset root")
	}
	b.once.Do(func() {
		if b.root == "" || (b.archive != "video4" && b.archive != "video8" && b.archive != "video") {
			b.err = fmt.Errorf("video: missing root or unknown archive")
			return
		}
		components := []string{"Allods", b.archive + ".res"}
		if b.archive == "video" {
			components = []string{"video.res"}
		}
		path, err := cutsceneInstallPath(b.root, components...)
		if err != nil {
			b.err = err
			return
		}
		// OpenFileIndex validates offsets. This additional video-specific bound
		// prevents an installed node count from requesting a huge index first.
		f, err := os.Open(path)
		if err != nil {
			b.err = err
			return
		}
		var header [24]byte
		_, err = io.ReadFull(f, header[:])
		_ = f.Close()
		if err != nil {
			b.err = err
			return
		}
		if binary.LittleEndian.Uint32(header[20:24]) > 16384 {
			b.err = fmt.Errorf("video: archive registry exceeds 16384 nodes")
			return
		}
		b.index, b.err = vfs.OpenFileBacked([]string{path}, nil)
	})
	return b.err
}

// Media returns exactly the named archive member. It is also the read-only
// release witness; a missing member never falls back to loose files. Logos
// explicitly use VIDEO4 on both speed routes.
func (b *CutsceneBank) Media(name string) ([]byte, error) {
	name, err := cutsceneMember(name)
	if err != nil {
		return nil, err
	}
	if b != nil && b.logos != nil && strings.HasPrefix(name, "logos/") {
		return b.logos.Media(name)
	}
	if err := b.open(); err != nil {
		return nil, err
	}
	for _, e := range b.index.Entries() {
		address := b.archive + "/" + name
		if e.Address != address {
			continue
		}
		if e.Size <= 0 || e.Size > video.MaxMediaBytes {
			return nil, fmt.Errorf("video: media size %d outside permitted range", e.Size)
		}
		return b.index.ReadFile(address)
	}
	return nil, fmt.Errorf("%w: %s in %s", video.ErrAbsent, name, b.archive)
}

// Open decodes the selected member and companion with the portable decoder.
func (b *CutsceneBank) Open(name string) (*video.Player, error) {
	if b == nil {
		return nil, fmt.Errorf("video: no asset root")
	}
	media, err := b.Media(name)
	if err != nil {
		return nil, err
	}
	sidecar, err := b.sidecar(name)
	if err != nil {
		return nil, err
	}
	return video.StartSmackerDecoderWith(media, sidecar)
}

// sidecar reads the registry beside a movie from the same archive (VIDEO-031).
// A movie with no registry plays without effects; a registry that does not
// parse refuses the movie.
func (b *CutsceneBank) sidecar(name string) (*video.Sidecar, error) {
	member, err := cutsceneMember(name)
	if err != nil {
		return nil, err
	}
	bank := b
	if b.logos != nil && strings.HasPrefix(member, "logos/") {
		bank = b.logos
	}
	if err := bank.open(); err != nil {
		return nil, err
	}
	address := bank.archive + "/" + video.SidecarName(member)
	for _, e := range bank.index.Entries() {
		if e.Address != address {
			continue
		}
		if e.Size <= 0 || e.Size > maxSidecarBytes {
			return nil, fmt.Errorf("video: sidecar size %d outside permitted range", e.Size)
		}
		data, err := bank.index.ReadFile(address)
		if err != nil {
			return nil, err
		}
		return video.ParseSidecar(data)
	}
	return nil, nil
}

// maxSidecarBytes bounds an installed registry before it is read.
const maxSidecarBytes = 1 << 20

// CutsceneArchive implements the explicit command-line half of VIDEO-029.
// The legacy machine registry is intentionally not consulted (DIV-508).
func CutsceneArchive(fourX, eightX bool) string {
	if eightX && !fourX {
		return "video8"
	}
	return "video4"
}

func cutsceneMember(name string) (string, error) {
	name = strings.ToLower(strings.ReplaceAll(name, `\`, "/"))
	if len(name) == 0 || len(name) > 240 || !strings.HasSuffix(name, ".smk") || strings.ContainsAny(name, ":\x00") {
		return "", fmt.Errorf("video: invalid member name")
	}
	for _, part := range strings.Split(name, "/") {
		if part == "" || part == "." || part == ".." {
			return "", fmt.Errorf("video: invalid member path")
		}
	}
	return name, nil
}

// cutsceneInstallPath folds each known component over the selected install.
// The resulting absolute DLL path is the only one given to the native helper.
func cutsceneInstallPath(root string, components ...string) (string, error) {
	if root == "" {
		return "", fmt.Errorf("video: no asset root")
	}
	path, err := filepath.Abs(root)
	if err != nil {
		return "", err
	}
	for _, component := range components {
		entries, err := os.ReadDir(path)
		if err != nil {
			return "", err
		}
		found := false
		for _, entry := range entries {
			if strings.EqualFold(entry.Name(), component) {
				path = filepath.Join(path, entry.Name())
				found = true
				break
			}
		}
		if !found {
			return "", fmt.Errorf("video: installed %s missing", component)
		}
	}
	return path, nil
}

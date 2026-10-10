package game

import (
	_ "embed"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"againrom/pkg/audio"
	"againrom/pkg/base"
	"againrom/pkg/ui"
	"againrom/pkg/vfs"
)

// MusicArchive is the first game's optional music container under Allods in
// the selected asset root. It is not one of RequiredArchives: absence is
// silence.
const MusicArchive = "MUSIC.RES"

// rom1MusicJSON and rom2MusicJSON are the two games' music as data for the
// music controller. Every value in them carries the claim, divergence row or
// owner ruling behind it.
//
//go:embed musics/rom1.json
var rom1MusicJSON []byte

//go:embed musics/rom2.json
var rom2MusicJSON []byte

// musicDescriptions are the decoded descriptions an edition can name, by that
// name. A description that does not decode is a build defect, so it stops the
// process at start.
var musicDescriptions = map[string]*ui.MusicDescription{
	"rom1": mustDecodeMusic(rom1MusicJSON),
	"rom2": mustDecodeMusic(rom2MusicJSON),
}

func mustDecodeMusic(data []byte) *ui.MusicDescription {
	d, err := ui.DecodeMusic(data)
	if err != nil {
		panic(err)
	}
	return d
}

// GameMusic is the music description the profile's edition names. Callers
// must not change it.
func GameMusic(p base.Profile) *ui.MusicDescription {
	return musicDescriptions[p.Edition().Music]
}

// MusicTrackNames returns the first game's complete shipped static-name
// manifest. inn_ssi is included in the archive manifest but intentionally
// belongs to no scene list.
func MusicTrackNames() []string {
	return append([]string(nil), GameMusic(base.Profile{}).Tracks...)
}

// MusicBank lazily opens a disk-indexed music archive. It retains the small
// registry only. Track reads and decodes one payload and caches neither
// success nor failure, so the retained player is the sole owner of current
// track bytes.
type MusicBank struct {
	path string
	desc *ui.MusicDescription
	once sync.Once
	src  *vfs.FS
}

// OpenMusic names the first game's optional archive without touching it.
func OpenMusic(root string) *MusicBank { return OpenMusicFor(root, GameMusic(base.Profile{})) }

// OpenMusicFor names the archive description d places under root without
// touching it.
func OpenMusicFor(root string, d *ui.MusicDescription) *MusicBank {
	if root == "" || d == nil {
		return nil
	}
	return &MusicBank{path: filepath.Join(root, filepath.FromSlash(d.Archive.Path)), desc: d}
}

func (b *MusicBank) open() *vfs.FS {
	if b == nil {
		return nil
	}
	b.once.Do(func() {
		b.src, _ = vfs.OpenFileBacked([]string{foldedHostPath(b.path)}, nil)
	})
	return b.src
}

// foldedHostPath is path, or the entry of its directory whose name equals
// its base ignoring case; the two games' roots spell the archive in either
// case.
func foldedHostPath(path string) string {
	if _, err := os.Stat(path); err == nil {
		return path
	}
	dir, name := filepath.Split(path)
	entries, err := os.ReadDir(dir)
	if err != nil {
		return path
	}
	for _, e := range entries {
		if !e.IsDir() && strings.EqualFold(e.Name(), name) {
			return filepath.Join(dir, e.Name())
		}
	}
	return path
}

// Track resolves one member of the description's manifest and decodes it as
// device-rate stereo. Unknown, missing and malformed names are silence.
func (b *MusicBank) Track(name string) (audio.Track, bool) {
	if b == nil || !b.desc.Known(name) {
		return audio.Track{}, false
	}
	src := b.open()
	if src == nil {
		return audio.Track{}, false
	}
	raw, err := src.ReadFile("music/" + name)
	if err != nil {
		return audio.Track{}, false
	}
	track, err := audio.DecodeTrackWAV(raw, audio.DeviceRate)
	if err != nil {
		return audio.Track{}, false
	}
	return track, true
}

// Manifest reports the archive registry without reading a payload.
func (b *MusicBank) Manifest() []string {
	src := b.open()
	if src == nil {
		return nil
	}
	entries := src.Entries()
	out := make([]string, 0, len(entries))
	for _, entry := range entries {
		out = append(out, strings.TrimPrefix(entry.Address, "music/"))
	}
	return out
}

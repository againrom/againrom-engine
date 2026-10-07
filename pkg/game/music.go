package game

import (
	"path/filepath"
	"strings"
	"sync"

	"againrom/pkg/audio"
	"againrom/pkg/vfs"
)

// MusicArchive is the optional ROM1 music container under Allods in the
// selected asset root. It is not one of RequiredArchives: absence is silence.
const MusicArchive = "MUSIC.RES"

var musicTrackNames = []string{
	"B00.wav", "B01.wav", "B02.wav", "B03.wav", "B04.wav", "B05.wav",
	"B06.wav", "B07.wav", "B08.wav", "B09.wav", "B10.wav", "B11.wav",
	"chrgen.wav", "inn.wav", "inn_ssi.wav", "map.wav", "menu.wav",
	"schoolm.wav", "schoolw.wav", "shop.wav", "town.wav",
}

// MusicTrackNames returns the complete shipped static-name manifest. inn_ssi is
// included in the archive manifest but intentionally belongs to no scene list.
func MusicTrackNames() []string { return append([]string(nil), musicTrackNames...) }

// MusicBank lazily opens a disk-indexed MUSIC.RES. It retains the small registry
// only. Track reads and decodes one payload and caches neither success nor
// failure, so the retained player is the sole owner of current track bytes.
type MusicBank struct {
	path string
	once sync.Once
	src  *vfs.FS
}

// OpenMusic names the optional archive without touching it.
func OpenMusic(root string) *MusicBank {
	if root == "" {
		return nil
	}
	return &MusicBank{path: filepath.Join(root, "Allods", MusicArchive)}
}

func (b *MusicBank) open() *vfs.FS {
	if b == nil {
		return nil
	}
	b.once.Do(func() {
		b.src, _ = vfs.OpenFileBacked([]string{b.path}, nil)
	})
	return b.src
}

// Track resolves one member of the fixed shipped manifest and decodes it as
// device-rate stereo. Unknown, missing and malformed names are silence.
func (b *MusicBank) Track(name string) (audio.Track, bool) {
	if !knownMusicTrack(name) {
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

func knownMusicTrack(name string) bool {
	for _, known := range musicTrackNames {
		if strings.EqualFold(name, known) {
			return true
		}
	}
	return false
}

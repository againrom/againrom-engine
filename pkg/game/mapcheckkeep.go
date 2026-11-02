package game

import (
	"bytes"
	"maps"
	"time"

	"againrom/pkg/vfs"
)

// installPagesKeep holds the install text pages of the last mission asked
// for. The archives of one install do not change while it is open, so the
// answer is keyed on the archive set and the mission number: another install
// or mission reads again.
type installPagesKeep struct {
	src   *vfs.FS
	n     int
	files map[string][]byte
	err   error
	held  bool
	reads int // how many times the archives were walked
}

// pages is installedPages, read once per archive set and mission. The map it
// returns is the caller's own; the page bytes are shared and never written.
func (k *installPagesKeep) pages(src *vfs.FS, n int) (map[string][]byte, error) {
	if !k.held || k.src != src || k.n != n {
		files, err := installedPages(src, n)
		k.src, k.n, k.files, k.err, k.held = src, n, files, err, true
		k.reads++
	}
	if k.err != nil {
		return nil, k.err
	}
	return maps.Clone(k.files), nil
}

// textsStampLife is how long a page change made outside the editor may stay
// unseen by the map report.
const textsStampLife = time.Second

// textsStampKeep keeps the texts stamp of the report while nothing the editor
// holds has changed, for at most textsStampLife.
type textsStampKeep struct {
	held      bool
	at        time.Time
	source    string
	image     []byte
	saved     []byte
	workspace *editorWorkspaceState
	mod       *editorMod
	archives  *Archives
	value     string
}

func (s *mapInspectionEdits) stampNow() time.Time {
	if s.clock != nil {
		return s.clock()
	}
	return time.Now()
}

// fresh reports whether the kept stamp still answers for the editor's state.
func (k *textsStampKeep) fresh(s *mapInspectionEdits, now time.Time) bool {
	age := now.Sub(k.at)
	return k.held && age >= 0 && age < textsStampLife &&
		k.source == s.source && k.workspace == s.workspace &&
		k.mod == s.inspector.mod && k.archives == s.inspector.archives &&
		bytes.Equal(k.image, s.workspaceImage) && bytes.Equal(k.saved, s.savedWorkspace)
}

func (k *textsStampKeep) keep(s *mapInspectionEdits, now time.Time, value string) {
	*k = textsStampKeep{held: true, at: now, source: s.source, workspace: s.workspace,
		mod: s.inspector.mod, archives: s.inspector.archives,
		image: bytes.Clone(s.workspaceImage), saved: bytes.Clone(s.savedWorkspace), value: value}
}
